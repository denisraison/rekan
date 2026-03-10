# PEP-017 — Cache Hygiene & Static Serving

**Status:** In Progress
**Date:** 2026-03-09

## Context

A Workbox service worker shipped months ago precached all JS/CSS/HTML in Cache Storage. On mobile (especially standalone PWA mode), the browser rarely checks for SW updates, so phones served months-old stale assets with no way to bust the cache. We removed Workbox and the PWA install banner, replaced the SW with a self-destructing stub, and added `Clear-Site-Data` on `/sw.js`. This PEP tracks the remaining cleanup and improvements.

## Current State

**What's deployed (v0.4.15):**
- `sw.js`: self-destructing stub (clears caches, unregisters itself)
- `app.html`: inline script that calls `getRegistrations()` → `update()` + `unregister()` and `caches.delete()` on every cache
- Caddy serves `sw.js` with `Clear-Site-Data: "cache", "storage"` header
- Caddy serves `_app/immutable/*` with `immutable, max-age=31536000`
- Caddy serves everything else with `no-cache`
- `vite-plugin-pwa` removed, no Workbox in build output

**Known issues:**
1. Race condition in `app.html`: `r.update()` and `r.unregister()` called without awaiting, unregister can win
2. No response compression (gzip/brotli) configured in Caddy
3. `manifest.webmanifest` still references `display: standalone` but there's no functional SW
4. `Clear-Site-Data: "storage"` not supported on iOS Safari, only client-side JS cleanup works there

## Waves

### Wave 1 — Fix Race Condition & Add Compression

**Goal:** Fix the update/unregister race in app.html. Add gzip compression to Caddy.

**Files:** `web/src/app.html`, `nix/module.nix`

**Changes:**
- [x] Fix app.html: await `r.update()` before calling `r.unregister()` using `.finally()`
- [x] Add `encode gzip` to Caddy's site block for text compression
- [ ] ~~Consider `precompress: true` in SvelteKit's adapter-static~~ Skipped, `encode gzip` sufficient for our payload sizes

**Gate:** `curl -sI --compressed https://rekan.com.br/ | grep content-encoding` returns `gzip` or `br`.

### Wave 2 — Cleanup Transition Code (after 2026-07-01)

**Goal:** Remove the self-destructing SW and all cleanup code once old Workbox SWs are gone.

**Files:** `web/static/sw.js`, `web/src/app.html`, `nix/module.nix`, `web/static/manifest.webmanifest`

**Changes:**
- [ ] Remove `web/static/sw.js` entirely
- [ ] Remove the inline SW cleanup script from `app.html`
- [ ] Remove the `handle /sw.js` block and `Clear-Site-Data` header from `nix/module.nix`
- [ ] Decide: keep `manifest.webmanifest` (for mobile "Add to Home Screen" shortcut) or remove it
- [ ] If keeping manifest, change `display` from `standalone` to `browser` (standalone without a SW gives a degraded experience)

**Gate:** No service worker registrations visible in Chrome DevTools Application tab on fresh visit.
