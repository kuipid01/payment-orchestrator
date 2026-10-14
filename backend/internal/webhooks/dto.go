package webhooks

import (
	"math"
	"time"
)

// DeliveryWindowDays is the reporting window behind the dashboard's delivery
// success metric. A week spans a full traffic cycle without letting a long-fixed
// endpoint carry a permanent scar.
const DeliveryWindowDays = 7

// SuccessRate turns a delivered/total pair into a percentage rounded to one
// decimal. It returns nil when nothing concluded in the window: an em dash is
// honest where 100% with no deliveries behind it would not be.
func SuccessRate(delivered, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	rate := math.Round(float64(delivered)/float64(total)*1000) / 10
	return &rate
}

// EndpointResponse is the public endpoint representation. The signing secret is
// returned only once, at creation; afterwards only its masked form is visible.
type EndpointResponse struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Environment   string   `json:"environment"`
	Status        string   `json:"status"`
	Events        []string `json:"events"`
	ApplicationID string   `json:"application_id,omitempty"`
	SecretMasked  string   `json:"signing_secret_masked"`
	FailureCount  int      `json:"failure_count"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	// DeliverySuccessRate is the share of deliveries that reached this endpoint
	// over the last DeliveryWindowDays, as a percentage. Omitted when nothing
	// concluded in the window.
	DeliverySuccessRate *float64 `json:"delivery_success_rate,omitempty"`
	// Deliveries and Delivered are the raw counts behind that percentage, so the
	// dashboard can show what the number is made of.
	Deliveries         int64 `json:"deliveries"`
	Delivered          int64 `json:"delivered"`
	DeliveryWindowDays int   `json:"delivery_window_days"`
}

// ToEndpointResponse maps an endpoint onto the public DTO.
func ToEndpointResponse(e *Endpoint) EndpointResponse {
	out := EndpointResponse{
		ID:           e.ID,
		URL:          e.URL,
		Environment:  e.Environment,
		Status:       e.Status,
		Events:       e.Events.Data,
		SecretMasked: e.SigningSecretMasked,
		FailureCount: e.FailureCount,
		CreatedAt:    e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    e.UpdatedAt.UTC().Format(time.RFC3339),
		// The window is a constant fact about the metric; the counts and rate are
		// attached per request from recorded deliveries.
		DeliveryWindowDays: DeliveryWindowDays,
	}
	if out.Events == nil {
		out.Events = []string{}
	}
	if e.ApplicationID != nil {
		out.ApplicationID = *e.ApplicationID
	}
	return out
}

// WithDeliveryStats records an endpoint's window outcome on the response, so
// every rate the dashboard renders is accompanied by the deliveries it came
// from.
func (r *EndpointResponse) WithDeliveryStats(stats EndpointDeliveryCount) {
	r.Delivered = stats.Delivered
	r.Deliveries = stats.Total
	r.DeliverySuccessRate = SuccessRate(stats.Delivered, stats.Total)
}

// CreatedEndpointResponse carries the one-time signing secret.
type CreatedEndpointResponse struct {
	Endpoint      EndpointResponse `json:"endpoint"`
	SigningSecret string           `json:"signing_secret"`
}

// ProviderEventResponse is the public representation of a received provider event.
type ProviderEventResponse struct {
	ID              string         `json:"id"`
	ConnectionID    string         `json:"connection_id"`
	Provider        string         `json:"provider"`
	EventType       string         `json:"event_type"`
	ProviderEventID string         `json:"provider_event_id"`
	Payload         map[string]any `json:"payload,omitempty"`
	ProcessedAt     *string        `json:"processed_at,omitempty"`
	CreatedAt       string         `json:"created_at"`
}

// ToProviderEventResponse maps an inbound event onto the public DTO.
func ToProviderEventResponse(e *ProviderEvent) ProviderEventResponse {
	return ProviderEventResponse{
		ID:              e.ID,
		ConnectionID:    e.ConnectionID,
		Provider:        e.Provider,
		EventType:       e.EventType,
		ProviderEventID: e.ProviderEventID,
		Payload:         e.Payload.Data,
		ProcessedAt:     formatTime(e.ProcessedAt),
		CreatedAt:       e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// DeliveryResponse is the public delivery representation.
type DeliveryResponse struct {
	ID            string         `json:"id"`
	EndpointID    string         `json:"endpoint_id"`
	Event         string         `json:"event"`
	Status        string         `json:"status"`
	Attempts      int            `json:"attempts"`
	ResponseCode  *int           `json:"response_code,omitempty"`
	ResponseBody  string         `json:"response_body,omitempty"`
	LastError     string         `json:"last_error,omitempty"`
	NextAttemptAt *string        `json:"next_attempt_at,omitempty"`
	Payload       map[string]any `json:"payload"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	DeliveredAt   *string        `json:"delivered_at,omitempty"`
}

// ToDeliveryResponse maps a delivery onto the public DTO.
func ToDeliveryResponse(d *Delivery) DeliveryResponse {
	return DeliveryResponse{
		ID:            d.ID,
		EndpointID:    d.EndpointID,
		Event:         d.Event,
		Status:        d.Status,
		Attempts:      d.Attempts,
		ResponseCode:  d.ResponseCode,
		ResponseBody:  d.ResponseBody,
		LastError:     d.LastError,
		NextAttemptAt: formatTime(d.NextAttemptAt),
		Payload:       d.Payload.Data,
		CreatedAt:     d.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     d.UpdatedAt.UTC().Format(time.RFC3339),
		DeliveredAt:   formatTime(d.DeliveredAt),
	}
}

// TestDeliveryResponse is the outcome of one sample delivery: where it went,
// what the endpoint answered, and why it failed when it did. The developer
// platform's own "send test" action returns the same fields, so a client reads
// one shape wherever a sample was sent.
type TestDeliveryResponse struct {
	EndpointID string `json:"endpoint_id"`
	// DeliveryID is the attempt that was just made, so the dashboard can link
	// straight to the delivery page it created.
	DeliveryID   string `json:"delivery_id"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	ResponseCode *int   `json:"response_code,omitempty"`
	// ResponseBody is what the endpoint replied, so a rejection reason is
	// visible without opening the delivery.
	ResponseBody string `json:"response_body,omitempty"`
	LastError    string `json:"last_error,omitempty"`
}

// ToTestDeliveryResponse maps a sample delivery onto the public DTO.
func ToTestDeliveryResponse(t *TestDelivery) TestDeliveryResponse {
	out := TestDeliveryResponse{
		EndpointID:   t.Delivery.EndpointID,
		DeliveryID:   t.Delivery.ID,
		Status:       t.Delivery.Status,
		ResponseCode: t.Delivery.ResponseCode,
		ResponseBody: t.Delivery.ResponseBody,
		LastError:    t.Delivery.LastError,
	}
	if t.Endpoint != nil {
		out.URL = t.Endpoint.URL
	}
	return out
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.UTC().Format(time.RFC3339)
	return &formatted
}
