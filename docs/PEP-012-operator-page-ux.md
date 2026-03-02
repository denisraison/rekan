# PEP-012 — Operator Page UX Improvements

**Status:** In Progress — Wave 1 complete
**Date:** 2026-03-02

## Context

The operator page (`/operador`) is the main daily tool for whoever manages Rekan clients. It handles the core loop: client messages on WhatsApp → operator generates a post → sends it back. It also handles proactive outreach (nudges, seasonal dates, monthly summaries) and subscription management.

A review session with PM and designer surfaced several issues that directly damage this loop. The most serious ones cause the operator to either lose access to their client list entirely (QR takeover), act without context (no dates in thread, no billing info visible), or get silently blocked with no feedback (disabled buttons). None of these require new features — they are corrections to what's already there.

The improvements are grouped into three waves: correctness issues that are actively breaking workflow, quality improvements that elevate the experience without blocking anything, and new features that extend what the operator can do without leaving the page.

## Waves

### Wave 1 — Fix What's Broken ✓ Done (2026-03-02)

**Goal:** Remove friction that causes the operator to lose context or get silently blocked.

**1. Auto-scroll message thread to bottom**

File: `web/src/routes/(app)/operador/+page.svelte`

The thread container (around line 1524) has no scroll behavior. When the operator selects a client or a new realtime message arrives, the thread renders from the top — newest messages are off-screen every time.

Fix: `bind:this={threadEl}` on the scroll container div. Add a `$effect` that reads `threadMessages` and `selectedId` as dependencies and calls `threadEl.scrollTop = threadEl.scrollHeight` after each change. In Svelte 5, `$effect` runs after DOM updates, so no `tick()` is needed.

**2. Date separators in message thread**

File: same

Timestamps show `HH:MM` only. A conversation spanning multiple days looks like a wall of same-day messages. Operator loses temporal context.

Fix: Derive `groupedMessages` from `threadMessages` — group consecutive messages that share the same calendar day. Render a centered horizontal rule with a date label between groups. Label logic: "Hoje" if today, "Ontem" if yesterday, `"Seg, 24 fev"` format otherwise. Replace the flat `{#each threadMessages}` with `{#each groupedMessages as group}` containing a date separator followed by the group's bubbles.

**3. QR pairing as overlay, not page replacement**

File: same

The block at `{:else if !waConnected && waQR}` (around line 908) replaces the entire `<main>` with a centered QR card. WhatsApp disconnections happen during the day (phone dies, server restart). When they do, the operator loses access to every client and message until pairing completes.

Fix: Move the QR card to a `fixed inset-0` overlay that renders on top of the normal layout. The `<main>` with the two-column layout renders unconditionally. When `!waConnected && waQR`, the overlay appears above it. When `waQR` clears (connected or no QR yet), the overlay disappears. The client list and thread remain accessible underneath.

**4. Explain disabled buttons and fix all user-facing text**

File: same

Four send buttons (nudge, summary, generated post, and the WhatsApp send for nudge/summary) go silent when `!waConnected || !selected?.phone`. The "Enviar pelo WhatsApp" after generation is inside `{#if waConnected && selected?.phone}` — it simply doesn't render, leaving the operator with a generated result and no send action, no explanation.

Fix:
- Derive a `blockReason` string for each send context: `"WhatsApp desconectado"` takes priority over `"Cliente sem telefone cadastrado"`.
- Show the reason as a `<span>` next to the button when it is the active blocker (not when the button is merely empty/sending).
- Add `title={blockReason}` for hover tooltip.
- For the generated post: always render the send button area. When blocked, show a muted line explaining why instead of hiding it.

**Language standard — all user-facing text in this page:**

Every string shown to the operator must be in correct, natural pt-BR — no Portuglish, no machine-translated phrases, no abbreviated slang that could confuse. This applies to button labels, error messages, empty states, loading indicators, and tooltip text introduced by this PEP and any that already exist in the file. As part of Wave 1, do a pass over all hardcoded strings and fix any that are awkward or unclear. Examples of things to catch: `"Gerando..."` is fine; `"Aguardando QR code"` as a heading when the server hasn't produced a QR yet is confusing — prefer `"Conectando ao WhatsApp..."`. Error messages should say what the operator can do, not just what went wrong.

**5. Persist `lastSeen` to localStorage**

File: same

`lastSeen` is plain `$state({})`. On every page reload all incoming messages appear as unread, lighting up the badge for every client. For an operator with 20 clients this is noise every time.

Fix: On `onMount`, initialize `lastSeen` from `localStorage.getItem('rekan_operator_last_seen')` (JSON.parse, fallback to `{}`). Wrap the `lastSeen` assignment in `selectClient` with a `localStorage.setItem` call after updating state.

Gate: `cd web && pnpm check`. In browser: select a client, confirm thread scrolls to bottom on select and on new realtime message. Confirm date separators appear between days and "Hoje"/"Ontem" labels are correct. Confirm QR overlay appears over the client list without hiding it. Confirm disabled send buttons show the reason ("WhatsApp desconectado" or "Cliente sem telefone cadastrado"). Reload the page and confirm unread counts are preserved per client.

### Wave 2 — Raise the Quality Bar

**Goal:** Make the operator view more informative and the interface more polished.

**1. Billing info in client header**

File: same

`tier`, `next_charge_date`, and `charge_pending` are in the `Business` type and fetched with the client list but displayed nowhere. The operator cannot see who's at billing risk without opening the edit form.

Fix: In the client header section (around line 1350), add a second line of metadata below the existing type/city/state line. Show tier as a badge (basico/parceiro/profissional), next charge date formatted as `"Próx. cobrança: 15 mar"`, and a red "Pagamento pendente" badge when `charge_pending` is true. Only show the charge date if `invite_status === 'active'`.

**2. Engagement panel height cap + collapse**

File: same

When all three sections appear simultaneously (nudge textarea + seasonal pills + summary textarea), the engagement panel can consume 300-400px, squeezing the thread to almost nothing on a laptop screen. The panel is `shrink-0` so it never yields space.

Fix: Add `max-height: 38vh; overflow-y: auto` to the engagement panel container. Additionally, make the nudge and summary textareas collapsed by default — show a single-line label+chevron button that expands on click (`nudgeOpen`, `summaryOpen` local booleans). The seasonal date pills stay always visible since they are compact. This keeps the panel at one compact row by default and the operator expands only what they intend to use.

**3. Generate panel max height when result is shown**

File: same

The generate panel is `shrink-0` and grows downward when a result appears (caption + hashtags + production note + send button). On short screens this pushes the thread up significantly.

Fix: Add `max-height: 50vh; overflow-y: auto` to the generate panel container. The result scrolls within the panel rather than collapsing the thread.

**4. Hover state on client list items**

File: same

Selected items get `background: var(--coral-pale)` inline. Non-selected items have `transition-colors` but no hover style because Tailwind hover classes don't apply to inline `style=` backgrounds.

Fix: Use conditional Tailwind classes instead of inline style for the background. Selected: `bg-(--coral-pale)`. Not selected: `hover:bg-(--coral-pale)/40`. Both are within the existing design system.

**5. Copy button race condition**

File: same

`copied` is a single shared string. Copying two fields quickly causes the first timeout to clear the label mid-display for the second field. Each field should use its own boolean or the previous timeout should be cancelled on each new copy.

Fix: Replace the shared `copied = label` pattern with a `Map<string, boolean>` or three separate booleans (`captionCopied`, `hashtagsCopied`, `noteCopied`). Cancel the previous timeout before setting a new one using a ref to the timeout id.

Gate: `cd web && pnpm check`. In browser: confirm billing info (tier, next charge date, charge_pending) appears in the client header for active clients. Confirm the engagement panel collapses nudge and summary by default and expands on click. Confirm the generate result panel scrolls internally rather than pushing the thread up. Confirm hover state on client list items works. Confirm copying caption then hashtags quickly shows correct labels for each.

### Wave 3 — New Features

**Goal:** The operator should never be stuck. Every client — active or quiet — should have a clear next action. These features close the gap between "client sent a message" (reactive) and "client has gone quiet" (proactive), while keeping the operator in control of everything that touches the client.

Features 1–6 are frontend-only. Features 7–8 require backend work.

**1. "Usar conversa recente" — full context in one click**

File: `web/src/routes/(app)/operador/+page.svelte`

Brazilian WhatsApp culture means clients never send one long paragraph — they send 4-5 short messages, maybe an audio, maybe a photo. The current "Usar última msg" only grabs the last incoming message, losing all that context.

Fix: Replace "Usar última msg" with "Usar conversa recente". Instead of grabbing just `latestIncoming`, derive `recentIncoming`: all incoming messages since the last outgoing message (or last 24h if no outgoing exists), sorted oldest-first, joined with line breaks. One click populates the textarea with the full conversation context. The operator can still edit before generating.

No backend change — `GenerateFromMessage` already takes a plain string.

**2. Quick reply input in the thread**

File: same

There is no way to send a short conversational reply ("Manda uma foto com luz natural?", "Perfeito, vou preparar!") without going through the nudge panel, which is conditional and above the thread. This forces the operator to switch to WhatsApp directly for anything conversational, breaking the thread history.

Fix: A compact single-line text input + send button sitting between the thread and the generate panel. Submitting calls `/api/messages:send` with caption set to the input text and empty hashtags/production_note — the same shape the nudge already sends. Clear on success. The message appears in the thread via the existing realtime subscription.

Show only when `waConnected && selected?.phone`. When blocked, show the `blockReason` from Wave 1.

**3. Client context card (read-only)**

File: same

`target_audience`, `brand_vibe`, `quirks`, and `services` with prices are on the `selected` object but invisible during normal operation. Seeing them requires clicking "Editar" and opening the full form — risky and clunky.

Fix: A collapsible read-only card below the client header. Collapsed by default ("Ver perfil do negócio ▼"). Expanded: services listed with prices in R$, target audience, brand vibe, quirks. An "Editar" link at the bottom opens the actual form. No editing in this card.

This context is what lets the operator write "Oi Ana, vamos postar sobre o combo de unhas por R$45?" instead of "Oi Ana, tem algo pra postar?".

**4. Post history panel per client**

File: same

The `posts` array is already loaded and subscribed via realtime. It is used only for the health indicator count. The operator has no way to browse what was previously generated for a client — if a client asks "manda aquele post de novo" or the operator wants to avoid repeating a topic, they have to scroll through the entire message thread.

Fix: A collapsible "Histórico de posts" section below the client header. Posts listed most-recent first. Each entry shows date + caption truncated to ~2 lines + "Copiar" button. Expand on click to see full caption and production note. Limit to last 10, with a "Ver todos" option.

A `clientPosts` derived value filtered by `selectedId` already has all the data.

**5. Hook counter — surface the duplicate protection**

File: same

The system already prevents duplicate posts: every generated post's hook (angle/theme) is saved and passed back to the AI on the next generation with explicit instructions not to repeat it. The operator does not know this exists. After generating several posts for the same client, she may wonder if the AI is running out of ideas.

Fix: Show a single line near the "Gerar post" button when a client has 3 or more previous posts: "X temas anteriores memorizados — o próximo post será diferente." No interaction, no list, just reassurance. Derived from `clientPosts.length`.

**6. Morning workflow summary**

File: same

Opening the page with 20+ clients means scanning the list to find where to focus. The data to answer "what do I do right now?" is already in memory.

Fix: A compact summary bar at the top of the left panel, between the filter tabs and the client list. Up to four lines, each clickable:
- "3 clientes com mensagens novas" → scrolls those clients to the top
- "2 clientes ficando inativos" → sets filter to "inativos"
- "Dia das Mães em 8 dias — 5 clientes elegíveis" → nearest upcoming seasonal date with eligible count
- "1 cliente com pagamento pendente" → highlights that client

Hide lines with zero count. If all zero, hide the bar. Derived entirely from `unreadCounts`, `clientHealth`, `SEASONAL_DATES`, and the clients array.

**7. Proactive idea generation — "Gerar 3 ideias"**

Files: `web/src/routes/(app)/operador/+page.svelte`, `api/internal/http/handlers/operator.go`

This is the highest-impact feature. When a client has gone quiet, the operator can nudge them but has nothing concrete to offer if the client replies "não sei o que postar." Right now the only path to content is waiting for the client to send a message.

The `GenerateContent` BAML function already exists in `eval/baml_src/content.baml`. It takes a business profile, content roles, and previous hooks, and produces 3 complete posts with no client message needed. It just has no API endpoint.

Fix — backend: New endpoint `POST /api/businesses/:id/posts:generateIdeas`. The `GenerateContent` BAML function already exists in `eval/baml_src/content.baml` and the API already imports the eval package (`operator.go` line 10). Add `GenerateIdeas` to the `Deps` struct alongside `GenerateFromMessage`, inject it the same way. The handler calls `GenerateContent` with the business profile and 3 varied content roles (hardcoded for now, same role definitions used in eval). Passes `previousHooks` from `loadPreviousHooks` to avoid repeats. Returns 3 draft posts — does NOT save them to the posts collection yet.

Fix — frontend: A "Gerar 3 ideias" button in the engagement panel, visible when a client is inactive (daysSinceMsg >= 5) or has no messages at all. Clicking it calls the new endpoint. Results appear as 3 draft cards in-page — each showing the caption preview and a "Usar este" button. When the operator clicks "Usar este", that draft loads into the existing generate result panel at the bottom. The operator reviews it and sends as normal. Only when sent (via "Enviar pelo WhatsApp") is the post saved to the posts collection with `source: "proactive"`. The other two drafts are discarded. This keeps the hook list and health counts clean.

The best flow: operator sends nudge ("Oi Ana, faz um tempo...") and simultaneously generates 3 ideas. If Ana replies "não sei o que postar," the operator immediately sends one of the ready ideas: "Fiz um post pra você, olha que ficou bom."

**8. Automated operator prep — pre-caching and seasonal batch**

Files: `api/` (cron jobs via PocketBase hooks), `web/src/routes/(app)/operador/+page.svelte`

Do NOT automate client-facing WhatsApp messages. The relationship between the operator and the client is personal and paid. An automated message that arrives at the wrong moment — client on vacation, bad week, shop closed — damages trust in a way that a human would have avoided. WhatsApp also has spam detection that can affect the entire number if clients report messages.

Instead, automate the operator's prep work:

**Pre-caching ideas:** When a client hits 5 days without an incoming message, a PocketBase cron job (configured via `OnCronJobRun` hook in `api/`) calls `generateIdeas` for that business and stores the 3 draft posts in a new `idea_drafts` PocketBase collection with fields: `business`, `caption`, `hashtags`, `production_note`, `created`. Drafts are not in the `posts` collection — they are separate so they do not pollute health counts or hooks. When the operator opens a client that has cached drafts, the "Gerar 3 ideias" button changes to "Ver ideias prontas" and renders the cached cards immediately. Drafts older than 7 days are discarded by the cron job on its next run. When the operator selects a draft and sends it, it is saved to `posts` with `source: "proactive"` and the remaining drafts for that business are deleted.

**Seasonal batch approval:** Seven days before a relevant seasonal date (Dia das Mães, Páscoa, etc.), the cron job queues a pre-filled seasonal nudge message for each eligible client into a new `scheduled_messages` PocketBase collection with fields: `business`, `text`, `scheduled_for`, `approved`, `dismissed`. The morning summary bar (feature 6) shows "3 mensagens sazonais prontas para aprovação" when any exist. Clicking that line opens an inline approval panel inside the left sidebar (not a new route, not a modal) — one compact card per client showing the business name, the message text (editable), and "Enviar" / "Descartar" buttons. Approved messages call `/api/messages:send` immediately and are removed from the queue. Nothing goes out without the operator seeing and approving it.

This gives the operator the time savings of automation — ideas pre-generated, seasonal messages pre-written — without removing human judgment from the client relationship.

Gate for Wave 3: `cd web && pnpm check`. In browser: verify "Usar conversa recente" grabs all incoming messages since the last outgoing and joins them. Verify "Gerar 3 ideias" returns 3 draft cards and "Usar este" loads one into the result panel without saving the others. Verify sending via "Enviar pelo WhatsApp" saves the post with `source: "proactive"`. Verify the hook counter appears after 3+ posts and the count is correct. Verify the morning summary bar shows the right counts and each line navigates correctly. Verify pre-cached idea drafts appear instantly for a client with 5+ days inactive. Verify the seasonal batch approval panel appears in the sidebar and approved messages are sent; dismissed messages disappear from the queue.

## Consequences

- The operator retains access to the client list even during WhatsApp disconnection. The `/operador/whatsapp` dedicated page remains as the detailed pairing view; the main page now shows a non-blocking overlay.
- Unread counts become meaningful across sessions instead of resetting on every reload.
- The thread becomes oriented in time, reducing the cognitive load of understanding where a conversation stands.
- Billing info surfaces a signal the operator previously had to guess at (who's at payment risk).
- The engagement panel no longer dominates the layout — the message thread regains vertical space by default.
- No new npm dependencies are introduced. All fixes and features use existing CSS variables, Tailwind utilities, and Svelte 5 primitives already in the project.
- The operator is never stuck: active clients have "Usar conversa recente", quiet clients have "Gerar 3 ideias", and pre-caching means the ideas are ready before the operator even opens the thread.
- Client-facing messages remain human-reviewed. Automation targets the operator's prep, not the client relationship.
- The key metric to watch across all of Wave 3: posts generated per client per month. If clients who were stuck at 1–2 posts/month start reaching 4+, the features are working.
