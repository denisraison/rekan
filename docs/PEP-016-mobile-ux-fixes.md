# PEP-016 — Mobile UX Fixes

**Status:** In Progress
**Date:** 2026-03-06

## Context

Several UX issues degrade the mobile experience for our target audience (50+, low digital literacy). The generate flow has a navigation trap where selecting one idea to review destroys the other options. WhatsApp status flashes "disconnected" on every app switch (camera, gallery). File attachments show raw filenames. The PWA install prompt never appears.

## Waves

### Wave 1 — Generate Flow Navigation

**Goal:** Let the operator preview an idea and go back to the list without losing the other options.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**Problem:** When the user taps "Revisar e enviar" on one idea, the code sets `ideaDrafts = null` and `selectedIdeas = new Set()` (lines ~2283-2286). The review overlay opens, but there is no way back. The other ideas are gone.

**Fix:**
- [x] Do NOT null out `ideaDrafts` or reset `selectedIdeas` when entering the review overlay. Only clear them after the user actually sends or explicitly discards.
- [x] The "Voltar" button in the review overlay should set `result = null` (which hides the overlay via the existing `$effect`) and return to the idea list.
- [x] The "Descartar" button and successful send should clear `ideaDrafts` to fully exit the flow.

**Notes:**
- The Voltar button is shared between single-post and ideas flows. Added conditional: if `ideaDrafts` exists (ideas flow), sets `result = null` to return to ideas list; otherwise hides the overlay as before (single-post flow keeps the "Ver post gerado" chip).
- Playwright tests required mocking the WhatsApp SSE stream (`page.route`) in `auth.setup.ts` and `helpers.ts` to prevent the QR code overlay from blocking UI.
- Changed `playwright.config.ts` to use `pnpm dev:mock` so generate endpoints return mock data without API keys.

**Gate:**

1. [x] `cd web && pnpm check`
2. [x] Update `tests/multi-select-ideas.spec.ts`:
   - [x] **New test:** "Voltar from review returns to idea list"
   - [x] **New test:** "review different idea after Voltar"
   - [x] **Update existing test:** "selecting one idea shows review button, opens overlay"
3. [x] Update `tests/post-review-overlay.spec.ts`:
   - [x] **New test:** "Descartar from 3-ideas flow clears ideaDrafts"
4. [x] `cd web && npx playwright test multi-select-ideas post-review-overlay` — 13 passed

### Wave 2 — WhatsApp Status Grace Period + Remove Filename from Attachments

**Goal:** Stop the false "WhatsApp desconectado" flash on app resume. Remove the raw filename from attachment previews.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**2a: WhatsApp status grace period**

When the SSE connection drops (user switches to camera), the catch block immediately sets `waConnected = false` and the red banner appears. Add a 5-second grace period: start a timer on disconnect, only show the banner if the SSE does not reconnect within that window. Cancel the timer on reconnect.

**2b: Remove filename from attachment preview**

The attachment preview card shows `{attachedFile?.name}` (line ~2612). The thumbnail alone is sufficient. Remove the filename `<span>`.

**Gate:**

1. `cd web && pnpm check`
2. Update `tests/attach-button.spec.ts`:
   - **New test:** "attachment preview shows no filename" — attach a photo, verify `getByAltText('Anexo')` is visible, verify no text content matching the filename pattern (e.g., `test-attach.png`) appears in the preview container.
3. `cd web && npx playwright test attach-button` — all pass
4. Manual verification for 2a (cannot automate visibility change + SSE drop): open app, switch to camera, return, no red banner flashes. Stop the backend, wait 5+ seconds, verify the banner appears.

### Wave 3 — PWA Install Prompt

**Goal:** Make the Chrome install prompt appear, and add a custom in-app install banner.

**Files:** `web/vite.config.ts`, `web/src/routes/(app)/operador/+page.svelte`

**3a: Fix service worker config**

The `navigateFallback` is set to `/200.html` which SvelteKit does not generate. Investigate what the adapter actually outputs and fix or remove this setting. Verify the service worker registers correctly in production.

**3b: Custom install banner**

Listen for the `beforeinstallprompt` event, stash the event, and show a banner in Portuguese ("Instalar Rekan no seu celular") with a single button. Show once per session, remember dismissal in localStorage. Style it as a subtle top bar, not a modal.

**Gate:**

1. `cd web && pnpm check`
2. `cd web && pnpm build` — service worker generates without errors
3. `cd web && npx playwright test` — full suite passes (no regressions)
4. Manual verification on deployed environment (PWA installability requires real HTTPS):
   - Open in Chrome on Android. Verify the custom install banner appears.
   - Tap "Instalar", verify the native install flow triggers.
   - Dismiss the banner, reload, verify it does not reappear (localStorage persists dismissal).
