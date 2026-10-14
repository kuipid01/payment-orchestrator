"use client";

import * as React from "react";
import Link from "next/link";
import { MessageSquare, ShieldCheck, UserPlus, Users } from "lucide-react";
import { Badge, Button, Input, Label, Modal, Select, Skeleton } from "@/components/ui";
import { useAuth } from "@/components/providers/auth-provider";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import { addTeamMember, listTeam, removeTeamMember } from "@/lib/api";
import { timeAgo } from "@/lib/utils";
import {
  SettingsRow,
  SettingsRows,
  SettingsSection,
} from "@/components/app/settings/section";

const ROLE_META: Record<string, { variant: "brand" | "success" | "info" | "neutral"; desc: string }> = {
  Owner: { variant: "brand", desc: "Full control including billing and provider credentials" },
  Admin: { variant: "success", desc: "Manage applications, developers and keys" },
  Developer: { variant: "info", desc: "Scoped API access to assigned applications" },
  Viewer: { variant: "neutral", desc: "Read-only access to payments and logs" },
};

export default function TeamSettingsPage() {
  const { user } = useAuth();
  const { data, loading, error, refetch } = useAsync(listTeam, [], ["settings", "team"]);
  const [inviteOpen, setInviteOpen] = React.useState(false);
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState<"Admin" | "Developer" | "Viewer">("Developer");
  const [adding, setAdding] = React.useState(false);
  const [removing, setRemoving] = React.useState<string | null>(null);
  const canManage = user?.role === "Owner" || user?.role === "Admin";

  const members = data ?? [];

  async function addMember(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAdding(true);
    try {
      await addTeamMember(email, role);
      setInviteOpen(false);
      setEmail("");
      await refetch();
      toast("success", "Team member added", `${email} was added as ${role}.`);
    } catch (cause) {
      toast("error", "Could not add team member", cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setAdding(false);
    }
  }

  async function removeMember(id: string, memberEmail: string) {
    setRemoving(id);
    try {
      await removeTeamMember(id);
      await refetch();
      toast("success", "Team member removed", `${memberEmail} no longer has workspace access.`);
    } catch (cause) {
      toast("error", "Could not remove team member", cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setRemoving(null);
    }
  }

  return (
    <>
      <SettingsSection
        icon={Users}
        title="Members"
        description="Workspace members and the role that decides what they can access."
        aside={
          canManage ? (
            <Button size="sm" onClick={() => setInviteOpen(true)}>
              <UserPlus className="size-4" />
              Add member
            </Button>
          ) : null
        }
        flush
      >
        {error ? (
          <div className="border-t border-slate-100 px-5 py-6 text-center sm:px-7">
            <p className="text-sm text-rose-700">
              {error instanceof Error ? error.message : "Could not load team members."}
            </p>
            <Button type="button" variant="secondary" size="sm" className="mt-3" onClick={() => void refetch()}>
              Retry
            </Button>
          </div>
        ) : loading ? (
          <div className="divide-y divide-slate-100 border-t border-slate-100">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 px-5 py-4 sm:px-7">
                <Skeleton className="size-9 rounded-full" />
                <Skeleton className="h-3.5 w-40" />
                <Skeleton className="ml-auto h-5 w-16 rounded-full" />
              </div>
            ))}
          </div>
        ) : (
          <SettingsRows>
            {members.map((m) => (
              <div key={m.id} className="px-5 sm:px-7">
                <SettingsRow
                  title={
                    <span className="flex items-center gap-3">
                      <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-slate-100 text-[11.5px] font-semibold text-slate-600">
                        {m.name
                          .split(" ")
                          .map((w) => w[0])
                          .join("")
                          .slice(0, 2)}
                      </span>
                      {m.name}
                    </span>
                  }
                  description={m.email}
                  control={
                    <span className="flex items-center gap-3">
                      <span className="hidden text-[12px] text-slate-400 lg:block">
                        {m.lastActiveAt ? `Active ${timeAgo(m.lastActiveAt)}` : "Invited, not yet joined"}
                      </span>
                      <Badge variant={ROLE_META[m.role]?.variant ?? "neutral"}>{m.role}</Badge>
                      {canManage && m.role !== "Owner" ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-slate-500"
                          disabled={removing === m.id}
                          onClick={() => void removeMember(m.id, m.email)}
                        >
                          {removing === m.id ? "Removing…" : "Remove"}
                        </Button>
                      ) : null}
                    </span>
                  }
                />
              </div>
            ))}
          </SettingsRows>
        )}
      </SettingsSection>

      <SettingsSection
        icon={ShieldCheck}
        tone="violet"
        title="Role permissions"
        description="What each role can do. Roles are derived from the membership the API returns, so a client cannot promote itself."
        flush
      >
        <SettingsRows>
          {Object.entries(ROLE_META).map(([roleName, meta]) => (
            <div key={roleName} className="px-5 sm:px-7">
              <SettingsRow
                title={
                  <Badge variant={meta.variant} className="min-w-20 justify-center">
                    {roleName}
                  </Badge>
                }
                description={meta.desc}
              />
            </div>
          ))}
        </SettingsRows>
      </SettingsSection>

      <p className="flex items-start gap-2 px-1 text-[12px] leading-5 text-slate-400">
        <MessageSquare className="mt-0.5 size-3.5 shrink-0" />
        <span>
          Building an integration? Developers should be invited from{" "}
          <Link href="/developers/invite" className="font-medium text-brand-700 hover:underline">
            Developers
          </Link>{" "}
          instead — that flow grants scoped API access without org-wide permissions.
        </span>
      </p>

      <Modal
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        title="Add a team member"
        description="Add an existing PayConnect account to this workspace."
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setInviteOpen(false)} disabled={adding}>
              Cancel
            </Button>
            <Button type="submit" form="add-team-member" disabled={adding || !email.trim()}>
              {adding ? "Adding…" : "Add member"}
            </Button>
          </div>
        }
      >
        <form id="add-team-member" onSubmit={addMember} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="team-email">Account email</Label>
            <Input
              id="team-email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="teammate@company.com"
            />
            <p className="text-[12px] text-slate-500">
              The person must already have a verified PayConnect account. Access is granted as soon as you add them.
            </p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="team-role">Role</Label>
            <Select id="team-role" value={role} onChange={(event) => setRole(event.target.value as typeof role)}>
              <option value="Admin">Admin</option>
              <option value="Developer">Developer</option>
              <option value="Viewer">Viewer</option>
            </Select>
          </div>
        </form>
      </Modal>
    </>
  );
}
