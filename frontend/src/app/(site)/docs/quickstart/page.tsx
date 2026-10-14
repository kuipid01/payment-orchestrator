import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "Quickstart" };

export default function QuickstartPage() {
  return (
    <DocsPage
      title="Quickstart"
      lede="Go from zero to a successful test payment in about five minutes."
      sections={[
        {
          id: "1-connect-a-provider",
          title: "1. Connect a provider",
          body: [
            "In the dashboard, open Providers → Connect and add your Paystack test secret key. The key is validated against the provider and then encrypted — you will only ever see its last 4 characters.",
          ],
        },
        {
          id: "2-create-an-application",
          title: "2. Create an application",
          body: [
            "Applications group payments, keys, webhooks and logs. Create one for the surface you're building (e.g. “My Ecommerce Store”) in the test environment.",
          ],
          code: {
            language: "bash",
            code: `# Via the dashboard, or the API:
curl -X POST https://api.payconnect.dev/v1/applications \\
  -H "Authorization: Bearer <merchant-session>" \\
  -d '{ "name": "My Ecommerce Store", "environment": "test" }'`,
          },
        },
        {
          id: "3-grab-your-api-key",
          title: "3. Grab your API key",
          body: ["Copy the vpay_test_… key shown once after creation. This is the key your developer uses — it is not a provider secret."],
          note: "Copy your key immediately. The full key is never shown again; only its masked form (vpay_test_••••7f83).",
        },
        {
          id: "4-create-your-first-payment",
          title: "4. Create your first payment",
          code: {
            language: "ts",
            code: `const payment = await vpay.payments.create({
  amount: 20000,
  currency: "NGN",
  customer: { email: "customer@example.com" },
  reference: "order_1042",
});

// → {
//     id: "pay_9x41k2",
//     status: "pending",
//     checkout_url: "https://checkout.payconnect.dev/pay_9x41k2"
//   }`,
          },
        },
        {
          id: "5-listen-for-webhooks",
          title: "5. Listen for webhooks",
          body: ["Register an endpoint and PayConnect will POST a unified event body for every payment state change:"],
          code: {
            language: "json",
            code: `{
  "event": "payment.successful",
  "data": {
    "id": "pay_9x41k2",
    "reference": "order_1042",
    "amount": 20000,
    "currency": "NGN",
    "status": "successful"
  }
}`,
          },
        },
      ]}
    />
  );
}
