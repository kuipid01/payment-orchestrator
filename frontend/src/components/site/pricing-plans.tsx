"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowRight, Check, Minus } from "lucide-react";
import { Badge, Button, Card, Tabs } from "@/components/ui";
import { cn } from "@/lib/utils";
import {
  INTERVAL_LABEL,
  INTERVALS,
  formatLimit,
  formatPrice,
  intervalNote,
  intervalPrice,
  paymentsAllowance,
  type BillingInterval,
  type PlanCatalog,
  type PlanOffer,
  type PriceBand,
} from "@/lib/plans";

/** The full comparison, in the order the ladder is explained. */
function comparisonRows(plan: PlanOffer, currency: string) {
  const e = plan.entitlements;
  return [
    { label: "Applications", value: formatLimit(e.applications) },
    {
      label: "Extra application",
      value: e.extra_application_minor > 0 ? `${formatPrice(e.extra_application_minor, currency)} each` : "—",
    },
    { label: "Developer seats", value: formatLimit(e.seats) },
    {
      label: "Extra seat",
      value: e.extra_seat_minor > 0 ? `${formatPrice(e.extra_seat_minor, currency)} each` : "—",
    },
    { label: "Environments", value: "Test + live" },
    { label: "Payments", value: paymentsAllowance(e.payments_per_month) },
    {
      label: "Routed volume included",
      value:
        e.routed_volume_allowance_minor > 0
          ? `${formatPrice(e.routed_volume_allowance_minor, currency)} / month`
          : "Never billed",
    },
    {
      label: "Above that",
      value:
        e.routed_volume_cap_minor > 0
          ? `0.05%, capped at ${formatPrice(e.routed_volume_cap_minor, currency)}`
          : "—",
    },
    { label: "Provider connections", value: e.provider_connections || "—" },
    { label: "Routing rules + failover", value: e.routing_rules },
    { label: "Webhook endpoints", value: formatLimit(e.webhook_endpoints) },
    { label: "Webhook replay + retry tuning", value: e.webhook_replay },
    { label: "Paid traffic dispatched first", value: e.priority_dispatch },
    { label: "Log retention", value: e.log_retention_days > 0 ? `${formatLimit(e.log_retention_days)} days` : "Custom" },
    { label: "Audit log export", value: e.audit_export },
    { label: "Support", value: e.support },
    {
      label: "Uptime target",
      value: e.sla_credits ? `${e.uptime_target} + service credits` : e.uptime_target,
    },
  ];
}

/** The handful of bullets a card can carry before it stops being readable. */
function highlights(plan: PlanOffer, currency: string): string[] {
  const e = plan.entitlements;
  const bullets: string[] = [
    `${formatLimit(e.applications)} application${e.applications === 1 ? "" : "s"}${
      e.extra_application_minor > 0 ? `, then ${formatPrice(e.extra_application_minor, currency)} each` : ""
    }`,
    `${formatLimit(e.seats)} developer seat${e.seats === 1 ? "" : "s"}${
      e.extra_seat_minor > 0 ? `, then ${formatPrice(e.extra_seat_minor, currency)} each` : ""
    }`,
    `Test + live, connecting your own ${plan.negotiated ? "provider accounts" : "provider keys"}`,
  ];
  if (e.payments_per_month > 0) {
    bullets.push(`${e.payments_per_month} payments a month`);
  } else {
    bullets.push("Unlimited payments");
  }
  if (e.routed_volume_allowance_minor > 0) {
    bullets.push(`${formatPrice(e.routed_volume_allowance_minor, currency)} of routed volume included`);
  } else if (plan.negotiated) {
    bullets.push("Flat negotiated volume rate");
  } else {
    bullets.push("No volume fee, ever");
  }
  if (e.routing_rules) bullets.push("Routing rules and automatic failover");
  if (e.webhook_replay) bullets.push("Webhook replay and retry tuning");
  bullets.push(`${formatLimit(e.webhook_endpoints)} webhook endpoint${e.webhook_endpoints === 1 ? "" : "s"}`);
  bullets.push(`${e.log_retention_days > 0 ? `${e.log_retention_days}-day` : "Custom"} log retention`);
  if (e.audit_export) bullets.push("Audit-log export");
  bullets.push(e.support);
  return bullets;
}

function priceCell(plan: PlanOffer, interval: BillingInterval): { headline: string; suffix: string } {
  if (plan.negotiated) {
    return { headline: `from ${formatPrice(plan.from_minor ?? 0, plan.currency)}`, suffix: "/ month" };
  }
  if (plan.monthly_minor === 0) {
    return { headline: formatPrice(0, plan.currency), suffix: "forever" };
  }
  const suffix = interval === "monthly" ? "/ month" : interval === "quarterly" ? "/ quarter" : "/ year";
  return { headline: formatPrice(intervalPrice(plan, interval), plan.currency), suffix };
}

function ComparisonValue({ value }: { value: string | boolean }) {
  if (typeof value === "boolean") {
    return value ? (
      <Check className="mx-auto size-4 text-emerald-500" />
    ) : (
      <Minus className="mx-auto size-4 text-slate-300" />
    );
  }
  return <span className="text-slate-600">{value}</span>;
}

export function PricingPlans({
  catalog,
  live,
  initialBand,
}: {
  catalog: PlanCatalog;
  live: boolean;
  initialBand: PriceBand;
}) {
  const band = initialBand;
  const [interval, setInterval] = React.useState<BillingInterval>("monthly");

  const current = catalog.bands.find((entry) => entry.band === band)
    ?? catalog.bands.find((entry) => entry.currency === "USD")
    ?? catalog.bands[0];
  const currency = current?.currency ?? "USD";
  const plans = current?.plans ?? [];
  const regionLabel = current?.band === "A"
    ? "Nigeria"
    : current?.band === "B"
      ? "Africa"
      : current?.band === "C"
        ? "Emerging markets"
        : "Global";

  return (
    <section className="mx-auto max-w-6xl px-5 py-12 lg:px-8">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-3">
          <div>
            <p className="text-[12px] font-semibold uppercase tracking-wider text-slate-400">Price region</p>
            <p className="mt-1 text-sm font-semibold text-slate-800">{regionLabel} ({currency === "NGN" ? "₦" : "$"})</p>
          </div>
          <span className="rounded-full border border-brand-100 bg-brand-50 px-2.5 py-1 text-[10px] font-medium text-brand-800">
            Based on your location
          </span>
        </div>
        <div>
          <p className="text-[12px] font-semibold uppercase tracking-wider text-slate-400">Billing</p>
          <Tabs
            className="mt-2"
            value={interval}
            onValueChange={(value) => setInterval(value as BillingInterval)}
            tabs={INTERVALS.map((value) => ({ value, label: INTERVAL_LABEL[value] }))}
          />
        </div>
      </div>

      <p className="mt-3 text-[12.5px] text-slate-500">
        We estimate your price region from your connection location, which may be affected by a VPN. If we can&apos;t
        determine it, prices default to USD. Your final billing region is based on your organization&apos;s country.
        {` ${intervalNote(interval)}`}
      </p>

      <div className="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {plans.map((plan) => {
          const price = priceCell(plan, interval);
          return (
            <Card
              key={plan.code}
              className={cn(
                "relative flex flex-col p-6",
                plan.highlighted ? "border-brand-300 ring-1 ring-brand-200 shadow-[var(--shadow-lift)]" : ""
              )}
            >
              {plan.highlighted ? (
                <Badge variant="brand" className="absolute -top-2.5 left-6">
                  Most popular
                </Badge>
              ) : null}
              <p className="text-[15px] font-semibold text-slate-900">{plan.name}</p>
              <p className="mt-1 text-[12.5px] text-slate-400">{plan.tagline}</p>
              <div className="mt-5 flex items-baseline gap-1.5">
                <span className="text-3xl font-semibold tracking-tight text-slate-900">{price.headline}</span>
                <span className="text-[12.5px] text-slate-400">{price.suffix}</span>
              </div>
              {plan.negotiated ? (
                <p className="mt-1 text-[12px] text-slate-400">Negotiated with a flat volume rate.</p>
              ) : interval !== "monthly" && plan.monthly_minor > 0 ? (
                <p className="mt-1 text-[12px] text-slate-400">
                  {formatPrice(Math.round(intervalPrice(plan, interval) / (interval === "quarterly" ? 3 : 12)), currency)}{" "}
                  a month, billed {interval}.
                </p>
              ) : null}

              <ul className="mt-6 flex-1 space-y-2.5">
                {highlights(plan, currency).map((feature) => (
                  <li key={feature} className="flex items-start gap-2.5 text-[13px] text-slate-600">
                    <Check className="mt-0.5 size-3.5 shrink-0 text-emerald-500" />
                    {feature}
                  </li>
                ))}
              </ul>

              <Link
                href={plan.negotiated ? "/register?plan=enterprise" : `/register?plan=${plan.code}&interval=${interval}`}
                className="mt-7"
              >
                <Button variant={plan.highlighted ? "primary" : "secondary"} className="w-full">
                  {plan.code === "free" ? "Start for free" : plan.negotiated ? "Talk to us" : `Choose ${plan.name}`}
                  <ArrowRight className="size-4" />
                </Button>
              </Link>
            </Card>
          );
        })}
      </div>

      <div className="mt-6 flex items-center gap-2 text-[12px] text-slate-400">
        {live ? null : <Badge variant="warning">Prices from the last generated catalog — the API was unreachable</Badge>}
        <span>
          All prices are per organization. Nothing here is charged from a payment: PayConnect never touches your
          settlement.
        </span>
      </div>

      <div className="mt-12 overflow-x-auto">
        <table className="w-full min-w-[720px] border-separate border-spacing-0 text-left">
          <thead>
            <tr>
              <th className="sticky left-0 bg-white py-3 pr-4 text-[12px] font-semibold uppercase tracking-wider text-slate-400">
                Compare
              </th>
              {plans.map((plan) => (
                <th key={plan.code} className="px-4 py-3 text-[13.5px] font-semibold text-slate-900">
                  {plan.name}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {plans[0]
              ? comparisonRows(plans[0], currency).map((row, rowIndex) => (
                  <tr key={row.label} className={rowIndex % 2 === 0 ? "bg-[#F8FAFC]" : ""}>
                    <th
                      scope="row"
                      className={cn(
                        "sticky left-0 py-2.5 pr-4 text-[13px] font-normal text-slate-500",
                        rowIndex % 2 === 0 ? "bg-[#F8FAFC]" : "bg-white"
                      )}
                    >
                      {row.label}
                    </th>
                    {plans.map((plan) => {
                      const value = comparisonRows(plan, currency)[rowIndex]?.value ?? "—";
                      return (
                        <td key={plan.code} className="px-4 py-2.5 text-[13px]">
                          <ComparisonValue value={value} />
                        </td>
                      );
                    })}
                  </tr>
                ))
              : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}
