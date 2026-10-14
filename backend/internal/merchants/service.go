package merchants

import (
	"errors"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/audit"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"gorm.io/gorm"
)

// Service owns merchant profile rules.
type Service struct {
	Repo  *Repository
	Audit audit.Recorder
}

// NewService builds the merchant service.
func NewService(repo *Repository) *Service { return &Service{Repo: repo} }

// Get returns a merchant profile.
func (s *Service) Get(merchantID string) (*Merchant, error) {
	m, err := s.Repo.MerchantByID(merchantID)
	if err != nil {
		return nil, apperrors.NotFound("", "Merchant not found.")
	}
	return m, nil
}

// UpdateInput carries the mutable merchant profile fields. Nil means "unchanged".
type UpdateInput struct {
	BusinessName    *string
	Country         *string
	DefaultCurrency *string
	ActorName       string
	ActorRole       string
}

// Update applies a partial profile update.
func (s *Service) Update(merchantID string, in UpdateInput) (*Merchant, error) {
	m, err := s.Repo.MerchantByID(merchantID)
	if err != nil {
		return nil, apperrors.NotFound("", "Merchant not found.")
	}
	changed := false
	if in.BusinessName != nil {
		name := strings.TrimSpace(*in.BusinessName)
		if name == "" {
			return nil, apperrors.Validation(apperrors.CodeValidationError, "business_name must not be empty.")
		}
		changed = changed || m.BusinessName != name
		m.BusinessName = name
	}
	if in.Country != nil {
		country := strings.ToUpper(strings.TrimSpace(*in.Country))
		if len(country) != 2 {
			return nil, apperrors.Validation(apperrors.CodeValidationError, "country must be an ISO 3166-1 alpha-2 code.")
		}
		changed = changed || m.Country != country
		m.Country = country
	}
	if in.DefaultCurrency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*in.DefaultCurrency))
		if !isSupportedCurrency(currency) {
			return nil, apperrors.InvalidCurrency("default_currency is not supported.")
		}
		changed = changed || m.DefaultCurrency != currency
		m.DefaultCurrency = currency
	}
	if err := s.Repo.UpdateMerchant(m); err != nil {
		return nil, err
	}
	if changed && s.Audit != nil {
		s.Audit.Record(audit.Entry{
			MerchantID: merchantID, ActorName: in.ActorName, ActorRole: in.ActorRole,
			Action: "Organization settings updated", Detail: "Workspace settings were updated.",
		})
	}
	return m, nil
}

// TeamMember is the public read model for a merchant's active account members.
type TeamMember struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	LastActiveAt *time.Time `json:"last_active_at"`
}

// Team lists persisted user memberships for one merchant.
func (s *Service) Team(merchantID string) ([]TeamMember, error) {
	rows, err := s.Repo.TeamByMerchant(merchantID)
	if err != nil {
		return nil, err
	}
	members := make([]TeamMember, 0, len(rows))
	for _, row := range rows {
		members = append(members, TeamMember{
			ID: row.MembershipID, Name: row.Name, Email: row.Email,
			Role: row.Role, Status: row.Status, LastActiveAt: row.LastActiveAt,
		})
	}
	return members, nil
}

// AddTeamMember adds an existing PayConnect account to a merchant organization.
// New account registration remains a separate, verified-email flow.
func (s *Service) AddTeamMember(merchantID, email, role, actorName, actorRole string) (*TeamMember, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.TrimSpace(role)
	if email == "" || !strings.Contains(email, "@") || strings.ContainsAny(email, " \t\r\n") {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "A valid email address is required.")
	}
	switch role {
	case "Admin", "Developer", "Viewer":
	default:
		return nil, apperrors.Validation(apperrors.CodeValidationError, "Role must be Admin, Developer or Viewer.")
	}
	user, err := s.Repo.UserByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NotFound("user_not_found", "This email is not registered yet. Ask the person to create a PayConnect account first.")
	}
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, apperrors.PermissionDenied("This account is not active.")
	}
	if !user.EmailVerified {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "The account email must be verified before it can be added to a workspace.")
	}
	now := time.Now().UTC()
	if err := s.Repo.AddTeamMember(merchantID, user.ID, role, now); err != nil {
		if errors.Is(err, ErrTeamMemberExists) {
			return nil, apperrors.Conflict("team_member_exists", "This user is already a member of the organization.")
		}
		return nil, err
	}
	rows, err := s.Repo.TeamByMerchant(merchantID)
	if err != nil {
		return nil, err
	}
	membershipID := ""
	for _, row := range rows {
		if row.UserID == user.ID {
			membershipID = row.MembershipID
			break
		}
	}
	if membershipID == "" {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "The new team membership could not be loaded.")
	}
	if s.Audit != nil {
		s.Audit.Record(audit.Entry{
			MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
			Action: "Team member added", Detail: email + " was added as " + role + ".",
		})
	}
	return &TeamMember{
		ID: membershipID, Name: user.Name, Email: user.Email, Role: role,
		Status: "active", LastActiveAt: user.LastActiveAt,
	}, nil
}

// RemoveTeamMember revokes one membership in the selected organization.
func (s *Service) RemoveTeamMember(merchantID, membershipID, actorUserID, actorName, actorRole string) error {
	if err := s.Repo.RevokeTeamMember(merchantID, membershipID, actorUserID, time.Now().UTC()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NotFound("team_member_not_found", "The team member was not found, is the organization owner, or is your own account.")
		}
		return err
	}
	if s.Audit != nil {
		s.Audit.Record(audit.Entry{
			MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
			Action: "Team member removed", Detail: "Membership " + membershipID + " was revoked.",
			Severity: "warning",
		})
	}
	return nil
}
