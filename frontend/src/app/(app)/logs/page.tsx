"use client";

import * as React from "react";
import { BookOpen } from "lucide-react";
import { Badge, Card, EmptyState, PageHeader, Select } from "@/components/ui";
import { useAsync } from "@/hooks/use-async";
import { listApiLogs, listApplications } from "@/lib/api";
import { formatTime, timeAgo } from "@/lib/utils";

function statusClass(code: number) {
  if (code < 300) return "text-emerald-600";
  if (code < 400) return "text-sky-600";
  if (code < 500) return "text-amber-600";
  return "text-red-600";
}

export default function LogsPage() {
  const { data, loading } = useAsync(listApiLogs, []);
  const apps = useAsync(listApplications, []);

  const [statusFilter, setStatusFilter] = React.useState("all");
  const [envFilter, setEnvFilter] = React.useState("all");
  const [appFilter, setAppFilter] = React.useState("all");

  const filtered = (data ?? []).filter((l) => {
    const bucket = l.statusCode < 300 ? "2xx" : l.statusCode < 400 ? "3xx" : l.statusCode < 500 ? "4xx" : "5xx";
    if (statusFilter !== "all" && bucket !== statusFilter) return false;
    if (envFilter !== "all" && l.environment !== envFilter) return false;
    if (appFilter !== "all" && l.applicationId !== appFilter) return false;
    return true;
  });

  return (
    <div className="animate-fade-in">
      <PageHeader
        title="Logs"
        description="Every API request made with your PayConnect keys — filterable by app, status and environment."
      />

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Select value={appFilter} onChange={(e) => setAppFilter(e.target.value)} className="w-48">
          <option value="all">All applications</option>
          {(apps.data ?? []).map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </Select>
        <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="w-36">
          <option value="all">All statuses</option>
          <option value="2xx">2xx success</option>
          <option value="3xx">3xx redirect</option>
          <option value="4xx">4xx client error</option>
          <option value="5xx">5xx server error</option>
        </Select>
        <Select value={envFilter} onChange={(e) => setEnvFilter(e.target.value)} className="w-36">
          <option value="all">All environments</option>
          <option value="live">Live</option>
          <option value="test">Test</option>
        </Select>
      </div>

      <Card>
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3.5">
          <h2 className="text-sm font-semibold text-slate-900">API requests</h2>
          <Badge variant="neutral">{filtered.length} events</Badge>
        </div>

        {loading ? (
          <div className="divide-y divide-slate-100">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 px-5 py-3">
                <div className="h-3.5 w-16 animate-pulse rounded bg-slate-100" />
                <div className="h-3.5 w-20 animate-pulse rounded bg-slate-100" />
                <div className="h-3.5 w-52 animate-pulse rounded bg-slate-100" />
                <div className="ml-auto h-3.5 w-10 animate-pulse rounded bg-slate-100" />
              </div>
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={<BookOpen className="size-5" />}
            title="No matching log entries"
            description="Try widening the filters — or make a request with one of your API keys."
          />
        ) : (
          <div className="divide-y divide-slate-100 font-mono text-[12.5px]">
            {filtered.map((l) => (
              <div key={l.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-3 hover:bg-slate-50/60">
                <span className="w-14 shrink-0 text-slate-400">{formatTime(l.at)}</span>
                <span className="w-12 shrink-0 font-semibold text-slate-700">{l.method}</span>
                <span className="min-w-0 flex-1 truncate text-slate-600">{l.path}</span>
                <span className="hidden text-slate-300 sm:inline">{l.sourceIp}</span>
                <span className="hidden w-14 text-right text-slate-400 md:inline">{l.durationMs}ms</span>
                <span className={`w-9 shrink-0 text-right font-semibold ${statusClass(l.statusCode)}`}>
                  {l.statusCode}
                </span>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
