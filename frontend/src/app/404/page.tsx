import { Compass } from "lucide-react";
import { SystemPage } from "@/components/system/system-page";

export const metadata = { title: "Page not found" };

export default function NotFoundPage() {
  return (
    <SystemPage
      code="404"
      title="Page not found"
      description="The page you're looking for doesn't exist or may have been moved. Check the URL or head back to your dashboard."
      primaryHref="/dashboard"
      primaryLabel="Back to dashboard"
      icon={<Compass className="size-6" />}
    />
  );
}
