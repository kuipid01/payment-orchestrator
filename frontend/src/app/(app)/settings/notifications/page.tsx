"use client";

import { Bell, CreditCard, ShieldCheck, type LucideIcon } from "lucide-react";
import { Switch } from "@/components/ui";
import {
  NotConnectedNotice,
  SettingsRow,
  SettingsRows,
  SettingsSection,
} from "@/components/app/settings/section";

/**
 * The preferences a merchant will be able to choose between. Everything defaults
 * to the behaviour the platform already has, which is why the switches show a
 * position — but nothing is persisted, so they cannot be moved.
 */
const GROUPS: {
  title: string;
  description: string;
  icon: LucideIcon;
  tone: "brand" | "emerald" | "violet";
  items: { id: string; label: string; description: string; on: boolean }[];
}[] = [
  {
    title: "Payments",
    description: "What reaches your inbox as money moves.",
    icon: CreditCard,
    tone: "brand",
    items: [
      {
        id: "payment.successful",
        label: "Payment successful",
        description: "A charge settled with a provider.",
        on: true,
      },
      {
        id: "payment.failed",
        label: "Payment failed",
        description: "A charge was declined or errored at the provider.",
        on: true,
      },
      {
        id: "payment.refunded",
        label: "Refund processed",
        description: "A refund completed, in full or in part.",
        on: true,
      },
      {
        id: "payment.pending_digest",
        label: "Pending payments digest",
        description: "One daily summary instead of an email per pending charge.",
        on: false,
      },
    ],
  },
  {
    title: "Security",
    description: "These affect your vault, so they are on by default.",
    icon: ShieldCheck,
    tone: "emerald",
    items: [
      {
        id: "security.new_login",
        label: "New device sign-in",
        description: "Somebody signed in from a device you have not used before.",
        on: true,
      },
      {
        id: "security.key_rotated",
        label: "Provider credentials rotated",
        description: "A provider secret was stored or rotated.",
        on: true,
      },
      {
        id: "security.api_key_created",
        label: "API key created or revoked",
        description: "A developer key changed state.",
        on: true,
      },
      {
        id: "security.developer_invited",
        label: "Developer invited or revoked",
        description: "Access to your applications changed hands.",
        on: true,
      },
    ],
  },
  {
    title: "Platform",
    description: "Operational alerts for your integrations.",
    icon: Bell,
    tone: "violet",
    items: [
      {
        id: "platform.webhook_failing",
        label: "Webhook endpoint failing",
        description: "Deliveries to an endpoint keep failing.",
        on: true,
      },
      {
        id: "platform.provider_degraded",
        label: "Provider degraded",
        description: "A connected provider is erroring or timing out.",
        on: true,
      },
      {
        id: "platform.product_updates",
        label: "Product updates",
        description: "Occasional news about what shipped.",
        on: false,
      },
    ],
  },
];

export default function NotificationsSettingsPage() {
  return (
    <>
      <NotConnectedNotice>
        Notification preferences are not stored by the API yet. Each switch shows the current
        default behaviour and cannot be changed, so nobody leaves this page believing they
        unsubscribed from something.
      </NotConnectedNotice>

      {GROUPS.map((group) => (
        <SettingsSection
          key={group.title}
          icon={group.icon}
          tone={group.tone}
          title={group.title}
          description={group.description}
          flush
        >
          <SettingsRows>
            {group.items.map((item) => (
              <div key={item.id} className="px-5 sm:px-7">
                <SettingsRow
                  title={item.label}
                  description={item.description}
                  control={
                    <Switch
                      checked={item.on}
                      onCheckedChange={() => {}}
                      disabled
                      aria-label={`Toggle ${item.label}`}
                    />
                  }
                />
              </div>
            ))}
          </SettingsRows>
        </SettingsSection>
      ))}
    </>
  );
}
