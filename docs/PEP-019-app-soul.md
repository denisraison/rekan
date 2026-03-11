# PEP-019 — App Soul: Making Rekan Feel Warm and Intuitive

**Status:** In Progress
**Date:** 2026-03-11

## Context

Feedback from Elenice and early users: the app feels cold, confusing, and hard to use. It looks like a developer admin panel, not a product for Brazilian micro-entrepreneurs (50+, low digital literacy). Specific complaints: tiny text, no guidance on what to do, modes that look identical, empty screens that feel broken.

The operator page (`web/src/routes/(app)/operador/+page.svelte`) is 2800+ lines of inline styles mixing hardcoded px values and Tailwind classes. All icons are hand-rolled inline SVGs with inconsistent styles (some filled, some stroked, varying weights). There are no illustrations, no personality, no warmth.

Target audience uses WhatsApp daily. The app needs to feel closer to WhatsApp than to a SaaS dashboard.

## Principles

1. **One screen, one action.** Never make the user guess what to do next.
2. **Big and warm.** Minimum 13px text, 48px touch targets, warm tones.
3. **Sound like a friend.** Micro-copy in casual pt-BR ("pra" not "para", "bora" not "vamos"). Always correct grammar: accents, cedillas, tildes. Informal tone does not mean sloppy spelling.
4. **Steal from WhatsApp.** Familiar patterns reduce learning curve.

## Waves

### Wave 1 — Warm Palette, Micro-copy, and Font Size Floor

**Goal:** Make the app feel alive. Warmer colors, friendly text, no tiny unreadable text. This is one sweep because the color changes, copy changes, and size changes are all in the same two files and should be reviewed together visually.

**Files:** `web/src/app.css`, `web/src/routes/(app)/operador/+page.svelte`

**1a: Warmer colors (app.css)**

- [x] Shift `--text-muted` from `#999aaa` (cool grey) to `#9a9590` (warm grey)
- [x] Add `--chat-bg: #fef8f7` (faint coral wash)
- [x] Apply `--chat-bg` to the message thread container: the `div` at ~line 2422 with `class="flex-1 overflow-y-auto px-4 py-3"`. Change its inline `style` or add `style="background: var(--chat-bg)"`.

**1b: Micro-copy rewrite (operador/+page.svelte)**

Replace cold functional text with warm casual pt-BR. All strings must use correct Portuguese grammar (accents, cedillas, tildes). These are the exact strings to search and replace:

| Search string | Replace with | Approx. line |
|---|---|---|
| `Nenhum cliente cadastrado.` | `Você ainda não tem clientes. Toca no + pra começar!` | ~1615 |
| `Nenhuma mensagem ainda.` | `Quando {selected.name} mandar mensagem, aparece aqui.` | ~2425 |
| `Selecione um cliente para ver as mensagens.` | `Escolhe uma cliente na lista pra começar.` | ~2732 |
| `placeholder="Mensagem..."` (chat input) | `placeholder="Escreve aqui..."` | ~2678 |
| `placeholder="Descreva o post..."` (generate input) | `placeholder="Sobre o que é o post?"` | ~2678 |
| `Gerando...` (button text) | `Criando o post...` | ~2698 |
| `Analisando...` (voice state) | `Lendo o que você falou...` | ~2880 |
| `Nenhuma mensagem pendente.` | `Tudo em dia! Nenhuma mensagem pra aprovar.` | ~1499 |
| `toque para expandir` | `ver mais` | ~2117 |
| `Carregando...` | `Já vou...` | ~1481 |

**Placeholder changes break these test files** (they match on placeholder text):

- `tests/helpers.ts`: `selectFirstClient` waits for `input[placeholder="Mensagem..."]`, `switchToGenerateMode` waits for `input[placeholder="Descreva o post..."]`. Update both to the new placeholders.
- `tests/navigation.spec.ts`: 4 references to `input[placeholder="Mensagem..."]`.
- `tests/small-screen.spec.ts`: 1 reference to `input[placeholder="Mensagem..."]`.
- `tests/accessibility-text.spec.ts`: 6 references to `input[placeholder="Mensagem..."]`.
- `tests/post-review-overlay.spec.ts`: 2 references to `input[placeholder="Descreva o post..."]`.
- `tests/attach-button.spec.ts`: 1 reference to `input[placeholder="Mensagem..."]`.

**1c: Font size floor (operador/+page.svelte)**

No text below 13px in the operator page. Replace all inline `font-size: 11px` and `font-size: 12px` with `font-size: 13px`. Exceptions: badges with background colors (the unread count pill) can stay as they are since the background gives them visibility.

Specific targets (search for these in inline styles):
- Section headers "SERVICOS", "PERFIL", "Posts recentes", "Datas proximas", "Sugestoes de perfil": all `font-size: 11px` -> `13px`
- Status legend row (Estado: Ativo, 5-9d, +10d): `font-size: 12px` -> `13px`
- SSE connection status dots: `font-size: 12px` -> `13px`

**Gate:**

1. [x] `cd web && pnpm check`
2. [x] Update placeholder strings in `tests/helpers.ts`, `tests/navigation.spec.ts`, `tests/small-screen.spec.ts`, `tests/accessibility-text.spec.ts`, `tests/post-review-overlay.spec.ts`, `tests/attach-button.spec.ts`
3. [x] `cd web && npx playwright test` — all pass, zero failures
4. [x] `grep -c 'font-size: 11px\|font-size: 12px' web/src/routes/\(app\)/operador/+page.svelte` returns 1 (the badge pill, exempt)
5. [x] Screenshot gate — added `tests/ux-warmth.spec.ts` with Moto G viewport (360x740):
   - Test "chat screen has warm background": navigate to client detail, verify the message thread container has `background-color` matching `--chat-bg` via `page.evaluate`
   - Test "empty chat shows client name": navigate to client with no messages, verify text contains "mandar mensagem" (personalized, not generic)
   - Test "section headers are at least 13px": navigate to client info, measure a section header's `font-size` via `page.evaluate(el => getComputedStyle(el).fontSize)`

**Notes:**
- `--muted-foreground` in the shadcn semantic tokens section was also updated from `#999aaa` to `#9a9590` to stay consistent with `--text-muted`.
- The placeholders were on a single dynamic line (`inputMode === 'generate' ? ... : ...`), not separate elements.
- Both "Gerando..." instances (3-ideas spinner and main Gerar button) were changed to "Criando o post...".
- Some tests are flaky under 10 parallel workers (pre-existing `selectFirstClient` timeout), all pass on retry.

---

### Wave 2 — Mic-First New Client Form + Mode Toggle

**Goal:** Two structural changes. (1) When Elenice taps "+ Novo", she sees the mic button first, no scrolling. (2) Chat vs Generate mode looks obviously different.

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**2a: Reorder new client form**

Current flow when `voiceMode === 'idle'`: 6 text fields (Nome, Email, Negocio, Tipo, Cidade/Estado, Telefone) appear first, then the mic card below the fold. The voice path is the happy path but users don't find it.

New flow when `voiceMode === 'idle'`:
1. Title "Novo cliente"
2. Mic card (the existing coral card with "Gravar descrição")
3. "Preencher manualmente" link

The 6 basic fields (Nome do cliente through Telefone, ~lines 1805-1841) should be wrapped in `{#if voiceMode === 'manual' || voiceMode === 'done'}`. The mic-idle block (~lines 1843-1856) and the manual-link move above the fields. The `voiceMode === 'manual'` and `voiceMode === 'done'` branches stay as they are since they already show fields.

**2b: Make mode toggle unmissable**

Current state: the Post/Chat toggle is a small pill in the chips bar. Generate mode looks nearly identical to chat mode (2px border color change).

Changes:
- [ ] When `inputMode === 'generate'`, change the entire input bar container background from `var(--surface)` to `var(--coral-pale)`. This is the `div.shrink-0.border-t` at ~line 2490. Add `background: {inputMode === 'generate' ? 'var(--coral-pale)' : 'var(--surface)'}`.
- [ ] Add a one-line banner inside the input bar (before the chips row, ~line 2568) visible only in generate mode: `<p class="text-sm" style="color: var(--coral);">Toque nas mensagens que quer usar no post</p>`. No dismiss button needed, it disappears when switching back to chat.
- [ ] Mode toggle button: already fixed to `min-h-11` in the current codebase. Increase to `min-h-12 px-4 text-base` for this wave.

**Gate:**

1. [ ] `cd web && pnpm check`
2. [ ] `cd web && npx playwright test` — all pass
3. [ ] New tests in `tests/ux-warmth.spec.ts` (or same file from Wave 1):
   - Test "new client form shows mic first": tap "+ Novo", verify `button[aria-label="Gravar descrição"]` is visible, verify `input[placeholder*="Nome"]` is NOT visible (fields hidden until manual mode)
   - Test "manual mode shows fields": tap "+ Novo", tap "Preencher manualmente", verify `input[placeholder*="Nome"]` is visible
   - Test "generate mode has distinct background": enter generate mode, verify input bar container's computed `background-color` differs from chat mode
   - Test "generate mode shows instruction banner": enter generate mode, verify text "Toque nas mensagens" is visible
4. [ ] Screenshot gate in same test file:
   - Test "screenshot: new client idle": tap "+ Novo", take screenshot to `/tmp/pep019-newclient.png`, read it to verify mic card is above the fold (mic button's `boundingBox().y < 400`)

---

### Wave 3 — Icon Consistency + Final Polish

**Goal:** Unify all inline SVG icons to one consistent style. All operator page icons should use rounded stroke, 2px weight, round caps/joins. Filled variants only for active states (white checkmarks on colored backgrounds).

**Files:** `web/src/routes/(app)/operador/+page.svelte`

**Current state:** ~22 inline SVG instances. Mixed styles:
- `fill` icons: mic (white on coral), checkmarks (white on circles), warning triangle
- `stroke` icons: navigation chevrons, paperclip, chat bubble, sparkles, camera, gallery, close X, arrow up
- Stroke widths: 1.5, 2, 2.5 (inconsistent)
- No consistent `stroke-linecap` or `stroke-linejoin`

**Target style:**
- All stroke icons: `stroke-width="2" stroke-linecap="round" stroke-linejoin="round"`
- Mic icon on coral button: stays `fill="white"` for contrast
- Checkmarks inside colored circles: stay `stroke="white"` with `stroke-width="3"`
- The sparkle icon on the Post toggle (3 sparkles, ~20 path segments) is too complex at 14px. Replace with text-only "Post" label or a simple pencil icon (single path).

**Approach:** Do NOT add an icon library. Standardize each inline SVG in place. Group changes:
- [ ] Navigation chevrons (~5 instances): normalize to `stroke-width="2" stroke-linecap="round" stroke-linejoin="round"`
- [ ] Action icons (paperclip, camera, gallery, close X): same normalization
- [ ] Utility icons (spinner, arrow up): same normalization
- [ ] Remove sparkle SVG from Post toggle, keep text label only
- [ ] Verify the chat bubble icon in the Chat toggle also matches

**Gate:**

1. [ ] `cd web && pnpm check`
2. [ ] `cd web && npx playwright test` — all pass (no functional changes, only SVG attributes)
3. [ ] `grep -P 'stroke-width="(?!2"|2\.5"|3")' web/src/routes/\(app\)/operador/+page.svelte | grep -v 'stroke-width="1.5"'` returns 0 unexpected stroke widths. Allowed: 2 (standard), 2.5 (emphasis), 3 (checkmarks).
4. [ ] Screenshot gate in `tests/ux-warmth.spec.ts`:
   - Test "screenshot: full flow": take screenshots of list, chat, generate, info, new client on Moto G viewport (360x740). Save to `/tmp/pep019-final-*.png`. Visual review that icons are consistent (no automated assertion, manual check by developer).

---

## Consequences

**Positive:**
- The app will feel warmer and more approachable for the target audience
- Empty states guide users instead of showing blank screens
- New client onboarding becomes one-tap (mic) instead of scroll-then-find
- Mode confusion between chat and generate is reduced
- Touch targets and text meet accessibility minimums across the operator page
- Test suite gains coverage for visual regressions and accessibility

**Trade-offs:**
- Changing placeholder text breaks 6 test files (all fixable in Wave 1)
- Micro-copy is subjective. "Já vou..." may feel too informal for some operators. Can be tuned after user feedback.
- The coral-pale background in generate mode adds visual noise. If it feels heavy, reduce opacity.
- 13px minimum font size means section headers are slightly larger than typical "uppercase label" conventions. Readability wins over convention for this audience.

## Out of Scope

- Splitting the 2800-line operator page into components (do when already editing for a feature, not standalone)
- Custom illustrations or artwork (warm colors and good copy do more)
- Onboarding tours or tooltips (better empty states solve the same problem)
- Icon library dependency (standardize inline SVGs instead)
