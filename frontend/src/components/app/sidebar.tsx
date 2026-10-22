"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  BookOpen,
  FlaskConical,
  Globe,
  KeyRound,
  LayoutDashboard,
  PanelLeftClose,
  PanelLeftOpen,
  PanelsTopLeft,
  Receipt,
  Settings,
  Users,
  Webhook,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Logo } from "@/components/ui/logo";
import { PlanBadge } from "./plan-badge";

const NAV: {
  section: string;
  items: { href: string; label: string; icon: React.ReactNode; badge?: string }[];
}[] = [
  {
    section: "Overview",
    items: [{ href: "/dashboard", label: "Dashboard", icon: <LayoutDashboard className="size-4.5" /> }],
  },
  {
    section: "Money",
    items: [{ href: "/payments", label: "Payments", icon: <Receipt className="size-4.5" /> }],
  },
  {
    section: "Infrastructure",
    items: [
      { href: "/providers", label: "Providers", icon: <Globe className="size-4.5" /> },
      { href: "/applications", label: "Applications", icon: <PanelsTopLeft className="size-4.5" /> },
      { href: "/developers", label: "Developers", icon: <Users className="size-4.5" /> },
    ],
  },
  {
    section: "Developer platform",
    items: [
      { href: "/api-keys", label: "API Keys", icon: <KeyRound className="size-4.5" /> },
      { href: "/webhooks", label: "Webhooks", icon: <Webhook className="size-4.5" /> },
      { href: "/logs", label: "Logs", icon: <BookOpen className="size-4.5" /> },
      { href: "/test-arena", label: "Test Arena", icon: <FlaskConical className="size-4.5" /> },
    ],
  },
  {
    section: "Organization",
    items: [{ href: "/settings", label: "Settings", icon: <Settings className="size-4.5" /> }],
  },
];

export function SidebarNav({
  onNavigate,
  collapsed = false,
}: {
  onNavigate?: () => void;
  /** Desktop icon rail: keep the icons, drop the labels and section headings. */
  collapsed?: boolean;
}) {
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
    <nav className={cn("flex flex-col py-4", collapsed ? "gap-3 px-2" : "gap-6 px-3")}>
      {NAV.map((group) => (
        <div key={group.section}>
          {collapsed ? (
            <div className="mx-2 mb-3 h-px bg-slate-100" aria-hidden />
          ) : (
            <p className="mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wider text-slate-400">
              {group.section}
            </p>
          )}
          <div className={cn("flex flex-col", collapsed ? "gap-1" : "gap-0.5")}>
            {group.items.map((item) => {
              const active = activeHref === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={onNavigate}
                  // The label is the only affordance in the rail, so expose it
                  // through the native tooltip and to screen readers.
                  title={collapsed ? item.label : undefined}
                  aria-label={collapsed ? item.label : undefined}
                  aria-current={active ? "page" : undefined}
                  className={cn(
                    "group relative flex items-center rounded-full text-[13.5px] font-medium transition-colors",
                    collapsed ? "size-10 justify-center self-center" : "gap-2.5 px-3 py-2",
                    active
                      ? "bg-brand-500 text-slate-950 shadow-[0_1px_2px_rgba(231,176,0,0.45)]"
                      : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                  )}
                >
                  <span
                    className={cn(
                      "flex shrink-0",
                      active ? "text-slate-950" : "text-slate-400 group-hover:text-slate-600"
                    )}
                  >
                    {item.icon}
                  </span>
                  {!collapsed && <span className="truncate">{item.label}</span>}
                  {item.badge &&
                    (collapsed ? (
                      <span className="absolute right-1.5 top-1.5 size-1.5 rounded-full bg-amber-500" />
                    ) : (
                      <span className="ml-auto rounded-full bg-amber-100 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wide text-amber-700">
                        {item.badge}
                      </span>
                    ))}
                </Link>
              );
            })}
          </div>
        </div>
      ))}
    </nav>
  );
}

const COLLAPSE_KEY = "payconnect.sidebar.collapsed";

export function Sidebar({ className }: { className?: string }) {
  const [collapsed, setCollapsed] = React.useState(false);

  // Read after mount so the server and first client render agree, then the
  // merchant's choice survives navigation and reloads.
  React.useEffect(() => {
    try {
      setCollapsed(window.localStorage.getItem(COLLAPSE_KEY) === "1");
    } catch {
      /* storage can be unavailable in private mode; the default is fine */
    }
  }, []);

  const toggleCollapsed = React.useCallback(() => {
    setCollapsed((value) => {
      const next = !value;
      try {
        window.localStorage.setItem(COLLAPSE_KEY, next ? "1" : "0");
      } catch {
        /* ignore */
      }
      return next;
    });
  }, []);

  return (
    <aside
      className={cn(
        "flex h-full shrink-0 flex-col border-r border-slate-100 bg-white transition-[width] duration-200 ease-out",
        collapsed ? "w-[72px]" : "w-60",
        className
      )}
    >
      <div
        className={cn(
          "flex shrink-0 border-b border-slate-100",
          collapsed ? "flex-col items-center justify-center gap-1 px-2 py-3" : "h-16 items-center justify-between pl-5 pr-2"
        )}
      >
        <Link href="/dashboard" className="min-w-0" aria-label="PayConnect dashboard">
          <Logo mark={collapsed} />
        </Link>
        <button
          type="button"
          onClick={toggleCollapsed}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-expanded={!collapsed}
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          className="flex size-8 shrink-0 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700"
        >
          {collapsed ? <PanelLeftOpen className="size-4.5" /> : <PanelLeftClose className="size-4.5" />}
        </button>
      </div>
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        <SidebarNav collapsed={collapsed} />
      </div>
      <div className={cn("shrink-0 border-t border-slate-100", collapsed ? "flex justify-center p-2" : "p-4")}>
        {/* The plan is furniture, not a page: it stays visible so a limit is
            never the first a merchant hears of what they are on. */}
        <PlanBadge collapsed={collapsed} />
      </div>
    </aside>
  );
}

export function MobileSidebar({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  React.useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    // Stop the page behind the drawer from scrolling while it is open.
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = previousOverflow;
    };
  }, [open, onClose]);

  if (!open || !mounted) return null;

  // Portalled to <body>: the topbar that owns the trigger uses backdrop-blur,
  // and a filtered ancestor becomes the containing block for fixed children —
  // which previously clamped this full-screen drawer to the header's height.
  return createPortal(
    <div className="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true" aria-label="Navigation">
      <div className="absolute inset-0 bg-slate-950/40" onClick={onClose} />
      <div className="absolute left-0 top-0 flex h-full w-[85vw] max-w-72 flex-col bg-white shadow-xl">
        <div className="flex h-16 shrink-0 items-center justify-between border-b border-slate-100 pl-5 pr-3">
          <Logo />
          <button
            type="button"
            onClick={onClose}
            className="flex size-9 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-700"
            aria-label="Close menu"
          >
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
    </div>,
    document.body
  );
}
