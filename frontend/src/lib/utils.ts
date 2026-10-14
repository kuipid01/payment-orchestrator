import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/* ---------------------------------- money ---------------------------------- */

const CURRENCIES: Record<string, { symbol: string; digits: number }> = {
  NGN: { symbol: "₦", digits: 2 },
  USD: { symbol: "$", digits: 2 },
  GHS: { symbol: "₵", digits: 2 },
  KES: { symbol: "KSh", digits: 2 },
  ZAR: { symbol: "R", digits: 2 },
  EUR: { symbol: "€", digits: 2 },
  GBP: { symbol: "£", digits: 2 },
};

export function formatAmount(minor: number, currency: string): string {
  const meta = CURRENCIES[currency] ?? { symbol: "", digits: 2 };
  const value = minor / 10 ** meta.digits;
  const formatted = value.toLocaleString("en-US", {
    minimumFractionDigits: meta.digits,
    maximumFractionDigits: meta.digits,
  });
  return `${meta.symbol}${formatted}`;
}

export function formatNumber(n: number): string {
  return n.toLocaleString("en-US");
}

export function formatCompact(n: number): string {
  return new Intl.NumberFormat("en-US", { notation: "compact", maximumFractionDigits: 1 }).format(n);
}

/* ----------------------------------- dates ---------------------------------- */

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" });
}

export function timeAgo(iso: string): string {
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  return formatDate(iso);
}

/* --------------------------------- statuses --------------------------------- */

export const PAYMENT_STATUSES = [
  "successful",
  "pending",
  "processing",
  "failed",
  "refunded",
  "partially_refunded",
  "cancelled",
  "expired",
] as const;
export type PaymentStatus = (typeof PAYMENT_STATUSES)[number];

export const PAYMENT_STATUS_META: Record<
  PaymentStatus,
  { label: string; className: string; dot: string }
> = {
  successful: {
    label: "Successful",
    className: "bg-emerald-50 text-emerald-700 border-emerald-200",
    dot: "bg-emerald-500",
  },
  pending: {
    label: "Pending",
    className: "bg-amber-50 text-amber-700 border-amber-200",
    dot: "bg-amber-500",
  },
  processing: {
    label: "Processing",
    className: "bg-sky-50 text-sky-700 border-sky-200",
    dot: "bg-sky-500",
  },
  failed: {
    label: "Failed",
    className: "bg-red-50 text-red-700 border-red-200",
    dot: "bg-red-500",
  },
  refunded: {
    label: "Refunded",
    className: "bg-sky-50 text-sky-700 border-sky-200",
    dot: "bg-sky-500",
  },
  partially_refunded: {
    label: "Partially refunded",
    className: "bg-sky-50 text-sky-700 border-sky-200",
    dot: "bg-sky-500",
  },
  cancelled: {
    label: "Cancelled",
    className: "bg-slate-100 text-slate-600 border-slate-200",
    dot: "bg-slate-400",
  },
  expired: {
    label: "Expired",
    className: "bg-slate-100 text-slate-600 border-slate-200",
    dot: "bg-slate-400",
  },
};

/* --------------------------------- api keys --------------------------------- */

export type Environment = "test" | "live";

export function maskKey(prefix: string, last4: string, starred = 12): string {
  return `${prefix}${"•".repeat(starred)}${last4}`;
}

export function generateApiKey(env: Environment): string {
  const rand = () => Math.random().toString(36).slice(2, 10);
  return `vpay_${env}_${rand()}${rand()}${rand()}${rand()}`;
}
