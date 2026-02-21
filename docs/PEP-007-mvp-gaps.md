# PEP-007: MVP Gaps

**Status:** Pending
**Date:** 2026-02-21

## Context

Post-commit review of the full MVP surface (PEP-001 through PEP-006) identified gaps in test coverage, incomplete feature integration, and missing verification steps. This PEP tracks all of them so nothing falls through the cracks before launch.

## Gap 1: Handler test coverage

**Relates to:** PEP-005 (Waves 3, 5), PEP-006

Three handlers have zero tests. These are the core business logic (billing, generation, operator tool). A subscription bug or generation failure would be invisible until a user hits it.

| Handler | File | What to test |
|---------|------|-------------|
| `GeneratePosts` | `handlers/generate.go` | Subscription check (trial ok, trial exhausted, active, cancelled), hook loading, trial increment, ownership verification, LLM error (502) |
| `CreateSubscription` | `handlers/subscribe.go` | Customer creation, subscription creation, Asaas error handling, duplicate subscription |
| `GetSubscription` | `handlers/subscribe.go` | Returns current status, no subscription case |
| `OperatorGenerate` | `handlers/operator.go` | Auth, ownership, empty message, generation error, response format |

Existing `webhooks_test.go` (5 tests) provides the pattern: PocketBase test app, test records, mock deps.

### Acceptance criteria

- [ ] `generate_test.go` covers happy path and subscription rejection
- [ ] `subscribe_test.go` covers customer+subscription creation and Asaas errors
- [ ] `operator_test.go` covers auth, ownership, and message validation

## Gap 2: Content rotation not wired into API (PEP-003)

**Relates to:** PEP-003

Content rotation (role selection + hook exclusion) works in the eval pipeline but is only partially integrated into the production API.

What works:
- BAML prompt accepts dynamic roles and `previousHooks`
- `eval.PickRoles()` and `eval.ExtractHooks()` exist
- `handlers/generate.go` calls `PickRoles(3, nil)` and loads previous hooks from `posts` collection

What's missing:
- Frontend has no visibility into which roles were selected or used
- No way to influence role selection from the UI
- Operator tool (`handlers/operator.go`) passes `nil` for `previousHooks`, so repeated generations for the same client repeat angles

### Acceptance criteria

- [ ] Operator handler loads previous hooks from `posts` collection when post storage is added
- [ ] Generated post response includes `role` field so the frontend can display it
- [ ] Evaluate whether manual role selection adds value or if random is sufficient

## Gap 3: Component library extraction (PEP-004)

**Relates to:** PEP-004 (Waves 2, 3)

Wave 1 (Tailwind v4 + shadcn-svelte base components) is done. Remaining waves:

**Wave 2 (component extraction):**
- Extract brand components: SectionLabel, PhoneFrame, IgPost, Container
- Install Histoire for component stories
- Each component gets a `.story.svelte` file

**Wave 3 (marketing page migration):**
- Rewrite scoped `<style>` blocks to Tailwind utilities
- Move animations to `app.css`
- Remove all scoped CSS from marketing page

### Acceptance criteria

- [ ] Brand components extracted to `web/src/lib/components/`
- [ ] Marketing page uses Tailwind utilities, no scoped CSS
- [ ] Histoire running with stories for each component

## Gap 4: Frontend testing

**Relates to:** PEP-005 (Wave 4), PEP-006

Zero test files in `web/src/`. The operador page, onboarding flow, and dashboard have no automated checks.

### Acceptance criteria

- [ ] Decide testing strategy: Vitest component tests, Playwright e2e, or both
- [ ] At minimum, Playwright smoke tests for: login redirect, onboarding flow, generation, operator page

## Gap 5: Manual verification checklist (PEP-005)

**Relates to:** PEP-005 (Wave 1)

Two acceptance criteria from PEP-005 Wave 1 are still unchecked:

- [ ] Google sign-in works end-to-end from SvelteKit
- [ ] Manual threat model verification: attempt each attack row from the threat model table and confirm rejection

These should be done before any real user touches the app.

## Priority

1. **Gap 1** (handler tests) — highest risk, core business logic unverified
2. **Gap 5** (manual verification) — quick to do, catches config issues
3. **Gap 4** (frontend testing) — decide strategy, add smoke tests
4. **Gap 2** (content rotation) — functional but incomplete integration
5. **Gap 3** (component library) — cosmetic, no user-facing risk
