// Package identity defines the authenticated request identity shared by every
// module, plus password hashing used by the auth module.
package identity

import (
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// Authentication modes.
const (
	ModeSession = "session" // dashboard JWT
	ModeAPIKey  = "apikey"  // pc_live_sk_… / pc_test_sk_…

	CookieAccessToken  = "payconnect_access_token"
	CookieRefreshToken = "payconnect_refresh_token"
)

// Principal is the authenticated caller attached to every request.
//
// OrgID is kept as an alias of MerchantID for the dashboard modules that were
// written before the merchant rename.
type Principal struct {
	Mode          string
	MerchantID    string
	OrgID         string
	UserID        string
	User          any // *merchants.User in session mode
	Environment   string
	APIKeyID      string
	KeyType       string
	ApplicationID string
	// DeveloperAccessID is set for developer-scoped keys; it names the
	// developer grant the key belongs to.
	DeveloperAccessID string
	// DeveloperPermissions are the permissions carried by that grant.
	DeveloperPermissions []string
	// Application carries the legacy application context for keys that are
	// scoped to one application (dashboard "applications" feature).
	Application any
	RequestID   string
}

// IsSession reports whether the caller authenticated with a dashboard session.
func (p *Principal) IsSession() bool { return p != nil && p.Mode == ModeSession }

// IsAPIKey reports whether the caller authenticated with a developer API key.
func (p *Principal) IsAPIKey() bool { return p != nil && p.Mode == ModeAPIKey }

// From returns the request principal (nil when unauthenticated).
func From(c *fiber.Ctx) *Principal {
	p, _ := c.Locals("identity").(*Principal)
	return p
}

// WithPrincipal attaches a principal to the request.
func WithPrincipal(c *fiber.Ctx, p *Principal) { c.Locals("identity", p) }

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword reports whether the password matches the stored hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
