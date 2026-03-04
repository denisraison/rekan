# PEP-014: Voice-Guided Business Profile Intake

**Status:** Draft
**Date:** 2026-03-04

## Context

When Elenice onboards a new client, she fills a form with 8+ fields:
- business name, type, city, state, phone (objective, fast)
- services list with prices (tedious to type)
- target_audience, brand_vibe, quirks (conceptually hard — business owners don't think in these terms)

This takes 15-20 minutes and produces low-quality data. The fields that matter most (quirks, brand_vibe) are the hardest to fill. Nobody types "ela amassa o pão com as próprias mãos desde 87" into a field labeled "Diferenciais."

Meanwhile Elenice is already having a conversation — with or about the client, in person or on the phone. The information exists. It's coming out of her mouth. It just doesn't land anywhere useful.

The key insight: **capture the conversation, not the form**. Elenice hits record, talks for 60 seconds about the business ("Essa é a Marina, tem salão no Adrianópolis, faz muito corte feminino e progressiva, o diferencial é que serve cafezinho na espera"), stops, and the form pre-fills from the transcript. She reviews, tweaks if needed, saves. Under 3 minutes.

This matters because first-batch post quality determines whether the client stays. Generic profiles (inferred from business type alone) produce generic posts. A client who sees "Bem-vindo ao seu salão de beleza!" churns before the profile ever improves.

**What we're NOT building:**
- Auto-inferred profile from business type only (too generic, high churn risk on first posts)
- WhatsApp conversational intake for the client (uncanny/impersonal for 50+ demographic, high abandonment risk mid-flow)
- Progressive enrichment without voice intake first (doesn't solve the cold-start problem)

## Waves

### Wave 1: Backend — Transcribe + Extract

New endpoint that accepts an audio file, transcribes it with Gemini, extracts structured profile fields from the transcript, and returns them ready to pre-fill the form.

**New BAML function** in `eval/baml_src/content.baml` (or a new `profile.baml`):

```
function ExtractBusinessProfile(transcript: string, businessType: string) -> PartialBusinessProfile
```

`PartialBusinessProfile` is a class with all fields optional: `services` (name + price), `targetAudience`, `brandVibe`, `quirks`. The function asks the LLM to extract only what is clearly mentioned, never invent. If a field isn't in the transcript, return null for it.

The prompt should instruct the LLM to:
- The input is casual, unstructured spoken Brazilian Portuguese — expect filler words, incomplete sentences, restarts. That is normal.
- Extract services and prices literally from speech ("selagem por R$150" → service: "Selagem", price: 150)
- Infer target_audience from context clues ("mulheres da região", "jovens que querem emagrecer")
- Capture brand_vibe from adjectives and atmosphere descriptions ("acolhedor", "bem-humorado", "família")
- Lift quirks verbatim when possible — these should sound like something a person said, not an AI summary
- If a field is not mentioned, return null. Never invent. A partial result with 2 fields is better than a complete result with hallucinated values.

**New backend endpoint** in `api/internal/http/handlers/`:

```
POST /api/businesses/profile:extract
Auth: required
Content-Type: multipart/form-data
Body: audio file + business_type (string)
Response: { services, target_audience, brand_vibe, quirks }
```

Flow:
1. Receive audio file from multipart body
2. Call `transcribe.Transcribe(ctx, audioBytes, mimeType)` — this already exists in `api/internal/transcribe/gemini.go`
3. Call the new BAML `ExtractBusinessProfile` with the transcript and business_type hint
4. Return the extracted partial profile as JSON

No database writes. This is a pure computation endpoint. The operator reviews the result and saves via the normal business create/update flow.

**Gate:**
- `go test ./api/...` passes
- Unit test for the handler using a mock transcription response: given a transcript "Ela tem salão em Manaus, faz hidratação por R$80 e progressiva por R$200, público é mulher de 30 a 50 anos", the extractor returns services=[{name:"Hidratação",price:80},{name:"Progressiva",price:200}], targetAudience="mulheres de 30 a 50 anos"

### Wave 2: Operator UI — Record, Extract, Pre-fill

Add a voice recording widget to the business creation and edit form in `web/src/routes/(app)/operador/+page.svelte`.

**Recording widget behavior:**
- Shows a microphone button labeled "Gravar descrição"
- On tap: requests microphone permission, starts `MediaRecorder`, button turns red with a pulsing indicator and a "Parar" label
- **While recording**, show a soft prompt card below the button — not a script, just anchors so Elenice doesn't freeze:
  > *Fala do jeito que quiser, sem pressa. Que serviços ela faz e quanto cobra? Quem são os clientes? O que ela faz diferente das outras?*
  She can answer in any order, skip anything she doesn't know, and talk as casually as she wants. Rambling is fine. The card is a visual cue, not a checklist.
- **No auto-stop.** Recording continues until Elenice taps "Parar" — no timeout, no silence detection, no prediction of when she's done. She controls when it ends, just like Claude's voice mode.
- On stop: sends audio to `POST /api/businesses/profile:extract` with current `formType` as `business_type`
- Shows a loading spinner with "Analisando..." while waiting
- On success: pre-fills `formServices`, `formTargetAudience`, `formBrandVibe`, `formQuirks` from the response. Fields that were already filled by Elenice are NOT overwritten — only empty fields get pre-filled.
- Shows a subtle success banner: "Perfil extraído da gravação. Revise os campos antes de salvar."
- On error: show "Não foi possível analisar o áudio. Preencha os campos manualmente." — the form remains fully editable

**Placement:** The recording widget goes between the basic fields (name, type, city, phone) and the content fields (services, target_audience, brand_vibe, quirks). The idea: Elenice fills the objective fields first (30 seconds), then records the description, then reviews the pre-filled content fields.

**Services pre-fill:** The services field is currently a list with add/remove rows. When the extraction returns services, replace the list entirely if it was empty, or append if some services were already entered manually.

**Mobile-first:** The record button must be large enough for a fat thumb. The pulsing red indicator must be unmistakable. Test on a real phone.

#### UI design decisions (from prototyping)

Four states were prototyped and validated. Prototype HTML files are in [`docs/designs/PEP-014/`](designs/PEP-014/):

| File | State |
|---|---|
| [`voice-idle.html`](designs/PEP-014/voice-idle.html) | Ready to record |
| [`voice-recording.html`](designs/PEP-014/voice-recording.html) | Recording in progress |
| [`voice-done.html`](designs/PEP-014/voice-done.html) | Post-extraction review |
| [`voice-manual.html`](designs/PEP-014/voice-manual.html) | Manual fallback form |

To preview: `open docs/designs/PEP-014/voice-idle.html` or screenshot with `npx playwright screenshot --viewport-size="390,844" "file:///path/to/file.html" out.png`.

**State 1: Idle (ready to record)**

The recording widget is a full-width tappable card with a coral mic button (72px circle) and "Toca no microfone e fala sobre a cliente". It sits directly below the 4 basic fields, visible without scrolling. The save button is grey/disabled — Elenice can't save until she either records or fills manually.

Layout: basic fields → recording card → "Preencher manualmente" link (quiet, secondary) → disabled save bar.

The card uses `--coral-pale` background and `--coral-light` border to draw the eye without being aggressive.

**State 2: Recording in progress**

The basic fields collapse into a single summary chip (business name + type + city) at the top. The whole screen becomes about one thing. Layout:

- Summary chip (tappable to edit)
- Large red pulsing mic button (96px) with two expanding ring animations
- Timer counting up (e.g. "0:42") with a blinking red dot
- "Gravando..." label
- Prompt card below: "PODE FALAR SOBRE..." with 3 emoji-anchored lines and the italic note "Fala do jeito que quiser, sem pressa. Não precisa seguir a ordem."
- Full-width black "Parar gravação" button pinned to bottom (60px tall)

No auto-stop. Elenice controls when it ends.

**State 3: Analyzing / done**

After tapping stop, the mic area shows a spinner with "Analisando...". On success, transition to the review state:

- Summary chip stays at top
- Sage-green banner: checkmark + "Perfil extraído da gravação" + "Revise os campos antes de salvar."
- Pre-filled fields rendered with `--sage` border and `--sage-pale` background tint to distinguish AI-filled from manually entered
- Services appear as editable rows (name + price + × remove)
- Público, Estilo, Diferenciais appear as editable textareas/rows
- "Gravar de novo" link at bottom of fields (secondary, quiet)
- Enabled coral "Salvar e continuar" pinned to bottom

**State 4: Manual fallback**

When Elenice taps "Preencher manualmente", the form expands with all fields. A "Gravar →" banner (full-width tappable, 64px tall, `--coral-pale` background) stays pinned at the top of the scroll area so she can switch back to voice any time.

**Field label translations** — the technical field names are never shown to Elenice:

| DB field | Label in UI |
|---|---|
| `target_audience` | Quem são os clientes? |
| `brand_vibe` | Como é o ambiente? |
| `quirks` | O que faz diferente? |

Each field has a hint line with a concrete example (14px, `--text-muted`):
- Público: *ex: mulheres de 25 a 50 anos que moram no bairro*
- Ambiente: *ex: acolhedor, descontraído, serve cafezinho*
- Diferente: *ex: agenda lotada às quintas*

**Touch target rules confirmed:**
- All inputs: min 52px tall
- Primary buttons (Parar, Salvar): min 56-60px tall, full-width, `--radius-full`
- Remove (×) buttons on service rows: 40px wide × 52px tall
- "Gravar →" banner in manual fallback: 64px tall, entire row is tappable (not just the text)
- Secondary text links ("Preencher manualmente", "Gravar de novo"): min 44px tap area via padding

**Svelte state machine:**

```
'idle' → (tap mic) → 'recording' → (tap stop) → 'analyzing' → 'done'
                                                              → 'error' → back to 'idle'
'idle' → (tap "Preencher manualmente") → 'manual'
'manual' → (tap "Gravar →") → 'idle'
'done' → (tap "Gravar de novo") → 'idle'
```

**Gate:**
- `cd web && pnpm check` passes (no type errors)
- Playwright screenshot confirms recording card visible without scrolling on 390×844 viewport
- Manual test on a real phone: recording card visible, tap starts mic, tap stops and shows analyzing spinner, fields pre-fill with sage tint, save button enables
- Manual fallback: tapping "Preencher manualmente" shows full form; "Gravar →" banner switches back

### Wave 3: Progressive Profile Enrichment (Phase 2)

After Wave 1 and Wave 2 ship, add a background enrichment layer: as the client sends WhatsApp messages, the system passively extracts profile-relevant signals and surfaces them as suggestions for Elenice to review.

**New background extraction step** in `api/internal/whatsapp/handler.go`:

After every incoming message is saved to the database, spawn a goroutine that checks whether the message content looks profile-relevant. Run a lightweight LLM check (single prompt, no BAML needed): "Does this message mention a service, price, differentiator, or business detail? If yes, return the relevant fields. If no, return null."

Only run this for messages where `content` is non-empty (text messages and image/video descriptions). Skip system messages and messages shorter than 20 characters.

**New `profile_suggestions` collection** (new migration):

| Field | Type |
|-------|------|
| business | relation → businesses |
| field | text (e.g. "services", "quirks") |
| suggestion | text |
| created | auto |
| dismissed | bool, default false |

When extraction finds something useful, insert a row. Do not update the business profile directly — always go through Elenice.

**Operator UI: "Sugestões de Perfil" section** in the business detail view:

Show a collapsible card listing pending suggestions. Each suggestion shows:
- What field it relates to ("Serviço detectado", "Diferencial detectado")
- The extracted text
- Two buttons: "Adicionar" (writes the value into the business profile) and "Ignorar" (sets dismissed=true)

Keep it subtle — a dot badge on the client list item when there are pending suggestions. This is a tool for Elenice to improve profiles over time, not an alert.

**Gate:**
- Migration applies cleanly
- On receiving a WhatsApp message containing "agora faço selagem também, R$150", a `profile_suggestions` row is created for that business
- The "Sugestões de Perfil" card appears in the operator UI with the detected service
- Accepting the suggestion adds it to `formServices` and saves

## Consequences

- Profile creation drops from 15-20 minutes to under 3 minutes
- First-batch post quality improves because profiles contain real, specific details (quirks, exact prices) rather than generic defaults
- Elenice still reviews and controls everything — no silent auto-updates to profiles
- Gemini API usage increases by 2 calls per new client (transcription + extraction) and approximately 1 call per 3 incoming WhatsApp messages (the lightweight relevance check in Wave 3)
- Wave 3 requires a new migration and a new collection; additive, non-breaking
- The `MediaRecorder` API requires HTTPS in production (already true for our NixOS deployment)
- Microphone permission must be granted by the browser; on mobile Chrome this requires a user gesture (the tap is the gesture, so it works)
