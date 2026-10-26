# PayConnect — launch copy kit

Everything below is written against the product as it actually ships: Paystack,
Flutterwave, Monnify, Stripe and PayPal in one dashboard, provider keys sealed in
an AES-256 vault, one `vpay_test_… / vpay_live_…` API key per application, signed
webhooks with retries, and an audit trail. PayConnect never holds your money and
never takes a cut of a payment.

Live links to use in every post:

- App: `https://payconnect.kuipid.com`
- Docs: `https://payconnect.kuipid.com/docs`
- API base (technical posts only): `https://apipayment.kuipid.com/v1`

---

## 1. Positioning blocks

**One-liner (7 words)**
> Every payment provider. One secure connection.

**Tagline (12 words)**
> Connect Paystack, Flutterwave, Monnify, Stripe or PayPal once — developers build without your keys.

**25 words**
> PayConnect is the secure layer between your payment providers and your developers. Connect a provider once, hand out a scoped API key, and your secret keys never leave the vault.

**50 words**
> Your business already uses Paystack, Flutterwave, Monnify, Stripe or PayPal. PayConnect connects them once, encrypts every secret in a vault, and gives your developer a single API key to build with. You keep control of the money and the keys. They get to ship without ever seeing either.

**100 words**
> Every provider you add today means another set of secret keys, another integration, and another screenshot pasted into a WhatsApp group. PayConnect replaces that with one secure connection: connect Paystack, Flutterwave, Monnify, Stripe or PayPal from your dashboard, and each key is encrypted the moment it is saved. Your developer gets a scoped API key and one payment endpoint. Webhooks arrive normalised and signed, every attempt is logged, and the audit trail tells you who used which key and when. You never hand over a provider secret, and you never lose sight of a transaction.

**One sentence for a business card / bio**
> The secure payment layer between your providers and your developers.

---

## 2. WhatsApp

### 2.1 Broadcast (primary)

> *Introducing PayConnect* 🚀
>
> One secure connection to every payment provider you use — Paystack, Flutterwave, Monnify, Stripe and PayPal.
>
> Here is the problem we kept seeing: to go live, someone has to paste their *secret* Paystack key into a developer's laptop, a shared note, or a group chat. That key can move money. It should never leave a vault.
>
> PayConnect fixes that:
>
> ✅ Connect your provider *once* — the key is encrypted immediately and only its last 4 characters are ever shown again
> ✅ Give your developer a scoped API key instead of your provider secret
> ✅ One integration for every provider, so switching or adding one is not another project
> ✅ Webhooks arrive signed, retried and logged — no more "did the payment land?"
> ✅ Every action recorded: who connected what, who used which key, when
>
> No card to start. Test mode is free forever. First test payment in about 15 minutes.
>
> 👉 payconnect.kuipid.com
>
> _If you run a store, a SaaS, or a dev team that keeps asking you for keys — this is for you._

### 2.2 Short forward (for groups)

> Payment providers sorted in one place 🔐
> Connect Paystack / Flutterwave / Monnify / Stripe / PayPal once, give your dev a single API key, keep the real keys in a vault.
> Test mode free, no card. 👉 payconnect.kuipid.com

### 2.3 Status / one-liner

> Your provider keys should never be in a chat. PayConnect keeps them in a vault and gives your developer an API key instead. payconnect.kuipid.com

### 2.4 Voice-note script (about 40 seconds, spoken)

> "So this is PayConnect. If you take payments online, you already have keys from Paystack, Flutterwave, Monnify, Stripe or PayPal — and at some point you have had to send one of those keys to a developer. That key can move money out of your account.
>
> PayConnect lets you connect the provider once, inside your own dashboard. The key is encrypted the second you save it, and after that nobody — not even you — sees it again. Your developer gets their own key, limited to the app they are building. They call one endpoint, and PayConnect talks to whichever provider you chose. Payments arrive with signatures checked, webhooks retried if they fail, and a log of everything that happened.
>
> You can start in test mode free, no card, and have your first test payment running in about fifteen minutes. Link is payconnect dot kuipid dot com."

---

## 3. Video

### 3.1 45-second intro (16:9, voiceover + captions)

| # | Time | On screen | Voiceover |
|---|---|---|---|
| 1 | 0–4s | Phone screen: a `sk_live_…` key pasted into a WhatsApp chat | "This is a payment secret key. It can move every naira your business earns." |
| 2 | 4–8s | The same key, forwarded again, screenshotted, dropped into a notes app | "By lunchtime, it lives in four places you cannot control." |
| 3 | 8–13s | Cut to the PayConnect landing hero, then the Providers page with the five logos | "PayConnect connects your providers once — Paystack, Flutterwave, Monnify, Stripe, PayPal." |
| 4 | 13–19s | Connect wizard: pasting a key, the "Connected" state, then the masked key `sk_live_••••9d21` | "The key is encrypted the moment you save it. After that, only the last four characters are ever shown." |
| 5 | 19–25s | API keys screen: `vpay_test_…` created, permissions list | "Your developer gets their own key, scoped to their app. They never touch yours." |
| 6 | 25–31s | Docs quickstart: one `POST /v1/payments` call; then a payment row appearing in the dashboard | "One endpoint. One integration. Whether you are on Paystack today or PayPal next year." |
| 7 | 31–37s | Webhooks screen: a delivery attempt, a retry, a 200; then audit log rows | "Webhooks arrive signed, retried and logged. The audit trail shows who did what, and when." |
| 8 | 37–42s | Payments list, provider health, a successful ₦ amount | "Your money still moves through the providers you already trust. PayConnect never holds it and never takes a cut." |
| 9 | 42–45s | Logo + URL | "PayConnect. Every provider. One secure connection." |

**On-screen text (no voiceover needed, for muted autoplay):**
Key leaked → 4 places. / Connect once. Encrypted. / Developer gets a scoped key. / One API, every provider. / Signed webhooks, full audit trail. / payconnect.kuipid.com

### 3.2 15-second vertical cut (Reels / TikTok / Status)

1. **0–2s** — Text on a plain background: "Where is your Paystack secret key right now?"
2. **2–6s** — Screen recording: WhatsApp forward → notes app → laptop. Text: "Be honest."
3. **6–11s** — Screen recording: Providers → Connect → key masked `sk_live_••••9d21`. Text: "Connect once. Encrypted. Never shown again."
4. **11–13s** — API keys screen: `vpay_test_…` copied. Text: "Your dev gets this instead."
5. **13–15s** — Logo over the dashboard. Text: "payconnect.kuipid.com — free to start."

### 3.3 6-second bumper

Visual: a key morphing into a locked tile. Text: **"Connect once. Never paste a secret key again."** End card: logo + URL.

---

## 4. Platform posts

### 4.1 Instagram (feed)

> We asked five merchants where their Paystack secret key lives.
>
> One kept it in a notebook. One in a WhatsApp group with their developer. One in a Google Doc called "keys lol". One had it in three old repos. One couldn't remember.
>
> Every one of those keys can move money.
>
> PayConnect is the boring fix: connect your provider once, the key gets encrypted in a vault, and your developer receives a scoped API key instead. They build. You keep control.
>
> Paystack. Flutterwave. Monnify. Stripe. PayPal. One dashboard, one integration, one audit trail.
>
> Test mode is free, no card, and your first test payment takes about 15 minutes.
>
> Link in bio → payconnect.kuipid.com
>
> #PayConnect #Paystack #Flutterwave #Fintech #Payments #NigeriaTech #BuildInLagos #SaaS #Developers #PaymentInfrastructure

**Carousel (5 slides)**
1. "Where is your provider secret key right now?" (plain, black on off-white)
2. "It can move every naira you earn." (key blurred in background)
3. Screenshot: Providers page with 5 logos — "Connect once."
4. Screenshot: masked key + API key creation — "They get a key. Not your keys."
5. Logo + "payconnect.kuipid.com — free to start."

### 4.2 LinkedIn

> Most payment problems are not payment problems. They are credential problems.
>
> A merchant signs up with Paystack. Someone builds the checkout. The quickest way to make it work is to paste the secret key into the project, into a staging `.env`, and into the chat where the developer asked for it. It works. Then the same thing happens with Flutterwave for a second corridor, and Monnify for a client who insists on it, and now three teams hold keys to three accounts and nobody can say which key touched which payment.
>
> PayConnect is the layer we built for that. The merchant connects a provider once in the dashboard; the key is encrypted in a vault and only its last four characters are ever rendered again. The developer gets a scoped API key tied to a specific application, calls one payments endpoint, and receives normalised, signed webhooks with retries and a delivery log. Adding a provider becomes a dashboard action rather than a sprint.
>
> Deliberate design decisions: PayConnect never holds merchant funds and never takes a cut of a payment — the money still moves through Paystack, Flutterwave, Monnify, Stripe or PayPal. Credentials are never exposed to developer accounts, not even read-only. Every connection, key and payout action is written to an audit trail.
>
> Test mode is free and card-free. If your team is still passing provider keys around, it is worth 15 minutes: https://payconnect.kuipid.com

### 4.3 X / Twitter

Single post:
> Your Paystack secret key is probably in a WhatsApp chat, a Google Doc and an old repo right now.
>
> PayConnect: connect the provider once, encrypt the key, give your dev a scoped API key instead. Paystack · Flutterwave · Monnify · Stripe · PayPal.
>
> payconnect.kuipid.com

Thread:
1. Every merchant has the same three keys problem: too many providers, one developer laptop, zero audit trail. We built a layer for it. 🧵
2. Connect Paystack / Flutterwave / Monnify / Stripe / PayPal once. The key is encrypted on save — you only ever see `sk_live_••••9d21` again.
3. Your developer gets `vpay_test_…` or `vpay_live_…` bound to one application, with scoped permissions. They call `POST /v1/payments`. That is the whole integration.
4. Webhooks arrive as one normalised event body, signed, retried on failure, every attempt logged. No more "customer says it failed".
5. We don't hold funds and we don't take a cut. Payments still run through the provider you chose. We remove the key-sharing.
6. Free test mode, no card. First test payment in ~15 min → payconnect.kuipid.com

### 4.4 Facebook

> Taking payments online should not mean passing secret keys around a WhatsApp group.
>
> PayConnect connects the providers you already use — Paystack, Flutterwave, Monnify, Stripe and PayPal — in one secure dashboard. Your provider keys are encrypted the moment you add them, and your developer works with a separate key that only reaches the app it was created for.
>
> You see every payment, every provider connection and every change in one place. You do not hand over keys, and you do not build the same integration five times.
>
> Free test mode to try it. 👉 payconnect.kuipid.com

### 4.5 TikTok / Shorts caption

> POV: your developer asks for the Paystack secret key in the group chat 😭
> There is a better way — connect once, encrypt it, hand out a scoped key.
> payconnect.kuipid.com
> #payments #paystack #flutterwave #fintech #devtok #nigeria

### 4.6 Product Hunt / directory listing

- **Tagline (60 chars):** The secure payment layer between providers and devs
- **Description:** PayConnect connects Paystack, Flutterwave, Monnify, Stripe and PayPal in one dashboard, keeps every provider key encrypted in a vault, and gives developers a scoped API key plus one payments endpoint with signed, retried webhooks and a full audit trail. It never holds funds and never takes a cut.
- **First comment:** We built this after watching a merchant paste a live Paystack key into three different chat threads in one week. The fix is not more policy, it is a smaller blast radius: the merchant connects once, the developer gets a key that only opens their own app, and every action is logged. Happy to answer anything about how the vault, routing or webhook retries work.

### 4.7 Email / newsletter blurb

**Subject:** Your provider keys should not be in a chat
**Preheader:** Connect once, encrypt once, hand out a scoped key.

> Hi {{first_name}},
>
> If you take payments online, you have at least one provider — and a secret key somewhere you would rather not think about.
>
> PayConnect connects Paystack, Flutterwave, Monnify, Stripe and PayPal once, encrypts each key in a vault, and gives your developer their own scoped API key. One endpoint, signed webhooks, retries, audit trail.
>
> Test mode is free and needs no card. 15 minutes gets you a first test payment.
>
> [Start free →](https://payconnect.kuipid.com)

### 4.8 SMS / very short

> PayConnect: connect Paystack/Flutterwave/Monnify/Stripe/PayPal once, keep keys encrypted, give devs a scoped key. Free test mode: payconnect.kuipid.com

---

## 5. Positives — what actually makes it better

Use these as captions, bullets or the "why" column of a comparison table.

1. **Secret keys stop travelling.** A provider key is added once, encrypted at rest, and after that only its last four characters can ever be rendered.
2. **Developers are not trusted with the money keys.** They get a key bound to one application with explicit scopes — payments, webhooks, logs — and never see a provider credential.
3. **One integration, five providers.** Build against one endpoint; adding Flutterwave or PayPal later is a dashboard step, not a rewrite.
4. **Every payment in one place.** One list across providers, with status, amount, reference and the provider that processed it.
5. **Webhooks you can trust.** One normalised event body, signatures verified, failed deliveries retried, and every attempt logged.
6. **Test and live are separated properly.** Test keys never touch live money, and each environment has its own connections and data.
7. **Nothing is invisible.** Connections, key creation, developer access and payment events land in an audit trail.
8. **You keep your money and your providers.** PayConnect does not hold funds and does not take a percentage of a payment.
9. **Onboarding is minutes, not a project.** Test mode is free, no card, and there is a guided connect flow per provider plus a sandbox arena for trying calls.
10. **Built for the team you actually have.** Owner, developer and merchant roles, invitations with scoped permissions, and documentation your developer can follow alone.

---

## 6. How to use it

### For the business owner (about 10 minutes)

1. **Create your account** at [payconnect.kuipid.com/register](https://payconnect.kuipid.com/register) — name, work email, password (entered twice), organization name.
2. **Confirm your email** — the verification link lands in your inbox; clicking it activates the workspace.
3. **Connect a provider** — Providers → Connect → pick Paystack, Flutterwave, Monnify, Stripe or PayPal → paste the *test* secret key first. The key is validated against the provider, then encrypted.
4. **Create an application** — Applications → New. This is the thing your developer will call; the name is what shows in the key label.
5. **Issue an API key** — copy the `vpay_test_…` key once. It is shown a single time; store it in your developer's own secret manager.
6. **Invite your developer** — Developers → Invite, choose their permissions. They get dashboard access to payments and logs, and never to provider keys.
7. **Switch to live when ready** — add the live key for the same provider and create a `vpay_live_…` key. Test and live stay apart.

### For the developer (about 15 minutes)

1. **Check your invite** and sign in.
2. **Read the quickstart** — [payconnect.kuipid.com/docs/quickstart](https://payconnect.kuipid.com/docs/quickstart).
3. **Create a payment:**
   ```bash
   curl -X POST https://apipayment.kuipid.com/v1/payments \
     -H "Authorization: Bearer vpay_test_…" \
     -H "Content-Type: application/json" \
     -d '{
       "amount_minor": 20000,
       "currency": "NGN",
       "reference": "order_1042",
       "customer": { "email": "buyer@example.com", "name": "Ada" },
       "adapter": "paystack"
     }'
   ```
   Drop `adapter` to let routing pick the provider, or name one (`paystack`, `flutterwave`, `monnify`, `stripe`, `paypal`) to pin the charge.
4. **Register a webhook endpoint** — you receive one event shape regardless of provider, with a signature header to verify and automatic retries on failure.
5. **Watch it happen** — the payment appears in Payments, the delivery in Webhooks, and every call in Logs. The test arena lets you fire events without waiting for a real customer.

### What to say when someone asks "what does it cost?"

> Test mode is free and needs no card. Paid plans are Lite and Pro, and the Pro trial is card-free. PayConnect never takes a cut of a payment — your provider fees stay exactly what they are today.

---

## 7. Image snippets

Capture these from the live app (1440px wide for web, 9:16 for Stories/Reels, 1:1 for feed). Use real data or seeded demo data, never a blurred-out fake dashboard.

| # | Shot | Where | Pairs with | Crop |
|---|---|---|---|---|
| 1 | Landing hero: "Every provider. One secure connection." | `/` | Instagram slide 1, LinkedIn header, WhatsApp broadcast | 16:9 + 1:1 |
| 2 | Five provider logos in one row (Paystack, Flutterwave, Monnify, Stripe, PayPal) | `/providers` | "Connect once" — the clearest proof of breadth | 3:2 |
| 3 | Connect wizard, key field being filled | `/providers/connect/paystack` | "Where is your key right now?" | 4:5 |
| 4 | Connected state with masked key `sk_live_••••9d21` | `/providers/[id]` | "Encrypted. Never shown again." | 4:5 |
| 5 | API key creation with `vpay_test_…` and permission list | `/api-keys` | "Your dev gets this instead." | 1:1 |
| 6 | Payments list, several providers, mixed statuses | `/payments` | "Every payment in one place." | 16:9 |
| 7 | Single payment detail: amount, provider, reference, timeline | `/payments/[id]` | "Know what happened, and when." | 4:5 |
| 8 | Webhook delivery log: attempts, response codes, retry | `/webhooks/[id]` | "Signed, retried, logged." | 16:9 |
| 9 | Test arena firing a test event | `/test-arena` | "Try it before you ship it." | 9:16 |
| 10 | Audit log rows: provider connected, key created, developer invited | `/settings/audit-logs` | "Nothing is invisible." | 16:9 |
| 11 | Docs quickstart with the `POST /v1/payments` snippet | `/docs/quickstart` | Developer-audience posts only | 4:5 |
| 12 | Phone view of the dashboard (mobile shell) | `/dashboard` at 390px | Stories, TikTok end card | 9:16 |

**Rules for the snippets**

- Blur or replace anything resembling a real customer name, email or phone number.
- Keep one masked key visible in most shots — `sk_live_••••9d21` is the whole story in one image.
- Never show an unmasked key, even a test one, in a published asset. A screenshot is a leak.
- Prefer tight crops on real UI over illustration; the product is the argument.
- Add a 1px slate border to white-background screenshots so they do not disappear into the feed.

---

## 8. Copy-paste furniture

**Link block**
```
App      payconnect.kuipid.com
Docs     payconnect.kuipid.com/docs
```

**Boilerplate (About, for directories and press)**
> PayConnect is a payment infrastructure layer for merchants and the developers who build for them. It connects Paystack, Flutterwave, Monnify, Stripe and PayPal in one dashboard, keeps provider credentials encrypted in a vault, and exposes a single payments API with signed webhooks, retries and an audit trail. PayConnect never holds merchant funds and never takes a cut of a payment.

**Hashtag sets**
- Nigerian/business: `#PayConnect #Paystack #Flutterwave #Monnify #FintechNigeria #NigeriaTech #LagosTech #Payments`
- Developer: `#DevTools #API #Webhooks #SaaS #BuildInPublic #PaymentInfrastructure`
- Short/clean: `#PayConnect #Payments #Fintech`

**Alt text**
- Providers grid: "PayConnect providers page showing connected Paystack, Flutterwave, Monnify, Stripe and PayPal accounts."
- Masked key: "A provider secret key stored in PayConnect, displayed only as sk_live with the last four characters visible."
- API key: "An application API key in PayConnect starting with vpay_test, with payments and webhooks permissions listed."
- Webhook log: "Webhook delivery log in PayConnect showing a signed event, one successful attempt and its response code."

---

## 9. Claims: keep and avoid

**Safe to say** — encrypted at rest (AES-256), keys never exposed to developers, five providers supported, normalised signed webhooks with retries, full audit trail, free card-free test mode, never holds funds, never takes a cut.

**Do not say** — "bank-grade security", "PCI-certified", "99.99% uptime", specific provider-fee discounts, "instant settlements", or anything implying PayConnect is the merchant of record. None of those are true today, and in payments an over-claim is the fastest way to lose a merchant's trust.
