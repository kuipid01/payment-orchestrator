"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Download, KeyRound, Receipt, RotateCcw, UserCheck } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  EnvironmentBadge,
  PageHeader,
  PaymentStatusBadge,
  Select,
} from "@/components/ui";
import { AmountDisplay, ProviderBadge } from "@/components/ui/part3";
import { useAsync } from "@/hooks/use-async";
import { listApiKeys, listApplications, listDevelopers, listPayments } from "@/lib/api";
import type { PaymentFilters } from "@/lib/api";
import { PAYMENT_STATUSES, formatNumber, timeAgo } from "@/lib/utils";
import { PROVIDER_META } from "@/lib/types";
import type { ApiKey, Application, Developer, Payment } from "@/lib/types";

const PROVIDERS = Object.keys(PROVIDER_META);

const PAGE_SIZE = 25;
const ALL = "all";

/** The sentinel that selects payments taken with a shared, non-developer key. */
const GENERAL_KEY = "general";

export default function PaymentsPage() {
  const router = useRouter();
  const applications = useAsync(listApplications, [], ["applications"]);
  const developers = useAsync(listDevelopers, [], ["developers"]);
  const apiKeys = useAsync(() => listApiKeys(), [], ["api-keys"]);

  const [filters, setFilters] = React.useState<PaymentFilters>({
    environment: ALL,
    applicationId: "",
    developerAccessId: "",
    status: ALL,
    provider: ALL,
  });
  const [search, setSearch] = React.useState("");
  const [page, setPage] = React.useState(1);

  // The search box is one request per pause, not one per keystroke: the whole
  // table is server-side, so each keystroke would otherwise be a round trip.
  const [debouncedSearch, setDebouncedSearch] = React.useState("");
  React.useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search.trim()), 350);
    return () => clearTimeout(timer);
  }, [search]);

  React.useEffect(() => {
    setPage(1);
  }, [debouncedSearch, filters]);

  const query = React.useMemo<PaymentFilters>(
    () => ({ ...filters, search: debouncedSearch, page, limit: PAGE_SIZE }),
    [filters, debouncedSearch, page],
  );

  // The query key carries every filter, so React Query caches each combination
  // separately and a change refetches exactly one page.
  const payments = useAsync(() => listPayments(query), [JSON.stringify(query)], ["payments", query]);

  const applicationById = React.useMemo(
    () => new Map<string, Application>((applications.data ?? []).map((app) => [app.id, app])),
    [applications.data],
  );
  const keyById = React.useMemo(
    () => new Map<string, ApiKey>((apiKeys.data ?? []).map((key) => [key.id, key])),
    [apiKeys.data],
  );
  const developerByAccessId = React.useMemo(
    () => new Map<string, Developer>((developers.data ?? []).map((dev) => [dev.accessId, dev])),
    [developers.data],
  );

  const rows = payments.data?.items ?? [];
  const total = payments.data?.total ?? 0;
  const summary = payments.data?.summary ?? {};
  const filtered = Boolean(
    debouncedSearch ||
      (filters.environment && filters.environment !== ALL) ||
      filters.applicationId ||
      filters.developerAccessId ||
      (filters.status && filters.status !== ALL) ||
      (filters.provider && filters.provider !== ALL),
  );

  function update<K extends keyof PaymentFilters>(key: K, value: PaymentFilters[K]) {
    setFilters((current) => ({ ...current, [key]: value }));
  }

  function reset() {
    setFilters({ environment: ALL, applicationId: "", developerAccessId: "", status: ALL, provider: ALL });
    setSearch("");
  }

  const lastPage = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="animate-fade-in">
      <PageHeader
        title="Payments"
        description="Every payment processed through your applications, across all providers — filterable by application, developer and environment."
        actions={
          <Button variant="secondary" size="sm">
            <Download className="size-4" /> Export
          </Button>
        }
      />

      {/* Filter bar. Every control narrows the same server-side query. */}
      <div className="mb-4 space-y-3">
        <div className="flex flex-wrap items-center gap-2">
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search reference, payment ID, payer name or email…"
            className="h-9 w-full max-w-sm rounded-[var(--radius-btn)] border border-slate-200 bg-white px-3 text-sm placeholder:text-slate-400 hover:border-slate-300 focus:border-brand-400 focus:outline-none focus:ring-2 focus:ring-brand-500/15"
          />
          <Select
            value={filters.applicationId ?? ""}
            onChange={(e) => update("applicationId", e.target.value)}
            className="w-48"
            aria-label="Filter by application"
          >
            <option value="">All applications</option>
            {(applications.data ?? []).map((app) => (
              <option key={app.id} value={app.id}>
                {app.name}
              </option>
            ))}
          </Select>
          <Select
            value={filters.developerAccessId ?? ""}
            onChange={(e) => update("developerAccessId", e.target.value)}
            className="w-52"
            aria-label="Filter by developer"
          >
            <option value="">All developers</option>
            <option value={GENERAL_KEY}>General (shared key)</option>
            {(developers.data ?? []).map((dev) => (
              <option key={dev.accessId} value={dev.accessId}>
                {dev.name}
                {dev.status === "pending" ? " (pending)" : ""}
              </option>
            ))}
          </Select>
          <Select
            value={filters.environment ?? ALL}
            onChange={(e) => update("environment", e.target.value as PaymentFilters["environment"])}
            className="w-40"
            aria-label="Filter by environment"
          >
            <option value={ALL}>All environments</option>
            <option value="live">Production</option>
            <option value="test">Test</option>
          </Select>
          <Select
            value={filters.status ?? ALL}
            onChange={(e) => update("status", e.target.value)}
            className="w-40"
            aria-label="Filter by status"
          >
            <option value={ALL}>All statuses</option>
            {PAYMENT_STATUSES.map((status) => (
              <option key={status} value={status}>
                {status.charAt(0).toUpperCase() + status.slice(1)}
                {summary[status] != null ? ` (${summary[status]})` : ""}
              </option>
            ))}
          </Select>
          <Select
            value={filters.provider ?? ALL}
            onChange={(e) => update("provider", e.target.value)}
            className="w-40"
            aria-label="Filter by provider"
          >
            <option value={ALL}>All providers</option>
            {PROVIDERS.map((provider) => (
              <option key={provider} value={provider}>
                {provider.charAt(0).toUpperCase() + provider.slice(1)}
              </option>
            ))}
          </Select>
          {filtered ? (
            <Button variant="ghost" size="sm" onClick={reset}>
              <RotateCcw className="size-3.5" /> Clear
            </Button>
          ) : null}
        </div>

        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant="neutral">{formatNumber(total)} payments</Badge>
          {PAYMENT_STATUSES.filter((status) => summary[status]).map((status) => (
            <button
              key={status}
              onClick={() => update("status", filters.status === status ? ALL : status)}
              className="rounded-full transition-opacity hover:opacity-80"
            >
              <Badge
                variant={status === "successful" ? "success" : status === "failed" ? "danger" : "warning"}
              >
                {formatNumber(summary[status])} {status}
              </Badge>
            </button>
          ))}
        </div>
      </div>

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1000px] text-left text-sm">
            <thead>
              <tr className="border-b border-slate-100 bg-slate-50/60 text-[12px] uppercase tracking-wide text-slate-400">
                <th className="px-5 py-2.5 font-medium">Payment</th>
                <th className="px-5 py-2.5 font-medium">Customer</th>
                <th className="px-5 py-2.5 font-medium">Amount</th>
                <th className="px-5 py-2.5 font-medium">Application</th>
                <th className="px-5 py-2.5 font-medium">Taken by</th>
                <th className="px-5 py-2.5 font-medium">Provider</th>
                <th className="px-5 py-2.5 font-medium">Status</th>
                <th className="px-5 py-2.5 text-right font-medium">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {payments.loading
                ? Array.from({ length: 6 }).map((_, index) => (
                    <tr key={index}>
                      {Array.from({ length: 8 }).map((__, cell) => (
                        <td key={cell} className="px-5 py-4">
                          <div className="h-3.5 w-24 animate-pulse rounded bg-slate-100" />
                        </td>
                      ))}
                    </tr>
                  ))
                : rows.map((payment) => (
                    <PaymentRow
                      key={payment.id}
                      payment={payment}
                      application={applicationById.get(payment.applicationId)}
                      keyLabel={keyById.get(payment.apiKeyId ?? "")?.label}
                      developer={developerByAccessId.get(payment.developerAccessId ?? "")}
                      onOpen={() => router.push(`/payments/${payment.id}`)}
                    />
                  ))}
            </tbody>
          </table>
        </div>

        {!payments.loading && rows.length === 0 ? (
          <EmptyState
            icon={<Receipt className="size-5" />}
            title="No payments found"
            description={
              filtered
                ? "No payment matches these filters yet. Try widening the environment or clearing the search."
                : "Payments appear here the moment an application takes its first charge."
            }
            action={
              filtered ? (
                <Button variant="secondary" onClick={reset}>
                  Clear filters
                </Button>
              ) : (
                <Link href="/providers">
                  <Button variant="secondary">Connect a provider</Button>
                </Link>
              )
            }
          />
        ) : null}

        {rows.length > 0 ? (
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 px-5 py-3.5">
            <p className="text-[12.5px] text-slate-400">
              Showing {(page - 1) * PAGE_SIZE + 1}–{(page - 1) * PAGE_SIZE + rows.length} of{" "}
              {formatNumber(total)}
            </p>
            <div className="flex items-center gap-2">
              <Button variant="secondary" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                Previous
              </Button>
              <span className="text-[12.5px] text-slate-400">
                Page {page} of {lastPage}
              </span>
              <Button
                variant="secondary"
                size="sm"
                disabled={page >= lastPage}
                onClick={() => setPage(page + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        ) : null}
      </Card>
    </div>
  );
}

/**
 * One payment row. Attribution is the point: the application tells an operator
 * where the money came from, and "taken by" tells them which key — and therefore
 * which developer — took it.
 */
function PaymentRow({
  payment,
  application,
  keyLabel,
  developer,
  onOpen,
}: {
  payment: Payment;
  application?: Application;
  keyLabel?: string;
  developer?: Developer;
  onOpen: () => void;
}) {
  const payer = payment.customer.name || payment.customer.email || "Unknown payer";
  const sharedKey = !payment.developerAccessId;
  const attribution = keyLabel ?? (sharedKey ? "General key" : developer?.name ?? "Developer key");

  return (
    <tr className="cursor-pointer hover:bg-slate-50/60" onClick={onOpen}>
      <td className="px-5 py-3.5">
        <div className="flex items-center gap-2">
          <p className="font-mono text-[12.5px] font-medium text-slate-900">{payment.id}</p>
          <EnvironmentBadge environment={payment.environment} />
        </div>
        <p className="font-mono text-[11px] text-slate-400">{payment.reference}</p>
      </td>
      <td className="px-5 py-3.5">
        <p className="text-[13px] font-medium text-slate-800">{payer}</p>
        <p className="text-[11.5px] text-slate-400">{payment.customer.email || "—"}</p>
      </td>
      <td className="px-5 py-3.5">
        <AmountDisplay minor={payment.amountMinor} currency={payment.currency} />
      </td>
      <td className="px-5 py-3.5">
        {application ? (
          <Link
            href={`/applications/${application.id}`}
            onClick={(event) => event.stopPropagation()}
            className="inline-flex items-center gap-1.5 text-[12.5px] text-brand-600 hover:underline"
          >
            {application.name}
          </Link>
        ) : (
          <span className="text-[12.5px] text-slate-400">—</span>
        )}
      </td>
      <td className="px-5 py-3.5">
        <div className="flex items-center gap-1.5">
          {sharedKey ? (
            <KeyRound className="size-3.5 shrink-0 text-slate-400" />
          ) : (
            <UserCheck className="size-3.5 shrink-0 text-emerald-500" />
          )}
          <div className="min-w-0">
            <p className="truncate text-[12.5px] text-slate-700">{attribution}</p>
            <p className="text-[11px] text-slate-400">{sharedKey ? "Shared with the team" : "Developer-scoped"}</p>
          </div>
        </div>
      </td>
      <td className="px-5 py-3.5">
        <ProviderBadge provider={payment.provider} size="sm" />
      </td>
      <td className="px-5 py-3.5">
        <PaymentStatusBadge status={payment.status} />
      </td>
      <td className="px-5 py-3.5 text-right text-[12.5px] text-slate-400" title={payment.createdAt}>
        {timeAgo(payment.createdAt)}
      </td>
    </tr>
  );
}
