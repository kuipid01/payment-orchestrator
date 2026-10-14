"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";

const AUTH_ERROR_CODE = "authentication_required";

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = React.useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            refetchOnWindowFocus: false,
            retry: 1,
          },
          mutations: {
            retry: false,
          },
        },
      }),
  );

  React.useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key === "payconnect_session" && !event.newValue) {
        queryClient.clear();
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, [queryClient]);

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

export function isAuthenticationError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const apiError = error as {
    error?: { code?: string };
    response?: { data?: { error?: { code?: string } } };
  };
  return apiError.error?.code === AUTH_ERROR_CODE || apiError.response?.data?.error?.code === AUTH_ERROR_CODE;
}
