package providers

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

func TestMapStatusIncludesBoundedProviderRejectionDetails(t *testing.T) {
	message := "The selected currency is not available."
	err := MapStatus("flutterwave", 400, map[string]any{"message": message})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("MapStatus() error = %T, want *AppError", err)
	}
	if appErr.Code != apperrors.CodeProviderRequestFailed {
		t.Fatalf("error code = %q, want %q", appErr.Code, apperrors.CodeProviderRequestFailed)
	}
	details, ok := appErr.Details.(map[string]any)
	if !ok {
		t.Fatalf("error details = %#v, want map", appErr.Details)
	}
	if details["provider_status"] != 400 {
		t.Fatalf("provider_status = %#v, want 400", details["provider_status"])
	}
	if details["provider_message"] != message {
		t.Fatalf("provider_message = %#v, want %q", details["provider_message"], message)
	}
}

func TestMapStatusUsesMonnifyResponseMessage(t *testing.T) {
	message := "Invalid API key or secret key"
	err := MapStatus("monnify", http.StatusUnauthorized, map[string]any{"responseMessage": message})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("MapStatus() error = %T, want *AppError", err)
	}
	if !strings.Contains(appErr.Error(), message) {
		t.Fatalf("MapStatus() error = %q, want provider response message", appErr.Error())
	}
	details, ok := appErr.Details.(map[string]any)
	if !ok || details["provider_status"] != http.StatusUnauthorized || details["provider_message"] != message {
		t.Fatalf("MapStatus() details = %#v, want bounded Monnify response", appErr.Details)
	}
}

func TestMapStatusBoundsProviderRejectionDetails(t *testing.T) {
	message := string(make([]rune, 300))
	err := MapStatus("flutterwave", 422, map[string]any{"message": message})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("MapStatus() error = %T, want *AppError", err)
	}
	details, ok := appErr.Details.(map[string]any)
	if !ok {
		t.Fatalf("error details = %#v, want map", appErr.Details)
	}
	got, ok := details["provider_message"].(string)
	if !ok || len([]rune(got)) > 256 {
		t.Fatalf("provider_message length = %d, want at most 256 runes", len([]rune(got)))
	}
}

func TestMapStatusClassifiesTLSConnectionFailureAsUnavailable(t *testing.T) {
	message := "Client network socket disconnected before secure TLS connection was established"
	assertProviderUnavailableWithDetails(t, message)
}

func TestMapStatusClassifiesProviderRuntimeFailureAsUnavailable(t *testing.T) {
	message := "Cannot read properties of undefined (reading 'filter')"
	assertProviderUnavailableWithDetails(t, message)
}

func assertProviderUnavailableWithDetails(t *testing.T, message string) {
	t.Helper()
	err := MapStatus("flutterwave", 400, map[string]any{"message": message})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("MapStatus() error = %T, want *AppError", err)
	}
	if appErr.Code != apperrors.CodeProviderUnavailable {
		t.Fatalf("error code = %q, want %q", appErr.Code, apperrors.CodeProviderUnavailable)
	}
	details, ok := appErr.Details.(map[string]any)
	if !ok {
		t.Fatalf("error details = %#v, want map", appErr.Details)
	}
	if details["provider_status"] != 400 || details["provider_message"] != message {
		t.Fatalf("error details = %#v, want original provider status and message", details)
	}
}
