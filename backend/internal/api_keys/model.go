// Package api_keys owns developer API keys: creation, rotation, revocation and
// the API-key half of request authentication.
package api_keys

import (
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
)

// Key scopes. A general key is shared by the team that owns the application; a
// developer key belongs to one developer grant and dies with it.
const (
	ScopeGeneral   = "general"
	ScopeDeveloper = "developer"
)

// Key mirrors the api_keys table. Authentication always uses the SHA-256 hash;
// SecretEncrypted is a decryptable copy kept for test keys only.
type Key struct {
	ID            string  `gorm:"primaryKey;size:64" json:"id"`
	MerchantID    string  `gorm:"column:merchant_id" json:"-"`
	ApplicationID *string `gorm:"column:application_id" json:"application_id,omitempty"`
	// DeveloperAccessID binds the key to one developer grant. Nil means general.
	DeveloperAccessID *string `gorm:"column:developer_access_id" json:"developer_access_id,omitempty"`
	Environment       string  `gorm:"column:environment" json:"environment"`
	Type              string  `gorm:"column:type" json:"type"`
	Prefix            string  `gorm:"column:prefix" json:"prefix"`
	Name              string  `gorm:"column:name" json:"name"`
	KeyHash           string  `gorm:"column:key_hash" json:"-"`
	// SecretEncrypted is the vault-encrypted plaintext of a test key, so it can be
	// shown again without rotating the integration. Nil for live keys (shown once
	// at creation) and for keys minted before recovery existed.
	SecretEncrypted []byte     `gorm:"column:secret_encrypted" json:"-"`
	SecretMasked    string     `gorm:"column:secret_masked" json:"secret_masked"`
	CreatedBy       string     `gorm:"column:created_by" json:"created_by"`
	LastUsedAt      *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
	ExpiresAt       *time.Time `gorm:"column:expires_at" json:"expires_at"`
	RevokedAt       *time.Time `gorm:"column:revoked_at" json:"revoked_at"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// Recoverable reports whether the plaintext can be shown again. Only live keys
// are write-only, and only because they are the ones that move real money.
func (k *Key) Recoverable() bool {
	return k != nil && k.Environment == constants.EnvironmentTest && len(k.SecretEncrypted) > 0
}

// TableName binds the model to the api_keys table.
func (Key) TableName() string { return "api_keys" }

// Status derives the lifecycle state for the public DTO.
func (k *Key) Status(now time.Time) string {
	switch {
	case k.RevokedAt != nil:
		return constants.APIKeyStatusRevoked
	case k.ExpiresAt != nil && k.ExpiresAt.Before(now):
		return constants.APIKeyStatusExpired
	default:
		return constants.APIKeyStatusActive
	}
}

// Usable reports whether the key may authenticate a request.
func (k *Key) Usable(now time.Time) error {
	if k.RevokedAt != nil {
		return errRevoked
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(now) {
		return errExpired
	}
	if k.Type == string(constants.APIKeyWebhook) {
		return errNotAuthenticable
	}
	return nil
}

// ApplicationIDValue returns the application id or "".
func (k *Key) ApplicationIDValue() string {
	if k.ApplicationID == nil {
		return ""
	}
	return *k.ApplicationID
}

// DeveloperAccessIDValue returns the developer grant id or "".
func (k *Key) DeveloperAccessIDValue() string {
	if k.DeveloperAccessID == nil {
		return ""
	}
	return *k.DeveloperAccessID
}

// Scope reports whether the key is shared (general) or tied to one developer.
func (k *Key) Scope() string {
	if k == nil || k.DeveloperAccessID == nil || *k.DeveloperAccessID == "" {
		return ScopeGeneral
	}
	return ScopeDeveloper
}
