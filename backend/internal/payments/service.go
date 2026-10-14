package payments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/events"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/providers"
	"github.com/kuipid01/payconnect/internal/redis"
)

/* --------------------------------- ports ------------------------------------ */
// These narrow interfaces keep the payments service free of imports from the
// provider-credentials, routing, customer and ledger domains, and make every
// external dependency substitutable in tests.

// CredentialProvider supplies decrypted provider credentials.
type CredentialProvider interface {
	CredentialsFor(merchantID, environment string, provider constants.Provider) (providers.Credentials, error)
}

// ProviderResolver orders the providers to try for a payment.
type ProviderResolver interface {
	Resolve(merchantID, environment, currency string, amountMinor int64, requested constants.Provider) ([]constants.Provider, error)
}

// CustomerResolver finds or creates the payer record.
type CustomerResolver interface {
	FindOrCreate(merchantID string, in CustomerInput) (string, error)
}

// CustomerDirectory resolves payer details for display. Payments store only the
// customer id, so listings would otherwise show an opaque identifier.
type CustomerDirectory interface {
	Lookup(merchantID string, ids []string) (map[string]CustomerRef, error)
}

// Ledger records normalized financial events.
type Ledger interface {
	RecordPayment(ctx context.Context, payment *Payment) error
}

// TestConnectionResolver validates that a provider connection may be used for a
// dashboard-initiated sandbox charge and reports its provider and environment.
// The providers domain implements it, so the payments domain never reads
// connection rows itself.
type TestConnectionResolver interface {
	TestableConnection(merchantID, connectionID string) (constants.Provider, string, error)
}

// Service orchestrates payment creation, verification and reads.
type Service struct {
	Repo      *Repository
	Providers CredentialProvider
	Registry  providers.Registry
	Routing   ProviderResolver
	Customers CustomerResolver
	// Directory resolves the payer behind a payment for display. Optional: when
	// it is nil, reads still work and simply omit the customer block.
	Directory CustomerDirectory
	Ledger    Ledger
	Events    events.Publisher
	// TestConnections gates dashboard-initiated sandbox charges. Optional: when
	// nil, test transactions are unavailable and the endpoint reports so.
	TestConnections TestConnectionResolver
	Now             func() time.Time

	// Store is the shared key-value store backing the idempotency fast
	// pre-check and the execution lock. It is optional: when it is nil or
	// unreachable, idempotency falls back to the database alone, which remains
	// the source of truth either way.
	Store  redis.Store
	Logger *slog.Logger
	// IdempotencyTTL is how long a completed outcome stays cached for replay.
	// Zero uses the package default.
	IdempotencyTTL time.Duration
	// LockTTL bounds an in-flight execution lock before it expires on its own.
	// Zero uses the stale window.
	LockTTL time.Duration
}

// NewService builds the payment service.
func NewService(repo *Repository, registry providers.Registry, eventsBus events.Publisher) *Service {
	return &Service{Repo: repo, Registry: registry, Events: eventsBus, Now: time.Now}
}

// CreateInput is the internal, already-authenticated create request.
//
// The credential fields are attribution, not authorization: the caller has
// already been authenticated, and these only record which key took the payment
// so the dashboard can filter by application, developer and key afterwards.
type CreateInput struct {
	MerchantID        string
	ApplicationID     string
	Environment       string
	APIKeyID          string
	DeveloperAccessID string
	Request           CreateRequest
	IdempotencyKey    string
	// RequestID joins every line this flow logs to the request log.
	RequestID string
}

// Create executes the payment flow: validate → idempotency → route → provider →
// persist → publish.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Response, error) {
	req, amount, currency, reference, err := normalizeCreate(in)
	if err != nil {
		s.log("payment create rejected", "request_id", in.RequestID, "merchant_id", in.MerchantID,
			"environment", in.Environment, "api_key_id", in.APIKeyID, "reference", in.Request.Reference,
			"reason", err.Error())
		return nil, err
	}

	// One line per step, in order: this is the trace an operator follows to see
	// which key took a payment, which provider it was routed to, and how it
	// ended. No credential material is ever logged — only ids.
	s.log("payment create start", "request_id", in.RequestID, "merchant_id", in.MerchantID,
		"environment", in.Environment, "application_id", in.ApplicationID, "api_key_id", in.APIKeyID,
		"developer_access_id", in.DeveloperAccessID, "amount_minor", amount, "currency", currency,
		"reference", reference, "idempotency_key", in.IdempotencyKey)

	// The idempotency identity is the *client-supplied* request, so hash the raw
	// reference rather than the one minted above: a retry that omits the
	// reference must still hash the same, otherwise every retry of a request
	// without an explicit reference is misread as a different body.
	requestHash := hashRequest(amount, currency, strings.TrimSpace(req.Reference), req)

	if in.IdempotencyKey != "" {
		// Fast pre-check: a completed request is served straight from the shared
		// store, skipping the database round trip. This is a cache, not the
		// authority — a miss or an expired entry simply falls through below.
		if cached, ok := s.cachedIdempotency(ctx, in.MerchantID, in.IdempotencyKey); ok {
			s.log("payment create replayed from cache", "request_id", in.RequestID,
				"idempotency_key", in.IdempotencyKey, "reference", reference)
			if cached.RequestHash != requestHash {
				return nil, apperrors.IdempotencyKeyReused()
			}
			if appErr := storedError(cached); appErr != nil {
				return nil, appErr
			}
			if replayed := replayResponse(cached); replayed != nil {
				return replayed, nil
			}
			// A malformed cache entry must never shadow the database: fall through.
		}

		// Distributed lock: only one request may execute a given key at a time, so
		// a concurrent retry is refused cheaply instead of racing on inserts. The
		// lock is best-effort; when the store is unavailable the database's unique
		// constraint arbitrates exactly as it did before.
		lockToken, locked, usable := s.lockIdempotency(ctx, in.MerchantID, in.IdempotencyKey)
		if usable && !locked {
			return nil, apperrors.IdempotencyConflict()
		}
		if usable {
			defer s.unlockIdempotency(ctx, in.MerchantID, in.IdempotencyKey, lockToken)
		}

		existing, created, err := s.Repo.BeginIdempotency(&IdempotencyRecord{
			ID:             gormx.NewID("idem"),
			MerchantID:     in.MerchantID,
			IdempotencyKey: in.IdempotencyKey,
			Endpoint:       "POST /v1/payments",
			RequestHash:    requestHash,
			Status:         idempotencyInProgress,
			CreatedAt:      s.Now().UTC(),
			UpdatedAt:      s.Now().UTC(),
		})
		if err != nil {
			return nil, err
		}
		// created == true means this very request reserved the key, so it owns
		// the work. Anything else is a retry of an earlier request.
		if !created {
			switch {
			case existing.RequestHash != requestHash:
				// Same key, different body: the client is trying to reuse a key.
				return nil, apperrors.IdempotencyKeyReused()
			case existing.Status == idempotencyCompleted:
				s.log("payment create replayed from database", "request_id", in.RequestID,
					"idempotency_key", in.IdempotencyKey, "reference", reference)
				// A finished request is replayed verbatim — including its failure. A
				// retry of a request that failed must return the same error, not a
				// partial payment body with a success status. Refresh the cache so the
				// next retry takes the fast path.
				s.rememberIdempotency(ctx, existing)
				if appErr := storedError(existing); appErr != nil {
					return nil, appErr
				}
				return replayResponse(existing), nil
			case existing.CreatedAt.Add(idempotencyStaleWindow).After(s.Now()):
				// Another request holds the key right now.
				return nil, apperrors.IdempotencyConflict()
				// Otherwise the in-progress record is stale (crashed request) and
				// this retry takes it over.
			}
		}
	}

	providersToTry, err := s.resolveProviders(in, amount, currency, req)
	if err != nil {
		s.log("payment routing failed", "request_id", in.RequestID, "reference", reference,
			"environment", in.Environment, "requested_adapter", req.Adapter, "reason", err.Error())
		return nil, err
	}
	s.log("payment provider chain resolved", "request_id", in.RequestID, "reference", reference,
		"environment", in.Environment, "providers", providerNames(providersToTry), "fallback", req.Fallback)

	now := s.Now().UTC()
	payment := &Payment{
		ID:             gormx.NewID("pay"),
		MerchantID:     in.MerchantID,
		Environment:    in.Environment,
		Reference:      reference,
		Provider:       providersToTry[0].String(),
		AmountMinor:    amount,
		Currency:       currency,
		Status:         string(constants.PaymentPending),
		IdempotencyKey: in.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if in.ApplicationID != "" {
		applicationID := in.ApplicationID
		payment.ApplicationID = &applicationID
	}
	if in.APIKeyID != "" {
		keyID := in.APIKeyID
		payment.APIKeyID = &keyID
	}
	if in.DeveloperAccessID != "" {
		accessID := in.DeveloperAccessID
		payment.DeveloperAccessID = &accessID
	}
	payment.Description = strings.TrimSpace(req.Description)
	if req.Metadata != nil {
		payment.Metadata = gormx.JSONB[map[string]any]{Data: req.Metadata}
	}
	payment.appendEvent("created", "Payment created", now)

	// The customer record is resolved before the provider call so the payment
	// always has a stable payer identity, even if the provider rejects it.
	if s.Customers != nil && req.Customer.Email != "" {
		customerID, err := s.Customers.FindOrCreate(in.MerchantID, req.Customer)
		if err != nil {
			return nil, err
		}
		payment.CustomerID = &customerID
	}

	if err := s.Repo.Create(payment); err != nil {
		s.log("payment persistence failed", "request_id", in.RequestID, "reference", reference,
			"reason", err.Error())
		return nil, err
	}
	s.log("payment recorded", "request_id", in.RequestID, "payment_id", payment.ID,
		"reference", payment.Reference, "environment", payment.Environment, "api_key_id", in.APIKeyID,
		"developer_access_id", in.DeveloperAccessID, "application_id", in.ApplicationID,
		"customer_id", customerIDValue(payment.CustomerID))

	if err := s.execute(ctx, payment, providersToTry, req, in.Environment); err != nil {
		s.log("payment create finished with error", "request_id", in.RequestID, "payment_id", payment.ID,
			"status", payment.Status, "failure_code", payment.FailureCode, "reason", err.Error())
		// The payment row survives a provider failure: the caller can verify it.
		s.completeIdempotency(ctx, in, payment, requestHash, err)
		return nil, err
	}

	response := ToResponse(payment)
	s.publish(ctx, payment)
	s.completeIdempotencyCreated(ctx, in, payment, requestHash, response)
	return &response, nil
}

// execute calls the provider chain, applying the fallback rules.
func (s *Service) execute(ctx context.Context, payment *Payment, chain []constants.Provider, req CreateRequest, environment string) error {
	var lastErr error
	for index, provider := range chain {
		credentials, err := s.Providers.CredentialsFor(payment.MerchantID, environment, provider)
		if err != nil {
			lastErr = err
			continue
		}
		adapter, err := s.Registry.Get(provider)
		if err != nil {
			lastErr = err
			continue
		}

		payment.Provider = provider.String()
		payment.appendEvent("provider_request", "Request sent to "+provider.String(), s.Now().UTC())
		s.log("payment sent to provider", "payment_id", payment.ID, "provider", provider.String(),
			"reference", payment.Reference, "environment", environment, "attempt", index+1,
			"amount_minor", payment.AmountMinor, "currency", payment.Currency)

		result, err := adapter.CreatePayment(ctx, credentials, providers.PaymentRequest{
			Reference:   payment.Reference,
			AmountMinor: payment.AmountMinor,
			Currency:    payment.Currency,
			Email:       req.Customer.Email,
			Name:        req.Customer.Name,
			Phone:       req.Customer.Phone,
			Description: req.Description,
			CallbackURL: req.CallbackURL,
			Metadata:    req.Metadata,
		})
		if err != nil {
			lastErr = err
			// A timeout or an ambiguous provider error must NOT be retried and
			// must NOT be reported as a failure: the charge state is unknown, so
			// the payment stays processing until verification reconciles it.
			if !isSafelyRetryable(err) || index == len(chain)-1 || !req.Fallback {
				return s.markUnknownOrFailed(payment, err)
			}
			payment.appendEvent("fallback", "Falling back after "+provider.String()+" rejected the request", s.Now().UTC())
			continue
		}

		s.applyProviderResult(payment, result)
		if err := s.Repo.Save(payment); err != nil {
			return err
		}
		s.log("payment provider outcome applied", "payment_id", payment.ID, "provider", payment.Provider,
			"status", payment.Status, "provider_status", result.Status)
		return nil
	}
	if lastErr == nil {
		lastErr = apperrors.ProviderNotConnected("any provider")
	}
	return s.markUnknownOrFailed(payment, lastErr)
}

// markUnknownOrFailed records an unusable provider outcome.
//
// ProviderUnavailable and ProviderTimeout mean "unknown": the payment remains
// processing so verification can resolve it later. A definite request rejection
// marks the payment failed.
func (s *Service) markUnknownOrFailed(payment *Payment, cause error) error {
	now := s.Now().UTC()
	message := "payment marked failed"
	switch {
	case apperrors.Is(cause, apperrors.CodeProviderTimeout), apperrors.Is(cause, apperrors.CodeProviderUnavailable):
		payment.Status = string(constants.PaymentProcessing)
		payment.appendEvent("provider_timeout", "Provider outcome unknown; verification required", now)
		message = "payment outcome unknown; verification required"
	case apperrors.Is(cause, apperrors.CodeProviderNotConnected), apperrors.Is(cause, apperrors.CodeProviderNotSupported):
		payment.Status = string(constants.PaymentFailed)
		payment.FailureCode = "provider_unavailable"
		payment.appendEvent("failed", "No usable provider for this request", now)
	case apperrors.Is(cause, apperrors.CodeProviderAuthenticationFails):
		payment.Status = string(constants.PaymentFailed)
		payment.FailureCode = "provider_credentials_rejected"
		payment.FailureMessage = publicMessage(cause)
		payment.appendEvent("failed", "The provider rejected the stored credentials", now)
		message = "payment failed: provider rejected the stored credentials"
	default:
		payment.Status = string(constants.PaymentFailed)
		payment.FailureCode = "provider_rejected"
		payment.FailureMessage = publicMessage(cause)
		payment.appendEvent("failed", "Provider rejected the request", now)
	}
	s.log(message, "payment_id", payment.ID, "provider", payment.Provider,
		"status", payment.Status, "failure_code", payment.FailureCode, "reason", cause.Error())
	payment.UpdatedAt = now
	if err := s.Repo.Save(payment); err != nil {
		return err
	}
	if s.Ledger != nil {
		_ = s.Ledger.RecordPayment(context.Background(), payment)
	}
	if s.Events != nil {
		s.Events.Publish(context.Background(), paymentEvent(payment))
	}
	return cause
}

// TestCreateInput is a dashboard-initiated sandbox charge against one provider
// connection. It carries no API key or application: the merchant is driving it
// from the provider's page to exercise the whole webhook path.
type TestCreateInput struct {
	MerchantID   string
	ConnectionID string
	Request      CreateRequest
	RequestID    string
}

// CreateTest charges a sandbox provider connection so a merchant can verify its
// webhooks end to end, without writing any integration code.
//
// It deliberately delegates to Create so the test payment is a *real* payment:
// it is stored, it appears in the dashboard, the provider's notification
// reconciles against it, and the resulting event fans out to the merchant's
// endpoints. That is what makes the test meaningful — the same code path a live
// payment takes, driven by hand.
func (s *Service) CreateTest(ctx context.Context, in TestCreateInput) (*Response, error) {
	if s.TestConnections == nil {
		return nil, apperrors.ServiceUnavailable("Test transactions are not configured.", nil)
	}
	provider, environment, err := s.TestConnections.TestableConnection(in.MerchantID, in.ConnectionID)
	if err != nil {
		return nil, err
	}
	req := in.Request
	// Pin the charge to the connection the merchant chose, and never fall back:
	// a test must exercise this provider, not whatever routing would pick.
	req.Adapter = provider.String()
	req.Fallback = false
	if strings.TrimSpace(req.Customer.Email) == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "customer.email is required.")
	}
	s.log("test transaction requested", "request_id", in.RequestID, "merchant_id", in.MerchantID,
		"connection_id", in.ConnectionID, "provider", provider.String(), "environment", environment)
	return s.Create(ctx, CreateInput{
		MerchantID:  in.MerchantID,
		Environment: environment,
		Request:     req,
		RequestID:   in.RequestID,
	})
}

// Verify reconciles a payment with its provider. The provider is authoritative:
// the local database alone is never trusted for a settlement decision.
func (s *Service) Verify(ctx context.Context, merchantID, paymentID string) (*Response, error) {
	payment, err := s.Repo.ByID(merchantID, paymentID)
	if err != nil {
		return nil, apperrors.PaymentNotFound()
	}
	adapter, err := s.Registry.Get(constants.Provider(payment.Provider))
	if err != nil {
		return nil, err
	}
	credentials, err := s.Providers.CredentialsFor(merchantID, payment.Environment, constants.Provider(payment.Provider))
	if err != nil {
		return nil, err
	}
	result, err := adapter.VerifyPayment(ctx, credentials, payment.Reference)
	if err != nil {
		s.log("payment verification failed", "payment_id", payment.ID, "provider", payment.Provider,
			"reference", payment.Reference, "reason", err.Error())
		return nil, err
	}
	s.applyProviderResult(payment, result)
	if err := s.Repo.Save(payment); err != nil {
		return nil, err
	}
	s.publish(ctx, payment)
	s.log("payment verified", "payment_id", payment.ID, "provider", payment.Provider,
		"status", payment.Status, "provider_status", result.Status)
	response := ToResponse(payment)
	return &response, nil
}

// VerifyByReference reconciles using the merchant-supplied reference.
func (s *Service) VerifyByReference(ctx context.Context, merchantID, environment, reference string) (*Response, error) {
	payment, err := s.Repo.ByReference(merchantID, environment, reference)
	if err != nil {
		return nil, apperrors.PaymentNotFound()
	}
	return s.Verify(ctx, merchantID, payment.ID)
}

// ReconcileWebhook applies a normalized inbound provider event to the matching
// payment. The provider notification is the authoritative settlement signal, so
// the same state machine used by Verify is applied here.
func (s *Service) ReconcileWebhook(ctx context.Context, merchantID, provider string, result *providers.WebhookResult) (*Payment, error) {
	if result == nil {
		return nil, apperrors.WebhookEventInvalid("")
	}
	payment, err := s.Repo.ByProviderReference(merchantID, provider, result.ProviderReference)
	if err != nil {
		payment, err = s.Repo.ByProviderReference(merchantID, provider, result.Reference)
	}
	if err != nil {
		return nil, apperrors.PaymentNotFound()
	}

	switch result.Type {
	case constants.EventRefundSucceeded:
		if result.AmountMinor > 0 && result.AmountMinor < payment.AmountMinor {
			payment.Status = string(constants.PaymentPartiallyRefunded)
		} else {
			payment.Status = string(constants.PaymentRefunded)
		}
		payment.AppendEvent("refund", "Refund confirmed by "+provider, s.Now().UTC())
		payment.UpdatedAt = s.Now().UTC()
	default:
		if result.Status == "" {
			// The event carries no state transition (for example a provider
			// notice): record it on the timeline and leave the status alone.
			if result.ProviderReference != "" {
				payment.ProviderPaymentID = &result.ProviderReference
			}
			payment.AppendEvent("provider_event", "Provider event "+result.Type, s.Now().UTC())
			payment.UpdatedAt = s.Now().UTC()
		} else {
			s.applyProviderResult(payment, &providers.PaymentResult{
				ProviderReference: result.ProviderReference,
				Status:            result.Status,
				AmountMinor:       result.AmountMinor,
				Currency:          result.Currency,
				ProviderResponse:  result.Raw,
			})
		}
	}
	if err := s.Repo.Save(payment); err != nil {
		return nil, err
	}
	// The ledger is updated here; the internal event is published by the caller
	// (the webhook gateway), so one inbound notification produces exactly one
	// merchant delivery.
	if s.Ledger != nil {
		_ = s.Ledger.RecordPayment(ctx, payment)
	}
	return payment, nil
}

// applyProviderResult maps a provider result onto the payment, enforcing the
// state machine: a terminal state is never overwritten by a later, weaker one.
func (s *Service) applyProviderResult(payment *Payment, result *providers.PaymentResult) {
	if result == nil {
		return
	}
	now := s.Now().UTC()
	current := constants.PaymentStatus(payment.Status)
	next := constants.PaymentStatus(result.Status)

	if current.IsTerminal() && next != current {
		switch {
		case current == constants.PaymentRefunded || current == constants.PaymentPartiallyRefunded:
			// Refund state outranks a later duplicate success notification.
			return
		case current == constants.PaymentSuccessful && next == constants.PaymentProcessing:
			return // never regress a settled payment to processing
		}
	}

	if result.ProviderReference != "" {
		reference := result.ProviderReference
		payment.ProviderPaymentID = &reference
	}
	if result.AuthorizationURL != "" {
		payment.AuthorizationURL = result.AuthorizationURL
	}
	if result.AmountMinor > 0 && result.AmountMinor != payment.AmountMinor {
		// Trust the provider amount, but keep our record of what we asked for.
		payment.appendEvent("amount_mismatch", "Provider reported a different amount", now)
	}
	if result.FeeMinor != nil {
		payment.FeeMinor = result.FeeMinor
	}
	if result.ProviderResponse != nil {
		payment.ProviderResponse = gormx.JSONB[map[string]any]{Data: result.ProviderResponse}
	}

	payment.Status = string(next)
	if next == constants.PaymentFailed {
		payment.FailureCode = firstNonEmpty(result.FailureCode, "provider_declined")
		payment.FailureMessage = result.FailureMessage
		payment.appendEvent("failed", "Provider reported a failure", now)
	} else if next == constants.PaymentSuccessful {
		payment.FailureCode = ""
		payment.FailureMessage = ""
		payment.appendEvent("successful", "Payment confirmed by "+payment.Provider, now)
	} else if next == constants.PaymentProcessing {
		payment.appendEvent("pending", "Awaiting customer authorization", now)
	}
	payment.UpdatedAt = now
}

// Get returns one payment, enriched with the payer behind it.
func (s *Service) Get(merchantID, paymentID string) (*Response, error) {
	payment, err := s.Repo.ByID(merchantID, paymentID)
	if err != nil {
		return nil, apperrors.PaymentNotFound()
	}
	response := ToResponse(payment)
	s.attachCustomers(merchantID, []*Response{&response})
	return &response, nil
}

// List returns a filtered page of payments, the total the filter matches, and
// the per-status tally of the same filter. The three come from one predicate, so
// the page, its total and its facets can never disagree.
func (s *Service) List(f Filter) ([]Response, int64, map[string]int64, error) {
	rows, err := s.Repo.List(f)
	if err != nil {
		return nil, 0, nil, err
	}
	total, err := s.Repo.Count(f)
	if err != nil {
		return nil, 0, nil, err
	}
	summary, err := s.Repo.StatusCounts(f)
	if err != nil {
		return nil, 0, nil, err
	}
	out := make([]Response, 0, len(rows))
	pointers := make([]*Response, 0, len(rows))
	for i := range rows {
		out = append(out, ToResponse(&rows[i]))
	}
	for i := range out {
		pointers = append(pointers, &out[i])
	}
	s.attachCustomers(f.MerchantID, pointers)
	return out, total, summary, nil
}

// attachCustomers fills in the payer block for a page of payments with a single
// lookup, so a fifty-row listing is still two queries. A missing directory or an
// unknown customer leaves the block empty rather than failing the read.
func (s *Service) attachCustomers(merchantID string, responses []*Response) {
	if s.Directory == nil || len(responses) == 0 {
		return
	}
	ids := make([]string, 0, len(responses))
	seen := make(map[string]struct{}, len(responses))
	for _, response := range responses {
		if response.CustomerID == "" {
			continue
		}
		if _, ok := seen[response.CustomerID]; ok {
			continue
		}
		seen[response.CustomerID] = struct{}{}
		ids = append(ids, response.CustomerID)
	}
	if len(ids) == 0 {
		return
	}
	customers, err := s.Directory.Lookup(merchantID, ids)
	if err != nil {
		return
	}
	for _, response := range responses {
		if customer, ok := customers[response.CustomerID]; ok {
			resolved := customer
			response.Customer = &resolved
		}
	}
}

// Metrics returns the dashboard aggregate for one environment.
func (s *Service) Metrics(merchantID, environment, currency string) MetricsResponse {
	volume, successful, failed, series, err := s.Repo.Metrics(merchantID, environment)
	if err != nil {
		return MetricsResponse{Currency: currency}
	}
	rate := 0.0
	if total := successful + failed; total > 0 {
		rate = float64(successful) / float64(total) * 100
	}
	out := MetricsResponse{
		VolumeMinor: volume,
		Currency:    currency,
		Successful:  successful,
		Failed:      failed,
		SuccessRate: rate,
	}
	for _, point := range series {
		out.VolumeSeries = append(out.VolumeSeries, SeriesPoint{
			Date:        point.Day.Format("Jan 2"),
			VolumeMinor: point.VolumeMinor,
		})
	}
	return out
}

/* --------------------------------- helpers ---------------------------------- */

func normalizeCreate(in CreateInput) (CreateRequest, int64, string, string, error) {
	req := in.Request
	amount := req.AmountMinor
	if amount == 0 {
		amount = req.Amount
	}
	if amount < constants.MinPaymentAmountMinor || amount > constants.MaxPaymentAmountMinor {
		return req, 0, "", "", apperrors.InvalidAmount("amount must be between 1.00 and 5,000,000.00 in minor units.")
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = constants.DefaultCurrency
	}
	if !constants.IsSupportedCurrency(currency) {
		return req, 0, "", "", apperrors.InvalidCurrency("The currency is not supported.")
	}
	reference := strings.TrimSpace(req.Reference)
	if reference == "" {
		reference = "ref_" + gormx.NewID("")
		reference = strings.ReplaceAll(reference, "ref__", "ref_")
	}
	if len(reference) > 128 {
		return req, 0, "", "", apperrors.Validation(apperrors.CodeInvalidReference, "reference must be at most 128 characters.")
	}
	if req.Customer.Email == "" && req.Email != "" {
		req.Customer.Email = strings.TrimSpace(req.Email)
	}
	if req.Customer.Email == "" {
		return req, 0, "", "", apperrors.Validation(apperrors.CodeValidationError, "customer.email is required.")
	}
	return req, amount, currency, reference, nil
}

func (s *Service) resolveProviders(in CreateInput, amount int64, currency string, req CreateRequest) ([]constants.Provider, error) {
	requested := constants.Provider(strings.TrimSpace(req.Adapter))
	if requested == "" {
		requested = constants.ProviderAuto
	}
	if requested != constants.ProviderAuto && !constants.IsValidProvider(requested) {
		return nil, apperrors.InvalidProvider("adapter must be one of paystack, flutterwave, moniepoint or auto.")
	}
	if s.Routing != nil {
		chain, err := s.Routing.Resolve(in.MerchantID, in.Environment, currency, amount, requested)
		if err == nil && len(chain) > 0 {
			if !req.Fallback {
				chain = chain[:1]
			}
			return chain, nil
		}
		if err != nil && requested != constants.ProviderAuto {
			return nil, err
		}
	}
	if requested == constants.ProviderAuto {
		return nil, apperrors.ProviderNotConnected("any provider")
	}
	return []constants.Provider{requested}, nil
}

func (s *Service) publish(ctx context.Context, payment *Payment) {
	if s.Ledger != nil {
		if err := s.Ledger.RecordPayment(ctx, payment); err != nil && s.Events == nil {
			return
		}
	}
	if s.Events != nil {
		s.Events.Publish(ctx, paymentEvent(payment))
	}
}

func paymentEvent(payment *Payment) events.Event {
	eventType := constants.EventPaymentCreated
	switch constants.PaymentStatus(payment.Status) {
	case constants.PaymentSuccessful:
		eventType = constants.EventPaymentSucceeded
	case constants.PaymentFailed:
		eventType = constants.EventPaymentFailed
	case constants.PaymentCancelled:
		eventType = constants.EventPaymentCancelled
	case constants.PaymentProcessing:
		eventType = constants.EventPaymentProcessing
	}
	event := events.New(eventType, payment.MerchantID, payment.Environment)
	event.PaymentID = payment.ID
	event.Reference = payment.Reference
	payload, err := json.Marshal(ToResponse(payment))
	if err == nil {
		event.Data = payload
	}
	return event
}

// completeIdempotency records the outcome of a request whose provider call
// failed, so a retry replays the same error instead of re-executing.
func (s *Service) completeIdempotency(ctx context.Context, in CreateInput, payment *Payment, requestHash string, cause error) {
	if in.IdempotencyKey == "" {
		return
	}
	body := map[string]any{"id": payment.ID, "status": payment.Status, "reference": payment.Reference}
	status := http.StatusCreated
	if cause != nil {
		appErr := apperrors.From(cause)
		status = appErr.HTTPStatus
		public := appErr.Public()
		public["type"] = appErr.Type
		body["error"] = public
	}
	s.finishIdempotency(ctx, completedRecord(in, requestHash, status, body, payment.ID))
}

func (s *Service) completeIdempotencyCreated(ctx context.Context, in CreateInput, payment *Payment, requestHash string, response Response) {
	if in.IdempotencyKey == "" {
		return
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		return
	}
	s.finishIdempotency(ctx, completedRecord(in, requestHash, http.StatusCreated, body, payment.ID))
}

// completedRecord builds the finished-request record for persistence and caching.
func completedRecord(in CreateInput, requestHash string, status int, body map[string]any, paymentID string) *IdempotencyRecord {
	record := &IdempotencyRecord{
		MerchantID:     in.MerchantID,
		IdempotencyKey: in.IdempotencyKey,
		RequestHash:    requestHash,
		Status:         idempotencyCompleted,
		ResponseBody:   gormx.JSONB[map[string]any]{Data: body},
	}
	if status != 0 {
		code := status
		record.StatusCode = &code
	}
	if paymentID != "" {
		id := paymentID
		record.PaymentID = &id
	}
	return record
}

// finishIdempotency marks the request complete in the database and, only on
// success, refreshes the shared-store cache. The database is the source of
// truth: the cache must never claim a completion the database does not hold, so
// a failed write leaves the cache untouched and the next retry re-reads the
// database.
func (s *Service) finishIdempotency(ctx context.Context, record *IdempotencyRecord) {
	if record == nil || record.MerchantID == "" || record.IdempotencyKey == "" {
		return
	}
	status := 0
	if record.StatusCode != nil {
		status = *record.StatusCode
	}
	paymentID := ""
	if record.PaymentID != nil {
		paymentID = *record.PaymentID
	}
	if err := s.Repo.CompleteIdempotency(record.MerchantID, record.IdempotencyKey, status, record.ResponseBody.Data, paymentID); err != nil {
		s.warn("idempotency completion failed", err)
		return
	}
	s.rememberIdempotency(ctx, record)
}

/* ------------------------- idempotency shared store ------------------------- */

// Shared-store key namespaces for idempotency. They sit apart from the rate
// limit (rl:) and lockout (authfail:) keys, and the store prefixes every key
// with the configured REDIS_PREFIX.
const (
	idempotencyCachePrefix = "idem:"
	idempotencyLockPrefix  = "idemlock:"
)

const (
	// idempotencyStaleWindow bounds how long an unfinished key blocks a
	// duplicate before a retry is allowed to take it over.
	idempotencyStaleWindow = 30 * time.Second
	// defaultIdempotencyCacheTTL is how long a completed outcome stays cached.
	// Shorter than the database's retention is safe: an expired entry only costs
	// a database round trip on the next replay.
	defaultIdempotencyCacheTTL = 24 * time.Hour
)

// cachedIdempotency is the compact form of a completed idempotent request kept
// in the shared store for a fast replay.
type cachedIdempotency struct {
	RequestHash string         `json:"request_hash"`
	StatusCode  int            `json:"status_code"`
	Body        map[string]any `json:"body"`
}

// record rehydrates the cache entry into the record shape the existing replay
// helpers already understand.
func (c cachedIdempotency) record() *IdempotencyRecord {
	status := c.StatusCode
	return &IdempotencyRecord{
		RequestHash:  c.RequestHash,
		Status:       idempotencyCompleted,
		StatusCode:   &status,
		ResponseBody: gormx.JSONB[map[string]any]{Data: c.Body},
	}
}

func idempotencyCacheKey(merchantID, key string) string {
	return idempotencyCachePrefix + merchantID + ":" + key
}

func idempotencyLockKey(merchantID, key string) string {
	return idempotencyLockPrefix + merchantID + ":" + key
}

// cachedIdempotency returns a completed outcome stored for this key. A nil
// store, a miss, or any store error is reported as "not cached" so the caller
// falls back to the database.
func (s *Service) cachedIdempotency(ctx context.Context, merchantID, key string) (*IdempotencyRecord, bool) {
	if s.Store == nil {
		return nil, false
	}
	raw, ok, err := s.Store.Get(ctx, idempotencyCacheKey(merchantID, key))
	if err != nil {
		s.warn("idempotency cache read failed", err)
		return nil, false
	}
	if !ok {
		return nil, false
	}
	var cached cachedIdempotency
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		s.warn("idempotency cache decode failed", err)
		return nil, false
	}
	return cached.record(), true
}

// rememberIdempotency caches a completed outcome for fast replay. Best effort:
// a store failure only costs a database round trip on the next retry.
func (s *Service) rememberIdempotency(ctx context.Context, record *IdempotencyRecord) {
	if s.Store == nil || record == nil {
		return
	}
	cached := cachedIdempotency{RequestHash: record.RequestHash, Body: record.ResponseBody.Data}
	if record.StatusCode != nil {
		cached.StatusCode = *record.StatusCode
	}
	encoded, err := json.Marshal(cached)
	if err != nil {
		return
	}
	if err := s.Store.Set(ctx, idempotencyCacheKey(record.MerchantID, record.IdempotencyKey), string(encoded), s.idempotencyCacheTTL()); err != nil {
		s.warn("idempotency cache write failed", err)
	}
}

// lockIdempotency takes the execution lock for a key. usable is false when the
// store is absent or unreachable, in which case the caller proceeds without a
// lock and relies on the database's unique constraint.
func (s *Service) lockIdempotency(ctx context.Context, merchantID, key string) (token string, locked, usable bool) {
	if s.Store == nil {
		return "", false, false
	}
	token = gormx.NewID("lock")
	acquired, err := s.Store.SetNX(ctx, idempotencyLockKey(merchantID, key), token, s.idempotencyLockTTL())
	if err != nil {
		s.warn("idempotency lock store unavailable; relying on the database", err)
		return "", false, false
	}
	return token, acquired, true
}

// unlockIdempotency releases a lock we own. The ownership check is not atomic
// with the delete, but a lock whose TTL expired and was re-acquired is still
// only a missed optimization: the database remains the authority on idempotency
// and its unique constraint cannot be violated.
func (s *Service) unlockIdempotency(ctx context.Context, merchantID, key, token string) {
	if s.Store == nil || token == "" {
		return
	}
	lockKey := idempotencyLockKey(merchantID, key)
	if current, ok, err := s.Store.Get(ctx, lockKey); err == nil && ok && current != token {
		return
	}
	if err := s.Store.Del(ctx, lockKey); err != nil {
		s.warn("idempotency lock release failed", err)
	}
}

func (s *Service) idempotencyCacheTTL() time.Duration {
	if s.IdempotencyTTL > 0 {
		return s.IdempotencyTTL
	}
	return defaultIdempotencyCacheTTL
}

func (s *Service) idempotencyLockTTL() time.Duration {
	if s.LockTTL > 0 {
		return s.LockTTL
	}
	return idempotencyStaleWindow
}

func (s *Service) warn(message string, err error) {
	if s.Logger != nil {
		s.Logger.Warn(message, "error", err)
	}
}

// log writes one step of a payment flow. It is a no-op when the service has no
// logger, so tests and embedded uses stay silent.
func (s *Service) log(message string, attrs ...any) {
	if s.Logger == nil {
		return
	}
	s.Logger.Info(message, attrs...)
}

// providerNames renders a routing chain for a log line.
func providerNames(chain []constants.Provider) []string {
	names := make([]string, 0, len(chain))
	for _, provider := range chain {
		names = append(names, provider.String())
	}
	return names
}

// customerIDValue renders an optional customer id for a log line.
func customerIDValue(id *string) string {
	if id == nil {
		return ""
	}
	return *id
}

// storedError reconstructs the typed error of a previously failed idempotent
// request, so a replay returns exactly what the first attempt returned.
func storedError(record *IdempotencyRecord) error {
	raw, ok := record.ResponseBody.Data["error"]
	if !ok {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var stored struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(encoded, &stored); err != nil || stored.Code == "" {
		return nil
	}
	status := http.StatusInternalServerError
	if record.StatusCode != nil {
		status = *record.StatusCode
	}
	return apperrors.New(stored.Type, stored.Code, stored.Message, status)
}

func replayResponse(record *IdempotencyRecord) *Response {
	if record.ResponseBody.Data == nil {
		return nil
	}
	encoded, err := json.Marshal(record.ResponseBody.Data)
	if err != nil {
		return nil
	}
	var response Response
	if err := json.Unmarshal(encoded, &response); err != nil {
		return nil
	}
	return &response
}

func hashRequest(amount int64, currency, reference string, req CreateRequest) string {
	canonical := map[string]any{
		"amount":    amount,
		"currency":  currency,
		"reference": reference,
		"adapter":   req.Adapter,
		"fallback":  req.Fallback,
		"customer":  req.Customer,
		"metadata":  req.Metadata,
	}
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// isSafelyRetryable reports whether a provider error definitely did NOT create a
// charge, which is the only case in which a fallback provider may be tried.
func isSafelyRetryable(err error) bool {
	return apperrors.Is(err, apperrors.CodeProviderRequestFailed) ||
		apperrors.Is(err, apperrors.CodeProviderAuthenticationFails)
}

func publicMessage(err error) string {
	appErr := apperrors.From(err)
	if appErr == nil {
		return ""
	}
	return appErr.Message
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
