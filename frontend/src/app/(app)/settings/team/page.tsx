"use client";

import * as React from "react";
import { UserPlus } from "lucide-react";
import { Badge, Button, Card, PageHeader, Select } from "@/components/ui";
import { useAsync } from "@/hooks/use-async";
import { listTeam } from "@/lib/api";
import { timeAgo } from "@/lib/utils";
import { toast } from "@/components/ui/toaster";

const ROLE_META: Record<string, { variant: "brand" | "success" | "info" | "neutral"; desc: string }> = {
  Owner: { variant: "brand", desc: "Full control including billing and provider credentials" },
  Admin: { variant: "success", desc: "Manage applications, developers and keys" },
  Developer: { variant: "info", desc: "Scoped API access to assigned applications" },
  Viewer: { variant: "neutral", desc: "Read-only access to payments and logs" },
};

export default function TeamSettingsPage() {
  const { data, loading } = useAsync(listTeam, []);
  const [inviteOpen, setInviteOpen] = React.useState(false);
  const [inviteEmail, setInviteEmail] = React.useState("");
  const [role, setRole] = React.useState("Developer");

  function sendInvite() {
    if (!inviteEmail.trim()) {
      toast("warning", "Email required", "Who are you inviting?");
      return;
    }
    setInviteOpen(false);
    toast("success", "Invite sent", `${inviteEmail} invited as ${role}.`);
    setInviteEmail("");
  }

  return (
    <div className="space-y-4">
      <Card>
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <div>
            <h2 className="text-sm font-semibold text-slate-900">Members</h2>
            <p className="text-[12px] text-slate-400">Roles control what each person can manage.</p>
          </div>
          <Button size="sm" onClick={() => setInviteOpen(true)}>
            <UserPlus className="size-4" /> Invite member
          </Button>
        </div>

        {loading ? (
          <div className="divide-y divide-slate-100">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 px-5 py-4">
                <div className="size-9 animate-pulse rounded-full bg-slate-100" />
                <div className="h-3.5 w-40 animate-pulse rounded bg-slate-100" />
                <div className="ml-auto h-5 w-16 animate-pulse rounded-full bg-slate-100" />
              </div>
            ))}
          </div>
        ) : (
          <div className="divide-y divide-slate-100">
            {(data ?? []).map((m) => (
              <div key={m.id} className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
                <div className="flex items-center gap-3">
                  <span className="flex size-9 items-center justify-center rounded-full bg-slate-100 text-[12px] font-semibold text-slate-600">
                    {m.name
                      .split(" ")
                      .map((w) => w[0])
                      .join("")
                      .slice(0, 2)}
                  </span>
                  <div>
                    <p className="text-[13.5px] font-medium text-slate-900">{m.name}</p>
                    <p className="text-[12px] text-slate-400">{m.email}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <p className="hidden text-[12px] text-slate-400 lg:block">
                    {m.lastActiveAt ? `Active ${timeAgo(m.lastActiveAt)}` : "Invited, not yet joined"}
                  </p>
                  <Badge variant={ROLE_META[m.role].variant}>{m.role}</Badge>
                  {m.role !== "Owner" ? (
                    <Button variant="ghost" size="sm" className="text-slate-400">
                      Remove
                    </Button>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card>
        <div className="p-5">
          <h2 className="text-sm font-semibold text-slate-900">Role permissions</h2>
          <div className="mt-4 space-y-3">
            {Object.entries(ROLE_META).map(([roleName, meta]) => (
              <div key={roleName} className="flex items-start gap-3">
                <Badge variant={meta.variant} className="mt-0.5 min-w-20 justify-center">
                  {roleName}
                </Badge>
                <p className="text-[13px] text-slate-500">{meta.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </Card>

      {/* Invite modal */}
      {inviteOpen ? (
        <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-slate-950/40" onClick={() => setInviteOpen(false)} />
          <div className="relative w-full max-w-md rounded-2xl border border-slate-200 bg-white p-6 shadow-[var(--shadow-lift)]">
            <h2 className="text-base font-semibold text-slate-900">Invite a team member</h2>
            <p className="mt-1 text-[13px] text-slate-500">
              They'll get an email with a join link. Developers building integrations should be invited from{" "}
              <span className="font-medium text-slate-700">Developers</span> instead — that flow grants scoped API
              access without org-wide permissions.
            </p>
            <div className="mt-5 space-y-3">
              <input
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                placeholder="teammate@company.com"
                className="h-9.5 w-full rounded-[var(--radius-btn)] border border-slate-200 px-3 text-sm focus:border-brand-400 focus:outline-none focus:ring-2 focus:ring-brand-500/15"
              />
              <Select value={role} onChange={(e) => setRole(e.target.value)}>
                <option>Admin</option>
                <option>Developer</option>
                <option>Viewer</option>
              </Select>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <Button variant="secondary" onClick={() => setInviteOpen(false)}>
                Cancel
              </Button>
              <Button onClick={sendInvite}>Send invite</Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
