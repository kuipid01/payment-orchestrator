package payments

import apperrors "github.com/kuipid01/payconnect/internal/errors"

// apperrorsForbiddenSession rejects session-authenticated payment creation:
// money movement must be attributable to a scoped, revocable API key.
func apperrorsForbiddenSession() error {
	return apperrors.PermissionDenied("payments.create requires a developer API key.")
}
