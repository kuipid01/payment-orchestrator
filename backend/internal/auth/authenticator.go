package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/identity"
	"github.com/kuipid01/payconnect/internal/platform/token"
)

// APIKeyLookup resolves a developer API key into a principal. The api_keys
// module implements it; auth only depends on this interface.
type APIKeyLookup interface {
	Authenticate(ctx context.Context, rawKey string) (*identity.Principal, error)
}

// Authenticate implements middleware.Authenticator: it resolves either a
// dashboard access token or a developer API key.
func (s *Service) Authenticate(ctx context.Context, raw string) (*identity.Principal, error) {
	if looksLikeAPIKey(raw) {
		if s.APIKeys == nil {
			return nil, apperrors.InvalidAPIKey()
		}
		return s.APIKeys.Authenticate(ctx, raw)
	}
	return s.authenticateSession(raw)
}

// authenticateSession verifies the access token and loads the current user and
// merchant, so a revoked or suspended account cannot keep using a live token.
func (s *Service) authenticateSession(raw string) (*identity.Principal, error) {
	if s.Signer == nil {
		return nil, apperrors.ServiceUnavailable("Session signing is not configured.", nil)
	}
	claims, err := s.Signer.Verify(raw)
	switch {
	case errors.Is(err, token.ErrExpiredToken):
		return nil, apperrors.TokenInvalid("Your session has expired. Sign in again.")
	case err != nil:
		return nil, apperrors.AuthenticationRequired()
	}
	user, err := s.Merchants.UserByID(claims.Sub)
	if err != nil {
		return nil, apperrors.AuthenticationRequired()
	}
	if !user.IsActive() {
		return nil, apperrors.PermissionDenied("This account has been revoked.")
	}
	membership, err := s.Merchants.UserMerchantByUserAndMerchant(user.ID, claims.Org)
	if err != nil {
		return nil, apperrors.AuthenticationRequired()
	}
	merchant, err := s.Merchants.MerchantByID(membership.MerchantID)
	if err != nil {
		return nil, apperrors.AuthenticationRequired()
	}
	if !merchant.IsActive() {
		return nil, apperrors.MerchantSuspended()
	}
	user.Role = membership.Role
	_ = s.Merchants.TouchLastActive(user.ID, time.Now().UTC())
	return &identity.Principal{
		Mode:       identity.ModeSession,
		MerchantID: merchant.ID,
		OrgID:      merchant.ID,
		UserID:     user.ID,
		User:       user,
	}, nil
}

// looksLikeAPIKey reports whether the bearer token is a PayConnect API key
// rather than a signed session token.
func looksLikeAPIKey(raw string) bool {
	return strings.HasPrefix(raw, "pc_test_") || strings.HasPrefix(raw, "pc_live_") ||
		strings.HasPrefix(raw, "vpay_test_") || strings.HasPrefix(raw, "vpay_live_")
}
