
"use client";

import * as React from "react";
import Link from "next/link";
import Image from "next/image";
import {
  Activity,
  ArrowRight,
  CheckCircle2,
  CircleAlert,
  Clock3,
  FlaskConical,
  Globe,
  LockKeyhole,
  Radio,
  RefreshCw,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  WalletCards,
  X,
} from "lucide-react";

import {
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
} from "@/components/ui";
import { EnvironmentBadge } from "@/components/ui/part3";
import { useAsync } from "@/hooks/use-async";
import { listProviders } from "@/lib/api";
import { timeAgo } from "@/lib/utils";
import {
  PROVIDER_META,
  type ProviderConnection,
} from "@/lib/types";

type Environment = "test" | "live";
type Filter = "all" | "connected" | "error" | "disconnected";

const PROVIDER_LOGOS: Partial<
  Record<ProviderConnection["provider"], string>
> = {
  paystack: "/providers/paystack.png",
  flutterwave: "/providers/flutterwave.png",
  moniepoint: "/providers/moniepoint.png",
};

function StatusPill({
  status,
}: {
  status: ProviderConnection["status"];
}) {
  const styles = {
    connected:
      "bg-emerald-50 text-emerald-700 ring-emerald-600/15",
    error: "bg-red-50 text-red-700 ring-red-600/15",
    disconnected:
      "bg-slate-100 text-slate-600 ring-slate-500/10",
  } as const;

  const labels = {
    connected: "Connected",
    error: "Connection error",
    disconnected: "Not connected",
  } as const;

  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold ring-1 ring-inset ${styles[status]}`}
    >
      {status === "connected" ? (
        <CheckCircle2 className="size-3.5" />
      ) : status === "error" ? (
        <CircleAlert className="size-3.5" />
      ) : (
        <span className="size-1.5 rounded-full bg-slate-400" />
      )}
      {labels[status]}
    </span>
  );
}

function EnvironmentPanel({
  environment,
  connection,
  provider,
}: {
  environment: Environment;
  connection?: ProviderConnection;
  provider: ProviderConnection["provider"];
}) {
  const connected = connection?.status === "connected";
  const errored = connection?.status === "error";
  const configured = Boolean(connection?.id);

  return (
    <div
      className={`flex min-w-0 flex-col rounded-xl border p-3.5 ${
        environment === "live"
          ? "border-indigo-100/90 bg-indigo-50/35"
          : "border-slate-200/80 bg-slate-50/70"
      }`}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          {environment === "test" ? (
            <FlaskConical className="size-4 shrink-0 text-slate-500" />
          ) : (
            <Radio className="size-4 shrink-0 text-indigo-600" />
          )}
          <span className="truncate text-[12px] font-semibold text-slate-800">
            {environment === "test" ? "Test" : "Live"}
          </span>
        </div>

        {environment === "live" && (
          <span className="text-[9px] font-bold uppercase tracking-wider text-indigo-600">
            Production
          </span>
        )}
      </div>

      <div className="mt-3">
        {connection ? (
          <StatusPill status={connection.status} />
        ) : (
          <StatusPill status="disconnected" />
        )}
      </div>

      <div className="mt-3 min-h-[46px] flex-1">
        {configured && connection ? (
          <>
            <p className="text-[11px] text-slate-500">
              {connected
                ? "Last verified"
                : errored
                  ? "Verification failed"
                  : "Awaiting verification"}
            </p>
            <p className="mt-1 text-xs font-medium text-slate-700">
              {connected
                ? connection.lastVerifiedAt
                  ? timeAgo(connection.lastVerifiedAt)
                  : "Recently"
                : errored
                  ? "Check your credentials"
                  : "Connection needs attention"}
            </p>
            <p className="mt-2 truncate font-mono text-[11px] text-slate-500">
              {connection.secretLast4
                ? `•••• ${connection.secretLast4}`
                : "Secret configured"}
              {connection.isDefault && (
                <span className="ml-1 font-sans text-emerald-700">
                  · Default
                </span>
              )}
            </p>
          </>
        ) : (
          <>
            <p className="text-[11px] leading-relaxed text-slate-500">
              {environment === "test"
                ? "Connect to test your integration safely."
                : "Connect to start accepting real payments."}
            </p>
            {environment === "live" && (
              <p className="mt-1 text-[10px] text-indigo-600">
                Real transactions
              </p>
            )}
          </>
        )}
      </div>

      <div className="mt-auto pt-4">
        <Link
          href={
            configured && connection
              ? `/providers/${connection.id}`
              : `/providers/connect/${provider}?environment=${environment}`
          }
          className="block min-w-0"
        >
          <Button
            variant={environment === "live" && !configured ? "primary" : "secondary"}
            size="sm"
            className="h-9 w-full min-w-0 justify-center gap-1.5 px-2 text-[11px] sm:text-xs"
          >
            <span className="truncate">
              {configured ? "Manage connection" : `Connect ${environment}`}
            </span>
            <ArrowRight className="size-3.5 shrink-0" />
          </Button>
        </Link>
      </div>
    </div>
  );
}

export default function ProvidersPage() {
  const { data, loading, error, refetch } = useAsync(
    listProviders,
    [],
    ["providers"],
  );

  const [search, setSearch] = React.useState("");
  const [filter, setFilter] = React.useState<Filter>("all");

  const connections = data ?? [];

  const allGroups = React.useMemo(() => {
    return Object.entries(
      connections.reduce<Record<string, ProviderConnection[]>>(
        (grouped, connection) => {
          (grouped[connection.provider] ??= []).push(connection);
          return grouped;
        },
        {},
      ),
    );
  }, [connections]);

  const connectedCount = allGroups.filter(([, items]) =>
    items.some((item) => item.status === "connected"),
  ).length;

  const errorCount = allGroups.filter(([, items]) =>
    items.some((item) => item.status === "error"),
  ).length;

  const providerGroups = allGroups.filter(([provider, items]) => {
    const meta =
      PROVIDER_META[provider as ProviderConnection["provider"]];

    const query = search.trim().toLowerCase();

    const matchesSearch =
      !query ||
      [
        provider,
        meta.name,
        meta.description,
        ...items.flatMap((item) => [
          item.environment,
          item.status,
          item.displayName,
          item.secretLast4,
        ]),
      ]
        .join(" ")
        .toLowerCase()
        .includes(query);

    const statuses = items.map((item) => item.status);

    const matchesFilter =
      filter === "all" ||
      (filter === "connected" &&
        statuses.includes("connected")) ||
      (filter === "error" && statuses.includes("error")) ||
      (filter === "disconnected" &&
        !items.some((item) => item.id));

    return matchesSearch && matchesFilter;
  });

  return (
    <div className="animate-fade-in space-y-6 pb-8">
      <PageHeader
        title="Providers"
        description="Connect and manage your payment integrations across test and live environments."
        actions={
          <Button
            variant="secondary"
            onClick={() => void refetch()}
            disabled={loading}
            className="gap-2"
          >
            <RefreshCw
              className={`size-4 ${loading ? "animate-spin" : ""}`}
            />
            <span className="hidden sm:inline">Refresh</span>
          </Button>
        }
      />

      {/* Overview */}
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Card className="border-slate-200/80 p-5 shadow-sm shadow-slate-900/[0.02]">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-sm text-slate-500">
                Connected providers
              </p>
              <p className="mt-2 text-3xl font-semibold tracking-tight text-slate-900">
                {connectedCount}
                <span className="ml-1.5 text-base font-normal text-slate-400">
                  / {allGroups.length}
                </span>
              </p>
            </div>
            <div className="rounded-xl bg-emerald-50 p-2.5 text-emerald-700">
              <WalletCards className="size-5" />
            </div>
          </div>
          <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-slate-100">
            <div
              className="h-full rounded-full bg-emerald-500 transition-all"
              style={{
                width: `${
                  allGroups.length
                    ? (connectedCount / allGroups.length) * 100
                    : 0
                }%`,
              }}
            />
          </div>
          <p className="mt-2 text-xs text-slate-500">
            Providers with at least one connected environment
          </p>
        </Card>

        <Card className="border-slate-200/80 p-5 shadow-sm shadow-slate-900/[0.02]">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-sm text-slate-500">
                Need attention
              </p>
              <p className="mt-2 text-3xl font-semibold tracking-tight text-slate-900">
                {errorCount}
              </p>
            </div>
            <div
              className={`rounded-xl p-2.5 ${
                errorCount
                  ? "bg-red-50 text-red-600"
                  : "bg-slate-100 text-slate-500"
              }`}
            >
              <Activity className="size-5" />
            </div>
          </div>
          <p className="mt-4 text-xs leading-relaxed text-slate-500">
            {errorCount
              ? "Some provider connections have reported errors."
              : "No provider connection errors reported."}
          </p>
        </Card>

        <Card className="border-slate-200/80 p-5 shadow-sm shadow-slate-900/[0.02]">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-sm text-slate-500">
                Environments
              </p>
              <p className="mt-2 text-3xl font-semibold tracking-tight text-slate-900">
                2
              </p>
            </div>
            <div className="rounded-xl bg-indigo-50 p-2.5 text-indigo-600">
              <Globe className="size-5" />
            </div>
          </div>
          <p className="mt-4 text-xs leading-relaxed text-slate-500">
            Test integrations safely before enabling live payments.
          </p>
        </Card>
      </section>

      {/* Search and filters */}
      <section className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full min-w-0 lg:max-w-sm">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
          <Input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search providers..."
            aria-label="Search provider connections"
            className="pl-9 pr-9"
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch("")}
              aria-label="Clear search"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-700"
            >
              <X className="size-4" />
            </button>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <SlidersHorizontal className="size-4 shrink-0 text-slate-400" />
          {(
            [
              ["all", "All providers"],
              ["connected", "Connected"],
              ["error", "Needs attention"],
              ["disconnected", "Not connected"],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => setFilter(value)}
              aria-pressed={filter === value}
              className={`shrink-0 rounded-lg border px-3 py-2 text-xs font-medium transition-colors ${
                filter === value
                  ? "border-slate-900 bg-slate-900 text-white"
                  : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </section>

      {/* Provider grid */}
      {loading ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }).map((_, index) => (
            <Card
              key={index}
              className="overflow-hidden border-slate-200/80 p-5"
            >
              <div className="flex animate-pulse items-center gap-3">
                <div className="size-12 rounded-xl bg-slate-100" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 w-28 rounded bg-slate-100" />
                  <div className="h-3 w-40 rounded bg-slate-100" />
                </div>
              </div>
              <div className="mt-5 grid animate-pulse grid-cols-1 gap-3 xl:grid-cols-2">
                <div className="h-36 rounded-xl bg-slate-50" />
                <div className="h-36 rounded-xl bg-slate-50" />
              </div>
            </Card>
          ))}
        </div>
      ) : error ? (
        <Card className="p-8 text-center">
          <div className="mx-auto flex size-12 items-center justify-center rounded-xl bg-red-50 text-red-600">
            <CircleAlert className="size-5" />
          </div>
          <h2 className="mt-4 font-semibold text-slate-900">
            Could not load providers
          </h2>
          <p className="mx-auto mt-2 max-w-md text-sm text-slate-500">
            {error instanceof Error
              ? error.message
              : "Something went wrong. Please try again."}
          </p>
          <Button
            className="mt-5"
            onClick={() => void refetch()}
          >
            Try again
          </Button>
        </Card>
      ) : connections.length === 0 ? (
        <Card className="p-6 sm:p-10">
          <EmptyState
            icon={<Globe className="size-5" />}
            title="Connect your first payment provider"
            description="Bring your payment accounts into one place. Start with a test connection before enabling live payments."
            action={
              <Link href="/providers/connect/paystack">
                <Button className="gap-2">
                  Connect a provider
                  <ArrowRight className="size-4" />
                </Button>
              </Link>
            }
          />
        </Card>
      ) : providerGroups.length === 0 ? (
        <Card className="p-8 text-center">
          <div className="mx-auto flex size-11 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
            <Search className="size-5" />
          </div>
          <h2 className="mt-4 font-semibold text-slate-900">
            No providers found
          </h2>
          <p className="mt-2 text-sm text-slate-500">
            Try a different search or change the selected filter.
          </p>
          <Button
            variant="secondary"
            className="mt-5"
            onClick={() => {
              setSearch("");
              setFilter("all");
            }}
          >
            Clear filters
          </Button>
        </Card>
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-sm text-slate-500">
              Showing{" "}
              <span className="font-semibold text-slate-800">
                {providerGroups.length}
              </span>{" "}
              {providerGroups.length === 1 ? "provider" : "providers"}
            </p>
            {filter !== "all" && (
              <button
                type="button"
                onClick={() => setFilter("all")}
                className="text-xs font-medium text-indigo-600 hover:text-indigo-700"
              >
                Clear filter
              </button>
            )}
          </div>

          <div className="grid items-stretch gap-4 md:grid-cols-2 xl:grid-cols-3">
            {providerGroups.map(([provider, items]) => {
              const kind =
                provider as ProviderConnection["provider"];
              const meta = PROVIDER_META[kind];

              const byEnvironment = new Map(
                items.map((item) => [item.environment, item]),
              );

              const hasConnected = items.some(
                (item) => item.status === "connected",
              );
              const hasError = items.some(
                (item) => item.status === "error",
              );

              return (
                <Card
                  key={kind}
                  className="flex min-w-0 flex-col overflow-hidden border-slate-200/80 p-4 shadow-sm shadow-slate-900/[0.02] transition-shadow hover:shadow-md hover:shadow-slate-900/[0.04] sm:p-5"
                >
                  <div className="flex items-start gap-3">
                    <div className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-slate-100 bg-white p-2">
                      {PROVIDER_LOGOS[kind] ? (
                        <Image
                          src={PROVIDER_LOGOS[kind]}
                          alt={`${meta.name} logo`}
                          width={48}
                          height={48}
                          className="size-full object-contain"
                        />
                      ) : (
                        <span className={`flex size-full items-center justify-center rounded-lg text-xs font-bold ${meta.accent}`}>
                          {meta.initials}
                        </span>
                      )}
                    </div>

                    <div className="min-w-0 flex-1 pt-0.5">
                      <h2 className="truncate text-[15px] font-semibold text-slate-900">
                        {meta.name}
                      </h2>
                      <p className="mt-1 line-clamp-2 text-xs leading-relaxed text-slate-500">
                        {meta.description}
                      </p>
                    </div>

                    {hasError ? (
                      <CircleAlert className="mt-1 size-4 shrink-0 text-red-500" />
                    ) : hasConnected ? (
                      <CheckCircle2 className="mt-1 size-4 shrink-0 text-emerald-600" />
                    ) : null}
                  </div>

                  <div className="mt-4 flex flex-wrap items-center gap-2">
                    {hasConnected ? (
                      <StatusPill status="connected" />
                    ) : hasError ? (
                      <StatusPill status="error" />
                    ) : (
                      <StatusPill status="disconnected" />
                    )}

                    <span className="text-[11px] text-slate-400">
                      {items.length
                        ? `${items.length} configured ${items.length === 1 ? "environment" : "environments"}`
                        : "No environments configured"}
                    </span>
                  </div>

                  <div className="mt-4 grid flex-1 grid-cols-1 gap-3 xl:grid-cols-[repeat(2,minmax(0,1fr))]">
                    <EnvironmentPanel
                      environment="test"
                      connection={byEnvironment.get("test")}
                      provider={kind}
                    />
                    <EnvironmentPanel
                      environment="live"
                      connection={byEnvironment.get("live")}
                      provider={kind}
                    />
                  </div>
                </Card>
              );
            })}
          </div>
        </>
      )}

      {/* Security reassurance */}
      <div className="flex flex-col gap-4 rounded-xl border border-indigo-100 bg-indigo-50/50 p-4 sm:flex-row sm:items-start sm:p-5">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-indigo-100 bg-white text-indigo-600">
          <LockKeyhole className="size-5" />
        </div>

        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-slate-900">
            Your payment credentials stay protected
          </h3>
          <p className="mt-1 max-w-3xl text-xs leading-relaxed text-slate-600">
            Secret keys are verified with the provider, encrypted using
            an organization-specific key, and stored in the vault. They
            are never shown in full, exported, or returned by the API.
            Developers only ever use PayConnect keys.
          </p>
          <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-[11px] font-medium text-slate-600">
            <span className="inline-flex items-center gap-1.5">
              <ShieldCheck className="size-3.5 text-emerald-600" />
              Encrypted at rest
            </span>
            <span className="inline-flex items-center gap-1.5">
              <LockKeyhole className="size-3.5 text-emerald-600" />
              Secrets masked
            </span>
            <span className="inline-flex items-center gap-1.5">
              <Clock3 className="size-3.5 text-emerald-600" />
              Connection verification
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}