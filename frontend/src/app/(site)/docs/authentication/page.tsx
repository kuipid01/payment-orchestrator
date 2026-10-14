import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "Authentication" };

export default function AuthenticationPage() {
  return (
    <DocsPage
      title="Authentication"
      lede="All API requests authenticate with a PayConnect API key sent as a bearer token. Keys are scoped to one application and one environment."
      sections={[
        {
          id: "bearer-tokens",
          title: "Bearer tokens",
          code: {
            language: "bash",
            code: `curl https://api.payconnect.dev/v1/payments \\
  -H "Authorization: Bearer vpay_test_7f83…9d21"`,
          },
          body: [
            "Live keys start with vpay_live_ and touch real money. Test keys start with vpay_test_ and hit the sandbox. There is no cross-environment access.",
          ],
        },
        {
          id: "key-scoping",
          title: "Key scoping",
          list: [
            "Each key belongs to exactly one application.",
            "Permissions on the key (create, verify, view payments; receive webhooks) are enforced server-side.",
            "Keys never grant access to provider credentials, other applications, or org settings.",
          ],
        },
        {
          id: "rotation",
          title: "Rotation & revocation",
          body: [
            "Rotate a key to get a new secret with the same permissions — the old key keeps working for 24 hours so you can deploy without downtime.",
            "Revoke kills a key immediately. All in-flight requests with that key fail with 401.",
          ],
          note: "Rotating a PayConnect key does NOT touch your Paystack/Flutterwave/Stripe credentials — those are managed separately under Providers.",
        },
        {
          id: "errors",
          title: "Auth errors",
          code: {
            language: "json",
            code: `{
  "error": {
    "type": "authentication_error",
    "code": "invalid_api_key",
    "message": "The API key provided is invalid, revoked, or missing."
  }
}`,
          },
        },
      ]}
    />
  );
}
