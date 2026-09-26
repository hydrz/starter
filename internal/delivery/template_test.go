package delivery_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/hydrz/starter/internal/delivery"
)

func TestTemplateRenderer_Verification(t *testing.T) {
	renderer, err := delivery.NewTemplateRenderer("Starter App")
	if err != nil {
		t.Fatalf("NewTemplateRenderer failed: %v", err)
	}

	data := delivery.EmailTemplateData{
		Email:     "user@example.com",
		Token:     "verify-token-123456",
		ActionURL: "https://example.com/verify?token=verify-token-123456",
	}

	rendered, err := renderer.Render(delivery.TopicAuthVerificationRequested, data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(rendered.Subject, "Verify your email address") {
		t.Errorf("subject %q does not contain expected title", rendered.Subject)
	}
	if !strings.Contains(rendered.Subject, "Starter App") {
		t.Errorf("subject %q does not contain AppName", rendered.Subject)
	}

	// Plain text checks
	if !strings.Contains(rendered.TextBody, "verify-token-123456") {
		t.Errorf("text body missing token: %s", rendered.TextBody)
	}
	if !strings.Contains(rendered.TextBody, "https://example.com/verify?token=verify-token-123456") {
		t.Errorf("text body missing action URL: %s", rendered.TextBody)
	}

	// HTML checks
	if !strings.Contains(rendered.HTMLBody, "<!DOCTYPE html>") {
		t.Errorf("html body missing DOCTYPE: %s", rendered.HTMLBody)
	}
	if !strings.Contains(rendered.HTMLBody, "verify-token-123456") {
		t.Errorf("html body missing token: %s", rendered.HTMLBody)
	}
	if !strings.Contains(rendered.HTMLBody, "https://example.com/verify?token=verify-token-123456") {
		t.Errorf("html body missing action URL: %s", rendered.HTMLBody)
	}
}

func TestTemplateRenderer_PasswordReset(t *testing.T) {
	renderer, err := delivery.NewTemplateRenderer("Starter App")
	if err != nil {
		t.Fatalf("NewTemplateRenderer failed: %v", err)
	}

	data := delivery.EmailTemplateData{
		Email:     "user@example.com",
		Token:     "reset-token-987654",
		ActionURL: "https://example.com/reset?token=reset-token-987654",
	}

	rendered, err := renderer.Render(delivery.TopicAuthPasswordResetRequested, data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(rendered.Subject, "Reset your password") {
		t.Errorf("subject %q does not contain expected title", rendered.Subject)
	}
	if !strings.Contains(rendered.Subject, "Starter App") {
		t.Errorf("subject %q does not contain AppName", rendered.Subject)
	}

	// Plain text checks
	if !strings.Contains(rendered.TextBody, "reset-token-987654") {
		t.Errorf("text body missing token: %s", rendered.TextBody)
	}
	if !strings.Contains(rendered.TextBody, "https://example.com/reset?token=reset-token-987654") {
		t.Errorf("text body missing action URL: %s", rendered.TextBody)
	}

	// HTML checks
	if !strings.Contains(rendered.HTMLBody, "<!DOCTYPE html>") {
		t.Errorf("html body missing DOCTYPE: %s", rendered.HTMLBody)
	}
	if !strings.Contains(rendered.HTMLBody, "reset-token-987654") {
		t.Errorf("html body missing token: %s", rendered.HTMLBody)
	}
}

func TestTemplateRenderer_UnknownTopic(t *testing.T) {
	renderer, err := delivery.NewTemplateRenderer("")
	if err != nil {
		t.Fatalf("NewTemplateRenderer failed: %v", err)
	}

	_, err = renderer.Render("unknown.topic", delivery.EmailTemplateData{})
	if err == nil {
		t.Fatal("expected error for unknown topic, got nil")
	}
	if !errors.Is(err, delivery.ErrUnknownTopic) {
		t.Errorf("expected ErrUnknownTopic, got %v", err)
	}
}

func TestTemplateRenderer_EscapesHTML(t *testing.T) {
	renderer, err := delivery.NewTemplateRenderer("")
	if err != nil {
		t.Fatalf("NewTemplateRenderer failed: %v", err)
	}

	data := delivery.EmailTemplateData{
		Email: "test<script>alert(1)</script>@example.com",
		Token: "<b>malicious-token</b>",
	}

	rendered, err := renderer.Render(delivery.TopicAuthVerificationRequested, data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if strings.Contains(rendered.HTMLBody, "<b>malicious-token</b>") {
		t.Errorf("HTML body contains unescaped HTML: %s", rendered.HTMLBody)
	}
	if !strings.Contains(rendered.HTMLBody, "&lt;b&gt;malicious-token&lt;/b&gt;") {
		t.Errorf("HTML body did not properly escape token: %s", rendered.HTMLBody)
	}
}
