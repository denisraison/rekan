# PEP-032: Agent tool simplification

**Status:** In Progress
**Date:** 2026-03-15
**Depends on:** PEP-030

## Context

The agent has 11 tools (5 read, 6 write) in `api/internal/agent/tools.go`. PEP-030 cleaned up the loop mechanics. This PEP looks at the tools themselves: what can we remove, merge, or simplify without losing capability.

**Redundant read tools.** `find_customer` and `list_customers` are two tools for one concept (look up customers) with different return shapes. Same with `find_post` and `list_posts`. Each additional tool is a decision boundary the model must evaluate on every turn. Two tools that overlap force the model to make a distinction that doesn't matter to the user.

**Posts are indistinguishable.** In production (2026-03-14), Nika asked for pending posts and got two Opalina posts distinguished only by opaque IDs (`n9cmjj2q131pfj8` vs `pef24izgeblsx69`). She had to say "o que começa com n9c" to reference one, then wait for a `find_post` round-trip to see its content before deciding to reject. With a caption snippet in the list, she could have gone straight to "rejeita o primeiro, muito longo" in one message. The full review flow took 5 turns instead of 2.

**Tools return prose, not data.** `reject_post` returns `"Elenice - Nika, post da Opalina rejeitado. Feedback: Texto muito longo."` The tool is formatting a user-facing sentence with the operator's name. The model already knows the name. This couples tools to presentation, makes them untestable without parsing Portuguese, and wastes tokens when the model rephrases anyway.

**Tool results contain prompt instructions.** `list_posts` appends `"O conteúdo completo dos posts será exibido automaticamente. Não inclua legendas, hashtags ou notas de produção na sua resposta."` to its output. Behavioral constraints belong in the system prompt once, not injected per tool call.

**Dead parameters cost tokens.** `approve_post` and `reject_post` accept `customer_name` but the implementation ignores it. Confirmed in production: the model fills it anyway (`"customer_name": "Opalina"`), burning output tokens on a no-op field.

**The `Posts` accumulator is a hidden side-effect.** `ToolExecutor.Posts` is a slice that read tools silently append to. After the loop, `formatPostDetails` injects post content into the reply. The model doesn't control this. If it calls `list_posts` to check something without wanting to display everything, tough luck. The "Não inclua legendas" instruction is a prompt band-aid for an architecture problem.

**`recent_activity` is noise.** The eval suite doesn't test it. Operators don't ask "what did the agent do recently?" because they're in the same WhatsApp group and can scroll up. The action log exists in the DB for debugging; it doesn't need an agent-facing tool.

**`pause_customer` is a one-field update.** It sets `invite_status = "cancelled"`. This could be a status change on `update_customer`. Having it separate also creates an asymmetry: there's no `unpause_customer`.

**Phone validation happens too late.** `NormalizePhone` runs inside `service.CreateBusiness` and `service.UpdateBusiness`. If the phone is invalid, the error surfaces after the model already committed to the action. Validate at the tool boundary so the model gets a clear error and can ask the user.

## Design

11 tools become 7. Every tool returns structured key:value data instead of Portuguese prose. The `Posts` accumulator is removed.

### Tool inventory after

| # | Tool | Type | Changes |
|---|------|------|---------|
| 1 | `search_customers` | read | Merges `find_customer` + `list_customers`. Optional `query` param. Returns customer IDs. |
| 2 | `search_posts` | read | Merges `find_post` + `list_posts`. List view includes 60-char caption snippet and 8-char short IDs. |
| 3 | `create_customer` | write | Phone validated before service call. Structured result. |
| 4 | `update_customer` | write | Absorbs `pause_customer` via `status` field ("paused"/"active"). Phone validated. |
| 5 | `generate_post` | write | Post content inline in result (no accumulator). |
| 6 | `approve_post` | write | `customer_name` removed. Accepts prefix IDs. |
| 7 | `reject_post` | write | `customer_name` removed. Accepts prefix IDs. |

### What `search_posts` returns

List view (multiple results):
```
id:n9cmjj2q customer:Opalina status:pendente preview:"Ontem uma cliente pegou um colar de pedra natural..."
id:pef24izg customer:Opalina status:pendente preview:"Hoje de manhã eu parei pra contar e não acreditei..."
```

Detail view (single `post_id` given):
```
id:n9cmjj2q131pfj8
customer:Opalina
status:pendente
caption:Ontem uma cliente pegou um colar de pedra natural...
hashtags:#semijoias #pedrasnaturais #prata925
production_note:Monte uma composição com 3 a 4 colares...
```

### Tool result format (all tools)

Before:
```
Elenice - Nika, post da Opalina rejeitado. Feedback: Texto muito longo, precisa ser mais curto..
```

After:
```
Post da Opalina rejeitado. Feedback: Texto muito longo, precisa ser mais curto.
```

No operator name. No behavioral instructions. Concise natural language that the model can relay directly. We initially tried a custom `key:value` format (`status:rejected\ncustomer:Opalina\n...`) but reverted after eval regressions: Anthropic's own guidance says models work best with "natural language names, terms, or identifiers" and our homebrew format was neither JSON nor prose, forcing the model to parse an unfamiliar format and convert it back to natural language for the user.

### Post ID prefix matching

`approve_post`, `reject_post`, and `search_posts` accept partial IDs (prefix match). The model sees `n9cmjj2q` from a list and passes that. The tool queries recent posts with `id LIKE 'prefix%'`, failing on 0 or 2+ matches.

### Removing the Posts accumulator

`executor.Posts` and `formatPostDetails` are deleted. Tools that touch posts (search, generate, approve, reject) include post content inline in their result. The system prompt gets one line: "O conteúdo dos posts aparece nos resultados das ferramentas. Não repita legendas, hashtags ou notas de produção na sua resposta."

## Waves

### Wave 1: Merge read tools, remove dead weight

Merge the 4 read tools into 2. Remove `recent_activity`. Clean up dead params.

**Files changed:**
- `api/internal/agent/tools.go` — `find_customer` + `list_customers` become `search_customers`. `find_post` + `list_posts` become `search_posts` with caption snippet in list view. `recent_activity` deleted. `customer_name` removed from `approve_post` and `reject_post` schemas.
- `api/internal/agent/agent.go` — `toolNameToActionType` updated for new tool names.
- `api/internal/agent/cases/core.yaml` — eval cases updated to assert `search_customers`/`search_posts` instead of old names.

**Gate:**
- [x] `grep -r "find_customer\|list_customers\|find_post\|list_posts\|recent_activity" api/internal/agent/tools.go` returns nothing
- [x] `make eval-agent` passes, no regressions

**Notes:**
- Updated `approve_by_customer_name` eval assertion from `post_abc123` to `post_abc` since list view now shows 8-char short IDs and the model passes those to `approve_post`.
- Also updated `toolNameToActionType` in agent.go, mock executor in eval.go, and test fixture JSON in agent_test.go and agent_wave4_test.go to use new tool names.

### Wave 2: Clean up results, remove Posts accumulator, absorb pause

Remove operator name and behavioral instructions from tool results. Delete `executor.Posts` and `formatPostDetails`. Absorb `pause_customer` into `update_customer`. Move phone validation into tool layer. Post content inline in generate/search results.

**Files changed:**
- `api/internal/agent/tools.go` — operator name removed from all tool results. `pause_customer` removed, `update_customer` gains `status` field. `executor.Posts` field and all `te.Posts = append(...)` lines removed. `formatPostDetails`, `appendPostFields`, `appendPostFieldsJSON` deleted. `resolveCustomer` simplified (no longer takes operator name). Phone normalization moved before service calls. Post content inline in `generatePost` result.
- `api/internal/agent/agent.go` — `executor.Posts` and `formatPostDetails` call site removed from `processWithTools`. `toolNameToActionType` drops `pause_customer`.
- `api/internal/agent/router.go` — `ActionCustomerPause` and `disambiguate` removed.
- `api/internal/agent/validate.go` — operator name removed from validation errors.
- `api/internal/agent/prompt.go` — post content instruction updated. Added pause tool hint.
- `api/internal/agent/eval.go` — mock executor updated: `pauseCustomer` removed, `updateCustomer` handles status field, operator name removed from mock results.
- `api/internal/agent/agent_wave4_test.go` — pause test uses `update_customer` with `status:paused`.
- `api/internal/agent/cases/core.yaml` — added `pause_via_update` eval case.

**Gate:**
- [x] `grep -r "executor\.Posts\|formatPostDetails\|pause_customer" api/internal/agent/ --include='*.go'` returns nothing
- [x] `make eval-agent` passes (no regressions vs Wave 1 baseline)
- [ ] Manual: send "pausa a [cliente]" via WhatsApp, verify it calls `update_customer` with `status:paused`

**Notes:**
- Initially implemented `key:value` structured results per the original design, but eval regressions showed the model struggled with the custom format. Reverted to concise natural language (without operator name or behavioral instructions). See Anthropic's "Writing effective tools for agents" guide: models work better with natural language identifiers than custom formats.
- `decodeHashtags` kept (still used by `sendPostToClient` and `searchPosts` detail view).
- `bizNameMap` kept (still used by `searchPosts` list view).

### Wave 3: ID resolution and prefix matching

Add customer ID passthrough on write tools. Add post ID prefix matching.

**Files changed:**
- `api/internal/agent/tools.go` — `generate_post` and `update_customer` gain optional `customer_id` param that skips fuzzy matching. `approve_post`, `reject_post`, `search_posts` accept partial post IDs via `id LIKE 'prefix%'` query.
- `api/internal/service/content.go` — `ListPosts` gains prefix-match filter option.
- `api/internal/agent/cases/core.yaml` — add eval case: approve by prefix ID. Add eval case: generate_post by customer ID.

**Gate:**
- [ ] `make eval-agent` passes
- [ ] Manual: send "aprova o n9cm" via WhatsApp after listing posts, verify it resolves and approves

## Consequences

- 11 → 7 tools. 36% fewer tool definitions for the model to parse each turn. Fewer tokens in the system message. Clearer selection boundaries.
- Post review flows that took 5 turns should take 2: list (with previews) → act. The operator never needs to type or read opaque IDs.
- Tool results are concise and consistent (no operator name, no behavioral instructions) while staying in natural language the model handles well.
- The `Posts` side-channel and its prompt band-aid are gone. The model controls what appears in its reply.
- Trade-off: merging tools changes selection behavior. The eval suite covers main flows but not every edge. Run eval after each wave and spot-check with real WhatsApp messages before deploying.
- Trade-off: 8-char prefix IDs have enough entropy for current scale. If we reach thousands of posts, prefix collisions become possible. The prefix match fails clearly on ambiguity.
