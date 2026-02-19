# Rekan

AI-powered Instagram content generator for Brazilian micro-entrepreneurs. Generates captions, hashtags, and stories so they can stay consistent on social media without hiring a manager.

Brazil-only product. Everything user-facing is in Brazilian Portuguese (pt-BR): generated content, UI copy, error messages, validation messages. No internationalization, no other locales.

## Stack

- **Backend**: Go + PocketBase in `api/`
- **Frontend**: SvelteKit 2 + Svelte 5 + TypeScript in `web/`
- **Prompts**: BAML (BoundaryML) for prompt definitions and structured LLM output
- **Dev environment**: Nix flake + direnv

## Commands

```bash
make dev                       # starts both backend and frontend
cd api && go run .             # PocketBase only (:8090)
cd web && pnpm dev             # SvelteKit only
cd web && pnpm check           # typecheck
cd web && pnpm build           # production build
```

## Project structure

```
api/           Go backend (PocketBase)
web/           SvelteKit frontend
flake.nix      Dev shell (Go, Node, pnpm)
```

## Conventions

- Early returns, simple and minimal code
- No unnecessary abstractions or over-engineering
- Comments only for workarounds, magic values, surprising defaults
- Match existing patterns before inventing new ones
