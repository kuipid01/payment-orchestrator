/**
 * PayConnect domain types.
 * Shared by UI + mock API facade; mirrors the Go backend models 1:1.
 */

import type { Environment, PaymentStatus } from "./utils";

/* --------------------------------- providers -------------------------------- */

export type ProviderKind = "paystack" | "flutterwave" | "moniepoint" | "stripe";

export const PROVIDER_META: Record<
  ProviderKind,
  { name: string; description: string; accent: string; initials: string }
> = {
  paystack: {
    name: "Paystack",
    description: "Nigeria & Ghana card, bank transfer, USSD and mobile money.",
    accent: "bg-sky-100 text-sky-700",
    initials: "PS",
  },
  flutterwave: {
    name: "Flutterwave",
    description: "Pan-African payments across 30+ countries and currencies.",
    accent: "bg-amber-100 text-amber-700",
    initials: "FW",
  },
  moniepoint: {
    name: "Moniepoint",
    description: "Accept payments through Moniepoint's merchant payment APIs.",
    accent: "bg-emerald-100 text-emerald-700",
    initials: "MP",
  },
  stripe: {
    name: "Stripe",
    description: "Global cards, wallets and local payment methods.",
    accent: "bg-violet-100 text-violet-700",
    initials: "ST",
  },
};

export type ProviderStatus = "connected" | "disconnected" | "error";

export interface ProviderConnection {
  id: string;
  provider: ProviderKind;
  status: ProviderStatus;
  environment: Environment;
  displayName: string;
  isDefault: boolean;
  /** Last 4 chars of the stored secret — never the secret itself. */
  secretLast4: string;
  lastVerifiedAt: string | null;
  connectedAt: string | null;
  successRate: number | null;
  volumeLast30dMinor: number | null;
  currency: string;
  /**
   * Inbound webhook configuration. `webhookUrl` is the URL the merchant pastes
   * into the provider dashboard; the rest describe what that provider requires,
   * so no component needs to special-case a provider by name.
   */
  webhookUrl: string;
  webhookSecretRequired: boolean;
  webhookSecretLabel: string;
  webhookSignatureHeader: string;
  webhookDashboardPath: string;
  webhookSecretMasked: string;
  webhookSecretSet: boolean;
  webhookLastEventAt: string | null;
}

/**
 * An inbound webhook received from a payment provider, after signature
 * verification. Distinct from WebhookDelivery, which is an outbound attempt to
 * the merchant's own endpoint.
 */
export interface ProviderEvent {
  id: string;
  connectionId: string;
  provider: ProviderKind;
  eventType: string;
  providerEventId: string;
  payload?: Record<string, unknown>;
  processedAt?: string | null;
  createdAt: string;
}

/* --------------------------------- payments --------------------------------- */

export interface PaymentCustomer {
  name: string;
  email: string;
  phone: string;
}

/**
 * Timeline entry types. The provider-driven ones (fallback, provider_timeout,
 * amount_mismatch) come straight from the payment state machine, so the union
 * mirrors the server rather than an idealised subset.
 */
export type PaymentEventType =
  | "created"
  | "provider_request"
  | "provider_confirmed"
  | "provider_event"
  | "provider_timeout"
  | "amount_mismatch"
  | "fallback"
  | "successful"
  | "failed"
  | "refunded"
  | "refund"
  | "pending";

export interface PaymentEvent {
  id: string;
  type: PaymentEventType;
  message: string;
  at: string;
}

export interface Payment {
  id: string;
  reference: string;
  providerReference: string | null;
  amountMinor: number;
  currency: string;
  /** The money environment the payment ran in — never mixed across a listing. */
  environment: Environment;
  status: PaymentStatus;
  provider: ProviderKind;
  /** Provider-hosted checkout page, when the provider returned one. */
  authorizationUrl: string;
  applicationId: string;
  /** The developer grant behind the key that took the payment; null = general. */
  developerAccessId: string | null;
  /** The API key that created the payment, for attribution; null when unknown. */
  apiKeyId: string | null;
  customer: PaymentCustomer;
  description: string;
  createdAt: string;
  updatedAt: string;
  timeline: PaymentEvent[];
  /** Redacted, credential-free provider response — safe to render. */
  providerResponse: Record<string, unknown>;
  feeMinor: number | null;
}

/* ------------------------------- applications ------------------------------- */

export interface Application {
  id: string;
  name: string;
  description: string;
  status: "active" | "paused";
  createdAt: string;
  lastUsedAt: string | null;
  stats: {
    apiRequests: number;
    successfulPayments: number;
    failedPayments: number;
    apiErrors: number;
    successRate: number;
  };
}

/* --------------------------------- api keys --------------------------------- */

/**
 * A key is either general (shared by everyone using the application) or
 * developer-scoped (belonging to one developer grant, revocable on its own).
 * Environment applies to either kind.
 */
export type ApiKeyScope = "general" | "developer";

export interface ApiKey {
  id: string;
  applicationId: string;
  environment: Environment;
  status: "active" | "revoked" | "expired";
  scope: ApiKeyScope;
  /** Set only for developer-scoped keys; names the developer grant. */
  developerAccessId: string | null;
  /**
   * True when the secret can be read again. Test keys stay copyable; live keys
   * are shown once, at creation.
   */
  revealable: boolean;
  label: string;
  masked: string;
  createdAt: string;
  lastUsedAt: string | null;
  createdByName: string;
}

/* -------------------------------- developers -------------------------------- */

export const DEVELOPER_PERMISSIONS = [
  { id: "payments.create", label: "Create payments" },
  { id: "payments.verify", label: "Verify payments" },
  { id: "payments.view", label: "View payments" },
  { id: "webhooks.receive", label: "Receive webhooks" },
  { id: "providers.manage", label: "Manage provider connections", dangerous: true },
  { id: "keys.manage", label: "Manage API keys", dangerous: true },
] as const;

export type DeveloperPermission = (typeof DEVELOPER_PERMISSIONS)[number]["id"];

export interface Developer {
  id: string;
  /** The developer grant. Developer-scoped keys are bound to it. */
  accessId: string;
  /** Empty until the invitation is accepted. */
  developerId: string;
  name: string;
  email: string;
  status: "active" | "pending" | "revoked";
  applicationIds: string[];
  permissions: DeveloperPermission[];
  invitedAt: string;
  lastActiveAt: string | null;
}

/* ---------------------------- developer platform --------------------------- */

/** A merchant grant held by the signed-in developer. */
export interface SelfAccess {
  access_id: string;
  merchant_id: string;
  merchant_name: string;
  status: "pending" | "active" | "revoked";
  permissions: string[];
  invited_at: string;
  accepted_at?: string;
  applications: SelfApplication[];
}

/** An application inside a merchant the developer was added to. */
export interface SelfApplication {
  id: string;
  name: string;
  description: string;
  status: string;
  keys: SelfKey[];
}

/** A masked developer-scoped key belonging to the signed-in developer. */
export interface SelfKey {
  id: string;
  name: string;
  environment: Environment;
  masked: string;
  status: string;
  created_at: string;
}

/* --------------------------------- webhooks --------------------------------- */

export interface WebhookEndpoint {
  id: string;
  /** Empty when the endpoint serves every application (the default). */
  applicationId: string;
  url: string;
  environment: Environment;
  status: "active" | "paused";
  events: string[];
  secretMasked: string;
  createdAt: string;
  /**
   * Delivery success over the reporting window, computed server-side from the
   * deliveries that concluded in it. Null when none did; the UI shows "—"
   * rather than a rate with nothing behind it.
   */
  successRate: number | null;
  /** Deliveries that concluded in the window — the denominator of successRate. */
  deliveries: number;
  /** How many of those reached the endpoint. */
  delivered: number;
  /** Length of the window successRate covers, in days. */
  deliveryWindowDays: number;
}

/**
 * Every event a receiving endpoint may subscribe to. These are the API's own
 * names: the server rejects an unknown type at creation, so a plausible-looking
 * name here would be a 422 rather than a subscription.
 */
export const WEBHOOK_EVENTS = [
  "payment.created",
  "payment.processing",
  "payment.succeeded",
  "payment.failed",
  "payment.cancelled",
  "refund.created",
  "refund.succeeded",
  "refund.failed",
  "chargeback.created",
  "chargeback.updated",
] as const;

export interface WebhookDelivery {
  id: string;
  endpointId: string;
  event: string;
  status: "delivered" | "failed" | "pending" | "retrying";
  attempts: number;
  responseCode: number | null;
  /**
   * What the endpoint replied on the most recent attempt, capped server-side.
   * Empty when no body was returned, or when the attempt never reached it.
   */
  responseBody: string;
  /** Server-side reason for the last failure, e.g. a refused connection. */
  lastError: string;
  createdAt: string;
  deliveredAt: string | null;
  payload: Record<string, unknown>;
}

/* ----------------------------------- logs ----------------------------------- */

export interface ApiLog {
  id: string;
  at: string;
  method: string;
  path: string;
  statusCode: number;
  durationMs: number;
  applicationId: string;
  environment: Environment | "";
  sourceIp: string;
}

export interface AuditLog {
  id: string;
  at: string;
  actorName: string;
  actorRole: string;
  action: string;
  detail: string;
  severity: "info" | "warning" | "critical";
}

/* --------------------------------- dashboard -------------------------------- */

export interface DashboardMetrics {
  volumeMinor: number;
  currency: string;
  successful: number;
  failed: number;
  successRate: number;
  volumeSeries: { date: string; volumeMinor: number }[];
}

export interface OrgMember {
  id: string;
  name: string;
  email: string;
  role: "Owner" | "Admin" | "Developer" | "Viewer";
  status: "active" | "invited";
  lastActiveAt: string | null;
}
