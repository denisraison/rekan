# Optimizer Skill Design

**Date:** 2026-02-19

## Goal

A Claude Code skill that automates the prompt optimization loop for Rekan's eval pipeline. Instead of manually following the 6-step process documented in `eval/CLAUDE.md`, invoking this skill runs the full loop autonomously.

## Decisions

- **Project-local plugin** in `.claude/plugins/rekan-eval/`, committed to the repo
- **Single rigid skill** (no helper scripts, no subagents)
- **Full automation** with no pauses for user approval during the loop
- **Both prompts** in scope: `content.baml` (generation) and `judges.baml` (judge calibration)

## Plugin structure

```
.claude/plugins/rekan-eval/
  .claude-plugin/
    plugin.json
  skills/
    optimize/
      SKILL.md
```

Minimal manifest with auto-discovery. No hooks, agents, commands, or MCP servers.

## Skill behavior

### Trigger

User says "optimize prompts", "run optimization loop", "improve content quality", "optimize eval", or similar.

### Pre-loop setup

1. Read the target prompt file (`eval/baml_src/content.baml` by default, or `judges.baml` if user specifies judge optimization). Save contents as revert baseline.
2. Run `make eval-judges` to get a baseline run.
3. Read the most recent run JSON from `eval/runs/` for structured results.

### Each cycle (max 5)

1. **Identify weakness:** From the run JSON, find the judge criterion with the lowest pass rate. Tiebreaker priority: variedade > engajamento > naturalidade > especificidade > acionavel.
2. **Pick a target:** Find a profile that failed on the weakest criterion. Prefer profiles that also fail heuristics.
3. **Analyze:** Read the judge reasoning for that profile from the run JSON.
4. **Hypothesize and edit:** Form one hypothesis. Make one focused edit to the prompt file.
5. **Re-run:** Run `make eval-judges` again.
6. **Diff:** Compare the two run JSONs. Check: did the target criterion improve? Did anything regress?
7. **Decide:** If improved with no regressions, keep. If regressed, revert to the cycle's baseline. Report the delta.

### Stopping conditions

- Max 5 cycles reached
- 2 consecutive cycles with no improvement (early stop)

### Judge optimization mode

When optimizing `judges.baml`, use `--from-run` with the baseline run to re-judge existing content instead of regenerating. This isolates judge prompt changes from generation variance.

### After the loop

Print a summary: what changed, what improved, what regressed across all cycles. Leave the prompt file in its final improved state.

### Rules

- One edit per cycle, never batch multiple changes
- Always diff before deciding to keep
- When reverting, restore the prompt file to the start-of-cycle state (not the original baseline, since previous cycles may have made improvements that were kept)

---

## Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a project-local Claude Code plugin with a single skill that automates the eval prompt optimization loop.

**Architecture:** Minimal skill-focused plugin. Three files total: plugin manifest, skill definition. Auto-discovery handles registration.

**Tech Stack:** Claude Code plugin system (plugin.json manifest, SKILL.md skill definition)

---

### Task 1: Create plugin manifest

**Files:**
- Create: `.claude/plugins/rekan-eval/.claude-plugin/plugin.json`

**Step 1: Create the plugin manifest**

```json
{
  "name": "rekan-eval",
  "version": "0.1.0",
  "description": "Eval pipeline prompt optimizer for Rekan content generation"
}
```

**Step 2: Verify structure**

Run: `ls -la .claude/plugins/rekan-eval/.claude-plugin/`
Expected: `plugin.json` exists

---

### Task 2: Write the optimizer SKILL.md

**Files:**
- Create: `.claude/plugins/rekan-eval/skills/optimize/SKILL.md`

The SKILL.md must follow these conventions:
- YAML frontmatter with `name` and `description` (third-person trigger phrases)
- Body in imperative/infinitive form (not second person)
- Rigid checklist structure (follow exactly, no adaptation)
- Under 3000 words

The skill body must encode:

**Frontmatter description triggers:** "optimize prompts", "run optimization loop", "improve content quality", "optimize eval", "optimize judges", "prompt optimization"

**Pre-loop setup section:**
1. Determine target: `eval/baml_src/content.baml` (default) or `eval/baml_src/judges.baml` (if user says "optimize judges")
2. Read the target file, store contents mentally as revert baseline
3. Run `make eval-judges` (working directory must be project root)
4. Find the most recent `.json` file in `eval/runs/` (by filename, they sort chronologically)
5. Read the run JSON to get structured baseline results

**Cycle section (max 5 iterations):**
1. Parse the run JSON `summary.judgeTotals` to find the criterion with lowest count out of 12. Tiebreaker order: variedade > engajamento > naturalidade > especificidade > acionavel
2. Scan `results[]` for a business where that judge's verdict is false. Prefer businesses that also have failed heuristic checks
3. Read that business's judge reasoning from the run JSON
4. Form one hypothesis about what prompt change would fix this. Make one focused edit to the BAML file
5. For content.baml: run `make eval-judges`. For judges.baml: run the eval CLI with `--from-run` pointing to the baseline run file
6. Read the new run JSON. Compare judge totals: target criterion should improve, nothing else should regress
7. If improved without regression: keep the edit, this run becomes the new baseline. If regressed: revert the BAML file to the start-of-cycle content
8. Report: cycle number, what was changed, which criterion was targeted, delta in judge totals

**Stopping conditions:**
- After 5 cycles, stop
- If 2 consecutive cycles produce no improvement, stop early

**Summary section:**
- After the loop ends, print a table of all cycles: cycle number, target criterion, hypothesis, result (kept/reverted), judge totals delta

**Step 2: Verify the skill file**

Run: `ls -la .claude/plugins/rekan-eval/skills/optimize/`
Expected: `SKILL.md` exists

---

### Task 3: Test plugin discovery

**Step 1: Verify full plugin structure**

Run: `find .claude/plugins/rekan-eval -type f`

Expected output:
```
.claude/plugins/rekan-eval/.claude-plugin/plugin.json
.claude/plugins/rekan-eval/skills/optimize/SKILL.md
```

**Step 2: Validate plugin.json is valid JSON**

Run: `python3 -c "import json; json.load(open('.claude/plugins/rekan-eval/.claude-plugin/plugin.json'))"`
Expected: No error

**Step 3: Validate SKILL.md has frontmatter**

Run: `head -5 .claude/plugins/rekan-eval/skills/optimize/SKILL.md`
Expected: Starts with `---`, contains `name:` and `description:`

**Step 4: Commit**

```bash
git add .claude/plugins/rekan-eval/
git commit -m "Add rekan-eval plugin with optimizer skill"
```
