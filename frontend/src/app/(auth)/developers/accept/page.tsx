import * as React from "react";
import { AcceptDeveloperInvitationContent } from "./page-content";

export default function AcceptDeveloperInvitationPage() {
  return (
    <React.Suspense
      fallback={
        <div className="text-center text-sm text-slate-500">Checking your invitation…</div>
      }
    >
      <AcceptDeveloperInvitationContent />
    </React.Suspense>
  );
}
