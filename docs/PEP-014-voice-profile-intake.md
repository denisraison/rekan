# PEP-014: Voice-Guided Business Profile Intake

**Status:** Wave 1 + Wave 2 + Wave 3 shipped.
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

### Wave 1: Backend — Transcribe + Extract ✓

New endpoint that accepts an audio file, transcribes it with Gemini, extracts structured profile fields from the transcript, and returns them ready to pre-fill the form.

**New BAML function** in `eval/baml_src/profile.baml`:

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

**Gate:** ✓

- `go test ./api/...` passes
- Unit test for the handler using a mock transcription response: given a transcript "Ela tem salão em Manaus, faz hidratação por R$80 e progressiva por R$200, público é mulher de 30 a 50 anos", the extractor returns services=[{name:"Hidratação",price:80},{name:"Progressiva",price:200}], targetAudience="mulheres de 30 a 50 anos"

**Shipped:**

- `eval/baml_src/profile.baml` — `PartialService`, `PartialBusinessProfile` types + `ExtractBusinessProfile` function (JudgeClient / Gemini Flash, temp 0.1)
- `eval/profile.go` — Go wrapper: `ExtractFromAudioFunc` type, `ExtractBusinessProfile()`, `PartialBusinessProfile`, `PartialService`
- `api/internal/transcribe/gemini.go` — `Transcribe()` now accepts `mimeType` parameter
- `api/internal/http/handlers/extract_profile.go` — handler for `POST /api/businesses/profile:extract`
- `api/internal/http/handlers/extract_profile_test.go` — success + missing-audio tests
- Route registered in `routes.go`, wired in `main.go`

### Wave 2: Operator UI — Record, Extract, Pre-fill ✓

Voice recording widget added to the business creation and edit form in `web/src/routes/(app)/operador/+page.svelte`.

**State machine (shipped):**

```
'idle' → (tap mic) → 'recording' → (tap ↑) → 'analyzing' → 'done'
                                 → (tap ×) → 'idle'
'idle' → (tap "Preencher manualmente") → 'manual'
'manual' → (tap "Gravar →") → 'idle'
'done' → (tap "Gravar de novo") → 'idle'
'done' → (tap "Editar") → 'idle'
```

New clients open in `idle`. Editing an existing client opens in `manual` (fields already filled, no reason to force the voice path).

**Recording UX — diverged from prototype:**

The original design had the basic fields collapse to a summary chip during recording (full-screen takeover). During implementation this was replaced with a Claude-style recording bar: basic fields stay visible and editable, and the widget area transforms into a 72px full-width bar with three zones:

- Left (red-tinted): **×** cancels recording with no audio sent
- Center: blinking red dot + tabular-nums timer + "Gravando" label
- Right (coral): **↑** stops recording and submits for extraction

This is simpler, less disorienting, and gives Elenice the option to correct a basic field mid-recording without cancelling.

**Cancel vs submit separation:**

Two distinct functions handle the two exit paths from recording:

- `cancelRecording()`: sets `voiceMode = 'idle'` _before_ calling `recorder.stop()`, so the async `onstop` guard (`if voiceMode !== 'analyzing') return`) skips extraction entirely.
- `submitRecording()`: sets `voiceMode = 'analyzing'` then stops, extraction proceeds.

No explicit error state. On extraction failure, `voiceError` is shown and the form transitions to `manual` so Elenice can fill it herself.

**Save button logic:**

- `idle`: greyed out, disabled — Elenice must either record or switch to manual
- `recording` / `analyzing`: buttons hidden entirely (× and ↑ are the only actions)
- `done`: enabled "Salvar e continuar" + "Salvar e Enviar Convite"
- `manual`: full save buttons always enabled

**Pre-fill rules:**

- Services: replace list entirely if empty, append if Elenice had entered some manually
- `target_audience`, `brand_vibe`, `quirks`: only pre-fill if the field was empty
- `quirks` from API is `string[]` — joined with `\n` into the single textarea
- AI-filled fields get `--sage` border and `--sage-pale` background tint; tracked per-field via `aiFilledFields: Set<string>`

**Field label translations:**

| DB field          | Label in UI           |
| ----------------- | --------------------- |
| `target_audience` | Quem são os clientes? |
| `brand_vibe`      | Como é o ambiente?    |
| `quirks`          | O que faz diferente?  |

Each label has a concrete hint example below it (14px, `--text-muted`).

**Shipped:**

- Voice state machine in `+page.svelte`: `voiceMode`, `aiFilledFields`, `recordingSeconds`, `mediaRecorderRef`
- `startVoiceRecording()`, `cancelRecording()`, `submitRecording()`, `extractVoiceProfile()`, `resetVoice()`, `fmtTime()`
- `POST /api/businesses/profile:extract` called via `fetch` with `Authorization` header (same pattern as WhatsApp stream)
- CSS `@keyframes`: `pulse`, `ring`, `blink`, `spin` added in `<style>` block

**Gate:** ✓

- `pnpm check` passes (0 errors, 0 warnings)
- Playwright screenshots confirm: idle card visible without scrolling on 390×844, manual mode shows "Gravar →" banner + all fields + active save buttons

### Wave 3: Progressive Profile Enrichment ✓

After Wave 1 and Wave 2 ship, add a background enrichment layer: as the client sends WhatsApp messages, the system passively extracts profile-relevant signals and surfaces them as suggestions for Elenice to review.

**New background extraction step** in `api/internal/whatsapp/handler.go`:

After every incoming message is saved to the database, spawn a goroutine that checks whether the message content looks profile-relevant. Run a lightweight LLM check (single prompt, no BAML needed): "Does this message mention a service, price, differentiator, or business detail? If yes, return the relevant fields. If no, return null."

Only run this for messages where `content` is non-empty (text messages and image/video descriptions). Skip system messages and messages shorter than 20 characters.

**New `profile_suggestions` collection** (new migration):

| Field      | Type                             |
| ---------- | -------------------------------- |
| business   | relation → businesses            |
| field      | text (e.g. "services", "quirks") |
| suggestion | text                             |
| created    | auto                             |
| dismissed  | bool, default false              |

When extraction finds something useful, insert a row. Do not update the business profile directly — always go through Elenice.

**Operator UI: "Sugestões de Perfil" section** in the business detail view:

Show a collapsible card listing pending suggestions. Each suggestion shows:

- What field it relates to ("Serviço detectado", "Diferencial detectado")
- The extracted text
- Two buttons: "Adicionar" (writes the value into the business profile) and "Ignorar" (sets dismissed=true)

Keep it subtle — a dot badge on the client list item when there are pending suggestions. This is a tool for Elenice to improve profiles over time, not an alert.

**Gate:** ✓

- Migration applies cleanly
- On receiving a WhatsApp message containing "agora faço selagem também, R$150", a `profile_suggestions` row is created for that business
- The "Sugestões de Perfil" card appears in the operator UI with the detected service
- Accepting the suggestion adds it to `formServices` and saves

**Shipped:**

- `eval/baml_src/profile.baml` — `ProfileSignal` class + `ExtractProfileSignal` function (JudgeClient / Gemini Flash, temp 0.1)
- `eval/profile.go` — `ProfileSignal`, `ExtractSignalFunc`, `ExtractProfileSignal()`
- `api/internal/domain/domain.go` — `CollProfileSuggestions` constant
- `api/migrations/1740000022_profile_suggestions.go` — new collection (business, field, suggestion, dismissed)
- `api/internal/whatsapp/handler.go` — `ExtractSignalFunc` type, `ProfileSignal` type, `ExtractSignal` in `HandlerDeps`, `extractAndSaveSignal()` goroutine spawned after incoming text messages ≥ 20 chars
- `api/main.go` — wires `ExtractSignal` when `GEMINI_API_KEY` is set
- `web/src/lib/types.ts` — `ProfileSuggestion` interface
- `web/src/routes/(app)/operador/+page.svelte` — `loadAllSuggestionCounts()`, `loadSuggestions()`, `acceptSuggestion()`, `dismissSuggestion()`, realtime subscription, dot badge on client list items, collapsible "Sugestões de Perfil" card with Adicionar/Ignorar buttons

## Consequences

- Profile creation drops from 15-20 minutes to under 3 minutes
- First-batch post quality improves because profiles contain real, specific details (quirks, exact prices) rather than generic defaults
- Elenice still reviews and controls everything — no silent auto-updates to profiles
- Gemini API usage increases by 2 calls per new client (transcription + extraction) and approximately 1 call per 3 incoming WhatsApp messages (the lightweight relevance check in Wave 3)
- Wave 3 requires a new migration and a new collection; additive, non-breaking
- The `MediaRecorder` API requires HTTPS in production (already true for our NixOS deployment)
- Microphone permission must be granted by the browser; on mobile Chrome this requires a user gesture (the tap is the gesture, so it works)
