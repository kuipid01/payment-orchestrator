"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  Check,
  Clock3,
  KeyRound,
  ShieldCheck,
  Trash2,
  Users,
} from "lucide-react";

import {
  Badge,
  Button,
  Card,
  PageHeader,
} from "@/components/ui";
import { Modal } from "@/components/ui/part2";
import { useAsync } from "@/hooks/use-async";
import {
  getDeveloper,
  listApplications,
  removeDeveloperAccess,
} from "@/lib/api";
import { formatDate, timeAgo } from "@/lib/utils";
import {
  DEVELOPER_PERMISSIONS,
} from "@/lib/types";
import { toast } from "@/components/ui/toaster";
import { DeveloperAvatar } from "@/components/app/developer-avatar";

const STATUS_META = {
  active: {
    label: "Active",
    variant: "success" as const,
    dot: "bg-emerald-500",
  },
  pending: {
    label: "Pending invitation",
    variant: "warning" as const,
    dot: "bg-amber-500",
  },
  revoked: {
    label: "Access revoked",
    variant: "danger" as const,
    dot: "bg-red-500",
  },
};

export default function DeveloperDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const {
    data: dev,
    loading,
  } = useAsync(
    () => getDeveloper(params.id),
    [params.id],
    ["developer", params.id]
  );

  const apps = useAsync(listApplications, []);

  const [revokeOpen, setRevokeOpen] = React.useState(false);
  const [removing, setRemoving] = React.useState(false);

  if (loading) {
    return (
      <div className="space-y-5">
        <div className="h-28 animate-pulse rounded-2xl border border-slate-200 bg-white" />
        <div className="h-64 animate-pulse rounded-2xl border border-slate-200 bg-white" />
      </div>
    );
  }

  if (!dev) {
    return (
      <div className="rounded-2xl border border-slate-200 bg-white p-10 text-center sm:p-14">
        <div className="mx-auto flex size-12 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
          <Users className="size-5" />
        </div>

        <h1 className="mt-4 text-lg font-semibold text-slate-900">
          Developer not found
        </h1>

        <p className="mt-1 text-sm text-slate-500">
          This developer may have been removed or the link may be incorrect.
        </p>

        <Link href="/developers" className="mt-6 inline-flex">
          <Button variant="secondary">
            <ArrowLeft className="size-4" />
            Back to developers
          </Button>
        </Link>
      </div>
    );
  }

  const appById = new Map(
    (apps.data ?? []).map((application) => [
      application.id,
      application,
    ])
  );

  const status = STATUS_META[dev.status] ?? STATUS_META.pending;

  async function confirmRemoval() {
    setRemoving(true);
    if(!dev) return;
    try {
      await removeDeveloperAccess(dev.id);

      setRevokeOpen(false);

      toast(
        "success",
        "Developer removed",
        `${dev?.email} no longer has access to this merchant.`
      );

      await queryClient.invalidateQueries({
        queryKey: ["developers"],
      });

      await queryClient.invalidateQueries({
        queryKey: ["developer", params.id],
      });

      router.replace("/developers");
    } catch (error) {
      toast(
        "error",
        "Could not remove developer",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    } finally {
      setRemoving(false);
    }
  }

  return (
    <div className="animate-fade-in space-y-6">
      <PageHeader
        breadcrumb={[
          { label: "Developers", href: "/developers" },
          { label: dev.name },
        ]}
        title="Developer access"
        description="Review the applications and permissions assigned to this developer."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={status.variant}>
              {status.label}
            </Badge>

            {dev.status !== "revoked" && (
              <Button
                variant="danger"
                size="sm"
                onClick={() => setRevokeOpen(true)}
              >
                <Trash2 className="size-3.5" />
                Remove access
              </Button>
            )}
          </div>
        }
      />

      <Card>
        <div className="p-5 sm:p-6">
          <div className="flex flex-wrap items-start gap-4">
            <DeveloperAvatar seed={dev.id || dev.email} className="size-14" />

            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-lg font-semibold tracking-tight text-slate-900">
                  {dev.name}
                </h2>

                <span className="flex items-center gap-1.5 text-xs text-slate-500">
                  <span className={`size-1.5 rounded-full ${status.dot}`} />
                  {status.label}
                </span>
              </div>

              <p className="mt-1 break-all text-sm text-slate-500">
                {dev.email}
              </p>

              <p className="mt-2 text-xs text-slate-400">
                Developer access ID:{" "}
                <span className="font-mono">{dev.accessId}</span>
              </p>
            </div>
          </div>

          <div className="mt-6 grid gap-3 border-t border-slate-100 pt-5 sm:grid-cols-3">
            <div className="rounded-xl border border-slate-200/80 bg-slate-50/50 p-4">
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <KeyRound className="size-4" />
                Applications
              </div>
              <p className="mt-2 text-2xl font-semibold tracking-tight text-slate-900">
                {dev.applicationIds.length}
              </p>
              <p className="mt-1 text-xs text-slate-400">
                Assigned to this developer
              </p>
            </div>

            <div className="rounded-xl border border-slate-200/80 bg-slate-50/50 p-4">
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <ShieldCheck className="size-4" />
                Permissions
              </div>
              <p className="mt-2 text-2xl font-semibold tracking-tight text-slate-900">
                {dev.permissions.length}
              </p>
              <p className="mt-1 text-xs text-slate-400">
                Granted capabilities
              </p>
            </div>

            <div className="rounded-xl border border-slate-200/80 bg-slate-50/50 p-4">
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <Clock3 className="size-4" />
                Last active
              </div>
              <p className="mt-2 text-lg font-semibold tracking-tight text-slate-900">
                {dev.lastActiveAt
                  ? timeAgo(dev.lastActiveAt)
                  : "Never"}
              </p>
              <p className="mt-1 text-xs text-slate-400">
                {dev.lastActiveAt
                  ? formatDate(dev.lastActiveAt)
                  : "No recorded activity"}
              </p>
            </div>
          </div>

          {dev.status === "pending" && (
            <div className="mt-4 rounded-xl border border-amber-200 bg-amber-50/70 p-4">
              <p className="text-sm font-medium text-amber-900">
                Invitation awaiting acceptance
              </p>
              <p className="mt-1 text-xs leading-relaxed text-amber-800">
                This developer has not accepted the invitation yet.
                Their access depends on the invitation being accepted.
              </p>
            </div>
          )}
        </div>
      </Card>

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
        <div className="space-y-5">
          <Card>
            <div className="border-b border-slate-100 px-5 py-4 sm:px-6">
              <div className="flex items-center gap-3">
                <span className="flex size-9 items-center justify-center rounded-xl bg-slate-100 text-slate-600">
                  <KeyRound className="size-4" />
                </span>

                <div>
                  <h2 className="text-sm font-semibold text-slate-900">
                    Assigned applications
                  </h2>
                  <p className="mt-0.5 text-xs text-slate-500">
                    Applications this developer can access
                  </p>
                </div>
              </div>
            </div>

            {dev.applicationIds.length > 0 ? (
              <div className="divide-y divide-slate-100">
                {dev.applicationIds.map((id) => {
                  const application = appById.get(id);

                  return (
                    <div
                      key={id}
                      className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 sm:px-6"
                    >
                      <div className="flex min-w-0 items-center gap-3">
                        <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500">
                          <KeyRound className="size-4" />
                        </div>

                        <div className="min-w-0">
                          {application ? (
                            <Link
                              href={`/applications/${application.id}`}
                              className="text-sm font-medium text-slate-900 hover:text-brand-600"
                            >
                              {application.name}
                            </Link>
                          ) : (
                            <p className="break-all text-sm font-medium text-slate-800">
                              {id}
                            </p>
                          )}

                          <p className="mt-1 text-xs text-slate-500">
                            {application
                              ? application.description ||
                                "No description provided"
                              : "Application details unavailable"}
                          </p>
                        </div>
                      </div>

                      <Badge variant="brand">Scoped access</Badge>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="p-8 text-center">
                <p className="text-sm font-medium text-slate-800">
                  No applications assigned
                </p>
                <p className="mt-1 text-xs text-slate-500">
                  Assign this developer to an application before
                  they can use application-scoped resources.
                </p>
              </div>
            )}
          </Card>

          <Card>
            <div className="border-b border-slate-100 px-5 py-4 sm:px-6">
              <h2 className="text-sm font-semibold text-slate-900">
                Permissions
              </h2>
              <p className="mt-1 text-xs text-slate-500">
                Capabilities granted to this developer
              </p>
            </div>

            <div className="grid gap-3 p-5 sm:grid-cols-2 sm:p-6">
              {DEVELOPER_PERMISSIONS.map((permission) => {
                const enabled = dev.permissions.includes(permission.id);
                const elevated =
                  "dangerous" in permission &&
                  permission.dangerous;

                return (
                  <div
                    key={permission.id}
                    className={`rounded-xl border p-4 ${
                      enabled
                        ? elevated
                          ? "border-amber-200 bg-amber-50/50"
                          : "border-emerald-200 bg-emerald-50/40"
                        : "border-slate-200 bg-white"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-start gap-2.5">
                        <span
                          className={`mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full ${
                            enabled
                              ? elevated
                                ? "bg-amber-100 text-amber-700"
                                : "bg-emerald-100 text-emerald-700"
                              : "bg-slate-100 text-slate-400"
                          }`}
                        >
                          {enabled ? (
                            <Check className="size-3" />
                          ) : (
                            <span className="text-xs">–</span>
                          )}
                        </span>

                        <div>
                          <p className="text-sm font-medium text-slate-800">
                            {permission.label}
                          </p>
                          <p className="mt-1 font-mono text-[11px] text-slate-400">
                            {permission.id}
                          </p>
                        </div>
                      </div>

                      {elevated && (
                        <span className="shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-800">
                          Elevated
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </Card>
        </div>

        <aside className="space-y-5">
          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">
                Access information
              </h2>
            </div>

            <div className="space-y-4 p-5">
              <div>
                <p className="text-xs text-slate-500">Invitation sent</p>
                <p className="mt-1 text-sm font-medium text-slate-800">
                  {formatDate(dev.invitedAt)}
                </p>
              </div>

              <div className="border-t border-slate-100 pt-4">
                <p className="text-xs text-slate-500">Last active</p>
                <p className="mt-1 text-sm font-medium text-slate-800">
                  {dev.lastActiveAt
                    ? timeAgo(dev.lastActiveAt)
                    : "No recorded activity"}
                </p>
              </div>

              <div className="border-t border-slate-100 pt-4">
                <p className="text-xs text-slate-500">Authentication</p>
                <p className="mt-1 text-sm font-medium text-slate-800">
                  PayConnect API key
                </p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="p-5">
              <div className="flex size-9 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700">
                <ShieldCheck className="size-4.5" />
              </div>

              <h3 className="mt-3 text-sm font-semibold text-slate-900">
                Provider credentials stay private
              </h3>

              <p className="mt-2 text-xs leading-relaxed text-slate-500">
                Developers can work with permitted payments, logs,
                and webhooks without being given access to provider
                secrets.
              </p>
            </div>
          </Card>
        </aside>
      </div>

      <Modal
        open={revokeOpen}
        onClose={() => {
          if (!removing) setRevokeOpen(false);
        }}
        title={`Remove ${dev.name}?`}
        description="This revokes the developer's access to your merchant account. Access to other merchants is not affected."
        footer={
          <>
            <Button
              variant="secondary"
              disabled={removing}
              onClick={() => setRevokeOpen(false)}
            >
              Cancel
            </Button>

            <Button
              variant="dangerSolid"
              disabled={removing}
              onClick={confirmRemoval}
            >
              <Trash2 className="size-4" />
              {removing ? "Removing…" : "Remove access"}
            </Button>
          </>
        }
      >
        <p className="text-sm leading-relaxed text-slate-600">
          <strong className="text-slate-900">{dev.email}</strong>{" "}
          will no longer be able to use this merchant's developer
          access. Historical invitation records and logs should remain
          available for auditing.
        </p>
      </Modal>
    </div>
  );
}