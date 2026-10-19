"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, CreditCard, Globe, Lock, PlugZap, RotateCcw, Unplug, X } from "lucide-react";
import { Badge, Button, Card, CopyField, EnvironmentBadge, Input, Label, PageHeader } from "@/components/ui";
import { Modal, SecretInput, SecurityNotice } from "@/components/ui/part2";
import { useAsync } from "@/hooks/use-async";
import {
  createProviderConnection,
  createTestTransaction,
  deleteProviderConnection,
  getProvider,
  listInboundProviderEvents,
  setProviderWebhookSecret,
  testProviderConnection,
  updateProviderConnection,
} from "@/lib/api";
import { formatDateTime, timeAgo } from "@/lib/utils";
import { PROVIDER_META, type ProviderConnection } from "@/lib/types";
import { toast } from "@/components/ui/toaster";

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 py-2.5">
      <dt className="text-[13px] text-slate-500">{label}</dt>
      <dd className="text-right text-[13px] font-medium text-slate-900">{children}</dd>
    </div>
  );
}

function Status({ status }: { status: ProviderConnection["status"] }) {
  const tone =
    status === "connected"
      ? { text: "text-emerald-600", dot: "bg-emerald-500", label: "Connected" }
      : status === "error"
        ? { text: "text-red-600", dot: "bg-red-500", label: "Connection error" }
        : { text: "text-slate-400", dot: "border border-slate-300", label: "Not connected" };
  return (
    <span className={`inline-flex items-center gap-1.5 text-[12.5px] font-medium ${tone.text}`}>
      <span className={`size-1.5 rounded-full ${tone.dot}`} />
      {tone.label}
    </span>
  );
}

/** Inbound webhooks: the URL for the provider dashboard, the signing value if needed, and recent events. */
function InboundWebhooksCard({ connection }: { connection: ProviderConnection }) {
  const queryClient = useQueryClient();
  const events = useAsync(() => listInboundProviderEvents(connection.id), [connection.id]);
  const [current, setCurrent] = React.useState(connection);
  const [secret, setSecret] = React.useState("");
  const [saving, setSaving] = React.useState(false);

  async function saveSecret(event: React.FormEvent) {
    event.preventDefault();
    if (!secret.trim()) return;
    setSaving(true);
    try {
      const updated = await setProviderWebhookSecret(current.id, secret.trim());
      setCurrent(updated);
      setSecret("");
      toast("success", `${current.webhookSecretLabel || "Webhook secret"} saved`, "Inbound webhooks from this provider will now verify.");
      void queryClient.invalidateQueries({ queryKey: ["provider", current.id] });
    } catch (error) {
      toast("error", "Could not save webhook secret", error instanceof Error ? error.message : "Please try again.");
    } finally {
      setSaving(false);
    }
  }

  const meta = PROVIDER_META[current.provider];
  const inbound = events.data ?? [];

  return (
    <Card>
      <div className="border-b border-slate-100 px-5 py-4">
        <h2 className="text-sm font-semibold text-slate-900">Provider webhooks</h2>
        <p className="text-[12px] text-slate-400">Paste this URL into {meta.name} so its events reach PayConnect.</p>
      </div>
      <div className="space-y-5 p-5">
        <CopyField value={current.webhookUrl} label={`${meta.name} webhook URL`} />

        <dl className="grid gap-x-8 sm:grid-cols-2">
          <DetailRow label="Signature header">
            <span className="font-mono text-[12px]">{current.webhookSignatureHeader}</span>
          </DetailRow>
          <DetailRow label="Last event received">
            {current.webhookLastEventAt ? timeAgo(current.webhookLastEventAt) : "None yet"}
          </DetailRow>
        </dl>

        {current.webhookSecretRequired ? (
          <form onSubmit={saveSecret} className="border-t border-slate-100 pt-5">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[13px] font-medium text-slate-800">{current.webhookSecretLabel || "Webhook secret"}</p>
                <p className="mt-0.5 font-mono text-[12px] text-slate-500">
                  {current.webhookSecretSet ? current.webhookSecretMasked : "Not set"}
                </p>
              </div>
              {current.webhookDashboardPath ? (
                <span className="text-[11.5px] text-slate-400">{current.webhookDashboardPath}</span>
              ) : null}
            </div>
            <p className="mt-2 max-w-prose text-[12px] leading-relaxed text-slate-500">
              {meta.name} signs webhooks with a value you set in its own dashboard. Enter the same value here so inbound
              webhooks verify.
            </p>
            <div className="mt-3 flex items-center gap-2">
              <div className="flex-1">
                <SecretInput
                  value={secret}
                  onChange={(e) => setSecret(e.target.value)}
                  placeholder="Enter the value from your provider dashboard"
                  aria-label={`${meta.name} webhook secret`}
                />
              </div>
              <Button type="submit" size="sm" disabled={saving || !secret.trim()}>
                {saving ? "Saving…" : current.webhookSecretSet ? "Rotate" : "Save"}
              </Button>
            </div>
          </form>
        ) : (
          <p className="flex items-start gap-2 border-t border-slate-100 pt-5 text-[12.5px] leading-relaxed text-slate-600">
            <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-500" />
            No extra setup. {meta.name} signs webhooks with your secret key, which is already stored. Paste the URL above
            and you're done.
          </p>
        )}

        <div className="border-t border-slate-100 pt-5">
          <h3 className="text-[13px] font-medium text-slate-800">Recent events</h3>
          {events.loading ? (
            <div className="mt-2 space-y-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-9 animate-pulse rounded-lg bg-slate-100" />
              ))}
            </div>
          ) : inbound.length === 0 ? (
            <p className="mt-2 text-[12.5px] text-slate-400">
              Nothing received yet. Events show up here as soon as {meta.name} calls the URL above.
            </p>
          ) : (
            <ul className="mt-1 divide-y divide-slate-100">
              {inbound.slice(0, 5).map((event) => (
                <li key={event.id} className="flex items-center justify-between gap-3 py-2.5">
                  <div className="min-w-0">
                    <p className="truncate font-mono text-[12px] text-slate-800">{event.eventType}</p>
                    <p className="text-[11px] text-slate-400">{timeAgo(event.createdAt)}</p>
                  </div>
                  <Badge variant={event.processedAt ? "success" : "neutral"}>{event.processedAt ? "Processed" : "Received"}</Badge>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </Card>
  );
}

export default function ProviderDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const { data: provider, loading, error, refetch } = useAsync(
    () => getProvider(params.id),
    [params.id],
    ["provider", params.id],
  );
  const [disconnectOpen, setDisconnectOpen] = React.useState(false);
  const [rotateOpen, setRotateOpen] = React.useState(false);
  const [newSecretKey, setNewSecretKey] = React.useState("");
  const [newAPIKey, setNewAPIKey] = React.useState("");
  const [newContractCode, setNewContractCode] = React.useState("");
  const [busy, setBusy] = React.useState<"test" | "default" | "delete" | "rotate" | null>(null);
  const [testOpen, setTestOpen] = React.useState(false);
  const [testAmount, setTestAmount] = React.useState("100");
  const [testEmail, setTestEmail] = React.useState("");
  const [testCurrency, setTestCurrency] = React.useState("");
  const [testBusy, setTestBusy] = React.useState(false);
  const [testError, setTestError] = React.useState<string | null>(null);
  const [returnedFromCheckout, setReturnedFromCheckout] = React.useState(
    () => searchParams.get("checkout") === "return",
  );

  if (loading) {
    return <div className="h-96 animate-pulse rounded-[var(--radius-card)] border border-slate-200 bg-white" />;
  }

  if (error) {
    return (
      <Card className="p-8 text-center">
        <h1 className="font-semibold text-slate-900">Could not load provider connection</h1>
        <p className="mt-2 text-sm text-slate-500">{error instanceof Error ? error.message : "Please try again."}</p>
        <Button className="mt-5" onClick={() => void refetch()}>
          Retry
        </Button>
      </Card>
    );
  }

  if (!provider) {
    return (
      <div className="rounded-[var(--radius-card)] border border-slate-200 bg-white p-14 text-center">
        <Globe className="mx-auto size-8 text-slate-300" />
        <h1 className="mt-4 text-lg font-semibold text-slate-900">Provider not connected</h1>
        <p className="mt-1 text-sm text-slate-500">Connect it first to manage credentials.</p>
        <Link href="/providers" className="mt-6 inline-block">
          <Button variant="secondary">Back to providers</Button>
        </Link>
      </div>
    );
  }

  const connection = provider;
  const meta = PROVIDER_META[connection.provider];

  function openTest() {
    setTestCurrency(connection.currency || "NGN");
    setTestError(null);
    setTestOpen(true);
  }

  async function submitTest(e: React.FormEvent) {
    e.preventDefault();
    const amountMajor = Number.parseFloat(testAmount);
    if (!Number.isFinite(amountMajor) || amountMajor <= 0) {
      toast("warning", "Enter an amount", "Provide an amount greater than zero.");
      return;
    }
    if (!testEmail.trim()) {
      toast("warning", "Email required", "Enter an email for the test payment.");
      return;
    }
    setTestBusy(true);
    setTestError(null);
    try {
      const payment = await createTestTransaction(connection.id, {
        amountMinor: Math.round(amountMajor * 100),
        currency: (testCurrency || connection.currency || "NGN").toUpperCase(),
        email: testEmail.trim(),
        callbackUrl: `${window.location.origin}/providers/${connection.id}?checkout=return`,
      });
      if (!payment.authorizationUrl) {
        setTestError("The provider accepted the charge but returned no checkout URL. Check this payment in the dashboard.");
        return;
      }
      window.location.assign(payment.authorizationUrl);
    } catch (error) {
      setTestError(error instanceof Error ? error.message : "Could not start the test transaction.");
    } finally {
      setTestBusy(false);
    }
  }

  async function refreshProvider() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["provider", connection.id] }),
      queryClient.invalidateQueries({ queryKey: ["providers"] }),
    ]);
  }

  async function testConnection() {
    setBusy("test");
    try {
      await testProviderConnection(connection.id);
      toast("success", "Connection verified", `${meta.name} credentials are working.`);
      await refreshProvider();
    } catch (error) {
      toast("error", "Connection test failed", error instanceof Error ? error.message : "Please check provider status.");
    } finally {
      setBusy(null);
    }
  }

  async function toggleDefault() {
    setBusy("default");
    try {
      await updateProviderConnection(connection.id, { isDefault: !connection.isDefault });
      toast("success", connection.isDefault ? "Default provider removed" : "Default provider updated");
      await refreshProvider();
    } catch (error) {
      toast("error", "Could not update default", error instanceof Error ? error.message : "Please try again.");
    } finally {
      setBusy(null);
    }
  }

  async function rotateCredentials(e: React.FormEvent) {
    e.preventDefault();
    if (!newSecretKey.trim()) return;
    setBusy("rotate");
    try {
      await createProviderConnection({
        provider: connection.provider,
        environment: connection.environment,
        secretKey: newSecretKey.trim(),
        apiKey: connection.provider === "monnify" ? newAPIKey.trim() : undefined,
        contractCode: connection.provider === "monnify" ? newContractCode.trim() : undefined,
        displayName: connection.displayName,
        isDefault: connection.isDefault,
      });
      setNewSecretKey("");
      setNewAPIKey("");
      setNewContractCode("");
      setRotateOpen(false);
      toast("success", "Credentials rotated", "New credentials verified and securely stored.");
      await refreshProvider();
    } catch (error) {
      toast("error", "Credential rotation failed", error instanceof Error ? error.message : "Check the new credentials.");
    } finally {
      setBusy(null);
    }
  }

  async function disconnect() {
    setBusy("delete");
    try {
      await deleteProviderConnection(connection.id);
      setDisconnectOpen(false);
      toast("success", `${meta.name} disconnected`, "Stored credentials were deleted.");
      await queryClient.invalidateQueries({ queryKey: ["providers"] });
      router.push("/providers");
    } catch (error) {
      toast("error", "Disconnect failed", error instanceof Error ? error.message : "Please try again.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="animate-fade-in">
      <PageHeader
        breadcrumb={[{ label: "Providers", href: "/providers" }, { label: meta.name }]}
        title={meta.name}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <EnvironmentBadge environment={provider.environment} />
            <Status status={provider.status} />
            <Button variant="secondary" onClick={testConnection} disabled={busy !== null}>
              <PlugZap className="size-4" /> {busy === "test" ? "Testing…" : "Test connection"}
            </Button>
            {connection.environment === "test" && connection.status === "connected" ? (
              <Button onClick={openTest} disabled={busy !== null}>
                <CreditCard className="size-4" /> Make test transaction
              </Button>
            ) : null}
          </div>
        }
      />

      {returnedFromCheckout ? (
        <div className="mb-5 flex items-start justify-between gap-4 rounded-xl border border-emerald-200 bg-emerald-50/70 p-4">
          <div className="flex items-start gap-2.5">
            <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-600" />
            <div>
              <p className="text-[13.5px] font-semibold text-emerald-900">You're back from checkout</p>
              <p className="mt-0.5 text-[12.5px] leading-relaxed text-emerald-900/80">
                If you completed the payment, {meta.name} will call PayConnect back. The events below, and the delivery to
                your endpoint, update shortly.
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => setReturnedFromCheckout(false)}
            className="rounded-md p-1 text-emerald-700/70 transition-colors hover:bg-emerald-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
            aria-label="Dismiss"
          >
            <X className="size-4" />
          </button>
        </div>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <div className="space-y-6">
          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">Credentials</h2>
              <p className="text-[12px] text-slate-400">Stored encrypted and never shown in full, on this page or anywhere else.</p>
            </div>
            <div className="p-5">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-[12px] text-slate-500">Secret key</p>
                  <p className="font-mono text-[13px] text-slate-700">••••••••••••••••••••{provider.secretLast4}</p>
                </div>
                <Button variant="secondary" size="sm" onClick={() => setRotateOpen(true)} disabled={busy !== null}>
                  <RotateCcw className="size-3.5" /> Rotate credentials
                </Button>
              </div>
              <p className="mt-4 flex items-start gap-2 border-t border-slate-100 pt-4 text-[12.5px] leading-relaxed text-slate-600">
                <Lock className="mt-0.5 size-3.5 shrink-0 text-brand-500" />
                <span>
                  Don't paste this secret anywhere else. Developers use PayConnect API keys and never need, or see, this
                  credential.
                </span>
              </p>
            </div>
          </Card>

          <InboundWebhooksCard connection={provider} />
        </div>

        <aside className="space-y-6">
          <Card>
            <div className="border-b border-slate-100 px-5 py-4">
              <h2 className="text-sm font-semibold text-slate-900">Details</h2>
            </div>
            <dl className="divide-y divide-slate-100 px-5 py-1">
              <DetailRow label="Provider">
                <span className="inline-flex items-center gap-2">
                  <span className={`flex size-5 items-center justify-center rounded text-[9px] font-bold ${meta.accent}`}>
                    {meta.initials}
                  </span>
                  {meta.name}
                </span>
              </DetailRow>
              <DetailRow label="Environment">{provider.environment === "live" ? "Production" : "Test"}</DetailRow>
              <DetailRow label="Currency">{provider.currency}</DetailRow>
              <DetailRow label="Connected since">{provider.connectedAt ? formatDateTime(provider.connectedAt) : "—"}</DetailRow>
              <DetailRow label="Last verified">{provider.lastVerifiedAt ? timeAgo(provider.lastVerifiedAt) : "—"}</DetailRow>
              <DetailRow label="Default">
                <button
                  type="button"
                  onClick={toggleDefault}
                  disabled={busy !== null}
                  className="text-brand-500 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-300 disabled:opacity-50"
                >
                  {provider.isDefault ? "Yes, unset" : `No, set as ${provider.environment} default`}
                </button>
              </DetailRow>
            </dl>
            <p className="border-t border-slate-100 px-5 py-3 text-[12px] leading-relaxed text-slate-500">
              Credential changes are written to the{" "}
              <Link href="/settings/audit-logs" className="text-brand-500 hover:underline">
                audit log
              </Link>{" "}
              with who, when and which environment.
            </p>
          </Card>

          <Card className="border-red-100">
            <div className="p-5">
              <h2 className="text-sm font-semibold text-slate-900">Disconnect {meta.name}</h2>
              <p className="mt-1 text-[12.5px] leading-relaxed text-slate-500">
                Stops routing payments through this connection and deletes its stored credentials.
              </p>
              <Button variant="danger" className="mt-4" onClick={() => setDisconnectOpen(true)} disabled={busy !== null}>
                <Unplug className="size-4" /> Disconnect
              </Button>
            </div>
          </Card>
        </aside>
      </div>

      {/* Rotate modal */}
      <Modal
        open={rotateOpen}
        onClose={() => {
          if (busy !== "rotate") {
            setRotateOpen(false);
            setNewSecretKey("");
            setNewAPIKey("");
            setNewContractCode("");
          }
        }}
        title="Rotate credentials"
        description={`Replace the stored ${meta.name} secret key with a new one. In-flight payments are unaffected.`}
        footer={
          <>
            <Button variant="secondary" onClick={() => setRotateOpen(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="provider-rotate-form"
              disabled={busy === "rotate" || !newSecretKey.trim() ||
                (connection.provider === "monnify" && (!newAPIKey.trim() || !newContractCode.trim()))}
            >
              <RotateCcw className="size-4" /> {busy === "rotate" ? "Validating…" : "Validate and rotate"}
            </Button>
          </>
        }
      >
        <form id="provider-rotate-form" onSubmit={rotateCredentials}>
          <p className="mb-3 text-[13px] leading-relaxed text-slate-600">
            We verify the new key with {meta.name} before it replaces the current one.
          </p>
          {connection.provider === "monnify" ? (
            <>
              <SecretInput
                value={newAPIKey}
                onChange={(event) => setNewAPIKey(event.target.value)}
                aria-label="Monnify API key"
                placeholder="New Monnify API key"
                required
                autoComplete="off"
                className="mb-3"
              />
              <Input
                value={newContractCode}
                onChange={(event) => setNewContractCode(event.target.value)}
                aria-label="Monnify contract code"
                placeholder="Monnify contract code"
                required
                autoComplete="off"
                className="mb-3"
              />
            </>
          ) : null}
          <SecretInput
            value={newSecretKey}
            onChange={(event) => setNewSecretKey(event.target.value)}
            aria-label={`${meta.name} new secret key`}
            placeholder="Enter new secret key"
            required
            autoComplete="new-password"
          />
        </form>
      </Modal>

      {/* Test transaction modal */}
      <Modal
        open={testOpen}
        onClose={() => {
          if (!testBusy) {
            setTestOpen(false);
            setTestError(null);
          }
        }}
        title="Make a test transaction"
        description={`Starts a sandbox charge on ${meta.name}. Complete it at the provider's checkout to watch the webhooks flow.`}
        footer={
          <>
            <Button variant="secondary" onClick={() => setTestOpen(false)} disabled={testBusy}>
              Cancel
            </Button>
            <Button type="submit" form="provider-test-form" disabled={testBusy}>
              <CreditCard className="size-4" /> {testBusy ? "Starting…" : "Start test payment"}
            </Button>
          </>
        }
      >
        <form id="provider-test-form" onSubmit={submitTest} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="test-amount">Amount</Label>
              <Input
                id="test-amount"
                type="number"
                min="1"
                step="0.01"
                value={testAmount}
                onChange={(event) => setTestAmount(event.target.value)}
                placeholder="100"
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="test-currency">Currency</Label>
              <Input
                id="test-currency"
                value={testCurrency}
                onChange={(event) => setTestCurrency(event.target.value.toUpperCase())}
                maxLength={3}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="test-email">Customer email</Label>
            <Input
              id="test-email"
              type="email"
              value={testEmail}
              onChange={(event) => setTestEmail(event.target.value)}
              placeholder="you@example.com"
              required
            />
            <p className="text-[12px] text-slate-400">The provider's sandbox checkout uses this to identify the payer.</p>
          </div>
          {testError ? (
            <p role="alert" className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-[12.5px] text-red-700">
              {testError}
            </p>
          ) : null}
          <SecurityNotice>
            This creates a real payment record in <b>test mode</b>. Completing it makes {meta.name} call PayConnect back,
            which then delivers the event to your webhook endpoint.
          </SecurityNotice>
        </form>
      </Modal>

      {/* Disconnect modal */}
      <Modal
        open={disconnectOpen}
        onClose={() => setDisconnectOpen(false)}
        title={`Disconnect ${meta.name}?`}
        description="This can't be undone."
        footer={
          <>
            <Button variant="secondary" onClick={() => setDisconnectOpen(false)}>
              Cancel
            </Button>
            <Button variant="dangerSolid" onClick={disconnect} disabled={busy === "delete"}>
              <Unplug className="size-4" /> {busy === "delete" ? "Disconnecting…" : "Disconnect"}
            </Button>
          </>
        }
      >
        <ul className="list-disc space-y-2 pl-5 text-[13px] text-slate-600">
          <li>Payments can no longer route through {meta.name}.</li>
          <li>Stored credentials are deleted from the vault.</li>
          <li>Historical payments and logs are kept.</li>
        </ul>
      </Modal>
    </div>
  );
}