import Link from "next/link";
import { WalletCards } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * The mark every authentication screen opens with. It lives here rather than in
 * each page because the sign-in and sign-up screens each grew their own copy,
 * and a product name that renders differently on two halves of one flow is a
 * bug nobody files.
 */
export function AuthBrand({ className }: { className?: string }) {
  return (
    <Link
      href="/"
      aria-label="PayConnect home"
      className={cn("inline-flex w-fit items-center gap-3", className)}
    >
      <span className="flex size-10 items-center justify-center rounded-xl bg-brand-500 text-slate-950 shadow-sm shadow-brand-600/20">
        <WalletCards className="size-5" strokeWidth={2.3} />
      </span>
      <span className="leading-tight">
        <span className="block text-xl font-bold tracking-tight text-slate-950">
          PayConnect
        </span>
        <span className="mt-0.5 block text-[11px] font-medium text-slate-500">
          by Nile
        </span>
      </span>
    </Link>
  );
}
