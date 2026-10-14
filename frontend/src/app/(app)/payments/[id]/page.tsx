"use client";

import * as React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ChevronDown, ChevronUp, Copy, KeyRound, Receipt, UserCheck } from "lucide-react";
import { Button, Card, EnvironmentBadge, PaymentStatusBadge, PageHeader } from "@/components/ui";
import { AmountDisplay, ProviderBadge, TransactionTimeline } from "@/components/ui/part3";
import { useAsync } from "@/hooks/use-async";
import { getPayment, listApiKeys, listApplications, listDevelopers } from "@/lib/api";
import { formatAmount, formatDateTime } from "@/lib/utils";
import { toast } from "@/components/ui/toaster";

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 py-2.5">
      <span className="text-[13px] text-slate-500">{label}</span>
      <span className="text-right text-[13px] font-medium text-slate-900">{children}</span>
    </div>
  );
}

export default function PaymentDetailPage() {
  const params = useParams<{ id: string }>();
  const { data: payment, loading } = useAsync(() => getPayment(params.id), [params.id]);
  const applications = useAsync(listApplications, [], ["applications"]);
  const developers = useAsync(listDevelopers, [], ["developers"]);
  const apiKeys = useAsync(() => listApiKeys(), [], ["api-keys"]);
  const [showRaw, setShowRaw] = React.useState(false);

  // Attribution is read from the payment itself; the supporting lists only turn
  // the stored ids into names. A revoked key or removed developer simply falls
  // back to the id, so the page never claims more than the record says.
  const application = (applications.data ?? []).find((app) => app.id === payment?.applicationId);
  const apiKey = (apiKeys.data ?? []).find((key) => key.id === payment?.apiKeyId);
  const developer = (developers.data ?? []).find((dev) => dev.accessId === payment?.developerAccessId);
  const hasRawResponse = Object.keys(payment?.providerResponse ?? {}).length > 0;

  if (loading) {
    return (
      <div>
        <div className="mb-6 space-y-2">
          <div className="h-5 w-40 animate-pulse rounded bg-slate-100" />
          <div className="h-9 w-48 animate-pulse rounded bg-slate-100" />
        </div>
        <div className="grid gap-4 lg:grid-cols-3">
          <div className="h-72 animate-pulse rounded-[var(--radius-card)] border border-slate-200 bg-white lg:col-span-2" />
          <div className="h-72 animate-pulse rounded-[var(--radius-card)] border border-slate-200 bg-white" />
        </div>
      </div>
    );
  }

  if (!payment) {
    return (
      <div className="rounded-[var(--radius-card)] border border-slate-200 bg-white p-14 text-center">
        <Receipt className="mx-auto size-8 text-slate-300" />
        <h1 className="mt-4 text-lg font-semibold text-slate-900">Payment not found</h1>
        <p className="mt-1 text-sm text-slate-500">
          It may belong to another application or environment.
        </p>
        <Link href="/payments" className="mt-6 inline-block">
          <Button variant="secondary">Back to payments</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="animate-fade-in">
      <PageHeader
        breadcrumb={[{ label: "Payments", href: "/payments" }, { label: payment.id }]}
        title={`Payment #${payment.id}`}
        actions={
          <div className="flex items-center gap-2">
            <EnvironmentBadge environment={payment.environment} />
            <PaymentStatusBadge status={payment.status} />
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                navigator.clipboard.writeText(payment.id);
                toast("info", "Copied", "Payment ID copied to clipboard.");
              }}
            >
              <Copy className="size-3.5" /> Copy ID
            </Button>
          </div>
        }
      />

      <div className="grid gap-4 lg:grid-cols-3">
        {/* Left column */}
        <div className="space-y-4 lg:col-span-2">
          <Card>
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 p-5">
              <div>
                <p className="text-[12.5px] text-slate-400">Amount</p>
                <p className="mt-1 font-mono text-3xl font-semibold tracking-tight text-slate-900">
                  {formatAmount(payment.amountMinor, payment.currency)}
                </p>
                <p className="mt-1 text-[12.5px] text-slate-400">{payment.description || "No description"}</p>
              </div>
              <PaymentStatusBadge status={payment.status} />
            </div>
            <div className="grid gap-x-8 px-5 py-2 sm:grid-cols-2">
              <Row label="Payment ID">
                <code className="font-mono text-[12.5px]">{payment.id}</code>
              </Row>
              <Row label="Reference">
                <code className="font-mono text-[12.5px]">{payment.reference}</code>
              </Row>
              <Row label="Provider">
                <ProviderBadge provider={payment.provider} size="sm" />
              </Row>
              <Row label="Payer reference">
                <code className="font-mono text-[12.5px]">{payment.customer.email || "—"}</code>
              </Row>
              <Row label="Application">
                {application ? (
                  <Link href={`/applications/${application.id}`} className="text-brand-600 hover:underline">
                    {application.name}
                  </Link>
                ) : (
                  <span className="text-slate-400">—</span>
                )}
              </Row>
              <Row label="Created">{formatDateTime(payment.createdAt)}</Row>
              <Row label="Updated">{formatDateTime(payment.updatedAt)}</Row>
              <Row label="Fee">
                {payment.feeMinor != null ? formatAmount(payment.feeMinor, payment.currency) : "—"}
              </Row>
              <Row label="Settlement">
                {payment.status === "successful" ? "T+1 (next business day)" : "—"}
              </Row>
            </div>
          </Card>

          {/* Raw provider response */}
          {hasRawResponse ? (
          <Card>
            <button
              onClick={() => setShowRaw((v) => !v)}
              className="flex w-full items-center justify-between px-5 py-4 text-left"
            >
              <div>
                <h2 className="text-sm font-semibold text-slate-900">Provider response</h2>
                <p className="text-[12px] text-slate-400">Redacted — credential material is stripped before storage.</p>
              </div>
              {showRaw ? <ChevronUp className="size-4 text-slate-400" /> : <ChevronDown className="size-4 text-slate-400" />}
            </button>
            {showRaw ? (
              <div className="px-5 pb-5">
                <pre className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-950 p-4 font-mono text-[12.5px] leading-relaxed text-slate-200">
                  {JSON.stringify(payment.providerResponse, null, 2)}
                </pre>
              </div>
            ) : null}
          </Card>
          ) : null}
        </div>

        {/* Right column */}
        <div className="space-y-4">
          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">Customer</h2>
            </div>
            <div className="flex items-center gap-3 p-5">
              <span className="flex size-10 items-center justify-center rounded-full bg-brand-100 text-[13px] font-semibold text-brand-700">
                {(payment.customer.name || payment.customer.email || "?")
                  .split(" ")
                  .map((w) => w[0])
                  .join("")
                  .slice(0, 2)
                  .toUpperCase()}
              </span>
              <div className="min-w-0">
                <p className="text-[13.5px] font-medium text-slate-900">
                  {payment.customer.name || "Unknown payer"}
                </p>
                <p className="truncate text-[12.5px] text-slate-400">{payment.customer.email || "—"}</p>
                <p className="text-[12.5px] text-slate-400">{payment.customer.phone || ""}</p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">Taken by</h2>
            </div>
            <div className="space-y-3 p-5">
              <div className="flex items-start gap-2.5">
                {payment.developerAccessId ? (
                  <UserCheck className="mt-0.5 size-4 shrink-0 text-emerald-500" />
                ) : (
                  <KeyRound className="mt-0.5 size-4 shrink-0 text-slate-400" />
                )}
                <div className="min-w-0">
                  <p className="text-[13px] font-medium text-slate-900">
                    {apiKey?.label ?? (payment.developerAccessId ? "Developer key" : "General key")}
                  </p>
                  <p className="text-[12px] text-slate-400">
                    {payment.developerAccessId
                      ? `Developer-scoped · ${developer?.name ?? developer?.email ?? payment.developerAccessId}`
                      : "Shared application key"}
                  </p>
                  {apiKey ? (
                    <p className="mt-0.5 font-mono text-[11.5px] text-slate-400">{apiKey.masked}</p>
                  ) : null}
                </div>
              </div>
              {developer ? (
                <Link
                  href={`/developers/${developer.id}`}
                  className="text-[12.5px] text-brand-500 hover:underline"
                >
                  Manage this developer →
                </Link>
              ) : null}
            </div>
          </Card>

          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">Timeline</h2>
            </div>
            <div className="p-5">
              <TransactionTimeline events={payment.timeline} />
            </div>
          </Card>

          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">Security</h2>
            </div>
            <div className="p-5">
              <div className="flex items-start gap-2.5 rounded-xl border border-brand-100 bg-brand-50/60 p-3.5">
                <span className="mt-0.5 text-brand-500">🔒</span>
                <p className="text-[12.5px] leading-relaxed text-brand-900">
                  Provider credentials were used <b>inside the vault</b> to process this payment. No secret material
                  appears in this page, the API response, or your logs.
                </p>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
