package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/kuipid01/payconnect/internal/api_keys"
	"github.com/kuipid01/payconnect/internal/application"
	"github.com/kuipid01/payconnect/internal/auth"
	"github.com/kuipid01/payconnect/internal/billing"
	"github.com/kuipid01/payconnect/internal/customers"
	"github.com/kuipid01/payconnect/internal/developer"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/logs"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/middleware"
	"github.com/kuipid01/payconnect/internal/payments"
	"github.com/kuipid01/payconnect/internal/providers"
	"github.com/kuipid01/payconnect/internal/refunds"
	"github.com/kuipid01/payconnect/internal/routing"
	"github.com/kuipid01/payconnect/internal/transactions"
	"github.com/kuipid01/payconnect/internal/webhooks"
)

// bodyLimitBytes caps request bodies. Provider webhook payloads are the largest
// thing this API accepts and are far below this ceiling.
const bodyLimitBytes = 1 << 20

// NewApp builds the Fiber application: transport middleware, public routes, then
// the authenticated API surface. This is the only place that knows the complete
// public API shape.
func NewApp(d *Dependencies) *fiber.App {
	config := fiber.Config{
		AppName:               d.Cfg.App.Name,
		ErrorHandler:          errorHandler(d),
		BodyLimit:             bodyLimit(d),
		DisableStartupMessage: true,
	}
	// Rate limits and the auth lockout key on the client IP, so honour
	// X-Forwarded-For only when a trusted proxy is configured. Without that the
	// header is ignored and every caller behind a load balancer would share one
	// budget — or, worse, a client could spoof its own address.
	if proxies := trustedProxies(d.Cfg.Security.TrustedProxies); len(proxies) > 0 {
		config.ProxyHeader = fiber.HeaderXForwardedFor
		config.EnableTrustedProxyCheck = true
		config.TrustedProxies = proxies
	}
	app := fiber.New(config)

	app.Use(middleware.RequestID())
	app.Use(middleware.Recovery(d.Logger))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(d.Cfg.Security.AllowedOrigins, ","),
		AllowHeaders:     "Authorization, Content-Type, Idempotency-Key, X-Request-ID",
		AllowCredentials: true,
		ExposeHeaders:    "X-Request-ID",
		AllowMethods:     "GET, POST, PATCH, PUT, DELETE, OPTIONS",
	}))
	app.Use(middleware.Logging(d.DB.Gorm, d.Logger))

	registerHealth(app, d)

	/* --------------------------------- public --------------------------------- */
	v1 := app.Group("/v1")
	authHandler := auth.NewHandler(d.Auth, d.Responder)
	webhookHandler := webhooks.NewHandler(d.Webhooks, d.Responder)
	// Login, register and password reset are the endpoints an attacker hammers
	// with candidate credentials, so they get a tighter budget plus a temporary
	// lockout after repeated failed attempts.
	auth.RegisterPublicRoutes(v1, auth.PublicGuards{
		General: middleware.RateLimit(d.Store, d.Logger, middleware.BucketAuth, d.Cfg.Limits.AuthPerMinute, "rl"),
		Strict:  middleware.RateLimit(d.Store, d.Logger, middleware.BucketAuthStrict, d.Cfg.Limits.AuthStrictPerMinute, "rl"),
		Lockout: middleware.NewAuthThrottle(d.Store, d.Logger,
			d.Cfg.Limits.AuthMaxFailures, d.Cfg.Limits.AuthLockoutDuration, "authfail").Handler(),
	}, authHandler)
	// Developer invitations are public for the same reason: the invitee has no
	// session yet, and the single-use token plus the password they present are
	// the authentication. Accepting can create an account, so the submit takes
	// the tighter credential budget.
	//
	// They must be registered here, before the authenticated group installs its
	// `Use` handler under /v1: Fiber applies a prefix `Use` handler to routes
	// registered after it, so a public route mounted later would require auth.
	developerHandler := developer.NewHandler(d.Developers, d.Responder)
	developer.RegisterPublicRoutes(v1, developer.PublicGuards{
		General: middleware.RateLimit(d.Store, d.Logger, middleware.BucketAuth, d.Cfg.Limits.AuthPerMinute, "rl"),
		Strict:  middleware.RateLimit(d.Store, d.Logger, middleware.BucketAuthStrict, d.Cfg.Limits.AuthStrictPerMinute, "rl"),
		Lockout: middleware.NewAuthThrottle(d.Store, d.Logger,
			d.Cfg.Limits.AuthMaxFailures, d.Cfg.Limits.AuthLockoutDuration, "authfail").Handler(),
	}, developerHandler)
	// Provider webhooks are public: the provider's signature, verified against
	// the connection's own credentials, is the authentication.
	webhooks.RegisterReceiverRoutes(v1, webhookHandler)
	// The price list is public for the same reason a price tag is: the marketing
	// page and the app both render it, and a page that hard-codes a tier is how
	// two surfaces start disagreeing about what a plan includes.
	billingHandler := billing.NewHandler(d.Billing, d.Responder)
	billing.RegisterPublicRoutes(v1,
		middleware.RateLimit(d.Store, d.Logger, middleware.BucketGlobal, d.Cfg.Limits.GlobalPerMinute, "rl"),
		billingHandler)

	/* ------------------------------ authenticated ----------------------------- */
	api := v1.Group("", middleware.Authentication(d.Authenticator))
	api.Use(middleware.RateLimit(d.Store, d.Logger, middleware.BucketGlobal, d.Cfg.Limits.GlobalPerMinute, "rl"))
	sessionOnly := middleware.SessionOnly()
	paymentGuard := middleware.RateLimit(d.Store, d.Logger, middleware.BucketPayment, d.Cfg.Limits.PaymentPerMinute, "rl")

	auth.RegisterAuthenticatedRoutes(api, authHandler)
	auth.RegisterSessionRoutes(api, sessionOnly, authHandler)
	merchants.RegisterRoutes(api, sessionOnly, merchants.NewHandler(d.Merchants, d.Responder))
	developer.RegisterRoutes(api, sessionOnly, developerHandler)
	application.RegisterRoutes(api, sessionOnly, application.NewHandler(d.Applications, d.Responder))
	api_keys.RegisterRoutes(api, sessionOnly, api_keys.NewHandler(d.APIKeys, d.Responder))
	providers.RegisterRoutes(api, sessionOnly, providers.NewHandler(d.Providers, d.Responder))
	customers.RegisterRoutes(api, sessionOnly, customers.NewHandler(d.Customers, d.Responder))
	payments.RegisterRoutes(api, sessionOnly, paymentGuard, payments.NewHandler(d.Payments, d.Responder))
	refunds.RegisterRoutes(api, refunds.NewHandler(d.Refunds, d.Responder))
	routing.RegisterRoutes(api, sessionOnly, routing.NewHandler(d.Routing, d.Responder))
	transactions.RegisterRoutes(api, transactions.NewHandler(d.Transactions, d.Responder))
	webhooks.RegisterRoutes(api, sessionOnly, webhookHandler)
	logs.RegisterRoutes(api, sessionOnly, logs.NewHandler(d.Logs, d.Responder))
	billing.RegisterRoutes(api, sessionOnly, billingHandler)

	return app
}

// registerHealth mounts the probes. Liveness never touches a dependency;
// readiness checks the infrastructure the API cannot serve without.
func registerHealth(app *fiber.App, d *Dependencies) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": d.Cfg.App.Name})
	})
	app.Get("/ready", func(c *fiber.Ctx) error {
		if err := d.DB.Ping(c.UserContext()); err != nil {
			return d.Responder.Error(c, apperrors.ServiceUnavailable("The database is not reachable.", err))
		}
		if d.Store != nil {
			if _, _, err := d.Store.Get(c.UserContext(), "ready:probe"); err != nil {
				return d.Responder.Error(c, apperrors.ServiceUnavailable("The cache is not reachable.", err))
			}
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
}

// errorHandler renders every unhandled error through the standard envelope.
// Router-level failures (404, 405, 413) keep their HTTP status but never leak
// framework internals.
func errorHandler(d *Dependencies) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			switch fiberErr.Code {
			case http.StatusNotFound:
				return d.Responder.Error(c, apperrors.NotFound(apperrors.CodeNotFound, "The requested route does not exist."))
			case http.StatusMethodNotAllowed:
				return d.Responder.Error(c, apperrors.New("invalid_request_error",
					"method_not_allowed", "That method is not allowed for this route.", http.StatusMethodNotAllowed))
			case http.StatusRequestEntityTooLarge:
				return d.Responder.Error(c, apperrors.New("invalid_request_error",
					"payload_too_large", "The request body is too large.", http.StatusRequestEntityTooLarge))
			default:
				return d.Responder.Error(c, apperrors.New("invalid_request_error",
					"http_error", fiberErr.Message, fiberErr.Code))
			}
		}
		return d.Responder.Error(c, err)
	}
}

// bodyLimit returns the configured request body ceiling.
func bodyLimit(d *Dependencies) int {
	if !d.Cfg.Security.EnableBodyLimit {
		return 0
	}
	return bodyLimitBytes
}

// trustedProxies parses the comma-separated TRUSTED_PROXIES list. An empty list
// means the process is directly exposed, so forwarded headers are ignored.
func trustedProxies(raw string) []string {
	parts := strings.Split(raw, ",")
	proxies := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			proxies = append(proxies, part)
		}
	}
	return proxies
}
