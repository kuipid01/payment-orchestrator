/* shadcn-style primitives — part 2: feedback, security, display components */

"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import { Check, Copy, Eye, EyeOff, ShieldCheck, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Card, Input, Skeleton } from "./primitives";

/* ----------------------------------- modal ---------------------------------- */

export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children?: React.ReactNode;
  footer?: React.ReactNode;
}) {
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  React.useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  if (!open || !mounted) return null;

  return createPortal(
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-slate-950/40 animate-fade-in" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        // A dialog taller than the viewport must scroll inside itself: the page
        // behind it has its scrolling locked, so anything past the fold — the
        // footer buttons included — would otherwise be unreachable.
        className="relative max-h-[calc(100vh-2rem)] w-full max-w-2xl overflow-y-auto overscroll-contain rounded-2xl border border-slate-200 bg-white p-6 shadow-[var(--shadow-lift)] animate-fade-up"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">{title}</h2>
            {description ? <p className="mt-1 text-[13px] text-slate-500">{description}</p> : null}
          </div>
          <button onClick={onClose} className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600" aria-label="Close dialog">
            <X className="size-4" />
          </button>
        </div>
        {children ? <div className="mt-4">{children}</div> : null}
        {footer ? <div className="mt-6 flex justify-end gap-2">{footer}</div> : null}
      </div>
    </div>,
    document.body,
  );
}

/* ------------------------------------ tabs ---------------------------------- */

export function Tabs({
  tabs,
  value,
  onValueChange,
  className,
}: {
  tabs: { value: string; label: string; count?: number }[];
  value: string;
  onValueChange: (v: string) => void;
  className?: string;
}) {
  return (
    <div className={cn("flex gap-1.5 overflow-x-auto", className)}>
      {tabs.map((t) => (
        <button
          key={t.value}
          onClick={() => onValueChange(t.value)}
          className={cn(
            "inline-flex items-center gap-1.5 whitespace-nowrap rounded-full border px-3.5 py-1.5 text-[13px] font-medium transition-colors",
            value === t.value
              ? "border-brand-500 bg-brand-500 text-slate-950 shadow-[var(--shadow-soft)]"
              : "border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:text-slate-800"
          )}
        >
          {t.label}
          {typeof t.count === "number" ? (
            <span className={cn("rounded-full px-1.5 text-[11px]", value === t.value ? "bg-white/60 text-brand-900" : "bg-slate-100 text-slate-500")}>
              {t.count}
            </span>
          ) : null}
        </button>
      ))}
    </div>
  );
}

/* ------------------------------- masked secret ------------------------------ */

/**
 * A credential display. Only the mask is known to the list, so a recoverable
 * secret (a test key) is fetched on demand through `onReveal` — reveal and copy
 * both ask the server for the plaintext rather than copying the mask.
 *
 * `onReveal` absent means the secret cannot be read again: it is shown once, at
 * creation, and this component offers no copy button that could only ever paste
 * dots.
 */
export function MaskedSecret({
  value,
  className,
  revealable = true,
  onReveal,
}: {
  value: string;
  className?: string;
  revealable?: boolean;
  /** Fetches the plaintext for a secret that stays recoverable. */
  onReveal?: () => Promise<string>;
}) {
  const [plaintext, setPlaintext] = React.useState<string | null>(null);
  const [revealed, setRevealed] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const [failed, setFailed] = React.useState(false);

  const load = async (): Promise<string | null> => {
    if (plaintext) return plaintext;
    if (!onReveal) return null;
    try {
      const secret = await onReveal();
      setPlaintext(secret);
      setFailed(false);
      return secret;
    } catch {
      setFailed(true);
      return null;
    }
  };

  const toggle = async () => {
    if (revealed) {
      setRevealed(false);
      return;
    }
    if (await load()) setRevealed(true);
  };

  const copy = async () => {
    const secret = await load();
    if (!secret) return;
    try {
      await navigator.clipboard.writeText(secret);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      /* clipboard unavailable */
    }
  };

  const canCopy = Boolean(onReveal);

  return (
    <div className={cn("flex items-center gap-1", className)}>
      <code className="truncate rounded-md border border-slate-200 bg-slate-50 px-2.5 py-1.5 font-mono text-[12.5px] text-slate-700">
        {revealed && plaintext
          ? plaintext
          : "•".repeat(Math.min(28, Math.max(12, value.length - 4))) + value.slice(-4)}
      </code>
      {revealable ? (
        <button
          onClick={() => void toggle()}
          className="rounded-md p-1.5 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
          aria-label={revealed ? "Hide secret" : "Reveal secret"}
          title={
            failed
              ? "This key cannot be shown again"
              : revealed
                ? "Hide"
                : canCopy
                  ? "Reveal the full key"
                  : "Masked"
          }
        >
          {revealed ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
        </button>
      ) : null}
      {canCopy ? (
        <button
          onClick={() => void copy()}
          className="rounded-md p-1.5 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
          aria-label="Copy the full key to the clipboard"
          title="Copy the full key"
        >
          {copied ? <Check className="size-4 text-emerald-600" /> : <Copy className="size-4" />}
        </button>
      ) : null}
    </div>
  );
}

/* --------------------------------- code block ------------------------------- */

export function CodeBlock({
  code,
  language = "ts",
  className,
}: {
  code: string;
  language?: string;
  className?: string;
}) {
  const [copied, setCopied] = React.useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <div className={cn("group relative overflow-hidden rounded-xl border border-slate-800 bg-slate-950", className)}>
      <div className="flex items-center justify-between border-b border-slate-800/80 px-4 py-2">
        <span className="font-mono text-[11px] uppercase tracking-wider text-slate-400">{language}</span>
        <button
          onClick={copy}
          className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-[12px] text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
        >
          {copied ? <Check className="size-3.5 text-emerald-400" /> : <Copy className="size-3.5" />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="overflow-x-auto p-4 font-mono text-[13px] leading-relaxed text-slate-200">
        <code>{code}</code>
      </pre>
    </div>
  );
}

/* --------------------------------- stat card -------------------------------- */

export function StatCard({
  label,
  value,
  sub,
  icon,
  accent = false,
}: {
  label: string;
  value: string;
  sub?: string;
  icon?: React.ReactNode;
  accent?: boolean;
}) {
  return (
    <Card>
      <div className="flex items-start justify-between p-5 pb-3">
        <div>
          <p className="text-[13px] font-medium text-slate-500">{label}</p>
          <p className={cn("mt-1.5 font-semibold tracking-tight", accent ? "text-brand-600" : "text-slate-900", "text-2xl")}>
            {value}
          </p>
        </div>
        {icon ? <div className="rounded-full border border-brand-100 bg-brand-50 p-2.5 text-brand-600">{icon}</div> : null}
      </div>
      {sub ? <div className="px-5 pb-4 text-[12.5px] text-slate-400">{sub}</div> : null}
    </Card>
  );
}

/* -------------------------------- page header ------------------------------- */

export function PageHeader({
  title,
  description,
  actions,
  breadcrumb,
}: {
  title: string;
  description?: string;
  actions?: React.ReactNode;
  breadcrumb?: { label: string; href?: string }[];
}) {
  return (
    <div className="mb-6">
      {breadcrumb ? <Breadcrumb items={breadcrumb} /> : null}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="font-display text-lg text-slate-950">{title}</h1>
          {description ? <p className="mt-1 text-sm text-slate-500">{description}</p> : null}
        </div>
        {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
      </div>
    </div>
  );
}

export function Breadcrumb({ items }: { items: { label: string; href?: string }[] }) {
  return (
    <nav aria-label="Breadcrumb" className="mb-2 flex items-center gap-1.5 text-[12.5px] text-slate-400">
      {items.map((item, i) => (
        <span key={i} className="flex items-center gap-1.5">
          {i > 0 && <span className="text-slate-300">/</span>}
          {item.href ? (
            <a href={item.href} className="hover:text-slate-600">
              {item.label}
            </a>
          ) : (
            <span className="text-slate-600">{item.label}</span>
          )}
        </span>
      ))}
    </nav>
  );
}

/* ------------------------------- security notice ---------------------------- */

export function SecurityNotice({
  children,
  className,
  icon,
}: {
  children: React.ReactNode;
  className?: string;
  icon?: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-2.5 rounded-xl border border-brand-100 bg-brand-50/60 px-4 py-3 text-[13px] text-brand-900",
        className
      )}
    >
      <span className="mt-0.5 text-brand-500">{icon ?? <ShieldCheck className="size-4" />}</span>
      <div className="[&_b]:font-semibold">{children}</div>
    </div>
  );
}

/* ------------------------------ secret input -------------------------------- */

export function SecretInput(props: React.ComponentProps<typeof Input>) {
  const [visible, setVisible] = React.useState(false);
  return (
    <div className="relative">
      <Input {...props} type={visible ? "text" : "password"} className="pr-10 font-mono" />
      <button
        type="button"
        onClick={() => setVisible((v) => !v)}
        className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
        aria-label={visible ? "Hide secret" : "Show secret"}
      >
        {visible ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  );
}

/* -------------------------------- copy field -------------------------------- */

/**
 * A read-only value with a copy-to-clipboard button. Used for the inbound
 * webhook URL a merchant pastes into a provider dashboard, so the exact string
 * is one click away and the copied state is confirmed visually.
 */
export function CopyField({
  value,
  label,
  className,
}: {
  value: string;
  label?: string;
  className?: string;
}) {
  const [copied, setCopied] = React.useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <div className={className}>
      {label ? <p className="mb-1.5 text-[12px] font-medium text-slate-500">{label}</p> : null}
      <div className="flex items-stretch gap-2">
        <code className="min-w-0 flex-1 truncate rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-[12.5px] text-slate-700">
          {value}
        </code>
        <button
          type="button"
          onClick={() => void copy()}
          className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 text-[12.5px] font-medium text-slate-600 transition-colors hover:bg-slate-50"
          aria-label="Copy to clipboard"
        >
          {copied ? <Check className="size-3.5 text-emerald-600" /> : <Copy className="size-3.5" />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
    </div>
  );
}

/* ------------------------------ loading helpers ----------------------------- */

export function SkeletonRows({ rows = 5 }: { rows?: number }) {
  return (
    <div className="divide-y divide-slate-100">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 px-5 py-3.5">
          <Skeleton className="size-8 rounded-full" />
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="ml-auto h-3.5 w-20" />
          <Skeleton className="h-5 w-20 rounded-full" />
          <Skeleton className="h-3.5 w-24" />
        </div>
      ))}
    </div>
  );
}
