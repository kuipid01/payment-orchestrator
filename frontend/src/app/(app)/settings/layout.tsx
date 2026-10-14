"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ArrowLeft,
  Building2,
  CreditCard,
  FileText,
  ShieldCheck,
  UserRound,
  Users,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * Each settings area describes itself, so the page heading, the navigation entry
 * and the copy under the heading cannot drift apart.
 */
interface SettingsTab {
  href: string;
  label: string;
  icon: LucideIcon;
  description: string;
  mvpRequired?: boolean;
}

const TABS: SettingsTab[] = [
  {
    href: "/settings/profile",
    label: "Profile",
    icon: UserRound,
    description: "Manage your personal details and how your team sees you.",
    mvpRequired: true,
  },
  {
    href: "/settings/organization",
    label: "Organization",
    icon: Building2,
    description: "Your workspace identity, region and configuration.",
    mvpRequired: true,
  },
  // {
  //   href: "/settings/team",
  //   label: "Team",
  //   icon: Users,
  //   description: "Manage members and their access to your workspace.",
  //   mvpRequired: true,
  // },
  {
    href: "/settings/security",
    label: "Security",
    icon: ShieldCheck,
    description: "Keep your account and organization secure.",
    mvpRequired: true,
  },
  // Notifications stay hidden until preferences are persisted by the API.
  // {
  //   href: "/settings/notifications",
  //   label: "Notifications",
  //   icon: Bell,
  //   description: "Choose which updates reach your inbox.",
  // },
  {
    href: "/settings/billing",
    label: "Billing",
    icon: CreditCard,
    description: "Manage your subscription, plan and usage.",
  },
  {
    href: "/settings/audit-logs",
    label: "Audit Logs",
    icon: FileText,
    description: "Review security events and workspace activity.",
    mvpRequired: true,
  },
];

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  // The longest matching href wins, so /settings/profile marks one entry active
  // rather than every prefix of the path.
  const active =
    TABS.filter((tab) => pathname === tab.href || pathname.startsWith(`${tab.href}/`)).sort(
      (a, b) => b.href.length - a.href.length,
    )[0] ?? TABS[0];

  return (
    <div className="animate-fade-in">
      <div className="lg:grid lg:grid-cols-[248px_minmax(0,1fr)] lg:gap-8">
        {/* Settings navigation: a sidebar on desktop, a scrollable strip on mobile. */}
        <aside className="mb-6 lg:mb-0">
          <Link
            href="/dashboard"
            className="mb-5 inline-flex items-center gap-2 text-[13px] text-slate-500 transition-colors hover:text-slate-900"
          >
            <ArrowLeft className="size-4" />
            Back to dashboard
          </Link>

          <div className="mb-4 hidden px-1 lg:block">
            <h1 className="font-display text-xl text-slate-950">Settings</h1>
            <p className="mt-1 text-[13px] leading-5 text-slate-500">
              Manage your profile, organization and security.
            </p>
          </div>

          <nav
            aria-label="Settings"
            className="-mx-1 flex gap-1 overflow-x-auto px-1 pb-1 lg:mx-0 lg:sticky lg:top-24 lg:flex-col lg:overflow-visible lg:px-0 lg:pb-0"
          >
            {TABS.map(({ href, label, icon: Icon, mvpRequired }) => {
              const isActive = href === active.href;
              return (
                <Link
                  key={href}
                  href={href}
                  aria-current={isActive ? "page" : undefined}
                  className={cn(
                    "relative flex shrink-0 items-center gap-2.5 rounded-xl px-3.5 py-2.5 text-[13.5px] font-medium transition-colors",
                    isActive
                      ? "bg-brand-50 text-brand-800"
                      : "text-slate-600 hover:bg-slate-100 hover:text-slate-900",
                  )}
                >
                  {isActive ? (
                    <span className="absolute inset-y-2 left-0 hidden w-0.75 rounded-r-full bg-brand-500 lg:block" />
                  ) : null}
                  <Icon
                    className={cn("size-4.5 shrink-0", isActive ? "text-brand-600" : "text-slate-400")}
                    strokeWidth={1.8}
                  />
                  <span>{label}</span>
                 
                </Link>
              );
            })}
          </nav>
        </aside>

        {/* The page itself. */}
        <main className="min-w-0">
          <div className="mb-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">
              Settings
            </p>
            <h2 className="mt-2 font-display text-[28px] leading-tight text-slate-950">
              {active.label}
            </h2>
            <p className="mt-1.5 text-sm text-slate-500">{active.description}</p>
          </div>

          <div className="space-y-4">{children}</div>
        </main>
      </div>
    </div>
  );
}
