package providers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/platform/vault"
)

// credentialPayload is the encrypted structure stored per connection. It is the
// only place provider secrets live outside memory.
type credentialPayload struct {
	SecretKey string `json:"secret_key"`
	PublicKey string `json:"public_key,omitempty"`
}

// Service owns provider connections: validation, encryption, verification and
// credential delivery to the payments domain.
type Service struct {
	Repo     *Repository
	Vault    *vault.Vault
	Registry Registry
	Audit    audit.Recorder
	// PublicBaseURL is the public origin of this API. It is used to build the
	// inbound webhook URL a merchant pastes into a provider dashboard.
	PublicBaseURL string
	Now           func() time.Time
}

// NewService builds the provider service.
func NewService(repo *Repository, v *vault.Vault, registry Registry, recorder audit.Recorder) *Service {
	return &Service{Repo: repo, Vault: v, Registry: registry, Audit: recorder, Now: time.Now}
}

// ConnectInput describes a new provider connection.
type ConnectInput struct {
	MerchantID  string
	ActorName   string
	ActorRole   string
	Provider    constants.Provider
	Environment string
	DisplayName string
	SecretKey   string
	PublicKey   string
	// WebhookSecret is the optional inbound webhook signing value for providers
	// whose signature key is not the API secret key (Flutterwave).
	WebhookSecret string
	IsDefault     bool
}

// Connect validates, encrypts and stores provider credentials.
func (s *Service) Connect(ctx context.Context, in ConnectInput) (*Connection, error) {
	if !constants.IsValidProvider(in.Provider) {
		return nil, apperrors.ProviderNotSupported(in.Provider.String())
	}
	adapter, err := s.Registry.Get(in.Provider)
	if err != nil {
		return nil, err
	}
	if !constants.IsValidPaymentEnvironment(in.Environment) {
		return nil, apperrors.InvalidEnvironment()
	}
	in.SecretKey = strings.TrimSpace(in.SecretKey)
	if err := validateCredentialShape(in.Provider, in.Environment, in.SecretKey); err != nil {
		return nil, err
	}

	credentials := Credentials{
		SecretKey:   in.SecretKey,
		PublicKey:   strings.TrimSpace(in.PublicKey),
		Environment: in.Environment,
	}
	if err := adapter.TestCredentials(ctx, credentials); err != nil {
		return nil, err
	}

	encrypted, err := s.Vault.Encrypt(mustJSON(credentialPayload{SecretKey: credentials.SecretKey, PublicKey: credentials.PublicKey}))
	if err != nil {
		return nil, err
	}

	now := s.Now().UTC()
	connection, err := s.Repo.ByProviderEnv(in.MerchantID, in.Provider.String(), in.Environment)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		connection = &Connection{
			ID:          gormx.NewID("conn"),
			MerchantID:  in.MerchantID,
			Provider:    in.Provider.String(),
			Environment: in.Environment,
			CreatedAt:   now,
		}
	case err != nil:
		return nil, err
	}

	if in.DisplayName != "" {
		connection.DisplayName = strings.TrimSpace(in.DisplayName)
	} else if connection.DisplayName == "" {
		connection.DisplayName = strings.Title(string(in.Provider)) + " " + in.Environment
	}
	connection.CredentialsEncrypted = encrypted
	connection.SecretLast4 = last4(in.SecretKey)
	if secret := strings.TrimSpace(in.WebhookSecret); secret != "" {
		masked, err := s.encryptWebhookSecret(secret)
		if err != nil {
			return nil, err
		}
		connection.WebhookSecretEncrypted = masked.encrypted
		connection.WebhookSecretMasked = masked.masked
	}
	connection.Status = "connected"
	connection.ConnectedAt = &now
	connection.LastVerifiedAt = &now
	connection.UpdatedAt = now
	if connection.Currency == "" {
		connection.Currency = defaultCurrency(in.Provider)
	}
	if in.IsDefault {
		if err := s.Repo.ClearDefault(in.MerchantID, in.Environment); err != nil {
			return nil, err
		}
		connection.IsDefault = true
	}

	if connection.CreatedAt.IsZero() {
		connection.CreatedAt = now
		if err := s.Repo.Create(connection); err != nil {
			return nil, err
		}
	} else if err := s.Repo.Save(connection); err != nil {
		return nil, err
	}

	s.Audit.Record(audit.Entry{
		MerchantID: in.MerchantID, ActorName: in.ActorName, ActorRole: in.ActorRole,
		Action:   "Provider connected",
		Detail:   in.Provider.String() + " (" + in.Environment + ")",
		Severity: "critical",
	})
	return connection, nil
}

// UpdateInput carries mutable connection fields.
type UpdateInput struct {
	DisplayName *string
	IsDefault   *bool
	SecretKey   *string
	PublicKey   *string
}

// Update changes a connection's label, default flag and optionally its secret
// (a rotate). The secret is re-validated and re-encrypted.
func (s *Service) Update(merchantID, actorName, actorRole, id string, in UpdateInput) (*Connection, error) {
	connection, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound("", "Provider connection not found.")
	}
	now := s.Now().UTC()

	if in.DisplayName != nil {
		connection.DisplayName = strings.TrimSpace(*in.DisplayName)
	}
	if in.IsDefault != nil && *in.IsDefault {
		if !connection.IsConnected() {
			return nil, apperrors.ProviderNotConnected(connection.Provider)
		}
		if err := s.Repo.ClearDefault(merchantID, connection.Environment); err != nil {
			return nil, err
		}
		connection.IsDefault = true
	} else if in.IsDefault != nil {
		connection.IsDefault = false
	}
	if in.SecretKey != nil {
		if err := validateCredentialShape(constants.Provider(connection.Provider), connection.Environment, *in.SecretKey); err != nil {
			return nil, err
		}
		payload := credentialPayload{SecretKey: *in.SecretKey}
		if in.PublicKey != nil {
			payload.PublicKey = *in.PublicKey
		}
		encrypted, err := s.Vault.Encrypt(mustJSON(payload))
		if err != nil {
			return nil, err
		}
		connection.CredentialsEncrypted = encrypted
		connection.SecretLast4 = last4(*in.SecretKey)
		connection.Status = "connected"
		connection.ConnectedAt = &now
		// A rotated secret must be re-verified before it is trusted.
		connection.LastVerifiedAt = nil
		s.Audit.Record(audit.Entry{
			MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
			Action: "Provider credentials rotated",
			Detail: connection.Provider + " " + connection.Environment, Severity: "critical",
		})
	}
	connection.UpdatedAt = now
	if err := s.Repo.Save(connection); err != nil {
		return nil, err
	}
	return connection, nil
}

// Delete disconnects a provider and destroys its stored credentials.
func (s *Service) Delete(merchantID, actorName, actorRole, id string) error {
	connection, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return apperrors.NotFound("", "Provider connection not found.")
	}
	if err := s.Repo.Delete(merchantID, id); err != nil {
		return err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Provider disconnected", Detail: connection.Provider + " " + connection.Environment,
		Severity: "critical",
	})
	return nil
}

// List returns a merchant's connections.
func (s *Service) List(merchantID string) ([]Connection, error) { return s.Repo.List(merchantID) }

// Get returns one connection.
func (s *Service) Get(merchantID, id string) (*Connection, error) {
	connection, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound("", "Provider connection not found.")
	}
	return connection, nil
}

// Test verifies stored credentials against the provider and records the result.
func (s *Service) Test(ctx context.Context, merchantID, actorName, actorRole, id string) (*Connection, error) {
	connection, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound("", "Provider connection not found.")
	}
	adapter, err := s.Registry.Get(constants.Provider(connection.Provider))
	if err != nil {
		return nil, err
	}
	credentials, err := s.decrypt(connection)
	if err != nil {
		return nil, err
	}
	if err := adapter.TestCredentials(ctx, credentials); err != nil {
		connection.Status = "error"
		connection.UpdatedAt = s.Now().UTC()
		_ = s.Repo.Save(connection)
		s.Audit.Record(audit.Entry{
			MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
			Action: "Provider verification failed", Detail: connection.Provider, Severity: "warning",
		})
		return nil, apperrors.ProviderAuthenticationFailed(connection.Provider, err)
	}
	now := s.Now().UTC()
	connection.Status = "connected"
	connection.LastVerifiedAt = &now
	connection.UpdatedAt = now
	if err := s.Repo.Save(connection); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Provider verified", Detail: connection.Provider + " " + connection.Environment, Severity: "info",
	})
	return connection, nil
}

// CredentialsFor returns decrypted credentials for a provider in one
// environment. It is the only credential accessor the payments domain uses.
func (s *Service) CredentialsFor(merchantID, environment string, provider constants.Provider) (Credentials, error) {
	connection, err := s.Repo.ByProviderEnv(merchantID, provider.String(), environment)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Credentials{}, apperrors.ProviderNotConnected(provider.String())
		}
		return Credentials{}, err
	}
	if !connection.IsConnected() {
		return Credentials{}, apperrors.ProviderNotConnected(provider.String())
	}
	return s.decrypt(connection)
}

// DefaultProvider returns the merchant's default (or first connected) provider
// for an environment — the entry point for routing when a client asks for
// adapter=auto.
func (s *Service) DefaultProvider(merchantID, environment string) (constants.Provider, error) {
	connections, err := s.Repo.List(merchantID)
	if err != nil {
		return "", err
	}
	var fallback constants.Provider
	for _, connection := range connections {
		if connection.Environment != environment || !connection.IsConnected() {
			continue
		}
		if connection.IsDefault {
			return constants.Provider(connection.Provider), nil
		}
		if fallback == "" {
			fallback = constants.Provider(connection.Provider)
		}
	}
	if fallback == "" {
		return "", apperrors.ProviderNotConnected("any provider")
	}
	return fallback, nil
}

// SetDefault marks a connected provider as the environment default. Routing
// calls it so a merchant can change its default route without rotating keys.
func (s *Service) SetDefault(merchantID, environment string, provider constants.Provider) error {
	connection, err := s.Repo.ByProviderEnv(merchantID, provider.String(), environment)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ProviderNotConnected(provider.String())
		}
		return err
	}
	if !connection.IsConnected() {
		return apperrors.ProviderNotConnected(provider.String())
	}
	if err := s.Repo.ClearDefault(merchantID, environment); err != nil {
		return err
	}
	connection.IsDefault = true
	connection.UpdatedAt = s.Now().UTC()
	return s.Repo.Save(connection)
}

// ConnectionWithCredentials loads a connection by id — for any merchant — and
// returns its decrypted credentials. The provider webhook gateway identifies the
// connection from the URL path, so it cannot scope the lookup by merchant.
func (s *Service) ConnectionWithCredentials(id string) (*Connection, Credentials, error) {
	connection, err := s.Repo.ByIDGlobal(id)
	if err != nil {
		return nil, Credentials{}, apperrors.NotFound("", "Provider connection not found.")
	}
	credentials, err := s.decrypt(connection)
	if err != nil {
		return nil, Credentials{}, err
	}
	return connection, credentials, nil
}

// ConnectedProviders lists provider names usable in an environment.
func (s *Service) ConnectedProviders(merchantID, environment string) ([]constants.Provider, error) {
	connections, err := s.Repo.List(merchantID)
	if err != nil {
		return nil, err
	}
	out := make([]constants.Provider, 0, len(connections))
	for _, connection := range connections {
		if connection.Environment == environment && connection.IsConnected() {
			out = append(out, constants.Provider(connection.Provider))
		}
	}
	return out, nil
}

// SetWebhookSecret stores (or rotates) the inbound webhook signing value for a
// connection. Only providers whose signing key is not the API secret key need
// one (Flutterwave's "verif-hash"); for the others the secret key is used and
// this returns a validation error.
func (s *Service) SetWebhookSecret(merchantID, actorName, actorRole, id, secret string) (*Connection, error) {
	connection, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound("", "Provider connection not found.")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "webhook_secret is required.")
	}
	support := constants.WebhookSupportFor(constants.Provider(connection.Provider))
	if !support.RequiresWebhookSecret {
		return nil, apperrors.Validation(apperrors.CodeValidationError,
			strings.Title(connection.Provider)+" verifies webhooks with its secret key; no separate webhook secret is needed.")
	}
	stored, err := s.encryptWebhookSecret(secret)
	if err != nil {
		return nil, err
	}
	connection.WebhookSecretEncrypted = stored.encrypted
	connection.WebhookSecretMasked = stored.masked
	connection.UpdatedAt = s.Now().UTC()
	if err := s.Repo.Save(connection); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Provider webhook secret updated",
		Detail: connection.Provider + " " + connection.Environment, Severity: "critical",
	})
	return connection, nil
}

// TestableConnection reports the provider and environment of one of a merchant's
// connections, and refuses the ones a dashboard-initiated test charge must not
// touch. Live connections are rejected outright: a test transaction forces a
// real authorization page, and it must never be able to move live money.
func (s *Service) TestableConnection(merchantID, connectionID string) (constants.Provider, string, error) {
	connection, err := s.Repo.ByID(merchantID, connectionID)
	if err != nil {
		return "", "", apperrors.NotFound("", "Provider connection not found.")
	}
	if !connection.IsConnected() {
		return "", "", apperrors.ProviderNotConnected(connection.Provider)
	}
	if connection.Environment != constants.EnvironmentTest {
		return "", "", apperrors.Validation(apperrors.CodeValidationError,
			"Test transactions are only available for connections in the test environment.")
	}
	return constants.Provider(connection.Provider), connection.Environment, nil
}

// TouchWebhookReceived stamps the last verified inbound webhook time so the
// dashboard can show connection health without scanning the event log.
func (s *Service) TouchWebhookReceived(id string, at time.Time) error {
	return s.Repo.TouchWebhookReceived(id, at)
}

// ToResponse maps a connection onto its public DTO, including the computed
// inbound webhook URL and the provider's webhook requirements. The URL is
// derived from the connection id, so it is stable for the connection's life.
func (s *Service) ToResponse(c *Connection) ConnectionResponse {
	resp := ToConnectionResponse(c)
	support := constants.WebhookSupportFor(constants.Provider(c.Provider))
	resp.WebhookSecretRequired = support.RequiresWebhookSecret
	resp.WebhookSecretLabel = support.SecretLabel
	resp.WebhookSignatureHeader = support.SignatureHeader
	resp.WebhookDashboardPath = support.DashboardPath
	resp.WebhookSecretSet = c.WebhookSecretMasked != ""
	if base := strings.TrimRight(s.PublicBaseURL, "/"); base != "" && c.ID != "" {
		resp.WebhookURL = base + "/v1/webhooks/" + c.Provider + "/" + c.ID
	}
	return resp
}

func (s *Service) decrypt(c *Connection) (Credentials, error) {
	if len(c.CredentialsEncrypted) == 0 {
		return Credentials{}, apperrors.ProviderNotConnected(c.Provider)
	}
	plaintext, err := s.Vault.Decrypt(c.CredentialsEncrypted)
	if err != nil {
		return Credentials{}, err
	}
	var payload credentialPayload
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		return Credentials{}, err
	}
	webhookSecret, err := s.decryptWebhookSecret(c)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{
		SecretKey:     payload.SecretKey,
		PublicKey:     payload.PublicKey,
		WebhookSecret: webhookSecret,
		Environment:   c.Environment,
	}, nil
}

// encryptWebhookSecret encrypts an inbound webhook signing value and derives its
// masked form for display.
func (s *Service) encryptWebhookSecret(secret string) (encryptedWebhookSecret, error) {
	encrypted, err := s.Vault.Encrypt(secret)
	if err != nil {
		return encryptedWebhookSecret{}, err
	}
	return encryptedWebhookSecret{encrypted: encrypted, masked: maskWebhookSecret(secret)}, nil
}

// decryptWebhookSecret returns the connection's inbound webhook signing value, or
// an empty string when none is stored.
func (s *Service) decryptWebhookSecret(c *Connection) (string, error) {
	if len(c.WebhookSecretEncrypted) == 0 {
		return "", nil
	}
	return s.Vault.Decrypt(c.WebhookSecretEncrypted)
}

// encryptedWebhookSecret is the encrypted-plus-masked pair stored on a connection.
type encryptedWebhookSecret struct {
	encrypted []byte
	masked    string
}

// maskWebhookSecret renders a webhook secret for display, keeping only its last
// four characters. It is never reversible to the stored value.
func maskWebhookSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "••••"
	}
	return "••••••••" + value[len(value)-4:]
}

func mustJSON(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
