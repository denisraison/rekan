#!/usr/bin/env node
/**
 * Scrape Instagram captions + post images from real accounts.
 *
 * Session is cached at /tmp/ig-session.json so login only happens once.
 * Post screenshots are saved to docs/caption-research/ for visual analysis.
 *
 * Usage:
 *   cd web && set -a && source ../.env && node ../scripts/scrape-captions.mjs
 *   cd web && set -a && source ../.env && node ../scripts/scrape-captions.mjs --headed
 *   cd web && set -a && source ../.env && node ../scripts/scrape-captions.mjs --accounts=magazineluiza,nubank
 *   cd web && set -a && source ../.env && node ../scripts/scrape-captions.mjs --posts=3
 *   cd web && node ../scripts/scrape-captions.mjs --relogin   # force fresh login
 */

import { createRequire } from "node:module";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { writeFileSync, existsSync, mkdirSync } from "node:fs";

const __dirname = dirname(fileURLToPath(import.meta.url));
const require = createRequire(resolve(__dirname, "../web/") + "/");
const { chromium } = require("@playwright/test");

const SESSION_PATH = "/tmp/ig-session.json";
const OUT_DIR = resolve(__dirname, "../docs/caption-research");

// -- Accounts ---------------------------------------------------------------

const ACCOUNT_GROUPS = {
  "big-brands": [
    "magazineluiza",
    "nubank",
    "oboticario",
    "havaianas",
  ],
  "confeitaria-mei": [
    "confeitariaanacristina",
    "brunarebelo",
    "bolosdacakau",
  ],
  "nails-mei": [
    "natiakemioficial",
    "manicuresinceraoficial",
  ],
  "hair-mei": [
    "institutoembelleze",
  ],
  "food-mei": [
    "isaaborges.fit",
  ],
};

// -- CLI args ---------------------------------------------------------------

const args = process.argv.slice(2);
const headed = args.includes("--headed");
const relogin = args.includes("--relogin");
const accountsArg = args.find((a) => a.startsWith("--accounts="))?.split("=")[1];
const postsPerAccount = parseInt(
  args.find((a) => a.startsWith("--posts="))?.split("=")[1] || "6"
);

let accountList;
if (accountsArg) {
  accountList = accountsArg
    .split(",")
    .map((a) => ({ username: a.trim(), group: "custom" }));
} else {
  accountList = Object.entries(ACCOUNT_GROUPS).flatMap(([group, accounts]) =>
    accounts.map((username) => ({ username, group }))
  );
}

// -- Browser + session ------------------------------------------------------

const browser = await chromium.launch({
  headless: !headed,
  args: ["--disable-blink-features=AutomationControlled"],
});

const hasSession = !relogin && existsSync(SESSION_PATH);

const context = await browser.newContext({
  locale: "pt-BR",
  userAgent:
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
  viewport: { width: 1280, height: 900 },
  ...(hasSession ? { storageState: SESSION_PATH } : {}),
});

const page = await context.newPage();

// -- Login (skipped if session exists) --------------------------------------

if (hasSession) {
  process.stderr.write("Using saved session... ");
  await page.goto("https://www.instagram.com/", {
    waitUntil: "domcontentloaded",
    timeout: 15000,
  });
  await page.waitForTimeout(2000);

  const stillLoggedIn = await page.evaluate(
    () => !document.querySelector('input[name="email"], input[name="username"]')
  );

  if (stillLoggedIn) {
    process.stderr.write("OK\n");
  } else {
    process.stderr.write("expired, logging in again...\n");
    await doLogin(page);
  }
} else {
  await doLogin(page);
}

// Dismiss any popups (notifications, save login)
for (const text of ["Agora não", "Not Now", "Ahora no"]) {
  const btn = page.locator(`button:has-text("${text}")`);
  if (await btn.first().isVisible({ timeout: 1500 }).catch(() => false)) {
    await btn.first().click();
    await page.waitForTimeout(800);
  }
}

async function doLogin(page) {
  const user = process.env.INSTAGRAM_USER;
  const pass = process.env.INSTAGRAM_PASS;
  if (!user || !pass) {
    console.error("No session found. Set INSTAGRAM_USER and INSTAGRAM_PASS env vars.");
    await browser.close();
    process.exit(1);
  }

  process.stderr.write("Logging in... ");
  await page.goto("https://www.instagram.com/accounts/login/", {
    waitUntil: "networkidle",
    timeout: 20000,
  });

  // Cookie consent
  for (const t of [
    "Permitir todos os cookies",
    "Allow all cookies",
    "Allow essential and optional cookies",
    "Aceitar",
  ]) {
    const btn = page.locator(`button:has-text("${t}")`);
    if (await btn.first().isVisible({ timeout: 1500 }).catch(() => false)) {
      await btn.first().click();
      await page.waitForTimeout(1500);
      break;
    }
  }

  await page.waitForSelector('input[name="email"], input[type="text"]', {
    timeout: 10000,
  });
  await page.locator('input[name="email"], input[type="text"]').first().fill(user);
  await page
    .locator('input[name="pass"], input[type="password"]')
    .first()
    .fill(pass);
  await page.getByRole("button", { name: "Entrar", exact: true }).click();

  try {
    await page.waitForURL(/instagram\.com\/(?!accounts)/, { timeout: 20000 });
  } catch {
    await page.screenshot({ path: "/tmp/ig-login-debug.png" });
    process.stderr.write(
      "might need 2FA. Try --headed. Screenshot: /tmp/ig-login-debug.png\n"
    );
  }
  await page.waitForTimeout(2000);

  // Save session
  await context.storageState({ path: SESSION_PATH });
  process.stderr.write("OK (session saved)\n");
}

// -- Ensure output dir ------------------------------------------------------

mkdirSync(OUT_DIR, { recursive: true });

// -- Caption + image extraction helpers -------------------------------------

async function extractCaption(page) {
  return page.evaluate(() => {
    // Best source: og:description meta tag.
    // Format: `N likes, N comments - username no DATE: "CAPTION". `
    // The closing pattern is `". ` or `"` at end.
    const ogDesc = document.querySelector('meta[property="og:description"]');
    if (ogDesc) {
      const content = ogDesc.getAttribute("content") || "";
      const match = content.match(/:\s*"(.+)"\.\s*$/s)
        || content.match(/:\s*["\u201c](.+)["\u201d]\s*$/s);
      if (match && match[1].length > 10) return match[1];
    }

    // Fallback: find the longest span[dir=auto] that isn't UI text
    const candidates = [];
    for (const span of document.querySelectorAll('span[dir="auto"]')) {
      const text = span.innerText?.trim();
      if (!text || text.length < 30) continue;
      if (text.startsWith("@")) continue;
      if (text.match(/^\d+\s*(curtida|like|comentário|comment)/i)) continue;
      if (text.match(/^(Responder|Ver\s|Curtir|Editar|Traduzir|Página|Notif)/)) continue;
      if (text.match(/\n\s*\d+\s*(sem|h|d|min)\n/)) continue;
      candidates.push(text);
    }
    if (candidates.length === 0) return null;
    return candidates.reduce((a, b) => (a.length > b.length ? a : b));
  });
}

async function screenshotPostImage(page, outPath) {
  // Screenshot the media area: image or video poster
  for (const selector of [
    'article img[sizes]',              // post image
    'article div[role="presentation"]', // carousel/video container
    'article video',                    // reel/video
  ]) {
    const el = page.locator(selector).first();
    try {
      if (await el.isVisible({ timeout: 1500 })) {
        await el.screenshot({ path: outPath });
        return true;
      }
    } catch {}
  }

  // Fallback: screenshot the visible viewport (captures the post)
  try {
    await page.screenshot({ path: outPath });
    return true;
  } catch {
    return false;
  }
}

async function extractPostMetadata(page) {
  return page.evaluate(() => {
    const meta = {};
    meta.isReel = window.location.pathname.includes("/reel/");

    const timeEl = document.querySelector("time[datetime]");
    if (timeEl) {
      meta.timestamp = timeEl.getAttribute("datetime");
      meta.timeAgo = timeEl.textContent?.trim();
    }

    // Try to get like count
    const ogDesc = document.querySelector('meta[property="og:description"]');
    if (ogDesc) {
      const content = ogDesc.getAttribute("content") || "";
      const likesMatch = content.match(/([\d,.]+)\s*(likes?|curtida)/i);
      if (likesMatch) meta.likes = likesMatch[1];
      const commentsMatch = content.match(/([\d,.]+)\s*(comments?|comentário)/i);
      if (commentsMatch) meta.comments = commentsMatch[1];
    }

    return meta;
  });
}

// -- Scrape loop ------------------------------------------------------------

const allData = [];

for (const { username, group } of accountList) {
  process.stderr.write(`\n>> @${username} (${group})\n`);

  try {
    await page.goto(`https://www.instagram.com/${username}/`, {
      waitUntil: "domcontentloaded",
      timeout: 15000,
    });
    await page.waitForTimeout(2500);

    const profileInfo = await page.evaluate(() => {
      const meta = document.querySelector('meta[property="og:description"]');
      const desc = meta?.getAttribute("content") || "";
      const followersMatch = desc.match(/([\d,.]+[KMkm]?)\s*(Followers|seguidores)/i);
      return {
        description: desc.slice(0, 300),
        followers: followersMatch?.[1] || null,
      };
    });

    const postLinks = await page.evaluate(() => {
      const links = [];
      for (const a of document.querySelectorAll(
        'a[href*="/p/"], a[href*="/reel/"]'
      )) {
        const href = a.getAttribute("href");
        if (href) links.push(href);
      }
      return [...new Set(links)];
    });

    process.stderr.write(
      `  ${postLinks.length} posts, followers: ${profileInfo.followers || "?"}\n`
    );

    const posts = [];
    const accountDir = resolve(OUT_DIR, username);
    mkdirSync(accountDir, { recursive: true });

    for (let i = 0; i < Math.min(postLinks.length, postsPerAccount); i++) {
      const link = postLinks[i];
      const fullUrl = link.startsWith("http")
        ? link
        : `https://www.instagram.com${link}`;
      try {
        await page.goto(fullUrl, {
          waitUntil: "domcontentloaded",
          timeout: 10000,
        });
        await page.waitForTimeout(2000);

        // Expand caption if truncated
        const moreBtn = page.locator('button:has-text("mais"), button:has-text("more")').first();
        if (await moreBtn.isVisible({ timeout: 800 }).catch(() => false)) {
          await moreBtn.click().catch(() => {});
          await page.waitForTimeout(500);
        }

        const caption = await extractCaption(page);
        const metadata = await extractPostMetadata(page);

        // Screenshot the post image
        const imgPath = resolve(accountDir, `post-${i + 1}.png`);
        const hasImage = await screenshotPostImage(page, imgPath);

        if (caption && caption.length > 15) {
          const cleanCaption = caption
            .replace(new RegExp(`^${username}\\s*`, "i"), "")
            .replace(/^\s*Editado\s*•?\s*/i, "")
            .replace(/^\s*\d+\s*(sem|h|d|min)\s*/i, "")
            .trim();

          posts.push({
            url: fullUrl,
            caption: cleanCaption,
            charCount: cleanCaption.length,
            wordCount: cleanCaption.split(/\s+/).length,
            imagePath: hasImage ? imgPath : null,
            ...metadata,
          });
          process.stderr.write(
            `  [${cleanCaption.length}c] ${cleanCaption.slice(0, 60)}...${hasImage ? " +img" : ""}\n`
          );
        } else if (hasImage) {
          // Still save posts that have images but no extractable caption
          posts.push({
            url: fullUrl,
            caption: null,
            charCount: 0,
            wordCount: 0,
            imagePath: imgPath,
            ...metadata,
          });
          process.stderr.write(`  [no caption] +img\n`);
        }
      } catch {
        // skip
      }
    }

    allData.push({
      username,
      group,
      followers: profileInfo.followers,
      postsScraped: posts.length,
      posts,
    });
  } catch (e) {
    process.stderr.write(`  ERROR: ${e.message}\n`);
    allData.push({
      username,
      group,
      followers: null,
      postsScraped: 0,
      posts: [],
      error: e.message,
    });
  }
}

// Save session again (refresh cookies)
await context.storageState({ path: SESSION_PATH });
await browser.close();

// -- Save raw data ----------------------------------------------------------

const jsonPath = resolve(OUT_DIR, "data.json");
writeFileSync(jsonPath, JSON.stringify(allData, null, 2));
process.stderr.write(`\nData saved to ${jsonPath}\n`);

// -- Analysis ---------------------------------------------------------------

const sep = "=".repeat(70);

const allCaptions = allData.flatMap((a) =>
  a.posts
    .filter((p) => p.caption)
    .map((p) => ({ ...p, username: a.username, group: a.group, followers: a.followers }))
);

if (allCaptions.length === 0) {
  console.log("No captions extracted. Check /tmp/ig-login-debug.png if login failed.");
  process.exit(0);
}

console.log(`\n${sep}`);
console.log(
  ` INSTAGRAM RESEARCH: ${allCaptions.length} captions, ${allData.reduce((n, a) => n + a.posts.filter((p) => p.imagePath).length, 0)} images`
);
console.log(sep);

// 1. Length by group
console.log("\n## 1. Caption Length by Group\n");
const groups = [...new Set(allCaptions.map((c) => c.group))];
for (const group of groups) {
  const gc = allCaptions.filter((c) => c.group === group);
  if (!gc.length) continue;
  const lens = gc.map((c) => c.charCount).sort((a, b) => a - b);
  const avg = Math.round(lens.reduce((a, b) => a + b, 0) / lens.length);
  const median = lens[Math.floor(lens.length / 2)];
  const words = Math.round(gc.reduce((s, c) => s + c.wordCount, 0) / gc.length);
  console.log(`  ${group} (${gc.length}): avg ${avg}c, median ${median}c, ~${words} words, range ${lens[0]}-${lens.at(-1)}`);
}

// 2. Opening hooks
console.log("\n## 2. Opening Hook Patterns\n");
const hooks = { question: 0, emoji_lead: 0, number_price: 0, exclamation: 0, scene_temporal: 0 };
for (const c of allCaptions) {
  const o = c.caption.slice(0, 80);
  if (o.match(/\?/)) hooks.question++;
  if (o.match(/^[\u{1F300}-\u{1FAFF}]/u)) hooks.emoji_lead++;
  if (o.match(/^\d|^R\$/)) hooks.number_price++;
  if (o.match(/!/)) hooks.exclamation++;
  if (o.match(/^(Hoje|Ontem|Essa|Quando|Aqui|Aquele|Essa manhã|Agora)/i)) hooks.scene_temporal++;
}
for (const [k, v] of Object.entries(hooks)) {
  console.log(`  ${k}: ${v}/${allCaptions.length} (${Math.round((v / allCaptions.length) * 100)}%)`);
}

// 3. Emoji density
console.log("\n## 3. Emoji Density\n");
const emojiRe = /[\u{1F300}-\u{1FAFF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/gu;
for (const group of groups) {
  const gc = allCaptions.filter((c) => c.group === group);
  if (!gc.length) continue;
  const counts = gc.map((c) => (c.caption.match(emojiRe) || []).length);
  const avg = (counts.reduce((a, b) => a + b, 0) / counts.length).toFixed(1);
  console.log(`  ${group}: avg ${avg} emojis/post`);
}

// 4. CTA patterns
console.log("\n## 4. CTA Patterns\n");
const cta = { zap: 0, link_bio: 0, comenta: 0, marca: 0, salva: 0, compartilha: 0, urgency: 0, none: 0 };
for (const c of allCaptions) {
  const end = c.caption.slice(-200).toLowerCase();
  let has = false;
  if (end.match(/whatsapp|zap|chama no/)) { cta.zap++; has = true; }
  if (end.match(/link.*bio|na bio/)) { cta.link_bio++; has = true; }
  if (end.match(/coment[ae]/)) { cta.comenta++; has = true; }
  if (end.match(/marca\s/)) { cta.marca++; has = true; }
  if (end.match(/salv[ae]/)) { cta.salva++; has = true; }
  if (end.match(/compartilh/)) { cta.compartilha++; has = true; }
  if (end.match(/corre|aproveite|garanta|não perca/)) { cta.urgency++; has = true; }
  if (!has) cta.none++;
}
for (const [k, v] of Object.entries(cta)) {
  console.log(`  ${k}: ${v}/${allCaptions.length} (${Math.round((v / allCaptions.length) * 100)}%)`);
}

// 5. Hashtags
console.log("\n## 5. Hashtags\n");
for (const group of groups) {
  const gc = allCaptions.filter((c) => c.group === group);
  if (!gc.length) continue;
  const counts = gc.map((c) => (c.caption.match(/#\w+/g) || []).length);
  const avg = (counts.reduce((a, b) => a + b, 0) / counts.length).toFixed(1);
  console.log(`  ${group}: avg ${avg} hashtags/post`);
}

// 6. Structure
console.log("\n## 6. Structure\n");
const st = { line_breaks: 0, prices: 0, questions: 0, ends_question: 0, dashes: 0 };
for (const c of allCaptions) {
  if (c.caption.includes("\n")) st.line_breaks++;
  if (c.caption.match(/R\$\s*[\d,.]+/)) st.prices++;
  if (c.caption.includes("?")) st.questions++;
  if (c.caption.trim().endsWith("?")) st.ends_question++;
  if (c.caption.match(/[—–]/)) st.dashes++;
}
for (const [k, v] of Object.entries(st)) {
  console.log(`  ${k}: ${v}/${allCaptions.length} (${Math.round((v / allCaptions.length) * 100)}%)`);
}

// 7. Full dump
console.log(`\n${sep}`);
console.log(" FULL CAPTIONS");
console.log(sep);
for (const a of allData) {
  if (!a.posts.some((p) => p.caption)) continue;
  console.log(`\n### @${a.username} (${a.group}, ${a.followers || "?"} followers)`);
  for (let i = 0; i < a.posts.length; i++) {
    const p = a.posts[i];
    if (!p.caption) continue;
    const engagement = [p.likes && `${p.likes} likes`, p.comments && `${p.comments} comments`].filter(Boolean).join(", ");
    console.log(`\n  --- post ${i + 1} [${p.charCount}c, ${p.wordCount}w${p.isReel ? ", reel" : ""}${engagement ? `, ${engagement}` : ""}] ---`);
    console.log(`  ${p.caption}`);
  }
}

// Summary
const allLens = allCaptions.map((c) => c.charCount).sort((a, b) => a - b);
const totalAvg = Math.round(allLens.reduce((a, b) => a + b, 0) / allLens.length);
const totalMedian = allLens[Math.floor(allLens.length / 2)];
console.log(`\n${sep}`);
console.log(` OVERALL: ${allCaptions.length} captions, avg ${totalAvg}c, median ${totalMedian}c`);
console.log(` Images saved to: ${OUT_DIR}/`);
console.log(` Rekan current limit: 700 chars`);
console.log(sep);
