# PEP-015 — Operator WhatsApp-style UX

**Status:** In Progress — Wave 1 done
**Date:** 2026-03-05

## Context

The operator page has two text inputs visible simultaneously: a "quick reply" input between the message thread and the generate panel, and a "Gerar post" textarea in a separate bottom panel with its own label, buttons, and result area. This is confusing. The operator must decide which input to use and context-switch between two workflows that look similar but behave differently.

WhatsApp solves this with one input bar that does one thing at a time. We adopt the same pattern: a single text input at the bottom, with a mode toggle ("Gerar Post" chip) in the client header bar that switches the input's purpose. The chat stays visible in both modes. When in generate mode, the operator can tap messages in the thread to select them as source material (including images and video descriptions). Generated posts appear in a full-screen review overlay where the caption is editable. The "3 ideias" feature changes from pick-one to pick-many, sending each selected idea as a separate WhatsApp message. An attach button lets the operator send photos from the chat window.

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

**Mode toggle in client header bar (around line 2020-2056):**

Add a "Gerar Post" chip/pill next to the client name/info. When tapped, toggles `inputMode` between `'chat'` and `'generate'`. Visual states:
- Inactive: outlined chip, muted color
- Active: filled coral background, white text

When toggling modes, clear the `message` text to avoid sending a half-typed reply as a generation prompt or vice versa.

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

### Wave 2 — Message Selection for Generation

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

### Wave 3 — Post Review Overlay

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

**Gate:** `cd web && pnpm check`. In browser: generate a post, verify the full-screen overlay appears. Verify the caption is editable. Verify editing the caption and sending uses the edited version. Verify "Voltar" closes overlay but preserves the result. Verify "Descartar" clears everything.

### Wave 4 — Multi-select Ideas

**Goal:** Change "3 ideias" from pick-one to pick-many, sending each selected idea as a separate WhatsApp message.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**State changes:**
- Add `selectedIdeas: Set<number>` (indices into `ideaDrafts` array)
- Add `sendingIdeas: boolean`

**UI changes to the ideas picker** (both mobile overlay and desktop panel):
- Each idea card gets a selectable checkbox/indicator (tapping the card toggles selection)
- Tapping still allows reading the full caption
- Bottom action bar: "Enviar X selecionadas" button (green, shows count)
- Tapping "Enviar X" sends each selected idea as a separate WhatsApp message via `/api/messages:send` sequentially (with the existing delay/typing simulation the backend already does)
- Each sent idea is also saved as a proactive post (same `posts:saveProactive` logic)
- Single-select behavior preserved: if only 1 is selected, it can also go through the review overlay (Wave 3) for editing before send
- "Cancelar" button clears everything

**Gate:** `cd web && pnpm check`. In browser: generate 3 ideas, select 2, verify "Enviar 2 selecionadas" sends both as separate messages. Verify selecting just 1 and tapping it opens the review overlay. Verify all sent ideas appear in the message thread via realtime.

### Wave 5 — Attach Button (Camera/Gallery)

**Goal:** Let the operator send photos and files from the chat window, like WhatsApp's attach button.

**Files:**
- `web/src/routes/(app)/operador/+page.svelte` (frontend)
- `api/internal/http/handlers/send_message.go` (backend: add media support)
- `api/internal/http/routes.go` (if new endpoint needed)

**Frontend:**
- A "+" icon button to the left of the text input (both modes)
- Tapping opens a small popover/bottom sheet with options:
  - "Camera" — `<input type="file" accept="image/*" capture="environment">`
  - "Galeria" — `<input type="file" accept="image/*,video/*">`
- Selected file is sent as a WhatsApp media message
- State: `showAttachMenu: boolean`, `sendingMedia: boolean`

**Backend:**
- The current `SendMessage` handler only sends text (`waE2E.Message.Conversation`)
- Need to add a new endpoint or extend existing one to accept `multipart/form-data` with a file field
- Use whatsmeow's `Upload` + `SendMessage` with `ImageMessage` or `VideoMessage` proto
- Store the media file in PocketBase's `messages` collection `media` field
- This is a significant backend addition. If it blocks, the attach button can be stubbed in the frontend (UI present but sends text-only for now) and the backend work tracked separately.

**Gate:** `cd web && pnpm check`. In browser: verify the "+" button appears next to the input. Verify tapping it shows Camera/Gallery options. Verify selecting a photo sends it as a WhatsApp media message (if backend is ready) or shows a "em breve" toast (if stubbed). Backend: `cd api && go test ./...` passes.

## Consequences

- The operator gets one unified input bar instead of two confusing text fields. The mode toggle makes the current purpose explicit.
- Message selection for generation gives the operator fine-grained control over what context feeds the AI, including images and video descriptions from the chat.
- The full-screen review overlay makes post editing a focused activity instead of something squeezed into a small bottom panel.
- Multi-select on ideas lets the operator send multiple suggestions at once without going through generate-review-send three times.
- The attach button completes the WhatsApp-like experience, letting the operator handle media without leaving the page.
- The send endpoint needs a media upload extension (Wave 5). This is the only backend change; Waves 1-4 are frontend-only.
- The `quickReply` state and its dedicated input are removed. All reply/send functionality goes through the unified bar.
