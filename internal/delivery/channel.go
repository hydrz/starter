package delivery

import (
	"context"
	"log/slog"
)

// Message represents an outgoing message to be delivered through a Channel.
type Message struct {
	ID             string
	Recipient      string
	Subject        string
	TextBody       string
	HTMLBody       string
	Headers        map[string]string
	IdempotencyKey string
}

// Channel abstracts delivery mechanisms (e.g. SMTP, SES, Webhook).
type Channel interface {
	Name() string
	Deliver(ctx context.Context, msg Message) (providerResponse string, err error)
}

// NoopChannel is a channel that logs delivered messages without sending them.
// Useful for local development and test environments where external email delivery is disabled.
type NoopChannel struct {
	name   string
	logger *slog.Logger
}

// NewNoopChannel creates a NoopChannel with the specified name and logger.
func NewNoopChannel(name string, logger *slog.Logger) *NoopChannel {
	if name == "" {
		name = "noop"
	}
	return &NoopChannel{name: name, logger: logger}
}

func (c *NoopChannel) Name() string {
	return c.name
}

func (c *NoopChannel) Deliver(ctx context.Context, msg Message) (string, error) {
	if c.logger != nil {
		c.logger.Info("noop channel delivered message",
			"channel", c.name,
			"recipient", msg.Recipient,
			"subject", msg.Subject,
			"idempotency_key", msg.IdempotencyKey,
		)
	}
	return "250 OK (noop)", nil
}
