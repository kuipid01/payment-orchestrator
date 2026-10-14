package application

import (
	"testing"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

func TestValidateInput(t *testing.T) {
	tests := []struct {
		name  string
		input Input
		err   string
	}{
		{
			name: "valid",
			input: Input{
				Name:        "Checkout",
				Description: "Storefront payments",
				Status:      StatusActive,
			},
		},
		{
			name: "missing name",
			input: Input{
				Status: StatusActive,
			},
			err: "name is required",
		},
		{
			name: "invalid status",
			input: Input{
				Name:   "Checkout",
				Status: "archived",
			},
			err: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInput(tt.input)
			if tt.err == "" {
				if err != nil {
					t.Fatalf("validateInput() error = %v", err)
				}
				return
			}
			if err == nil || !contains(err.Error(), tt.err) {
				t.Fatalf("validateInput() error = %v, want error containing %q", err, tt.err)
			}
			if !apperrors.Is(err, apperrors.CodeValidationError) {
				t.Fatalf("validateInput() error code = %v, want validation error", err)
			}
		})
	}
}

func contains(s, substring string) bool {
	return len(substring) <= len(s) && (s == substring || len(s) > len(substring) && containsAt(s, substring))
}

func containsAt(s, substring string) bool {
	for i := 0; i+len(substring) <= len(s); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
