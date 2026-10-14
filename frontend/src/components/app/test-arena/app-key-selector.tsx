"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { FlaskConical, ChevronDown, AlertTriangle, KeyRound, CheckCircle2 } from "lucide-react";
import { listApplications, listApiKeys, revealApiKey } from "@/lib/api";
import { Skeleton } from "@/components/ui";
import { cn } from "@/lib/utils";
import type { Application, ApiKey } from "@/lib/types";

interface Props {
  selectedAppId: string;
  selectedKeyId: string;
  onAppChange: (appId: string) => void;
  onKeyChange: (keyId: string, key: ApiKey | null, revealedSecret?: string) => void;
}

export function AppKeySelector({ selectedAppId, selectedKeyId, onAppChange, onKeyChange }: Props) {
  /* ── fetch applications ── */
  const { data: apps = [], isLoading: appsLoading } = useQuery<Application[]>({
    queryKey: ["applications"],
    queryFn: listApplications,
  });

  /* ── fetch keys for the selected app, test-only ── */
  const { data: allKeys = [], isLoading: keysLoading } = useQuery<ApiKey[]>({
    queryKey: ["api-keys", selectedAppId],
    queryFn: () => listApiKeys(selectedAppId || undefined),
    enabled: !!selectedAppId,
  });

  const testKeys = allKeys.filter(
    (k) => k.environment === "test" && k.status === "active"
  );

  const selectedKey = testKeys.find((k) => k.id === selectedKeyId) ?? null;

  /* ── auto-reveal test secret for execution and snippets ── */
  const { data: revealedSecret, isLoading: isRevealingSecret } = useQuery<string>({
    queryKey: ["revealed-key", selectedKeyId],
    queryFn: () => revealApiKey(selectedKeyId),
    enabled: !!selectedKeyId && selectedKey?.environment === "test",
    staleTime: 1000 * 60 * 30, // 30 minutes cache
  });

  /* Notify parent when key or secret changes */
  React.useEffect(() => {
    onKeyChange(selectedKeyId, selectedKey, revealedSecret);
  }, [selectedKeyId, revealedSecret]);

  /* When app changes, clear key selection */
  const handleAppChange = (id: string) => {
    onAppChange(id);
    onKeyChange("", null, "");
  };

  /* When key changes in dropdown */
  const handleKeyChange = (id: string) => {
    const found = testKeys.find((k) => k.id === id) ?? null;
    onKeyChange(id, found, undefined);
  };

  const isLiveKey = selectedKey?.environment === "live";

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-2xl border border-slate-200 bg-white px-4 py-3 shadow-sm">
      {/* Title */}
      <div className="flex items-center gap-2 pr-3 border-r border-slate-100">
        <span className="flex size-7 items-center justify-center rounded-full bg-amber-100">
          <FlaskConical className="size-3.5 text-amber-600" />
        </span>
        <span className="text-sm font-semibold text-slate-800">Test Arena</span>
      </div>

      {/* Application selector */}
      <div className="flex flex-col gap-0.5 min-w-[180px]">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-400">
          Application
        </span>
        {appsLoading ? (
          <Skeleton className="h-8 w-44" />
        ) : (
          <SelectorDropdown
            id="arena-app-select"
            value={selectedAppId}
            onChange={handleAppChange}
            placeholder="Select application…"
            options={apps.map((a) => ({ value: a.id, label: a.name }))}
          />
        )}
      </div>

      {/* API Key selector */}
      <div className="flex flex-col gap-0.5 min-w-[200px]">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-400">
          Test API Key
        </span>
        {keysLoading && selectedAppId ? (
          <Skeleton className="h-8 w-48" />
        ) : (
          <SelectorDropdown
            id="arena-key-select"
            value={selectedKeyId}
            onChange={handleKeyChange}
            placeholder={selectedAppId ? "Select test key…" : "Select app first"}
            disabled={!selectedAppId}
            options={testKeys.map((k) => ({
              value: k.id,
              label: k.label ? `${k.label} (${k.masked})` : k.masked,
            }))}
            emptyLabel={
              selectedAppId && !keysLoading && testKeys.length === 0
                ? "No test keys found"
                : undefined
            }
          />
        )}
      </div>

      {/* Secret decryption status indicator */}
      {selectedKey && (
        <div className="flex items-center gap-2 self-end mb-0.5">
          {isRevealingSecret ? (
            <Skeleton className="h-6 w-28 rounded-full" />
          ) : revealedSecret ? (
            <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700">
              <CheckCircle2 className="size-3 text-emerald-500" />
              Key Ready
            </span>
          ) : (
            <span className="inline-flex items-center gap-1.5 rounded-full border border-slate-200 bg-slate-50 px-2.5 py-0.5 text-xs font-medium text-slate-600">
              <KeyRound className="size-3 text-slate-400" />
              {selectedKey.masked}
            </span>
          )}

          {/* Mode Badge */}
          <span
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium",
              selectedKey.environment === "test"
                ? "border-amber-200 bg-amber-50 text-amber-700"
                : "border-red-200 bg-red-50 text-red-700"
            )}
          >
            <span
              className={cn(
                "size-1.5 rounded-full",
                selectedKey.environment === "test" ? "bg-amber-400" : "bg-red-500"
              )}
            />
            {selectedKey.environment === "test" ? "Test mode" : "Live mode"}
          </span>
        </div>
      )}

      {/* Live key warning */}
      {isLiveKey && (
        <div className="flex w-full items-center gap-2 rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
          <AlertTriangle className="size-3.5 shrink-0" />
          Live keys are not allowed in Test Arena. Select a test key.
        </div>
      )}
    </div>
  );
}

/* ── tiny reusable dropdown ── */
interface DropdownProps {
  id: string;
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
  options: { value: string; label: string }[];
  disabled?: boolean;
  emptyLabel?: string;
}

function SelectorDropdown({
  id,
  value,
  onChange,
  placeholder,
  options,
  disabled,
  emptyLabel,
}: DropdownProps) {
  return (
    <div className="relative">
      <select
        id={id}
        value={value}
        disabled={disabled || (options.length === 0 && !emptyLabel)}
        onChange={(e) => onChange(e.target.value)}
        className={cn(
          "h-8 w-full appearance-none rounded-lg border border-slate-200 bg-white pl-2.5 pr-8 text-[13px] text-slate-800 transition-colors",
          "hover:border-slate-300 focus:border-amber-400 focus:outline-none focus:ring-2 focus:ring-amber-400/20",
          "disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-400"
        )}
      >
        <option value="">{emptyLabel ?? placeholder}</option>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
      <ChevronDown className="pointer-events-none absolute right-2 top-1/2 size-3.5 -translate-y-1/2 text-slate-400" />
    </div>
  );
}
