import { Logo } from "@/components/ui/logo";

export function SessionLoading() {
  return (
    <div
      role="status"
      aria-label="Loading PayConnect"
      className="flex min-h-screen items-center justify-center bg-[#F8FAFC]"
    >
      <div className="flex flex-col items-center gap-3">
        <Logo mark />
        <span
          aria-hidden="true"
          className="payconnect-loading-wordmark text-xl font-semibold tracking-tight"
        >
          PayConnect
        </span>
      </div>
    </div>
  );
}
