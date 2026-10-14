package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendVerificationUsesBrevoPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("api-key") != "brevo-test-key" {
			t.Errorf("Brevo API key header was not set")
		}
		var payload struct {
			Sender struct {
				Email string `json:"email"`
			} `json:"sender"`
			To []struct {
				Email string `json:"email"`
			} `json:"to"`
			Subject     string `json:"subject"`
			HTMLContent string `json:"htmlContent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode Brevo payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload.Sender.Email != "no-reply@example.com" || len(payload.To) != 1 || payload.To[0].Email != "user@example.com" {
			t.Errorf("unexpected sender or recipient: %+v", payload)
		}
		if payload.Subject == "" || !strings.Contains(payload.HTMLContent, "https://app.example.com/verify?token=opaque") {
			t.Errorf("verification message missing subject or link: %+v", payload)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := NewBrevo("brevo-test-key", "no-reply@example.com", "PayConnect")
	sender.Endpoint = server.URL
	sender.Client = server.Client()
	if err := sender.SendVerification(context.Background(), "user@example.com", "User", "https://app.example.com/verify?token=opaque"); err != nil {
		t.Fatal(err)
	}
}

func TestSendVerificationRejectsMissingConfiguration(t *testing.T) {
	sender := NewBrevo("", "", "")

	err := sender.SendVerification(context.Background(), "user@example.com", "User", "https://app.example.com/verify?token=opaque")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("SendVerification() error = %v, want ErrNotConfigured", err)
	}
}

func TestSendVerificationReturnsBrevoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid API key", http.StatusUnauthorized)
	}))
	defer server.Close()

	sender := NewBrevo("brevo-test-key", "no-reply@example.com", "PayConnect")
	sender.Endpoint = server.URL
	sender.Client = server.Client()

	err := sender.SendVerification(context.Background(), "user@example.com", "User", "https://app.example.com/verify?token=opaque")
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("SendVerification() error = %v, want Brevo HTTP 401 error", err)
	}
}
