# PEP-002: Multi-Model Judge Panel

**Status:** Done
**Date:** 2026-02-20

## Context

The eval pipeline (PEP-001) uses a single model (Gemini 3 Flash) for all 5 judges. In practice, the judges are too lenient: most content passes most judges, making the optimization loop ineffective. If the judges don't catch problems, the optimizer has nothing to push against.

This is a known issue in LLM-as-judge systems. The PoLL paper (Verma et al., 2024) showed that a panel of 3 smaller models from different families outperforms a single GPT-4 judge on Cohen's Kappa (0.763 vs 0.627) while costing 7-8x less. The key finding: diversity of model families matters more than model size. A survey of LLM-as-judge research (arXiv 2411.15594) confirms that adding concrete failure examples to judge rubrics is the single highest-impact calibration technique.

We planned two changes: tighten the judge prompts with failure examples, then add a multi-model panel with majority vote. During implementation we discovered a third problem (judge/heuristic overlap) that required a prompt rewrite.

## Wave 1: Calibrate Judge Prompts with Failure Examples

**Status: Done (insufficient on its own)**

Added "Exemplo de reprovação" sections to each of the 5 judge prompts in `eval/baml_src/judges.baml`. Each example is a short snippet of generated content that should fail that criterion.

The examples were written in pt-BR: corporate press release for naturalidade, name-only generic caption for especificidade, content-idea-not-a-post for acionavel, same-template-three-times for variedade, passive-informational for engajamento.

**Outcome:** These examples were too obvious. They caught content that was clearly bad (corporate English, incomplete drafts) but didn't raise the bar for content that technically follows the generator's instructions. The pass rate barely moved because the generator already produces content that avoids these obvious failures. This became clear only after Wave 2, when the multi-model panel also voted 60/60.

## Wave 2: Multi-Model Panel with Majority Vote

**Status: Done**

### Model selection

Three models from three families, all via OpenRouter:

| Client name | Family | OpenRouter ID | Input $/M | Output $/M |
|---|---|---|---|---|
| `JudgeClient` | Google | `google/gemini-3-flash-preview` | $0.50 | $3.00 |
| `JudgeClientClaude` | Anthropic | `anthropic/claude-haiku-4.5` | $1.00 | $5.00 |
| `JudgeClientDeepSeek` | DeepSeek | `deepseek/deepseek-v3.2` | $0.24 | $0.38 |

### Decisions made

**BAML client override (Option B):** BAML's Go client supports `WithClient(clientName)` as a call option, so we kept the 5 judge functions and override the client at runtime. No need for 15 function copies.

**Error tolerance:** DeepSeek V3.2 intermittently returns 1-token responses that fail BAML parsing. Rather than treating this as fatal, `RunJudge` now collects failed votes with a `Vote.Error` field and computes majority from successful votes only. If 2 of 3 models respond, the verdict uses those 2. Only fails if all 3 models fail. This matches the PEP's original intent: "if one goes down, the eval degrades but doesn't break."

### Files changed

- `eval/baml_src/clients.baml`: Added `JudgeClientClaude` and `JudgeClientDeepSeek` alongside existing `JudgeClient`.
- `eval/judge.go`: `Vote` struct with `Error` field. `RunJudge` fans out to all 3 clients in parallel, tolerates individual failures, majority vote from successful responses. Dissenting reasoning picked for debugging.
- `eval/cmd/eval/main.go`: Verbose mode shows per-model vote breakdown (`JudgeClient:+, JudgeClientClaude:+, JudgeClientDeepSeek:ERR`). Run JSON includes votes with error field.

### Gate results

- `make test-judges`: PASS
- `make eval-judges`: 12/12 profiles, all 5 judges pass, 28.7s runtime
- All 3 models vote unanimously `+` on everything. **60/60 pass rate, same as before.**

The panel infrastructure works, but model diversity alone didn't solve leniency. All 3 models agreed because the evaluation was easy: the judges were checking the same things as heuristics, and the generator already passes those checks.

## Wave 3: Eliminate Judge/Heuristic Overlap

**Status: Done**

After Wave 2, we diagnosed why judges never fail: **the 5 judges overlap with the 7 heuristics.** The heuristics already verify structural compliance (business name present, location mentioned, has CTA, has hashtags, pt-BR markers, caption length, production note). The judges then re-ask the same questions with an LLM. Since the generator is engineered to pass these structural checks, the judges rubber-stamp everything.

### What changed

Rewrote all 5 judge prompts in `eval/baml_src/judges.baml`. Each prompt now:

1. **Explicitly states what heuristics already cover** ("Já foi verificado que X. Sua tarefa é diferente.") to prevent the LLM from re-checking structural compliance.
2. **Evaluates subjective quality beyond compliance**, which is what LLM judges are actually good at.
3. **Uses near-miss failure examples** instead of strawmen. Content that looks good on paper but has subtle issues, the kind of content the generator actually produces.

Judge-by-judge changes:

- **Naturalidade**: Was "uses informal Portuguese." Now: "sounds like a real person, not an AI performing informality." Checks for emoji density, formulaic structure, stacked informal markers, generic filler phrases, uniform enthusiasm.
- **Especificidade**: Was "mentions business name and details." Now: "goes beyond the profile data." The judge receives the full profile and checks whether the content adds anything an AI couldn't generate from the JSON alone (scenes, anecdotes, sensory details).
- **Acionavel**: Was "has legenda + hashtags + CTA + production note." Now: "those elements have quality." Checks hashtag specificity (not generic #empreendedorismo), production note actionability (not "tire uma foto"), CTA variation between posts, caption flow.
- **Variedade**: Was "posts have different formats." Now: "posts have different structures, rhythms, and energy." Explicit template test: swap themes between posts, does the structure still work?
- **Engajamento**: Was "has a hook." Now: "the hook creates genuine curiosity, not a formula." Specific rejected patterns: "Você sabia que...?", "Gente, prepara o coração!", generic "comenta aqui + marca um amigo."

Also rewrote `eval/judge_test.go` `knownGoodContent` fixture. The old fixture was AI-generated content that passed the lenient judges but fails the calibrated ones. The new fixture has authentic voice, invented scenes beyond the profile data, varied structures, and genuine hooks.

### Gate results

Re-judged the same content from Wave 2 (using `--from-run`):

| Judge | Before (Wave 1+2) | After (Wave 3) |
|---|---|---|
| Naturalidade | 12/12 | 0/12 |
| Especificidade | 12/12 | 10/12 |
| Acionavel | 12/12 | 12/12 |
| Variedade | 12/12 | 4/12 |
| Engajamento | 12/12 | 3/12 |
| **Total** | **60/60** | **29/60** |

- `make test-judges`: PASS (known-good 5/5, known-bad 0/5)
- Runtime: 28.7s

Naturalidade (0/12) and engajamento (3/12) are now the clear optimization targets for the generation prompt.

## Consequences

- The optimization loop has signal. Naturalidade and engajamento fail consistently, giving the prompt optimizer concrete criteria to push against.
- Judges and heuristics now cover complementary concerns. Heuristics check structural compliance (fast, deterministic). Judges check subjective quality (slower, requires LLM).
- Acionavel still passes 12/12, suggesting the generator already produces quality structural elements. This judge may need further tightening if the optimization loop stalls.
- Judge cost is ~$0.54 per run (3x the original). Still negligible.
- DeepSeek V3.2 is unreliable (frequent 1-token responses). Error tolerance handles this, but if it persists we should consider swapping it for another model. The panel degrades gracefully to 2-model majority when this happens.
- The `knownGoodContent` test fixture now sets a high bar. If the generation prompt improves enough to pass the calibrated judges, the test will confirm it.
