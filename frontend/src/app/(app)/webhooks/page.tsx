
"use client";

import * as React from "react";
import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  ArrowUpRight,
  CheckCircle2,
  ChevronRight,
  CircleAlert,
  Clock3,
  Globe,
  Plus,
  Search,
  Send,
  ShieldCheck,
  Webhook as WebhookIcon,
} from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Checkbox,
  EmptyState,
  Input,
  Label,
  PageHeader,
  Select,
  Tabs,
} from "@/components/ui";
import { EnvironmentBadge } from "@/components/ui/part3";
import { CopyField, Modal, SecurityNotice } from "@/components/ui/part2";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import {
  createWebhook,
  listApplications,
  listInboundEvents,
  listWebhookDeliveries,
  listWebhooks,
  testWebhookEndpoint,
  type CreatedWebhook,
  type TestWebhookResult,
} from "@/lib/api";
import {
  PROVIDER_META,
  WEBHOOK_EVENTS,
  type WebhookEndpoint,
} from "@/lib/types";
import { timeAgo } from "@/lib/utils";

const STATUS_STYLE: Record<string, string> = {
  delivered: "text-emerald-700 bg-emerald-50 border-emerald-100",
  failed: "text-red-700 bg-red-50 border-red-100",
  pending: "text-amber-700 bg-amber-50 border-amber-100",
  retrying: "text-amber-700 bg-amber-50 border-amber-100",
};

function Metric({
  label,
  value,
  detail,
  icon,
  tone = "slate",
}: {
  label: string;
  value: React.ReactNode;
  detail: string;
  icon: React.ReactNode;
  tone?: "slate" | "green" | "amber" | "red";
}) {
  const tones = {
    slate: "bg-slate-100 text-slate-600",
    green: "bg-emerald-50 text-emerald-700",
    amber: "bg-amber-50 text-amber-700",
    red: "bg-rose-50 text-rose-700",
  };

  return (
    <Card>
      <div className="p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3">
          <p className="text-xs font-medium text-slate-500">{label}</p>
          <span
            className={`flex size-9 items-center justify-center rounded-xl ${tones[tone]}`}
          >
            {icon}
          </span>
        </div>
        <p className="mt-3 text-2xl font-semibold tracking-tight text-slate-900">
          {value}
        </p>
        <p className="mt-1 text-xs text-slate-400">{detail}</p>
      </div>
    </Card>
  );
}

function failureReason(result: TestWebhookResult) {
  if (result.response_code) {
    const body = (result.response_body ?? "")
      .trim()
      .replace(/\s+/g, " ");

    return body
      ? `${result.url} answered ${result.response_code}: ${body.slice(0, 160)}`
      : `${result.url} answered ${result.response_code} with an empty body.`;
  }

  return (
    result.last_error ||
    `We could not reach ${result.url}. Check that the URL is reachable from the internet.`
  );
}

/**
 * The caption under the delivery-success metric. It names the deliveries the
 * percentage was computed from, so the number is never a bare assertion: an em
 * dash reads as "nothing concluded in the window", not "we stopped measuring".
 */
function deliverySuccessDetail(endpoint: WebhookEndpoint | undefined) {
  const window = `last ${endpoint?.deliveryWindowDays ?? 7} days`;

  if (!endpoint || endpoint.deliveries === 0) {
    return `No concluded deliveries · ${window}`;
  }

  return `${endpoint.delivered} of ${endpoint.deliveries} deliveries · ${window}`;
}

export default function WebhooksPage() {
  const queryClient = useQueryClient();
  const [section, setSection] = React.useState("endpoints");

  const { data, loading } = useAsync(
    listWebhooks,
    [],
    ["webhook-endpoints"]
  );

  const deliveries = useAsync(
    listWebhookDeliveries,
    [],
    ["webhook-deliveries"]
  );

  const inbound = useAsync(listInboundEvents, []);
  const apps = useAsync(listApplications, []);

  const [addOpen, setAddOpen] = React.useState(false);
  const [created, setCreated] =
    React.useState<CreatedWebhook | null>(null);

  const [url, setUrl] = React.useState("");
  const [environment, setEnvironment] =
    React.useState<"test" | "live">("test");

  const [events, setEvents] = React.useState<string[]>([]);
  const [applicationId, setApplicationId] = React.useState("");
  const [creating, setCreating] = React.useState(false);
  const [sendingId, setSendingId] = React.useState<string | null>(null);
  const [deliverySearch, setDeliverySearch] = React.useState("");

  const endpoints = data ?? [];

  const primary =
    endpoints.find((webhook) => !webhook.applicationId) ??
    endpoints[0];

  const overrides = endpoints.filter(
    (webhook) => webhook.id !== primary?.id
  );

  const appById = new Map(
    (apps.data ?? []).map((app) => [app.id, app])
  );

  const rows = (deliveries.data ?? []).filter((delivery) =>
    `${delivery.event} ${delivery.status}`
      .toLowerCase()
      .includes(deliverySearch.trim().toLowerCase())
  );

  const deliveredCount = (deliveries.data ?? []).filter(
    (delivery) => delivery.status === "delivered"
  ).length;

  const failedCount = (deliveries.data ?? []).filter(
    (delivery) => delivery.status === "failed"
  ).length;

  function closeAdd() {
    if (creating) return;

    setAddOpen(false);
    setCreated(null);
    setUrl("");
    setEnvironment("test");
    setEvents([]);
    setApplicationId("");
  }

  function toggleEvent(name: string) {
    setEvents((current) =>
      current.includes(name)
        ? current.filter((item) => item !== name)
        : [...current, name]
    );
  }

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();

    const target = url.trim();

    if (!/^https?:\/\/.+/i.test(target)) {
      toast(
        "warning",
        "Enter an absolute URL",
        "Use a complete http:// or https:// URL."
      );
      return;
    }

    setCreating(true);

    try {
      const result = await createWebhook({
        url: target,
        environment,
        events,
        applicationId: applicationId || undefined,
      });

      setCreated(result);

      await queryClient.invalidateQueries({
        queryKey: ["webhook-endpoints"],
      });

      toast(
        "success",
        "Endpoint saved",
        "Copy the signing secret before closing this dialog."
      );
    } catch (error) {
      toast(
        "error",
        "Could not save the endpoint",
        error instanceof Error ? error.message : "Please try again."
      );
    } finally {
      setCreating(false);
    }
  }

  async function sendTest(id: string) {
    setSendingId(id);

    try {
      const result = await testWebhookEndpoint(id);
      const ok = result.status === "delivered";

      toast(
        ok ? "success" : "error",
        ok ? "Test delivery delivered" : "Test delivery failed",
        ok
          ? `${result.url} answered ${result.response_code ?? "2xx"}.`
          : failureReason(result)
      );

      await queryClient.invalidateQueries({
        queryKey: ["webhook-deliveries"],
      });
    } catch (error) {
      toast(
        "error",
        "The test could not be sent",
        error instanceof Error ? error.message : "Please try again."
      );
    } finally {
      setSendingId(null);
    }
  }

  return (
    <div className="mx-auto w-full max-w-[1440px] animate-fade-in space-y-6">
      <PageHeader
        title="Webhooks"
        description="Send payment events to your systems, inspect delivery attempts, and troubleshoot receiver responses."
        actions={
          <Button size="sm" onClick={() => setAddOpen(true)}>
            <Plus className="size-4" />
            Add endpoint
          </Button>
        }
      />

      {/* Overview metrics */}
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Metric
          label="Endpoints"
          value={endpoints.length}
          detail="Registered destinations"
          icon={<WebhookIcon className="size-4" />}
        />

        <Metric
          label="Delivery success"
          value={
            primary?.successRate == null
              ? "—"
              : `${primary.successRate}%`
          }
          detail={deliverySuccessDetail(primary)}
          icon={<CheckCircle2 className="size-4" />}
          tone="green"
        />

        <Metric
          label="Delivered"
          value={deliveredCount}
          detail="Visible delivery records"
          icon={<Activity className="size-4" />}
          tone="green"
        />

        <Metric
          label="Failed"
          value={failedCount}
          detail="May need investigation"
          icon={<CircleAlert className="size-4" />}
          tone="red"
        />
      </div>

      <Tabs
        tabs={[
          { value: "endpoints", label: "Endpoints", count: endpoints.length },
          { value: "deliveries", label: "Delivery history", count: (deliveries.data ?? []).length },
          { value: "incoming", label: "Incoming events", count: (inbound.data ?? []).length },
        ]}
        value={section}
        onValueChange={setSection}
        className="w-fit max-w-full"
      />

      {section === "endpoints" && (
      <div className="space-y-6">
      {/* Main endpoint */}
      {loading ? (
        <div className="h-52 animate-pulse rounded-2xl border border-slate-200 bg-white" />
      ) : !primary ? (
        <Card>
          <EmptyState
            icon={<WebhookIcon className="size-5" />}
            title="No receiving endpoint"
            description="Register one URL to receive unified events from your applications."
            action={
              <Button onClick={() => setAddOpen(true)}>
                <Plus className="size-4" />
                Add endpoint
              </Button>
            }
          />
        </Card>
      ) : (
        <Card>
          <div className="flex flex-col gap-4 border-b border-slate-100 px-5 py-5 sm:flex-row sm:items-start sm:justify-between sm:px-6">
            <div className="flex items-start gap-3">
              <div className="flex size-10 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
                <Globe className="size-5" />
              </div>

              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="text-sm font-semibold text-slate-900">
                    Default receiving endpoint
                  </h2>

                  <EnvironmentBadge environment={primary.environment} />

                  <Badge
                    variant={
                      primary.status === "active" ? "success" : "neutral"
                    }
                  >
                    {primary.status === "active" ? "Active" : "Paused"}
                  </Badge>
                </div>

                <p className="mt-1 text-xs text-slate-500">
                  Used by every application unless an override is configured.
                </p>
              </div>
            </div>

            <div className="flex gap-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={sendingId !== null}
                onClick={() => void sendTest(primary.id)}
              >
                <Send
                  className={`size-3.5 ${
                    sendingId === primary.id ? "animate-pulse" : ""
                  }`}
                />
                {sendingId === primary.id ? "Sending…" : "Send test"}
              </Button>

              <Link href="/docs/webhooks">
                <Button variant="ghost" size="sm">
                  Docs
                  <ArrowUpRight className="size-3.5" />
                </Button>
              </Link>
            </div>
          </div>

          <div className="p-5 sm:p-6">
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-3 sm:flex sm:items-center sm:justify-between sm:gap-4">
              <code className="block min-w-0 break-all font-mono text-xs text-slate-700">
                {primary.url}
              </code>

              <span className="mt-2 inline-flex shrink-0 items-center gap-1.5 text-xs text-slate-500 sm:mt-0">
                <ShieldCheck className="size-3.5 text-emerald-600" />
                Signed requests
              </span>
            </div>

            <div className="mt-4 grid gap-4 sm:grid-cols-3">
              <div>
                <p className="text-xs text-slate-500">Application scope</p>
                <p className="mt-1 text-sm font-medium text-slate-800">
                  {primary.applicationId
                    ? appById.get(primary.applicationId)?.name ??
                      "One application"
                    : "All applications"}
                </p>
              </div>

              <div>
                <p className="text-xs text-slate-500">Signing secret</p>
                <p className="mt-1 font-mono text-sm text-slate-800">
                  {primary.secretMasked}
                </p>
              </div>

              <div>
                <p className="text-xs text-slate-500">Created</p>
                <p className="mt-1 text-sm font-medium text-slate-800">
                  {timeAgo(primary.createdAt)}
                </p>
              </div>
            </div>

            <div className="mt-4 flex flex-wrap gap-1.5 border-t border-slate-100 pt-4">
              {primary.events.length === 0 ? (
                <span className="rounded-md border border-slate-200 bg-white px-2.5 py-1 font-mono text-[11px] text-slate-600">
                  All events
                </span>
              ) : (
                primary.events.map((event) => (
                  <span
                    key={event}
                    className="rounded-md border border-slate-200 bg-white px-2.5 py-1 font-mono text-[11px] text-slate-600"
                  >
                    {event}
                  </span>
                ))
              )}
            </div>
          </div>
        </Card>
      )}

      {/* Application-specific overrides */}
      {overrides.length > 0 && (
        <Card>
          <div className="border-b border-slate-100 px-5 py-4 sm:px-6">
            <h2 className="text-sm font-semibold text-slate-900">
              Application-specific overrides
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Advanced destinations that replace the default for one application.
            </p>
          </div>

          <div className="divide-y divide-slate-100">
            {overrides.map((webhook) => (
              <div
                key={webhook.id}
                className="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6"
              >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="text-sm font-medium text-slate-900">
                      {webhook.applicationId
                        ? appById.get(webhook.applicationId)?.name ??
                          "Unknown application"
                        : "All applications"}
                    </p>

                    <EnvironmentBadge environment={webhook.environment} />

                    <Badge
                      variant={
                        webhook.status === "active" ? "success" : "neutral"
                      }
                    >
                      {webhook.status}
                    </Badge>
                  </div>

                  <code className="mt-1 block break-all font-mono text-xs text-slate-500">
                    {webhook.url}
                  </code>
                </div>

                <Button
                  variant="secondary"
                  size="sm"
                  disabled={sendingId !== null}
                  onClick={() => void sendTest(webhook.id)}
                >
                  <Send className="size-3.5" />
                  {sendingId === webhook.id ? "Sending…" : "Send test"}
                </Button>
              </div>
            ))}
          </div>
        </Card>
      )}
      </div>
      )}

      {/* Recent deliveries */}
      {section === "deliveries" && (
      <Card>
        <div className="flex flex-col gap-3 border-b border-slate-100 px-5 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6">
          <div>
            <h2 className="text-sm font-semibold text-slate-900">
              Recent deliveries
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Open a delivery to inspect its payload, response, and attempts.
            </p>
          </div>

          <div className="relative sm:w-60">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />

            <input
              value={deliverySearch}
              onChange={(e) => setDeliverySearch(e.target.value)}
              placeholder="Filter event or status…"
              aria-label="Filter deliveries"
              className="h-9 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-3 text-xs outline-none focus:border-brand-300 focus:ring-2 focus:ring-brand-100"
            />
          </div>
        </div>

        {deliveries.loading ? (
          <div className="space-y-3 p-5">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="h-12 animate-pulse rounded-lg bg-slate-100"
              />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <p className="px-5 py-10 text-center text-sm text-slate-500">
            No matching deliveries yet. Send a test to create a record.
          </p>
        ) : (
          <div className="divide-y divide-slate-100">
            {rows.map((delivery) => (
              <Link
                key={delivery.id}
                href={`/webhooks/${delivery.id}`}
                className="flex flex-col gap-3 px-5 py-4 transition-colors hover:bg-slate-50/70 sm:flex-row sm:items-center sm:justify-between sm:px-6"
              >
                <div className="flex min-w-0 items-start gap-3">
                  <span
                    className={`mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg border ${
                      STATUS_STYLE[delivery.status] ??
                      "border-slate-200 bg-slate-50 text-slate-600"
                    }`}
                  >
                    {delivery.status === "delivered" ? (
                      <CheckCircle2 className="size-4" />
                    ) : delivery.status === "failed" ? (
                      <CircleAlert className="size-4" />
                    ) : (
                      <Clock3 className="size-4" />
                    )}
                  </span>

                  <div className="min-w-0">
                    <p className="break-all font-mono text-xs font-medium text-slate-800">
                      {delivery.event}
                    </p>
                    <p className="mt-1 text-xs text-slate-400">
                      {timeAgo(delivery.createdAt)}
                    </p>
                  </div>
                </div>

                <div className="flex items-center justify-between gap-4 pl-11 sm:pl-0">
                  <span className="text-xs text-slate-500">
                    {delivery.attempts} attempt
                    {delivery.attempts === 1 ? "" : "s"} · HTTP{" "}
                    {delivery.responseCode ?? "—"}
                  </span>

                  <span className="flex items-center gap-1 text-xs font-medium capitalize text-brand-600">
                    {delivery.status}
                    <ChevronRight className="size-3.5" />
                  </span>
                </div>
              </Link>
            ))}
          </div>
        )}
      </Card>
      )}

      {/* Provider notification guidance */}
      {section === "endpoints" && (
      <Card>
        <div className="grid gap-4 p-5 sm:grid-cols-[1fr_auto] sm:items-center sm:px-6">
          <div className="flex items-start gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
              <ArrowUpRight className="size-4" />
            </span>

            <div>
              <h2 className="text-sm font-semibold text-slate-900">
                Provider notifications are configured separately
              </h2>
              <p className="mt-1 max-w-2xl text-xs leading-relaxed text-slate-500">
                Paystack, Flutterwave and Moniepoint each have their own inbound
                URL. Configure those URLs in the provider dashboard.
              </p>
            </div>
          </div>

          <Link href="/providers">
            <Button variant="secondary" size="sm">
              Manage providers
              <ArrowUpRight className="size-3.5" />
            </Button>
          </Link>
        </div>
      </Card>
      )}

      {/* Raw inbound provider events */}
      {section === "incoming" && (
      <Card>
        <div className="border-b border-slate-100 px-5 py-4 sm:px-6">
          <div className="flex items-center gap-2">
            <Activity className="size-4 text-slate-500" />
            <h2 className="text-sm font-semibold text-slate-900">
              Received from providers
            </h2>
          </div>
          <p className="mt-1 text-xs text-slate-500">
            Raw provider payloads before normalization.
          </p>
        </div>

        {inbound.loading ? (
          <div className="space-y-3 p-5">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="h-10 animate-pulse rounded-lg bg-slate-100"
              />
            ))}
          </div>
        ) : (inbound.data ?? []).length === 0 ? (
          <p className="px-5 py-9 text-center text-sm text-slate-500">
            No provider events received yet.
          </p>
        ) : (
          <div className="divide-y divide-slate-100">
            {(inbound.data ?? []).map((event) => (
              <details key={event.id} className="group px-5 py-4">
                <summary className="flex cursor-pointer list-none items-center justify-between gap-4">
                  <div className="min-w-0">
                    <p className="break-all font-mono text-xs font-medium text-slate-800">
                      {event.eventType}
                    </p>
                    <p className="mt-1 text-xs text-slate-400">
                      {PROVIDER_META[event.provider]?.name ?? event.provider} ·{" "}
                      {timeAgo(event.createdAt)}
                    </p>
                  </div>

                  <span className="shrink-0 text-xs text-brand-600 group-open:hidden">
                    View payload
                  </span>
                  <span className="hidden shrink-0 text-xs text-slate-400 group-open:inline">
                    Hide payload
                  </span>
                </summary>

                <pre className="mt-3 max-h-96 overflow-auto rounded-xl border border-slate-800 bg-slate-950 p-4 font-mono text-xs leading-relaxed text-slate-200">
                  {JSON.stringify(event.payload ?? {}, null, 2)}
                </pre>
              </details>
            ))}
          </div>
        )}
      </Card>
      )}

      {/* Add endpoint modal */}
      <Modal
        open={addOpen}
        onClose={closeAdd}
        title={created ? "Endpoint created" : "Add receiving endpoint"}
        description={
          created
            ? "This signing secret is shown once. Copy it before closing."
            : "Choose the destination, environment, and events this endpoint should receive."
        }
        footer={
          created ? (
            <Button onClick={closeAdd}>Done</Button>
          ) : (
            <>
              <Button
                variant="secondary"
                onClick={closeAdd}
                disabled={creating}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                form="add-endpoint"
                disabled={creating || !url.trim()}
              >
                {creating ? "Saving…" : "Save endpoint"}
              </Button>
            </>
          )
        }
      >
        {created ? (
          <div className="space-y-4">
            <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-3 text-xs leading-relaxed text-emerald-900">
              Save this secret in your server-side environment. The full value
              will not be shown again.
            </div>

            <CopyField
              label="Signing secret"
              value={created.signingSecret}
            />

            <div className="rounded-xl border border-slate-200 p-4">
              <p className="text-xs leading-relaxed text-slate-600">
                Verify the HMAC-SHA256 signature before trusting a webhook.
                PayConnect sends <code>X-PayConnect-Signature</code>,{" "}
                <code>X-PayConnect-Event</code>, and{" "}
                <code>X-PayConnect-Delivery-ID</code>.
              </p>

              <Button
                variant="secondary"
                size="sm"
                className="mt-3"
                disabled={sendingId !== null}
                onClick={() => void sendTest(created.endpoint.id)}
              >
                <Send className="size-3.5" />
                {sendingId === created.endpoint.id
                  ? "Sending…"
                  : "Send test delivery"}
              </Button>
            </div>
          </div>
        ) : (
          <form
            id="add-endpoint"
            onSubmit={onCreate}
            className="space-y-5"
          >
            <div className="space-y-1.5">
              <Label htmlFor="endpoint-url">Endpoint URL</Label>
              <Input
                id="endpoint-url"
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://yourapp.com/hooks/payconnect"
                autoComplete="off"
                spellCheck={false}
                required
              />
              <p className="text-xs leading-relaxed text-slate-400">
                Use a complete URL. Production endpoints should use HTTPS and
                be publicly reachable.
              </p>
            </div>

            <fieldset>
              <legend className="text-sm font-medium text-slate-700">
                Environment
              </legend>

              <div className="mt-2 grid gap-3 sm:grid-cols-2">
                {(["test", "live"] as const).map((env) => (
                  <label
                    key={env}
                    className={`cursor-pointer rounded-xl border p-4 transition-colors ${
                      environment === env
                        ? "border-brand-300 bg-brand-50/50 ring-1 ring-brand-100"
                        : "border-slate-200 hover:border-slate-300"
                    }`}
                  >
                    <span className="flex items-center gap-2">
                      <input
                        type="radio"
                        name="endpoint-environment"
                        checked={environment === env}
                        onChange={() => setEnvironment(env)}
                        className="size-4 accent-brand-600"
                      />
                      <span className="text-sm font-semibold text-slate-900">
                        {env === "live" ? "Production" : "Test"}
                      </span>
                    </span>

                    <span className="mt-2 block pl-6 text-xs leading-relaxed text-slate-500">
                      {env === "live"
                        ? "Receives real-payment events."
                        : "Receives test-payment events only."}
                    </span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset>
              <div className="flex items-center justify-between gap-3">
                <legend className="text-sm font-medium text-slate-700">
                  Events to receive
                </legend>
                <span className="text-xs text-slate-400">
                  {events.length === 0
                    ? "All events"
                    : `${events.length} selected`}
                </span>
              </div>

              <p className="mt-1 text-xs text-slate-400">
                Leave all unchecked to receive every supported event.
              </p>

              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                {WEBHOOK_EVENTS.map((event) => (
                  <label
                    key={event}
                    className={`flex min-w-0 cursor-pointer items-center gap-2.5 rounded-lg border px-3 py-2.5 transition-colors ${
                      events.includes(event)
                        ? "border-brand-200 bg-brand-50/40"
                        : "border-slate-200 hover:border-slate-300"
                    }`}
                  >
                    <Checkbox
                      checked={events.includes(event)}
                      onCheckedChange={() => toggleEvent(event)}
                    />
                    <span className="truncate font-mono text-xs text-slate-700">
                      {event}
                    </span>
                  </label>
                ))}
              </div>
            </fieldset>

            <div className="space-y-1.5">
              <Label htmlFor="endpoint-application">
                Application scope{" "}
                <span className="font-normal text-slate-400">(optional)</span>
              </Label>

              <Select
                id="endpoint-application"
                value={applicationId}
                onChange={(e) => setApplicationId(e.target.value)}
              >
                <option value="">All applications</option>
                {(apps.data ?? []).map((app) => (
                  <option key={app.id} value={app.id}>
                    {app.name} only
                  </option>
                ))}
              </Select>

              <p className="text-xs text-slate-400">
                Use an override only when this application needs a different
                destination.
              </p>
            </div>

            <SecurityNotice>
              The signing secret is displayed once after creation. Store it
              securely and verify every incoming signature.
            </SecurityNotice>
          </form>
        )}
      </Modal>
    </div>
  );
}
