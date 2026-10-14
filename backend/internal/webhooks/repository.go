package webhooks

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/kuipid01/payconnect/internal/constants"
)

// Repository owns webhook persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the webhook repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

/* ------------------------------ provider events ------------------------------ */

// RecordProviderEvent inserts an inbound event, returning created=false when the
// same (provider, provider_event_id) was already processed. That single check is
// what makes webhook handling idempotent.
func (r *Repository) RecordProviderEvent(e *ProviderEvent) (bool, error) {
	result := r.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(e)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

// ProviderEventFilter narrows an inbound-event listing.
type ProviderEventFilter struct {
	MerchantID   string
	ConnectionID string
	Provider     string
	EventType    string
	Offset       int
	Limit        int
}

// ProviderEvents lists a merchant's inbound provider events, newest first.
func (r *Repository) ProviderEvents(f ProviderEventFilter) ([]ProviderEvent, error) {
	q := r.DB.Where("merchant_id = ?", f.MerchantID)
	if f.ConnectionID != "" {
		q = q.Where("connection_id = ?", f.ConnectionID)
	}
	if f.Provider != "" {
		q = q.Where("provider = ?", f.Provider)
	}
	if f.EventType != "" {
		q = q.Where("event_type = ?", f.EventType)
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []ProviderEvent
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountProviderEvents returns the number of inbound events matching a filter.
func (r *Repository) CountProviderEvents(f ProviderEventFilter) (int64, error) {
	q := r.DB.Model(&ProviderEvent{}).Where("merchant_id = ?", f.MerchantID)
	if f.ConnectionID != "" {
		q = q.Where("connection_id = ?", f.ConnectionID)
	}
	if f.Provider != "" {
		q = q.Where("provider = ?", f.Provider)
	}
	if f.EventType != "" {
		q = q.Where("event_type = ?", f.EventType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// MarkProviderEventProcessed stamps the reconcile time.
func (r *Repository) MarkProviderEventProcessed(id string, at time.Time) error {
	return r.DB.Model(&ProviderEvent{}).Where("id = ?", id).Update("processed_at", at).Error
}

/* --------------------------------- endpoints --------------------------------- */

// EndpointFilter narrows an endpoint listing.
type EndpointFilter struct {
	MerchantID  string
	Environment string
	Status      string
}

// Endpoints lists a merchant's endpoints.
func (r *Repository) Endpoints(f EndpointFilter) ([]Endpoint, error) {
	q := r.DB.Where("merchant_id = ?", f.MerchantID)
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var rows []Endpoint
	if err := q.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// EndpointByID loads an endpoint scoped to its merchant.
func (r *Repository) EndpointByID(merchantID, id string) (*Endpoint, error) {
	var e Endpoint
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&e, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// SubscribedEndpoints returns the active endpoints that want an event type.
func (r *Repository) SubscribedEndpoints(merchantID, environment, eventType string) ([]Endpoint, error) {
	q := r.DB.Where("merchant_id = ? AND environment = ? AND status = 'active'",
		merchantID, environment)
	if eventType != "" {
		q = q.Where("events @> ?::jsonb", `["`+eventType+`"]`)
	}
	var rows []Endpoint
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateEndpoint inserts an endpoint.
func (r *Repository) CreateEndpoint(e *Endpoint) error { return r.DB.Create(e).Error }

// SaveEndpoint persists endpoint changes.
func (r *Repository) SaveEndpoint(e *Endpoint) error { return r.DB.Save(e).Error }

// DeleteEndpoint removes an endpoint.
func (r *Repository) DeleteEndpoint(merchantID, id string) (int64, error) {
	res := r.DB.Where("merchant_id = ? AND id = ?", merchantID, id).Delete(&Endpoint{})
	return res.RowsAffected, res.Error
}

// IncrementEndpointFailure bumps an endpoint's consecutive failure counter.
func (r *Repository) IncrementEndpointFailure(id string) error {
	return r.DB.Model(&Endpoint{}).Where("id = ?", id).
		UpdateColumn("failure_count", gorm.Expr("failure_count + 1")).Error
}

// ResetEndpointFailures clears the failure counter after a successful delivery.
func (r *Repository) ResetEndpointFailures(id string) error {
	return r.DB.Model(&Endpoint{}).Where("id = ?", id).UpdateColumn("failure_count", 0).Error
}

/* -------------------------------- deliveries --------------------------------- */

// DeliveryFilter narrows a delivery listing.
type DeliveryFilter struct {
	MerchantID string
	EndpointID string
	Event      string
	Status     string
	Offset     int
	Limit      int
}

// Deliveries lists a merchant's deliveries.
func (r *Repository) Deliveries(f DeliveryFilter) ([]Delivery, error) {
	q := r.DB.Where("merchant_id = ?", f.MerchantID)
	if f.EndpointID != "" {
		q = q.Where("endpoint_id = ?", f.EndpointID)
	}
	if f.Event != "" {
		q = q.Where("event = ?", f.Event)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []Delivery
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountDeliveries returns the number of deliveries matching a filter.
func (r *Repository) CountDeliveries(f DeliveryFilter) (int64, error) {
	q := r.DB.Model(&Delivery{}).Where("merchant_id = ?", f.MerchantID)
	if f.EndpointID != "" {
		q = q.Where("endpoint_id = ?", f.EndpointID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// DeliveryByID loads a delivery scoped to its merchant.
func (r *Repository) DeliveryByID(merchantID, id string) (*Delivery, error) {
	var d Delivery
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&d, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// EndpointDeliveryCount is the concluded-delivery outcome for one endpoint over
// a reporting window. Total counts only deliveries that reached a terminal
// state, so an attempt still in flight is never charged as a failure.
type EndpointDeliveryCount struct {
	EndpointID string
	Delivered  int64
	Total      int64
}

// DeliveryCountsByEndpoint tallies each endpoint's concluded deliveries created
// since a cut-off. One grouped query covers every endpoint a listing returns, so
// the dashboard's success metric costs a single round trip rather than one per
// endpoint.
func (r *Repository) DeliveryCountsByEndpoint(merchantID string, since time.Time) (map[string]EndpointDeliveryCount, error) {
	const query = `
		SELECT endpoint_id,
		       COUNT(*) FILTER (WHERE status = ?) AS delivered,
		       COUNT(*) AS total
		FROM webhook_deliveries
		WHERE merchant_id = ?
		  AND created_at >= ?
		  AND status IN ?
		GROUP BY endpoint_id`
	concluded := []string{constants.DeliveryDelivered, constants.DeliveryFailed, constants.DeliveryDead}
	var rows []EndpointDeliveryCount
	if err := r.DB.Raw(query, constants.DeliveryDelivered, merchantID, since, concluded).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]EndpointDeliveryCount, len(rows))
	for _, row := range rows {
		out[row.EndpointID] = row
	}
	return out, nil
}

// CreateDelivery inserts a delivery row.
func (r *Repository) CreateDelivery(d *Delivery) error { return r.DB.Create(d).Error }

// SaveDelivery persists delivery changes.
func (r *Repository) SaveDelivery(d *Delivery) error { return r.DB.Save(d).Error }

// DueDeliveries returns deliveries whose backoff has elapsed, oldest first.
func (r *Repository) DueDeliveries(at time.Time, limit int) ([]Delivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []Delivery
	if err := r.DB.Where("status IN ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)",
		[]string{"pending", "retrying", "failed"}, at).
		Order("created_at").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
