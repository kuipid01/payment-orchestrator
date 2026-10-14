"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { CheckCircle2, LoaderCircle, LogIn, ShieldCheck, TriangleAlert, UserPlus } from "lucide-react";
import { Button, Input, Label } from "@/components/ui";
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

  if (phase === "loading") {
    return (
      <div className="text-center">
        <LoaderCircle className="mx-auto size-9 animate-spin text-brand-500" />
        <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">Checking your invitation…</h1>
        <p className="mt-2 text-sm text-slate-500">One moment while we look up the invitation.</p>
      </div>
    );
  }

  if (phase === "unavailable") {
    return (
      <div className="text-center">
        <div className="mx-auto flex size-12 items-center justify-center rounded-2xl border border-amber-200 bg-amber-50">
          <TriangleAlert className="size-6 text-amber-600" />
        </div>
        <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">Invitation unavailable</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-500">
          {message || "This link may be invalid, expired, or already used."}
        </p>
        <Link href="/login" className="mt-8 inline-block">
          <Button variant="secondary" size="lg" className="w-full">
            Return to sign in
          </Button>
        </Link>
      </div>
    );
  }

  if (phase === "accepted") {
    return (
      <div className="text-center">
        <CheckCircle2 className="mx-auto size-11 text-emerald-500" />
        <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">You&apos;re in</h1>
        <p className="mt-2 text-sm text-slate-500">
          Your developer access to {preview?.merchant_name || "the organization"} is active. Taking you to your
          workspace…
        </p>
        <Link href="/dashboard" className="mt-8 inline-block">
          <Button size="lg" className="w-full">
            <ShieldCheck className="size-4" /> Continue to dashboard
          </Button>
        </Link>
      </div>
    );
  }

  const isReturningDeveloper = preview?.is_developer ?? false;

  return (
    <div>
      <div className="mx-auto flex size-12 items-center justify-center rounded-2xl border border-brand-100 bg-brand-50">
        {preview?.has_identity ? (
          <LogIn className="size-6 text-brand-500" />
        ) : (
          <UserPlus className="size-6 text-brand-500" />
        )}
      </div>
      <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">
        {isReturningDeveloper
          ? "Join another organization"
          : preview?.has_identity
            ? "You already have an account"
            : "Create your developer account"}
      </h1>
      <p className="mt-2 text-sm leading-relaxed text-slate-500">
        <span className="font-medium text-slate-700">{preview?.merchant_name || "An organization"}</span> invited{" "}
        <span className="font-medium text-slate-700">{preview?.email}</span> to build against their payment
        infrastructure.
        {preview?.has_identity
          ? " Enter your password to continue."
          : " Choose a password to get started."}
      </p>

      {signInAsSomeoneElse ? (
        <div className="mt-6 rounded-xl border border-amber-200 bg-amber-50 p-3.5 text-left text-[12.5px] leading-relaxed text-amber-800">
          You are signed in as {user?.email}, but this invitation is for {preview?.email}. Sign out to accept it as
          the invited address.
        </div>
      ) : null}

      {preview && preview.permissions.length > 0 ? (
        <ul className="mt-6 space-y-1.5 text-left">
          {preview.permissions.map((permission) => (
            <li key={permission} className="flex items-center gap-2 text-[13px] text-slate-600">
              <ShieldCheck className="size-4 shrink-0 text-emerald-500" />
              {permissionLabel(permission)}
            </li>
          ))}
        </ul>
      ) : null}

      <form onSubmit={onSubmit} className="mt-7 space-y-4 text-left">
        {preview?.has_identity ? (
          <div className="space-y-1.5">
            <Label htmlFor="invite-email">Email address</Label>
            <Input id="invite-email" type="email" value={preview.email} readOnly disabled />
          </div>
        ) : (
          <div className="space-y-1.5">
            <Label htmlFor="invite-name">Your name</Label>
            <Input
              id="invite-name"
              autoComplete="name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Ada Lovelace"
              required
            />
          </div>
        )}

        <div className="space-y-1.5">
          <Label htmlFor="invite-password">Password</Label>
          <Input
            id="invite-password"
            type="password"
            autoComplete={preview?.has_identity ? "current-password" : "new-password"}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </div>

        {preview?.has_identity ? null : (
          <div className="space-y-1.5">
            <Label htmlFor="invite-confirm">Confirm password</Label>
            <Input
              id="invite-confirm"
              type="password"
              autoComplete="new-password"
              value={confirm}
              onChange={(event) => setConfirm(event.target.value)}
              required
            />
            <p className="text-[12px] text-slate-400">At least {MINIMUM_PASSWORD_LENGTH} characters.</p>
          </div>
        )}

        {message ? (
          <p role="alert" className="text-[13px] text-red-600">
            {message}
          </p>
        ) : null}

        <Button type="submit" size="lg" className="w-full" disabled={phase === "submitting"}>
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
          <Link href={`/forgot-password?email=${encodeURIComponent(preview.email)}`} className="text-brand-500 hover:underline">
            Reset it
          </Link>{" "}
          — then reopen this link.
        </p>
      ) : null}
    </div>
  );
}
