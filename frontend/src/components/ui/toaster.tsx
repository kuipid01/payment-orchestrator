"use client";

/**
 * Lightweight toast store so non-hook code can trigger toasts.
 * Rendered once from the root layout.
 */

import * as React from "react";
import { Check, Info, TriangleAlert, X } from "lucide-react";
import { cn } from "@/lib/utils";

type ToastKind = "success" | "error" | "info" | "warning";

interface ToastItem {
  id: number;
  kind: ToastKind;
  title: string;
  description?: string;
}

let listeners: ((t: ToastItem) => void)[] = [];
let counter = 1;

export function toast(kind: ToastKind, title: string, description?: string) {
  const item = { id: counter++, kind, title, description };
  listeners.forEach((l) => l(item));
}

const ICONS: Record<ToastKind, React.ReactNode> = {
  success: <Check className="size-4 text-emerald-600" />,
  error: <TriangleAlert className="size-4 text-red-600" />,
  warning: <TriangleAlert className="size-4 text-amber-600" />,
  info: <Info className="size-4 text-sky-600" />,
};

export function Toaster() {
  const [items, setItems] = React.useState<ToastItem[]>([]);

  React.useEffect(() => {
    const onToast = (t: ToastItem) => {
      setItems((prev) => [...prev, t]);
      setTimeout(() => setItems((prev) => prev.filter((i) => i.id !== t.id)), 4500);
    };
    listeners.push(onToast);
    return () => {
      listeners = listeners.filter((l) => l !== onToast);
    };
  }, []);

  return (
    <div className="pointer-events-none fixed bottom-5 right-5 z-[70] flex w-96 max-w-[calc(100vw-2rem)] flex-col gap-2">
      {items.map((t) => (
        <div
          key={t.id}
          role="status"
          className="pointer-events-auto flex items-start gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-[var(--shadow-lift)] animate-fade-in"
        >
          <div className="mt-0.5">{ICONS[t.kind]}</div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-slate-900">{t.title}</p>
            {t.description ? <p className="mt-0.5 text-[13px] text-slate-500">{t.description}</p> : null}
          </div>
          <button
            onClick={() => setItems((prev) => prev.filter((i) => i.id !== t.id))}
            className="rounded p-0.5 text-slate-400 hover:text-slate-600"
            aria-label="Dismiss notification"
          >
            <X className="size-4" />
          </button>
        </div>
      ))}
    </div>
  );
}
