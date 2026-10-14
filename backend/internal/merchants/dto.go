package merchants

import "github.com/kuipid01/payconnect/internal/constants"

// Response is the public merchant representation.
type Response struct {
	ID              string `json:"id"`
	BusinessName    string `json:"business_name"`
	Email           string `json:"email"`
	Country         string `json:"country"`
	Status          string `json:"status"`
	DefaultCurrency string `json:"default_currency"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ToResponse maps the domain model onto the public DTO.
func ToResponse(m *Merchant) Response {
	return Response{
		ID:              m.ID,
		BusinessName:    m.BusinessName,
		Email:           m.Email,
		Country:         m.Country,
		Status:          m.Status,
		DefaultCurrency: m.DefaultCurrency,
		CreatedAt:       m.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       m.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func isSupportedCurrency(code string) bool { return constants.IsSupportedCurrency(code) }
