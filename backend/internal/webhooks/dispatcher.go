package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/events"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/platform/vault"
)

// Dispatcher delivers domain events to a merchant's webhook endpoints. It is an
// events.Subscriber: payments and refunds publish, the dispatcher fans out.
//
// Delivery failures never fail the event: they are recorded on the delivery row
// with an exponential backoff, retried by the worker, and dead-lettered after
// MaxAttempts.
type Dispatcher struct {
	Repo   *Repository
	Vault  *vault.Vault
	Client *http.Client
	Log    *slog.Logger
	// Verbose emits a line per fan-out and per delivery attempt
	// (LOG_WEBHOOK_VERBOSE). Failures are logged regardless.
	Verbose     bool
	Now         func() time.Time
	MaxAttempts int
}

// debug logs an outbound delivery step when verbose logging is enabled.
func (d *Dispatcher) debug(message string, attrs ...any) {
	if !d.Verbose || d.Log == nil {
		return
	}
	d.Log.Debug(message, attrs...)
}

// NewDispatcher builds the dispatcher with bounded delivery timeouts.
func NewDispatcher(repo *Repository, v *vault.Vault, log *slog.Logger) *Dispatcher {
	return &Dispatcher{
		Repo:        repo,
		Vault:       v,
		Client:      &http.Client{Timeout: 10 * time.Second},
		Log:         log,
		Now:         time.Now,
		MaxAttempts: 6,
	}
}

// Types subscribes to every event type; each endpoint filters what it wants.
func (d *Dispatcher) Types() []string { return nil }

// Handle creates one delivery per subscribed endpoint and attempts it.
func (d *Dispatcher) Handle(ctx context.Context, e events.Event) error {
	endpoints, err := d.Repo.SubscribedEndpoints(e.MerchantID, e.Environment, e.Type)
	if err != nil {
		return err
	}
	if len(endpoints) == 0 {
		d.debug("webhook event has no subscribed endpoint",
			"event", e.Type, "merchant_id", e.MerchantID, "environment", e.Environment)
		return nil
	}
	d.debug("webhook event fanning out",
		"event", e.Type, "merchant_id", e.MerchantID, "environment", e.Environment, "endpoints", len(endpoints))
	for i := range endpoints {
		endpoint := endpoints[i]
		if !endpoint.Subscribes(e.Type) {
			d.debug("webhook endpoint skipped: not subscribed", "endpoint_id", endpoint.ID, "event", e.Type)
			continue
		}
		now := d.Now().UTC()
		delivery := &Delivery{
			ID:         gormx.NewID("whd"),
			MerchantID: e.MerchantID,
			EndpointID: endpoint.ID,
			Event:      e.Type,
			Status:     constants.DeliveryPending,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		// The body carries its own delivery id, so the payload a merchant stores
		// can be matched to the row they see in the dashboard.
		delivery.Payload = gormx.JSONB[map[string]any]{Data: envelope(e, delivery.ID)}
		if err := d.Repo.CreateDelivery(delivery); err != nil {
			return err
		}
		d.attempt(ctx, delivery, &endpoint)
	}
	return nil
}

// Retry attempts a delivery immediately, regardless of its backoff window.
func (d *Dispatcher) Retry(ctx context.Context, delivery *Delivery) (*Delivery, error) {
	endpoint, err := d.endpointByID(delivery.EndpointID)
	if err != nil {
		return nil, err
	}
	d.attempt(ctx, delivery, endpoint)
	return delivery, nil
}

// RunDue retries deliveries whose backoff has elapsed. The server calls it on a
// ticker; it is safe to run in more than one instance because each attempt is
// recorded on the delivery row before it is sent.
func (d *Dispatcher) RunDue(ctx context.Context) {
	if d.Repo == nil {
		return
	}
	due, err := d.Repo.DueDeliveries(d.Now().UTC(), 50)
	if err != nil {
		if d.Log != nil {
			d.Log.Warn("webhook retry scan failed", "error", err)
		}
		return
	}
	for i := range due {
		endpoint, err := d.endpointByID(due[i].EndpointID)
		if err != nil {
			continue
		}
		d.attempt(ctx, &due[i], endpoint)
	}
}

// Start runs the retry worker until the context is cancelled.
func (d *Dispatcher) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.RunDue(ctx)
			}
		}
	}()
}

// attempt performs one signed POST and records the outcome.
func (d *Dispatcher) attempt(ctx context.Context, delivery *Delivery, endpoint *Endpoint) {
	now := d.Now().UTC()
	body, err := json.Marshal(delivery.Payload.Data)
	if err != nil {
		d.fail(delivery, endpoint, err.Error(), nil)
		return
	}
	secret, err := d.signingSecret(endpoint)
	if err != nil {
		d.fail(delivery, endpoint, err.Error(), nil)
		return
	}

	d.debug("webhook delivery attempt",
		"delivery_id", delivery.ID, "event", delivery.Event, "endpoint_id", endpoint.ID,
		"url", endpoint.URL, "attempt", delivery.Attempts+1)

	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		d.fail(delivery, endpoint, err.Error(), nil)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PayConnect-Webhooks/1.0")
	req.Header.Set(constants.HeaderSignature, vault.SignWebhook(secret, body))
	req.Header.Set(constants.HeaderEvent, delivery.Event)
	req.Header.Set(constants.HeaderDeliveryID, delivery.ID)

	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	delivery.Attempts++
	delivery.UpdatedAt = now
	if err != nil {
		d.fail(delivery, endpoint, err.Error(), nil)
		return
	}
	defer resp.Body.Close()
	// Keep what the endpoint said. A rejection reason is almost always in the
	// body, and the status code alone leaves the merchant guessing.
	delivery.ResponseBody = responseSnippet(resp.Body)
	code := resp.StatusCode
	delivery.ResponseCode = &code

	if code >= 200 && code < 300 {
		delivery.Status = constants.DeliveryDelivered
		delivery.LastError = ""
		delivery.NextAttemptAt = nil
		delivery.DeliveredAt = &now
		_ = d.Repo.SaveDelivery(delivery)
		_ = d.Repo.ResetEndpointFailures(endpoint.ID)
		d.debug("webhook delivered",
			"delivery_id", delivery.ID, "event", delivery.Event, "endpoint_id", endpoint.ID,
			"attempts", delivery.Attempts, "response_code", code)
		return
	}
	d.fail(delivery, endpoint, "endpoint responded "+resp.Status, &code)
}

// fail records a failed attempt and schedules the next one, dead-lettering after
// MaxAttempts.
func (d *Dispatcher) fail(delivery *Delivery, endpoint *Endpoint, message string, code *int) {
	now := d.Now().UTC()
	delivery.LastError = message
	delivery.UpdatedAt = now
	if code != nil {
		delivery.ResponseCode = code
	}
	maxAttempts := d.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 6
	}
	if delivery.Attempts >= maxAttempts {
		delivery.Status = constants.DeliveryDead
		delivery.NextAttemptAt = nil
	} else {
		delivery.Status = constants.DeliveryRetrying
		next := now.Add(backoff(delivery.Attempts))
		delivery.NextAttemptAt = &next
	}
	_ = d.Repo.SaveDelivery(delivery)
	if endpoint != nil {
		_ = d.Repo.IncrementEndpointFailure(endpoint.ID)
	}
	if d.Log != nil {
		d.Log.Warn("webhook delivery failed",
			"delivery_id", delivery.ID, "event", delivery.Event,
			"attempts", delivery.Attempts, "status", delivery.Status, "error", message)
	}
}

// signingSecret decrypts the endpoint's merchant webhook secret.
func (d *Dispatcher) signingSecret(endpoint *Endpoint) (string, error) {
	if len(endpoint.SigningSecretEncrypted) == 0 {
		return "", errMissingSigningSecret
	}
	return d.Vault.Decrypt(endpoint.SigningSecretEncrypted)
}

// endpointByID loads the endpoint a delivery targets. It is deliberately not
// merchant-scoped: delivery rows are already merchant-scoped when they are read
// through the API, and the worker acts on rows it loaded itself.
func (d *Dispatcher) endpointByID(id string) (*Endpoint, error) {
	var endpoint Endpoint
	if err := d.Repo.DB.First(&endpoint, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &endpoint, nil
}

// envelope is the merchant-facing webhook body. It is deliberately one shape for
// every provider and every event: the same keys, in the same place, whatever
// processed the payment. `data` is the canonical representation the payments API
// also returns, and `meta` holds the delivery-level facts — the contract
// version, the environment, and which delivery this is — so a consumer never has
// to reconstruct them from headers.
func envelope(e events.Event, deliveryID string) map[string]any {
	body := map[string]any{
		"id":         e.ID,
		"type":       e.Type,
		"created_at": e.CreatedAt.UTC().Format(time.RFC3339),
		"meta": map[string]any{
			"webhook_version": constants.WebhookPayloadVersion,
			"environment":     e.Environment,
			"delivery_id":     deliveryID,
		},
	}
	if len(e.Data) > 0 {
		var data any
		if err := json.Unmarshal(e.Data, &data); err == nil {
			body["data"] = data
		}
	}
	if _, ok := body["data"]; !ok {
		body["data"] = map[string]any{
			"payment_id": e.PaymentID,
			"refund_id":  e.RefundID,
			"reference":  e.Reference,
		}
	}
	return body
}

// ResponseBodyLimit bounds how much of an endpoint's reply is retained. The body
// belongs to the merchant, but it is also unbounded input from the internet, so
// only the head is kept — which is where a human-readable error lives.
const ResponseBodyLimit = 2048

// responseSnippet reads the head of an endpoint's reply. It returns valid UTF-8
// so a body truncated mid-character cannot make the stored row unserializable,
// and it never fails the delivery: the attempt's outcome must not depend on
// whether its body could be read.
func responseSnippet(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, ResponseBodyLimit+1))
	if err != nil {
		return ""
	}
	if len(raw) > ResponseBodyLimit {
		return strings.ToValidUTF8(string(raw[:ResponseBodyLimit]), "") + "\n… (truncated)"
	}
	return strings.ToValidUTF8(string(raw), "")
}

// backoff returns an exponential delay: 30s, 1m, 2m, 4m, 8m, capped at 1h.
func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	delay := 30 * time.Second
	for i := 1; i < attempts; i++ {
		delay *= 2
		if delay >= time.Hour {
			return time.Hour
		}
	}
	return delay
}

var _ events.Subscriber = (*Dispatcher)(nil)
