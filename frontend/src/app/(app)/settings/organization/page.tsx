"use client";

import * as React from "react";
import { Copy } from "lucide-react";
import { Button, Card, Input, Label, Separator } from "@/components/ui";
import { toast } from "@/components/ui/toaster";

export default function OrganizationSettingsPage() {
  const [saving, setSaving] = React.useState(false);

  function onSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setTimeout(() => {
      setSaving(false);
      toast("success", "Organization updated");
    }, 700);
  }

  return (
    <div className="space-y-4">
      <Card>
        <form onSubmit={onSave} className="p-6">
          <h2 className="text-sm font-semibold text-slate-900">Organization</h2>
          <p className="mt-0.5 text-[12.5px] text-slate-400">Your workspace identity.</p>

          <div className="mt-6 space-y-1.5">
            <Label htmlFor="org-name">Organization name</Label>
            <Input id="org-name" defaultValue="Acme Inc" />
          </div>

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="org-id">Organization ID</Label>
              <div className="flex gap-2">
                <Input id="org-id" value="org_9x41k2m8" readOnly className="font-mono bg-slate-50" />
                <Button
                  type="button"
                  variant="secondary"
                  size="icon"
                  aria-label="Copy organization ID"
                  onClick={() => {
                    navigator.clipboard.writeText("org_9x41k2m8");
                    toast("info", "Copied", "Organization ID copied.");
                  }}
                >
                  <Copy className="size-3.5" />
                </Button>
              </div>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="created">Created</Label>
              <Input id="created" value="June 14, 2026" readOnly className="bg-slate-50" />
            </div>
          </div>

          <Separator className="my-6" />
          <h3 className="text-[13px] font-semibold text-slate-900">Region</h3>
          <p className="mt-0.5 text-[12.5px] text-slate-400">
            Vault region: <span className="font-mono text-slate-700">eu-west-1</span> — credentials never leave this
            region.
          </p>

          <div className="mt-6 flex justify-end">
            <Button type="submit" disabled={saving}>
              {saving ? "Saving…" : "Save changes"}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
