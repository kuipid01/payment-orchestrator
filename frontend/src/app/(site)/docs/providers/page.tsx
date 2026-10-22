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
            "GET /v1/providers lists each provider's supported_options — the payment.options fields it accepts — so a client can discover capabilities instead of hard-coding them. Supplying options pins the payment to that provider and turns fallback off, because a fallback provider would silently ignore them.",
          ],
        },
        {
          id: "payment-options",
          title: "Provider payment options",
          body: [
            "Provider-specific settings belong on the payment request, under a namespace: options.stripe or options.paypal. They are typed and validated against the routed provider before any provider API call, so an unsupported field is a 400 with an actionable message rather than something silently dropped.",
            "Stripe accepts merchant_of_record (\"standard\" or \"managed\"), product_name, product_description, product_tax_code, customer_email, success_url, cancel_url, submit_type, locale, billing_address_collection, allow_promotion_codes, setup_future_usage and payment_intent_metadata. Managed Payments sets managed_payments.enabled on the Checkout Session and requires an eligible product tax code; without one the request is rejected. Standard Payments never sends that configuration.",
            "PayPal accepts description, invoice_id, custom_id, payer_email, brand_name, locale, landing_page, user_action and shipping_preference, mapped onto the Orders v2 purchase unit and application context. Capture/authorization behaviour is not caller-selectable: PayConnect creates CAPTURE-intent orders and captures on verification.",
            "Paystack, Flutterwave and Monnify accept no options; sending any is a validation error.",
          ],
          code: {
            language: "bash",
            code: `curl -X POST https://api.payconnect.dev/v1/payments \\
  -H "Authorization: Bearer vpay_live_…" \\
  -H "Idempotency-Key: order_1042" \\
  -H "Content-Type: application/json" \\
  -d '{
        "amount": 20000,
        "currency": "USD",
        "customer": { "email": "customer@example.com" },
        "callback_url": "https://store.example/return",
        "options": {
          "stripe": {
            "merchant_of_record": "standard",
            "product_name": "Pro plan",
            "product_tax_code": "txcd_10000000"
          }
        }
      }'`,
          },
          note: "Options participate in the idempotency hash: replaying the same Idempotency-Key with different provider settings is refused instead of returning the earlier payment. Dashboard test transactions always use standard Payments, so merchant_of_record \"managed\" is rejected there.",
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
