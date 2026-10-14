package application

import (
	"context"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/api_keys"
	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/billing"
	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

const (
	StatusActive    = "active"
	StatusPaused    = "paused"
	EnvironmentTest = "test"
	EnvironmentLive = "live"
)

// PlanGates is the one billing question this package asks: is there room for
// another application on the organization's plan? It is an interface so the
// package needs neither a database nor a billing service to be tested, and so a
// nil gate (tests, the seeder, a build with billing switched off) means "no
// limit" rather than a runtime error.
//
// The gate answers with a Decision, not an error, and only Refusal() turns it
// into one — which is how the whole enforcement stays log-only until a gate is
// deliberately turned on.
type PlanGates interface {
	CheckApplications(ctx context.Context, merchantID string) billing.Decision
}

// Service owns application lifecycle rules.
type Service struct {
	Repo    *Repository
	APIKeys *api_keys.Service
	Audit   audit.Recorder
	Gates   PlanGates
	Now     func() time.Time
}

// NewService builds the application service.
func NewService(repo *Repository, apiKeys *api_keys.Service, recorder audit.Recorder) *Service {
	return &Service{Repo: repo, APIKeys: apiKeys, Audit: recorder, Now: time.Now}
}

// Create validates and persists an environment-neutral application, then mints
// its test and production key pair in one step. The pair is either general
// (team-shared) or bound to the developer grant named in the input, so a
// developer's keys can be revoked without rotating the integration key.
//
// The keys are minted after the application row exists because a key must carry
// its application id. If any mint fails, the whole creation is rolled back:
// an application is never left behind without the credentials it promised.
func (s *Service) Create(ctx context.Context, merchantID, actorName, actorRole string, in Input) (*CreatedResult, error) {
	if err := validateInput(in); err != nil {
		return nil, err
	}
	// The plan is consulted after validation, so a request that was going to be
	// rejected anyway never records a would-be refusal.
	if err := s.checkPlan(ctx, merchantID); err != nil {
		return nil, err
	}
	var developerAccessID *string
	if in.KeyScope == api_keys.ScopeDeveloper {
		id := strings.TrimSpace(in.DeveloperAccessID)
		if id == "" {
			return nil, apperrors.Validation(apperrors.CodeValidationError,
				"developer_access_id is required for developer-scoped keys.")
		}
		developerAccessID = &id
	}
	now := s.Now().UTC()
	app := &Application{
		ID:          gormx.NewID("app"),
		MerchantID:  merchantID,
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		Environment: EnvironmentTest,
		Status:      in.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.Repo.Create(app); err != nil {
		return nil, err
	}

	environments := []struct{ environment, label string }{
		{EnvironmentTest, app.Name + " test key"},
		{EnvironmentLive, app.Name + " production key"},
	}
	result := &CreatedResult{Application: app, Keys: make([]CreatedKey, 0, len(environments))}
	for _, spec := range environments {
		created, err := s.APIKeys.Create(api_keys.CreateInput{
			MerchantID:        merchantID,
			ActorName:         actorName,
			ActorRole:         actorRole,
			Name:              spec.label,
			Type:              constants.APIKeySecret,
			Environment:       spec.environment,
			ApplicationID:     &app.ID,
			DeveloperAccessID: developerAccessID,
		})
		if err != nil {
			s.rollbackCreate(merchantID, actorName, actorRole, app.ID, now, result.Keys)
			return nil, err
		}
		result.Keys = append(result.Keys, CreatedKey{
			ID:          created.Key.ID,
			Environment: spec.environment,
			Scope:       created.Key.Scope(),
			Secret:      created.Secret,
		})
		if spec.environment == EnvironmentTest {
			result.APIKey = created.Secret
		}
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Application created", Detail: app.ID + " (" + scopeLabel(developerAccessID) + ")", Severity: "info",
	})
	return result, nil
}

// checkPlan asks the plan gate, when one is wired. In log-only mode — the
// default — the gate measures and logs and this returns nil, so creating an
// application cannot fail for a billing reason until enforcement is turned on.
func (s *Service) checkPlan(ctx context.Context, merchantID string) error {
	if s.Gates == nil {
		return nil
	}
	return s.Gates.CheckApplications(ctx, merchantID).Refusal()
}

// rollbackCreate undoes a partially built application: the keys already minted
// are revoked and the application is soft-deleted.
func (s *Service) rollbackCreate(merchantID, actorName, actorRole, applicationID string, now time.Time, minted []CreatedKey) {
	for _, key := range minted {
		_ = s.APIKeys.Revoke(merchantID, actorName, actorRole, key.ID)
	}
	_, _ = s.Repo.SoftDelete(merchantID, applicationID, now)
}

// scopeLabel names the key scope for audit detail.
func scopeLabel(developerAccessID *string) string {
	if developerAccessID == nil {
		return api_keys.ScopeGeneral
	}
	return api_keys.ScopeDeveloper
}

// Get returns one application, enforcing the caller's application scope.
func (s *Service) Get(merchantID, applicationID string, principal *identity.Principal) (*Application, error) {
	if principal != nil && principal.ApplicationID != "" && principal.ApplicationID != applicationID {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "Application not found.")
	}
	app, err := s.Repo.ByID(merchantID, applicationID)
	if err != nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "Application not found.")
	}
	return app, nil
}

// WithStats returns an application with its lifetime usage statistics.
func (s *Service) WithStats(app *Application) (ApplicationResponse, error) {
	stats, err := s.Repo.Stats([]string{app.ID})
	if err != nil {
		return ApplicationResponse{}, err
	}
	return ToResponse(app, stats[app.ID]), nil
}

// List returns applications visible to the caller.
func (s *Service) List(merchantID, applicationID string, principal *identity.Principal) ([]ApplicationResponse, error) {
	if principal != nil && principal.ApplicationID != "" {
		applicationID = principal.ApplicationID
	}
	apps, err := s.Repo.List(ListFilter{MerchantID: merchantID, ApplicationID: applicationID, Limit: 100})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(apps))
	for i := range apps {
		ids = append(ids, apps[i].ID)
	}
	stats, err := s.Repo.Stats(ids)
	if err != nil {
		return nil, err
	}
	out := make([]ApplicationResponse, 0, len(apps))
	for i := range apps {
		out = append(out, ToResponse(&apps[i], stats[apps[i].ID]))
	}
	return out, nil
}

// Update applies partial application changes.
func (s *Service) Update(merchantID, id string, actorName, actorRole string, in Input) (*Application, error) {
	if err := validateUpdateInput(in); err != nil {
		return nil, err
	}
	app, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "Application not found.")
	}
	if in.Name != "" {
		app.Name = strings.TrimSpace(in.Name)
	}
	if in.Description != "" {
		app.Description = strings.TrimSpace(in.Description)
	}
	if in.Status != "" {
		app.Status = in.Status
	}
	app.UpdatedAt = s.Now().UTC()
	if err := s.Repo.Save(app); err != nil {
		return nil, err
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Application updated", Detail: app.ID, Severity: "info",
	})
	return app, nil
}

// Delete revokes application keys, then soft-deletes the application.
func (s *Service) Delete(merchantID, id, actorName, actorRole string) error {
	now := s.Now().UTC()
	if err := s.APIKeys.RevokeByApplication(merchantID, id, now); err != nil {
		return err
	}
	affected, err := s.Repo.SoftDelete(merchantID, id, now)
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperrors.NotFound(apperrors.CodeNotFound, "Application not found.")
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID, ActorName: actorName, ActorRole: actorRole,
		Action: "Application deleted", Detail: id, Severity: "warning",
	})
	return nil
}

func validateInput(in Input) error {
	if strings.TrimSpace(in.Name) == "" {
		return apperrors.Validation(apperrors.CodeValidationError, "name is required.")
	}
	if in.Status == "" {
		in.Status = StatusActive
	}
	if in.Status != StatusActive && in.Status != StatusPaused {
		return apperrors.Validation(apperrors.CodeValidationError, "status must be active or paused.")
	}
	if in.KeyScope != "" && in.KeyScope != api_keys.ScopeGeneral && in.KeyScope != api_keys.ScopeDeveloper {
		return apperrors.Validation(apperrors.CodeValidationError, "key_scope must be general or developer.")
	}
	return nil
}

func validateUpdateInput(in Input) error {
	if in.Name != "" && strings.TrimSpace(in.Name) == "" {
		return apperrors.Validation(apperrors.CodeValidationError, "name must not be empty.")
	}
	if in.Status != "" && in.Status != StatusActive && in.Status != StatusPaused {
		return apperrors.Validation(apperrors.CodeValidationError, "status must be active or paused.")
	}
	return nil
}
