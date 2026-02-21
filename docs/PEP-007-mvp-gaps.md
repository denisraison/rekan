# PEP-007: MVP Gaps

**Status:** In Progress
**Date:** 2026-02-21
**Updated:** 2026-02-22

## Context

Post-commit review of the full MVP surface (PEP-001 through PEP-006) plus a product review identified gaps across three areas: untested business logic, missing operator tooling that will bottleneck Elenice, and product gaps that weaken the value proposition for MEIs. This PEP tracks all of them organized into implementation waves.

The product review surfaced that the current MVP solves "content creation" but not the real problem: **consistency**. The operator workflow has too much friction for Elenice to sustain at 15+ clients, and several low-effort changes would dramatically strengthen the offering.

The biggest architectural change: instead of Elenice manually copying messages between WhatsApp and the operator page, we connect whatsmeow directly so messages (text, voice notes, images) arrive in the operator page automatically and replies go back through WhatsApp. This collapses what was originally planned as separate "operator workflow" and "voice notes" waves into one integration.

### Language

All user-facing text (UI copy, error messages, validation messages, nudge templates, summary messages, WhatsApp replies) must be in natural Brazilian Portuguese (pt-BR). No formal/stiff phrasing. Write the way a real person in Brazil would talk to a client or colleague.

### Ban risk acknowledgement

whatsmeow uses the unofficial WhatsApp Web multi-device protocol. This violates WhatsApp ToS. Account bans are possible. Our risk profile is lower than spam bots (clients message first, low volume, 20-40 clients), but the risk is real. Mitigations: send typing indicators before replies, randomize small delays, never initiate to unknown numbers, keep the official WhatsApp Business app as fallback on a second number.

---

## Wave 1: WhatsApp Integration (the operator backbone) -- DONE

Replace the manual copy-paste workflow with a live WhatsApp connection. Messages arrive in the operator page automatically. Elenice generates content and sends it back without leaving the browser.

All 9 sub-items implemented in a single pass. Key files: `api/internal/whatsapp/` (client + handler), `api/internal/transcribe/whisper.go`, `api/internal/http/handlers/send_message.go`, `api/internal/http/handlers/whatsapp.go`, migrations 1740000007-1740000009, operator page overhaul.

### 1.1 whatsmeow connection and session management

**Current state:** No WhatsApp integration. Elenice alt-tabs between WhatsApp and the operator page for every interaction.

**Change:** Add whatsmeow as a dependency. Run it as a goroutine inside the PocketBase process. Persist the session in SQLite (separate file from PocketBase's DB). On first run, show a QR code in the operator page that Elenice scans with the Rekan WhatsApp Business number. Subsequent restarts reconnect automatically.

**Files:** `api/go.mod`, new package `api/internal/whatsapp/` (client setup, event handler, session store)

- [x] whatsmeow added to `go.mod`
- [x] WhatsApp client starts as a goroutine alongside PocketBase (`main.go` OnServe hook)
- [x] Session persisted in SQLite (`whatsapp.db` in PocketBase data dir)
- [x] QR code pairing page in operator UI (shown only when no session exists)
- [x] Automatic reconnection on restart (checks `Store.ID`)
- [x] Graceful shutdown (disconnect on OnTerminate)

### 1.2 Message receiving and storage

**Current state:** No message history. Everything lives in WhatsApp.

**Change:** Create a `messages` collection in PocketBase. When whatsmeow receives a message, store it with the sender's phone number, message type, content, and timestamp. Match sender phone to a `business` record (requires adding a `phone` field to businesses).

New `messages` collection schema:

| Field | Type | Description |
|-------|------|-------------|
| `business` | relation | Link to businesses collection (nullable if unknown sender) |
| `phone` | text | Sender phone in E.164 format (e.g. "5511999998888") |
| `type` | select | `text`, `audio`, `image` |
| `content` | text | Message text, or transcript for audio |
| `media` | file | PocketBase file attachment for images (FileField, 10MB max) |
| `direction` | select | `incoming`, `outgoing` |
| `wa_timestamp` | date | WhatsApp message timestamp |
| `wa_message_id` | text | WhatsApp message ID (for deduplication) |

**Files:** new migration for `messages` collection, migration to add `phone` field to `businesses`, `api/internal/whatsapp/handler.go`

- [x] `messages` collection created with schema above (migration 1740000008)
- [x] `phone` field added to `businesses` collection (migration 1740000007)
- [x] Incoming text messages stored with sender phone matched to business
- [x] Unknown senders stored with `business: null` (Elenice can link them later)
- [x] Deduplication by `wa_message_id` (unique index + code check)

### 1.3 Voice note transcription

**Current state:** No audio handling at all.

**Change:** When whatsmeow receives a voice note (PTT audio message), download the OGG/Opus bytes, transcribe via OpenAI Whisper API (or Gemini audio), and store the transcript as the message `content`. The raw audio is not persisted (privacy, storage cost). Cost is < R$0.01 per 30-second clip.

**Files:** `api/internal/whatsapp/handler.go`, new `api/internal/transcribe/` package

- [x] Voice notes downloaded via `client.Download()`
- [x] OGG/Opus audio sent to Whisper API for transcription (`api/internal/transcribe/whisper.go`)
- [x] Transcript stored as message `content` with `type: "audio"`
- [x] Transcription errors logged, message stored with empty content
- [x] Raw audio bytes discarded after transcription

### 1.4 Image handling

**Current state:** No image support.

**Change:** When whatsmeow receives an image, download and store it as a PocketBase file attachment on the message record. Images are displayed in the operator page so Elenice can see what the client sent. The image can optionally be described by the LLM during content generation (pass image URL to the prompt).

**Files:** `api/internal/whatsapp/handler.go`, `messages` collection file field

- [x] Images downloaded via `client.Download()`
- [x] Stored as PocketBase file attachment on the message record (`media` FileField, 10MB max)
- [x] `media` field populated with the file attachment (schema uses FileField instead of text `media_url`)
- [x] Operator page displays images inline in the message thread

### 1.5 Operator page: message thread view

**Current state:** Operator page has a flat client list and a single textarea. No message history.

**Change:** Replace the single textarea with a conversation thread per client. Left side: client list (with health indicators). Right side: message history for the selected client, showing incoming messages (text, transcribed audio, images) and outgoing messages (generated content sent back). New messages from WhatsApp appear in real-time (PocketBase real-time subscriptions).

**Files:** `web/src/routes/(app)/operador/+page.svelte`

- [x] Client list shows unread message count per client
- [x] Selecting a client shows their message thread (incoming + outgoing)
- [x] Voice note messages show transcript with an "audio" indicator ("Audio transcrito" label)
- [x] Image messages render inline
- [x] Real-time updates via PocketBase subscriptions (new messages appear without refresh)
- [x] "Usar ultima msg" button pre-fills with the latest incoming message text

### 1.6 Send replies through WhatsApp

**Current state:** Elenice copies generated content and manually pastes into WhatsApp.

**Change:** After generating content, Elenice clicks "Enviar pelo WhatsApp" and the formatted message (caption + hashtags + production note for Elenice's eyes only) is sent back through whatsmeow. The outgoing message is stored in the `messages` collection with `direction: "outgoing"`.

Production note handling: the production note is shown in the operator page but NOT sent to the client. Instead, it's sent as a separate follow-up message prefixed with a visual marker so the client knows it's a tip, not part of the post.

**Files:** new endpoint `POST /api/messages:send`, `api/internal/whatsapp/send.go`, `web/src/routes/(app)/operador/+page.svelte`

- [x] "Enviar pelo WhatsApp" button sends formatted caption + hashtags to client via WhatsApp
- [x] Production note sent as a separate message ("*Dica de foto:* ...")
- [x] Outgoing messages stored in `messages` collection
- [x] Typing indicator sent before message (`SendChatPresence`)
- [x] Small random delay (1-3s) before sending to simulate human behavior (ban mitigation)
- [x] Elenice can edit the generated content before sending

### 1.7 Persist operator-generated posts

**Current state:** `OperatorGenerate` returns the result but does not save it.

**Change:** Save each operator-generated post to the `posts` collection. Link it to the source message. Load `previousHooks` from saved posts so content angles don't repeat.

**Files:** `handlers/operator.go`, migration to add `source` and `message` fields to posts

- [x] Operator handler saves generated post to `posts` collection with `source: "operator"`
- [x] Post linked to the source `message` record (via `message_id` in request body)
- [x] Operator handler loads `previousHooks` from existing posts for the business
- [x] Response includes `hook` field (migration 1740000009 adds `source` and `message` to posts)

### 1.8 Client health indicators

**Current state:** Client list is a flat sidebar with name/type/city. No visibility into activity.

**Change:** Show per-client indicators based on message and post data: last message received, posts this month, color indicator (green < 5 days since last message, yellow 5-9 days, red 10+ days).

**Files:** `web/src/routes/(app)/operador/+page.svelte`

- [x] Client list shows days since last incoming message with color coding (green < 5d, yellow 5-9d, red 10+d)
- [x] Client list shows post count for current month
- [x] Clients sorted by urgency (red first, then yellow, then green)

### 1.9 Expand production notes in the prompt

**Current state:** `production_note` is typically one generic line.

**Change:** Update the BAML prompt to generate 3-4 sentence mini-briefs with phone-camera-specific instructions. Concrete, amateur-friendly directions that reference the specific item/scene from the client's message.

**Files:** `eval/baml_src/content.baml`

- [x] Prompt instructs model to give phone-specific, step-by-step photo/video directions (4-point mini-roteiro)
- [x] Directions reference the specific item/scene from the client's message
- [x] Eval pipeline still passes after prompt change (`make eval`). Also removed `business_name` and `location` heuristic checks (noisy, ESP judge covers specificity better). Now 4 heuristics: hashtags, pt-BR markers, caption length, production note.

---

## Wave 2: Proactive Engagement (solve consistency)

The product is currently 100% reactive: if the client doesn't send a message, nothing happens. This wave adds the mechanisms for Elenice to keep clients engaged. With WhatsApp integration from Wave 1, nudges and templates can be sent directly from the operator page.

### 2.1 Nudge system for inactive clients

**Current state:** No way to see who has gone quiet.

**Change:** The operator page highlights clients who haven't sent a message in 5+ days. A dedicated "Inativos" filter shows only these clients, sorted by days since last message. Elenice can send a nudge directly from the operator page (it goes through WhatsApp).

**Files:** `web/src/routes/(app)/operador/+page.svelte`

- [ ] "Inativos" filter/tab on client list showing clients with 5+ days since last incoming message
- [ ] Each inactive client shows days of silence
- [ ] "Enviar lembrete" button sends a pre-filled nudge through WhatsApp
- [ ] Nudge templates use the client's name and niche

### 2.2 Re-engagement playbook templates

**Current state:** No templates for handling quiet clients.

**Change:** Tiered re-engagement templates that Elenice can send with one click:
- 5-7 days: casual check-in ("Oi Maria, como foi a semana? Tem algo legal pra gente postar?")
- 8-14 days: seasonal/topical prompt ("Mes que vem e Dia das Maes, vamos preparar posts especiais?")
- 15+ days: value reminder ("Maria, vi que faz um tempo. Quer retomar? Posso te mandar ideias de conteudo pra essa semana!")

The right template is auto-selected based on inactivity duration. Elenice can edit before sending.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

- [ ] Three template tiers auto-selected by inactivity duration
- [ ] Templates personalized with client name
- [ ] Editable before sending
- [ ] Sent through WhatsApp via Wave 1.6 send endpoint

### 2.3 Seasonal content calendar

**Current state:** No mechanism for proactive seasonal content.

**Change:** A simple niche-specific calendar of key dates. The operator page shows upcoming dates (next 30 days) and suggests Elenice reach out to relevant clients with seasonal content ideas. Hardcoded data, not a database.

Key dates by niche:
- **Confeiteiras:** Pascoa, Dia das Maes, Dia dos Namorados, Festas Juninas, Dia das Criancas, Natal
- **Cabeleireiras:** Carnaval, Dia da Mulher, Dia das Maes, Dia do Cabeleireiro (dez), Natal/Reveillon
- **Personal trainers:** Verao (starts Oct), Carnaval (body prep), Dia do Educador Fisico (set)

**Files:** `web/src/routes/(app)/operador/+page.svelte` (or a small helper module)

- [ ] Hardcoded seasonal calendar with niche-specific dates
- [ ] Operator page shows upcoming dates within 30 days
- [ ] Each date has a suggested outreach message template
- [ ] Clicking a date + client sends the message through WhatsApp

---

## Wave 3: Client Value Proof (reduce churn)

Churn happens when clients can't see value. This wave makes the value tangible and shareable.

### 3.1 Monthly client summary

**Current state:** No summary, no tracking of posts delivered per client.

**Change:** Generate a monthly WhatsApp-ready summary per client. Based on data from the `posts` collection: posts delivered this month vs last month. Elenice sends it directly through WhatsApp from the operator page.

Example: "*Maria, resumo de fevereiro:* a gente criou *11 posts* pro seu Instagram (contra 3 em janeiro). O post sobre o bolo de casamento foi o destaque! Mes que vem vamos manter esse ritmo."

The summary uses WhatsApp bold formatting and is designed to be screenshot-worthy (clients share it in professional groups, driving organic referrals).

**Files:** `web/src/routes/(app)/operador/+page.svelte` (summary component), query against `posts` collection

- [ ] Per-client monthly summary view on operator page
- [ ] Shows: posts this month, posts last month, delta
- [ ] "Enviar resumo" button sends formatted summary through WhatsApp
- [ ] Summary uses WhatsApp formatting (*bold*, _italic_) for visual impact

### 3.2 Trial restructuring

**Current state:** BUSINESS.md proposes a 7-day free trial delivering 2-3 posts.

**Change:** R$19 first month instead of free 7-day trial. Rationale:
- 7 days is too short to show Instagram consistency results
- "Free" signals low quality or scam in Brazil
- R$19 filters for serious prospects and creates commitment
- 30 days allows 12+ posts, enough for the client to feel the difference

**Files:** BUSINESS.md, `handlers/subscribe.go` (if implementing promotional pricing)

- [ ] BUSINESS.md updated with R$19 first month model
- [ ] Decide whether promotional pricing is code-managed or Elenice-managed (coupon vs manual)

### 3.3 Consistent pricing across codebase

**Current state:** `subscribe.go` has R$89.90, dashboard UI shows R$49.90, BUSINESS.md says R$49-99 range.

- [ ] Agree on launch price
- [ ] Update `monthlyPriceBRL` in `subscribe.go`
- [ ] Update any hardcoded prices in frontend
- [ ] Update BUSINESS.md

---

## Wave 4: Existing Technical Gaps

The original gaps from PEP-007 v1. Still relevant, but don't block Elenice from operating.

### 4.1 Handler test coverage

Three handlers have zero tests. Core business logic (billing, generation, operator tool).

| Handler | File | What to test |
|---------|------|-------------|
| `GeneratePosts` | `handlers/generate.go` | Subscription check, hook loading, trial increment, ownership, LLM error |
| `CreateSubscription` | `handlers/subscribe.go` | Customer creation, subscription creation, Asaas errors, duplicate |
| `GetSubscription` | `handlers/subscribe.go` | Returns current status, no subscription case |
| `OperatorGenerate` | `handlers/operator.go` | Auth, ownership, empty message, generation error, response format |

Existing `webhooks_test.go` (5 tests) provides the pattern.

- [ ] `generate_test.go` covers happy path and subscription rejection
- [ ] `subscribe_test.go` covers customer+subscription creation and Asaas errors
- [ ] `operator_test.go` covers auth, ownership, and message validation

### 4.2 Content rotation wiring -- DONE

Operator tool passes `nil` for `previousHooks`. Covered by Wave 1.7 (persist operator posts).

- [x] Covered by Wave 1.7 (operator.go now calls `loadPreviousHooks`)

### 4.3 Frontend testing

Zero test files in `web/src/`.

- [ ] Decide testing strategy: Vitest component tests, Playwright e2e, or both
- [ ] Playwright smoke tests for: login redirect, onboarding flow, generation, operator page

### 4.4 Manual verification checklist

- [ ] Google sign-in works end-to-end from SvelteKit
- [ ] Manual threat model verification: attempt each attack row and confirm rejection

### 4.5 Component library extraction (PEP-004)

Wave 1 (Tailwind v4 + shadcn-svelte) is done. Remaining:
- Extract brand components (SectionLabel, PhoneFrame, IgPost, Container)
- Marketing page migration to Tailwind utilities

- [ ] Brand components extracted to `web/src/lib/components/`
- [ ] Marketing page uses Tailwind utilities, no scoped CSS

---

## Implementation Order

| Wave | Focus | Status | Impact |
|------|-------|--------|--------|
| **Wave 1** | WhatsApp integration + operator overhaul | **Done** (9/9 items) | Eliminates all manual copy-paste, enables voice/image, makes the product real |
| **Wave 2** | Proactive engagement | Pending | Solves consistency (the actual problem), reduces churn |
| **Wave 3** | Client value proof | Pending | Makes value visible, reduces churn, drives referrals |
| **Wave 4** | Technical gaps | Partial (4.2 done via Wave 1.7) | Test coverage, verification, component cleanup |

Wave 1 is done. WhatsApp messages flow into the operator page and replies go back through WhatsApp. Waves 2 and 3 are natural extensions (nudges and summaries are just messages sent through the same pipeline). Wave 4 is housekeeping that can happen in parallel.

### Dependencies

```
Wave 1.1 (whatsmeow connection)
  └─> Wave 1.2 (message receiving)
        ├─> Wave 1.3 (voice transcription)
        ├─> Wave 1.4 (image handling)
        └─> Wave 1.5 (operator thread view)
              └─> Wave 1.6 (send replies)
                    ├─> Wave 1.7 (persist posts)
                    ├─> Wave 1.8 (health indicators)
                    ├─> Wave 2.x (proactive engagement)
                    └─> Wave 3.1 (monthly summaries)
Wave 1.9 (production notes prompt) — independent, can happen anytime
Wave 3.2-3.3 (pricing) — independent
Wave 4.x (technical gaps) — independent
```
