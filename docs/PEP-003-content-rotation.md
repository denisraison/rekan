# PEP-003: Content Rotation System

**Status:** Wave 2 Implemented
**Date:** 2026-02-20

## Context

The content generation prompt now scores well on single-batch quality (naturalidade 12/12, engajamento 11/12, variedade 10/12). But when a user generates content weekly for the same business, the model converges on the same angles. Testing 4 consecutive batches for Cantina da Nonna Rosa showed the 1982 oven story appearing in every batch and the same role interpretations repeating (bastidor = oven, útil = massa myths, pessoal = family origin).

For a product where micro-entrepreneurs generate content weekly, this means followers notice repetition after 2-3 weeks. The prompt is currently stateless: it has no awareness of what was already posted.

We need two mechanisms working together:
- **Exclusion context**: pass previously used hooks/angles so the model avoids repeating them
- **Rotating role pool**: expand from 3 fixed post types (bastidor/útil/pessoal) to ~12, picking 3 per batch so the structural framing itself varies week to week

The generation currently lives only in the eval pipeline (`eval/baml_src/content.baml`). The API is a bare PocketBase instance with no generation endpoint. This PEP covers the prompt and eval changes. API integration (storage, endpoint, UI) is out of scope and will be a separate PEP.

## Role Pool

Current roles (3, hardcoded in prompt):
- Bastidor, Útil, Pessoal

Proposed pool (~12, each with a provocative description that forces non-generic content):

| Role | Description (pt-BR, goes in prompt) |
|------|-------------------------------------|
| Bastidor | Algo do seu trabalho que o cliente nunca vê e ficaria surpreso de saber. |
| Útil | Um erro que todo mundo comete ou uma verdade incômoda sobre o seu nicho. |
| Pessoal | Um momento real que mudou algo no jeito que você trabalha. |
| Cliente | Uma história real de um cliente que te marcou, sem inventar final feliz. |
| Opinião | Algo que você pensa diferente da maioria no seu ramo. Pode ser polêmico. |
| Dia a dia | Um momento comum do seu dia que parece banal mas mostra como o negócio funciona. |
| Antes/depois | Uma transformação que você fez, mostrando o processo e não só o resultado. |
| Tendência | Algo que mudou no seu mercado e como isso afeta quem te contrata. |
| Pergunta | Uma dúvida real que você tem ou que seus clientes sempre trazem, aberta pra debate. |
| Marco | Algo que você conquistou no negócio e o que aprendeu no caminho. |
| Temporada | Algo que muda no seu negócio nessa época do ano. |
| Desafio | Um problema real que você enfrenta no negócio e como lida com ele. |

The pool is not exhaustive. New roles can be added over time. Not all roles make sense for all business types (antes/depois is natural for salões and tattoo studios, less so for restaurants). Role filtering by business type is a future optimization, not part of this PEP.

## Wave 1: Dynamic Role Selection in Prompt (Implemented)

**Goal:** Make the prompt accept roles as a parameter instead of hardcoding 3 types. Verify that rotating roles produces different hooks across batches.

**Changes (as implemented):**

`eval/baml_src/judges.baml`: Added `ContentRole` class with `name string` and `description string`.

`eval/baml_src/content.baml`: Changed signature to `GenerateContent(profile: BusinessProfile, roles: ContentRole[])`. Replaced hardcoded role block with a Jinja `{% for r in roles %}` loop rendering each role name and description.

`eval/role.go` (new): `Role` struct, `RolePool` var with all 12 roles and pt-BR descriptions, `PickRoles(n int, exclude []string) []Role` using `math/rand/v2`.

`eval/generate.go`: `Generate()` now accepts `[]Role`, converts to `[]types.ContentRole`, passes to BAML.

`eval/judge.go`: Added `toBamlRoles()` conversion helper.

`eval/cmd/eval/main.go`: Added `--roles` flag (comma-separated names). When set, parses and resolves from pool. Otherwise calls `PickRoles(3, nil)` per profile so each run gets different random roles.

**Design note:** The plan originally proposed a `GenerationContext` wrapper class. We went with `roles: ContentRole[]` as a direct parameter instead, keeping the signature simpler. The `previousHooks` field from `GenerationContext` will be added in Wave 2 as a separate parameter.

**Gate results:** `make eval-fast` with random roles scored 27/28 checks, judges all passing (NAT 4/4, ESP 4/4, ACI 4/4, VAR 4/4, ENG 4/4). No regression. All tests (unit + integration) pass.

## Wave 2: Exclusion Context (Implemented)

**Goal:** Pass previously used hooks to the prompt so the model avoids repeating angles.

**Changes (as implemented):**

`eval/hooks.go` (new): `ExtractHooks(content string) []string` splits generated content into posts (using existing `splitPosts()`), extracts the first sentence of each as the "hook". A sentence ends at the first `.`, `!`, or `?` in the first paragraph.

`eval/hooks_test.go` (new): Tests for `ExtractHooks` covering multi-post content and empty input.

`eval/baml_src/content.baml`: Added `previousHooks: string[]` parameter. Conditional Jinja block renders before `ctx.output_format` when the list is non-empty: "IMPORTANTE: Estes ganchos já foram usados em posts anteriores. NÃO repita o mesmo ângulo, tema ou cena."

`eval/generate.go`: `Generate()` now accepts `previousHooks []string`, passes through to BAML client.

`eval/cmd/eval/main.go`: Added `--chain N` flag. Requires `--profile`. Generates N sequential batches for one profile, accumulating hooks between batches. Prints hook summary to stderr after all batches. Supports `--roles` for fixed roles across batches. Existing callers pass `nil` for previousHooks (no behavior change).

**Gate results:** `--chain 2 --profile "Brigadeiros da Dani"` produced distinct hooks across batches (14/14 heuristics pass). Batch 2 hooks ("O barulhinho do batedor de arame...", "Lavei a mão umas 15 vezes...", "R$ 110,00 bem investidos...") are clearly different angles from batch 1 ("O cheirinho de cravo e canela...", "Minha mão chega a cansar...", "4 da manhã..."). `make eval` shows no regression (82/84).

## Consequences

**Content quality stays high.** The per-batch quality (judge scores) should hold because the role descriptions are written with the same provocative style that drove engajamento to 11/12. The variety across batches increases because both the structural framing (roles) and the content angles (exclusion) change each time.

**Generation becomes stateful.** The prompt now depends on what was generated before. This means the eval pipeline needs to support chained generation (`--chain`). When we build the product API, it will need to store generated hooks per business.

**The role pool needs maintenance.** 12 roles is a starting point. Some roles will produce better content than others. The eval pipeline can measure this with `--roles` flag to test specific combinations. Underperforming roles get rewritten or dropped.

**Prompt gets longer.** The dynamic role block and exclusion context add tokens. With 3 roles + 6 previous hooks, the prompt grows by ~200 tokens. At Gemini Flash pricing this is negligible. For Sonnet it matters more if we ever switch generators.

**Not all roles work for all businesses.** A tattoo studio has natural antes/depois content; a restaurant doesn't. For now we accept occasional awkward role-business combinations. Role filtering by business type is a future improvement.
