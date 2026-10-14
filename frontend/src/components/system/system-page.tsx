import Link from "next/link";
import { Logo } from "@/components/ui/logo";
import { Button } from "@/components/ui";

export function SystemPage({
  code,
  title,
  description,
  primaryHref = "/",
  primaryLabel = "Back home",
  secondaryHref = "/docs",
  secondaryLabel = "Read the docs",
  icon,
}: {
  code: string;
  title: string;
  description: string;
  primaryHref?: string;
  primaryLabel?: string;
  secondaryHref?: string | null;
  secondaryLabel?: string;
  icon: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-white px-5">
      <Link href="/" aria-label="PayConnect home">
        <Logo />
      </Link>

      <div className="mt-12 flex flex-col items-center text-center animate-fade-up">
        <div className="flex size-14 items-center justify-center rounded-2xl border border-slate-200 bg-slate-50 text-slate-400">
          {icon}
        </div>
        <p className="mt-6 font-mono text-[13px] font-medium tracking-widest text-brand-500">{code}</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight text-slate-900">{title}</h1>
        <p className="mt-2 max-w-md text-[14px] leading-relaxed text-slate-500">{description}</p>
        <div className="mt-8 flex gap-2.5">
          <Link href={primaryHref}>
            <Button>{primaryLabel}</Button>
          </Link>
          {secondaryHref ? (
            <Link href={secondaryHref}>
              <Button variant="secondary">{secondaryLabel}</Button>
            </Link>
          ) : null}
        </div>
      </div>

      <p className="mt-16 text-xs text-slate-400">PayConnect · Secure payment infrastructure</p>
    </div>
  );
}
