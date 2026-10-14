
"use client";

import * as React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import {
  ArrowLeft,
  CheckCircle2,
  Clock3,
  Copy,
  RotateCw,
  ShieldAlert,
  Webhook,
  XCircle,
} from "lucide-react";

import { Badge, Button, Card, PageHeader } from "@/components/ui";
import { useAsync } from "@/hooks/use-async";
import {
  getWebhookDelivery,
  listWebhooks,
  retryWebhookDelivery,
} from "@/lib/api";
import { formatDateTime, timeAgo } from "@/lib/utils";
import { toast } from "@/components/ui/toaster";

const STATUS_META: Record<
  string,
  {
    label: string;
    variant: "success" | "danger" | "warning";
    icon: React.ReactNode;
  }
> = {
  delivered: {
    label: "Delivered",
    variant: "success",
    icon: <CheckCircle2 className="size-4" />,
  },
  failed: {
    label: "Failed",
    variant: "danger",
    icon: <XCircle className="size-4" />,
  },
  retrying: {
    label: "Retrying",
    variant: "warning",
    icon: <RotateCw className="size-4" />,
  },
  pending: {
    label: "Pending",
    variant: "warning",
    icon: <Clock3 className="size-4" />,
  },
};

function Info({
  label,
  value,
  mono = false,
  capitalize = false,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  capitalize?: boolean;
}) {
  return (
    <div className="min-w-0">
      <p className="text-xs text-slate-500">{label}</p>
      <p
        className={`mt-1 break-words text-sm font-medium text-slate-800 ${
          mono ? "font-mono text-xs" : ""
        } ${capitalize ? "capitalize" : ""}`}
      >
        {value}
      </p>
    </div>
  );
}

function CodePanel({
  title,
  subtitle,
  value,
  emptyText,
}: {
  title: string;
  subtitle?: string;
  value: unknown;
  emptyText?: string;
}) {
  const [copied, setCopied] = React.useState(false);

  const content =
    typeof value === "string"
      ? value
      : JSON.stringify(value ?? {}, null, 2);

  async function copyContent() {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);

      toast("success", "Copied", `${title} copied to clipboard.`);

      window.setTimeout(() => setCopied(false), 1400);
    } catch {
      toast(
        "error",
        "Copy failed",
        "Select and copy the content manually."
      );
    }
  }

  return (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-100 px-5 py-4 sm:px-6">
        <div>
          <h2 className="text-sm font-semibold text-slate-900">{title}</h2>
          {subtitle && (
            <p className="mt-1 text-xs leading-relaxed text-slate-500">
              {subtitle}
            </p>
          )}
        </div>

        <Button
          variant="secondary"
          size="sm"
          onClick={() => void copyContent()}
        >
          <Copy className="size-3.5" />
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>

      <div className="p-4 sm:p-5">
        {value === null || value === undefined || value === "" ? (
          <p className="text-sm text-slate-500">
            {emptyText ?? "No response body was recorded."}
          </p>
        ) : (
          <pre className="max-h-[480px] overflow-auto rounded-xl border border-slate-800 bg-slate-950 p-4 font-mono text-xs leading-relaxed text-slate-200">
            {content}
          </pre>
        )}
      </div>
    </Card>
  );
}

export default function WebhookDeliveryPage() {
  const params = useParams<{ id: string }>();

  const {
    data: delivery,
    loading,
    refetch,
  } = useAsync(() => getWebhookDelivery(params.id), [params.id]);

  const endpoints = useAsync(listWebhooks, []);
  const [retrying, setRetrying] = React.useState(false);

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="h-28 animate-pulse rounded-2xl border border-slate-200 bg-white" />
        <div className="h-96 animate-pulse rounded-2xl border border-slate-200 bg-white" />
      </div>
    );
  }

  if (!delivery) {
    return (
      <div className="rounded-2xl border border-slate-200 bg-white p-10 text-center sm:p-14">
        <div className="mx-auto flex size-12 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
          <Webhook className="size-5" />
        </div>

        <h1 className="mt-4 text-lg font-semibold text-slate-900">
          Delivery not found
        </h1>

        <p className="mt-1 text-sm text-slate-500">
          This delivery may have expired or the link may be incorrect.
        </p>

        <Link href="/webhooks" className="mt-6 inline-flex">
          <Button variant="secondary">
            <ArrowLeft className="size-4" />
            Back to webhooks
          </Button>
        </Link>
      </div>
    );
  }

  const endpoint = (endpoints.data ?? []).find(
    (item) => item.id === delivery.endpointId
  );

  const status = STATUS_META[delivery.status] ?? STATUS_META.pending;
  const delivered = delivery.status === "delivered";
  const failed = delivery.status === "failed";

  async function retry() {
    if (retrying || delivered || !delivery) return;

    setRetrying(true);

    try {
      await retryWebhookDelivery(delivery.id);
      await refetch();

      toast(
        "success",
        "Retry queued",
        "The delivery attempt was queued and the record has been refreshed."
      );
    } catch (error) {
      toast(
        "error",
        "Retry failed",
        error instanceof Error ? error.message : "Please try again."
      );
    } finally {
      setRetrying(false);
    }
  }

  const timeline = [
    {
      label: "Delivery created",
      detail: formatDateTime(delivery.createdAt),
      state: "done" as const,
    },
    {
      label: "Request sent",
      detail:
        delivery.attempts > 0
          ? `${delivery.attempts} attempt${
              delivery.attempts === 1 ? "" : "s"
            } recorded`
          : "No attempt recorded yet",
      state: delivery.attempts > 0 ? ("done" as const) : ("waiting" as const),
    },
    {
      label: delivered
        ? "Endpoint acknowledged"
        : failed
          ? "Endpoint rejected request"
          : "Awaiting acknowledgement",
      detail: delivery.deliveredAt
        ? formatDateTime(delivery.deliveredAt)
        : delivery.responseCode !== null
          ? `HTTP ${delivery.responseCode}`
          : "No successful response recorded",
      state: delivered
        ? ("done" as const)
        : failed
          ? ("failed" as const)
          : ("waiting" as const),
    },
  ];

  return (
    <div className="mx-auto w-full max-w-[1440px] animate-fade-in space-y-6">
      <PageHeader
        breadcrumb={[
          { label: "Webhooks", href: "/webhooks" },
          { label: "Delivery details" },
        ]}
        title="Delivery details"
        description={`Inspect the payload, endpoint response, and retry history for ${delivery.event}.`}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={status.variant}>{status.label}</Badge>

            <Button
              variant="secondary"
              size="sm"
              onClick={() => void retry()}
              disabled={retrying || delivered}
            >
              <RotateCw
                className={`size-3.5 ${
                  retrying ? "animate-spin" : ""
                }`}
              />
              {retrying ? "Queueing…" : "Retry delivery"}
            </Button>
          </div>
        }
      />

      {/* Delivery status summary */}
      <Card>
        <div
          className={`border-b p-5 sm:px-6 ${
            delivered
              ? "border-emerald-100 bg-emerald-50/50"
              : failed
                ? "border-red-100 bg-red-50/50"
                : "border-amber-100 bg-amber-50/50"
          }`}
        >
          <div className="flex items-start gap-3">
            <span
              className={`mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl ${
                delivered
                  ? "bg-emerald-100 text-emerald-700"
                  : failed
                    ? "bg-red-100 text-red-700"
                    : "bg-amber-100 text-amber-700"
              }`}
            >
              {status.icon}
            </span>

            <div className="min-w-0 flex-1">
              <h2
                className={`text-sm font-semibold ${
                  delivered
                    ? "text-emerald-950"
                    : failed
                      ? "text-red-950"
                      : "text-amber-950"
                }`}
              >
                {delivered
                  ? "Your endpoint acknowledged this event"
                  : failed
                    ? "The endpoint did not accept this delivery"
                    : "This delivery is still in progress"}
              </h2>

              <p
                className={`mt-1 text-xs leading-relaxed ${
                  delivered
                    ? "text-emerald-800"
                    : failed
                      ? "text-red-800"
                      : "text-amber-800"
                }`}
              >
                {delivery.responseCode !== null
                  ? `Most recent response: HTTP ${delivery.responseCode}.`
                  : "No HTTP response was recorded for the most recent attempt."}

                {delivery.responseBody
                  ? ` ${delivery.responseBody
                      .trim()
                      .replace(/\s+/g, " ")
                      .slice(0, 180)}`
                  : ""}
              </p>
            </div>

            <span className="hidden shrink-0 text-xs text-slate-500 sm:block">
              {timeAgo(delivery.createdAt)}
            </span>
          </div>
        </div>

        <div className="grid gap-4 p-5 sm:grid-cols-2 lg:grid-cols-4 sm:px-6">
          <Info label="Event type" value={delivery.event} mono />

          <Info
            label="HTTP response"
            value={
              delivery.responseCode === null
                ? "No response"
                : `HTTP ${delivery.responseCode}`
            }
          />

          <Info label="Attempts" value={delivery.attempts} />

          <Info
            label="Created"
            value={formatDateTime(delivery.createdAt)}
          />
        </div>
      </Card>

      <div className="grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_320px]">
        <div className="min-w-0 space-y-5">
          <CodePanel
            title="Event payload"
            subtitle="The normalized JSON body sent to your endpoint."
            value={delivery.payload}
            emptyText="No payload was recorded for this delivery."
          />

          <CodePanel
            title="Endpoint response"
            subtitle="The response body from the most recent attempt, useful for debugging rejected deliveries."
            value={delivery.responseBody}
            emptyText={
              delivery.responseCode === null
                ? "This attempt did not receive an HTTP response. Check endpoint reachability, TLS, and server logs."
                : `The endpoint returned HTTP ${delivery.responseCode} with an empty body.`
            }
          />

          {/* Attempt history */}
          <Card>
            <div className="border-b border-slate-100 px-5 py-4 sm:px-6">
              <h2 className="text-sm font-semibold text-slate-900">
                Attempt history
              </h2>
              <p className="mt-1 text-xs text-slate-500">
                Recorded attempt count: {delivery.attempts}
              </p>
            </div>

            {delivery.attempts === 0 ? (
              <p className="p-5 text-sm text-slate-500">
                No delivery attempts have been recorded yet.
              </p>
            ) : (
              <div className="divide-y divide-slate-100">
                {Array.from({ length: delivery.attempts }).map((_, index) => {
                  const latest = index === delivery.attempts - 1;
                  const failedLatest = latest && failed;
                  const pendingLatest = latest && !delivered && !failed;

                  return (
                    <div
                      key={index}
                      className="flex flex-wrap items-center gap-3 px-5 py-4 sm:px-6"
                    >
                      <span
                        className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${
                          failedLatest
                            ? "bg-red-50 text-red-700"
                            : pendingLatest
                              ? "bg-amber-50 text-amber-700"
                              : "bg-emerald-50 text-emerald-700"
                        }`}
                      >
                        {failedLatest ? (
                          <XCircle className="size-4" />
                        ) : pendingLatest ? (
                          <Clock3 className="size-4" />
                        ) : (
                          <CheckCircle2 className="size-4" />
                        )}
                      </span>

                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium text-slate-800">
                          Attempt {index + 1}
                          {latest ? " · Latest" : ""}
                        </p>

                        <p className="mt-1 break-all font-mono text-xs text-slate-500">
                          POST {endpoint?.url ?? "Endpoint URL unavailable"}
                        </p>
                      </div>

                      <div className="text-right">
                        <p
                          className={`text-xs font-semibold ${
                            failedLatest
                              ? "text-red-700"
                              : pendingLatest
                                ? "text-amber-700"
                                : "text-emerald-700"
                          }`}
                        >
                          {failedLatest
                            ? `HTTP ${delivery.responseCode ?? "failed"}`
                            : pendingLatest
                              ? "Pending"
                              : delivery.responseCode === null
                                ? "Sent"
                                : `HTTP ${delivery.responseCode}`}
                        </p>

                        <p className="mt-1 text-xs text-slate-400">
                          {latest
                            ? timeAgo(delivery.createdAt)
                            : "Earlier attempt"}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </div>

        {/* Right sidebar */}
        <aside className="space-y-5">
          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">
                Delivery summary
              </h2>
            </div>

            <div className="space-y-4 p-5">
              <Info
                label="Status"
                value={delivery.status}
                capitalize
              />

              <Info
                label="Endpoint"
                value={endpoint?.url ?? "Unavailable"}
              />

              <Info
                label="Response code"
                value={delivery.responseCode ?? "—"}
              />

              <Info
                label="Delivered at"
                value={
                  delivery.deliveredAt
                    ? formatDateTime(delivery.deliveredAt)
                    : "Not delivered"
                }
              />

              <Info
                label="Delivery ID"
                value={delivery.id}
                mono
              />
            </div>
          </Card>

          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">
                Delivery timeline
              </h2>
            </div>

            <ol className="p-5">
              {timeline.map((step, index) => (
                <li
                  key={step.label}
                  className="relative flex gap-3 pb-6 last:pb-0"
                >
                  {index < timeline.length - 1 && (
                    <span className="absolute left-[15px] top-8 h-[calc(100%-8px)] w-px bg-slate-200" />
                  )}

                  <span
                    className={`relative z-10 flex size-8 shrink-0 items-center justify-center rounded-full ${
                      step.state === "done"
                        ? "bg-emerald-50 text-emerald-700"
                        : step.state === "failed"
                          ? "bg-red-50 text-red-700"
                          : "bg-slate-100 text-slate-400"
                    }`}
                  >
                    {step.state === "done" ? (
                      <CheckCircle2 className="size-4" />
                    ) : step.state === "failed" ? (
                      <XCircle className="size-4" />
                    ) : (
                      <Clock3 className="size-4" />
                    )}
                  </span>

                  <div className="min-w-0 pt-1">
                    <p className="text-sm font-medium text-slate-800">
                      {step.label}
                    </p>
                    <p className="mt-1 break-words text-xs leading-relaxed text-slate-500">
                      {step.detail}
                    </p>
                  </div>
                </li>
              ))}
            </ol>
          </Card>

          <Card>
            <div className="p-5">
              <div className="flex items-start gap-2.5">
                <ShieldAlert className="mt-0.5 size-4 shrink-0 text-slate-500" />

                <div>
                  <h3 className="text-sm font-semibold text-slate-900">
                    Troubleshooting
                  </h3>

                  <p className="mt-1 text-xs leading-relaxed text-slate-500">
                    Confirm your endpoint is publicly reachable, verifies the
                    signing signature, and returns a 2xx response. Failed
                    deliveries may be retried according to your delivery
                    policy.
                  </p>

                  <Link
                    href="/docs/webhooks"
                    className="mt-3 inline-flex items-center gap-1 text-xs font-medium text-brand-600 hover:text-brand-700"
                  >
                    Read webhook docs
                    <ArrowLeft className="size-3 rotate-180" />
                  </Link>
                </div>
              </div>
            </div>
          </Card>
        </aside>
      </div>
    </div>
  );
}
