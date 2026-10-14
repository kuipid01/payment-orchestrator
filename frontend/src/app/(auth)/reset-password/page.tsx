"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { KeyRound, LogOut, ShieldCheck } from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
import { toast } from "@/components/ui/toaster";
import {
  AuthAsideStory,
  AuthHeading,
  AuthShell,
} from "@/components/auth/auth-shell";
import { resetPassword } from "@/lib/api";

export default function ResetPasswordPage() {
  const router = useRouter();
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError("");
    const form = new FormData(e.currentTarget);
    const password = String(form.get("password") ?? "");
    const confirm = String(form.get("confirm") ?? "");
    const token = new URLSearchParams(window.location.search).get("token") ?? "";
    if (password !== confirm) {
      setError("The passwords do not match.");
      setLoading(false);
      return;
    }
    if (!token) {
      setError("The reset link is missing or invalid.");
      setLoading(false);
      return;
    }
    try {
      await resetPassword(token, password);
      toast("success", "Password updated", "Sign in with your new password.");
      router.push("/login");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Unable to reset password.");
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
              A new password.
              <br />
              <span className="text-slate-400">A clean slate.</span>
            </>
          }
          description="Pick something you have not used here before. The moment it is saved, this link is spent."
          points={[
            {
              icon: KeyRound,
              title: "At least 12 characters",
              description: "Length beats cleverness: a passphrase of four words is stronger and easier to remember.",
            },
            {
              icon: LogOut,
              title: "Everyone is signed out",
              description: "Every active session — including anyone else's — ends when the password changes.",
            },
            {
              icon: ShieldCheck,
              title: "Nothing else changes",
              description: "Your providers, applications and keys are untouched.",
            },
          ]}
        />
      }
      footnote={
        <p className="mt-6 flex items-center justify-center gap-1.5 text-center text-[11px] text-slate-400">
          <ShieldCheck className="size-3.5 text-emerald-600" />
          All active sessions will be signed out after reset.
        </p>
      }
    >
      <AuthHeading
        eyebrow="Step 2 of 2"
        title="Set a new password"
        description="Choose something strong — at least 12 characters."
      />

      <form onSubmit={onSubmit} className="space-y-5">
        <div className="space-y-2">
          <Label htmlFor="password" className="text-[13px] font-semibold text-slate-800">
            New password
          </Label>
          <Input
            id="password"
            name="password"
            type="password"
            placeholder="At least 12 characters"
            autoComplete="new-password"
            minLength={12}
            required
            disabled={loading}
            className="h-12 rounded-xl border-slate-200 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="confirm" className="text-[13px] font-semibold text-slate-800">
            Confirm password
          </Label>
          <Input
            id="confirm"
            name="confirm"
            type="password"
            placeholder="Repeat your new password"
            autoComplete="new-password"
            minLength={12}
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
          {loading ? "Updating…" : "Update password"}
        </Button>
      </form>

      <div className="mt-7 border-t border-slate-100 pt-6 text-center">
        <p className="text-sm text-slate-500">
          Changed your mind?{" "}
          <Link
            href="/login"
            className="font-semibold text-brand-700 transition-colors hover:text-brand-800"
          >
            Back to sign in
          </Link>
        </p>
      </div>
    </AuthShell>
  );
}
