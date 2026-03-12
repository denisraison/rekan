# Rekan

Everything user-facing is in pt-BR. No other locales.

## Commands

```bash
make dev                       # starts both backend and frontend
cd api && go run ./cmd/rekan   # PocketBase only (:8090)
cd web && pnpm dev             # SvelteKit only
cd web && pnpm check           # typecheck
cd web && pnpm build           # production build
make eval                      # heuristics only (~5s, needs CLAUDE_API_KEY + GEMINI_API_KEY)
make eval-judges               # heuristics + LLM judges (~25s)
make test-judges               # integration tests for judge verdicts
```

Playwright browsers come from Nix, not npm. If updating `@playwright/test`, its version must match the Playwright version pinned in `flake.lock`.

## E2E tests (Playwright)

- Auth is handled by `tests/auth.setup.ts` via `storageState`. Tests never log in themselves.
- Use helpers from `tests/helpers.ts` (`loginAsOperador`, `selectFirstClient`, `switchToGenerateMode`).
- Never use `waitForTimeout`. Wait for a visible element instead (`locator.waitFor()`, `expect().toBeVisible()`).
- Never use `waitForLoadState('networkidle')` on the operator page. SSE streams keep connections open.
- Config sets `ignoreHTTPSErrors` and `baseURL` globally. Tests should not override these.

## Prompt optimization

See `api/internal/content/CLAUDE.md`.

## Test credentials

`make seed` (via `scripts/seed.sh`) resets the DB and creates:

| Role     | Email                | Password       |
|----------|----------------------|----------------|
| Admin    | admin@rekan.local    | admin1234567   |
| Operador | operador@rekan.local | senha1234567   |

Overridable via `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `SEED_USER_EMAIL`, `SEED_USER_PASSWORD` env vars.

## Browser inspection

```bash
cd web && npx playwright screenshot http://localhost:5173 /tmp/screenshot.png
```

Then read the screenshot with the Read tool to verify visual output.
