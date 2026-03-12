# PEP-022: API Service Layer

**Status:** In Progress
**Date:** 2026-03-12

## Context

Business logic lives inside HTTP handlers. Every operation (create business, send message, generate post, approve scheduled message) is only reachable via an HTTP request. This was fine when the operator web UI was the only consumer.

PEP-023 introduces a WhatsApp group agent that needs to call the same operations. The agent's action router receives a typed action from BAML (e.g. `CUSTOMER_CREATE`) and needs to execute it against PocketBase and WhatsApp. Right now, the only way to "create a customer" is to POST to an HTTP endpoint, which means either the agent calls its own HTTP API (wasteful, fragile) or we duplicate the logic (worse).

The fix is a service layer that both HTTP handlers and the agent import. Handlers become thin: parse request, call service, format response. The agent calls the same service functions directly.

Secondary benefit: the `whatsapp/handler.go` currently drops group messages (`if evt.Info.IsGroup { return }`). PEP-023 needs group message handling. This PEP restructures the handler to make that extension clean.

## Waves

### Wave 1: Extract business and messaging services

Move business logic out of HTTP handlers into `internal/service/`. Handlers keep HTTP concerns only: parse request body, validate auth, call service, return JSON.

**New files:**

| File | Extracted from | Functions |
|------|---------------|-----------|
| `internal/service/business.go` | `invite.go`, `operator.go` | `FindBusiness(app, id)`, `FindBusinessByPhone(app, phone)`, `FindBusinessByInvite(app, token)`, `CreateBusiness(app, params)`, `UpdateBusiness(app, id, fields)`, `ListActiveBusinesses(app)` |
| `internal/service/invite.go` | `invite.go` | `SendInvite(ctx, app, wa, asaas, businessID, appURL)`, `AcceptInvite(ctx, app, asaas, token, cpfCnpj)`, `CancelAuthorization(ctx, app, asaas, businessID)` |
| `internal/service/message.go` | `send_message.go`, `send_media.go`, `helpers.go` | `SendTextMessage(ctx, app, wa, businessID, text)`, `SendMediaMessage(ctx, app, wa, transcribe, businessID, mediaData, mimeType, caption)`, `StoreOutgoingMessage(app, businessID, phone, msgType, content, media)` |
| `internal/service/content.go` | `generate.go`, `operator.go`, `generate_ideas.go`, `save_proactive.go` | `GeneratePosts(ctx, app, generate, businessID)`, `GenerateFromMessage(ctx, app, genFn, businessID, message, messageID)`, `GenerateIdeas(ctx, app, generate, businessID)`, `SaveProactivePost(app, businessID, post)` |
| `internal/service/schedule.go` | `scheduled_messages.go` | `ListScheduledMessages(app)`, `ApproveScheduledMessage(ctx, app, wa, id)`, `DismissScheduledMessage(app, id)` |

**Approach:**
- Start with the simplest handlers (`scheduled_messages.go`, `send_message.go`) to establish the pattern
- Service functions take `core.App` and specific clients as parameters, not the full `Deps` struct. This keeps them testable and avoids pulling in unrelated dependencies
- Service functions return domain values and errors, not HTTP status codes
- Handlers become ~10-15 lines: decode, validate, call service, encode response
- The `Deps` struct stays in the handlers package. Services don't know about it
- `helpers.go` functions that are pure HTTP helpers (like `formFileData`) stay in handlers. `storeOutgoingMessage` and `simulateTyping` move to the service layer since the agent needs them too
- `revertToInvited` moves into `service/invite.go` as unexported helper

**What doesn't move:**
- `whatsapp.go` (SSE stream, status endpoint): pure HTTP, no business logic
- `webhooks.go` (Asaas webhook): tightly coupled to HTTP request verification, stays as handler but calls service functions for state transitions
- `terms.go`: static content, no logic to extract
- `demo.go`: self-contained demo endpoint, not reused

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] `cd web && pnpm check` passes (frontend unchanged)
- [x] `cd web && npx playwright test --project=default` passes (78/78 passed)
- [x] No handler file imports `domain.Coll*` for write operations (only webhooks.go, which stays as handler per PEP)
- [x] `grep -r 'e.App.Save\|e.App.FindRecordById' api/internal/http/handlers/ | grep -v webhook | grep -v whatsapp.go | grep -v helpers_test` returns 0 matches (down from ~25)

**Notes:**
- `postingtime.Tip` computation moved into `service.SendTextMessage` since the service already reads the business record, avoiding a duplicate DB read in the handler
- `InviteGet` handler still reads the raw business record for the expiry check (`invite_sent_at`), since the service returns a structured `InviteInfo` that doesn't expose the raw timestamp. This is a read-only operation in the handler, consistent with the PEP's "reads for validation are OK" guideline
- `simulateTyping` and `storeOutgoingMessage` moved to service as `SimulateTyping` and `StoreOutgoingMessage` (exported, for use by WhatsApp handler in PEP-023)
- `helpers.go` now contains only `formFileData` (pure HTTP utility)

### Wave 2: Restructure WhatsApp handler for group support

The current `whatsapp/handler.go` processes 1:1 messages and drops group messages. PEP-023 needs group message handling with a different pipeline (intent detection, debounce, agent processing). This wave restructures the handler to dispatch by message context instead of dropping groups.

**Changes to `internal/whatsapp/handler.go`:**
- Rename `handleMessage` to `handleDirectMessage` (clarity)
- Add `handleGroupMessage` function that the PEP-023 agent will implement
- Add a dispatcher that routes based on `evt.Info.IsGroup`
- Extract shared logic (LID resolution, media download, deduplication) into helper functions that both direct and group handlers can use

**New file: `internal/whatsapp/media.go`**
- Move `transcribeAudio`, `processImage`, `processVideo` from `handler.go`
- These are pure media processing, not message handling. Both direct message handler and the future agent need them

**New file: `internal/whatsapp/contacts.go`**
- Move `findOrCreateBusiness`, `refreshProfilePicture`, `extractAndSaveSignal` from `handler.go`
- These are contact management operations, not message handling

**Handler structure after this wave:**
```
internal/whatsapp/
  client.go         # unchanged
  handler.go        # dispatcher + handleDirectMessage (slimmed down)
  group.go          # handleGroupMessage (stub: logs and returns, PEP-023 fills this in)
  media.go          # transcribeAudio, processImage, processVideo
  contacts.go       # findOrCreateBusiness, refreshProfilePicture, extractAndSaveSignal
```

**The `group.go` stub:**
```go
func handleGroupMessage(deps HandlerDeps, evt *events.Message) {
    // PEP-023 implements this. For now, log and return.
    deps.Logger.Debug("whatsapp: group message ignored (agent not configured)")
}
```

The dispatcher in `handler.go` replaces the early `if evt.Info.IsGroup { return }` with a call to `handleGroupMessage`. This means PEP-023 only needs to fill in `group.go` instead of restructuring the handler.

**Gate:**
- [ ] `cd api && go build ./...` compiles
- [ ] `cd api && go vet ./...` passes
- [ ] Existing E2E tests pass (direct message handling unchanged)
- [ ] `handler.go` is under 80 lines (dispatcher + direct message handler, media/contacts extracted)
- [ ] Group messages are logged at debug level (verify with `DEV_MODE=true`, send a message in a test group, check logs)

## Consequences

- HTTP handlers become thin request/response translators. Business logic is reusable.
- The WhatsApp agent (PEP-023) can call service functions directly without HTTP overhead.
- The whatsapp package is structured for group message handling without a full rewrite.
- Trade-off: more files, more function parameters. Services take explicit dependencies instead of a single Deps struct. This is intentional, each function declares exactly what it needs.
- Trade-off: some handlers will look almost empty (3-4 lines). That's correct. The handler's job is HTTP, not business logic.
- No new dependencies. No interface abstractions beyond what's needed. Plain functions, explicit parameters.
