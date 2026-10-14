"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { MailCheck, MailWarning } from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
import { resendVerification, verifyEmail } from "@/lib/api";

export default function VerifyEmailPage() {
  const router = useRouter();
  const [phase, setPhase] = React.useState<"checking" | "awaiting" | "verified" | "invalid">("checking");
  const [email, setEmail] = React.useState("");
  const [resending, setResending] = React.useState(false);
  const [message, setMessage] = React.useState("");
  const verificationStarted = React.useRef(false);

  React.useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token");
    setEmail(params.get("email") ?? "");
    if (!token) {
      setPhase("awaiting");
      return;
    }
    if (verificationStarted.current) return;
    verificationStarted.current = true;
    window.history.replaceState({}, "", `/verify-email${params.has("email") ? `?email=${encodeURIComponent(params.get("email") ?? "")}` : ""}`);
    void verifyEmail(token).then(() => {
      setPhase("verified");
    }).catch((error: unknown) => {
      setMessage(error instanceof Error ? error.message : "This verification link is invalid or expired.");
      setPhase("invalid");
    });
  }, []);

  async function onResend(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setResending(true);
    setMessage("");
    try {
      await resendVerification(email);
      setMessage("If this account needs verification, a new link will arrive shortly.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not send a verification email.");
    } finally {
      setResending(false);
    }
  }

  return (
    <div className="text-center">
      <div className="mx-auto flex size-12 items-center justify-center rounded-2xl border border-brand-100 bg-brand-50">
        {phase === "invalid" ? <MailWarning className="size-6 text-amber-600" /> : <MailCheck className="size-6 text-brand-500" />}
      </div>
      <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">
        {phase === "checking" ? "Verifying your email…" : phase === "verified" ? "Email verified" : "Verify your email"}
      </h1>
      <p className="mt-2 text-sm leading-relaxed text-slate-500">
        {phase === "verified"
          ? "Your account is verified. You can now sign in."
          : phase === "checking"
            ? "Please wait while we check your verification link."
            : phase === "invalid"
              ? message || "This verification link is invalid or expired. Request a new one below."
              : <>We sent a verification link to <span className="font-medium text-slate-700">{email || "your email address"}</span>. The link expires in 1 hour.</>}
      </p>

      {phase === "verified" ? (
        <Button size="lg" className="mt-8 w-full" onClick={() => router.push("/login")}>
          Continue to sign in
        </Button>
      ) : (
        <form onSubmit={onResend} className="mt-8 space-y-3 text-left">
          <div className="space-y-1.5">
            <Label htmlFor="verification-email">Email address</Label>
            <Input
              id="verification-email"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </div>
          <Button type="submit" size="lg" className="w-full" disabled={resending || phase === "checking"}>
            {resending ? "Sending…" : "Resend verification email"}
          </Button>
        </form>
      )}

      {message && phase !== "invalid" && <p role="status" className="mt-4 text-sm text-slate-600">{message}</p>}

      <p className="mt-6 text-[12.5px] text-slate-400">
        Need to use another address?{" "}
        <Link href="/register" className="text-brand-500 hover:underline">
          Re-register
        </Link>
      </p>
    </div>
  );
}
