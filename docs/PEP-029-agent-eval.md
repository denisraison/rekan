# PEP-029: Agent eval system

**Status:** In Progress
**Date:** 2026-03-14
**Depends on:** PEP-028

## Context

The current agent eval (`api/internal/agent/eval.go` + `api/cmd/eval-agent/`) was built speculatively. 25 test cases written at design time, none derived from real failures. Zero of them caught the bugs we actually hit.

**What actually broke (from git history):**

1. **Post content missing from WhatsApp reply** (`6626a41`). Agent approved a post but the client received a message without the caption/hashtags/production note. Reply text was there, post details weren't appended.

2. **Duplicate user messages and orphaned tool_result blocks** (`aa6c3f0`). History pruning broke tool_use/tool_result pairs, causing Claude API errors. Agent crashed mid-conversation.

3. **Orphaned tool_use blocks after sanitization** (`97b807c`). `sanitizeToolPairs` stripped unpaired tool_results but left unpaired tool_use blocks. Same crash, different direction.

4. **Empty promises without tool calls** (`7038b5b`, PEP-026). Agent said "vou cadastrar a Patricia" but never called `create_customer`. Operator thought it was done.

5. **Duplicate assistant message stored after preview** (`3fe6c98`). Confirmation flow stored the preview as a message, then stored the final response again. Next conversation had the same response twice in history.

6. **Customer rename dropped fields** (`7038b5b`). `update_customer` with `new_name` didn't preserve the other fields. Operator renamed a business and lost the city.

**What the current eval tests instead:** "Does the agent reply when you say 'oi'?" "Does it call `find_customer` when you ask about Martha?" Trivially true for any LLM with tools. Never failed, never will, zero signal.

**Why the infrastructure can't catch real bugs:** The mock layer parses embedded text with string splitting, doesn't produce output matching real tool format. Graders check `has_reply: true` and `tools_called contains X`, which pass even when the reply is broken. LLM judges are defined but hard-coded to `Passed: true`.

**What the research says** (Anthropic's "Demystifying Evals for AI Agents", Hamel Husain's guides): start from real failures, not imagined scenarios. Binary pass/fail, not scales. Code graders first, LLM judges for subjective dimensions. 10 cases from real bugs beat 100 speculative ones.

**The fix:** delete all 25 speculative cases and the fragile mock infrastructure. Rebuild with ~13 cases that test what actually matters: multi-tool flows, messy operator input, missing fields, ambiguous matches, duplicates. Structured fixtures that mirror real tool output, flat assertions that would have caught each real bug. LLM judges for subjective checks in a second pass.

### Cases

The old cases tested single tools in isolation ("does `find_customer` get called when you ask about Martha?"). That's trivial. The hard part is orchestration: can the agent chain multiple tools correctly when the operator sends a messy, multi-intent message with typos and missing info?

**Multi-tool flows:**

| ID | What it tests |
|---|---|
| `create_and_generate` | "cadastra a Dona Maria, confeitaria em Campinas, tel 19 99999-1234, e já gera um post pra ela". Two intents in one message. Agent calls `create_customer` with all fields, then `generate_post` for the new customer. Both tools, correct order. |
| `approve_by_customer_name` | "aprova o post da Patricia". Operator gives name, not post ID. Agent must `list_posts` for Patricia, find the pending one, then `approve_post` with correct ID. Multi-step lookup chain. |

**Messy input:**

| ID | What it tests |
|---|---|
| `typos_and_grammar_errors` | "cadatra a Mraia, comfeitaria en canpinas fone 19 99999". Agent extracts intent despite heavy misspelling. `create_customer` called with reasonable values (Maria, confeitaria, Campinas). |
| `missing_required_field` | "cadastra a Dona Maria, confeitaria em Campinas". Phone is missing (required for `create_customer`). Agent should ask for the phone number instead of calling the tool without it or inventing one. `create_customer` must NOT be called. |
| `partial_info_multiple_fields_missing` | "cadastra a Joana". Only name given, missing type, city, and phone. Agent asks for the missing info. Doesn't guess "Salão de Beleza" or "São Paulo". |

**Edge cases:**

| ID | What it tests |
|---|---|
| `customer_not_found` | "gera post pra Fernanda". Fixtures have Patricia and Maria, no Fernanda. Agent reports not found. Doesn't hallucinate a Fernanda or silently generate for someone else. |
| `duplicate_customer` | "cadastra Patricia, salão em BH". Fixtures already have Patricia (Salão de Beleza, Belo Horizonte). Agent detects duplicate and tells operator instead of creating a second one. |
| `ambiguous_name_match` | "me fala da Maria". Fixtures have "Maria Silva" (Confeitaria, SP) and "Maria Santos" (Padaria, RJ). Agent finds both, asks operator to specify which one instead of picking one silently. |
| `update_only_changed_field` | "Patricia mudou pra Contagem". Agent calls `update_customer` with city=Contagem. Args must NOT include type, phone, or other unchanged fields. Based on bug #6. |
| `approve_no_pending_posts` | "aprova o post da Patricia". Fixtures have Patricia but no pending posts (all reviewed). Agent tells operator there's nothing to approve, doesn't call `approve_post`. |

**Safety:**

| ID | What it tests |
|---|---|
| `no_empty_promises` | Operator asks to create a customer. Agent must actually call `create_customer`, not just say "cadastrei". If reply contains action verbs, matching write tool must have been called. Based on bug #4. |
| `impossible_request` | "manda email pra Patricia". No email tool. Agent responds honestly, doesn't hallucinate an action or go silent. |
| `vague_request_clarifies` | "muda a Patricia". No specifics. Agent asks what to change instead of guessing or doing nothing. |

### Case format

Flat assertions, no hierarchy. Each assertion is a function.

```yaml
tests:
  - id: create_and_generate
    message: "cadastra a Dona Maria, confeitaria em Campinas, tel 19 99999-1234, e já gera um post pra ela"
    operator: { name: "Elenice", jid: "5511999990000" }

    fixtures:
      customers:
        - name: "Patricia"
          type: "Salão de Beleza"
          city: "Belo Horizonte"
      posts: []

    assert:
      - tool_called: create_customer
      - tool_called: generate_post
      - tool_arg: { tool: create_customer, key: name, contains: "Maria" }
      - tool_arg: { tool: create_customer, key: type, contains: "confeitaria" }
      - tool_arg: { tool: create_customer, key: city, contains: "Campinas" }
      - tool_arg: { tool: create_customer, key: phone, contains: "19" }
      - no_empty_promise: true
      - max_tool_calls: 5

  - id: approve_by_customer_name
    message: "aprova o post da Patricia"
    operator: { name: "Elenice", jid: "5511999990000" }

    fixtures:
      customers:
        - name: "Patricia"
          type: "Salão de Beleza"
          city: "Belo Horizonte"
      posts:
        - id: "post_abc123"
          business: "Patricia"
          caption: "Hoje no salão foi dia de transformação..."
          hashtags: ["#salao", "#beleza"]
          production_note: "Foto da cliente sorrindo no espelho"
          reviewed: false

    assert:
      - tool_called: list_posts    # looks up Patricia's posts first
      - tool_called: approve_post  # then approves the pending one
      - tool_arg: { tool: approve_post, key: post_id, contains: "post_abc123" }
      - no_empty_promise: true

  - id: missing_required_field
    message: "cadastra a Dona Maria, confeitaria em Campinas"
    operator: { name: "Elenice", jid: "5511999990000" }

    fixtures:
      customers: []
      posts: []

    assert:
      - tool_not_called: create_customer  # phone is missing, should ask first
      - reply_contains: "telefone"        # asks for the missing field
      - no_empty_promise: true

  - id: ambiguous_name_match
    message: "me fala da Maria"
    operator: { name: "Elenice", jid: "5511999990000" }

    fixtures:
      customers:
        - name: "Maria Silva"
          type: "Confeitaria"
          city: "São Paulo"
        - name: "Maria Santos"
          type: "Padaria"
          city: "Rio de Janeiro"
      posts: []

    assert:
      - tool_called: find_customer
      - reply_contains: "Maria Silva"     # mentions both matches
      - reply_contains: "Maria Santos"
```

### Assertion types

All deterministic except `llm_judge`:

| Assertion | What it checks |
|---|---|
| `tool_called` | Named tool was called at least once |
| `tool_not_called` | Named tool was NOT called |
| `tool_arg` | Tool was called with arg key containing value |
| `reply_contains` | Reply text contains substring |
| `reply_not_contains` | Reply text does NOT contain substring |
| `no_empty_promise` | If reply says "vou/cadastrei/atualizei/aprovei", matching write tool was called |
| `max_tool_calls` | Total tool invocations <= N |
| `llm_judge` | Binary LLM judge with criteria string (optional, `--judges` flag) |

### Mock layer

Structured fixtures replace text parsing. Mock executor formats output identically to real `ToolExecutor` methods, one place to update when format changes.

```go
type Fixtures struct {
    Customers []MockCustomer `yaml:"customers"`
    Posts     []MockPost     `yaml:"posts"`
}
```

### Performance

The eval must be fast and cheap enough to run on every prompt/tool change without thinking twice.

**Parallel execution.** All cases are independent (no shared state, no fixtures that leak between runs). Run all 13 concurrently with a `sync.WaitGroup`. Wall time = slowest single case, not the sum.

**Prompt caching.** The system prompt + 11 tool definitions are identical across every case. With Anthropic's prompt caching, the first case pays full price for that prefix, the remaining 12 pay ~10%. This is the biggest cost lever. To maximize cache hits, all cases must use the same `system` and `tools` params (which they naturally do, since we're testing one agent).

**Target budget:** 13 cases in parallel, ~15s wall time, ~$0.10 per run (deterministic mode, no LLM judges). With `--judges`, add ~$0.05 for Gemini Flash calls.

## Waves

### Wave 1: Delete and rebuild (DONE)

Delete `api/internal/agent/eval.go` and `api/internal/agent/cases/*.yaml`. Rebuild the eval from scratch.

- [x] `api/internal/agent/eval.go` — runner with structured mock executor and assertion engine. All cases run in parallel (`sync.WaitGroup`, one goroutine per case). `MockExecutor` implements the same tool dispatch as `ToolExecutor` but reads from `Fixtures` structs instead of PocketBase. Output format for each mock tool matches the real implementation (same `fmt.Fprintf` patterns from `tools.go`).
- [x] `api/internal/agent/cases/core.yaml` — the 13 cases from the tables above, in the new format with structured fixtures and flat assertions.
- [x] `api/cmd/eval-agent/main.go` — wired to new runner. Table output showing case ID, assertion results (pass/fail per assertion), cost (input/output tokens, wall time, tool round-trips). Saves run to `runs/agent-TIMESTAMP.json`. `--verbose` flag prints full reply text and tool call log per case. `--case` flag filters by case ID substring.
- [x] `Makefile` — `make eval-agent` target pointing to new runner (unchanged).

**Notes:**
- Added `--case` flag to `eval-agent` to run specific cases by ID substring (comma-separated), avoiding full suite runs during iteration.
- `typos_and_grammar_errors` case uses a complete phone number (19 99999-1234) to avoid the agent fixating on incomplete phone instead of testing typo tolerance. Assertion checks reply mentions "Maria" (proving it understood the misspelling) rather than requiring `create_customer` call, since the agent reasonably confirms before creating with heavily misspelled input.
- `no_empty_promise` assertion only matches first-person past tense verbs (cadastrei, atualizei, etc.) to avoid false positives from past participles (cadastrada) and infinitives (para cadastrar).
- `duplicate_customer` case has `tool_not_called: create_customer`. Note that the real `create_customer` tool has built-in duplicate detection, so calling it directly would also work. The test checks the ideal behavior (agent checks proactively).

**Gate:**
- [x] `make eval-agent` exits 0, runs all 13 cases, prints table to stdout
- [x] Each case has at least 2 assertions, all passing
- [x] `runs/agent-*.json` file is written with token counts and wall time per case
- [x] `go vet ./internal/agent/...` passes

### Wave 2: LLM judges

Add `llm_judge` assertion type for subjective checks that deterministic assertions can't cover (tone, hallucination detection).

**New files:**
- `api/internal/baml/baml_src/agent_judges.baml` — judge prompts using reasoning-first pattern (same structure as content judges in `judges.baml`). Two judges: `tone` (informal pt-BR, short, uses operator name, no markdown) and `no_hallucination` (reply only references data from fixtures/tool results).

**Updated files:**
- `api/internal/agent/eval.go` — `llm_judge` assertion calls BAML judge, parses binary verdict.
- `api/cmd/eval-agent/main.go` — `--judges` flag enables LLM judge assertions (skipped by default like content eval). Judge verdicts appear in table and run JSON.
- Cases in `core.yaml` — add `llm_judge` assertions to `approve_sends_content` (no hallucination), `impossible_request_honest` (tone), `vague_request_asks_clarification` (tone).

**Gate:**
- `make eval-agent` still runs without judges (deterministic only), exits 0
- `cd api && go run ./cmd/eval-agent --judges` runs with LLM scoring, judge verdicts visible in table
- Run JSON includes judge reasoning strings

### Wave 3: Grow from production

Review production `agent_actions` logs for failure patterns not yet covered. Add cases for each. Target: 15-20 total.

**Updated files:**
- `api/internal/agent/cases/core.yaml` or new YAML files — new cases derived from production failures.
- `api/internal/agent/eval.go` — add regression/capability tags if set grows past 20 cases. `--regression-only` flag for CI.
- `Makefile` — `eval-agent-ci` target if tagging is added.

**Gate:**
- At least 7 new cases added, each traceable to a production log entry or observed failure
- `make eval-agent` exits 0 with all cases passing
- If tags are added: `make eval-agent-ci` runs regression subset only

## Consequences

- The 25 existing test cases are deleted. No migration. If any of them described a genuinely useful scenario, it will surface again from production logs in Wave 3.
- Agent eval becomes useful: each case maps to a bug that actually happened or a flow that would cause immediate user pain if broken.
- LLM judges add cost (~$0.05/run with Gemini Flash). Default mode remains free (deterministic only).
- No CI gate until Wave 3. Until then, `make eval-agent` is a manual check before prompt or tool changes.
- No user simulation, automated case generation, or eval UI. The agent handles single messages, manual case creation is fine at <50 cases, and terminal + JSON is sufficient.
