# PayConnect — Webhook flow (tight plan)

> Goal: one intuitive webhook story. Today the code already implements most of
> what we need; the gap is the *surface* (URL not shown, per-provider secret
> missing) and a muddled mental model. This plan fixes both.

---

## 1. The mental model (the thing to get right first)

There are **two directions**, and they are scoped differently on purpose:

| | Direction | Scope | Count | Who creates it |
| --- | --- | --- | --- | --- |
| **A** | **Inbound** — provider → PayConnect | **one per provider connection** | many (test + live per provider) | auto-generated when the merchant connects the provider |
| **B** | **Outbound** — PayConnect → merchant | **one per merchant** (optional per-app override) | usually **one** | merchant adds their URL once |

So the answer to "should a merchant have a single webhook?" is:

- **Outbound: yes — one destination, and every application reuses it.** That is the
  `https://yourapp.com/hooks/payconnect` URL the merchant pastes once. Apps inherit it.
- **Inbound: no — one URL *per provider connection*, auto-generated.** It cannot be
  shared across providers because the URL is what selects the credential that verifies
  the signature. Share it by provider, not by app.

This is exactly the intuition in the request, just split into the two halves.

---

## 2. What already exists (don't rebuild it)

**Inbound — implemented and correct in shape**
- Route: `POST /v1/webhooks/:provider/:connection_id` — public, signature is the auth.
  [`routes.go`](backend/internal/webhooks/routes.go)
- Pipeline: resolve connection → verify signature → normalize → dedupe on
  `(provider, provider_event_id)` → reconcile payment → publish domain event → `200`.
  [`service.go`](backend/internal/webhooks/service.go)
- Per-provider signature verification lives in each adapter:
  [`paystack/webhook.go`](backend/internal/providers/paystack/webhook.go),
  [`flutterwave/webhook.go`](backend/internal/providers/flutterwave/webhook.go),
  [`moniepoint/webhook.go`](backend/internal/providers/moniepoint/webhook.go)
- Duplicate / unknown-reference events already return `200` (no retry storm). Good.

**Outbound — implemented and correct in shape**
- `webhook_endpoints` (merchant-scoped, optional `application_id`) + `webhook_deliveries`.
  [`model.go`](backend/internal/webhooks/model.go)
- Dispatcher: domain event → subscribed active endpoints → sign with `whsec_` → POST →
  exponential backoff → dead-letter after 6 attempts.
  [`dispatcher.go`](backend/internal/webhooks/dispatcher.go)
- Developer "send test webhook" already exists for apps:
  `POST /v1/developers/me/applications/:id/test-webhook`.

**The gap (what this plan adds)**
1. The generated inbound URL is **never shown** to the merchant (`GET /providers/connections/:id`
   omits it — [`dto.go`](backend/internal/providers/dto.go)).
2. There is **no per-connection webhook secret**, so providers whose signing key differs
   from the API secret cannot be verified (see matrix below — Flutterwave).
3. The webhooks UI is framed around a per-application endpoint, which fights the
   "one destination, reused by all apps" story.
4. No UI to see inbound health ("last event received").

---

## 3. Provider capability matrix (drives the UI, one table)

Signature verification uses the connection's **webhook secret** when present, else the
**secret key**. This is the only place provider differences should live.

| Provider | Signature header | Key used | Separate webhook secret? | Merchant action |
| --- | --- | --- | --- | --- |
| Paystack | `X-Paystack-Signature` | HMAC-SHA512(secret key) | **No** | Paste inbound URL only |
| Flutterwave | `verif-hash` | merchant-defined hash | **Yes** | Set the same hash in Flutterwave dashboard |
| Moniepoint | `x-moniepoint-signature` | HMAC-SHA256(secret key) | **No** | Paste inbound URL only |

Add a constant descriptor so the UI never hard-codes provider special cases:

```go
// constants/provider.go
type WebhookSupport struct {
    SignatureHeader        string
    RequiresWebhookSecret  bool
    SecretLabel            string // e.g. "Secret hash"
    DashboardPath          string // e.g. "Settings → Webhooks"
}
func WebhookSupportFor(p Provider) WebhookSupport
```

> **Bug this exposes:** Flutterwave's `verif-hash` is a value the merchant *chooses*,
> independent of the API secret key. The current adapter compares `verif-hash` against
> the stored API secret (`signatureValid(secretKey, …)`), so it only passes if the
> merchant set their dashboard hash equal to their secret key. The per-connection
> `webhook_secret` below fixes it properly.

---

## 4. The tight flow

### A. Inbound (provider → PayConnect)

```
Merchant connects Paystack
      │
      ▼
We mint connection id  ──►  inbound URL =
                            https://api.payconnect.com/v1/webhooks/paystack/conn_7f3a…
      │
      ▼
UI shows URL + "paste this in your Paystack dashboard"
      │
      ▼
Paystack POSTs event ──► receive → resolve → verify → normalize → dedupe → reconcile
      │
      ▼
domain event published  ──►  outbound fan-out (B)
```

Rules that keep it tight:
- **Versioned path stays** `/v1/webhooks/{provider}/{connection_id}` (the request's
  `api.payconnect.com/webhook/paystack/{id}` is the same idea, minus the version).
- `{connection_id}` is already an opaque random token (`conn_…`) — good enough, and it
  identifies the signing credential. **Decided: reuse `{connection_id}`** for the public
  URL. A rotatable `webhook_token` is deferred as later hardening, not built now.
- Verification order is fixed and non-negotiable: **verify signature on the raw body
  before parsing**, constant-time compare, never trust a header for identity.
- Always answer `2xx` for duplicates and events about references we don't own — never
  invite a provider retry loop.

### B. Outbound (PayConnect → merchant)

```
domain event (payment.succeeded, refund.failed, …)
      │
      ▼
find merchant's active endpoints for this environment whose `events` include the type
      │
      ▼
sign body (HMAC, X-PayConnect-Signature) → POST → 2xx?
      │                                      │ no
      ▼ yes                                  ▼
delivered                          retry with backoff → dead after 6
```

- **Decided: one endpoint per merchant, all apps inherit it.** Applications do not each
  need their own endpoint; the developer's test-webhook already falls back to the
  merchant-wide endpoint (`TestDeliver` in [`service.go`](backend/internal/webhooks/service.go)).
- Optional `application_id` stays as an **advanced override** for the rare app that must
  receive events at a different URL — never the default path.

---

## 5. Data model changes (one migration)

`00016_provider_webhook_secrets.sql`

```sql
ALTER TABLE provider_connections
    ADD COLUMN webhook_secret_encrypted BYTEA,           -- nullable; verif-hash etc.
    ADD COLUMN webhook_secret_masked    TEXT NOT NULL DEFAULT '',
    ADD COLUMN webhook_last_event_at    TIMESTAMPTZ;     -- UI "last received" health
```

Outbound `webhook_endpoints` / `webhook_deliveries` need **no schema change**.

---

## 6. API surface changes

| Change | Endpoint | Notes |
| --- | --- | --- |
| **Show inbound URL** | `GET /v1/providers/connections/:id`, and the connect `POST` response | add `webhook_url`, `webhook_secret_required`, `webhook_secret_masked`, `webhook_signature_header` |
| **Set / rotate inbound secret** | `POST /v1/providers/connections/:id/webhook-secret` ⛨ session | validates shape per provider, encrypts, audit-logs |
| **Provider capabilities** | `GET /v1/providers` | add `webhook_secret_required`, `signature_header` per provider |
| Outbound (unchanged) | `/v1/webhook-endpoints`, `/v1/webhook-deliveries` | already merchant-level |

`webhook_url` is **computed**, never stored: `{PUBLIC_API_URL}/v1/webhooks/{provider}/{connection_id}`.
Add `PUBLIC_API_URL` to config if not already present.

---

## 7. UI plan (the intuitive part)

### Screen 1 — Connect provider wizard (existing page, add a step)
After credentials validate, replace the immediate redirect with a **"Wire up webhooks"**
step:

```
┌─ Paystack connected ✓ ────────────────────────────────┐
│ 1. Open Paystack → Settings → API Keys & Webhooks     │
│ 2. Paste this URL:                                     │
│    ┌──────────────────────────────────────┬─────────┐ │
│    │ https://api.payconnect.com/v1/webhooks │  Copy  │ │
│    │ /paystack/conn_7f3a9c21               │         │ │
│    └──────────────────────────────────────┴─────────┘ │
│ 3. Paystack verifies automatically — no secret needed │
│                                       [ Done ]        │
└───────────────────────────────────────────────────────┘
```

For Flutterwave/Moniepoint the same step adds one field ("Secret hash — from your
Flutterwave dashboard"), because their signature key is separate.

### Screen 2 — Provider detail page (existing page, add a card)
New **"Provider webhooks (inbound)"** card: the URL with copy button, signature header
name, `webhook_secret_masked` + rotate, and **"Last event received 2m ago"** health from
`webhook_last_event_at`. This is where a merchant returns to fix a broken hook.

### Screen 3 — Webhooks page (reframe outbound)
Lead with the single destination, not a list of app-bound endpoints:

```
Your receiving endpoint
https://yourapp.com/hooks/payconnect     [Active]   signed · 99.2% delivered
Used by all applications · payment.succeeded, payment.failed …
                                          [Edit]  [Send test]  [View deliveries]
```

- Drop the per-application column as the primary framing; move app-scoping into an
  "Advanced" drawer.
- Keep the deliveries list and the delivery-detail page as-is.
- Add a small **Inbound** tab/section listing recent provider events per connection, so
  both directions are legible in one place.

---

## 8. Phasing

**Phase 1 — make inbound legible (highest value, smallest change)**
1. `WebhookSupportFor` capability matrix in `constants`.
2. Add `webhook_url` / `webhook_secret_required` to the connection DTO (computed).
3. Migration + `webhook_secret` on the connection; wire into `VerifyWebhook`
   (Flutterwave uses it; others fall back to secret key).
4. Provider detail card + connect-wizard step (Screens 1 & 2).

**Phase 2 — polish**
5. Reframe the webhooks page as one destination (Screen 3).
6. Inbound event list + `webhook_last_event_at` health.
7. Rewrite [`docs/webhooks`](frontend/src/app/(site)/docs/webhooks/page.tsx) around the two directions.

---

## 9. Decisions (locked)

1. **Inbound URL identity** — **reuse `{connection_id}`**. No new token column; a
   rotatable `webhook_token` is deferred hardening.
2. **Outbound scoping** — **one merchant-level endpoint, all applications inherit it.**
   Per-application override stays as an advanced option only.

---

## 10. Invariants (must hold)

- Provider and merchant signing secrets are encrypted at rest and **never returned** —
  `*_masked` at most.
- Signature verified on the raw body, constant-time, before any parsing.
- Inbound is idempotent on `(provider, provider_event_id)`; duplicates return `200`.
- Outbound delivery failure never fails the business transaction; it is recorded and retried.
- Environment isolation: `test` keys never see or write `live` endpoints.
- The inbound URL is public by design — integrity comes from the signature, not secrecy.
