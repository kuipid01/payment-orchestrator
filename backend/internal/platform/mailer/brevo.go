package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("mailer: Brevo sender is not configured")

type VerificationSender interface {
	SendVerification(ctx context.Context, email, name, link string) error
	SendPasswordReset(ctx context.Context, email, name, link string) error
	SendDeveloperInvitation(ctx context.Context, email, name, link string) error
}

// DeveloperInvitation carries the optional context shown in the invitation
// email. Only Email and Link are required; the rest enrich the template.
type DeveloperInvitation struct {
	Email        string
	Name         string
	Link         string
	MerchantName string   // workspace the developer is being invited to
	InviterName  string   // who sent the invitation
	Permissions  []string // e.g. DeveloperMerchantAccess.Permissions
}

var _ VerificationSender = (*Brevo)(nil)

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
	m, err := renderVerification(name, link)
	if err != nil {
		return err
	}
	return b.send(ctx, email, name, m)
}

func (b *Brevo) SendPasswordReset(ctx context.Context, email, name, link string) error {
	m, err := renderPasswordReset(name, link)
	if err != nil {
		return err
	}
	return b.send(ctx, email, name, m)
}

// SendDeveloperInvitation satisfies VerificationSender with the minimal
// invitation. Prefer SendDeveloperInvitationDetails when the workspace name,
// inviter and permissions are known.
func (b *Brevo) SendDeveloperInvitation(ctx context.Context, email, name, link string) error {
	return b.SendDeveloperInvitationDetails(ctx, DeveloperInvitation{Email: email, Name: name, Link: link})
}

// SendDeveloperInvitationDetails sends the invitation with workspace, inviter
// and permission details rendered in the email.
func (b *Brevo) SendDeveloperInvitationDetails(ctx context.Context, inv DeveloperInvitation) error {
	m, err := renderDeveloperInvitation(inv)
	if err != nil {
		return err
	}
	return b.send(ctx, inv.Email, inv.Name, m)
}

type brevoContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoPayload struct {
	Sender      brevoContact   `json:"sender"`
	To          []brevoContact `json:"to"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent"`
	TextContent string         `json:"textContent,omitempty"`
}

func (b *Brevo) send(ctx context.Context, email, name string, m message) error {
	if b.APIKey == "" || b.SenderEmail == "" {
		return ErrNotConfigured
	}
	if strings.TrimSpace(email) == "" {
		return errors.New("mailer: recipient email is required")
	}

	body, err := json.Marshal(brevoPayload{
		Sender:      brevoContact{Email: b.SenderEmail, Name: b.SenderName},
		To:          []brevoContact{{Email: email, Name: name}},
		Subject:     m.Subject,
		HTMLContent: m.HTML,
		TextContent: m.Text,
	})
	if err != nil {
		return fmt.Errorf("mailer: encode Brevo message: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, b.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mailer: create Brevo request: %w", err)
	}
	request.Header.Set("api-key", b.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	client := b.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("mailer: send email: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("mailer: Brevo returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}
