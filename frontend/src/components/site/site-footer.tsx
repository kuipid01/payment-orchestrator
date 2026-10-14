import Link from "next/link";
import { ArrowUpRight, LockKeyhole, ShieldCheck } from "lucide-react";
import { Logo } from "@/components/ui/logo";

const LINKS: { title: string; links: { label: string; href: string }[] }[] = [
  {
    title: "Product",
    links: [
      { label: "How it works", href: "/how-it-works" },
      { label: "Pricing", href: "/pricing" },
      { label: "Status", href: "/maintenance" },
    ],
  },
  {
    title: "Developers",
    links: [
      { label: "Documentation", href: "/docs" },
      { label: "Quickstart", href: "/docs/quickstart" },
      { label: "API reference", href: "/docs/payments" },
      { label: "Webhooks", href: "/docs/webhooks" },
    ],
  },
  {
    title: "Resources",
    links: [
      { label: "Providers", href: "/docs/providers" },
      { label: "Errors", href: "/docs/errors" },
      { label: "Support", href: "/docs" },
    ],
  },
];

export function SiteFooter() {
  return (
    <footer className="bg-slate-100 px-3 py-4 sm:px-6 sm:py-6">
      <div className="mx-auto grid max-w-7xl gap-3 rounded-3xl border border-slate-200 bg-white p-3 sm:gap-4 sm:p-4 lg:grid-cols-[0.9fr_1.1fr_0.95fr]">
        <section className="flex min-h-64 flex-col justify-between rounded-2xl bg-slate-50 p-5 sm:min-h-72 sm:p-7">
          <div>
            <Logo />
            <p className="mt-4 max-w-xs text-sm leading-6 text-slate-600">
              One secure connection for your payment providers, applications,
              and development team.
            </p>
          </div>

          <div className="mt-8">
            <Link
              href="/register"
              className="inline-flex h-10 items-center gap-2 rounded-full bg-brand-500 pl-4 pr-1.5 text-[13px] font-semibold text-slate-950 transition-colors hover:bg-brand-400"
            >
              Get started
              <span className="flex size-7 items-center justify-center rounded-full bg-slate-950 text-white">
                <ArrowUpRight className="size-3.5" />
              </span>
            </Link>
            <Link
              href="/docs/quickstart"
              className="mt-4 flex w-fit items-center gap-2 text-xs font-medium text-slate-500 transition-colors hover:text-brand-800"
            >
              Start with the quickstart <ArrowUpRight className="size-3.5" />
            </Link>
          </div>
        </section>

        <section className="relative flex min-h-64 flex-col justify-between overflow-hidden rounded-2xl border border-brand-100 bg-brand-50/50 p-5 sm:min-h-72 sm:p-7">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -right-14 -top-16 size-64 rounded-full border border-brand-200/70 sm:size-72"
          />
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -right-2 -top-4 size-44 rounded-full border border-brand-200/60 sm:size-52"
          />
          <div
            aria-hidden="true"
            className="pointer-events-none absolute right-9 top-8 size-24 rounded-full border border-brand-200/60 sm:right-12 sm:top-10 sm:size-28"
          />

          <span className="relative flex size-11 items-center justify-center rounded-xl bg-brand-500 text-slate-950">
            <LockKeyhole className="size-5" />
          </span>

          <div className="relative mt-12 max-w-sm">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-brand-800">
              Our approach to privacy
            </p>
            <h2 className="mt-2 text-2xl font-semibold leading-tight tracking-tight text-slate-950 sm:text-3xl">
              Your provider secrets stay yours.
            </h2>
            <Link
              href="/how-it-works"
              className="mt-4 inline-flex items-center gap-1.5 text-xs font-semibold text-brand-800 transition-colors hover:text-brand-900"
            >
              Learn how the vault works <ArrowUpRight className="size-3.5" />
            </Link>
          </div>

          <ShieldCheck
            aria-hidden="true"
            className="absolute bottom-6 right-6 size-5 text-brand-700/70"
          />
        </section>

        <div className="grid min-h-64 gap-3 sm:min-h-72 sm:gap-4">
          <nav
            aria-label="Footer"
            className="grid grid-cols-2 gap-x-4 gap-y-6 rounded-2xl bg-slate-50 p-5 sm:grid-cols-3 sm:p-6"
          >
            {LINKS.map((column) => (
              <div key={column.title}>
                <h3 className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-500">
                  {column.title}
                </h3>
                <ul className="mt-3 space-y-2">
                  {column.links.map((link) => (
                    <li key={link.label}>
                      <Link
                        href={link.href}
                        className="text-xs text-slate-700 transition-colors hover:text-brand-800"
                      >
                        {link.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </nav>

          <div className="flex flex-col justify-between gap-3 rounded-2xl bg-slate-50 p-5 sm:flex-row sm:items-end sm:p-6">
            <div>
              <p className="text-xs font-semibold text-slate-800">
                PayConnect by Nile
              </p>
              <p className="mt-1 text-[11px] text-slate-500">
                © 2026 PayConnect, Inc. All rights reserved.
              </p>
            </div>
            <p className="font-mono text-[10px] text-slate-400">
              Your secrets never leave the vault.
            </p>
          </div>
        </div>
      </div>
    </footer>
  );
}
