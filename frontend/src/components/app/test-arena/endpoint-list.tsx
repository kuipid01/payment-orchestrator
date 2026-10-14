"use client";

import * as React from "react";
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
  return (
    <aside className="flex h-full flex-col overflow-y-auto border-r border-slate-100 bg-white">
      <div className="border-b border-slate-100 px-4 py-3">
        <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
          Endpoints
        </p>
      </div>

      <nav className="flex flex-col gap-4 p-3">
        {ENDPOINT_GROUPS.map((group) => {
          const items = ENDPOINTS.filter((e) => e.group === group);
          if (!items.length) return null;
          return (
            <div key={group}>
              <p className="mb-1 px-2 text-[10.5px] font-semibold uppercase tracking-wider text-slate-400">
                {group}
              </p>
              <div className="flex flex-col gap-0.5">
                {items.map((endpoint) => {
                  const active = selected?.id === endpoint.id;
                  return (
                    <button
                      key={endpoint.id}
                      id={`arena-endpoint-${endpoint.id}`}
                      type="button"
                      onClick={() => onSelect(endpoint)}
                      className={cn(
                        "flex w-full items-start gap-2.5 rounded-xl px-2.5 py-2 text-left transition-colors",
                        active
                          ? "bg-amber-50 ring-1 ring-amber-200"
                          : "hover:bg-slate-50"
                      )}
                    >
                      {/* Method badge */}
                      <span
                        className={cn(
                          "mt-0.5 shrink-0 rounded-md px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wide",
                          METHOD_STYLES[endpoint.method] ?? "bg-slate-100 text-slate-600"
                        )}
                      >
                        {endpoint.method}
                      </span>

                      <div className="min-w-0">
                        <p
                          className={cn(
                            "truncate text-[13px] font-medium",
                            active ? "text-amber-900" : "text-slate-700"
                          )}
                        >
                          {endpoint.label}
                        </p>
                        <p className="mt-0.5 truncate font-mono text-[10.5px] text-slate-400">
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
