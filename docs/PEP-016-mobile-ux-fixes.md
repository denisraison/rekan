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

- [x] When the SSE connection drops (user switches to camera), the catch block immediately sets `waConnected = false` and the red banner appears. Add a 5-second grace period: start a timer on disconnect, only show the banner if the SSE does not reconnect within that window. Cancel the timer on reconnect.

**2b: Remove filename from attachment preview**

- [x] The attachment preview card shows `{attachedFile?.name}` (line ~2612). The thumbnail alone is sufficient. Remove the filename `<span>`.

**Notes:**
- Added `waDisconnectTimer` variable. On SSE error, starts a 5s timeout before setting `waConnected = false`. On reconnect (SSE callback), the timer is cancelled. Timer is also cleaned up in `onDestroy`.

**Gate:**

1. [x] `cd web && pnpm check`
2. [x] Update `tests/attach-button.spec.ts`:
   - [x] **New test:** "attachment preview shows no filename"
3. [x] `cd web && npx playwright test attach-button` — 9 passed
4. Manual verification for 2a (cannot automate visibility change + SSE drop): open app, switch to camera, return, no red banner flashes. Stop the backend, wait 5+ seconds, verify the banner appears.

### Wave 3 — PWA Install Prompt

**Goal:** Make the Chrome install prompt appear, and add a custom in-app install banner.

**Files:** `web/vite.config.ts`, `web/src/app.html`, `web/src/routes/(app)/operador/+page.svelte`

**3a: Fix service worker registration**

- [x] The `navigateFallback` is set to `/200.html` which SvelteKit does not generate. Investigate what the adapter actually outputs and fix or remove this setting. Verify the service worker registers correctly in production.
- [x] VitePWA's HTML injection (`transformIndexHtml`) does not work with SvelteKit (no `index.html` in the Vite pipeline). The built HTML had no `<link rel="manifest">` or SW registration script. Fix by adding them manually to `app.html`.

**3b: Custom install banner**

- [x] Listen for the `beforeinstallprompt` event, stash the event, and show a banner in Portuguese ("Instalar Rekan no seu celular") with a single button. Show once per session, remember dismissal in localStorage. Style it as a subtle top bar, not a modal.

**Notes:**
- 3a: `adapter-static` with `fallback: '200.html'` does generate `200.html` in the build output. The existing `navigateFallback: '/200.html'` config is correct. No change needed there.
- 3a: VitePWA's `injectRegister` does not work with SvelteKit, so set `injectRegister: false` in `vite.config.ts`. Added `<link rel="manifest" href="/manifest.webmanifest" />` to `<head>` and an inline SW registration script to `<body>` in `app.html`. The script tries `/sw.js` (production) and falls back to `/dev-sw.js?dev-sw` (dev mode with `devOptions: { enabled: true }`).
- 3b: Banner listens for `beforeinstallprompt`, stashes the event. Shows a fixed top bar with "Instalar" and "Fechar" buttons. Dismissal saved to `rekan_install_dismissed` in localStorage. Cleanup in `onDestroy`.

**Gate:**

1. [x] `cd web && pnpm check`
2. [x] `cd web && pnpm build` — service worker generates without errors, `200.html` contains manifest link and SW script
3. [x] `cd web && npx playwright test` — full suite passes (41 passed, no regressions)
4. [x] New e2e test `tests/pwa.spec.ts`: verifies SW registers and manifest link is present
5. Manual verification on deployed environment (PWA installability requires real HTTPS):
   - Open in Chrome on Android. Verify the custom install banner appears.
   - Tap "Instalar", verify the native install flow triggers.
   - Dismiss the banner, reload, verify it does not reappear (localStorage persists dismissal).

### Wave 4 — Small Screen / Zoomed-in Accessibility

**Goal:** Make the profile recording card, recording bar, and message input bar usable on narrow viewports (320px) and when the user has OS-level zoom or large text enabled.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**4a: Profile record card (idle mic button)**

**Problem:** The card uses `gap-4 p-5` with a fixed 72px circular button. On a 320px screen (or zoomed), the button + text overflow or feel cramped. The text gets very little horizontal space.

**Fix:**
- [x] Reduce the mic button to 56px on small screens (keep 72px on md+). Use a CSS class with a media query instead of inline `width`/`height`.
- [x] Reduce card padding from `p-5` to `p-3` on small screens (`p-3 md:p-5`).
- [x] Reduce gap from `gap-4` to `gap-3` on small screens (`gap-3 md:gap-4`).
- [x] Ensure the text block has `min-w-0` so it can shrink and wrap properly.

**4b: Recording bar**

**Problem:** The recording bar has fixed `height: 72px` and 72px-wide side buttons. On narrow screens, the center timer/label section gets squeezed, and the overall bar is taller than necessary.

**Fix:**
- [x] Reduce bar height to 56px and side buttons to 56px on small screens (keep 72px on md+). Use a CSS class or `@media` in the `<style>` block.
- [x] Reduce timer font-size from 26px to 20px on small screens.
- [x] Hide the "Gravando" label on very narrow screens (below 360px) since the blinking red dot already indicates recording.

**4c: Message input bar**

**Problem:** The send buttons ("Gerar", "Enviar") use `px-5 py-3` which takes too much horizontal space. Combined with the attach button and input padding, the text input gets very narrow on 320px screens.

**Fix:**
- [x] Reduce send button horizontal padding from `px-5` to `px-3` on small screens (`px-3 md:px-5`).
- [x] Reduce input horizontal padding from `px-4` to `px-3` on small screens (`px-3 md:px-4`).
- [x] Reduce the container horizontal padding from `px-4` to `px-3` on small screens (`px-3 md:px-4`).

**4d: Analyzing bar**

**Problem:** The analyzing bar also uses fixed `height: 72px`, inconsistent with the recording bar fix.

**Fix:**
- [x] Match the recording bar: 56px on small screens, 72px on md+.

**Gate:**

1. [x] `cd web && pnpm check`
2. [x] `cd web && npx playwright test` — full suite passes (44 passed, no regressions)
3. [x] New e2e test `tests/small-screen.spec.ts`:
   - Viewport set to 320x568 (iPhone SE)
   - Profile record card: mic button and text are both visible, nothing overflows
   - Message input bar: input field has at least 120px width, send button is visible
   - Recording bar: timer and buttons are visible and tappable (min 44x44 touch target)
4. Manual verification: open on a real device or Chrome DevTools at 320px width. Verify all interactive elements are reachable and text is readable. Test with OS text size set to "Largest".

**Notes:**
- Used CSS classes (`mic-btn`, `rec-bar`, `rec-side-btn`, `rec-timer`, `rec-label`) with `@media (min-width: 768px)` to scale from 56px (mobile) to 72px (desktop). Avoids inline style overrides.
- Added `@media (max-width: 359px)` to hide the "Gravando" label on very narrow screens.
- The recording bar test verifies the mic button touch target (56x56 >= 44x44 WCAG minimum) since starting an actual recording requires microphone permissions.
- The scrollWidth check (instead of bounding box position) correctly detects horizontal overflow regardless of CSS `overflow: hidden` clipping.
