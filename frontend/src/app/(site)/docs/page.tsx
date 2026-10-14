import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "Introduction" };

export default function DocsIntroPage() {
  return (
    <DocsPage
      title="Introduction"
      lede="PayConnect is a secure payment infrastructure layer between merchants and developers. Connect payment providers once, then let your team build against one clean API — without anyone touching a provider secret."
      sections={[
        {
          id: "what-is-payconnect",
          title: "What is PayConnect?",
          body: [
            "Merchants connect Paystack, Flutterwave or Stripe in the dashboard. Credentials are encrypted in the vault and are never exposed again — not to developers, not in logs, not in API responses.",
            "Developers receive a platform API key (vpay_live_… or vpay_test_…) scoped to a specific application. Payments, webhooks and logs all flow through PayConnect's unified interface.",
          ],
          code: {
            language: "ts",
            code: `const payment = await vpay.payments.create({
  amount: 20000,
  currency: "NGN",
});`,
          },
        },
        {
          id: "core-concepts",
          title: "Core concepts",
          list: [
            "Provider connection — a stored, encrypted credential for one payment provider (per environment).",
            "Application — a product surface that gets its own API keys, webhooks and logs.",
            "API key — the developer-facing credential. Scoped, maskable, rotatable, revocable.",
            "Vault — the encrypted store holding provider secrets. Secrets go in; nothing but payment results comes out.",
          ],
        },
        {
          id: "environments",
          title: "Environments",
          body: [
            "Every resource lives in either the test or live environment. Test keys (vpay_test_…) hit simulated providers — perfect for development. Live keys (vpay_live_…) move real money.",
          ],
          note: "Never mix environments. A test key cannot read live data, and a live key cannot create test payments.",
        },
      ]}
    />
  );
}
