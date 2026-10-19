
"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  ExternalLink,
  Lock,
  ShieldCheck,
  TriangleAlert,
} from "lucide-react";

import {
  Button,
  Card,
  CopyField,
  Label,
  PageHeader,
  Radio,
} from "@/components/ui";
import {
  SecretInput,
  SecurityNotice,
} from "@/components/ui/part2";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import {
  createProviderConnection,
  listProviderCapabilities,
  type ProviderWebhookCapability,
} from "@/lib/api";
import {
  PROVIDER_META,
  type ProviderConnection,
  type ProviderKind,
} from "@/lib/types";

const VALID: ProviderKind[] = [
  "paystack",
  "flutterwave",
  "moniepoint",
];

type Environment = "test" | "live";

function providerRejectionDetails(error: unknown): string | null {
  if (typeof error !== "object" || error === null || !("error" in error)) {
    return null;
  }
  const apiError = error.error;
  if (typeof apiError !== "object" || apiError === null || !("details" in apiError)) {
    return null;
  }
  const details = apiError.details;
  if (typeof details !== "object" || details === null) {
    return null;
  }
  const message = "provider_message" in details ? details.provider_message : null;
  const status = "provider_status" in details ? details.provider_status : null;
  if (typeof message !== "string" || message === "") {
    return null;
  }
  return typeof status === "number"
    ? `${message} (HTTP ${status})`
    : message;
}

function Progress({ step }: { step: 1 | 2 }) {
  const items = [
    {
      title: "Credentials",
      description: "Verify your API access",
    },
    {
      title: "Webhooks",
      description: "Receive payment updates",
    },
  ];

  return (
    <ol
      className="grid grid-cols-2 gap-3"
      aria-label="Connection progress"
    >
      {items.map((item, index) => {
        const number = index + 1;
        const active = number === step;
        const complete = number < step;

        return (
          <li
            key={item.title}
            aria-current={active ? "step" : undefined}
            className={`relative rounded-xl border p-4 transition-colors ${
              active
                ? "border-brand-300 bg-brand-50/60"
                : complete
                  ? "border-emerald-200 bg-emerald-50/50"
                  : "border-slate-200 bg-white"
            }`}
          >
            <div className="flex items-start gap-3">
              <span
                className={`flex size-8 shrink-0 items-center justify-center rounded-lg text-sm font-semibold ${
                  complete
                    ? "bg-emerald-100 text-emerald-700"
                    : active
                      ? "bg-brand-500 text-white"
                      : "bg-slate-100 text-slate-500"
                }`}
              >
                {complete ? (
                  <Check className="size-4" />
                ) : (
                  number
                )}
              </span>

              <div className="min-w-0">
                <p
                  className={`text-sm font-semibold ${
                    active
                      ? "text-brand-700"
                      : "text-slate-900"
                  }`}
                >
                  {item.title}
                </p>

                <p className="mt-1 text-xs leading-relaxed text-slate-500">
                  {item.description}
                </p>
              </div>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

function ProviderMark({
  provider,
}: {
  provider: ProviderKind;
}) {
  const meta = PROVIDER_META[provider];

  return (
    <div
      className={`flex size-12 shrink-0 items-center justify-center rounded-xl text-sm font-bold ${meta.accent}`}
      aria-label={meta.name}
    >
      {meta.initials}
    </div>
  );
}

export default function ConnectProviderPage() {
  const params = useParams<{ provider: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();

  const provider = VALID.includes(
    params.provider as ProviderKind
  )
    ? (params.provider as ProviderKind)
    : null;

  const [environment, setEnvironment] =
    React.useState<Environment>(
      searchParams.get("environment") === "live"
        ? "live"
        : "test"
    );

  const [secretKey, setSecretKey] = React.useState("");
  const [webhookSecret, setWebhookSecret] = React.useState("");
  const [validating, setValidating] = React.useState(false);
  const [connected, setConnected] =
    React.useState<ProviderConnection | null>(null);

  const capabilities = useAsync(
    listProviderCapabilities,
    []
  );

  if (!provider) {
    return (
      <div className="mx-auto max-w-xl rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm sm:p-12">
        <div className="mx-auto flex size-12 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
          <TriangleAlert className="size-5" />
        </div>

        <h1 className="mt-4 text-lg font-semibold text-slate-900">
          Provider not supported
        </h1>

        <p className="mt-2 text-sm leading-relaxed text-slate-500">
          This payment provider is not available for connection
          yet. Choose one of the supported providers to continue.
        </p>

        <Link href="/providers" className="mt-6 inline-flex">
          <Button variant="secondary">
            <ArrowLeft className="size-4" />
            Back to providers
          </Button>
        </Link>
      </div>
    );
  }

  const providerKind = provider;
  const meta = PROVIDER_META[providerKind];

  const capability: ProviderWebhookCapability | undefined = (
    capabilities.data ?? []
  ).find((item) => item.provider === providerKind);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!secretKey.trim()) {
      toast(
        "warning",
        "Secret key required",
        "Paste the secret key from your provider dashboard."
      );
      return;
    }

    if (
      environment === "live" &&
      !window.confirm(
        "You are connecting production credentials. These credentials may be used to process real customer payments. Continue?"
      )
    ) {
      return;
    }

    setValidating(true);

    try {
      const connection = await createProviderConnection({
        provider: providerKind,
        environment,
        secretKey: secretKey.trim(),
        webhookSecret: capability?.webhookSecretRequired
          ? webhookSecret.trim()
          : undefined,
      });

      setSecretKey("");
      setWebhookSecret("");

      void queryClient.invalidateQueries({
        queryKey: ["providers"],
      });

      toast(
        "success",
        `${meta.name} connected`,
        `${
          environment === "live" ? "Production" : "Test"
        } credentials verified and stored securely.`
      );

      setConnected(connection);
    } catch (error) {
      toast(
        "error",
        `${meta.name} connection failed`,
        providerRejectionDetails(error) ??
          (error instanceof Error
            ? error.message
            : "Check your credentials and try again.")
      );
    } finally {
      setValidating(false);
    }
  }

  if (connected) {
    return (
      <div className="mx-auto max-w-4xl animate-fade-in space-y-6">
        <PageHeader
          breadcrumb={[
            { label: "Providers", href: "/providers" },
            { label: `Connect ${meta.name}` },
          ]}
          title={`${meta.name} is connected`}
          description="Your credentials have been verified. Complete the webhook setup so PayConnect can receive payment updates."
        />

        <Progress step={2} />

        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_280px]">
          <Card>
            <div className="border-b border-slate-100 p-5 sm:p-6">
              <div className="flex items-center gap-3">
                <ProviderMark provider={providerKind} />

                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="font-semibold text-slate-900">
                      {meta.name}
                    </h2>

                    <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700">
                      <Check className="size-3.5" />
                      Connected
                    </span>
                  </div>

                  <p className="mt-1 text-xs text-slate-500">
                    {environment === "live"
                      ? "Production environment"
                      : "Test environment"}
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-7 p-5 sm:p-6">
              <section>
                <div className="flex items-start gap-3">
                  <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-semibold text-slate-700">
                    1
                  </span>

                  <div>
                    <h3 className="text-sm font-semibold text-slate-900">
                      Open your webhook settings
                    </h3>

                    <p className="mt-1 text-sm leading-relaxed text-slate-500">
                      In {meta.name}, open the webhook settings
                      {connected.webhookDashboardPath
                        ? ` under ${connected.webhookDashboardPath}`
                        : ""}.
                    </p>
                  </div>
                </div>
              </section>

              <section>
                <div className="flex items-start gap-3">
                  <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-semibold text-slate-700">
                    2
                  </span>

                  <div className="min-w-0 flex-1">
                    <h3 className="text-sm font-semibold text-slate-900">
                      Add your PayConnect endpoint
                    </h3>

                    <p className="mt-1 text-sm leading-relaxed text-slate-500">
                      Copy this URL and paste it into the webhook
                      endpoint field in your provider dashboard.
                    </p>

                    <CopyField
                      className="mt-3"
                      value={connected.webhookUrl}
                    />
                  </div>
                </div>
              </section>

              <section>
                <div className="flex items-start gap-3">
                  <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-semibold text-slate-700">
                    3
                  </span>

                  <div>
                    <h3 className="text-sm font-semibold text-slate-900">
                      Verify webhook signatures
                    </h3>

                    <p className="mt-1 text-sm leading-relaxed text-slate-500">
                      {connected.webhookSecretSet
                        ? `PayConnect will verify signatures using the ${
                            connected.webhookSecretLabel ||
                            "webhook secret"
                          } saved for this connection.`
                        : connected.webhookSecretRequired
                          ? `Set a ${
                              connected.webhookSecretLabel ||
                              "webhook secret"
                            } in ${meta.name}, then save the same value on this connection so signatures can be verified.`
                          : `${meta.name} signs webhooks using your secret key, so no additional webhook secret is required here.`}
                    </p>
                  </div>
                </div>
              </section>

              <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 pt-5">
                <Link href="/providers">
                  <Button variant="secondary">
                    Back to providers
                  </Button>
                </Link>

                <Button
                  onClick={() =>
                    router.push(`/providers/${connected.id}`)
                  }
                >
                  Open connection
                  <ArrowRight className="size-4" />
                </Button>
              </div>
            </div>
          </Card>

          <aside className="space-y-4">
            <Card>
              <div className="p-5">
                <div className="flex size-10 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700">
                  <ShieldCheck className="size-5" />
                </div>

                <h3 className="mt-4 text-sm font-semibold text-slate-900">
                  Credentials secured
                </h3>

                <p className="mt-2 text-xs leading-relaxed text-slate-500">
                  Your credentials have been accepted by PayConnect.
                  Keep your provider secret private.
                </p>
              </div>
            </Card>

            <Card>
              <div className="p-5">
                <h3 className="text-sm font-semibold text-slate-900">
                  What happens next?
                </h3>

                <ul className="mt-3 space-y-3 text-xs leading-relaxed text-slate-500">
                  <li className="flex gap-2">
                    <Check className="size-4 shrink-0 text-emerald-600" />
                    Configure the provider webhook.
                  </li>
                  <li className="flex gap-2">
                    <Check className="size-4 shrink-0 text-emerald-600" />
                    Send a test event where supported.
                  </li>
                  <li className="flex gap-2">
                    <Check className="size-4 shrink-0 text-emerald-600" />
                    Monitor incoming provider events in PayConnect.
                  </li>
                </ul>
              </div>
            </Card>
          </aside>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl animate-fade-in space-y-6">
      <PageHeader
        breadcrumb={[
          { label: "Providers", href: "/providers" },
          { label: `Connect ${meta.name}` },
        ]}
        title={`Connect ${meta.name}`}
        description="Connect your payment account to manage credentials and receive payment updates through PayConnect."
      />

      <Progress step={1} />

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_280px]">
        <Card>
          <form onSubmit={onSubmit}>
            <div className="border-b border-slate-100 p-5 sm:p-6">
              <div className="flex items-center gap-3">
                <ProviderMark provider={providerKind} />

                <div className="min-w-0">
                  <h2 className="text-base font-semibold text-slate-900">
                    {meta.name}
                  </h2>

                  <p className="mt-1 text-sm leading-relaxed text-slate-500">
                    {meta.description}
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-6 p-5 sm:p-6">
              <fieldset>
                <legend className="text-sm font-semibold text-slate-900">
                  Choose an environment
                </legend>

                <p className="mt-1 text-xs leading-relaxed text-slate-500">
                  Test and production credentials are separate
                  connections. Choose the environment you intend
                  to configure.
                </p>

                <div className="mt-4 grid gap-3 sm:grid-cols-2">
                  {(["test", "live"] as const).map((env) => {
                    const selected = environment === env;
                    const isLive = env === "live";

                    return (
                      <label
                        key={env}
                        className={`flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors ${
                          selected
                            ? isLive
                              ? "border-amber-300 bg-amber-50/50"
                              : "border-brand-300 bg-brand-50/50"
                            : "border-slate-200 bg-white hover:border-slate-300"
                        }`}
                      >
                        <Radio
                          name="environment"
                          checked={selected}
                          onChange={() => setEnvironment(env)}
                          className="mt-0.5"
                        />

                        <span className="min-w-0">
                          <span className="flex flex-wrap items-center gap-2 text-sm font-semibold text-slate-900">
                            {isLive ? "Production" : "Test"}

                            <span
                              className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${
                                isLive
                                  ? "bg-amber-100 text-amber-800"
                                  : "bg-brand-100 text-brand-700"
                              }`}
                            >
                              {isLive ? "LIVE" : "SANDBOX"}
                            </span>
                          </span>

                          <span className="mt-1.5 block text-xs leading-relaxed text-slate-500">
                            {isLive
                              ? "Connect credentials used for real customer payments."
                              : "Use sandbox credentials for development and testing."}
                          </span>
                        </span>
                      </label>
                    );
                  })}
                </div>

                {environment === "live" && (
                  <div className="mt-3 flex items-start gap-2.5 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs leading-relaxed text-amber-800">
                    <TriangleAlert className="mt-0.5 size-4 shrink-0" />

                    <p>
                      <strong>Production environment.</strong> These
                      credentials may authorize real payments.
                      Confirm that you are using the correct live
                      secret key.
                    </p>
                  </div>
                )}
              </fieldset>

              <div className="border-t border-slate-100" />

              <section>
                <div className="mb-4">
                  <h3 className="text-sm font-semibold text-slate-900">
                    API credentials
                  </h3>

                  <p className="mt-1 text-xs leading-relaxed text-slate-500">
                    Enter the secret key from your {meta.name} dashboard.
                    Your public key is not a replacement for this key.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="secret">
                    {meta.name} secret key
                  </Label>

                  <SecretInput
                    id="secret"
                    value={secretKey}
                    onChange={(e) => setSecretKey(e.target.value)}
                    placeholder={
                      providerKind === "paystack"
                        ? `sk_${environment}_••••••••••••`
                        : providerKind === "flutterwave"
                          ? "FLWSECK-••••••••••••"
                          : "Enter your Moniepoint secret token"
                    }
                    required
                  />

                  <p className="text-xs leading-relaxed text-slate-400">
                    Find this under API keys or API settings in
                    your provider dashboard.
                  </p>
                </div>

                {capability?.webhookSecretRequired && (
                  <div className="mt-5 space-y-2">
                    <Label htmlFor="webhook-secret">
                      {capability.webhookSecretLabel ||
                        "Webhook secret"}

                      <span className="ml-1 font-normal text-slate-400">
                        (optional during connection)
                      </span>
                    </Label>

                    <SecretInput
                      id="webhook-secret"
                      value={webhookSecret}
                      onChange={(e) =>
                        setWebhookSecret(e.target.value)
                      }
                      placeholder="Enter your webhook signing secret"
                    />

                    <p className="text-xs leading-relaxed text-slate-500">
                      Set this value in your provider dashboard
                      {capability.webhookDashboardPath
                        ? ` under ${capability.webhookDashboardPath}`
                        : ""}.
                      Use the same value here to verify webhook
                      signatures.
                    </p>
                  </div>
                )}
              </section>

              <SecurityNotice>
                Your credentials are{" "}
                <strong>encrypted and never shared with developers</strong>.
                After validation, PayConnect displays only the last
                four characters of your stored secret.
              </SecurityNotice>

              <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 pt-5">
                <Link href="/providers">
                  <Button variant="ghost">
                    <ArrowLeft className="size-4" />
                    Cancel
                  </Button>
                </Link>

                <Button
                  type="submit"
                  disabled={validating || !secretKey.trim()}
                >
                  {validating ? (
                    <>
                      <ShieldCheck className="size-4 animate-pulse" />
                      Validating…
                    </>
                  ) : (
                    <>
                      <Lock className="size-4" />
                      Validate and connect
                      <ArrowRight className="size-4" />
                    </>
                  )}
                </Button>
              </div>
            </div>
          </form>
        </Card>

        <aside className="space-y-4">
          <Card>
            <div className="p-5">
              <div className="flex size-10 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700">
                <ShieldCheck className="size-5" />
              </div>

              <h3 className="mt-4 text-sm font-semibold text-slate-900">
                Built for secure payments
              </h3>

              <p className="mt-2 text-xs leading-relaxed text-slate-500">
                Your provider credentials stay separate from
                developer access. Only authorized payment services
                should use these secrets.
              </p>

              <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                {[
                  "Credentials verified before use",
                  "Encrypted credential storage",
                  "Separate test and production connections",
                ].map((item) => (
                  <div key={item} className="flex gap-2">
                    <Check className="size-4 shrink-0 text-emerald-600" />
                    <p className="text-xs leading-relaxed text-slate-600">
                      {item}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </Card>

          <Card>
            <div className="p-5">
              <h3 className="text-sm font-semibold text-slate-900">
                Need help?
              </h3>

              <p className="mt-1 text-xs leading-relaxed text-slate-500">
                Open your provider dashboard to find your API
                credentials and webhook settings.
              </p>

              <div className="mt-4 space-y-2">
                <a
                  href={
                    providerKind === "paystack"
                      ? "https://dashboard.paystack.com/"
                      : providerKind === "flutterwave"
                        ? "https://app.flutterwave.com/"
                        : "https://moniepoint.com/"
                  }
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex w-full items-center justify-between rounded-lg border border-slate-200 px-3 py-2.5 text-xs font-medium text-slate-700 transition-colors hover:border-brand-200 hover:bg-brand-50/40 hover:text-brand-700"
                >
                  Open {meta.name} dashboard
                  <ExternalLink className="size-3.5" />
                </a>

                <p className="text-[11px] leading-relaxed text-slate-400">
                  Make sure you are signed in to the correct
                  provider account before copying credentials.
                </p>
              </div>
            </div>
          </Card>
        </aside>
      </div>
    </div>
  );
}