package delivery

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
)

//go:embed templates/*.html templates/*.txt
var templateFS embed.FS

const (
	TopicAuthVerificationRequested  = "auth.verification_requested"
	TopicAuthPasswordResetRequested = "auth.password_reset_requested"

	defaultAppName   = "Starter"
	defaultExpiresIn = "15 minutes"
)

var (
	ErrUnknownTopic = errors.New("delivery: unknown template topic")
)

// EmailTemplateData holds data provided to email templates.
type EmailTemplateData struct {
	Email      string
	Token      string
	ActionURL  string
	AppName    string
	ExpiresIn  string
	SupportURL string
}

// RenderedEmail represents the dual-mode output of template rendering.
type RenderedEmail struct {
	Subject  string
	TextBody string
	HTMLBody string
}

// TemplateRenderer compiles and executes embedded HTML and plain-text templates.
type TemplateRenderer struct {
	appName       string
	htmlTemplates map[string]*htmltemplate.Template
	textTemplates map[string]*texttemplate.Template
}

// NewTemplateRenderer initializes a TemplateRenderer with embedded email templates.
func NewTemplateRenderer(appName string) (*TemplateRenderer, error) {
	if appName == "" {
		appName = defaultAppName
	}

	htmlMap := make(map[string]*htmltemplate.Template)
	textMap := make(map[string]*texttemplate.Template)

	topics := []struct {
		topic    string
		htmlFile string
		textFile string
	}{
		{
			topic:    TopicAuthVerificationRequested,
			htmlFile: "templates/auth_verification.html",
			textFile: "templates/auth_verification.txt",
		},
		{
			topic:    TopicAuthPasswordResetRequested,
			htmlFile: "templates/auth_password_reset.html",
			textFile: "templates/auth_password_reset.txt",
		},
	}

	for _, t := range topics {
		hTmpl, err := htmltemplate.ParseFS(templateFS, t.htmlFile)
		if err != nil {
			return nil, fmt.Errorf("parse html template %s: %w", t.htmlFile, err)
		}
		htmlMap[t.topic] = hTmpl

		tTmpl, err := texttemplate.ParseFS(templateFS, t.textFile)
		if err != nil {
			return nil, fmt.Errorf("parse text template %s: %w", t.textFile, err)
		}
		textMap[t.topic] = tTmpl
	}

	return &TemplateRenderer{
		appName:       appName,
		htmlTemplates: htmlMap,
		textTemplates: textMap,
	}, nil
}

// Render executes both HTML and plain-text templates for the given topic.
func (r *TemplateRenderer) Render(topic string, data EmailTemplateData) (RenderedEmail, error) {
	hTmpl, hasHTML := r.htmlTemplates[topic]
	tTmpl, hasText := r.textTemplates[topic]
	if !hasHTML || !hasText {
		return RenderedEmail{}, fmt.Errorf("%w: %s", ErrUnknownTopic, topic)
	}

	if data.AppName == "" {
		data.AppName = r.appName
	}
	if data.ExpiresIn == "" {
		data.ExpiresIn = defaultExpiresIn
	}

	var subject string
	switch topic {
	case TopicAuthVerificationRequested:
		subject = fmt.Sprintf("Verify your email address - %s", data.AppName)
	case TopicAuthPasswordResetRequested:
		subject = fmt.Sprintf("Reset your password - %s", data.AppName)
	default:
		subject = fmt.Sprintf("Notification from %s", data.AppName)
	}

	var textBuf bytes.Buffer
	if err := tTmpl.Execute(&textBuf, data); err != nil {
		return RenderedEmail{}, fmt.Errorf("execute text template for %s: %w", topic, err)
	}

	var htmlBuf bytes.Buffer
	if err := hTmpl.Execute(&htmlBuf, data); err != nil {
		return RenderedEmail{}, fmt.Errorf("execute html template for %s: %w", topic, err)
	}

	return RenderedEmail{
		Subject:  subject,
		TextBody: textBuf.String(),
		HTMLBody: htmlBuf.String(),
	}, nil
}
