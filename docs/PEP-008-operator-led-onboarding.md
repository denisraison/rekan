# PEP-008: Operator-Led Onboarding

**Status:** In Progress (Wave 1 done, Wave 2 pending)
**Date:** 2026-02-24
**Updated:** 2026-02-24

## Language requirement

All user-facing text (UI copy, error messages, validation messages, T&Cs, WhatsApp messages, button labels, status messages) must be in correct Brazilian Portuguese with proper accents and punctuation: á, é, ã, ç, ê, etc. No English, no missing diacritics. This applies to both frontend copy and backend error responses returned to the client.

## Context

The current architecture assumes two user types: self-serve clients (sign up, onboard themselves, generate content from a dashboard) and operator-managed clients (Elenice manages via the operator tool). In practice, every client will come through Elenice. She meets them, pitches the service, and manages their content through WhatsApp. The self-serve dashboard is dead weight that complicates the product without serving anyone.

This PEP replaces the self-serve flow with an operator-led onboarding model:

1. Elenice creates the client in the operator tool (business details, phone, client name, email)
2. She presses "Enviar Convite", Rekan sends the invite link to the client via WhatsApp
3. The client clicks the link, reviews and accepts Terms & Conditions, enters CPF/CNPJ
4. The client is sent to Asaas payment (R$19 first month)
5. Payment completes, client sees a confirmation with a WhatsApp link to start messaging
6. Done. Elenice manages everything from the operator tool. The client only uses WhatsApp.

No email infrastructure. The invite goes through WhatsApp using the existing whatsmeow integration. The conversation between Rekan and the client is already open (Elenice added their phone number), so the message arrives naturally in the same thread.

Clients never create a Rekan account. No login, no password, no Google OAuth for them. They interact with one link once (T&Cs + payment), then everything happens through WhatsApp with Elenice.

This is a pre-launch change. There is no production data, no existing subscriptions, no real users to migrate.

### What gets removed

- Self-serve dashboard (`/dashboard`)
- Client-facing login page (`/login`)
- Client-facing onboarding form (`/onboarding`)
- Google OAuth for clients (keep it for Elenice only)
- Trial generation limit logic (clients pay upfront, no free tier)
- `generations_used` field on users (stop reading/writing, keep the column to avoid a destructive migration)
- `POST /api/subscriptions` and `GET /api/subscriptions/current` endpoints (subscription created via invite flow now)

### What stays

- Marketing page (Elenice sends prospects there to see what Rekan is)
- Operator tool (enhanced with invite flow)
- WhatsApp integration (reused for invite delivery)
- Content generation and eval pipeline (unchanged)
- Asaas integration (modified: subscription created from invite accept endpoint, not from dashboard)

### Key architectural shift

Subscription tracking moves from the `users` collection to the `businesses` collection. Currently, `subscription_status` and `subscription_id` live on the user record because each client had their own account. Now all businesses belong to Elenice's account, so subscription state must live on the business itself.

### CPF/CNPJ requirement

Asaas requires CPF or CNPJ to create a customer for PIX payments. Individual MEIs may use either their personal CPF or their business CNPJ. The current `CreateCustomer` call only sends name and email, which works in sandbox but will fail in production. The invite accept endpoint collects CPF/CNPJ from the client in the form, passes it straight to the Asaas API, and does not store it in our database. Asaas retains it as the payment processor. This minimizes our LGPD surface.

### Cancellation and CDC Art. 49

Under the Código de Defesa do Consumidor, clients have 7 days to cancel a remote purchase with full refund (Art. 49). The operator tool needs a "Cancelar assinatura" action that calls the Asaas cancel API. The T&Cs must disclose the cancellation process (through Elenice or directly via Asaas customer portal).

## Consequences

- The product becomes strictly operator-first. No self-serve path exists. If we want self-serve later, it's a new PEP.
- Elenice is a single point of failure for client acquisition and onboarding. If she's unavailable, no new clients can be added. The data model supports adding more operators later (any authenticated user can own businesses), but that is out of scope for this PEP.
- Clients have no way to see their posts, billing status, or account details. All communication goes through Elenice via WhatsApp.
- The marketing page becomes a sales tool (Elenice shares the link), not a conversion funnel. Its CTAs should point to WhatsApp contact, not signup.
- CPF/CNPJ is collected in the invite form and passed straight to Asaas's `CreateCustomer` API. We do not store it in our database. Less PII stored means less LGPD liability.

---

## Wave 1: Data Model + Backend Endpoints -- DONE

Move subscription tracking to businesses, add invite infrastructure, create public endpoints for the invite flow. After this wave, the invite-accept-pay cycle works end-to-end via API calls.

All items implemented. Key files: `api/internal/asaas/client.go` (added cpfCnpj, Callback, GetSubscription, CancelSubscription), `api/migrations/1740000010_business_invite_fields.go`, `api/internal/http/handlers/invite.go` (InviteSend, InviteGet, InviteAccept, SubscriptionCancel), `api/internal/http/handlers/webhooks.go` (businesses instead of users), `api/internal/http/handlers/generate.go` (removed trial gate). Deleted `subscribe.go`. 35 handler tests pass.

**Deviation from original plan:** InviteAccept saves `terms_accepted_at` immediately but defers `invite_status: "accepted"` until after both Asaas calls succeed. This prevents a stuck state where a failed CreateCustomer leaves the business in `accepted` with no `subscription_id`, making retries impossible. See section 1.3 steps 6-9.

### 1.1 Migration: invite and subscription fields on businesses

New fields on the `businesses` collection:

| Field | Type | Purpose |
|-------|------|---------|
| `client_name` | text | Client's personal name (distinct from business name) |
| `client_email` | text | Client's email (required by Asaas for customer creation) |
| `invite_token` | text, unique | Unguessable token for the invite link |
| `invite_status` | select | `draft`, `invited`, `accepted`, `active`, `payment_failed`, `cancelled` |
| `invite_sent_at` | date | When the invite was sent (for expiry check) |
| `subscription_id` | text | Asaas subscription ID (moved from users) |
| `terms_accepted_at` | date | When the client accepted T&Cs |

The `invite_status` lifecycle:
- `draft`: Elenice is filling in details, hasn't sent the invite yet
- `invited`: invite link sent via WhatsApp, waiting for client
- `accepted`: client accepted T&Cs and completed payment setup, waiting for payment confirmation
- `active`: payment confirmed (via Asaas webhook)
- `payment_failed`: payment not confirmed within 48 hours of `accepted` (detected by webhook or operator visibility)
- `cancelled`: subscription cancelled (via Asaas webhook or operator action)

All businesses point to Elenice's user account. The unique index on `businesses.user` was already dropped in migration 1740000006.

The `subscription_status`, `subscription_id`, and `generations_used` fields on `users` become unused. Don't delete them (avoids a destructive migration), just stop reading/writing them. Also update the OAuth2 hook in `api/hooks.go` to stop setting `subscription_status = "trial"` and `generations_used = 0` on new users.

The `onboarding_step` field on businesses is also dead weight now. Stop setting it to `3` in the operator frontend on create. Don't remove the column.

New migration file: `api/migrations/1740000010_business_invite_fields.go`

**API rules for new fields:** The invite fields (`invite_token`, `invite_status`, `invite_sent_at`, `terms_accepted_at`, `subscription_id`) must be server-managed. Add `:isset = false` guards to the businesses updateRule so the frontend can't tamper with them. `client_name` and `client_email` are operator-editable.

### 1.2 Invite send endpoint

```
POST /api/businesses/{id}/invites:send
Auth: required (Elenice)
Response: { invite_url: string }
```

Flow:
1. Verify the authenticated user owns the business
2. Verify `phone` is set on the business (needed for WhatsApp delivery)
3. Verify `invite_status` allows sending: must be `draft`, `invited`, `payment_failed`, or `cancelled`. Reject `accepted` (payment in progress) and `active` (already subscribed).
4. Generate a crypto-random invite token (32 bytes, hex-encoded)
5. Build the WhatsApp message: "Oi {client_name}! Segue o link pra ativar seu acesso ao Rekan: {APP_URL}/convite/{token}". Construct `types.JID` from the business phone (E.164 format).
6. Send via `WhatsApp.SendMessage`. If send fails, return 502 with error. Do not update invite status on failure.
7. On successful send: save token to business, set `invite_status: "invited"`, set `invite_sent_at: now()`
8. Store the outgoing message in the `messages` collection with `direction: "outgoing"`, `type: "text"`, `business` relation set. This is a new pattern (existing message storage only handles incoming messages).
9. Return 200 with `invite_url` (so Elenice can also copy it manually if needed)

**Token expiry:** Tokens expire after 7 days. The accept endpoint checks `invite_sent_at` and rejects tokens older than 7 days with a friendly message ("Link expirado. Peça pra Elenice te enviar um novo."). Elenice can re-invite with one click.

New handler file: `api/internal/http/handlers/invite.go`

**Price constants:** Define `PriceFirstMonth = 19.00` and `PriceMonthly = 108.90` as package-level constants in the invite handler. The webhook handler imports them for the upgrade call. No config, no env vars.

### 1.3 Invite endpoints (public)

**Get invite info:**

```
GET /api/invites/{token}
Auth: none (public)
Response: { business_name, client_name, invite_status, price_first_month, price_monthly }
```

Returns the business name and client name so the invite page can greet the client. Also returns `invite_status` so the frontend can route correctly:
- `invited`: show the T&Cs + CPF/CNPJ form
- `accepted`: redirect to confirmation page (client already accepted, waiting for payment or already paid)
- `active`: show "already active" message with WhatsApp link
- Expired token (>7 days since `invite_sent_at`): return 410 Gone with a friendly message

**Accept invite:**

```
POST /api/invites/{token}/accept
Auth: none (public)
Body: { cpf_cnpj: string }
Response: { payment_url: string }
```

Flow:
1. Find business by `invite_token`
2. Check token expiry (reject if >7 days since `invite_sent_at`)
3. **Idempotency:** If `invite_status` is already `accepted` and `subscription_id` exists, call `asaas.GetSubscription(subscription_id)` and return its `paymentLink`. No new Asaas resources created.
4. If `invite_status` is `active`, return error "Assinatura já está ativa"
5. Verify `invite_status` is `invited`
6. Save `terms_accepted_at: now()` (status stays `invited` so retries work if Asaas fails)
7. Create Asaas customer with `client_name`, `client_email`, and `cpf_cnpj` from request body (passed through to Asaas, not stored in our DB). On failure, return 502. The business is still `invited`, so the client can retry.
8. Create Asaas subscription with:
   - `billingType: "PIX"`
   - `value: 19.00` (first month)
   - `nextDueDate: today`
   - `cycle: "MONTHLY"`
   - `description: "Rekan - Primeiro Mês"`
   - `externalReference: business.id`
   - `callback: { successUrl: "{APP_URL}/convite/{token}/confirmacao", autoRedirect: true }`
9. Both Asaas calls succeeded: set `invite_status: "accepted"` and save `subscription_id` on the business in one save. This avoids a stuck state where the business is `accepted` with no `subscription_id`.
10. Return `payment_url` (the `paymentLink` from Asaas response, which is the hosted page with the PIX QR code)

The confirmation page URL (`{APP_URL}/convite/{token}/confirmacao`) is built client-side after redirect, not returned by this endpoint.

**No `billing_type` parameter.** We create the subscription with `billingType: "PIX"`. The Asaas hosted payment page shows the PIX QR code. If we want to support credit card later, we change this field.

**Redirect after payment:** The subscription is created with a `callback` object:
```json
{
  "callback": {
    "successUrl": "{APP_URL}/convite/{token}/confirmacao",
    "autoRedirect": true
  }
}
```
After the client pays via PIX, Asaas auto-redirects them to our confirmation page. The `successUrl` domain must match the domain registered in Asaas account settings (Account Settings > Information > Commercial data).

### 1.4 Update webhook handler

The webhook currently finds users by `subscription_id`. Change it to find businesses by `subscription_id` instead.

Changes to `api/internal/http/handlers/webhooks.go`:
- Query `businesses` collection instead of `users` collection: `dbx.HashExp{"subscription_id": subscriptionID}`
- Update `invite_status` on the business instead of `subscription_status` on the user
- `PAYMENT_CONFIRMED`: set `invite_status: "active"`
- `PAYMENT_OVERDUE`: keep `invite_status` as-is (Asaas handles dunning, don't downgrade an active client)
- `SUBSCRIPTION_DELETED`: set `invite_status: "cancelled"`
- **First payment upgrade:** if `invite_status` was `accepted` and new status is `active`, call `UpdateSubscription` to change price from R$19 to R$108.90. The `UpdateSubscription` method already exists in the Asaas client. If the upgrade call fails, log the error. Elenice will see the client is active but the price needs manual correction in Asaas. At 29 clients, this is a manageable edge case.

### 1.5 Update generate handler

The generate handler (`api/internal/http/handlers/generate.go`) currently checks `subscription_status` on the user record and enforces the 3-generation trial limit.

Changes:
- Remove the trial generation limit check entirely (clients pay upfront)
- The operator generate handler (`operator.go`) already has no subscription check, no changes needed
- Remove the self-serve generate endpoint and `POST /api/subscriptions`, `GET /api/subscriptions/current` routes (dead code after self-serve removal). Handler code lives in `api/internal/http/handlers/subscribe.go` (delete file). Route registration in `api/internal/http/routes.go`.

### 1.6 Asaas client updates

The existing `asaas.Client` needs several changes to support the invite flow. All changes in `api/internal/asaas/client.go`.

**a) `CreateCustomer`: add `cpfCnpj` parameter**

```go
func (c *Client) CreateCustomer(ctx context.Context, name, email, cpfCnpj string) (Customer, error)
```

Sends `"cpfCnpj"` in the request body. Asaas requires this for PIX payments in production (sandbox is lenient). The existing `subscribe.go` caller is being removed, so this is a clean change.

**b) `CreateSubscriptionReq`: add `callback` and `externalReference` fields**

```go
type Callback struct {
    SuccessURL   string `json:"successUrl"`
    AutoRedirect bool   `json:"autoRedirect"`
}

type CreateSubscriptionReq struct {
    Customer          string   `json:"customer"`
    BillingType       string   `json:"billingType"`
    Value             float64  `json:"value"`
    NextDueDate       string   `json:"nextDueDate"`
    Cycle             string   `json:"cycle"`
    Description       string   `json:"description"`
    ExternalReference string   `json:"externalReference,omitempty"`
    Callback          *Callback `json:"callback,omitempty"`
}
```

The `callback.successUrl` redirects the client back to our confirmation page after PIX payment. The `autoRedirect: true` makes this automatic (no "go to site" button). The `externalReference` stores the business ID for traceability.

Important: `successUrl` must be on the same domain registered in Asaas account settings.

**c) `GetSubscription`: fetch existing subscription for idempotent retries**

```go
type Subscription struct {
    ID          string `json:"id"`
    PaymentLink string `json:"paymentLink"`
}

func (c *Client) GetSubscription(ctx context.Context, id string) (Subscription, error)
```

Used by the idempotent accept endpoint: if a `subscription_id` already exists on the business, fetch the subscription to get the `paymentLink` instead of creating a duplicate.

**d) `CancelSubscription`: cancel a subscription**

```go
func (c *Client) CancelSubscription(ctx context.Context, id string) error
```

Calls `DELETE /v3/subscriptions/{id}`. Used by the operator cancel action. Asaas will also fire a `SUBSCRIPTION_DELETED` webhook.

### How PIX subscriptions work (reference)

Understanding the Asaas PIX subscription flow is critical for testing:

1. `POST /v3/subscriptions` with `billingType: "PIX"` creates the subscription
2. Asaas auto-generates the first charge and fires a `PAYMENT_CREATED` webhook
3. The response includes `paymentLink`, a hosted Asaas page showing the PIX QR code
4. Client opens `paymentLink`, scans the QR code with their bank app, pays
5. Asaas fires `PAYMENT_CONFIRMED` webhook (seconds for PIX)
6. If `callback.autoRedirect` is true, client is redirected to `successUrl`
7. Each month, Asaas generates a new charge (new PIX QR code) and sends notifications to the client
8. If the client doesn't pay, Asaas fires `PAYMENT_OVERDUE`

PIX QR codes are dynamic, expire 12 months after due date, and can only be paid once. The Asaas account must have a registered PIX key.

For recurring months, the client receives Asaas notifications (email/SMS) with a new payment link. This is Asaas's standard subscription behavior. No action needed from us or Elenice, though she can nudge inactive payers.

### Asaas account setup checklist (pre-launch)

- [ ] Register a PIX key on the Asaas account
- [ ] Configure the production domain in Account Settings > Information > Commercial data (must match `successUrl`)
- [ ] Set `ASAAS_WEBHOOK_TOKEN` and configure webhook URL in Asaas pointing to `POST /api/webhooks/asaas`
- [ ] Verify `ASAAS_API_KEY` is the production key (not sandbox)
- [ ] Test the full PIX flow in sandbox first (sandbox has a "Confirm Payment" button for simulating PIX)

### 1.7 Cancel subscription endpoint

```
POST /api/businesses/{id}/subscription:cancel
Auth: required (Elenice)
```

Flow:
1. Verify the authenticated user owns the business
2. Verify `subscription_id` exists and `invite_status` is `active`
3. Call Asaas delete subscription API (new method: `asaas.CancelSubscription(ctx, id)`)
4. Set `invite_status: "cancelled"` on the business
5. Return 200

This gives Elenice a cancel button in the operator tool. The Asaas webhook for `SUBSCRIPTION_DELETED` will also fire, but the status is already `cancelled` so it's a no-op.

### Gate

- `go test ./api/...` passes
- New handler tests (all using mock Asaas httptest server):
  - Invite send: requires phone, generates token, sets status to `invited`, stores `invite_sent_at`, sends WhatsApp message
  - Invite send re-invite: regenerates token for `invited`, `payment_failed`, or `cancelled` businesses
  - Invite send rejects re-invite for `accepted` or `active` businesses
  - Invite get: returns business info and `invite_status`, returns 410 for expired tokens
  - Invite accept: validates token, checks expiry, creates Asaas customer with CPF/CNPJ, creates subscription with `callback.successUrl` and `externalReference`, returns `paymentLink`
  - Invite accept idempotent: if already `accepted` with `subscription_id`, calls `GetSubscription` and returns existing `paymentLink`
  - Invite accept expired: rejects tokens older than 7 days
  - Webhook: finds business (not user) by `subscription_id`, updates `invite_status`, upgrades price on first payment
  - Cancel: calls Asaas DELETE, sets status to `cancelled`
- Asaas client unit tests:
  - `CreateCustomer` sends `cpfCnpj` in request body
  - `CreateSubscription` sends `callback` and `externalReference`
  - `GetSubscription` fetches by ID, returns `paymentLink`
  - `CancelSubscription` sends DELETE request
- Manual sandbox test: full flow end-to-end. Create subscription with `billingType: "PIX"`, use sandbox "Confirm Payment" button to simulate PIX, verify `PAYMENT_CONFIRMED` webhook fires, verify redirect to confirmation page, verify price upgrade from R$19 to R$108.90

---

## Wave 2: Frontend (Operator Flow + Invite Pages)

Remove self-serve routes, enhance operator client creation with invite flow, add public invite pages. After this wave, the full onboarding cycle works from browser.

### 2.1 Remove self-serve routes

Delete:
- `web/src/routes/(app)/dashboard/` (entire directory)
- `web/src/routes/(app)/onboarding/` (entire directory)
- `web/src/routes/login/` (entire directory)

Keep:
- `web/src/routes/(app)/operador/` (the operator tool)
- `web/src/routes/(app)/+layout.ts` (SSR disabled, still needed for operator)
- `web/src/routes/(marketing)/` (the landing page)

The `(app)/+layout.svelte` currently redirects unauthenticated users to `/login` via `goto('/login')`. After deleting `/login`, change the layout to show a Google login button inline instead of redirecting. Elenice navigates to `/operador` directly, sees the login button if not authenticated, logs in, and lands on the operator tool. No separate login page needed.

### 2.2 Update marketing page CTAs

The marketing page currently links CTAs to `/entrar` (which is a 404). Change all CTAs to point to Elenice's WhatsApp number with a pre-filled message:

```
https://wa.me/55XXXXXXXXXXX?text=Oi,%20quero%20saber%20mais%20sobre%20o%20Rekan
```

This turns the marketing page into a sales funnel that ends in a WhatsApp conversation with Elenice, which is exactly the acquisition model.

Also add `og:image` and `og:description` meta tags so the page renders a good preview when shared via WhatsApp (Elenice's primary distribution channel). The `og:image` needs a static image at a public URL (1200x630px recommended for WhatsApp previews). Place it in `web/static/og-image.png`.

### 2.3 Enhance operator client creation form

The current form collects: name, type, city, state, phone, services, target_audience, brand_vibe, quirks.

Add two fields at the top:
- `client_name` (text, required): "Nome do cliente" (the person, not the business)
- `client_email` (email, required): "Email do cliente" (needed for Asaas customer creation)

Change the save button behavior:
- "Salvar" creates/updates the business as before (sets `invite_status: "draft"` on create)
- New "Salvar e Enviar Convite" button: saves, then calls `POST /api/businesses/{id}/invites:send`, which sends the invite link to the client via WhatsApp
- After invite is sent, show the invite URL with a "Copiar link" button (fallback if WhatsApp delivery fails)

Show invite status badge on each client in the sidebar:
- `draft`: gray
- `invited`: yellow
- `accepted`: blue (payment in progress)
- `active`: green
- `payment_failed`: red with warning icon
- `cancelled`: red strikethrough

For clients with `invite_status: "accepted"` for more than 48 hours, show a warning indicator so Elenice knows payment may have failed and she should follow up.

Add a "Cancelar assinatura" button visible when `invite_status` is `active`. Calls `POST /api/businesses/{id}/subscription:cancel` with a confirmation dialog.

### 2.4 Public invite page

New route: `web/src/routes/convite/[token]/+page.svelte`

This is a public page (no auth, no app layout). Clean, branded design (Rekan logo, coral accent color).

Flow:
1. On mount, call `GET /api/invites/{token}` to load business name, client name, and invite status
2. **Route by status:**
   - `invited`: show the T&Cs + payment form (steps 3-7 below)
   - `accepted`: redirect to `/convite/{token}/confirmacao` (already accepted, waiting for payment)
   - `active`: show "Sua conta já está ativa!" with WhatsApp link
   - 410 Gone (expired): show "Link expirado. Peça pra Elenice te enviar um novo." with WhatsApp link
   - 404 (invalid token): show "Link inválido."
3. Show greeting: "Oi {client_name}! Confirme seus dados pra começar a usar o Rekan."
4. Show Terms & Conditions (inline expandable section). Content covers: service description (AI-generated Instagram content managed by Elenice), monthly price (R$19 first month, R$108,90/month after), cancellation policy (contact Elenice or cancel within 7 days for full refund per CDC Art. 49), data usage (business info used for content generation, CPF/CNPJ sent to Asaas for payment processing only, not stored by Rekan), LGPD data controller identification. The T&Cs text itself must be in correct pt-BR with proper legal phrasing.
5. CPF/CNPJ input field. Auto-detect by length: 11 digits = CPF (mask `XXX.XXX.XXX-XX`), 14 digits = CNPJ (mask `XX.XXX.XXX/XXXX-XX`). Validate checksum on the frontend before submit. Show clear error on invalid input.
6. Checkbox: "Li e aceito os Termos de Uso"
7. "Aceitar e continuar" button calls `POST /api/invites/{token}/accept`. On success, redirect to Asaas payment URL. On Asaas error (e.g. CPF/CNPJ rejected), show pt-BR error message: "Não foi possível processar. Confira seu CPF/CNPJ e tente novamente."

### 2.5 Confirmation page

New route: `web/src/routes/convite/[token]/confirmacao/+page.svelte`

This page is where the client lands after completing payment on Asaas (via browser back or Asaas redirect). It polls for payment confirmation.

States:
1. **Waiting for payment** (`invite_status: "accepted"`): show "Aguardando confirmação do pagamento..." with a subtle spinner. Poll `GET /api/invites/{token}` every 5 seconds. Stop polling after 10 minutes and show: "Ainda não recebemos o pagamento. Se você já pagou, aguarde alguns minutos ou fale com a Elenice." with a WhatsApp link.
2. **Payment confirmed** (`invite_status: "active"`): show:
   - Rekan logo
   - "Tudo certo, {client_name}!"
   - "Sua assinatura está ativa. A Elenice vai enviar seu conteúdo pelo WhatsApp. É só mandar suas fotos e novidades que a gente cuida do resto."
   - WhatsApp link to the Rekan number: "Mandar mensagem pelo WhatsApp"
3. **Error states**: if the token is expired or invalid, show appropriate message with WhatsApp link to Elenice.

No "Ativar minha conta" button. There is no account to activate. The message is clear: you're set, use WhatsApp.

### Gate

- Playwright tests: invite page loads with business info, CPF/CNPJ validation works, expired token shows friendly message, confirmation page renders both waiting and confirmed states
- Manual: full flow from operator creating client, invite sent via WhatsApp, client clicks link, accepts T&Cs, pays, sees confirmation
- Marketing page CTAs point to WhatsApp, og:image renders in WhatsApp link preview
- Operator sidebar shows invite status badges with correct colors
- Cancel button works and updates status
