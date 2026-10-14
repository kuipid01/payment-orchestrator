"use client";

import * as React from "react";
import { useMutation } from "@tanstack/react-query";
import {
  Play,
  ChevronDown,
  ChevronUp,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Clock,
  KeyRound,
  Eye,
  EyeOff,
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
}

/* ── helpers ── */
function buildDefaultBody(schema: FieldDef[] = []): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  schema.forEach((f) => {
    if (f.defaultValue !== undefined) out[f.key] = f.defaultValue;
    else if (f.type === "number") out[f.key] = "";
    else out[f.key] = "";
  });
  return out;
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
}: Props) {
  const [pathValues, setPathValues] = React.useState<Record<string, string>>({});
  const [bodyValues, setBodyValues] = React.useState<Record<string, unknown>>({});
  const [queryValues, setQueryValues] = React.useState<Record<string, string>>({});
  const [showSnippet, setShowSnippet] = React.useState(false);
  const [showKeyFull, setShowKeyFull] = React.useState(false);
  const [manualKeyOverride, setManualKeyOverride] = React.useState("");

  /* Reset fields when endpoint changes */
  React.useEffect(() => {
    if (!endpoint) return;
    setPathValues(
      Object.fromEntries((endpoint.pathParams ?? []).map((p) => [p.name, ""]))
    );
    setBodyValues(buildDefaultBody(endpoint.bodySchema));
    setQueryValues(
      Object.fromEntries((endpoint.querySchema ?? []).map((f) => [f.key, String(f.defaultValue ?? "")]))
    );
  }, [endpoint?.id]);

  const effectiveApiKey =
    manualKeyOverride.trim() ||
    revealedSecret ||
    (selectedKey?.environment === "test" ? selectedKey.masked : "");

  /* Check required fields to prevent invalid API calls */
  const missingBodyFields = (endpoint?.bodySchema ?? [])
    .filter((f) => f.required && (!bodyValues[f.key] || String(bodyValues[f.key]).trim() === ""))
    .map((f) => f.label);

  const missingPathParams = (endpoint?.pathParams ?? [])
    .filter((p) => !pathValues[p.name] || pathValues[p.name].trim() === "")
    .map((p) => p.name);

  const missingQueryParams = (endpoint?.querySchema ?? [])
    .filter((q) => q.required && (!queryValues[q.key] || queryValues[q.key].trim() === ""))
    .map((q) => q.label);

  const missingRequired = [...missingBodyFields, ...missingPathParams, ...missingQueryParams];
  const hasMissingRequired = missingRequired.length > 0;

  const isLiveKey = selectedKey?.environment === "live";
  const canRun = !!selectedKey && !isLiveKey && !!endpoint && !!effectiveApiKey && !hasMissingRequired;

  /* Build the actual URL shown in the pill */
  const currentPath = endpoint
    ? resolvedPath(endpoint.path, pathValues)
    : "";
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

      const params = Object.fromEntries(
        Object.entries(queryValues).filter(([, v]) => v !== "")
      );
      const hasBody = ["POST", "PATCH", "PUT"].includes(endpoint.method);

      // Strip empty strings so the server doesn't get blank optional fields
      const cleanBody: Record<string, unknown> = {};
      if (hasBody) {
        Object.entries(bodyValues).forEach(([k, v]) => {
          if (v !== "" && v !== undefined) cleanBody[k] = v;
        });

        // Ensure customer email compatibility
        if (cleanBody.email && !cleanBody.customer) {
          cleanBody.customer = { email: cleanBody.email };
        }
      }

      // Call our Next.js API proxy handler
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

  if (!endpoint) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mx-auto mb-3 flex size-12 items-center justify-center rounded-2xl bg-slate-100">
            <Play className="size-5 text-slate-400" />
          </div>
          <p className="text-sm font-medium text-slate-500">Select an endpoint to get started</p>
          <p className="mt-1 text-[12px] text-slate-400">Pick one from the list on the left</p>
        </div>
      </div>
    );
  }

  const bodySchema = endpoint.bodySchema ?? [];
  const querySchema = endpoint.querySchema ?? [];
  const pathParams = endpoint.pathParams ?? [];

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-4">
      {/* ── Endpoint pill (read-only) ── */}
      <div className="flex items-center gap-2 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2">
        <span
          className={cn(
            "shrink-0 rounded-md px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wide",
            {
              GET:    "bg-sky-100 text-sky-700",
              POST:   "bg-emerald-100 text-emerald-700",
              DELETE: "bg-red-100 text-red-700",
              PATCH:  "bg-amber-100 text-amber-700",
            }[endpoint.method] ?? "bg-slate-100 text-slate-600"
          )}
        >
          {endpoint.method}
        </span>
        <code className="flex-1 truncate text-[12.5px] text-slate-700">{fullUrl}</code>
        <span className="shrink-0 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-[10px] text-slate-400">
          read-only
        </span>
      </div>

      {/* Description */}
      <p className="text-[13px] text-slate-500">{endpoint.description}</p>

      {/* ── Path params ── */}
      {pathParams.length > 0 && (
        <Section title="Path Parameters">
          {pathParams.map((p) => (
            <FieldRow key={p.name} label={p.name} description={p.description} required>
              <Input
                id={`arena-path-${p.name}`}
                value={pathValues[p.name] ?? ""}
                onChange={(e) => setPathValues((prev) => ({ ...prev, [p.name]: e.target.value }))}
                placeholder={p.placeholder}
                className="font-mono text-[13px]"
              />
            </FieldRow>
          ))}
        </Section>
      )}

      {/* ── Query params ── */}
      {querySchema.length > 0 && (
        <Section title="Query Parameters">
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

      {/* ── Body fields ── */}
      {bodySchema.length > 0 && (
        <Section title="Request Body">
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

      {/* ── Active Test Key status ── */}
      {selectedKey && (
        <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200 bg-slate-50/70 p-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <KeyRound className="size-3.5 text-slate-400" />
              <span className="text-[12px] font-medium text-slate-600">Bearer Token</span>
            </div>
            {effectiveApiKey && (
              <button
                type="button"
                onClick={() => setShowKeyFull((v) => !v)}
                className="inline-flex items-center gap-1 text-[11px] font-medium text-amber-600 hover:text-amber-700"
              >
                {showKeyFull ? (
                  <>
                    <EyeOff className="size-3" /> Hide
                  </>
                ) : (
                  <>
                    <Eye className="size-3" /> Show Key
                  </>
                )}
              </button>
            )}
          </div>
          <code className="font-mono text-[12px] text-slate-700 break-all">
            {effectiveApiKey
              ? showKeyFull
                ? effectiveApiKey
                : `${effectiveApiKey.slice(0, 14)}••••••••${effectiveApiKey.slice(-4)}`
              : selectedKey.masked}
          </code>

          {/* Fallback override input if key was not automatically revealed */}
          {!revealedSecret && (
            <div className="mt-2 border-t border-slate-200/60 pt-2">
              <p className="mb-1 text-[11px] text-slate-500">
                Key secret not auto-revealed. Paste test secret key (or rotate in API Keys):
              </p>
              <Input
                type="password"
                placeholder="Paste vpay_test_... or pc_test_..."
                value={manualKeyOverride}
                onChange={(e) => setManualKeyOverride(e.target.value)}
                className="font-mono text-[12px]"
              />
            </div>
          )}
        </div>
      )}

      {/* ── No-key warning ── */}
      {!selectedKey && (
        <div className="flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-2.5 text-[13px] text-amber-800">
          <AlertTriangle className="size-4 shrink-0" />
          Select an application and test API key above to run this request.
        </div>
      )}

      {/* ── Required fields warning ── */}
      {selectedKey && hasMissingRequired && (
        <div className="flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-2.5 text-[12.5px] text-amber-800">
          <AlertTriangle className="size-4 shrink-0 text-amber-600" />
          <span>
            Required field missing: <strong>{missingRequired.join(", ")}</strong>. Please fill before running.
          </span>
        </div>
      )}

      {/* ── Run button ── */}
      <Button
        id="arena-run-btn"
        disabled={!canRun || run.isPending}
        onClick={() => run.mutate()}
        className="w-full gap-2 py-4"
      >
        {run.isPending ? (
          <>
            <Clock className="size-4 animate-spin" />
            Running…
          </>
        ) : hasMissingRequired ? (
          <>
            <AlertTriangle className="size-4" />
            Fill required: {missingRequired[0]}
          </>
        ) : (
          <>
            <Play className="size-4" />
            Run Test
          </>
        )}
      </Button>

      {/* ── Response ── */}
      {(run.isPending || run.data || run.error) && (
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-2">
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
              </>
            )}
          </div>

          {run.isPending && <Skeleton className="h-32 rounded-xl" />}

          {run.data && (
            <pre className="overflow-x-auto rounded-xl border border-slate-200 bg-slate-950 p-4 text-[12px] leading-relaxed text-slate-300">
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

      {/* ── Code snippet (collapsible) ── */}
      <div>
        <button
          id="arena-toggle-snippet"
          type="button"
          onClick={() => setShowSnippet((v) => !v)}
          className="flex w-full items-center justify-between rounded-xl border border-slate-200 bg-white px-3 py-2 text-[13px] font-medium text-slate-600 transition hover:bg-slate-50"
        >
          Code snippet
          {showSnippet ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
        </button>

        {showSnippet && (
          <div className="mt-2">
            <CodeSnippet
              endpoint={endpoint}
              resolvedPath={currentPath}
              body={bodyValues as Record<string, unknown>}
              query={queryValues}
              apiKey={effectiveApiKey || selectedKey?.masked || "vpay_test_..."}
            />
          </div>
        )}
      </div>
    </div>
  );
}

/* ── small layout helpers ── */

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-3 rounded-2xl border border-slate-100 bg-slate-50/60 p-4">
      <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">{title}</p>
      {children}
    </div>
  );
}

function FieldRow({
  label,
  description,
  required,
  children,
}: {
  label: string;
  description?: string;
  required?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-[13px] font-medium text-slate-700">
        {label}
        {required && <span className="ml-1 text-red-500">*</span>}
      </label>
      {description && <p className="text-[11px] text-slate-400">{description}</p>}
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

  return (
    <FieldRow label={field.label} description={field.description} required={field.required}>
      {field.type === "select" ? (
        <div className="relative">
          <select
            id={id}
            value={String(value ?? "")}
            onChange={(e) => onChange(e.target.value)}
            className="h-9 w-full appearance-none rounded-lg border border-slate-200 bg-white pl-3 pr-9 text-[13px] text-slate-800 transition hover:border-slate-300 focus:border-amber-400 focus:outline-none focus:ring-2 focus:ring-amber-400/20"
          >
            {(field.options ?? []).map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 size-3.5 -translate-y-1/2 text-slate-400" />
        </div>
      ) : (
        <Input
          id={id}
          type={field.type === "number" ? "number" : field.type === "email" ? "email" : "text"}
          value={String(value ?? "")}
          placeholder={field.placeholder}
          onChange={(e) =>
            onChange(field.type === "number" ? Number(e.target.value) : e.target.value)
          }
        />
      )}
    </FieldRow>
  );
}
