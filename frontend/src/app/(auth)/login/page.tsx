"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import Script from "next/script";
import {
  ArrowRight,
  BarChart3,
  CheckCircle2,
  Globe2,
  LockKeyhole,
  Mail,
  ShieldCheck,
  WalletCards,
  Eye,
  EyeOff,
  LoaderCircle,
} from "lucide-react";

import { Button, Input, Label } from "@/components/ui";
import { SessionLoading } from "@/components/app/session-loading";
import { toast } from "@/components/ui/toaster";
import { loginAccount, loginWithGoogle } from "@/lib/api";
import { useAuth } from "@/components/providers/auth-provider";

interface GoogleCredentialResponse {
  credential: string;
}

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (options: {
            client_id: string;
            callback: (response: GoogleCredentialResponse) => void;
            auto_select?: boolean;
          }) => void;
          renderButton: (
            parent: HTMLElement,
            options: {
              theme: "outline";
              size: "large";
              text: "continue_with";
              shape: "rectangular";
              width: number;
            },
          ) => void;
        };
      };
    };
  }
}

const highlights = [
  {
    icon: Globe2,
    title: "Connect your payment providers",
    description: "Paystack, Flutterwave, and more in one place.",
  },
  {
    icon: BarChart3,
    title: "See the full picture",
    description: "Track payment activity across your integrations.",
  },
  {
    icon: LockKeyhole,
    title: "Security built in",
    description: "Your provider credentials stay protected in the vault.",
  },
];

function Brand() {
  return (
    <Link href="/" aria-label="PayConnect home" className="inline-flex items-center gap-3">
      <span className="flex size-10 items-center justify-center rounded-xl bg-brand-500 text-slate-950 shadow-sm shadow-brand-600/20">
        <WalletCards className="size-5" strokeWidth={2.3} />
      </span>
      <span className="leading-tight">
        <span className="block text-xl font-bold tracking-tight text-slate-950">
          PayConnect
        </span>
        <span className="mt-0.5 block text-[11px] font-medium text-slate-500">
          by Nile
        </span>
      </span>
    </Link>
  );
}

export default function LoginPage() {
  const router = useRouter();
  const { refresh, status } = useAuth();

  const [loading, setLoading] = React.useState(false);
  const [googleReady, setGoogleReady] = React.useState(false);
  const [googleError, setGoogleError] = React.useState("");
  const [showPassword, setShowPassword] = React.useState(false);

  const googleButtonRef = React.useRef<HTMLDivElement>(null);
  const googleClientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;

  React.useEffect(() => {
    if (status === "authenticated") router.replace("/dashboard");
  }, [status, router]);

  const onGoogleCredential = React.useRef<
    (credential: string) => Promise<void>
  >(async () => {});

  onGoogleCredential.current = async (credential) => {
    setLoading(true);
    setGoogleError("");

    try {
      await loginWithGoogle(credential);
      await refresh();
      toast("success", "Welcome back!", "Your secure session has been started.");
      router.push("/dashboard");
    } catch (error) {
      setGoogleError(
        error instanceof Error ? error.message : "Google sign-in failed.",
      );
    } finally {
      setLoading(false);
    }
  };

  React.useEffect(() => {
    const target = googleButtonRef.current;
    const googleID = window.google?.accounts.id;

    if (!googleReady || !googleClientId || !target || !googleID) return;

    target.replaceChildren();

    googleID.initialize({
      client_id: googleClientId,
      callback: (response) => {
        void onGoogleCredential.current(response.credential);
      },
    });

    googleID.renderButton(target, {
      theme: "outline",
      size: "large",
      text: "continue_with",
      shape: "rectangular",
      width: Math.min(target.clientWidth || 420, 420),
    });
  }, [googleReady, googleClientId]);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (loading) return;

    setLoading(true);
    const form = new FormData(event.currentTarget);

    try {
      await loginAccount({
        email: String(form.get("email") ?? "").trim(),
        password: String(form.get("password") ?? ""),
      });

      await refresh();
      toast("success", "Welcome back!", "Your secure session has been started.");
      router.push("/dashboard");
    } catch (error) {
      toast(
        "error",
        "Sign in failed",
        error instanceof Error ? error.message : "Unable to sign in.",
      );
    } finally {
      setLoading(false);
    }
  }

  if (status !== "unauthenticated") {
    return <SessionLoading />;
  }

  return (
    <main className="min-h-screen bg-white text-slate-950">
      <div className="grid min-h-screen lg:grid-cols-[minmax(0,1.02fr)_minmax(480px,0.98fr)]">
        {/* Brand / product panel */}
        <section className="relative hidden min-h-screen overflow-hidden border-r border-brand-100/80 bg-gradient-to-br from-brand-50 via-white to-brand-100/70 px-10 py-9 lg:flex lg:flex-col xl:px-16">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -left-36 top-[35%] size-[32rem] rounded-full bg-brand-200/30 blur-3xl"
          />
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -right-32 bottom-[-8rem] h-[34rem] w-[25rem] rotate-[-22deg] rounded-[48%] border-[48px] border-white/55"
          />

          <header className="relative z-10 flex items-center justify-between gap-5">
            <Brand />
            <span className="text-xs font-medium tracking-wide text-slate-500">
              Payments, connected
            </span>
          </header>

          <div className="relative z-10 my-auto max-w-2xl py-14">
            <p className="mb-5 inline-flex items-center gap-2 rounded-full border border-brand-200 bg-white/75 px-3 py-1.5 text-xs font-semibold text-brand-800 shadow-sm">
              <span className="size-1.5 rounded-full bg-brand-500" />
              Payments, connected
            </p>

            <h1 className="max-w-xl text-5xl font-semibold leading-[1.08] tracking-[-0.055em] text-slate-950 xl:text-6xl">
              One integration.
              <span className="mt-2 block bg-gradient-to-r from-brand-700 to-brand-500 bg-clip-text text-transparent">
                All your payments.
              </span>
            </h1>

            <p className="mt-6 max-w-lg text-base leading-7 text-slate-600 xl:text-lg">
              Accept payments, manage providers, and power your business with
              one secure platform built for businesses everywhere.
            </p>

            <div className="mt-9 space-y-5">
              {highlights.map(({ icon: Icon, title, description }) => (
                <div key={title} className="flex items-start gap-4">
                  <span className="flex size-11 shrink-0 items-center justify-center rounded-2xl border border-brand-100 bg-white text-brand-700 shadow-sm">
                    <Icon className="size-5" />
                  </span>
                  <div className="pt-0.5">
                    <p className="text-sm font-semibold text-slate-900">
                      {title}
                    </p>
                    <p className="mt-1 text-sm leading-5 text-slate-500">
                      {description}
                    </p>
                  </div>
                </div>
              ))}
            </div>

            {/* Lightweight product preview made from real UI elements, not a static image */}
            <div className="relative mt-10 rounded-2xl border border-white/90 bg-white/85 p-4 shadow-[0_24px_80px_-35px_rgba(194,147,0,0.22)] backdrop-blur-sm xl:p-5">
              <div className="flex items-center justify-between gap-4 border-b border-slate-100 pb-4">
                <div>
                  <p className="text-xs font-medium text-slate-500">
                    Payment overview
                  </p>
                  <p className="mt-1 text-lg font-semibold tracking-tight text-slate-950">
                    Your payments, in one view
                  </p>
                </div>
                <span className="rounded-lg bg-emerald-50 px-2.5 py-1.5 text-[11px] font-semibold text-emerald-700">
                  <CheckCircle2 className="mr-1 inline size-3.5" />
                  Secure vault
                </span>
              </div>

              <div className="mt-4 grid grid-cols-3 gap-3">
                {[
                  { label: "Providers", value: "One place", tint: "bg-brand-50 text-brand-800" },
                  { label: "Visibility", value: "Real-time", tint: "bg-brand-50 text-brand-800" },
                  { label: "Credentials", value: "Protected", tint: "bg-emerald-50 text-emerald-700" },
                ].map((item) => (
                  <div key={item.label} className="min-w-0 rounded-xl border border-slate-100 bg-white p-3">
                    <span className={`inline-flex rounded-lg px-2 py-1 text-[10px] font-semibold ${item.tint}`}>
                      {item.label}
                    </span>
                    <p className="mt-2 truncate text-sm font-semibold text-slate-800">
                      {item.value}
                    </p>
                  </div>
                ))}
              </div>

              <div className="mt-4 flex h-14 items-end gap-1.5 overflow-hidden rounded-xl bg-gradient-to-b from-brand-50/80 to-white px-4 pt-3">
                {[22, 32, 27, 42, 36, 48, 39, 56, 47, 63, 52, 72, 62, 82, 70, 90, 77, 100].map(
                  (height, index) => (
                    <span
                      key={index}
                      className="flex-1 rounded-t-sm bg-brand-400/80"
                      style={{ height: `${height}%` }}
                    />
                  ),
                )}
              </div>
            </div>
          </div>

          <footer className="relative z-10 flex flex-wrap items-center justify-between gap-4 border-t border-brand-200/60 pt-5 text-xs text-slate-500">
            <span>Built for ambitious businesses everywhere.</span>
            <span className="inline-flex items-center gap-2">
              <ShieldCheck className="size-4 text-emerald-600" />
              Secure by design
            </span>
          </footer>
        </section>

        {/* Login panel */}
        <section className="flex min-h-screen flex-col px-5 py-6 sm:px-8 lg:px-10 xl:px-16">
          <header className="flex items-center justify-between lg:hidden">
            <Brand />
          </header>

          <div className="flex flex-1 items-center justify-center py-10 sm:py-14 lg:py-8">
            <div className="w-full max-w-[440px]">
              <div className="mb-8 lg:hidden">
                <p className="text-3xl font-semibold leading-tight tracking-[-0.04em]">
                  One integration.
                  <span className="block text-brand-700">All your payments.</span>
                </p>
                <p className="mt-3 text-sm leading-6 text-slate-500">
                  Manage your payment providers from one secure workspace.
                </p>
              </div>

              <div className="rounded-2xl border border-slate-200/80 bg-white p-6 shadow-[0_20px_70px_-45px_rgba(15,23,42,0.22)] sm:p-9">
                <div className="hidden lg:block">
                  <Brand />
                </div>

                <div className="mt-2 lg:mt-9">
                  <h2 className="text-3xl font-semibold tracking-[-0.045em] text-slate-950">
                    Sign in to PayConnect
                  </h2>
                  <p className="mt-2 text-sm leading-6 text-slate-500">
                    Welcome back — your vault is waiting.
                  </p>
                </div>

                <form onSubmit={onSubmit} className="mt-8 space-y-5">
                  <div className="space-y-2">
                    <Label htmlFor="email" className="text-[13px] font-medium text-slate-800">
                      Email address
                    </Label>
                    <div className="relative">
                      <Mail className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
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

                  <div className="space-y-2">
                    <div className="flex items-center justify-between gap-3">
                      <Label htmlFor="password" className="text-[13px] font-medium text-slate-800">
                        Password
                      </Label>
                      <Link
                        href="/forgot-password"
                        className="text-xs font-medium text-brand-700 transition-colors hover:text-brand-800 hover:underline"
                      >
                        Forgot password?
                      </Link>
                    </div>
                    <div className="relative">
                      <LockKeyhole className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
                      <Input
                        id="password"
                        name="password"
                        type={showPassword ? "text" : "password"}
                        placeholder="Enter your password"
                        autoComplete="current-password"
                        required
                        disabled={loading}
                        className="h-12 rounded-xl border-slate-200 pl-10 pr-11 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword((value) => !value)}
                        aria-label={showPassword ? "Hide password" : "Show password"}
                        aria-pressed={showPassword}
                        className="absolute right-3.5 top-1/2 -translate-y-1/2 rounded-md p-1 text-slate-400 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                      >
                        {showPassword ? (
                          <EyeOff className="size-4" />
                        ) : (
                          <Eye className="size-4" />
                        )}
                      </button>
                    </div>
                  </div>

                  <Button
                    type="submit"
                    className="h-12 w-full gap-2 rounded-xl text-sm font-semibold"
                    disabled={loading}
                  >
                    {loading ? (
                      <>
                        <LoaderCircle className="size-4 animate-spin" />
                        Signing in…
                      </>
                    ) : (
                      <>
                        Sign in
                        <ArrowRight className="size-4" />
                      </>
                    )}
                  </Button>
                </form>

                <div className="my-6 flex items-center gap-3">
                  <span className="h-px flex-1 bg-slate-200" />
                  <span className="text-xs text-slate-400">or continue with</span>
                  <span className="h-px flex-1 bg-slate-200" />
                </div>

                {googleClientId ? (
                  <>
                    <Script
                      src="https://accounts.google.com/gsi/client"
                      strategy="afterInteractive"
                      onLoad={() => setGoogleReady(true)}
                      onError={() =>
                        setGoogleError("Google sign-in could not load. Please try again.")
                      }
                    />
                    <div
                      ref={googleButtonRef}
                      className="flex min-h-11 w-full justify-center overflow-hidden rounded-xl [&>div]:max-w-full"
                    />
                    {loading && (
                      <p className="mt-2 text-center text-xs text-slate-500">
                        Completing secure sign-in…
                      </p>
                    )}
                  </>
                ) : (
                  <Button
                    type="button"
                    variant="secondary"
                    size="lg"
                    className="h-12 w-full rounded-xl"
                    disabled
                  >
                    Google sign-in is not configured
                  </Button>
                )}

                {googleError && (
                  <p role="alert" className="mt-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
                    {googleError}
                  </p>
                )}

                <p className="mt-6 text-center text-[11px] leading-5 text-slate-400">
                  By continuing, you agree to our{" "}
                  <Link href="/terms" className="font-medium text-slate-600 underline underline-offset-2 hover:text-brand-700">
                    Terms of Service
                  </Link>{" "}
                  and{" "}
                  <Link href="/privacy" className="font-medium text-slate-600 underline underline-offset-2 hover:text-brand-700">
                    Privacy Policy
                  </Link>
                  .
                </p>

                <div className="mt-7 border-t border-slate-100 pt-6 text-center">
                  <p className="text-sm text-slate-500">
                    New to PayConnect?{" "}
                    <Link
                      href="/register"
                      className="inline-flex items-center gap-1 font-semibold text-brand-700 transition-colors hover:text-brand-800"
                    >
                      Create account
                      <ArrowRight className="size-3.5" />
                    </Link>
                  </p>
                </div>
              </div>

              <p className="mt-6 flex items-center justify-center gap-2 text-xs text-slate-400">
                <ShieldCheck className="size-4 text-emerald-600" />
                Your session is protected with secure authentication.
              </p>
            </div>
          </div>

          <footer className="py-2 text-center text-[11px] text-slate-400">
            PayConnect by Nile · Payment infrastructure for modern businesses
          </footer>
        </section>
      </div>
    </main>
  );
}
