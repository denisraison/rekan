# PEP 011: NixOS Native Packaging

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | In Progress      |
| **Created** | 2026-02-26       |

## Context

A shared NixOS VPS (Hetzner CAX11, ARM64) hosts multiple apps behind Caddy. The infra repo (`../infra`) owns the server configuration and is app-agnostic: it should not contain app-specific build logic, service definitions, or domain knowledge. Each app is responsible for packaging itself as a Nix flake that exports a NixOS module. The infra repo just adds them as flake inputs and enables them.

Today the infra repo downloads vanilla PocketBase from GitHub releases and generates identical systemd services via `pb-instances.nix`. This was a placeholder to get something running, but every app will be a custom Go binary extending PocketBase with its own routes, integrations, and frontend. Each app needs to build and package itself. The infra repo should not know how to build any of them.

This PEP makes Rekan self-contained for deployment: it exports Nix packages (Go binary, static frontend) and a NixOS module. The infra repo adds `rekan` as a flake input, imports the module, and sets a few options. No app-specific build steps live in infra.

### Current state

- `flake.nix` only exports `devShells.default` (dev tooling)
- `api/go.mod` uses `replace ../eval` for the local eval module
- `web/svelte.config.js` uses `adapter-auto` (not deployable)
- Infra repo fetches vanilla PocketBase binary, not the custom Rekan binary
- No way to deploy the frontend

### Target state

- `flake.nix` exports `packages.api` (Go binary) and `packages.web` (static frontend)
- `flake.nix` exports `nixosModules.default` (systemd service + Caddy vhost)
- Infra repo adds `rekan` as a flake input, imports the module, sets `services.rekan.enable = true` and a few options
- The pattern is reusable: any future app can follow the same structure (export a NixOS module, infra repo imports it)

## Wave 1: SvelteKit static build

Switch the frontend from `adapter-auto` to `adapter-static` with SPA fallback. The app is client-rendered with PocketBase as the API backend, so static output with a fallback `200.html` works. Caddy will serve the static files and proxy `/api` to the Go backend.

### Tasks

- [x] Install `@sveltejs/adapter-static` in `web/`, remove `@sveltejs/adapter-auto`
- [x] Update `web/svelte.config.js` to use `adapter-static` with `fallback: '200.html'`
- [x] Add a root `+layout.ts` with `export const prerender = true` and `export const ssr = false`
- [x] Verify `pnpm build` produces `web/build/` with static files
- [x] Test locally: serve `web/build/` with a static server, confirm the app works against the API

### Gate

`cd web && pnpm build` succeeds. `ls web/build/200.html` exists. App loads in browser from the static build.

## Wave 2: Nix package outputs

Add `packages.api` and `packages.web` to `flake.nix`. The Go binary must be built from the repo root since `api/go.mod` has a `replace ../eval` directive. The frontend is a standard Node build.

### Tasks

- [x] Add `packages.api` using `buildGoModule.override { go = go_1_26; }` with `modRoot = "api"`, fileset source including `api/` and `eval/`
- [x] Add `packages.web` using `stdenvNoCC.mkDerivation` with `pnpmConfigHook` and `fetchPnpmDeps`
- [x] Verify both build on the dev machine: `nix build .#api` and `nix build .#web`
- [x] ~~Cross-compilation for aarch64~~: not possible because BAML's Go bindings require CGO (platform-specific native code). The binary must be built natively on aarch64, which `nixos-rebuild switch` on the VPS handles automatically.

### Gate

`nix build .#api` produces a binary at `result/bin/api`. `nix build .#web` produces static files at `result/`. Both build cleanly without network access (Nix sandbox).

## Wave 3: NixOS module

Create a NixOS module in the Rekan repo that declares the systemd service, env file, and Caddy virtualHost. This module is what the infra repo (or any NixOS config) imports to run Rekan. The infra-side changes (adding the flake input, enabling the module, removing the vanilla PocketBase entry) are trivial and out of scope for this PEP.

### Tasks

- [x] Create `nix/module.nix` that defines `services.rekan` with options for `enable`, `domain`, `port`, `envFile`
- [x] The module creates a systemd service running the custom Go binary with `DynamicUser`, `StateDirectory`, `ProtectSystem = "strict"`, `ProtectHome`, `NoNewPrivileges`, `PrivateTmp`
- [x] The module adds a Caddy virtualHost: serves static files from `packages.web` at the root, reverse-proxies `/api/*` and `/_/*` (PocketBase routes) to the Go backend
- [x] Export `nixosModules.default` from `flake.nix`
- [x] Test locally with `nixos-rebuild build --flake .#` in a VM or container, or verify the module evaluates: `nix eval .#nixosModules.default --apply 'f: builtins.typeOf f'` returns "lambda"

### Gate

`nix eval .#nixosModules.default` does not error. Module options are well-typed.

## Wave 4: Infra cleanup

Strip the placeholder scaffolding from the infra repo. Once Rekan brings its own NixOS module, the infra repo no longer needs the vanilla PocketBase derivation, `pb-instances.nix`, or the generated systemd/Caddy logic. The infra repo becomes a thin flake that owns server basics (hardware, networking, users, Caddy enable, firewall) and imports app modules.

### Tasks

- [x] Convert infra repo to a flake with `rekan` as an input
- [x] Delete the `pocketbase` derivation from `configuration.nix` (the `pkgs.stdenv.mkDerivation` block fetching the vanilla binary)
- [x] Delete `pb-instances.nix`
- [x] Delete the `systemd.services` block generated from `pbInstances` in `configuration.nix`
- [x] Simplify `services/caddy.nix`: remove the `pbInstances` import and generated `caddyHosts`. Keep only `services.caddy.enable = true` and `networking.firewall.allowedTCPPorts = [ 80 443 ]`. App modules add their own virtualHosts.
- [x] Delete `scripts/` (empty directory)
- [x] Import `rekan.nixosModules.default` and configure `services.rekan`
- [x] Rename hostname from `postador-prod` to `prod`
- [x] Update `CLAUDE.md` to reflect the new structure
- [ ] `nixos-rebuild switch` on the VPS, verify Rekan is running

### Gate

Infra repo has no app-specific build logic. `configuration.nix` has no PocketBase derivation, no `pb-instances.nix` import. `systemctl status rekan` shows active on the VPS.

## Consequences

- Rekan owns its own deployment definition. The infra repo stays generic, it just imports app modules and sets options. Adding a new app to the VPS means adding a flake input and `services.foo.enable = true`, not writing systemd units and Caddy configs by hand.
- Deployment is `nix flake update rekan && nixos-rebuild switch` in the infra repo. No build scripts, no rsync of binaries, no Docker.
- The frontend is served by Caddy as static files, no Node process in production.
- The Go binary is built by Nix with the exact same toolchain every time. Reproducible across machines.
- The `adapter-auto` to `adapter-static` switch means SSR is gone. All rendering happens client-side. This is fine because the app is behind auth and not SEO-sensitive, except for the terms/landing pages. Those can use `prerender = true` on specific routes if needed.
- Cross-compilation from x86_64 to aarch64 is not possible because BAML's Go bindings require CGO with native code. The Go binary must be built natively on aarch64. `nixos-rebuild switch` on the VPS handles this (builds locally on the target). First build will be slow (~2 min on CAX11), subsequent builds are cached.
