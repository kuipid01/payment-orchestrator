import { Wrench } from "lucide-react";
import { SystemPage } from "@/components/system/system-page";

export const metadata = { title: "Scheduled maintenance" };

export default function MaintenancePage() {
  return (
    <SystemPage
      code="503"
      title="Scheduled maintenance"
      description="We're upgrading the vault. Payments in flight are safe and will complete — new API requests will resume shortly."
      primaryHref="/"
      primaryLabel="Back home"
      secondaryHref={null}
      icon={<Wrench className="size-6" />}
    />
  );
}
