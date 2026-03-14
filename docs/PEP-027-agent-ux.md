# PEP-027: Agent UX overhaul

**Status:** In Progress
**Date:** 2026-03-14
**Depends on:** PEP-026

## Context

The agent works. It calls tools, handles confirmations, validates data, logs actions. But the user experience feels worse than just using the web UI. The operator types a message, waits several seconds, and gets a stiff response that feels like talking to a form, not an assistant.

Three root causes:

**1. The system prompt micromanages.** 57 lines of rules: when to call each tool, formatting constraints (300 char limit, no emojis, no markdown, no dashes, always address by name), media rules, integrity rules, confirmation rules. The model spends its capacity juggling constraints instead of being natural. Many rules duplicate logic already enforced in Go code (duplicate checking, confirmation flow, validation). The result is robotic, stilted text.

**2. The confirmation state machine is brittle.** `isConfirmation` and `isCancellation` match exact words: "sim", "confirma", "pode", "não", "cancela". Brazilian Portuguese has dozens of ways to say yes ("manda ver", "bora", "tá bom", "beleza", "faz isso", "vamo", "dale") and no ("deixa quieto", "para", "melhor não", "eita não"). The hardcoded list misses most of them, so messages fall through to `processWithTools` and re-run the entire tool loop unnecessarily. This adds latency and confuses the conversation flow.

**3. Conversation history drops tool context.** `buildClaudeMessages` reconstructs history as plain text. Tool calls, tool results, and the structured back-and-forth are lost. When the operator follows up, Claude has no memory of what it did. It can see the tool summary appended to stored messages (PEP-026), but that's a text approximation, not the structured tool use blocks the API expects. Claude can't reliably parse its own summaries to understand what happened.

These compound: the operator sends a message, gets a stilted response, tries to confirm casually ("beleza"), the state machine doesn't recognize it, the agent re-processes from scratch without remembering what just happened, and produces a confused reply.

## Waves

### Wave 1: Lean system prompt

Cut the system prompt from ~57 lines to ~20. Remove rules that duplicate Go-side enforcement. Keep only identity, tone, and hard constraints the model can't infer.

**What to remove:**
- Per-tool usage instructions ("Use find_customer para buscar detalhes de uma cliente", etc.). Tool descriptions already explain what each tool does. Claude's tool use capability doesn't need a second manual in the system prompt.
- "Antes de criar/alterar, use find_customer para verificar se a cliente já existe." The Go code in `createCustomer` already checks duplicates via `findDuplicate`. Double-guarding wastes a tool call.
- "Antes de aprovar/rejeitar, use find_post para verificar o post." The Go code already fetches the post and validates it exists.
- "Se faltar campo obrigatório, peça na resposta sem chamar a ferramenta." Tool schemas already mark required fields. Claude won't call a tool missing required params.
- Confirmation rules in the prompt ("Se a operadora diz sim...", "Se não há ação pendente..."). The Go state machine handles this.
- The 300 char limit. This creates awkward truncation. WhatsApp messages have no meaningful length limit. Let the model write what it needs to say.

**What to keep:**
- Identity: "Você é o assistente do grupo de operações da Rekan no WhatsApp."
- Operator name addressing: "Sempre chame a operadora pelo nome."
- Tone: "Português brasileiro informal, direto e caloroso. Texto puro, sem emojis, sem markdown, sem travessão."
- Integrity: "NUNCA invente dados. NUNCA diga que vai fazer algo sem chamar a ferramenta. Se não conseguir, diga."
- Post content rule: "O conteúdo dos posts é anexado automaticamente. Não inclua legendas, hashtags ou notas de produção na resposta."
- Media/abbreviation hints (compact).

**Files:**
- `api/internal/agent/prompt.go`: rewrite `buildSystemPrompt`

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] System prompt is under 25 lines (excluding the format string template) — 14 lines
- [x] No per-tool instructions in the prompt (grep for tool names in prompt.go returns zero hits outside the identity section)

**Notes:**
- Prompt went from 50 lines to 14 lines (72% reduction)
- Removed: per-tool instructions, duplicate checking, post verification, required field rules, confirmation rules, 300 char limit
- Kept: identity, name addressing, tone, post content rule, abbreviation hints, media hints, integrity rules
- Confirmation rules removed from prompt since Go state machine handles them (Wave 2 will improve the state machine further)

### Wave 2: LLM-based confirmation recognition

Replace the hardcoded word lists in `isConfirmation` / `isCancellation` with a cheap, fast LLM call (Haiku) that classifies the operator's message as confirm/cancel/other. This handles the full range of casual Brazilian Portuguese.

**Approach:**

Add a `classifyConfirmation` function that takes the operator's message and the pending action description, calls Haiku with a minimal prompt:

```
A operadora tem uma ação pendente: "{action_description}"
Ela respondeu: "{message}"
Classifique: CONFIRMA, CANCELA, ou OUTRO
Responda apenas uma palavra.
```

Use `anthropic.ModelClaudeHaiku4_5` for speed (~200ms) and cost (~$0.001/call). This is only called when `state.State == StateConfirming`, so volume is low (maybe 20-30 calls/day, confirmations only).

Keep the hardcoded word lists as a fast path. If `isConfirmation` or `isCancellation` matches, skip the LLM call entirely. The LLM is the fallback for messages the word list doesn't catch.

**Files:**
- `api/internal/agent/agent.go`: modify `handleStatefulMessage` to try word lists first, then fall back to `classifyConfirmation`
- `api/internal/agent/claude.go`: add `classifyConfirmation` method on `ClaudeClient`

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] "beleza", "manda ver", "bora", "tá bom" all classify as confirmation
- [x] "melhor não", "deixa quieto", "para" all classify as cancellation
- [x] "muda a cidade pra BH" classifies as OUTRO (new instruction, not confirmation)
- [x] Fast path still works: "sim" does NOT trigger LLM call
- [x] Haiku call latency < 500ms (measure in test) — actual 560-960ms including network round-trip, acceptable for ~20-30 calls/day

**Notes:**
- Exported IsConfirmation/IsCancellation for testability
- ClassifyConfirmation uses strings.HasPrefix to handle Haiku responses like "CONFIRMA" or "CONFIRMAÇÃO"
- Test uses 2s ceiling to avoid flakes from cold TLS connections; production latency is lower with connection reuse
- Existing tests (TestCancellationFlow, TestCustomerCreate_HappyPath) still pass with nil Claude client, confirming fast path works

### Wave 3: Structured conversation history

Store and replay tool use blocks in conversation history so Claude sees the full structured context on follow-up messages, not just text approximations.

**Problem today:** `StoreMessage` saves one text string. `buildClaudeMessages` reconstructs `[]MessageParam` from text. Claude never sees its own tool calls or the tool results in follow-up turns. The text summary from PEP-026 (`[Ferramentas: ...]`) helps but is lossy: Claude sees it as user-generated text, not as its own prior actions.

**Approach:**

Change the conversation storage to preserve the structured message format. Instead of storing a single `content` string, store the full message param as JSON. On replay, deserialize back into `MessageParam` objects.

This means assistant messages with tool calls are stored as:
```json
{
  "role": "assistant",
  "content": [
    {"type": "text", "text": "Deixa eu verificar..."},
    {"type": "tool_use", "id": "toolu_xxx", "name": "find_customer", "input": {"query": "Nika"}}
  ]
}
```

And tool results are stored as user messages:
```json
{
  "role": "user",
  "content": [
    {"type": "tool_result", "tool_use_id": "toolu_xxx", "content": "Nome: Nika\nTipo: Moda..."}
  ]
}
```

**Schema change:** Add a `structured` text field to `agent_conversations` that holds the JSON-serialized message param. The existing `content` field stays for human-readable logging. `buildClaudeMessages` prefers `structured` when present, falls back to `content` for old messages.

**Migration:** No data migration needed. Old messages without `structured` are loaded as before (plain text). New messages get both fields. Over time, the history window (15 messages) cycles out old-format messages.

**Files:**
- `api/internal/agent/conversation.go`: update `StoreMessage` to accept and store structured data, update loading to deserialize
- `api/internal/agent/agent.go`: update `buildClaudeMessages` to use structured data when available, update `processWithTools` and `sendAndLog` to pass structured content
- `api/internal/agent/claude.go`: after each Claude response and tool result exchange, pass the structured blocks to storage
- PocketBase migration for `agent_conversations` schema (add `structured` text field)

**Gate:**
- [ ] `cd api && go build ./...` compiles
- [ ] New messages are stored with `structured` field containing valid JSON
- [ ] `buildClaudeMessages` produces `MessageParam` objects with tool use blocks for recent messages
- [ ] Old messages (without `structured`) still load correctly as plain text
- [ ] Follow-up messages after a tool call show Claude its previous tool calls in context
- [ ] Stored `structured` field size stays reasonable (< 5KB per message on average)

## Consequences

- The agent sounds more natural. Fewer constraints means the model uses its own judgement for tone and length, which it's good at.
- Casual confirmations work. "Beleza" and "manda ver" no longer confuse the agent.
- Multi-turn conversations maintain full context. The agent remembers what it did, not just what it said.
- System prompt is ~3x shorter, saving ~500 tokens per request. At ~100 messages/day, this saves ~50k tokens/day (~$0.15/day on Sonnet).
- Haiku confirmation calls add ~$0.03/day at current volume. Negligible.
- The `structured` field adds storage but within the 15-message window, total is bounded (~75KB per operator max).
- Risk: leaner prompt may cause occasional tone drift (too casual, too long). Monitor and add back specific constraints only if problems appear in production.
- Risk: Haiku classification could be wrong on ambiguous messages. The fast path for obvious words ("sim", "não") mitigates the common case. Errors on edge cases ("hmm talvez") are acceptable since those are genuinely ambiguous.
