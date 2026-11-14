"use client";

import * as React from "react";
import { useSearchParams } from "next/navigation";
import {
  ArrowRight,
  CheckCircle2,
  Globe2,
  KeyRound,
  LoaderCircle,
  Lock,
  Mail,
  Shield,
  ShieldCheck,
  XCircle,
} from "lucide-react";

import { Button, Input, Label } from "@/components/ui";
import { AuthShell } from "@/components/auth/auth-shell";
import {
  API_BASE_URL,
  apiClient,
  createApplication,
  createApplicationApiKey,
  createWebhook,
  listApplications,
  listWebhooks,
} from "@/lib/api";

type Phase = "sending" | "code" | "verifying" | "authorizing" | "done" | "error";

const BIZUP_ORIGIN = (() => {
  const configured = process.env.NEXT_PUBLIC_BIZUP_APP_URL || "http://localhost:3000";
  try {
    return new URL(configured).origin;
  } catch {
    return "http://localhost:3000";
  }
})();

/** Only accept a webhook URL that actually points at the configured BizUp app. */
function trustedWebhook(raw: string): string {
  if (!raw) return "";
  try {
    return new URL(raw).origin === BIZUP_ORIGIN ? raw : "";
  } catch {
    return "";
  }
}

export function OAuthAuthorizeContent() {
  const searchParams = useSearchParams();

  const [email, setEmail] = React.useState("");
  const [emailResolved, setEmailResolved] = React.useState(false);
  const [code, setCode] = React.useState("");
  const [maskedEmail, setMaskedEmail] = React.useState("");
  const [phase, setPhase] = React.useState<Phase>("sending");
  const [error, setError] = React.useState("");
  const [verified, setVerified] = React.useState(false);
  const [devCode, setDevCode] = React.useState("");
  const requestedFor = React.useRef("");
  const delivered = React.useRef(false);

  const businessId = searchParams.get("business_id") || "";
  const state = searchParams.get("state") || "";
  const webhookUrl = trustedWebhook(searchParams.get("webhook_url") || "");

  // The email travels in the URL fragment so it is not sent to the server in a
  // request line or stored in access logs. The query fallback keeps old links
  // working during rollout.
  React.useEffect(() => {
    const fromHash = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    const value = (fromHash.get("email") || searchParams.get("email") || "")
      .trim()
      .toLowerCase();
    setEmail(value);
    setEmailResolved(true);
  }, [searchParams]);

  React.useEffect(() => {
    if (!email || requestedFor.current === email) return;
    requestedFor.current = email;
    void requestCode(email);
    // requestCode is stable for this flow; the email guard prevents re-sends.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [email]);

  async function requestCode(targetEmail: string) {
    setPhase("sending");
    setError("");
    try {
      const response = await apiClient.post("/auth/oauth/code", { email: targetEmail });
      const data = response.data?.data ?? response.data;
      setMaskedEmail(data?.email_masked || targetEmail);
      if (data?.dev_code) {
        setDevCode(data.dev_code);
        setCode(data.dev_code);
      }
      setPhase("code");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not send a sign-in code.");
      setPhase("error");
    }
  }

  async function handleVerify(event: React.FormEvent) {
    event.preventDefault();
    if (code.length !== 6) {
      setError("Enter the 6-digit code from your email.");
      return;
    }
    setPhase("verifying");
    setError("");
    try {
      await apiClient.post("/auth/oauth/verify", { email, code });
      setVerified(true);
      await authorize();
    } catch (err) {
      setError(err instanceof Error ? err.message : "That code is invalid or has expired.");
      setPhase("code");
    }
  }

  async function authorize() {
    setPhase("authorizing");
    setError("");
    try {
      const apps = await listApplications();
      let targetApp = apps.find(
        (app) => app.name.toLowerCase().includes("bizup") && app.status === "active",
      );

      let secret = "";
      if (!targetApp) {
        const created = await createApplication({
          name: "BizUp Store",
          description: `Automated WhatsApp Commerce (Business: ${businessId || "default"})`,
          keyScope: "general",
        });
        targetApp = created.application;
        secret =
          created.apiKey ||
          created.keys?.find((key) => key.environment === "live")?.secret ||
          created.keys?.[0]?.secret ||
          "";
      }

      if (!secret && targetApp) {
        const key = await createApplicationApiKey({
          applicationId: targetApp.id,
          environment: "live",
          name: "BizUp Store Token",
          scope: "general",
        });
        secret = key.secret;
      }

      if (!secret) {
        throw new Error("PayConnect could not create an API key for BizUp.");
      }

      if (webhookUrl && targetApp) {
        try {
          const webhooks = await listWebhooks();
          if (!webhooks.some((webhook) => webhook.url === webhookUrl)) {
            await createWebhook({
              url: webhookUrl,
              environment: "live",
              events: ["payment.success", "payment.failed"],
              applicationId: targetApp.id,
            });
          }
        } catch {
          // A webhook can be retried from the dashboard; the connection itself is valid.
        }
      }

      setPhase("done");

      if (!delivered.current) {
        delivered.current = true;
        if (window.opener && !window.opener.closed) {
          window.opener.postMessage(
            {
              type: "PAYCONNECT_CONNECTED",
              apiKey: secret,
              baseUrl: API_BASE_URL,
              businessId,
              state,
            },
            BIZUP_ORIGIN,
          );
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to authorize BizUp.");
      setPhase("error");
    }
  }

  if (!emailResolved) {
    return (
      <AuthShell>
        <div className="space-y-4 py-10 text-center">
          <LoaderCircle className="mx-auto size-9 animate-spin text-brand-500" />
          <p className="text-sm font-semibold text-slate-800">Opening BizUp connection…</p>
        </div>
      </AuthShell>
    );
  }

  if (!email && phase === "sending") {
    return (
      <AuthShell>
        <div className="space-y-4 py-8 text-center">
          <XCircle className="mx-auto size-10 text-red-500" />
          <h1 className="text-lg font-bold text-slate-950">Open this from BizUp</h1>
          <p className="text-xs text-slate-500">
            This window only works when BizUp starts the connection.
          </p>
        </div>
      </AuthShell>
    );
  }

  if (phase === "sending" || phase === "verifying" || phase === "authorizing") {
    const message =
      phase === "sending"
        ? "Sending a sign-in code…"
        : phase === "verifying"
          ? "Verifying your code…"
          : "Connecting BizUp…";
    return (
      <AuthShell>
        <div className="space-y-4 py-10 text-center">
          <LoaderCircle className="mx-auto size-9 animate-spin text-brand-500" />
          <p className="text-sm font-semibold text-slate-800">{message}</p>
          <p className="text-xs text-slate-500">
            {phase === "sending" ? `Checking ${email}` : "This only takes a moment."}
          </p>
        </div>
      </AuthShell>
    );
  }

  if (phase === "done") {
    return (
      <AuthShell
        footnote={
          <div className="mt-6 flex items-center justify-center gap-1.5 text-xs text-slate-400">
            <Shield className="size-3.5 text-brand-500" />
            <span>Connected securely to BizUp</span>
          </div>
        }
      >
        <div className="space-y-5 py-2">
          <div className="space-y-2 text-center">
            <div className="mx-auto flex size-14 items-center justify-center rounded-2xl bg-emerald-100 text-emerald-600">
              <CheckCircle2 className="size-8" />
            </div>
            <h1 className="text-xl font-bold text-slate-950">BizUp is connected</h1>
            <p className="text-xs text-slate-500">
              Your processors and routing are now available inside BizUp. You can close this
              window.
            </p>
          </div>

          <div className="space-y-2 rounded-xl border border-slate-200 bg-white p-4 text-[11px] leading-relaxed text-slate-600">
            <p className="flex items-start gap-2">
              <ShieldCheck className="mt-0.5 size-4 shrink-0 text-emerald-600" />
              A BizUp-specific application was created in PayConnect. The key stays in
              PayConnect and is delivered to BizUp securely.
            </p>
            <p className="flex items-start gap-2">
              <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-600" />
              BizUp now checks your processors and only allows live payment routing when a
              live provider is available.
            </p>
          </div>

          <Button type="button" variant="secondary" className="w-full" onClick={() => window.close()}>
            Close window
          </Button>
        </div>
      </AuthShell>
    );
  }

  if (phase === "error" && !maskedEmail) {
    return (
      <AuthShell>
        <div className="space-y-4 py-8 text-center">
          <XCircle className="mx-auto size-10 text-red-500" />
          <h1 className="text-lg font-bold text-slate-950">Could not connect</h1>
          <p className="text-xs leading-relaxed text-slate-500">{error}</p>
          <div className="flex gap-2">
            {email && (
              <Button type="button" className="flex-1" onClick={() => void requestCode(email)}>
                Try again
              </Button>
            )}
            <Button type="button" variant="secondary" className="flex-1" onClick={() => window.close()}>
              Close window
            </Button>
          </div>
        </div>
      </AuthShell>
    );
  }

  if (phase === "error" && verified) {
    return (
      <AuthShell>
        <div className="space-y-4 py-8 text-center">
          <XCircle className="mx-auto size-10 text-red-500" />
          <h1 className="text-lg font-bold text-slate-950">Could not finish the connection</h1>
          <p className="text-xs leading-relaxed text-slate-500">{error}</p>
          <div className="flex gap-2">
            <Button type="button" className="flex-1" onClick={() => void authorize()}>
              Retry connection
            </Button>
            <Button type="button" variant="secondary" className="flex-1" onClick={() => window.close()}>
              Close window
            </Button>
          </div>
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell
      footnote={
        <div className="mt-6 flex items-center justify-center gap-1.5 text-xs text-slate-400">
          <Shield className="size-3.5 text-brand-500" />
          <span>Secured by PayConnect Vault · Zero plaintext credentials</span>
        </div>
      }
    >
      <div className="space-y-6">
        <div className="space-y-1">
          <h1 className="text-xl font-bold tracking-tight text-slate-950">
            Confirm your PayConnect account
          </h1>
          <p className="text-xs text-slate-500">
            BizUp is connecting to the PayConnect account for{" "}
            <span className="font-semibold text-slate-700">{maskedEmail || email}</span>.
          </p>
        </div>

        <div className="flex items-start gap-2.5 rounded-xl border border-brand-200 bg-brand-50/70 p-3">
          <Mail className="mt-0.5 size-4 shrink-0 text-brand-700" />
          <p className="text-[11px] leading-relaxed text-brand-900">
            {devCode
              ? "Email delivery is bypassed locally. Use the demo code below to prove this is your account."
              : "We sent a 6-digit code to your email. Enter it below to prove this is your account. There is no sign-up in this window."}
          </p>
        </div>

        {devCode && (
          <div className="rounded-xl border border-amber-200 bg-amber-50 p-3">
            <p className="text-[11px] font-semibold uppercase tracking-wider text-amber-800">
              Development demo code
            </p>
            <p className="mt-1 font-mono text-lg tracking-[0.25em] text-amber-950">{devCode}</p>
            <p className="mt-1 text-[11px] leading-relaxed text-amber-800">
              Email delivery is bypassed because OTP_DEV_MODE is on. This is not available in
              production.
            </p>
          </div>
        )}

        <form onSubmit={handleVerify} className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="otp">Sign-in code</Label>
            <Input
              id="otp"
              inputMode="numeric"
              autoComplete="one-time-code"
              autoFocus
              maxLength={6}
              value={code}
              onChange={(event) => setCode(event.target.value.replace(/\D/g, "").slice(0, 6))}
              placeholder="000000"
              className="text-center font-mono text-lg tracking-[0.4em]"
            />
          </div>

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-2.5 text-xs text-red-700">
              {error}
            </div>
          )}

          <Button type="submit" className="w-full" disabled={code.length !== 6}>
            <Lock className="mr-2 size-3.5" />
            Verify and connect BizUp
            <ArrowRight className="ml-2 size-3.5" />
          </Button>
        </form>

        <button
          type="button"
          onClick={() => {
            requestedFor.current = "";
            void requestCode(email);
          }}
          className="w-full text-center text-[11px] font-semibold text-brand-700 hover:underline"
        >
          Didn&apos;t get a code? Send another
        </button>

        <div className="space-y-2.5 rounded-xl border border-slate-200/80 bg-white p-4">
          <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            BizUp will be able to
          </p>
          <Permission icon={<ShieldCheck className="size-4 text-emerald-600" />}>
            Route card, bank transfer and USSD charges through processors already connected
            to this account.
          </Permission>
          <Permission icon={<Globe2 className="size-4 text-sky-600" />}>
            Receive payment webhooks so paid WhatsApp orders are reconciled automatically.
          </Permission>
          <Permission icon={<KeyRound className="size-4 text-brand-600" />}>
            Use a BizUp-specific API key; your provider secret keys stay inside PayConnect.
          </Permission>
        </div>
      </div>
    </AuthShell>
  );
}

function Permission({ icon, children }: { icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2.5 text-[11px] leading-relaxed text-slate-600">
      <span className="mt-0.5 shrink-0">{icon}</span>
      <span>{children}</span>
    </div>
  );
}
