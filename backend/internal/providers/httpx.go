package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// HTTPOptions configures one provider call.
type HTTPOptions struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    any
	// RetrySafe marks the call as idempotent (reads). Unsafe calls (payment or
	// refund creation) are never retried: an ambiguous write must be reconciled
	// with the provider, not repeated blindly.
	RetrySafe  bool
	MaxRetries int
	Provider   string
	Client     *http.Client
	UserAgent  string
}

// HTTPResult carries the decoded provider response.
type HTTPResult struct {
	Status int
	Body   []byte
	JSON   map[string]any
}

// DoJSON performs a provider HTTP call with timeouts, bounded retries for safe
// reads, and provider-agnostic error mapping.
func DoJSON(ctx context.Context, opts HTTPOptions) (*HTTPResult, error) {
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	attempts := 1
	if opts.RetrySafe {
		attempts = 1 + maxInt(opts.MaxRetries, 0)
		if attempts > 4 {
			attempts = 4
		}
	}

	var payload []byte
	if opts.Body != nil {
		encoded, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, apperrors.ProviderRequestFailed(opts.Provider, err)
		}
		payload = encoded
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, apperrors.ProviderTimeout(opts.Provider, ctx.Err())
			case <-time.After(backoff(attempt)):
			}
		}

		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, opts.Method, opts.URL, reader)
		if err != nil {
			return nil, apperrors.ProviderRequestFailed(opts.Provider, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if opts.UserAgent != "" {
			req.Header.Set("User-Agent", opts.UserAgent)
		}
		for key, value := range opts.Headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, apperrors.ProviderTimeout(opts.Provider, err)
			}
			continue // network failure on a safe read
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		result := &HTTPResult{Status: resp.StatusCode, Body: body}
		if len(body) > 0 {
			_ = json.Unmarshal(body, &result.JSON)
		}
		if resp.StatusCode >= 500 {
			lastErr = errors.New("provider returned " + resp.Status)
			continue // server-side failure: retry only when safe
		}
		if err := MapStatus(opts.Provider, resp.StatusCode, result.JSON); err != nil {
			return nil, err
		}
		return result, nil
	}

	if lastErr != nil {
		return nil, apperrors.ProviderUnavailable(opts.Provider, lastErr)
	}
	return nil, apperrors.ProviderUnavailable(opts.Provider, nil)
}

// MapStatus converts a non-success provider HTTP status into a typed error.
func MapStatus(provider string, status int, body map[string]any) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return apperrors.ProviderAuthenticationFailed(provider, errors.New(providerMessage(body)))
	case status == http.StatusNotFound:
		return apperrors.NotFound("provider_resource_not_found", providerMessageOrDefault(body, "The provider does not know this reference."))
	case status == http.StatusTooManyRequests:
		return apperrors.ProviderUnavailable(provider, errors.New("provider rate limit"))
	case status >= 400:
		message := providerMessage(body)
		return apperrors.ProviderRequestFailed(provider, errors.New(message)).WithDetails(map[string]any{
			"provider_status":  status,
			"provider_message": boundedProviderMessage(message),
		})
	default:
		return nil
	}
}

func boundedProviderMessage(message string) string {
	const maxRunes = 256
	runes := []rune(strings.TrimSpace(message))
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return string(runes)
}

// ProviderMessage extracts a human-readable provider error message.
func ProviderMessage(body map[string]any) string { return providerMessage(body) }

func providerMessage(body map[string]any) string {
	if body == nil {
		return "provider rejected the request"
	}
	if msg, ok := body["message"].(string); ok && msg != "" {
		return msg
	}
	if msg, ok := body["error"].(string); ok && msg != "" {
		return msg
	}
	if nested, ok := body["error"].(map[string]any); ok {
		if msg, ok := nested["message"].(string); ok && msg != "" {
			return msg
		}
	}
	if detail, ok := body["detail"].(string); ok && detail != "" {
		return detail
	}
	return "provider rejected the request"
}

func providerMessageOrDefault(body map[string]any, fallback string) string {
	if body == nil {
		return fallback
	}
	message := providerMessage(body)
	if strings.Contains(message, "rejected the request") {
		return fallback
	}
	return message
}

func backoff(attempt int) time.Duration {
	base := 200 * time.Millisecond
	for i := 1; i < attempt; i++ {
		base *= 2
	}
	if base > 2*time.Second {
		base = 2 * time.Second
	}
	return base
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
