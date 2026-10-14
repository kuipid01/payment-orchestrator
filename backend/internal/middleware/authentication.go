package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Authenticator resolves a bearer token into a principal. The auth module
// implements it; middleware depends only on this interface (no import cycle).
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*identity.Principal, error)
}

// Authentication requires a valid bearer token and attaches the principal to
// the request. Session tokens and developer API keys share this gate; each
// endpoint decides which mode it accepts via SessionOnly/APIKeyOnly.
func Authentication(authn Authenticator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(identity.CookieAccessToken)
		if token == "" {
			header := c.Get(fiber.HeaderAuthorization)
			if !strings.HasPrefix(header, "Bearer ") {
				return apperrors.AuthenticationRequired()
			}
			token = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		}
		if token == "" {
			return apperrors.AuthenticationRequired()
		}
		principal, err := authn.Authenticate(c.UserContext(), token)
		if err != nil {
			return apperrors.From(err)
		}
		principal.RequestID = RequestIDValue(c)
		identity.WithPrincipal(c, principal)
		return c.Next()
	}
}

// SessionOnly rejects developer API keys: the dashboard credential endpoints
// must never be reachable with a key a merchant hands to a third party.
func SessionOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		p := identity.From(c)
		if p == nil || !p.IsSession() {
			return apperrors.PermissionDenied("This endpoint requires a merchant session, not an API key.")
		}
		return c.Next()
	}
}

// APIKeyOnly is the mirror of SessionOnly, used by developer-facing endpoints.
func APIKeyOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		p := identity.From(c)
		if p == nil || !p.IsAPIKey() {
			return apperrors.PermissionDenied("This endpoint requires a developer API key.")
		}
		return c.Next()
	}
}

// RequireRoles limits a route to session users holding one of the given roles.
// Role-bearing principals expose RoleName(); API keys never satisfy this gate.
func RequireRoles(roles ...string) fiber.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *fiber.Ctx) error {
		principal := identity.From(c)
		if principal == nil {
			return apperrors.AuthenticationRequired()
		}
		if principal.IsAPIKey() {
			return apperrors.PermissionDenied("API keys cannot perform dashboard administration.")
		}
		role, ok := principal.User.(interface{ RoleName() string })
		if !ok {
			return apperrors.PermissionDenied("")
		}
		if !allowed[role.RoleName()] {
			return apperrors.PermissionDenied("Your role does not allow this action.")
		}
		return c.Next()
	}
}
