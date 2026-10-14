import { ShieldAlert } from "lucide-react";
import { SystemPage } from "@/components/system/system-page";

export const metadata = { title: "Access denied" };

export default function ForbiddenPage() {
  return (
    <SystemPage
      code="403"
      title="Access denied"
      description="Your key or role doesn't have permission for this resource. If you think this is a mistake, ask an organization admin to check your permissions."
      primaryHref="/dashboard"
      primaryLabel="Back to dashboard"
      icon={<ShieldAlert className="size-6" />}
    />
  );
}
