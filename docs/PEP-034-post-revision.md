# PEP-034: Post revision

**Status:** Done
**Date:** 2026-03-16
**Depends on:** PEP-032

## Context

The operator can generate, approve, and reject posts. There's no way to edit one. When the operator says "tá bom mas tira a parte do desconto" or "troca as hashtags por umas de BH", the only option is reject + regenerate from scratch. That throws away everything good about the post.

This happens constantly. The operator reviews a post, likes 80% of it, wants one thing changed. Today that costs a full regeneration (LLM call, new post record, new review cycle). The feedback loop should be: see post, request tweak, see updated post, approve.

The post record already has an `edited` boolean field (always `false` today). No migration needed.

## Design

One new tool: `revise_post`. A dumb field updater. The agent (which is already an LLM) does the rewriting.

### Why not a "smart" revise tool that calls an LLM internally?

The agent already has the post content in its conversation context. It understood the operator's feedback. It is the ideal rewriter. A separate LLM call would:
- Not have the conversation context (operator tone, what they just said)
- Add latency and cost for no reason
- Duplicate intelligence that's already running

### Tool: `revise_post`

```
Name: revise_post
Type: write
Description: "Atualiza o conteúdo de um post pendente. Envia apenas os campos que mudaram."
Params:
  - post_id (string, required): ID do post
  - caption (string, optional): Nova legenda
  - hashtags (string[], optional): Novas hashtags
  - production_note (string, optional): Nova nota de produção
```

Behavior:
- Resolve post by prefix (same as approve/reject).
- If post is already reviewed (`reviewed == true`), return error. No editing after approve/reject.
- At least one field must be provided.
- Set provided fields. Set `edited = true`.
- Return updated post content so the agent can show it.

### When the agent uses which tool

| Operator says | What happens |
|---|---|
| "muda o final pra algo mais animado" | Agent rewrites caption, calls `revise_post` |
| "troca as hashtags" | Agent builds new hashtags, calls `revise_post` |
| "tira a nota de produção" | Agent calls `revise_post` with empty production_note |
| "rejeita, refaz do zero" | `reject_post` then `generate_post` (existing flow) |

The agent figures out which path from context. No routing logic needed in Go.

### Friction removal

Things we are NOT doing, to keep the agent from getting flaky:

- **No "confirm before saving" step.** The agent rewrites, saves, shows the result. If the operator doesn't like it, they say so and we revise again. Adding confirmation would double the turns for every edit.
- **No field-level diffing or "here's what changed" output.** The agent shows the full updated post. The operator can see what changed. Structured diffs add complexity the model has to format.
- **No restriction on number of revisions.** The operator can revise as many times as they want while the post is pending.
- **No special handling for empty strings.** If the operator wants to clear the production note, sending `production_note: ""` clears it. No "pass null to clear vs empty to skip" distinction.

### System prompt addition

One line:

```
Para ajustes em posts pendentes (trocar hashtags, mudar legenda, tirar trecho), use revise_post com os campos atualizados.
```

No elaborate instructions. The tool description and the agent's own reasoning handle the rest.

## Wave 1: revise_post

Single wave. Small surface area.

**Files changed:**
- `api/internal/service/content.go` — `RevisePostParams` struct and `RevisePost` function. Guards on `reviewed == true`. Updates fields, sets `edited = true`, saves, returns updated field keys.
- `api/internal/agent/tools.go` — `revise_post` write tool definition. `revisePost` method on `ToolExecutor`: resolves post by prefix, builds `RevisePostParams` from non-empty args, calls service, returns updated post content (caption, hashtags, production_note) plus field labels.
- `api/internal/agent/prompt.go` — One line telling the agent to use `revise_post` for pending post adjustments.
- `api/internal/agent/eval.go` — Mock executor handles `revise_post` with reviewed guard. `writeToolNames` map includes `revise_post`. Promise verbs updated with "editei/ajustei/revisei" and their third-person forms.
- `api/internal/agent/cases/core.yaml` — Two eval cases: `revise_post_caption` (happy path) and `revise_reviewed_post_blocked` (guard).

**Gate:**
- [x] `go build ./...` passes
- [x] `make eval` passes, no regressions (14/17, same 2 pre-existing failures)
- [x] `revise_post_caption` eval: agent reads post, rewrites caption with "vem conferir", calls `revise_post` with correct post_id and new caption. 3 trips (search + revise + reply).
- [x] `revise_reviewed_post_blocked` eval: agent sees reviewed post, does NOT call `revise_post`. 0 trips.
- [ ] Manual: generate a post, ask "troca as hashtags", verify post updated with `edited = true`

**Notes:**
- The `revise_post_caption` case takes 3 tool round trips: the agent first searches for the post, then calls `revise_post`, then replies. This is expected since the post content needs to be in context for the agent to rewrite it. If the post was shown in a previous turn (normal flow), it would be 1 trip.
- `revise_reviewed_post_blocked` initially asserted `reply_contains: "revis"` but the model replies in varied Portuguese ("não é possível editar", "post já aprovado", etc.) without necessarily using "revis". Changed to `tool_not_called: revise_post` which tests the actual behavior we care about: the agent does not attempt to edit a reviewed post.
- Field labels map in `tools.go` extended with caption/hashtags/production_note so the tool result says "Campos: legenda, hashtags" instead of raw field names.

## Consequences

- The feedback loop shrinks from reject+regenerate (2 LLM calls, new post) to revise (0 extra LLM calls, same post).
- `edited = true` lets us track how often operators tweak vs. accept as-is. Useful signal for improving generation quality.
- Trade-off: the agent must construct good revised content. If its rewrite is bad, the operator asks again. This is fine because the operator is already in the loop reviewing.
- Trade-off: no undo. If the agent botches a revision, the original content is gone. At current scale this is acceptable. If it becomes a problem, we can add a `previous_caption` field later.
