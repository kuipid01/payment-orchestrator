import { Compass } from "lucide-react";
import { SystemPage } from "@/components/system/system-page";

export default function GlobalNotFound() {
  return (
    <SystemPage
      code="404"
      title="Page not found"
      description="The page you're looking for doesn't exist or may have been moved."
      primaryHref="/dashboard"
      primaryLabel="Back to dashboard"
      icon={<Compass className="size-6" />}
    />
  );
}
