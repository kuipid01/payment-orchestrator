package merchants

import (
	"strings"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// Service owns merchant profile rules.
type Service struct {
	Repo *Repository
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
}

// Update applies a partial profile update.
func (s *Service) Update(merchantID string, in UpdateInput) (*Merchant, error) {
	m, err := s.Repo.MerchantByID(merchantID)
	if err != nil {
		return nil, apperrors.NotFound("", "Merchant not found.")
	}
	if in.BusinessName != nil {
		name := strings.TrimSpace(*in.BusinessName)
		if name == "" {
			return nil, apperrors.Validation(apperrors.CodeValidationError, "business_name must not be empty.")
		}
		m.BusinessName = name
	}
	if in.Country != nil {
		country := strings.ToUpper(strings.TrimSpace(*in.Country))
		if len(country) != 2 {
			return nil, apperrors.Validation(apperrors.CodeValidationError, "country must be an ISO 3166-1 alpha-2 code.")
		}
		m.Country = country
	}
	if in.DefaultCurrency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*in.DefaultCurrency))
		if !isSupportedCurrency(currency) {
			return nil, apperrors.InvalidCurrency("default_currency is not supported.")
		}
		m.DefaultCurrency = currency
	}
	if err := s.Repo.UpdateMerchant(m); err != nil {
		return nil, err
	}
	return m, nil
}
