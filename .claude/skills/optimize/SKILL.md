---
name: prompt-optimizer
description: This skill should be used when the user asks to "optimize prompts", "run optimization loop", "improve content quality", "optimize eval", "optimize judges", "prompt optimization", or wants to iteratively improve the generation or judge prompts using the eval pipeline.
---

# Prompt Optimizer

Automate the eval prompt optimization loop using research-backed strategies (OPRO trajectory feedback, EvoPrompt multi-candidate generation). Run eval, identify the weakest criterion, generate multiple candidate edits, eval all, keep the best or revert. Max 5 cycles.

This is a rigid skill. Follow every step exactly.

## Determine target

If the user says "optimize judges" or mentions judge calibration, the target file is `eval/baml_src/judges.baml`. Otherwise the target is `eval/baml_src/content.baml`.

## Pre-loop setup

1. Read the target BAML file. Remember its full contents as the original baseline.
2. Run `make eval-fast` from the project root. This uses a single judge and 4 profiles for faster iteration.
3. Find the most recent `.json` file in `eval/runs/` (filenames sort chronologically). Read it.
4. Note the baseline judge totals from `summary.judgeTotals`. Each value is out of 4 (one per business profile in fast mode).
5. Initialize an empty **attempt history** list. This persists across all cycles and tracks what was tried.

## Optimization cycle

Repeat up to 5 times.

### Step 1: Identify the weakest criterion

From `summary.judgeTotals`, find the criterion with the lowest count. The five criteria are: `naturalidade`, `especificidade`, `acionavel`, `variedade`, `engajamento`.

If tied, use this priority (hardest to fix first): variedade > engajamento > naturalidade > especificidade > acionavel.

### Step 2: Pick a failing profile

Scan `results[]` in the run JSON. Find a business where the weakest criterion's judge verdict is `false`. If multiple businesses failed, prefer one that also has failing heuristic checks (look at `checks[]` for entries where `pass` is `false`).

### Step 3: Read the judge reasoning

From the same run JSON entry, read the `reasoning` field for the failing judge. This explains why the content failed.

### Step 4: Generate candidate edits

Before editing, remember the current file contents as the cycle baseline (distinct from the original baseline, since previous cycles may have made improvements that were kept).

Using the judge reasoning AND the attempt history, generate **3 distinct candidate edits** to the target BAML file. Each candidate should:
- Address the same failing criterion from a different angle
- Be a focused, single-concept change (not a batch of changes)
- Not repeat any hypothesis already in the attempt history

Present the 3 candidates briefly (one line each describing the hypothesis), then create 3 copies of the BAML file: write `candidate_1.baml`, `candidate_2.baml`, `candidate_3.baml` into `/tmp/optimizer/`. Each applies one candidate edit to the cycle baseline.

### Step 5: Eval all candidates

For each candidate:
1. Copy it over the target BAML file.
2. Run eval:
   - If optimizing `content.baml`: run `make eval-fast`.
   - If optimizing `judges.baml`: run `cd eval && go run ./cmd/eval --judges --from-run runs/BASELINE.json` where `BASELINE.json` is the run file from step 3.
3. Read the resulting run JSON and record its `summary.judgeTotals`.

After all 3 candidates are evaluated, restore the cycle baseline to the target file.

### Step 6: Select the best candidate

Compare all 3 candidates against the cycle baseline:
- A candidate is **valid** if the target criterion improved (higher count) and no other criterion regressed (lower count).
- Among valid candidates, pick the one with the highest total score across all criteria.
- If no candidate is valid, all 3 failed this cycle.

### Step 7: Decide and update history

**If a valid candidate exists:** Apply the winning edit to the target file. This run becomes the new baseline for the next cycle. Add all 3 hypotheses to the attempt history with their outcomes (winner marked).

**If no valid candidate:** Keep the cycle baseline unchanged. Add all 3 hypotheses to the attempt history marked as failed.

Report the cycle result: cycle number, target criterion, the 3 hypotheses (mark winner or note all failed), judge totals before and after.

### Early stop

Stop the loop if EITHER condition is met:
- **3 consecutive failed cycles** (no valid candidate found in 3 back-to-back cycles). The search space near the current prompt is likely exhausted.
- **Score plateau**: the best score across the last 2 cycles (including all candidates evaluated) equals the current baseline. No signal of possible improvement remains.

## Cleanup

Delete `/tmp/optimizer/` and its contents.

## After the loop

Print a summary table of all cycles:

```
Cycle  Target         Candidates (winner*)                          Result    Delta
1      variedade      *Format mixing | Question hooks | Emoji ban   Kept      VAR 2->3
2      engajamento    *Story opener | Direct CTA | Slang rule       Kept      ENG 1->3
3      naturalidade   Filler words | Contractions | Regional slang  Failed    -
```

State the final judge totals and how they compare to the original baseline from pre-loop setup.

Leave the prompt file in its final improved state (all kept edits applied, all reverted edits undone).
