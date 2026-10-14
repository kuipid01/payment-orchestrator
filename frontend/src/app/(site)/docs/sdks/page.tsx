import { DocsPage } from "@/components/docs/docs-page";

export const metadata = { title: "SDKs" };

export default function SdksDocsPage() {
  return (
    <DocsPage
      title="SDKs & libraries"
      lede="Official SDKs wrap the REST API with typed models, retries and idempotency handling."
      sections={[
        {
          id: "javascript-typescript",
          title: "JavaScript / TypeScript",
          code: {
            language: "bash",
            code: `npm install @payconnect/node`,
          },
          body: [
            "Node 18+ and all major runtimes (Next.js, Remix, Bun, Deno) are supported.",
          ],
        },
        {
          id: "usage",
          title: "Usage",
          code: {
            language: "ts",
            code: `import { PayConnect } from "@payconnect/node";

const vpay = new PayConnect(process.env.PAYCONNECT_API_KEY!);

const payment = await vpay.payments.create({
  amount: 20000,
  currency: "NGN",
  customer: { email: "customer@example.com" },
});`,
          },
        },
        {
          id: "coming-soon",
          title: "Community libraries",
          list: [
            "@payconnect/python — Python 3.10+",
            "@payconnect/go — Go 1.22+ (pairs well with the PayConnect backend)",
            "@payconnect/php — Laravel integration",
          ],
          note: "Community libraries are unofficial — verify webhook signatures yourself when using them.",
        },
      ]}
    />
  );
}
