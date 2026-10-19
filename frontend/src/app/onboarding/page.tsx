"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Copy,
  KeyRound,
  Rocket,
} from "lucide-react";
import { Button, Card, Input, Label } from "@/components/ui";
import { MaskedSecret, SecretInput, SecurityNotice } from "@/components/ui/part2";
import { Logo } from "@/components/ui/logo";
import { toast } from "@/components/ui/toaster";
import { generateApiKey } from "@/lib/utils";
import { PROVIDER_META, type ProviderKind } from "@/lib/types";
import { createProviderConnection } from "@/lib/api";

const USE_CASES = ["Ecommerce", "SaaS", "Agency", "Mobile app", "Custom application"];

const STEPS = ["Use case", "Provider", "Application", "Ready"];

export default function OnboardingPage() {
  const router = useRouter();
  const [step, setStep] = React.useState(0);
  const [useCase, setUseCase] = React.useState<string | null>(null);
  const [provider, setProvider] = React.useState<ProviderKind | null>(null);
  const [secretKey, setSecretKey] = React.useState("");
  const [monnifyAPIKey, setMonnifyAPIKey] = React.useState("");
  const [contractCode, setContractCode] = React.useState("");
  const [appName, setAppName] = React.useState("");
  const [connecting, setConnecting] = React.useState(false);
  const [apiKey, setApiKey] = React.useState<string | null>(null);
  const [copied, setCopied] = React.useState(false);

  const providerOrder: ProviderKind[] = ["paystack", "flutterwave", "monnify"];

  async function connectProvider() {
    if (!provider || !secretKey || (provider === "monnify" && (!monnifyAPIKey || !contractCode))) {
      toast("warning", "Almost there", "Choose a provider and paste its secret key.");
      return;
    }
    setConnecting(true);
    try {
      await createProviderConnection({
        provider,
        environment: "test",
        secretKey: secretKey.trim(),
        apiKey: provider === "monnify" ? monnifyAPIKey.trim() : undefined,
        contractCode: provider === "monnify" ? contractCode.trim() : undefined,
      });
      setSecretKey("");
      setMonnifyAPIKey("");
      setContractCode("");
      toast("success", `${PROVIDER_META[provider].name} connected`, "Test credentials verified and stored securely.");
      setStep(2);
    } catch (error) {
      toast("error", "Provider connection failed", error instanceof Error ? error.message : "Check your credentials and try again.");
    } finally {
      setConnecting(false);
    }
  }

  function createApplication() {
    if (!appName.trim()) {
      toast("warning", "Name your application", "Give it a name you'll recognize later.");
      return;
    }
    setApiKey(generateApiKey("test"));
    setStep(3);
  }

  async function copyKey() {
    if (!apiKey) return;
    await navigator.clipboard.writeText(apiKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <div className="min-h-screen bg-[#F8FAFC]">
      <header className="border-b border-slate-100 bg-white">
        <div className="mx-auto flex h-16 max-w-3xl items-center justify-between px-5">
          <Logo />
          <span className="text-[13px] text-slate-400">
            Step {step + 1} of {STEPS.length}
          </span>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-5 py-10">
        {/* Progress */}
        <div className="mb-10 flex items-center gap-2">
          {STEPS.map((label, i) => (
            <React.Fragment key={label}>
              <div className="flex items-center gap-2">
                <span
                  className={`flex size-6 items-center justify-center rounded-full text-[11px] font-semibold transition-colors ${
                    i < step
                      ? "bg-brand-500 text-slate-950"
                      : i === step
                        ? "border-2 border-brand-500 text-brand-600"
                        : "border border-slate-300 text-slate-400"
                  }`}
                >
                  {i < step ? <Check className="size-3" /> : i + 1}
                </span>
                <span className={`hidden text-[12.5px] font-medium sm:block ${i <= step ? "text-slate-900" : "text-slate-400"}`}>
                  {label}
                </span>
              </div>
              {i < STEPS.length - 1 ? (
                <div className={`h-px flex-1 ${i < step ? "bg-brand-400" : "bg-slate-200"}`} />
              ) : null}
            </React.Fragment>
          ))}
        </div>

        {/* Step 0 — use case (optional) */}
        {step === 0 ? (
          <Card className="animate-fade-up">
            <div className="p-8">
              <h1 className="text-xl font-semibold tracking-tight text-slate-900">Welcome to PayConnect</h1>
              <p className="mt-1.5 text-sm text-slate-500">
                What are you building? <span className="text-slate-400">(optional)</span>
              </p>
              <div className="mt-6 flex flex-wrap gap-2">
                {USE_CASES.map((u) => (
                  <button
                    key={u}
                    onClick={() => setUseCase(u)}
                    className={`rounded-full border px-4 py-2 text-[13px] font-medium transition-colors ${
                      useCase === u
                        ? "border-brand-400 bg-brand-50 text-brand-700"
                        : "border-slate-200 bg-white text-slate-600 hover:border-slate-300"
                    }`}
                  >
                    {u}
                  </button>
                ))}
              </div>
              <div className="mt-8 flex justify-between">
                <Link href="/dashboard">
                  <Button variant="ghost">Skip setup</Button>
                </Link>
                <Button onClick={() => setStep(1)}>
                  Continue <ArrowRight className="size-4" />
                </Button>
              </div>
            </div>
          </Card>
        ) : null}

        {/* Step 1 — connect provider */}
        {step === 1 ? (
          <Card className="animate-fade-up">
            <div className="p-8">
              <h1 className="text-xl font-semibold tracking-tight text-slate-900">Connect your first provider</h1>
              <p className="mt-1.5 text-sm text-slate-500">
                We'll validate the key, encrypt it, and never show it again.
              </p>

              <div className="mt-6 grid gap-2.5 sm:grid-cols-3">
                {providerOrder.map((p) => {
                  const meta = PROVIDER_META[p];
                  return (
                    <button
                      key={p}
                      onClick={() => setProvider(p)}
                      className={`flex flex-col items-start gap-2 rounded-xl border p-4 text-left transition-colors ${
                        provider === p
                          ? "border-brand-400 bg-brand-50/60 ring-1 ring-brand-200"
                          : "border-slate-200 bg-white hover:border-slate-300"
                      }`}
                    >
                      <span className={`flex size-8 items-center justify-center rounded-lg text-[10px] font-bold ${meta.accent}`}>
                        {meta.initials}
                      </span>
                      <span className="text-[13.5px] font-semibold text-slate-900">{meta.name}</span>
                      <span className="text-[11.5px] leading-snug text-slate-400">{meta.description}</span>
                    </button>
                  );
                })}
              </div>

              {provider === "monnify" ? (
                <>
                  <div className="mt-6 space-y-1.5">
                    <Label htmlFor="api-key">Monnify API key</Label>
                    <SecretInput id="api-key" value={monnifyAPIKey} onChange={(e) => setMonnifyAPIKey(e.target.value)} placeholder="Monnify API key" />
                  </div>
                  <div className="mt-6 space-y-1.5">
                    <Label htmlFor="contract-code">Monnify contract code</Label>
                    <Input id="contract-code" value={contractCode} onChange={(e) => setContractCode(e.target.value)} placeholder="Contract code" />
                  </div>
                </>
              ) : null}

              <div className="mt-6 space-y-1.5">
                <Label htmlFor="secret">{provider ? `${PROVIDER_META[provider].name} secret key` : "Secret key"}</Label>
                <Input
                  id="secret"
                  type="password"
                  value={secretKey}
                  onChange={(e) => setSecretKey(e.target.value)}
                  placeholder={provider === "paystack" ? "sk_test_…" : provider === "flutterwave" ? "FLWSECK-…" : "Monnify secret key"}
                  className="font-mono"
                />
              </div>

              <SecurityNotice className="mt-5">
                Your credentials are <b>encrypted and never shared with developers</b> — not in logs, exports or
                API responses.
              </SecurityNotice>

              <div className="mt-8 flex justify-between">
                <Button variant="ghost" onClick={() => setStep(0)}>
                  <ArrowLeft className="size-4" /> Back
                </Button>
                <Button onClick={connectProvider} disabled={connecting || !provider || !secretKey.trim()}>
                  {connecting ? "Validating & encrypting…" : "Validate & Connect"}
                </Button>
              </div>
            </div>
          </Card>
        ) : null}

        {/* Step 2 — create application */}
        {step === 2 ? (
          <Card className="animate-fade-up">
            <div className="p-8">
              <div className="flex size-10 items-center justify-center rounded-xl border border-emerald-200 bg-emerald-50">
                <Check className="size-5 text-emerald-600" />
              </div>
              <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">
                {PROVIDER_META[provider ?? "paystack"].name} connected
              </h1>
              <p className="mt-1.5 text-sm text-slate-500">Now create your first application to get an API key.</p>

              <div className="mt-6 space-y-1.5">
                <Label htmlFor="app-name">Application name</Label>
                <Input
                  id="app-name"
                  value={appName}
                  onChange={(e) => setAppName(e.target.value)}
                  placeholder="My Ecommerce Store"
                />
              </div>

              <div className="mt-8 flex justify-between">
                <Button variant="ghost" onClick={() => setStep(1)}>
                  <ArrowLeft className="size-4" /> Back
                </Button>
                <Button onClick={createApplication}>
                  Create application <ArrowRight className="size-4" />
                </Button>
              </div>
            </div>
          </Card>
        ) : null}

        {/* Step 3 — ready */}
        {step === 3 ? (
          <Card className="animate-fade-up">
            <div className="p-8 text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-2xl border border-brand-100 bg-brand-50">
                <Rocket className="size-6 text-brand-500" />
              </div>
              <h1 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">You're ready.</h1>
              <p className="mx-auto mt-1.5 max-w-sm text-sm text-slate-500">
                This test API key is for <span className="font-medium text-slate-700">{appName || "your application"}</span>.
                Hand it to your developer — never your provider secret.
              </p>

              <div className="mx-auto mt-6 max-w-md rounded-xl border border-slate-200 bg-slate-50 p-4">
                <div className="flex items-center gap-2">
                  <KeyRound className="size-4 shrink-0 text-brand-500" />
                  <code className="min-w-0 flex-1 truncate font-mono text-[13px] text-slate-800">{apiKey}</code>
                  <button
                    onClick={copyKey}
                    className="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[12px] font-medium text-slate-600 hover:bg-slate-50"
                  >
                    {copied ? <Check className="size-3.5 text-emerald-500" /> : <Copy className="size-3.5" />}
                    {copied ? "Copied" : "Copy API Key"}
                  </button>
                </div>
              </div>

              <p className="mx-auto mt-4 max-w-md text-left text-[12px] leading-relaxed text-amber-700">
                ⚠️ Copy this key now — you won't be able to view the full key again.
              </p>

              <Button size="lg" className="mt-8" onClick={() => router.push("/dashboard")}>
                Go to dashboard <ArrowRight className="size-4" />
              </Button>
            </div>
          </Card>
        ) : null}
      </main>
    </div>
  );
}
