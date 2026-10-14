"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  ArrowRight,
  ArrowRightLeft,
  Building2,
  CheckCircle2,
  CircleAlert,
  CreditCard,
  Globe,
  LockKeyhole,
  Plus,
  Receipt,
  ShieldCheck,
  Users,
  Wallet,
} from "lucide-react";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  PaymentStatusBadge,
  Tabs,
} from "@/components/ui";
import { AmountDisplay, ProviderBadge } from "@/components/ui/part3";
import { SkeletonRows } from "@/components/ui/part2";
import { toast } from "@/components/ui/toaster";
import { useAuth } from "@/components/providers/auth-provider";
import { useAsync } from "@/hooks/use-async";
import {
  getDashboardMetrics,
  listOrganizationMemberships,
  listPayments,
  listProviders,
  switchOrganization,
} from "@/lib/api";
import { formatAmount, formatCompact, formatNumber, timeAgo } from "@/lib/utils";

const brandGold = "#E7B000";

export default function DashboardPage() {
  const metrics = useAsync(getDashboardMetrics, []);
  const payments = useAsync(() => listPayments({ limit: 5 }), [], ["payments", "recent"]);
  const providers = useAsync(listProviders, [], ["providers"]);
  const { user } = useAuth();
  const [tab, setTab] = React.useState<"overview" | "organizations">("overview");

  const firstName = user?.name?.trim().split(/\s+/)[0] || "there";
  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 18 ? "Good afternoon" : "Good evening";

  const chartData = (metrics.data?.volumeSeries ?? []).map((point) => ({
    date: point.date,
    volume: point.volumeMinor / 100,
  }));
  const range = chartData.length > 1
    ? `${chartData[0].date} – ${chartData[chartData.length - 1].date}`
    : "Last 30 days";
  const connectedProviders = (providers.data ?? []).filter((provider) => provider.status === "connected").length;
  const totalProviders = providers.data?.length ?? 0;
  const totalPayments = (metrics.data?.successful ?? 0) + (metrics.data?.failed ?? 0);

  return (
    <div className="min-w-0 animate-fade-in space-y-6 pb-8">
      <header className="relative isolate overflow-hidden rounded-2xl border border-brand-100 bg-gradient-to-br from-white via-white to-brand-50/80 px-5 py-6 sm:px-7 sm:py-7">
        <div aria-hidden="true" className="pointer-events-none absolute -right-16 -top-28 -z-10 size-72 rounded-full border-[34px] border-brand-100/60" />
        <div aria-hidden="true" className="pointer-events-none absolute -right-5 -top-14 -z-10 size-52 rounded-full border-[20px] border-brand-50/70" />
        <div className="relative flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-brand-700">Nile overview</p>
            <h1 className="font-display text-2xl font-semibold tracking-tight text-slate-950 sm:text-3xl">
              {greeting}, {firstName} <span aria-hidden="true">👋</span>
            </h1>
            <p className="mt-2 max-w-xl text-sm leading-6 text-slate-500">
              Here’s what’s happening across your connected payment providers.
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Link href="/providers/connect/paystack">
              <Button variant="secondary" className="gap-2 border-brand-200 bg-white hover:bg-brand-50">
                <Globe className="size-4" /> Connect provider
              </Button>
            </Link>
            <Link href="/applications/new">
              <Button className="gap-2">
                <Plus className="size-4" /> New application
              </Button>
            </Link>
          </div>
        </div>
      </header>

      <Tabs
        tabs={[
          { value: "overview", label: "Overview" },
          { value: "organizations", label: "Organizations" },
        ]}
        value={tab}
        onValueChange={(next) => setTab(next as "overview" | "organizations")}
        className="w-fit max-w-full"
      />

      {tab === "organizations" ? (
        <OrganizationsPanel currentMerchantId={user?.merchant_id ?? null} />
      ) : (
        <div className="space-y-6">
          {/* At-a-glance metrics */}
          <section aria-label="Payment summary" className="grid gap-3 sm:grid-cols-2 2xl:grid-cols-4">
            <MetricCard
              label="Payment volume"
              value={metrics.loading ? null : metrics.data ? formatAmount(metrics.data.volumeMinor, metrics.data.currency) : "—"}
              note={range}
              icon={<Wallet className="size-5" />}
              tone="brand"
            />
            <MetricCard
              label="Successful payments"
              value={metrics.loading ? null : metrics.data ? formatNumber(metrics.data.successful) : "—"}
              note="Successfully processed"
              icon={<CheckCircle2 className="size-5" />}
              tone="success"
            />
            <MetricCard
              label="Failed payments"
              value={metrics.loading ? null : metrics.data ? formatNumber(metrics.data.failed) : "—"}
              note={metrics.data?.failed ? "Review transactions that need attention" : "No failed payments reported"}
              icon={<CircleAlert className="size-5" />}
              tone={metrics.data?.failed ? "danger" : "slate"}
            />
            <MetricCard
              label="Connected providers"
              value={providers.loading ? null : `${connectedProviders} / ${totalProviders}`}
              note="Providers with active connections"
              icon={<CreditCard className="size-5" />}
              tone="brand"
            />
          </section>

          {/* Volume chart + success summary */}
          <Card className="overflow-hidden border-slate-200/80 shadow-sm shadow-slate-900/[0.025]">
            <div className="grid lg:grid-cols-[minmax(0,1fr)_260px]">
              <section className="min-w-0 p-5 sm:p-6">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="flex size-8 items-center justify-center rounded-lg bg-brand-50 text-brand-700">
                        <Wallet className="size-4" />
                      </span>
                      <h2 className="text-sm font-semibold text-slate-900">Payment volume</h2>
                    </div>
                    <p className="mt-3 text-xs text-slate-500">Total transaction volume across your providers</p>
                    {metrics.loading ? (
                      <div className="mt-2 h-9 w-48 animate-pulse rounded-lg bg-slate-100" />
                    ) : (
                      <p className="mt-1 text-2xl font-semibold tracking-tight text-slate-950 sm:text-3xl">
                        {metrics.data ? formatAmount(metrics.data.volumeMinor, metrics.data.currency) : "—"}
                      </p>
                    )}
                    <p className="mt-1 text-[11px] text-slate-400">{range}</p>
                  </div>
                  <span className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs font-medium text-slate-600">Last 30 days</span>
                </div>

                <div className="mt-5 h-60 min-w-0 sm:h-64">
                  {metrics.loading ? (
                    <div className="h-full animate-pulse rounded-xl bg-slate-50" />
                  ) : chartData.length === 0 ? (
                    <div className="flex h-full flex-col items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 text-center">
                      <Wallet className="size-6 text-slate-300" />
                      <p className="mt-2 text-sm font-medium text-slate-700">Your volume chart will appear here</p>
                      <p className="mt-1 text-xs text-slate-500">Transaction activity will populate this chart.</p>
                    </div>
                  ) : (
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={chartData} margin={{ top: 8, right: 8, bottom: 0, left: -12 }}>
                        <defs>
                          <linearGradient id="nileVolumeFill" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="0%" stopColor={brandGold} stopOpacity={0.18} />
                            <stop offset="95%" stopColor={brandGold} stopOpacity={0.01} />
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 5" stroke="#E9EAF0" vertical={false} />
                        <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{ fill: "#94A3B8", fontSize: 10 }} interval="preserveStartEnd" minTickGap={28} />
                        <YAxis tickLine={false} axisLine={false} tick={{ fill: "#94A3B8", fontSize: 10 }} tickFormatter={(value: number) => `₦${formatCompact(value)}`} width={52} />
                        <Tooltip
                          cursor={{ stroke: "#EFD35B", strokeDasharray: "3 3" }}
                          formatter={(value) => [`₦${Number(value).toLocaleString()}`, "Volume"]}
                          labelStyle={{ color: "#0F172A", fontWeight: 600 }}
                          contentStyle={{ borderRadius: 12, border: "1px solid #F5E191", boxShadow: "0 8px 24px rgba(15,23,42,.08)", fontSize: 12 }}
                        />
                        <Area type="monotone" dataKey="volume" stroke={brandGold} strokeWidth={2.5} fill="url(#nileVolumeFill)" activeDot={{ r: 4, fill: brandGold, stroke: "white", strokeWidth: 2 }} />
                      </AreaChart>
                    </ResponsiveContainer>
                  )}
                </div>
              </section>
              <Outcomes loading={metrics.loading} data={metrics.data} />
            </div>
          </Card>

          {/* Activity and provider overview */}
          <div className="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,1.45fr)_minmax(300px,0.85fr)]">
            <Card className="min-w-0 overflow-hidden border-slate-200/80 shadow-sm shadow-slate-900/[0.025]">
              <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-5 py-4">
                <div>
                  <h2 className="text-sm font-semibold text-slate-900">Recent payments</h2>
                  <p className="mt-1 text-xs text-slate-500">The latest activity across your integrations</p>
                </div>
                <Link href="/payments" className="inline-flex items-center gap-1 text-xs font-semibold text-brand-700 hover:text-brand-800">
                  View all <ArrowRight className="size-3.5" />
                </Link>
              </div>
              {payments.loading ? (
                <div className="px-5 py-3"><SkeletonRows rows={4} /></div>
              ) : (payments.data?.items.length ?? 0) === 0 ? (
                <div className="p-5">
                  <EmptyState icon={<Receipt className="size-5" />} title="No payments yet" description="Payments will appear here when your application receives its first transaction." />
                </div>
              ) : (
                <ul className="divide-y divide-slate-100">
                  {payments.data!.items.map((payment) => (
                    <li key={payment.id}>
                      <Link
                        href={`/payments/${payment.id}`}
                        className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 px-5 py-3.5 outline-none transition-colors hover:bg-brand-50/50 focus-visible:bg-brand-50/50 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand-300 sm:grid-cols-[minmax(0,1fr)_auto_auto]"
                      >
                        <span className="flex min-w-0 items-center gap-3">
                          <span className="flex size-9 shrink-0 items-center justify-center rounded-xl border border-slate-100 bg-slate-50 text-slate-500"><Receipt className="size-4" /></span>
                          <span className="min-w-0">
                            <span className="block truncate text-[13px] font-semibold text-slate-900">{payment.customer.name || "Unknown payer"}</span>
                            <span className="mt-0.5 block truncate text-[11px] text-slate-400">{payment.customer.email || payment.reference}</span>
                          </span>
                        </span>
                        <span className="text-right sm:min-w-[125px] sm:text-left"><AmountDisplay minor={payment.amountMinor} currency={payment.currency} /></span>
                        <span className="col-start-2 row-start-1 sm:col-auto sm:row-auto"><PaymentStatusBadge status={payment.status} /></span>
                        <span className="col-start-1 pl-12 text-[11px] text-slate-400 sm:hidden">{timeAgo(payment.createdAt)}</span>
                        <span className="hidden text-right text-[11px] text-slate-400 sm:block">{timeAgo(payment.createdAt)}</span>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </Card>

            <div className="space-y-5">
              <Card className="overflow-hidden border-brand-100 bg-gradient-to-br from-brand-50/80 via-white to-white p-5 shadow-sm shadow-brand-900/[0.025]">
                <div className="flex items-start gap-3">
                  <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-700 shadow-sm shadow-brand-900/[0.04]"><ShieldCheck className="size-5" /></span>
                  <div className="min-w-0">
                    <h2 className="text-sm font-semibold text-slate-900">Your credentials stay protected</h2>
                    <p className="mt-1.5 text-xs leading-5 text-slate-600">Provider secrets are encrypted and stored in your vault. Full credentials are never exposed through the dashboard.</p>
                    <div className="mt-3 flex items-center gap-1.5 text-[11px] font-medium text-emerald-700"><LockKeyhole className="size-3.5" /> Encrypted storage · Masked secrets</div>
                  </div>
                </div>
              </Card>

              <Card className="border-slate-200/80 p-5 shadow-sm shadow-slate-900/[0.025]">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h2 className="text-sm font-semibold text-slate-900">Payment providers</h2>
                    <p className="mt-1 text-xs text-slate-500">Your connected payment integrations</p>
                  </div>
                  <Link href="/providers" className="text-xs font-semibold text-brand-700 hover:text-brand-800">Manage</Link>
                </div>
                <div className="mt-4 flex items-end justify-between gap-3">
                  <div>
                    <p className="text-2xl font-semibold tracking-tight text-slate-950">{providers.loading ? "—" : `${connectedProviders} / ${totalProviders}`}</p>
                    <p className="mt-1 text-xs text-slate-500">providers connected</p>
                  </div>
                  <span className="rounded-lg bg-brand-50 px-2.5 py-1.5 text-xs font-semibold text-brand-800">{totalProviders ? Math.round((connectedProviders / totalProviders) * 100) : 0}%</span>
                </div>
                <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-slate-100"><div className="h-full rounded-full bg-brand-500 transition-all" style={{ width: `${totalProviders ? (connectedProviders / totalProviders) * 100 : 0}%` }} /></div>
                <div className="mt-4 divide-y divide-slate-100">
                  {providers.loading ? (
                    <div className="space-y-3 py-2">{[0, 1, 2].map((item) => <div key={item} className="h-7 animate-pulse rounded bg-slate-50" />)}</div>
                  ) : (providers.data?.length ?? 0) === 0 ? (
                    <div className="py-4 text-center"><p className="text-xs text-slate-500">No providers connected yet.</p><Link href="/providers/connect/paystack" className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-brand-700">Connect your first provider <ArrowRight className="size-3.5" /></Link></div>
                  ) : (
                    providers.data!.slice(0, 4).map((provider) => (
                      <div key={provider.id} className="flex items-center justify-between gap-3 py-3 first:pt-1 last:pb-0">
                        <div className="flex min-w-0 items-center gap-2.5"><ProviderBadge provider={provider.provider} size="sm" /><span className="truncate text-xs font-medium text-slate-800">{provider.displayName || provider.provider}</span></div>
                        <StatusDot status={provider.status} />
                      </div>
                    ))
                  )}
                </div>
              </Card>
            </div>
          </div>

          {/* Quick actions */}
          <section>
            <div className="mb-3 flex items-center justify-between gap-3">
              <div><h2 className="text-sm font-semibold text-slate-900">Quick actions</h2><p className="mt-1 text-xs text-slate-500">Shortcuts to common tasks</p></div>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <QuickAction href="/providers/connect/paystack" icon={<Globe className="size-4" />} title="Connect a provider" description="Add another payment integration." />
              <QuickAction href="/applications/new" icon={<Plus className="size-4" />} title="Create an application" description="Set up an application to use PayConnect." />
              <QuickAction href="/payments" icon={<Receipt className="size-4" />} title="Review payments" description="Inspect transaction statuses and details." />
            </div>
          </section>
        </div>
      )}
    </div>
  );
}

type Metrics = Awaited<ReturnType<typeof getDashboardMetrics>>;

type MetricTone = "brand" | "success" | "danger" | "slate";

function MetricCard({ label, value, note, icon, tone }: { label: string; value: string | null; note: string; icon: React.ReactNode; tone: MetricTone }) {
  const tones: Record<MetricTone, { icon: string; border: string; background: string }> = {
    brand: { icon: "bg-brand-100 text-brand-800", border: "border-brand-100", background: "from-brand-50/80 to-white" },
    success: { icon: "bg-emerald-100 text-emerald-700", border: "border-emerald-100", background: "from-emerald-50/70 to-white" },
    danger: { icon: "bg-red-100 text-red-700", border: "border-red-100", background: "from-red-50/70 to-white" },
    slate: { icon: "bg-slate-100 text-slate-600", border: "border-slate-200", background: "from-slate-50 to-white" },
  };
  const style = tones[tone];

  return (
    <Card className={`border ${style.border} bg-gradient-to-br ${style.background} p-4 shadow-sm shadow-slate-900/[0.02] sm:p-5`}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-xs font-medium text-slate-500">{label}</p>
          {value === null ? <div className="mt-3 h-8 w-28 animate-pulse rounded-lg bg-slate-100" /> : <p className="mt-2 truncate text-2xl font-semibold tracking-tight text-slate-950">{value}</p>}
        </div>
        <span className={`flex size-10 shrink-0 items-center justify-center rounded-xl ${style.icon}`}>{icon}</span>
      </div>
      <p className="mt-3 line-clamp-2 text-[11px] leading-relaxed text-slate-500">{note}</p>
    </Card>
  );
}

function QuickAction({ href, icon, title, description }: { href: string; icon: React.ReactNode; title: string; description: string }) {
  return (
    <Link href={href} className="group flex min-w-0 items-center gap-3 rounded-xl border border-slate-200/80 bg-white p-4 transition-all hover:border-brand-200 hover:bg-brand-50/50 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-300">
      <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-700 transition-colors group-hover:bg-brand-100">{icon}</span>
      <span className="min-w-0 flex-1"><span className="block text-sm font-semibold text-slate-900">{title}</span><span className="mt-1 block text-xs leading-relaxed text-slate-500">{description}</span></span>
      <ArrowRight className="size-4 shrink-0 text-slate-300 transition-all group-hover:translate-x-0.5 group-hover:text-brand-700" />
    </Link>
  );
}

function Outcomes({ loading, data }: { loading: boolean; data?: Metrics | null }) {
  if (loading || !data) {
    return <aside className="border-t border-slate-100 p-5 sm:p-6 lg:border-l lg:border-t-0"><div className="h-3.5 w-24 animate-pulse rounded bg-slate-100" /><div className="mt-3 h-10 w-28 animate-pulse rounded bg-slate-100" /><div className="mt-6 h-2 animate-pulse rounded-full bg-slate-100" /></aside>;
  }

  const total = data.successful + data.failed;
  const successPct = total === 0 ? 0 : (data.successful / total) * 100;

  return (
    <aside className="flex flex-col border-t border-slate-100 bg-slate-50/40 p-5 sm:p-6 lg:border-l lg:border-t-0">
      <p className="text-xs font-medium text-slate-500">Payment success rate</p>
      <p className="mt-2 text-4xl font-semibold tracking-tight text-slate-950">{data.successRate}%</p>
      <p className="mt-1 text-xs text-slate-400">Across all providers</p>
      <div className="mt-6 flex h-2 overflow-hidden rounded-full bg-slate-200" role="img" aria-label={`${formatNumber(data.successful)} successful, ${formatNumber(data.failed)} failed`}>
        <div className="bg-emerald-500 transition-all" style={{ width: `${successPct}%` }} />
        <div className="bg-red-500 transition-all" style={{ width: `${total === 0 ? 0 : 100 - successPct}%` }} />
      </div>
      <dl className="mt-5 space-y-3 text-xs">
        <div className="flex items-center justify-between"><dt className="flex items-center gap-2 text-slate-600"><span className="size-2 rounded-full bg-emerald-500" />Successful</dt><dd className="font-semibold tabular-nums text-slate-900">{formatNumber(data.successful)}</dd></div>
        <div className="flex items-center justify-between"><dt className="flex items-center gap-2 text-slate-600"><span className="size-2 rounded-full bg-red-500" />Failed</dt><dd className="font-semibold tabular-nums text-slate-900">{formatNumber(data.failed)}</dd></div>
        <div className="flex items-center justify-between border-t border-slate-200 pt-3"><dt className="text-slate-500">Total attempts</dt><dd className="font-semibold tabular-nums text-slate-900">{formatNumber(total)}</dd></div>
      </dl>
      {data.failed > 0 && <Link href="/payments" className="mt-auto inline-flex items-center gap-1 pt-6 text-xs font-semibold text-brand-700 hover:text-brand-800">Review failed payments <ArrowRight className="size-3.5" /></Link>}
    </aside>
  );
}

function StatusDot({ status }: { status: string }) {
  if (status === "connected") return <span className="inline-flex shrink-0 items-center gap-1.5 text-[11px] font-medium text-emerald-700"><span className="size-1.5 rounded-full bg-emerald-500" />Connected</span>;
  if (status === "error") return <span className="inline-flex shrink-0 items-center gap-1.5 text-[11px] font-medium text-red-700"><span className="size-1.5 rounded-full bg-red-600" />Error</span>;
  return <span className="inline-flex shrink-0 items-center gap-1.5 text-[11px] font-medium text-slate-400"><span className="size-1.5 rounded-full border border-slate-300" />Not connected</span>;
}

function OrganizationsPanel({ currentMerchantId }: { currentMerchantId: string | null }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { refresh } = useAuth();
  const memberships = useAsync(listOrganizationMemberships, [], ["memberships"]);
  const [switching, setSwitching] = React.useState<string | null>(null);

  async function onSwitch(merchantId: string) {
    setSwitching(merchantId);
    try {
      await switchOrganization(merchantId);
      await refresh();
      await queryClient.invalidateQueries();
      toast("success", "Organization switched", "Your session now acts as that organization.");
      router.push("/dashboard");
    } catch (error) {
      toast("error", "Could not switch", error instanceof Error ? error.message : "Please try again.");
    } finally {
      setSwitching(null);
    }
  }

  if (memberships.loading) {
    return <Card className="overflow-hidden"><div className="divide-y divide-slate-100">{Array.from({ length: 3 }).map((_, index) => <div key={index} className="h-20 animate-pulse bg-slate-50" />)}</div></Card>;
  }

  const rows = memberships.data ?? [];
  if (rows.length === 0) {
    return <Card className="p-6"><EmptyState icon={<ShieldCheck className="size-5" />} title="No organizations yet" description="You are not a member of any organization. Ask an owner to invite you." /></Card>;
  }

  return (
    <div className="space-y-3">
      <div><h2 className="text-sm font-semibold text-slate-900">Your organizations</h2><p className="mt-1 text-xs text-slate-500">Switch between organizations you have access to.</p></div>
      <Card className="overflow-hidden border-slate-200/80">
        <ul className="divide-y divide-slate-100">
          {rows.map((membership) => {
            const isCurrent = membership.current || membership.merchant_id === currentMerchantId;
            return (
              <li key={membership.merchant_id} className={`flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:p-5 ${isCurrent ? "bg-brand-50/50" : ""}`}>
                <span className={`flex size-11 shrink-0 items-center justify-center rounded-xl ${isCurrent ? "bg-brand-100 text-brand-800" : "bg-slate-100 text-slate-600"}`}><Building2 className="size-5" /></span>
                <div className="min-w-0 flex-1">
                  <p className="flex flex-wrap items-center gap-2 text-sm font-semibold text-slate-900">{membership.name}{isCurrent && <Badge variant="brand">Current</Badge>}</p>
                  <p className="mt-1 truncate font-mono text-[11px] text-slate-400">{membership.merchant_id}</p>
                  <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-slate-500"><span><span className="text-slate-400">Role:</span> {membership.role}</span><span><span className="text-slate-400">Joined:</span> {timeAgo(membership.joined_at)}</span><span><span className="text-slate-400">Access:</span> {membership.owner ? "Owner" : "Developer"}</span></div>
                </div>
                {isCurrent ? (
                  <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-700"><CheckCircle2 className="size-4" /> Current organization</span>
                ) : (
                  <Button variant="secondary" size="sm" disabled={switching !== null} onClick={() => void onSwitch(membership.merchant_id)} className="shrink-0 gap-2"><ArrowRightLeft className="size-3.5" />{switching === membership.merchant_id ? "Switching…" : "Switch organization"}</Button>
                )}
              </li>
            );
          })}
        </ul>
      </Card>
      <p className="flex items-center gap-2 px-1 text-xs text-slate-500"><Users className="size-3.5" /> Your role and permissions may differ between organizations.</p>
    </div>
  );
}
