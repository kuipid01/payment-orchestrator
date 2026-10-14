"use client";

import * as React from "react";
import Script from "next/script";

import { Button } from "@/components/ui";

interface GoogleCredentialResponse {
  credential: string;
}

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (options: {
            client_id: string;
            callback: (response: GoogleCredentialResponse) => void;
            auto_select?: boolean;
          }) => void;
          renderButton: (
            parent: HTMLElement,
            options: {
              theme: "outline";
              size: "large";
              text: "continue_with" | "signup_with";
              shape: "rectangular";
              width: number;
            },
          ) => void;
        };
      };
    };
  }
}

export interface GoogleAuthButtonProps {
  /** Receives the Google ID token; the caller exchanges it with the API. */
  onCredential: (credential: string) => void | Promise<void>;
  /** Called when the library itself fails, so the page can show why. */
  onError?: (message: string) => void;
  /**
   * Google's own wording on the rendered button. Sign-in and sign-up use the
   * same OAuth credential, so the difference is only what the button promises.
   */
  text?: "continue_with" | "signup_with";
  /** Shown instead of the button when no client ID is configured. */
  notConfiguredLabel?: string;
  /** Shown under the button while the credential is being exchanged. */
  pendingLabel?: string;
  pending?: boolean;
}

/**
 * Google Identity Services, rendered inside the page's own layout.
 *
 * Both sign-in and registration go through this one button: the API's
 * /auth/google creates the account when the Google identity is unknown and
 * signs in when it is not, so the flow is identical either way and only the
 * label differs.
 */
export function GoogleAuthButton({
  onCredential,
  onError,
  text = "continue_with",
  notConfiguredLabel = "Google sign-in is not configured",
  pendingLabel = "Completing secure sign-in…",
  pending = false,
}: GoogleAuthButtonProps) {
  const clientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;
  const [ready, setReady] = React.useState(false);
  const containerRef = React.useRef<HTMLDivElement>(null);

  // The callback is read through a ref so a re-render never re-initializes the
  // library: initialize() replaces the previous handler outright.
  const credentialHandler = React.useRef(onCredential);
  credentialHandler.current = onCredential;

  React.useEffect(() => {
    const target = containerRef.current;
    const googleID = window.google?.accounts.id;

    if (!ready || !clientId || !target || !googleID) return;

    target.replaceChildren();

    googleID.initialize({
      client_id: clientId,
      callback: (response) => {
        void credentialHandler.current(response.credential);
      },
    });

    googleID.renderButton(target, {
      theme: "outline",
      size: "large",
      text,
      shape: "rectangular",
      width: Math.min(target.clientWidth || 420, 420),
    });
  }, [ready, clientId, text]);

  if (!clientId) {
    return (
      <Button
        type="button"
        variant="secondary"
        size="lg"
        className="h-12 w-full rounded-xl"
        disabled
      >
        {notConfiguredLabel}
      </Button>
    );
  }

  return (
    <>
      <Script
        src="https://accounts.google.com/gsi/client"
        strategy="afterInteractive"
        onLoad={() => setReady(true)}
        onError={() =>
          onError?.("Google sign-in could not load. Please try again.")
        }
      />
      <div
        ref={containerRef}
        className="flex min-h-11 w-full justify-center overflow-hidden rounded-xl [&>div]:max-w-full"
      />
      {pending && (
        <p className="mt-2 text-center text-xs text-slate-500">{pendingLabel}</p>
      )}
    </>
  );
}

/**
 * The "or continue with" rule that separates a password form from the Google
 * button. Both auth screens show the same one.
 */
export function GoogleDivider({ label }: { label: string }) {
  return (
    <div className="my-6 flex items-center gap-3">
      <span className="h-px flex-1 bg-slate-200" />
      <span className="text-xs text-slate-400">{label}</span>
      <span className="h-px flex-1 bg-slate-200" />
    </div>
  );
}
