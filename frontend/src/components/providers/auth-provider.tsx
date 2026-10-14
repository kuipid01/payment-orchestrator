"use client";

import * as React from "react";
import { getSession, logoutAccount, type SessionUser } from "@/lib/api";

export type AuthStatus = "loading" | "authenticated" | "unauthenticated";

export interface AuthContextValue {
  user: SessionUser | null;
  organization: string | null;
  status: AuthStatus;
  /** Re-reads the session from the API (call after sign-in). */
  refresh: () => Promise<void>;
  /** Revokes the session server-side and clears local state. */
  signOut: () => Promise<void>;
}

const AuthContext = React.createContext<AuthContextValue | null>(null);

/**
 * Owns the dashboard's session state. It hydrates once on mount; the cookie-aware
 * API client transparently refreshes an expired access token before surfacing a
 * 401, so a valid refresh cookie keeps the user signed in across reloads.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = React.useState<SessionUser | null>(null);
  const [organization, setOrganization] = React.useState<string | null>(null);
  const [status, setStatus] = React.useState<AuthStatus>("loading");

  const refresh = React.useCallback(async () => {
    try {
      const session = await getSession();
      if (session.mode === "session" && session.user) {
        setUser(session.user);
        setOrganization(session.organization ?? null);
        setStatus("authenticated");
        return;
      }
      setUser(null);
      setOrganization(null);
      setStatus("unauthenticated");
    } catch {
      setUser(null);
      setOrganization(null);
      setStatus("unauthenticated");
    }
  }, []);

  React.useEffect(() => {
    void refresh();
  }, [refresh]);

  const signOut = React.useCallback(async () => {
    try {
      await logoutAccount();
    } catch {
      // The local session is cleared regardless; the cookie is dropped either way.
    } finally {
      setUser(null);
      setOrganization(null);
      setStatus("unauthenticated");
    }
  }, []);

  const value = React.useMemo<AuthContextValue>(
    () => ({ user, organization, status, refresh, signOut }),
    [user, organization, status, refresh, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** Reads the session context. Throws when used outside an AuthProvider. */
export function useAuth(): AuthContextValue {
  const context = React.useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within an AuthProvider");
  return context;
}
