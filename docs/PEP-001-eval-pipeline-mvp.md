# PEP-001: Eval Pipeline MVP

**Status:** Wave 1 Complete
**Date:** 2026-02-19

## Context

Rekan generates Instagram content (captions, hashtags, stories) for Brazilian micro-entrepreneurs. The quality of that content determines whether anyone uses the product. Right now there's no way to measure whether a prompt change helped or hurt, so iteration is guesswork.

We need a repeatable evaluation pipeline that's cheap enough to run after every prompt edit. The full vision (eval.md) includes multi-LLM juries, synthetic dataset generation, and scoring formulas. For MVP, we're cutting to what actually unblocks prompt iteration: heuristic checks, binary LLM judges, a set of generated test profiles, and a disciplined optimization loop.

What we're explicitly deferring: multi-LLM jury (Layer 3), conditional prompting dataset generation, Silver/Gold/Platinum dataset tiers, weighted scoring formulas.

The eval pipeline lives in its own top-level `eval/` directory, separate from the API. It's a development tool, not part of the product.

## Project structure

```
eval/
  cmd/eval/main.go      CLI entrypoint
  heuristic.go           deterministic checks
  judge.go               LLM judge runner
  testdata/              generated business profiles (JSON)
  baml_src/              BAML prompt definitions for judges
```

## Wave 1: Test Profiles and Heuristic Checks

Build the foundation: generated test profiles and deterministic checks that catch structural failures instantly at zero cost.

### Test profiles

Generate 10 to 15 JSON profiles in `eval/testdata/`. Claude generates these during this wave, not at runtime. They're committed to the repo as fixtures. Each profile represents a realistic MEI (micro-empreendedor individual) that Elenice would onboard.

Diversity requirements: spread across business types (salão, restaurante, personal trainer, nail designer, confeitaria, barbearia, loja de roupas, etc.), cities beyond São Paulo (Manaus, Curitiba, Salvador, Recife, Belo Horizonte, Florianópolis), vibes (casual, premium, family, trendy), and target audiences (women 25-45, men 18-35, families, young professionals).

Each profile needs: business name, business type, city, neighbourhood, services with prices in BRL, target audience, brand vibe, and quirks (e.g. "fecha segunda", "só delivery", "aceita encomenda com 48h de antecedência").

### Heuristic checks

Create `eval/heuristic.go`. These are deterministic pass/fail checks on generated content:

- Business name appears in output
- City or neighbourhood mentioned
- Contains hashtags (at least 3)
- Contains a call to action
- Uses Brazilian Portuguese informal markers (tá, bora, gente, né, pra) and does not use Portugal Portuguese markers (consigo, telemóvel, autocarro)
- Caption length within Instagram limits (under 2200 chars)
- At least one production note / photo suggestion present

Each check returns a name, pass/fail, and a short reason on failure. The function takes the generated content string and the business profile, returns a list of check results.

### Gate

`go test ./eval/...` passes. Heuristic checks run against a hardcoded sample output and correctly detect both passing and failing cases.

### Status: Done (2026-02-19)

Implemented in `eval/`. 12 test profiles committed, 7 heuristic checks passing (21 test cases). Accent-normalized matching for business names and locations. Unicode-aware hashtag and word-boundary regexes for Portuguese content. No external dependencies.

## Wave 2: Binary LLM Judges

Five separate judges, each asking one yes/no question about one quality dimension. Uses BAML for prompt definitions and structured output parsing.

### BAML setup

Add BAML to the project. Create `eval/baml_src/` with judge prompt definitions. Each judge is a BAML function that takes the business profile and generated content, returns a structured response with reasoning (2-3 sentences) and a boolean verdict.

### The five judges

All defined in `eval/baml_src/`:

- **Naturalidade**: Does this sound like a real Brazilian Instagram user wrote it? Looks for informal language, natural emoji use, conversational tone. Fails on corporate speak, Portugal Portuguese, robotic patterns.
- **Especificidade**: Does the content reference this specific business? Name, city/neighbourhood, actual services. Fails if you could swap in any other business name and the caption still works.
- **Acionável**: Could the owner copy-paste and post today? Needs complete caption, hashtags, CTA, and a production note about what photo/video to take.
- **Variedade**: Are the generated posts genuinely different from each other? Mix of formats (educational, behind-the-scenes, before/after, social proof, tips). Fails if all posts follow the same template.
- **Engajamento**: Would this make someone stop scrolling? Strong hook in the first line, reason to read/comment/save/share.

Each judge runs at temperature 0.1 for consistency. Reasoning is generated before the verdict (G-Eval pattern).

### Judge runner

Create `eval/judge.go`. Takes a business profile, generated content, and judge name. Calls the BAML function, returns the structured result. A `RunAllJudges` function runs all five and returns the aggregate.

### Gate

`go test ./eval/...` passes. A test calls each judge with a known-good and known-bad sample, verifies the judges return sensible verdicts. Does not need to be deterministic (LLM output varies), but the known-bad sample should fail at least 3 of 5 judges consistently.

## Wave 3: Eval CLI and Optimization Loop

Wire everything together into a runnable command and document the optimization workflow.

### Eval command

Create `eval/cmd/eval/main.go` as a standalone CLI. It:

1. Loads test profiles from `eval/testdata/`
2. Generates content for each profile using the current system prompt
3. Runs heuristic checks on each output
4. Runs LLM judges on each output (optional, enabled with `--judges` flag)
5. Prints a summary table: business name, heuristic pass count, judge verdicts per criterion
6. Exits non-zero if any business fails all heuristics

The system prompt lives in a single file (`eval/prompts/system.txt` or a BAML file) that the optimizer edits.

Add a `make eval` and `make eval-judges` target to the Makefile.

### Optimization loop

Document in CLAUDE.md or a skill file. The loop is:

1. Run eval with judges, identify the weakest criterion across all businesses
2. Pick one failing business, run verbose eval to read the output and judge reasoning
3. Form one hypothesis about what to change in the system prompt
4. Make one edit
5. Re-run eval, compare: did the target judge improve? Did anything regress?
6. Keep or revert. Max 5 cycles per session.

### Gate

`make eval-judges` against the test profiles produces a readable summary table. The system prompt exists and generates content that passes at least 50% of heuristic checks on the first try.

## Consequences

- Prompt iteration becomes measurable. Every change has a before/after comparison.
- The eval runs in under 2 minutes with judges, under 1 second without. Cheap enough to never skip.
- We're locked into BAML for prompt definitions early. This is intentional since it gives us typed outputs and makes judges portable across model providers later.
- No multi-LLM jury means we have single-model bias in evaluation. Acceptable for MVP since we're iterating fast, not shipping to production yet.
- Test profiles are generated once and committed. They'll be replaced with real client data after launch.
- The eval pipeline is decoupled from the API. It can evolve independently and doesn't add weight to the production binary.
