"use client";

import * as React from "react";
import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";
import { KeyRound, Plus, RotateCcw, Trash2, UserCheck } from "lucide-react";
import { Badge, Button, Card, EmptyState, EnvironmentBadge, PageHeader } from "@/components/ui";
import { MaskedSecret, Modal } from "@/components/ui/part2";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import { listApiKeys, listApplications, listDevelopers, permanentlyDeleteRevokedApiKey, revokeApiKey, revealApiKey, rotateApiKey } from "@/lib/api";
import { formatDate, timeAgo } from "@/lib/utils";
import type { ApiKey } from "@/lib/types";

export default function ApiKeysPage() {
  const queryClient = useQueryClient();
  const { data, loading, refetch } = useAsync(() => listApiKeys(), [], ["api-keys"]);
  const apps = useAsync(listApplications, [], ["applications"]);
  const developers = useAsync(listDevelopers, [], ["developers"]);
  const [busy, setBusy] = React.useState<string | null>(null);
  const [rotated, setRotated] = React.useState<{ key: ApiKey; secret: string } | null>(null);
  const [confirmRevoke, setConfirmRevoke] = React.useState<ApiKey | null>(null);
  const [confirmDelete, setConfirmDelete] = React.useState<ApiKey | null>(null);

  const appById = new Map((apps.data ?? []).map((a) => [a.id, a]));
  const developerByAccessId = new Map((developers.data ?? []).map((d) => [d.accessId, d]));
  const live = (data ?? []).filter((k) => k.environment === "live");
  const test = (data ?? []).filter((k) => k.environment === "test");

  async function onRotate(key: ApiKey) {
    setBusy(key.id);
    try {
      const result = await rotateApiKey(key.id);
      setRotated(result);
      toast("success", "Key rotated", "The previous key stopped working immediately.");
      await refetch();
      await queryClient.invalidateQueries({ queryKey: ["payments"] });
    } catch (error) {
      toast("error", "Rotation failed", error instanceof Error ? error.message : "Please try again.");
    } finally {
      setBusy(null);
    }
  }

  async function onRevoke(key: ApiKey) {
    setBusy(key.id);
    try {
      await revokeApiKey(key.id);
      toast("success", "Key revoked", `${key.masked} can no longer authenticate requests.`);
      setConfirmRevoke(null);
      await refetch();
    } catch (error) {
      toast("error", "Revoke failed", error instanceof Error ? error.message : "Please try again.");
    } finally {
      setBusy(null);
    }
  }

  async function onDeleteRevoked(key: ApiKey) {
    setBusy(key.id);
    try {
      await permanentlyDeleteRevokedApiKey(key.id);
      toast("success", "Revoked key deleted", `${key.label} was permanently deleted.`);
      setConfirmDelete(null);
      await refetch();
      await queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    } catch (error) {
      toast("error", "Delete failed", error instanceof Error ? error.message : "Please try again.");
    } finally {
      setBusy(null);
    }
  }

  function KeyRow({ k }: { k: ApiKey }) {
    const app = appById.get(k.applicationId);
    const developer = k.developerAccessId ? developerByAccessId.get(k.developerAccessId) : undefined;
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-[13.5px] font-medium text-slate-900">{k.label}</p>
            {app ? (
              <Link href={`/applications/${app.id}`} className="text-[12px] text-brand-500 hover:underline">
                {app.name}
              </Link>
            ) : null}
            <Badge variant={k.scope === "developer" ? "brand" : "neutral"}>
              {k.scope === "developer" ? (
                <span className="flex items-center gap-1">
                  <UserCheck className="size-3" /> Developer
                </span>
              ) : (
                "General (shared)"
              )}
            </Badge>
            <Badge variant={k.status === "active" ? "success" : "danger"}>{k.status}</Badge>
          </div>
          <p className="mt-0.5 text-[12px] text-slate-400">
            {k.developerAccessId ? `${developer?.name ?? developer?.email ?? k.developerAccessId} · ` : ""}
            Created {formatDate(k.createdAt)}
            {k.createdByName ? ` by ${k.createdByName}` : ""} · Last used {k.lastUsedAt ? timeAgo(k.lastUsedAt) : "never"}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <MaskedSecret
            value={k.masked}
            revealable={k.revealable && k.status === "active"}
            onReveal={k.revealable && k.status === "active" ? () => revealApiKey(k.id) : undefined}
          />
          <Button
            variant="secondary"
            size="icon"
            title="Rotate key"
            aria-label="Rotate key"
            disabled={busy !== null || k.status !== "active"}
            onClick={() => void onRotate(k)}
          >
            <RotateCcw className="size-3.5" />
          </Button>
          {k.status === "active" ? (
            <Button
              variant="danger"
              size="icon"
              title="Revoke key"
              aria-label={`Revoke ${k.label}`}
              disabled={busy !== null}
              onClick={() => setConfirmRevoke(k)}
            >
              <Trash2 className="size-3.5" />
            </Button>
          ) : k.status === "revoked" ? (
            <Button
              variant="danger"
              size="icon"
              title="Delete revoked key"
              aria-label={`Delete revoked key ${k.label}`}
              disabled={busy !== null}
              onClick={() => setConfirmDelete(k)}
            >
              <Trash2 className="size-3.5" />
            </Button>
          ) : null}
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="animate-fade-in">
        <PageHeader title="API Keys" description="Developer-facing keys for your applications." />
        <div className="h-64 animate-pulse rounded-[var(--radius-card)] border border-slate-200 bg-white" />
      </div>
    );
  }

  return (
    <div className="animate-fade-in">
      <PageHeader
        title="API Keys"
        description="These keys are what developers use — never your provider secrets."
        actions={
          <Link href="/applications/new">
            <Button size="sm">
              <Plus className="size-4" /> Create key via application
            </Button>
          </Link>
        }
      />

      {rotated ? (
        <Card className="mb-4 border-amber-200 bg-amber-50">
          <div className="flex flex-wrap items-center justify-between gap-3 p-5">
            <div className="min-w-0">
              <p className="text-[13px] font-semibold text-amber-900">
                New {rotated.key.environment === "live" ? "production" : "test"} key for {rotated.key.label}
              </p>
              <code className="mt-1 block truncate font-mono text-[12.5px] text-amber-950">{rotated.secret}</code>
              <p className="mt-1 text-[12px] text-amber-800">
                Copy it now — a live key is never shown again, while a test key stays readable here.
              </p>
            </div>
            <Button variant="ghost" onClick={() => setRotated(null)}>
              Dismiss
            </Button>
          </div>
        </Card>
      ) : null}

      <div className="space-y-4">
        <Card>
          <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3.5">
            <h2 className="text-sm font-semibold text-slate-900">Production</h2>
            <span className="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700">
              pc_live_…
            </span>
          </div>
          {live.length ? (
            <div className="divide-y divide-slate-100">
              {live.map((k) => (
                <KeyRow key={k.id} k={k} />
              ))}
            </div>
          ) : (
            <EmptyState title="No live keys" description="Create a production application to generate a live key." />
          )}
        </Card>

        <Card>
          <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3.5">
            <h2 className="text-sm font-semibold text-slate-900">Test</h2>
            <span className="rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-700">
              pc_test_…
            </span>
          </div>
          <div className="divide-y divide-slate-100">
            {test.map((k) => (
              <KeyRow key={k.id} k={k} />
            ))}
            {test.length === 0 ? (
              <EmptyState
                title="No test keys"
                description="A test key is safe to copy and paste anywhere — it is pinned to test data."
              />
            ) : null}
          </div>
        </Card>

        <Card className="border-brand-100 bg-brand-50/50">
          <div className="flex items-start gap-3 p-5">
            <KeyRound className="mt-0.5 size-4 shrink-0 text-brand-500" />
            <div>
              <p className="text-[13.5px] font-semibold text-brand-900">
                Test keys stay copyable. Live keys are shown once.
              </p>
              <p className="mt-1 text-[12.5px] leading-relaxed text-brand-900/80">
                A test key cannot touch live money, so reading it again costs nothing — use the eye or copy icon on a
                test row to get the full value back. A live key exists only as a hash after creation: if it is lost,
                rotate it. These are <b>not</b> your Paystack / Flutterwave / Stripe secret; PayConnect keys are
                scoped per application and environment, while provider credentials stay in the vault.
              </p>
            </div>
          </div>
        </Card>
      </div>

      <Modal
        open={confirmRevoke !== null}
        onClose={() => {
          if (busy === null) setConfirmRevoke(null);
        }}
        title={`Revoke ${confirmRevoke?.label ?? "this key"}?`}
        description="Anything still using this key starts failing immediately. This cannot be undone."
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmRevoke(null)} disabled={busy !== null}>
              Cancel
            </Button>
            <Button
              variant="dangerSolid"
              disabled={busy !== null}
              onClick={() => confirmRevoke && void onRevoke(confirmRevoke)}
            >
              <Trash2 className="size-4" /> {busy ? "Revoking…" : "Revoke key"}
            </Button>
          </>
        }
      >
        <p className="text-[13px] leading-relaxed text-slate-600">
          {confirmRevoke?.developerAccessId
            ? "This is a developer-scoped key: revoking it does not affect the application's shared key."
            : "This is a general key. If it is deployed in a server configuration, that deployment will need a new key."}
        </p>
      </Modal>

      <Modal
        open={confirmDelete !== null}
        onClose={() => {
          if (busy === null) setConfirmDelete(null);
        }}
        title={`Permanently delete ${confirmDelete?.label ?? "this revoked key"}?`}
        description="The key is already unusable. This permanently removes its record and clears its API-key attribution from payment records. Audit events remain."
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmDelete(null)} disabled={busy !== null}>
              Cancel
            </Button>
            <Button
              variant="dangerSolid"
              disabled={busy !== null || !confirmDelete}
              onClick={() => confirmDelete && void onDeleteRevoked(confirmDelete)}
            >
              <Trash2 className="size-4" /> {busy ? "Deleting…" : "Delete permanently"}
            </Button>
          </>
        }
      >
        <p className="text-[13px] leading-relaxed text-slate-600">
          This action only deletes an already-revoked API key.
        </p>
      </Modal>
    </div>
  );
}
