package api_keys

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/platform/identity"
	"github.com/kuipid01/payconnect/internal/platform/vault"
)

var (
	errRevoked          = errors.New("apikeys: key revoked")
	errExpired          = errors.New("apikeys: key expired")
	errNotAuthenticable = errors.New("apikeys: webhook secrets cannot authenticate requests")
)

// Service owns API key lifecycle and authentication.
type Service struct {
	Repo      *Repository
	Merchants *merchants.Repository
	Vault     *vault.Vault
	Audit     audit.Recorder
	Prefix    string
	Now       func() time.Time
	// Logger records the credential half of a request trace — which key acted,
	// for which application and developer. Optional: nil means silent.
	Logger *slog.Logger
}

// NewService builds the API key service.
func NewService(repo *Repository, merchantRepo *merchants.Repository, v *vault.Vault, recorder audit.Recorder, prefix string) *Service {
	if prefix == "" {
		prefix = constants.KeyPrefixDefault
	}
	return &Service{Repo: repo, Merchants: merchantRepo, Vault: v, Audit: recorder, Prefix: prefix, Now: time.Now}
}

// CreateInput describes a new key.
type CreateInput struct {
	MerchantID    string
	ActorName     string
	ActorRole     string
	Name          string
	Type          constants.APIKeyType
	Environment   string
	ApplicationID *string
	// DeveloperAccessID, when set, makes the key developer-scoped: it belongs
	// to one developer grant and is revoked with that grant.
	DeveloperAccessID *string
	ExpiresIn         time.Duration
}

// Created is the one-time result of minting a key.
type Created struct {
	Key    *Key
	Secret string // full key, returned exactly once
}

// Create mints a key and returns its secret exactly once.
func (s *Service) Create(in CreateInput) (*Created, error) {
	if in.Name = strings.TrimSpace(in.Name); in.Name == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "name is required.")
	}
	if !constants.IsValidAPIKeyType(in.Type) {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "type must be one of secret, public, webhook.")
	}
	if !constants.IsValidPaymentEnvironment(in.Environment) {
		return nil, apperrors.InvalidEnvironment()
	}
	if in.ApplicationID != nil {
		var applicationID string
		err := s.Repo.DB.Table("applications").
			Select("id").
			Where("id = ? AND merchant_id = ? AND deleted_at IS NULL", *in.ApplicationID, in.MerchantID).
			Take(&applicationID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound(apperrors.CodeNotFound, "Application not found.")
		}
		if err != nil {
			return nil, err
		}
	}
	if in.DeveloperAccessID != nil && *in.DeveloperAccessID != "" {
		if _, err := s.grant(in.MerchantID, *in.DeveloperAccessID); err != nil {
			return nil, err
		}
	} else {
		in.DeveloperAccessID = nil
	}
	now := s.Now().UTC()
	key, secret, err := s.mint(in, now)
	if err != nil {
		return nil, err
	}
	if err := s.Repo.Create(key); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: in.MerchantID, ActorName: in.ActorName, ActorRole: in.ActorRole,
		Action:   "API key created",
		Detail:   key.SecretMasked + " (" + string(in.Type) + ", " + in.Environment + ", " + key.Scope() + ")",
		Severity: "critical",
	})
	s.log("api key minted", "key_id", key.ID, "masked", key.SecretMasked, "environment", key.Environment,
		"scope", key.Scope(), "application_id", key.ApplicationIDValue(),
		"developer_access_id", key.DeveloperAccessIDValue(), "revealable", key.Recoverable())
	return &Created{Key: key, Secret: secret}, nil
}

// grant loads a developer grant and confirms it belongs to the merchant.
func (s *Service) grant(merchantID, accessID string) (*merchants.DeveloperMerchantAccess, error) {
	access, err := s.Merchants.DeveloperMerchantAccessByID(accessID)
	if err != nil || access.MerchantID != merchantID {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "Developer access not found.")
	}
	return access, nil
}

// Reveal returns the plaintext of a test key.
//
// The policy is deliberate and asymmetric: a live key is displayed exactly once,
// at creation, because its plaintext is the thing that moves real money. A test
// key stays readable, because re-reading it is routine — a developer switching
// laptops, a CI secret being re-pasted — and forcing a rotation for that would
// churn config for no security benefit, given a test key is pinned to test data.
//
// The caller must be a dashboard session for the owning merchant; the handler
// enforces that, and every reveal is audited.
func (s *Service) Reveal(merchantID, actorName, actorRole, id string) (string, error) {
	key, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return "", apperrors.NotFound(apperrors.CodeNotFound, "API key not found.")
	}
	now := s.Now().UTC()
	if errors.Is(key.Usable(now), errRevoked) {
		return "", apperrors.Conflict(apperrors.CodeAPIKeyRevoked, "This API key has been revoked.")
	}
	if key.Environment != constants.EnvironmentTest {
		return "", apperrors.PermissionDenied(
			"Only test keys can be shown again. A live key is displayed once, when it is created.")
	}
	if len(key.SecretEncrypted) == 0 || s.Vault == nil {
		return "", apperrors.Conflict("key_not_recoverable",
			"This test key was created before keys stayed copyable. Rotate it to get one you can copy.")
	}
	secret, err := s.Vault.Decrypt(key.SecretEncrypted)
	if err != nil {
		return "", apperrors.Internal(err).WithMessage("This key could not be decrypted. Rotate it to mint a new one.")
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action:   "API key revealed",
		Detail:   key.SecretMasked + " (" + key.Environment + ", " + key.Scope() + ")",
		Severity: "warning",
	})
	return secret, nil
}

// Rotate revokes a key and mints its replacement with the same scope.
func (s *Service) Rotate(merchantID, actorName, actorRole, id string) (*Created, error) {
	old, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "API key not found.")
	}
	if old.RevokedAt != nil {
		return nil, apperrors.Conflict(apperrors.CodeAPIKeyRevoked, "This API key has already been revoked.")
	}
	now := s.Now().UTC()
	fresh, secret, err := s.mint(CreateInput{
		MerchantID:        merchantID,
		ActorName:         actorName,
		Name:              old.Name,
		Type:              constants.APIKeyType(old.Type),
		Environment:       old.Environment,
		ApplicationID:     old.ApplicationID,
		DeveloperAccessID: old.DeveloperAccessID,
	}, now)
	if err != nil {
		return nil, err
	}
	if _, err := s.Repo.Revoke(merchantID, old.ID, now); err != nil {
		return nil, err
	}
	if err := s.Repo.Create(fresh); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "API key rotated", Detail: fresh.SecretMasked + " replaced " + old.SecretMasked,
		Severity: "critical",
	})
	return &Created{Key: fresh, Secret: secret}, nil
}

// Revoke permanently disables a key.
func (s *Service) Revoke(merchantID, actorName, actorRole, id string) error {
	affected, err := s.Repo.Revoke(merchantID, id, s.Now().UTC())
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperrors.NotFound(apperrors.CodeNotFound, "API key not found.")
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "API key revoked", Detail: "key " + id, Severity: "critical",
	})
	return nil
}

// DeleteRevoked permanently removes a key only after it has been revoked.
func (s *Service) DeleteRevoked(merchantID, actorName, actorRole, id string) error {
	key, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return apperrors.NotFound(apperrors.CodeNotFound, "API key not found.")
	}
	if key.RevokedAt == nil {
		return apperrors.Conflict(apperrors.CodeAPIKeyRevoked, "Revoke this API key before permanently deleting it.")
	}
	affected, err := s.Repo.DeleteRevoked(merchantID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperrors.NotFound(apperrors.CodeNotFound, "API key not found.")
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Revoked API key deleted", Detail: key.SecretMasked + " (" + key.Environment + ", " + key.Scope() + ")",
		Severity: "critical",
	})
	return nil
}

// RevokeByApplication permanently disables all active keys in an application.
func (s *Service) RevokeByApplication(merchantID, applicationID string, at time.Time) error {
	_, err := s.Repo.RevokeByApplication(merchantID, applicationID, at)
	return err
}

// RevokeByDeveloperAccess revokes every active key bound to one developer
// grant. Removing a developer does not touch the application's shared key.
func (s *Service) RevokeByDeveloperAccess(merchantID, accessID string, at time.Time) error {
	_, err := s.Repo.RevokeByDeveloperAccess(merchantID, accessID, at)
	return err
}

// List returns a merchant's keys with secrets masked. developerAccessID may be
// a grant id (only that developer's keys) or "general" (team-shared keys).
func (s *Service) List(merchantID, keyType, environment, developerAccessID string) ([]Key, error) {
	return s.Repo.List(ListFilter{
		MerchantID:        merchantID,
		Type:              keyType,
		Environment:       environment,
		DeveloperAccessID: developerAccessID,
	})
}

// Authenticate resolves a raw API key into a request principal. It is the
// API-key half of the shared authentication middleware.
func (s *Service) Authenticate(ctx context.Context, raw string) (*identity.Principal, error) {
	env, keyType, ok := ParseRaw(raw)
	if !ok {
		return nil, apperrors.InvalidAPIKey()
	}
	key, err := s.Repo.ByHash(vault.HashKey(raw))
	if err != nil {
		return nil, apperrors.InvalidAPIKey()
	}
	now := s.Now().UTC()
	switch {
	case errors.Is(key.Usable(now), errRevoked):
		return nil, apperrors.APIKeyRevoked()
	case errors.Is(key.Usable(now), errExpired):
		return nil, apperrors.APIKeyExpired()
	case errors.Is(key.Usable(now), errNotAuthenticable):
		return nil, apperrors.InvalidAPIKey().WithMessage("Webhook signing secrets cannot authenticate API requests.")
	}
	// The key's environment is authoritative: a test key can never act on live data.
	if key.Environment != env {
		return nil, apperrors.InvalidAPIKey()
	}
	merchant, err := s.Merchants.MerchantByID(key.MerchantID)
	if err != nil {
		return nil, apperrors.InvalidAPIKey()
	}
	if !merchant.IsActive() {
		return nil, apperrors.MerchantSuspended()
	}
	_ = s.Repo.TouchLastUsed(key.ID, now)

	applicationID := key.ApplicationIDValue()
	s.log("api key authenticated", "key_id", key.ID, "masked", key.SecretMasked, "environment", key.Environment,
		"merchant_id", key.MerchantID, "application_id", applicationID,
		"developer_access_id", key.DeveloperAccessIDValue(), "revealable", key.Recoverable())
	principal := &identity.Principal{
		Mode:          identity.ModeAPIKey,
		MerchantID:    key.MerchantID,
		OrgID:         key.MerchantID,
		Environment:   key.Environment,
		APIKeyID:      key.ID,
		KeyType:       keyType,
		ApplicationID: applicationID,
	}
	if applicationID != "" {
		principal.Application = map[string]string{"id": applicationID}
	}
	// A developer-scoped key acts only while its grant is active: revoking the
	// developer (or leaving an invitation unaccepted) disables the key without
	// touching anyone else's credentials.
	if accessID := key.DeveloperAccessIDValue(); accessID != "" {
		access, err := s.grant(key.MerchantID, accessID)
		if err != nil || access.Status != "active" {
			return nil, apperrors.InvalidAPIKey()
		}
		principal.DeveloperAccessID = accessID
		principal.DeveloperPermissions = access.Permissions.Data
	}
	return principal, nil
}

// log writes one credential-trace line. No-op without a logger.
func (s *Service) log(message string, attrs ...any) {
	if s.Logger == nil {
		return
	}
	s.Logger.Info(message, attrs...)
}

// ParseRaw splits a raw key into its environment and type markers.
func ParseRaw(raw string) (environment, keyType string, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), "_", 4)
	if len(parts) != 4 {
		return "", "", false
	}
	env := parts[1]
	switch env {
	case constants.KeyEnvTest, constants.KeyEnvLive:
	default:
		return "", "", false
	}
	switch parts[2] {
	case constants.KeyTypeSecret:
		return env, string(constants.APIKeySecret), true
	case constants.KeyTypePublic:
		return env, string(constants.APIKeyPublic), true
	case constants.KeyTypeWebhook:
		return env, string(constants.APIKeyWebhook), true
	}
	return "", "", false
}

// mint builds a key record and its one-time secret.
func (s *Service) mint(in CreateInput, now time.Time) (*Key, string, error) {
	secret, masked, err := vault.GenerateKey(s.Prefix, in.Environment, constants.KeyTypeMarker(in.Type))
	if err != nil {
		return nil, "", err
	}
	var expiresAt *time.Time
	if in.ExpiresIn > 0 {
		expiry := now.Add(in.ExpiresIn)
		expiresAt = &expiry
	}
	key := &Key{
		ID:                gormx.NewID("key"),
		MerchantID:        in.MerchantID,
		ApplicationID:     in.ApplicationID,
		DeveloperAccessID: in.DeveloperAccessID,
		Environment:       in.Environment,
		Type:              string(in.Type),
		Prefix:            s.Prefix,
		Name:              in.Name,
		KeyHash:           vault.HashKey(secret),
		SecretMasked:      masked,
		CreatedBy:         in.ActorName,
		ExpiresAt:         expiresAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	// A test key keeps a decryptable copy so it can be shown again; a live key is
	// write-only and exists here as nothing but a hash.
	if in.Environment == constants.EnvironmentTest && s.Vault != nil {
		encrypted, err := s.Vault.Encrypt(secret)
		if err != nil {
			return nil, "", err
		}
		key.SecretEncrypted = encrypted
	}
	return key, secret, nil
}
