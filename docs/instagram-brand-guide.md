# Rekan Instagram Brand Guide

Decisions made during the initial brand and content strategy session (Feb 2026).

## Account

- **Username:** @chamaorekan
- **Account type:** Instagram Business
- **WhatsApp link:** bit.ly/chamaorekan
- **Bio:**
  ```
  rekan | seu parceiro de conteúdo
  Sem tempo pra pensar em legenda?
  Chama no WhatsApp, manda o que fez hoje.
  A partir de R$69,90/mês 👇
  ```

## Positioning

Rekan is a **partner**, not a tool, not an app, not a platform. Everything we say should reflect that.

- Always say "parceiro de conteudo", never "ferramenta" or "app"
- Always mention WhatsApp as the delivery channel
- Never position as self-serve SaaS (we're not GalilAI)
- The founder voice speaks FROM empathy, not AT the audience
- We understand MEI life because we live next to it (amigo padeiro, vizinha que tem salao, tio que faz churrasco pra fora)

## Visual Identity

### Brand elements
- **Logo:** `web/static/brand/logo-mark.svg` (coral + sage green leaf/flame intertwined)
- **Font:** Urbanist, weight 300, letter-spacing 0.05em, text-transform: lowercase
- **Colors:**
  - Coral: #F97368 / rgb(249,115,104)
  - Sage green: #87AA8C / rgb(135,170,140)
  - Dark charcoal: #44444A
  - Off-white: #F5F2ED

### Grid pattern: hybrid checkerboard
Alternating branded graphics and raw photos. When someone lands on the profile, they see both personality (raw) and credibility (branded).

```
GRAPHIC    RAW        GRAPHIC
RAW        GRAPHIC    RAW
GRAPHIC    RAW        GRAPHIC
```

### Raw photos (story/empathy posts)
- Scenes from real MEI life: kitchens, workshops, desks, counters
- No photorealistic people (AI faces look uncanny)
- Imperfection is the goal: cracked screens, coffee rings, flour dust, tangled cables, scratched surfaces
- Lighting should feel real: desk lamp, fluorescent tubes, window blinds. Not warm studio lighting.
- These prove "I know your world"

### Branded graphics (value/info/CTA posts)
- Off-white or sage green backgrounds with subtle texture
- Logo mark small, centered at top
- Typography: thin, wide-spaced, lowercase for brand name. Bold for key phrases.
- Kraft paper texture works well for tip cards
- WhatsApp icon in CTAs
- These deliver value and direct action

### What to avoid
- Stock photo vibes (perfect lighting, smiling people, clean desks)
- Canva template energy (alternating color blocks with buzzwords)
- English text anywhere
- Marketing jargon (CTA, copywriting, engagement, digital presence, brand voice, content planning)
- Generic motivational phrases ("Create. Connect. Grow.")
- The word "app" or "tool" or "plataforma"

## Pricing in Content

Three tiers: Basico (R$69,90), Parceiro (R$108,90, founder discount from R$149,90), Profissional (R$249,90).

- Lead with "a partir de R$69,90" or "Parceiro por R$108,90/mês"
- Always mention "garantia de 30 dias" and "sem contrato"
- Anchor against social media managers: "10x mais barato que um social media"
- Frame in daily terms: "menos de R$4 por dia"
- Frame in MEI terms: "menos que um lanche no iFood", "menos que um almoço executivo"
- Never say "free trial" (there isn't one, and "free" signals low quality in Brazil)
- The 30-day money-back guarantee is the trust mechanism, not a discounted first month

## Content Pillars

1. **Founder stories** (raw photos): real conversations with MEIs, building the product late at night, empathy moments
2. **Value tips** (branded cards): practical Instagram advice MEIs can use without Rekan ("poste o video real", "nao copie agencia grande")
3. **How it works** (branded infographic): the WhatsApp flow (manda mensagem > recebe post pronto > copia e posta)
4. **MEI reality** (raw photos): scenes from workshops, kitchens, salons. The texture of small business life.
5. **Social proof** (raw or branded): concrete, visual, no metrics dashboards
   - Before/after grid screenshots: the client's Instagram profile 1 month ago vs. now. The difference is visible without any analytics.
   - Client voice: "recebi 3 encomendas pelo Instagram essa semana", "minha cliente viu meu stories e mandou mensagem". Always stories, never statistics.
   - Posting frequency: "A Claudia postava 1 vez por mes. Agora posta 2 vezes por semana e nao pensa em legenda."
   - WhatsApp screenshots of happy clients (with permission)
6. **Vergonha/relief** (raw photos): the guilt of not posting. Scenes that hit the emotional nerve: empty caption field at 11pm, Instagram grid with 3-week gaps, the competitor who posts every day. These self-select for MEIs who feel the problem. End with relief, not a sales pitch.
7. **CTA** (branded card): pricing, "chama no WhatsApp", low-pressure invitation

### Rules from the BAML prompt (GenerateRekanContent)
- 2 of 3 posts must deliver standalone value WITHOUT mentioning the product
- Each post mentions a specific named person with a non-digital detail ("A Dona Cida faz 200 coxinhas por dia e ainda cuida da neta")
- Never end with generic engagement questions ("qual seu favorito?", "marca um amigo")
- Open with a concrete micro-moment, not a generic declaration
- Vary structure: one narrative, one direct, one list. Never the same skeleton.

### Numbers in content
Numbers that work for MEIs (concrete, felt, need no dashboard):
- Output: "2 posts por semana sem voce pensar em nada"
- Time: "Em 2 minutos voce tem o post pronto"
- Price: "R$108,90/mes", "menos de R$4 por dia", "menos que um almoco executivo por semana"
- Anchor: "10x mais barato que um social media", "garantia de 30 dias"

Numbers that do NOT work (require a baseline, feel like marketing jargon):
- "Aumente seu engajamento em 40%"
- "Nossos clientes ganharam X seguidores"
- "ROI de Y%"
- Any metric that assumes the MEI tracks analytics

## Image Generation Lessons

### The anti-slop approach
Generic prompts ("authentic feel", "natural lighting", "casual photo") produce polished AI images that look like stock photos. The fix: describe specific imperfections as primary subjects.

**Bad:** "A workspace photo with natural lighting and authentic feel"
**Good:** "A scratched silver MacBook with stickers on the lid, tangled charger cable, loose coins, crumpled napkin with coffee stain, dark kitchen cabinets in background, single bare-bulb desk lamp. Slight grain from low light phone camera."

### Generation workflow
1. Generate 4 candidates with Gemini Flash (screening only, these will be too polished)
2. Pick best 2 compositions
3. Rewrite prompts with aggressive imperfection, generate with Gemini Pro at 2K
4. For text fixes on otherwise good images, use image-to-image with Pro
5. For branded graphics, pass logo as --input-image to maintain mark accuracy

### What works
- Beat-up Android phones with cracked screen protectors
- Formica tables, not wooden farmhouse tables
- Fluorescent lighting, not warm ambient
- Cheap glass coffee cups, not ceramic mugs
- Sticky notes on walls with handwritten Portuguese
- Laptop stickers (GitHub, Linux, etc.)
- Half-eaten pao de queijo on a napkin

### What doesn't work
- Handwritten text (AI generates plausible-looking but incorrect words)
- People's faces (uncanny valley)
- "Warm, cozy" scenes (too styled, too perfect)
- Minimal abstract designs (too bare, no story)

## Posting Schedule

- 2x per week (Mon, Thu) works for the hybrid grid
- Best times for Brazilian MEI audience: 11h-13h or 19h-21h (BRT)
- Hashtags go in the first comment, not the caption (cleaner look, same reach)
- Initial grid: seed 3 posts day 1, then 1 per day for 6 days

## Future Considerations

- TikTok: not now. Requires video production pipeline. Revisit at 15-20 clients when Elenice can record short selfie videos about MEI stories.
- Highlight covers: Como funciona, Dicas, Bastidores, MEIs (sage green / coral / charcoal backgrounds with simple icons)
- Stories: repurpose raw photos with text overlay, behind-the-scenes of using the WhatsApp flow
