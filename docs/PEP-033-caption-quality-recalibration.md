# PEP-033: Caption Quality Recalibration

**Status:** In Progress
**Date:** 2026-03-15

## Context

Users report that generated captions are too long. We scraped 282 real Instagram captions from 60 accounts across 11 niches using Playwright (`scripts/discover-and-scrape.mjs`). Of those, 164 posts come from 35 MEI-sized accounts (500-100K followers) across 7 niches: confeitaria, costureira, diarista, hair, loja, marmiteira, nails.

All research data lives in `docs/caption-research/` (JSON + post screenshots + SQLite). The database can be rebuilt with `node scripts/research-db.mjs` and queried with `--query="SQL"` or `--summary`.

### Instagram algorithm context (2026)

The algorithm changed significantly. Understanding these shifts is critical because our prompt and judge changes must align with how Instagram actually distributes content today.

**Algorithm signal weights (2026):**

| Signal | Weight | Implication |
|---|---|---|
| Watch time / time on post | 35% | Longer captions can increase time spent, but only if the reader actually reads them. Wall-of-text captions get skipped. |
| Saves | 25% | Content worth revisiting. "Salva pra depois" CTAs drive this signal. |
| Shares via DM | 20% | "Manda pra uma amiga" CTAs drive this. Strongest distribution signal. |
| Comments >7 words | 15% | Meaningful conversation, not "lindo!" |
| Likes | 5% | Nearly irrelevant now. |

Our engagement metric (likes + comments / followers) only captures ~20% of what the algorithm actually cares about. Saves and shares are invisible to public scraping. This is a known limitation of our dataset, not a reason to ignore the data, but worth noting when interpreting results.

**Platform changes:**
- December 2025: Instagram capped hashtags at 5 per post (down from 30). The old "sempre 5" / "sempre 30" strategy is officially dead.
- Instagram now functions as a search engine. The algorithm reads caption text to understand and categorize content. Keyword-rich captions generate ~30% more reach than hashtag-heavy posts.
- Shares and saves replaced likes as the primary distribution signals. Content that people send via DM gets the strongest algorithmic boost.
- Nano/micro accounts (1K-100K) average 2-6% engagement, well above the platform average of 0.48%. MEI-sized accounts are in the algorithm's sweet spot.

**Professional consensus (Brazilian marketing, Sebrae, Rafael Terra):**
- 3-5 specific hashtags per post (not 0, not 30). Our data shows 0 performs best for MEI, but professional consensus says a few relevant ones still help categorization.
- Keyword SEO in captions matters more than hashtags for discovery. The first two sentences are what the algorithm reads most closely.
- Reels dominate reach (4+ weekly recommended), but feed posts still matter for product showcases and carousels.
- "Save" and "share" CTAs boost the algorithm signals that matter most. "Link na bio" and "chama no zap" CTAs push people off-platform and hurt distribution.

### What the data says

**Caption length vs engagement (MEI accounts, 500-100K followers):**

| Length | Posts | Avg engagement rate |
|---|---|---|
| < 100 chars | 61 | 1.93% |
| 100-200 | 23 | 1.94% |
| **200-400** | **28** | **2.28%** |
| 400-600 | 21 | 1.24% |
| 600+ | 31 | 1.04% |

The 200-400 char range has the highest engagement. Under 200 also performs well. Above 400 drops off. Our current 700 char limit pushes content into the worst-performing range.

**CTA type matters more than CTA presence:**

| | Posts | Avg engagement |
|---|---|---|
| No CTA | 123 | 2.04% |
| With CTA | 41 | 0.82% |

Posts without CTAs get 2.5x the engagement in our data. But this needs nuance: professional research (HubSpot 2026) shows CTAs boost engagement 23-31% when they are the right type. The difference is CTA type:

- **Platform-native CTAs** ("salva esse post", "manda pra uma amiga que precisa", "comenta aqui"): Drive saves, shares, and comments, the three signals the algorithm weights most (25% + 20% + 15% = 60%). These HELP distribution.
- **Exit CTAs** ("chama no zap", "link na bio", "acesse o site"): Push people off Instagram. The algorithm penalizes content that drives users away from the platform. These HURT distribution.

Most MEI CTAs in our dataset are exit-type ("chama no zap" dominates). That explains why "no CTA" beats "with CTA" in our data. The real rule: prefer platform-native CTAs, avoid exit CTAs, and never force either.

**Hashtags: less is more (and Instagram agrees):**

| Hashtags | Posts | Avg engagement |
|---|---|---|
| 0 | 92 | 2.03% |
| 1-3 | 28 | 1.21% |
| 4+ | 44 | 1.45% |

Zero hashtags performs best in our MEI data. The 4+ bucket is inflated by a few costureira accounts. Instagram officially capped hashtags at 5 per post in December 2025, confirming that hashtag stuffing is dead. Professional consensus (Sebrae, Rafael Terra) recommends 3-5 specific hashtags for categorization, but our data suggests MEI accounts do fine with 0-3. Hashtags now serve as topic labels for the algorithm, not as reach amplifiers.

**Other patterns (all 282 posts):**
- 77% of posts have no CTA at all
- 66% use line breaks / paragraph structure
- Only 4% mention prices in the caption
- Only 5% use em dashes
- Average emoji count: 2-4 per post depending on niche
- Confeitaria MEIs use 0 hashtags. Nails uses the most (6.7 avg) but those are spam-style.

**Niche engagement ranking (MEI accounts):**

| Niche | Accounts | Posts | Avg chars | Avg engagement |
|---|---|---|---|---|
| costureira | 7 | 30 | 431 | 2.20% |
| loja | 6 | 24 | 357 | 2.00% |
| confeitaria | 4 | 21 | 239 | 1.85% |
| diarista | 6 | 32 | 440 | 1.78% |
| marmiteira | 6 | 25 | 289 | 1.57% |
| nails | 3 | 16 | 370 | 1.43% |
| hair | 3 | 16 | 67 | 0.77% |

**Top engaged MEI posts** share a pattern: genuine voice, personal stories about the business, no formulaic structure. A confeiteira writing about her business growing (442c, 15.87% engagement), a diarista sharing a personal moment (97c, 12.74%), a small atelier showing their work at a fashion show (118c, 10.51%). None of these follow our current template.

### Mismatches between our system and reality

| Dimension | Our system | Real Instagram (MEI data + professional research) |
|---|---|---|
| Caption length | Max 700 chars | Sweet spot 200-400c (2.28% engagement). Longer only for personal stories. |
| Hashtags | "sempre 5" (forced) | Platform cap is now 5. Our data: 0 best. Pros say 3-5 specific. Target: 0-3. |
| CTA | Required on every post | Exit CTAs ("chama no zap") hurt. Platform CTAs ("salva", "manda pra amiga") help. Never force. |
| Keywords | Not mentioned | Algorithm reads captions as search text. Niche keywords in first 2 sentences boost reach ~30%. |
| Line breaks | Not mentioned | 66% of posts use paragraph breaks |
| Structure | Freeform block of text | Short paragraphs with whitespace |
| Dashes | Common in our output | Only 5% of real posts use em dashes. Strong AI-tell. |

The judges and heuristics enforce the old rules. JudgeAcionavel requires quality hashtags and CTA, which would reprove the highest-engaging real posts. The hashtag heuristic requires >= 3. If we update the prompt without updating the eval stack, the optimization loop will fight itself.

## Wave 1: Prompt + heuristic alignment [DONE]

Update the generation prompts and heuristic checks to match real Instagram patterns.

### `api/internal/baml/baml_src/content.baml` (both `GenerateContent` and `GenerateFromMessage`)

- [x] Change `Legenda (máximo 700 caracteres)` to `Legenda (200 a 400 caracteres)`. Not a hard max, but a target range. The data shows this is where engagement peaks.
- [x] Change `Hashtags do nicho e da região (sempre 5)` to `Hashtags do nicho (0 a 3, só se fizer sentido para o post)`. Instagram capped at 5 in Dec 2025. Our data shows 0 performs best, pros say 3-5. Target 0-3 relevant ones.
- [x] Rewrite CTA instruction to distinguish types: `CTA é opcional. Se incluir, prefira CTAs que mantêm a pessoa no Instagram: "salva esse post", "manda pra uma amiga que precisa", "comenta aqui". Evite CTAs que tiram a pessoa do Instagram: "link na bio", "chama no zap". O algoritmo penaliza saídas da plataforma.`
- [x] Add keyword SEO instruction: `Use palavras-chave do nicho naturalmente na legenda, especialmente nas duas primeiras frases. Ex: "bolo de aniversario personalizado" em vez de só "bolo lindo". O Instagram funciona como buscador e lê o texto da legenda pra distribuir o conteúdo.`
- [x] Add line break instruction: `Separe ideias em parágrafos curtos com linha em branco entre eles. Evite blocos de texto corrido.`
- [x] Anti-dash instruction already existed (`NUNCA use travessão`). No change needed.
- [x] Simplify production note instructions. Reduced from 4 detailed items to 2-3 sentences.

### `api/internal/baml/baml_src/rekan.baml` (`GenerateRekanContent`)

- [x] Same changes: 700 -> "200 a 400" chars, "mínimo 5" -> "0 a 3", CTA type distinction, keyword SEO, line breaks, no dashes.

### `api/internal/content/heuristic.go`

- [x] `checkHashtags`: Always passes. Parameter removed, function is a single return statement.
- [x] `checkCaptionLength`: Keeps 2200 hard cap. Added soft warnings at >500 and <50 chars. Fixed a bug where soft warning early returns could skip the hard limit on later posts in a batch.
- [x] `checkBrazilianPortuguese`: No change.
- [x] `checkProductionNote`: No change.

### `api/internal/content/heuristic_test.go`

- [x] Updated tests for new hashtag behavior (always passes).
- [x] Added tests for soft caption length warnings (>500 and <50).

### Gate

```bash
cd api && go test ./internal/content/...
cd web && pnpm check
make eval   # heuristics pass with new rules
```

All gates passed. `make eval` scored 72/72 (18 profiles, 4/4 checks each).

## Wave 2: Judge recalibration [DONE]

Update judge prompts to align with real Instagram patterns, then validate with `--from-run`.

### `api/internal/baml/baml_src/judges.baml`

**JudgeAcionavel** (biggest change): This judge currently requires quality hashtags, CTA, production note, and fluidity. The highest-engaging real posts have zero CTAs and zero hashtags. Rewrite to:
- [x] Remove hashtag quality as a criterion (hashtags are optional now)
- [x] Make CTA quality conditional: only evaluate CTA quality IF a CTA is present. Absence of CTA is fine.
- [x] If CTA is present, penalize exit CTAs ("link na bio", "chama no zap") unless the post is explicitly a sales/promo post. Prefer platform-native CTAs ("salva", "manda pra amiga", "comenta").
- [x] Keep production note quality check (still important for our users)
- [x] Keep fluidity check (still valid)
- [x] Reprove threshold: reprove if production note is vague OR if present CTA is generic/disconnected OR if exit CTA is used on a non-sales post. Don't reprove for absence of CTA or hashtags.
- [x] Removed stale "Já foi verificado" preamble that referenced old assumptions.
- [x] Updated reprovação example to remove hashtag block (no longer a criterion).

**JudgeEngajamento**: Minor adjustment. Currently expects every post to have a "genuinely interesting hook." Real top-performing posts include personal stories about the business, product showcases, and straightforward announcements. Update:
- [x] Keep the anti-formula criteria (penalize "Voce sabia que...?", "Gente, prepara o coracao!")
- [x] But soften the approval bar: a post with genuine voice and personality passes even if it's a straightforward announcement. Not every post needs to be a storytelling masterpiece.
- [x] Add: course/product announcements with specific details (price, date, link) are valid if they sound natural, not salesy.
- [x] Add: penalize em dashes (—). Only 5% of real posts use them, but our system overuses them. They signal AI-generated content.
- [x] Removed "service description without surprising angle" criterion (too strict for genuine announcements).
- [x] Removed stale "Já foi verificado" preamble.
- [x] Em dash threshold aligned with JudgeNaturalidade: "múltiplos travessões" (2+), not any single occurrence.

**JudgeNaturalidade**: Add em dash detection. Real MEI posts almost never use — but LLMs love them. A caption with multiple em dashes should be flagged as AI-sounding.
- [x] Added em dash as AI signal with "múltiplos travessões" threshold.

**JudgeEspecificidade**: No change. The "decoracao de pitch" test is good.

**JudgeVariedade**: No change.

### `api/internal/content/judge_test.go`

- [x] Added `TestJudgeNoCTAContent` with a no-CTA/no-hashtag natural announcement post. Asserts JudgeAcionavel and JudgeEngajamento both pass.
- [x] Existing golden tests (`TestJudgeKnownGoodPassesMost`, `TestJudgeKnownBadFailsMost`) unchanged and still pass.

### `api/cmd/eval/main.go`

- [x] Added concurrency semaphore (4 profiles max) to judge evaluation to avoid API rate limits. Was hitting 429s with 18 profiles x 5 judges x 2 models = 180 concurrent requests.

### Gate

```bash
cd api && go test ./internal/content/... -run TestJudge
# Save a baseline before changes:
make eval-judges  # save output as before.json
# After changes, re-judge same content:
go run ./cmd/eval --from-run runs/BEFORE.json --judges
go run ./cmd/eval --diff runs/BEFORE.json runs/AFTER.json
```

The diff should show: acionavel pass rate goes UP (fewer false negatives from missing CTA/hashtags), other judges stay stable or improve. No judge should regress by more than 1 profile.

Results: ACI 8/18 -> 18/18 (+10), ESP 16/18 -> 15/18 (-1, within tolerance), NAT/VAR/ENG stable at 18/18.

## Wave 3: End-to-end validation

Full generation + judgment run to verify the complete pipeline produces better content.

### Steps

- Run `make eval-judges` with all changes applied. Save as AFTER.json.
- Compare against the baseline from Wave 2 with `--diff`.
- Run `--verbose` on 2-3 profiles to read the actual generated captions. Verify they are shorter (~200-400c), use line breaks, have 0-3 hashtags, no em dashes, and only include CTAs when natural.
- Spot-check production notes are simpler and more practical.

### Gate

```bash
make eval-judges
go run ./cmd/eval --diff runs/BEFORE.json runs/AFTER.json
go run ./cmd/eval --judges --verbose --profile "Closet da Re"
go run ./cmd/eval --judges --verbose --profile "Salao da Vania"
```

Overall pass rate should be >= baseline. Caption average length should be under 450 chars (check in verbose output). No judge criterion should have pass rate drop by more than 10% compared to baseline.

## Optimization strategy Loop

We added a `--cheap` flag to the eval command and a `CheapGeneratorClient` (Gemini Flash, temp 0.7) in `clients.baml`. This enables a two-phase optimization approach:

**Phase 1: Iterate with Gemini Flash** (`make eval-cheap`)
- Uses `CheapGeneratorClient` (Gemini Flash) for generation instead of Opus
- Combined with `--fast` (single judge, 4 profiles) for maximum speed
- ~10x cheaper per run than Opus generation
- Use this for all iteration cycles during prompt optimization (`/optimize` skill)

**Phase 2: Validate with Opus** (`make eval-judges`)
- Final run uses the production model (Opus) with both judge models
- Compare Flash vs Opus results with `--diff` to ensure quality holds
- If Opus results regress on any criterion vs Flash, the prompt may be over-fitted to Flash's style

**New CLI flags and targets:**
- `--cheap`: Sets `content.CheapMode = true`, overrides `GeneratorClient` with `CheapGeneratorClient` via BAML's `WithClient` option
- `make eval-cheap`: Shorthand for `--cheap --fast`
- The `/optimize` skill now uses `make eval-cheap` for all iteration cycles and runs a final `make eval-judges` for validation

**Files changed:**
- `api/internal/baml/baml_src/clients.baml`: Added `CheapGeneratorClient`
- `api/internal/content/generate.go`: Added `CheapMode` var and `generatorOpts()` helper
- `api/cmd/eval/main.go`: Added `--cheap` flag
- `Makefile`: Added `eval-cheap` target
- `.claude/skills/optimize/SKILL.md`: Updated to use cheap generation during iteration

## Research tools

- `scripts/discover-and-scrape.mjs`: Discovers accounts via Instagram search API, scrapes captions and post images. Supports `--niche=X`, `--mei-only` (500-100K followers), `--skip-google`, `--posts=N`. Session persists at `/tmp/ig-session.json`.
- `scripts/research-db.mjs`: Imports `docs/caption-research/data.json` into SQLite at `docs/caption-research/research.db`. Supports `--query="SQL"` and `--summary`. Computes engagement rate per post.
- `scripts/scrape-captions.mjs`: Original scraper for specific accounts. Handles Instagram login and session persistence.
- Database schema includes: char_count, word_count, likes_num, comments_num, engagement_rate, has_cta, cta_type, hashtag_count, has_line_breaks, has_price, has_question, emoji_count, opens_with_question, opens_with_exclamation, has_dash.

Not committed to git, not part of CI.

## Consequences

- Captions will be noticeably shorter (target 200-400c vs current 500-700c). Data shows this is the engagement sweet spot for MEI accounts.
- Hashtags become optional (0-3 relevant). Instagram capped at 5 in Dec 2025. Our data shows 0 performs best for MEI. Professional consensus says 3-5 specific ones help categorization. We target 0-3 as a compromise.
- CTAs shift from "always required" to "type matters." Platform-native CTAs (salva, manda, comenta) are encouraged because they drive saves/shares (45% of algorithm weight). Exit CTAs (link na bio, chama no zap) are discouraged except for explicit sales posts.
- Keyword SEO added to caption generation. Instagram reads caption text as search content. Niche keywords in the first two sentences boost discovery ~30% vs hashtag-heavy approaches.
- Em dashes are explicitly discouraged. Only 5% of real posts use them, but LLMs default to them heavily. This is a strong AI-tell.
- Production notes become simpler. Less prescriptive, more practical for MEIs who just have a phone.
- The research database (282 posts, 60 accounts, 11 niches) is available for future analysis. Engagement data enables data-driven decisions about content patterns.

## Sources

**Our data:** 282 posts scraped from 60 real Instagram accounts across 11 niches. 164 posts from 35 MEI-sized accounts (500-100K followers) with engagement rate analysis.

**Professional / industry:**
- [26 Mudancas do Algoritmo Instagram 2026, Rafael Terra](https://rafaelterra.com.br/guia-algoritmo-instagram-2026/) - Algorithm signal weights, hashtag/caption strategy
- [Como vender pelo Instagram 2026, Sebrae RN](https://blog.rn.sebrae.com.br/como-vender-pelo-instagram-2026/) - MEI-specific Instagram strategy
- [Instagram Hashtag Limit, Social Media Today](https://www.socialmediatoday.com/news/instagram-implements-new-limits-on-hashtag-use/808309/) - December 2025 cap at 5
- [Instagram Engagement Benchmarks 2026, Social Insider](https://www.socialinsider.io/social-media-benchmarks/instagram) - 35M posts analyzed, engagement by account size
- [Captions vs Hashtags 2026, Lamplight Creatives](https://lamplightcreatives.com/captions-vs-hashtags-instagram-2026/) - Keyword-rich captions vs hashtag-heavy posts
- [Algoritmo Instagram 2026, Informe Capixaba](https://www.informecapixaba.com.br/2026/03/03/novo-algoritmo-do-instagram-curtidas-valem-apenas-5-e-estrategia-muda-em-2026/) - Likes = 5%, watch time = 35%, saves = 25%
- [Instagram Engagement by Follower Count, CreatorFlow](https://creatorflow.so/blog/instagram-engagement-rate/) - Nano/micro accounts outperform larger accounts
- [Instagram Algorithm 2026, Buffer](https://buffer.com/resources/instagram-algorithms/) - Search-engine behavior, caption SEO
