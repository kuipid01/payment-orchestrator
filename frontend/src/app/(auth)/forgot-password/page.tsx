"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowLeft, MailCheck } from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
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

  if (sent) {
    return (
      <div className="text-center">
        <div className="mx-auto flex size-12 items-center justify-center rounded-2xl border border-emerald-200 bg-emerald-50">
          <MailCheck className="size-6 text-emerald-600" />
        </div>
        <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">Check your inbox</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-500">
          If an account exists for that email, we've sent a link to reset your password. It expires in 30 minutes.
        </p>
        <Link href="/login" className="mt-6 inline-flex items-center gap-1.5 text-sm text-brand-500 hover:underline">
          <ArrowLeft className="size-4" /> Back to sign in
        </Link>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-xl font-semibold tracking-tight text-slate-900">Forgot your password?</h1>
      <p className="mt-1.5 text-sm text-slate-500">
        Enter your work email and we'll send you a reset link.
      </p>
      <form onSubmit={onSubmit} className="mt-8 space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="email">Email</Label>
          <Input id="email" name="email" type="email" placeholder="you@company.com" autoComplete="email" required />
        </div>
        <Button type="submit" className="w-full" size="lg" disabled={loading}>
          {loading ? "Sending…" : "Send reset link"}
        </Button>
        {error && <p role="alert" className="text-center text-sm text-red-600">{error}</p>}
      </form>
      <p className="mt-6 text-center text-sm text-slate-500">
        <Link href="/login" className="inline-flex items-center gap-1.5 font-medium text-brand-500 hover:underline">
          <ArrowLeft className="size-3.5" /> Back to sign in
        </Link>
      </p>
    </div>
  );
}
