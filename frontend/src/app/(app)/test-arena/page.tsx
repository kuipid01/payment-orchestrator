"use client";

import * as React from "react";
import { AppKeySelector } from "@/components/app/test-arena/app-key-selector";
import { EndpointList } from "@/components/app/test-arena/endpoint-list";
import { RequestPanel } from "@/components/app/test-arena/request-panel";
import type { EndpointDef } from "@/components/app/test-arena/endpoint-definitions";
import type { ApiKey } from "@/lib/types";
import { cn } from "@/lib/utils";

export default function TestArenaPage() {
  const [selectedAppId, setSelectedAppId] = React.useState("");
  const [selectedKeyId, setSelectedKeyId] = React.useState("");
  const [selectedKey, setSelectedKey] = React.useState<ApiKey | null>(null);
  const [revealedSecret, setRevealedSecret] = React.useState("");
  const [selectedEndpoint, setSelectedEndpoint] = React.useState<EndpointDef | null>(null);

  /* Mobile shows one pane at a time: the endpoint list, then the request builder. */
  const [mobileView, setMobileView] = React.useState<"list" | "request">("list");

  const handleKeyChange = (id: string, key: ApiKey | null, secret?: string) => {
    setSelectedKeyId(id);
    setSelectedKey(key);
    if (secret !== undefined) {
      setRevealedSecret(secret);
    }
  };

  const handleSelectEndpoint = (endpoint: EndpointDef) => {
    setSelectedEndpoint(endpoint);
    setMobileView("request");
  };

  return (
    <div className="flex h-[calc(100dvh-4rem)] min-h-0 flex-col gap-3 overflow-hidden">
      {/* ── Header selector bar ── */}
      <div className="shrink-0">
        <AppKeySelector
          selectedAppId={selectedAppId}
          selectedKeyId={selectedKeyId}
          onAppChange={setSelectedAppId}
          onKeyChange={handleKeyChange}
        />
      </div>

      {/* ── Arena: two columns on md+, single swapped pane on mobile ── */}
      <div className="flex min-h-0 flex-1 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        {/* Left: endpoint list */}
        <div
          className={cn(
            "h-full min-h-0 w-full shrink-0 md:w-64 lg:w-72",
            mobileView === "list" ? "block" : "hidden md:block"
          )}
        >
          <EndpointList selected={selectedEndpoint} onSelect={handleSelectEndpoint} />
        </div>

        {/* Right: request builder + response */}
        <div
          className={cn(
            "h-full min-h-0 min-w-0 flex-1 md:border-l md:border-slate-100",
            mobileView === "request" ? "block" : "hidden md:block"
          )}
        >
          <RequestPanel
            endpoint={selectedEndpoint}
            selectedKey={selectedKey}
            revealedSecret={revealedSecret}
            onSecretChange={setRevealedSecret}
            onBack={() => setMobileView("list")}
          />
        </div>
      </div>
    </div>
  );
}