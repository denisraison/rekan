# PEP-003: Content Rotation System

**Status:** Draft
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

## Wave 1: Dynamic Role Selection in Prompt

**Goal:** Make the prompt accept roles as a parameter instead of hardcoding 3 types. Verify that rotating roles produces different hooks across batches.

**Changes:**

`eval/baml_src/judges.baml`: Add a `ContentRole` class with `name` and `description` fields. Add a `GenerationContext` class with `roles ContentRole[]` and `previousHooks string[]` (empty for now).

`eval/baml_src/content.baml`: Change function signature from `GenerateContent(profile: BusinessProfile)` to `GenerateContent(profile: BusinessProfile, ctx: GenerationContext)`. Replace the hardcoded role block with a Jinja loop over `ctx.roles`. Render each role name and description dynamically.

`eval/generate.go`: Update the `Generate` function to accept roles. Add a `RolePool` variable containing all ~12 roles. Add a `PickRoles(n int, exclude []string) []Role` function that selects n roles randomly, excluding any in the exclude list.

`eval/cmd/eval/main.go`: For each profile, call `PickRoles(3, nil)` (no exclusion yet) and pass to `Generate`. Add a `--roles` flag to specify role names manually for testing (`--roles "bastidor,opinião,marco"`).

**Gate:** Run `make eval-judges` with random role selection. Scores should not regress from current baseline (NAT 12, ESP 12, ACI 12, VAR 10, ENG 11). Then generate 4 batches for the same profile, verify that hooks vary across batches more than they do today.

## Wave 2: Exclusion Context

**Goal:** Pass previously used hooks to the prompt so the model avoids repeating angles.

**Changes:**

`eval/baml_src/content.baml`: Add a conditional block that renders `previousHooks` when the list is non-empty. Something like: "Estes ganchos já foram usados em posts anteriores. NÃO repita o mesmo ângulo: [hooks]".

`eval/hooks.go` (new file): Add a `ExtractHooks(content string) []string` function that pulls the first sentence of each post from generated content. This is the "hook" that gets stored and passed as exclusion context.

`eval/cmd/eval/main.go`: Add a `--chain N` flag that generates N consecutive batches for one profile, passing extracted hooks from each batch to the next. Print a summary of hooks across batches to verify no repetition.

**Gate:** Run `--chain 4 --profile "Cantina da Nonna Rosa"` and verify: (1) no hook repetition across batches, (2) quality scores don't degrade on later batches, (3) the model doesn't just negate exclusions ("Hoje NÃO vou falar do forno" is still about the oven).

## Consequences

**Content quality stays high.** The per-batch quality (judge scores) should hold because the role descriptions are written with the same provocative style that drove engajamento to 11/12. The variety across batches increases because both the structural framing (roles) and the content angles (exclusion) change each time.

**Generation becomes stateful.** The prompt now depends on what was generated before. This means the eval pipeline needs to support chained generation (`--chain`). When we build the product API, it will need to store generated hooks per business.

**The role pool needs maintenance.** 12 roles is a starting point. Some roles will produce better content than others. The eval pipeline can measure this with `--roles` flag to test specific combinations. Underperforming roles get rewritten or dropped.

**Prompt gets longer.** The dynamic role block and exclusion context add tokens. With 3 roles + 6 previous hooks, the prompt grows by ~200 tokens. At Gemini Flash pricing this is negligible. For Sonnet it matters more if we ever switch generators.

**Not all roles work for all businesses.** A tattoo studio has natural antes/depois content; a restaurant doesn't. For now we accept occasional awkward role-business combinations. Role filtering by business type is a future improvement.
