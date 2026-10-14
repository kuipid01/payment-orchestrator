
"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowRight,
  Check,
  LockKeyhole,
  Zap,
  Layers3,
  Globe2,
  Building2,
  Mail,
  UserRound,
  Eye,
  EyeOff,
  LoaderCircle,
} from "lucide-react";

import { Button, Input, Label } from "@/components/ui";
import { toast } from "@/components/ui/toaster";
import { AuthBrand } from "@/components/auth/auth-brand";
import {
  GoogleAuthButton,
  GoogleDivider,
} from "@/components/auth/google-auth-button";
import { useAuth } from "@/components/providers/auth-provider";
import { loginWithGoogle, registerAccount } from "@/lib/api";

const PERKS = [
  {
    title: "Encrypted credential vault",
    description: "Keep your payment credentials secure and organized.",
    icon: LockKeyhole,
  },
  {
    title: "Test environment free forever",
    description: "Build and test integrations without upfront costs.",
    icon: Zap,
  },
  {
    title: "Unified developer API",
    description: "One integration for multiple payment providers.",
    icon: Layers3,
  },
];

export default function RegisterPage() {
  const router = useRouter();
  const { refresh } = useAuth();

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");
  const [googleError, setGoogleError] = React.useState("");
  const [showPassword, setShowPassword] = React.useState(false);

  /**
   * Registration through Google is the same call as sign-in: the API creates
   * the merchant and its owner when the Google identity is new, and signs the
   * person in when it is not. No email confirmation is needed, because Google
   * has already verified the address it hands us.
   */
  async function signUpWithGoogle(credential: string) {
    if (loading) return;

    setLoading(true);
    setError("");
    setGoogleError("");

    try {
      await loginWithGoogle(credential);
      await refresh();
      toast("success", "Workspace created", "Your organization is ready.");
      router.push("/dashboard");
    } catch (cause) {
      setGoogleError(
        cause instanceof Error
          ? cause.message
          : "Google sign-up failed. Please try again.",
      );
    } finally {
      setLoading(false);
    }
  }

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();

    if (loading) return;

    setLoading(true);
    setError("");

    const form = new FormData(e.currentTarget);

    const name = String(form.get("name") ?? "").trim();
    const email = String(form.get("email") ?? "").trim();
    const password = String(form.get("password") ?? "");
    const organizationName = String(
      form.get("organizationName") ?? ""
    ).trim();

    try {
      await registerAccount({
        name,
        email,
        password,
        organizationName,
      });

      router.push(
        `/verify-email?email=${encodeURIComponent(email)}`
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Account registration failed. Please try again."
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen bg-white text-slate-950">
      <div className="grid min-h-screen lg:grid-cols-[minmax(0,1.02fr)_minmax(480px,0.98fr)]">
        {/* Left: Brand and product introduction */}
        <aside className="relative hidden min-h-screen flex-col overflow-hidden border-r border-brand-100/80 bg-linear-to-br from-brand-50 via-white to-brand-100/70 px-10 py-9 lg:flex xl:px-16">
          {/* Brand */}
          <AuthBrand />

          {/* Intro */}
          <div className="relative z-10 mx-auto mt-12 w-full max-w-lg lg:my-auto lg:py-16">
            <span className="inline-flex items-center gap-2 border border-slate-200 bg-white px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.13em] text-slate-600">
              <span className="size-1.5 rounded-full bg-brand-500" />
              Developer payment infrastructure
            </span>

            <h1 className="mt-6 max-w-xl text-4xl font-semibold leading-[1.08] tracking-[-0.045em] sm:text-5xl xl:text-[3.7rem]">
              Your payments.
              <br />
              Your keys.
              <br />
              <span className="text-slate-400">
                Under control.
              </span>
            </h1>

            <p className="mt-5 max-w-md text-base leading-7 text-slate-600 sm:text-[17px]">
              Create your organization, secure your payment
              credentials, and give your developers the tools
              they need to build with confidence.
            </p>

            {/* Product benefits */}
            <div className="mt-10 space-y-7">
              {PERKS.map((perk) => {
                const Icon = perk.icon;

                return (
                  <div
                    key={perk.title}
                    className="flex items-start gap-4"
                  >
                    <div
                      className={[
                        "flex size-11 shrink-0 items-center justify-center",
                        "border border-slate-200 bg-white text-slate-800",
                      ].join(" ")}
                    >
                      <Icon
                        className="size-5"
                        strokeWidth={1.8}
                      />
                    </div>

                    <div className="pt-0.5">
                      <h2 className="text-sm font-semibold tracking-tight text-slate-900">
                        {perk.title}
                      </h2>

                      <p className="mt-1 max-w-sm text-[13px] leading-5 text-slate-500">
                        {perk.description}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>

            {/* Compact infrastructure visual */}
            <div className="mt-12 hidden border border-slate-200 bg-white p-5 sm:block">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-[10px] font-semibold uppercase tracking-[0.15em] text-slate-400">
                    Your payment infrastructure
                  </p>
                  <p className="mt-1 text-sm font-semibold text-slate-900">
                    One API. Multiple providers.
                  </p>
                </div>

                <div className="flex -space-x-2">
                  <span className="flex size-9 items-center justify-center rounded-full border-2 border-white bg-sky-100 text-[10px] font-bold text-sky-800">
                    PS
                  </span>
                  <span className="flex size-9 items-center justify-center rounded-full border-2 border-white bg-amber-100 text-[10px] font-bold text-amber-900">
                    FW
                  </span>
                  <span className="flex size-9 items-center justify-center rounded-full border-2 border-white bg-brand-50 text-[10px] font-bold text-brand-800">
                    ST
                  </span>
                </div>
              </div>

              <div className="mt-5 flex items-center gap-3">
                <span className="flex size-9 items-center justify-center bg-slate-950 text-brand-400">
                  <LockKeyhole className="size-4" />
                </span>

                <div className="min-w-0 flex-1">
                  <p className="text-xs font-semibold text-slate-800">
                    Credential vault
                  </p>
                  <p className="mt-0.5 text-[11px] text-slate-500">
                    Protected access for your team
                  </p>
                </div>

                <span className="inline-flex items-center gap-1.5 text-[11px] font-medium text-emerald-700">
                  <Check className="size-3.5" />
                  Secure
                </span>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="relative z-10 mt-10 flex items-center justify-between border-t border-slate-200 pt-5 text-[11px] text-slate-500 lg:mt-0">
            <span>Built for developers and growing businesses.</span>

            <span className="hidden items-center gap-1.5 sm:inline-flex">
              <Globe2 className="size-3.5" />
              Available worldwide
            </span>
          </div>
        </aside>

        {/* Right: Registration */}
        <section className="flex min-h-screen min-w-0 flex-col px-5 py-6 sm:px-8 lg:px-10 xl:px-16">
          {/* Top navigation */}
          <header className="flex items-center justify-between lg:hidden">
            <AuthBrand />
          </header>

          {/* Form container */}
          <div className="flex flex-1 items-center justify-center py-10 sm:py-14 lg:py-8">
            <div className="w-full max-w-110 rounded-2xl border border-slate-200/80 bg-white p-6 shadow-[0_20px_70px_-45px_rgba(15,23,42,0.22)] sm:p-9">
            <div className="hidden lg:block">
              <AuthBrand />
            </div>
            <div className="mt-2 mb-8 lg:mt-9">
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-brand-700">
                Your workspace starts here
              </p>

              <h2 className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-slate-950 sm:text-[2.25rem]">
                Create your account
              </h2>

              <p className="mt-3 text-sm leading-6 text-slate-500">
                Set up your organization and secure vault.
                You&apos;ll be ready to integrate in minutes.
              </p>
            </div>

            <form onSubmit={onSubmit} className="space-y-5">
              {/* Full name */}
              <div className="space-y-2">
                <Label
                  htmlFor="name"
                  className="text-[13px] font-semibold text-slate-800"
                >
                  Full name
                </Label>

                <div className="relative">
                  <UserRound
                    aria-hidden="true"
                    className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400"
                  />

                  <Input
                    id="name"
                    name="name"
                    placeholder="John Merchant"
                    autoComplete="name"
                    required
                    disabled={loading}
                    className="h-12 rounded-xl border-slate-200 pl-10 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                  />
                </div>
              </div>

              {/* Email */}
              <div className="space-y-2">
                <Label
                  htmlFor="email"
                  className="text-[13px] font-semibold text-slate-800"
                >
                  Work email
                </Label>

                <div className="relative">
                  <Mail
                    aria-hidden="true"
                    className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400"
                  />

                  <Input
                    id="email"
                    name="email"
                    type="email"
                    placeholder="you@company.com"
                    autoComplete="email"
                    required
                    disabled={loading}
                    className="h-12 rounded-xl border-slate-200 pl-10 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                  />
                </div>
              </div>

              {/* Password */}
              <div className="space-y-2">
                <Label
                  htmlFor="password"
                  className="text-[13px] font-semibold text-slate-800"
                >
                  Password
                </Label>

                <div className="relative">
                  <LockKeyhole
                    aria-hidden="true"
                    className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400"
                  />

                  <Input
                    id="password"
                    name="password"
                    type={showPassword ? "text" : "password"}
                    placeholder="At least 12 characters"
                    autoComplete="new-password"
                    minLength={12}
                    required
                    disabled={loading}
                    className="h-12 rounded-xl border-slate-200 pl-10 pr-11 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                  />

                  <button
                    type="button"
                    onClick={() => setShowPassword((value) => !value)}
                    aria-label={
                      showPassword ? "Hide password" : "Show password"
                    }
                    aria-pressed={showPassword}
                    disabled={loading}
                    className="absolute right-3.5 top-1/2 -translate-y-1/2 rounded-md p-1 text-slate-400 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50"
                  >
                    {showPassword ? (
                      <EyeOff className="size-4" />
                    ) : (
                      <Eye className="size-4" />
                    )}
                  </button>
                </div>

                <p className="flex items-center gap-2 text-[11px] leading-5 text-slate-500">
                  <Check className="size-3.5 shrink-0 text-emerald-600" />
                  Use at least 12 characters.
                </p>
              </div>

              {/* Organization */}
              <div className="space-y-2">
                <Label
                  htmlFor="organizationName"
                  className="text-[13px] font-semibold text-slate-800"
                >
                  Organization name
                </Label>

                <div className="relative">
                  <Building2
                    aria-hidden="true"
                    className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400"
                  />

                  <Input
                    id="organizationName"
                    name="organizationName"
                    placeholder="Acme Inc."
                    autoComplete="organization"
                    required
                    disabled={loading}
                    className="h-12 rounded-xl border-slate-200 pl-10 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                  />
                </div>
              </div>

              {/* Error */}
              {error && (
                <div
                  role="alert"
                  aria-live="polite"
                  className="border border-red-200 bg-red-50 px-4 py-3 text-sm leading-5 text-red-700"
                >
                  {error}
                </div>
              )}

              {/* Submit */}
              <Button
                type="submit"
                size="lg"
                disabled={loading}
                className="h-12 w-full gap-2 rounded-xl text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
              >
                {loading ? (
                  <>
                    <LoaderCircle className="size-4 animate-spin" />
                    Creating account…
                  </>
                ) : (
                  <>
                    Create account
                    <ArrowRight className="ml-1 size-4" />
                  </>
                )}
              </Button>

              {/* Google sign-up. The organization is named from the Google
                  profile, so there is nothing else to ask for here. */}
              <GoogleDivider label="or sign up with" />

              <GoogleAuthButton
                onCredential={signUpWithGoogle}
                onError={setGoogleError}
                text="signup_with"
                notConfiguredLabel="Google sign-up is not configured"
                pendingLabel="Creating your workspace…"
                pending={loading}
              />

              {googleError && (
                <div
                  role="alert"
                  aria-live="polite"
                  className="border border-red-200 bg-red-50 px-4 py-3 text-sm leading-5 text-red-700"
                >
                  {googleError}
                </div>
              )}

              <p className="text-center text-[11px] leading-5 text-slate-400">
                By creating an account, you agree to our{" "}
                <Link
                  href="/terms"
                  className="font-medium text-slate-600 underline underline-offset-2 hover:text-brand-700"
                >
                  Terms of Service
                </Link>{" "}
                and{" "}
                <Link
                  href="/privacy"
                  className="font-medium text-slate-600 underline underline-offset-2 hover:text-brand-700"
                >
                  Privacy Policy
                </Link>
                .
              </p>
            </form>

            <div className="mt-7 border-t border-slate-100 pt-6 text-center">
              <p className="text-sm text-slate-500">
                Already have an account?{" "}
                <Link
                  href="/login"
                  className="inline-flex items-center gap-1 font-semibold text-brand-700 transition-colors hover:text-brand-800"
                >
                  Sign in
                  <ArrowRight className="size-3.5" />
                </Link>
              </p>
            </div>

            {/* Included features */}
            <div className="mt-8 border border-slate-200 bg-[#F8FAFC] p-4 sm:p-5">
              <p className="text-[10px] font-semibold uppercase tracking-[0.13em] text-slate-400">
                Included with your account
              </p>

              <ul className="mt-4 space-y-3">
                {PERKS.map((perk) => (
                  <li
                    key={perk.title}
                    className="flex items-center gap-2.5 text-[12px] font-medium text-slate-600"
                  >
                    <span className="flex size-5 shrink-0 items-center justify-center bg-emerald-50 text-emerald-700">
                      <Check className="size-3" strokeWidth={2.5} />
                    </span>
                    {perk.title}
                  </li>
                ))}
              </ul>
            </div>

            <p className="mt-7 text-center text-[11px] text-slate-400">
              Your payment credentials stay protected by design.
            </p>
            </div>
          </div>

          <footer className="border-t border-slate-100 py-4 text-center text-[11px] text-slate-400">
            PayConnect · Secure payment infrastructure
          </footer>
        </section>
      </div>
    </main>
  );
}
