package payments

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repository owns payment persistence only.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the payment repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// Filter narrows a payment listing.
//
// Every field is optional except MerchantID; the zero value of the rest means
// "do not narrow". DeveloperAccessID additionally accepts the sentinel
// "general", which selects payments taken with a team-shared key.
type Filter struct {
	MerchantID        string
	Environment       string
	ApplicationID     string
	DeveloperAccessID string
	Status            string
	Provider          string
	Currency          string
	Reference         string
	CustomerID        string
	// Search matches the reference, the payment id, and the payer's name or
	// email. It exists so the dashboard's one search box can find a payment by
	// anything the operator remembers about it.
	Search string
	From   *time.Time
	To     *time.Time
	Offset int
	Limit  int
}

// GeneralKeyScope is the filter value that selects payments taken with a shared,
// non-developer key. It is not a real grant id — grant ids are prefixed — so it
// can never collide with one.
const GeneralKeyScope = "general"

// AnyScope is the filter value that means "do not narrow by this field". The
// dashboard sends it for every dropdown still on its default, so it must add no
// predicate at all: status=all has to mean every status, not the literal status
// "all", which no payment has.
const AnyScope = "all"

// scoped builds the predicate shared by List, Count and StatusCounts. Sharing it
// is what keeps a page of results and its total from disagreeing.
func (r *Repository) scoped(f Filter) *gorm.DB {
	q := r.DB.Model(&Payment{}).Where("merchant_id = ?", f.MerchantID)
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if f.ApplicationID != "" {
		q = q.Where("application_id = ?", f.ApplicationID)
	}
	switch f.DeveloperAccessID {
	case "", AnyScope:
		// Every key, general and developer-scoped alike.
	case GeneralKeyScope:
		q = q.Where("developer_access_id IS NULL")
	default:
		q = q.Where("developer_access_id = ?", f.DeveloperAccessID)
	}
	if f.Status != "" && f.Status != AnyScope {
		q = q.Where("status = ?", f.Status)
	}
	if f.Provider != "" && f.Provider != AnyScope {
		q = q.Where("provider = ?", f.Provider)
	}
	if f.Currency != "" {
		q = q.Where("currency = ?", f.Currency)
	}
	if f.Reference != "" {
		q = q.Where("reference = ?", f.Reference)
	}
	if f.CustomerID != "" {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	if f.Search != "" {
		// The payer lookup is a read-only EXISTS over the customer directory: it
		// keeps "search for the customer's email" working without making the
		// payments domain depend on the customers package.
		like := "%" + strings.ToLower(strings.TrimSpace(f.Search)) + "%"
		q = q.Where(`(lower(reference) LIKE ? OR lower(id) LIKE ?
			OR EXISTS (SELECT 1 FROM customers c
				WHERE c.id = payments.customer_id
				AND (lower(c.email) LIKE ? OR lower(c.name) LIKE ?)))`, like, like, like, like)
	}
	return q
}

// Create inserts a payment.
func (r *Repository) Create(p *Payment) error { return r.DB.Create(p).Error }

// Save persists a payment.
func (r *Repository) Save(p *Payment) error { return r.DB.Save(p).Error }

// ByID loads a payment scoped to its merchant.
func (r *Repository) ByID(merchantID, id string) (*Payment, error) {
	var p Payment
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ByReference loads a payment by merchant reference within an environment.
func (r *Repository) ByReference(merchantID, environment, reference string) (*Payment, error) {
	var p Payment
	if err := r.DB.Where("merchant_id = ? AND environment = ? AND reference = ?",
		merchantID, environment, reference).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ByProviderReference finds a payment from an inbound provider webhook.
func (r *Repository) ByProviderReference(merchantID, provider, providerReference string) (*Payment, error) {
	var p Payment
	err := r.DB.Where("merchant_id = ? AND provider = ? AND (provider_payment_id = ? OR reference = ?)",
		merchantID, provider, providerReference, providerReference).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List returns a filtered, paginated page of payments.
func (r *Repository) List(f Filter) ([]Payment, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []Payment
	if err := r.scoped(f).Order("created_at DESC").Offset(f.Offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Count returns the number of payments matching a filter.
func (r *Repository) Count(f Filter) (int64, error) {
	var total int64
	if err := r.scoped(f).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// StatusCounts tallies a filter by status, so a listing can label its own
// facets ("120 successful, 4 failed") without a second, differently-filtered
// request.
func (r *Repository) StatusCounts(f Filter) (map[string]int64, error) {
	var rows []struct {
		Status string
		Total  int64
	}
	if err := r.scoped(f).Select("status, count(*) AS total").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Total
	}
	return counts, nil
}

// Metrics returns dashboard aggregates for one environment.
func (r *Repository) Metrics(merchantID, environment string) (volume int64, successful, failed int64, series []VolumePoint, err error) {
	base := r.DB.Model(&Payment{}).Where("merchant_id = ?", merchantID)
	if environment != "" {
		base = base.Where("environment = ?", environment)
	}
	if err = base.Session(&gorm.Session{}).
		Where("status = 'successful'").
		Select("COALESCE(SUM(amount_minor),0)").Scan(&volume).Error; err != nil {
		return
	}
	if err = base.Session(&gorm.Session{}).Where("status = 'successful'").Count(&successful).Error; err != nil {
		return
	}
	if err = base.Session(&gorm.Session{}).Where("status = 'failed'").Count(&failed).Error; err != nil {
		return
	}
	query := `SELECT date_trunc('day', created_at) AS day, COALESCE(SUM(amount_minor),0) AS volume_minor
	          FROM payments WHERE merchant_id = ? AND status = 'successful'
	          AND created_at >= now() - interval '30 days'`
	args := []any{merchantID}
	if environment != "" {
		query += " AND environment = ?"
		args = append(args, environment)
	}
	query += " GROUP BY day ORDER BY day"
	err = r.DB.Raw(query, args...).Scan(&series).Error
	return
}

// VolumePoint is one day of the dashboard series.
type VolumePoint struct {
	Day         time.Time `gorm:"column:day"`
	VolumeMinor int64     `gorm:"column:volume_minor"`
}

/* ------------------------------- idempotency -------------------------------- */

// ErrIdempotencyExists signals that the key was already stored.
var ErrIdempotencyExists = errors.New("payments: idempotency key already stored")

// BeginIdempotency reserves a key.
//
// created reports whether this call inserted the reservation. When it is false,
// the returned record is the one already stored and the caller must decide
// whether to replay it, reject it, or take it over.
func (r *Repository) BeginIdempotency(record *IdempotencyRecord) (existing *IdempotencyRecord, created bool, err error) {
	insertErr := r.DB.Create(record).Error
	if insertErr == nil {
		return record, true, nil
	}
	if !isUniqueViolation(insertErr) {
		return nil, false, insertErr
	}
	var stored IdempotencyRecord
	if err := r.DB.Where("merchant_id = ? AND idempotency_key = ?",
		record.MerchantID, record.IdempotencyKey).First(&stored).Error; err != nil {
		return nil, false, err
	}
	return &stored, false, nil
}

// CompleteIdempotency stores the response for a finished request.
func (r *Repository) CompleteIdempotency(merchantID, key string, status int, body map[string]any, paymentID string) error {
	var payment *string
	if paymentID != "" {
		payment = &paymentID
	}
	return r.DB.Model(&IdempotencyRecord{}).
		Where("merchant_id = ? AND idempotency_key = ?", merchantID, key).
		Updates(map[string]any{
			"status":        idempotencyCompleted,
			"response_body": gormxJSON(body),
			"status_code":   status,
			"payment_id":    payment,
			"updated_at":    time.Now().UTC(),
		}).Error
}

// IdempotencyByKey loads a stored idempotent request.
func (r *Repository) IdempotencyByKey(merchantID, key string) (*IdempotencyRecord, error) {
	var record IdempotencyRecord
	if err := r.DB.Where("merchant_id = ? AND idempotency_key = ?", merchantID, key).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (gorm.ErrDuplicatedKey.Error() == err.Error() ||
		containsAny(err.Error(), []string{"duplicate key", "unique constraint", "23505"}))
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if len(needle) > 0 && len(value) >= len(needle) {
			for i := 0; i+len(needle) <= len(value); i++ {
				if equalFold(value[i:i+len(needle)], needle) {
					return true
				}
			}
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
