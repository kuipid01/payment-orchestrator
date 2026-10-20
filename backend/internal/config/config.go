package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
)

// Config is the single, immutable configuration object built at startup.
type Config struct {
	App       AppConfig
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Security  SecurityConfig
	Mail      MailConfig
	Providers ProviderConfig
	Limits    LimitConfig
	Billing   BillingConfig
	Log       LogConfig
}

// BillingConfig controls subscription enforcement.
type BillingConfig struct {
	// EnforcedGates names the plan gates allowed to refuse a request. An empty
	// list — the default — is log-only: every gate still measures and records
	// what it would have blocked, and nothing is blocked. Gates are flipped one
	// at a time, once the logs of a real deployment show no false positives.
	EnforcedGates []string
}

// LogConfig controls logging verbosity.
type LogConfig struct {
	// Level is the minimum level that is emitted: debug | info | warn | error.
	Level string
	// WebhookVerbose emits a line for every step of the inbound provider webhook
	// pipeline and every outbound delivery attempt. It is high volume, so it is
	// enabled outside production and off there by default.
	WebhookVerbose bool
}

// AppConfig identifies this deployment.
type AppConfig struct {
	Name        string
	Environment string // development | test | production
	BaseURL     string // public origin of this API
	FrontendURL string
}

// ServerConfig holds transport tuning.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// SecurityConfig holds secrets and crypto settings.
type SecurityConfig struct {
	EncryptionKey   string
	JWTSecret       string
	JWTAccessTTL    time.Duration
	JWTRefreshTTL   time.Duration
	APIKeyPrefix    string
	AllowedOrigins  []string
	GoogleClientID  string
	SeedPassword    string
	TrustedProxies  string
	EnableBodyLimit bool
}

// MailConfig holds transactional email settings.
type MailConfig struct {
	BrevoAPIKey      string
	BrevoSenderEmail string
	BrevoSenderName  string
}

// ProviderConfig holds provider base URLs (overridable for tests/sandboxes).
type ProviderConfig struct {
	PaystackBaseURL       string
	FlutterwaveBaseURL    string
	MonnifyTestBaseURL    string
	MonnifyLiveBaseURL    string
	StripeBaseURL         string
	PayPalTestBaseURL     string
	PayPalLiveBaseURL     string
	WebhookBaseURL        string
	RequestTimeout        time.Duration
	MaxRetries            int
	StripeManagedPayments bool
	StripeDefaultTaxCode  string
}

// LimitConfig holds rate limit budgets per endpoint category, plus the
// failed-attempt lockout applied to the public authentication endpoints.
type LimitConfig struct {
	GlobalPerMinute  int
	WritePerMinute   int
	PaymentPerMinute int

	// AuthPerMinute caps every public auth endpoint per client IP.
	AuthPerMinute int
	// AuthStrictPerMinute is the tighter budget for credential endpoints
	// (login, register, password reset), where guessing is the threat.
	AuthStrictPerMinute int
	// AuthMaxFailures is how many rejected credentials a single IP may present
	// before it is temporarily locked out. Zero disables the lockout.
	AuthMaxFailures int
	// AuthLockoutDuration is how long a lockout lasts, and the window in which
	// failures accumulate.
	AuthLockoutDuration time.Duration
}

// Load reads configuration from the environment, applying development defaults.
// It never panics: Validate reports anything missing.
func Load() Config {
	appEnv := envOr("APP_ENV", constants.EnvDevelopment)

	cfg := Config{
		App: AppConfig{
			Name:        envOr("APP_NAME", "payconnect-api"),
			Environment: appEnv,
			BaseURL:     envOr("API_BASE_URL", "http://localhost:"+envOr("PORT", "8080")),
			FrontendURL: envOr("FRONTEND_URL", "http://localhost:3000"),
		},
		Server: ServerConfig{
			Port:         envOr("PORT", "8080"),
			ReadTimeout:  envDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: envDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  envDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:      os.Getenv("DATABASE_URL"),
			Host:     envOr("DATABASE_HOST", "localhost"),
			Port:     envOr("DATABASE_PORT", "5432"),
			User:     envOr("DATABASE_USER", "payconnect"),
			Password: envOr("DATABASE_PASSWORD", "payconnect"),
			Name:     envOr("DATABASE_NAME", "payconnect"),
			SSLMode:  envOr("DATABASE_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			URL:      os.Getenv("REDIS_URL"),
			Prefix:   envOr("REDIS_PREFIX", "payconnect:"),
			PoolSize: envInt("REDIS_POOL_SIZE", 10),
			Timeout:  envDuration("REDIS_TIMEOUT", 2*time.Second),
		},
		Security: SecurityConfig{
			EncryptionKey:   envOr("ENCRYPTION_KEY", "payconnect-dev-encryption-key-32b!"),
			JWTSecret:       os.Getenv("JWT_SECRET"),
			JWTAccessTTL:    envDuration("JWT_ACCESS_TTL", 15*time.Minute),
			JWTRefreshTTL:   envDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
			APIKeyPrefix:    envOr("API_KEY_PREFIX", constants.KeyPrefixDefault),
			AllowedOrigins:  splitCSV(envOr("ALLOWED_ORIGINS", "http://localhost:3000, http://localhost:3001, http://localhost:3002, http://localhost:3003, http://localhost:3004, http://localhost:3005, http://localhost:3006, http://localhost:3007, http://localhost:3008, http://localhost:3009")),
			GoogleClientID:  os.Getenv("GOOGLE_CLIENT_ID"),
			SeedPassword:    envOr("SEED_PASSWORD", "Password123!"),
			TrustedProxies:  os.Getenv("TRUSTED_PROXIES"),
			EnableBodyLimit: envBool("ENABLE_BODY_LIMIT", true),
		},
		Mail: MailConfig{
			BrevoAPIKey:      os.Getenv("BREVO_API_KEY"),
			BrevoSenderEmail: os.Getenv("BREVO_SENDER_EMAIL"),
			BrevoSenderName:  envOr("BREVO_SENDER_NAME", "PayConnect"),
		},
		Providers: ProviderConfig{
			PaystackBaseURL:    envOr("PAYSTACK_BASE_URL", "https://api.paystack.co"),
			FlutterwaveBaseURL: envOr("FLUTTERWAVE_BASE_URL", "https://api.flutterwave.com/v3"),
			MonnifyTestBaseURL: envOr("MONNIFY_TEST_BASE_URL", "https://sandbox.monnify.com"),
			MonnifyLiveBaseURL: envOr("MONNIFY_LIVE_BASE_URL", "https://api.monnify.com"),
			StripeBaseURL:      envOr("STRIPE_BASE_URL", "https://api.stripe.com"),
			PayPalTestBaseURL:  envOr("PAYPAL_TEST_BASE_URL", "https://api-m.sandbox.paypal.com"),
			PayPalLiveBaseURL:  envOr("PAYPAL_LIVE_BASE_URL", "https://api-m.paypal.com"),
			WebhookBaseURL:     envOr("WEBHOOK_BASE_URL", "http://localhost:8080"),
			RequestTimeout:     envDuration("PROVIDER_TIMEOUT", 10*time.Second),
			MaxRetries:         envInt("PROVIDER_MAX_RETRIES", 2),
		},
		Billing: BillingConfig{
			EnforcedGates: billingGates(os.Getenv("BILLING_ENFORCED")),
		},
		Limits: LimitConfig{
			GlobalPerMinute:     envInt("RATE_LIMIT_PER_MINUTE", 300),
			WritePerMinute:      envInt("RATE_LIMIT_WRITE_PER_MINUTE", 120),
			PaymentPerMinute:    envInt("RATE_LIMIT_PAYMENT_PER_MINUTE", 60),
			AuthPerMinute:       envInt("RATE_LIMIT_AUTH_PER_MINUTE", 30),
			AuthStrictPerMinute: envInt("RATE_LIMIT_AUTH_STRICT_PER_MINUTE", 10),
			AuthMaxFailures:     envInt("AUTH_MAX_FAILURES", 5),
			AuthLockoutDuration: envDuration("AUTH_LOCKOUT_DURATION", 15*time.Minute),
		},
		Log: LogConfig{
			Level:          envOr("LOG_LEVEL", defaultLogLevel(appEnv)),
			WebhookVerbose: envBool("LOG_WEBHOOK_VERBOSE", appEnv != constants.EnvProduction),
		},
	}
	return cfg
}

// billingGates interprets BILLING_ENFORCED: empty, "false" and "none" mean
// log-only; "true" and "all" mean every gate; anything else is a comma-separated
// list of gate names. The names themselves are validated by Validate, so a typo
// is a startup failure rather than a gate that quietly never fires.
func billingGates(spec string) []string {
	trimmed := strings.ToLower(strings.TrimSpace(spec))
	switch trimmed {
	case "", "false", "0", "no", "none", "off":
		return nil
	case "true", "1", "yes", "on", constants.BillingGateAll:
		return constants.EnforceableGates()
	}
	return splitCSV(trimmed)
}

// defaultLogLevel keeps development chatty and production quiet.
func defaultLogLevel(appEnv string) string {
	if appEnv == constants.EnvProduction {
		return "info"
	}
	return "debug"
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
