"use client";

import * as React from "react";
import { Camera } from "lucide-react";
import { Button, Card, Input, Label } from "@/components/ui";
import { toast } from "@/components/ui/toaster";

export default function ProfileSettingsPage() {
  const [saving, setSaving] = React.useState(false);

  function onSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setTimeout(() => {
      setSaving(false);
      toast("success", "Profile saved");
    }, 700);
  }

  return (
    <div className="space-y-4">
      <Card>
        <form onSubmit={onSave} className="p-6">
          <h2 className="text-sm font-semibold text-slate-900">Profile</h2>
          <p className="mt-0.5 text-[12.5px] text-slate-400">How you appear to your team.</p>

          <div className="mt-6 flex items-center gap-4">
            <span className="flex size-16 items-center justify-center rounded-full bg-brand-100 text-lg font-semibold text-brand-700">
              JM
            </span>
            <Button type="button" variant="secondary" size="sm">
              <Camera className="size-3.5" /> Change avatar
            </Button>
          </div>

          <div className="mt-6 grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="name">Full name</Label>
              <Input id="name" defaultValue="John Merchant" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" defaultValue="john@mycompany.com" />
            </div>
          </div>

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
