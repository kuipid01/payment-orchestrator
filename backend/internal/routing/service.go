package routing

import (
	"context"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/billing"
	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// ConnectionLister reports which providers a merchant can actually use. The
// providers domain implements it, so routing never imports provider credentials.
type ConnectionLister interface {
	ConnectedProviders(merchantID, environment string) ([]constants.Provider, error)
	DefaultProvider(merchantID, environment string) (constants.Provider, error)
}

// DefaultSetter records which connected provider routes by default in an
// environment. The providers domain owns the flag, so routing only asks.
type DefaultSetter interface {
	SetDefault(merchantID, environment string, provider constants.Provider) error
}

// PlanGates is the one billing question this package asks: does the
// organization's plan include routing rules at all? Optional, and a nil gate
// means no limit.
//
// Only rule administration is gated. Resolve — the code that picks a provider
// for a live payment — is deliberately never gated: a plan decides which rules
// may exist, never whether a customer's payment can be routed.
type PlanGates interface {
	CheckRoutingRules(ctx context.Context, merchantID string) billing.Decision
}

// Service resolves providers for a payment.
type Service struct {
	Repo        *Repository
	Connections ConnectionLister
	Preferences DefaultSetter
	Audit       audit.Recorder
	Gates       PlanGates
	Now         func() time.Time
}

// NewService builds the routing service.
func NewService(repo *Repository, connections ConnectionLister, recorder audit.Recorder) *Service {
	return &Service{Repo: repo, Connections: connections, Audit: recorder, Now: time.Now}
}

// SetDefaultProvider marks a connected provider as the environment default.
func (s *Service) SetDefaultProvider(merchantID, actorName, actorRole, environment string, provider constants.Provider) error {
	if !constants.IsValidProvider(provider) {
		return apperrors.InvalidProvider("")
	}
	if !constants.IsValidPaymentEnvironment(environment) {
		return apperrors.InvalidEnvironment()
	}
	if s.Preferences == nil {
		return apperrors.ServiceUnavailable("Routing preferences are not configured.", nil)
	}
	connected, err := s.Connections.ConnectedProviders(merchantID, environment)
	if err != nil {
		return err
	}
	for _, candidate := range connected {
		if candidate == provider {
			if err := s.Preferences.SetDefault(merchantID, environment, provider); err != nil {
				return err
			}
			s.Audit.Record(audit.Entry{
				MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
				Action: "Default provider changed", Detail: provider.String() + " (" + environment + ")",
				Severity: "warning",
			})
			return nil
		}
	}
	return apperrors.ProviderNotConnected(provider.String())
}

// ConnectedProviders exposes the provider list for the routing overview.
func (s *Service) ConnectedProviders(merchantID, environment string) ([]constants.Provider, error) {
	return s.Connections.ConnectedProviders(merchantID, environment)
}

// DefaultProvider exposes the current environment default.
func (s *Service) DefaultProvider(merchantID, environment string) (constants.Provider, error) {
	return s.Connections.DefaultProvider(merchantID, environment)
}

// Resolve returns the ordered provider chain for a payment.
//
// An explicit provider request wins. Otherwise matching rules are evaluated in
// priority order and their providers are filtered to those the merchant has
// actually connected in this environment — PayConnect never routes to a
// provider that cannot settle the payment.
func (s *Service) Resolve(merchantID, environment, currency string, amountMinor int64, requested constants.Provider) ([]constants.Provider, error) {
	if requested != "" && requested != constants.ProviderAuto {
		if !constants.IsValidProvider(requested) {
			return nil, apperrors.InvalidProvider("")
		}
		return []constants.Provider{requested}, nil
	}

	connected, err := s.Connections.ConnectedProviders(merchantID, environment)
	if err != nil {
		return nil, err
	}
	connectedSet := make(map[constants.Provider]bool, len(connected))
	for _, provider := range connected {
		connectedSet[provider] = true
	}

	rules, err := s.Repo.List(merchantID)
	if err != nil {
		return nil, err
	}

	chain := make([]constants.Provider, 0, len(connected))
	seen := make(map[constants.Provider]bool)
	for _, rule := range rules {
		if !rule.Matches(currency, amountMinor) {
			continue
		}
		for _, name := range rule.OrderedProviders() {
			provider := constants.Provider(name)
			if !constants.IsValidProvider(provider) || seen[provider] || !connectedSet[provider] {
				continue
			}
			seen[provider] = true
			chain = append(chain, provider)
		}
		if len(chain) > 0 {
			break // the first matching rule wins
		}
	}

	if len(chain) == 0 {
		fallback, err := s.Connections.DefaultProvider(merchantID, environment)
		if err != nil {
			return nil, err
		}
		chain = append(chain, fallback)
	}

	// Round out the chain with any other connected provider so a safely
	// retryable rejection can fall back without a second routing pass.
	for _, provider := range connected {
		if !seen[provider] {
			seen[provider] = true
			chain = append(chain, provider)
		}
	}
	return chain, nil
}

// RuleInput is a create/update payload for a routing rule.
type RuleInput struct {
	Name      string
	Currency  string
	Country   string
	MinAmount int64
	MaxAmount int64
	Providers []Preference
	Priority  int
	Enabled   *bool
}

// CreateRule adds a rule, validating its provider list.
func (s *Service) CreateRule(ctx context.Context, merchantID, actorName, actorRole string, in RuleInput) (*Rule, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "name is required.")
	}
	preferences, err := normalizePreferences(in.Providers)
	if err != nil {
		return nil, err
	}
	if in.MinAmount > 0 && in.MaxAmount > 0 && in.MinAmount > in.MaxAmount {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "min_amount must not exceed max_amount.")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	now := s.Now().UTC()
	rule := &Rule{
		ID:         gormx.NewID("route"),
		MerchantID: merchantID,
		Name:       strings.TrimSpace(in.Name),
		Currency:   strings.ToUpper(strings.TrimSpace(in.Currency)),
		Country:    strings.ToUpper(strings.TrimSpace(in.Country)),
		MinAmount:  in.MinAmount,
		MaxAmount:  in.MaxAmount,
		Providers:  gormx.JSONB[[]Preference]{Data: preferences},
		Priority:   in.Priority,
		Enabled:    enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if rule.Currency != "" && !constants.IsSupportedCurrency(rule.Currency) {
		return nil, apperrors.InvalidCurrency("currency is not supported.")
	}
	if err := s.checkRoutingPlan(ctx, merchantID); err != nil {
		return nil, err
	}
	if err := s.Repo.Create(rule); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Routing rule created", Detail: rule.Name, Severity: "warning",
	})
	return rule, nil
}

// checkRoutingPlan asks the plan gate, when one is wired.
func (s *Service) checkRoutingPlan(ctx context.Context, merchantID string) error {
	if s.Gates == nil {
		return nil
	}
	return s.Gates.CheckRoutingRules(ctx, merchantID).Refusal()
}

// UpdateRule applies a partial update to a rule.
func (s *Service) UpdateRule(merchantID, actorName, actorRole, id string, in RuleInput) (*Rule, error) {
	rule, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.RouteNotFound()
	}
	if in.Name != "" {
		rule.Name = strings.TrimSpace(in.Name)
	}
	if in.Currency != "" {
		currency := strings.ToUpper(strings.TrimSpace(in.Currency))
		if !constants.IsSupportedCurrency(currency) {
			return nil, apperrors.InvalidCurrency("currency is not supported.")
		}
		rule.Currency = currency
	}
	if in.Country != "" {
		rule.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	}
	if in.MinAmount > 0 {
		rule.MinAmount = in.MinAmount
	}
	if in.MaxAmount > 0 {
		rule.MaxAmount = in.MaxAmount
	}
	if in.Providers != nil {
		preferences, err := normalizePreferences(in.Providers)
		if err != nil {
			return nil, err
		}
		rule.Providers = gormx.JSONB[[]Preference]{Data: preferences}
	}
	if in.Priority != 0 {
		rule.Priority = in.Priority
	}
	if in.Enabled != nil {
		rule.Enabled = *in.Enabled
	}
	if rule.MinAmount > 0 && rule.MaxAmount > 0 && rule.MinAmount > rule.MaxAmount {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "min_amount must not exceed max_amount.")
	}
	rule.UpdatedAt = s.Now().UTC()
	if err := s.Repo.Save(rule); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Routing rule updated", Detail: rule.Name, Severity: "warning",
	})
	return rule, nil
}

// DeleteRule removes a rule.
func (s *Service) DeleteRule(merchantID, actorName, actorRole, id string) error {
	affected, err := s.Repo.Delete(merchantID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperrors.RouteNotFound()
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Routing rule deleted", Detail: id, Severity: "warning",
	})
	return nil
}

// List returns a merchant's rules.
func (s *Service) List(merchantID string) ([]Rule, error) { return s.Repo.List(merchantID) }

// Get returns one rule.
func (s *Service) Get(merchantID, id string) (*Rule, error) {
	rule, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.RouteNotFound()
	}
	return rule, nil
}

func normalizePreferences(in []Preference) ([]Preference, error) {
	if len(in) == 0 {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "providers must contain at least one entry.")
	}
	out := make([]Preference, 0, len(in))
	for _, preference := range in {
		provider := constants.Provider(preference.Provider)
		if !constants.IsValidProvider(provider) {
			return nil, apperrors.InvalidProvider("")
		}
		out = append(out, Preference{Provider: provider.String(), Priority: preference.Priority})
	}
	return out, nil
}
