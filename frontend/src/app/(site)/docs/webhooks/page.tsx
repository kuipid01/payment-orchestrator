import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "Webhooks" };

export default function WebhooksDocsPage() {
  return (
    <DocsPage
      title="Webhooks"
      lede="Webhooks move in two directions. Providers notify PayConnect on a URL per connection; PayConnect notifies you on one URL for the whole merchant. Both are signed, retried and fully logged."
      sections={[
        {
          id: "directions",
          title: "The two directions",
          body: [
            "Inbound: Paystack, Flutterwave and Monnify send their events to PayConnect. Each provider connection gets its own URL, generated when you connect the provider, because the URL is what selects the credentials that verify the signature.",
            "Outbound: PayConnect sends normalized events to your endpoint. There is one destination per merchant and every application reuses it, so you register a URL once and all of your integrations receive the same events.",
          ],
          list: [
            "Inbound URL — one per provider connection (test and live are separate). Found on the provider's page in the dashboard.",
            "Outbound URL — one per merchant. Found on the Webhooks page.",
          ],
        },
        {
          id: "inbound",
          title: "Inbound: receiving your providers' events",
          body: [
            "When you connect a provider, PayConnect shows you a webhook URL shaped like https://api.payconnect.com/v1/webhooks/{provider}/{connection_id}. Paste it into that provider's dashboard to turn its notifications on. The connection id is an opaque token — it identifies the credentials that verify the request, not your account.",
          ],
          list: [
            "Paystack — paste the URL only. Paystack signs with your secret key, so there is nothing else to configure.",
            "Flutterwave — paste the URL, then set a secret hash in your Flutterwave dashboard and save the same value on the connection.",
            "Monnify — paste the URL only. Production notifications are signed using a SHA-512 hash derived from your secret key and the request body; sandbox notifications do not include a signature.",
          ],
          note: "The inbound URL is public by design. Anyone can POST to it, but nothing is trusted until its signature is verified against your stored credentials, so a forged request is rejected.",
        },
        {
          id: "outbound",
          title: "Outbound: delivering events to you",
          body: [
            "Register one endpoint on the Webhooks page and every application's events are delivered to it. PayConnect signs each request with the endpoint's whsec_… secret and retries failures automatically.",
            "If a single application must receive events somewhere else, you can add a per-application override, but the default is one shared destination.",
          ],
        },
        {
          id: "events",
          title: "Events",
          list: [
            "payment.created — a payment was initialized",
            "payment.processing — the provider is processing the charge",
            "payment.succeeded — funds confirmed by the provider",
            "payment.failed — provider declined or errored",
            "payment.cancelled — the payment was cancelled",
            "refund.created / refund.succeeded / refund.failed — refund lifecycle",
            "chargeback.created / chargeback.updated — dispute lifecycle",
          ],
        },
        {
          id: "payload",
          title: "Payload",
          body: [
            "Every event has the same shape, whichever provider processed the payment. Only the event type and the contents of data change; the envelope never does, so you write one parser and it keeps working when you connect another provider.",
            "data is the same object the payments API returns for that payment — the normalized representation, not the provider's. provider names who processed it, so you never have to branch on their format.",
            "meta carries the delivery-level facts: the contract version to pin against, the environment the event belongs to, and the delivery id, which matches the row in your dashboard's delivery log.",
          ],
          code: {
            language: "json",
            code: `{
  "id": "evt_7f3a9c21",
  "type": "payment.succeeded",
  "created_at": "2026-10-13T09:30:45Z",
  "meta": {
    "webhook_version": "v1",
    "environment": "test",
    "delivery_id": "whd_4b81c0de"
  },
  "data": {
    "id": "pay_9x41k2",
    "reference": "order_1042",
    "amount": 20000,
    "currency": "NGN",
    "status": "successful",
    "provider": "paystack"
  }
}`,
          },
          note: "The same payment delivered through a different provider produces the same envelope with a different meta.delivery_id. Nothing about the shape depends on which provider it came from.",
        },
        {
          id: "verifying-signatures",
          title: "Verifying signatures",
          body: [
            "Every outbound delivery is signed with your endpoint's whsec_… secret in the X-PayConnect-Signature header. Verify before trusting the payload.",
          ],
          code: {
            language: "ts",
            code: `import crypto from "crypto";

export function verifyWebhook(rawBody: string, signature: string, secret: string) {
  const expected = crypto
    .createHmac("sha256", secret)
    .update(rawBody)
    .digest("hex");
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}`,
          },
          note: "Verify against the RAW request body, not a parsed-and-re-serialized object.",
        },
        {
          id: "retries",
          title: "Retries",
          body: [
            "A non-2xx response is retried with exponential backoff: 30s, 1m, 2m, 4m, 8m, then capped at 1h, dead-lettered after 6 attempts. Every attempt appears in the delivery log with its response code, and you can replay one by hand.",
            "Inbound events are deduplicated on the provider's event id, so a provider retry never applies the same event twice.",
          ],
        },
        {
          id: "delivery-success",
          title: "Delivery success",
          body: [
            "The Webhooks page reports one figure for your default endpoint: the share of deliveries that reached you in the last 7 days. It is computed from the delivery log rather than estimated, and the caption names the attempts behind it — 6 of 8 deliveries · last 7 days — so you can reconcile the percentage with the rows.",
            "Only deliveries that concluded count. One still pending or being retried is in flight, not a failure, so a slow receiver does not dent the figure while it is still trying. When nothing has concluded in the window the dashboard shows an em dash rather than a rate with nothing behind it.",
          ],
          note: "A delivery counts as successful when your endpoint answers 2xx. Anything else — a 4xx, a 5xx, a timeout, a refused connection — counts as a failure once its retries are exhausted.",
        },
      ]}
    />
  );
}
