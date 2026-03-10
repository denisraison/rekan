# PEP 018: Fast Deploys from Local Machine

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | In Progress      |
| **Created** | 2026-03-10       |

## Context

Every deploy rebuilds the Go API and SvelteKit frontend from source on the production server, a Hetzner CAX11 with 2 ARM64 vCPUs and 4 GB RAM. A full rebuild takes several minutes. The server has no binary cache, so even unchanged dependencies are rebuilt after garbage collection.

The deploy command today:

```bash
git tag v0.x.x && git push origin v0.x.x
cd ../infra && nix flake lock --override-input rekan github:denisraison/rekan/v0.x.x
git add flake.lock && git commit && git push
ssh root@46.225.161.186 "nixos-rebuild switch --flake github:denisraison/infra#prod --refresh"
```

The server fetches the flake from GitHub, evaluates it, builds everything from source, then activates. All compilation happens on 2 tiny ARM cores.

The local dev machine is NixOS x86_64 with 20 cores and 62 GB RAM, 10x the server.

### Approaches tested

**binfmt/QEMU emulation** (`boot.binfmt.emulatedSystems = ["aarch64-linux"]`): registers QEMU as a transparent handler for aarch64 binaries. Nix builds aarch64 derivations locally under emulation. Result: 4m27s cold, 4m29s incremental. Too slow, every instruction is emulated.

**Nix cross-compilation** (`pkgsCross.aarch64-multiplatform`): Go cross-compiles natively on x86_64 using a cross-toolchain that Nix provides. The Go compiler runs at full native speed, only the output targets aarch64. Result: **1m43s cold** (includes one-time Go cross-compiler build), **40s incremental**. 6.7x faster than binfmt.

**Building full NixOS closure locally**: even with cross-compiled API, NixOS system closures contain hundreds of small aarch64 derivations (systemd unit scripts, config files) that require binfmt to build on x86_64. This makes it impossible to build the full closure locally without QEMU registered.

A micro VM (Firecracker/etc.) would not help because KVM only accelerates VMs matching the host architecture. An aarch64 VM on x86_64 still uses QEMU, same speed as binfmt.

### Decision

Cross-compile the heavy packages (Go API) locally, push them to the server's Nix store via `nix copy`, then run `nixos-rebuild` on the server. The server finds the expensive packages already in its store and only needs to build trivial NixOS scripts natively (seconds). This avoids both slow remote Go compilation and the need for binfmt locally.

### Goal

Deploy goes from minutes of remote Go compilation to: cross-build (~40s) + upload (seconds) + server activation (seconds under a minute total).

## Wave 1: Cross-compilation from x86_64

Add a cross-compiled aarch64 API package that builds natively on x86_64 using `pkgsCross.aarch64-multiplatform`. No QEMU, no binfmt, no emulation.

### Tasks

- [x] Add `api-cross-aarch64` package to `flake.nix` using `pkgsCross.aarch64-multiplatform.buildGoModule`
- [x] Extract shared `buildApi` helper to avoid duplication between native and cross packages
- [x] Verify: `nix build .#packages.x86_64-linux.api-cross-aarch64` produces an aarch64 binary (confirmed: `ELF 64-bit LSB executable, ARM aarch64`)
- [x] Benchmark: 1m43s cold, 40s incremental (vs 4m27s/4m29s with binfmt)
- [x] Fix pnpm deps hash in `flake.nix`

### Gate

`nix build .#packages.x86_64-linux.api-cross-aarch64` succeeds without binfmt registered. `file result/bin/api` shows `aarch64`.

## Wave 2: Local build + push deploy flow

Cross-build the expensive packages locally, push to server, let `nixos-rebuild` on the server activate using the pre-populated store paths.

### Architecture

The infra flake gets two NixOS configurations:

- **`prod`** — overrides prod's API and web with x86_64 cross-compiled packages via `lib.mkForce`. Used by the deploy script to build packages locally. The server's `nixos-rebuild` also uses this config: it finds the cross-compiled packages already in its store (pushed via `nix copy`) and only builds trivial NixOS scripts natively.
- **`prod-remote`** — no overrides, uses native aarch64 packages. Fallback for server-side rebuilds when away from the workstation.

Staging stays on native aarch64 packages (unchanged by deploy).

### Tasks

- [x] Add `prod` and `prod-remote` NixOS configs to infra flake
- [x] `prod` config overrides prod instance with cross-compiled API + web
- [x] Update `make deploy` in the Makefile
- [ ] Test end-to-end: `make deploy TAG=v0.x.x`

### Deploy flow

```bash
make deploy TAG=v0.4.3
```

1. Tag and push to GitHub
2. Lock infra flake to the new tag
3. Cross-build API (~40s incremental) and web (cached) locally via infra flake
4. `nix copy` both to the server
5. Commit and push infra (so server can pull the updated flake)
6. `nixos-rebuild switch` on the server (finds heavy packages in store, only builds trivial scripts)

### Gate

`make deploy TAG=test` completes. `ssh root@46.225.161.186 "systemctl status rekan-prod"` shows active. Total deploy time under 2 minutes.

## Consequences

- Deploys shift from minutes of remote Go compilation to: cross-build (~40s incremental) + upload (seconds) + server activation (seconds).
- The server still runs `nixos-rebuild`, preserving full NixOS declarative model. It just skips the expensive builds because the store paths already exist.
- No external services needed. No Cachix, no GitHub Actions, no CI. The local machine is the builder.
- No binfmt/QEMU needed on the local machine. Cross-compilation runs at native x86_64 speed.
- Graceful degradation: if away from the dev machine, `ssh root@server "nixos-rebuild switch --flake github:denisraison/infra#prod-remote --refresh"` still works (slow but functional).
- The infra git commit happens before `nixos-rebuild` so the server can pull it, but after successful local build.
- The web package (static HTML/CSS/JS) is architecture-independent. Built on x86_64, served on aarch64.
- Staging is unaffected by prod deploys (separate flake input, native aarch64 build).
- Future option: add Cachix or CI later for multi-developer setups. This PEP focuses on the single-developer fast path.
