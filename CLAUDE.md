# Rekan

AI-powered Instagram content generator for Brazilian micro-entrepreneurs. Generates captions, hashtags, and stories so they can stay consistent on social media without hiring a manager.

Brazil-only product. Everything user-facing is in Brazilian Portuguese (pt-BR): generated content, UI copy, error messages, validation messages. No internationalization, no other locales.

## Stack

- **Backend**: Go + PocketBase in `api/`
- **Frontend**: SvelteKit 2 + Svelte 5 + TypeScript in `web/`
- **Prompts**: BAML (BoundaryML) for prompt definitions and structured LLM output
- **Dev environment**: Nix flake + direnv

### Nix Dev Environment

The project uses a Nix flake (`flake.nix`) for reproducible tooling. Playwright browsers are provided by Nix via `playwright-driver.browsers`, NOT downloaded by npm. When updating `@playwright/test` in `web/package.json`, the npm version **must match** the Playwright version in `flake.lock`'s nixpkgs pin. If they drift apart, browser binaries won't be found and e2e tests will fail. To align them: update `flake.lock` with `nix flake lock --update-input nixpkgs`, check the new Playwright version with `nix eval`, then pin the same version in `package.json`.

## Commands

```bash
make dev                       # starts both backend and frontend
cd api && go run .             # PocketBase only (:8090)
cd web && pnpm dev             # SvelteKit only
cd web && pnpm check           # typecheck
cd web && pnpm build           # production build
make eval                      # heuristics only (~5s, needs OPENROUTER_API_KEY)
make eval-judges               # heuristics + LLM judges (~25s)
make test-judges               # integration tests for judge verdicts
```

## Project structure

```
api/           Go backend (PocketBase)
web/           SvelteKit frontend
eval/          Eval pipeline (heuristics, LLM judges, content generation)
flake.nix      Dev shell (Go, Node, pnpm)
```

## Prompt optimization loop

See `eval/CLAUDE.md` for full eval pipeline docs. The short version:

1. `make eval-judges`, identify the weakest criterion
2. Pick a failing business, run verbose to read judge reasoning
3. Edit `eval/baml_src/content.baml`
4. `make eval-judges`, diff the two runs
5. Keep or revert. Max 5 cycles per session.

## Browser inspection

Use Playwright to inspect the running frontend:

```bash
cd web && npx playwright screenshot http://localhost:5173 /tmp/screenshot.png
```

Then read the screenshot with the Read tool to verify visual output.

## Conventions

- Early returns, simple and minimal code
- No unnecessary abstractions or over-engineering
- Comments only for workarounds, magic values, surprising defaults
- Match existing patterns before inventing new ones
