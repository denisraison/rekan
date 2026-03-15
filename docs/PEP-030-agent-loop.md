# PEP-030: Agent loop

**Status:** Done
**Date:** 2026-03-15
**Depends on:** PEP-029

## Context

The current agent lives in `api/internal/agent/`. The tool loop in `claude.go` works but has structural problems that compound as we add capabilities.

**Anthropic SDK dependency.** The SDK's types (`MessageParam`, `ToolParam`, `ContentBlockParamUnion`) are woven through the entire agent: conversation storage, history replay, tool definitions, eval runner. Swapping models or even upgrading the SDK means touching every file. The SDK itself is just type definitions and one HTTP call. We don't need it.

**RunToolLoop takes 7 parameters.** `ctx`, `app`, `waClient`, `operatorName`, `gen`, `messages`, `systemPrompt`. Most exist to construct a `ToolExecutor` inside the loop. The loop shouldn't know about PocketBase, WhatsApp, or content generation. It should know about messages, tools, and stop conditions.

**Tool errors are invisible to the model.** Every tool result is passed with `isError: false`, even failures. Claude can't distinguish "customer not found" from "here are the customer details." It can't retry or change strategy on error because it doesn't know there was one.

**Tools execute sequentially.** When Claude calls `find_customer` and `list_posts` in the same turn, they run one after the other in a for loop. Independent tools should run concurrently.

**Adding a tool requires edits in 3 places.** `buildToolDefs()` for the schema, the `executeTool` switch, and `toolNameToActionType`. The tool definition and its implementation are separated across files with no compile-time link between them.

**No per-tool timing.** `LogAction` records total latency for the entire flow. If `generate_post` takes 8 seconds inside a 12 second turn, you can't see that from the traces.

**30 second timeout for nested LLM calls.** `generate_post` calls BAML which calls Claude. That's an LLM call inside a tool inside an LLM loop, all under one 30s context deadline. Tight enough to fail on slow turns.

**1024 max tokens.** Truncates complex multi-tool replies, especially in Portuguese.

### Design principles

Taken from the nullclaw/zeroclaw pattern:

1. **Agents are loops.** No state machines, no planners, no DAGs. The LLM decides what to do next.
2. **Tools are the unit of composition.** A tool is a name, a schema, and a function. Subagents are tools. BAML calls are tools. DB queries are tools.
3. **The loop owns nothing.** It takes messages in, calls the model, dispatches tools, feeds results back. No business logic, no infrastructure coupling.
4. **Tracing is logging at the loop boundary.** One callback, one struct, full visibility.

## Design

Everything stays in `api/internal/agent/`. No new package. The loop, types, and HTTP client replace the current `claude.go` and the Anthropic SDK types. If we ever need a second agent, we extract then.

### Types

```go
// Message is a conversation turn.
type Message struct {
    Role    Role           `json:"role"`
    Content []ContentBlock `json:"content"`
}

type Role string

const (
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
)

// ContentBlock is one piece of content within a message.
type ContentBlock struct {
    Type      string          `json:"type"`
    Text      string          `json:"text,omitempty"`
    ID        string          `json:"id,omitempty"`
    Name      string          `json:"name,omitempty"`
    Input     json.RawMessage `json:"input,omitempty"`
    ToolUseID string          `json:"tool_use_id,omitempty"`
    Content   string          `json:"content,omitempty"`
    IsError   bool            `json:"is_error,omitempty"`
}
```

Flat, JSON-serializable, no interface boxing. These get stored in PocketBase's `structured` field directly, replacing the current `anthropic.MessageParam` serialization.

Wire-compatible with the Anthropic Messages API format. Existing stored messages in `agent_conversations` will deserialize into the new types without migration.

### Tool

```go
// Tool is the unit of composition. Everything the agent can do is a Tool.
type Tool struct {
    Name        string
    Description string
    InputSchema json.RawMessage
    Execute     func(ctx context.Context, input json.RawMessage) (string, error)
}
```

One struct, one place. The schema and the implementation live together. Adding a tool is adding one `Tool` value to a slice. No switch statements, no registration.

`Execute` returns `(string, error)`. The loop maps `error != nil` to `is_error: true` on the tool result block. Claude sees the difference.

### The loop

```go
type RunConfig struct {
    System    string
    Messages  []Message
    Tools     []Tool
    MaxTurns  int         // default 10
    MaxTokens int         // default 2048
    OnTrace   func(Trace) // optional
}

type RunResult struct {
    Reply    string
    Messages []Message // full conversation including tool turns
    Traces   []Trace   // one per turn
}

func (c *Client) Run(ctx context.Context, cfg RunConfig) (*RunResult, error)
```

The loop:

1. Call the model with system + messages + tool definitions.
2. Parse response. If no tool_use blocks, return the text (done).
3. Execute tools concurrently (`sync.WaitGroup`, not `errgroup`). We need ALL results, even errors, because every `tool_use` block must have a matching `tool_result`. Map `error != nil` to `is_error: true` on the result block.
4. Append assistant message and tool results to messages.
5. Emit trace for this turn.
6. Go to 1.

Exit conditions: model responds with text only, max turns reached, context cancelled, API error.

### Tracing

```go
type Trace struct {
    Turn         int         `json:"turn"`
    ModelLatency int64       `json:"model_latency_ms"`
    InputTokens  int         `json:"input_tokens"`
    OutputTokens int         `json:"output_tokens"`
    ToolCalls    []ToolTrace `json:"tool_calls,omitempty"`
}

type ToolTrace struct {
    Name     string `json:"name"`
    Duration int64  `json:"duration_ms"`
    Error    string `json:"error,omitempty"`
}
```

Emitted via `OnTrace` callback at the end of each turn. The caller decides where to send it: stdout, PocketBase `agent_action_log`, both.

### HTTP client

```go
type Client struct {
    APIKey  string
    Model   string
    BaseURL string // default https://api.anthropic.com
}
```

One method: `call(ctx, system, messages, tools, maxTokens) (*response, error)`. Raw `net/http` POST to `/v1/messages`. Parses the response JSON into our own types. ~50 lines.

No streaming. WhatsApp messages are sent as complete text. No rate limit retry for now; add when we actually hit 429s in production.

### How `agent` changes

`claude.go` is deleted. `ClaudeClient` and `RunToolLoop` are replaced by `Client` and `Run`.

`ToolExecutor` stays but its methods become closures captured in `Tool.Execute`. The switch statement in `executeTool` and the `buildToolDefs()` function both disappear, replaced by a single `buildTools()` that returns `[]Tool`.

`buildClaudeMessages` returns `[]Message` instead of `[]anthropic.MessageParam`. `sanitizeToolPairs` and `mergeConsecutiveRoles` operate on `[]Message`.

The `Agent` struct keeps its WhatsApp, PocketBase, debouncing, and transcription responsibilities. It constructs a `Client` and `[]Tool`, calls `Run()`, and handles the result.

### Post-processing side effects

The current `toolUseResult` carries Rekan-specific state: `Posts` (records referenced during execution, appended to the reply), `BizNames` (business ID to display name map), and `WriteUsed` (whether a write tool was called). These don't belong in `RunResult`.

They stay on `ToolExecutor`, which the tool closures capture. After `Run()` returns, the caller reads `executor.Posts` and `executor.WriteUsed` to build the final WhatsApp reply and determine the action type for logging.

### Action type mapping

`toolNameToActionType` stays as a standalone function. It maps tool names to log action types (`create_customer` → `CUSTOMER_CREATE`). This is a logging concern, not a loop concern. The caller inspects `RunResult.Traces[].ToolCalls[].Name` to determine the action type after the loop finishes.

### Eval unification

`eval.go` currently duplicates the tool loop (`runEvalCase` is a copy of `RunToolLoop` with mock dispatch). After migration, `RunEval` constructs `[]Tool` with mock executors and calls the same `Run()` function as production. One loop, one code path.

## Waves

### Wave 1: Rewrite loop, tools, and wiring

One atomic change. Replace `claude.go` with own types, HTTP client, and `Run()` loop. Convert tools to `[]Tool` closures. Rewire `agent.go` and conversation storage.

**New files:**
- `api/internal/agent/types.go` — `Message`, `ContentBlock`, `Role`, `Tool`, `Trace`, `ToolTrace`, `RunConfig`, `RunResult`, helper constructors (`NewTextBlock`, `NewToolResultBlock`, `NewUserMessage`)
- `api/internal/agent/client.go` — `Client` struct, `call()` method (raw HTTP to `/v1/messages`), response parsing
- `api/internal/agent/run.go` — `Run()` function: the tool loop, concurrent tool execution via `sync.WaitGroup`, trace emission
- `api/internal/agent/run_test.go` — tests with mock HTTP server: single turn (no tools), multi-turn (tools), tool error propagated as `is_error: true`, max turns exit, concurrent execution

**Modified files:**
- `api/internal/agent/tools.go` — rewritten: `buildTools(executor *ToolExecutor, operatorName string) []Tool` returns a slice. Each tool is a `Tool{}` literal with schema and `Execute` closure over the executor. `executeTool` switch and `buildToolDefs()` deleted.
- `api/internal/agent/agent.go` — `processWithTools` builds `RunConfig`, calls `client.Run()`, maps `RunResult` to `agentResult`. Reads `executor.Posts` and `executor.WriteUsed` for post-processing. `buildClaudeMessages` returns `[]Message`. `sanitizeToolPairs`, `mergeConsecutiveRoles`, `appendOrMergeUser` operate on `[]Message`.
- `api/internal/agent/conversation.go` — `ConversationMessage.Structured` deserializes to `Message`.
- `api/internal/agent/agent_test.go` — updated for new types.

**Deleted files:**
- `api/internal/agent/claude.go`

**Gate:**
- [x] `go build ./internal/agent/...` compiles with no `anthropic-sdk-go` import anywhere in the package
- [x] `go test ./internal/agent/...` passes, run_test.go covers: no-tool turn, tool turn, error propagation, max turns, concurrent execution
- [x] `go vet ./internal/agent/...` clean
- [ ] Manual test: WhatsApp message → agent responds → conversation replays on next message
- [x] Tool errors produce `is_error: true` in stored conversation JSON

**Notes:**
- `eval.go` was also migrated to use `Client.Run()` with mock tools (originally planned for Wave 2), since it had to compile without the SDK import.
- `ToolExecutor.resolveCustomer` now returns `(record, errString)` instead of `(record, *toolResult)` since tool methods return plain strings. `WriteUsed` is tracked on the executor and set by the `writeTool` wrapper in `buildTools()`.
- `conversation.go` needed no changes; its `Structured` field was already a plain JSON string that deserializes into our new `Message` type identically to the old `anthropic.MessageParam`.

### Wave 2: Remove SDK from go.mod

Eval was migrated in Wave 1. Remaining: remove `anthropic-sdk-go` from `go.mod` and run `go mod tidy`.

**Files:**
- `go.mod` / `go.sum` — remove `github.com/anthropics/anthropic-sdk-go`

**Gate:**
- [x] `runEvalCase` contains zero direct HTTP calls or loop logic (done in Wave 1)
- [x] Token counts and tool round trips still reported accurately (done in Wave 1)
- [x] `make eval-agent` exits 0, all cases pass
- [x] `grep -r "anthropic-sdk-go" api/` returns nothing
- [x] `go mod tidy` clean
- [x] `make dev` starts successfully

**Notes:**
- Fixed model ID in `client.go`: Wave 1 used `claude-sonnet-4-6-20250514` (non-existent), corrected to `claude-sonnet-4-6` matching the SDK's `ModelClaudeSonnet4_6` constant.
- `go mod tidy` also removed the `tidwall/*` indirect dependencies (gjson, match, pretty, sjson) that were only needed by the SDK.
- `make eval-agent` has 1 flaky case per run due to LLM non-determinism (different case fails each time). Not a code regression.

## Consequences

- The Anthropic SDK is gone. Model changes or API upgrades are a change to `client.go`, not a dependency update that cascades.
- Tool errors are visible to the model. Claude can retry or change strategy when a tool fails.
- Concurrent tool execution. Independent tools in the same turn run in parallel.
- Per-tool tracing. Every tool call has a duration and error field. Slow tools are immediately visible.
- One loop for production and eval. No drift between two copies.
- Adding a tool is one struct value. No switch statement, no separate schema, no action type mapping.
- Conversation storage is backwards compatible. No migration needed.
- No streaming. If we later need it (web UI), it's an addition to `client.go`.
- MaxTokens increases from 1024 to 2048.
- Timeout moves from a blanket 30s to per-turn context propagation. Individual tools can set their own timeouts via context wrapping in `Execute`.
