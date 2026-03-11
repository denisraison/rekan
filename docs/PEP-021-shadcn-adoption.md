# PEP-021: Adopt shadcn Components Across the App

**Status:** In Progress
**Date:** 2026-03-11

## Context

PEP-004 installed shadcn-svelte (Button, Badge, Card, Separator) and PEP-020 extracted the operator page into 11 components. But none of the components actually use the shadcn primitives. The app has ~60 raw `<button>` elements, ~15 `<input>`, ~8 `<textarea>`, and ~3 `<select>` scattered across operator components, login, experimentation, invite, and marketing pages. Each repeats its own Tailwind classes for padding, rounding, disabled states, and colors.

The shadcn Button component handles focus rings, disabled states, aria attributes, and polymorphic rendering (button or anchor). Using it (and adding Input, Textarea, Select, Label) eliminates repeated styling and gives consistent accessible behavior everywhere.

The main obstacle: the app uses pill-shaped buttons (`rounded-full`, variable heights) while shadcn defaults to `rounded-md h-9`. The fix is adding custom variants to the Button component rather than overriding on every usage.

## Button Variant Mapping

Current button patterns in the codebase map to these variants:

| Pattern | Occurrences | Proposed variant |
|---------|-------------|-----------------|
| `bg-coral text-white rounded-full` | ~20 | `default` (modify base to `rounded-full`) |
| `bg-[#25D366] text-white rounded-full` | ~10 | `whatsapp` (new) |
| `bg-coral-pale text-coral rounded-full` | ~5 | `soft` (new) |
| `bg-sage-pale text-sage-dark rounded-full` | ~4 | `secondary` (remap) |
| `border border-border rounded-full` | ~3 | `outline` (existing) |
| `text-coral` (no bg, icon-like) | ~5 | `ghost` (existing) |
| `bg-red-500 text-white` | ~2 | `destructive` (existing) |

Size variants:

| Pattern | Proposed size |
|---------|--------------|
| `px-3 py-1.5 text-sm` (chips, filters) | `sm` |
| `px-5 py-3 text-base min-h-12` (primary actions) | `default` (adjust) |
| `px-6 py-3 text-base min-h-13` (full-width CTAs) | `lg` |
| `w-9 h-9` / `w-8 h-8` (icon buttons) | `icon` / `icon-sm` (existing) |

## Waves

### Wave 1: Extend Button variants, install Input/Textarea/Label (done)

Add `whatsapp` and `soft` variants to `button.svelte`. Change base border-radius to `rounded-full`. Adjust default size to match the app's `min-h-12 px-5 py-3` pattern. Install shadcn Input, Textarea, and Label components via the CLI and customize to match the app's styling (rounded-xl, min-h-13, brand border colors).

**Files:**
- `web/src/lib/components/ui/button/button.svelte` (add variants, change base radius)
- `web/src/lib/components/ui/input/` (new, via shadcn CLI)
- `web/src/lib/components/ui/textarea/` (new, via shadcn CLI)
- `web/src/lib/components/ui/label/` (new, via shadcn CLI)
- `web/src/lib/utils.ts` (added `WithoutChildren` type required by Textarea component)

**Notes:**
- Removed dark mode variants from all components (app has no dark mode).
- Removed `shadow-xs` from Button variants to match the app's flat button style.
- `secondary` variant remapped from shadcn default (neutral) to sage-pale/sage-dark to match the app's secondary action style.
- Input and Textarea both use `min-h-13` (52px) matching the existing form field heights in the app.
- Textarea includes `resize-none` by default since the app uses `field-sizing-content` for auto-grow.
- `disabled:opacity-50` changed to `disabled:opacity-60` on Button to match what the operator components already used.

**Gate:**
- [x] `cd web && pnpm check` passes
- [x] `cd web && pnpm build` succeeds
- [ ] Button component renders all variants correctly (manual check in dev)

### Wave 2: Operator components

Replace raw `<button>` elements in all 11 operator components with `<Button>`. Replace raw `<input>` and `<textarea>` in NewClientForm with the new Input/Textarea components. Replace raw `<input>` in InputBar with Input.

**Components to update (estimated button count):**
- InputBar.svelte (~12 buttons, 1 input)
- NewClientForm.svelte (~10 buttons, 4 inputs, 3 textareas)
- InfoScreen.svelte (~10 buttons, 1 textarea)
- IdeaPicker.svelte (~4 buttons)
- PostReviewOverlay.svelte (~4 buttons, 1 textarea)
- ApprovalPanel.svelte (~3 buttons)
- ChatHeader.svelte (~2 buttons)
- ClientList.svelte (~3 buttons)
- ClientCard.svelte (1 button)
- `operador/+page.svelte` (1 button, 1 link)

**Gate:**
- [ ] `cd web && pnpm check` passes
- [ ] `cd web && npx playwright test --project=default` passes
- [ ] `grep -c '<button' web/src/lib/components/operator/*.svelte` returns 0
- [ ] `npx playwright test screenshot-all.spec.ts --project=default` passes (no visual regression)

### Wave 3: Login, experimentation, invite pages

Replace raw form elements in:
- `entrar/+page.svelte` (2 inputs, 1 button)
- `experimentar/+page.svelte` (4 inputs, 1 select, 1 textarea, 3 buttons)
- `convite/[token]/+page.svelte` (2 inputs, 1 checkbox, 3 buttons)

**Gate:**
- [ ] `cd web && pnpm check` passes
- [ ] `cd web && pnpm build` succeeds
- [ ] Each page renders correctly (Playwright screenshot)

### Wave 4: Marketing page

Replace raw `<button>` and styled `<a>` elements in the marketing homepage. This page has its own button patterns (CTA buttons with hover lifts), so some may stay as custom elements. Only adopt shadcn where it fits naturally.

**Gate:**
- [ ] `cd web && pnpm check` passes
- [ ] Marketing page screenshot matches (no visual regression)

## New Dependencies

None. shadcn components are copied into the project via CLI. bits-ui (already installed) provides the headless primitives.

## Consequences

- Consistent button/input/textarea behavior across the entire app: focus rings, disabled states, aria attributes
- New button patterns only need a variant added to one file instead of repeating classes everywhere
- Slightly more verbose imports (`import { Button } from '$lib/components/ui/button'`) but much less repeated Tailwind
- Marketing page may keep some custom elements where shadcn variants don't fit the design
- The operator components become shorter as repeated class strings collapse into variant props
