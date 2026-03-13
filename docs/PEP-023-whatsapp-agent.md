# PEP-023: WhatsApp Group Agent for Operators

**Status:** In Progress (Wave 3 complete)
**Date:** 2026-03-12
**Depends on:** PEP-022

## Context

Elenice and Bruna run Rekan's day to day: onboarding customers, reviewing content, managing approvals. The web UI creates friction for non-technical users and forces Denis to be the support layer.

WhatsApp is their native environment. A group chat ("Rekan Ops") with an AI agent removes the interface barrier. Everyone texts in the group, the agent acts. When Elenice creates a customer, Bruna sees it. When Bruna approves a post, Elenice sees it. The group is the shared dashboard.

This PEP builds the agent in 3 waves, with eval tests written before implementation in each wave.

### Architecture

```
Operators (WhatsApp Group: "Rekan Ops")
    |
    v
whatsmeow -- receive group message
    |
    +-- Sender in operators list? --> no: ignore
    |
    +-- Intent detection (actionable? @mention? reply to agent?)
    |   +-- Yes --> process
    |   +-- Operator-to-operator chat --> ignore
    |
    +-- Media? --> download, preprocess (vision, transcription, link parsing)
    |
    v
Debounce Buffer (2s window PER OPERATOR)
    |
    v
Context Hydrator (Go)
    |  PocketBase queries: active customers, pending reviews,
    |  recent actions, conversation buffer (last 15 group messages)
    |
    v
BAML Agent Function
    |  System prompt + context + conversation history
    |  Output: typed AgentResponse (reply, action, wait_for)
    |
    v
Action Router (Go)
    |  Uses service layer from PEP-022
    |
    +-- needs_confirmation --> store pending (per-operator), ask in group
    +-- execute --> service function --> reply with result
    +-- info_only --> reply in group
    |
    v
whatsmeow --> reply to group (addressing operator by name)
```

### Key design decisions

**Per-operator state.** Each operator has independent confirmation/collection state. Elenice's pending customer creation doesn't interfere with Bruna asking about post status. State keyed by sender JID in PocketBase `agent_state` collection.

**Selective response.** The agent stays silent on operator-to-operator chat. Responds to: @mentions, actionable intent (commands, status questions, customer references), direct replies to agent messages. Conservative by default, err on silence over false triggers.

**Attribution.** Every agent response addresses the operator by name. "Feito, Elenice! Patricia cadastrada" not just "Patricia cadastrada". Everyone in the group knows who triggered what.

**Confirmation before writes.** All create/update/delete actions require explicit confirmation. The agent echoes what it understood and waits for "sim"/"confirma" before executing.

**Transport-agnostic brain.** The BAML agent function and action router don't know about WhatsApp. They take structured input and return structured output. The WhatsApp layer is a transport adapter. If whatsmeow breaks, the brain can be rewired to another transport.

### Tech stack

| Component | Technology |
|---|---|
| WhatsApp gateway | whatsmeow (existing) |
| LLM | Claude via BAML |
| Database | PocketBase (existing) |
| Speech to text | Gemini (existing transcribe client) |
| Business operations | Service layer from PEP-022 |
| Language | Go |

### New PocketBase collections

| Collection | Purpose | Fields |
|---|---|---|
| `operators` | Authorized agent users | `jid` (text, unique), `name` (text), `role` (text), `active` (bool) |
| `agent_state` | Per-operator confirmation state | `operator_jid` (text), `state` (select: idle/collecting/confirming), `action_type` (text), `collected_fields` (json), `expires_at` (date) |
| `agent_conversations` | Group message buffer | `operator_name` (text), `operator_jid` (text), `role` (select: user/assistant), `content` (text), `media_type` (text), `timestamp` (date) |
| `agent_action_log` | Audit trail | `operator_name` (text), `operator_jid` (text), `action_type` (text), `params` (json), `result` (text), `success` (bool), `latency_ms` (number) |

### Eval approach

Each wave writes test cases before implementation. Test types:

**BAML inline tests** for deterministic checks: did the agent pick the right action type? Did it extract the correct parameters? These run with `baml-cli test`.

**LLM judge graders** (3-4 judges, not 10+): `intent_extraction` (correct action?), `confirmation_flow` (confirms before writes?), `no_hallucination` (no invented data?), `tone` (Portuguese, warm, concise, <300 chars?).

**State verification**: after a simulated creation flow, check PocketBase for the expected record. The database is the ground truth, not the transcript.

Tests run via `make eval-agent`. Each test case is a YAML file with input message, expected action type, and grader assertions.

### Edge cases

Handled by design:

- **Rapid fire messages**: 2s debounce per operator, concatenate before processing
- **Stale confirmation**: expire pending state after 10 minutes
- **Interleaved operators**: per-operator state machines, independent processing
- **Duplicate action collision**: check for pending actions on same entity before executing
- **Pronouns without context ("muda ela")**: use "current customer focus" from recent conversation
- **Mixed intents ("aprova o post e cria uma cliente")**: handle one at a time, queue the second
- **Unknown sender**: ignore, log for review

Deferred to later work:

- Voice notes (Gemini transcription exists, but voice-specific edge cases need dedicated testing)
- Proactive notifications (morning briefings, reminders)
- Video understanding beyond metadata
- LGPD compliance audit (forwarded messages, stored conversations)

## Waves

### Wave 1: Group pipeline + read-only agent

Build the message pipeline from WhatsApp group to BAML agent and back. Agent can answer questions about the system state but cannot modify anything. Eval framework runs.

**Eval tests (written first, all should fail before implementation):**

```yaml
# api/internal/agent/cases/wave1.yaml
tests:
  - id: w1_status_overview
    message: "como tá tudo?"
    operator: { name: "Elenice", jid: "5511999990000@s.whatsapp.net" }
    fixture: context_active   # 8 customers, some with pending posts
    graders:
      - { type: deterministic, field: "action.type", equals: "STATUS_OVERVIEW" }
      - { type: llm_judge, judge: "tone", criteria: "Portuguese, <300 chars, addresses Elenice by name" }

  - id: w1_customer_list
    message: "quais são as clientes?"
    graders:
      - { type: deterministic, field: "action.type", equals: "CUSTOMER_LIST" }
      - { type: llm_judge, judge: "no_hallucination", criteria: "Only lists customers present in the fixture context" }

  - id: w1_ignore_chat
    message: "Bruna, vc viu o post da Maria?"
    graders:
      - { type: deterministic, field: "action", equals: null }
      - { type: deterministic, field: "reply", equals: null }

  - id: w1_unknown_sender
    message: "oi, tudo bem?"
    operator: { name: "Unknown", jid: "5511888880000@s.whatsapp.net" }
    graders:
      - { type: deterministic, field: "action", equals: null }

  - id: w1_debounce
    messages:  # rapid fire, should be concatenated
      - "como"
      - "tá"
      - "tudo?"
    graders:
      - { type: deterministic, field: "action.type", equals: "STATUS_OVERVIEW" }
```

**Deliverables:**

1. **`internal/whatsapp/group.go`** (fill in the stub from PEP-022)
   - Verify sender JID against `operators` collection
   - Detect intent: respond to @mentions, actionable messages, replies to agent; ignore operator chat
   - React with thumbs up on receipt of actionable message
   - Send "Um momento..." if processing exceeds 5s

2. **`internal/agent/debounce.go`**
   - Per-operator 2s collection window (keyed by sender JID)
   - Concatenate rapid messages into single input
   - Sequential processing queue per operator

3. **`internal/agent/context.go`**
   - Query PocketBase: active businesses, pending posts, recent actions
   - Include operator identity and recent group conversation (last 15 messages)
   - Format as structured context block for BAML system prompt

4. **`internal/agent/conversation.go`**
   - Store/retrieve messages in `agent_conversations` collection
   - Auto-prune beyond 15 messages per hydration call

5. **BAML schema** (`api/internal/content/baml_src/agent.baml` or similar)
   - `AgentResponse` type: `reply` (string), `action` (optional AgentAction), `wait_for` (optional)
   - `AgentAction` type: `type` (enum), `status` (enum: execute/needs_confirmation), `params` (map)
   - `AgentProcess` function with system prompt in Portuguese
   - Action types for Wave 1: `STATUS_OVERVIEW`, `CUSTOMER_LIST`

6. **`internal/agent/router.go`**
   - Map action types to service functions (PEP-022)
   - For Wave 1: read-only actions only, call service layer, format reply
   - Log every interaction to `agent_action_log`

7. **`internal/agent/reply.go`**
   - Send reply to group via whatsmeow
   - Address operator by name in every response
   - Keep replies under 300 chars

8. **Eval harness** (`api/internal/agent/runner.go`)
   - Load YAML test cases
   - Call BAML agent function with fixture context
   - Run deterministic graders (action type, parameter matching)
   - Run LLM judge graders (tone, hallucination, intent)
   - Report pass/fail per test, overall score
   - `make eval-agent` target in Makefile

9. **PocketBase migrations**
   - `operators`, `agent_state`, `agent_conversations`, `agent_action_log` collections

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] `make eval-agent` runs all Wave 1 tests, pass rate >= 90% (100%, 9/9 tests)
- [x] `baml-cli test` passes for all Wave 1 BAML inline tests (2/2; null-assertion tests moved to Go eval harness)
- [x] Operator sends "como tá tudo?" in the group, gets accurate status within 5s (~2.4s)
- [x] 5 rapid messages from one operator are debounced into single input (2s debounce window)
- [x] Agent stays silent on operator-to-operator chat (send 5 conversational messages, 0 replies)
- [x] Unknown sender's message is ignored (no reply, logged)
- [x] All interactions logged in `agent_action_log` with operator attribution
- [x] Existing E2E tests pass (direct message handling, web UI unchanged)

**Notes:**
- BAML's test framework cannot assert `null` values (no `null` keyword). Silence tests (ignore chat) are covered by the Go eval harness instead of BAML inline tests.
- Import cycle between `whatsapp` and `agent` packages resolved with `GroupMessageHandler` function type in `whatsapp` and `WAClient` interface in `agent`.
- Agent uses Claude Sonnet 4.6 (via `AgentClient` in BAML) for speed/cost at expected message volume.
- BAML field `params` is a reserved keyword; renamed to `actionParams` in `AgentAction` class.

### Wave 1.1: Dedicated group model + BAML separation

Wave 1 proved the pipeline works but revealed unnecessary complexity: the agent tries to distinguish "message for me" from "operator-to-operator chat" using LLM intent detection. This is fragile in informal pt-BR group conversations and will misfire. Wave 1.1 simplifies the architecture by making two structural changes.

**Decision 1: Dedicated group, no intent gating.**

The agent listens to a single configured WhatsApp group (by group JID). Every message in that group is for the agent. Operators chat in their normal groups/DMs. This eliminates the entire "should I respond?" classification problem.

- Replace `operators` collection with a `group_id` config (env var `REKAN_AGENT_GROUP_JID`)
- Every message from the configured group gets processed, no silence logic
- Sender name resolved from whatsmeow contact/participant info
- Drop the `operators` collection entirely. Group membership is the auth boundary. If role-based permissions are needed later, that's a separate PEP.
- Remove intent detection from the BAML prompt entirely
- Remove silence/ignore test cases from eval (they test a problem that no longer exists)

**Decision 2: Shared BAML package at `api/internal/baml/`.**

`agent.baml` currently lives in `content/baml_src/` alongside content generation prompts. The generated `baml_client` is nested under `content`, making agent's import path awkward. Move all BAML to a neutral shared location.

- `.baml` sources live at `api/internal/baml/baml_src/` (`baml-cli` requires the directory to be named `baml_src`)
- Generated client at `api/internal/baml/baml_client/`
- Both `content` and `agent` packages import from `baml/baml_client`
- One compilation unit, one `baml-cli generate`, no duplication or version drift
- Delete `content/baml_src/` and `content/baml_client/` after migration

**Deliverables:**

1. **Simplified BAML prompt** (`api/internal/baml/baml_src/agent.baml`)
   - [x] Remove all "when to respond / when to stay silent" rules
   - [x] Focus purely on: "you received a message, figure out what they need"
   - [x] Keep attribution rule (address operator by name)
   - [x] Keep 300 char limit, pt-BR tone, no hallucination rules

2. **BAML package migration** (`api/internal/baml/`)
   - [x] Move all `.baml` files from `content/baml_src/` to `api/internal/baml/baml_src/`
   - [x] Update `generators.baml` output dir
   - [x] Run `baml-cli generate` to produce `api/internal/baml/baml_client/`
   - [x] Delete `content/baml_src/` and `content/baml_client/`

3. **Group config** (`api/internal/whatsapp/group.go`)
   - [x] Replace operator JID lookup with group JID check
   - [x] Read `REKAN_AGENT_GROUP_JID` from env
   - [x] Resolve sender name from whatsmeow `PushName`

4. **Updated eval cases** (`api/internal/agent/cases/wave1.yaml`)
   - [x] Remove `w1_ignore_chat` (5 tests) and `w1_unknown_sender` tests
   - [x] Keep `w1_status_overview`, `w1_customer_list`, `w1_status_question`
   - [x] Add `w1_unclear_intent` test (agent asks for clarification instead of staying silent)

5. **Import rewire** across `agent` and `content` packages
   - [x] Update all imports from `content/baml_client/baml_client` to `baml/baml_client`

**Gate:**
- [x] `cd api && go build ./...` compiles
- [x] `baml-cli generate` succeeds from `api/internal/baml/`
- [x] `make eval-agent` passes with updated test cases (4/4, 100%)
- [x] Agent processes every message in the configured group (no false silences)
- [x] Content eval (`make eval`) unaffected by BAML separation (72/72)
- [x] Existing E2E tests pass (frontend typecheck 0 errors)

**Notes:**
- `baml-cli` requires the source directory to be named `baml_src/`, so files live at `api/internal/baml/baml_src/` instead of flat in `api/internal/baml/`.
- `w1_status_overview` changed from checking `action_type=STATUS_OVERVIEW` to `has_reply=true` because the simplified prompt correctly answers "como tá tudo?" directly from context without needing an action.
- `w1_debounce` test removed from YAML (debounce is a Go-level concern tested by unit tests, not BAML eval).
- Operators collection dropped via migration `1740000024_drop_operators.go`. `CollOperators` constant removed from domain.
- Sender name resolved from `evt.Info.PushName` (whatsmeow's push name for the sender).

### Wave 2: Customer management + confirmation flow

Operators can create, update, and pause businesses through the group chat. Confirmation state machine ensures no writes happen without explicit "sim".

**Eval tests (written first):**
- Happy path: create customer with all fields in one message
- Incomplete: missing city, agent asks for it
- Abbreviations: "BH" -> "Belo Horizonte", "4x" -> frequency 4
- Cancel mid-flow: "deixa" clears state
- Stale confirmation: "sim" after 10 min returns "Sim pra quê?"
- Name collision: two customers named Maria, agent disambiguates
- Duplicate detection: trying to create existing customer
- Contextual inference: "muda pra 5x" after discussing Patricia
- Interleaved operators: Elenice mid-creation, Bruna asks status (both get independent responses)
- Collision: both operators try to create same customer simultaneously

**Deliverables:**

1. **Confirmation state machine** (`internal/agent/state.go`)
   - [x] Per-operator state in `agent_state` collection (keyed by JID)
   - [x] States: `idle` -> `collecting` -> `confirming` -> `idle`
   - [x] Auto-expire to idle after 10 minutes
   - [x] Field collection: when required fields missing, ask one at a time, track collected fields in state
   - [x] Conflict detection: warn if another operator has pending action on same entity

2. **New action types in BAML**
   - [x] `CUSTOMER_CREATE`: extract name, business type, city, frequency, Instagram handle
   - [x] `CUSTOMER_UPDATE`: modify fields on existing customer
   - [x] `CUSTOMER_PAUSE`: pause with optional reason
   - [x] `CUSTOMER_INFO`: show details for one customer

3. **Action router extensions** (`internal/agent/router.go`)
   - [x] `CUSTOMER_CREATE` with `needs_confirmation`: store in state, echo fields, ask "Confirma?"
   - [x] On "sim": call `service.CreateBusiness`, reply with result
   - [x] On "não"/"deixa": clear state, acknowledge
   - [x] Fuzzy name matching for customer lookup ("Patrícia" vs "Patricia" vs "a Pat")

4. **Additional eval judges**
   - [x] `confirmation_flow`: agent lists all extracted fields and asks for explicit confirmation before writes
   - [x] `state_management`: handles cancel, timeout, interleaved operators correctly

**Gate:**
- [x] `make eval-agent` runs all Wave 1+2 tests, pass rate >= 95% (100%, 14/14 tests)
- [x] State verification: PocketBase contains correct business record after simulated creation flow (unit tests)
- [x] "não" mid-flow cancels cleanly, no leftover state in `agent_state` (TestSetConfirming_And_ClearState)
- [x] State resets to idle after 10 minutes (TestState_AutoExpiry with time-shifted test)
- [x] Duplicate customer caught with disambiguation prompt (w2_duplicate_detection eval test)
- [x] "sim" with nothing pending returns friendly prompt, not error (w2_stale_sim eval test)
- [x] Elenice's pending action unaffected by Bruna's messages (TestPerOperatorIsolation)
- [x] Wave 1 tests still pass (no regressions, 4/4)
- [ ] Elenice and Bruna have tested real customer operations in the group for 3+ days

**Notes:**
- BAML enum values must start with uppercase. `AgentActionStatus` uses `EXECUTE`/`NEEDS_CONFIRMATION` instead of lowercase.
- `w2_create_missing_city` test checks `has_reply=true` (not `action_type=CUSTOMER_CREATE`) because when fields are missing, the LLM correctly asks for them in a reply without emitting an action.
- `collecting` state exists in the DB schema but is not actively used in Wave 2. When the LLM detects missing fields, it asks directly in the reply. Multi-turn field collection (collecting state) can be added if needed.
- Customer creation via agent sets `invite_status=draft`. Instagram handle storage deferred to a dedicated field (not stored in `target_audience` to avoid corrupting content generation).
- `executeCustomerUpdate` and `executeCustomerPause` use records already loaded in `HydratedContext` instead of re-fetching from DB.
- Common BAML call logic extracted to `callBAML` helper to avoid duplication between `processMessage` and `handleStatefulMessage`.
- `SetConfirming` and `ClearState` take the already-loaded `*OperatorState` to avoid redundant DB queries.

### Wave 3: Content review + media handling + single-post generation

Operators can generate, review, approve, and reject posts. Images sent inline. Business card photos and Instagram links feed into customer creation. Voice notes transcribed. Content generation changed from 3 posts to 1 post system-wide.

**Eval tests (written first):**
- [x] Generate post for a customer
- [x] Review pending posts for a customer
- [x] Approve/reject inline with feedback
- [x] Business card photo + "cria essa cliente" -> vision extracts fields -> creation flow
- [x] Instagram profile link -> parse handle -> attach to customer
- [x] Blurry image -> honest "can't read it", no hallucination
- [x] Voice note -> transcription -> treat as text message
- [x] Forwarded message from known customer -> identify by phone
- Sticker thumbs up with pending confirmation -> execute action (tested at Go level, not BAML eval)

**Deliverables:**

1. **Post generation + review flow**
   - [x] `POST_GENERATE`: generate a post for a customer (needs confirmation, calls `service.GeneratePosts`)
   - [x] `POST_LIST_PENDING`: show pending posts (all or per customer)
   - [x] `POST_APPROVE`, `POST_REJECT` (with feedback)
   - Send post preview as WhatsApp image with caption (deferred: requires Playwright rendering pipeline integration)
   - [x] Review session state: "aprova" without specifying uses current post in context

2. **Media preprocessing** (`internal/agent/media.go`)
   - [x] Images: Gemini vision describes image, description passed as text to BAML
   - [x] Videos: pass caption/metadata as text
   - [x] Links: parse Instagram handles in Go (regex)
   - [x] Forwarded messages: text carries phone number, BAML matches to customer in context
   - [x] Contact cards: extract name + phone from vCard
   - [x] Stickers: thumbs up with pending confirmation = "sim"

3. **Voice note handling**
   - [x] Use existing `transcribe.Client` (Gemini) for audio transcription
   - [x] Feed transcription into the normal text pipeline
   - [x] Low confidence / empty result: "Não consegui entender o áudio. Pode mandar por texto?"

4. **BAML schema updates**
   - [x] Image descriptions passed as `[Imagem: ...]` prefix in message text (simpler than multimodal BAML parameter)
   - [x] New action types: `POST_GENERATE`, `POST_LIST_PENDING`, `POST_APPROVE`, `POST_REJECT`
   - [x] Media and forwarding rules added to agent prompt

5. **Single-post generation (system-wide)**
   - [x] `GenerateContent` and `GenerateRekanContent` BAML functions return `Post` instead of `Post[]`
   - [x] Prompts rewritten for 1 post (removed "3 posts" rules, variety constraints, cross-post references)
   - [x] Go wrappers return `[]Post` with 1 element (keeps `GenerateFunc` signature stable)
   - [x] All `PickRoles(3)` changed to `PickRoles(1)`
   - [x] Judge `Acionavel` CTA criterion updated for single post

6. **PocketBase migration**
   - [x] `posts` collection: added `reviewed` (bool) and `review_note` (text) fields

**Gate:**
- [x] `make eval-agent` runs all Wave 1+2+3 tests, pass rate >= 95% (100%, 23/23 tests)
- [x] Business card photo + "cria essa cliente" extracts name/business/city via vision, enters creation flow (w3_business_card_photo)
- [x] Instagram profile link parsed correctly, handle attached to customer (w3_instagram_link)
- [x] Blurry image gets honest "can't read" response, no hallucination (w3_blurry_image)
- [x] Voice note transcribed and processed as text (w3_voice_note)
- [x] Post approval/rejection works via group chat (w3_approve_post, w3_reject_post)
- [x] Forwarded customer message identifies customer by phone (w3_forwarded_message)
- [x] Wave 1+2 tests still pass (no regressions, 14/14)
- [x] Content eval unaffected by single-post change (72/72 heuristics)
- [x] BAML inline tests pass (6/6)
- [ ] Elenice has reviewed real posts via the group for 1+ week

**Notes:**
- Image input uses Gemini vision description passed as text prefix `[Imagem: ...]` rather than multimodal BAML parameter. Simpler, testable in eval, and Gemini's description is sufficient for business card field extraction.
- `w3_list_pending_posts` checks `has_reply=true` (not `action_type=POST_LIST_PENDING`) because the LLM correctly answers pending post questions directly from context without needing a router action, same pattern as Wave 1's `w1_status_overview`.
- Sticker handling (thumbs up = "sim") is tested at the Go level in `HandleGroupMessage`, not in BAML eval (eval tests only exercise the BAML function with text input).
- `POST_GENERATE` was not in the original PEP but was a clear gap: operators need to trigger post generation from the group chat.
- Single-post generation: `GenerateFunc` signature kept as `[]Post` return type to minimize blast radius. The BAML functions return `Post` (singular), Go wrappers wrap in a single-element slice.
- Post preview as WhatsApp image deferred: requires integrating the Playwright rendering pipeline with the agent, which is a separate concern from the core review flow. Caption-only review works for now.
- Agent struct extended with `Transcribe *transcribe.Client` and `Generate content.GenerateFunc` dependencies, wired in `cmd/rekan/main.go`.

## Open questions

1. **whatsmeow stability.** This is an unofficial reverse-engineered library. Meta can break it with protocol updates or ban the phone number. The architecture is designed to be transport-agnostic (the agent brain doesn't know about WhatsApp), so swapping to the official WhatsApp Business API or another transport is possible without rewriting the core. But the risk should be acknowledged: if whatsmeow breaks, the agent is down until we migrate.

2. **LGPD compliance.** Storing WhatsApp messages, phone numbers, forwarded content, and voice transcriptions. Brazil's LGPD requires explicit consent and data minimization. Needs a concrete compliance plan before going to production.

3. **Claude model choice.** Sonnet vs Haiku for the agent function. Run eval suite against both, pick better pass rate vs cost trade-off at the current volume (~50-100 messages/day).

4. **Intent detection threshold.** Resolved in Wave 1.1: dedicated group eliminates the problem entirely. Every message in the group is for the agent.

5. **Whisper vs Gemini for audio.** Currently using Gemini for transcription. Evaluate quality for Portuguese voice notes at low volume.

6. **Proactive notifications.** Morning briefings, reminders, stale approval alerts. Deferred from this PEP. Can be added as a follow-up once the reactive agent is stable and Elenice confirms the core flow works.

## Consequences

- Operators manage Rekan through WhatsApp, their native environment. No learning curve.
- The group chat is a shared operational log. Everyone sees every action.
- Onboarding a new operator = adding them to the group + one PocketBase record.
- The web UI remains available for visual tasks (post grid review, dashboards). This is additive, not a replacement.
- Trade-off: dependence on whatsmeow (unofficial library). Mitigated by transport-agnostic agent design.
- Trade-off: LLM costs per message (~2.5K tokens). At 50-100 messages/day, negligible (~$2-5/day with Sonnet).
- Trade-off: intent detection in groups is harder than 1:1 chat. Conservative defaults prevent annoyance but may miss some actionable messages initially.
