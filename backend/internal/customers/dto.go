package customers

import "time"

// Response is the public customer representation.
type Response struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Phone     string         `json:"phone"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

// ToResponse maps the model onto the public DTO.
func ToResponse(c *Customer) Response {
	return Response{
		ID:        c.ID,
		Email:     c.Email,
		Name:      c.Name,
		Phone:     c.Phone,
		Status:    c.Status,
		Metadata:  c.Metadata.Data,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
