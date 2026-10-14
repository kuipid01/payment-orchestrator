"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowRight, BadgeCheck, MailCheck, MailWarning, RefreshCw } from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
import {
  AuthAsideStory,
  AuthHeading,
  AuthShell,
  AuthStatusIcon,
} from "@/components/auth/auth-shell";
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
    <AuthShell
      aside={
        <AuthAsideStory
          eyebrow="Account verification"
          title={
            <>
              One click.
              <br />
              Then your
              <br />
              <span className="text-slate-400">workspace opens.</span>
            </>
          }
          description="We ask you to confirm your address before taking live payments, so nobody can register a domain or an inbox they do not control."
          points={[
            {
              icon: MailCheck,
              title: "One link, 1 hour",
              description: "Verification links are single-use and expire an hour after they are sent.",
            },
            {
              icon: BadgeCheck,
              title: "Then you are set",
              description: "Verified addresses can take live payments and invite developers.",
            },
          ]}
        />
      }
      footnote={
        <p className="mt-6 text-center text-[11px] text-slate-400">
          Need to use another address?{" "}
          <Link href="/register" className="text-brand-700 hover:underline">
            Start over
          </Link>
        </p>
      }
    >
      <div className="text-center">
        <div className="flex justify-center">
          <AuthStatusIcon
            icon={phase === "invalid" ? MailWarning : phase === "verified" ? BadgeCheck : MailCheck}
            tone={phase === "invalid" ? "warning" : phase === "verified" ? "success" : "brand"}
          />
        </div>

        <div className="mt-5">
          <AuthHeading
            align="center"
            title={
              phase === "checking"
                ? "Verifying your email…"
                : phase === "verified"
                  ? "Email verified"
                  : "Verify your email"
            }
            description={
              phase === "verified" ? (
                "Your account is verified. You can now sign in."
              ) : phase === "checking" ? (
                "Please wait while we check your verification link."
              ) : phase === "invalid" ? (
                message || "This verification link is invalid or expired. Request a new one below."
              ) : (
                <>
                  We sent a verification link to{" "}
                  <span className="font-medium text-slate-700">
                    {email || "your email address"}
                  </span>
                  . The link expires in 1 hour.
                </>
              )
            }
          />
        </div>

        {phase === "verified" ? (
          <Button
            size="lg"
            className="h-12 w-full gap-2 rounded-xl text-sm font-semibold"
            onClick={() => router.push("/login")}
          >
            Continue to sign in
            <ArrowRight className="size-4" />
          </Button>
        ) : (
          <form onSubmit={onResend} className="space-y-5 text-left">
            <div className="space-y-2">
              <Label htmlFor="verification-email" className="text-[13px] font-semibold text-slate-800">
                Email address
              </Label>
              <Input
                id="verification-email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                disabled={resending}
                required
                className="h-12 rounded-xl border-slate-200 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
              />
            </div>

            <Button
              type="submit"
              size="lg"
              disabled={resending || phase === "checking"}
              className="h-12 w-full gap-2 rounded-xl text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
            >
              <RefreshCw className="size-4" />
              {resending ? "Sending…" : "Resend verification email"}
            </Button>
          </form>
        )}

        {message && phase !== "invalid" && (
          <p role="status" className="mt-4 text-sm leading-5 text-slate-600">
            {message}
          </p>
        )}
      </div>
    </AuthShell>
  );
}
