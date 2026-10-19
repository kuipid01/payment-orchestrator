"use client";

import * as React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import {
  Check,
  Copy,
  PanelsTopLeft,
  Pause,
  Play,
  Trash2,
} from "lucide-react";
import {
  Badge,
  Button,
  Card,
  EnvironmentBadge,
  PageHeader,
  Tabs,
} from "@/components/ui";
import {
  AmountDisplay,
  PaymentStatusBadge,
} from "@/components/ui/part3";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  MaskedSecret,
  Modal,
} from "@/components/ui/part2";
import { useAsync } from "@/hooks/use-async";
import {
  deleteApplication,
  createApplicationApiKey,
  getApplication,
  listApiKeys,
  permanentlyDeleteRevokedApiKey,
  revokeApiKey,
  revealApiKey,
  listPayments,
  listWebhookDeliveries,
  listApiLogs,
} from "@/lib/api";
import { formatNumber, timeAgo } from "@/lib/utils";
import { toast } from "@/components/ui/toaster";
import { useAuth } from "@/components/providers/auth-provider";

const TABS = [
  { value: "overview", label: "Overview" },
  { value: "keys", label: "API Keys" },
  { value: "payments", label: "Payments" },
  { value: "webhooks", label: "Webhooks" },
  { value: "logs", label: "Logs" },
  { value: "settings", label: "Settings" },
];

export default function ApplicationDetailPage() {
  const params = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const canManageApplications = user?.role === "Owner" || user?.role === "Admin";

  const app = useAsync(
    () => getApplication(params.id),
    [params.id]
  );

  const keys = useAsync(
    () => listApiKeys(params.id),
    [params.id],
    ["api-keys", params.id]
  );

  const payments = useAsync(
    () =>
      listPayments({
        applicationId: params.id,
        limit: 25,
      }),
    [params.id],
    ["payments", "application", params.id]
  );

  const logs = useAsync(listApiLogs, []);
  const deliveries = useAsync(listWebhookDeliveries, []);

  const [tab, setTab] = React.useState("overview");
  const [confirmOpen, setConfirmOpen] = React.useState(false);

  const [keyToRevoke, setKeyToRevoke] = React.useState<{
    id: string;
    label: string;
  } | null>(null);
  const [keyToDelete, setKeyToDelete] = React.useState<{
    id: string;
    label: string;
  } | null>(null);

  const [createdKey, setCreatedKey] = React.useState<{
    secret: string;
    environment: "test" | "live";
  } | null>(null);

  const [copied, setCopied] = React.useState(false);

  const remove = useMutation({
    mutationFn: () => deleteApplication(params.id),

    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["applications"],
      });

      window.location.assign("/applications");
    },

    onError: (error) => {
      toast(
        "error",
        "Delete failed",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    },
  });

  const createKey = useMutation({
    mutationFn: (environment: "test" | "live") =>
      createApplicationApiKey({
        applicationId: params.id,
        environment,
        name: `${
          app.data?.name ?? "Application"
        } ${
          environment === "live" ? "Production" : "Test"
        } API key`,
      }),

    onSuccess: async (result) => {
      setCreatedKey({
        secret: result.secret,
        environment: result.key.environment,
      });

      toast(
        "success",
        "API key created",
        `Your ${
          result.key.environment === "live"
            ? "production"
            : "test"
        } key is ready.`
      );

      await queryClient.invalidateQueries({
        queryKey: ["api-keys"],
      });
    },

    onError: (error) => {
      toast(
        "error",
        "API key creation failed",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    },
  });

  const revokeKey = useMutation({
    mutationFn: (keyId: string) => revokeApiKey(keyId),

    onSuccess: async () => {
      const label = keyToRevoke?.label ?? "API key";

      setKeyToRevoke(null);

      toast(
        "success",
        "API key revoked",
        `${label} can no longer authenticate requests.`
      );

      await queryClient.invalidateQueries({
        queryKey: ["api-keys"],
      });
    },

    onError: (error) => {
      toast(
        "error",
        "Could not revoke API key",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    },
  });

  const deleteRevokedKey = useMutation({
    mutationFn: (keyId: string) => permanentlyDeleteRevokedApiKey(keyId),
    onSuccess: async () => {
      const label = keyToDelete?.label ?? "API key";
      setKeyToDelete(null);
      toast("success", "Revoked key deleted", `${label} was permanently deleted.`);
      await queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    },
    onError: (error) => {
      toast(
        "error",
        "Could not delete revoked key",
        error instanceof Error ? error.message : "Please try again.",
      );
    },
  });

  const appKeys = (keys.data ?? []).filter(
    (key) => key.applicationId === params.id
  );

  const appPayments = payments.data?.items ?? [];

  if (app.loading) {
    return (
      <div className="h-96 animate-pulse rounded-card border border-slate-200 bg-white" />
    );
  }

  if (!app.data) {
    return (
      <div className="rounded-2xl border border-slate-200 bg-white p-10 text-center shadow-sm sm:p-14">
        <PanelsTopLeft className="mx-auto size-8 text-slate-300" />

        <h1 className="mt-4 text-lg font-semibold text-slate-900">
          Application not found
        </h1>

        <Link href="/applications" className="mt-6 inline-block">
          <Button variant="secondary">
            Back to applications
          </Button>
        </Link>
      </div>
    );
  }

  const application = app.data;

  function onDelete() {
    setConfirmOpen(true);
  }

  async function copyCreatedKey() {
    if (!createdKey) return;

    try {
      await navigator.clipboard.writeText(createdKey.secret);
      setCopied(true);

      setTimeout(() => setCopied(false), 1800);
    } catch {
      toast(
        "error",
        "Copy failed",
        "Select and copy the key manually."
      );
    }
  }

  return (
    <div className="animate-fade-in space-y-6">
      <PageHeader
        breadcrumb={[
          { label: "Applications", href: "/applications" },
          { label: application.name },
        ]}
        title={application.name}
        description={application.description}
        actions={canManageApplications ? (
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm">
              {application.status === "active" ? (
                <Pause className="size-3.5" />
              ) : (
                <Play className="size-3.5" />
              )}

              {application.status === "active"
                ? "Pause"
                : "Resume"}
            </Button>
          </div>
        ) : null}
      />

      <Tabs
        tabs={canManageApplications ? TABS : TABS.filter((item) => item.value !== "settings")}
        value={tab}
        onValueChange={setTab}
        className="mb-4 w-fit max-w-full"
      />

      {tab === "overview" && (
        <div className="grid gap-5 md:grid-cols-3">
          <Card className="md:col-span-2">
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">
                Usage overview
              </h2>
              <p className="mt-1 text-xs text-slate-500">
                Lifetime application activity
              </p>
            </div>

            <dl className="grid grid-cols-2 gap-5 p-5 sm:grid-cols-4">
              {[
                [
                  "API requests",
                  formatNumber(application.stats.apiRequests),
                ],
                [
                  "Successful payments",
                  formatNumber(
                    application.stats.successfulPayments
                  ),
                ],
                [
                  "Failed payments",
                  formatNumber(application.stats.failedPayments),
                ],
                [
                  "API errors",
                  formatNumber(application.stats.apiErrors),
                ],
              ].map(([label, value]) => (
                <div key={label} className="min-w-0">
                  <dt className="text-[12px] text-slate-400">
                    {label}
                  </dt>

                  <dd className="mt-1 break-words font-mono text-xl font-semibold text-slate-900">
                    {value}
                  </dd>
                </div>
              ))}
            </dl>

            <div className="border-t border-slate-100 px-5 py-4 text-[12.5px] text-slate-500">
              Created {timeAgo(application.createdAt)} · Last used{" "}
              {application.lastUsedAt
                ? timeAgo(application.lastUsedAt)
                : "never"}
            </div>
          </Card>

          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">
                Developer access
              </h2>
              <p className="mt-1 text-xs text-slate-500">
                Manage access to this application
              </p>
            </div>

            <div className="p-5">
              <div className="flex size-10 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
                <PanelsTopLeft className="size-5" />
              </div>

              <p className="mt-3 text-[13px] leading-relaxed text-slate-600">
                Developers can receive scoped access to application
                resources without seeing provider credentials.
              </p>

              <Link
                href="/developers"
                className="mt-4 inline-flex items-center text-[12.5px] font-medium text-brand-600 hover:underline"
              >
                Manage developer access →
              </Link>
            </div>
          </Card>
        </div>
      )}

      {tab === "keys" && (
        <Card>
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-100 px-5 py-4">
            <div>
              <h2 className="text-sm font-semibold text-slate-900">
                Environment API keys
              </h2>

              <p className="mt-1 text-[12px] text-slate-500">
                Create test and production keys independently for
                this application.
              </p>
            </div>

            {canManageApplications && <div className="flex flex-wrap gap-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={createKey.isPending}
                onClick={() => createKey.mutate("test")}
              >
                {createKey.isPending
                  ? "Creating…"
                  : "Create test key"}
              </Button>

              <Button
                size="sm"
                disabled={createKey.isPending}
                onClick={() => createKey.mutate("live")}
              >
                {createKey.isPending
                  ? "Creating…"
                  : "Create production key"}
              </Button>
            </div>}
          </div>

          {createdKey && (
            <div className="border-b border-amber-200 bg-amber-50 px-5 py-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <p className="text-[12px] font-semibold text-amber-900">
                    Copy this{" "}
                    {createdKey.environment === "live"
                      ? "production"
                      : "test"}{" "}
                    key now — it will not be shown again.
                  </p>

                  <code className="mt-2 block break-all font-mono text-[12px] text-amber-950">
                    {createdKey.secret}
                  </code>
                </div>

                <div className="flex shrink-0 gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={copyCreatedKey}
                  >
                    {copied ? (
                      <Check className="size-3.5" />
                    ) : (
                      <Copy className="size-3.5" />
                    )}
                    {copied ? "Copied" : "Copy"}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setCreatedKey(null)}
                  >
                    Dismiss
                  </Button>
                </div>
              </div>
            </div>
          )}

          <div className="divide-y divide-slate-100">
            {appKeys.map((key) => (
              <div
                key={key.id}
                className="flex flex-col gap-4 px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="min-w-0">
                  <p className="break-words text-[13.5px] font-medium text-slate-900">
                    {key.label}
                  </p>

                  <p className="mt-1 text-[12px] text-slate-400">
                    Created {timeAgo(key.createdAt)} · Last used{" "}
                    {key.lastUsedAt
                      ? timeAgo(key.lastUsedAt)
                      : "never"}
                  </p>

                  {key.status !== "active" && (
                    <p className="mt-1 text-[11px] font-medium capitalize text-red-600">
                      {key.status}
                    </p>
                  )}
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  <EnvironmentBadge
                    environment={key.environment}
                  />

                  <Badge
                    variant={
                      key.scope === "developer"
                        ? "brand"
                        : "neutral"
                    }
                  >
                    {key.scope === "developer"
                      ? "Developer"
                      : "General"}
                  </Badge>

                  <MaskedSecret
                    value={key.masked}
                    revealable={
                      key.revealable && key.status === "active"
                    }
                    onReveal={
                      key.revealable && key.status === "active"
                        ? () => revealApiKey(key.id)
                        : undefined
                    }
                  />

                  {canManageApplications && key.status === "active" && (
                    <Button
                      variant="danger"
                      size="icon"
                      title="Revoke key"
                      aria-label={`Revoke ${key.label}`}
                      onClick={() =>
                        setKeyToRevoke({
                          id: key.id,
                          label: key.label,
                        })
                      }
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  )}
                  {canManageApplications && key.status === "revoked" && (
                    <Button
                      variant="danger"
                      size="icon"
                      title="Delete revoked key"
                      aria-label={`Delete revoked key ${key.label}`}
                      onClick={() =>
                        setKeyToDelete({ id: key.id, label: key.label })
                      }
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  )}
                </div>
              </div>
            ))}

            {appKeys.length === 0 && (
              <div className="p-8 text-center">
                <div className="mx-auto flex size-10 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
                  <Copy className="size-4" />
                </div>

                <p className="mt-3 text-sm font-medium text-slate-800">
                  No API keys yet
                </p>

                <p className="mt-1 text-xs text-slate-500">
                  Create a test key to start integrating this
                  application.
                </p>
              </div>
            )}
          </div>

          {canManageApplications && <div className="border-t border-slate-100 p-4 text-center">
            <Link
              href="/api-keys"
              className="text-[12.5px] font-medium text-brand-600 hover:underline"
            >
              Manage all API keys →
            </Link>
          </div>}
        </Card>
      )}

      {tab === "payments" && (
        <Card>
          <div className="border-b border-slate-100 px-5 py-4">
            <h2 className="text-sm font-semibold text-slate-900">
              Recent payments
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Payments associated with this application
            </p>
          </div>

          <div className="divide-y divide-slate-100">
            {appPayments.slice(0, 6).map((payment) => (
              <Link
                key={payment.id}
                href={`/payments/${payment.id}`}
                className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 transition-colors hover:bg-slate-50/70"
              >
                <div className="min-w-0">
                  <p className="break-all font-mono text-[12.5px] text-slate-900">
                    {payment.id}
                  </p>
                  <p className="mt-1 text-[11.5px] text-slate-500">
                    {payment.customer.email}
                  </p>
                </div>

                <div className="flex items-center gap-3">
                  <AmountDisplay
                    minor={payment.amountMinor}
                    currency={payment.currency}
                  />
                  <PaymentStatusBadge status={payment.status} />
                </div>
              </Link>
            ))}

            {appPayments.length === 0 && (
              <p className="p-8 text-center text-sm text-slate-500">
                No payments found for this application.
              </p>
            )}
          </div>
        </Card>
      )}

      {tab === "webhooks" && (
        <Card>
          <div className="border-b border-slate-100 px-5 py-4">
            <h2 className="text-sm font-semibold text-slate-900">
              Webhook deliveries
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Recent delivery attempts and their status
            </p>
          </div>

          <div className="divide-y divide-slate-100">
            {(deliveries.data ?? []).slice(0, 5).map((delivery) => (
              <div
                key={delivery.id}
                className="flex flex-wrap items-center justify-between gap-3 px-5 py-4"
              >
                <div className="min-w-0">
                  <p className="break-all font-mono text-[12.5px] text-slate-900">
                    {delivery.event}
                  </p>
                  <p className="mt-1 text-[11.5px] text-slate-500">
                    {timeAgo(delivery.createdAt)}
                  </p>
                </div>

                <span
                  className={`inline-flex items-center gap-1.5 text-[12px] font-medium ${
                    delivery.status === "delivered"
                      ? "text-emerald-600"
                      : delivery.status === "failed"
                        ? "text-red-600"
                        : "text-amber-600"
                  }`}
                >
                  <span
                    className={`size-1.5 rounded-full ${
                      delivery.status === "delivered"
                        ? "bg-emerald-500"
                        : delivery.status === "failed"
                          ? "bg-red-500"
                          : "bg-amber-500"
                    }`}
                  />
                  {delivery.status}
                </span>
              </div>
            ))}

            {(deliveries.data ?? []).length === 0 && (
              <p className="p-8 text-center text-sm text-slate-500">
                No webhook deliveries available.
              </p>
            )}
          </div>
        </Card>
      )}

      {tab === "logs" && (
        <Card>
          <div className="border-b border-slate-100 px-5 py-4">
            <h2 className="text-sm font-semibold text-slate-900">
              API request logs
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Recent requests associated with this application
            </p>
          </div>

          <div className="divide-y divide-slate-100">
            {(logs.data ?? [])
              .filter((log) => log.applicationId === params.id)
              .slice(0, 8)
              .map((log) => (
                <div
                  key={log.id}
                  className="flex flex-wrap items-center gap-3 px-5 py-3.5 font-mono text-[12px]"
                >
                  <span className="text-slate-400">
                    {timeAgo(log.at)}
                  </span>

                  <span className="font-semibold text-slate-700">
                    {log.method}
                  </span>

                  <span className="min-w-0 flex-1 break-all text-slate-600">
                    {log.path}
                  </span>

                  <span
                    className={`font-semibold ${
                      log.statusCode < 400
                        ? "text-emerald-600"
                        : "text-red-600"
                    }`}
                  >
                    {log.statusCode}
                  </span>
                </div>
              ))}

            {(logs.data ?? []).filter(
              (log) => log.applicationId === params.id
            ).length === 0 && (
              <p className="p-8 text-center text-sm text-slate-500">
                No API logs available for this application.
              </p>
            )}
          </div>
        </Card>
      )}

      {canManageApplications && tab === "settings" && (
        <Card>
          <div className="p-5 sm:p-6">
            <div className="flex items-start gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-red-50 text-red-600">
                <Trash2 className="size-5" />
              </div>

              <div>
                <h2 className="text-sm font-semibold text-slate-900">
                  Danger zone
                </h2>

                <p className="mt-1 max-w-xl text-[13px] leading-relaxed text-slate-500">
                  Deleting this application revokes its API keys
                  immediately and stops its webhooks. Payment
                  history is preserved, but the application
                  cannot be restored.
                </p>

                <Button
                  variant="danger"
                  className="mt-4"
                  disabled={remove.isPending}
                  onClick={onDelete}
                >
                  <Trash2 className="size-4" />
                  Delete application
                </Button>
              </div>
            </div>
          </div>
        </Card>
      )}

      <Modal
        open={canManageApplications && confirmOpen}
        onClose={() => {
          if (!remove.isPending) setConfirmOpen(false);
        }}
        title={`Delete ${application.name}?`}
        description="This revokes its API keys immediately and stops all webhooks. Payment history is preserved, but the application cannot be restored."
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setConfirmOpen(false)}
              disabled={remove.isPending}
            >
              Cancel
            </Button>

            <Button
              variant="dangerSolid"
              disabled={remove.isPending}
              onClick={() => remove.mutate()}
            >
              <Trash2 className="size-4" />
              {remove.isPending
                ? "Deleting…"
                : "Delete application"}
            </Button>
          </>
        }
      >
        <p className="text-[13px] leading-relaxed text-slate-600">
          Make sure you have a copy of any configuration you rely
          on. Deleted applications cannot be restored.
        </p>
      </Modal>

      <Modal
        open={canManageApplications && keyToRevoke !== null}
        onClose={() => {
          if (!revokeKey.isPending) setKeyToRevoke(null);
        }}
        title="Revoke this API key?"
        description="The key will stop working immediately. Its record will remain available for audit history."
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setKeyToRevoke(null)}
              disabled={revokeKey.isPending}
            >
              Cancel
            </Button>

            <Button
              variant="dangerSolid"
              disabled={revokeKey.isPending || !keyToRevoke}
              onClick={() =>
                keyToRevoke && revokeKey.mutate(keyToRevoke.id)
              }
            >
              <Trash2 className="size-4" />
              {revokeKey.isPending
                ? "Revoking…"
                : "Revoke key"}
            </Button>
          </>
        }
      >
        {keyToRevoke && (
          <p className="text-[13px] leading-relaxed text-slate-600">
            <span className="font-medium text-slate-900">
              {keyToRevoke.label}
            </span>{" "}
            will no longer be able to authenticate API requests.
          </p>
        )}
      </Modal>

      <Modal
        open={canManageApplications && keyToDelete !== null}
        onClose={() => {
          if (!deleteRevokedKey.isPending) setKeyToDelete(null);
        }}
        title="Permanently delete this revoked key?"
        description="This permanently removes the revoked key record and clears its API-key attribution from payment records. Audit events remain; this action cannot be undone."
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setKeyToDelete(null)}
              disabled={deleteRevokedKey.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="dangerSolid"
              disabled={deleteRevokedKey.isPending || !keyToDelete}
              onClick={() =>
                keyToDelete && deleteRevokedKey.mutate(keyToDelete.id)
              }
            >
              <Trash2 className="size-4" />
              {deleteRevokedKey.isPending ? "Deleting…" : "Delete permanently"}
            </Button>
          </>
        }
      >
        {keyToDelete && (
          <p className="text-[13px] leading-relaxed text-slate-600">
            <span className="font-medium text-slate-900">{keyToDelete.label}</span>{" "}
            will be removed from this application&apos;s key history.
          </p>
        )}
      </Modal>
    </div>
  );
}