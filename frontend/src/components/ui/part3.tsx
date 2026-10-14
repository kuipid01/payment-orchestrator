/* shadcn-style primitives — part 3: payment-domain + table components */

"use client";

import * as React from "react";
import { ChevronLeft, ChevronRight, Search } from "lucide-react";
import { cn, PAYMENT_STATUS_META, type PaymentStatus } from "@/lib/utils";
import { PROVIDER_META, type ProviderKind } from "@/lib/types";
import { Button, Skeleton } from "./primitives";

/* ----------------------------- payment status badge -------------------------- */

export function PaymentStatusBadge({ status }: { status: PaymentStatus }) {
  const meta = PAYMENT_STATUS_META[status];
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium ${meta.className}`}>
      <span className={`size-1.5 rounded-full ${meta.dot}`} />
      {meta.label}
    </span>
  );
}

/* -------------------------------- provider badge ----------------------------- */

export function ProviderBadge({
  provider,
  size = "md",
}: {
  provider: ProviderKind;
  size?: "sm" | "md";
}) {
  const meta = PROVIDER_META[provider];
  return (
    <span className="inline-flex items-center gap-2">
      <span
        className={cn(
          "inline-flex items-center justify-center rounded-md font-semibold",
          meta.accent,
          size === "sm" ? "size-5 text-[9px]" : "size-6 text-[10px]"
        )}
      >
        {meta.initials}
      </span>
      <span className="text-[13px] font-medium text-slate-700">{meta.name}</span>
    </span>
  );
}

/* ------------------------------ environment badge ---------------------------- */

export function EnvironmentBadge({ environment }: { environment: "test" | "live" }) {
  return environment === "live" ? (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700">
      <span className="size-1.5 rounded-full bg-emerald-500" />
      Production
    </span>
  ) : (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700">
      <span className="size-1.5 rounded-full bg-amber-500" />
      Test
    </span>
  );
}

/* -------------------------------- amount display ----------------------------- */

export function AmountDisplay({
  minor,
  currency,
  className,
  strong = true,
}: {
  minor: number;
  currency: string;
  className?: string;
  strong?: boolean;
}) {
  const meta: Record<string, string> = { NGN: "₦", USD: "$", GHS: "₵", KES: "KSh", ZAR: "R", EUR: "€", GBP: "£" };
  const value = (minor / 100).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  return (
    <span className={cn("font-mono text-[13px]", strong ? "font-medium text-slate-900" : "text-slate-600", className)}>
      {meta[currency] ?? ""}
      {value}
    </span>
  );
}

/* ----------------------------- transaction timeline -------------------------- */

const TIMELINE_META: Record<string, { label: string; dot: string; line: string }> = {
  created: { label: "Payment created", dot: "bg-brand-500", line: "bg-slate-200" },
  provider_request: { label: "Provider request sent", dot: "bg-brand-500", line: "bg-slate-200" },
  provider_confirmed: { label: "Provider confirmed", dot: "bg-brand-500", line: "bg-slate-200" },
  successful: { label: "Payment successful", dot: "bg-emerald-500", line: "" },
  pending: { label: "Awaiting authorization", dot: "bg-amber-400", line: "" },
  failed: { label: "Failed", dot: "bg-red-500", line: "" },
  refunded: { label: "Refunded", dot: "bg-sky-500", line: "" },
};

export function TransactionTimeline({
  events,
}: {
  events: { id: string; type: string; message: string; at: string }[];
}) {
  return (
    <ol className="relative ml-1.5 border-l border-slate-200">
      {events.map((e) => {
        const meta = TIMELINE_META[e.type] ?? TIMELINE_META.created;
        return (
          <li key={e.id} className="relative pb-5 pl-6 last:pb-0">
            <span className={cn("absolute -left-[5px] top-1 size-2.5 rounded-full ring-4 ring-white", meta.dot)} />
            <p className="text-[13px] font-medium text-slate-900">{e.message}</p>
            <p className="mt-0.5 text-xs text-slate-400">
              {new Date(e.at).toLocaleString("en-US", {
                month: "short",
                day: "numeric",
                hour: "numeric",
                minute: "2-digit",
              })}
            </p>
          </li>
        );
      })}
    </ol>
  );
}

/* ---------------------------------- data table ------------------------------- */

export function DataTable<T>({
  columns,
  data,
  loading,
  empty,
  keyOf,
  onRowClick,
  pageSize = 10,
}: {
  columns: {
    key: string;
    header: React.ReactNode;
    className?: string;
    render: (row: T) => React.ReactNode;
  }[];
  data: T[] | undefined;
  loading?: boolean;
  empty?: React.ReactNode;
  keyOf: (row: T) => string;
  onRowClick?: (row: T) => void;
  pageSize?: number;
}) {
  const [page, setPage] = React.useState(0);
  const [query, setQuery] = React.useState("");

  const filtered = React.useMemo(() => {
    if (!data) return [];
    return data;
  }, [data]);

  React.useEffect(() => setPage(0), [query, data]);

  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const pageRows = filtered.slice(page * pageSize, page * pageSize + pageSize);
  const isEmpty = !loading && filtered.length === 0;

  return (
    <div>
      <div className="flex items-center justify-between gap-3 border-b border-slate-100 px-5 py-3">
        <div className="relative w-full max-w-xs">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search…"
            className="h-9 w-full rounded-[var(--radius-btn)] border border-slate-200 bg-white pl-9 pr-3 text-sm placeholder:text-slate-400 hover:border-slate-300 focus:border-brand-400 focus:outline-none focus:ring-2 focus:ring-brand-500/15"
          />
        </div>
        <p className="hidden shrink-0 text-xs text-slate-400 sm:block">
          {data ? `${filtered.length} record${filtered.length === 1 ? "" : "s"}` : ""}
        </p>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead>
            <tr className="border-b border-slate-100 bg-slate-50/60 text-[12px] uppercase tracking-wide text-slate-400">
              {columns.map((c) => (
                <th key={c.key} className={cn("px-5 py-2.5 font-medium", c.className)}>
                  {c.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {loading
              ? Array.from({ length: 4 }).map((_, i) => (
                  <tr key={i}>
                    {columns.map((c, j) => (
                      <td key={c.key} className="px-5 py-3.5">
                        <Skeleton className={cn("h-4", j === 0 ? "w-36" : j === columns.length - 1 ? "w-24" : "w-20")} />
                      </td>
                    ))}
                  </tr>
                ))
              : pageRows.map((row) => (
                  <tr
                    key={keyOf(row)}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                    className={cn(
                      "transition-colors",
                      onRowClick && "cursor-pointer hover:bg-slate-50/80"
                    )}
                  >
                    {columns.map((c) => (
                      <td key={c.key} className={cn("px-5 py-3.5", c.className)}>
                        {c.render(row)}
                      </td>
                    ))}
                  </tr>
                ))}
          </tbody>
        </table>
      </div>

      {isEmpty && <div className="p-5">{empty}</div>}

      {!loading && filtered.length > pageSize ? (
        <div className="flex items-center justify-between border-t border-slate-100 px-5 py-3">
          <p className="text-xs text-slate-400">
            Page {page + 1} of {pageCount}
          </p>
          <div className="flex gap-1">
            <Button variant="secondary" size="icon" disabled={page === 0} onClick={() => setPage((p) => p - 1)} aria-label="Previous page">
              <ChevronLeft className="size-4" />
            </Button>
            <Button
              variant="secondary"
              size="icon"
              disabled={page >= pageCount - 1}
              onClick={() => setPage((p) => p + 1)}
              aria-label="Next page"
            >
              <ChevronRight className="size-4" />
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

/* --------------------------------- empty state ------------------------------- */

export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-14 text-center">
      {icon ? (
        <div className="mb-4 flex size-11 items-center justify-center rounded-full border border-brand-100 bg-brand-50 text-brand-600">
          {icon}
        </div>
      ) : null}
      <h3 className="text-sm font-semibold text-slate-900">{title}</h3>
      {description ? <p className="mt-1 max-w-sm text-[13px] text-slate-500">{description}</p> : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}
