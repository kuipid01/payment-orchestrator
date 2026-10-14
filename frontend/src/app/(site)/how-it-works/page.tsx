import Link from "next/link";
import { ArrowRight, Check, KeyRound, Lock, Mail, ShieldCheck } from "lucide-react";
import { Button, Card, CodeBlock } from "@/components/ui";

const STEPS = [
  {
    n: "01",
    title: "Connect your payment provider",
    body: "Add your Paystack, Flutterwave or Stripe secret key once. It is encrypted immediately and never displayed again — only its last 4 characters are ever rendered.",
    demo: (
      <div className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3">
        <div className="flex items-center gap-3">
          <span className="flex size-8 items-center justify-center rounded-lg bg-sky-100 text-[10px] font-bold text-sky-700">PS</span>
          <div>
            <p className="text-[13px] font-semibold text-slate-900">Paystack</p>
            <p className="font-mono text-[11px] text-slate-400">sk_live_••••••••••••9d21</p>
          </div>
        </div>
        <span className="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700">
          <Check className="size-3" /> Connected
        </span>
      </div>
    ),
  },
  {
    n: "02",
    title: "Invite your developer",
    body: "Grant access to specific applications with scoped permissions. Developers see payments and logs — never provider credentials.",
    demo: (
      <div className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3">
        <div className="flex items-center gap-3">
          <span className="flex size-8 items-center justify-center rounded-full bg-brand-100 text-[11px] font-semibold text-brand-700">JD</span>
          <div>
            <p className="text-[13px] font-semibold text-slate-900">John Developer</p>
            <p className="text-[11px] text-slate-400">developer@example.com</p>
          </div>
        </div>
        <span className="rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-[11px] text-slate-500">4 permissions</span>
      </div>
    ),
  },
  {
    n: "03",
    title: "Build without exposing secrets",
    body: "Your developer integrates with a PayConnect API key. The platform forwards provider requests from the vault — the secret never reaches their code, logs or machine.",
    demo: (
      <div className="rounded-xl border border-slate-200 bg-white px-4 py-3">
        <p className="font-mono text-[12.5px] text-slate-700">vpay_live_••••••••••••9d21</p>
        <p className="mt-1 text-[11px] text-slate-400">Platform API key — scoped to “My Ecommerce Store”</p>
      </div>
    ),
  },
];

const ARCH_CODE = `# Merchant connects once (dashboard only)
POST /v1/providers/paystack/connect
Authorization: Bearer <merchant-session>
{ "secretKey": "sk_live_••••••" }        → stored encrypted

# Developer integrates with the platform key
const payment = await vpay.payments.create({
  amount: 20000,
  currency: "NGN"
});`;

export default function HowItWorksPage() {
  return (
    <div>
      <section className="border-b border-slate-100 bg-[#FDF8E7]">
        <div className="mx-auto max-w-6xl px-5 py-16 text-center lg:px-8 lg:py-20">
          <span className="inline-flex items-center gap-1.5 rounded-full border border-brand-200 bg-white px-3 py-1 text-[12.5px] font-semibold text-brand-800">
            <ShieldCheck className="size-3.5" /> Architecture
          </span>
          <h1 className="mx-auto mt-5 max-w-2xl font-display text-4xl text-slate-950 sm:text-5xl">
            How PayConnect <span className="text-brand-500">works</span>
          </h1>
          <p className="mx-auto mt-4 max-w-2xl text-[15px] leading-relaxed text-slate-600">
            A secure vault between your merchant account and your developers. Providers process payments;
            PayConnect protects the credentials that access them.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-5 py-16 lg:px-8">
        <div className="grid gap-6 lg:grid-cols-3">
          {STEPS.map((s) => (
            <Card key={s.n}>
              <div className="p-6">
                <span className="font-mono text-[12px] font-semibold text-brand-500">{s.n}</span>
                <h2 className="mt-3 text-[16px] font-semibold text-slate-900">{s.title}</h2>
                <p className="mt-2 text-[13.5px] leading-relaxed text-slate-500">{s.body}</p>
                <div className="mt-5">{s.demo}</div>
              </div>
            </Card>
          ))}
        </div>
      </section>

      <section className="border-y border-slate-100 bg-[#F8FAFC]">
        <div className="mx-auto grid max-w-6xl items-center gap-12 px-5 py-16 lg:grid-cols-2 lg:px-8">
          <div>
            <p className="text-[12px] font-semibold uppercase tracking-wider text-brand-500">Architecture</p>
            <h2 className="mt-2 text-2xl font-semibold tracking-tight text-slate-900">
              The secret never crosses the boundary.
            </h2>
            <div className="mt-8 space-y-5">
              {[
                { icon: <Lock className="size-4" />, t: "Encrypted vault", d: "Provider credentials are encrypted with per-organization keys and never leave the vault service." },
                { icon: <KeyRound className="size-4" />, t: "Scoped platform keys", d: "Developer keys (vpay_live_*, vpay_test_*) carry per-application permissions, not provider access." },
                { icon: <Mail className="size-4" />, t: "Audited invitations", d: "Every developer invitation, permission change and key rotation is written to the audit log." },
              ].map((f) => (
                <div key={f.t} className="flex gap-3.5">
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-brand-100 bg-white text-brand-500">
                    {f.icon}
                  </span>
                  <div>
                    <h3 className="text-[14.5px] font-semibold text-slate-900">{f.t}</h3>
                    <p className="mt-0.5 text-[13.5px] leading-relaxed text-slate-500">{f.d}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <CodeBlock code={ARCH_CODE} language="bash" />
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-5 py-16 text-center lg:px-8">
        <h2 className="text-2xl font-semibold tracking-tight text-slate-900">Ready to wire it up?</h2>
        <p className="mx-auto mt-3 max-w-lg text-[15px] text-slate-500">
          Connect a provider in the test environment and generate your first API key in under five minutes.
        </p>
        <div className="mt-8 flex justify-center gap-3">
          <Link href="/register">
            <Button size="lg">
              Get Started <ArrowRight className="size-4" />
            </Button>
          </Link>
          <Link href="/docs/quickstart">
            <Button variant="secondary" size="lg">
              Read the quickstart
            </Button>
          </Link>
        </div>
      </section>
    </div>
  );
}
