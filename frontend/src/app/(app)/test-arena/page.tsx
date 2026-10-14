"use client";

import * as React from "react";
import { AppKeySelector } from "@/components/app/test-arena/app-key-selector";
import { EndpointList } from "@/components/app/test-arena/endpoint-list";
import { RequestPanel } from "@/components/app/test-arena/request-panel";
import type { EndpointDef } from "@/components/app/test-arena/endpoint-definitions";
import type { ApiKey } from "@/lib/types";

export default function TestArenaPage() {
  const [selectedAppId, setSelectedAppId] = React.useState("");
  const [selectedKeyId, setSelectedKeyId] = React.useState("");
  const [selectedKey, setSelectedKey] = React.useState<ApiKey | null>(null);
  const [revealedSecret, setRevealedSecret] = React.useState("");
  const [selectedEndpoint, setSelectedEndpoint] = React.useState<EndpointDef | null>(null);

  const handleKeyChange = (id: string, key: ApiKey | null, secret?: string) => {
    setSelectedKeyId(id);
    setSelectedKey(key);
    if (secret !== undefined) {
      setRevealedSecret(secret);
    }
  };

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col gap-3 overflow-hidden">
      {/* ── Header selector bar ── */}
      <AppKeySelector
        selectedAppId={selectedAppId}
        selectedKeyId={selectedKeyId}
        onAppChange={setSelectedAppId}
        onKeyChange={handleKeyChange}
      />

      {/* ── Two-column arena ── */}
      <div className="flex min-h-0 flex-1 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        {/* Left: endpoint list */}
        <div className="w-64 shrink-0">
          <EndpointList
            selected={selectedEndpoint}
            onSelect={setSelectedEndpoint}
          />
        </div>

        {/* Right: request builder + response */}
        <div className="flex-1 overflow-y-auto border-l border-slate-100">
          <RequestPanel
            endpoint={selectedEndpoint}
            selectedKey={selectedKey}
            revealedSecret={revealedSecret}
            onSecretChange={setRevealedSecret}
          />
        </div>
      </div>
    </div>
  );
}
