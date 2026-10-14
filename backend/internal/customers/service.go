package customers

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/audit"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Service owns customer rules.
type Service struct {
	Repo  *Repository
	Audit audit.Recorder
	Now   func() time.Time
}

// NewService builds the customer service.
func NewService(repo *Repository, recorder audit.Recorder) *Service {
	return &Service{Repo: repo, Audit: recorder, Now: time.Now}
}

// Input is a create/update payload.
type Input struct {
	Email    string
	Name     string
	Phone    string
	Metadata map[string]any
}

// FindOrCreate resolves the customer for a payment, creating it on first use.
// It is what the payments domain calls before a provider request.
func (s *Service) FindOrCreate(merchantID, email, name, phone string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", apperrors.Validation(apperrors.CodeValidationError, "customer.email is required.")
	}
	if existing, err := s.Repo.ByEmail(merchantID, email); err == nil {
		updated := false
		if existing.Name == "" && name != "" {
			existing.Name = name
			updated = true
		}
		if existing.Phone == "" && phone != "" {
			existing.Phone = phone
			updated = true
		}
		if updated {
			existing.UpdatedAt = s.Now().UTC()
			_ = s.Repo.Save(existing)
		}
		return existing.ID, nil
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	now := s.Now().UTC()
	customer := &Customer{
		ID:         gormx.NewID("cus"),
		MerchantID: merchantID,
		Email:      email,
		Name:       name,
		Phone:      phone,
		Status:     "active",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.Repo.Create(customer); err != nil {
		return "", err
	}
	return customer.ID, nil
}

// Create adds a customer explicitly.
func (s *Service) Create(merchantID string, in Input) (*Customer, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(email, "@") {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "A valid email is required.")
	}
	if _, err := s.Repo.ByEmail(merchantID, email); err == nil {
		return nil, apperrors.Conflict("customer_exists", "A customer with this email already exists.")
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	now := s.Now().UTC()
	customer := &Customer{
		ID:         gormx.NewID("cus"),
		MerchantID: merchantID,
		Email:      email,
		Name:       strings.TrimSpace(in.Name),
		Phone:      strings.TrimSpace(in.Phone),
		Status:     "active",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if in.Metadata != nil {
		customer.Metadata = gormx.JSONB[map[string]any]{Data: in.Metadata}
	}
	if err := s.Repo.Create(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

// Lookup resolves several customers at once for read models that only stored a
// customer id. Unknown or deleted ids are absent from the result, so a caller
// degrades to showing the id instead of failing the whole listing.
func (s *Service) Lookup(merchantID string, ids []string) ([]Customer, error) {
	if merchantID == "" || len(ids) == 0 {
		return nil, nil
	}
	return s.Repo.Refs(merchantID, ids)
}

// Get returns one customer.
func (s *Service) Get(merchantID, id string) (*Customer, error) {
	customer, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.CustomerNotFound()
	}
	return customer, nil
}

// List returns a page of customers.
func (s *Service) List(f ListFilter) ([]Customer, int64, error) {
	rows, err := s.Repo.List(f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.Repo.Count(f.MerchantID)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Update applies a partial update.
func (s *Service) Update(merchantID, id string, in Input) (*Customer, error) {
	customer, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.CustomerNotFound()
	}
	if in.Email != "" {
		email := strings.ToLower(strings.TrimSpace(in.Email))
		if !strings.Contains(email, "@") {
			return nil, apperrors.Validation(apperrors.CodeValidationError, "A valid email is required.")
		}
		if email != customer.Email {
			if _, err := s.Repo.ByEmail(merchantID, email); err == nil {
				return nil, apperrors.Conflict("customer_exists", "A customer with this email already exists.")
			} else if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}
		customer.Email = email
	}
	if in.Name != "" {
		customer.Name = strings.TrimSpace(in.Name)
	}
	if in.Phone != "" {
		customer.Phone = strings.TrimSpace(in.Phone)
	}
	if in.Metadata != nil {
		customer.Metadata = gormx.JSONB[map[string]any]{Data: in.Metadata}
	}
	customer.UpdatedAt = s.Now().UTC()
	if err := s.Repo.Save(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

// Delete soft-deletes a customer.
func (s *Service) Delete(merchantID, actorName, actorRole, id string) error {
	affected, err := s.Repo.SoftDelete(merchantID, id, s.Now().UTC())
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperrors.CustomerNotFound()
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Customer deleted", Detail: id, Severity: "warning",
	})
	return nil
}
