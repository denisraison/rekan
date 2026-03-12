# PEP-024: Merge eval into api module

**Status:** In Progress
**Date:** 2026-03-12

## Context

The `eval/` directory is a separate Go module (`github.com/denisraison/rekan/eval`) that `api/` depends on via a `replace ../eval` directive. This made sense early on when eval was a standalone tool, but today the two are tightly coupled:

- `api/` imports eval types (`BusinessProfile`, `Post`, `Role`, etc.) across 6 production files and 6 test files
- eval's only dependency beyond stdlib is BAML, which the app needs anyway
- The BAML prompts and generated client define the core content generation interface, not just evaluation tooling
- Nobody outside this repo consumes the eval module

The separate module creates friction: two `go.mod`/`go.sum` to maintain, the `replace` hack that breaks normal Go tooling, and the conceptual split between "content generation" and "the app" when they're really the same thing.

We'll merge eval into `api/` as `internal/content`, move both binaries under `cmd/`, and delete the eval module entirely.

**Package name: `content` not `eval`.** The package does content generation, profile extraction, quality checks, and judging. "eval" was the name for the tooling, not the domain. `content` is clearer.

**Under `internal/`.** No external consumers. Matches the rest of the api codebase.

**PocketBase under `cmd/rekan/`.** Current `api/main.go` moves to `cmd/rekan/main.go`, following standard Go multi-binary layout.

**BAML stays with content.** `baml_src/` and `baml_client/` move into `internal/content/`. The generator config output path gets updated.

**testdata moves with the package.** The business profile JSON fixtures move to `internal/content/testdata/`.

Target layout after all waves:

```
api/
  cmd/
    rekan/main.go              # PocketBase server (current api/main.go)
    eval/main.go               # eval runner CLI (current eval/cmd/eval/main.go)
  internal/
    content/                   # current eval/*.go, renamed package
      content.go, heuristic.go, judge.go, hooks.go, profile.go, role.go
      baml_src/                # BAML source files (unchanged)
      baml_client/             # generated BAML client (unchanged)
      testdata/                # business profile JSON fixtures
    service/                   # already exists
    http/                      # already exists
    ...
  go.mod                       # single module, no replace directive
  go.sum
```

## Waves

### Wave 1: Move PocketBase to cmd/rekan and eval package to internal/content [DONE]

Move the PocketBase entry point from `api/main.go` to `api/cmd/rekan/main.go`. Then move all `eval/*.go` files to `api/internal/content/`, renaming the package from `eval` to `content`. Move `baml_src/`, `baml_client/`, and `testdata/` alongside.

The package rename touches every file: `package eval` becomes `package content`, and all internal references between files in the package stay the same since they use unqualified names. The BAML generated client (`baml_client/`) has a hardcoded package declaration that also needs updating, but regenerating after updating `generators.baml` output path handles this.

Files created/moved:
- [x] `api/cmd/rekan/main.go` (from `api/main.go`)
- [x] `api/internal/content/*.go` (from `eval/*.go`)
- [x] `api/internal/content/baml_src/` (from `eval/baml_src/`)
- [x] `api/internal/content/baml_client/` (from `eval/baml_client/`)
- [x] `api/internal/content/testdata/` (from `eval/testdata/`)
- [x] Update `generators.baml` output path to point to new location

Do not update imports in `api/` consumers yet. Do not delete `eval/` yet. The old module stays in place so existing code still compiles until Wave 2 rewires everything.

Notes:
- The baml_client generated code had internal import paths (`github.com/denisraison/rekan/eval/baml_client/...`) that were updated to the new path (`github.com/denisraison/rekan/api/internal/content/baml_client/...`) via find-and-replace rather than regeneration, since the BAML generator tooling wasn't invoked.
- `api/main.go` was copied (not moved) to `api/cmd/rekan/main.go` so existing code continues to compile. The original will be removed in Wave 2.

**Gate:**
- [x] `cd api && go build ./cmd/rekan` exits 0
- [x] `cd api && go build ./internal/content` exits 0
- [x] `cd api && go test ./internal/content/...` passes
- [x] `ls api/internal/content/baml_src/*.baml` shows BAML files present

### Wave 2: Rewire imports, move eval CLI, delete old module

Update all `api/` files that import the eval module to import `internal/content` instead. Move `eval/cmd/eval/main.go` to `api/cmd/eval/main.go` and update its imports. Remove the `replace` directive from `api/go.mod`. Delete `eval/go.mod`, `eval/go.sum`, and the now-empty `eval/` directory. Update Makefile targets: `make dev` uses `go run ./cmd/rekan`, eval commands use `go run ./cmd/eval`.

Import changes are mechanical. Every `import "github.com/denisraison/rekan/eval"` becomes an import of the `internal/content` package path. Type references change from `eval.Post` to `content.Post` (or whatever alias is clearest). Check for named imports like `import eval "..."` and update them.

Files changed:
- `api/go.mod` (remove replace directive and eval dependency)
- `api/cmd/rekan/main.go` (update imports)
- `api/internal/service/content.go` (update imports)
- `api/internal/http/handlers/deps.go` (update imports)
- `api/internal/http/handlers/generate.go` (update imports)
- `api/internal/whatsapp/handler.go` (update imports)
- `api/internal/operator/operator.go` (update imports)
- All test files referencing eval types
- `api/cmd/eval/main.go` (from `eval/cmd/eval/main.go`, update imports)
- `Makefile` (update dev, eval, eval-judges, test-judges targets)
- Delete `eval/` directory entirely

**Gate:**
- `cd api && go build ./...` exits 0 (all packages compile)
- `cd api && go test ./...` passes (all tests pass)
- `make eval` runs successfully (eval CLI works from new location)
- `grep -r 'replace.*../eval' api/go.mod` returns nothing (replace directive gone)
- `test ! -d eval` confirms old module deleted
- Update `CLAUDE.md` commands section and `eval/CLAUDE.md` (move relevant content to `api/internal/content/CLAUDE.md` or merge into root)

## Consequences

- **Single module simplifies tooling.** `go mod tidy`, `go test ./...`, and IDE navigation work without the `replace` workaround. Dependabot/renovate see one module.
- **Import paths get longer.** External tooling or scripts referencing `eval` package paths will break. Since nothing outside this repo imports it, this only affects the Makefile and CLAUDE.md (handled in Wave 2).
- **BAML regeneration required.** After moving `baml_src/`, the generator output path changes. Any future BAML schema changes need to run the generator from the new location. The `baml_client/` package declaration changes from `baml_client` to whatever the new path dictates (likely stays `baml_client` as a directory name).
- **Git history for eval files becomes harder to trace.** `git log --follow` works for individual files but not directories. This is a one-time cost accepted for cleaner structure going forward.
