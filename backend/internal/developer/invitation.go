package developer

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// minimumPasswordLength matches the dashboard sign-up rule so an invitation
// cannot create a weaker credential than registration would.
const minimumPasswordLength = 12

// Session is the credential pair handed to a developer who has just joined.
// It is deliberately a local type: the developer package must not depend on the
// auth package just to describe a token.
type Session struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	MerchantID   string
	Role         string
}

// SessionIssuer mints a session for a membership that already exists. The auth
// service implements it behind an adapter; declaring the port here keeps the two
// packages independent and lets tests substitute a fake.
type SessionIssuer interface {
	IssueFor(userID, merchantID string) (*Session, error)
}

// InvitationPreview is everything the accept screen needs before the invitee
// types anything. It never exposes a user id, a credential hash, or anything
// about other invitations.
type InvitationPreview struct {
	Email        string   `json:"email"`
	MerchantID   string   `json:"merchant_id"`
	MerchantName string   `json:"merchant_name"`
	Permissions  []string `json:"permissions"`
	Status       string   `json:"status"`
	ExpiresAt    string   `json:"expires_at,omitempty"`
	// HasIdentity reports whether a login exists for this address, which decides
	// whether the screen asks for a password or asks to create one. It is a
	// different question from IsDeveloper and must not be conflated with it.
	HasIdentity bool `json:"has_identity"`
	// IsDeveloper reports whether the address already has a developer profile,
	// i.e. this is a returning developer joining one more merchant.
	IsDeveloper     bool   `json:"is_developer"`
	DeveloperStatus string `json:"developer_status,omitempty"`
}

// AcceptInput is the invitee's submission.
type AcceptInput struct {
	Token    string
	Name     string
	Password string
}

// AcceptResult reports where the developer landed.
type AcceptResult struct {
	MerchantID   string
	MerchantName string
	Role         string
	Session      *Session
}

// Preview resolves an invitation token to the context the accept screen renders.
// It creates nothing and mutates nothing: an expired, revoked or already-used
// token is refused before any lookup of the invitee.
func (s *Service) Preview(token string) (*InvitationPreview, error) {
	access, err := s.pendingAccess(token)
	if err != nil {
		return nil, err
	}

	preview := &InvitationPreview{
		Email:       access.InviteeEmail,
		MerchantID:  access.MerchantID,
		Permissions: access.Permissions.Data,
		Status:      access.Status,
		HasIdentity: false,
		IsDeveloper: false,
	}
	if preview.Permissions == nil {
		preview.Permissions = []string{}
	}
	if access.ExpiresAt != nil {
		preview.ExpiresAt = access.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if merchant, mErr := s.Repo.MerchantByID(access.MerchantID); mErr == nil {
		preview.MerchantName = merchant.BusinessName
	}

	// The identity question and the developer question are separate. Only an
	// identity carries a password; only a developer profile means we have seen
	// this person develop before.
	user, err := s.Repo.UserByEmail(strings.ToLower(strings.TrimSpace(access.InviteeEmail)))
	if err != nil {
		return preview, nil
	}
	preview.HasIdentity = true
	if dev, devErr := s.Repo.DeveloperByUser(user.ID); devErr == nil {
		preview.IsDeveloper = true
		preview.DeveloperStatus = dev.Status
	}
	return preview, nil
}

// AcceptInvitation joins the invitee to the inviting merchant and returns the
// session they continue with.
//
// The token alone is not sufficient: the caller must also prove the identity by
// presenting the existing password, or by creating one. That is why this
// endpoint is unauthenticated yet still safe — the token is single-purpose and
// single-use, and the credential is verified with the same constant-time check
// as login.
func (s *Service) AcceptInvitation(ctx context.Context, in AcceptInput) (*AcceptResult, error) {
	if s.Sessions == nil {
		return nil, apperrors.ServiceUnavailable("Session issuance is not configured.", nil)
	}
	access, err := s.pendingAccess(in.Token)
	if err != nil {
		return nil, err
	}
	email := strings.ToLower(strings.TrimSpace(access.InviteeEmail))
	if email == "" {
		return nil, apperrors.TokenInvalid("The invitation link is invalid or has expired.")
	}
	now := time.Now().UTC()

	user, isNewUser, err := s.resolveIdentity(email, in, now)
	if err != nil {
		return nil, err
	}

	// One transaction: identity (when new) + developer profile + membership +
	// consuming the invitation. A failure at any step must leave nothing behind,
	// which is why the write belongs to the repository.
	developer := &merchants.Developer{
		ID:        newDeveloperID(),
		UserID:    user.ID,
		Name:      user.Name,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := s.Repo.AcceptDeveloperInvitation(merchants.AcceptDeveloperInvitationInput{
		User:         user,
		IsNewUser:    isNewUser,
		Developer:    developer,
		MerchantID:   access.MerchantID,
		InvitationID: access.ID,
		Now:          now,
	}); err != nil {
		return nil, invitationError(err)
	}

	// The session is minted only now, so it can never exist without the
	// membership that justifies it.
	session, err := s.Sessions.IssueFor(user.ID, access.MerchantID)
	if err != nil {
		return nil, err
	}

	result := &AcceptResult{
		MerchantID: access.MerchantID,
		Role:       session.Role,
		Session:    session,
	}
	if merchant, mErr := s.Repo.MerchantByID(access.MerchantID); mErr == nil {
		result.MerchantName = merchant.BusinessName
	}
	return result, nil
}

// resolveIdentity finds the login for the invited address, verifying the supplied
// password when one exists, or builds a new verified identity from the invitation.
func (s *Service) resolveIdentity(email string, in AcceptInput, now time.Time) (*merchants.User, bool, error) {
	user, err := s.Repo.UserByEmail(email)
	switch {
	case err == nil:
		if !identity.CheckPassword(user.PasswordHash, in.Password) {
			// Deliberately the same error as Login: it must not reveal which half
			// of the credential pair was wrong.
			return nil, false, apperrors.InvalidCredentials()
		}
		if !user.IsActive() {
			return nil, false, apperrors.PermissionDenied("This account has been revoked.")
		}
		return user, false, nil

	case errors.Is(err, gorm.ErrRecordNotFound):
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, false, apperrors.Validation(apperrors.CodeValidationError, "name is required.")
		}
		if len(in.Password) < minimumPasswordLength {
			return nil, false, apperrors.Validation(apperrors.CodeValidationError,
				"password must be at least 12 characters.")
		}
		hash, hashErr := identity.HashPassword(in.Password)
		if hashErr != nil {
			return nil, false, hashErr
		}
		return &merchants.User{
			ID:           newUserID(),
			Name:         name,
			Email:        email,
			Status:       "active",
			PasswordHash: hash,
			// Holding a valid invitation token proves control of the mailbox it
			// was sent to, so the address starts verified. This is why the token
			// must stay short-lived and single-use.
			EmailVerified: true,
			CreatedAt:     now,
		}, true, nil

	default:
		return nil, false, err
	}
}

// pendingAccess resolves a token to a usable invitation.
func (s *Service) pendingAccess(token string) (*merchants.DeveloperMerchantAccess, error) {
	hash, err := invitationHash(token)
	if err != nil {
		return nil, apperrors.TokenInvalid("The invitation link is invalid or has expired.")
	}
	access, err := s.Repo.DeveloperMerchantAccessByTokenHash(hash)
	if err != nil {
		return nil, apperrors.TokenInvalid("The invitation link is invalid or has expired.")
	}
	if access.Status != "pending" {
		return nil, apperrors.TokenInvalid("This invitation has already been used.")
	}
	if access.ExpiresAt != nil && access.ExpiresAt.Before(time.Now().UTC()) {
		return nil, apperrors.TokenInvalid("The invitation link has expired.")
	}
	return access, nil
}

// invitationError maps the repository's sentinels onto the API vocabulary.
func invitationError(err error) error {
	switch {
	case errors.Is(err, merchants.ErrDeveloperRevoked):
		return apperrors.PermissionDenied("This developer account has been revoked.")
	case errors.Is(err, merchants.ErrInvitationUnavailable):
		return apperrors.TokenInvalid("This invitation is no longer available.")
	case isDuplicateEmail(err):
		// Two concurrent accepts for the same new address: the loser is told to
		// sign in with the account the winner created.
		return apperrors.Conflict("email_already_registered",
			"That email was just registered. Use the sign-in option instead.")
	default:
		return err
	}
}

// isDuplicateEmail reports a unique-index violation on users.email.
func isDuplicateEmail(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "users_email_key") || strings.Contains(message, "duplicate key")
}

// newUserID mirrors the auth package's identity prefix.
func newUserID() string { return "usr_" + randomID() }
