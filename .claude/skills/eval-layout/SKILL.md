---
name: eval-layout
description: Evaluate the operator page layout by capturing screenshots of every view and scoring them against spatial/layout criteria. Use when the user says "evaluate the layout", "eval layout", "screenshot the app", "how does the app look", "check the UI", "evaluate the pages", "evaluate the app", "rate the screens", or wants to assess the current state of the operator page visually.
---

# Layout Evaluator

Capture screenshots of every operator page view and evaluate each one against spatial/layout criteria tuned for Rekan's audience (Brazilian micro-entrepreneurs, 50+, low digital literacy, Moto G devices).

This is a rigid skill. Follow every step exactly.

## Step 1: Capture screenshots

Run the Playwright screenshot test to capture all views at Moto G viewport (360x740):

```bash
cd /home/denis/workspace/rekan/web && npx playwright test screenshot-all.spec.ts --project=default
```

This produces screenshots in `web/test-results/screenshots/`:
- `01-client-list.png` — client list with morning summary
- `02-chat-empty.png` — chat thread (empty state)
- `03-generate-mode.png` — post generation mode
- `04-info-screen.png` — client info/profile screen
- `05-new-client.png` — new client form (mic-first)

If the test fails, check that the dev server is running (`curl -sk https://localhost:5173`). If not, start it with `make dev` from the project root.

## Step 2: Read each screenshot

Use the Read tool to view each screenshot file. Claude has vision and can analyze the layout directly.

## Step 3: Evaluate each view

For each screenshot, evaluate against these five criteria. Score each 1-5.

### Criteria

**SPATIAL BALANCE** — Does the screen use its 360x740 viewport well? Content distributed without large dead zones or cramped clusters? Elements breathe without wasting space?
- 5 = every region has purpose, no wasted space
- 3 = some dead zones or mild crowding
- 1 = huge empty areas or everything piled in one spot

**VISUAL HIERARCHY** — Can you instantly tell what's most important? Clear reading order? Size/weight/color differences guide the eye, or everything competes equally?
- 5 = immediate focal point, clear primary > secondary > tertiary
- 3 = some structure but multiple elements compete
- 1 = flat wall of equal-weight elements

**TOUCH FRIENDLINESS** — Interactive elements big enough and spaced enough for a 50+ year old? Buttons clearly tappable? Enough gap between targets to avoid mis-taps?
- 5 = generous, clearly distinct touch targets with comfortable spacing
- 3 = mostly fine but some tight spots
- 1 = tiny or crowded targets, easy to mis-tap

**INFORMATION DENSITY** — For this specific view, is the amount of information appropriate? Not too sparse (wasting a trip), not too dense (overwhelming for low-literacy users)?
- 5 = right amount for the task
- 3 = slightly off, either sparse or busy
- 1 = barren or overwhelming

**COMPOSITION** — Does the layout feel intentional and cohesive? Alignments consistent? Margins/paddings follow a rhythm? Or elements placed ad hoc?
- 5 = clean grid, consistent spacing, feels designed
- 3 = mostly aligned but some inconsistencies
- 1 = randomly placed, inconsistent gaps

### Output format per view

```
## [View Name]
Screenshot: web/test-results/screenshots/XX-name.png

| Criterion          | Score | Reason                          |
|--------------------|-------|---------------------------------|
| Spatial balance    |   X   | one sentence                    |
| Visual hierarchy   |   X   | one sentence                    |
| Touch friendliness |   X   | one sentence                    |
| Information density|   X   | one sentence                    |
| Composition        |   X   | one sentence                    |

Top issue: [the single most impactful thing to fix]
```

## Step 4: Summary

After evaluating all views:

1. Show a summary table with per-criterion averages across all views
2. Rank the views from weakest to strongest overall
3. List the top 3 layout problems across the whole app, with specific observations (which view, which elements, what's wrong spatially)
4. For each of the top 3 problems, suggest a concrete fix (what to change in the layout, not code)

## Step 5: Save the run

Write the evaluation results to a timestamped JSON file:

```bash
mkdir -p /home/denis/workspace/rekan/web/test-results/layout-runs
```

Save to `web/test-results/layout-runs/YYYY-MM-DD-HH-MM.json` with this structure:

```json
{
  "timestamp": "ISO string",
  "views": {
    "client-list": {
      "spatial_balance": { "score": N, "reason": "..." },
      "visual_hierarchy": { "score": N, "reason": "..." },
      "touch_friendliness": { "score": N, "reason": "..." },
      "information_density": { "score": N, "reason": "..." },
      "composition": { "score": N, "reason": "..." },
      "top_issue": "..."
    }
  },
  "averages": {
    "spatial_balance": N,
    "visual_hierarchy": N,
    "touch_friendliness": N,
    "information_density": N,
    "composition": N,
    "overall": N
  },
  "top_problems": ["...", "...", "..."]
}
```

## Step 6: Compare (if previous run exists)

Check if there are previous runs in `web/test-results/layout-runs/`. If so, read the most recent one and compare:

- Show which criteria improved, regressed, or stayed the same
- Note if the top problems from last run were addressed
- Call out any new regressions

## Notes

- The screenshot test file is at `web/tests/screenshot-all.spec.ts`. If new views are added to the operator page, update that file to capture them.
- `test-results/` is gitignored, so runs are local only.
- The operator page source is at `web/src/routes/(app)/operador/+page.svelte`.
- When scoring, be honest and critical. A 3 is "fine but could be better". Reserve 4-5 for genuinely good layouts. The point is to surface real issues, not to flatter.
