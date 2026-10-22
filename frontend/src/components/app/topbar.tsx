"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Bell, CircleHelp, LogOut, Menu, Search } from "lucide-react";
import { Logo } from "@/components/ui/logo";
import { useAuth } from "@/components/providers/auth-provider";
import { MobileSidebar } from "./sidebar";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function AccountMenu() {
  const { user, organization, signOut } = useAuth();
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [signingOut, setSigningOut] = React.useState(false);
  const ref = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [open]);

  async function onSignOut() {
    setSigningOut(true);
    await signOut();
    router.replace("/login");
  }

  if (!user) return null;

  const subtitle = [user.role, organization].filter(Boolean).join(" · ");

  return (
    <div className="relative ml-1 flex items-center gap-2.5 border-l border-slate-100 pl-3" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Account menu"
        className="flex items-center gap-2.5 rounded-lg py-1 pr-1 text-left hover:bg-slate-50"
      >
        <span className="flex size-8 items-center justify-center rounded-full bg-brand-100 text-[12px] font-semibold text-brand-700">
          {initials(user.name)}
        </span>
        <span className="hidden leading-tight sm:block">
          <span className="block text-[13px] font-medium text-slate-900">{user.name}</span>
          <span className="block text-[11px] text-slate-400">{subtitle || user.email}</span>
        </span>
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full z-50 mt-2 w-56 overflow-hidden rounded-xl border border-slate-200 bg-white p-1 shadow-[var(--shadow-soft)]"
        >
          <div className="px-3 py-2">
            <p className="truncate text-[13px] font-medium text-slate-900">{user.name}</p>
            <p className="truncate text-[11.5px] text-slate-400">{user.email}</p>
          </div>
          <div className="my-1 h-px bg-slate-100" />
          <button
            type="button"
            role="menuitem"
            onClick={onSignOut}
            disabled={signingOut}
            className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-[13px] text-slate-700 hover:bg-slate-50 disabled:opacity-60"
          >
            <LogOut className="size-4" />
            {signingOut ? "Signing out…" : "Sign out"}
          </button>
        </div>
      )}
    </div>
  );
}

export function Topbar() {
  return (
    <header className="sticky top-0 z-40 flex h-16 items-center gap-2 border-b border-slate-100 bg-white/90 px-3 backdrop-blur sm:gap-3 sm:px-4 lg:px-8">
      {/* Mobile: menu + logo */}
      <MobileMenu />
      <Link href="/dashboard" className="min-w-0 lg:hidden">
        <Logo />
      </Link>

      {/* Search */}
      <div className="relative mx-auto hidden w-full max-w-md md:block">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
        <input
          placeholder="Search payments, applications, developers…"
          className="h-9 w-full rounded-[var(--radius-btn)] border border-slate-200 bg-slate-50/60 pl-9 pr-14 text-sm placeholder:text-slate-400 hover:border-slate-300 focus:border-brand-400 focus:bg-white focus:outline-none focus:ring-2 focus:ring-brand-500/15"
        />
        <kbd className="absolute right-2.5 top-1/2 hidden -translate-y-1/2 rounded border border-slate-200 bg-white px-1.5 py-0.5 font-mono text-[10px] text-slate-400 lg:block">
          ⌘K
        </kbd>
      </div>

      <div className="ml-auto flex shrink-0 items-center gap-1">
        <Link
          href="/docs"
          className="hidden items-center gap-1.5 rounded-lg px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 hover:text-slate-900 sm:flex"
        >
          <CircleHelp className="size-4" />
          Docs
        </Link>
        <button
          className="relative rounded-lg p-2 text-slate-500 hover:bg-slate-100 hover:text-slate-700"
          aria-label="Notifications"
        >
          <Bell className="size-4.5" />
          <span className="absolute right-1.5 top-1.5 size-2 rounded-full bg-brand-500 ring-2 ring-white" />
        </button>
        <AccountMenu />
      </div>
    </header>
  );
}

function MobileMenu() {
  const [open, setOpen] = React.useState(false);

  // Resizing to the desktop breakpoint hides the drawer; close it too, so it
  // does not keep the page scroll-locked behind a hidden panel.
  React.useEffect(() => {
    const desktop = window.matchMedia("(min-width: 1024px)");
    const onChange = () => {
      if (desktop.matches) setOpen(false);
    };
    desktop.addEventListener("change", onChange);
    return () => desktop.removeEventListener("change", onChange);
  }, []);

  return (
    <>
      <button
        type="button"
        className="-ml-1 flex size-9 shrink-0 items-center justify-center rounded-lg text-slate-600 hover:bg-slate-100 lg:hidden"
        onClick={() => setOpen(true)}
        aria-label="Open menu"
        aria-expanded={open}
      >
        <Menu className="size-5" />
      </button>
      <MobileSidebar open={open} onClose={() => setOpen(false)} />
    </>
  );
}
