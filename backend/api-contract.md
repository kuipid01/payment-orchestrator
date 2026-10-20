# PayConnect API contract (v1)

The Next.js frontend talks to the backend **only** through `frontend/src/lib/api.ts`.
That module is the mock implementation of the contract below — implement these
endpoints in Go and swap the mock bodies for `fetch` calls.

Conventions:

- Base URL: `https://api.payconnect.dev/v1`
- Auth: `Authorization: Bearer vpay_live_… | vpay_test_…` (developer keys) or a
  merchant session token (dashboard-only endpoints, marked ⛨ below).
- Money is sent as **minor units** (kobo/cents) + ISO currency code.
- Provider secrets are accepted **only** on the two ⛨ connect/rotate endpoints,
  encrypted immediately, and never returned by any endpoint.
- Errors use the envelope from `/docs/errors`.

## Endpoints the frontend expects

| Frontend function | Method & path | Auth | Notes |
| --- | --- | --- | --- |
| `getDashboardMetrics` | `GET /dashboard/metrics` | Bearer | volume, success/failed counts, 30-day series |
| `listPayments` | `GET /payments` | Bearer | filters: status, provider, currency, created_after/before |
| `getPayment` | `GET /payments/:id` | Bearer | includes timeline + redacted provider response |
| `createTestTransaction` | `POST /payments/test` ⛨ | session | body: `{ connection_id, amount, currency, email, reference? }`; starts a sandbox charge on a **test** connection and returns the provider's checkout URL. Live connections are refused. |
| `listProviders` | `GET /providers` | Bearer | the catalogue plus `connected`; masked secrets only (`secret_last4`) |
| `getProvider` | `GET /providers/connections/:id` ⛨ | session | masked secrets only; includes `webhook_url` and the provider's webhook requirements |
| list connections | `GET /providers/connections` ⛨ | session | all of the merchant's connections |
| connect provider | `POST /providers/connections` ⛨ | session | body: `{ provider, environment, display_name?, is_default?, webhook_secret?, credentials{…} }` → validate against the provider, encrypt, store |
| rotate credentials | `PATCH /providers/connections/:id` ⛨ | session | `credentials` replaces the secret; fields omitted are kept from the stored credential set |
| test connection | `POST /providers/connections/:id/test` ⛨ | session | re-validates the stored credentials against the provider |
| disconnect | `DELETE /providers/connections/:id` ⛨ | session | destroys the stored credentials |
| set provider webhook secret | `POST /providers/connections/:id/webhook-secret` ⛨ | session | body: `{ webhook_secret }`; only for providers whose signing key is not the API secret key |
| `listApplications` | `GET /applications` | Bearer | includes lifetime stats |
| `getApplication` | `GET /applications/:id` | Bearer | |
| create application | `POST /applications` ⛨ | session | environment-neutral; returns an initial test API key **once** |
| `listApiKeys` | `GET /api-keys` ⛨ | session | masked only |
| create API key | `POST /api-keys` ⛨ | session | body: application, environment, name, type; returns full key **once** |
| rotate / revoke key | `POST /api-keys/:id/rotate`, `DELETE /api-keys/:id` ⛨ | session | |
| permanently delete revoked key | `DELETE /api-keys/:id/purge` ⛨ | session | revoked keys only; clears payment API-key attribution, audit events remain |
| `listDevelopers` | `GET /developers` ⛨ | session | |
| `getDeveloper` | `GET /developers/:id` ⛨ | session | includes permissions |
| invite developer | `POST /developers/invite` ⛨ | session | body: email, applicationId, permissions[] |
| revoke developer | `POST /developers/:id/revoke` ⛨ | session | |
| `listWebhooks` | `GET /webhooks` | Bearer | masked signing secret; per-endpoint `delivery_success_rate`, `deliveries`, `delivered`, `delivery_window_days` |
| `listInboundProviderEvents` | `GET /webhook-events` | Bearer | received provider events; filter: connection, provider, event |
| `listWebhookDeliveries` | `GET /webhook-deliveries` | Bearer | filter: endpoint, status |
| `getWebhookDelivery` | `GET /webhook-deliveries/:id` | Bearer | includes payload + attempts |
| retry delivery | `POST /webhook-deliveries/:id/retry` | Bearer | |
| `listApiLogs` | `GET /logs` | Bearer | filter: application, status class, environment |
| `listAuditLogs` | `GET /audit-logs` ⛨ | session | severity filter |
| `listTeam` | `GET /org/members` ⛨ | session | |
| `listPlans` | `GET /plans` | public | the complete price catalog: every band, every plan, its entitlements, and the volume-fee terms. Public on purpose — the marketing page renders prices it does not own, so it cannot drift from what the API enforces. |
| `getBillingPlan` | `GET /billing/plan` ⛨ | session | current plan, status, interval, renewal/trial dates, the merchant's own band's options, and `checkout_enabled` |
| `getBillingUsage` | `GET /billing/usage` ⛨ | session | this calendar month: applications, seats, successful **live** payments and routed volume against the plan's limits, plus the accrued volume fee (capped and uncapped) |
| `listSubscriptionEvents` | `GET /billing/events` ⛨ | session | append-only subscription trail, newest first |
| `choosePlan` | `POST /billing/checkout` ⛨ | session | body: `{ plan, interval }`. Records the choice as `pending_payment`; returns the provider checkout URL once a collection provider is wired (`checkout_enabled`). While it is not, the response carries a `notice` saying nothing was charged |
| `startPlanTrial` | `POST /billing/trial` ⛨ | session | 14 days of Pro, no card, once per organization (409 `trial_already_used` after that) |
| `cancelSubscription` | `POST /billing/cancel` ⛨ | session | cancel at period end; a plan that is only a recorded intent ends immediately |
| `resumeSubscription` | `POST /billing/resume` ⛨ | session | withdraw a pending cancellation |

### Billing invariants

- Prices, entitlements and the volume-fee multiples live in
  `internal/billing/catalog.go` — one value, served to the UI. Do not add a
  second copy: the marketing page renders a *generated* copy of the same catalog
  (`go run ./cmd/catalogdump > ../frontend/src/lib/plan-catalog.json`) only for
  the case where the API is unreachable.
- Usage is **derived**, never stored in a counter: applications, seats, payments
  and routed volume come from the tables that own those facts, so the dashboard
  and the invoice can never disagree with the payments table.
- A `pending_payment` plan grants **no** paid entitlements. Until a provider
  confirms payment the merchant keeps what they have; only `active` and
  `trialing` grant the plan.
- The volume fee is measured from successful **live** payments in the merchant's
  billing currency, net of successful refunds. Test traffic is never billed, and
  the fee is invoiced, never split off a settlement.
- Billing never blocks the money path: payments, refunds and webhook delivery
  are unaffected by plan state, and every gate fails open.

## Security invariants (enforce in Go)

1. Provider secret keys exist only inside the vault service; responses never
   contain them — `secret_last4` at most. The per-connection inbound webhook
   secret is treated the same way: `webhook_secret_masked` at most. Provider
   connection responses also carry the computed `webhook_url`
   (`{API_BASE_URL}/v1/webhooks/{provider}/{connection_id}`) and the provider's
   `webhook_secret_required` / `webhook_signature_header` so the dashboard does
   not hard-code provider differences.
2. Developer API keys are shown in full **only** in the create/rotate response.
3. Every ⛨ mutation writes an audit log entry (actor, action, target, env).

## Provider credential shapes

`credentials` is the same object on connect and rotate. Every field is a string;
only the fields a provider needs are required.

| Provider | `credentials.secret_key` | `credentials.public_key` | `credentials.api_key` | `credentials.contract_code` | Inbound `webhook_secret` |
| --- | --- | --- | --- | --- | --- |
| Paystack | `sk_test_…` / `sk_live_…` | — | — | — | not needed (signs with the secret key) |
| Flutterwave | `FLWSECK-…` | — | — | — | **required**: the dashboard “secret hash” |
| Monnify | API secret | — | **required** | **required** | not needed (signs with the secret key) |
| Stripe | `sk_test_…` / `sk_live_…` (or `rk_…` restricted key) | — | — | — | **required**: endpoint signing secret, `whsec_…` |
| PayPal | REST app **client secret** | REST app **client ID** | — | — | **required**: the registered **webhook ID** |

Environment gating is enforced at connect: Stripe accepts `sk_test_`/`rk_test_`
for a `test` connection and `sk_live_`/`rk_live_` for `live`. PayPal has no key
prefix, so the connection environment selects the host — sandbox in `test`, live
in `live` — and a credential set only validates against the host it belongs to.

Stripe and PayPal are hosted-checkout providers. `POST /v1/payments` returns the
provider's `authorization_url`; the customer pays there. Stripe confirms through
its signed webhook. PayPal orders are captured by PayConnect (on the approval
webhook or the customer-return verify path) because PayPal authorizes and
captures separately. Both webhook receivers are the generic
`POST /v1/webhooks/{provider}/{connection_id}`; nothing a webhook says is
trusted before its signature is verified, and reconciliation reads the provider
again before a payment is marked settled.

## Delivery success metric

The dashboard's **Delivery success** figure is not stored: each endpoint response
carries the numbers it was computed from, so the same rate is reproducible from
`GET /webhook-deliveries`.

- Window: the last `delivery_window_days` (7) days, by delivery `created_at`.
- Denominator `deliveries`: attempts that reached a terminal state in the window
  — `delivered`, `failed` or `dead`. A `pending`/`retrying` attempt is still in
  flight and is never counted as a failure.
- Numerator `delivered`: how many of those reached the endpoint, i.e. the attempt
  received a 2xx.
- `delivery_success_rate` is `delivered / deliveries × 100`, rounded to one
  decimal, and is **omitted** when `deliveries` is 0 so the UI shows “—” instead
  of a 100% that no delivery backs.

A rate is therefore always accompanied by its counts: `6 of 8 deliveries · last
7 days`, never a bare percentage.
4. Environment isolation: a `vpay_test_…` key can never read or mutate live data.
