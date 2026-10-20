package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kuipid01/payconnect/internal/api_keys"
	"github.com/kuipid01/payconnect/internal/application"
	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/auth"
	"github.com/kuipid01/payconnect/internal/billing"
	"github.com/kuipid01/payconnect/internal/config"
	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/customers"
	"github.com/kuipid01/payconnect/internal/database"
	"github.com/kuipid01/payconnect/internal/developer"
	"github.com/kuipid01/payconnect/internal/events"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/logs"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/payments"
	"github.com/kuipid01/payconnect/internal/platform/mailer"
	"github.com/kuipid01/payconnect/internal/platform/token"
	"github.com/kuipid01/payconnect/internal/platform/vault"
	"github.com/kuipid01/payconnect/internal/providers"
	"github.com/kuipid01/payconnect/internal/providers/factory"
	"github.com/kuipid01/payconnect/internal/redis"
	"github.com/kuipid01/payconnect/internal/refunds"
	"github.com/kuipid01/payconnect/internal/routing"
	"github.com/kuipid01/payconnect/internal/transactions"
	"github.com/kuipid01/payconnect/internal/webhooks"
)

// Dependencies is the fully wired application graph. It is built exactly once in
// main and passed by value: nothing here is a global.
type Dependencies struct {
	Cfg       config.Config
	DB        *database.DB
	Store     redis.Store
	Logger    *slog.Logger
	Vault     *vault.Vault
	Bus       *events.Bus
	Audit     *audit.Store
	Responder *apihttp.Responder

	Merchants    *merchants.Service
	Auth         *auth.Service
	Developers   *developer.Service
	Applications *application.Service
	APIKeys      *api_keys.Service
	Providers    *providers.Service
	Payments     *payments.Service
	Refunds      *refunds.Service
	Customers    *customers.Service
	Transactions *transactions.Service
	Routing      *routing.Service
	Billing      *billing.Service
	Webhooks     *webhooks.Service
	Dispatcher   *webhooks.Dispatcher
	Logs         *logs.Repository

	// Authenticator resolves a bearer token (session or API key) to a principal.
	Authenticator *auth.Service
}

// devJWTSecret is the fallback signing secret outside production so a fresh
// clone can boot without a full environment.
const devJWTSecret = "payconnect-local-development-secret"

// Build wires the whole application graph. Every dependency is created here and
// injected downward; no service constructs its own collaborators.
func Build(cfg config.Config, db *database.DB, store redis.Store, logger *slog.Logger) (*Dependencies, error) {
	vlt, err := vault.New(cfg.Security.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}

	signingSecret := cfg.Security.JWTSecret
	if signingSecret == "" {
		if cfg.IsProduction() {
			return nil, fmt.Errorf("config: JWT_SECRET is required in production")
		}
		signingSecret = devJWTSecret
		logger.Warn("JWT_SECRET is unset; using the development signing key")
	}
	signer, err := token.New(signingSecret, "payconnect", "payconnect-api", cfg.Security.JWTAccessTTL)
	if err != nil {
		return nil, fmt.Errorf("token signer: %w", err)
	}

	bus := events.NewBus(logger)
	bus.Subscribe(events.LogSubscriber(logger))

	auditStore := audit.NewStore(db.Gorm, logger)
	responder := apihttp.NewResponder()

	registry := factory.NewRegistry(factory.Config{
		PaystackBaseURL:       cfg.Providers.PaystackBaseURL,
		FlutterwaveBaseURL:    cfg.Providers.FlutterwaveBaseURL,
		MonnifyTestBaseURL:    cfg.Providers.MonnifyTestBaseURL,
		MonnifyLiveBaseURL:    cfg.Providers.MonnifyLiveBaseURL,
		StripeBaseURL:         cfg.Providers.StripeBaseURL,
		PayPalTestBaseURL:     cfg.Providers.PayPalTestBaseURL,
		PayPalLiveBaseURL:     cfg.Providers.PayPalLiveBaseURL,
		Timeout:               cfg.Providers.RequestTimeout,
		MaxRetries:            cfg.Providers.MaxRetries,
		UserAgent:             cfg.App.Name,
		StripeManagedPayments: cfg.Providers.StripeManagedPayments,
		StripeDefaultTaxCode:  cfg.Providers.StripeDefaultTaxCode,
	})

	/* ------------------------------- persistence ------------------------------- */
	merchantRepo := merchants.NewRepository(db.Gorm)
	applicationRepo := application.NewRepository(db.Gorm)
	apiKeyRepo := api_keys.NewRepository(db.Gorm)
	providerRepo := providers.NewRepository(db.Gorm)
	paymentRepo := payments.NewRepository(db.Gorm)
	refundRepo := refunds.NewRepository(db.Gorm)
	customerRepo := customers.NewRepository(db.Gorm)
	txRepo := transactions.NewRepository(db.Gorm)
	routingRepo := routing.NewRepository(db.Gorm)
	webhookRepo := webhooks.NewRepository(db.Gorm)

	/* -------------------------------- services -------------------------------- */
	merchantService := merchants.NewService(merchantRepo)
	merchantService.Audit = auditStore

	providerService := providers.NewService(providerRepo, vlt, registry, auditStore)
	// The inbound webhook URL a merchant pastes into a provider dashboard is
	// built from this public origin, so it is stable regardless of request Host.
	providerService.PublicBaseURL = cfg.App.BaseURL

	apiKeyService := api_keys.NewService(apiKeyRepo, merchantRepo, vlt, auditStore, cfg.Security.APIKeyPrefix)
	// The credential logs join a payment's trace to the key that started it.
	apiKeyService.Logger = logger
	applicationService := application.NewService(applicationRepo, apiKeyService, auditStore)

	customerService := customers.NewService(customerRepo, auditStore)
	txService := transactions.NewService(txRepo)

	routingService := routing.NewService(routingRepo, providerService, auditStore)
	routingService.Preferences = providerService

	paymentService := payments.NewService(paymentRepo, registry, bus)
	paymentService.Providers = providerService
	paymentService.Routing = routingService
	paymentService.Customers = customerResolver{customers: customerService}
	paymentService.Directory = customerDirectory{customers: customerService}
	paymentService.Ledger = txService
	// A test transaction charges a sandbox connection the merchant picked, so the
	// payments domain asks the providers domain which connections are eligible.
	paymentService.TestConnections = providerService
	// The shared store gives idempotency a fast pre-check and an executionfactory.NewRegistry
	// lock; the database stays the source of truth.
	paymentService.Store = store
	paymentService.Logger = logger

	refundService := refunds.NewService(
		refundRepo, paymentRepo, providerService, registry,
		refundLedger{transactions: txService}, bus, auditStore,
	)

	webhookService := webhooks.NewService(
		webhookRepo, providerService, registry, paymentService, vlt, bus, auditStore,
	)
	// The webhook gateway traces every inbound step and every outbound attempt
	// when LOG_WEBHOOK_VERBOSE is set, so a provider calling us can be followed
	// from receipt to delivery.
	webhookService.Logger = logger
	webhookService.Verbose = cfg.Log.WebhookVerbose
	dispatcher := webhooks.NewDispatcher(webhookRepo, vlt, logger)
	dispatcher.Verbose = cfg.Log.WebhookVerbose
	webhookService.Dispatcher = dispatcher
	bus.Subscribe(dispatcher)

	billingService := billing.NewService(
		billing.NewRepository(db.Gorm), merchantRepo, store, auditStore, logger,
	)
	// Checkout stays closed until a collection provider (Paystack for NGN,
	// Stripe for USD) is wired: a chosen plan is recorded and reported as
	// pending, and nothing is charged in the meantime.
	billingService.CheckoutEnabled = false

	authService := auth.NewService(merchantRepo, signer, logger)
	authService.Audit = auditStore
	authService.GoogleClientID = cfg.Security.GoogleClientID
	authService.FrontendURL = cfg.App.FrontendURL
	authService.RefreshTTL = cfg.Security.JWTRefreshTTL
	authService.APIKeys = apiKeyService
	if cfg.Mail.BrevoAPIKey != "" {
		brevo := mailer.NewBrevo(cfg.Mail.BrevoAPIKey, cfg.Mail.BrevoSenderEmail, cfg.Mail.BrevoSenderName)
		authService.Mailer = brevo
	}
	developerService := developer.NewService(merchantRepo, authService.Mailer, cfg.App.FrontendURL,
		developerSessions{auth: authService})
	// The developer platform reads the applications and developer-scoped keys a
	// grant reaches, and revokes those keys when the grant is removed.
	developerService.Apps = applicationRepo
	developerService.Keys = apiKeyService
	developerService.Webhooks = developerWebhooks{service: webhookService}

	/* --------------------------------- gates ---------------------------------- */
	// The plan gates are wired here and nowhere else. Every call site refuses
	// through Decision.Refusal, which returns nil until a gate is named in
	// BILLING_ENFORCED — so this wiring cannot block anything by itself, and a
	// deployment that says nothing runs in log-only mode.
	billingService.SetEnforcedGates(cfg.Billing.EnforcedGates)
	applicationService.Gates = billingService
	webhookService.Gates = billingService
	routingService.Gates = billingService
	developerService.Gates = billingService
	if enforced := billingService.EnforcedGates(); len(enforced) == 0 {
		logger.Info("billing gates are log-only; limits are measured and logged, never refused",
			"gates", constants.EnforceableGates())
	} else {
		logger.Info("billing gates enforced; any gate not listed stays log-only",
			"enforced", enforced, "all_gates", constants.EnforceableGates())
	}

	return &Dependencies{
		Cfg:           cfg,
		DB:            db,
		Store:         store,
		Logger:        logger,
		Vault:         vlt,
		Bus:           bus,
		Audit:         auditStore,
		Responder:     responder,
		Merchants:     merchantService,
		Auth:          authService,
		Developers:    developerService,
		Applications:  applicationService,
		APIKeys:       apiKeyService,
		Providers:     providerService,
		Payments:      paymentService,
		Refunds:       refundService,
		Customers:     customerService,
		Transactions:  txService,
		Routing:       routingService,
		Billing:       billingService,
		Webhooks:      webhookService,
		Dispatcher:    dispatcher,
		Logs:          logs.NewRepository(db.Gorm),
		Authenticator: authService,
	}, nil
}

// developerWebhooks adapts the webhooks domain to the developer platform's
// sample-delivery port. The adapter lives here so the developer package does not
// have to import webhooks just to name a delivery.
type developerWebhooks struct{ service *webhooks.Service }

// TestDeliver sends a sample event and reports what the endpoint answered.
func (a developerWebhooks) TestDeliver(ctx context.Context, merchantID, applicationID string) (developer.TestDelivery, error) {
	result, err := a.service.TestDeliver(ctx, merchantID, applicationID)
	if err != nil {
		return developer.TestDelivery{}, err
	}
	out := developer.TestDelivery{
		EndpointID:   result.Delivery.EndpointID,
		Status:       result.Delivery.Status,
		ResponseCode: result.Delivery.ResponseCode,
		LastError:    result.Delivery.LastError,
	}
	if result.Endpoint != nil {
		out.URL = result.Endpoint.URL
	}
	return out, nil
}

// developerSessions adapts the auth service to the developer domain's
// SessionIssuer port. The adapter lives here so neither domain has to import the
// other just to name a token.
type developerSessions struct{ auth *auth.Service }

// IssueFor mints a session for a membership that already exists.
func (s developerSessions) IssueFor(userID, merchantID string) (*developer.Session, error) {
	tokens, _, err := s.auth.IssueFor(userID, merchantID)
	if err != nil {
		return nil, err
	}
	return &developer.Session{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
		MerchantID:   tokens.MerchantID,
		Role:         tokens.Role,
	}, nil
}

// customerResolver adapts the customer directory to the payments domain's port.
type customerResolver struct{ customers *customers.Service }

// FindOrCreate resolves the payer record for a payment.
func (r customerResolver) FindOrCreate(merchantID string, in payments.CustomerInput) (string, error) {
	return r.customers.FindOrCreate(merchantID, in.Email, in.Name, in.Phone)
}

// customerDirectory adapts the customer directory to the payments domain's
// display port, so a payment listing can show its payer without the payments
// package importing the customers package.
type customerDirectory struct{ customers *customers.Service }

// Lookup resolves a set of payer ids into display references.
func (d customerDirectory) Lookup(merchantID string, ids []string) (map[string]payments.CustomerRef, error) {
	rows, err := d.customers.Lookup(merchantID, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]payments.CustomerRef, len(rows))
	for i := range rows {
		row := &rows[i]
		out[row.ID] = payments.CustomerRef{
			ID:    row.ID,
			Name:  row.Name,
			Email: row.Email,
			Phone: row.Phone,
		}
	}
	return out, nil
}

// refundLedger adapts the transactions ledger to the refunds domain's port, so
// neither package needs to know about the other's types.
type refundLedger struct{ transactions *transactions.Service }

// RecordRefund persists the normalized refund movement.
func (l refundLedger) RecordRefund(refund *refunds.RefundRecord) error {
	if refund == nil {
		return nil
	}
	return l.transactions.RecordRefund(&transactions.RefundRecord{
		ID:          refund.ID,
		MerchantID:  refund.MerchantID,
		PaymentID:   refund.PaymentID,
		Provider:    refund.Provider,
		Environment: refund.Environment,
		AmountMinor: refund.AmountMinor,
		Currency:    refund.Currency,
		Status:      refund.Status,
		Reference:   refund.Reference,
	})
}
