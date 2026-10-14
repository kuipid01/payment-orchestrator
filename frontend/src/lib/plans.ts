/**
 * The plan catalog, as the API serves it.
 *
 * These types mirror `internal/billing/dto.go` field for field, including its
 * snake_case: a translation layer here would be one more place for a price to
 * be wrong.
 *
 * `plan-catalog.json` is *generated* from the Go catalog
 * (`cd backend && go run ./cmd/catalogdump > ../frontend/src/lib/plan-catalog.json`)
 * and is only used when the API cannot be reached. Because it is generated
 * rather than hand-kept, it cannot disagree with what the API serves — which is
 * the whole point, given that two hand-written pricing pages claiming different
 * free-tier limits is what this replaced.
 */

import { API_BASE_URL } from "./api";
import generatedCatalog from "./plan-catalog.json";

export type PlanCode = "free" | "lite" | "pro" | "enterprise";
export type PriceBand = "A" | "B" | "C" | "D";
export type BillingInterval = "monthly" | "quarterly" | "annual" | "none";

/** Subscription lifecycle, mirroring the backend's status constants. */
export type SubscriptionStatus =
  | "free"
  | "trialing"
  | "active"
  | "past_due"
  | "pending_payment"
  | "cancelled";

/** What a plan grants. -1 means no ceiling. */
export interface PlanEntitlements {
  applications: number;
  extra_application_minor: number;
  seats: number;
  extra_seat_minor: number;
  payments_per_month: number;
  routed_volume_allowance_minor: number;
  routed_volume_cap_minor: number;
  routed_volume_ceiling_minor?: number;
  webhook_endpoints: number;
  routing_rules: boolean;
  priority_dispatch: boolean;
  /** A supported shape, not a gate: e.g. "All three, both environments". */
  provider_connections: string;
  /** Replay plus retry tuning. */
  webhook_replay: boolean;
  dedicated_region: boolean;
  log_retention_days: number;
  audit_export: boolean;
  sla_credits: boolean;
  support: string;
  uptime_target: string;
}

/** One plan priced in one band. Every amount is minor units of `currency`. */
export interface PlanOffer {
  code: PlanCode;
  name: string;
  tagline: string;
  /** Quoted per merchant: the price is a "from" and the volume fee is flat. */
  negotiated: boolean;
  highlighted: boolean;
  band: PriceBand;
  currency: string;
  monthly_minor: number;
  quarterly_minor: number;
  annual_minor: number;
  from_minor?: number;
  entitlements: PlanEntitlements;
}

export interface PlanBand {
  band: PriceBand;
  currency: string;
  countries: string[];
  plans: PlanOffer[];
}

export interface VolumeFeeTerms {
  rate_bps: number;
  rate_label: string;
  allowance_multiple: number;
  cap_multiple: number;
  scope: string;
  billed: string;
  promise: string;
}

export interface PlanCatalog {
  trial_days: number;
  volume_fee: VolumeFeeTerms;
  bands: PlanBand[];
}

/** A merchant's own billing state, from `GET /v1/billing/plan`. */
export interface SubscriptionPlan {
  plan: PlanOffer;
  band: PriceBand;
  status: SubscriptionStatus;
  interval: BillingInterval;
  current_period_start?: string;
  current_period_end?: string;
  trial_ends_at?: string;
  cancel_at_period_end: boolean;
  trial_used: boolean;
  provider?: string;
  /** False until a collection provider is wired: a chosen plan is not charged. */
  checkout_enabled: boolean;
  notice?: string;
  /** Every plan in the merchant's own band, at the prices they would pay. */
  options: PlanOffer[];
}

/** One metered dimension: what is used against what the plan includes. */
export interface UsageMetric {
  used: number;
  limit: number;
}

/** A month of usage, from `GET /v1/billing/usage`. */
export interface BillingUsage {
  period: string;
  plan: PlanCode;
  currency: string;
  applications: UsageMetric;
  seats: UsageMetric;
  payments: UsageMetric;
  routed_volume: {
    used: number;
    refunded: number;
    allowance: number;
    cap: number;
    ceiling?: number;
  };
  volume_fee: { amount: number; uncapped: number; rate_bps: number };
}

/** One subscription state change, from `GET /v1/billing/events`. */
export interface SubscriptionEvent {
  id: string;
  type: string;
  from_plan: string;
  to_plan: string;
  source: string;
  detail: string;
  actor_name: string;
  created_at: string;
}

/**
 * The offline catalog: the last generated copy of the Go catalog. Used by the
 * marketing page when the API is unreachable, so a price list exists even when
 * the process that serves it does not.
 */
export const OFFLINE_CATALOG = generatedCatalog as unknown as PlanCatalog;

export interface CatalogLoad {
  catalog: PlanCatalog;
  /** True when the live catalog was served; false when the local copy was used. */
  live: boolean;
}

/**
 * Loads the public catalog. The marketing page renders prices it does not own,
 * and a pricing page that 500s because the API blipped is worse than a slightly
 * stale one, so a failure falls back to the generated copy.
 */
export async function fetchPlanCatalog(): Promise<CatalogLoad> {
  try {
    const response = await fetch(`${API_BASE_URL}/plans`, { next: { revalidate: 300 } });
    if (!response.ok) throw new Error(`catalog request failed with ${response.status}`);
    const body = (await response.json()) as { data?: PlanCatalog };
    if (!body?.data?.bands?.length) throw new Error("catalog response had no bands");
    return { catalog: body.data, live: true };
  } catch {
    return { catalog: OFFLINE_CATALOG, live: false };
  }
}

/* ---------------------------------- display --------------------------------- */

const CURRENCY_SYMBOL: Record<string, string> = {
  NGN: "₦",
  USD: "$",
  GHS: "₵",
  KES: "KSh",
  ZAR: "R",
  EUR: "€",
  GBP: "£",
};

/**
 * Renders a price in minor units. Prices here are always whole major units
 * (₦3,500, $9), so the decimals are dropped when they are zero — "₦3,500.00"
 * reads like an exchange rate, which is an invitation to argue.
 */
export function formatPrice(minor: number, currency: string): string {
  const symbol = CURRENCY_SYMBOL[currency] ?? "";
  const major = minor / 100;
  const hasFraction = Math.round(minor) % 100 !== 0;
  const formatted = major.toLocaleString("en-US", {
    minimumFractionDigits: hasFraction ? 2 : 0,
    maximumFractionDigits: hasFraction ? 2 : 0,
  });
  return `${symbol}${formatted}`;
}

/** Renders a limit the way the catalog states it, including "Unlimited". */
export function formatLimit(limit: number): string {
  return limit < 0 ? "Unlimited" : limit.toLocaleString("en-US");
}

/** "100" vs "Unlimited" for a payments-per-month ceiling. */
export function paymentsAllowance(limit: number): string {
  return limit < 0 ? "Unlimited" : `${limit.toLocaleString("en-US")} / month`;
}

/**
 * The same ceiling as a sentence fragment: "Unlimited payments" / "100 payments a
 * month". `paymentsAllowance` returns "100 / month", which is right in a table
 * cell and reads like a spreadsheet everywhere else.
 */
export function paymentsPhrase(limit: number): string {
  return limit < 0 ? "Unlimited payments" : `${limit.toLocaleString("en-US")} payments a month`;
}

/**
 * A count with its noun, pluralised: "1 application" / "3 applications". The
 * limits a merchant reads are the sales pitch, so they are written as prose.
 */
export function countWithNoun(limit: number, singular: string): string {
  return `${formatLimit(limit)} ${singular}${limit === 1 ? "" : "s"}`;
}

export const BAND_REGION: Record<PriceBand, { label: string; blurb: string }> = {
  A: { label: "Nigeria", blurb: "Naira pricing, set independently of the dollar bands." },
  B: { label: "Africa", blurb: "Ghana, Kenya, South Africa, Egypt and Rwanda." },
  C: { label: "Emerging markets", blurb: "India, Pakistan, Indonesia, the Philippines, Brazil and Vietnam." },
  D: { label: "Global", blurb: "US, Canada, UK, EU, Australia, New Zealand, Singapore and the UAE." },
};

export const INTERVAL_LABEL: Record<BillingInterval, string> = {
  monthly: "Monthly",
  quarterly: "Quarterly",
  annual: "Annual",
  none: "No billing",
};

/**
 * The per-interval price of a plan, plus the honest note about what the cadence
 * buys. Quarterly is deliberately *not* discounted: it saves three collections a
 * year, and the numbers say that is worth more to the merchant than to us.
 */
export function intervalPrice(plan: PlanOffer, interval: BillingInterval): number {
  switch (interval) {
    case "quarterly":
      return plan.quarterly_minor;
    case "annual":
      return plan.annual_minor;
    default:
      return plan.monthly_minor;
  }
}

export function intervalNote(interval: BillingInterval): string {
  switch (interval) {
    case "quarterly":
      return "Three months at the full price — no discount, and three collections a year instead of twelve.";
    case "annual":
      return "Eleven months for twelve — one month free.";
    case "monthly":
      return "Cancel any time; access continues to the end of the period you paid for.";
    default:
      return "";
  }
}

/** The cadence toggle's options, in the order they are offered. */
export const INTERVALS: BillingInterval[] = ["monthly", "quarterly", "annual"];

export const STATUS_LABEL: Record<SubscriptionStatus, string> = {
  free: "Free plan",
  trialing: "Trial",
  active: "Active",
  past_due: "Payment pending",
  pending_payment: "Awaiting payment",
  cancelled: "Cancelled",
};

/** The fee rate in basis points (0.05%), as the catalog states it. */
export const VOLUME_FEE_RATE_BPS = 5;

/**
 * The fee on one month of routed volume, computed the same way the backend
 * computes it. It exists so the pricing page can show the ladder *from the
 * catalog numbers* rather than from a hand-typed table that can drift.
 */
export function volumeFee(routedMinor: number, plan: PlanOffer): number {
  if (plan.negotiated) return 0;
  const { routed_volume_allowance_minor: allowance, routed_volume_cap_minor: cap } = plan.entitlements;
  if (routedMinor <= allowance) return 0;
  const fee = Math.floor(((routedMinor - allowance) * VOLUME_FEE_RATE_BPS) / 10_000);
  return Math.min(fee, cap);
}

/** The plan at a given code inside a band, or undefined when it is not sold. */
export function findPlan(catalog: PlanCatalog, band: PriceBand, code: PlanCode): PlanOffer | undefined {
  return catalog.bands.find((entry) => entry.band === band)?.plans.find((plan) => plan.code === code);
}
