# Rekan Eval & Optimisation Pipeline

## What Rekan Does

Rekan generates ready-to-post Instagram content (captions, hashtags, stories) for Brazilian micro-entrepreneurs so they can stay consistent on social media without hiring a social media manager.

The quality of that content is everything. If the output sounds robotic, generic, or incomplete, business owners won't use it. This pipeline exists to systematically measure and improve content quality without relying on human evaluators.

---

## The Problem We're Solving

LLM-generated content is hard to evaluate. You can read 10 outputs and feel like they're "pretty good," but that doesn't tell you whether your prompt change actually helped or if you're just pattern-matching to whatever you saw last. We need something repeatable, measurable, and cheap enough to run hundreds of times during development.

The research is clear on a few things: asking an LLM to score content 0 to 10 is unreliable (models can't consistently distinguish a 6 from a 7), single-model evaluation has blind spots (models tend to prefer their own style), and vague multi-dimensional rubrics confuse evaluators. Our pipeline is designed around these findings.

---

## Three Layers of Evaluation

The pipeline has three layers, each adding cost and depth. You pick how thorough you need to be based on what you're doing.

### Layer 1: Heuristic Checks

These are simple, deterministic rules that catch structural problems instantly and for free. They check things like whether the output mentions the business name and city, whether it includes hashtags and calls to action, whether it uses Brazilian Portuguese informal markers (tá, bora, gente) instead of formal or Portugal Portuguese, and whether individual posts fall within reasonable Instagram caption lengths.

Think of this as a smoke test. If the heuristics fail, there's no point running the more expensive LLM judges. These checks also serve as a sanity baseline that never drifts, since they're just pattern matching with no model involved.

### Layer 2: Binary LLM Judges

This is where the real quality assessment happens. Instead of asking one model to rate content across five dimensions on a 10-point scale (which research shows is unreliable), we run five separate judges, each asking one yes-or-no question about one specific quality.

The five judges are:

**Naturalidade** asks whether the content sounds like it was written by a real Brazilian who uses Instagram daily. It looks for informal language, natural emoji use, and conversational tone. Red flags are corporate speak, Portugal Portuguese, and robotic patterns.

**Especificidade** asks whether the content actually references this specific business by name, mentions the city or neighbourhood, and describes real services. The test is simple: could you swap in any other business name and the caption would still work? If yes, it's too generic.

**Acionável** asks whether a business owner could copy-paste this and post it today. Each post needs a complete caption, relevant hashtags, a clear call to action, and a production note about what photo or video to take.

**Variedade** asks whether the generated posts are genuinely different from each other. It looks for a mix of formats like educational carousels, behind-the-scenes content, before-and-after posts, social proof, and quick tips. If all five posts follow the same template with different words, it fails.

**Engajamento** asks whether this content would make someone stop scrolling. Does the first line hook attention? Is there a reason to read more, comment, save, or share?

Each judge follows the same pattern from the G-Eval research: write your reasoning first in two to three sentences, then give your verdict. This order matters. Research shows that when models score first and explain after, the explanation is often rationalisation rather than genuine analysis. Reasoning-first produces more reliable judgments.

All judges run at very low randomness (temperature 0.1) to keep evaluations consistent across runs.

### Layer 3: Multi-LLM Jury

For important decisions like deploying a prompt change to production, we escalate to a panel of judges from different model families. This is based on the PoLL (Panel of LLM evaluators) research, which found that three smaller models from different families (Claude, GPT, Gemini) correlate better with human judgment than a single large model, while being significantly cheaper.

The same five binary questions get sent to all available models. Each criterion is decided by majority vote. The output shows which models agreed and which disagreed, so you can spot cases where one model family has a blind spot.

This matters because models have known self-preference bias. If Claude generates the content and Claude judges it, there's a systematic tilt toward approval. Adding GPT and Gemini as independent jury members catches things Claude might be blind to.

---

## Synthetic Dataset

### Why Synthetic

We don't have real client data yet. When Rekan launches and Elenice starts onboarding real businesses, we'll build a platinum-tier dataset from actual client profiles, real feedback, and content that was posted versus edited versus rejected. Until then, we need realistic test cases.

### Conditional Prompting for Diversity

The naive approach to generating test profiles is to ask an LLM for "20 different Brazilian businesses." The problem is that even at high temperature, models produce repetitive outputs. You get five salons, four restaurants, and everything is set in São Paulo.

The research-backed fix is conditional prompting: you define attribute dimensions (business type, city, vibe, target audience) and systematically vary combinations. We have 15 business types, 12 cities, 10 vibes, and 8 audience segments, giving 14,400 possible combinations. We sample strategically, ensuring every attribute value appears at least twice before any repeats.

This produces profiles like a luxury nail designer in Manaus targeting young women, a family-friendly pizzeria in Curitiba with a casual vibe, or a motivational personal trainer in Salvador targeting men 18 to 40. Each feels distinct because the constraints force diversity.

### Cross-LLM Validation

When Claude generates profiles and Claude validates them, there's a risk of rubber-stamping. A profile might seem realistic to Claude because it follows patterns Claude itself would produce.

Cross-LLM validation addresses this by having a different model family check whether the profile is realistic. Claude generates, GPT or Gemini validates. If the validator flags a profile as unrealistic (the neighbourhood doesn't exist, the prices don't make sense for a MEI), it gets discarded.

### Dataset Maturity

The dataset matures through three tiers:

**Silver** is where we start. LLM-generated profiles validated by the same or different model. Good enough for prompt iteration.

**Gold** adds multi-LLM consensus. Both the generating model and a different validator agree the profile is realistic and the reference output is high quality. This is the standard for benchmarking.

**Platinum** comes after launch. Real client profiles from Elenice's network, actual feedback on what worked and what didn't, content that was posted as-is versus edited versus rejected. This is the ground truth that everything else calibrates against.

---

## Prompt Optimisation

### The Loop

Optimisation follows a simple, disciplined cycle:

1. Run the evaluation with judges enabled and look at which judges are failing most across all test businesses
2. Pick the single weakest criterion and run a verbose evaluation on a failing business to read the actual output and the judge's reasoning for why it failed
3. Form one specific hypothesis about what to change in the system prompt, like "adding example sentences with informal tone will fix the naturalidade failures"
4. Make that one edit to the system prompt
5. Re-evaluate and compare: did the target judge improve? Did anything else regress?
6. If it helped, keep it and move to the next weakest judge. If it didn't help or caused regression, revert and try a different approach

### Why One Change Per Cycle

When you change three things at once and the score goes up, you don't know which change helped. When it goes down, you don't know which one broke it. One hypothesis, one edit, one measurement. It's slower per cycle but faster to converge because you never have to untangle confounded variables.

### Why Max Five Cycles Per Session

Diminishing returns kick in fast. The first couple of changes usually address obvious failures. After five cycles, you're either in a good place or you need to step back and rethink the approach rather than micro-optimising.

### The Optimizer Agent

Instead of a human running this loop manually, Claude Code acts as the optimizer agent. It reads the evaluation results, identifies patterns in judge failures, edits the system prompt, and re-evaluates. Same loop, same discipline, just automated. The skill file gives Claude Code the exact rules: what to edit, what not to touch, when to stop.

---

## Scoring

The final score blends both layers:

Heuristic checks contribute 40% of the score, covering structural completeness that's objectively measurable. LLM judges (or jury) contribute 60%, covering subjective quality that only a language model can assess.

A business passes if it scores above 70%. The target is 80% average across all test businesses. When using the jury, the score is based on majority vote across model families rather than any single model's opinion.

The judge breakdown shows exactly which criteria are failing for which businesses, making it actionable. You don't just know the score is low; you know it's low because the açaí shop in Manaus keeps getting generic content that doesn't mention the neighbourhood.

---

## When to Use Each Layer

**Heuristic only** for rapid iteration when you're making structural changes to the prompt and want instant feedback at zero cost.

**Single-model judges** for daily prompt development. This is the workhorse mode. Cheap enough to run after every change, reliable enough to guide decisions.

**Multi-LLM jury** before deploying a prompt change. This is the quality gate. If the jury disagrees with the single judge, you investigate before shipping.

**Cross-LLM dataset validation** when expanding the test set. Ensures new synthetic profiles are realistic from multiple perspectives.

---

## Cost

The entire monthly budget for evaluation and optimisation sits around $10 to $15 AUD. A single heuristic run is free. A judge run costs roughly 10 cents. A full jury run costs 30 to 50 cents. An optimisation session of five cycles runs about $1.50. This is intentionally cheap so there's never a reason to skip evaluation.
