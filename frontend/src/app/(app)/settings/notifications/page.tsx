"use client";

import * as React from "react";
import { Card, Switch } from "@/components/ui";
import { toast } from "@/components/ui/toaster";

const GROUPS: { title: string; description: string; items: { id: string; label: string; def: boolean }[] }[] = [
  {
    title: "Payments",
    description: "What lands in your inbox as money moves.",
    items: [
      { id: "payment.successful", label: "Payment successful", def: true },
      { id: "payment.failed", label: "Payment failed", def: true },
      { id: "payment.refunded", label: "Refund processed", def: true },
      { id: "payment.pending_digest", label: "Pending payments digest (daily)", def: false },
    ],
  },
  {
    title: "Security",
    description: "Always recommended — these affect your vault.",
    items: [
      { id: "security.new_login", label: "New device sign-in", def: true },
      { id: "security.key_rotated", label: "Provider credentials rotated", def: true },
      { id: "security.api_key_created", label: "API key created or revoked", def: true },
      { id: "security.developer_invited", label: "Developer invited or revoked", def: true },
    ],
  },
  {
    title: "Platform",
    description: "Operational alerts for your integrations.",
    items: [
      { id: "platform.webhook_failing", label: "Webhook endpoint failing", def: true },
      { id: "platform.provider_degraded", label: "Provider degraded or erroring", def: true },
      { id: "platform.product_updates", label: "Product updates", def: false },
    ],
  },
];

export default function NotificationsSettingsPage() {
  const [prefs, setPrefs] = React.useState<Record<string, boolean>>(() =>
    Object.fromEntries(GROUPS.flatMap((g) => g.items.map((i) => [i.id, i.def])))
  );

  return (
    <div className="space-y-4">
      {GROUPS.map((group) => (
        <Card key={group.title}>
          <div className="border-b border-slate-100 px-5 py-4">
            <h2 className="text-sm font-semibold text-slate-900">{group.title}</h2>
            <p className="text-[12px] text-slate-400">{group.description}</p>
          </div>
          <div className="divide-y divide-slate-100">
            {group.items.map((item) => (
              <div key={item.id} className="flex items-center justify-between px-5 py-3.5">
                <p className="text-[13.5px] text-slate-700">{item.label}</p>
                <Switch
                  checked={prefs[item.id]}
                  onCheckedChange={(v) => {
                    setPrefs((prev) => ({ ...prev, [item.id]: v }));
                    toast("info", v ? "Subscribed" : "Unsubscribed", item.label);
                  }}
                  aria-label={`Toggle ${item.label}`}
                />
              </div>
            ))}
          </div>
        </Card>
      ))}
    </div>
  );
}
