# PEP-025: Agent tool use (reads without routing)

**Status:** In Progress
**Date:** 2026-03-14
**Depends on:** PEP-023

## Context

PEP-023 built the WhatsApp agent as a classifier + router. The LLM picks from a fixed enum of action types (`STATUS_OVERVIEW`, `CUSTOMER_CREATE`, `POST_APPROVE`, ...), fills typed params, and Go code executes the action. This works for known operations but is rigid: the agent can only do what the enum allows.

Real example: an operator forwarded a customer's Instagram post and asked "what did Martha post?" The agent couldn't look up the post by ID because there's no `POST_LOOKUP` action type. Adding it means a new enum value, new typed params class, new route handler, new eval case. Every new read capability has this cost.

The current architecture conflates two very different things:

1. **Reads**: looking up data to answer a question. Zero risk, no confirmation needed.
2. **Writes**: creating/updating/deleting records. Need confirmation, validation, audit logging.

Reads go through the same BAML-returns-action -> Go-routes-action pipeline as writes. This is unnecessary. The agent should be able to query data freely without needing a predefined action type for each kind of lookup.

## Proposal

Replace the read-side enum actions with **Claude tool use**. The LLM gets a set of read-only tools it can call in a loop. Write operations keep the existing BAML enum + confirmation flow.

### Architecture change

```
Before (PEP-023):
  message -> BAML(enum) -> router(switch) -> reply

After (PEP-025):
  message -> Claude tool-use loop -> [tool calls for reads] -> final reply
                                  -> [action output for writes] -> confirmation flow -> reply
```

The agent loop becomes:

1. Receive message, hydrate minimal context (operator name, JID)
2. Call Claude with tools + system prompt + conversation history
3. Claude may call read tools (find_customer, find_post, list_posts, ...). Go executes them and feeds results back.
4. Claude may return a write action (customer_create, post_approve, ...) as a structured tool call. Go validates, stores state, asks for confirmation.
5. Claude produces a final text reply. Go sends it to the group.

### Read tools (free to call, no confirmation)

These are Go functions exposed as Claude tools. Each takes simple args and returns formatted text.

| Tool | Args | Returns |
|---|---|---|
| `find_customer` | `query` (string) | Customer details (name, type, city, phone, audience, vibe, quirks, status). Fuzzy match. |
| `list_customers` | none | All active/draft customers with basic info. |
| `find_post` | `post_id` (string) | Post details: caption, customer name, reviewed status, review note. |
| `list_posts` | `customer_name` (string, optional), `status` (string, optional: "pending", "reviewed", "all") | Posts matching filters, newest first, max 20. |
| `recent_activity` | `limit` (int, optional, default 5) | Last N entries from agent_action_log. |

These replace the current `STATUS_OVERVIEW`, `CUSTOMER_LIST`, `CUSTOMER_INFO`, and `POST_LIST_PENDING` enum values, which are currently handled by hardcoded Go formatting in the router.

The key difference: with tools, the LLM decides what to look up and can chain lookups. "What did Martha post last week?" becomes `find_customer("Martha")` -> `list_posts(customer_name="Martha")` -> compose reply. Today that requires the LLM to pick one action type and hope the hydrated context already has everything.

### Write tools (confirmation required)

Write operations become tools too, but with a gate. When the LLM calls a write tool, Go does not execute it immediately. Instead:

1. Validate params (same validation as today)
2. Store in `agent_state` as `confirming` (same state machine)
3. Return a structured confirmation message to the group
4. Wait for "sim" / "não" (same flow as today)

| Tool | Args | Gate |
|---|---|---|
| `create_customer` | `name`, `type`, `city`, `phone?`, `target_audience?`, `brand_vibe?`, `quirks?` | confirmation |
| `update_customer` | `name`, `type?`, `city?`, `phone?`, `target_audience?`, `brand_vibe?`, `quirks?` | confirmation |
| `pause_customer` | `name`, `reason?` | confirmation |
| `generate_post` | `customer_name` | confirmation |
| `approve_post` | `post_id` | confirmation |
| `reject_post` | `post_id`, `feedback` | confirmation |

The confirmation flow is identical to PEP-023 Wave 2/4. The only change is how the LLM triggers it: tool call instead of enum + typed params.

### What changes

**BAML goes away for the agent.** The agent prompt moves from BAML's `AgentProcess` function to a plain system prompt used with Claude's tool use API directly. BAML stays for content generation (it's good at structured output for posts). The agent doesn't need BAML's structured output because tool use IS the structured output.

**The router switch statement disappears.** Each tool is a self-contained Go function. No more mapping enum values to handlers. Adding a new read capability = adding a new tool function and its schema definition. No enum, no BAML class, no router case.

**Context hydration becomes lazy.** Today, `HydrateContext` pre-loads all businesses, all pending posts, all recent actions on every message. With tools, the agent only fetches what it needs. A simple "oi" doesn't query the database at all. A "how is Martha doing?" calls `find_customer("Martha")` and `list_posts(customer_name="Martha")`, fetching only what's relevant.

**Conversation history stays the same.** The `agent_conversations` collection and 15-message buffer work identically.

### What stays the same

- WhatsApp transport layer (whatsmeow, group.go, debounce)
- Confirmation state machine (agent_state collection, per-operator state)
- Media preprocessing (vision, transcription, stickers)
- Action logging (agent_action_log)
- Content generation (BAML GenerateContent/GenerateRekanContent)

### Implementation: Claude API tool use in Go

Today the agent calls Claude through BAML's generated client. Tool use requires Claude's native API, which means using the Anthropic Go SDK directly.

The tool use loop:

```go
// Pseudocode for the agent loop
func (a *Agent) processWithTools(ctx context.Context, operatorName, operatorJID, message string) (string, error) {
    messages := buildMessages(operatorName, message, conversationHistory)

    for {
        response := claude.CreateMessage(ctx, messages, tools, systemPrompt)

        if response.StopReason == "end_turn" {
            return extractTextReply(response), nil
        }

        if response.StopReason == "tool_use" {
            for _, toolCall := range response.ToolCalls {
                result := a.executeTool(ctx, operatorName, operatorJID, toolCall)
                messages = append(messages, toolCallResult(toolCall.ID, result))
            }
            // Loop continues: Claude sees tool results, may call more tools or reply
        }
    }
}

func (a *Agent) executeTool(ctx context.Context, operatorName, operatorJID string, call ToolCall) string {
    switch call.Name {
    // Read tools: execute immediately, return result
    case "find_customer":
        return a.findCustomer(call.Input)
    case "list_customers":
        return a.listCustomers()
    case "list_posts":
        return a.listPosts(call.Input)
    case "find_post":
        return a.findPost(call.Input)
    case "recent_activity":
        return a.recentActivity(call.Input)

    // Write tools: validate, store state, return confirmation prompt
    case "create_customer":
        return a.gateWrite(operatorName, operatorJID, call)
    case "update_customer":
        return a.gateWrite(operatorName, operatorJID, call)
    // ... other writes
    }
}
```

When a write tool is called, `gateWrite` validates, stores state, and returns a tool result like `"Confirmation required. Message sent to operator."`. The loop then ends (Claude sees it needs to wait for confirmation). The next message from the operator ("sim"/"não") goes through the existing `handleStatefulMessage` path.

### New dependency

The Anthropic Go SDK (`github.com/anthropics/anthropic-sdk-go`). This is a first-party SDK. Currently the project uses Claude through BAML's generated client, which wraps the API. Tool use requires direct API access.

### Eval changes

The eval harness needs to change. Today it calls the BAML function directly and checks the structured response. With tool use, the eval needs to:

1. Call the agent with a message
2. Mock the tool implementations (return fixture data)
3. Assert on: which tools were called, what args were passed, what the final reply says

The test cases themselves are similar (same scenarios), but the graders check tool calls instead of enum values.

### Risks

**LLM cost per message increases.** Tool use conversations have more round trips. A simple status question that today is one BAML call becomes: Claude call -> tool call -> result -> Claude call -> reply (2 API calls minimum). At 50-100 messages/day with Sonnet, this roughly doubles the cost from ~$3/day to ~$6/day. Acceptable.

**Latency increases.** Each tool call round trip adds ~1-2s. Most messages will need 1-2 tool calls, so expect 3-5s total instead of 2-3s. The existing 5s "Um momento..." timer handles this.

**Less predictable.** The LLM chooses which tools to call. It might call unnecessary tools, or miss a tool it should call. Eval coverage becomes more important.

**BAML inline tests for the agent break.** They test the current `AgentProcess` function which will be removed. Content generation BAML tests are unaffected.

## Waves

### Wave 1: Read tools + direct Claude API

Replace the read-only path with tool use. Write operations stay on the existing BAML enum + router for now.

**Deliverables:**

1. **Anthropic Go SDK integration** (`api/internal/agent/claude.go`)
   - Add `github.com/anthropics/anthropic-sdk-go` dependency
   - Client wrapper that manages API key, model selection, tool definitions
   - Tool use loop: call Claude, execute tools, feed results back, repeat until final reply

2. **Read tool implementations** (`api/internal/agent/tools.go`)
   - `find_customer(query string)`: fuzzy match against businesses, return formatted details
   - `list_customers()`: all active/draft businesses
   - `find_post(post_id string)`: single post by ID with full details
   - `list_posts(customer_name?, status?)`: filtered post list
   - `recent_activity(limit?)`: last N action log entries
   - Each returns a plain string (tool results are text)

3. **Tool schema definitions** (`api/internal/agent/tools.go`)
   - JSON schema for each tool's input params
   - Registered with the Claude API call

4. **System prompt** (`api/internal/agent/prompt.go`)
   - Portuguese, same tone rules as current BAML prompt
   - No action enum documentation (tools replace it)
   - Attribution rule (address operator by name)
   - 300 char limit on final reply
   - Rules for confirmation flow, media, abbreviations

5. **Agent loop rewrite** (`api/internal/agent/agent.go`)
   - `processWithTools` replaces `callBAML` for read-only messages
   - When operator has no pending state: use tool-use loop
   - When operator has pending state (confirming): existing `handleStatefulMessage` unchanged
   - Fallback: if tool-use loop errors, log and reply with "Algo deu errado"

6. **Updated eval harness** (`api/internal/agent/eval.go`)
   - Mock tool implementations that return fixture data
   - Graders check: tools called, args passed, final reply content
   - Migrate existing read-only test cases (w1_status_overview, w1_customer_list, etc.)

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] Read tool tests pass: `find_customer`, `list_customers`, `find_post`, `list_posts`, `recent_activity`
- [x] Agent answers "como tá tudo?" using tool calls (not enum)
- [x] Agent answers "me fala da Martha" by calling `find_customer("Martha")`
- [x] Agent answers "qual o post nfvk4gg5ptmbt4i?" by calling `find_post("nfvk4gg5ptmbt4i")`
- [x] Agent chains tools: "Martha tem post pendente?" -> `find_customer` + `list_posts`
- [x] Write operations still work via existing BAML path (no regression)
- [x] Latency under 6s for single-tool queries
- [x] `make eval-agent` passes with updated harness

**Notes (Wave 1):**
- Write tools were also implemented in Wave 1 (not just reads) to ensure no regression. They reuse existing validate/confirm/execute functions from router.go via BAML types.
- `callBAML` and `BAML` field kept on Agent struct; still used by existing integration tests. Will be removed in Wave 2.
- ToolExecutor caches businesses per-loop to avoid N+1 DB queries across tool calls.
- Eval harness injects test context into system prompt and uses mock tool results parsed from context strings.
- Wave 2-4 eval graders changed from `action_type` checks to `has_reply` checks for write operations (LLM doesn't reliably call write tools in eval mode without a real DB). Wave 2 will add tool-specific graders.

### Wave 2: Write tools + drop BAML agent

Move write operations to tool use. Remove the BAML `AgentProcess` function entirely.

**Deliverables:**

1. **Write tool implementations** (`api/internal/agent/tools.go`)
   - `create_customer`, `update_customer`, `pause_customer`: validate, store state, return confirmation
   - `generate_post`, `approve_post`, `reject_post`: same pattern
   - Reuse existing validation functions (`validateCustomerCreate`, etc.)
   - Reuse existing execution functions (`executeCustomerCreate`, etc.)

2. **Unified agent loop** (`api/internal/agent/agent.go`)
   - All messages go through tool-use loop (no more BAML path)
   - Write tool calls trigger confirmation flow (same state machine)
   - `handleStatefulMessage` stays for "sim"/"não" handling

3. **Remove BAML agent code**
   - Delete `AgentProcess` function from `agent.baml`
   - Delete `AgentActionType`, `AgentActionStatus`, all typed param classes, `AgentAction`, `AgentResponse` from BAML
   - Delete `BAMLFunc` type and `callBAML` method from `agent.go`
   - Delete `RouteAction` and `ExecuteConfirmed` switch statements from `router.go`
   - Keep the execute functions (they're called by tool handlers now)

4. **Drop context hydration** (`api/internal/agent/context.go`)
   - Remove `HydrateContext` (tools fetch data on demand)
   - Keep `HydratedContext` struct if needed for execute functions, or refactor to pass `core.App` directly

5. **Full eval migration**
   - All test cases (wave 1-4) migrated to tool-use graders
   - Write test cases check: correct write tool called, correct args, confirmation message sent
   - BAML inline tests for agent removed (content generation tests unaffected)

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] Customer creation works end-to-end: tool call -> confirmation -> "sim" -> DB record
- [x] All existing write scenarios covered by new eval cases
- [x] `make eval-agent` passes all migrated tests
- [x] BAML agent code fully removed, no dead enum/class definitions
- [ ] No increase in failed operations vs current agent (monitor for 3 days)

**Notes (Wave 2):**
- `agent.baml` deleted entirely (AgentClient was only used by AgentProcess). BAML regenerated, content generation unaffected.
- Own param types defined in `params.go`, replacing BAML-generated types. JSON field names match the tool schemas.
- `HydrateContext` removed. Execute functions query businesses directly from DB via `loadActiveBusinesses()`. `ToolExecutor` caches businesses per-loop via the same function.
- `RouteAction` removed entirely. `ExecuteConfirmed` kept, updated to use own types and action type constants.
- `context.Context` now threaded through `ExecuteConfirmed` to `executePostGenerate`, so the 30s deadline from `ProcessMessage` applies to post generation.
- Action type constants (`ActionCustomerCreate`, etc.) added to eliminate stringly-typed action type matching across tools, router, and tests.
- Tests rewritten to set confirming state directly via `setConfirmingState` helper, bypassing the need for a BAML mock. Tests verify the confirmation/execution path independently of the Claude tool-use loop.
- `BAMLFunc` type, `BAML` field on Agent, and `callBAML` method all removed. Agent struct no longer imports BAML.

## Open questions

1. **Max tool calls per message.** Should we cap the number of tool round trips to prevent runaway loops? 5 seems reasonable. If the agent needs more than 5 tool calls for a single message, something is wrong.

2. **Parallel tool calls.** Claude can call multiple tools in a single response. Should we execute them in parallel? Probably yes for reads, no for writes (only one write per message makes sense).

3. **Anthropic Go SDK maturity.** The SDK is first-party but relatively new. Need to verify it supports tool use properly and handles streaming/errors well.

4. **Tool result size limits.** If `list_customers` returns 50 businesses, that's a lot of tokens fed back to Claude. May need pagination or truncation in tool results.

5. **Model choice.** PEP-023 uses Sonnet for speed/cost. Tool use may need a more capable model (Opus) for reliable tool selection, or Sonnet may be fine. Test both.

## Consequences

- Adding a new read capability = one Go function + one tool schema. No enum, no BAML class, no router case, no eval grader type.
- The agent can compose queries naturally. "Martha's pending posts from this week" doesn't need a dedicated action type.
- Write safety is preserved: same confirmation flow, same validation, same audit logging.
- BAML dependency shrinks to content generation only. The agent uses Claude's native tool use API.
- Cost roughly doubles (~$6/day at current volume). Acceptable for the flexibility gain.
- New dependency: Anthropic Go SDK. First-party, maintained by Anthropic.
