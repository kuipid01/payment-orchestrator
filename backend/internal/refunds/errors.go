package refunds

import apperrors "github.com/kuipid01/payconnect/internal/errors"

// permissionDeniedSession rejects session-authenticated refunds: a refund moves
// money out, so it must be attributable to a scoped, revocable API key.
func permissionDeniedSession() error {
	return apperrors.PermissionDenied("refunds.create requires a developer API key.")
}
