/**
 * Demo dataset for the parts of the dashboard that are still seeded rather than
 * fetched: API logs, audit logs and the team list.
 * Payments, applications, API keys, developers, metrics and webhooks — endpoints,
 * deliveries and inbound provider events — all come from the API.
 *
 * Dates are pinned around Sep 2026 so relative times ("2m ago") render sensibly.
 */

import type { ApiLog, AuditLog, OrgMember } from "./types";

/* ----------------------------------- logs ----------------------------------- */

export const API_LOGS: ApiLog[] = [
  { id: "log_01", at: "2026-09-30T09:32:10Z", method: "POST", path: "/v1/payments", statusCode: 200, durationMs: 142, applicationId: "app_8f3k21", environment: "live", sourceIp: "102.89.44.7" },
  { id: "log_02", at: "2026-09-30T09:31:02Z", method: "POST", path: "/v1/payments", statusCode: 200, durationMs: 131, applicationId: "app_8f3k21", environment: "live", sourceIp: "102.89.44.7" },
  { id: "log_03", at: "2026-09-30T09:28:40Z", method: "GET", path: "/v1/payments/pay_9x41k2", statusCode: 200, durationMs: 38, applicationId: "app_8f3k21", environment: "live", sourceIp: "197.210.85.2" },
  { id: "log_04", at: "2026-09-30T09:25:19Z", method: "POST", path: "/v1/payments", statusCode: 500, durationMs: 2104, applicationId: "app_2m9x47", environment: "test", sourceIp: "105.112.33.9" },
  { id: "log_05", at: "2026-09-30T09:24:51Z", method: "GET", path: "/v1/payments?limit=20", statusCode: 200, durationMs: 45, applicationId: "app_8f3k21", environment: "live", sourceIp: "197.210.85.2" },
  { id: "log_06", at: "2026-09-30T09:21:07Z", method: "POST", path: "/v1/payments", statusCode: 402, durationMs: 96, applicationId: "app_2m9x47", environment: "test", sourceIp: "105.112.33.9" },
  { id: "log_07", at: "2026-09-30T09:18:33Z", method: "GET", path: "/v1/webhooks/wh_01", statusCode: 200, durationMs: 29, applicationId: "app_8f3k21", environment: "live", sourceIp: "102.89.44.7" },
  { id: "log_08", at: "2026-09-30T09:15:12Z", method: "POST", path: "/v1/payments", statusCode: 200, durationMs: 122, applicationId: "app_2m9x47", environment: "test", sourceIp: "105.112.33.9" },
];

export const AUDIT_LOGS: AuditLog[] = [
  { id: "aud_01", at: "2026-09-30T08:55:00Z", actorName: "John Merchant", actorRole: "Owner", action: "Provider credentials rotated", detail: "Paystack live secret key rotated", severity: "critical" },
  { id: "aud_02", at: "2026-09-29T17:20:00Z", actorName: "John Merchant", actorRole: "Owner", action: "Developer invited", detail: "sam@agency.co invited to My Ecommerce Store", severity: "info" },
  { id: "aud_03", at: "2026-09-29T12:02:00Z", actorName: "John Merchant", actorRole: "Owner", action: "API key generated", detail: "New test key for Trex Mobile App", severity: "warning" },
  { id: "aud_04", at: "2026-09-28T10:14:00Z", actorName: "System", actorRole: "System", action: "API key revoked", detail: "vpay_test_••••7a21 revoked after inactivity", severity: "warning" },
  { id: "aud_05", at: "2026-09-27T09:41:00Z", actorName: "Grace Nwosu", actorRole: "Developer", action: "Webhook endpoint updated", detail: "URL changed for dev.trex.app/hooks/payconnect", severity: "info" },
  { id: "aud_06", at: "2026-09-26T16:30:00Z", actorName: "John Merchant", actorRole: "Owner", action: "Provider connected", detail: "Flutterwave connected (live)", severity: "critical" },
];

export const TEAM: OrgMember[] = [
  { id: "mem_01", name: "John Merchant", email: "john@mycompany.com", role: "Owner", status: "active", lastActiveAt: "2026-09-30T09:30:00Z" },
  { id: "mem_02", name: "Amara Ude", email: "amara@mycompany.com", role: "Admin", status: "active", lastActiveAt: "2026-09-29T18:22:00Z" },
  { id: "mem_03", name: "John Developer", email: "john.dev@example.com", role: "Developer", status: "active", lastActiveAt: "2026-09-30T09:31:00Z" },
  { id: "mem_04", name: "Ravi Patel", email: "ravi@consult.io", role: "Viewer", status: "invited", lastActiveAt: null },
];

