package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"time"
)

var ErrNotConfigured = errors.New("mailer: Brevo sender is not configured")

type VerificationSender interface {
	SendVerification(ctx context.Context, email, name, link string) error
	SendPasswordReset(ctx context.Context, email, name, link string) error
	SendDeveloperInvitation(ctx context.Context, email, name, link string) error
}

type Brevo struct {
	APIKey      string
	SenderEmail string
	SenderName  string
	Client      *http.Client
	Endpoint    string
}

func NewBrevo(apiKey, senderEmail, senderName string) *Brevo {
	return &Brevo{
		APIKey:      apiKey,
		SenderEmail: senderEmail,
		SenderName:  senderName,
		Client:      &http.Client{Timeout: 10 * time.Second},
		Endpoint:    "https://api.brevo.com/v3/smtp/email",
	}
}

func (b *Brevo) SendVerification(ctx context.Context, email, name, link string) error {
	return b.sendMessage(ctx, email, name, "Verify your PayConnect email", fmt.Sprintf(
		`<p>Hello %s,</p><p>Verify your email to finish creating your PayConnect account.</p><p><a href="%s">Verify email</a></p><p>This link expires in 1 hour.</p>`,
		html.EscapeString(name), html.EscapeString(link),
	))
}

func (b *Brevo) SendPasswordReset(ctx context.Context, email, name, link string) error {
	return b.sendMessage(ctx, email, name, "Reset your PayConnect password", fmt.Sprintf(
		`<p>Hello %s,</p><p>Use the link below to reset your PayConnect password. It expires in 30 minutes.</p><p><a href="%s">Reset password</a></p>`,
		html.EscapeString(name), html.EscapeString(link),
	))
}

func (b *Brevo) SendDeveloperInvitation(ctx context.Context, email, name, link string) error {
	return b.sendMessage(ctx, email, name, "Join PayConnect as a developer", fmt.Sprintf(
		`<p>Hello %s,</p><p>You have been invited to join a PayConnect merchant workspace as a developer.</p><p><a href="%s">Accept developer invitation</a></p><p>This link expires in 7 days.</p>`,
		html.EscapeString(name), html.EscapeString(link),
	))
}

func (b *Brevo) sendMessage(ctx context.Context, email, name, subject, htmlContent string) error {
	if b.APIKey == "" || b.SenderEmail == "" {
		return ErrNotConfigured
	}

	payload := struct {
		Sender struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"sender"`
		To []struct {
			Email string `json:"email"`
			Name  string `json:"name,omitempty"`
		} `json:"to"`
		Subject     string `json:"subject"`
		HTMLContent string `json:"htmlContent"`
	}{}
	payload.Sender.Name = b.SenderName
	payload.Sender.Email = b.SenderEmail
	payload.To = append(payload.To, struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}{Email: email, Name: name})
	payload.Subject = subject
	payload.HTMLContent = htmlContent

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailer: encode Brevo message: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, b.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mailer: create Brevo request: %w", err)
	}
	request.Header.Set("api-key", b.APIKey)
	request.Header.Set("Content-Type", "application/json")
	client := b.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("mailer: send verification email: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("mailer: Brevo returned HTTP %d", response.StatusCode)
	}
	return nil
}
