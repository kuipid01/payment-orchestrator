"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  BookOpen,
  FlaskConical,
  Globe,
  KeyRound,
  LayoutDashboard,
  Lock,
  PanelsTopLeft,
  Receipt,
  Settings,
  Users,
  Webhook,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Logo } from "@/components/ui/logo";
import { Badge } from "@/components/ui";
import { PlanBadge } from "./plan-badge";

const NAV: {
  section: string;
  items: { href: string; label: string; icon: React.ReactNode; badge?: string }[];
}[] = [
  {
    section: "Overview",
    items: [{ href: "/dashboard", label: "Dashboard", icon: <LayoutDashboard className="size-4" /> }],
  },
  {
    section: "Money",
    items: [{ href: "/payments", label: "Payments", icon: <Receipt className="size-4" /> }],
  },
  {
    section: "Infrastructure",
    items: [
      { href: "/providers", label: "Providers", icon: <Globe className="size-4" /> },
      { href: "/applications", label: "Applications", icon: <PanelsTopLeft className="size-4" /> },
      { href: "/developers", label: "Developers", icon: <Users className="size-4" /> },
    ],
  },
  {
    section: "Developer platform",
    items: [
      { href: "/api-keys", label: "API Keys", icon: <KeyRound className="size-4" /> },
      { href: "/webhooks", label: "Webhooks", icon: <Webhook className="size-4" /> },
      { href: "/logs", label: "Logs", icon: <BookOpen className="size-4" /> },
      { href: "/test-arena", label: "Test Arena", icon: <FlaskConical className="size-4" />, badge: "new" },
    ],
  },
  {
    section: "Organization",
    items: [{ href: "/settings", label: "Settings", icon: <Settings className="size-4" /> }],
  },
];

export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();

  // Resolve the single best nav match for the current path. The most specific
  // href wins, so a nested route like /settings/profile highlights one entry
  // rather than lighting up both it and the parent /settings item.
  const activeHref = React.useMemo(() => {
    return NAV.flatMap((group) => group.items)
      .map((item) => item.href)
      .filter((href) => pathname === href || pathname.startsWith(href + "/"))
      .sort((a, b) => b.length - a.length)[0];
  }, [pathname]);

  return (
    <nav className="flex flex-col gap-6 px-3 py-4">
      {NAV.map((group) => (
        <div key={group.section}>
          <p className="mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            {group.section}
  </p>
          <div className="flex flex-col gap-0.5">
            {group.items.map((item) => {
              const active = activeHref === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={onNavigate}
                  className={cn(
                    "flex items-center gap-2.5 rounded-full px-3 py-2 text-[13.5px] font-medium transition-colors",
                    active
                      ? "bg-brand-500 text-slate-950 shadow-[0_1px_2px_rgba(231,176,0,0.45)]"
                      : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                  )}
                >
                  <span className={active ? "text-slate-950" : "text-slate-400"}>{item.icon}</span>
                  {item.label}
                  {item.badge && (
                    <span className="ml-auto rounded-full bg-amber-100 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wide text-amber-700">
                      {item.badge}
                    </span>
                  )}
                </Link>
              );
            })}
          </div>
        </div>
      ))}
    </nav>
  );
}

export function Sidebar({ className }: { className?: string }) {
  return (
    <aside className={cn("flex h-full w-60 shrink-0 flex-col border-r border-slate-100 bg-white", className)}>
      <div className="flex h-16 items-center border-b border-slate-100 px-5">
        <Link href="/dashboard">
          <Logo />
        </Link>
      </div>
      <div className="flex-1 overflow-y-auto">
        <SidebarNav />
      </div>
      <div className="space-y-2 border-t border-slate-100 p-4">
        {/* The plan is furniture, not a page: it stays visible so a limit is
            never the first a merchant hears of what they are on. */}
        <PlanBadge />
        <div className="flex items-start gap-2.5 rounded-xl border border-brand-100 bg-brand-50/60 p-3">
          <Lock className="mt-0.5 size-4 shrink-0 text-brand-500" />
          <p className="text-[12px] leading-snug text-brand-900">
            Vault active — provider secrets are encrypted and never exposed to developers.
          </p>
        </div>
      </div>
    </aside>
  );
}

export function MobileSidebar({ open, onClose }: { open: boolean; onClose: () => void }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 lg:hidden">
      <div className="absolute inset-0 bg-slate-950/40" onClick={onClose} />
      <div className="absolute left-0 top-0 flex h-full w-72 flex-col bg-white shadow-xl">
        <div className="flex h-16 shrink-0 items-center justify-between border-b border-slate-100 px-5">
          <Logo />
          <button onClick={onClose} className="rounded-md p-1.5 text-slate-400 hover:bg-slate-100" aria-label="Close menu">
            <X className="size-5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          <SidebarNav onNavigate={onClose} />
        </div>
        <div className="shrink-0 border-t border-slate-100 p-4">
          <PlanBadge />
        </div>
      </div>
    </div>
  );
}
