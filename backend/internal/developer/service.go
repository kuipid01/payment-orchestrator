package developer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/api_keys"
	"github.com/kuipid01/payconnect/internal/application"
	"github.com/kuipid01/payconnect/internal/billing"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/platform/mailer"
)

type Service struct {
	Repo    *merchants.Repository
	Mailer  mailer.VerificationSender
	BaseURL string
	// Apps and Keys let the developer platform show, and revoke, the resources
	// a developer grant actually reaches. Both are optional so the service stays
	// usable in the invitation-only tests.
	Apps *application.Repository
	Keys *api_keys.Service
	// Webhooks sends the developer platform's sample deliveries. Optional, so
	// the invitation-only tests can build the service without a dispatcher.
	Webhooks WebhookTester
	Sessions SessionIssuer
	// Gates answers the one billing question this package asks: does the
	// organization's plan have room for another seat? Optional, and a nil gate
	// means no limit (tests, and any build wired without billing).
	Gates PlanGates
}

// PlanGates is the seat half of the plan gate. It answers with a Decision and
// only Refusal() turns that into an error, which is what keeps seat enforcement
// log-only until it is deliberately switched on.
type PlanGates interface {
	CheckSeats(ctx context.Context, merchantID string) billing.Decision
}

type InviteInput struct {
	MerchantID  string
	Email       string
	Name        string
	Permissions []string
	InvitedBy   string
}

type InvitationResponse struct {
	ID           string   `json:"id"`
	MerchantID   string   `json:"merchant_id"`
	InviteeEmail string   `json:"invitee_email"`
	Status       string   `json:"status"`
	Permissions  []string `json:"permissions"`
	InvitedAt    string   `json:"invited_at"`
}

type DeveloperResponse struct {
	ID string `json:"id"`
	// AccessID names the developer grant. Keys are scoped to it, so it is the
	// handle the dashboard uses when it issues a developer-specific key.
	AccessID string `json:"access_id"`
	// DeveloperID is empty until the invitation is accepted.
	DeveloperID    string   `json:"developer_id,omitempty"`
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	Status         string   `json:"status"`
	ApplicationIDs []string `json:"application_ids"`
	Permissions    []string `json:"permissions"`
	InvitedAt      string   `json:"invited_at"`
	LastActiveAt   string   `json:"last_active_at,omitempty"`
}

func NewService(repo *merchants.Repository, sender mailer.VerificationSender, baseURL string, sessions SessionIssuer) *Service {
	return &Service{Repo: repo, Mailer: sender, BaseURL: strings.TrimRight(baseURL, "/"), Sessions: sessions}
}

func (s *Service) List(merchantID, actorID, actorRole string) ([]DeveloperResponse, error) {
	if merchantID == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "merchantId is required.")
	}
	active, err := s.Repo.DeveloperMerchantAccessesByMerchantAndStatus(merchantID, "active")
	if err != nil {
		return nil, err
	}
	pending, err := s.Repo.DeveloperMerchantAccessesByMerchantAndStatus(merchantID, "pending")
	if err != nil {
		return nil, err
	}
	accesses := append(active, pending...)
	if actorRole != "Owner" && actorRole != "Admin" {
		visible := accesses[:0]
		for _, access := range accesses {
			if access.InvitedBy == actorID {
				visible = append(visible, access)
			}
		}
		accesses = visible
	}
	applicationIDs := s.merchantApplicationIDs(merchantID)
	developers := make([]DeveloperResponse, 0, len(accesses))
	for _, access := range accesses {
		name := access.InviteeEmail
		email := access.InviteeEmail
		id := access.ID
		developerID := ""
		if access.Status == "active" {
			if access.DeveloperID == nil {
				continue
			}
			developer, err := s.Repo.DeveloperByID(*access.DeveloperID)
			if err != nil {
				continue
			}
			user, err := s.Repo.UserByID(developer.UserID)
			if err != nil {
				continue
			}
			id = developer.ID
			developerID = developer.ID
			name = user.Name
			email = user.Email
		}
		developers = append(developers, DeveloperResponse{
			ID:             id,
			AccessID:       access.ID,
			DeveloperID:    developerID,
			Name:           name,
			Email:          email,
			Status:         access.Status,
			ApplicationIDs: applicationIDs,
			Permissions:    access.Permissions.Data,
			InvitedAt:      access.CreatedAt.Format(time.RFC3339),
		})
	}
	return developers, nil
}

// merchantApplicationIDs lists the merchant's live application ids, so a
// developer row can state what its grant actually covers.
func (s *Service) merchantApplicationIDs(merchantID string) []string {
	ids := []string{}
	if s.Apps == nil {
		return ids
	}
	apps, err := s.Apps.List(application.ListFilter{MerchantID: merchantID, Limit: 100})
	if err != nil {
		return ids
	}
	for i := range apps {
		ids = append(ids, apps[i].ID)
	}
	return ids
}

func (s *Service) Invite(ctx context.Context, in InviteInput) (*InvitationResponse, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	if in.MerchantID == "" || in.Email == "" || in.InvitedBy == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "merchantId, email and inviter are required.")
	}
	if !validEmail(in.Email) {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "email is invalid.")
	}
	if len(in.Permissions) == 0 {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "at least one permission is required.")
	}

	if existing, err := s.Repo.DeveloperMerchantAccessByEmail(in.MerchantID, in.Email); err == nil {
		if existing.Status == "pending" {
			return nil, apperrors.Conflict("invitation_already_exists", "A pending invitation already exists for this email.")
		}
		return nil, apperrors.Conflict("invitation_already_exists", "This developer already has access to the merchant.")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Seats are gated after the duplicate check, so inviting somebody who is
	// already on the team is reported as such rather than as a plan limit.
	if err := s.checkSeat(ctx, in.MerchantID); err != nil {
		return nil, err
	}

	token, hash, err := newInvitationToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour)
	access := &merchants.DeveloperMerchantAccess{
		ID:                  newInvitationID(),
		MerchantID:          in.MerchantID,
		InviteeEmail:        in.Email,
		Status:              "pending",
		Permissions:         gormx.JSONB[[]string]{Data: in.Permissions},
		InvitedBy:           in.InvitedBy,
		InvitationTokenHash: hash,
		ExpiresAt:           &expiresAt,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.Repo.CreateDeveloperMerchantAccess(access); err != nil {
		return nil, err
	}
	if s.Mailer == nil {
		return nil, apperrors.ServiceUnavailable("Developer invitation email is not configured.", nil)
	}
	link := s.BaseURL + "/developers/accept?token=" + token
	if err := s.Mailer.SendDeveloperInvitation(ctx, in.Email, in.Name, link); err != nil {
		_ = s.Repo.RevokeDeveloperMerchantAccess(access.ID)
		return nil, apperrors.ServiceUnavailable("Developer invitation email could not be sent.", err)
	}
	return invitationResponse(access), nil
}

// checkSeat asks the plan gate, when one is wired. Pending invitations count as
// seats, so this refuses the crowd that would otherwise be invited onto a plan
// that includes one of them.
func (s *Service) checkSeat(ctx context.Context, merchantID string) error {
	if s.Gates == nil {
		return nil
	}
	return s.Gates.CheckSeats(ctx, merchantID).Refusal()
}

func (s *Service) Accept(ctx context.Context, token, userID string) error {
	hash, err := invitationHash(token)
	if err != nil {
		return apperrors.TokenInvalid("The invitation link is invalid or has expired.")
	}
	access, err := s.Repo.DeveloperMerchantAccessByTokenHash(hash)
	if err != nil {
		return apperrors.TokenInvalid("The invitation link is invalid or has expired.")
	}
	if access.Status != "pending" {
		return apperrors.TokenInvalid("The invitation has already been processed.")
	}
	if access.ExpiresAt != nil && access.ExpiresAt.Before(time.Now().UTC()) {
		return apperrors.TokenInvalid("The invitation link has expired.")
	}
	user, err := s.Repo.UserByID(userID)
	if err != nil {
		return apperrors.AuthenticationRequired()
	}
	if !strings.EqualFold(access.InviteeEmail, user.Email) {
		return apperrors.PermissionDenied("This invitation was sent to a different email address.")
	}

	now := time.Now().UTC()
	return s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		developer := &merchants.Developer{ID: newDeveloperID(), UserID: userID, Name: user.Name, Status: "active", CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(developer).Error; err != nil {
			return err
		}
		if err := tx.Create(&merchants.UserMerchant{ID: newMembershipID(), UserID: userID, MerchantID: access.MerchantID, Role: "Developer", Status: "active", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			return err
		}
		return tx.Model(&merchants.DeveloperMerchantAccess{}).Where("id = ?", access.ID).Updates(map[string]any{"developer_id": developer.ID, "status": "active", "accepted_at": now, "updated_at": now}).Error
	})
}

func (s *Service) Revoke(merchantID, accessID, userID string) error {
	access, err := s.Repo.DeveloperMerchantAccessByIDOrDeveloper(merchantID, accessID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NotFound(apperrors.CodeNotFound, "Developer access not found.")
		}
		return err
	}
	if access.InvitedBy != userID {
		return apperrors.PermissionDenied("You cannot remove this developer.")
	}
	now := time.Now().UTC()
	if err := s.Repo.RemoveDeveloperMerchantAccess(merchantID, access.ID, now); err != nil {
		return err
	}
	// Removing the developer also revokes the keys issued to them. The
	// application's shared keys are untouched, so the integration keeps working.
	if s.Keys != nil {
		if err := s.Keys.RevokeByDeveloperAccess(merchantID, access.ID, now); err != nil {
			return err
		}
	}
	return nil
}

func invitationResponse(access *merchants.DeveloperMerchantAccess) *InvitationResponse {
	return &InvitationResponse{ID: access.ID, MerchantID: access.MerchantID, InviteeEmail: access.InviteeEmail, Status: access.Status, Permissions: access.Permissions.Data, InvitedAt: access.CreatedAt.Format(time.RFC3339)}
}

func newInvitationToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b)
	return token, hashToken(token), nil
}

func newInvitationID() string { return "inv_" + randomID() }
func newDeveloperID() string  { return "dev_" + randomID() }
func newMembershipID() string { return "um_" + randomID() }
func randomID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func invitationHash(token string) (string, error) {
	if token == "" {
		return "", errors.New("empty token")
	}
	return hashToken(token), nil
}
func validEmail(email string) bool {
	return strings.Contains(email, "@") && !strings.ContainsAny(email, " \t\r\n")
}
