# Rekan

AI-powered Instagram content generator for Brazilian micro-entrepreneurs. Everything user-facing is in pt-BR. No other locales.

## Commands

```bash
make dev                       # starts both backend and frontend
cd api && go run .             # PocketBase only (:8090)
cd web && pnpm dev             # SvelteKit only
cd web && pnpm check           # typecheck
cd web && pnpm build           # production build
make eval                      # heuristics only (~5s, needs CLAUDE_API_KEY + GEMINI_API_KEY)
make eval-judges               # heuristics + LLM judges (~25s)
make test-judges               # integration tests for judge verdicts
```

Playwright browsers come from Nix, not npm. If updating `@playwright/test`, its version must match the Playwright version pinned in `flake.lock`.

## Prompt optimization loop

See `eval/CLAUDE.md` for full eval pipeline docs. The short version:

1. `make eval-judges`, identify the weakest criterion
2. Pick a failing business, run verbose to read judge reasoning
3. Edit `eval/baml_src/content.baml`
4. `make eval-judges`, diff the two runs
5. Keep or revert. Max 5 cycles per session.

## Browser inspection

```bash
cd web && npx playwright screenshot http://localhost:5173 /tmp/screenshot.png
```

Then read the screenshot with the Read tool to verify visual output.
