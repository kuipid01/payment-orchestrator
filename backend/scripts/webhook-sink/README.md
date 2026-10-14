# Webhook sink — a local receiver for outbound deliveries

A tiny, dependency-free HTTP server that stands in for a merchant's endpoint while
you develop. Point an endpoint's URL at it, and every delivery PayConnect makes
arrives here with its signature, headers and body intact — and the reply this
server sends is what the dashboard shows on the delivery page.

```bash
cd backend
make webhook-sink            # → http://localhost:4000
# or, with a signing secret so signatures are verified:
node scripts/webhook-sink/server.mjs --secret whsec_…
```

Then:

1. Open <http://localhost:4000/> — that is the control page, not a webhook route.
2. Copy the URL it shows: `http://localhost:4000/hooks/payconnect`.
3. Paste it as the receiving endpoint's URL (it accepts `http://` for local work).
4. Add the endpoint's signing secret on the page to turn on signature checking.
5. Trigger an event — the "Send test" action, a test payment, or "Retry delivery"
   in the dashboard. It shows up on the page within a second.

## What the page gives you

- **The live request log.** Method, path, time, event type, whether the signature
  matched, and the status we replied with. Expanding a row shows every header
  (including `X-PayConnect-Signature`, `X-PayConnect-Event`,
  `X-PayConnect-Delivery-ID`) and the payload, pretty-printed.
- **A reply simulator.** Set the status, `Content-Type`, body and an optional
  delay, and presets for the usual rejections — `401 invalid signature`,
  `422 unhandled event`, `500`. Because the dispatcher now keeps the endpoint's
  reply, each of these is visible on the delivery page as the reason the event
  was rejected.
- **Signature verification.** Given the endpoint's signing secret, the sink
  recomputes the HMAC-SHA256 of the raw body and compares it in constant time,
  exactly as the dispatcher signs it.

Only the first 2 KB of a reply is stored by PayConnect (`ResponseBodyLimit`), so
keep the body you are testing with small enough to see the whole thing.

## Options

| Flag | Default | Purpose |
| --- | --- | --- |
| `--port <n>` | `4000` | Listen on another port when 4000 is taken. |
| `--host <addr>` | `127.0.0.1` | Bind address. Use `0.0.0.0` to accept from other machines or a tunnel; the page then advertises your LAN address. |
| `--secret <whsec>` | — | Signing secret, to verify incoming signatures. |
| `--status <n>` | `200` | Reply status. |
| `--body <text>` | `{"received":true}` | Reply body. |
| `--delay <ms>` | `0` | Reply delay, to exercise the 10s delivery timeout. |

`SIGINT` (Ctrl+C) stops it. Requests and settings live in memory only, so a
restart starts clean.

## Troubleshooting

- **Nothing arrives.** The endpoint must be `active` and in the same environment
  as the event (`test` events are never delivered to `live` endpoints). The
  dispatcher logs each attempt — and with `LOG_WEBHOOK_VERBOSE=true` it logs the
  fan-out and skips too.
- **Signature shows ✗.** You are most likely verifying with the *masked* secret
  (`whsec_…abcd`). Only the full secret, shown once when the endpoint was
  created, produces a matching HMAC.
- **`EADDRINUSE`.** Another process holds the port; start the sink with
  `--port 4001` and use the URL the page then shows.
- **A deployed backend cannot reach it.** `127.0.0.1` is not routable from
  outside your machine: bind with `--host 0.0.0.0` and expose the port through a
  tunnel (ngrok, Cloudflare Tunnel, Tailscale), then paste the tunnel's URL.
