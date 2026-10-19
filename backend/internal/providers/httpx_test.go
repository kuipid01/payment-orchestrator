package providers

import (
	"errors"
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
