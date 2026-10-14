"use client";

import * as React from "react";
import { Download, ScrollText } from "lucide-react";
import { Badge, Button, Card, EmptyState, PageHeader, Select } from "@/components/ui";
import { useAsync } from "@/hooks/use-async";
import { listAuditLogs } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";

const SEVERITY: Record<string, { variant: "neutral" | "warning" | "danger"; label: string }> = {
  info: { variant: "neutral", label: "Info" },
  warning: { variant: "warning", label: "Warning" },
  critical: { variant: "danger", label: "Critical" },
};

export default function AuditLogsPage() {
  const { data, loading } = useAsync(listAuditLogs, []);
  const [severity, setSeverity] = React.useState("all");

  const filtered = (data ?? []).filter((l) => severity === "all" || l.severity === severity);

  return (
    <div className="animate-fade-in">
      <PageHeader
        title="Audit Logs"
        description="Every security-relevant event: credential changes, key operations, invitations and more."
        actions={
          <Button variant="secondary" size="sm">
            <Download className="size-4" /> Export CSV
          </Button>
        }
      />

      <div className="mb-4">
        <Select value={severity} onChange={(e) => setSeverity(e.target.value)} className="w-40">
          <option value="all">All severities</option>
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="critical">Critical</option>
        </Select>
      </div>

      <Card>
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3.5">
          <h2 className="text-sm font-semibold text-slate-900">Security events</h2>
          <span className="text-[12px] text-slate-400">Retained 12 months</span>
        </div>

        {loading ? (
          <div className="divide-y divide-slate-100">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 px-5 py-4">
                <div className="h-3.5 w-36 animate-pulse rounded bg-slate-100" />
                <div className="h-3.5 w-52 animate-pulse rounded bg-slate-100" />
                <div className="ml-auto h-5 w-16 animate-pulse rounded-full bg-slate-100" />
              </div>
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={<ScrollText className="size-5" />}
            title="No audit events"
            description="Credential and access events will appear here as your team works."
          />
        ) : (
          <div className="divide-y divide-slate-100">
            {filtered.map((l) => (
              <div key={l.id} className="flex flex-wrap items-start justify-between gap-3 px-5 py-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-[13.5px] font-medium text-slate-900">{l.action}</p>
                    <Badge variant={SEVERITY[l.severity].variant}>{SEVERITY[l.severity].label}</Badge>
                  </div>
                  <p className="mt-0.5 text-[12.5px] text-slate-500">{l.detail}</p>
                  <p className="mt-0.5 text-[12px] text-slate-400">
                    {l.actorName} · {l.actorRole}
                  </p>
                </div>
                <time className="shrink-0 text-[12px] text-slate-400" dateTime={l.at}>
                  {formatDateTime(l.at)}
                </time>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
