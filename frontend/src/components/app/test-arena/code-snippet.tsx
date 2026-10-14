"use client";

import * as React from "react";
import { Copy, Check } from "lucide-react";
import { API_BASE_URL } from "@/lib/api";
import { cn } from "@/lib/utils";
import type { EndpointDef } from "./endpoint-definitions";

interface Props {
  endpoint: EndpointDef;
  resolvedPath: string; // path with :param replaced
  body: Record<string, unknown>;
  query: Record<string, string>;
  apiKey: string; // masked or actual key label
}

type SnippetTab = "nextjs" | "curl";

export function CodeSnippet({ endpoint, resolvedPath, body, query, apiKey }: Props) {
  const [tab, setTab] = React.useState<SnippetTab>("nextjs");
  const [copied, setCopied] = React.useState(false);

  const baseUrl = API_BASE_URL;
  const queryString = Object.entries(query)
    .filter(([, v]) => v)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join("&");
  const fullUrl = `${baseUrl}${resolvedPath}${queryString ? `?${queryString}` : ""}`;

  const hasBody = ["POST", "PATCH", "PUT"].includes(endpoint.method);
  const bodyJson = hasBody ? JSON.stringify(body, null, 2) : null;

  /* ── Next.js snippet ── */
  const nextjsSnippet = `// ${endpoint.label}
// Install: npm install axios  (or use fetch)

const response = await fetch("${fullUrl}", {
  method: "${endpoint.method}",
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer ${apiKey || "vpay_test_YOUR_KEY"}",
  },${
    bodyJson
      ? `
  body: JSON.stringify(${bodyJson}),`
      : ""
  }
});

const data = await response.json();
console.log(data);`;

  /* ── cURL snippet ── */
  const curlSnippet = [
    `curl -X ${endpoint.method} "${fullUrl}" \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -H "Authorization: Bearer ${apiKey || "vpay_test_YOUR_KEY"}"${bodyJson ? ` \\` : ""}`,
    bodyJson ? `  -d '${JSON.stringify(body)}'` : null,
  ]
    .filter(Boolean)
    .join("\n");

  const snippet = tab === "nextjs" ? nextjsSnippet : curlSnippet;

  const handleCopy = () => {
    navigator.clipboard.writeText(snippet).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="rounded-2xl border border-slate-200 bg-slate-950 overflow-hidden">
      {/* Tab bar */}
      <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2">
        <div className="flex gap-1">
          {(["nextjs", "curl"] as SnippetTab[]).map((t) => (
            <button
              key={t}
              id={`arena-snippet-tab-${t}`}
              type="button"
              onClick={() => setTab(t)}
              className={cn(
                "rounded-md px-3 py-1 text-[12px] font-medium transition-colors",
                tab === t
                  ? "bg-slate-800 text-white"
                  : "text-slate-500 hover:text-slate-300"
              )}
            >
              {t === "nextjs" ? "Next.js" : "cURL"}
            </button>
          ))}
        </div>

        <button
          id="arena-copy-snippet"
          type="button"
          onClick={handleCopy}
          className="flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[12px] font-medium text-slate-400 transition-colors hover:bg-slate-800 hover:text-white"
        >
          {copied ? (
            <Check className="size-3.5 text-emerald-400" />
          ) : (
            <Copy className="size-3.5" />
          )}
          {copied ? "Copied!" : "Copy"}
        </button>
      </div>

      {/* Code body */}
      <pre className="overflow-x-auto p-4 text-[12.5px] leading-relaxed text-slate-300">
        <code>{snippet}</code>
      </pre>
    </div>
  );
}
