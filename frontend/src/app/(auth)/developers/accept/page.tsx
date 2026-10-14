import * as React from "react";
import { LoaderCircle } from "lucide-react";

import { AuthShell } from "@/components/auth/auth-shell";
import { AcceptDeveloperInvitationContent } from "./page-content";

export default function AcceptDeveloperInvitationPage() {
  return (
    <React.Suspense
      fallback={
        <AuthShell>
          <div className="py-4 text-center">
            <LoaderCircle className="mx-auto size-9 animate-spin text-brand-500" />
            <h1 className="mt-5 text-2xl font-semibold tracking-[-0.035em] text-slate-950">
              Checking your invitation…
            </h1>
            <p className="mt-2 text-sm text-slate-500">
              One moment while we look up the invitation.
            </p>
          </div>
        </AuthShell>
      }
    >
      <AcceptDeveloperInvitationContent />
    </React.Suspense>
  );
}
