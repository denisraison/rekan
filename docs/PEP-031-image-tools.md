# PEP-031: Image generation tools

**Status:** Draft
**Date:** 2026-03-15
**Depends on:** PEP-030

## Context

The current image pipeline lives outside the app. `scripts/create-post-image.mjs` renders HTML to PNG via Playwright. `.claude/skills/rekan-post/scripts/generate-image.sh` calls Gemini for image generation. The eval-layout skill screenshots the app UI. All orchestrated by Claude Code skills, not by the agent.

The operator workflow today: ask the agent to generate a post (text only), then a human uses Claude Code to run the image skill manually. Two disconnected steps. The operator can't say "gera um post com imagem pra Patricia" and get back a complete post with image in WhatsApp.

PEP-030 gives us the tool loop. Image generation is three tools plugged into it. The agent iterates on image quality the same way it iterates on any tool output: call, evaluate, adjust, repeat.

### What we already have

- **Playwright** in the Nix flake. Chromium available at runtime.
- **`create-post-image.mjs`** with three templates (overlay, card, custom). Brand constants (colors, logo SVG, Urbanist font).
- **Gemini image generation** via `generate-image.sh`. Two models: Flash (fast/cheap screening) and Pro (refinement).
- **Gemini vision** in `api/internal/transcribe/gemini.go`. Already describes images. Can be extended to evaluate them.
- **Brand assets**: logo at `web/static/brand/logo-mark.svg`, colors (#F97368 coral, #87AA8C sage, #44444A charcoal, #F5F2ED off-white), Urbanist font.

### Why no sandbox

The "code" is HTML. Chromium renders it in its own process sandbox (namespaces, seccomp, site isolation). The HTML can't touch the filesystem, network, or host processes. We're not executing arbitrary code. We're rendering a template.

## Design

Three tools. The agent's tool loop handles iteration.

### Tool: `render_image`

Takes HTML string, renders it to PNG via chromedp (Go library, Chrome DevTools Protocol). No Node, no shell out.

```go
Tool{
    Name:        "render_image",
    Description: "Renderiza HTML em imagem PNG (1080x1350 para feed, 1080x1080 quadrado, 1080x1920 story).",
    InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "html":   {"type": "string", "description": "HTML completo para renderizar"},
            "width":  {"type": "integer", "description": "Largura em pixels (padrão: 1080)"},
            "height": {"type": "integer", "description": "Altura em pixels (padrão: 1350)"}
        },
        "required": ["html"]
    }`),
    Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
        // parse args, default 1080x1350
        // chromedp: navigate to data URI, screenshot
        // write PNG to temp dir
        // return file path
    },
}
```

chromedp manages Chromium's lifecycle. One browser instance reused across calls. ~20 lines of rendering logic:

```go
func (r *Renderer) Screenshot(ctx context.Context, html string, w, h int) (string, error) {
    ctx, cancel := chromedp.NewContext(r.allocCtx)
    defer cancel()

    var buf []byte
    if err := chromedp.Run(ctx,
        chromedp.EmulateViewport(int64(w), int64(h)),
        chromedp.Navigate("data:text/html;charset=utf-8,"+url.PathEscape(html)),
        chromedp.FullScreenshot(&buf, 100),
    ); err != nil {
        return "", err
    }

    path := filepath.Join(r.tmpDir, fmt.Sprintf("render-%d.png", time.Now().UnixNano()))
    if err := os.WriteFile(path, buf, 0644); err != nil {
        return "", err
    }
    return path, nil
}
```

The HTML has access to brand constants injected as CSS variables and base64-encoded logo/background images, same as `create-post-image.mjs` does today.

### Tool: `evaluate_image`

Sends a PNG to Gemini vision, asks for a structured score. The agent uses the feedback to decide whether to iterate.

```go
Tool{
    Name:        "evaluate_image",
    Description: "Avalia qualidade de uma imagem gerada. Retorna nota (1-10) e feedback.",
    InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "image_path": {"type": "string", "description": "Caminho do PNG"},
            "criteria":   {"type": "string", "description": "O que avaliar (composição, legibilidade, marca)"}
        },
        "required": ["image_path"]
    }`),
    Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
        // read PNG, base64 encode
        // call Gemini vision with evaluation prompt
        // return structured score + feedback
    },
}
```

Uses `gemini-3.1-flash-lite-preview` (same model as transcription, cheap). The evaluation prompt asks for: readability (text size, contrast), brand consistency (colors, logo placement), composition (balance, whitespace), and overall score.

### Tool: `generate_image`

Calls Gemini image generation API. For cases where the agent needs a photo or illustration, not an HTML render.

```go
Tool{
    Name:        "generate_image",
    Description: "Gera imagem com IA a partir de um prompt. Para fotos e ilustrações, não para gráficos com texto.",
    InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "prompt":       {"type": "string", "description": "Descrição da imagem desejada"},
            "aspect_ratio": {"type": "string", "enum": ["4:5", "1:1", "9:16"], "description": "Proporção (padrão: 4:5)"},
            "input_image":  {"type": "string", "description": "Caminho de imagem de referência para edição (opcional)"}
        },
        "required": ["prompt"]
    }`),
    Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
        // call Gemini image gen API (Flash for first pass)
        // write result to temp dir
        // return file path
    },
}
```

Two models: `gemini-3.1-flash-image-preview` for drafts, `gemini-3-pro-image-preview` for final refinement. The tool uses Flash by default. The agent can call it again with `input_image` pointing to a previous result for refinement.

### Agent flow

The agent decides the strategy. No hardcoded pipeline. Examples:

**Branded graphic (text on background):**
```
Operator: "gera post pra Patricia sobre promoção de inverno"
Agent:
  1. generate_post → caption, hashtags, production note
  2. render_image → HTML with hook text, brand colors, logo
  3. evaluate_image → "7/10, text too small on mobile"
  4. render_image → adjusted HTML with larger text
  5. evaluate_image → "9/10"
  6. Reply with caption + image path
```

**Photo-based post:**
```
Operator: "gera post pra confeitaria da Maria com foto de bolo"
Agent:
  1. generate_post → caption
  2. generate_image → "artisan cake, warm lighting, bakery counter"
  3. evaluate_image → "8/10, looks natural"
  4. Reply with caption + image path
```

**Composite (photo + text overlay):**
```
Agent:
  1. generate_post → caption, hook
  2. generate_image → background photo
  3. render_image → HTML overlay with hook text on generated photo
  4. evaluate_image → feedback
  5. iterate or done
```

The loop handles all three. The agent picks the tools based on the production note and business type.

### Sending images via WhatsApp

The `WAClient` interface already has `SendImage` in `reply.go`. After the agent settles on an image, it's sent alongside the post text. The `approve_post` tool already sends content to clients. Extend it to attach the image if one exists.

### Chromium lifecycle

One browser allocator created on app start (alongside WhatsApp client, PocketBase, etc.). Reused across all `render_image` calls. Shut down on app termination.

```go
// In main.go setup
allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
    chromedp.Flag("headless", true),
    chromedp.Flag("disable-gpu", true),
    chromedp.Flag("no-sandbox", true), // already inside Nix sandbox
)
renderer := &Renderer{allocCtx: allocCtx, tmpDir: os.TempDir()}
// Pass renderer to tool constructors
```

### Brand injection

The HTML template gets brand constants as CSS variables, same pattern as `create-post-image.mjs`:

```css
:root {
    --coral: #F97368;
    --green: #87AA8C;
    --charcoal: #44444A;
    --off-white: #F5F2ED;
}
```

Logo SVG and background images are base64-encoded and injected as data URIs. The Urbanist font is loaded from the local filesystem (already in the Nix flake).

The agent generates the full HTML. It knows the brand system from the system prompt. No separate template engine.

## Waves

### Wave 1: render_image + evaluate_image

The core loop: generate HTML, screenshot, evaluate, iterate.

**Files:**
- `api/internal/agent/renderer.go` — `Renderer` struct, chromedp screenshot logic, brand CSS injection, temp file management
- `api/internal/agent/tools_image.go` — `renderImageTool()` and `evaluateImageTool()` constructors returning `Tool` values
- `api/internal/agent/renderer_test.go` — test that renders a simple HTML string to PNG, verifies file exists and is valid PNG

**Gate:**
- [ ] `render_image` tool produces a valid PNG from HTML input
- [ ] `evaluate_image` tool returns structured score + feedback from Gemini
- [ ] Agent can loop: render → evaluate → render again with adjustments
- [ ] Chromium starts once, reused across calls

### Wave 2: generate_image + WhatsApp delivery

Add Gemini image generation and wire images into the WhatsApp reply flow.

**Files:**
- `api/internal/agent/tools_image.go` — add `generateImageTool()` constructor
- `api/internal/agent/gemini.go` — Gemini image generation API client (reuse auth pattern from `transcribe/gemini.go`)
- `api/internal/agent/tools.go` — add image tools to `buildTools()`, gated on renderer being non-nil
- `api/internal/agent/reply.go` — extend reply flow to send image when present

**Gate:**
- [ ] `generate_image` tool produces a valid PNG from a text prompt
- [ ] Agent can combine: generate_post → render_image or generate_image → evaluate → reply
- [ ] Image is sent via WhatsApp alongside post text
- [ ] Tools are only registered when Chromium and Gemini API key are available

### Wave 3: Production hardening

Temp file cleanup, image size limits, timeout per render, cost tracking.

**Files:**
- `api/internal/agent/renderer.go` — add render timeout (10s), max HTML size (100KB), temp file cleanup on context cancellation
- `api/internal/agent/gemini.go` — add cost tracking (tokens/images per call) to traces
- `api/internal/agent/tools_image.go` — add max iterations hint in tool description so agent doesn't loop forever

**Gate:**
- [ ] Temp files cleaned up after WhatsApp delivery
- [ ] Render timeout prevents hung Chromium from blocking the agent
- [ ] Image generation costs visible in traces
- [ ] No leaked temp files after 24h of operation

## Consequences

- The operator can request complete posts (text + image) in one WhatsApp message. No manual Claude Code step.
- Image iteration is free. The agent's tool loop already handles retry logic. No new orchestration code.
- chromedp adds a dependency but it's pure Go (no CGo). Chromium is already in the Nix flake for Playwright.
- No sandbox beyond Chromium's own. The agent generates HTML, not arbitrary code. If we later need to run arbitrary scripts, that's a different problem.
- Gemini image generation costs ~$0.02 per image (Flash) or ~$0.05 (Pro). A typical post with 2-3 iterations costs ~$0.10. Worth tracking.
- The `generate-image.sh` and `create-post-image.mjs` scripts become redundant once this is stable. They stay for the Claude Code skill workflow until we're confident the agent does it better.
- Brand consistency depends on the agent knowing the brand system. This goes in the system prompt, not in code. If the agent generates ugly HTML, the evaluate_image loop catches it.
