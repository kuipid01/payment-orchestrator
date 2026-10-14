package billing

import (
	"net/http"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// Limit is the public shape of an upgrade wall: which plan the caller is on,
// which metric they hit and what that plan allows. The dashboard renders the
// prompt from this, so it never has to restate a limit of its own.
type Limit struct {
	Plan     string `json:"plan"`
	PlanName string `json:"plan_name"`
	Metric   string `json:"metric"`
	Limit    int    `json:"limit"`
	Current  int    `json:"current"`
	// UpgradeTo is the plan that lifts the limit, so the CTA can name it.
	UpgradeTo string `json:"upgrade_to"`
}

// UpgradeRequired builds the error a gate returns when a plan's limit is
// reached. It is a 402: the request is understood and refused for a billing
// reason, which is exactly what a client should switch on.
func UpgradeRequired(limit Limit) *apperrors.AppError {
	if limit.UpgradeTo == "" {
		limit.UpgradeTo = PlanLite
	}
	message := "Your " + limit.PlanName + " plan does not include this."
	switch {
	case limit.Metric != "" && limit.Limit <= 0:
		// A capability gate rather than a count: Free does not include routing
		// rules at all, and "includes 0 routing rules" would be an odd way to say so.
		message = "Your " + limit.PlanName + " plan does not include " + limit.Metric + "."
	case limit.Metric != "":
		// The refusal states the arithmetic the merchant can check: what the plan
		// includes, and what they already have. "Plan includes 1 application"
		// alone reads like a fact they are meant to argue with.
		message = "Your " + limit.PlanName + " plan includes " + plural(limit.Limit, limit.Metric) +
			", and this organization is already using " + plural(limit.Current, limit.Metric) + "."
	}
	return apperrors.New("billing_error", apperrors.CodeUpgradeRequired, message, http.StatusPaymentRequired).
		WithDetails(limit)
}

// PlanNotFound is returned for a plan code the catalog does not sell.
func PlanNotFound() *apperrors.AppError {
	return apperrors.NotFound(apperrors.CodePlanNotFound, "That plan does not exist.")
}

// SubscriptionNotFound is returned when a merchant has no paid subscription to
// cancel or resume.
func SubscriptionNotFound() *apperrors.AppError {
	return apperrors.NotFound(apperrors.CodeSubscriptionNotFound,
		"You are not on a paid plan.")
}

// TrialAlreadyUsed is returned when a second trial is requested. The trial is
// once per organization, permanently.
func TrialAlreadyUsed() *apperrors.AppError {
	return apperrors.Conflict(apperrors.CodeTrialAlreadyUsed,
		"This organization has already used its free trial.")
}

// BillingNotConfigured is returned by checkout while no collection provider is
// wired: the request is recorded, but nothing is charged and no checkout URL
// exists yet.
func BillingNotConfigured() *apperrors.AppError {
	return apperrors.ServiceUnavailable(
		"Billing is not open yet: no payment provider is connected to PayConnect. Your choice is recorded and you will not be charged.",
		nil)
}

// plural renders a count with its noun: "1 application", "3 applications".
func plural(n int, noun string) string {
	if n == 1 {
		return itoa(n) + " " + noun
	}
	return itoa(n) + " " + noun + "s"
}

// itoa renders a limit for a human message. Negative means unlimited, which
// never reaches here, but must not print as "-1" if it does.
func itoa(n int) string {
	if n < 0 {
		return "unlimited"
	}
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
