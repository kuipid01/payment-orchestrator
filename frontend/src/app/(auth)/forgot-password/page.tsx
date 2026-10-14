"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowLeft, Clock, MailCheck, ShieldCheck } from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
import {
  AuthAsideStory,
  AuthHeading,
  AuthShell,
  AuthStatusIcon,
} from "@/components/auth/auth-shell";
import { forgotPassword } from "@/lib/api";

export default function ForgotPasswordPage() {
  const [sent, setSent] = React.useState(false);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError("");
    const form = new FormData(e.currentTarget);
    try {
      await forgotPassword(String(form.get("email") ?? ""));
      setSent(true);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Unable to send reset email.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      aside={
        <AuthAsideStory
          eyebrow="Password recovery"
          title={
            <>
              Locked out?
              <br />
              Back in
              <br />
              <span className="text-slate-400">two minutes.</span>
            </>
          }
          description="Tell us your work email and we'll send a single-use link to set a new password. Nobody — including us — can read the old one."
          points={[
            {
              icon: MailCheck,
              title: "One link, one use",
              description: "A reset link stops working the moment your password changes.",
            },
            {
              icon: Clock,
              title: "Expires in 30 minutes",
              description: "Stale links are refused, so an old inbox is not a permanent way in.",
            },
            {
              icon: ShieldCheck,
              title: "Old sessions end",
              description: "Resetting signs out every other device at the same time.",
            },
          ]}
        />
      }
      footnote={
        <p className="mt-6 text-center text-[11px] text-slate-400">
          PayConnect will never email you a password — only a link.
        </p>
      }
    >
      {sent ? (
        <div className="text-center">
          <div className="flex justify-center">
            <AuthStatusIcon icon={MailCheck} tone="success" />
          </div>
          <div className="mt-5">
            <AuthHeading
              align="center"
              title="Check your inbox"
              description="If an account exists for that email, we've sent a link to reset your password. It expires in 30 minutes."
            />
          </div>
          <Link href="/login" className="block">
            <Button variant="secondary" size="lg" className="h-12 w-full gap-2 rounded-xl">
              <ArrowLeft className="size-4" />
              Back to sign in
            </Button>
          </Link>
        </div>
      ) : (
        <>
          <AuthHeading
            eyebrow="Step 1 of 2"
            title="Forgot your password?"
            description="Enter your work email and we'll send you a reset link."
          />

          <form onSubmit={onSubmit} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="email" className="text-[13px] font-semibold text-slate-800">
                Email address
              </Label>
              <Input
                id="email"
                name="email"
                type="email"
                placeholder="you@company.com"
                autoComplete="email"
                required
                disabled={loading}
                className="h-12 rounded-xl border-slate-200 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
              />
            </div>

            {error && (
              <div
                role="alert"
                aria-live="polite"
                className="border border-red-200 bg-red-50 px-4 py-3 text-sm leading-5 text-red-700"
              >
                {error}
              </div>
            )}

            <Button
              type="submit"
              size="lg"
              disabled={loading}
              className="h-12 w-full gap-2 rounded-xl text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
            >
              {loading ? "Sending…" : "Send reset link"}
            </Button>
          </form>

          <div className="mt-7 border-t border-slate-100 pt-6 text-center">
            <p className="text-sm text-slate-500">
              Remembered it?{" "}
              <Link
                href="/login"
                className="inline-flex items-center gap-1 font-semibold text-brand-700 transition-colors hover:text-brand-800"
              >
                Sign in
                <ArrowLeft className="size-3.5 rotate-180" />
              </Link>
            </p>
          </div>
        </>
      )}
    </AuthShell>
  );
}
