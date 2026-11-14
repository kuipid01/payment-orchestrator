package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
)

const (
	oauthCodeTTL       = 10 * time.Minute
	oauthCodeResendTTL = time.Minute
	oauthCodeAttempts  = 5
)

// RequestOAuthCode sends a one-time sign-in code to an existing PayConnect
// account. It deliberately never creates an account: the BizUp popup is an
// authorization screen, not a registration surface.
// The returned devCode is populated only when OTP_DEV_MODE is on, and is never
// emailed. Callers must not log or persist it.
func (s *Service) RequestOAuthCode(ctx context.Context, email string) (string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", "", apperrors.Validation(apperrors.CodeValidationError, "email is required.")
	}
	user, err := s.Merchants.UserByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", apperrors.NotFound("account_not_found", "No PayConnect account exists for that email.")
	}
	if err != nil {
		return "", "", err
	}
	if !user.IsActive() {
		return "", "", apperrors.PermissionDenied("This PayConnect account is not active.")
	}
	if s.Store == nil || (s.Mailer == nil && !s.DevOTP) {
		return "", "", apperrors.ServiceUnavailable("PayConnect email sign-in is not configured.", nil)
	}

	key := oauthCodeKey(email)
	if !s.DevOTP {
		first, err := s.Store.SetNX(ctx, "oauth:code:send:"+key, "1", oauthCodeResendTTL)
		if err != nil {
			return "", "", err
		}
		if !first {
			// A code was sent moments ago. Do not send another one; the first is
			// still valid and this response intentionally looks the same.
			return maskEmail(email), "", nil
		}
	}

	code, err := sixDigitCode()
	if err != nil {
		_ = s.Store.Del(ctx, "oauth:code:send:"+key)
		return "", "", err
	}
	if err := s.Store.Set(ctx, "oauth:code:value:"+key, s.hashOAuthCode(email, code), oauthCodeTTL); err != nil {
		_ = s.Store.Del(ctx, "oauth:code:send:"+key)
		return "", "", err
	}
	_ = s.Store.Del(ctx, "oauth:code:attempts:"+key)
	if s.DevOTP {
		return maskEmail(email), code, nil
	}
	if err := s.Mailer.SendLoginCode(ctx, email, user.Name, code); err != nil {
		_ = s.Store.Del(ctx, "oauth:code:value:"+key)
		_ = s.Store.Del(ctx, "oauth:code:send:"+key)
		return "", "", apperrors.ServiceUnavailable("The sign-in code email could not be sent.", err)
	}
	return maskEmail(email), "", nil
}

// VerifyOAuthCode consumes the one-time code and issues a normal PayConnect
// session. The popup then uses that session to create the BizUp application and
// key through the existing dashboard API.
func (s *Service) VerifyOAuthCode(ctx context.Context, email, code string) (*Tokens, *merchants.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)
	if !isSixDigits(code) {
		return nil, nil, apperrors.AuthenticationRequired().WithMessage("The sign-in code is invalid or expired.")
	}
	if s.Store == nil {
		return nil, nil, apperrors.ServiceUnavailable("PayConnect email sign-in is not configured.", nil)
	}
	key := oauthCodeKey(email)
	attempts, err := s.Store.Incr(ctx, "oauth:code:attempts:"+key, oauthCodeTTL)
	if err != nil {
		return nil, nil, err
	}
	if attempts > oauthCodeAttempts {
		_ = s.Store.Del(ctx, "oauth:code:value:"+key)
		return nil, nil, apperrors.AuthenticationRequired().WithMessage("Too many attempts. Request a new code.")
	}
	expected, ok, err := s.Store.Get(ctx, "oauth:code:value:"+key)
	if err != nil {
		return nil, nil, err
	}
	if !ok || subtle.ConstantTimeCompare([]byte(expected), []byte(s.hashOAuthCode(email, code))) != 1 {
		return nil, nil, apperrors.AuthenticationRequired().WithMessage("The sign-in code is invalid or expired.")
	}
	_ = s.Store.Del(ctx, "oauth:code:value:"+key)
	_ = s.Store.Del(ctx, "oauth:code:attempts:"+key)

	user, err := s.Merchants.UserByEmail(email)
	if err != nil || !user.IsActive() {
		return nil, nil, apperrors.AuthenticationRequired().WithMessage("This PayConnect account is not available.")
	}
	// A correct code proves control of the inbox, so it also settles email
	// verification. Accounts created earlier without clicking the verification
	// link must not be locked out of the connection flow.
	if !user.EmailVerified {
		if err := s.Merchants.MarkEmailVerified(user.ID); err != nil {
			return nil, nil, err
		}
		user.EmailVerified = true
	}
	membership, err := s.Merchants.DefaultUserMerchant(user.ID)
	if err != nil {
		return nil, nil, apperrors.PermissionDenied("This user has no active merchant membership.")
	}
	merchant, err := s.Merchants.MerchantByID(membership.MerchantID)
	if err != nil || !merchant.IsActive() {
		return nil, nil, apperrors.MerchantSuspended()
	}
	user.Role = membership.Role
	_ = s.Merchants.TouchLastActive(user.ID, time.Now().UTC())
	tokens, err := s.issue(user, merchant)
	if err != nil {
		return nil, nil, err
	}
	return tokens, user, nil
}

func (s *Service) hashOAuthCode(email, code string) string {
	secret := s.OTPSecret
	if secret == "" {
		secret = "payconnect-oauth-code-development"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.ToLower(strings.TrimSpace(email)) + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func oauthCodeKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

func sixDigitCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate sign-in code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func isSixDigits(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email
	}
	name := parts[0]
	if len(name) <= 2 {
		return name[:1] + "***@" + parts[1]
	}
	return name[:2] + "***@" + parts[1]
}
