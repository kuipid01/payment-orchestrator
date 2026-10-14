package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
	"github.com/kuipid01/payconnect/internal/platform/mailer"
	"github.com/kuipid01/payconnect/internal/platform/token"
)

// Service implements the auth use cases. It owns no persistence of its own: the
// merchant repository owns users and refresh tokens.
type Service struct {
	Merchants      *merchants.Repository
	Signer         *token.Signer
	Mailer         mailer.VerificationSender
	FrontendURL    string
	GoogleClientID string
	RefreshTTL     time.Duration
	APIKeys        APIKeyLookup
	Log            *slog.Logger
}

// NewService builds the auth service.
func NewService(repo *merchants.Repository, signer *token.Signer, log *slog.Logger) *Service {
	return &Service{Merchants: repo, Signer: signer, RefreshTTL: 30 * 24 * time.Hour, Log: log}
}

// RegisterInput is the dashboard sign-up payload.
type RegisterInput struct {
	Name         string
	Email        string
	Password     string
	BusinessName string
}

// Register creates a merchant and its owner user, then sends a verification mail.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*merchants.User, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.BusinessName = strings.TrimSpace(in.BusinessName)
	if in.Name == "" || in.Email == "" || len(in.Password) < 12 || in.BusinessName == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError,
			"name, email, organizationName and a 12+ character password are required.")
	}
	if _, err := s.Merchants.UserByEmail(in.Email); err == nil {
		return nil, apperrors.Conflict("email_already_registered", "That email is already registered.")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	passwordHash, err := identity.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	verificationToken, verificationHash, err := newOpaqueToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(emailVerificationTTL)

	merchant := &merchants.Merchant{
		ID:              newID("mch"),
		OwnerUserID:     "",
		BusinessName:    in.BusinessName,
		Email:           in.Email,
		Country:         "NG",
		Status:          "active",
		DefaultCurrency: "NGN",
	}
	user := &merchants.User{
		ID:                         newID("usr"),
		Name:                       in.Name,
		Email:                      in.Email,
		PasswordHash:               passwordHash,
		EmailVerified:              false,
		EmailVerificationTokenHash: &verificationHash,
		EmailVerificationExpiresAt: &expiresAt,
		Status:                     "active",
	}
	if err := s.Merchants.CreateMerchantAndUser(merchant, user); err != nil {
		return nil, err
	}
	if err := s.sendVerification(ctx, user, verificationToken); err != nil {
		return user, err
	}
	return user, nil
}

// Merchant returns the merchant an id refers to. It exists so the handler can
// hydrate /auth/me without reaching into the repository itself.
func (s *Service) Merchant(id string) (*merchants.Merchant, error) {
	return s.Merchants.MerchantByID(id)
}

// Login verifies credentials and issues a session.
func (s *Service) Login(email, password string) (*Tokens, *merchants.User, error) {
	user, err := s.Merchants.UserByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, nil, apperrors.InvalidCredentials()
	}
	if !identity.CheckPassword(user.PasswordHash, password) {
		return nil, nil, apperrors.InvalidCredentials()
	}
	if !user.IsActive() {
		return nil, nil, apperrors.PermissionDenied("This account has been revoked.")
	}
	if !user.EmailVerified {
		return nil, nil, apperrors.EmailNotVerified()
	}
	membership, err := s.Merchants.DefaultUserMerchant(user.ID)
	if err != nil {
		return nil, nil, apperrors.PermissionDenied("This user has no active merchant membership.")
	}
	merchant, err := s.Merchants.MerchantByID(membership.MerchantID)
	if err != nil {
		return nil, nil, err
	}
	if !merchant.IsActive() {
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

// MembershipView describes one organization the signed-in user can act in.
//
// `current` marks the organization the present session is acting as; the others
// are reachable through Switch. A user can be an Owner of one organization and a
// Developer in several others, which is why the role travels with each entry
// rather than being read once from the session.
type MembershipView struct {
	MerchantID string `json:"merchant_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Current    bool   `json:"current"`
	Owner      bool   `json:"owner"`
	JoinedAt   string `json:"joined_at"`
}

// Memberships lists the organizations a user can act in. A developer who has
// accepted invitations from several merchants sees all of them here; the session
// still acts in exactly one at a time.
func (s *Service) Memberships(userID, currentMerchantID string) ([]MembershipView, error) {
	rows, err := s.Merchants.MembershipsByUser(userID)
	if err != nil {
		return nil, err
	}
	views := make([]MembershipView, 0, len(rows))
	for _, row := range rows {
		views = append(views, MembershipView{
			MerchantID: row.MerchantID,
			Name:       row.Merchant,
			Role:       row.Role,
			Status:     row.Status,
			Current:    row.MerchantID == currentMerchantID,
			Owner:      row.Owner,
			JoinedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return views, nil
}

// IssueFor mints a session for a membership that already exists. It is the seam
// for flows that join a user to a merchant and then hand them a session — the
// invitation accept flow is the first. It re-derives the role from the
// membership rather than trusting a caller-supplied one, and refuses a merchant
// that is not active, so it cannot mint a session broader than Login would.
func (s *Service) IssueFor(userID, merchantID string) (*Tokens, *merchants.User, error) {
	user, err := s.Merchants.UserByID(userID)
	if err != nil || !user.IsActive() {
		return nil, nil, apperrors.AuthenticationRequired()
	}
	membership, err := s.Merchants.UserMerchantByUserAndMerchant(userID, merchantID)
	if err != nil {
		return nil, nil, apperrors.PermissionDenied("There is no active membership for that organization.")
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

// Refresh rotates a refresh token: the presented token is revoked and a new
// pair is issued. Presenting an already-revoked token revokes every session for
// that user (reuse detection).
func (s *Service) Refresh(refreshToken string) (*Tokens, *merchants.User, error) {
	hash := hashOpaque(refreshToken)
	record, err := s.Merchants.RefreshTokenByHash(hash)
	if err != nil {
		return nil, nil, apperrors.TokenInvalid("The refresh token is invalid or has expired.")
	}
	now := time.Now().UTC()
	if record.RevokedAt != nil {
		// Reuse of a rotated token: assume compromise and end every session.
		_ = s.Merchants.RevokeUserRefreshTokens(record.UserID, now)
		return nil, nil, apperrors.TokenInvalid("The refresh token is invalid or has expired.")
	}
	if record.ExpiresAt.Before(now) {
		return nil, nil, apperrors.TokenInvalid("The refresh token is invalid or has expired.")
	}
	user, err := s.Merchants.UserByID(record.UserID)
	if err != nil || !user.IsActive() {
		return nil, nil, apperrors.TokenInvalid("The refresh token is invalid or has expired.")
	}
	if err := s.Merchants.RevokeRefreshToken(hash, now); err != nil {
		return nil, nil, err
	}
	membership, err := s.Merchants.UserMerchantByUserAndMerchant(user.ID, record.MerchantID)
	if err != nil {
		return nil, nil, apperrors.TokenInvalid("The refresh token is invalid or has expired.")
	}
	merchant, err := s.Merchants.MerchantByID(membership.MerchantID)
	if err != nil || !merchant.IsActive() {
		return nil, nil, apperrors.TokenInvalid("The refresh token is invalid or has expired.")
	}
	user.Role = membership.Role
	tokens, err := s.issue(user, merchant)
	if err != nil {
		return nil, nil, err
	}
	return tokens, user, nil
}

// Logout revokes one refresh token.
func (s *Service) Logout(refreshToken string) {
	if refreshToken == "" {
		return
	}
	_ = s.Merchants.RevokeRefreshToken(hashOpaque(refreshToken), time.Now().UTC())
}

// LogoutAll revokes every session for a user.
func (s *Service) LogoutAll(userID string) {
	_ = s.Merchants.RevokeUserRefreshTokens(userID, time.Now().UTC())
}

// VerifyEmail consumes an email-verification token.
func (s *Service) VerifyEmail(token string) error {
	tokenHash := hashOpaque(token)
	user, err := s.Merchants.UserByVerificationToken(tokenHash, time.Now().UTC())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.TokenInvalid("The verification link is invalid or has expired.")
	}
	if err != nil {
		return err
	}
	if err := s.Merchants.VerifyEmail(user.ID, tokenHash, time.Now().UTC()); errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.TokenInvalid("The verification link is invalid or has expired.")
	} else if err != nil {
		return err
	}
	return nil
}

// ResendVerification issues a fresh verification link. The response never
// reveals whether the address exists.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	user, err := s.Merchants.UserByEmail(strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && user.EmailVerified) {
		return nil
	}
	if err != nil {
		return err
	}
	token, tokenHash, err := newOpaqueToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(emailVerificationTTL)
	if err := s.Merchants.SetEmailVerification(user.ID, tokenHash, expiresAt); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	return s.sendVerification(ctx, user, token)
}

// ForgotPassword issues a reset link. Like ResendVerification it is silent about
// account existence.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.Merchants.UserByEmail(strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if s.Mailer == nil {
		return nil
	}
	token, tokenHash, err := newOpaqueToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(passwordResetTTL)
	if err := s.Merchants.SetPasswordReset(user.ID, tokenHash, expiresAt); err != nil {
		return err
	}
	link := strings.TrimRight(s.FrontendURL, "/") + "/reset-password?token=" + token
	if err := s.Mailer.SendPasswordReset(ctx, user.Email, user.Name, link); err != nil {
		return apperrors.ServiceUnavailable("Password reset email could not be sent.", err)
	}
	return nil
}

// ResetPassword sets a new password, consumes the token and ends every session.
func (s *Service) ResetPassword(token, password string) (*merchants.User, error) {
	if len(password) < 12 {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "password must be at least 12 characters.")
	}
	tokenHash := hashOpaque(token)
	user, err := s.Merchants.UserByPasswordResetToken(tokenHash, time.Now().UTC())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.TokenInvalid("The reset link is invalid or has expired.")
	}
	if err != nil {
		return nil, err
	}
	passwordHash, err := identity.HashPassword(password)
	if err != nil {
		return nil, err
	}
	updated, err := s.Merchants.ResetPassword(user.ID, tokenHash, passwordHash, time.Now().UTC())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.TokenInvalid("The reset link is invalid or has expired.")
	}
	if err != nil {
		return nil, err
	}
	s.LogoutAll(user.ID)
	return updated, nil
}

// LoginWithGoogle verifies a Google ID token and signs the user in, creating an
// account on first use.
func (s *Service) LoginWithGoogle(ctx context.Context, credential string) (*Tokens, *merchants.User, error) {
	if s.GoogleClientID == "" {
		return nil, nil, apperrors.ServiceUnavailable("Google sign-in is not configured.", nil)
	}
	payload, err := idtoken.Validate(ctx, credential, s.GoogleClientID)
	if err != nil || payload.Subject == "" {
		return nil, nil, apperrors.AuthenticationRequired().WithMessage("The Google credential is invalid.")
	}
	email, _ := payload.Claims["email"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	if !verified || email == "" {
		return nil, nil, apperrors.AuthenticationRequired().WithMessage("The Google credential is invalid.")
	}
	email = strings.ToLower(email)
	name, _ := payload.Claims["name"].(string)
	if strings.TrimSpace(name) == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}

	if user, err := s.Merchants.UserByGoogleSub(payload.Subject); err == nil {
		membership, membershipErr := s.Merchants.DefaultUserMerchant(user.ID)
		if membershipErr != nil {
			return nil, nil, apperrors.PermissionDenied("This user has no active merchant membership.")
		}
		merchant, merchantErr := s.Merchants.MerchantByID(membership.MerchantID)
		if merchantErr != nil || !merchant.IsActive() {
			return nil, nil, apperrors.MerchantSuspended()
		}
		user.Role = membership.Role
		_ = s.Merchants.TouchLastActive(user.ID, time.Now().UTC())
		tokens, err := s.issue(user, merchant)
		return tokens, user, err
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	if user, err := s.Merchants.UserByEmail(email); err == nil {
		if user.GoogleSub != nil && *user.GoogleSub != payload.Subject {
			return nil, nil, apperrors.AuthenticationRequired().WithMessage("This Google identity is linked to another account.")
		}
		membership, membershipErr := s.Merchants.DefaultUserMerchant(user.ID)
		if membershipErr != nil {
			return nil, nil, apperrors.PermissionDenied("This user has no active merchant membership.")
		}
		merchant, merchantErr := s.Merchants.MerchantByID(membership.MerchantID)
		if merchantErr != nil || !merchant.IsActive() {
			return nil, nil, apperrors.MerchantSuspended()
		}
		if err := s.Merchants.LinkGoogleSub(user.ID, payload.Subject); err != nil {
			return nil, nil, err
		}
		user.GoogleSub = &payload.Subject
		user.EmailVerified = true
		user.Role = membership.Role
		tokens, err := s.issue(user, merchant)
		return tokens, user, err
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	merchant := &merchants.Merchant{
		ID:              newID("mch"),
		BusinessName:    name + "'s business",
		Email:           email,
		Country:         "NG",
		Status:          "active",
		DefaultCurrency: "NGN",
	}
	user := &merchants.User{
		ID:            newID("usr"),
		Name:          name,
		Email:         email,
		GoogleSub:     &payload.Subject,
		EmailVerified: true,
		Status:        "active",
	}
	if err := s.Merchants.CreateMerchantAndUser(merchant, user); err != nil {
		return nil, nil, err
	}
	user.Role = "Owner"
	tokens, err := s.issue(user, merchant)
	return tokens, user, err
}

// issue mints an access token and a stored refresh token.
func (s *Service) issue(user *merchants.User, merchant *merchants.Merchant) (*Tokens, error) {
	if s.Signer == nil {
		return nil, apperrors.ServiceUnavailable("Session signing is not configured.", nil)
	}
	accessToken, expiresAt, err := s.Signer.Issue(user.ID, merchant.ID)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshHash, err := newOpaqueToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.Merchants.CreateRefreshToken(merchant.ID, user.ID, refreshHash, now.Add(s.RefreshTTL), now); err != nil {
		return nil, err
	}
	return &Tokens{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken,
		MerchantID:   merchant.ID,
		Role:         user.Role,
	}, nil
}

func (s *Service) sendVerification(ctx context.Context, user *merchants.User, token string) error {
	if s.Mailer == nil {
		return apperrors.ServiceUnavailable("Email verification is not configured.", nil)
	}
	link := strings.TrimRight(s.FrontendURL, "/") + "/verify-email?token=" + token
	if err := s.Mailer.SendVerification(ctx, user.Email, user.Name, link); err != nil {
		return apperrors.ServiceUnavailable("The verification email could not be sent.", err)
	}
	return nil
}

func newOpaqueToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	return value, hashOpaque(value), nil
}

// hashOpaque hashes a session or email token before storage.
func hashOpaque(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func newID(prefix string) string { return fmt.Sprintf("%s_%s", prefix, randomHex(6)) }

func randomHex(n int) string {
	raw := make([]byte, n)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}
