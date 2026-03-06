---
name: rekan-post
description: Create Instagram posts for Rekan's own account (@chamaorekan) with AI-generated images. Generates content using Rekan's BAML prompts, creates image candidates with Gemini Flash, evaluates them, refines the best with Gemini Pro. Use when user says 'create a rekan post', 'post for rekan', 'rekan instagram', 'social media post', 'create a post for us', 'instagram content for rekan'.
---

# Rekan Post Creator

Generate Instagram posts for Rekan (@chamaorekan): content from the BAML pipeline, plus AI-generated images.

## Brand Context

- **Positioning:** "seu parceiro de conteudo" (WhatsApp-first content partner for Brazilian MEIs)
- **Not** a SaaS app, not a tool. A partner that lives in WhatsApp.
- **Logo font:** Urbanist, weight 300, letter-spacing 0.05em, all lowercase
- **Colors:** coral #F97368 / rgb(249,115,104), sage green #87AA8C / rgb(135,170,140), dark charcoal #44444A, off-white #F5F2ED
- **Logo:** `web/static/brand/logo-mark.svg` (coral + sage green leaf/flame shapes)
- **Pricing:** Three tiers: Basico R$69,90, Parceiro ~~R$149,90~~ R$108,90 (founder discount), Profissional R$249,90. 30-day money-back guarantee. No free trial.
- **Voice:** founder-led, empathetic, direct. Speaks to MEIs as a partner, not a brand selling a product.

## Prerequisites

```bash
[[ -n "${GEMINI_API_KEY:-}" ]] && echo "GEMINI_API_KEY set" || echo "GEMINI_API_KEY not set"
[[ -n "${CLAUDE_API_KEY:-}" ]] && echo "CLAUDE_API_KEY set" || echo "CLAUDE_API_KEY not set"
```

Both required. Export from `.env`:
```bash
export $(grep -v '^#' /home/denis/workspace/rekan/.env | xargs)
```

## Pipeline

### Stage 1: Content Generation

```bash
cd /home/denis/workspace/rekan/eval && go run ./cmd/eval --rekan --verbose 2>&1
```

Find the latest run JSON:
```bash
ls -t /home/denis/workspace/rekan/eval/runs/*.json | head -1
```

Extract the 3 posts from `results[0].posts[]` (caption, hashtags, productionNote). Present to user, ask which to generate images for.

### Stage 2: Prompt Engineering

Translate productionNote into image prompts. Two types of images:

**Raw photos** (for story/empathy posts):

The key lesson: describe specific imperfections as PRIMARY SUBJECTS, not as modifiers. "A phone with a cracked screen protector on a greasy counter" works. "An authentic-feeling phone photo" does not.

| Note type | Prompt approach |
|---|---|
| "screenshot", "tela", "print" | Phone lying on a surface showing the screen. Describe the surface (greasy counter, flour-dusted table). Add context objects (wrench, coffee, napkin). |
| "selfie", "pessoa" | DO NOT generate people. Use overhead workspace shot, flat lay, or graphic card instead. |
| "flat lay", "de cima" | Overhead scene with specific imperfections: chipped plates, crossed-out handwriting, bitten food, coffee rings, scattered crumbs. |
| "notebook", "trabalhando" | Real desk chaos: stickered laptop, tangled cables, sticky notes, loose coins, half-eaten snack. Specify the light source (desk lamp, laptop glow, window blinds). |

For raw photos, every prompt must include 3+ specific imperfections. Examples:
- Cracked screen protector, oil smudges, dirty rag
- Coffee dregs in mug, crumpled napkin, loose coins, tangled charger
- Flour scattered unevenly, chipped ceramic plate, bitten-open food
- Peeling water bottle label, scratched desk, visible cables

Avoid: "authentic feel", "natural", "casual" as generic modifiers. These don't work. Describe the actual mess.

**Branded graphics** (for value/info/CTA posts):

- Logo at `web/static/brand/logo-mark.svg` (convert to PNG: `magick -background none web/static/brand/logo-mark.svg -resize 512x512 /tmp/rekan-logo.png`)
- Pass logo via `--input-image` to maintain brand consistency
- Text: all lowercase for brand name, Urbanist Light style (describe as "thin weight, wide letter-spacing, modern geometric sans-serif")
- Background: off-white with paper texture OR sage green solid
- Always mention WhatsApp in CTAs, never "app" or "tool"
- Pricing: "A partir de R$69,90/mes" (Basico), "Parceiro por R$108,90/mes" (founder discount), "Profissional R$249,90/mes". "Garantia de 30 dias."

Generate 4 prompt variations per post, present to user before generating.

### Stage 3: Image Compositing

All grid images are rendered locally via Playwright. Never use Gemini for text rendering.

```bash
cd /home/denis/workspace/rekan/web && node ../scripts/create-post-image.mjs --config /path/to/posts.json
```

**Two template types:**

| Template | Use for | Config |
|----------|---------|--------|
| `overlay` | Quick drafts, iterating on photo + hook text | JSON with hook, emphasis, backgroundImage |
| `custom` | Final posts, any post needing unique design | Standalone HTML file |

Use `overlay` to quickly test hook text on a photo. Use `custom` for everything final. Every final grid post should go through `/frontend-design` for unique visual treatment.

**Overlay config:**
```json
{
  "type": "overlay",
  "hook": "Hook text",
  "emphasis": ["words"],
  "emphasisColor": "coral",
  "backgroundImage": "path/to/photo.png",
  "overlayOpacity": 0.5,
  "output": "path/to/output.png"
}
```

**Custom config:**
```json
{
  "type": "custom",
  "htmlFile": "rekan-posts/{date}/templates/post-name.html",
  "backgroundImage": "path/to/photo.png",
  "output": "path/to/output.png"
}
```
Available placeholders in HTML: `{{logo}}` (SVG markup), `{{backgroundImage}}` (base64 data URI, requires `backgroundImage` in config). CSS variables: `--coral`, `--green`, `--charcoal`, `--off-white`. Font: Urbanist (300/400/600/700/800) auto-loaded.

Store templates in `rekan-posts/{date}/templates/`. See `rekan-posts/2026-02-24/templates/` for examples.

**Design principles (not rules):**

- Each post on the grid must look like it was designed individually, not stamped from a template. Vary background, layout, typography, and accent style across posts.
- The grid should have visual rhythm when viewed as a whole: alternate light/dark, photo/graphic, centered/left-aligned.
- Match the design energy to the message. A provocative tip needs different energy than an encouraging one. A CTA needs different energy than a BTS story.
- The logo SVG has inline fills (coral + green). On colored backgrounds use `!important` to override to monochrome or adjust opacity for the context.
- Gemini generates raw photos only. All text rendering goes through the local HTML pipeline for consistent fonts and brand colors.

**Workflow:** Use Gemini for atmospheric photos (Stage 4-6). All text compositing goes through custom HTML.

**Content principles:**

- **Message first, design second.** Validate the message before designing. Would a MEI understand this instantly, without marketing vocabulary? No "template", "agência", "content strategy". Write in words they use daily.
- **Hook must work with the image**, not against it. Never put dismissive text on someone's work.
- **All text in pt-BR, no exceptions.** Badge labels, CTAs, niche tags, everything.

### Stage 4: Flash Generation (screening only)

Flash images will be too polished. Use them only to screen compositions and concepts, not as final candidates.

```bash
SCRIPT="/home/denis/workspace/rekan/.claude/skills/rekan-post/scripts/generate-image.sh"
DATE=$(date +%Y-%m-%d)
OUTDIR="/home/denis/workspace/rekan/rekan-posts/${DATE}"
mkdir -p "${OUTDIR}/flash"

$SCRIPT --prompt "..." --model flash --output "${OUTDIR}/flash/post1-v1.png" &
$SCRIPT --prompt "..." --model flash --output "${OUTDIR}/flash/post1-v2.png" &
$SCRIPT --prompt "..." --model flash --output "${OUTDIR}/flash/post1-v3.png" &
$SCRIPT --prompt "..." --model flash --output "${OUTDIR}/flash/post1-v4.png" &
wait
```

### Stage 5: Evaluation

Read all images. Score on: Authenticity, Brand fit, Scene match, Scroll stop, Text quality (1-5 each).

**Disqualify** images with: garbled text, uncanny faces, plastic textures, visual artifacts, English text (watch for English product labels like "WHEAT FLOUR" on bags).

Be honest about AI slop. If every image looks too clean, too warm, too perfectly arranged, say so. The user will notice.

**Grid uniqueness check:** Before picking winners, compare against ALL existing grid photos. No two posts can share the same background photo or look similar as thumbnails. Each row of 3 must be visually distinct.

Pick top 2 compositions for Pro refinement.

### Stage 6: Pro Refinement

For raw photos, rewrite prompts from scratch with aggressive imperfection:
- Replace "warm lighting" with specific light source ("single bare-bulb desk lamp", "harsh overhead fluorescent")
- Replace "wooden table" with "scratched formica table with water rings"
- Add random real-life objects ("bag of bread", "dish rack", "charging cable")
- Add camera imperfections ("slight grain from low light", "slightly off-center framing")
- NEVER include notebooks or paper with handwritten text. AI always generates gibberish.
- Check for English text on product labels, packaging, or screens. Brazilian scenes should have Portuguese or no text.

For branded graphics, use `--input-image` with the logo PNG to maintain mark accuracy.

```bash
mkdir -p "${OUTDIR}/pro"
$SCRIPT --prompt "..." --model pro --output "${OUTDIR}/pro/post1-r1.png" --image-size 2K &
```

For fixing specific issues (e.g. wrong text on an otherwise good image), use image-to-image:
```bash
$SCRIPT --input-image "${OUTDIR}/pro/post1-r1.png" \
    --prompt "Fix ONLY the text on the notebook. Keep everything else identical. The text should read: [exact text]" \
    --model pro --output "${OUTDIR}/pro/post1-r2.png" --image-size 2K
```

### Stage 7: User Review

Show each final candidate. Ask user to pick or request changes. Iterate as needed.

Copy winners to final:
```bash
mkdir -p "${OUTDIR}/final"
cp "${OUTDIR}/pro/post1-r1.png" "${OUTDIR}/final/post1.png"
```

### Stage 8: Post Summary

Write a markdown file per post with caption, hashtags, and image reference ready to copy-paste.

## Output Structure

```
rekan-posts/{date}/
    flash/          # screening candidates
    pro/            # refined versions
    final/          # winners + markdown summaries
```

## Aspect Ratios

Default 4:5 (Instagram feed portrait). Ask user if not specified.

| Format | Ratio |
|---|---|
| Feed portrait | 4:5 |
| Feed square | 1:1 |
| Story/Reel | 9:16 |

## Checklist Before Generating Branded Graphics

- [ ] Logo converted to PNG at /tmp/rekan-logo.png
- [ ] Brand name in lowercase ("rekan" not "Rekan")
- [ ] WhatsApp mentioned in CTAs (not "link na bio", not "app")
- [ ] Pricing correct (Basico R$69,90, Parceiro R$108,90, Profissional R$249,90, garantia 30 dias)
- [ ] No English text anywhere
- [ ] "parceiro" language, not "tool" or "app" language
