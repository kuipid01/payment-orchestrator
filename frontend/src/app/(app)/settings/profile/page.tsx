"use client";

import * as React from "react";
import { BadgeCheck, Building2, Mail, UserRound } from "lucide-react";
import { Badge, Button, Input, Label } from "@/components/ui";
import { useAuth } from "@/components/providers/auth-provider";
import { toast } from "@/components/ui/toaster";
import { updateProfile } from "@/lib/api";
import {
  SettingsRow,
  SettingsRows,
  SettingsSection,
} from "@/components/app/settings/section";

/** Two letters for the avatar, from the name the session actually carries. */
function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export default function ProfileSettingsPage() {
  const { user, organization, refresh } = useAuth();
  const [name, setName] = React.useState("");
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    setName(user?.name ?? "");
  }, [user?.name]);

  async function saveProfile(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    try {
      await updateProfile(name);
      await refresh();
      toast("success", "Profile updated");
    } catch (cause) {
      toast("error", "Could not update profile", cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <SettingsSection
        icon={UserRound}
        title="Your profile"
        description="How you appear to your team across the dashboard."
      >
        <form onSubmit={saveProfile} className="border-t border-slate-100 pt-5">
          <div className="flex items-center gap-4">
            <span className="flex size-14 items-center justify-center rounded-full bg-brand-100 text-base font-semibold text-brand-700">
              {initials(name)}
            </span>
            <div className="min-w-0">
              <p className="text-[13.5px] font-medium text-slate-800">{name || "—"}</p>
              <p className="mt-0.5 truncate text-[12.5px] text-slate-500">{user?.email || "—"}</p>
            </div>
          </div>

          <div className="mt-6 grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="name">Full name</Label>
              <Input
                id="name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                maxLength={120}
                required
                autoComplete="name"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={user?.email ?? ""}
                readOnly
                disabled
                className="bg-slate-50/70"
              />
              <p className="text-[11.5px] text-slate-400">Email changes require a verification flow.</p>
            </div>
          </div>

          <div className="mt-6 flex justify-end border-t border-slate-100 pt-5">
          <Button type="submit" disabled={saving || !user}>
            {saving ? "Saving…" : "Save changes"}
          </Button>
          </div>
        </form>
      </SettingsSection>

      <SettingsSection
        icon={Building2}
        tone="violet"
        title="Membership"
        description="Where this identity sits in your organization today, as the API reports it."
        flush
      >
        <SettingsRows>
          <div className="px-5 sm:px-7">
            <SettingsRow
              title="Organization"
              description="The workspace this session is acting in."
              control={
                <span className="text-[13px] font-medium text-slate-800">
                  {organization || "—"}
                </span>
              }
            />
          </div>
          <div className="px-5 sm:px-7">
            <SettingsRow
              title="Role"
              description="Your permissions come from this membership, not from the browser."
              control={<Badge variant="brand">{user?.role || "—"}</Badge>}
            />
          </div>
          <div className="px-5 sm:px-7">
            <SettingsRow
              title="Email verification"
              description="Verified addresses can take live payments and invite developers."
              control={
                user?.email_verified ? (
                  <Badge variant="success">
                    <BadgeCheck className="size-3.5" />
                    Verified
                  </Badge>
                ) : (
                  <Badge variant="warning">Not verified</Badge>
                )
              }
            />
          </div>
          <div className="px-5 sm:px-7">
            <SettingsRow
              title="Merchant ID"
              description="Quote this when contacting support about your workspace."
              control={
                <code className="rounded-md border border-slate-200 bg-slate-50 px-2.5 py-1.5 font-mono text-[12px] text-slate-600">
                  {user?.merchant_id || "—"}
                </code>
              }
            />
          </div>
        </SettingsRows>
      </SettingsSection>

      <p className="flex items-center gap-2 px-1 text-[12px] text-slate-400">
        <Mail className="size-3.5" />
        Change of email address is deliberately not self-service — it needs the verification flow
        first.
      </p>
    </>
  );
}
