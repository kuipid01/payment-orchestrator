/**
 * Data-access facade. Every page talks to the backend through one cookie-aware
 * Axios client. The client refreshes a session once per concurrent 401 and
 * retries the original request exactly once.
 */

import axios, { AxiosError, AxiosInstance, AxiosRequestConfig } from "axios";
import { DEVELOPER_PERMISSIONS } from "./types";
import type { Environment } from "./utils";
import type {
  BillingInterval,
  BillingUsage,
  PlanCatalog,
  PlanCode,
  SubscriptionEvent,
  SubscriptionPlan,
} from "./plans";
import type {
  ApiKey,
  ApiKeyScope,
  ApiLog,
  Application,
  AuditLog,
  DashboardMetrics,
  Developer,
  DeveloperPermission,
  OrgMember,
  Payment,
  PaymentEventType,
  ProviderConnection,
  ProviderEvent,
  SelfAccess,
  WebhookDelivery,
  WebhookEndpoint,
} from "./types";

const LATENCY = 450;

/**
 * The API root. Exported because the marketing pages fetch the plan catalog
 * server-side, and a second copy of this string is one more thing that can
 * point somewhere else.
 */
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/v1";

export interface SessionUser {
  id: string;
  name: string;
  email: string;
  role: string;
  status: string;
  merchant_id?: string;
  email_verified?: boolean;
}

export interface AuthSession {
  user: SessionUser;
}

export interface AuthSessionRecord {
  id: string;
  created_at: string;
  expires_at: string;
  current: boolean;
}

export interface MerchantProfile {
  id: string;
  business_name: string;
  country: string;
  default_currency: string;
  created_at: string;
  updated_at: string;
}

/** The signed-in identity returned by GET /auth/me. */
export interface Session {
  mode: "session" | "apikey" | string;
  merchant_id?: string;
  user_id?: string;
  environment?: string;
  organization?: string;
  user?: SessionUser;
}

/**
 * One organization the signed-in user can act in. A session acts in exactly one
 * at a time; `current` marks which. A user can own one organization and be a
 * Developer in several others, so the role travels per entry.
 */
export interface OrganizationMembership {
  merchant_id: string;
  name: string;
  role: string;
  status: string;
  current: boolean;
  owner: boolean;
  joined_at: string;
}

export interface RegistrationInput {
  name: string;
  email: string;
  password: string;
  organizationName: string;
}

export interface LoginInput {
  email: string;
  password: string;
}

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  headers: { "Content-Type": "application/json" },
});

let refreshPromise: Promise<void> | null = null;

async function refreshSession(): Promise<void> {
  if (!refreshPromise) {
    refreshPromise = apiClient.post("/auth/refresh", {}).then(() => undefined).catch((error) => {
      throw error;
    }).finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

/**
 * Endpoints that answer 401 as a verdict on the credentials just submitted,
 * rather than as "your session expired". Refreshing on these replaced the real
 * message with a reload, which is how a wrong password showed nothing at all.
 */
const CREDENTIAL_ENDPOINTS = [
  "/auth/login",
  "/auth/register",
  "/auth/google",
  "/auth/verify-email",
  "/auth/resend-verification",
  "/auth/forgot-password",
  "/auth/reset-password",
];

function isCredentialRequest(url?: string): boolean {
  return !!url && CREDENTIAL_ENDPOINTS.some((endpoint) => url.startsWith(endpoint));
}

/** The backend's error entry, mirroring apihttp.Responder.Error. */
interface ApiErrorPayload {
  type?: string;
  code?: string;
  message?: string;
  details?: unknown;
}

/**
 * Rebuilds a rejected request as an Error carrying the message the API chose.
 * Axios rejects a non-2xx before the caller ever sees the body, so without this
 * every failure reached the UI as "Request failed with status code 502" and the
 * reason a merchant can act on — a provider being unreachable, a reference that
 * was already used — was thrown away. A request that never got a response is
 * reported as a connectivity problem rather than a bare "Network Error".
 */
function apiError(error: AxiosError): Error {
  const payload = (error.response?.data as { error?: ApiErrorPayload } | undefined)?.error;
  if (payload?.message) {
    return Object.assign(new Error(payload.message), {
      error: payload,
      response: error.response,
    });
  }
  if (!error.response) {
    return Object.assign(
      new Error("Could not reach the PayConnect API. Check your connection and try again."),
      { cause: error },
    );
  }
  return error;
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config;
    const isRefreshRequest = originalRequest?.url === "/auth/refresh";
    // The session probe must never trigger a hard redirect: on public pages
    // there is simply no session, and redirecting would reload the page in a
    // loop. The AuthProvider handles the rejection instead.
    const isSessionProbe = originalRequest?.url === "/auth/me";
    const shouldRefresh =
      originalRequest &&
      error.response?.status === 401 &&
      !isRefreshRequest &&
      !isCredentialRequest(originalRequest.url);

    if (!shouldRefresh) return Promise.reject(apiError(error));
    try {
      await refreshSession();
      return apiClient.request({
        ...originalRequest,
        headers: { ...originalRequest.headers, Authorization: undefined },
      });
    } catch (refreshError) {
      if (!isSessionProbe) window.location.assign("/login");
      return Promise.reject(apiError(refreshError as AxiosError));
    }
  },
);

async function request<T>(path: string, init: AxiosRequestConfig = {}, unwrap = true): Promise<T> {
  const response = await apiClient.request({ url: path, method: "GET", ...init });
  const result = response.data as { success?: boolean; data?: T; error?: { message?: string } };
  if (result.error) {
    throw Object.assign(new Error(result.error.message ?? "The request could not be completed."), {
      error: result.error,
      response: { data: result },
    });
  }
  return unwrap ? (result.data as T) : (result as T);
}

/** The pagination block the backend returns alongside a list envelope. */
interface EnvelopePage {
  page?: number;
  limit?: number;
  total?: number;
  has_more?: boolean;
  summary?: Record<string, number>;
}

/**
 * Reads a paginated list without discarding the page metadata: list screens need
 * the total and the facet summary, which the unwrapping `request` drops.
 */
async function requestPage<T>(path: string): Promise<{ items: T[]; page: EnvelopePage }> {
  const body = await request<{ data?: T[]; pagination?: EnvelopePage }>(path, {}, false);
  return { items: body.data ?? [], page: body.pagination ?? {} };
}

async function authRequest<T>(path: string, body: unknown): Promise<T> {
  return request<T>(path, { method: "POST", data: body });
}

export function registerAccount(input: RegistrationInput): Promise<{ verificationRequired: boolean }> {
  return authRequest("/auth/register", input);
}

export function loginAccount(input: LoginInput): Promise<AuthSession> {
  return authRequest("/auth/login", input);
}

/** Hydrates the dashboard with the current session (or API-key) principal. */
export function getSession(): Promise<Session> {
  return request<Session>("/auth/me");
}

export async function updateProfile(name: string): Promise<void> {
  await request("/auth/profile", { method: "PATCH", data: { name } });
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  await request("/auth/password", {
    method: "POST",
    data: { current_password: currentPassword, new_password: newPassword },
  });
}

export function listSessions(): Promise<AuthSessionRecord[]> {
  return request<AuthSessionRecord[]>("/auth/sessions");
}

export async function revokeSession(sessionId: string): Promise<void> {
  await request(`/auth/sessions/${encodeURIComponent(sessionId)}`, { method: "DELETE" });
}

/** Revokes the current session and clears the session cookies. */
export async function logoutAccount(): Promise<void> {
  await request<{ revoked: boolean }>("/auth/logout", { method: "POST", data: {} });
}

/** Lists every organization the signed-in user belongs to. */
export function listOrganizationMemberships(): Promise<OrganizationMembership[]> {
  return request<OrganizationMembership[]>("/auth/memberships");
}

/**
 * Moves the session to another organization the user is already a member of and
 * returns the freshly-issued session. The server re-derives the role from that
 * membership, so this cannot escalate.
 */
export function switchOrganization(merchantId: string): Promise<AuthSession> {
  return authRequest<AuthSession>("/auth/switch", { merchant_id: merchantId });
}

export async function verifyEmail(token: string): Promise<void> {
  await authRequest("/auth/verify-email", { token });
}

export async function resendVerification(email: string): Promise<void> {
  await authRequest("/auth/resend-verification", { email });
}

export async function forgotPassword(email: string): Promise<void> {
  await authRequest("/auth/forgot-password", { email });
}

export async function resetPassword(token: string, password: string): Promise<void> {
  await authRequest("/auth/reset-password", { token, password });
}

function delay<T>(value: T, ms = LATENCY): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms));
}

export async function loginWithGoogle(credential: string): Promise<AuthSession> {
  const result = await authRequest<AuthSession>("/auth/google", { credential });
  return result;
}

/* --------------------------------- dashboard -------------------------------- */

export async function getDashboardMetrics(): Promise<DashboardMetrics> {
  const metrics = await request<{
    volume_minor?: number;
    currency?: string;
    successful?: number;
    failed?: number;
    success_rate?: number;
    volume_series?: { date: string; volume_minor: number }[];
  }>("/dashboard/metrics");
  return {
    volumeMinor: Number(metrics.volume_minor ?? 0),
    currency: metrics.currency ?? "NGN",
    successful: Number(metrics.successful ?? 0),
    failed: Number(metrics.failed ?? 0),
    successRate: Number(metrics.success_rate ?? 0),
    volumeSeries: (metrics.volume_series ?? []).map((point) => ({
      date: point.date,
      volumeMinor: Number(point.volume_minor ?? 0),
    })),
  };
}

/* --------------------------------- payments --------------------------------- */

/**
 * The payment-list filters the dashboard sends. Every field is optional and an
 * empty string means "do not narrow"; the server applies all of them to one
 * predicate, so the total and the status tally always describe the same rows.
 */
export interface PaymentFilters {
  environment?: Environment | "all";
  applicationId?: string;
  /** A developer grant id, or "general" for payments taken with a shared key. */
  developerAccessId?: string;
  status?: string;
  provider?: string;
  currency?: string;
  /** Free text: reference, payment id, payer name or payer email. */
  search?: string;
  from?: string;
  to?: string;
  page?: number;
  limit?: number;
}

export interface PaymentListResult {
  items: Payment[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
  /** Per-status counts for the same filter, used to label the filter chips. */
  summary: Record<string, number>;
}

interface PaymentResponse {
  id: string;
  reference?: string;
  amount?: number;
  currency?: string;
  status?: string;
  provider?: string;
  environment?: Environment;
  authorization_url?: string;
  description?: string;
  customer_id?: string;
  customer?: { id?: string; name?: string; email?: string; phone?: string };
  application_id?: string;
  api_key_id?: string;
  developer_access_id?: string;
  fee_minor?: number | null;
  failure_code?: string;
  failure_message?: string;
  metadata?: Record<string, unknown>;
  timeline?: { id: string; type: string; message: string; at: string }[];
  provider_response?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
}

function toPayment(payment: PaymentResponse): Payment {
  return {
    id: payment.id,
    reference: payment.reference ?? "",
    // The provider's own reference is not part of the public payment payload;
    // the timeline carries the provider's answers instead.
    providerReference: null,
    amountMinor: Number(payment.amount ?? 0),
    currency: payment.currency ?? "NGN",
    environment: payment.environment === "test" ? "test" : "live",
    status: (payment.status ?? "pending") as Payment["status"],
    provider: (payment.provider ?? "paystack") as Payment["provider"],
    authorizationUrl: payment.authorization_url ?? "",
    applicationId: payment.application_id ?? "",
    developerAccessId: payment.developer_access_id ?? null,
    apiKeyId: payment.api_key_id ?? null,
    customer: {
      name: payment.customer?.name ?? "",
      email: payment.customer?.email ?? "",
      phone: payment.customer?.phone ?? "",
    },
    description: payment.description ?? "",
    createdAt: payment.created_at ?? "",
    updatedAt: payment.updated_at ?? payment.created_at ?? "",
    timeline: (payment.timeline ?? []).map((event) => ({
      id: event.id,
      type: event.type as PaymentEventType,
      message: event.message,
      at: event.at,
    })),
    providerResponse: payment.provider_response ?? {},
    feeMinor: payment.fee_minor ?? null,
  };
}

export async function listPayments(filters: PaymentFilters = {}): Promise<PaymentListResult> {
  const params = new URLSearchParams();
  if (filters.environment && filters.environment !== "all") params.set("environment", filters.environment);
  else if (filters.environment === "all") params.set("environment", "all");
  if (filters.applicationId) params.set("application_id", filters.applicationId);
  if (filters.developerAccessId) params.set("developer_access_id", filters.developerAccessId);
  // "all" is the unfiltered default, so it must not be sent as a literal value:
  // the server would narrow to status/provider "all", which nothing matches.
  if (filters.status && filters.status !== "all") params.set("status", filters.status);
  if (filters.provider && filters.provider !== "all") params.set("provider", filters.provider);
  if (filters.currency) params.set("currency", filters.currency);
  if (filters.search?.trim()) params.set("q", filters.search.trim());
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  if (filters.page) params.set("page", String(filters.page));
  if (filters.limit) params.set("limit", String(filters.limit));

  const query = params.toString();
  const { items, page } = await requestPage<PaymentResponse>(`/payments${query ? `?${query}` : ""}`);
  return {
    items: items.map(toPayment),
    total: Number(page.total ?? items.length),
    page: Number(page.page ?? 1),
    limit: Number(page.limit ?? items.length),
    hasMore: Boolean(page.has_more),
    summary: page.summary ?? {},
  };
}

export function getPayment(id: string): Promise<Payment> {
  return request<PaymentResponse>(`/payments/${encodeURIComponent(id)}`).then(toPayment);
}

/** The values a merchant supplies for a dashboard-initiated test charge. */
export interface TestTransactionInput {
  /** Charge amount in minor units (kobo/cents). */
  amountMinor: number;
  currency: string;
  email: string;
  reference?: string;
  description?: string;
  /**
   * Where the provider sends the customer's browser after checkout. Without it
   * the provider shows its own default page and the dashboard never gets the
   * user back.
   */
  callbackUrl?: string;
}

/**
 * Starts a sandbox charge on one provider connection. The payment is real (it is
 * stored and appears in the dashboard), so completing it at the provider's
 * checkout makes the provider call PayConnect back, which in turn delivers the
 * event to your endpoint — exercising the whole webhook path end to end.
 * Only test connections are eligible; the server refuses live ones.
 */
export function createTestTransaction(connectionId: string, input: TestTransactionInput): Promise<Payment> {
  return request<PaymentResponse>("/payments/test", {
    method: "POST",
    data: {
      connection_id: connectionId,
      amount: input.amountMinor,
      currency: input.currency,
      email: input.email,
      reference: input.reference ?? "",
      description: input.description ?? "",
      callback_url: input.callbackUrl ?? "",
      customer: { email: input.email },
    },
  }).then(toPayment);
}

/* -------------------------------- providers --------------------------------- */

export function listProviders(): Promise<ProviderConnection[]> {
  return Promise.all([
    request<{ provider: ProviderConnection["provider"]; default_currency: string }[]>("/providers"),
    request<ProviderConnectionResponse[]>("/providers/connections"),
  ]).then(([supported, rawConnections]) => {
    const connections = rawConnections.map(toProviderConnection);
    return supported.flatMap(({ provider, default_currency }) => {
      const providerConnections = connections.filter((connection) => connection.provider === provider);
      return (["test", "live"] as const).map((environment) =>
        providerConnections.find((connection) => connection.environment === environment) ?? {
          id: "",
          provider,
          status: "disconnected",
          environment,
          displayName: "",
          isDefault: false,
          secretLast4: "",
          lastVerifiedAt: null,
          connectedAt: null,
          successRate: null,
          volumeLast30dMinor: null,
          currency: default_currency,
          webhookUrl: "",
          webhookSecretRequired: false,
          webhookSecretLabel: "",
          webhookSignatureHeader: "",
          webhookDashboardPath: "",
          webhookSecretMasked: "",
          webhookSecretSet: false,
          webhookLastEventAt: null,
        },
      );
    });
  });
}

/**
 * What a provider requires for inbound webhooks. Read from the API so the UI
 * never hard-codes a provider's secret/header rules.
 */
export interface ProviderWebhookCapability {
  provider: ProviderConnection["provider"];
  defaultCurrency: string;
  webhookSecretRequired: boolean;
  webhookSecretLabel: string;
  webhookSignatureHeader: string;
  webhookDashboardPath: string;
}

/** The supported-provider catalogue with each provider's webhook requirements. */
export function listProviderCapabilities(): Promise<ProviderWebhookCapability[]> {
  return request<
    {
      provider: ProviderConnection["provider"];
      default_currency: string;
      webhook_secret_required?: boolean;
      webhook_secret_label?: string;
      webhook_signature_header?: string;
      webhook_dashboard_path?: string;
    }[]
  >("/providers").then((rows) =>
    rows.map((row) => ({
      provider: row.provider,
      defaultCurrency: row.default_currency,
      webhookSecretRequired: row.webhook_secret_required ?? false,
      webhookSecretLabel: row.webhook_secret_label ?? "",
      webhookSignatureHeader: row.webhook_signature_header ?? "",
      webhookDashboardPath: row.webhook_dashboard_path ?? "",
    })),
  );
}

interface ProviderConnectionResponse {
  id: string;
  provider: ProviderConnection["provider"];
  environment: ProviderConnection["environment"];
  display_name: string;
  status: ProviderConnection["status"];
  is_default: boolean;
  secret_last4: string;
  currency: string;
  last_verified_at: string | null;
  connected_at: string | null;
  webhook_url?: string;
  webhook_secret_required?: boolean;
  webhook_secret_label?: string;
  webhook_signature_header?: string;
  webhook_dashboard_path?: string;
  webhook_secret_masked?: string;
  webhook_secret_set?: boolean;
  webhook_last_event_at?: string | null;
}

function toProviderConnection(connection: ProviderConnectionResponse): ProviderConnection {
  return {
    id: connection.id,
    provider: connection.provider,
    environment: connection.environment,
    displayName: connection.display_name,
    status: connection.status,
    isDefault: connection.is_default,
    secretLast4: connection.secret_last4,
    lastVerifiedAt: connection.last_verified_at,
    connectedAt: connection.connected_at,
    successRate: null,
    volumeLast30dMinor: null,
    currency: connection.currency,
    webhookUrl: connection.webhook_url ?? "",
    webhookSecretRequired: connection.webhook_secret_required ?? false,
    webhookSecretLabel: connection.webhook_secret_label ?? "",
    webhookSignatureHeader: connection.webhook_signature_header ?? "",
    webhookDashboardPath: connection.webhook_dashboard_path ?? "",
    webhookSecretMasked: connection.webhook_secret_masked ?? "",
    webhookSecretSet: connection.webhook_secret_set ?? false,
    webhookLastEventAt: connection.webhook_last_event_at ?? null,
  };
}

export function getProvider(id: string): Promise<ProviderConnection> {
  return request<ProviderConnectionResponse>(`/providers/connections/${encodeURIComponent(id)}`)
    .then(toProviderConnection);
}

export function createProviderConnection(input: {
  provider: ProviderConnection["provider"];
  environment: ProviderConnection["environment"];
  secretKey: string;
  displayName?: string;
  isDefault?: boolean;
  /** Optional inbound webhook signing value, for providers that need one. */
  webhookSecret?: string;
}): Promise<ProviderConnection> {
  return request<ProviderConnectionResponse>("/providers/connections", {
    method: "POST",
    data: {
      provider: input.provider,
      environment: input.environment,
      display_name: input.displayName ?? "",
      is_default: input.isDefault ?? false,
      webhook_secret: input.webhookSecret ?? "",
      credentials: { secret_key: input.secretKey },
    },
  }).then(toProviderConnection);
}

/**
 * Stores (or rotates) the inbound webhook signing value for a connection. Only
 * providers whose signing key is not the API secret key need one.
 */
export function setProviderWebhookSecret(id: string, webhookSecret: string): Promise<ProviderConnection> {
  return request<ProviderConnectionResponse>(`/providers/connections/${encodeURIComponent(id)}/webhook-secret`, {
    method: "POST",
    data: { webhook_secret: webhookSecret },
  }).then(toProviderConnection);
}

export function updateProviderConnection(
  id: string,
  input: { displayName?: string; isDefault?: boolean; secretKey?: string },
): Promise<ProviderConnection> {
  return request<ProviderConnectionResponse>(`/providers/connections/${encodeURIComponent(id)}`, {
    method: "PATCH",
    data: {
      ...(input.displayName === undefined ? {} : { display_name: input.displayName }),
      ...(input.isDefault === undefined ? {} : { is_default: input.isDefault }),
      ...(input.secretKey === undefined ? {} : { credentials: { secret_key: input.secretKey } }),
    },
  }).then(toProviderConnection);
}

export function testProviderConnection(id: string): Promise<{ status: string; provider: string; verified: boolean }> {
  return request(`/providers/connections/${encodeURIComponent(id)}/test`, { method: "POST", data: {} });
}

export function deleteProviderConnection(id: string): Promise<void> {
  return request<void>(`/providers/connections/${encodeURIComponent(id)}`, { method: "DELETE" });
}

/* ------------------------------- applications ------------------------------- */

export function listApplications(): Promise<Application[]> {
  return request<unknown[]>("/applications").then((applications) => applications.map(toApplication));
}

export function getApplication(id: string): Promise<Application> {
  return request<unknown>(`/applications/${encodeURIComponent(id)}`).then(toApplication);
}

function toApplication(value: unknown): Application {
  if (!value || typeof value !== "object") throw new Error("The application response was invalid.");
  const application = value as Record<string, unknown>;
  const stats = application.stats && typeof application.stats === "object"
    ? application.stats as Record<string, unknown>
    : {};
  return {
    id: String(application.id ?? ""),
    name: String(application.name ?? ""),
    description: String(application.description ?? ""),
    status: application.status === "paused" ? "paused" : "active",
    createdAt: String(application.createdAt ?? application.created_at ?? ""),
    lastUsedAt: typeof (application.lastUsedAt ?? application.last_used_at) === "string"
      ? String(application.lastUsedAt ?? application.last_used_at)
      : null,
    stats: {
      apiRequests: Number(stats.apiRequests ?? stats.api_requests ?? 0),
      successfulPayments: Number(stats.successfulPayments ?? stats.successful_payments ?? 0),
      failedPayments: Number(stats.failedPayments ?? stats.failed_payments ?? 0),
      apiErrors: Number(stats.apiErrors ?? stats.api_errors ?? 0),
      successRate: Number(stats.successRate ?? stats.success_rate ?? 0),
    },
  };
}

/** One key minted with an application, with its one-time secret. */
export interface CreatedKey {
  id: string;
  environment: ApiKey["environment"];
  scope: ApiKeyScope;
  secret: string;
}

export interface CreateApplicationResult {
  application: Application;
  /** The test key, for callers that predate the key pair. */
  apiKey: string;
  /** Both the test and production keys, in that order. */
  keys: CreatedKey[];
}

interface CreateApplicationResponse {
  application: Application;
  api_key: string;
  keys?: { id: string; environment: ApiKey["environment"]; scope: ApiKeyScope; secret: string }[];
}

export interface CreateApplicationInput {
  name: string;
  description: string;
  /** "general" (shared, the default) or "developer" (one grant). */
  keyScope?: ApiKeyScope;
  /** Required when keyScope is "developer"; names the developer grant. */
  developerAccessId?: string;
}

export async function createApplication(input: CreateApplicationInput): Promise<CreateApplicationResult> {
  const result = await request<CreateApplicationResponse>("/applications", {
    method: "POST",
    data: {
      name: input.name,
      description: input.description,
      key_scope: input.keyScope ?? "general",
      ...(input.developerAccessId ? { developer_access_id: input.developerAccessId } : {}),
    },
  });
  const keys = (result.keys ?? []).map((key) => ({
    id: key.id,
    environment: key.environment,
    scope: key.scope,
    secret: key.secret,
  }));
  return {
    application: toApplication(result.application),
    apiKey: result.api_key,
    keys,
  };
}

export function updateApplication(id: string, input: Partial<{
  name: string;
  description: string;
  status: "active" | "paused";
}>): Promise<Application> {
  return request<unknown>(`/applications/${encodeURIComponent(id)}`, {
    method: "PATCH",
    data: input,
  }).then(toApplication);
}

export function deleteApplication(id: string): Promise<void> {
  return request<void>(`/applications/${encodeURIComponent(id)}`, { method: "DELETE" });
}

/* --------------------------------- api keys --------------------------------- */

interface ApiKeyResponse {
  id: string;
  application_id?: string;
  environment: ApiKey["environment"];
  status?: ApiKey["status"];
  name: string;
  scope?: string;
  developer_access_id?: string;
  /** True for test keys, whose secret the API can show again. */
  revealable?: boolean;
  masked?: string;
  secret_masked?: string;
  created_at: string;
  last_used_at?: string | null;
  created_by?: string;
}

function toApiKey(key: ApiKeyResponse): ApiKey {
  return {
    id: key.id,
    applicationId: key.application_id ?? "",
    environment: key.environment,
    status: key.status ?? "active",
    scope: key.scope === "developer" ? "developer" : "general",
    developerAccessId: key.developer_access_id ?? null,
    revealable: Boolean(key.revealable),
    label: key.name,
    masked: key.masked ?? key.secret_masked ?? "",
    createdAt: key.created_at,
    lastUsedAt: key.last_used_at ?? null,
    createdByName: key.created_by ?? "",
  };
}

export async function listApiKeys(applicationId?: string): Promise<ApiKey[]> {
  const keys = await request<ApiKeyResponse[]>("/api-keys");
  const mapped = keys.map(toApiKey);
  return applicationId ? mapped.filter((key) => key.applicationId === applicationId) : mapped;
}

export async function revokeApiKey(id: string): Promise<void> {
  await request<{ revoked: boolean }>(`/api-keys/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function permanentlyDeleteRevokedApiKey(id: string): Promise<void> {
  await request<{ deleted: boolean }>(`/api-keys/${encodeURIComponent(id)}/purge`, { method: "DELETE" });
}

/**
 * Reads a test key's secret again so it can be copied. Live keys are shown once,
 * at creation, and the server refuses this call for them.
 */
export async function revealApiKey(id: string): Promise<string> {
  const result = await request<{ secret: string }>(
    `/api-keys/${encodeURIComponent(id)}/reveal`,
    { method: "POST", data: {} },
  );
  return result.secret;
}

/** Rotates a key: the old one is revoked and a fresh secret is returned once. */
export async function rotateApiKey(id: string): Promise<{ key: ApiKey; secret: string }> {
  const result = await request<{ key: ApiKeyResponse; secret: string }>(
    `/api-keys/${encodeURIComponent(id)}/rotate`,
    { method: "POST", data: {} },
  );
  return { key: toApiKey(result.key), secret: result.secret };
}

export async function createApplicationApiKey(input: {
  applicationId: string;
  environment: ApiKey["environment"];
  name: string;
  /** "general" (shared) or "developer" (bound to a developer grant). */
  scope?: ApiKeyScope;
  developerAccessId?: string;
}): Promise<{ key: ApiKey; secret: string }> {
  const result = await request<{ key: ApiKeyResponse; secret: string }>("/api-keys", {
    method: "POST",
    data: {
      application_id: input.applicationId,
      environment: input.environment,
      name: input.name,
      type: "secret",
      ...(input.scope === "developer" && input.developerAccessId
        ? { developer_access_id: input.developerAccessId }
        : {}),
    },
  });
  return { key: toApiKey(result.key), secret: result.secret };
}

/* -------------------------------- developers -------------------------------- */

export interface DeveloperInvitationInput {
  merchant_id?: string;
  email: string;
  name: string;
  permissions: DeveloperPermission[];
}

interface DeveloperResponse {
  id: string;
  access_id?: string;
  developer_id?: string;
  name: string;
  email: string;
  status: Developer["status"];
  application_ids?: string[] | null;
  applicationIds?: string[] | null;
  permissions?: string[] | null;
  invited_at?: string;
  invitedAt?: string;
  last_active_at?: string | null;
  lastActiveAt?: string | null;
}

export async function listDevelopers(): Promise<Developer[]> {
  const developers = await request<DeveloperResponse[]>("/developers");
  const validPermissions = new Set<string>(DEVELOPER_PERMISSIONS.map(({ id }) => id));
  return developers.map((developer) => ({
    id: developer.id,
    accessId: developer.access_id ?? developer.id,
    developerId: developer.developer_id ?? "",
    name: developer.name,
    email: developer.email,
    status: developer.status,
    applicationIds: developer.applicationIds ?? developer.application_ids ?? [],
    permissions: (developer.permissions ?? []).filter((permission): permission is DeveloperPermission =>
      validPermissions.has(permission),
    ),
    invitedAt: developer.invitedAt ?? developer.invited_at ?? "",
    lastActiveAt: developer.lastActiveAt ?? developer.last_active_at ?? null,
  }));
}

/** Sends an invitation and returns the grant id that developer-scoped keys bind to. */
export async function inviteDeveloper(input: DeveloperInvitationInput): Promise<{ accessId: string }> {
  const result = await request<{ id?: string }>("/developers/invite", { method: "POST", data: input });
  return { accessId: result?.id ?? "" };
}

/** Every merchant grant held by the signed-in identity, with apps and keys. */
export function getDeveloperSelfAccess(): Promise<SelfAccess[]> {
  return request<SelfAccess[]>("/developers/me");
}

/** The outcome of one sample webhook delivery, from either test surface. */
export interface TestWebhookResult {
  endpoint_id: string;
  /** The attempt that was just made, for linking to the delivery page. */
  delivery_id?: string;
  url: string;
  status: string;
  response_code?: number;
  /** What the endpoint replied, so a rejection reason is readable inline. */
  response_body?: string;
  last_error?: string;
}

/**
 * Delivers a sample event to an application's endpoint. The application must sit
 * behind one of the caller's active developer grants; the server derives the
 * merchant from the grant, never from the request.
 */
export function sendDeveloperTestWebhook(applicationId: string): Promise<TestWebhookResult> {
  return request<TestWebhookResult>(
    `/developers/me/applications/${encodeURIComponent(applicationId)}/test-webhook`,
    { method: "POST", data: {} },
  );
}

export async function removeDeveloperAccess(id: string): Promise<void> {
  await request<void>("/developers/invitations", {
    method: "DELETE",
    data: { invitation_id: id },
  });
}

/**
 * What the accept screen must know before the invitee types anything.
 *
 * `hasIdentity` and `isDeveloper` answer different questions and both are
 * needed: an address can be a registered merchant without ever having been a
 * developer, and it can have a developer profile while already owning a
 * merchant account of its own.
 */
export interface DeveloperInvitationPreview {
  email: string;
  merchant_id: string;
  merchant_name: string;
  permissions: string[];
  status: string;
  expires_at?: string;
  /** A login exists for this address, so a password is expected. */
  has_identity: boolean;
  /** A developer profile exists, so this is a returning developer. */
  is_developer: boolean;
  developer_status?: string;
}

/** Resolves an invitation token. Public: the token is the authorization. */
export function previewDeveloperInvitation(token: string): Promise<DeveloperInvitationPreview> {
  return request<DeveloperInvitationPreview>(`/developers/invitations/${encodeURIComponent(token)}`);
}

/**
 * Accepts an invitation and returns the credentials the invitee continues with.
 * Public by design: the invitee has no session yet, so the single-use token plus
 * the password they present are the authentication.
 */
export async function acceptDeveloperInvitation(
  token: string,
  credentials: { password: string; name?: string },
): Promise<{ merchant_id: string; merchant_name: string; role: string }> {
  return request<{ merchant_id: string; merchant_name: string; role: string }>(
    `/developers/invitations/${encodeURIComponent(token)}/accept`,
    { method: "POST", data: credentials },
  );
}

export function getDeveloper(id: string): Promise<Developer | undefined> {
  return listDevelopers().then((developers) => developers.find((dev) => dev.id === id));
}

/* --------------------------------- webhooks --------------------------------- */

interface WebhookEndpointResponse {
  id: string;
  application_id?: string;
  url: string;
  environment: WebhookEndpoint["environment"];
  status: WebhookEndpoint["status"];
  events?: string[] | null;
  signing_secret_masked?: string;
  created_at: string;
  delivery_success_rate?: number | null;
  deliveries?: number;
  delivered?: number;
  delivery_window_days?: number;
}

function toWebhookEndpoint(endpoint: WebhookEndpointResponse): WebhookEndpoint {
  return {
    id: endpoint.id,
    applicationId: endpoint.application_id ?? "",
    url: endpoint.url,
    environment: endpoint.environment,
    status: endpoint.status,
    events: endpoint.events ?? [],
    secretMasked: endpoint.signing_secret_masked ?? "",
    createdAt: endpoint.created_at,
    // Computed server-side from deliveries recorded in the reporting window.
    // Null (rather than 0) when nothing concluded, so the UI shows an em dash
    // instead of a rate no delivery backs.
    successRate: endpoint.delivery_success_rate ?? null,
    deliveries: endpoint.deliveries ?? 0,
    delivered: endpoint.delivered ?? 0,
    deliveryWindowDays: endpoint.delivery_window_days ?? 7,
  };
}

export function listWebhooks(): Promise<WebhookEndpoint[]> {
  return request<WebhookEndpointResponse[]>("/webhook-endpoints").then((rows) => rows.map(toWebhookEndpoint));
}

export interface CreateWebhookInput {
  url: string;
  environment: Environment;
  /** Empty means every event — the server treats an empty list as "all". */
  events: string[];
  applicationId?: string;
}

export interface CreatedWebhook {
  endpoint: WebhookEndpoint;
  /** Returned once, at creation: afterwards the API only ever returns the mask. */
  signingSecret: string;
}

/** Registers a receiving endpoint and returns its signing secret exactly once. */
export async function createWebhook(input: CreateWebhookInput): Promise<CreatedWebhook> {
  const result = await request<{ endpoint: WebhookEndpointResponse; signing_secret: string }>("/webhook-endpoints", {
    method: "POST",
    data: {
      url: input.url.trim(),
      environment: input.environment,
      status: "active",
      events: input.events,
      application_id: input.applicationId ?? "",
    },
  });
  return { endpoint: toWebhookEndpoint(result.endpoint), signingSecret: result.signing_secret };
}

/**
 * Sends one signed sample event to this endpoint and reports what it answered.
 * The delivery is recorded like any other, so the attempt appears in the list.
 */
export function testWebhookEndpoint(id: string): Promise<TestWebhookResult> {
  return request<TestWebhookResult>(`/webhook-endpoints/${encodeURIComponent(id)}/test`, {
    method: "POST",
    data: {},
  });
}

interface WebhookDeliveryResponse {
  id: string;
  endpoint_id: string;
  event: string;
  status: WebhookDelivery["status"];
  attempts: number;
  response_code?: number | null;
  response_body?: string;
  last_error?: string;
  next_attempt_at?: string | null;
  payload: Record<string, unknown>;
  created_at: string;
  delivered_at?: string | null;
}

function toWebhookDelivery(delivery: WebhookDeliveryResponse): WebhookDelivery {
  return {
    id: delivery.id,
    endpointId: delivery.endpoint_id,
    event: delivery.event,
    status: delivery.status,
    attempts: delivery.attempts,	    responseCode: delivery.response_code ?? null,
    responseBody: delivery.response_body ?? "",
    lastError: delivery.last_error ?? "",
    createdAt: delivery.created_at,
	    deliveredAt: delivery.delivered_at ?? null,
	    payload: delivery.payload ?? {},
  };
}

export async function listWebhookDeliveries(): Promise<WebhookDelivery[]> {
  const { items } = await requestPage<WebhookDeliveryResponse>("/webhook-deliveries?limit=20");
  return items.map(toWebhookDelivery);
}

export async function getWebhookDelivery(id: string): Promise<WebhookDelivery | undefined> {
  return request<WebhookDeliveryResponse>(`/webhook-deliveries/${encodeURIComponent(id)}`).then(toWebhookDelivery);
}

/** Queues one more attempt for a delivery, whatever its backoff window says. */
export async function retryWebhookDelivery(id: string): Promise<WebhookDelivery> {
  return request<WebhookDeliveryResponse>(`/webhook-deliveries/${encodeURIComponent(id)}/retry`, {
    method: "POST",
    data: {},
  }).then(toWebhookDelivery);
}

/* --------------------------- inbound provider events ------------------------- */

interface ProviderEventResponse {
  id: string;
  connection_id: string;
  provider: ProviderConnection["provider"];
  event_type: string;
  provider_event_id: string;
  payload?: Record<string, unknown>;
  processed_at?: string | null;
  created_at: string;
}

function toProviderEvent(event: ProviderEventResponse): ProviderEvent {
  return {
    id: event.id,
    connectionId: event.connection_id,
    provider: event.provider,
    eventType: event.event_type,
    providerEventId: event.provider_event_id,
    payload: event.payload,
    processedAt: event.processed_at ?? null,
    createdAt: event.created_at,
  };
}

/**
 * Recent inbound webhooks received for one provider connection, newest first.
 * This is the inbound counterpart to `listWebhookDeliveries`.
 */
export async function listInboundProviderEvents(connectionId: string): Promise<ProviderEvent[]> {
  const { items } = await requestPage<ProviderEventResponse>(
    `/webhook-events?connection=${encodeURIComponent(connectionId)}&limit=10`,
  );
  return items.map(toProviderEvent);
}

/**
 * Recent inbound provider events for the whole merchant, newest first — what the
 * providers sent PayConnect, across every connection. This is the raw material
 * the outbound deliveries are built from, so it is the place to look when a
 * payload does not match expectations.
 */
export async function listInboundEvents(): Promise<ProviderEvent[]> {
  const { items } = await requestPage<ProviderEventResponse>("/webhook-events?limit=10");
  return items.map(toProviderEvent);
}

/* ----------------------------------- logs ----------------------------------- */

interface ApiLogResponse {
  id: string;
  application_id?: string;
  environment?: ApiLog["environment"];
  method: string;
  path: string;
  status_code: number;
  duration_ms: number;
  source_ip?: string;
  created_at: string;
}

function toApiLog(log: ApiLogResponse): ApiLog {
  return {
    id: log.id,
    at: log.created_at,
    method: log.method,
    path: log.path,
    statusCode: log.status_code,
    durationMs: log.duration_ms,
    applicationId: log.application_id ?? "",
    environment: log.environment ?? "",
    sourceIp: log.source_ip ?? "",
  };
}

export async function listApiLogs(): Promise<ApiLog[]> {
  const logs: ApiLog[] = [];
  let pageNumber = 1;

  while (true) {
    const params = new URLSearchParams({ page: String(pageNumber), limit: "100" });
    const { items, page } = await requestPage<ApiLogResponse>(`/logs?${params.toString()}`);
    logs.push(...items.map(toApiLog));
    if (!page.has_more) return logs;
    if (items.length === 0) throw new Error("The API returned an empty log page while more results were expected.");
    pageNumber += 1;
  }
}

export async function getMerchantProfile(): Promise<MerchantProfile> {
  return request<MerchantProfile>("/merchant");
}

export async function updateMerchantProfile(input: { business_name: string }): Promise<MerchantProfile> {
  return request<MerchantProfile>("/merchant", { method: "PATCH", data: input });
}

export async function listAuditLogs(severity = "all"): Promise<AuditLog[]> {
  const logs: AuditLog[] = [];
  let pageNumber = 1;
  while (true) {
    const params = new URLSearchParams({ page: String(pageNumber), limit: "100" });
    if (severity !== "all") params.set("severity", severity);
    const { items, page } = await requestPage<AuditLog>(`/audit-logs?${params.toString()}`);
    logs.push(...items);
    if (!page.has_more) return logs;
    if (items.length === 0) throw new Error("The API returned an empty audit page while more results were expected.");
    pageNumber += 1;
  }
}

export async function exportAuditLogs(severity = "all"): Promise<Blob> {
  const params = new URLSearchParams();
  if (severity !== "all") params.set("severity", severity);
  const response = await apiClient.get(`/audit-logs/export?${params.toString()}`, {
    responseType: "blob",
  });
  return response.data as Blob;
}

/* ----------------------------------- team ----------------------------------- */

interface TeamMemberResponse {
  id: string;
  name: string;
  email: string;
  role: OrgMember["role"];
  status: OrgMember["status"];
  last_active_at: string | null;
}

export async function listTeam(): Promise<OrgMember[]> {
  const members = await request<TeamMemberResponse[]>("/team");
  return members.map((member) => ({
    id: member.id,
    name: member.name,
    email: member.email,
    role: member.role,
    status: member.status,
    lastActiveAt: member.last_active_at,
  }));
}

export async function addTeamMember(email: string, role: OrgMember["role"]): Promise<void> {
  await request("/team", { method: "POST", data: { email, role } });
}

export async function removeTeamMember(membershipId: string): Promise<void> {
  await request(`/team/${encodeURIComponent(membershipId)}`, { method: "DELETE" });
}

/* ---------------------------------- billing --------------------------------- */

/**
 * The public plan catalog. It is also fetched server-side by the marketing page
 * (see lib/plans.ts); this client copy exists for the in-app plan chooser, which
 * needs the merchant's own band from /billing/plan anyway.
 */
export function listPlans(): Promise<PlanCatalog> {
  return request<PlanCatalog>("/plans");
}

/** The merchant's plan, entitlements, renewal date and available options. */
export function getBillingPlan(): Promise<SubscriptionPlan> {
  return request<SubscriptionPlan>("/billing/plan");
}

/** This month's usage — applications, seats, payments and routed volume. */
export function getBillingUsage(): Promise<BillingUsage> {
  return request<BillingUsage>("/billing/usage");
}

/** The subscription trail, newest first. */
export function listSubscriptionEvents(): Promise<SubscriptionEvent[]> {
  return request<SubscriptionEvent[]>("/billing/events");
}

/**
 * Records the chosen plan and cadence. While no collection provider is wired
 * the response comes back `pending_payment` with a notice saying nothing was
 * charged — the UI must show that notice rather than a success check.
 */
export function choosePlan(input: {
  plan: PlanCode;
  interval: BillingInterval;
}): Promise<SubscriptionPlan> {
  return request<SubscriptionPlan>("/billing/checkout", {
    method: "POST",
    data: { plan: input.plan, interval: input.interval },
  });
}

/** Starts the 14-day Pro trial. Once per organization, no card. */
export function startPlanTrial(): Promise<SubscriptionPlan> {
  return request<SubscriptionPlan>("/billing/trial", { method: "POST" });
}

/** Cancels at period end; a plan that is only a recorded intent ends at once. */
export function cancelSubscription(): Promise<SubscriptionPlan> {
  return request<SubscriptionPlan>("/billing/cancel", { method: "POST" });
}

/** Withdraws a cancellation that has not taken effect yet. */
export function resumeSubscription(): Promise<SubscriptionPlan> {
  return request<SubscriptionPlan>("/billing/resume", { method: "POST" });
}
