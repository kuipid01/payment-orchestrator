"use client";

import * as React from "react";
import { Search, X, Layers, Filter } from "lucide-react";
import { cn } from "@/lib/utils";
import { ENDPOINTS, ENDPOINT_GROUPS } from "./endpoint-definitions";
import type { EndpointDef } from "./endpoint-definitions";

interface Props {
  selected: EndpointDef | null;
  onSelect: (endpoint: EndpointDef) => void;
}

const METHOD_STYLES: Record<string, string> = {
  GET:    "bg-sky-100 text-sky-700",
  POST:   "bg-emerald-100 text-emerald-700",
  DELETE: "bg-red-100 text-red-700",
  PATCH:  "bg-amber-100 text-amber-700",
};

/** Highlights matching terms in text */
function HighlightMatch({ text, query }: { text: string; query: string }) {
  if (!query.trim()) return <>{text}</>;
  const tokens = query.trim().split(/\s+/).filter(Boolean);
  if (!tokens.length) return <>{text}</>;

  const escaped = tokens.map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|");
  const regex = new RegExp(`(${escaped})`, "gi");
  const parts = text.split(regex);

  return (
    <>
      {parts.map((part, i) =>
        regex.test(part) ? (
          <mark key={i} className="rounded-xs bg-amber-200/90 px-0.5 font-medium text-amber-950">
            {part}
          </mark>
        ) : (
          part
        )
      )}
    </>
  );
}

/** Robust multi-field matching */
function matchEndpoint(endpoint: EndpointDef, query: string, methodFilter: string): boolean {
  if (methodFilter !== "ALL" && endpoint.method !== methodFilter) {
    return false;
  }

  const cleanQuery = query.trim().toLowerCase();
  if (!cleanQuery) return true;

  const tokens = cleanQuery.split(/\s+/).filter(Boolean);

  // Collect all searchable strings for this endpoint (including parameter keys and labels)
  const paramNames = [
    ...(endpoint.pathParams ?? []).map((p) => `${p.name} ${p.placeholder} ${p.description ?? ""}`),
    ...(endpoint.querySchema ?? []).map((q) => `${q.key} ${q.label} ${q.description ?? ""}`),
    ...(endpoint.bodySchema ?? []).map((b) => `${b.key} ${b.label} ${b.description ?? ""}`),
  ].join(" ").toLowerCase();

  const searchableText = [
    endpoint.label,
    endpoint.path,
    endpoint.method,
    endpoint.group,
    endpoint.description,
    paramNames,
  ].join(" ").toLowerCase();

  return tokens.every((token) => searchableText.includes(token));
}

export function EndpointList({ selected, onSelect }: Props) {
  const [searchQuery, setSearchQuery] = React.useState("");
  const [methodFilter, setMethodFilter] = React.useState<"ALL" | "GET" | "POST">("ALL");
  const searchInputRef = React.useRef<HTMLInputElement>(null);

  // Keyboard shortcut listener: Cmd+K or / to focus search, Esc to clear
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
      } else if (e.key === "/" && document.activeElement?.tagName !== "INPUT" && document.activeElement?.tagName !== "TEXTAREA") {
        e.preventDefault();
        searchInputRef.current?.focus();
      } else if (e.key === "Escape" && document.activeElement === searchInputRef.current) {
        setSearchQuery("");
        searchInputRef.current?.blur();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  // Filter endpoints
  const filteredEndpoints = React.useMemo(() => {
    return ENDPOINTS.filter((e) => matchEndpoint(e, searchQuery, methodFilter));
  }, [searchQuery, methodFilter]);

  // Method counts
  const counts = React.useMemo(() => {
    return {
      ALL: ENDPOINTS.length,
      GET: ENDPOINTS.filter((e) => e.method === "GET").length,
      POST: ENDPOINTS.filter((e) => e.method === "POST").length,
    };
  }, []);

  const hasFilterActive = searchQuery.trim() !== "" || methodFilter !== "ALL";

  // Arrow key navigation through filtered list
  const handleKeyDownNav = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (filteredEndpoints.length === 0) return;
    const currentIndex = selected ? filteredEndpoints.findIndex((ep) => ep.id === selected.id) : -1;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      const nextIndex = currentIndex < filteredEndpoints.length - 1 ? currentIndex + 1 : 0;
      onSelect(filteredEndpoints[nextIndex]);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      const prevIndex = currentIndex > 0 ? currentIndex - 1 : filteredEndpoints.length - 1;
      onSelect(filteredEndpoints[prevIndex]);
    } else if (e.key === "Enter" && filteredEndpoints.length > 0) {
      e.preventDefault();
      if (currentIndex === -1) {
        onSelect(filteredEndpoints[0]);
      }
    }
  };

  return (
    <aside className="flex h-full flex-col overflow-hidden border-r border-slate-100 bg-white">
      {/* ── Search and Filter Header ── */}
      <div className="flex flex-col gap-2.5 border-b border-slate-100 p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-slate-400" />
          <input
            ref={searchInputRef}
            id="arena-endpoint-search"
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={handleKeyDownNav}
            placeholder="Search endpoints, paths, fields…"
            className="h-8.5 w-full rounded-lg border border-slate-200 bg-slate-50/70 pl-8 pr-14 text-[12.5px] text-slate-800 placeholder:text-slate-400 transition hover:border-slate-300 focus:border-amber-400 focus:bg-white focus:outline-none focus:ring-2 focus:ring-amber-400/20"
          />
          {searchQuery ? (
            <button
              type="button"
              onClick={() => {
                setSearchQuery("");
                searchInputRef.current?.focus();
              }}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-slate-400 hover:bg-slate-200 hover:text-slate-600"
              aria-label="Clear search"
            >
              <X className="size-3.5" />
            </button>
          ) : (
            <kbd className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 rounded border border-slate-200 bg-white px-1.5 py-0.5 font-mono text-[9px] text-slate-400">
              ⌘K
            </kbd>
          )}
        </div>

        {/* Method filter pills */}
        <div className="flex items-center gap-1">
          {(["ALL", "GET", "POST"] as const).map((method) => {
            const active = methodFilter === method;
            return (
              <button
                key={method}
                type="button"
                onClick={() => setMethodFilter(method)}
                className={cn(
                  "flex items-center gap-1 rounded-md px-2 py-0.5 text-[11px] font-medium transition",
                  active
                    ? "bg-slate-900 text-white"
                    : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                )}
              >
                <span>{method}</span>
                <span className={cn("text-[10px]", active ? "text-slate-300" : "text-slate-400")}>
                  {counts[method]}
                </span>
              </button>
            );
          })}

          {hasFilterActive && (
            <span className="ml-auto text-[11px] text-slate-400">
              {filteredEndpoints.length} match{filteredEndpoints.length === 1 ? "" : "es"}
            </span>
          )}
        </div>
      </div>

      {/* ── Scrollable Endpoints List ── */}
      <nav className="flex-1 overflow-y-auto p-2.5">
        {filteredEndpoints.length === 0 ? (
          /* Empty Search State */
          <div className="flex flex-col items-center justify-center p-6 text-center">
            <div className="mb-2 flex size-9 items-center justify-center rounded-xl bg-slate-100 text-slate-400">
              <Search className="size-4" />
            </div>
            <p className="text-[13px] font-medium text-slate-700">No matching endpoints</p>
            <p className="mt-1 text-[11px] text-slate-400">
              No results found for &ldquo;{searchQuery}&rdquo;
            </p>
            <button
              type="button"
              onClick={() => {
                setSearchQuery("");
                setMethodFilter("ALL");
              }}
              className="mt-3 rounded-lg border border-slate-200 bg-white px-2.5 py-1 text-[11px] font-medium text-slate-600 hover:bg-slate-50"
            >
              Reset filters
            </button>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {ENDPOINT_GROUPS.map((group) => {
              const groupItems = filteredEndpoints.filter((e) => e.group === group);
              if (!groupItems.length) return null;

              return (
                <div key={group}>
                  <div className="mb-1 flex items-center justify-between px-2">
                    <p className="text-[10px] font-bold uppercase tracking-wider text-slate-400">
                      {group}
                    </p>
                    <span className="text-[10px] text-slate-400">{groupItems.length}</span>
                  </div>

                  <div className="flex flex-col gap-0.5">
                    {groupItems.map((endpoint) => {
                      const active = selected?.id === endpoint.id;
                      return (
                        <button
                          key={endpoint.id}
                          id={`arena-endpoint-${endpoint.id}`}
                          type="button"
                          onClick={() => onSelect(endpoint)}
                          className={cn(
                            "flex w-full items-start gap-2 rounded-xl px-2.5 py-2 text-left transition-colors",
                            active
                              ? "bg-amber-50 ring-1 ring-amber-200"
                              : "hover:bg-slate-50"
                          )}
                        >
                          {/* Method badge */}
                          <span
                            className={cn(
                              "mt-0.5 shrink-0 rounded px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wide",
                              METHOD_STYLES[endpoint.method] ?? "bg-slate-100 text-slate-600"
                            )}
                          >
                            {endpoint.method}
                          </span>

                          <div className="min-w-0 flex-1">
                            <p
                              className={cn(
                                "truncate text-[12.5px] font-medium",
                                active ? "text-amber-900" : "text-slate-700"
                              )}
                            >
                              <HighlightMatch text={endpoint.label} query={searchQuery} />
                            </p>
                            <p className="mt-0.5 truncate font-mono text-[10.5px] text-slate-400">
                              <HighlightMatch text={endpoint.path} query={searchQuery} />
                            </p>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </nav>
    </aside>
  );
}
