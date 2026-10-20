import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "Providers" };

export default function ProvidersDocsPage() {
  return (
    <DocsPage
      title="Providers"
      lede="Provider connections are the only place credentials exist — managed by merchants in the dashboard, invisible to developers."
      sections={[
        {
          id: "supported-providers",
          title: "Supported providers",
          list: [
            "Paystack — Nigeria & Ghana; cards, bank transfer, USSD, mobile money",
            "Flutterwave — Pan-African; 30+ countries and currencies",
            "Monnify — Nigerian bank transfer and hosted checkout",
            "Stripe — global cards, wallets and local payment methods",
            "PayPal — global PayPal balance and card checkout (no NGN, GHS or KES)",
          ],
        },
        {
          id: "connecting",
          title: "Connecting a provider",
          body: [
            "Merchants connect a provider in the dashboard under Providers → Connect. Choose an environment, paste the credentials, and PayConnect validates them against the provider before storing them encrypted.",
            "PayPal needs two values: the REST app client ID (`credentials.public_key`) and its client secret. Monnify needs the API key, contract code and secret key together.",
          ],
          code: {
            language: "bash",
            code: `curl -X POST https://api.payconnect.dev/v1/providers/connections \\
  -H "Authorization: Bearer <merchant-session>" \\
  -H "Content-Type: application/json" \\
  -d '{
        "provider": "stripe",
        "environment": "live",
        "credentials": { "secret_key": "sk_live_…" }
      }'

# → {
#     "id": "conn_01",
#     "provider": "stripe",
#     "status": "connected",
#     "secret_last4": "9d21",
#     "webhook_url": "https://api.payconnect.dev/v1/webhooks/stripe/conn_01"
#   }`,
          },
          note: "The full secret is never returned by any API — including this one. Only secret_last4 is ever exposed. Rotate with PATCH and send only the credential that changed; omitted fields keep their stored values.",
        },
        {
          id: "inbound-webhooks",
          title: "Inbound webhooks",
          body: [
            "Each connection gets its own endpoint at /v1/webhooks/{provider}/{connection_id}. Paste that URL into the provider dashboard, then save the value the provider shows for it: Stripe's whsec_ signing secret, PayPal's webhook ID, or the verif-hash you chose for Flutterwave. Paystack and Monnify sign with the stored secret key and need no extra value.",
          ],
        },
        {
          id: "routing",
          title: "Routing & failover",
          body: [
            "With provider: \"auto\", PayConnect routes each payment to the first healthy connected provider that supports the currency. If a provider errors before authorization, the request fails fast with a provider_error so you can retry.",
          ],
        },
        {
          id: "rotation",
          title: "Rotating provider credentials",
          body: [
            "Rotate from the provider's detail page. PayConnect stores the new credential, re-validates against the provider, and writes an entry to the audit log. In-flight payments are unaffected.",
          ],
        },
      ]}
    />
  );
}
