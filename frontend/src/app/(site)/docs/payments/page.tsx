import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "Payments" };

export default function PaymentsDocsPage() {
  return (
    <DocsPage
      title="Payments"
      lede="Create, verify and list payments through one interface — regardless of which provider settles the money."
      sections={[
        {
          id: "create-a-payment",
          title: "Create a payment",
          code: {
            language: "ts",
            code: `const payment = await vpay.payments.create({
  amount: 20000,              // minor units — kobo for NGN
  currency: "NGN",
  customer: {
    email: "customer@example.com",
    name: "John Doe",
    phone: "+2348012345678"
  },
  reference: "order_1042",    // your idempotency handle
  provider: "auto"            // or pin: "paystack" | "flutterwave" | "monnify" | "stripe" | "paypal"
});`,
          },
          body: [
            "amount is always in minor units (kobo for NGN, cents for USD). provider: \"auto\" routes to the first healthy connected provider for the currency.",
          ],
        },
        {
          id: "verify-a-payment",
          title: "Verify a payment",
          code: {
            language: "ts",
            code: `const payment = await vpay.payments.verify("pay_9x41k2");
// → { id: "pay_9x41k2", status: "successful", amount: 20000, ... }`,
          },
          body: ["Always verify server-side before fulfilling orders — never trust client redirects alone."],
        },
        {
          id: "list-payments",
          title: "List payments",
          code: {
            language: "bash",
            code: `curl "https://api.payconnect.dev/v1/payments?status=successful&limit=20" \\
  -H "Authorization: Bearer vpay_live_…"`,
          },
          list: [
            "status — successful | pending | failed | refunded | cancelled",
            "provider — paystack | flutterwave | monnify | stripe | paypal",
            "created_after / created_before — ISO 8601 bounds",
            "cursor — pagination cursor from the previous response",
          ],
        },
        {
          id: "payment-object",
          title: "The payment object",
          code: {
            language: "json",
            code: `{
  "id": "pay_9x41k2",
  "reference": "order_1042",
  "provider_reference": "ps_1a2b3c4d",
  "amount": 20000,
  "currency": "NGN",
  "status": "successful",
  "provider": "paystack",
  "customer": { "email": "customer@example.com" },
  "fee": 30000,
  "created_at": "2026-09-30T09:30:12Z",
  "timeline": [
    { "type": "created", "at": "2026-09-30T09:30:12Z" },
    { "type": "provider_request", "at": "2026-09-30T09:30:13Z" },
    { "type": "successful", "at": "2026-09-30T09:30:44Z" }
  ]
}`,
          },
          note: "Provider responses are redacted before storage. No credential material ever appears in the payment object.",
        },
      ]}
    />
  );
}
