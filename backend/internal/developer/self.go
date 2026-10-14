package developer

import (
	"context"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/application"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
)

// WebhookTester delivers a sample event to one of a merchant's endpoints. The
// webhooks domain implements it; keeping it a port means the developer package
// never learns how a delivery is actually signed or retried.
type WebhookTester interface {
	TestDeliver(ctx context.Context, merchantID, applicationID string) (TestDelivery, error)
}

// TestDelivery is the outcome of one sample delivery: where it went, what the
// endpoint answered, and why it failed when it did.
type TestDelivery struct {
	EndpointID   string `json:"endpoint_id"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	ResponseCode *int   `json:"response_code,omitempty"`
	LastError    string `json:"last_error,omitempty"`
}

// SelfAccess is one merchant a signed-in developer has been added to, together
// with the applications and developer-scoped keys that grant reaches. It is the
// read model behind the developer platform tab: a developer who was invited to
// another merchant sees exactly this, and nothing about provider credentials.
type SelfAccess struct {
	AccessID     string            `json:"access_id"`
	MerchantID   string            `json:"merchant_id"`
	MerchantName string            `json:"merchant_name"`
	Status       string            `json:"status"`
	Permissions  []string          `json:"permissions"`
	InvitedAt    string            `json:"invited_at"`
	AcceptedAt   *string           `json:"accepted_at,omitempty"`
	Applications []SelfApplication `json:"applications"`
}

// SelfApplication is one application inside a merchant the developer can use.
type SelfApplication struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Keys        []SelfKey `json:"keys"`
}

// SelfKey is a masked, developer-scoped key. Secrets are never re-shown.
type SelfKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Masked      string `json:"masked"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// MyAccess lists every merchant grant that belongs to the signed-in identity:
// accepted grants by developer record, and still-pending invitations by email.
// Keys and applications are resolved per grant so a developer only ever sees the
// scope they were given.
func (s *Service) MyAccess(userID, email string) ([]SelfAccess, error) {
	if userID == "" {
		return nil, apperrors.AuthenticationRequired()
	}
	accesses, err := s.Repo.DeveloperMerchantAccessesByUser(userID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(accesses))
	for _, access := range accesses {
		seen[access.ID] = struct{}{}
	}
	if email != "" {
		pending, err := s.Repo.DeveloperMerchantAccessesByEmail(email)
		if err != nil {
			return nil, err
		}
		for _, access := range pending {
			if _, ok := seen[access.ID]; ok || access.Status == "revoked" {
				continue
			}
			seen[access.ID] = struct{}{}
			accesses = append(accesses, access)
		}
	}

	out := make([]SelfAccess, 0, len(accesses))
	for i := range accesses {
		access := &accesses[i]
		if access.Status == "revoked" {
			continue
		}
		out = append(out, s.selfAccess(access))
	}
	return out, nil
}

// TestWebhook sends a sample event to an application the signed-in developer
// actually has access to.
//
// Authorization comes from the grant, not the caller: the request names an
// application, and only an accepted grant that reaches that application can map
// it to the merchant whose endpoint receives the delivery. A developer therefore
// cannot probe anybody else's receiver.
func (s *Service) TestWebhook(ctx context.Context, userID, email, applicationID string) (*TestDelivery, error) {
	if s.Webhooks == nil {
		return nil, apperrors.ServiceUnavailable("Webhook delivery is not configured.", nil)
	}
	applicationID = strings.TrimSpace(applicationID)
	if applicationID == "" {
		return nil, apperrors.Validation(apperrors.CodeValidationError, "application id is required.")
	}
	accesses, err := s.MyAccess(userID, email)
	if err != nil {
		return nil, err
	}
	merchantID := ""
	for i := range accesses {
		access := &accesses[i]
		if access.Status != "active" {
			continue
		}
		for _, app := range access.Applications {
			if app.ID == applicationID {
				merchantID = access.MerchantID
				break
			}
		}
		if merchantID != "" {
			break
		}
	}
	if merchantID == "" {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "Application not found.")
	}
	delivery, err := s.Webhooks.TestDeliver(ctx, merchantID, applicationID)
	if err != nil {
		return nil, err
	}
	return &delivery, nil
}

// selfAccess builds one grant view, degrading gracefully when the optional
// application and key repositories are not wired (invitation-only tests).
func (s *Service) selfAccess(access *merchants.DeveloperMerchantAccess) SelfAccess {
	row := SelfAccess{
		AccessID:     access.ID,
		MerchantID:   access.MerchantID,
		MerchantName: access.MerchantID,
		Status:       access.Status,
		Permissions:  access.Permissions.Data,
		InvitedAt:    access.CreatedAt.UTC().Format(time.RFC3339),
		Applications: []SelfApplication{},
	}
	if row.Permissions == nil {
		row.Permissions = []string{}
	}
	if access.AcceptedAt != nil {
		formatted := access.AcceptedAt.UTC().Format(time.RFC3339)
		row.AcceptedAt = &formatted
	}
	if merchant, err := s.Repo.MerchantByID(access.MerchantID); err == nil {
		if merchant.BusinessName != "" {
			row.MerchantName = merchant.BusinessName
		}
	}

	keysByApplication := s.scopedKeys(access)
	if s.Apps == nil {
		return row
	}
	apps, err := s.Apps.List(application.ListFilter{MerchantID: access.MerchantID, Limit: 100})
	if err != nil {
		return row
	}
	for i := range apps {
		app := &apps[i]
		keys := keysByApplication[app.ID]
		if keys == nil {
			keys = []SelfKey{}
		}
		row.Applications = append(row.Applications, SelfApplication{
			ID:          app.ID,
			Name:        app.Name,
			Description: app.Description,
			Status:      app.Status,
			Keys:        keys,
		})
	}
	return row
}

// scopedKeys groups a grant's developer-scoped keys by application.
func (s *Service) scopedKeys(access *merchants.DeveloperMerchantAccess) map[string][]SelfKey {
	grouped := map[string][]SelfKey{}
	if s.Keys == nil {
		return grouped
	}
	keys, err := s.Keys.List(access.MerchantID, "", "", access.ID)
	if err != nil {
		return grouped
	}
	now := time.Now().UTC()
	for i := range keys {
		key := &keys[i]
		applicationID := key.ApplicationIDValue()
		grouped[applicationID] = append(grouped[applicationID], SelfKey{
			ID:          key.ID,
			Name:        key.Name,
			Environment: key.Environment,
			Masked:      key.SecretMasked,
			Status:      key.Status(now),
			CreatedAt:   key.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return grouped
}
