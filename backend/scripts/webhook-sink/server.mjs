#!/usr/bin/env node
/**
 * A local webhook receiver — the "sink" you point an outbound endpoint at while
 * developing, and the quickest way to see the delivery page's new response-body
 * capture work end to end.
 *
 *   node scripts/webhook-sink/server.mjs
 *   → paste http://localhost:4000/hooks/payconnect into your endpoint's URL
 *
 * Every request that arrives is recorded verbatim — method, headers, raw body —
 * and the page at `/` lists them newest first, checking the HMAC signature when
 * you give it the endpoint's signing secret. The reply the sink sends is what
 * PayConnect stores as the delivery's response code and body, so switching the
 * sink between "accept" and a rejection is how you reproduce a merchant whose
 * endpoint is refusing events.
 *
 * Zero dependencies on purpose: there is nothing to install, and the file can be
 * read top to bottom in one sitting.
 */

import { createHmac, timingSafeEqual } from "node:crypto";
import { readFileSync } from "node:fs";
import { createServer } from "node:http";
import { networkInterfaces } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const PAGE = readFileSync(join(HERE, "index.html"), "utf8");

/** Bodies are bounded: the sink listens on the open internet's front porch. */
const MAX_BODY_BYTES = 1024 * 1024;
/** How many requests the page can show; older ones fall off the end. */
const HISTORY = 50;
/** Mirrors webhooks.ResponseBodyLimit — how much of our reply PayConnect keeps. */
const RESPONSE_BODY_STORED = 2048;

const USAGE = `Local webhook receiver for PayConnect deliveries.

  node scripts/webhook-sink/server.mjs [options]

  --port <n>        listen on this port                  (default 4000)
  --host <addr>     bind to this address                 (default 127.0.0.1)
  --secret <whsec>  endpoint signing secret, to verify signatures
  --status <n>      reply with this status code          (default 200)
  --body <text>     reply with this body
  --delay <ms>      wait before replying, to test timeouts
  -h, --help        show this text

The page at http://localhost:4000/ changes the reply live, so the options above
are only the starting point.`;

/** Reply presets, shared with the page so both offer the same choices. */
const PRESETS = [
  { key: "accept", label: "Accept", status: 200, contentType: "application/json", body: '{"received":true}' },
  {
    key: "invalid-signature",
    label: "Reject: bad signature",
    status: 401,
    contentType: "application/json",
    body: '{"error":"invalid signature"}',
  },
  {
    key: "unsupported-event",
    label: "Reject: unhandled event",
    status: 422,
    contentType: "application/json",
    body: '{"error":"unsupported event type","detail":"This endpoint does not handle payment.succeeded"}',
  },
  {
    key: "server-error",
    label: "Fail: 500",
    status: 500,
    contentType: "application/json",
    body: '{"error":"database unavailable","trace_id":"sink-demo"}',
  },
];

const options = parseArgs(process.argv.slice(2));
if (options.help) {
  console.log(USAGE);
  process.exit(0);
}

const config = {
  status: options.status,
  contentType: "application/json",
  body: options.body,
  delayMs: options.delay,
  secret: options.secret,
};

/** Received requests, newest first. */
const requests = [];
let received = 0;

const server = createServer((req, res) => {
  handle(req, res).catch((error) => {
    // A throw here is a bug in the sink, not in the delivery: say so plainly and
    // answer 500 rather than leaving the caller waiting for a socket timeout.
    console.error("sink error:", error);
    if (!res.headersSent) {
      res.writeHead(500, { "Content-Type": "text/plain" });
    }
    res.end("sink error");
  });
});

server.on("error", (error) => {
  if (error.code === "EADDRINUSE") {
    console.error(
      `Port ${options.port} is already in use. Start the sink elsewhere: node scripts/webhook-sink/server.mjs --port ${options.port + 1}`,
    );
    process.exit(1);
  }
  throw error;
});

server.listen(options.port, options.host, () => {
  const { port } = server.address();
  console.log(`Webhook sink listening on ${publicURL()}`);
  console.log(`Paste this into your endpoint's URL:\n\n  ${publicURL()}\n`);
  console.log(`Open http://${describeHost(options.host)}:${port}/ to watch what arrives.`);
  if (config.secret) {
    console.log("Signature verification is on for the configured signing secret.");
  } else {
    console.log("No signing secret set yet — add it on the page to verify signatures.");
  }
  console.log("Ctrl+C to stop.\n");
});

/* --------------------------------- requests --------------------------------- */

async function handle(req, res) {
  const url = new URL(req.url, `http://${req.headers.host ?? "localhost"}`);

  // The page's own noise must never show up in the request log.
  if (url.pathname === "/favicon.ico") {
    res.writeHead(204).end();
    return;
  }
  if (url.pathname === "/" || url.pathname === "/index.html") {
    res.writeHead(200, { "Content-Type": "text/html; charset=utf-8", "Cache-Control": "no-store" });
    res.end(PAGE);
    return;
  }
  if (url.pathname.startsWith("/__sink/")) {
    await control(req, res, url);
    return;
  }
  await capture(req, res, url);
}

/** Records a delivered webhook and answers with the configured reply. */
async function capture(req, res, url) {
  const raw = await readBody(req);
  const headers = flattenHeaders(req.headers);
  const body = raw.text;
  const signature = headers["x-payconnect-signature"] ?? "";

  let payload = null;
  try {
    payload = body.length > 0 ? JSON.parse(body) : null;
  } catch {
    // Not every endpoint posts JSON. Keep the raw text; just skip the summary.
  }

  const entry = {
    id: `req_${++received}_${Date.now().toString(36)}`,
    at: new Date().toISOString(),
    method: req.method,
    path: url.pathname + url.search,
    headers,
    body,
    bytes: raw.bytes,
    truncated: raw.truncated,
    payload,
    event: pickString(payload?.type, headers["x-payconnect-event"]),
    deliveryId: pickString(payload?.meta?.delivery_id, headers["x-payconnect-delivery-id"]),
    environment: pickString(payload?.meta?.environment),
    signature,
    reply: { status: config.status, contentType: config.contentType, body: config.body, delayMs: config.delayMs },
  };

  requests.unshift(entry);
  if (requests.length > HISTORY) {
    requests.length = HISTORY;
  }

  // The verdict here is only for the console line: the page recomputes it on
  // every poll so the whole log is re-checked when a secret is set.
  const atArrival = checkSignature(config.secret, signature, raw.buffer);
  const verdict =
    atArrival === true ? " signature ok" : atArrival === false ? " SIGNATURE INVALID" : "";
  console.log(
    `${entry.at}  ${entry.method} ${entry.path}  →  ${entry.reply.status}${entry.event ? `  ${entry.event}` : ""}${verdict}`,
  );

  if (config.delayMs > 0) {
    await sleep(config.delayMs);
  }

  // 204 and 304 must not carry a body, whatever we are configured to say.
  const replyBody = config.status === 204 || config.status === 304 ? "" : config.body;
  const headersOut = { "Content-Type": config.contentType };
  if (replyBody.length > 0) {
    headersOut["Content-Length"] = String(Buffer.byteLength(replyBody));
  }
  res.writeHead(config.status, headersOut);
  res.end(replyBody);
}

/* ------------------------------ control routes ------------------------------ */

async function control(req, res, url) {
  const route = url.pathname.slice("/__sink/".length);

  if (req.method === "GET" && route === "state") {
    return sendJSON(res, 200, snapshot());
  }
  if (req.method === "POST" && route === "config") {
    const patch = await readJSON(req);
    applyConfig(patch);
    return sendJSON(res, 200, snapshot());
  }
  if (req.method === "POST" && route === "clear") {
    // The counter goes with the log: an empty list that still claims a dozen
    // requests reads like a bug rather than a reset.
    requests.length = 0;
    received = 0;
    return sendJSON(res, 200, snapshot());
  }
  return sendJSON(res, 404, { error: "unknown sink route" });
}

/** Every mutable knob the page can change, validated in one place. */
function applyConfig(patch) {
  if (!patch || typeof patch !== "object") {
    return;
  }
  if (patch.status !== undefined) {
    const status = Number(patch.status);
    if (Number.isInteger(status) && status >= 100 && status <= 599) {
      config.status = status;
    }
  }
  if (typeof patch.contentType === "string" && patch.contentType.trim() !== "") {
    config.contentType = patch.contentType.trim();
  }
  if (typeof patch.body === "string") {
    config.body = patch.body;
  }
  if (patch.delayMs !== undefined) {
    const delay = Number(patch.delayMs);
    if (Number.isFinite(delay) && delay >= 0 && delay <= 30000) {
      config.delayMs = Math.round(delay);
    }
  }
  if (typeof patch.secret === "string") {
    config.secret = patch.secret.trim();
  }
}

function snapshot() {
  // Verdicts are recomputed, not stored: pasting the signing secret re-checks
  // the requests that already arrived instead of only the ones that follow.
  const withVerdicts = requests.map((request) => ({
    ...request,
    signatureValid: checkSignature(config.secret, request.signature, Buffer.from(request.body, "utf8")),
  }));
  return {
    url: publicURL(),
    config: { ...config },
    presets: PRESETS,
    stats: {
      received: received,
      showing: withVerdicts.length,
      rejected: withVerdicts.filter((r) => r.reply.status >= 400).length,
      lastAt: withVerdicts.length > 0 ? withVerdicts[0].at : null,
    },
    limits: { requestBodyBytes: MAX_BODY_BYTES, responseBodyStored: RESPONSE_BODY_STORED },
    requests: withVerdicts,
  };
}

/** Same HMAC the dispatcher signs with, over the bytes exactly as they arrived. */
function checkSignature(secret, signature, body) {
  if (!secret || signature === "") {
    return null;
  }
  const expected = createHmac("sha256", secret).update(body).digest("hex");
  const got = Buffer.from(signature.trim(), "utf8");
  const want = Buffer.from(expected, "utf8");
  return got.length === want.length && timingSafeEqual(got, want);
}

/* ---------------------------------- helpers ---------------------------------- */

async function readBody(req) {
  const chunks = [];
  let bytes = 0;
  for await (const chunk of req) {
    bytes += chunk.length;
    if (bytes <= MAX_BODY_BYTES) {
      chunks.push(chunk);
    }
  }
  const buffer = Buffer.concat(chunks);
  return { buffer, text: buffer.toString("utf8"), bytes, truncated: bytes > MAX_BODY_BYTES };
}

async function readJSON(req) {
  const { text } = await readBody(req);
  if (text === "") {
    return {};
  }
  try {
    return JSON.parse(text);
  } catch {
    return {};
  }
}

function flattenHeaders(headers) {
  const out = {};
  for (const [key, value] of Object.entries(headers)) {
    out[key] = Array.isArray(value) ? value.join(", ") : String(value ?? "");
  }
  return out;
}

function pickString(...candidates) {
  for (const candidate of candidates) {
    if (typeof candidate === "string" && candidate !== "") {
      return candidate;
    }
  }
  return "";
}

function sendJSON(res, status, payload) {
  const body = JSON.stringify(payload);
  res.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Content-Length": String(Buffer.byteLength(body)),
    "Cache-Control": "no-store",
  });
  res.end(body);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * The URL to paste into an endpoint. Binding to all interfaces means the sink is
 * reachable from a tunnel or another machine, so advertise the LAN address
 * instead of a loopback one that would not resolve from outside.
 */
function publicURL() {
  return `http://${describeHost(options.host)}:${options.port}/hooks/payconnect`;
}

function describeHost(host) {
  if (host === "0.0.0.0" || host === "::") {
    return lanAddress();
  }
  if (host === "::1") {
    return "[::1]";
  }
  return host;
}

function lanAddress() {
  for (const addresses of Object.values(networkInterfaces())) {
    for (const address of addresses ?? []) {
      if (address.family === "IPv4" && !address.internal) {
        return address.address;
      }
    }
  }
  return "127.0.0.1";
}

function parseArgs(args) {
  const options_ = {
    port: 4000,
    host: "127.0.0.1",
    secret: "",
    status: 200,
    body: '{"received":true}',
    delay: 0,
    help: false,
  };
  const value = (index, flag) => {
    const next = args[index + 1];
    if (next === undefined || next.startsWith("--")) {
      console.error(`${flag} needs a value.`);
      process.exit(1);
    }
    return next;
  };
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    switch (arg) {
      case "--port":
        options_.port = Number(value(i, arg));
        i++;
        break;
      case "--host":
        options_.host = value(i, arg);
        i++;
        break;
      case "--secret":
        options_.secret = value(i, arg);
        i++;
        break;
      case "--status":
        options_.status = Number(value(i, arg));
        i++;
        break;
      case "--body":
        options_.body = value(i, arg);
        i++;
        break;
      case "--delay":
        options_.delay = Number(value(i, arg));
        i++;
        break;
      case "-h":
      case "--help":
        options_.help = true;
        break;
      default:
        console.error(`Unknown option: ${arg}\n`);
        console.error(USAGE);
        process.exit(1);
    }
  }
  if (!Number.isInteger(options_.port) || options_.port < 1 || options_.port > 65535) {
    console.error("--port must be a port number.");
    process.exit(1);
  }
  if (!Number.isInteger(options_.status) || options_.status < 100 || options_.status > 599) {
    console.error("--status must be an HTTP status code.");
    process.exit(1);
  }
  return options_;
}
