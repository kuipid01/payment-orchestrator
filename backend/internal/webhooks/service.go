package webhooks

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/billing"
	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/events"
	"github.com/kuipid01/payconnect/internal/payments"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/platform/vault"
	"github.com/kuipid01/payconnect/internal/providers"
)

/* --------------------------------- ports ------------------------------------ */

// Connections loads a provider connection and its decrypted credentials. The
// connection id comes from the webhook URL, so the lookup is not merchant-scoped
// — the signature check is what authenticates the caller.
type Connections interface {
	ConnectionWithCredentials(id string) (*providers.Connection, providers.Credentials, error)
	// TouchWebhookReceived stamps the connection's last verified inbound webhook
	// time, so the dashboard can show connection health.
	TouchWebhookReceived(id string, at time.Time) error
}

// Registry resolves a provider to its adapter.
type Registry interface {
	Get(provider constants.Provider) (providers.Adapter, error)
}

// PaymentReconciler applies a normalized provider event to the payment domain.
type PaymentReconciler interface {
	ReconcileWebhook(ctx context.Context, merchantID, provider string, result *providers.WebhookResult) (*payments.Payment, error)
}

// Service owns the webhook gateway and endpoint administration.
type Service struct {
	Repo        *Repository
	Connections Connections
	Registry    Registry
	Payments    PaymentReconciler
	Events      events.Publisher
	Vault       *vault.Vault
	Audit       audit.Recorder
	Dispatcher  *Dispatcher
	// Logger receives the webhook trace. Optional.
	Logger *slog.Logger
	// Verbose emits a line per inbound pipeline step and per outbound delivery
	// attempt (LOG_WEBHOOK_VERBOSE). Warnings and errors are always logged.
	Verbose bool
	// Gates answers the one billing question endpoint administration asks.
	// Optional, and a nil gate means no limit. Delivery is never gated.
	Gates PlanGates
	Now   func() time.Time
}

// NewService builds the webhook service.
func NewService(repo *Repository, connections Connections, registry Registry, reconciler PaymentReconciler,
	v *vault.Vault, bus events.Publisher, recorder audit.Recorder) *Service {
	return &Service{
		Repo: repo, Connections: connections, Registry: registry, Payments: reconciler,
		Vault: v, Events: bus, Audit: recorder, Now: time.Now,
	}
}

/* --------------------------------- logging ----------------------------------- */

// traceLog logs one step of the webhook pipeline. It is gated on verbose because
// a healthy integration produces several lines per event; the flag
// (LOG_WEBHOOK_VERBOSE) turns the full trace on when something needs diagnosing.
func traceLog(logger *slog.Logger, verbose bool, message string, attrs ...any) {
	if !verbose || logger == nil {
		return
	}
	logger.Debug(message, attrs...)
}

// warn logs a webhook problem. Unlike trace, it is never gated: a rejected
// signature or a failed reconcile is the signal an operator must always see.
func (s *Service) warn(message string, attrs ...any) {
	if s.Logger == nil {
		return
	}
	s.Logger.Warn(message, attrs...)
}

// tracef logs one pipeline step when verbose logging is enabled.
func (s *Service) tracef(message string, attrs ...any) {
	traceLog(s.Logger, s.Verbose, message, attrs...)
}

// hasSignatureHeader reports whether any header looks like a provider
// signature, without ever logging its value.
func hasSignatureHeader(headers map[string]string) bool {
	for key := range headers {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "signature") || lower == "verif-hash" {
			return true
		}
	}
	return false
}

/* ------------------------------ inbound provider ----------------------------- */

// Inbound is the outcome of one provider webhook.
type Inbound struct {
	EventID      string `json:"event_id"`
	Type         string `json:"type"`
	Duplicate    bool   `json:"duplicate"`
	PaymentID    string `json:"payment_id,omitempty"`
	PaymentState string `json:"payment_status,omitempty"`
}

// HandleInbound runs the provider webhook pipeline: identify connection → verify
// signature → normalize → dedupe → reconcile payment → publish internal event.
func (s *Service) HandleInbound(ctx context.Context, requestID string, provider constants.Provider, connectionID string,
	headers map[string]string, body []byte) (*Inbound, error) {
	if !constants.IsValidProvider(provider) {
		s.warn("webhook rejected: unsupported provider", "request_id", requestID, "provider", provider.String())
		return nil, apperrors.ProviderNotSupported(provider.String())
	}
	connection, credentials, err := s.Connections.ConnectionWithCredentials(connectionID)
	if err != nil {
		s.warn("webhook rejected: connection not found",
			"request_id", requestID, "provider", provider.String(), "connection_id", connectionID, "reason", err.Error())
		return nil, err
	}
	if connection.Provider != provider.String() {
		s.warn("webhook rejected: provider does not match the endpoint",
			"request_id", requestID, "path_provider", provider.String(), "connection_provider", connection.Provider,
			"connection_id", connectionID)
		return nil, apperrors.WebhookEventInvalid("The webhook was sent to the wrong provider endpoint.")
	}
	adapter, err := s.Registry.Get(provider)
	if err != nil {
		s.warn("webhook rejected: no adapter", "request_id", requestID, "provider", provider.String(), "reason", err.Error())
		return nil, err
	}
	result, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:       headers,
		Body:          body,
		SecretKey:     credentials.SecretKey,
		WebhookSecret: credentials.WebhookSecret,
	})
	if err != nil {
		s.warn("webhook rejected: signature verification failed",
			"request_id", requestID, "provider", provider.String(), "connection_id", connection.ID,
			"signature_present", hasSignatureHeader(headers),
			"used_webhook_secret", credentials.WebhookSecret != "", "reason", err.Error())
		return nil, apperrors.From(err)
	}

	// A verified webhook means the connection is alive: stamp it regardless of
	// whether the event is a duplicate, so health reflects real provider traffic.
	now := s.Now().UTC()
	_ = s.Connections.TouchWebhookReceived(connection.ID, now)

	eventID := result.EventID
	if eventID == "" {
		eventID = provider.String() + ":" + result.Type + ":" + firstNonEmpty(result.ProviderReference, result.Reference)
	}
	event := &ProviderEvent{
		ID:              gormx.NewID("pev"),
		MerchantID:      connection.MerchantID,
		ConnectionID:    connection.ID,
		Provider:        provider.String(),
		ProviderEventID: eventID,
		EventType:       result.Type,
		Payload:         gormx.JSONB[map[string]any]{Data: result.Raw},
		CreatedAt:       now,
	}
	created, err := s.Repo.RecordProviderEvent(event)
	if err != nil {
		s.warn("webhook event could not be recorded", "request_id", requestID,
			"connection_id", connection.ID, "event_type", result.Type, "reason", err.Error())
		return nil, err
	}
	if !created {
		// The provider is retrying an event we already handled. Answering 200
		// stops the retry loop without touching the payment twice.
		s.logInbound(requestID, connection, result, "", "", "duplicate")
		return &Inbound{EventID: eventID, Type: result.Type, Duplicate: true}, nil
	}

	out := &Inbound{EventID: eventID, Type: result.Type}
	outcome := "reconciled"
	payment, err := s.Payments.ReconcileWebhook(ctx, connection.MerchantID, provider.String(), result)
	switch {
	case err == nil && payment != nil:
		out.PaymentID = payment.ID
		out.PaymentState = payment.Status
		s.publish(ctx, requestID, connection, result, payment)
	case apperrors.Is(err, apperrors.CodePaymentNotFound):
		// A provider may notify about a payment created outside PayConnect.
		// Accept and ignore it rather than asking the provider to retry forever.
		outcome = "no_matching_payment"
	default:
		s.warn("webhook reconcile failed",
			"request_id", requestID, "provider", provider.String(), "event_type", result.Type, "reason", err.Error())
		return out, err
	}
	_ = s.Repo.MarkProviderEventProcessed(event.ID, s.Now().UTC())
	s.logInbound(requestID, connection, result, out.PaymentID, out.PaymentState, outcome)
	return out, nil
}

// logInbound writes the one-line summary for a handled inbound webhook. It is
// the only per-event line: the individual pipeline steps are deliberately not
// logged, so a busy integration stays readable. Failures are logged separately
// at warning level.
func (s *Service) logInbound(requestID string, connection *providers.Connection, result *providers.WebhookResult,
	paymentID, paymentStatus, outcome string) {
	s.tracef("webhook inbound processed",
		"request_id", requestID, "provider", connection.Provider, "connection_id", connection.ID,
		"event_type", result.Type, "reference", firstNonEmpty(result.Reference, result.ProviderReference),
		"amount_minor", result.AmountMinor, "currency", result.Currency,
		"payment_id", paymentID, "payment_status", paymentStatus, "outcome", outcome)
}

func (s *Service) publish(ctx context.Context, requestID string, connection *providers.Connection, result *providers.WebhookResult, payment *payments.Payment) {
	if s.Events == nil || payment == nil {
		return
	}
	eventType := result.Type
	if !constants.IsMerchantEvent(eventType) {
		switch constants.PaymentStatus(payment.Status) {
		case constants.PaymentSuccessful:
			eventType = constants.EventPaymentSucceeded
		case constants.PaymentFailed:
			eventType = constants.EventPaymentFailed
		default:
			s.tracef("webhook outbound event skipped: no merchant event for this state",
				"request_id", requestID, "provider_event_type", result.Type, "payment_status", payment.Status)
			return
		}
	}
	event := events.New(eventType, connection.MerchantID, payment.Environment)
	event.PaymentID = payment.ID
	event.Reference = payment.Reference
	// Carry the canonical payment representation so the merchant webhook body
	// is the same shape the payment API returns.
	if payload, err := json.Marshal(payments.ToResponse(payment)); err == nil {
		event.Data = payload
	}
	s.Events.Publish(ctx, event)
}

/* -------------------------------- endpoints ---------------------------------- */

// EndpointInput is the create/update payload for a delivery target.
type EndpointInput struct {
	URL           string
	Environment   string
	Status        string
	Events        []string
	ApplicationID *string
}

// PlanGates is the one billing question this package asks: does the
// organization's plan have room for another delivery endpoint? Optional, and a
// nil gate means no limit — the same shape the application package uses, so the
// whole enforcement story reads the same way everywhere.
type PlanGates interface {
	CheckWebhookEndpoints(ctx context.Context, merchantID string) billing.Decision
}

// checkEndpointPlan asks the plan gate, when one is wired.
func (s *Service) checkEndpointPlan(ctx context.Context, merchantID string) error {
	if s.Gates == nil {
		return nil
	}
	return s.Gates.CheckWebhookEndpoints(ctx, merchantID).Refusal()
}

// CreateEndpoint registers a delivery target and returns its signing secret
// exactly once.
func (s *Service) CreateEndpoint(ctx context.Context, merchantID, actorName, actorRole string, in EndpointInput) (*Endpoint, string, error) {
	url := strings.TrimSpace(in.URL)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, "", apperrors.Validation(apperrors.CodeInvalidWebhookTarget, "url must be an absolute http(s) URL.")
	}
	if !constants.IsValidPaymentEnvironment(in.Environment) {
		return nil, "", apperrors.InvalidEnvironment()
	}
	if err := validateEvents(in.Events); err != nil {
		return nil, "", err
	}
	status := in.Status
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "paused" {
		return nil, "", apperrors.Validation(apperrors.CodeValidationError, "status must be active or paused.")
	}
	// The plan is consulted last, so a malformed endpoint is reported as the
	// mistake it is rather than as a plan limit. Nothing on the delivery path
	// itself is ever gated: this checks a new endpoint, never a delivery.
	if err := s.checkEndpointPlan(ctx, merchantID); err != nil {
		return nil, "", err
	}
	secret, masked, err := vault.GenerateWebhookSecret()
	if err != nil {
		return nil, "", err
	}
	encrypted, err := s.Vault.Encrypt(secret)
	if err != nil {
		return nil, "", err
	}
	now := s.Now().UTC()
	endpoint := &Endpoint{
		ID:                     gormx.NewID("whe"),
		MerchantID:             merchantID,
		ApplicationID:          in.ApplicationID,
		URL:                    url,
		Environment:            in.Environment,
		Status:                 status,
		Events:                 gormx.JSONB[[]string]{Data: in.Events},
		SigningSecretEncrypted: encrypted,
		SigningSecretMasked:    masked,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := s.Repo.CreateEndpoint(endpoint); err != nil {
		return nil, "", err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Webhook endpoint created", Detail: url + " (" + in.Environment + ")", Severity: "critical",
	})
	return endpoint, secret, nil
}

// ListEndpoints returns a merchant's delivery targets.
func (s *Service) ListEndpoints(f EndpointFilter) ([]Endpoint, error) { return s.Repo.Endpoints(f) }

// EndpointDeliveryStats returns each endpoint's concluded-delivery outcome over
// the dashboard's reporting window. The number the dashboard shows is read back
// from recorded attempts, never estimated from a single page of history.
func (s *Service) EndpointDeliveryStats(merchantID string) (map[string]EndpointDeliveryCount, error) {
	since := s.Now().UTC().AddDate(0, 0, -DeliveryWindowDays)
	return s.Repo.DeliveryCountsByEndpoint(merchantID, since)
}

// GetEndpoint returns one delivery target.
func (s *Service) GetEndpoint(merchantID, id string) (*Endpoint, error) {
	endpoint, err := s.Repo.EndpointByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound(apperrors.CodeWebhookEndpointNotFound, "Webhook endpoint not found.")
	}
	return endpoint, nil
}

// UpdateEndpoint applies a partial update.
func (s *Service) UpdateEndpoint(merchantID, actorName, actorRole, id string, in EndpointInput) (*Endpoint, error) {
	endpoint, err := s.GetEndpoint(merchantID, id)
	if err != nil {
		return nil, err
	}
	if in.URL != "" {
		url := strings.TrimSpace(in.URL)
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return nil, apperrors.Validation(apperrors.CodeInvalidWebhookTarget, "url must be an absolute http(s) URL.")
		}
		endpoint.URL = url
	}
	if in.Environment != "" {
		if !constants.IsValidPaymentEnvironment(in.Environment) {
			return nil, apperrors.InvalidEnvironment()
		}
		endpoint.Environment = in.Environment
	}
	if in.Events != nil {
		if err := validateEvents(in.Events); err != nil {
			return nil, err
		}
		endpoint.Events = gormx.JSONB[[]string]{Data: in.Events}
	}
	if in.Status != "" {
		if in.Status != "active" && in.Status != "paused" {
			return nil, apperrors.Validation(apperrors.CodeValidationError, "status must be active or paused.")
		}
		endpoint.Status = in.Status
	}
	endpoint.UpdatedAt = s.Now().UTC()
	if err := s.Repo.SaveEndpoint(endpoint); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Webhook endpoint updated", Detail: endpoint.URL, Severity: "warning",
	})
	return endpoint, nil
}

// DeleteEndpoint removes a delivery target.
func (s *Service) DeleteEndpoint(merchantID, actorName, actorRole, id string) error {
	affected, err := s.Repo.DeleteEndpoint(merchantID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperrors.NotFound(apperrors.CodeWebhookEndpointNotFound, "Webhook endpoint not found.")
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Webhook endpoint deleted", Detail: id, Severity: "critical",
	})
	return nil
}

/* -------------------------------- deliveries --------------------------------- */

/* ------------------------------ inbound events ------------------------------- */

// ListInboundEvents returns a page of received provider events plus the total.
func (s *Service) ListInboundEvents(f ProviderEventFilter) ([]ProviderEvent, int64, error) {
	rows, err := s.Repo.ProviderEvents(f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.Repo.CountProviderEvents(f)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListDeliveries returns a page of delivery attempts.
func (s *Service) ListDeliveries(f DeliveryFilter) ([]Delivery, int64, error) {
	rows, err := s.Repo.Deliveries(f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.Repo.CountDeliveries(f)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetDelivery returns one delivery attempt.
func (s *Service) GetDelivery(merchantID, id string) (*Delivery, error) {
	delivery, err := s.Repo.DeliveryByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound(apperrors.CodeWebhookDeliveryNotFound, "Webhook delivery not found.")
	}
	return delivery, nil
}

// TestDelivery is one sample delivery plus the endpoint it targeted.
type TestDelivery struct {
	Delivery *Delivery
	Endpoint *Endpoint
}

// TestDeliver sends a sample event to a merchant's active endpoint so an
// integrator can confirm their receiver is reachable, signed correctly and
// answering 2xx — without waiting for a real payment. The endpoint belongs to
// the merchant, so the caller (the developer platform) must have already
// authorized the request against a developer grant.
//
// An endpoint scoped to the application wins; otherwise a merchant-wide
// endpoint is used, because that is the one that receives the application's
// events in practice.
func (s *Service) TestDeliver(ctx context.Context, merchantID, applicationID string) (*TestDelivery, error) {
	endpoints, err := s.Repo.Endpoints(EndpointFilter{MerchantID: merchantID, Status: "active"})
	if err != nil {
		return nil, err
	}
	var target *Endpoint
	for i := range endpoints {
		if endpoints[i].ApplicationID != nil && *endpoints[i].ApplicationID == applicationID {
			target = &endpoints[i]
			break
		}
	}
	if target == nil {
		for i := range endpoints {
			if endpoints[i].ApplicationID == nil {
				target = &endpoints[i]
				break
			}
		}
	}
	if target == nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound,
			"No active webhook endpoint is configured for this application yet.")
	}
	return s.sendTest(ctx, target, applicationID)
}

// TestEndpoint sends a sample event to one endpoint the merchant names. It backs
// the dashboard's "Send test" action, where the merchant is looking at a single
// destination and wants to know whether it answers. The endpoint is read
// merchant-scoped, so a session can only ever test a receiver it owns.
func (s *Service) TestEndpoint(ctx context.Context, merchantID, endpointID string) (*TestDelivery, error) {
	endpoint, err := s.GetEndpoint(merchantID, endpointID)
	if err != nil {
		return nil, err
	}
	if endpoint.Status != "active" {
		return nil, apperrors.Validation(apperrors.CodeValidationError,
			"This endpoint is paused, so it would not receive real events either. Resume it and try again.")
	}
	applicationID := ""
	if endpoint.ApplicationID != nil {
		applicationID = *endpoint.ApplicationID
	}
	return s.sendTest(ctx, endpoint, applicationID)
}

// webhookTestEvent is the synthetic event a sample delivery carries. It is not a
// subscribable type: it is never fanned out to other endpoints, only sent to the
// one receiver the caller asked about.
const webhookTestEvent = "webhook.test"

// sendTest records one sample delivery and attempts it immediately, so the
// caller learns the outcome in the response rather than from a later poll. The
// row is written before the attempt, so a receiver that times out still leaves a
// delivery the merchant can inspect.
func (s *Service) sendTest(ctx context.Context, target *Endpoint, applicationID string) (*TestDelivery, error) {
	if s.Dispatcher == nil {
		return nil, apperrors.ServiceUnavailable("Webhook delivery is not configured.", nil)
	}
	now := s.Now().UTC()
	data := map[string]any{
		"message": "This is a test delivery from PayConnect. If your endpoint answered 2xx, your integration is wired up.",
	}
	// Only named when the endpoint serves one application; a merchant-wide
	// receiver has no application to report.
	if applicationID != "" {
		data["application_id"] = applicationID
	}
	delivery := &Delivery{
		ID:         gormx.NewID("whd"),
		MerchantID: target.MerchantID,
		EndpointID: target.ID,
		Event:      webhookTestEvent,
		Status:     constants.DeliveryPending,
		Payload: gormx.JSONB[map[string]any]{Data: map[string]any{
			"id":         gormx.NewID("evt"),
			"type":       webhookTestEvent,
			"created_at": now.Format(time.RFC3339),
			"data":       data,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Repo.CreateDelivery(delivery); err != nil {
		return nil, err
	}
	delivered, err := s.Dispatcher.Retry(ctx, delivery)
	if err != nil {
		return nil, err
	}
	return &TestDelivery{Delivery: delivered, Endpoint: target}, nil
}

// RetryDelivery re-attempts a delivery immediately.
func (s *Service) RetryDelivery(ctx context.Context, merchantID, id string) (*Delivery, error) {
	delivery, err := s.GetDelivery(merchantID, id)
	if err != nil {
		return nil, err
	}
	if s.Dispatcher == nil {
		return nil, apperrors.ServiceUnavailable("Webhook delivery is not configured.", nil)
	}
	return s.Dispatcher.Retry(ctx, delivery)
}

func validateEvents(events []string) error {
	for _, name := range events {
		if !constants.IsMerchantEvent(name) {
			return apperrors.Validation(apperrors.CodeValidationError, "Unknown event type: "+name)
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
