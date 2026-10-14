"use client";

import { TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui";

export default function RouteError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center px-5 text-center">
      <div className="flex size-12 items-center justify-center rounded-2xl border border-red-200 bg-red-50">
        <TriangleAlert className="size-6 text-red-500" />
      </div>
      <h1 className="mt-5 text-lg font-semibold text-slate-900">Something went wrong</h1>
      <p className="mt-1.5 max-w-md text-sm text-slate-500">
        {error.message || "An unexpected error occurred while loading this page."}
      </p>
      <Button className="mt-6" onClick={reset}>
        Try again
      </Button>
    </div>
  );
}
