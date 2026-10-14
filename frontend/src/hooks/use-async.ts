"use client";

import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { isAuthenticationError } from "@/components/providers/query-provider";

/**
 * Compatibility wrapper for existing pages. It keeps page code simple while
 * providing the same data/loading/error contract backed by TanStack Query.
 */
export function useAsync<T>(
  fn: () => Promise<T>,
  deps: React.DependencyList = [],
  queryKey?: readonly unknown[],
) {
  const query = useQuery({
    queryKey: queryKey ?? [fn.toString(), ...deps],
    queryFn: fn,
    retry: false,
    refetchOnWindowFocus: false,
  });

  React.useEffect(() => {
    if (!isAuthenticationError(query.error)) return;
    window.location.assign("/login");
  }, [query.error]);

  return {
    data: query.data,
    loading: query.isPending,
    error: query.error,
    refetch: query.refetch,
  };
}
