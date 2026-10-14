"use client";

import * as React from "react";
import { KeyRound, Laptop, MapPin, ShieldCheck, Smartphone } from "lucide-react";
import { Badge, Button, Card, Input, Label, Switch } from "@/components/ui";
import { toast } from "@/components/ui/toaster";

const SESSIONS = [
  { device: "MacBook Pro — Chrome", location: "Lagos, NG · 102.89.44.7", current: true, icon: Laptop },
  { device: "iPhone 15 — PayConnect App", location: "Lagos, NG · 102.89.44.7", current: false, icon: Smartphone },
  { device: "Windows PC — Firefox", location: "London, UK · 193.11.44.2", current: false, icon: Laptop },
];

export default function SecuritySettingsPage() {
  const [twoFa, setTwoFa] = React.useState(true);
  const [savingPw, setSavingPw] = React.useState(false);

  function changePassword(e: React.FormEvent) {
    e.preventDefault();
    setSavingPw(true);
    setTimeout(() => {
      setSavingPw(false);
      toast("success", "Password updated", "Other sessions were signed out.");
    }, 800);
  }

  return (
    <div className="space-y-4">
      <Card>
        <form onSubmit={changePassword} className="p-6">
          <h2 className="text-sm font-semibold text-slate-900">Password</h2>
          <div className="mt-5 grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="current">Current password</Label>
              <Input id="current" type="password" placeholder="••••••••••••" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="new">New password</Label>
              <Input id="new" type="password" placeholder="At least 12 characters" minLength={12} />
            </div>
          </div>
          <div className="mt-5 flex justify-end">
            <Button type="submit" disabled={savingPw}>
              {savingPw ? "Updating…" : "Update password"}
            </Button>
          </div>
        </form>
      </Card>

      <Card>
        <div className="flex items-center justify-between p-6">
          <div>
            <div className="flex items-center gap-2">
              <ShieldCheck className="size-4 text-brand-500" />
              <h2 className="text-sm font-semibold text-slate-900">Two-factor authentication</h2>
              <Badge variant={twoFa ? "success" : "warning"}>{twoFa ? "Enabled" : "Disabled"}</Badge>
            </div>
            <p className="mt-1 max-w-md text-[12.5px] leading-relaxed text-slate-500">
              Require a TOTP code from your authenticator app at sign-in. Strongly recommended for anyone with access
              to provider credentials.
            </p>
          </div>
          <Switch checked={twoFa} onCheckedChange={(v) => { setTwoFa(v); toast("info", v ? "2FA enabled" : "2FA disabled"); }} aria-label="Toggle 2FA" />
        </div>
      </Card>

      <Card>
        <div className="border-b border-slate-100 px-5 py-4">
          <h2 className="text-sm font-semibold text-slate-900">Active sessions</h2>
          <p className="text-[12px] text-slate-400">Sign out any device you don't recognize.</p>
        </div>
        <div className="divide-y divide-slate-100">
          {SESSIONS.map((s) => {
            const Icon = s.icon;
            return (
              <div key={s.device} className="flex items-center justify-between px-5 py-4">
                <div className="flex items-start gap-3">
                  <span className="mt-0.5 flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
                    <Icon className="size-4" />
                  </span>
                  <div>
                    <p className="text-[13.5px] font-medium text-slate-900">
                      {s.device}{" "}
                      {s.current ? (
                        <span className="ml-1 rounded-full bg-emerald-50 px-2 py-0.5 text-[10.5px] font-medium text-emerald-600">
                          This device
                        </span>
                      ) : null}
                    </p>
                    <p className="mt-0.5 flex items-center gap-1.5 text-[12px] text-slate-400">
                      <MapPin className="size-3" /> {s.location}
                    </p>
                  </div>
                </div>
                {!s.current ? (
                  <Button variant="ghost" size="sm" className="text-slate-400" onClick={() => toast("info", "Session revoked")}>
                    Sign out
                  </Button>
                ) : null}
              </div>
            );
          })}
        </div>
      </Card>

      <Card>
        <div className="flex items-start gap-3 p-5">
          <KeyRound className="mt-0.5 size-4 shrink-0 text-slate-400" />
          <p className="text-[12.5px] leading-relaxed text-slate-500">
            Login history and credential events are recorded in{" "}
            <span className="font-medium text-slate-700">Audit Logs</span> and retained for 12 months.
          </p>
        </div>
      </Card>
    </div>
  );
}
