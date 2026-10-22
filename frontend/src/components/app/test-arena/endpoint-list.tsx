"use client";

import * as React from "react";
import { Search, X, SearchX } from "lucide-react";
import { cn } from "@/lib/utils";
import { ENDPOINTS, ENDPOINT_GROUPS } from "./endpoint-definitions";
import type { EndpointDef } from "./endpoint-definitions";

interface Props {
  selected: EndpointDef | null;
  onSelect: (endpoint: EndpointDef) => void;
}

const METHOD_STYLES: Record<string, string> = {
  GET: "bg-sky-100 text-sky-700",
  POST: "bg-emerald-100 text-emerald-700",
  DELETE: "bg-red-100 text-red-700",
  PATCH: "bg-amber-100 text-amber-700",
};

export function EndpointList({ selected, onSelect }: Props) {
  const [query, setQuery] = React.useState("");

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return ENDPOINTS;
    return ENDPOINTS.filter((e) =>
      [e.label, e.path, e.method, e.group, e.description].some((s) => s.toLowerCase().includes(q))
    );
  }, [query]);

  return (
    <aside className="flex h-full min-h-0 flex-col bg-white">
      {/* Sticky header with search */}
      <div className="shrink-0 border-b border-slate-100 px-3 pb-3 pt-3">
        <div className="mb-2 flex items-center justify-between px-1">
          <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            Endpoints
          </p>
          <span className="text-[11px] text-slate-400">{filtered.length}</span>
        </div>
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search endpoints…"
            aria-label="Search endpoints"
            className="h-10 w-full rounded-lg border border-slate-200 bg-slate-50 pl-9 pr-9 text-base text-slate-800 transition placeholder:text-slate-400 hover:border-slate-300 focus:border-amber-400 focus:bg-white focus:outline-none focus:ring-2 focus:ring-amber-400/20 sm:h-9 sm:text-[13px] [&::-webkit-search-cancel-button]:hidden"
          />
          {query && (
            <button
              type="button"
              onClick={() => setQuery("")}
              aria-label="Clear search"
              className="absolute right-2 top-1/2 flex size-6 -translate-y-1/2 items-center justify-center rounded-md text-slate-400 hover:bg-slate-200/60 hover:text-slate-600"
            >
              <X className="size-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Scrollable list */}
      <nav className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-3" aria-label="Endpoints">
        {filtered.length === 0 && (
          <div className="flex flex-col items-center gap-2 px-4 py-10 text-center">
            <SearchX className="size-6 text-slate-300" />
            <p className="text-[13px] font-medium text-slate-500">No endpoints match “{query}”</p>
            <button
              type="button"
              onClick={() => setQuery("")}
              className="text-[12px] font-medium text-amber-600 hover:text-amber-700"
            >
              Clear search
            </button>
          </div>
        )}

        {ENDPOINT_GROUPS.map((group) => {
          const items = filtered.filter((e) => e.group === group);
          if (!items.length) return null;
          return (
            <div key={group}>
              <p className="mb-1 flex items-center gap-1.5 px-2 text-[10.5px] font-semibold uppercase tracking-wider text-slate-400">
                {group}
                <span className="rounded-full bg-slate-100 px-1.5 text-[9.5px] font-medium text-slate-400">
                  {items.length}
                </span>
              </p>
              <div className="flex flex-col gap-0.5">
                {items.map((endpoint) => {
                  const active = selected?.id === endpoint.id;
                  return (
                    <button
                      key={endpoint.id}
                      id={`arena-endpoint-${endpoint.id}`}
                      type="button"
                      aria-current={active ? "true" : undefined}
                      onClick={() => onSelect(endpoint)}
                      className={cn(
                        "flex w-full items-start gap-2.5 rounded-xl px-2.5 py-3 text-left transition-colors md:py-2",
                        active ? "bg-amber-50 ring-1 ring-amber-200" : "hover:bg-slate-50 active:bg-slate-100"
                      )}
                    >
                      <span
                        className={cn(
                          "mt-0.5 w-12 shrink-0 rounded-md py-0.5 text-center font-mono text-[9px] font-bold uppercase tracking-wide",
                          METHOD_STYLES[endpoint.method] ?? "bg-slate-100 text-slate-600"
                        )}
                      >
                        {endpoint.method}
                      </span>

                      <div className="min-w-0 flex-1">
                        <p
                          className={cn(
                            "truncate text-sm font-medium md:text-[13px]",
                            active ? "text-amber-900" : "text-slate-700"
                          )}
                        >
                          {endpoint.label}
                        </p>
                        <p className="mt-0.5 truncate font-mono text-[11px] text-slate-400 md:text-[10.5px]">
                          {endpoint.path}
                        </p>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          );
        })}
      </nav>
    </aside>
  );
}