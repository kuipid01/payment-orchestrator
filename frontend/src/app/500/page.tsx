import { TriangleAlert } from "lucide-react";
import { SystemPage } from "@/components/system/system-page";

export const metadata = { title: "Something went wrong" };

// Dynamic rendering avoids clashing with Next's internally generated static 500.html.
export const dynamic = "force-dynamic";

export default function ServerErrorPage() {
  return (
    <SystemPage
      code="500"
      title="Something went wrong"
      description="An unexpected error occurred on our side — no action from you is needed. Safe to retry: all payment operations are idempotent."
      primaryHref="/dashboard"
      primaryLabel="Back to dashboard"
      icon={<TriangleAlert className="size-6" />}
    />
  );
}
