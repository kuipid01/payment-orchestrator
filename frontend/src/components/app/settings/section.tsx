import * as React from "react";
import type { LucideIcon } from "lucide-react";
import { Plug } from "lucide-react";

import { Card } from "@/components/ui";
import { cn } from "@/lib/utils";

/**
 * The pieces every settings page is built from: one section per concern, each
 * with an icon, a heading and — when the API behind it does not exist yet — an
 * explicit notice. The settings pages used to be loosely-styled cards that
 * succeeded at whatever they pretended to do; these primitives make the
 * structure uniform and make "not wired up" a visible state rather than a
 * success toast.
 */

export type SettingsTone = "brand" | "emerald" | "violet" | "slate";

const TONES: Record<SettingsTone, string> = {
  brand: "bg-brand-50 text-brand-600",
  emerald: "bg-emerald-50 text-emerald-600",
  violet: "bg-violet-50 text-violet-600",
  slate: "bg-slate-100 text-slate-500",
};

/** The rounded, tinted square that opens a section. */
export function SettingsIconTile({
  icon: Icon,
  tone = "brand",
}: {
  icon: LucideIcon;
  tone?: SettingsTone;
}) {
  return (
    <span
      className={cn(
        "flex size-12 shrink-0 items-center justify-center rounded-xl",
        TONES[tone],
      )}
    >
      <Icon className="size-6" strokeWidth={1.8} />
    </span>
  );
}

export function SettingsSection({
  icon,
  tone = "brand",
  title,
  description,
  aside,
  notice,
  children,
  flush = false,
}: {
  icon: LucideIcon;
  tone?: SettingsTone;
  title: string;
  description?: React.ReactNode;
  /** A control that belongs with the heading: a badge, a switch, a button. */
  aside?: React.ReactNode;
  /** Shown directly under the heading, before the body. */
  notice?: React.ReactNode;
  children?: React.ReactNode;
  /** The body brings its own padding — use for divided rows. */
  flush?: boolean;
}) {
  return (
    <Card className="overflow-hidden rounded-xl border-slate-200/90 shadow-sm shadow-slate-900/[0.02]">
      <div className="flex flex-col gap-4 p-5 sm:flex-row sm:items-start sm:justify-between sm:gap-6 sm:p-7 sm:pb-6">
        <div className="flex items-start gap-4">
          <SettingsIconTile icon={icon} tone={tone} />
          <div className="min-w-0 pt-0.5">
            <h2 className="text-[15px] font-semibold tracking-tight text-slate-950">{title}</h2>
            {description ? (
              <p className="mt-1 max-w-xl text-[13px] leading-5 text-slate-500">{description}</p>
            ) : null}
          </div>
        </div>
        {aside ? <div className="flex shrink-0 items-center gap-3 sm:pl-0 pl-16">{aside}</div> : null}
      </div>

      {notice ? <div className="px-5 pb-5 sm:px-7">{notice}</div> : null}

      {children ? <div className={flush ? "" : "px-5 pb-5 sm:px-7 sm:pb-7"}>{children}</div> : null}
    </Card>
  );
}

/**
 * States plainly that the control beside it has nothing behind it yet. A
 * disabled button with no explanation reads as a broken product; this reads as
 * work that has not shipped.
 */
export function NotConnectedNotice({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-2.5 rounded-xl border border-slate-200 bg-slate-50/80 px-4 py-3 text-[12.5px] leading-5 text-slate-600",
        className,
      )}
    >
      <Plug className="mt-0.5 size-4 shrink-0 text-slate-400" />
      <p>
        <span className="font-semibold text-slate-700">Not connected yet.</span> {children}
      </p>
    </div>
  );
}

/** A label/description on the left, a value or control on the right. */
export function SettingsRow({
  title,
  description,
  control,
  className,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  control?: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between sm:gap-6",
        className,
      )}
    >
      <div className="min-w-0">
        <p className="text-[13.5px] font-medium text-slate-800">{title}</p>
        {description ? (
          <p className="mt-0.5 max-w-xl text-[12.5px] leading-5 text-slate-500">{description}</p>
        ) : null}
      </div>
      {control ? <div className="shrink-0 sm:pl-6">{control}</div> : null}
    </div>
  );
}

/** A divided list of SettingsRow, for settings that are rows rather than forms. */
export function SettingsRows({ children }: { children: React.ReactNode }) {
  return <div className="divide-y divide-slate-100 border-t border-slate-100">{children}</div>;
}

/** A quiet empty state for a list that has nothing to show yet. */
export function SettingsEmptyState({
  icon: Icon,
  title,
  description,
}: {
  icon: LucideIcon;
  title: string;
  description: string;
}) {
  return (
    <div className="rounded-xl border border-dashed border-slate-200 px-5 py-10 text-center">
      <Icon className="mx-auto size-6 text-slate-400" />
      <p className="mt-3 text-[13.5px] font-medium text-slate-800">{title}</p>
      <p className="mx-auto mt-1 max-w-md text-[12.5px] leading-5 text-slate-500">{description}</p>
    </div>
  );
}
