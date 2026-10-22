"use client";

import Link from "next/link";
import { ArrowUpRight, Sparkles } from "lucide-react";
import { Badge, Skeleton } from "@/components/ui";
import { useAsync } from "@/hooks/use-async";
import { getBillingPlan } from "@/lib/api";
import { paymentsPhrase, type SubscriptionPlan } from "@/lib/plans";
import { formatDate } from "@/lib/utils";

/**
 * What the merchant is on, where they can always see it.
 *
 * The limits are the sales engine, so they have to be legible before they are
 * hit — a merchant who only discovers their plan when something is refused has
 * been ambushed, not sold to. Every number here comes from the API's own plan
 * response, so it can never describe a plan we do not actually enforce.
 */
function describe(view: SubscriptionPlan): { title: string; detail: string; cta: string } {
  const planName = view.plan.name;
  const renews = view.current_period_end ? formatDate(view.current_period_end) : "";

  switch (view.status) {
    case "trialing":
      return {
        title: `Pro trial`,
        detail: view.trial_ends_at ? `Ends ${formatDate(view.trial_ends_at)}` : "Trial active",
        cta: "Manage",
      };
    case "active":
      return {
        title: planName,
        detail: view.cancel_at_period_end ? `Cancels ${renews}` : `Renews ${renews}`,
        cta: "Manage",
      };
    case "past_due":
      return { title: planName, detail: "Payment pending", cta: "Fix billing" };
    case "pending_payment":
      return { title: planName, detail: "Awaiting payment", cta: "Review" };
    case "cancelled":
      return { title: planName, detail: "Cancelled", cta: "Manage" };
    default:
      return {
        title: `${planName} plan`,
        detail: paymentsPhrase(view.plan.entitlements.payments_per_month),
        cta: "Upgrade",
      };
  }
}

export function PlanBadge({ className, collapsed = false }: { className?: string; collapsed?: boolean }) {
  const { data, loading, error } = useAsync(getBillingPlan, [], ["billing", "plan"]);

  if (loading) {
    return (
      <div className={className}>
        <Skeleton className={collapsed ? "size-10 rounded-full" : "h-9 w-full"} />
      </div>
    );
  }

  if (error || !data) {
    // Collapsed to an icon rail there is no room to explain a failure; leave
    // the slot empty rather than show a paragraph in a 72px column.
    if (collapsed) return null;
    return (
      <div className={className}>
        <p className="text-[12px] text-slate-400">
          Plan unavailable —{" "}
          <Link href="/settings/billing" className="underline hover:text-slate-600">
            check billing
          </Link>
          .
        </p>
      </div>
    );
  }

  const { title, detail, cta } = describe(data);
  const paid = data.plan.code !== "free";
  // The trial is Pro: showing "Free" next to it would understate what the
  // merchant currently has.
  const label = data.status === "trialing" ? "Pro" : data.plan.name;

  if (collapsed) {
    return (
      <Link
        href="/settings/billing"
        title={`${title} — ${detail}`}
        aria-label={`${title} plan. ${detail}. ${cta}`}
        className={
          className ??
          `flex size-10 items-center justify-center rounded-full border bg-white transition-colors hover:border-brand-300 ${
            paid ? "border-brand-200 text-brand-700" : "border-slate-200 text-slate-500"
          }`
        }
      >
        <Sparkles className="size-4" />
      </Link>
    );
  }

  return (
    <Link
      href="/settings/billing"
      className={
        className ??
        "flex items-start gap-2.5 rounded-xl border border-slate-200 bg-white p-3 transition-colors hover:border-brand-300"
      }
    >
      <span
        className={`mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full ${
          paid ? "bg-brand-100 text-brand-700" : "bg-slate-100 text-slate-500"
        }`}
      >
        <Sparkles className="size-3.5" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5">
          <span className="truncate text-[13px] font-semibold text-slate-900">{title}</span>
          <Badge variant={paid ? "brand" : "neutral"} className="px-1.5 py-0 text-[10.5px]">
            {label}
          </Badge>
        </span>
        <span className="mt-0.5 block truncate text-[11.5px] text-slate-400">{detail}</span>
      </span>
      <span className="mt-0.5 flex shrink-0 items-center gap-0.5 text-[11.5px] font-semibold text-brand-600">
        {cta}
        <ArrowUpRight className="size-3" />
      </span>
    </Link>
  );
}
