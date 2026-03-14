# PEP-026: Agent robustness

**Status:** In Progress
**Date:** 2026-03-14
**Depends on:** PEP-025

## Context

The agent (PEP-025) works for the happy path but falls apart on real conversations. A real session exposed this:

```
agent: Nika cadastrada! (Moda Feminina sob Medida, São Caetano do Sul)
user:  O nome do negócio é "roupas confortáveis da Nika"
agent: Deixa eu verificar e já atualizo!
user:  Atualizou?
agent: Peço desculpa! Confirma a alteração do nome para "Roupas Confortáveis da Nika"?
user:  Sim
agent: (no response)
user:  Fez?
agent: Deixa eu fazer agora!
user:  fez?
agent: deixa eu fazer agora..
(end, agent never did anything)
```

Three bugs compound here:

1. **`update_customer` can't change the business name.** The `name` field identifies the customer (fuzzy lookup). The updatable fields are type, city, phone, target_audience, brand_vibe, quirks. There is no `new_name` field. The agent literally cannot do what was asked.

2. **Agent promises actions without calling tools.** When it can't fulfill a request (or is uncertain), Claude produces text like "Deixa eu verificar e já atualizo!" without calling any tool. The loop sees zero tool calls, treats the text as the final reply, and sends it. The operator thinks something happened. Nothing happened.

3. **Conversation history loses tool context.** `StoreMessage` saves only the text reply. Tool calls and results are discarded. On follow-up messages, Claude sees text history but has no memory of what tools it called (or didn't call). It keeps saying "let me do it" because it doesn't know it already failed.

These are three different failure modes, but they share a root: the agent has no guardrails against making empty promises.

The eval suite doesn't catch this because:
- All eval cases are single-turn (one message in, one response out). Multi-turn loops where the agent gets stuck are invisible.
- Eval uses mock tool results injected via context strings. The real tool executor, state machine, and conversation storage are never exercised together.
- There are no eval cases for "agent should admit it can't do something."

## Waves

### Wave 1: Fix all three root causes

Fix the concrete bugs, add prompt guardrails, and store tool context in conversation history.

**1. Add `new_name` field to `update_customer`**

Files: `api/internal/agent/tools.go`, `api/internal/agent/params.go`, `api/internal/agent/router.go`

- Add `new_name` (optional string) to the `update_customer` tool schema, described as "Novo nome do negócio (se quiser renomear)"
- Add `NewName *string` field to `CustomerUpdateParams`
- In `executeCustomerUpdate`, if `NewName` is set, update the `name` field on the record and include "nome" in the updated fields list

**2. Harden the system prompt**

File: `api/internal/agent/prompt.go`

Add these rules to `buildSystemPrompt`:

- "NUNCA diga que vai fazer algo sem chamar a ferramenta correspondente. Se não conseguir fazer, diga honestamente."
- "Se a operadora pedir algo que nenhuma ferramenta suporta, diga que não consegue fazer isso ainda."
- "Quando uma ferramenta de escrita for chamada, a confirmação é automática pelo sistema. NÃO peça confirmação no texto da resposta, o sistema já faz isso."

The third rule addresses a subtler issue: Claude sometimes writes "Confirma?" in its reply text while also calling the write tool. The state machine already handles confirmation. Double-asking confuses the operator.

**3. Store tool call summaries in conversation history**

Files: `api/internal/agent/agent.go`, `api/internal/agent/conversation.go`

After the tool-use loop completes, append a brief tool summary to the stored assistant message. Format:

```
[reply text]\n\n[Ferramentas: find_customer("Nika") -> encontrada, update_customer(name="Nika", new_name="Roupas Confortáveis da Nika") -> confirmação pendente]
```

This is stored in `agent_conversations` but stripped before sending to WhatsApp. On follow-up messages, Claude sees what tools were called previously, so it knows whether it actually acted or just talked.

Implementation: add a `ToolSummary` field to `agentResult`. In `processWithTools`, build the summary from `tuResult.ToolsCalled`. In `sendAndLog`, store `replyText + "\n\n" + toolSummary` in conversations but send only `replyText` to WhatsApp. When building Claude messages from history, include the full stored content (with tool summary).

**4. Add eval cases for edge cases**

File: `api/internal/agent/cases/edge_cases.yaml`

New test cases:

| ID | Scenario | What it tests |
|---|---|---|
| `edge_rename_customer` | "muda o nome da Nika pra Roupas Confortáveis da Nika" | Agent calls `update_customer` with `new_name` |
| `edge_impossible_request` | "manda um email pra Patricia" | Agent admits it can't do this (no email tool) |
| `edge_vague_update` | "muda a Patricia" (no field specified) | Agent asks what to change instead of doing nothing |
| `edge_update_multiple_fields` | "Patricia mudou pra Contagem e agora faz unha também" | Agent calls `update_customer` with both city and type |
| `edge_confirm_without_pending` | conversation history has agent asking "confirma?" in text, user says "sim" | Agent recognizes no real pending action |
| `edge_customer_not_found` | "atualiza a Fernanda, muda pra SP" (Fernanda doesn't exist) | Agent says customer not found, doesn't hallucinate |

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] `update_customer` tool schema includes `new_name` field
- [x] `executeCustomerUpdate` updates business name when `NewName` is set
- [x] System prompt includes the three new rules
- [x] Stored assistant messages include tool summaries
- [x] WhatsApp replies do NOT include tool summaries
- [x] `make eval-agent` passes with new edge case file (>= 95% pass rate) (100%, 35/35)
- [x] Existing eval cases still pass (no regression)

**Notes (Wave 1):**
- Added fallback reply when Claude calls a write tool but produces no TextBlock. Previously this resulted in silent failures (operator sees nothing). Now returns "{name}, anotado! Aguardando sua confirmação." Both production code and eval harness have this fallback.
- Prompt rule about not asking for confirmation refined to "describe what you'll do, but don't ask 'confirma?'" to avoid suppressing all text output.
- Tool summaries include abbreviated args and results, e.g. `[Ferramentas: find_customer({"query":"Nika"}) -> Nome: Nika...]`

### Wave 2: Multi-turn eval infrastructure

The current eval is single-turn: one message, one response. Real bugs happen across turns. This wave adds multi-turn eval so conversation-level failures are caught before they hit production.

**1. Multi-turn eval infrastructure**

File: `api/internal/agent/eval.go`

Extend `TestCase` to support a `turns` field (list of messages). When `turns` is present, the eval runs them sequentially against the same conversation, carrying tool state between turns. Each turn can have its own graders.

```yaml
- id: multi_rename_flow
  operator: { name: "Elenice", jid: "5511999990000" }
  context: |
    Clientes ativas: 1
      - Nika (Moda Feminina sob Medida, São Caetano do Sul)
  turns:
    - message: "o nome do negócio da Nika é Roupas Confortáveis da Nika"
      graders:
        - type: deterministic
          field: tools_called
          contains: "update_customer"
    - message: "sim"
      graders:
        - type: deterministic
          field: has_reply
          equals: "true"
        - type: llm_judge
          judge: confirmation_flow
          criteria: "Agente confirma que o nome foi alterado"
```

The multi-turn eval loop:
1. Start with empty message history
2. For each turn: append user message, run tool-use loop (with mock tools), evaluate graders, append assistant response to history
3. Report pass/fail per turn

**2. Multi-turn eval cases**

File: `api/internal/agent/cases/multi_turn.yaml`

| ID | Scenario | Turns |
|---|---|---|
| `multi_rename_flow` | Register customer, then rename | "o nome da Nika é Roupas Confortáveis da Nika" -> confirm |
| `multi_create_then_generate` | Create customer, then generate posts | "cria a Ana, manicure, BH" -> "sim" -> "gera post pra Ana" -> "sim" |
| `multi_correction_flow` | Ask to create, realize wrong city, correct | "cria Patricia, salão, SP" -> "não, é BH" -> agent re-asks with BH |
| `multi_stuck_loop` | Agent should not repeat itself | "muda a Patricia" -> "muda o quê?" -> "a cidade" -> "pra onde?" -> "Contagem" -> confirm |
| `multi_context_carryover` | Info from first turn used in second | "como tá a Patricia?" -> "gera post pra ela" (agent resolves "ela" = Patricia) |

**Gate:**
- [ ] `cd api && go build ./...` compiles
- [ ] Multi-turn eval cases run and pass (>= 90% pass rate, multi-turn is harder)
- [ ] Existing single-turn eval cases unaffected
- [ ] The "rename customer" multi-turn flow passes end-to-end

## Consequences

- The agent can rename businesses, closing the most common operator complaint.
- The prompt explicitly forbids empty promises, reducing "let me do it" ghost responses.
- Conversation history includes tool context, so follow-up messages have continuity.
- Multi-turn eval catches conversation-level failures that single-turn eval misses.
- Eval suite grows from ~20 cases to ~30+ cases, covering edge cases and multi-turn flows.
- Tool summaries add ~50-100 tokens per stored message. At 15 messages per buffer, this is ~1500 extra tokens per conversation. Negligible cost impact.
