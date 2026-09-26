package delivery

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/hydrz/starter/internal/platform/config"
)

// SMTPChannelOptions allows injecting custom dialers or TLS config for testing.
type SMTPChannelOptions struct {
	Dialer    func(ctx context.Context, network, addr string) (net.Conn, error)
	TLSConfig *tls.Config
}

// SMTPChannel implements Channel using the standard SMTP protocol.
type SMTPChannel struct {
	cfg     config.SMTPConfig
	options SMTPChannelOptions
}

// NewSMTPChannel creates a new SMTPChannel.
func NewSMTPChannel(cfg config.SMTPConfig, opts ...SMTPChannelOptions) *SMTPChannel {
	var opt SMTPChannelOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	return &SMTPChannel{
		cfg:     cfg,
		options: opt,
	}
}

func (c *SMTPChannel) Name() string {
	return "smtp"
}

// Deliver sends the message via SMTP with proper TLS/STARTTLS and MIME multipart support.
func (c *SMTPChannel) Deliver(ctx context.Context, msg Message) (string, error) {
	if c.cfg.Host == "" {
		return "", errors.New("smtp: host is not configured")
	}

	fromAddr, err := mail.ParseAddress(c.cfg.FromAddress)
	if err != nil {
		return "", fmt.Errorf("smtp: parse from address %q: %w", c.cfg.FromAddress, err)
	}

	toAddr, err := mail.ParseAddress(msg.Recipient)
	if err != nil {
		return "", fmt.Errorf("smtp: parse recipient address %q: %w", msg.Recipient, err)
	}

	port := c.cfg.Port
	if port <= 0 {
		port = 587
	}
	addr := net.JoinHostPort(c.cfg.Host, strconv.Itoa(port))

	tlsConfig := c.options.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			ServerName: c.cfg.Host,
			MinVersion: tls.VersionTLS12,
		}
	}

	client, err := c.connect(ctx, addr, port, tlsConfig)
	if err != nil {
		return "", fmt.Errorf("smtp connect: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials are provided
	if c.cfg.Username != "" {
		auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return "", fmt.Errorf("smtp auth: %w", err)
		}
	}

	// Set envelope sender
	if err := client.Mail(fromAddr.Address); err != nil {
		return "", fmt.Errorf("smtp mail from: %w", err)
	}

	// Set envelope recipient
	if err := client.Rcpt(toAddr.Address); err != nil {
		return "", fmt.Errorf("smtp rcpt to: %w", err)
	}

	// Send message content
	w, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("smtp data: %w", err)
	}

	rawMessage, err := BuildMIMEMessage(fromAddr.String(), toAddr.String(), msg.Subject, msg.TextBody, msg.HTMLBody, msg.Headers)
	if err != nil {
		_ = w.Close()
		return "", fmt.Errorf("build mime message: %w", err)
	}

	if _, err := w.Write(rawMessage); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("write data: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close data: %w", err)
	}

	if err := client.Quit(); err != nil {
		// Some servers close the connection upon QUIT; if message was accepted, treat as success
		return "250 OK", nil
	}

	return "250 OK: message queued", nil
}

func (c *SMTPChannel) connect(ctx context.Context, addr string, port int, tlsConfig *tls.Config) (*smtp.Client, error) {
	// Port 465 is implicit TLS (SMTPS)
	if port == 465 {
		var conn net.Conn
		var err error
		if c.options.Dialer != nil {
			conn, err = c.options.Dialer(ctx, "tcp", addr)
			if err != nil {
				return nil, err
			}
			tlsConn := tls.Client(conn, tlsConfig)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("tls handshake: %w", err)
			}
			conn = tlsConn
		} else {
			dialer := &net.Dialer{Timeout: 15 * time.Second}
			conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
			if err != nil {
				return nil, fmt.Errorf("tls dial: %w", err)
			}
		}
		return smtp.NewClient(conn, c.cfg.Host)
	}

	// Ports 587, 25, 2525 etc.: start plain and upgrade with STARTTLS if supported
	var conn net.Conn
	var err error
	if c.options.Dialer != nil {
		conn, err = c.options.Dialer(ctx, "tcp", addr)
	} else {
		dialer := &net.Dialer{Timeout: 15 * time.Second}
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("new client: %w", err)
	}

	// Check for STARTTLS support and upgrade
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("starttls: %w", err)
		}
	}

	return client, nil
}

// BuildMIMEMessage formats an RFC 2822/5322 MIME message with dual-mode text/HTML support.
func BuildMIMEMessage(from, to, subject, textBody, htmlBody string, customHeaders map[string]string) ([]byte, error) {
	var buf bytes.Buffer

	// Standard Headers
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	buf.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")

	// Generate Message-ID
	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)
	msgID := fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), hex.EncodeToString(randBytes), extractDomain(from))
	buf.WriteString("Message-ID: " + msgID + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Custom headers
	for k, v := range customHeaders {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	hasText := strings.TrimSpace(textBody) != ""
	hasHTML := strings.TrimSpace(htmlBody) != ""

	switch {
	case hasText && hasHTML:
		boundary := "bnd_" + hex.EncodeToString(randBytes)
		buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")

		// Text part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&buf, textBody); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")

		// HTML part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&buf, htmlBody); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "--\r\n")

	case hasHTML:
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&buf, htmlBody); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")

	default:
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&buf, textBody); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")
	}

	return buf.Bytes(), nil
}

func writeQuotedPrintable(w io.Writer, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := qp.Write([]byte(s)); err != nil {
		return err
	}
	return qp.Close()
}

func extractDomain(addr string) string {
	parsed, err := mail.ParseAddress(addr)
	if err == nil {
		addr = parsed.Address
	}
	parts := strings.Split(addr, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return "localhost"
}
