package observability

import "sync/atomic"

// Metrics is a tiny counter registry. It intentionally avoids a heavyweight
// dependency: swap the implementation for Prometheus/OTel when a collector
// exists, without changing call sites.
type Metrics struct {
	requests   atomic.Int64
	errors     atomic.Int64
	payments   atomic.Int64
	webhooks   atomic.Int64
	providerMS atomic.Int64
}

// NewMetrics builds an empty registry.
func NewMetrics() *Metrics { return &Metrics{} }

// ObserveRequest records one handled request.
func (m *Metrics) ObserveRequest(status int) {
	m.requests.Add(1)
	if status >= 500 {
		m.errors.Add(1)
	}
}

// ObservePayment counts a payment attempt.
func (m *Metrics) ObservePayment() { m.payments.Add(1) }

// ObserveWebhook counts an inbound provider webhook.
func (m *Metrics) ObserveWebhook() { m.webhooks.Add(1) }

// ObserveProviderCall records provider latency in milliseconds.
func (m *Metrics) ObserveProviderCall(ms int64) { m.providerMS.Add(ms) }

// Snapshot returns the current counters.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"requests_total":            m.requests.Load(),
		"errors_total":              m.errors.Load(),
		"payments_total":            m.payments.Load(),
		"provider_webhooks_total":   m.webhooks.Load(),
		"provider_latency_total_ms": m.providerMS.Load(),
	}
}
