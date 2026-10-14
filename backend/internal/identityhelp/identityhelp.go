// Package identityhelp holds the tiny actor helpers shared by handlers, so every
// handler reports the same actor name and role to the audit log.
package identityhelp

import (
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// ActorName returns the acting user's name, or "system" for API-key callers.
func ActorName(principal *identity.Principal) string {
	if principal == nil {
		return "system"
	}
	if user, ok := principal.User.(*merchants.User); ok {
		return user.Name
	}
	return "system"
}

// ActorRole returns the acting user's role, or "system" for API-key callers.
func ActorRole(principal *identity.Principal) string {
	if principal == nil {
		return "system"
	}
	if user, ok := principal.User.(*merchants.User); ok {
		return user.Role
	}
	return "system"
}
