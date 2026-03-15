#!/usr/bin/env node
/**
 * Discover real MEI accounts via Instagram search, then scrape their captions.
 *
 * Uses the saved session from scrape-captions.mjs. Searches Instagram's
 * explore/tags pages and "suggested" accounts to find real small businesses.
 *
 * Usage:
 *   cd web && node ../scripts/discover-and-scrape.mjs
 *   cd web && node ../scripts/discover-and-scrape.mjs --headed
 *   cd web && node ../scripts/discover-and-scrape.mjs --niche=confeitaria
 */

import { createRequire } from "node:module";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { writeFileSync, existsSync, mkdirSync, readFileSync } from "node:fs";

const __dirname = dirname(fileURLToPath(import.meta.url));
const require = createRequire(resolve(__dirname, "../web/") + "/");
const { chromium } = require("@playwright/test");

const SESSION_PATH = "/tmp/ig-session.json";
const OUT_DIR = resolve(__dirname, "../docs/caption-research");

// -- MEI search queries per niche -------------------------------------------
// These are small real businesses, not brands or suppliers.
// We search Google for "site:instagram.com <query>" to find them,
// then scrape their captions directly on Instagram.

const NICHES = {
  confeitaria: {
    googleQueries: [],
    igSearchTerms: [
      "confeiteira artesanal",
      "bolo caseiro encomenda",
      "doces artesanais",
      "confeitaria",
      "bolo personalizado",
      "cake designer",
      "doces finos",
      "brigadeiro gourmet",
    ],
    knownAccounts: [
      "confeitariaanacristina",
      "brunarebelo",
      "docfranciscana",
      "caborges_confeitaria",
      "alebalduino_cakes",
      "sweet.carol.cakes",
      "boleiradalu",
    ],
  },
  nails: {
    googleQueries: [
      'site:instagram.com "manicure" "agendamento" -curso -marca',
      'site:instagram.com "nail designer" "agenda aberta"',
      'site:instagram.com "unhas em gel" "atendimento" cidade',
    ],
    igSearchTerms: [
      "nail designer",
      "manicure profissional",
      "unhas em gel",
    ],
    knownAccounts: [
      "manicuresinceraoficial",
      "natiakemioficial",
      "manicur3profissional",
      "manu_naildesigner",
      "dayanenaildesigner",
      "naildesignerflavia",
    ],
  },
  hair: {
    googleQueries: [
      'site:instagram.com "cabeleireira" "agendamento" -curso',
      'site:instagram.com "salão de beleza" "agende" cidade',
      'site:instagram.com "colorista" "transformação capilar"',
    ],
    igSearchTerms: [
      "cabeleireira",
      "salao de beleza",
      "colorista capilar",
    ],
    knownAccounts: [
      "institutoembelleze",
      "pefrancohair",
      "andressahair_",
      "carol.cabeleireira",
      "espacobelissimaa",
    ],
  },
  marmiteira: {
    googleQueries: [
      'site:instagram.com "marmita" "cardápio" "delivery"',
      'site:instagram.com "marmitex" "encomendas" whatsapp',
    ],
    igSearchTerms: [
      "marmitex caseiro",
      "comida caseira delivery",
      "marmita caseira",
      "quentinha delivery",
      "marmitaria",
      "comida por encomenda",
    ],
    // Verified accounts from previous scrape runs
    knownAccounts: [
      "marmitexcaseiro_",
      "marmitex_caseiro.norte",
      "delivery.lacasadebarro",
      "oficialrestaurantebomsabor",
      "deliciasdaraytimon",
    ],
  },
  costureira: {
    googleQueries: [],
    igSearchTerms: [
      "costureira",
      "atelie costura",
      "costura sob medida",
      "ajuste de roupas",
      "costureira profissional",
      "atelie de costura",
    ],
    // abordarcomsonia verified from previous run
    knownAccounts: [
      "abordarcomsonia",
      "profissaocostureira",
    ],
  },
  loja: {
    googleQueries: [],
    igSearchTerms: [
      "lojinha online",
      "brecho online",
      "loja de roupas femininas",
      "lojinha artesanal",
      "acessorios artesanais",
      "brecho feminino",
    ],
    knownAccounts: [],
  },
  diarista: {
    googleQueries: [],
    igSearchTerms: [
      "diarista",
      "diarista profissional",
      "faxineira",
      "limpeza residencial",
      "personal organizer casa",
      "organizacao domestica",
    ],
    knownAccounts: [],
  },
};

// -- CLI args ---------------------------------------------------------------

const args = process.argv.slice(2);
const headed = args.includes("--headed");
const nicheArg = args.find((a) => a.startsWith("--niche="))?.split("=")[1];
const postsPerAccount = parseInt(
  args.find((a) => a.startsWith("--posts="))?.split("=")[1] || "6"
);
const skipDiscovery = args.includes("--skip-discovery");
const skipGoogle = args.includes("--skip-google");
const meiOnly = args.includes("--mei-only");
const minFollowers = parseInt(args.find((a) => a.startsWith("--min-followers="))?.split("=")[1] || (meiOnly ? "500" : "0"));
const maxFollowers = parseInt(args.find((a) => a.startsWith("--max-followers="))?.split("=")[1] || (meiOnly ? "100000" : "0"));

const nichesToScrape = nicheArg
  ? { [nicheArg]: NICHES[nicheArg] }
  : NICHES;

if (nicheArg && !NICHES[nicheArg]) {
  console.error(`Unknown niche: ${nicheArg}. Available: ${Object.keys(NICHES).join(", ")}`);
  process.exit(1);
}

// -- Browser setup ----------------------------------------------------------

if (!existsSync(SESSION_PATH)) {
  console.error("No session found. Run scrape-captions.mjs first to log in.");
  process.exit(1);
}

const browser = await chromium.launch({
  headless: !headed,
  args: ["--disable-blink-features=AutomationControlled"],
});

const context = await browser.newContext({
  locale: "pt-BR",
  userAgent:
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
  viewport: { width: 1280, height: 900 },
  storageState: SESSION_PATH,
});

const page = await context.newPage();

// Verify session
await page.goto("https://www.instagram.com/", {
  waitUntil: "domcontentloaded",
  timeout: 15000,
});
await page.waitForTimeout(2000);
const loggedIn = await page.evaluate(
  () => !document.querySelector('input[name="email"], input[name="username"]')
);
if (!loggedIn) {
  console.error("Session expired. Run scrape-captions.mjs first to re-login.");
  await browser.close();
  process.exit(1);
}
process.stderr.write("Session OK\n");

// Dismiss popups
for (const text of ["Agora não", "Not Now"]) {
  const btn = page.locator(`button:has-text("${text}")`);
  if (await btn.first().isVisible({ timeout: 1500 }).catch(() => false)) {
    await btn.first().click();
    await page.waitForTimeout(800);
  }
}

// Parse "1,234" / "5K" / "2.5M" into a number
function parseMetric(s) {
  if (!s) return 0;
  s = s.replace(/,/g, "").trim();
  const m = s.match(/^([\d.]+)\s*([KkMm]?)$/);
  if (!m) return 0;
  let n = parseFloat(m[1]);
  if (m[2] === "K" || m[2] === "k") n *= 1000;
  if (m[2] === "M" || m[2] === "m") n *= 1000000;
  return Math.round(n);
}

// -- Discovery via Google ---------------------------------------------------

async function discoverFromGoogle(page, queries, maxAccounts = 8) {
  const found = new Set();
  const googlePage = await context.newPage();

  for (const q of queries) {
    if (found.size >= maxAccounts) break;
    try {
      const url = `https://www.google.com/search?q=${encodeURIComponent(q)}&num=15&hl=pt-BR`;
      await googlePage.goto(url, { waitUntil: "domcontentloaded", timeout: 12000 });
      await googlePage.waitForTimeout(1000);

      // Accept cookies
      const consent = googlePage.locator('button:has-text("Aceitar"), button:has-text("Accept all")');
      if (await consent.first().isVisible({ timeout: 1000 }).catch(() => false)) {
        await consent.first().click();
        await googlePage.waitForTimeout(500);
      }

      const usernames = await googlePage.evaluate(() => {
        const ignore = new Set([
          "p", "reel", "reels", "explore", "stories", "accounts",
          "directory", "about", "legal", "developer", "help", "privacy",
          "terms", "tags", "instagram", "popular", "search", "tv", "static", "web",
        ]);
        const results = [];
        for (const a of document.querySelectorAll("a[href]")) {
          const match = a.href.match(/instagram\.com\/([a-zA-Z0-9_.]+)/);
          if (!match) continue;
          const u = match[1].toLowerCase();
          if (!ignore.has(u) && u.length > 3) results.push(u);
        }
        return [...new Set(results)];
      });

      for (const u of usernames) {
        if (found.size < maxAccounts) found.add(u);
      }

      process.stderr.write(`  google "${q.slice(0, 50)}...": ${usernames.length} accounts\n`);
    } catch {
      // skip
    }
    await googlePage.waitForTimeout(1500 + Math.random() * 1000);
  }

  await googlePage.close();
  return [...found];
}

// -- Discovery via Instagram search ------------------------------------------

async function discoverFromInstagram(page, searchTerms, maxAccounts = 8) {
  const found = new Set();

  for (const term of searchTerms) {
    if (found.size >= maxAccounts) break;
    try {
      const encoded = encodeURIComponent(term);
      const apiUrl = `https://www.instagram.com/api/v1/web/search/topsearch/?context=blended&query=${encoded}&include_reel=false`;

      const response = await page.evaluate(async (url) => {
        const res = await fetch(url, { credentials: "include" });
        if (!res.ok) return null;
        return res.json();
      }, apiUrl);

      if (response?.users) {
        const users = response.users
          .filter((u) => u.user?.username && !u.user?.is_private)
          .map((u) => u.user.username);
        for (const u of users) {
          if (found.size < maxAccounts) found.add(u);
        }
        process.stderr.write(`  ig search "${term}": ${users.length} public accounts\n`);
      } else {
        process.stderr.write(`  ig search "${term}": no results\n`);
      }
    } catch (e) {
      process.stderr.write(`  ig search "${term}": error ${e.message}\n`);
    }
    await page.waitForTimeout(1500 + Math.random() * 1000);
  }

  return [...found];
}

// -- Caption extraction (same as scrape-captions.mjs) -----------------------

async function extractCaption(page) {
  return page.evaluate(() => {
    const ogDesc = document.querySelector('meta[property="og:description"]');
    if (ogDesc) {
      const content = ogDesc.getAttribute("content") || "";
      const match = content.match(/:\s*"(.+)"\.\s*$/s)
        || content.match(/:\s*["\u201c](.+)["\u201d]\s*$/s);
      if (match && match[1].length > 10) return match[1];
    }
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
  for (const selector of [
    'article img[sizes]',
    'article div[role="presentation"]',
    'article video',
  ]) {
    const el = page.locator(selector).first();
    try {
      if (await el.isVisible({ timeout: 1500 })) {
        await el.screenshot({ path: outPath });
        return true;
      }
    } catch {}
  }
  try {
    await page.screenshot({ path: outPath });
    return true;
  } catch {
    return false;
  }
}

async function scrapeAccount(page, username, niche) {
  try {
    await page.goto(`https://www.instagram.com/${username}/`, {
      waitUntil: "domcontentloaded",
      timeout: 15000,
    });
    await page.waitForTimeout(1200);

    // Detect non-existent or private accounts fast
    const profileStatus = await page.evaluate(() => {
      const body = document.body?.innerText || "";
      if (body.includes("Esta página não está disponível") || body.includes("this page isn't available"))
        return "not_found";
      if (body.includes("Esta conta é privada") || body.includes("This account is private"))
        return "private";
      return "ok";
    });

    if (profileStatus !== "ok") {
      process.stderr.write(`  ${profileStatus}, skipping\n`);
      return { username, group: niche, followers: null, postsScraped: 0, posts: [], status: profileStatus };
    }

    const profileInfo = await page.evaluate(() => {
      const meta = document.querySelector('meta[property="og:description"]');
      const desc = meta?.getAttribute("content") || "";
      const followersMatch = desc.match(/([\d,.]+[KMkm]?)\s*(Followers|seguidores)/i);
      return {
        description: desc.slice(0, 300),
        followers: followersMatch?.[1] || null,
      };
    });

    // Check follower range
    const fNum = parseMetric(profileInfo.followers);
    if (minFollowers > 0 && fNum < minFollowers) {
      process.stderr.write(`  ${profileInfo.followers || "?"} followers (< ${minFollowers}), skipping\n`);
      return { username, group: niche, followers: profileInfo.followers, postsScraped: 0, posts: [], status: "too_small" };
    }
    if (maxFollowers > 0 && fNum > maxFollowers) {
      process.stderr.write(`  ${profileInfo.followers} followers (> ${maxFollowers}), skipping\n`);
      return { username, group: niche, followers: profileInfo.followers, postsScraped: 0, posts: [], status: "too_big" };
    }

    const postLinks = await page.evaluate(() => {
      const links = [];
      for (const a of document.querySelectorAll('a[href*="/p/"], a[href*="/reel/"]')) {
        const href = a.getAttribute("href");
        if (href) links.push(href);
      }
      return [...new Set(links)];
    });

    process.stderr.write(
      `  ${postLinks.length} posts, followers: ${profileInfo.followers || "?"}\n`
    );

    if (postLinks.length === 0) {
      return { username, group: niche, followers: profileInfo.followers, postsScraped: 0, posts: [] };
    }

    const accountDir = resolve(OUT_DIR, username);
    mkdirSync(accountDir, { recursive: true });

    const posts = [];
    for (let i = 0; i < Math.min(postLinks.length, postsPerAccount); i++) {
      const link = postLinks[i];
      const fullUrl = link.startsWith("http") ? link : `https://www.instagram.com${link}`;
      try {
        await page.goto(fullUrl, { waitUntil: "domcontentloaded", timeout: 10000 });
        await page.waitForTimeout(1000);

        // Expand caption
        const moreBtn = page.locator('button:has-text("mais"), button:has-text("more")').first();
        if (await moreBtn.isVisible({ timeout: 800 }).catch(() => false)) {
          await moreBtn.click().catch(() => {});
          await page.waitForTimeout(500);
        }

        const caption = await extractCaption(page);
        const imgPath = resolve(accountDir, `post-${i + 1}.png`);
        const hasImage = await screenshotPostImage(page, imgPath);

        // Get engagement data
        const metadata = await page.evaluate(() => {
          const meta = {};
          meta.isReel = window.location.pathname.includes("/reel/");
          const timeEl = document.querySelector("time[datetime]");
          if (timeEl) {
            meta.timestamp = timeEl.getAttribute("datetime");
            meta.timeAgo = timeEl.textContent?.trim();
          }
          const ogDesc = document.querySelector('meta[property="og:description"]')?.getAttribute("content") || "";
          const likesMatch = ogDesc.match(/([\d,.]+[KMk]?)\s*(likes?|curtida)/i);
          if (likesMatch) meta.likes = likesMatch[1];
          const commentsMatch = ogDesc.match(/([\d,.]+[KMk]?)\s*(comments?|comentário)/i);
          if (commentsMatch) meta.comments = commentsMatch[1];
          return meta;
        });

        if (caption && caption.length > 10) {
          posts.push({
            url: fullUrl,
            caption,
            charCount: caption.length,
            wordCount: caption.split(/\s+/).length,
            imagePath: hasImage ? imgPath : null,
            ...metadata,
          });
          process.stderr.write(
            `  [${caption.length}c] ${caption.slice(0, 60)}...\n`
          );
        } else if (hasImage) {
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
      } catch {}
    }

    return {
      username,
      group: niche,
      followers: profileInfo.followers,
      postsScraped: posts.length,
      posts,
    };
  } catch (e) {
    process.stderr.write(`  ERROR: ${e.message}\n`);
    return { username, group: niche, followers: null, postsScraped: 0, posts: [], error: e.message };
  }
}

// -- Main loop --------------------------------------------------------------

mkdirSync(OUT_DIR, { recursive: true });

// Load existing data to merge
const dataPath = resolve(OUT_DIR, "data.json");
let existingData = [];
if (existsSync(dataPath)) {
  try {
    existingData = JSON.parse(readFileSync(dataPath, "utf8"));
  } catch {}
}
const existingUsernames = new Set(existingData.map((a) => a.username));

const newData = [];

for (const [niche, config] of Object.entries(nichesToScrape)) {
  process.stderr.write(`\n${"=".repeat(50)}\n NICHE: ${niche}\n${"=".repeat(50)}\n`);

  // Combine known accounts + discovered accounts
  let accounts = [...config.knownAccounts];

  if (!skipDiscovery) {
    if (!skipGoogle) {
      process.stderr.write("\nDiscovering via Google...\n");
      const discovered = await discoverFromGoogle(page, config.googleQueries, 6);
      for (const u of discovered) {
        if (!accounts.includes(u)) accounts.push(u);
      }
    }

    if (config.igSearchTerms) {
      process.stderr.write("\nDiscovering via Instagram search...\n");
      const igFound = await discoverFromInstagram(page, config.igSearchTerms, 15);
      for (const u of igFound) {
        if (!accounts.includes(u)) accounts.push(u);
      }
    }

    process.stderr.write(`Total accounts for ${niche}: ${accounts.length}\n`);
  } // !skipDiscovery

  for (const username of accounts) {
    // Skip if already scraped with captions
    const existing = existingData.find((a) => a.username === username);
    if (existing && existing.posts.some((p) => p.caption)) {
      process.stderr.write(`\n>> @${username} (${niche}) — already scraped, skipping\n`);
      continue;
    }

    process.stderr.write(`\n>> @${username} (${niche})\n`);
    const result = await scrapeAccount(page, username, niche);
    newData.push(result);

    await page.waitForTimeout(500 + Math.random() * 500);
  }
}

// Refresh session
await context.storageState({ path: SESSION_PATH });
await browser.close();

// -- Merge with existing data -----------------------------------------------

// Replace existing entries with new data, add new ones
const mergedMap = new Map();
for (const a of existingData) mergedMap.set(a.username, a);
for (const a of newData) {
  // Only replace if new data has more captions
  const existing = mergedMap.get(a.username);
  if (!existing || a.posts.filter((p) => p.caption).length > existing.posts.filter((p) => p.caption).length) {
    mergedMap.set(a.username, a);
  }
}
const allData = [...mergedMap.values()];

writeFileSync(dataPath, JSON.stringify(allData, null, 2));
process.stderr.write(`\nData saved to ${dataPath} (${allData.length} total accounts)\n`);

// -- Analysis ---------------------------------------------------------------

const allCaptions = allData.flatMap((a) =>
  a.posts
    .filter((p) => p.caption)
    .map((p) => ({ ...p, username: a.username, group: a.group, followers: a.followers }))
);

if (allCaptions.length === 0) {
  console.log("No captions extracted.");
  process.exit(0);
}

const sep = "=".repeat(70);
console.log(`\n${sep}`);
console.log(
  ` INSTAGRAM RESEARCH: ${allCaptions.length} captions from ${allData.filter((a) => a.posts.some((p) => p.caption)).length} accounts`
);
console.log(sep);

// By niche
console.log("\n## Caption Length by Niche\n");
const groups = [...new Set(allCaptions.map((c) => c.group))].sort();
for (const group of groups) {
  const gc = allCaptions.filter((c) => c.group === group);
  if (!gc.length) continue;
  const lens = gc.map((c) => c.charCount).sort((a, b) => a - b);
  const avg = Math.round(lens.reduce((a, b) => a + b, 0) / lens.length);
  const median = lens[Math.floor(lens.length / 2)];
  const words = Math.round(gc.reduce((s, c) => s + c.wordCount, 0) / gc.length);
  const accounts = [...new Set(gc.map((c) => c.username))].length;
  console.log(`  ${group} (${accounts} accounts, ${gc.length} posts): avg ${avg}c, median ${median}c, ~${words} words`);
}

// Hooks
console.log("\n## Opening Hook Patterns\n");
const hooks = { question: 0, exclamation: 0, scene_temporal: 0, emoji_lead: 0, number_price: 0 };
for (const c of allCaptions) {
  const o = c.caption.slice(0, 80);
  if (o.match(/\?/)) hooks.question++;
  if (o.match(/!/)) hooks.exclamation++;
  if (o.match(/^(Hoje|Ontem|Essa|Quando|Aqui|Aquele|Agora|Essa manhã)/i)) hooks.scene_temporal++;
  if (o.match(/^[\u{1F300}-\u{1FAFF}]/u)) hooks.emoji_lead++;
  if (o.match(/^\d|^R\$/)) hooks.number_price++;
}
for (const [k, v] of Object.entries(hooks)) {
  console.log(`  ${k}: ${v}/${allCaptions.length} (${Math.round((v / allCaptions.length) * 100)}%)`);
}

// Emoji density by niche
console.log("\n## Emoji Density\n");
const emojiRe = /[\u{1F300}-\u{1FAFF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/gu;
for (const group of groups) {
  const gc = allCaptions.filter((c) => c.group === group);
  if (!gc.length) continue;
  const counts = gc.map((c) => (c.caption.match(emojiRe) || []).length);
  const avg = (counts.reduce((a, b) => a + b, 0) / counts.length).toFixed(1);
  console.log(`  ${group}: avg ${avg} emojis/post`);
}

// CTA
console.log("\n## CTA Patterns\n");
const cta = { zap: 0, link_bio: 0, comenta: 0, marca: 0, salva: 0, urgency: 0, none: 0 };
for (const c of allCaptions) {
  const end = c.caption.slice(-200).toLowerCase();
  let has = false;
  if (end.match(/whatsapp|zap|chama no/)) { cta.zap++; has = true; }
  if (end.match(/link.*bio|na bio/)) { cta.link_bio++; has = true; }
  if (end.match(/coment[ae]/)) { cta.comenta++; has = true; }
  if (end.match(/marca\s/)) { cta.marca++; has = true; }
  if (end.match(/salv[ae]/)) { cta.salva++; has = true; }
  if (end.match(/corre|aproveite|garanta|não perca/)) { cta.urgency++; has = true; }
  if (!has) cta.none++;
}
for (const [k, v] of Object.entries(cta)) {
  console.log(`  ${k}: ${v}/${allCaptions.length} (${Math.round((v / allCaptions.length) * 100)}%)`);
}

// Hashtags
console.log("\n## Hashtags\n");
for (const group of groups) {
  const gc = allCaptions.filter((c) => c.group === group);
  if (!gc.length) continue;
  const counts = gc.map((c) => (c.caption.match(/#\w+/g) || []).length);
  const avg = (counts.reduce((a, b) => a + b, 0) / counts.length).toFixed(1);
  console.log(`  ${group}: avg ${avg} hashtags/post`);
}

// Structure
console.log("\n## Structure\n");
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

// Overall
const allLens = allCaptions.map((c) => c.charCount).sort((a, b) => a - b);
const totalAvg = Math.round(allLens.reduce((a, b) => a + b, 0) / allLens.length);
const totalMedian = allLens[Math.floor(allLens.length / 2)];
console.log(`\n${sep}`);
console.log(` OVERALL: ${allCaptions.length} captions from ${allData.filter((a) => a.posts.some((p) => p.caption)).length} accounts`);
console.log(` Average: ${totalAvg}c, Median: ${totalMedian}c, Range: ${allLens[0]}-${allLens.at(-1)}`);
console.log(` Images saved to: ${OUT_DIR}/`);
console.log(sep);
