import * as React from "react";
import type { LucideIcon } from "lucide-react";
import { Globe2, ShieldCheck } from "lucide-react";

import { AuthBrand } from "@/components/auth/auth-brand";
import { cn } from "@/lib/utils";

/**
 * The authentication shell.
 *
 * Sign-in and sign-up were redesigned into a split screen — a product panel on
 * the left, a card on the right — while the recovery, verification and
 * invitation screens kept the older centred layout, so the flow changed shape
 * halfway through. Every auth screen now renders this shell, which is the
 * redesigned look in one place: one brand mark, one card, one footer.
 */
export function AuthShell({
  aside,
  children,
  footnote,
  contentClassName,
}: {
  /** Product panel content, below the brand mark. Hidden below `lg`. */
  aside?: React.ReactNode;
  /** The card's content, below the brand mark. */
  children: React.ReactNode;
  /** Small print under the card. */
  footnote?: React.ReactNode;
  /** Width of the card column. */
  contentClassName?: string;
}) {
  return (
    <main className="min-h-screen bg-white text-slate-950">
      <div className="grid min-h-screen lg:grid-cols-[minmax(0,1.02fr)_minmax(480px,0.98fr)]">
        {/* Left: brand and product panel */}
        <aside className="relative hidden min-h-screen flex-col overflow-hidden border-r border-brand-100/80 bg-linear-to-br from-brand-50 via-white to-brand-100/70 px-10 py-9 lg:flex xl:px-16">
          <AuthBrand />
          <div className="relative z-10 mx-auto mt-12 w-full max-w-lg lg:my-auto lg:py-16">
            {aside}
          </div>
          <AuthAsideFooter />
        </aside>

        {/* Right: the task itself */}
        <section className="flex min-h-screen min-w-0 flex-col px-5 py-6 sm:px-8 lg:px-10 xl:px-16">
          <header className="flex items-center justify-between lg:hidden">
            <AuthBrand />
          </header>

          <div className="flex flex-1 items-center justify-center py-10 sm:py-14 lg:py-8">
            <div className={cn("w-full max-w-115", contentClassName)}>
              <div className="rounded-2xl border border-slate-200/80 bg-white p-6 shadow-[0_20px_70px_-45px_rgba(15,23,42,0.22)] sm:p-9">
                <div className="hidden lg:block">
                  <AuthBrand />
                </div>
                <div className="mt-2 lg:mt-9">{children}</div>
              </div>
              {footnote}
            </div>
          </div>

          <footer className="border-t border-slate-100 py-4 text-center text-[11px] text-slate-400">
            PayConnect · Secure payment infrastructure
          </footer>
        </section>
      </div>
    </main>
  );
}

/** The standing footer of the product panel. */
function AuthAsideFooter() {
  return (
    <div className="relative z-10 mt-10 flex items-center justify-between border-t border-slate-200 pt-5 text-[11px] text-slate-500 lg:mt-0">
      <span>Built for developers and growing businesses.</span>
      <span className="hidden items-center gap-1.5 sm:inline-flex">
        <Globe2 className="size-3.5" />
        Available worldwide
      </span>
    </div>
  );
}

export interface AuthAsidePoint {
  icon: LucideIcon;
  title: string;
  description: string;
}

/**
 * The product panel body: a claim, a sentence of support, and the reasons the
 * page can be trusted with what it is about to ask for.
 */
export function AuthAsideStory({
  eyebrow,
  title,
  description,
  points = [],
}: {
  eyebrow: string;
  title: React.ReactNode;
  description: string;
  points?: AuthAsidePoint[];
}) {
  return (
    <>
      <span className="inline-flex items-center gap-2 border border-slate-200 bg-white px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.13em] text-slate-600">
        <span className="size-1.5 rounded-full bg-brand-500" />
        {eyebrow}
      </span>

      <h2 className="mt-6 max-w-xl text-4xl font-semibold leading-[1.08] tracking-[-0.045em] sm:text-5xl xl:text-[3.4rem]">
        {title}
      </h2>

      <p className="mt-5 max-w-md text-base leading-7 text-slate-600 sm:text-[17px]">
        {description}
      </p>

      {points.length > 0 && (
        <div className="mt-10 space-y-7">
          {points.map(({ icon: Icon, title: pointTitle, description: pointDescription }) => (
            <div key={pointTitle} className="flex items-start gap-4">
              <div className="flex size-11 shrink-0 items-center justify-center border border-slate-200 bg-white text-slate-800">
                <Icon className="size-5" strokeWidth={1.8} />
              </div>
              <div className="pt-0.5">
                <h3 className="text-sm font-semibold tracking-tight text-slate-900">
                  {pointTitle}
                </h3>
                <p className="mt-1 max-w-sm text-[13px] leading-5 text-slate-500">
                  {pointDescription}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}

      <p className="mt-10 inline-flex items-center gap-2 text-[12px] text-slate-500">
        <ShieldCheck className="size-4 text-emerald-600" />
        Credentials stay encrypted, in the vault.
      </p>
    </>
  );
}

const AUTH_ICON_TONES = {
  brand: "border-brand-100 bg-brand-50 text-brand-500",
  success: "border-emerald-200 bg-emerald-50 text-emerald-600",
  warning: "border-amber-200 bg-amber-50 text-amber-600",
} as const;

/** The rounded icon that opens a status-led auth screen. */
export function AuthStatusIcon({
  icon: Icon,
  tone = "brand",
}: {
  icon: LucideIcon;
  tone?: keyof typeof AUTH_ICON_TONES;
}) {
  return (
    <div
      className={cn(
        "flex size-12 items-center justify-center rounded-2xl border",
        AUTH_ICON_TONES[tone],
      )}
    >
      <Icon className="size-6" />
    </div>
  );
}

/** The page's own title block, so every screen states its heading the same way. */
export function AuthHeading({
  eyebrow,
  title,
  description,
  align = "left",
}: {
  eyebrow?: string;
  title: string;
  description?: React.ReactNode;
  align?: "left" | "center";
}) {
  return (
    <div className={cn("mb-8", align === "center" && "text-center")}>
      {eyebrow && (
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-brand-700">
          {eyebrow}
        </p>
      )}
      <h1
        className={cn(
          "text-3xl font-semibold tracking-[-0.04em] text-slate-950",
          eyebrow ? "mt-3" : "",
        )}
      >
        {title}
      </h1>
      {description && (
        <p className="mt-3 text-sm leading-6 text-slate-500">{description}</p>
      )}
    </div>
  );
}
