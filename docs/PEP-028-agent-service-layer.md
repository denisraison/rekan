# PEP-028: Agent service layer extraction

**Status:** Done
**Date:** 2026-03-14
**Depends on:** PEP-022

## Context

Tools in `agent/tools.go` mix three concerns: JSON parsing, business logic, and response formatting. Read tools (`findCustomer`, `listCustomers`, `findPost`, `listPosts`, `recentActivity`) query the database directly, build string responses inline, and have no service layer at all. Write tools are slightly better: they parse JSON, then delegate to `router.go` functions, which do the actual DB work. But `router.go` itself is an odd middle layer. It's not a service (it formats operator-facing Portuguese strings, handles disambiguation, references `operatorName`). It's not a handler either. It's agent-specific business logic that lives outside the tool but isn't reusable.

The analogy to HTTP handlers is exact. PEP-022 extracted business logic from HTTP handlers into `internal/service/` so both handlers and the agent could call the same functions. That worked for `service.GeneratePosts` and `service.SendTextMessage`. But the agent's own operations (customer CRUD, post approve/reject) never went through that extraction. `router.go` became the agent's private service layer, duplicating patterns that exist in `service/` but with agent-specific formatting baked in.

Problems this causes:

1. **Read tools can't be reused.** If we add a REST API for customers or a scheduled job that lists pending posts, the query logic has to be rewritten. The formatting (Portuguese strings, `---` separators) is welded to the query.

2. **router.go mixes concerns.** `executeCustomerUpdate` does fuzzy matching, field updates, DB save, AND formats the response string. The response string includes `operatorName`, making it impossible to call from a non-agent context.

3. **Testing requires the full tool executor.** To test "find customer by fuzzy name" you need a `ToolExecutor` with a `core.App`. The fuzzy matching logic (`findBusinessRecords`, `findDuplicate`, `normalizeForMatch`) is reusable but trapped in the agent package.

4. **The agent package is 1100+ lines of tools.go + router.go.** Most of it is business logic that doesn't depend on the agent at all.

The fix: tools become thin dispatchers (parse JSON, call service, format response), like HTTP handlers became after PEP-022. Business logic moves to `internal/service/`. `router.go` dissolves.

## Waves

### Wave 1: Extract customer service

Move customer operations out of `router.go` into `service/business.go` (which already exists with `FindBusiness` helpers from PEP-022).

**What moves:**

| From `router.go` | To `service/business.go` | Change |
|---|---|---|
| `executeCustomerCreate` | `CreateBusiness(app, params) (*core.Record, error)` | Return record instead of formatted string. Drop `operatorName` from signature. |
| `executeCustomerUpdate` | `UpdateBusiness(app, name, params) (*core.Record, []string, error)` | Return record + list of updated field names. Caller formats the message. |
| `executeCustomerPause` | `PauseBusiness(app, name) (*core.Record, error)` | Return record. Caller adds reason text. |
| `findBusinessRecords` | `FindBusinessByName(businesses, query) []*core.Record` | Same logic, exported. |
| `findDuplicate` | `FindDuplicate(businesses, name) *core.Record` | Same logic, exported. |
| `disambiguate` | stays in agent (it's presentation) | — |
| `loadActiveBusinesses` | `ListActiveBusinesses(app) []*core.Record` | Already partially exists. |
| `normalizeForMatch` | `normalizeForMatch` (unexported in service) | Pure string helper. |

**Tool methods shrink to:**
```go
func (te *ToolExecutor) createCustomer(input json.RawMessage, operatorName string) toolResult {
    // 1. Parse JSON into params
    // 2. Validate
    // 3. Check duplicate via service.FindDuplicate
    // 4. Call service.CreateBusiness
    // 5. Format response string with operatorName
    return toolResult{Text: fmt.Sprintf(...), IsWrite: true}
}
```

**Read tools also thin out.** `findCustomer` currently queries + formats. After:
```go
func (te *ToolExecutor) findCustomer(input json.RawMessage) string {
    // 1. Parse JSON
    // 2. Call service.FindBusinessByName
    // 3. Format records into tool response string
}
```

The formatting stays in the tool (it's presentation for Claude, not business logic). The query logic moves to the service.

**Files:**
- `api/internal/service/business.go`: add `CreateBusiness`, `UpdateBusiness`, `PauseBusiness`, `FindBusinessByName`, `FindDuplicate`, `ListActiveBusinesses`
- `api/internal/agent/tools.go`: rewrite tool methods to call service functions
- `api/internal/agent/router.go`: remove customer functions, keep post functions for Wave 2
- `api/internal/agent/router_test.go` (if exists): move relevant tests to `service/business_test.go`

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] `cd api && go test ./internal/service/...` passes
- [x] `cd api && go test ./internal/agent/...` passes
- [x] No customer DB queries remain in `agent/tools.go` (grep for `RecordQuery.*businesses` in tools.go returns nothing)
- [x] `findBusinessRecords` and `findDuplicate` no longer exist in the agent package

### Wave 2: Extract post service and dissolve router.go

Move post operations out of `router.go` into `service/content.go` (which already has `GeneratePosts`).

**What moves:**

| From `router.go` | To `service/content.go` | Change |
|---|---|---|
| `executePostApprove` | `ApprovePost(app, postID) (*core.Record, error)` | Return record. Drop `operatorName`. Caller formats message and handles WhatsApp send. |
| `executePostReject` | `RejectPost(app, postID, feedback) (*core.Record, error)` | Return record. Caller formats message. |
| `executePostGenerate` | Already calls `service.GeneratePosts`. Inline the thin remaining logic into the tool method. | Fuzzy name matching uses `service.FindBusinessByName` from Wave 1. |

**Read tools for posts also thin out.** `listPosts` has a 70-line method with query building, fuzzy matching, and formatting. After extraction, the query logic moves to `service/content.go` as `ListPosts(app, filter) ([]*core.Record, error)`.

**What remains in router.go after this wave:** Nothing. Delete it. `LogAction` moves to `service/` or stays in agent as a small helper (it's logging, not business logic). `resolveBusinessName` is a presentation helper, stays in agent or moves to a shared formatter.

**Files:**
- `api/internal/service/content.go`: add `ApprovePost`, `RejectPost`, `ListPosts`, `FindPost`
- `api/internal/agent/tools.go`: rewrite post tool methods to call service functions
- `api/internal/agent/router.go`: delete
- `api/internal/agent/router.go` helpers (`LogAction`, `resolveBusinessName`): relocate

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] `cd api && go test ./internal/service/...` passes
- [x] `cd api && go test ./internal/agent/...` passes
- [x] `router.go` no longer exists in the agent package
- [x] `tools.go` contains no direct `app.RecordQuery` calls (all queries go through service)
- [x] `cd web && npx playwright test --project=default` passes (end-to-end still works)

## Consequences

- Tools become ~15 lines each: parse, validate, call service, format. Same pattern as HTTP handlers post-PEP-022.
- `service/` becomes the single source of truth for business operations. New consumers (scheduled jobs, REST endpoints, CLI tools) call the same functions.
- The agent package shrinks significantly. `router.go` disappears. `tools.go` drops from ~650 lines to ~350.
- Fuzzy name matching is reusable outside the agent.
- Testing improves: service functions can be tested with just `core.App`, no `ToolExecutor` setup needed.
- Trade-off: one more layer of indirection for simple operations like "set reviewed = true". For a 2-line DB update, calling through a service function feels like ceremony. Worth it for consistency.
- Trade-off: formatting splits across two places. The service returns data, the tool formats it for Claude, the HTTP handler formats it for JSON. This is the correct split but means you read two files to understand a full tool response.
