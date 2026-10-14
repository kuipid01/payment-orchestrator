"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { SessionLoading } from "@/components/app/session-loading";
import { useAuth } from "@/components/providers/auth-provider";

/**
 * Client gate for the authenticated app shell. It waits for session hydration,
 * then sends unauthenticated visitors to the sign-in page. The API client also
 * redirects on a hard 401, so this mainly prevents a flash of dashboard content.
 */
export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const router = useRouter();

  React.useEffect(() => {
    if (status === "unauthenticated") router.replace("/login");
  }, [status, router]);

  if (status !== "authenticated") {
    return <SessionLoading />;
  }

  return <>{children}</>;
}
