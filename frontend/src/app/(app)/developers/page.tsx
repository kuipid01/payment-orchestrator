"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowRight,
  ArrowRightLeft,
  Check,
  KeyRound,
  Search,
  Send,
  ShieldCheck,
  Trash2,
  UserRoundPlus,
  Users,
} from "lucide-react";

import {
  Badge,
  Button,
  Card,
  EmptyState,
  EnvironmentBadge,
  Modal,
  PageHeader,
  Tabs,
} from "@/components/ui";
import { toast } from "@/components/ui/toaster";
import { useAuth } from "@/components/providers/auth-provider";
import { useAsync } from "@/hooks/use-async";
import { DeveloperAvatar } from "@/components/app/developer-avatar";
import {
  getDeveloperSelfAccess,
  listApplications,
  listDevelopers,
  revokeApiKey,
  sendDeveloperTestWebhook,
  switchOrganization,
} from "@/lib/api";
import type { SelfAccess } from "@/lib/types";
import { timeAgo } from "@/lib/utils";

const STATUS_META: Record<
  string,
  {
    label: string;
    variant: "success" | "warning" | "danger";
  }
> = {
  active: { label: "Active", variant: "success" },
  pending: { label: "Pending invite", variant: "warning" },
  revoked: { label: "Revoked", variant: "danger" },
};

export default function DevelopersPage() {
  const [tab, setTab] = React.useState("team");

  return (
    <div className="animate-fade-in space-y-6">
      <PageHeader
        title="Developers"
        description="Manage developer access to your payment infrastructure and the organizations you work with."
        actions={
          <Link href="/developers/invite">
            <Button size="sm">
              <UserRoundPlus className="size-4" />
              Invite developer
            </Button>
          </Link>
        }
      />

      <Tabs
        tabs={[
          { value: "team", label: "Your developers" },
          { value: "access", label: "My access" },
        ]}
        value={tab}
        onValueChange={setTab}
        className="w-fit max-w-full"
      />

      {tab === "access" ? <MyAccessPanel /> : <TeamPanel />}
    </div>
  );
}

function TeamPanel() {
  const { user } = useAuth();
  const canManageTeam = user?.role === "Owner" || user?.role === "Admin";
  const { data, loading } = useAsync(
    listDevelopers,
    [],
    ["developers"]
  );

  const apps = useAsync(listApplications, []);
  const [search, setSearch] = React.useState("");
  const [statusFilter, setStatusFilter] = React.useState("all");

  const developers = data ?? [];

  const filtered = developers.filter((dev) => {
    const query = search.trim().toLowerCase();

    const matchesQuery =
      !query ||
      dev.name.toLowerCase().includes(query) ||
      dev.email.toLowerCase().includes(query) ||
      dev.permissions.some((permission) =>
        permission.toLowerCase().includes(query)
      );

    const matchesStatus =
      statusFilter === "all" || dev.status === statusFilter;

    return matchesQuery && matchesStatus;
  });

  const activeCount = developers.filter(
    (dev) => dev.status === "active"
  ).length;

  const pendingCount = developers.filter(
    (dev) => dev.status === "pending"
  ).length;
  const revokedCount = developers.filter(
    (dev) => dev.status === "revoked"
  ).length;

  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, index) => (
          <div
            key={index}
            className="h-24 animate-pulse rounded-2xl border border-slate-200 bg-white"
          />
        ))}
      </div>
    );
  }

  if (developers.length === 0) {
    return (
      <Card>
        <EmptyState
          icon={<Users className="size-5" />}
          title="No developers yet"
          description={
            canManageTeam
              ? "Invite a developer to start building with your payment infrastructure."
              : "Developers you invite will appear here."
          }
          action={
            <Link href="/developers/invite">
              <Button>
                <UserRoundPlus className="size-4" />
                Invite developer
              </Button>
            </Link>
          }
        />
      </Card>
    );
  }

  return (
    <div className="space-y-5">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="Total developers"
          value={developers.length}
          detail="Developer access records"
          icon={<Users className="size-4" />}
        />

        <MetricCard
          label="Active"
          value={activeCount}
          detail="Access currently active"
          icon={<ShieldCheck className="size-4" />}
          accent="green"
        />

        <MetricCard
          label="Pending invitations"
          value={pendingCount}
          detail="Awaiting acceptance"
          icon={<Send className="size-4" />}
          accent="amber"
        />
        <MetricCard
          label="Revoked"
          value={revokedCount}
          detail="No longer have access"
          icon={<Users className="size-4" />}
        />
      </div>

      <Card>
        <div className="flex flex-col gap-4 border-b border-slate-100 p-4 sm:p-5">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 className="text-sm font-semibold text-slate-900">
                {canManageTeam ? "Team members" : "Developers you invited"}
              </h2>
              <p className="mt-1 text-xs text-slate-500">
                {canManageTeam
                  ? `${filtered.length} of ${developers.length} developers`
                  : "Only invitations you sent appear here."}
              </p>
            </div>
            <div className="relative w-full lg:max-w-xs">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search name, email, permission…"
                aria-label="Search developers"
                className="h-10 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-3 text-sm text-slate-800 outline-none transition focus:border-brand-300 focus:ring-2 focus:ring-brand-100"
              />
            </div>
          </div>

          <div className="flex flex-wrap gap-2" role="group" aria-label="Filter developers by status">
            {([
              ["all", "All", developers.length],
              ["active", "Active", activeCount],
              ["pending", "Pending", pendingCount],
              ["revoked", "Revoked", revokedCount],
            ] as const).map(([value, label, count]) => (
              <button
                key={value}
                type="button"
                onClick={() => setStatusFilter(value)}
                aria-pressed={statusFilter === value}
                className={`inline-flex items-center gap-1.5 rounded-lg border px-3 py-2 text-xs font-medium transition-colors ${
                  statusFilter === value
                    ? "border-slate-900 bg-slate-900 text-white"
                    : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                }`}
              >
                {label}
                <span className={statusFilter === value ? "text-white/70" : "text-slate-400"}>
                  {count}
                </span>
              </button>
            ))}
          </div>
        </div>

        <div className="hidden grid-cols-[minmax(0,1.6fr)_90px_minmax(160px,1fr)_130px_100px] items-center gap-4 bg-slate-50/70 px-5 py-3 text-[10px] font-semibold uppercase tracking-wide text-slate-400 lg:grid">
          <span>Developer</span>
          <span>Apps</span>
          <span>Permissions</span>
          <span>Invited</span>
          <span className="text-right">Access</span>
        </div>
        <div className="divide-y divide-slate-100">
          {filtered.map((dev) => {
            const status =
              STATUS_META[dev.status] ?? STATUS_META.pending;

            return (
              <Link
                key={dev.id}
                href={`/developers/${dev.id}`}
                className="grid gap-3 px-4 py-4 transition-colors hover:bg-slate-50/60 sm:px-5 lg:grid-cols-[minmax(0,1.6fr)_90px_minmax(160px,1fr)_130px_100px] lg:items-center lg:gap-4"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <DeveloperAvatar seed={dev.id || dev.email} className="size-10 rounded-xl" />
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="truncate text-sm font-semibold text-slate-900">{dev.name}</span>
                      <Badge variant={status.variant}>{status.label}</Badge>
                    </div>
                    <p className="mt-1 truncate text-xs text-slate-500">{dev.email}</p>
                  </div>
                </div>

                <span className="text-xs text-slate-600 lg:block">
                  <span className="lg:hidden">Applications · </span>
                  {dev.applicationIds.length}
                </span>

                <div className="flex flex-wrap gap-1.5">
                  {dev.permissions.slice(0, 2).map((permission) => (
                    <span key={permission} className="rounded-md border border-slate-200 bg-white px-2 py-1 font-mono text-[10px] text-slate-500">
                      {permission}
                    </span>
                  ))}
                  {dev.permissions.length > 2 && (
                    <span className="rounded-md bg-slate-100 px-2 py-1 text-[10px] text-slate-500">
                      +{dev.permissions.length - 2}
                    </span>
                  )}
                </div>

                <div className="text-xs text-slate-500">
                  <span className="lg:hidden">Invited · </span>{timeAgo(dev.invitedAt)}
                </div>

                <span className="inline-flex items-center gap-1.5 text-xs font-medium text-brand-600 lg:justify-end">
                  View <ArrowRight className="size-3.5" />
                </span>
              </Link>
            );
          })}

          {filtered.length === 0 && (
            <div className="p-10 text-center">
              <Search className="mx-auto size-5 text-slate-300" />
              <p className="mt-3 text-sm font-medium text-slate-800">
                No matching developers
              </p>
              <p className="mt-1 text-xs text-slate-500">
                Try another search term or status filter.
              </p>
              <Button
                variant="ghost"
                size="sm"
                className="mt-3"
                onClick={() => {
                  setSearch("");
                  setStatusFilter("all");
                }}
              >
                Clear filters
              </Button>
            </div>
          )}
        </div>

        <div className="border-t border-slate-100 px-5 py-3">
          <p className="text-xs text-slate-400">
            Developer access is scoped to your merchant account.
          </p>
        </div>
      </Card>
    </div>
  );
}

function MetricCard({
  label,
  value,
  detail,
  icon,
  accent = "brand",
}: {
  label: string;
  value: number;
  detail: string;
  icon: React.ReactNode;
  accent?: "brand" | "green" | "amber";
}) {
  const styles = {
    brand: "bg-brand-50 text-brand-600",
    green: "bg-emerald-50 text-emerald-700",
    amber: "bg-amber-50 text-amber-700",
  };

  return (
    <Card>
      <div className="p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3">
          <p className="text-xs font-medium text-slate-500">
            {label}
          </p>
          <span className={`flex size-8 items-center justify-center rounded-lg ${styles[accent]}`}>
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

function MyAccessPanel() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user, refresh } = useAuth();

  const {
    data,
    loading,
    refetch,
  } = useAsync(
    getDeveloperSelfAccess,
    [],
    ["developer-self"]
  );

  const [search, setSearch] = React.useState("");
  const [busyKey, setBusyKey] = React.useState<string | null>(null);
  const [testingId, setTestingId] = React.useState<string | null>(null);
  const [switching, setSwitching] = React.useState<string | null>(null);
  const [revokeTarget, setRevokeTarget] = React.useState<{
    keyId: string;
    masked: string;
  } | null>(null);

  const currentMerchantId = user?.merchant_id ?? null;

  const grants = React.useMemo(
    () =>
      [...(data ?? [])].sort(
        (a, b) =>
          Number(b.status === "active") -
          Number(a.status === "active")
      ),
    [data]
  );

  const filtered = grants.filter((grant) => {
    const query = search.trim().toLowerCase();

    return (
      !query ||
      grant.merchant_name.toLowerCase().includes(query) ||
      grant.merchant_id.toLowerCase().includes(query) ||
      grant.permissions.some((permission) =>
        permission.toLowerCase().includes(query)
      )
    );
  });

  async function onSwitch(merchantId: string) {
    setSwitching(merchantId);

    try {
      await switchOrganization(merchantId);
      await refresh();
      await queryClient.invalidateQueries();

      toast(
        "success",
        "Organization switched",
        "Your session now acts in that organization."
      );

      router.push("/dashboard");
    } catch (error) {
      toast(
        "error",
        "Could not switch organization",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    } finally {
      setSwitching(null);
    }
  }

  async function onRevoke() {
    if (!revokeTarget) return;

    const target = revokeTarget;
    setBusyKey(target.keyId);

    try {
      await revokeApiKey(target.keyId);

      toast(
        "success",
        "Key revoked",
        `${target.masked} can no longer authenticate requests.`
      );

      setRevokeTarget(null);

      await refetch();

      await queryClient.invalidateQueries({
        queryKey: ["api-keys"],
      });
    } catch (error) {
      toast(
        "error",
        "Could not revoke key",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    } finally {
      setBusyKey(null);
    }
  }

  async function onTestWebhook(applicationId: string) {
    setTestingId(applicationId);

    try {
      const result = await sendDeveloperTestWebhook(applicationId);
      const ok = result.status === "delivered";

      toast(
        ok ? "success" : "error",
        ok ? "Test webhook delivered" : "Test webhook failed",
        ok
          ? `${result.url} answered ${result.response_code ?? "2xx"}.`
          : result.last_error || `Delivery status: ${result.status}.`
      );
    } catch (error) {
      toast(
        "error",
        "Test webhook could not be sent",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    } finally {
      setTestingId(null);
    }
  }

  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 2 }).map((_, index) => (
          <div
            key={index}
            className="h-52 animate-pulse rounded-2xl border border-slate-200 bg-white"
          />
        ))}
      </div>
    );
  }

  if (grants.length === 0) {
    return (
      <Card>
        <EmptyState
          icon={<ShieldCheck className="size-5" />}
          title="No developer access yet"
          description="When an organization invites you, its applications, permissions and developer keys will appear here."
        />
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-sm font-semibold text-slate-900">
            Organizations that invited you
          </h2>
          <p className="mt-1 text-xs text-slate-500">
            Switch organizations to work within the correct merchant account.
          </p>
        </div>

        <div className="relative sm:w-64">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search organizations…"
            aria-label="Search organizations"
            className="h-10 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-3 text-sm outline-none focus:border-brand-300 focus:ring-2 focus:ring-brand-100"
          />
        </div>
      </div>

      {filtered.map((grant) => (
        <GrantCard
          key={grant.access_id}
          grant={grant}
          isCurrent={grant.merchant_id === currentMerchantId}
          busyKey={busyKey}
          testingId={testingId}
          switching={switching}
          onRevoke={(keyId, masked) =>
            setRevokeTarget({ keyId, masked })
          }
          onTestWebhook={onTestWebhook}
          onSwitch={onSwitch}
        />
      ))}

      {filtered.length === 0 && (
        <Card>
          <div className="p-8 text-center">
            <p className="text-sm font-medium text-slate-800">
              No matching organizations
            </p>
            <p className="mt-1 text-xs text-slate-500">
              Try searching by organization name or permission.
            </p>
          </div>
        </Card>
      )}

      <Modal
        open={revokeTarget !== null}
        onClose={() => {
          if (!busyKey) setRevokeTarget(null);
        }}
        title="Revoke this API key?"
        description="The key will no longer authenticate requests. This action cannot be undone."
        footer={
          <>
            <Button
              variant="secondary"
              disabled={busyKey !== null}
              onClick={() => setRevokeTarget(null)}
            >
              Cancel
            </Button>
            <Button
              variant="dangerSolid"
              disabled={busyKey !== null}
              onClick={onRevoke}
            >
              <Trash2 className="size-4" />
              {busyKey ? "Revoking…" : "Revoke key"}
            </Button>
          </>
        }
      >
        {revokeTarget && (
          <p className="text-sm text-slate-600">
            Revoke key{" "}
            <code className="font-mono text-slate-900">
              {revokeTarget.masked}
            </code>
            ?
          </p>
        )}
      </Modal>
    </div>
  );
}

function GrantCard({
  grant,
  isCurrent,
  busyKey,
  testingId,
  switching,
  onRevoke,
  onTestWebhook,
  onSwitch,
}: {
  grant: SelfAccess;
  isCurrent: boolean;
  busyKey: string | null;
  testingId: string | null;
  switching: string | null;
  onRevoke: (keyId: string, masked: string) => void;
  onTestWebhook: (applicationId: string) => void;
  onSwitch: (merchantId: string) => void;
}) {
  const meta = STATUS_META[grant.status] ?? STATUS_META.pending;
  const pending = grant.status !== "active";

  return (
    <Card>
      <div className="p-5 sm:p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-xl border border-brand-100 bg-brand-50 text-brand-700">
              <Users className="size-5" />
            </div>

            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="text-sm font-semibold text-slate-900">
                  {grant.merchant_name}
                </h3>

                <Badge variant={meta.variant}>{meta.label}</Badge>

                {isCurrent && (
                  <Badge variant="brand">Current organization</Badge>
                )}
              </div>

              <p className="mt-1 break-all font-mono text-[11px] text-slate-400">
                {grant.merchant_id}
              </p>

              <p className="mt-1 text-xs text-slate-500">
                Invited {timeAgo(grant.invited_at)}
                {grant.accepted_at
                  ? ` · Accepted ${timeAgo(grant.accepted_at)}`
                  : ""}
              </p>
            </div>
          </div>

          {!isCurrent && !pending && (
            <Button
              variant="secondary"
              size="sm"
              disabled={switching !== null}
              onClick={() => onSwitch(grant.merchant_id)}
            >
              <ArrowRightLeft className="size-3.5" />
              {switching === grant.merchant_id
                ? "Switching…"
                : "Switch organization"}
            </Button>
          )}
        </div>

        {grant.permissions.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2 border-t border-slate-100 pt-4">
            {grant.permissions.map((permission) => (
              <span
                key={permission}
                className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-[10.5px] text-slate-500"
              >
                {permission}
              </span>
            ))}
          </div>
        )}

        {pending && (
          <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50/70 p-3 text-xs leading-relaxed text-amber-900">
            This invitation has not been accepted. Check your email
            for the invitation link before trying to use its applications.
          </div>
        )}

        <div className="mt-5 border-t border-slate-100 pt-4">
          <div className="mb-3 flex items-center justify-between gap-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Applications
            </h4>
            <span className="text-xs text-slate-400">
              {grant.applications.length} total
            </span>
          </div>

          {grant.applications.length === 0 ? (
            <p className="rounded-lg border border-dashed border-slate-200 p-4 text-xs text-slate-500">
              No applications in this organization yet.
            </p>
          ) : (
            <div className="space-y-3">
              {grant.applications.map((app) => (
                <div
                  key={app.id}
                  className="rounded-xl border border-slate-200/80 p-4"
                >
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-slate-900">
                        {app.name}
                      </p>
                      <p className="mt-1 text-xs leading-relaxed text-slate-500">
                        {app.description || "No description provided"}
                      </p>
                      <p className="mt-1 text-xs capitalize text-slate-400">
                        Status: {app.status}
                      </p>
                    </div>

                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={pending || testingId !== null}
                      onClick={() => onTestWebhook(app.id)}
                    >
                      <Send className="size-3.5" />
                      {testingId === app.id
                        ? "Sending…"
                        : "Test webhook"}
                    </Button>
                  </div>

                  {app.keys.length === 0 ? (
                    <p className="mt-3 rounded-lg bg-slate-50 p-3 text-xs leading-relaxed text-slate-500">
                      No developer-scoped keys for this application
                      yet. The organization can issue them from its
                      application's API Keys tab.
                    </p>
                  ) : (
                    <ul className="mt-3 space-y-2">
                      {app.keys.map((key) => (
                        <li
                          key={key.id}
                          className="flex flex-col gap-3 rounded-lg border border-slate-100 bg-slate-50/60 p-3 sm:flex-row sm:items-center sm:justify-between"
                        >
                          <div className="flex min-w-0 items-start gap-2.5">
                            <KeyRound className="mt-0.5 size-4 shrink-0 text-brand-500" />

                            <div className="min-w-0">
                              <p className="break-words text-xs font-medium text-slate-800">
                                {key.name}
                              </p>
                              <p className="mt-1 break-all font-mono text-[11px] text-slate-400">
                                {key.masked}
                              </p>
                            </div>
                          </div>

                          <div className="flex flex-wrap items-center gap-2">
                            <EnvironmentBadge
                              environment={key.environment}
                            />

                            <Badge
                              variant={
                                key.status === "active"
                                  ? "success"
                                  : "danger"
                              }
                            >
                              {key.status}
                            </Badge>

                            <Button
                              variant="ghost"
                              size="sm"
                              disabled={
                                busyKey !== null ||
                                !isCurrent ||
                                key.status !== "active"
                              }
                              title={
                                !isCurrent
                                  ? "Switch to this organization to manage the key"
                                  : undefined
                              }
                              onClick={() =>
                                onRevoke(key.id, key.masked)
                              }
                            >
                              <Trash2 className="size-3.5" />
                              Revoke
                            </Button>
                          </div>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {isCurrent && (
          <div className="mt-4 flex items-start gap-2 border-t border-slate-100 pt-4">
            <Check className="mt-0.5 size-4 shrink-0 text-emerald-600" />
            <p className="text-xs leading-relaxed text-slate-500">
              You are currently acting in this organization. Key
              management actions are available where permitted.
            </p>
          </div>
        )}
      </div>
    </Card>
  );
}