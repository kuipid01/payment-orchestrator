import Link from "next/link";
import { headers } from "next/headers";
import { ArrowRight, Check, Lock, ScrollText, ShieldCheck, Webhook, Zap } from "lucide-react";
import { Badge } from "@/components/ui";
import { PricingPlans } from "@/components/site/pricing-plans";
import {
  fetchPlanCatalog,
  formatPrice,
  paymentsPhrase,
  volumeFee,
  type PlanCatalog,
  type PlanOffer,
  type PriceBand,
} from "@/lib/plans";

export const metadata = {
  title: "Pricing",
  description:
    "One integration for every provider you already have. Bring your own Paystack, Flutterwave or Moniepoint keys — PayConnect never holds your money or takes a cut of a payment.",
};

/** The Lite ladder, computed from the catalog rather than typed out again. */
function ladderRows(plan: PlanOffer) {
  const currency = plan.currency;
  return [5_000_000, 10_000_000, 20_000_000, 50_000_000].map((naira) => {
    const routed = naira * 100;
    const fee = volumeFee(routed, plan);
    return {
      routed: formatPrice(routed, currency),
      fee: fee === 0 ? "Included" : formatPrice(fee, currency),
      total: formatPrice(plan.monthly_minor + fee, currency),
    };
  });
}

export default async function PricingPage() {
  const requestHeaders = await headers();
  const { catalog, live } = await fetchPlanCatalog();

  const countryHeaders = [
    "x-vercel-ip-country",
    "cf-ipcountry",
    "cloudfront-viewer-country",
    "x-country-code",
  ] as const;

  const countryHeader = countryHeaders
    .map((name) => ({
      name,
      value: requestHeaders.get(name)?.trim(),
    }))
    .find(
      (header) =>
        header.value && /^[A-Za-z]{2}$/.test(header.value),
    );

  const localCountry =
    process.env.NODE_ENV === "development"
      ? process.env.DEV_COUNTRY?.trim().toUpperCase()
      : undefined;

  const country = (
    localCountry && /^[A-Z]{2}$/.test(localCountry)
      ? localCountry
      : countryHeader?.value?.toUpperCase()
  )?.toUpperCase();

  console.log("Country detection:", {
    country,
    source: localCountry
      ? "development override"
      : countryHeader?.name ?? "none",
  });
  const detectedBand = country
    ? catalog.bands.find((entry) =>
      entry.countries.includes(country.toUpperCase()),
    )?.band
    : undefined;
  const initialBand: PriceBand =
    detectedBand ??
    catalog.bands.find((entry) => entry.band === "D" && entry.currency === "USD")
      ?.band ??
    catalog.bands.find((entry) => entry.currency === "USD")?.band ??
    catalog.bands[0]?.band ??
    "D";
  const bandA = catalog.bands.find((band) => band.band === "A") ?? catalog.bands[0];
  const lite = bandA?.plans.find((plan) => plan.code === "lite");
  const free = bandA?.plans.find((plan) => plan.code === "free");
  const pro = bandA?.plans.find((plan) => plan.code === "pro");
  const fee = catalog.volume_fee;

  return (
    <div>
      <section className="border-b border-slate-100 bg-[#FDF8E7]">
        <div className="mx-auto max-w-6xl px-5 py-16 lg:px-8 lg:py-20">
          <div className="max-w-3xl">
            <Badge variant="brand">Bring your own provider</Badge>
            <h1 className="mt-4 font-display text-4xl text-slate-950 sm:text-5xl">
              One integration for every provider you already have —
              <span className="text-brand-500"> and your keys never leave the vault.</span>
            </h1>
            <p className="mt-5 max-w-2xl text-[15px] leading-relaxed text-slate-600">
              PayConnect is not your PSP. You keep your Paystack, Flutterwave and Moniepoint accounts, your
              settlement and your rates — we make them one system: one API, one webhook format, one vault, one
              dashboard. That is why we can never take a cut of a payment, and why the prices below are the whole
              bill.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Link href="/register" className="inline-flex">
                <span className="inline-flex h-11 items-center gap-2 rounded-full bg-brand-500 px-6 text-[14px] font-semibold text-slate-950 hover:bg-brand-400">
                  Start free
                  <ArrowRight className="size-4" />
                </span>
              </Link>
              <Link href="/docs/quickstart" className="inline-flex">
                <span className="inline-flex h-11 items-center rounded-full border border-slate-300 px-6 text-[14px] font-semibold text-slate-800 hover:bg-white">
                  Read the quickstart
                </span>
              </Link>
              {lite && free ? (
                <p className="text-[13px] text-slate-500">
                  Free includes live with {paymentsPhrase(free.entitlements.payments_per_month)}; Lite is{" "}
                  {formatPrice(lite.monthly_minor, lite.currency)} a month, less than the hour a month you already
                  lose to three integrations.
                </p>
              ) : null}
            </div>
          </div>
        </div>
      </section>

      <PricingPlans catalog={catalog} live={live} initialBand={initialBand} />

      {/* The one fee above the plan, stated in full or not at all. */}
      <section className="border-y border-slate-100 bg-[#F8FAFC]">
        <div className="mx-auto max-w-6xl px-5 py-16 lg:px-8">
          <div className="grid gap-10 lg:grid-cols-[1.1fr_1fr]">
            <div>
              <Badge variant="neutral">The only fee above your plan</Badge>
              <h2 className="mt-4 font-display text-3xl text-slate-950">
                {fee.rate_label} of the volume we routed — measured, then invoiced.
              </h2>
              <p className="mt-4 text-[14.5px] leading-relaxed text-slate-600">
                On a ₦1,000 payment that is <strong>50 kobo</strong>. Not 0.3% — that would be ₦3, a fifth of what
                Paystack itself charges on the same ticket, and the moment a fee like that appears the promise above
                stops being true.
              </p>
              <ul className="mt-6 space-y-3">
                {[
                  {
                    icon: <ShieldCheck className="mt-0.5 size-4 shrink-0 text-emerald-600" />,
                    title: "Only volume we routed",
                    body: fee.scope,
                  },
                  {
                    icon: <ScrollText className="mt-0.5 size-4 shrink-0 text-emerald-600" />,
                    title: "Measured, then invoiced",
                    body: fee.billed,
                  },
                  {
                    icon: <Lock className="mt-0.5 size-4 shrink-0 text-emerald-600" />,
                    title: fee.promise,
                    body: `The allowance is ${fee.allowance_multiple} × your plan price and the fee caps at ${fee.cap_multiple} ×, so a bill is always a multiple of a price you already know. Above the cap we do not invent a bigger number — we quote you Enterprise.`,
                  },
                ].map((item) => (
                  <li key={item.title} className="flex gap-3">
                    {item.icon}
                    <div>
                      <p className="text-[14px] font-semibold text-slate-900">{item.title}</p>
                      <p className="mt-0.5 text-[13.5px] leading-relaxed text-slate-600">{item.body}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </div>

            {lite ? (
              <div className="rounded-[var(--radius-card)] border border-slate-200 bg-white p-6">
                <div className="flex items-baseline justify-between">
                  <p className="text-[14px] font-semibold text-slate-900">Lite at volume</p>
                  <p className="text-[12.5px] text-slate-400">
                    {formatPrice(lite.monthly_minor, lite.currency)} base + {fee.rate_label}
                  </p>
                </div>
                <table className="mt-4 w-full text-left text-[13px]">
                  <thead>
                    <tr className="text-[11.5px] uppercase tracking-wider text-slate-400">
                      <th className="pb-2 font-semibold">Routed / month</th>
                      <th className="pb-2 font-semibold">Volume fee</th>
                      <th className="pb-2 text-right font-semibold">Total bill</th>
                    </tr>
                  </thead>
                  <tbody className="text-slate-600">
                    {ladderRows(lite).map((row) => (
                      <tr key={row.routed} className="border-t border-slate-100">
                        <td className="py-2 font-mono text-[12.5px]">{row.routed}</td>
                        <td className="py-2 font-mono text-[12.5px]">{row.fee}</td>
                        <td className="py-2 text-right font-mono text-[12.5px] text-slate-900">{row.total}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <p className="mt-4 text-[12.5px] leading-relaxed text-slate-500">
                  Past ₦10m the volume fee is larger than the subscription — that is the design, not an accident, and
                  it is why the base price can be this low. Everything below{" "}
                  {formatPrice(lite.entitlements.routed_volume_allowance_minor, lite.currency)} is included, and the
                  fee never exceeds {formatPrice(lite.entitlements.routed_volume_cap_minor, lite.currency)}.
                </p>
              </div>
            ) : null}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-5 py-16 lg:px-8">
        <div className="grid gap-10 lg:grid-cols-2">
          <div>
            <h2 className="font-display text-3xl text-slate-950">What a plan actually buys</h2>
            <p className="mt-4 text-[14.5px] leading-relaxed text-slate-600">
              Every perk below already exists in the product — none of it is a promise to build. The visible wall is
              the point: the second application, the sixth seat, payment 101 of the month. You should hit a limit you
              can feel and understand what lifting it costs.
            </p>
            <p className="mt-3 text-[14.5px] leading-relaxed text-slate-600">
              And when you hit it, nothing on the money path stops. Your customer&apos;s checkout does not care what
              plan you are on — it never has, and it never will.
            </p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            {[
              { icon: <Lock className="size-4 text-brand-500" />, text: "Encrypted credential vault — masked, rotatable, audited" },
              { icon: <Webhook className="size-4 text-brand-500" />, text: "One webhook envelope, one signature, retries and replay" },
              { icon: <Zap className="size-4 text-brand-500" />, text: "Failover to the next connected provider that supports the currency" },
              { icon: <ShieldCheck className="size-4 text-brand-500" />, text: "API keys scoped per application and per developer" },
              { icon: <ScrollText className="size-4 text-brand-500" />, text: "Idempotency keys on every write" },
              { icon: <Check className="size-4 text-brand-500" />, text: "Test and live environments on every plan, Free included" },
            ].map((item) => (
              <div key={item.text} className="rounded-[var(--radius-card)] border border-slate-200 bg-white p-4">
                <span className="flex size-8 items-center justify-center rounded-full bg-brand-50">{item.icon}</span>
                <p className="mt-2.5 text-[13px] text-slate-600">{item.text}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-slate-100 bg-[#FDF8E7]">
        <div className="mx-auto max-w-6xl px-5 py-14 lg:px-8">
          <h2 className="font-display text-2xl text-slate-950">Questions worth answering before you pay</h2>
          <div className="mt-8 grid gap-6 md:grid-cols-2">
            {faqs(catalog, free, lite, pro).map((item) => (
              <div key={item.q}>
                <p className="text-[14px] font-semibold text-slate-900">{item.q}</p>
                <p className="mt-1.5 text-[13.5px] leading-relaxed text-slate-600">{item.a}</p>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}

/** The FAQ, with every number read from the catalog. */
function faqs(
  catalog: PlanCatalog,
  free: PlanOffer | undefined,
  lite: PlanOffer | undefined,
  pro: PlanOffer | undefined,
): { q: string; a: string }[] {
  const trial = `${catalog.trial_days} days`;
  return [
    {
      q: "Is the trial card-free?",
      a: `Yes — ${trial} of Pro, no card, once per organization. When it ends you land on Lite rather than Free, so your keys, routing and webhooks keep working while you decide.`,
    },
    {
      q: "Does live traffic cost extra on Free?",
      a: `No. Free includes live with a real ceiling: ${free ? paymentsPhrase(free.entitlements.payments_per_month) : "100 payments a month"}${free?.entitlements.routed_volume_ceiling_minor
          ? `, or ${formatPrice(free.entitlements.routed_volume_ceiling_minor, free.currency)} of routed volume, whichever comes first`
          : ""
        }. Test-only free tiers teach you nothing. If you outgrow it, upgrade — nothing you already built changes.`,
    },
    {
      q: "What happens if a payment fails because of my plan?",
      a: "Nothing can. Payments, refunds and webhook delivery are never throttled or blocked by billing. If a subscription lapses we degrade account management — after a grace period you drop to Free, keeping your oldest application live and your data for 90 days.",
    },
    {
      q: "Can I cancel?",
      a: "Any time. Your access continues to the end of the period you paid for, and monthly has no minimum term. Quarterly is the same price as three months of monthly — you are not paying for a discount, you are paying three times a year instead of twelve.",
    },
    {
      q: "Why is annual eleven months rather than ten?",
      a: "Because at this price ten months gives away more than it saves. Eleven months is one month free and keeps the arithmetic honest on both sides.",
    },
    {
      q: "Why is Free one application per organization?",
      a: `Because the ladder is the product. Extra applications are ${lite ? formatPrice(lite.entitlements.extra_application_minor, lite.currency) : "₦1,250"} on Lite and ${pro ? formatPrice(pro.entitlements.extra_application_minor, pro.currency) : "₦900"} on Pro, and a second free application would make the first paid one a riddle.`,
    },
    {
      q: "Do you ever touch my money?",
      a: "No. Your keys stay in the vault and your settlement happens in an account we have no access to. The volume fee is measured from our own records and itemised on a monthly invoice — it is never split off a payment and never taken from a customer.",
    },
    {
      q: "What is the difference between Lite and Pro?",
      a: `Pro is where the company is paid for the cheap price: ${pro ? formatPrice(pro.monthly_minor, pro.currency) : "₦9,500"} a month, three times the applications, five times the seats, multiple provider accounts, 90-day retention, audit export and priority support. Lite ${lite ? `(${formatPrice(lite.monthly_minor, lite.currency)})` : ""} is the honest starting point for one real product.`,
    },
  ];
}
