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
            "Stripe — global cards, wallets and local payment methods",
          ],
        },
        {
          id: "connecting",
          title: "Connecting a provider",
          body: [
            "Merchants connect a provider in the dashboard under Providers → Connect. Choose an environment, paste the secret key, and PayConnect validates it against the provider before storing it encrypted.",
          ],
          code: {
            language: "bash",
            code: `curl -X POST https://api.payconnect.dev/v1/providers/paystack/connect \\
  -H "Authorization: Bearer <merchant-session>" \\
  -d '{ "environment": "live", "secretKey": "sk_live_…" }'

# → {
#     "id": "prv_ps_01",
#     "status": "connected",
#     "secret_last4": "9d21"
#   }`,
          },
          note: "The full secret is never returned by any API — including this one. Only secret_last4 is ever exposed.",
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
