"use client";

import * as React from "react";
import { Building2, Fingerprint, Globe2 } from "lucide-react";
import { Badge, Button, CopyField, Input, Label } from "@/components/ui";
import { useAuth } from "@/components/providers/auth-provider";
import { useAsync } from "@/hooks/use-async";
import { toast } from "@/components/ui/toaster";
import { getMerchantProfile, updateMerchantProfile } from "@/lib/api";
import {
  NotConnectedNotice,
  SettingsRow,
  SettingsRows,
  SettingsSection,
} from "@/components/app/settings/section";

export default function OrganizationSettingsPage() {
  const { user, refresh } = useAuth();
  const { data, loading, error, refetch } = useAsync(
    getMerchantProfile,
    [],
    ["settings", "merchant"],
  );
  const [name, setName] = React.useState("");
  const [saving, setSaving] = React.useState(false);
  const canManage = user?.role === "Owner" || user?.role === "Admin";

  React.useEffect(() => {
    setName(data?.business_name ?? "");
  }, [data?.business_name]);

  async function saveOrganization(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    try {
      await updateMerchantProfile({ business_name: name });
      await Promise.all([refetch(), refresh()]);
      toast("success", "Organization updated");
    } catch (cause) {
      toast("error", "Could not update organization", cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <SettingsSection
        icon={Building2}
        title="Workspace identity"
        description="The name your developers see when they accept an invitation, and on every dashboard screen."
        notice={!canManage ? <>Only an organization owner or admin can rename this workspace.</> : undefined}
      >
        <form onSubmit={saveOrganization} className="border-t border-slate-100 pt-5">
          <div className="space-y-1.5">
            <Label htmlFor="org-name">Organization name</Label>
            {error ? (
              <p className="text-[12px] text-rose-700">
                {error instanceof Error ? error.message : "Could not load workspace details."}{" "}
                <button type="button" className="font-medium underline" onClick={() => void refetch()}>
                  Retry
                </button>
              </p>
            ) : null}
            <Input
              id="org-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              maxLength={120}
              required
              disabled={!canManage || loading || Boolean(error)}
              className="sm:max-w-md"
            />
          </div>

          <div className="mt-6 flex justify-end border-t border-slate-100 pt-5">
            <Button type="submit" disabled={!canManage || saving || loading || Boolean(error)}>
              {saving ? "Saving…" : "Save changes"}
            </Button>
          </div>
        </form>
      </SettingsSection>

      <SettingsSection
        icon={Fingerprint}
        tone="violet"
        title="Identifiers"
        description="Stable values for this workspace. Safe to quote in a support request or a webhook log."
        flush
      >
        <SettingsRows>
          <div className="px-5 sm:px-7">
            <SettingsRow
              title="Workspace ID"
              description="Used by the API in logs, events and support conversations."
              control={
                <CopyField value={user?.merchant_id ?? ""} className="sm:w-80" />
              }
            />
          </div>
          <div className="px-5 sm:px-7">
            <SettingsRow
              title="Your role"
              description="Owner is the only role that can manage billing and provider credentials."
              control={<Badge variant="brand">{user?.role || "—"}</Badge>}
            />
          </div>
        </SettingsRows>
      </SettingsSection>

    </>
  );
}
