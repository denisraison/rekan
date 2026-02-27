#!/usr/bin/env node
/**
 * Generate branded Instagram post images (1080x1350) using Playwright.
 *
 * Templates:
 *   overlay  — hook text on a darkened photo background
 *   card     — text on a solid/gradient background
 *
 * Usage:
 *   cd web && node ../scripts/create-post-image.mjs --config ../posts.json
 *   cd web && node ../scripts/create-post-image.mjs --config ../single.json
 *
 * JSON config is an array of post objects (or a single object).
 */

import { createRequire } from "node:module";
import { resolve, dirname, isAbsolute } from "node:path";
import { fileURLToPath } from "node:url";
import { readFileSync, existsSync } from "node:fs";

const __dirname = dirname(fileURLToPath(import.meta.url));
const require = createRequire(resolve(__dirname, "../web/") + "/");
const { chromium } = require("@playwright/test");

// -- Brand constants ----------------------------------------------------------

const BRAND = {
  coral: "#F97368",
  green: "#87AA8C",
  charcoal: "#44444A",
  offWhite: "#F5F2ED",
  darkBg: "#0a0a0c",
};

const WIDTH = 1080;
const HEIGHT = 1350;

// -- Helpers ------------------------------------------------------------------

function resolvePath(p) {
  if (isAbsolute(p)) return p;
  return resolve(process.cwd(), p);
}

function readLogoSvg() {
  const logoPath = resolve(__dirname, "../web/static/brand/logo-mark.svg");
  return readFileSync(logoPath, "utf-8");
}

function imageToDataUri(imagePath) {
  const resolved = resolvePath(imagePath);
  if (!existsSync(resolved)) {
    throw new Error(`Background image not found: ${resolved}`);
  }
  const buf = readFileSync(resolved);
  const ext = resolved.split(".").pop().toLowerCase();
  const mime = ext === "jpg" || ext === "jpeg" ? "image/jpeg" : `image/${ext}`;
  return `data:${mime};base64,${buf.toString("base64")}`;
}

/**
 * Wrap emphasis words in <em> tags. Handles overlapping/nested by processing
 * longest phrases first.
 */
function applyEmphasis(text, emphasisWords, color) {
  if (!emphasisWords || emphasisWords.length === 0) return escapeHtml(text);
  // Sort longest first to avoid partial matches
  const sorted = [...emphasisWords].sort((a, b) => b.length - a.length);
  // Replace each phrase with a placeholder, then swap placeholders for <em>
  const placeholders = [];
  let result = text;
  for (const phrase of sorted) {
    const idx = result.toLowerCase().indexOf(phrase.toLowerCase());
    if (idx === -1) continue;
    const original = result.slice(idx, idx + phrase.length);
    const ph = `\x00${placeholders.length}\x00`;
    placeholders.push(original);
    result = result.slice(0, idx) + ph + result.slice(idx + phrase.length);
  }
  result = escapeHtml(result);
  for (let i = 0; i < placeholders.length; i++) {
    result = result.replace(
      `\x00${i}\x00`,
      `<em style="color:${color};font-style:normal">${escapeHtml(placeholders[i])}</em>`,
    );
  }
  return result;
}

function escapeHtml(s) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/\n/g, "<br>");
}

function resolveColor(name) {
  if (!name) return BRAND.coral;
  if (name.startsWith("#")) return name;
  return BRAND[name] || BRAND.coral;
}

// -- Templates ----------------------------------------------------------------

function overlayTemplate(post, logoSvg) {
  const emphColor = resolveColor(post.emphasisColor);
  const hookHtml = applyEmphasis(post.hook, post.emphasis, emphColor);
  const bgImage = post.backgroundImage
    ? imageToDataUri(post.backgroundImage)
    : null;
  const bgColor = post.backgroundColor || BRAND.darkBg;
  const overlayOpacity = post.overlayOpacity ?? 0.6;

  return `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  @import url('https://fonts.googleapis.com/css2?family=Urbanist:wght@300;400;600;700;800&display=swap');

  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    width: ${WIDTH}px;
    height: ${HEIGHT}px;
    font-family: 'Urbanist', system-ui, sans-serif;
    overflow: hidden;
    position: relative;
    background: ${bgColor};
  }

  .bg-image {
    position: absolute;
    inset: 0;
    background: ${bgImage ? `url('${bgImage}') center/cover no-repeat` : bgColor};
  }

  .bg-overlay {
    position: absolute;
    inset: 0;
    background: rgba(0, 0, 0, ${overlayOpacity});
  }

  .content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 60px 64px;
  }

  .niche-tag {
    font-size: 22px;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: rgba(255, 255, 255, 0.6);
    margin-bottom: auto;
  }

  .hook {
    font-size: 54px;
    font-weight: 800;
    line-height: 1.15;
    color: #ffffff;
    margin-bottom: 28px;
    max-width: 95%;
  }

  .subtitle {
    font-size: 26px;
    font-weight: 400;
    color: rgba(255, 255, 255, 0.55);
    margin-bottom: auto;
    line-height: 1.4;
  }

  .bottom-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 40px;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .logo svg {
    width: 44px;
    height: auto;
  }

  .logo-text {
    font-size: 26px;
    font-weight: 300;
    letter-spacing: 0.05em;
    color: rgba(255, 255, 255, 0.7);
    text-transform: lowercase;
  }

  .cta {
    font-size: 22px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.6);
    letter-spacing: 0.02em;
  }
</style>
</head>
<body>
  <div class="bg-image"></div>
  <div class="bg-overlay"></div>
  <div class="content">
    ${post.nicheTag ? `<div class="niche-tag">${escapeHtml(post.nicheTag)}</div>` : '<div style="margin-bottom:auto"></div>'}
    <div class="hook">${hookHtml}</div>
    ${post.subtitle ? `<div class="subtitle">${escapeHtml(post.subtitle)}</div>` : ""}
    <div class="bottom-bar">
      <div class="logo">
        ${logoSvg}
        <span class="logo-text">rekan</span>
      </div>
      ${post.cta ? `<div class="cta">${escapeHtml(post.cta)}</div>` : ""}
    </div>
  </div>
</body>
</html>`;
}

function cardTemplate(post, logoSvg) {
  const emphColor = resolveColor(post.emphasisColor);
  const hookHtml = applyEmphasis(post.hook, post.emphasis, emphColor);
  const bgColor = post.backgroundColor || BRAND.offWhite;
  const textColor = isLightColor(bgColor) ? BRAND.charcoal : "#ffffff";
  const subtitleColor = isLightColor(bgColor)
    ? "rgba(68, 68, 74, 0.6)"
    : "rgba(255, 255, 255, 0.55)";
  const ctaColor = isLightColor(bgColor)
    ? "rgba(68, 68, 74, 0.5)"
    : "rgba(255, 255, 255, 0.6)";

  return `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  @import url('https://fonts.googleapis.com/css2?family=Urbanist:wght@300;400;600;700;800&display=swap');

  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    width: ${WIDTH}px;
    height: ${HEIGHT}px;
    font-family: 'Urbanist', system-ui, sans-serif;
    overflow: hidden;
    background: ${bgColor};
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 80px 72px;
  }

  .logo-mark {
    margin-bottom: 40px;
  }

  .logo-mark svg {
    width: 72px;
    height: auto;
  }

  .hook {
    font-size: 58px;
    font-weight: 800;
    line-height: 1.12;
    color: ${textColor};
    text-align: center;
    margin: auto 0;
    max-width: 100%;
  }

  .subtitle {
    font-size: 28px;
    font-weight: 400;
    color: ${subtitleColor};
    text-align: center;
    line-height: 1.4;
    margin-top: 36px;
    margin-bottom: auto;
  }

  .bottom-bar {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 14px;
    width: 100%;
    margin-top: 40px;
  }

  .cta {
    font-size: 22px;
    font-weight: 600;
    color: ${ctaColor};
    letter-spacing: 0.02em;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .whatsapp-icon {
    width: 28px;
    height: 28px;
    fill: ${ctaColor};
  }

  .brand-line {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .brand-line svg {
    width: 32px;
    height: auto;
  }

  .brand-line span {
    font-size: 22px;
    font-weight: 300;
    letter-spacing: 0.05em;
    color: ${ctaColor};
    text-transform: lowercase;
  }

  .divider {
    width: 120px;
    height: 3px;
    background: linear-gradient(90deg, ${BRAND.coral}, ${BRAND.green});
    border-radius: 2px;
    margin: 32px auto;
  }
</style>
</head>
<body>
  <div class="logo-mark">${logoSvg}</div>
  <div class="hook">${hookHtml}</div>
  ${post.subtitle ? `<div class="divider"></div><div class="subtitle">${escapeHtml(post.subtitle)}</div>` : ""}
  <div class="bottom-bar">
    ${post.cta ? `<div class="cta">${whatsappSvg(ctaColor)} ${escapeHtml(post.cta)}</div>` : `<div class="brand-line">${logoSvg}<span>rekan</span></div>`}
  </div>
</body>
</html>`;
}

function whatsappSvg(color) {
  return `<svg class="whatsapp-icon" viewBox="0 0 24 24" fill="${color}"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/></svg>`;
}

function customTemplate(post, logoSvg) {
  // Read HTML from an external file — full design control per post,
  // with brand fonts and CSS variables pre-loaded.
  const htmlPath = resolvePath(post.htmlFile);
  if (!existsSync(htmlPath)) {
    throw new Error(`HTML file not found: ${htmlPath}`);
  }
  let body = readFileSync(htmlPath, "utf-8");
  // Replace {{logo}} placeholder with the actual SVG
  body = body.replace(/\{\{logo\}\}/g, logoSvg);
  // Replace {{backgroundImage}} with base64 data URI
  if (post.backgroundImage) {
    const dataUri = imageToDataUri(post.backgroundImage);
    body = body.replace(/\{\{backgroundImage\}\}/g, dataUri);
  }

  return `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  @import url('https://fonts.googleapis.com/css2?family=Urbanist:wght@300;400;600;700;800&display=swap');

  :root {
    --coral: ${BRAND.coral};
    --green: ${BRAND.green};
    --charcoal: ${BRAND.charcoal};
    --off-white: ${BRAND.offWhite};
  }

  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    width: ${WIDTH}px;
    height: ${HEIGHT}px;
    font-family: 'Urbanist', system-ui, sans-serif;
    overflow: hidden;
  }
</style>
</head>
<body>
${body}
</body>
</html>`;
}

function isLightColor(hex) {
  const c = hex.replace("#", "");
  if (c.length < 6) return true;
  const r = parseInt(c.slice(0, 2), 16);
  const g = parseInt(c.slice(2, 4), 16);
  const b = parseInt(c.slice(4, 6), 16);
  // Relative luminance
  return r * 0.299 + g * 0.587 + b * 0.114 > 140;
}

// -- Main ---------------------------------------------------------------------

const TEMPLATES = {
  overlay: overlayTemplate,
  card: cardTemplate,
  custom: customTemplate,
};

async function generateImage(browser, post, logoSvg) {
  const templateFn = TEMPLATES[post.type];
  if (!templateFn) {
    throw new Error(
      `Unknown template type: ${post.type}. Use "overlay" or "card".`,
    );
  }

  const html = templateFn(post, logoSvg);
  const page = await browser.newPage();
  await page.setViewportSize({ width: WIDTH, height: HEIGHT });
  await page.setContent(html, { waitUntil: "networkidle" });

  // Wait for fonts
  await page.evaluate(() => document.fonts.ready);

  const output = resolvePath(post.output);
  await page.screenshot({ path: output, type: "png" });
  await page.close();

  console.log(`Saved: ${output}`);
}

async function main() {
  const args = process.argv.slice(2);
  const configIdx = args.indexOf("--config");
  if (configIdx === -1 || !args[configIdx + 1]) {
    console.error("Usage: create-post-image.mjs --config <posts.json>");
    process.exit(1);
  }

  const configPath = resolvePath(args[configIdx + 1]);
  const raw = readFileSync(configPath, "utf-8");
  const config = JSON.parse(raw);
  const posts = Array.isArray(config) ? config : [config];

  const logoSvg = readLogoSvg();

  const browser = await chromium.launch({ headless: true });
  try {
    for (const post of posts) {
      await generateImage(browser, post, logoSvg);
    }
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
