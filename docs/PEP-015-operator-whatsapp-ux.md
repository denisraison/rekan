# PEP-015 — Operator WhatsApp-style UX

**Status:** Complete — all 5 waves done + performance follow-up
**Date:** 2026-03-05

## Context

The operator page has two text inputs visible simultaneously: a "quick reply" input between the message thread and the generate panel, and a "Gerar post" textarea in a separate bottom panel with its own label, buttons, and result area. This is confusing. The operator must decide which input to use and context-switch between two workflows that look similar but behave differently.

WhatsApp solves this with one input bar that does one thing at a time. We adopt the same pattern: a single text input at the bottom, with a mode toggle pill on the action chips bar that switches the input's purpose. The chat stays visible in both modes. When in generate mode, the operator can tap messages in the thread to select them as source material (including images and video descriptions). Generated posts appear in a full-screen review overlay where the caption is editable. The "3 ideias" feature changes from pick-one to pick-many, sending each selected idea as a separate WhatsApp message. An attach button lets the operator send photos from the chat window.

## Waves

### Wave 1 — Unified Input Bar + Mode Toggle ✓ Done (2026-03-05)

**Goal:** Replace the dual input areas with a single WhatsApp-style input bar that changes purpose based on mode.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**State changes:**

- Remove `quickReply`, `sendingQuick`, `quickReplyError` (lines 196-198)
- Add `inputMode: 'chat' | 'generate'` state (default `'chat'`)
- Reuse existing `message` state for both modes

**Input bar (replaces lines ~2164-2296):**

One bar at the bottom of the chat area, always visible when `!blockReason`:

- **Chat mode (default):** placeholder "Mensagem...", green send button, Enter submits. Calls the existing `sendQuickReply` logic (rewritten to use `message` instead of `quickReply`).
- **Generate mode:** placeholder "Descreva o post...", coral "Gerar" button, existing "Usar conversa" and "3 ideias" buttons appear as compact chips above the input. The input bar gets a colored top border (coral) so the mode change is visually obvious.

**Mode toggle (action chips bar, right-aligned):**

A pill button on the right side of the action chips bar above the input. Uses icon + short label:

- In chat mode: sparkle icon + "Post" (coral) to switch to generate mode
- In generate mode: chat bubble icon + "Chat" (green) to switch back

When toggling modes, clear `message` text and `selectedMessages` to avoid cross-mode confusion.

**Remove:**

- The "Quick reply row" section (lines ~2164-2196)
- The entire "Generate panel" section (lines ~2198-2296), including the `<textarea>` for generation input and the "Usar conversa" / "3 ideias" / "Gerar" buttons. These move into the unified bar.
- The "Gerar post" label that existed in the old panel

**Keep the `result` display for now** (lines ~2298-2367) but move it to Wave 3 where it becomes a full-screen overlay.

**Gate:** `cd web && pnpm check`. In browser: verify single input bar appears at the bottom. Verify tapping "Gerar Post" chip changes the input placeholder, button color, and shows the generate-mode chips. Verify sending a quick reply works in chat mode. Verify generating a post works in generate mode. Verify switching modes clears the input text.

**Implementation notes:**

- `quickReply` state removed entirely; `message` state reused for both modes
- `quickReplyError` and `sendingQuick` kept since they serve the unified bar's chat mode
- `selectClient` resets `inputMode` to `'chat'` and clears `message` when switching clients
- Desktop idea drafts UI preserved inside the unified bar container (above the input row)
- Result display kept inline for now (Wave 3 moves it to a full-screen overlay)
- The input uses `<input>` (single line) rather than `<textarea>` to match WhatsApp feel; generate mode will get multi-line via message selection in Wave 2

### Wave 2 — Message Selection for Generation ✓ Done (2026-03-05)

**Goal:** When in generate mode, the operator can tap messages in the thread to select them as source material for post generation.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**State changes:**

- Add `selectedMessages: Set<string>` (message IDs)

**Behavior:**

- When `inputMode === 'generate'`, tapping a message bubble toggles its ID in `selectedMessages`
- Selected messages get a visual indicator: a colored left border or a checkbox overlay
- A small badge near the input shows the count of selected messages (e.g., "3 mensagens selecionadas")
- When generating, the payload `message` is built by concatenating:
  1. Content from selected messages (text, and for image messages: include the image description or "[Imagem]" marker, for video: "[Video]" or transcription if available), sorted by timestamp
  2. Whatever the operator typed in the text input (appended as additional context)
- The existing `message_id` optimization (line 1112-1113) is replaced: if exactly one message is selected and no additional text is typed, pass that message's ID
- "Usar conversa" becomes "Selecionar recentes": it selects all recent incoming messages (same logic as `recentContext`) by adding their IDs to `selectedMessages` rather than filling the textarea
- Deselecting all messages and clearing the input returns to a clean state
- Switching back to chat mode clears `selectedMessages`

**Gate:** `cd web && pnpm check`. In browser: switch to generate mode, tap 2-3 messages, verify they get highlighted and the count badge updates. Verify generating with selected messages sends their concatenated content. Verify "Selecionar recentes" selects the right messages. Verify switching to chat mode clears selections.

**Implementation notes:**

- `recentContext` (string concatenation) removed; replaced by `recentContextIds` (Set of message IDs) for the "Selecionar recentes" button
- `latestIncoming` derived removed; `message_id` optimization now uses `selectedMessages` directly (if exactly one selected and no typed text)
- `buildSelectedContent()` concatenates selected messages sorted by thread order, with `[Imagem]`/`[Video]` markers for media without text content
- Selected messages get a coral left border and coral-pale background; the count badge appears as a chip above the input
- "Selecionar recentes" chip only shows when no messages are selected yet (avoids confusion with manual selections)

### Wave 3 — Post Review Overlay ✓ Done (2026-03-05)

**Goal:** After generation completes, show the result in a full-screen overlay where the operator can review and edit the caption before sending.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**State changes:**

- Add `editingCaption: string` (initialized from `result.caption` when overlay opens)

**Overlay layout** (follows the existing mobile ideas picker pattern at lines ~2059-2097):

- Full-screen overlay (`absolute inset-0 z-10` on the detail view container)
- Header: "Voltar" button (closes overlay, keeps result) + "Post gerado" title
- Body (scrollable):
  - Caption as an editable `<textarea>`, auto-sized, bound to `editingCaption`
  - Hashtags displayed below (read-only, with "Copiar" button)
  - Production note in italic (read-only, with "Copiar" button)
- Footer actions:
  - "Enviar pelo WhatsApp" (green, full-width) — sends using `editingCaption` (not original `result.caption`)
  - "Descartar" (text button, destructive color) — clears `result` and closes overlay

**Remove** the inline result display (lines ~2298-2367) that currently lives inside the generate panel.

**Gate:** `cd web && pnpm check` test in in browser with a new e2e test generate a post, verify the full-screen overlay appears. Verify the caption is editable. Verify editing the caption and sending uses the edited version. Verify "Voltar" closes overlay but preserves the result. Verify "Descartar" clears everything.

**Implementation notes:**

- Added `showReviewOverlay` boolean state to decouple overlay visibility from `result` presence, allowing "Voltar" to close the overlay while preserving the result
- A `$effect` on `result` sets `editingCaption` and opens the overlay when result becomes non-null, closes it when null
- "Ver post gerado" chip appears in the action chips bar when result exists but overlay is closed, letting the operator re-open it
- Edited caption persists through Voltar/reopen round-trips since the effect only fires when `result` changes
- E2e tests added in `web/tests/post-review-overlay.spec.ts` covering overlay display, Voltar, Descartar, and caption edit persistence

### Wave 4 — Multi-select Ideas ✓ Done (2026-03-05)

**Goal:** Change "3 ideias" from pick-one to pick-many, sending each selected idea as a separate WhatsApp message.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**State changes:**

- [x] Add `selectedIdeas: Set<number>` (indices into `ideaDrafts` array)
- [x] Add `sendingIdeas: boolean`

**UI changes to the ideas picker** (both mobile overlay and desktop panel):

- [x] Each idea card gets a selectable checkbox/indicator (tapping the card toggles selection)
- [x] Tapping still allows reading the full caption
- [x] Bottom action bar: "Enviar X selecionadas" button (green, shows count)
- [x] Tapping "Enviar X" sends each selected idea as a separate WhatsApp message via `/api/messages:send` sequentially (with the existing delay/typing simulation the backend already does)
- [x] Each sent idea is also saved as a proactive post (same `posts:saveProactive` logic)
- [x] Single-select behavior preserved: if only 1 is selected, it can also go through the review overlay (Wave 3) for editing before send
- [x] "Cancelar" button clears everything

**Gate:** `cd web && pnpm check` test in in browser with a new e2e test generate 3 ideas, select 2, verify "Enviar 2 selecionadas" sends both as separate messages. Verify selecting just 1 and tapping it opens the review overlay. Verify all sent ideas appear in the message thread via realtime.

**Implementation notes:**

- Idea cards changed from `<div>` with nested button to clickable `<button>` elements with a circular checkbox indicator
- Selected cards get a coral border (2px) for clear visual feedback
- Single selection opens the review overlay (Wave 3) via "Revisar e enviar" button, multi-selection shows "Enviar N selecionadas" green button
- `sendSelectedIdeas()` iterates selected indices in order, calling `posts:saveProactive` then `messages:send` for each
- `selectedIdeas` is cleared on client switch, on "Cancelar"/"Limpar", and when closing the ideas picker
- Desktop and mobile overlays both support the same multi-select flow with consistent UI

### Wave 5 — Attach Button (Camera/Gallery) ✓ Done (2026-03-05)

**Goal:** Let the operator send photos and files from the chat window, like WhatsApp's attach button and camera icons

**Files:**

- `web/src/routes/(app)/operador/+page.svelte` (frontend)
- `api/internal/http/handlers/send_media.go` (new: send media via WhatsApp)
- `api/internal/http/handlers/describe_media.go` (new: Gemini image description)
- `api/internal/http/handlers/deps.go` (added Transcribe field)
- `api/internal/whatsapp/client.go` (Upload method)
- `api/internal/http/routes.go` (two new endpoints)
- `api/main.go` (wire transcribe client into Deps)

**Frontend:**

- [x] A paperclip icon button to the left of the text input (both modes)
- [x] Tapping opens a popup menu with Galeria, Camera, and Video options
- [x] Selecting a file shows a thumbnail preview above the input (not sent immediately)
- [x] Preview has a remove button to clear the attachment
- [x] **Generate mode:** attached image is described via `/api/media:describe` (Gemini), then the description feeds into the generation prompt alongside selected messages and typed text
- [x] **Chat mode:** attached image is sent directly via `/api/messages:sendMedia` as a WhatsApp media message
- [x] State: `showAttachMenu`, `sendingMedia`, `attachedFile`, `attachedPreview`
- [x] Attachment cleared on mode switch and client switch
- [x] Toast notification system for error feedback
- [x] Backdrop closes menu when tapping outside

**Backend:**

- [x] New `SendMedia` handler accepts `multipart/form-data` with `business_id`, `caption`, and `file` fields
- [x] New `DescribeMedia` handler accepts `multipart/form-data` with `file`, returns `{ description }` via Gemini
- [x] Added `Upload` method to whatsapp Client wrapper (wraps `whatsmeow.Upload`)
- [x] Added `Transcribe` field to handler Deps, wired in main.go
- [x] Uploads media to WhatsApp servers, sends as `ImageMessage` or `VideoMessage` based on MIME type
- [x] Stores outgoing media in PocketBase's `messages` collection with correct type and `media` field
- [x] New endpoints: `POST /api/messages:sendMedia`, `POST /api/media:describe`

**Gate:** `cd web && pnpm check` test in in browser with a new e2e test verify the buttons appears next to the input. Verify tapping it shows Camera/Gallery options. Verify selecting a photo sends it as a WhatsApp media message (if backend is ready) or shows a "em breve" toast (if stubbed). Backend: `cd api && go test ./...` passes.

**Implementation notes:**

- Attach does NOT send immediately. It shows a preview, letting the operator use the photo as context for post generation (the main use case: visit customer, take photo, generate post from it)
- In generate mode, `generate()` calls `/api/media:describe` first to get a Gemini description, then prepends `[Foto do operador] <description>` to the message payload sent to `posts:generateFromMessage`
- In chat mode, `sendQuickReply()` delegates to `sendMedia()` when an attachment is present, sending directly via WhatsApp
- Created separate `SendMedia` and `DescribeMedia` handlers rather than extending existing ones
- Camera option uses `accept="image/*;capture=camera"` to trigger native camera on mobile devices
- Video support included alongside images, auto-detected from MIME type
- Toast notification system added (3s auto-dismiss) instead of blocking `alert()` for media errors
- E2e tests (10 cases) cover: button visibility in both modes, menu options, backdrop close, preview display/remove, preview persistence, mode-switch clearing, and button enable state with attachment

## E2E Testing (Playwright)

The operator page requires authentication and a running backend. Playwright tests live in `web/tests/`.

**Setup:**

- Config: `web/playwright.config.ts` (baseURL `http://localhost:5173`, `reuseExistingServer: true`)
- Dev server uses HTTPS with a self-signed cert, so tests that hit the running server need `ignoreHTTPSErrors`:

```ts
import { expect, test } from '@playwright/test';

test.use({ ignoreHTTPSErrors: true, baseURL: 'https://localhost:5173' });
```

**Login helper:**

```ts
async function loginAsOperador(page: any) {
  await page.goto('/entrar');
  await page.getByLabel('Email').fill('operador@rekan.local');
  await page.getByLabel('Senha').fill('senha1234567');
  await page.getByRole('button', { name: 'Entrar' }).click();
  await page.waitForURL('**/operador**');
  await page.waitForTimeout(2000); // SSE stream prevents networkidle
}
```

**Gotchas:**

- Do NOT use `waitForLoadState('networkidle')` on the operator page. The WhatsApp SSE stream keeps the connection open indefinitely. Use `waitForTimeout` instead.
- "Post" as button text matches many elements ("sem postar", "Post criado" etc). Use `getByRole('button', { name: 'Post', exact: true })` or a more specific locator.
- Both backend (`make dev-api`) and frontend (`make dev-web`) must be running. Use `make dev` to start both.
- Seed the DB first with `make seed` if starting fresh.

**Running:**

```bash
cd web && pnpm exec playwright test tests/my-test.spec.ts --reporter=list --timeout=30000
```

**Screenshots for visual verification:**

```ts
await page.screenshot({ path: '/tmp/my-screenshot.png', fullPage: true });
```

Then read the screenshot with the `Read` tool to verify visual output.

### Performance Follow-up (2026-03-06)

**Problem:** Attaching an image and clicking "Gerar Post" took over 10 seconds just for the image description step, with a 30s timeout risk on larger photos.

**Changes:**

- **Switched Gemini model** for image/video description from `gemini-2.5-flash` to `gemini-3.1-flash-lite-preview` (`api/internal/transcribe/gemini.go`)
- **Added image downscaling** before sending to Gemini: images larger than 1024px in either dimension are resized proportionally and re-encoded as JPEG 80% quality. Uses `golang.org/x/image/draw` (promoted from indirect to direct dependency in `go.mod`). Reduces a typical 2MB phone photo to ~100-200KB. Result: description time dropped from ~10s to ~3s.
- **Removed video attach button** from the operator generate UI. Video descriptions can't be optimized the same way without ffmpeg, and operators rarely attach video for post generation. Video handling remains in the WhatsApp message ingestion pipeline.
- **Fixed camera capture attribute**: changed from `accept="image/*;capture=camera"` (non-standard) to separate `accept="image/*"` + `capture="environment"` HTML attribute, which correctly triggers native camera on mobile devices.

### SSE Reconnection on Resume (2026-03-06)

**Problem:** When the operator locks/unlocks their phone or switches tabs, the browser suspends the page. The SSE streams (WhatsApp status, message updates) die silently and never reconnect, leaving the UI stale until a manual refresh.

**Changes:**

- **Operator page** (`+page.svelte`): added a `visibilitychange` listener that aborts the current SSE connection and reconnects when the page becomes visible again. Also refreshes messages, posts, scheduled messages, and suggestion counts to catch anything missed while suspended.
- **WhatsApp status page** (`whatsapp/+page.svelte`): same `visibilitychange` reconnection for the QR/status SSE stream.
- Both listeners are cleaned up in `onDestroy`.

## Consequences

- The operator gets one unified input bar instead of two confusing text fields. The mode toggle makes the current purpose explicit.
- Message selection for generation gives the operator fine-grained control over what context feeds the AI, including images and video descriptions from the chat.
- The full-screen review overlay makes post editing a focused activity instead of something squeezed into a small bottom panel.
- Multi-select on ideas lets the operator send multiple suggestions at once without going through generate-review-send three times.
- The attach button completes the WhatsApp-like experience, letting the operator handle media without leaving the page.
- The send endpoint needs a media upload extension (Wave 5). This is the only backend change; Waves 1-4 are frontend-only.
- The `quickReply` state and its dedicated input are removed. All reply/send functionality goes through the unified bar.
