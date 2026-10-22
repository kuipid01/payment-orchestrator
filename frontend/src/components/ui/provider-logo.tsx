"use client";

import * as React from "react";
import Image from "next/image";
import { cn } from "@/lib/utils";
import { PROVIDER_META, type ProviderKind } from "@/lib/types";

/**
 * Every provider's brand mark, served from our own origin.
 *
 * The map is a `Record<ProviderKind, …>` on purpose: adding a provider to
 * PROVIDER_META fails to compile until its logo is added here, so a new provider
 * can never silently ship as a pair of initials. (Monnify used to load its mark
 * from the vendor's site, which answers 403 to anything that is not a browser —
 * the dashboard showed a broken tile.)
 */
export const PROVIDER_LOGOS: Record<ProviderKind, string> = {
  paystack: "/providers/paystack.png",
  flutterwave: "/providers/flutterwave.png",
  monnify: "/providers/monnify.png",
  moniepoint: "/providers/moniepoint.png",
  stripe: "/providers/stripe.png",
  paypal: "/providers/paypal.png",
};

const SIZES = {
  xs: { box: "size-5 rounded", pad: "p-0.5", text: "text-[8px]" },
  sm: { box: "size-6 rounded-md", pad: "p-0.5", text: "text-[9px]" },
  md: { box: "size-8 rounded-lg", pad: "p-1", text: "text-[10px]" },
  lg: { box: "size-12 rounded-xl", pad: "p-1.5", text: "text-xs" },
} as const;

/**
 * The mark on its own, sized for the slot it sits in. Always adjacent to the
 * provider's name, so it is decorative to a screen reader.
 */
export function ProviderLogo({
  provider,
  size = "md",
  className,
}: {
  provider: ProviderKind;
  size?: keyof typeof SIZES;
  className?: string;
}) {
  const meta = PROVIDER_META[provider];
  const src = PROVIDER_LOGOS[provider];
  // A missing or unreadable asset falls back to the initials tile rather than
  // leaving an empty box where the brand should be.
  const [failed, setFailed] = React.useState(false);

  React.useEffect(() => {
    setFailed(false);
  }, [provider]);

  const style = SIZES[size];

  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-flex shrink-0 items-center justify-center overflow-hidden border border-slate-100 bg-white",
        style.box,
        src && !failed ? style.pad : null,
        className
      )}
    >
      {src && !failed ? (
        <Image
          src={src}
          alt=""
          width={128}
          height={128}
          unoptimized
          onError={() => setFailed(true)}
          className="size-full object-contain"
        />
      ) : (
        <span className={cn("flex size-full items-center justify-center font-bold", meta.accent, style.text)}>
          {meta.initials}
        </span>
      )}
    </span>
  );
}
