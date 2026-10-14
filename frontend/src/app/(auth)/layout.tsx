"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import { Check, Lock } from "lucide-react";
import { Logo } from "@/components/ui/logo";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  if (pathname === "/login" || pathname === "/register") {
    return children;
  }

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      {/* Form side */}
      <div className="flex flex-col px-5 py-8 sm:px-10">
        <Link href="/" aria-label="PayConnect home">
          <Logo />
        </Link>
        <div className="flex flex-1 items-center justify-center py-10">
          <div className="w-full max-w-sm">{children}</div>
        </div>
        <p className="text-center text-xs text-slate-400">
          © 2026 PayConnect, Inc.
        </p>
      </div>

      {/* Brand side */}
      <div className="relative hidden overflow-hidden border-l border-brand-100 bg-[#FDF8E7] lg:block">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(55%_45%_at_75%_10%,rgba(231,176,0,0.22),transparent_70%)]"
        />
        <div className="relative flex h-full flex-col justify-center px-14">
          <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-[var(--shadow-soft)]">
            <div className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3">
              <div className="flex items-center gap-3">
                <span className="flex size-8 items-center justify-center rounded-lg bg-sky-100 text-[10px] font-bold text-sky-700">PS</span>
                <div>
                  <p className="text-[13px] font-semibold text-slate-900">Paystack</p>
                  <p className="font-mono text-[11px] text-slate-400">sk_live_••••••••••••9d21</p>
                </div>
              </div>
              <span className="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700">
                <Check className="size-3" /> Connected
              </span>
            </div>
            <div className="mx-auto h-4 w-px bg-slate-200" />
            <div className="flex items-center gap-3 rounded-xl border border-brand-200 bg-brand-50/70 px-4 py-3">
              <span className="flex size-8 items-center justify-center rounded-lg bg-brand-500 text-slate-950">
                <Lock className="size-4" />
              </span>
              <div>
                <p className="text-[13px] font-semibold text-brand-900">PayConnect Secure Vault</p>
                <p className="text-[11px] text-brand-700/80">Encrypted · never exposed to developers</p>
              </div>
            </div>
            <div className="mx-auto h-4 w-px bg-slate-200" />
            <div className="flex items-center gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3">
              <span className="flex size-8 items-center justify-center rounded-lg bg-brand-100 font-mono text-[9px] font-semibold text-brand-700">API</span>
              <div>
                <p className="text-[13px] font-semibold text-slate-900">Developer API key</p>
                <p className="font-mono text-[11px] text-slate-400">vpay_live_••••9d21</p>
              </div>
            </div>
          </div>

          <div className="mt-8">
            <p className="text-[15px] font-medium text-slate-900">Your provider credentials never leave the vault.</p>
            <p className="mt-1 text-[13.5px] leading-relaxed text-slate-500">
              Developers integrate with scoped PayConnect keys — Paystack, Flutterwave and Stripe secrets stay
              encrypted and out of your codebase.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
