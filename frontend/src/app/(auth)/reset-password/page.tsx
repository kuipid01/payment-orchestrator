"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Lock, ShieldCheck } from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
import { toast } from "@/components/ui/toaster";
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
    <div>
      <div className="flex size-12 items-center justify-center rounded-2xl border border-brand-100 bg-brand-50">
        <Lock className="size-5 text-brand-500" />
      </div>
      <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">Set a new password</h1>
      <p className="mt-1.5 text-sm text-slate-500">
        Choose something strong — at least 12 characters.
      </p>
      <form onSubmit={onSubmit} className="mt-8 space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="password">New password</Label>
          <Input id="password" name="password" type="password" placeholder="••••••••••••" autoComplete="new-password" minLength={12} required />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="confirm">Confirm password</Label>
          <Input id="confirm" name="confirm" type="password" placeholder="••••••••••••" autoComplete="new-password" minLength={12} required />
        </div>
        <Button type="submit" className="w-full" size="lg" disabled={loading}>
          {loading ? "Updating…" : "Update password"}
        </Button>
        {error && <p role="alert" className="text-center text-sm text-red-600">{error}</p>}
      </form>
      <p className="mt-6 flex items-center justify-center gap-1.5 text-[12.5px] text-slate-400">
        <ShieldCheck className="size-3.5 text-emerald-500" />
        All active sessions will be signed out after reset.
      </p>
      <p className="mt-4 text-center text-sm text-slate-500">
        <Link href="/login" className="font-medium text-brand-500 hover:underline">
          Back to sign in
        </Link>
      </p>
    </div>
  );
}
