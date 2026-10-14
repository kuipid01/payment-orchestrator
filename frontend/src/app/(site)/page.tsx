"use client";

import * as React from "react";
import Link from "next/link";
import {
  ArrowRight,
  ArrowUpRight,
  BookOpen,
  Check,
  CheckCircle2,
  Code2,
  CreditCard,
  KeyRound,
  Lock,
  LockKeyhole,
  ShieldCheck,
  Webhook,
  Zap,
} from "lucide-react";
import { cn } from "@/lib/utils";

const TRUST = [
  { icon: Zap, value: "15 min", label: "to first test payment" },
  { icon: ShieldCheck, value: "AES-256", label: "vault encryption at rest" },
  { icon: Webhook, value: "1 API", label: "for every provider" },
  { icon: KeyRound, value: "0", label: "secrets in your codebase" },
];

function PrimaryLink({
  href,
  children,
  className,
}: {
  href: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <Link
      href={href}
      className={cn(
        "inline-flex min-h-12 items-center justify-center gap-3 rounded-md bg-brand-500 px-5 text-sm font-semibold text-slate-950 transition-colors hover:bg-brand-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2",
        className,
      )}
    >
      {children}
      <ArrowUpRight aria-hidden="true" className="size-4" />
    </Link>
  );
}

function LandingProductPreview() {
  return (
    <div className="relative isolate overflow-hidden rounded-4xl border border-slate-200 bg-slate-100 p-4 shadow-[0_28px_90px_-42px_rgba(15,23,42,0.24)] sm:p-7 lg:p-9">
      <div aria-hidden="true" className="pointer-events-none absolute -right-20 -top-28 -z-10 size-80 rounded-full border-42 border-white/40" />
      <div aria-hidden="true" className="pointer-events-none absolute -bottom-40 -left-24 -z-10 size-96 rounded-full bg-white/50 blur-3xl" />

      <div className="mx-auto grid max-w-5xl items-center gap-3 md:grid-cols-[1fr_1.15fr_1fr] md:gap-4">
        <article className="rounded-2xl border border-white/80 bg-white p-4 shadow-[0_18px_45px_-28px_rgba(15,23,42,0.28)] sm:p-5">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="flex size-8 items-center justify-center rounded-lg bg-brand-50 text-brand-700">
                <CreditCard className="size-4" />
              </span>
              <p className="text-xs font-semibold text-slate-800">Payment activity</p>
            </div>
            <span className="rounded-full bg-emerald-50 px-2 py-1 text-[10px] font-semibold text-emerald-700">Test mode</span>
          </div>
          <div className="mt-5 rounded-xl bg-slate-50 p-3.5">
            <p className="text-[11px] text-slate-500">Latest transaction</p>
            <p className="mt-1 text-xl font-semibold tracking-tight text-slate-950">₦24,500.00</p>
            <div className="mt-3 flex items-center justify-between border-t border-slate-200 pt-3 text-[10px]">
              <span className="font-mono text-slate-500">pay_8f21…</span>
              <span className="inline-flex items-center gap-1 font-medium text-emerald-700"><CheckCircle2 className="size-3" /> Successful</span>
            </div>
          </div>
          <div className="mt-4 flex h-14 items-end gap-1.5">
            {[28, 43, 34, 62, 49, 72, 55, 86, 67, 96, 76, 100].map((height, index) => (
              <span key={index} className="flex-1 rounded-t-sm bg-brand-400/80" style={{ height: `${height}%` }} />
            ))}
          </div>
        </article>

        <article className="relative z-10 rounded-2xl border border-white/90 bg-white p-4 shadow-[0_24px_60px_-28px_rgba(15,23,42,0.3)] sm:p-5 md:-my-4">
          <div className="flex items-center justify-between gap-3 border-b border-slate-100 pb-4">
            <div>
              <p className="text-xs font-semibold text-slate-900">Secure provider vault</p>
              <p className="mt-1 text-[10px] text-slate-500">Credentials stay private by design</p>
            </div>
            <span className="flex size-9 items-center justify-center rounded-xl bg-brand-100 text-brand-800"><Lock className="size-4" /></span>
          </div>
          <div className="mt-4 space-y-2.5">
            {[
              { name: "Paystack", key: "sk_live_••••••••9d21" },
              { name: "Flutterwave", key: "FLWSECK_••••••••7a3c" },
            ].map((provider) => (
              <div key={provider.name} className="flex items-center justify-between gap-3 rounded-xl border border-slate-100 px-3 py-3">
                <div className="min-w-0">
                  <p className="text-xs font-semibold text-slate-800">{provider.name}</p>
                  <p className="mt-1 truncate font-mono text-[10px] text-slate-400">{provider.key}</p>
                </div>
                <span className="size-2 shrink-0 rounded-full bg-emerald-500" aria-label="Connected" />
              </div>
            ))}
          </div>
          <div className="mt-4 flex items-center gap-2 rounded-xl bg-brand-50 px-3 py-2.5 text-[10px] font-medium text-brand-800">
            <ShieldCheck className="size-3.5 shrink-0" />
            Encrypted · never exposed to developers
          </div>
        </article>

        <article className="rounded-2xl border border-white/80 bg-white p-4 shadow-[0_18px_45px_-28px_rgba(15,23,42,0.28)] sm:p-5">
          <div className="flex items-center justify-between gap-3">
            <p className="text-xs font-semibold text-slate-800">Provider health</p>
            <span className="text-[10px] font-medium text-slate-400">2 connected</span>
          </div>
          <div className="mt-4 space-y-2">
            {[
              { name: "Paystack", value: "Healthy", icon: "P" },
              { name: "Flutterwave", value: "Healthy", icon: "F" },
              { name: "Stripe", value: "Add provider", icon: "+" },
            ].map((provider, index) => (
              <div key={provider.name} className="flex items-center gap-2.5 rounded-xl bg-slate-50 px-3 py-3">
                <span className={cn("flex size-7 shrink-0 items-center justify-center rounded-lg text-[10px] font-bold", index === 2 ? "bg-white text-slate-400" : "bg-brand-100 text-brand-800")}>
                  {provider.icon}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[11px] font-semibold text-slate-800">{provider.name}</p>
                  <p className={cn("mt-0.5 text-[10px]", index === 2 ? "text-slate-400" : "text-emerald-700")}>{provider.value}</p>
                </div>
                {index !== 2 && <CheckCircle2 className="size-4 shrink-0 text-emerald-600" />}
              </div>
            ))}
          </div>
          <Link href="/providers" className="mt-4 inline-flex items-center gap-1 text-[11px] font-semibold text-brand-700 hover:text-brand-800">
            Manage providers <ArrowRight className="size-3.5" />
          </Link>
        </article>
      </div>
      <p className="mt-5 text-center text-[11px] font-medium text-slate-600">
        One integration. A clearer view of every provider.
      </p>
    </div>
  );
}

export default function LandingPage() {
  return (
    <main className="bg-white text-slate-950">
      {/* Hero */}
      <section className="border-b bg-[#edeff2] border-slate-200">
        <div className="mx-auto max-w-7xl px-5 py-14 sm:px-8 sm:py-16 lg:py-20">
          <div className="mx-auto max-w-4xl text-center">
            <div className="inline-flex items-center gap-2 rounded-full border border-brand-200 bg-brand-50 px-3.5 py-2 text-xs font-semibold text-brand-800">
              <Lock className="size-3.5" /> Payment infrastructure with a secure vault
            </div>
            <h1 className="mt-6 text-4xl font-extrabold uppercase leading-[0.98] tracking-[-0.055em] text-slate-950 sm:text-6xl lg:text-7xl">
              Every provider.
              <span className="mt-1 block text-brand-800">
                One secure connection.
              </span>
            </h1>
            <p className="mx-auto mt-5 max-w-2xl text-base leading-7 text-slate-600 sm:text-lg sm:leading-8">
              Connect Paystack, Flutterwave and Stripe once. Give developers a single API while PayConnect protects credentials, helps route payments and keeps an audit trail.
            </p>
            <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
              <PrimaryLink href="/register">Get started free</PrimaryLink>
              <Link href="/docs" className="inline-flex min-h-12 items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-5 text-sm font-semibold text-slate-900 transition-colors hover:border-brand-300 hover:bg-brand-50/50">
                <BookOpen className="size-4" /> Read the documentation <ArrowUpRight className="size-4" />
              </Link>
            </div>
          </div>

          <div className="mx-auto mt-12 max-w-6xl">
            <LandingProductPreview />
          </div>

          <div className="mx-auto mt-8 grid max-w-5xl grid-cols-2 gap-5 border-t border-slate-200 pt-6 sm:grid-cols-4 sm:gap-8">
            {TRUST.map(({ icon: Icon, value, label }) => (
              <div key={value} className="flex items-start gap-3">
                <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-700">
                  <Icon className="size-4" aria-hidden="true" />
                </span>
                <div>
                  <p className="text-lg font-semibold tracking-tight text-slate-950">{value}</p>
                  <p className="mt-0.5 text-xs leading-5 text-slate-500">{label}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Feature overview */}
      <section className="mx-auto max-w-6xl px-5 py-16 sm:px-8 sm:py-24">
        <div className="mx-auto max-w-xl text-center">
          <p className="text-sm leading-6 text-slate-600 sm:text-base">
            Payment providers make it possible to move money. Managing each
            integration, credential and developer handoff shouldn&apos;t slow
            your team down.
          </p>
          <h2 className="mt-10 text-3xl font-semibold leading-tight tracking-tight text-slate-950 sm:mt-14 sm:text-4xl">
            PayConnect brings it all together,
            <span className="block text-brand-700"> securely and simply.</span>
          </h2>
        </div>

        <div className="mt-12 grid grid-cols-2 gap-4 sm:mt-16 sm:gap-6 lg:gap-8">
          <article className="min-w-0">
            <div className="relative flex aspect-[1.15] items-center justify-center overflow-hidden rounded-2xl bg-slate-50 shadow-[0_12px_38px_-25px_rgba(15,23,42,0.28)] sm:aspect-[1.35]">
              <div aria-hidden="true" className="absolute size-32 rounded-full border border-brand-300/70 sm:size-44" />
              <div aria-hidden="true" className="absolute size-24 rounded-full border border-brand-200 sm:size-32" />
              <div aria-hidden="true" className="absolute size-16 rounded-full border border-brand-100 sm:size-20" />
              <span className="relative flex size-12 items-center justify-center rounded-2xl bg-brand-100 text-brand-800 sm:size-16">
                <Lock className="size-6 sm:size-8" />
              </span>
            </div>
            <div className="px-1 pt-4 sm:px-2 sm:pt-5">
              <h3 className="text-sm font-semibold leading-5 text-slate-950 sm:text-base">Provider keys stay private</h3>
              <p className="mt-1.5 text-xs leading-5 text-slate-600 sm:text-sm">
                Keep credentials encrypted in one vault, out of chats and code.
              </p>
            </div>
          </article>

          <article className="min-w-0">
            <div className="relative flex aspect-[1.15] items-center justify-center overflow-hidden rounded-2xl bg-slate-50 p-3 shadow-[0_12px_38px_-25px_rgba(15,23,42,0.28)] sm:aspect-[1.35] sm:p-6">
              <div className="w-full max-w-64 rounded-xl border border-white bg-white p-3 shadow-sm sm:p-4">
                <div className="mb-3 flex items-center justify-between">
                  <span className="text-[10px] font-semibold text-slate-800 sm:text-xs">Provider connections</span>
                  <span className="size-2 rounded-full bg-emerald-500" />
                </div>
                {[
                  ["P", "Paystack"],
                  ["F", "Flutterwave"],
                  ["S", "Stripe"],
                ].map(([initial, provider]) => (
                  <div key={provider} className="mt-2 flex items-center gap-2 rounded-lg bg-slate-50 px-2 py-2 sm:gap-3 sm:px-3">
                    <span className="flex size-6 items-center justify-center rounded-md bg-brand-100 text-[9px] font-bold text-brand-800 sm:size-7 sm:text-[10px]">{initial}</span>
                    <span className="text-[10px] font-medium text-slate-700 sm:text-xs">{provider}</span>
                    <CheckCircle2 className="ml-auto size-3.5 text-emerald-600 sm:size-4" />
                  </div>
                ))}
              </div>
            </div>
            <div className="px-1 pt-4 sm:px-2 sm:pt-5">
              <h3 className="text-sm font-semibold leading-5 text-slate-950 sm:text-base">One API, multiple providers</h3>
              <p className="mt-1.5 text-xs leading-5 text-slate-600 sm:text-sm">
                Build once and manage your payment integrations from one place.
              </p>
            </div>
          </article>

          <article className="min-w-0">
            <div className="relative flex aspect-[1.15] items-center justify-center overflow-hidden rounded-2xl bg-slate-50 p-3 shadow-[0_12px_38px_-25px_rgba(15,23,42,0.28)] sm:aspect-[1.35] sm:p-6">
              <div className="w-full max-w-64 rounded-xl border border-white bg-white p-3 shadow-sm sm:p-4">
                <div className="flex items-center gap-2 border-b border-slate-100 pb-3">
                  <span className="flex size-7 items-center justify-center rounded-lg bg-brand-100 text-brand-800"><Code2 className="size-3.5" /></span>
                  <div>
                    <p className="text-[10px] font-semibold text-slate-800 sm:text-xs">Application access</p>
                    <p className="font-mono text-[8px] text-slate-400 sm:text-[10px]">vpay_test_••••••21</p>
                  </div>
                </div>
                <div className="mt-3 space-y-2">
                  {["Payments: read and write", "Webhooks: read"].map((permission) => (
                    <p key={permission} className="flex items-center gap-1.5 text-[8px] text-slate-600 sm:text-[10px]">
                      <Check className="size-3 text-emerald-600" /> {permission}
                    </p>
                  ))}
                </div>
              </div>
            </div>
            <div className="px-1 pt-4 sm:px-2 sm:pt-5">
              <h3 className="text-sm font-semibold leading-5 text-slate-950 sm:text-base">Access that fits each team</h3>
              <p className="mt-1.5 text-xs leading-5 text-slate-600 sm:text-sm">
                Give developers scoped access without handing over provider secrets.
              </p>
            </div>
          </article>

          <article className="min-w-0">
            <div className="relative flex aspect-[1.15] items-center justify-center overflow-hidden rounded-2xl bg-slate-50 p-3 shadow-[0_12px_38px_-25px_rgba(15,23,42,0.28)] sm:aspect-[1.35] sm:p-6">
              <div className="w-full max-w-64 rounded-xl border border-white bg-white p-3 shadow-sm sm:p-4">
                <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                  <span className="text-[10px] font-semibold text-slate-800 sm:text-xs">Latest activity</span>
                  <span className="size-2 rounded-full bg-emerald-500" />
                </div>
                <div className="mt-3 space-y-3">
                  {[
                    ["Payment received", "Just now"],
                    ["Provider connected", "Today"],
                    ["Developer invited", "Yesterday"],
                  ].map(([event, time]) => (
                    <div key={event} className="flex items-center gap-2">
                      <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-brand-50 text-brand-700"><CheckCircle2 className="size-3" /></span>
                      <span className="min-w-0 flex-1 truncate text-[9px] font-medium text-slate-700 sm:text-[10px]">{event}</span>
                      <span className="text-[8px] text-slate-400 sm:text-[9px]">{time}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
            <div className="px-1 pt-4 sm:px-2 sm:pt-5">
              <h3 className="text-sm font-semibold leading-5 text-slate-950 sm:text-base">Know what changed</h3>
              <p className="mt-1.5 text-xs leading-5 text-slate-600 sm:text-sm">
                Follow payments, integrations and team access in one clear view.
              </p>
            </div>
          </article>
        </div>

        <div className="mt-12 text-center sm:mt-16">
          <PrimaryLink href="/register">Get started free</PrimaryLink>
        </div>
      </section>

      {/* Product story */}
      <section aria-labelledby="product-story-heading">
        <div className="mx-auto grid max-w-6xl items-center gap-10 px-5 py-16 sm:px-8 sm:py-20 md:grid-cols-2 md:gap-14">
          <div className="max-w-lg">
            <span className="mb-5 inline-flex size-10 items-center justify-center rounded-xl bg-brand-100 text-brand-800">
              <CreditCard className="size-5" />
            </span>
            <p className="text-xs font-semibold uppercase tracking-[0.15em] text-brand-800">
              Clear payment operations
            </p>
            <h2 id="product-story-heading" className="mt-3 text-3xl font-semibold leading-tight tracking-tight text-slate-950 sm:text-4xl">
              Know what&apos;s happening with every payment.
            </h2>
            <p className="mt-4 text-sm leading-7 text-slate-600 sm:text-base">
              See transaction activity, provider health and integration events
              from one workspace. Give your team the context to act without
              juggling separate dashboards.
            </p>
            <Link href="/dashboard" className="mt-6 inline-flex items-center gap-2 text-sm font-semibold text-brand-800 transition-colors hover:text-brand-900">
              Explore the dashboard <ArrowRight className="size-4" />
            </Link>
          </div>

          <div className="relative flex min-h-80 items-center justify-center overflow-hidden rounded-3xl bg-slate-100 px-5 pt-8 sm:min-h-96">
            <div aria-hidden="true" className="absolute -right-14 -top-16 size-60 rounded-full border border-brand-200/70 sm:size-72" />
            <div className="relative w-52 rounded-4xl border-[7px] border-slate-900 bg-slate-950 p-2 shadow-[0_24px_55px_-20px_rgba(15,23,42,0.4)] sm:w-60">
              <div className="overflow-hidden rounded-[1.4rem] bg-[#01474A] text-white">
                <div className="flex items-center justify-between px-4 pb-3 pt-5 text-[9px] text-white/70">
                  <span>9:41</span>
                  <span className="h-1.5 w-10 rounded-full bg-slate-950/70" />
                  <span>•••</span>
                </div>
                <div className="px-4 pb-5">
                  <p className="text-center text-[10px] font-semibold">Payment health</p>
                  <div className="mt-5 rounded-xl bg-white/10 p-3">
                    <p className="text-[9px] text-white/60">Successful payments</p>
                    <p className="mt-1 text-2xl font-semibold">98.6%</p>
                    <div className="mt-3 flex h-10 items-end gap-1">
                      {[30, 45, 35, 62, 51, 78, 60, 90, 72, 100].map((height, index) => (
                        <span key={index} className="flex-1 rounded-t-sm bg-brand-400" style={{ height: `${height}%` }} />
                      ))}
                    </div>
                  </div>
                  <div className="mt-3 space-y-2">
                    {["Paystack connected", "Vault protected"].map((item) => (
                      <div key={item} className="flex items-center gap-2 rounded-lg bg-white/10 px-2.5 py-2 text-[9px]">
                        <CheckCircle2 className="size-3.5 text-emerald-300" />
                        {item}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
            <span className="absolute bottom-6 left-4 rounded-full border border-white bg-white/90 px-3 py-1.5 text-[10px] font-medium text-slate-600 shadow-sm sm:bottom-8 sm:left-8">
              Activity, all in one place
            </span>
          </div>
        </div>

        <div className="relative overflow-hidden bg-emerald-900 text-white">
          <div aria-hidden="true" className="pointer-events-none absolute -right-28 -top-32 size-96 rounded-full border-42 border-white/5" />
          <div className="mx-auto max-w-6xl px-5 pb-12 pt-14 sm:px-8 sm:pb-16 sm:pt-20">
            <div className="max-w-2xl">
              <span className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/5 px-3 py-1.5 text-[11px] font-medium text-brand-200">
                <ShieldCheck className="size-3.5" /> Secure by design
              </span>
              <h2 className="mt-5 max-w-xl text-3xl font-semibold leading-tight tracking-tight sm:text-5xl">
                Never share your provider keys again.
              </h2>
              <p className="mt-4 max-w-lg text-sm leading-6 text-white/70 sm:text-base">
                Your providers connect to PayConnect. Your developers work with
                scoped API keys. Sensitive credentials stay encrypted in the
                vault.
              </p>
            </div>

            <div className="relative z-10 mt-9 grid overflow-hidden rounded-2xl border border-slate-200 bg-white text-slate-900 shadow-[0_28px_70px_-32px_rgba(0,0,0,0.5)] md:grid-cols-[0.9fr_1.1fr]">
              <div className="border-b border-slate-200 p-4 sm:p-6 md:border-b-0 md:border-r">
                <div className="flex items-center justify-between">
                  <p className="text-xs font-semibold text-slate-800">Recent payments</p>
                  <span className="rounded-full bg-brand-50 px-2 py-1 text-[9px] font-semibold text-brand-800">Test mode</span>
                </div>
                <div className="mt-4 flex items-end justify-between gap-2">
                  <div>
                    <p className="text-[10px] text-slate-500">Payment volume</p>
                    <p className="mt-1 text-2xl font-semibold tracking-tight">₦2,759.00</p>
                  </div>
                  <span className="text-[10px] font-medium text-emerald-700">+12.8%</span>
                </div>
                <div className="mt-4 flex h-20 items-end gap-1.5 rounded-xl bg-slate-50 px-3 pt-3">
                  {[25, 36, 30, 54, 45, 63, 49, 75, 60, 83, 70, 100].map((height, index) => (
                    <span key={index} className="flex-1 rounded-t-sm bg-brand-400/90" style={{ height: `${height}%` }} />
                  ))}
                </div>
                <div className="mt-4 space-y-2">
                  {[
                    ["Payment received", "₦1,250.00"],
                    ["Payment received", "₦840.00"],
                  ].map(([label, amount], index) => (
                    <div key={`${label}-${index}`} className="flex items-center justify-between border-t border-slate-100 pt-2.5 text-[10px]">
                      <span className="text-slate-600">{label}</span>
                      <span className="font-semibold text-slate-800">{amount}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="bg-slate-50 p-4 sm:p-6">
                <div className="flex items-center justify-between">
                  <p className="text-xs font-semibold text-slate-800">Provider security</p>
                  <ShieldCheck className="size-4 text-emerald-700" />
                </div>
                <div className="mt-4 space-y-2.5">
                  {[
                    ["Paystack credentials", "Encrypted"],
                    ["Developer access", "Scoped"],
                    ["Audit history", "Recording"],
                  ].map(([label, value]) => (
                    <div key={label} className="flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-3 py-3">
                      <div className="flex min-w-0 items-center gap-2">
                        <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-brand-50 text-brand-800">
                          {label.startsWith("Paystack") ? <LockKeyhole className="size-3.5" /> : <CheckCircle2 className="size-3.5" />}
                        </span>
                        <span className="truncate text-[10px] font-medium text-slate-700">{label}</span>
                      </div>
                      <span className="shrink-0 rounded-full bg-emerald-50 px-2 py-1 text-[9px] font-medium text-emerald-700">{value}</span>
                    </div>
                  ))}
                </div>
                <p className="mt-4 text-[10px] leading-5 text-slate-500">
                  Credentials remain protected while your team builds, tests and ships.
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
