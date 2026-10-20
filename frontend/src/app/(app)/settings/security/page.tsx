"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { FileText, KeyRound, Laptop, LockKeyhole, ShieldCheck } from "lucide-react";
import { Badge, Button, Input, Label, Switch } from "@/components/ui";
import { useAuth } from "@/components/providers/auth-provider";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import { changePassword, listSessions, revokeSession } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import {
  SettingsEmptyState,
  SettingsSection,
} from "@/components/app/settings/section";

export default function SecuritySettingsPage() {
  const router = useRouter();
  const { signOut } = useAuth();
  const [currentPassword, setCurrentPassword] = React.useState("");
  const [newPassword, setNewPassword] = React.useState("");
  const [savingPassword, setSavingPassword] = React.useState(false);
  const [revoking, setRevoking] = React.useState<string | null>(null);
  const { data: sessions, loading, error, refetch } = useAsync(
    listSessions,
    [],
    ["settings", "sessions"],
  );

  async function submitPassword(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingPassword(true);
    try {
      await changePassword(currentPassword, newPassword);
      setCurrentPassword("");
      setNewPassword("");
      toast(
        "success",
        "Password changed",
        "Saved refresh sessions were revoked. Other access tokens expire within 15 minutes; sign in again with your new password.",
      );
      await signOut();
      router.replace("/login");
    } catch (cause) {
      toast("error", "Could not change password", cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setSavingPassword(false);
    }
  }

  async function signOutSession(id: string) {
    setRevoking(id);
    try {
      await revokeSession(id);
      await refetch();
      toast("success", "Session revoked");
    } catch (cause) {
      toast("error", "Could not revoke session", cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setRevoking(null);
    }
  }

  return (
    <>
      <SettingsSection
        icon={LockKeyhole}
        title="Password"
        description="Update your password. This revokes saved refresh sessions; already-issued access tokens expire within 15 minutes."
      >
        <form onSubmit={submitPassword} className="border-t border-slate-100 pt-5">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="current">Current password</Label>
              <Input
                id="current"
                type="password"
                autoComplete="current-password"
                value={currentPassword}
                onChange={(event) => setCurrentPassword(event.target.value)}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="new">New password</Label>
              <Input
                id="new"
                type="password"
                autoComplete="new-password"
                minLength={12}
                value={newPassword}
                onChange={(event) => setNewPassword(event.target.value)}
                required
              />
              <p className="text-[11.5px] text-slate-400">Use at least 12 characters.</p>
            </div>
          </div>
          <div className="mt-6 flex justify-end border-t border-slate-100 pt-5">
            <Button type="submit" disabled={savingPassword}>
              {savingPassword ? "Updating…" : "Update password"}
            </Button>
          </div>
        </form>
      </SettingsSection>

      <SettingsSection
        icon={ShieldCheck}
        tone="emerald"
        title="Two-factor authentication"
        description="Require a code from your authenticator app at sign-in."
        aside={
          <>
            <Badge variant="warning">Not available yet</Badge>
            <Switch
              checked={false}
              onCheckedChange={() => {}}
              disabled
              aria-label="Two-factor authentication is unavailable"
            />
          </>
        }
      >
        <p className="text-[12.5px] leading-5 text-slate-500">
          Authenticator enrolment and recovery codes are not implemented by the API yet, so this setting cannot be
          enabled.
        </p>
      </SettingsSection>

      <SettingsSection
        icon={Laptop}
        tone="violet"
        title="Active sessions"
        description="Sessions are stored by the API. Revoking one prevents it from refreshing its access token."
        flush
      >
        {error ? (
          <div className="border-t border-slate-100 px-5 py-6 text-center sm:px-7">
            <p className="text-sm text-rose-700">
              {error instanceof Error ? error.message : "Could not load sessions."}
            </p>
            <Button type="button" variant="secondary" size="sm" className="mt-3" onClick={() => void refetch()}>
              Retry
            </Button>
          </div>
        ) : loading ? (
          <div className="space-y-3 border-t border-slate-100 px-5 py-5 sm:px-7">
            <div className="h-4 w-48 animate-pulse rounded bg-slate-100" />
            <div className="h-4 w-64 animate-pulse rounded bg-slate-100" />
          </div>
        ) : (sessions ?? []).length === 0 ? (
          <div className="border-t border-slate-100 px-5 py-5 sm:px-7">
            <SettingsEmptyState
              icon={Laptop}
              title="No active sessions found"
              description="Your current session may have expired. Sign in again to refresh this list."
            />
          </div>
        ) : (
          <div className="divide-y divide-slate-100 border-t border-slate-100">
            {(sessions ?? []).map((session) => (
              <div
                key={session.id}
                className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 sm:px-7"
              >
                <div className="flex items-start gap-3">
                  <span className="mt-0.5 flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
                    <Laptop className="size-4" />
                  </span>
                  <div>
                    <p className="flex items-center gap-2 text-[13.5px] font-medium text-slate-900">
                      {session.current ? "This device" : "Signed-in session"}
                      {session.current ? <Badge variant="success">Current</Badge> : null}
                    </p>
                    <p className="mt-0.5 text-[12px] text-slate-500">
                      Started {formatDateTime(session.created_at)} · Expires {formatDateTime(session.expires_at)}
                    </p>
                  </div>
                </div>
                {!session.current ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    disabled={revoking === session.id}
                    onClick={() => void signOutSession(session.id)}
                  >
                    {revoking === session.id ? "Revoking…" : "Revoke"}
                  </Button>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </SettingsSection>

      <SettingsSection
        icon={FileText}
        title="Login history"
        description="Credential changes, new sign-ins and key operations are recorded for audit."
      >
        <Link
          href="/settings/audit-logs"
          className="inline-flex items-center gap-2 text-[13px] font-semibold text-brand-700 transition-colors hover:text-brand-800"
        >
          <KeyRound className="size-4" />
          Open Audit Logs
        </Link>
      </SettingsSection>
    </>
  );
}
