import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "Errors" };

export default function ErrorsDocsPage() {
  return (
    <DocsPage
      title="Errors"
      lede="PayConnect uses conventional HTTP status codes and returns a consistent error envelope across all endpoints."
      sections={[
        {
          id: "error-format",
          title: "Error format",
          code: {
            language: "json",
            code: `{
  "error": {
    "type": "invalid_request_error",
    "code": "amount_below_minimum",
    "message": "Amount must be at least 10000 minor units (₦100).",
    "param": "amount"
  }
}`,
          },
        },
        {
          id: "http-statuses",
          title: "HTTP status codes",
          list: [
            "200 OK — request succeeded",
            "400 Bad Request — malformed request or invalid params",
            "401 Unauthorized — missing, invalid or revoked API key",
            "403 Forbidden — key lacks the required permission",
            "404 Not Found — resource doesn't exist (or belongs to another application)",
            "429 Too Many Requests — rate limited; back off and retry",
            "500 / 502 / 503 — something broke on our side; safe to retry idempotent requests",
          ],
        },
        {
          id: "error-types",
          title: "Error types",
          list: [
            "invalid_request_error — fix the request before retrying",
            "authentication_error — check your API key and environment",
            "provider_error — the upstream provider failed; inspect the payment timeline",
            "rate_limit_error — respect the Retry-After header",
            "api_error — unexpected server error; retry with backoff",
          ],
        },
        {
          id: "handling",
          title: "Handling errors robustly",
          code: {
            language: "ts",
            code: `try {
  const payment = await vpay.payments.create({ ... });
} catch (err) {
  if (err.type === "provider_error") {
    // surface to the user as "try another payment method"
  } else if (err.type === "invalid_request_error") {
    // this is a bug in our request — log it
  } else {
    // retry with exponential backoff
  }
}`,
          },
        },
      ]}
    />
  );
}
