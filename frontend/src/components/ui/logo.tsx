"use client"

import { cn } from "@/lib/utils";

export function Logo({
  className,
  mark = false,
}: {
  className?: string;
  /** Render only the mark, without the PayConnect wordmark. */
  mark?: boolean;
}) {
  return (
    <span className={cn("inline-flex min-w-0 items-center gap-2", className)}>
      <span className="flex size-7 shrink-0 items-center justify-center rounded-[10px] bg-slate-950 shadow-[0_1px_2px_rgba(15,23,42,0.25)]">
        <svg viewBox="0 0 24 24" fill="none" className="size-4 text-brand-500">
          <path
            d="M12 3 5 6.5v5.1c0 4.6 3 8 7 9.4 4-1.4 7-4.8 7-9.4V6.5L12 3Z"
            stroke="currentColor"
            strokeWidth="1.8"
            strokeLinejoin="round"
          />
          <path d="m9 12 2.2 2.2L15.5 10" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </span>
      {!mark && (
        <span className="truncate text-[15px] font-semibold tracking-tight text-slate-900">PayConnect</span>
      )}
    </span>
  );
}
