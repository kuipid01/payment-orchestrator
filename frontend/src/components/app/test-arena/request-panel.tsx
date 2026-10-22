"use client";

import * as React from "react";
import { useMutation } from "@tanstack/react-query";
import {
  Play,
  ChevronDown,
  ChevronUp,
  ChevronLeft,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Clock,
  KeyRound,
  Eye,
  EyeOff,
  Copy,
  Check,
  Lock,
} from "lucide-react";
import axios from "axios";
import { API_BASE_URL } from "@/lib/api";
import { Button, Input, Skeleton } from "@/components/ui";
import { cn } from "@/lib/utils";
import { CodeSnippet } from "./code-snippet";
import type { EndpointDef, FieldDef } from "./endpoint-definitions";
import type { ApiKey } from "@/lib/types";

interface Props {
  endpoint: EndpointDef | null;
  selectedKey: ApiKey | null;
  revealedSecret?: string;
  onSecretChange?: (secret: string) => void;
  /** Mobile only: return to the endpoint list. */
  onBack?: () => void;
}

const METHOD_STYLES: Record<string, string> = {
  GET: "bg-sky-100 text-sky-700",
  POST: "bg-emerald-100 text-emerald-700",
  DELETE: "bg-red-100 text-red-700",
  PATCH: "bg-amber-100 text-amber-700",
};

/* ── helpers ── */
function buildDefaultBody(schema: FieldDef[] = []): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  schema.forEach((f) => {
    out[f.key] = f.defaultValue !== undefined ? f.defaultValue : "";
  });
  return out;
}

function isBlank(value: unknown): boolean {
  return value === undefined || value === null || String(value).trim() === "";
}

function isJsonValue(value: unknown): boolean {
  try {
    JSON.parse(String(value));
    return true;
  } catch {
    return false;
  }
}

function resolvedPath(path: string, pathValues: Record<string, string>): string {
  let result = path;
  Object.entries(pathValues).forEach(([k, v]) => {
    result = result.replace(`:${k}`, encodeURIComponent(v || `:${k}`));
  });
  return result;
}

function statusColor(code: number) {
  if (code >= 200 && code < 300) return "text-emerald-600 bg-emerald-50 border-emerald-200";
  if (code >= 400 && code < 500) return "text-amber-700 bg-amber-50 border-amber-200";
  return "text-red-600 bg-red-50 border-red-200";
}

function statusIcon(code: number) {
  if (code >= 200 && code < 300) return <CheckCircle2 className="size-4 text-emerald-500" />;
  if (code >= 400) return <XCircle className="size-4 text-red-500" />;
  return null;
}

/* ─────────────────────────── main component ─────────────────────────── */

export function RequestPanel({
  endpoint,
  selectedKey,
  revealedSecret = "",
  onSecretChange,
  onBack,
}: Props) {
  const [pathValues, setPathValues] = React.useState<Record<string, string>>({});
  const [bodyValues, setBodyValues] = React.useState<Record<string, unknown>>({});
  const [queryValues, setQueryValues] = React.useState<Record<string, string>>({});
  const [showSnippet, setShowSnippet] = React.useState(false);
  const [showKeyFull, setShowKeyFull] = React.useState(false);
  const [manualKeyOverride, setManualKeyOverride] = React.useState("");
  const [copied, setCopied] = React.useState(false);
  const responseRef = React.useRef<HTMLDivElement>(null);
  const scrollRef = React.useRef<HTMLDivElement>(null);

  /* Only a real secret may be sent — never the masked preview. */
  const effectiveApiKey =
    manualKeyOverride.trim() || (selectedKey?.environment === "test" ? revealedSecret : "");

  /* Required-field validation */
  const missingBodyFields = (endpoint?.bodySchema ?? [])
    .filter((f) => {
      const raw = bodyValues[f.key];
      const empty = isBlank(raw);
      if (f.type === "json") {
        // A non-empty JSON field must parse; a blank one is simply omitted.
        return empty ? Boolean(f.required) : !isJsonValue(raw);
      }
      return Boolean(f.required) && empty;
    })
    .map((f) => f.label);

  const missingPathParams = (endpoint?.pathParams ?? [])
    .filter((p) => isBlank(pathValues[p.name]))
    .map((p) => p.name);

  const missingQueryParams = (endpoint?.querySchema ?? [])
    .filter((q) => q.required && isBlank(queryValues[q.key]))
    .map((q) => q.label);

  const missingRequired = [...missingBodyFields, ...missingPathParams, ...missingQueryParams];
  const hasMissingRequired = missingRequired.length > 0;

  const isLiveKey = selectedKey?.environment === "live";
  const canRun = !!selectedKey && !isLiveKey && !!endpoint && !!effectiveApiKey && !hasMissingRequired;

  const currentPath = endpoint ? resolvedPath(endpoint.path, pathValues) : "";
  const fullUrl = `${API_BASE_URL}${currentPath}`;

  /* ── fire the test call via our Next.js API proxy ── */
  interface RunResult {
    status: number;
    durationMs: number;
    data: unknown;
  }

  const run = useMutation<RunResult, Error>({
    mutationFn: async () => {
      if (!endpoint || !selectedKey) throw new Error("No endpoint or key selected");
      if (!effectiveApiKey) throw new Error("No valid test key found. Please select or enter a test key.");
      if (hasMissingRequired) {
        throw new Error(`Required field missing: ${missingRequired.join(", ")}`);
      }

      const params = Object.fromEntries(Object.entries(queryValues).filter(([, v]) => v !== ""));
      const hasBody = ["POST", "PATCH", "PUT"].includes(endpoint.method);

      // Strip empty strings so the server doesn't get blank optional fields
      const cleanBody: Record<string, unknown> = {};
      if (hasBody) {
        Object.entries(bodyValues).forEach(([k, v]) => {
          if (v === "" || v === undefined) return;
          const field = (endpoint.bodySchema ?? []).find((f) => f.key === k);
          if (field?.type === "json") {
            // JSON fields are sent as parsed objects, not strings.
            cleanBody[k] = JSON.parse(String(v));
            return;
          }
          cleanBody[k] = v;
        });

        // Ensure customer email compatibility
        if (cleanBody.email && !cleanBody.customer) {
          cleanBody.customer = { email: cleanBody.email };
        }
      }

      const resp = await axios.post<{
        status: number;
        statusText: string;
        durationMs: number;
        data: unknown;
      }>("/api/test-arena/run", {
        url: fullUrl,
        method: endpoint.method,
        apiKey: effectiveApiKey,
        params: Object.keys(params).length ? params : undefined,
        data: hasBody ? cleanBody : undefined,
      });

      return {
        status: resp.data.status,
        durationMs: resp.data.durationMs,
        data: resp.data.data,
      };
    },
  });

  /* Reset form + previous response when the endpoint changes */
  React.useEffect(() => {
    if (!endpoint) return;
    setPathValues(Object.fromEntries((endpoint.pathParams ?? []).map((p) => [p.name, ""])));
    setBodyValues(buildDefaultBody(endpoint.bodySchema));
    setQueryValues(
      Object.fromEntries((endpoint.querySchema ?? []).map((f) => [f.key, String(f.defaultValue ?? "")]))
    );
    setShowSnippet(false);
    run.reset();
    scrollRef.current?.scrollTo({ top: 0 });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [endpoint?.id]);

  /* Bring the response into view once a run starts */
  React.useEffect(() => {
    if (run.isPending) {
      responseRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }, [run.isPending]);

  const copyResponse = async () => {
    if (!run.data) return;
    try {
      await navigator.clipboard.writeText(JSON.stringify(run.data.data, null, 2));
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable */
    }
  };

  if (!endpoint) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <div className="mx-auto mb-3 flex size-12 items-center justify-center rounded-2xl bg-slate-100">
            <Play className="size-5 text-slate-400" />
          </div>
          <p className="text-sm font-medium text-slate-500">Select an endpoint to get started</p>
          <p className="mt-1 text-[12px] text-slate-400">Pick one from the list</p>
        </div>
      </div>
    );
  }

  const bodySchema = endpoint.bodySchema ?? [];
  const querySchema = endpoint.querySchema ?? [];
  const pathParams = endpoint.pathParams ?? [];

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* ── Sticky header: back (mobile), title, method + URL ── */}
      <header className="shrink-0 border-b border-slate-100 bg-white px-3 py-3 sm:px-4">
        <div className="flex items-center gap-2">
          {onBack && (
            <button
              type="button"
              onClick={onBack}
              aria-label="Back to endpoints"
              className="-ml-1 flex size-9 shrink-0 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 active:bg-slate-200 md:hidden"
            >
              <ChevronLeft className="size-5" />
            </button>
          )}
          <div className="min-w-0 flex-1">
            <h2 className="truncate text-[15px] font-semibold text-slate-800">{endpoint.label}</h2>
            <p className="text-[12px] text-slate-500 line-clamp-2">{endpoint.description}</p>
          </div>
          {endpoint.requiresTestKey && (
            <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[10.5px] font-medium text-amber-700">
              <Lock className="size-3" />
              <span className="hidden sm:inline">Test key only</span>
              <span className="sm:hidden">Test</span>
            </span>
          )}
        </div>

        <div className="mt-2.5 flex items-start gap-2 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2">
          <span
            className={cn(
              "mt-px shrink-0 rounded-md px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wide",
              METHOD_STYLES[endpoint.method] ?? "bg-slate-100 text-slate-600"
            )}
          >
            {endpoint.method}
          </span>
          <code className="min-w-0 flex-1 break-all text-[12px] leading-5 text-slate-700 sm:text-[12.5px]">
            {fullUrl}
          </code>
        </div>
      </header>

      {/* ── Scrollable form + response ── */}
      <div ref={scrollRef} className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-3 sm:p-4">
        {/* Path params */}
        {pathParams.length > 0 && (
          <Section title="Path Parameters">
            {pathParams.map((p) => (
              <FieldRow key={p.name} label={p.name} description={p.description} required>
                <Input
                  id={`arena-path-${p.name}`}
                  value={pathValues[p.name] ?? ""}
                  onChange={(e) => setPathValues((prev) => ({ ...prev, [p.name]: e.target.value }))}
                  placeholder={p.placeholder}
                  autoCapitalize="none"
                  autoCorrect="off"
                  className="font-mono text-base sm:text-[13px]"
                />
              </FieldRow>
            ))}
          </Section>
        )}

        {/* Query params */}
        {querySchema.length > 0 && (
          <Section title="Query Parameters" grid>
            {querySchema.map((f) => (
              <FormField
                key={f.key}
                field={f}
                value={String(queryValues[f.key] ?? "")}
                onChange={(v) => setQueryValues((prev) => ({ ...prev, [f.key]: String(v) }))}
                idPrefix="arena-query"
              />
            ))}
          </Section>
        )}

        {/* Body fields */}
        {bodySchema.length > 0 && (
          <Section title="Request Body" grid>
            {bodySchema.map((f) => (
              <FormField
                key={f.key}
                field={f}
                value={bodyValues[f.key] ?? ""}
                onChange={(v) => setBodyValues((prev) => ({ ...prev, [f.key]: v }))}
                idPrefix="arena-body"
              />
            ))}
          </Section>
        )}

        {/* Key status */}
        {selectedKey && (
          <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200 bg-slate-50/70 p-3">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <KeyRound className="size-3.5 text-slate-400" />
                <span className="text-[12px] font-medium text-slate-600">Bearer Token</span>
                {effectiveApiKey && (
                  <span className="rounded-full bg-emerald-50 px-1.5 py-0.5 text-[10px] font-medium text-emerald-600">
                    ready
                  </span>
                )}
              </div>
              {effectiveApiKey && (
                <button
                  type="button"
                  onClick={() => setShowKeyFull((v) => !v)}
                  className="inline-flex h-8 items-center gap-1 px-1 text-[11px] font-medium text-amber-600 hover:text-amber-700"
                >
                  {showKeyFull ? (
                    <>
                      <EyeOff className="size-3" /> Hide
                    </>
                  ) : (
                    <>
                      <Eye className="size-3" /> Show
                    </>
                  )}
                </button>
              )}
            </div>
            <code className="break-all font-mono text-[12px] text-slate-700">
              {effectiveApiKey
                ? showKeyFull
                  ? effectiveApiKey
                  : `${effectiveApiKey.slice(0, 14)}••••••••${effectiveApiKey.slice(-4)}`
                : selectedKey.masked}
            </code>

            {!revealedSecret && !isLiveKey && (
              <div className="mt-2 border-t border-slate-200/60 pt-2">
                <p className="mb-1.5 text-[11px] text-slate-500">
                  Key secret not auto-revealed. Paste your test secret key (or rotate it in API Keys):
                </p>
                <Input
                  type="password"
                  placeholder="Paste vpay_test_... or pc_test_..."
                  value={manualKeyOverride}
                  onChange={(e) => setManualKeyOverride(e.target.value)}
                  autoComplete="off"
                  className="font-mono text-base sm:text-[12px]"
                />
              </div>
            )}
          </div>
        )}

        {/* Warnings */}
        {!selectedKey && (
          <Notice>Select an application and test API key above to run this request.</Notice>
        )}
        {selectedKey && isLiveKey && (
          <Notice tone="red">
            Live keys can’t be used in the Test Arena. Switch to a test key above.
          </Notice>
        )}

        {/* Response */}
        {(run.isPending || run.data || run.error) && (
          <div ref={responseRef} className="flex flex-col gap-3 scroll-mb-4">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
                Response
              </span>
              {run.data && (
                <>
                  <span
                    className={cn(
                      "flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[12px] font-medium",
                      statusColor(run.data.status)
                    )}
                  >
                    {statusIcon(run.data.status)}
                    {run.data.status}
                  </span>
                  <span className="text-[11px] text-slate-400">{run.data.durationMs}ms</span>
                  <button
                    type="button"
                    onClick={copyResponse}
                    className="ml-auto inline-flex h-8 items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 text-[12px] font-medium text-slate-600 hover:bg-slate-50 active:bg-slate-100"
                  >
                    {copied ? (
                      <>
                        <Check className="size-3.5 text-emerald-500" /> Copied
                      </>
                    ) : (
                      <>
                        <Copy className="size-3.5" /> Copy
                      </>
                    )}
                  </button>
                </>
              )}
            </div>

            {run.isPending && <Skeleton className="h-32 rounded-xl" />}

            {run.data && (
              <pre className="max-h-[60vh] overflow-auto rounded-xl border border-slate-200 bg-slate-950 p-3 text-[12px] leading-relaxed text-slate-300 sm:p-4">
                <code>{JSON.stringify(run.data.data, null, 2)}</code>
              </pre>
            )}

            {run.error && (
              <div className="rounded-xl border border-red-200 bg-red-50 p-3 text-[13px] text-red-700">
                {run.error.message}
              </div>
            )}
          </div>
        )}

        {/* Code snippet */}
        <div>
          <button
            id="arena-toggle-snippet"
            type="button"
            onClick={() => setShowSnippet((v) => !v)}
            aria-expanded={showSnippet}
            className="flex min-h-11 w-full items-center justify-between rounded-xl border border-slate-200 bg-white px-3 py-2 text-[13px] font-medium text-slate-600 transition hover:bg-slate-50 active:bg-slate-100"
          >
            Code snippet
            {showSnippet ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
          </button>

          {showSnippet && (
            <div className="mt-2 overflow-x-auto">
              <CodeSnippet
                endpoint={endpoint}
                resolvedPath={currentPath}
                body={bodyValues as Record<string, unknown>}
                query={queryValues}
                apiKey={effectiveApiKey || "vpay_test_..."}
              />
            </div>
          )}
        </div>
      </div>

      {/* ── Pinned run bar ── */}
      <div className="shrink-0 border-t border-slate-100 bg-white/95 p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] backdrop-blur sm:px-4">
        <Button
          id="arena-run-btn"
          disabled={!canRun || run.isPending}
          onClick={() => run.mutate()}
          className="min-h-11 w-full gap-2 py-3"
        >
          {run.isPending ? (
            <>
              <Clock className="size-4 animate-spin" />
              Running…
            </>
          ) : !selectedKey ? (
            <>
              <KeyRound className="size-4" />
              Select a test key
            </>
          ) : isLiveKey ? (
            <>
              <Lock className="size-4" />
              Live keys not allowed
            </>
          ) : !effectiveApiKey ? (
            <>
              <KeyRound className="size-4" />
              Paste your test secret
            </>
          ) : hasMissingRequired ? (
            <>
              <AlertTriangle className="size-4" />
              <span className="truncate">Fill required: {missingRequired[0]}</span>
              {missingRequired.length > 1 && (
                <span className="text-[11px] opacity-80">+{missingRequired.length - 1}</span>
              )}
            </>
          ) : (
            <>
              <Play className="size-4" />
              Run Test
            </>
          )}
        </Button>
      </div>
    </div>
  );
}

/* ── small layout helpers ── */

function Notice({
  children,
  tone = "amber",
}: {
  children: React.ReactNode;
  tone?: "amber" | "red";
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-2 rounded-xl border px-3 py-2.5 text-[13px]",
        tone === "amber"
          ? "border-amber-200 bg-amber-50 text-amber-800"
          : "border-red-200 bg-red-50 text-red-700"
      )}
    >
      <AlertTriangle className="mt-0.5 size-4 shrink-0" />
      <span>{children}</span>
    </div>
  );
}

function Section({
  title,
  children,
  grid,
}: {
  title: string;
  children: React.ReactNode;
  grid?: boolean;
}) {
  return (
    <div className="flex flex-col gap-3 rounded-2xl border border-slate-100 bg-slate-50/60 p-3 sm:p-4">
      <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">{title}</p>
      <div className={cn(grid ? "grid grid-cols-1 gap-3 sm:grid-cols-2" : "flex flex-col gap-3")}>
        {children}
      </div>
    </div>
  );
}

function FieldRow({
  label,
  description,
  required,
  className,
  htmlFor,
  children,
}: {
  label: string;
  description?: string;
  required?: boolean;
  className?: string;
  htmlFor?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      <label htmlFor={htmlFor} className="text-[13px] font-medium text-slate-700">
        {label}
        {required && <span className="ml-1 text-red-500">*</span>}
      </label>
      {description && <p className="text-[11px] leading-snug text-slate-400">{description}</p>}
      {children}
    </div>
  );
}

function FormField({
  field,
  value,
  onChange,
  idPrefix,
}: {
  field: FieldDef;
  value: unknown;
  onChange: (v: unknown) => void;
  idPrefix: string;
}) {
  const id = `${idPrefix}-${field.key}`;
  const wide =
    field.type === "json" ||
    field.type === "textarea" ||
    Boolean(field.description && field.description.length > 60) ||
    field.label.length > 32;

  return (
    <FieldRow
      label={field.label}
      description={field.description}
      required={field.required}
      htmlFor={id}
      className={wide ? "sm:col-span-2" : undefined}
    >
      {field.type === "select" ? (
        <div className="relative">
          <select
            id={id}
            value={String(value ?? "")}
            onChange={(e) => onChange(e.target.value)}
            className="h-11 w-full appearance-none rounded-lg border border-slate-200 bg-white pl-3 pr-9 text-base text-slate-800 transition hover:border-slate-300 focus:border-amber-400 focus:outline-none focus:ring-2 focus:ring-amber-400/20 sm:h-9 sm:text-[13px]"
          >
            {(field.options ?? []).map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 size-3.5 -translate-y-1/2 text-slate-400" />
        </div>
      ) : field.type === "json" || field.type === "textarea" ? (
        <textarea
          id={id}
          value={String(value ?? "")}
          placeholder={field.placeholder}
          rows={field.type === "json" ? 4 : 3}
          spellCheck={false}
          autoCapitalize="none"
          autoCorrect="off"
          onChange={(e) => onChange(e.target.value)}
          className={cn(
            "w-full resize-y rounded-lg border bg-white px-3 py-2 text-base text-slate-800 transition sm:text-[13px]",
            "hover:border-slate-300 focus:border-amber-400 focus:outline-none focus:ring-2 focus:ring-amber-400/20",
            field.type === "json" && "font-mono sm:text-[12px]",
            value && field.type === "json" && !isJsonValue(value) ? "border-red-300" : "border-slate-200"
          )}
        />
      ) : (
        <Input
          id={id}
          type={field.type === "number" ? "number" : field.type === "email" ? "email" : "text"}
          inputMode={field.type === "number" ? "numeric" : undefined}
          value={String(value ?? "")}
          placeholder={field.placeholder}
          className="text-base sm:text-[13px]"
          onChange={(e) =>
            onChange(
              field.type === "number"
                ? e.target.value === ""
                  ? ""
                  : Number(e.target.value)
                : e.target.value
            )
          }
        />
      )}
      {field.type === "json" && value && !isJsonValue(value) ? (
        <p className="text-[11px] text-red-500">Invalid JSON</p>
      ) : null}
    </FieldRow>
  );
}