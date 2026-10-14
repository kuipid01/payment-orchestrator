"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowUpRight, Menu, X } from "lucide-react";
import { Logo } from "@/components/ui/logo";

const LINKS = [
  { href: "/how-it-works", label: "How it works" },
  { href: "/pricing", label: "Pricing" },
  { href: "/docs", label: "Docs" },
];

export function SiteNav() {
  const [open, setOpen] = React.useState(false);

  return (
    <header className="sticky top-0 z-40 border-b border-slate-100 bg-[#edeff2] backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-5 lg:px-8">
        <Link href="/" aria-label="PayConnect home">
          <Logo />
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {LINKS.map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className="rounded-full px-3.5 py-2 text-[13.5px] font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
            >
              {l.label}
            </Link>
          ))}
        </nav>

        <div className="hidden items-center gap-2.5 md:flex">
          <Link href="/login" className="px-2 text-[13.5px] font-semibold text-slate-700 hover:text-slate-950">
            Login
          </Link>
          <Link
            href="/register"
            className="inline-flex h-9.5 items-center gap-2 rounded-full bg-brand-500 pl-4 pr-1.5 text-[13.5px] font-semibold text-slate-950 transition-colors hover:bg-brand-400"
          >
            Sign up
            <span className="flex size-6.5 items-center justify-center rounded-full bg-slate-950 text-white">
              <ArrowUpRight className="size-3.5" />
            </span>
          </Link>
        </div>

        <button
          className="rounded-lg p-2 text-slate-600 hover:bg-slate-100 md:hidden"
          onClick={() => setOpen((v) => !v)}
          aria-label="Toggle navigation"
          aria-expanded={open}
        >
          {open ? <X className="size-5" /> : <Menu className="size-5" />}
        </button>
      </div>

      {open ? (
        <div className="border-t border-slate-100 bg-white px-5 py-4 md:hidden">
          <nav className="flex flex-col gap-1">
            {LINKS.map((l) => (
              <Link
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className="rounded-lg px-3 py-2.5 text-[15px] text-slate-700 hover:bg-slate-50"
              >
                {l.label}
              </Link>
            ))}
            <div className="mt-3 flex gap-2">
              <Link href="/login" className="flex-1">
                <span className="flex h-11 items-center justify-center rounded-full border border-slate-300 font-semibold text-slate-900">
                  Login
                </span>
              </Link>
              <Link href="/register" className="flex-1">
                <span className="flex h-11 items-center justify-center rounded-full bg-brand-500 font-semibold text-slate-950">
                  Sign up
                </span>
              </Link>
            </div>
          </nav>
        </div>
      ) : null}
    </header>
  );
}
