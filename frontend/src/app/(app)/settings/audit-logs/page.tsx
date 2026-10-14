"use client";

import * as React from "react";
import { Download, ScrollText } from "lucide-react";
import { Badge, Button, Select, Skeleton } from "@/components/ui";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import { exportAuditLogs, listAuditLogs } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import {
  SettingsEmptyState,
  SettingsSection,
} from "@/components/app/settings/section";

const SEVERITY: Record<string, { variant: "neutral" | "warning" | "danger"; label: string }> = {
  info: { variant: "neutral", label: "Info" },
  warning: { variant: "warning", label: "Warning" },
  critical: { variant: "danger", label: "Critical" },
};

export default function AuditLogsPage() {
  const [severity, setSeverity] = React.useState("all");
  const [exporting, setExporting] = React.useState(false);
  const { data, loading, error, refetch } = useAsync(
    () => listAuditLogs(severity),
    [severity],
    ["audit-logs", severity],
  );

  async function downloadCsv() {
    setExporting(true);
    try {
      const blob = await exportAuditLogs(severity);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = "audit-logs.csv";
      anchor.click();
      URL.revokeObjectURL(url);
    } catch (cause) {
      toast("error", "Export failed", cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setExporting(false);
    }
  }

  return (
    <SettingsSection
      icon={ScrollText}
      title="Security events"
      description="Credential changes, key operations, invitations and provider connections, retained for 12 months."
      aside={
        <>
          <Select
            value={severity}
            onChange={(e) => setSeverity(e.target.value)}
            aria-label="Filter by severity"
            className="w-36"
          >
            <option value="all">All severities</option>
            <option value="info">Info</option>
            <option value="warning">Warning</option>
            <option value="critical">Critical</option>
          </Select>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={downloadCsv}
            disabled={exporting}
          >
            <Download className="size-4" />
            {exporting ? "Exporting…" : "Export CSV"}
          </Button>
        </>
      }
      flush
    >
      {error ? (
        <div className="border-t border-slate-100 px-5 py-6 text-center sm:px-7">
          <p className="text-sm text-rose-700">
            {error instanceof Error ? error.message : "Could not load audit events."}
          </p>
          <Button type="button" variant="secondary" size="sm" className="mt-3" onClick={() => void refetch()}>
            Retry
          </Button>
        </div>
      ) : loading ? (
        <div className="divide-y divide-slate-100 border-t border-slate-100">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 px-5 py-4 sm:px-7">
              <Skeleton className="h-3.5 w-36" />
              <Skeleton className="h-3.5 w-52" />
              <Skeleton className="ml-auto h-5 w-16 rounded-full" />
            </div>
          ))}
        </div>
      ) : (data ?? []).length === 0 ? (
        <div className="px-5 pb-6 pt-5 sm:px-7">
          <SettingsEmptyState
            icon={ScrollText}
            title="No events at this severity"
            description="Try another severity, or check back once your team starts rotating credentials."
          />
        </div>
      ) : (
        <div className="divide-y divide-slate-100 border-t border-slate-100">
          {(data ?? []).map((l) => (
            <div
              key={l.id}
              className="flex flex-wrap items-start justify-between gap-3 px-5 py-4 sm:px-7"
            >
              <div className="flex min-w-0 items-start gap-3">
                <span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-500">
                  <ScrollText className="size-4" />
                </span>
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="text-[13.5px] font-medium text-slate-900">{l.action}</p>
                    <Badge variant={SEVERITY[l.severity].variant}>
                      {SEVERITY[l.severity].label}
                    </Badge>
                  </div>
                  <p className="mt-0.5 text-[12.5px] text-slate-500">{l.detail}</p>
                  <p className="mt-0.5 text-[12px] text-slate-400">
                    {l.actorName} · {l.actorRole}
                  </p>
                </div>
              </div>
              <time className="shrink-0 text-[12px] text-slate-400" dateTime={l.at}>
                {formatDateTime(l.at)}
              </time>
            </div>
          ))}
        </div>
      )}
    </SettingsSection>
  );
}
