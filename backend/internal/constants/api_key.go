package constants

// APIKeyType distinguishes what a key can do.
type APIKeyType string

const (
	APIKeySecret  APIKeyType = "secret"  // pc_live_sk_… / pc_test_sk_…
	APIKeyPublic  APIKeyType = "public"  // pc_live_pk_… (browser-safe)
	APIKeyWebhook APIKeyType = "webhook" // pc_live_wh_… (signing secret)
)

// Key prefix layout: <API_KEY_PREFIX>_<env>_<type>_<random>
const (
	KeyPrefixDefault = "pc"

	KeyEnvTest = "test"
	KeyEnvLive = "live"

	KeyTypeSecret  = "sk"
	KeyTypePublic  = "pk"
	KeyTypeWebhook = "wh"
)

// KeyTypeMarker maps a key type to its env-embedded marker.
func KeyTypeMarker(t APIKeyType) string {
	switch t {
	case APIKeyPublic:
		return KeyTypePublic
	case APIKeyWebhook:
		return KeyTypeWebhook
	default:
		return KeyTypeSecret
	}
}

// IsValidAPIKeyType reports whether t is a known key type.
func IsValidAPIKeyType(t APIKeyType) bool {
	return t == APIKeySecret || t == APIKeyPublic || t == APIKeyWebhook
}

// API key lifecycle states.
const (
	APIKeyStatusActive  = "active"
	APIKeyStatusRevoked = "revoked"
	APIKeyStatusExpired = "expired"
)
