# PEP-006: WhatsApp Operator Tool

**Status:** Done
**Date:** 2026-02-21

## Context

Rekan is pivoting from a self-serve web app to a WhatsApp-based content service. In Phase 1, Elenice (the operator) receives WhatsApp messages from clients, pastes them into a web tool, gets AI-generated Instagram content back, and sends it to the client manually.

The AI pipeline already exists (PEP-001). We need a new input layer: message-based generation instead of profile-based generation.

## Decisions

- **Reuse `businesses` collection.** No separate `whatsapp_clients` collection. The operator creates client businesses under her account using the same schema. Dropped the one-business-per-user unique constraint (migration 1740000006).
- **Single post, not batch.** `GenerateFromMessage` returns one `Post`, not three. The client's WhatsApp message determines the content angle, so roles/variety don't apply.
- **No subscription checks.** The operator tool is internal, not self-serve. No trial limits or payment gates on the new endpoint.
- **No post storage.** Generated posts are returned directly, not saved to the `posts` collection. The operator copies the content to WhatsApp.
- **Message-driven prompt.** The BAML prompt extracts concrete details (names, prices, dates) from the client's WhatsApp message and centers the post on them.
- **Eval support.** The existing eval pipeline works with `--message` flag. Runs 4 of 5 judges (skips `variedade`, which compares across posts).

## What was built

### Backend

1. **BAML function `GenerateFromMessage`** (`eval/baml_src/content.baml`) — Takes `clientMessage` string + `BusinessProfile`, returns single `Post`. Prompt prioritizes concrete details from the message.

2. **Go wrapper** (`eval/generate.go`) — `GenerateFromMessage(ctx, profile, message, previousHooks)` returns `(Post, error)`.

3. **Handler** (`api/internal/http/handlers/operator.go`) — Loads business, checks ownership, calls `GenerateFromMessage`, returns `{caption, hashtags, production_note}`.

4. **Route** — `POST /api/businesses/{id}/posts:generateFromMessage` (same `:verb` pattern as existing `:generate`).

5. **Migration** (`api/migrations/1740000006_drop_businesses_user_unique.go`) — Removes unique index on `businesses.user` so the operator can create multiple client businesses.

### Eval

6. **`--message` flag** (`eval/cmd/eval/main.go`) — Generates a single post from a WhatsApp message against a test profile. Automatically excludes `variedade` judge. Example:
   ```bash
   cd eval && go run ./cmd/eval --judges --verbose \
     --message "Quero um post divulgando meu kit festa de 50 doces por R\$250" \
     --profile "Doces da Flavinha"
   ```

7. **Dynamic judge columns** — `JudgeNames` exported from `eval/judge.go`. Table output adapts to whichever judges are active.

### Frontend

8. **Operator page** (`web/src/routes/(app)/operador/+page.svelte`) — Two-panel layout. Left: client list with add/edit. Right: message textarea, generate button, result card with copy buttons for caption and hashtags.

## What's NOT in scope

- No `previousHooks` loading (no post storage for message-based generation)
- No WhatsApp API integration (Elenice copies/pastes manually)
- No multi-operator support (single authenticated user)

## Verification

- `baml-cli generate` succeeds
- `go build .` compiles (api and eval)
- `pnpm check` passes with 0 errors
- Eval run with `--message --judges` produces 6/6 heuristics, 3/4 judges passing
