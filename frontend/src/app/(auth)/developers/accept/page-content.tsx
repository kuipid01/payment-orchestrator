"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Building2,
  CheckCircle2,
  Eye,
  EyeOff,
  KeyRound,
  LoaderCircle,
  LogIn,
  ShieldCheck,
  Sparkles,
  TriangleAlert,
  UserPlus,
} from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
import {
  AuthAsideStory,
  AuthHeading,
  AuthShell,
  AuthStatusIcon,
} from "@/components/auth/auth-shell";
import { useAuth } from "@/components/providers/auth-provider";
import {
  acceptDeveloperInvitation,
  previewDeveloperInvitation,
  type DeveloperInvitationPreview,
} from "@/lib/api";
import { DEVELOPER_PERMISSIONS } from "@/lib/types";

type Phase = "loading" | "form" | "submitting" | "unavailable" | "accepted";

/** Matches the server's rule so the form fails fast rather than round-tripping. */
const MINIMUM_PASSWORD_LENGTH = 12;

function permissionLabel(id: string): string {
  return DEVELOPER_PERMISSIONS.find((p) => p.id === id)?.label ?? id;
}

export function AcceptDeveloperInvitationContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const { user, status: authStatus, refresh } = useAuth();

  const [phase, setPhase] = React.useState<Phase>("loading");
  const [preview, setPreview] = React.useState<DeveloperInvitationPreview | null>(null);
  const [message, setMessage] = React.useState("");
  const [name, setName] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [confirm, setConfirm] = React.useState("");
  const [showPassword, setShowPassword] = React.useState(false);
  const [showConfirm, setShowConfirm] = React.useState(false);
  const previewStarted = React.useRef(false);

  // Mount resolves the token; it never accepts. Accepting is an explicit action.
  React.useEffect(() => {
    if (!token) {
      setMessage("This invitation link is missing its token.");
      setPhase("unavailable");
      return;
    }
    if (previewStarted.current) return;
    previewStarted.current = true;
    void previewDeveloperInvitation(token)
      .then((result) => {
        setPreview(result);
        setPhase("form");
      })
      .catch((error: unknown) => {
        setMessage(error instanceof Error ? error.message : "This invitation link is invalid or expired.");
        setPhase("unavailable");
      });
  }, [token]);

  // Land the invitee in the workspace they just joined.
  React.useEffect(() => {
    if (phase !== "accepted") return;
    const timer = setTimeout(() => router.replace("/dashboard"), 1200);
    return () => clearTimeout(timer);
  }, [phase, router]);

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    if (!preview) return;

    if (!preview.has_identity) {
      if (password.length < MINIMUM_PASSWORD_LENGTH) {
        setMessage(`Choose a password of at least ${MINIMUM_PASSWORD_LENGTH} characters.`);
        return;
      }
      if (password !== confirm) {
        setMessage("Those passwords do not match.");
        return;
      }
    }
    setMessage("");
    setPhase("submitting");
    try {
      await acceptDeveloperInvitation(token, {
        password,
        ...(preview.has_identity ? {} : { name: name.trim() }),
      });
      // Hydrate the shell before navigating: the app layout is gated, and the
      // session was only just issued.
      await refresh();
      setPhase("accepted");
    } catch (error: unknown) {
      setMessage(error instanceof Error ? error.message : "This invitation could not be accepted.");
      setPhase("form");
    }
  }

  const signInAsSomeoneElse =
    authStatus === "authenticated" &&
    !!user?.email &&
    !!preview &&
    user.email.toLowerCase() !== preview.email.toLowerCase();

  const isReturningDeveloper = preview?.is_developer ?? false;

  return (
    <AuthShell
      aside={
        <AuthAsideStory
          eyebrow="Developer invitation"
          title={
            preview?.merchant_name ? (
              <>
                Build on
                <br />
                {preview.merchant_name}
                <span className="text-slate-400">.</span>
              </>
            ) : (
              <>
                You have been
                <br />
                <span className="text-slate-400">invited in.</span>
              </>
            )
          }
          description="Accepting links this address to the organization that invited you, scoped to exactly the permissions they chose."
          points={[
            {
              icon: ShieldCheck,
              title: "Scoped by default",
              description: "You reach the applications and keys your grant names — and nothing else.",
            },
            {
              icon: KeyRound,
              title: "One account, many teams",
              description: "Keep this identity to work with other organizations later; grants stack on it.",
            },
            {
              icon: Building2,
              title: "Their vault, your key",
              description: "Provider credentials stay encrypted and are never handed to you directly.",
            },
          ]}
        />
      }
      footnote={
        <p className="mt-6 flex items-center justify-center gap-2 text-center text-[11px] text-slate-400">
          <ShieldCheck className="size-3.5 text-emerald-600" />
          Invitation links are single-use and expire on their own.
        </p>
      }
    >
      {phase === "loading" ? (
        <div className="py-4 text-center">
          <LoaderCircle className="mx-auto size-9 animate-spin text-brand-500" />
          <div className="mt-5">
            <AuthHeading
              align="center"
              title="Checking your invitation…"
              description="One moment while we look up the invitation."
            />
          </div>
        </div>
      ) : phase === "unavailable" ? (
        <div className="text-center">
          <div className="flex justify-center">
            <AuthStatusIcon icon={TriangleAlert} tone="warning" />
          </div>
          <div className="mt-5">
            <AuthHeading
              align="center"
              title="Invitation unavailable"
              description={message || "This link may be invalid, expired, or already used."}
            />
          </div>
          <Link href="/login" className="block">
            <Button variant="secondary" size="lg" className="h-12 w-full rounded-xl">
              Return to sign in
            </Button>
          </Link>
        </div>
      ) : phase === "accepted" ? (
        <div className="text-center">
          <div className="flex justify-center">
            <AuthStatusIcon icon={CheckCircle2} tone="success" />
          </div>
          <div className="mt-5">
            <AuthHeading
              align="center"
              title="You&apos;re in"
              description={`Your developer access to ${preview?.merchant_name || "the organization"} is active. Taking you to your workspace…`}
            />
          </div>
          <Link href="/dashboard" className="block">
            <Button size="lg" className="h-12 w-full gap-2 rounded-xl">
              <ShieldCheck className="size-4" />
              Continue to dashboard
            </Button>
          </Link>
        </div>
      ) : (
        <>
          <div className="mb-6 flex items-start gap-4">
            <AuthStatusIcon icon={preview?.has_identity ? LogIn : UserPlus} />
            <div className="pt-0.5">
              <h1 className="text-2xl font-semibold tracking-[-0.035em] text-slate-950">
                {isReturningDeveloper
                  ? "Join another organization"
                  : preview?.has_identity
                    ? "You already have an account"
                    : "Create your developer account"}
              </h1>
              <p className="mt-2 text-sm leading-6 text-slate-500">
                <span className="font-medium text-slate-700">
                  {preview?.merchant_name || "An organization"}
                </span>{" "}
                invited{" "}
                <span className="font-medium text-slate-700">{preview?.email}</span> to build against their
                payment infrastructure.{" "}
                {preview?.has_identity
                  ? "Enter your password to continue."
                  : "Choose a password to get started."}
              </p>
            </div>
          </div>

          {signInAsSomeoneElse ? (
            <div className="mb-6 rounded-xl border border-amber-200 bg-amber-50 p-3.5 text-left text-[12.5px] leading-relaxed text-amber-800">
              You are signed in as {user?.email}, but this invitation is for {preview?.email}. Sign out to
              accept it as the invited address.
            </div>
          ) : null}

          {preview && preview.permissions.length > 0 ? (
            <div className="mb-6 border border-slate-200 bg-[#F8FAFC] p-4">
              <p className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-[0.13em] text-slate-400">
                <Sparkles className="size-3" />
                What you can do
              </p>
              <ul className="mt-3 space-y-2">
                {preview.permissions.map((permission) => (
                  <li key={permission} className="flex items-center gap-2.5 text-[12.5px] font-medium text-slate-600">
                    <span className="flex size-5 shrink-0 items-center justify-center bg-emerald-50 text-emerald-700">
                      <ShieldCheck className="size-3" strokeWidth={2.5} />
                    </span>
                    {permissionLabel(permission)}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}

          <form onSubmit={onSubmit} className="space-y-5">
            {preview?.has_identity ? (
              <div className="space-y-2">
                <Label htmlFor="invite-email" className="text-[13px] font-semibold text-slate-800">
                  Email address
                </Label>
                <Input
                  id="invite-email"
                  type="email"
                  value={preview.email}
                  readOnly
                  disabled
                  className="h-12 rounded-xl border-slate-200 text-sm"
                />
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="invite-name" className="text-[13px] font-semibold text-slate-800">
                  Your name
                </Label>
                <Input
                  id="invite-name"
                  autoComplete="name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="Ada Lovelace"
                  required
                  className="h-12 rounded-xl border-slate-200 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                />
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="invite-password" className="text-[13px] font-semibold text-slate-800">
                Password
              </Label>
              <div className="relative">
                <Input
                  id="invite-password"
                  type={showPassword ? "text" : "password"}
                  autoComplete={preview?.has_identity ? "current-password" : "new-password"}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder={preview?.has_identity ? "Your existing password" : "At least 12 characters"}
                  required
                  className="h-12 rounded-xl border-slate-200 pr-11 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((value) => !value)}
                  aria-label={showPassword ? "Hide password" : "Show password"}
                  aria-pressed={showPassword}
                  className="absolute right-3.5 top-1/2 -translate-y-1/2 rounded-md p-1 text-slate-400 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                >
                  {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                </button>
              </div>
            </div>

            {preview?.has_identity ? null : (
              <div className="space-y-2">
                <Label htmlFor="invite-confirm" className="text-[13px] font-semibold text-slate-800">
                  Confirm password
                </Label>
                <div className="relative">
                  <Input
                    id="invite-confirm"
                    type={showConfirm ? "text" : "password"}
                    autoComplete="new-password"
                    value={confirm}
                    onChange={(event) => setConfirm(event.target.value)}
                    placeholder="Repeat your password"
                    required
                    className="h-12 rounded-xl border-slate-200 pr-11 text-sm placeholder:text-slate-400 focus-visible:ring-brand-500"
                  />
                  <button
                    type="button"
                    onClick={() => setShowConfirm((value) => !value)}
                    aria-label={showConfirm ? "Hide confirmation password" : "Show confirmation password"}
                    aria-pressed={showConfirm}
                    className="absolute right-3.5 top-1/2 -translate-y-1/2 rounded-md p-1 text-slate-400 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                  >
                    {showConfirm ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
                <p className="text-[11px] leading-5 text-slate-500">
                  At least {MINIMUM_PASSWORD_LENGTH} characters.
                </p>
              </div>
            )}

            {message ? (
              <div
                role="alert"
                aria-live="polite"
                className="border border-red-200 bg-red-50 px-4 py-3 text-sm leading-5 text-red-700"
              >
                {message}
              </div>
            ) : null}

            <Button
              type="submit"
              size="lg"
              disabled={phase === "submitting"}
              className="h-12 w-full gap-2 rounded-xl text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
            >
              {phase === "submitting"
                ? "Accepting…"
                : preview?.has_identity
                  ? "Sign in and accept"
                  : "Create account and accept"}
            </Button>
          </form>

          {preview?.has_identity ? (
            <p className="mt-5 text-center text-[12.5px] text-slate-400">
              Forgotten your password?{" "}
              <Link
                href={`/forgot-password?email=${encodeURIComponent(preview.email)}`}
                className="font-medium text-brand-700 hover:underline"
              >
                Reset it
              </Link>{" "}
              — then reopen this link.
            </p>
          ) : null}

          <div className="mt-7 border-t border-slate-100 pt-6 text-center">
            <p className="text-sm text-slate-500">
              Already set up?{" "}
              <Link
                href="/login"
                className="font-semibold text-brand-700 transition-colors hover:text-brand-800"
              >
                Sign in
              </Link>
            </p>
          </div>
        </>
      )}
    </AuthShell>
  );
}
