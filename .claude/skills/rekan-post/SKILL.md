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
- **Pricing:** R$19,90 first month, R$108,90/month after. No free trial.
- **Voice:** founder-led, empathetic, direct. Speaks to MEIs as a partner, not a brand selling a product.

## Prerequisites

```bash
[[ -n "${GEMINI_API_KEY:-}" ]] && echo "GEMINI_API_KEY set" || echo "GEMINI_API_KEY not set"
[[ -n "${OPENROUTER_API_KEY:-}" ]] && echo "OPENROUTER_API_KEY set" || echo "OPENROUTER_API_KEY not set"
```

Both required. OPENROUTER_API_KEY must be exported from `.env`:
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
- Pricing: "Primeiro mes por R$19,90" (trial), "R$108,90/mes" (regular)

Generate 4 prompt variations per post, present to user before generating.

### Stage 3: Flash Generation (screening only)

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

### Stage 4: Evaluation

Read all images. Score on: Authenticity, Brand fit, Scene match, Scroll stop, Text quality (1-5 each).

**Disqualify** images with: garbled text, uncanny faces, plastic textures, visual artifacts, English text.

Be honest about AI slop. If every image looks too clean, too warm, too perfectly arranged, say so. The user will notice.

Pick top 2 compositions for Pro refinement.

### Stage 5: Pro Refinement

For raw photos, rewrite prompts from scratch with aggressive imperfection:
- Replace "warm lighting" with specific light source ("single bare-bulb desk lamp", "harsh overhead fluorescent")
- Replace "wooden table" with "scratched formica table with water rings"
- Add random real-life objects ("bag of bread", "dish rack", "charging cable")
- Add camera imperfections ("slight grain from low light", "slightly off-center framing")

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

### Stage 6: User Review

Show each final candidate. Ask user to pick or request changes. Iterate as needed.

Copy winners to final:
```bash
mkdir -p "${OUTDIR}/final"
cp "${OUTDIR}/pro/post1-r1.png" "${OUTDIR}/final/post1.png"
```

### Stage 7: Post Summary

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
- [ ] Pricing correct (R$19,90 first month, R$108,90/month)
- [ ] No English text anywhere
- [ ] "parceiro" language, not "tool" or "app" language
