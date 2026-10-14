# PayConnect

Secure payment infrastructure layer between **merchants** and **developers**.

```
Merchant ──▶ connects Paystack / Flutterwave / Stripe
              │
              ▼
        PayConnect secure vault  (credentials encrypted at rest, never exposed)
              │
              ▼
        Developer uses PayConnect API key (vpay_live_… / vpay_test_…)
              │
              ▼
        Payment provider processes the charge
```

Your provider secrets never leave the vault. Developers integrate with the
PayConnect API and never see a provider credential.

## Structure

```
mypayconnect/
├── frontend/   Next.js 15 (App Router) + TypeScript + Tailwind + shadcn-style UI
└── backend/    Go (placeholder: module + cmd/api main + API contract notes)
```

## Frontend

```bash
cd frontend
npm install
npm run dev      # http://localhost:3000
```

Everything currently runs on **typed mock data** (`src/lib/mock-data.ts`) behind a
thin `src/lib/api.ts` client facade — swap the mock adapters for real `fetch`
calls against the Go backend without touching UI code.

### Google sign-in

Set the same Google OAuth Web client ID in both environments:

```env
# backend/.env
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com

# frontend/.env.local
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
NEXT_PUBLIC_API_URL=http://localhost:8080/v1
```

Add `http://localhost:3000` as an authorized JavaScript origin for that Google
client. Air loads `backend/.env`; Next.js loads `frontend/.env.local`.

### Production app subdomain

The dashboard and authentication pages can be served from `app.payconnect.com`,
while the landing page and public documentation stay on `payconnect.com`.

1. Configure the frontend deployment to serve both `payconnect.com` (and
   optionally `www.payconnect.com`) and `app.payconnect.com`. Point the DNS
   records at the frontend host required by your deployment provider.
2. Set `APP_URL=https://app.payconnect.com` in the frontend deployment
   environment. The host middleware redirects dashboard/auth routes requested on
   the marketing domain to this URL and sends the app subdomain root to
   `/dashboard`. Localhost and unrelated preview hosts are not redirected.
3. Set the backend `ALLOWED_ORIGINS` to include the exact origin
   `https://app.payconnect.com` (alongside any other intentional origins). The
   API uses credentialed CORS for the session cookies; do not use a wildcard
   origin.
4. Add `https://app.payconnect.com` as an authorized JavaScript origin for the
   Google OAuth Web client, and set `NEXT_PUBLIC_GOOGLE_CLIENT_ID` to that
   client's ID in the frontend deployment.

The API session cookies are HTTP-only and remain scoped to the API host; browser
requests from the app subdomain use credentialed CORS. Serve both hosts over
HTTPS in production.

- `src/lib/types.ts` — domain types (Payment, Provider, Application, ApiKey…)
- `src/lib/api.ts` — data access facade (mock today, REST tomorrow)
- `src/lib/mock-data.ts` — demo dataset used across every page

### Route tree

```
/                        Landing
/pricing                 Pricing
/how-it-works            How it works
/docs/*                  Documentation (8 pages)
/login | /register | /forgot-password | /reset-password | /verify-email
/onboarding              4-step onboarding
/dashboard               Merchant overview
/payments | /payments/[id]
/providers | /providers/connect/[provider] | /providers/[id]
/applications | /applications/new | /applications/[id]
/api-keys
/developers | /developers/invite | /developers/[id]
/webhooks | /webhooks/[id]
/logs
/settings/*              profile · organization · team · security ·
                         notifications · billing · audit-logs
/403 | /404 | /500 | /maintenance
```

## Backend (Go)

```bash
cd backend
go mod init github.com/kuipid01/payconnect   # already done in go.mod
make dev                                    # Air reloads the API on :8080
```

Google sign-in verifies ID tokens on the backend. The first Google sign-in
creates an organization; subsequent sign-ins link by verified email or Google
subject. The database migration adds the Google subject column.

### Webhook logging

A provider calling us used to be invisible in the logs. Now the whole webhook
pipeline traces, gated by a flag:

```env
# backend/.env
LOG_LEVEL=debug              # debug | info | warn | error
LOG_WEBHOOK_VERBOSE=true     # full inbound + outbound webhook trace
```

`LOG_WEBHOOK_VERBOSE` defaults to **on outside production** and off in
production. It emits a deliberately small trace: **one line per inbound
webhook** and one per outbound delivery, so a busy integration stays readable.

```
# inbound — one summary line per event, carrying everything needed
webhook inbound processed   request_id=req_… provider=paystack connection_id=conn_… event_type=payment.succeeded reference=ref_… amount_minor=10000 currency=NGN payment_id=pay_… payment_status=successful outcome=reconciled

# outbound — one line per fan-out and per delivery attempt
webhook event fanning out   event=payment.succeeded environment=test endpoints=1
webhook delivery attempt    delivery_id=whd_… endpoint_id=whe_… url=https://… attempt=1
webhook delivered           delivery_id=whd_… attempts=1 response_code=200
```

`outcome` is `reconciled`, `no_matching_payment` (a provider event we do not own,
accepted and ignored), or `duplicate` (a provider retry).

Rejections (bad signature, unknown connection, reconcile failure) are logged at
warning level **regardless of the flag** — the flag only controls the happy-path
trace. Signature values and bodies are never logged, only their presence.

Setting `LOG_WEBHOOK_VERBOSE=false` leaves only those warnings, so the log goes
quiet for normal traffic while still reporting anything that goes wrong.

If an inbound call is still silent with the flag on, check that the request
reached the process at all: a wrong `connection_id` logs
`webhook rejected: connection not found`, and a signature mismatch logs
`webhook rejected: signature verification failed` with the reason.
# payment-orchestrator
