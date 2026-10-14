// Package auth owns registration, sign-in and session lifecycle for the
// dashboard, and implements the bearer-token authenticator used by every
// authenticated route.
package auth

import "time"

// Credential and session lifetimes.
const (
	emailVerificationTTL = time.Hour
	passwordResetTTL     = 30 * time.Minute
)

// Tokens is the credential pair returned by sign-in and refresh.
type Tokens struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	RefreshToken string    `json:"refresh_token"`
	MerchantID   string    `json:"-"`
	Role         string    `json:"-"`
}
