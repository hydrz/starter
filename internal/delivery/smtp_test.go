package delivery_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
)

func TestBuildMIMEMessage_DualMode(t *testing.T) {
	from := "sender@example.com"
	to := "recipient@example.com"
	subject := "Test Verification Code"
	textBody := "Your code is: 123456"
	htmlBody := "<h1>Your code is: 123456</h1>"
	headers := map[string]string{
		"X-Custom-Header": "custom-value",
	}

	msg, err := delivery.BuildMIMEMessage(from, to, subject, textBody, htmlBody, headers)
	if err != nil {
		t.Fatalf("BuildMIMEMessage failed: %v", err)
	}

	content := string(msg)
	if !strings.Contains(content, "From: sender@example.com") {
		t.Errorf("missing From header: %s", content)
	}
	if !strings.Contains(content, "To: recipient@example.com") {
		t.Errorf("missing To header: %s", content)
	}
	if !strings.Contains(content, "X-Custom-Header: custom-value") {
		t.Errorf("missing custom header: %s", content)
	}
	if !strings.Contains(content, "multipart/alternative") {
		t.Errorf("missing multipart/alternative: %s", content)
	}
	if !strings.Contains(content, "text/plain") {
		t.Errorf("missing text/plain part: %s", content)
	}
	if !strings.Contains(content, "text/html") {
		t.Errorf("missing text/html part: %s", content)
	}
	if !strings.Contains(content, "123456") {
		t.Errorf("missing body content: %s", content)
	}
}

func TestBuildMIMEMessage_TextOnly(t *testing.T) {
	msg, err := delivery.BuildMIMEMessage("sender@example.com", "recipient@example.com", "Plain Subject", "Hello Plain Text", "", nil)
	if err != nil {
		t.Fatalf("BuildMIMEMessage failed: %v", err)
	}

	content := string(msg)
	if strings.Contains(content, "multipart/alternative") {
		t.Errorf("text-only message should not be multipart: %s", content)
	}
	if !strings.Contains(content, "Content-Type: text/plain") {
		t.Errorf("missing text/plain Content-Type: %s", content)
	}
	if !strings.Contains(content, "Hello Plain Text") {
		t.Errorf("missing text content: %s", content)
	}
}

func TestBuildMIMEMessage_NonASCIISubject(t *testing.T) {
	subject := "验证码 Verification Code 🚀"
	msg, err := delivery.BuildMIMEMessage("sender@example.com", "recipient@example.com", subject, "Body", "", nil)
	if err != nil {
		t.Fatalf("BuildMIMEMessage failed: %v", err)
	}

	content := string(msg)
	if !strings.Contains(content, "Subject: =?utf-8?") {
		t.Errorf("expected MIME encoded-word in subject, got: %s", content)
	}
}

func TestNoopChannel(t *testing.T) {
	ch := delivery.NewNoopChannel("test-noop", nil)
	if ch.Name() != "test-noop" {
		t.Errorf("expected name test-noop, got %s", ch.Name())
	}

	resp, err := ch.Deliver(context.Background(), delivery.Message{
		Recipient: "user@example.com",
		Subject:   "Hello",
		TextBody:  "Test",
	})
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}
	if !strings.Contains(resp, "250 OK") {
		t.Errorf("unexpected response: %s", resp)
	}
}

// startMockSMTPServer starts an in-memory test SMTP server.
func startMockSMTPServer(t *testing.T, authUser, authPass string) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	addr := listener.Addr().String()
	closed := make(chan struct{})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-closed:
					return
				default:
					return
				}
			}
			go handleMockSMTPConn(conn, authUser, authPass)
		}
	}()

	cleanup := func() {
		close(closed)
		_ = listener.Close()
	}

	return addr, cleanup
}

func handleMockSMTPConn(conn net.Conn, expectedUser, expectedPass string) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	send := func(msg string) {
		_, _ = writer.WriteString(msg + "\r\n")
		_ = writer.Flush()
	}

	send("220 mock.smtp.local Service ready")

	inData := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				send("250 2.0.0 OK: message queued")
			}
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
			if expectedUser != "" {
				send("250-mock.smtp.local")
				send("250-AUTH PLAIN")
				send("250 OK")
			} else {
				send("250 mock.smtp.local OK")
			}
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			send("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			send("250 2.1.0 Sender OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			if strings.Contains(strings.ToLower(line), "reject@example.com") {
				send("550 5.1.1 User unknown")
			} else {
				send("250 2.1.5 Recipient OK")
			}
		case upper == "DATA":
			inData = true
			send("354 Start mail input; end with <CRLF>.<CRLF>")
		case upper == "QUIT":
			send("221 2.0.0 Service closing transmission channel")
			return
		case upper == "RSET":
			send("250 2.0.0 OK")
		case upper == "NOOP":
			send("250 2.0.0 OK")
		default:
			send("500 5.5.2 Syntax error, command unrecognized")
		}
	}
}

func TestSMTPChannel_DeliverSuccess(t *testing.T) {
	addr, cleanup := startMockSMTPServer(t, "", "")
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	cfg := config.SMTPConfig{
		Enabled:     true,
		Host:        host,
		Port:        port,
		FromAddress: "noreply@example.com",
	}

	channel := delivery.NewSMTPChannel(cfg)
	if channel.Name() != "smtp" {
		t.Errorf("expected channel name smtp, got %s", channel.Name())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := channel.Deliver(ctx, delivery.Message{
		Recipient: "user@example.com",
		Subject:   "Welcome to Starter",
		TextBody:  "Your verification code is: 123456",
		HTMLBody:  "<p>Your verification code is: 123456</p>",
	})
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	if !strings.Contains(resp, "250") {
		t.Errorf("expected 250 response, got %s", resp)
	}
}

func TestSMTPChannel_DeliverWithAuth(t *testing.T) {
	addr, cleanup := startMockSMTPServer(t, "user123", "pass123")
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	cfg := config.SMTPConfig{
		Enabled:     true,
		Host:        host,
		Port:        port,
		Username:    "user123",
		Password:    "pass123",
		FromAddress: "noreply@example.com",
	}

	channel := delivery.NewSMTPChannel(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := channel.Deliver(ctx, delivery.Message{
		Recipient: "user@example.com",
		Subject:   "Password Reset Request",
		TextBody:  "Your reset code is: 654321",
		HTMLBody:  "<p>Your reset code is: 654321</p>",
	})
	if err != nil {
		t.Fatalf("Deliver with auth failed: %v", err)
	}

	if !strings.Contains(resp, "250") {
		t.Errorf("expected 250 response, got %s", resp)
	}
}

func TestSMTPChannel_DeliverRejectedRecipient(t *testing.T) {
	addr, cleanup := startMockSMTPServer(t, "", "")
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	cfg := config.SMTPConfig{
		Enabled:     true,
		Host:        host,
		Port:        port,
		FromAddress: "noreply@example.com",
	}

	channel := delivery.NewSMTPChannel(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := channel.Deliver(ctx, delivery.Message{
		Recipient: "reject@example.com",
		Subject:   "Test",
		TextBody:  "Hello",
	})
	if err == nil {
		t.Fatal("expected error for rejected recipient, got nil")
	}
	if !strings.Contains(err.Error(), "550") {
		t.Errorf("expected 550 error, got %v", err)
	}
}
