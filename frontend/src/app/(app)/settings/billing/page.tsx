"use client";

import * as React from "react";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, ArrowRight, Check, Info, RefreshCw } from "lucide-react";
import { Badge, Button, Card, EmptyState, Separator, Skeleton, Tabs } from "@/components/ui";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import {
  cancelSubscription,
  choosePlan,
  getBillingPlan,
  getBillingUsage,
  listSubscriptionEvents,
  resumeSubscription,
  startPlanTrial,
} from "@/lib/api";
import {
  INTERVAL_LABEL,
  INTERVALS,
  OFFLINE_CATALOG,
  STATUS_LABEL,
  countWithNoun,
  formatPrice,
  intervalPrice,
  paymentsPhrase,
  type BillingInterval,
  type BillingUsage,
  type PlanOffer,
  type SubscriptionPlan,
} from "@/lib/plans";
import { cn, formatDate } from "@/lib/utils";

/* ---------------------------------- pieces ---------------------------------- */

const STATUS_VARIANT: Record<SubscriptionPlan["status"], "neutral" | "brand" | "success" | "warning" | "danger"> = {
  free: "neutral",
  trialing: "brand",
  active: "success",
  past_due: "warning",
  pending_payment: "warning",
  cancelled: "neutral",
};

/**
 * A meter against a plan limit. Unlimited dimensions show the count without a
 * bar, because a bar to nowhere is worse than no bar.
 */
function Meter({
  label,
  used,
  limit,
  hint,
}: {
  label: string;
  used: number;
  limit: number;
  hint?: string;
}) {
  const unlimited = limit < 0;
  const pct = unlimited || limit === 0 ? 0 : Math.min(100, Math.round((used / limit) * 100));
  return (
    <div>
      <div className="flex items-center justify-between text-[13px]">
        <span className="text-slate-600">{label}</span>
        <span className="font-mono text-slate-500">
          {used.toLocaleString("en-US")}
          {unlimited ? "" : ` / ${limit.toLocaleString("en-US")}`}
        </span>
      </div>
      {unlimited ? (
        <p className="mt-1.5 text-[12px] text-slate-400">No ceiling on your plan.</p>
      ) : (
        <div className="mt-2 h-1.5 rounded-full bg-slate-100">
          <div
            className={cn("h-1.5 rounded-full", pct > 85 ? "bg-amber-500" : "bg-brand-500")}
            style={{ width: `${pct}%` }}
          />
        </div>
      )}
      {hint ? <p className="mt-1.5 text-[12px] text-slate-400">{hint}</p> : null}
    </div>
  );
}

/** The routed-volume meter: usage, the free allowance, and the fee cap. */
function VolumeMeter({ usage }: { usage: BillingUsage }) {
  const { used, allowance, cap, ceiling, refunded } = usage.routed_volume;
  const currency = usage.currency;
  const scale = Math.max(cap > 0 ? cap * 4 : used, used, ceiling ?? 0, 1);
  const toPct = (value: number) => `${Math.min(100, Math.round((value / scale) * 100))}%`;

  return (
    <div>
      <div className="flex items-center justify-between text-[13px]">
        <span className="text-slate-600">Routed volume this month</span>
        <span className="font-mono text-slate-500">{formatPrice(used, currency)}</span>
      </div>
      <div className="relative mt-2 h-2 rounded-full bg-slate-100">
        <div className="h-2 rounded-full bg-brand-500" style={{ width: toPct(used) }} />
        {allowance > 0 ? (
          <div
            className="absolute top-[-3px] h-3.5 w-px bg-slate-400"
            style={{ left: toPct(allowance) }}
            title={`Free allowance: ${formatPrice(allowance, currency)}`}
          />
        ) : null}
      </div>
      <div className="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-[12px] text-slate-400">
        <span>
          {allowance > 0 ? `${formatPrice(allowance, currency)} included free` : "No allowance on this plan"}
        </span>
        {ceiling ? <span>Ceiling {formatPrice(ceiling, currency)} on Free</span> : null}
        {cap > 0 ? <span>Fee capped at {formatPrice(cap, currency)}</span> : null}
        {refunded > 0 ? <span>Refunds netted off: {formatPrice(refunded, currency)}</span> : null}
      </div>
    </div>
  );
}

function feeNote(usage: BillingUsage, current: PlanOffer, currency: string): string {
  const { allowance, cap, used } = usage.routed_volume;
  const ceiling = usage.routed_volume.ceiling ?? 0;
  const fee = usage.volume_fee;

  if (fee.amount > 0) {
    const capped = fee.uncapped > fee.amount
      ? `, capped at ${formatPrice(fee.amount, currency)} (uncapped: ${formatPrice(fee.uncapped, currency)})`
      : "";
    return `0.05% of ${formatPrice(used, currency)} routed, less the ${formatPrice(allowance, currency)} allowance${capped}. The fee is measured and invoiced; it is never deducted from a settlement.`;
  }

  // A zero cap is a plan that is never invoiced a percentage at all — Free and
  // Enterprise. Quoting them an "allowance" of ₦0 would describe a fee that
  // cannot exist, so they are told what actually binds instead.
  if (cap === 0) {
    const limitation = ceiling > 0
      ? `, and it is limited to ${formatPrice(ceiling, currency)} of routed volume a month — past that we ask you to move up a plan rather than invoice you`
      : "";
    return current.negotiated
      ? `${current.name} is invoiced at its negotiated volume rate, not a percentage${limitation}. Nothing here is ever deducted from a settlement.`
      : `${current.name} never carries a volume fee at all${limitation}. Nothing here is ever deducted from a settlement.`;
  }

  return `Nothing accrued — you are inside the ${formatPrice(allowance, currency)} of routed volume included with ${current.name}. The fee is measured and invoiced; it is never deducted from a settlement.`;
}

/** One offer in the chooser, with the price for the selected cadence. */
function PlanChoice({
  plan,
  interval,
  current,
  busy,
  onChoose,
  onTrial,
  trialUsed,
  canTrial,
}: {
  plan: PlanOffer;
  interval: BillingInterval;
  current: boolean;
  busy: boolean;
  onChoose: () => void;
  onTrial: () => void;
  trialUsed: boolean;
  canTrial: boolean;
}) {
  const price = intervalPrice(plan, interval);
  const perMonth = interval === "quarterly" ? price / 3 : interval === "annual" ? price / 12 : price;
  return (
    <Card className={cn("flex flex-col", current ? "border-brand-300 ring-1 ring-brand-200" : "")}>
      <div className="flex flex-1 flex-col p-5">
        <div className="flex items-center justify-between">
          <p className="text-[14.5px] font-semibold text-slate-900">{plan.name}</p>
          {current ? <Badge variant="brand">Current plan</Badge> : plan.highlighted ? <Badge variant="neutral">Most popular</Badge> : null}
        </div>
        <p className="mt-1 text-[12.5px] text-slate-400">{plan.tagline}</p>
        <p className="mt-3 text-2xl font-semibold tracking-tight text-slate-900">
          {plan.negotiated ? `from ${formatPrice(plan.from_minor ?? 0, plan.currency)}` : formatPrice(price, plan.currency)}
        </p>
        <p className="mt-0.5 text-[12px] text-slate-400">
          {plan.negotiated
            ? "negotiated, flat volume rate"
            : plan.monthly_minor === 0
              ? "no card, no expiry"
              : `${formatPrice(Math.round(perMonth), plan.currency)} a month, billed ${interval}`}
        </p>

        <ul className="mt-4 flex-1 space-y-1.5">
          {[
            countWithNoun(plan.entitlements.applications, "application"),
            countWithNoun(plan.entitlements.seats, "developer seat"),
            paymentsPhrase(plan.entitlements.payments_per_month),
            plan.entitlements.routed_volume_allowance_minor > 0
              ? `${formatPrice(plan.entitlements.routed_volume_allowance_minor, plan.currency)} routed volume included`
              : plan.negotiated
                ? "Flat negotiated volume rate"
                : "Volume fee never applies",
            plan.entitlements.support,
          ].map((feature) => (
            <li key={feature} className="flex items-start gap-2 text-[12.5px] text-slate-600">
              <Check className="mt-0.5 size-3.5 shrink-0 text-emerald-500" />
              {feature}
            </li>
          ))}
        </ul>

        {plan.negotiated ? (
          <Link href="/docs" className="mt-4">
            <Button variant="secondary" size="sm" className="w-full">
              Talk to us
            </Button>
          </Link>
        ) : current ? (
          <Button variant="secondary" size="sm" className="mt-4 w-full" disabled>
            Your plan
          </Button>
        ) : (
          <Button
            variant={plan.highlighted ? "primary" : "secondary"}
            size="sm"
            className="mt-4 w-full"
            disabled={busy}
            onClick={plan.code === "pro" && canTrial && !trialUsed ? onTrial : onChoose}
          >
            {plan.code === "pro" && canTrial && !trialUsed ? "Start 14-day trial" : `Choose ${plan.name}`}
          </Button>
        )}
        {plan.code === "pro" && canTrial && !trialUsed ? (
          <button
            type="button"
            disabled={busy}
            onClick={onChoose}
            className="mt-2 text-[12px] text-slate-400 underline-offset-2 hover:text-slate-600 hover:underline disabled:opacity-50"
          >
            or subscribe to Pro directly
          </button>
        ) : null}
      </div>
    </Card>
  );
}

/* ----------------------------------- page ----------------------------------- */

export default function BillingSettingsPage() {
  const queryClient = useQueryClient();
  const [interval, setInterval] = React.useState<BillingInterval>("monthly");

  const view = useAsync(getBillingPlan, [], ["billing", "plan"]);
  const usage = useAsync(getBillingUsage, [], ["billing", "usage"]);
  const events = useAsync(listSubscriptionEvents, [], ["billing", "events"]);

  const invalidate = React.useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["billing", "plan"] }),
      queryClient.invalidateQueries({ queryKey: ["billing", "usage"] }),
      queryClient.invalidateQueries({ queryKey: ["billing", "events"] }),
    ]);
  }, [queryClient]);

  const onError = (error: unknown) =>
    toast("error", "That did not go through", error instanceof Error ? error.message : "Please try again.");

  const choose = useMutation({
    mutationFn: (plan: PlanOffer) => choosePlan({ plan: plan.code, interval }),
    onSuccess: async (result) => {
      await invalidate();
      if (result.notice) toast("warning", "Plan recorded", result.notice);
      else toast("success", `${result.plan.name} selected`, "Your new plan is active.");
    },
    onError,
  });

  const trial = useMutation({
    mutationFn: () => startPlanTrial(),
    onSuccess: async (result) => {
      await invalidate();
      toast(
        "success",
        "Trial started",
        `${result.plan.name} for 14 days${result.trial_ends_at ? `, until ${formatDate(result.trial_ends_at)}` : ""} — no card needed.`,
      );
    },
    onError,
  });

  const cancel = useMutation({
    mutationFn: () => cancelSubscription(),
    onSuccess: async (result) => {
      await invalidate();
      // Two different facts wear the same button. A paid plan runs to the end of
      // the period it was paid for; a plan that was only a recorded intention
      // ends now, because there was never anything to keep working.
      if (result.status === "cancelled") {
        toast("success", "Plan choice cancelled", "You are on Free — nothing was ever charged.");
        return;
      }
      toast("success", "Cancellation recorded", "Access continues to the end of the period you already have.");
    },
    onError,
  });

  const resume = useMutation({
    mutationFn: () => resumeSubscription(),
    onSuccess: async () => {
      await invalidate();
      toast("success", "Welcome back", "Your cancellation was withdrawn.");
    },
    onError,
  });

  const busy = choose.isPending || trial.isPending || cancel.isPending || resume.isPending;

  if (view.loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-56 w-full" />
      </div>
    );
  }

  if (view.error || !view.data) {
    return (
      <Card>
        <EmptyState
          icon={<AlertTriangle className="size-5" />}
          title="We could not read your plan"
          description={view.error instanceof Error ? view.error.message : "Please reload the page and try again."}
          action={
            <Button variant="secondary" onClick={() => view.refetch()}>
              <RefreshCw className="size-4" /> Retry
            </Button>
          }
        />
      </Card>
    );
  }

  const data = view.data;
  const current = data.plan;
  const currency = current.currency;
  const options = data.options.length ? data.options : OFFLINE_CATALOG.bands[0].plans;
  const paidOptions = options.filter((plan) => plan.code === "lite" || plan.code === "pro");
  const canTrial = data.status === "free";
  // Cancel is offered only while there is something to cancel. A pending
  // checkout has something (the recorded intention); Free and an already
  // cancelled subscription do not, and the endpoint answers those with a
  // not-found — a button that can only ever fail is worse than no button.
  const canCancel =
    !data.cancel_at_period_end &&
    (["trialing", "active", "past_due", "pending_payment"] as const).some(
      (status) => status === data.status,
    );

  const periodLine = (() => {
    if (data.status === "trialing") {
      return data.trial_ends_at ? `Trial ends ${formatDate(data.trial_ends_at)}` : "Trial active";
    }
    if (data.status === "active" && data.current_period_end) {
      return data.cancel_at_period_end
        ? `Cancels on ${formatDate(data.current_period_end)}`
        : `Renews ${formatDate(data.current_period_end)}`;
    }
    return null;
  })();

  return (
    <div className="space-y-4">
      {/* Current plan */}
      <Card>
        <div className="flex flex-wrap items-start justify-between gap-3 p-6">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold text-slate-900">Current plan</h2>
              <Badge variant={STATUS_VARIANT[data.status]}>{STATUS_LABEL[data.status]}</Badge>
            </div>
            <p className="mt-2 font-display text-2xl text-slate-950">
              {current.name}
              {current.monthly_minor > 0 ? (
                <span className="ml-2 text-[13px] font-sans font-normal text-slate-400">
                  {formatPrice(intervalPrice(current, data.interval === "none" ? "monthly" : data.interval), currency)} /{" "}
                  {INTERVAL_LABEL[data.interval === "none" ? "monthly" : data.interval].toLowerCase()}
                </span>
              ) : null}
            </p>
            <p className="mt-1 text-[12.5px] text-slate-400">
              {current.tagline}
              {periodLine ? ` · ${periodLine}` : ""}
            </p>
            <p className="mt-1 text-[12px] text-slate-400">
              Band {data.band} · {currency} — derived from your organization&apos;s country, so nothing here is a
              setting to game.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {data.cancel_at_period_end ? (
              <Button variant="secondary" disabled={busy} onClick={() => resume.mutate()}>
                Resume plan
              </Button>
            ) : canCancel ? (
              <Button variant="secondary" disabled={busy} onClick={() => cancel.mutate()}>
                {data.status === "pending_payment" ? "Cancel request" : "Cancel plan"}
              </Button>
            ) : null}
            <a href="#plans">
              <Button>
                {data.status === "free" ? "Choose a plan" : "Change plan"} <ArrowRight className="size-4" />
              </Button>
            </a>
          </div>
        </div>

        {data.notice ? (
          <div className="mx-6 mb-6 flex items-start gap-2.5 rounded-xl border border-amber-200 bg-amber-50 p-3.5">
            <Info className="mt-0.5 size-4 shrink-0 text-amber-600" />
            <p className="text-[12.5px] leading-relaxed text-amber-900">{data.notice}</p>
          </div>
        ) : null}
      </Card>

      {/* Usage */}
      <Card>
        <div className="flex flex-wrap items-center justify-between gap-2 p-6 pb-4">
          <div>
            <h2 className="text-sm font-semibold text-slate-900">Usage this month</h2>
            <p className="mt-1 text-[12.5px] text-slate-400">
              Measured from our own records — applications, seats, payments and the volume we actually routed.
              {usage.data ? ` Period ${usage.data.period}.` : ""}
            </p>
          </div>
        </div>
        <Separator />
        {usage.loading || !usage.data ? (
          <div className="grid gap-5 p-6 sm:grid-cols-2">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : (
          <>
            <div className="grid gap-6 p-6 sm:grid-cols-2">
              <Meter
                label="Applications"
                used={usage.data.applications.used}
                limit={usage.data.applications.limit}
                hint={
                  usage.data.applications.limit > 0 && usage.data.applications.used >= usage.data.applications.limit
                    ? "At the limit — extra applications are billed as add-ons on a paid plan."
                    : current.entitlements.extra_application_minor > 0
                      ? `Beyond that, ${formatPrice(current.entitlements.extra_application_minor, currency)} each.`
                      : undefined
                }
              />
              <Meter
                label="Developer seats"
                used={usage.data.seats.used}
                limit={usage.data.seats.limit}
                hint={
                  current.entitlements.extra_seat_minor > 0
                    ? `Beyond that, ${formatPrice(current.entitlements.extra_seat_minor, currency)} each.`
                    : undefined
                }
              />
              <Meter label="Live payments" used={usage.data.payments.used} limit={usage.data.payments.limit} />
              <VolumeMeter usage={usage.data} />
            </div>
            <Separator />
            <div className="flex flex-wrap items-center justify-between gap-3 p-6">
              <div>
                <p className="text-[13px] font-medium text-slate-900">Platform fee on routed volume</p>
                <p className="mt-1 text-[12.5px] text-slate-500">
                  {data.checkout_enabled
                    ? "Card payments are collected by our billing provider and reconciled into this subscription. Nothing is ever taken from a payment you route."
                    : feeNote(usage.data, current, currency)}
                </p>
              </div>
              <p className="font-mono text-lg text-slate-900">{formatPrice(usage.data.volume_fee.amount, currency)}</p>
            </div>
          </>
        )}
      </Card>

      {/* Chooser */}
      <div id="plans" className="scroll-mt-20">
        <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold text-slate-900">Change plan</h2>
            <p className="mt-1 text-[12.5px] text-slate-400">
              Priced in {currency} for your band. Quarterly is the same price as three months of monthly; annual is
              eleven months for twelve.
            </p>
          </div>
          <Tabs
            value={interval}
            onValueChange={(value) => setInterval(value as BillingInterval)}
            tabs={INTERVALS.map((value) => ({ value, label: INTERVAL_LABEL[value] }))}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          {paidOptions.map((plan) => (
            <PlanChoice
              key={plan.code}
              plan={plan}
              interval={interval}
              current={plan.code === current.code && data.status !== "free"}
              busy={busy}
              trialUsed={data.trial_used}
              canTrial={canTrial}
              onChoose={() => choose.mutate(plan)}
              onTrial={() => trial.mutate()}
            />
          ))}
        </div>

        <div className="mt-3 flex items-start gap-2.5 rounded-xl border border-slate-200 bg-[#F8FAFC] p-4">
          <Info className="mt-0.5 size-4 shrink-0 text-slate-400" />
          <p className="text-[12.5px] leading-relaxed text-slate-600">
            {data.checkout_enabled
              ? "Card payments are collected by our billing provider and reconciled into this subscription. Nothing is ever taken from a payment you route."
              : "Card payments are not open yet — there is no payment provider connected to PayConnect itself. Choosing a plan records your intention and changes nothing you are charged. When billing opens, we will confirm with you before taking anything."}{" "}
            Enterprise is quoted individually:{" "}
            <Link href="/docs" className="underline hover:text-slate-900">
              talk to us
            </Link>
            .
          </p>
        </div>
      </div>

      {/* What the plan includes */}
      <Card>
        <div className="p-6">
          <h2 className="text-sm font-semibold text-slate-900">What {current.name} includes</h2>
          <div className="mt-4 grid gap-x-6 gap-y-2 sm:grid-cols-2">
            {[
              countWithNoun(current.entitlements.applications, "application"),
              countWithNoun(current.entitlements.seats, "developer seat"),
              paymentsPhrase(current.entitlements.payments_per_month),
              countWithNoun(current.entitlements.webhook_endpoints, "webhook endpoint"),
              current.entitlements.routing_rules ? "Routing rules and automatic failover" : "No routing rules yet",
              current.entitlements.webhook_replay ? "Webhook replay and retry tuning" : "Basic webhook retries",
              `${current.entitlements.log_retention_days > 0 ? `${current.entitlements.log_retention_days}-day` : "Custom"} log retention`,
              current.entitlements.audit_export ? "Audit-log export" : "No audit-log export",
              current.entitlements.provider_connections || "Provider connections limited",
              `${current.entitlements.support} · ${current.entitlements.uptime_target} target`,
            ].map((line) => (
              <p key={line} className="flex items-start gap-2 text-[12.5px] text-slate-600">
                <Check className="mt-0.5 size-3.5 shrink-0 text-emerald-500" />
                {line}
              </p>
            ))}
          </div>
        </div>
      </Card>

      {/* Activity */}
      <Card>
        <div className="p-6 pb-3">
          <h2 className="text-sm font-semibold text-slate-900">Billing activity</h2>
          <p className="mt-1 text-[12.5px] text-slate-400">
            Every plan change is a billing fact, and it is written to the audit log unchanged.
          </p>
        </div>
        <Separator />
        {events.loading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        ) : (events.data ?? []).length === 0 ? (
          <p className="p-6 text-[12.5px] text-slate-400">
            Nothing yet — no plan has been changed on this organization.
          </p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {(events.data ?? []).map((event) => (
              <li key={event.id} className="flex flex-wrap items-center justify-between gap-2 px-6 py-3">
                <div>
                  <p className="text-[13px] text-slate-800">{event.detail || event.type}</p>
                  <p className="mt-0.5 text-[12px] text-slate-400">
                    {event.from_plan && event.to_plan && event.from_plan !== event.to_plan
                      ? `${event.from_plan} → ${event.to_plan} · `
                      : ""}
                    {event.source}
                    {event.actor_name && event.actor_name !== "system" ? ` · ${event.actor_name}` : ""}
                  </p>
                </div>
                <p className="text-[12px] text-slate-400">{formatDate(event.created_at)}</p>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
