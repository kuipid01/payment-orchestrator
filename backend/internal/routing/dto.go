package routing

import "time"

// Response is the public routing-rule representation.
type Response struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Currency  string       `json:"currency,omitempty"`
	Country   string       `json:"country,omitempty"`
	MinAmount int64        `json:"min_amount"`
	MaxAmount int64        `json:"max_amount"`
	Providers []Preference `json:"providers"`
	Priority  int          `json:"priority"`
	Enabled   bool         `json:"enabled"`
	CreatedAt string       `json:"created_at"`
	UpdatedAt string       `json:"updated_at"`
}

// ToResponse maps a rule onto the public DTO.
func ToResponse(rule *Rule) Response {
	return Response{
		ID:        rule.ID,
		Name:      rule.Name,
		Currency:  rule.Currency,
		Country:   rule.Country,
		MinAmount: rule.MinAmount,
		MaxAmount: rule.MaxAmount,
		Providers: rule.Providers.Data,
		Priority:  rule.Priority,
		Enabled:   rule.Enabled,
		CreatedAt: rule.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: rule.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// OverviewResponse is the routing configuration screen payload.
type OverviewResponse struct {
	Environment     string     `json:"environment"`
	DefaultProvider string     `json:"default_provider,omitempty"`
	Connected       []string   `json:"connected_providers"`
	Rules           []Response `json:"rules"`
}

// RuleRequest is the create/update payload for a rule.
type RuleRequest struct {
	Name      string       `json:"name"`
	Currency  string       `json:"currency"`
	Country   string       `json:"country"`
	MinAmount int64        `json:"min_amount"`
	MaxAmount int64        `json:"max_amount"`
	Providers []Preference `json:"providers"`
	Priority  int          `json:"priority"`
	Enabled   *bool        `json:"enabled"`
}

// toInput maps the request onto the service input.
func (r RuleRequest) toInput() RuleInput {
	return RuleInput{
		Name:      r.Name,
		Currency:  r.Currency,
		Country:   r.Country,
		MinAmount: r.MinAmount,
		MaxAmount: r.MaxAmount,
		Providers: r.Providers,
		Priority:  r.Priority,
		Enabled:   r.Enabled,
	}
}
