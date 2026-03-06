#!/usr/bin/env node
/**
 * Discover Instagram accounts to follow in a niche.
 *
 * Uses Playwright to search Google and extract Instagram profiles.
 * Real browser = no CAPTCHAs from search engines.
 *
 * Usage:
 *   cd web && node ../scripts/discover-accounts.mjs confeitaria
 *   cd web && node ../scripts/discover-accounts.mjs confeitaria --location "Sao Paulo"
 *   cd web && node ../scripts/discover-accounts.mjs nails --headed
 *   cd web && node ../scripts/discover-accounts.mjs --all
 */

import { createRequire } from "node:module";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

// Playwright is installed in web/node_modules, resolve from there
const __dirname = dirname(fileURLToPath(import.meta.url));
const require = createRequire(resolve(__dirname, "../web/") + "/");
const { chromium } = require("@playwright/test");

// -- Niche seed data (from PEP-009) -----------------------------------------

const NICHES = {
  confeitaria: {
    keywords: [
      "confeitaria artesanal",
      "confeiteira",
      "bolo decorado",
      "cake designer brasileira",
      "doceria artesanal",
      "bolos personalizados",
    ],
    hashtags: [
      "confeitariaartesanal",
      "confeiteira",
      "bolosdecorados",
      "cakedesign",
      "doceriaartesanal",
      "bolospersonalizados",
    ],
    seeds: [
      "lojasantoantonio",
      "magoindustria",
      "_mavalerio",
      "mixingredientes",
      "confeitariabrasilcursosonline",
      "confeitariaanacristina",
      "brunarebelo",
    ],
  },
  nails: {
    keywords: [
      "manicure profissional",
      "nail designer",
      "unhas decoradas",
      "manicure e pedicure",
    ],
    hashtags: [
      "unhasdecoradas",
      "naildesigner",
      "manicureprofissional",
      "unhasdegel",
    ],
    seeds: [
      "voliacosmeticos",
      "institutonati",
      "manicuresinceraoficial",
      "manicur3profissional",
      "natiakemioficial",
    ],
  },
  hair: {
    keywords: [
      "cabeleireira",
      "salao de beleza",
      "colorista capilar",
      "hair stylist brasileira",
    ],
    hashtags: [
      "cabeleireira",
      "hairstylistbrasil",
      "salaodebeleza",
      "coloristacapilar",
    ],
    seeds: [
      "wellaprobrasil",
      "cadiveu",
      "itallianhairtech",
      "lowelloficial",
      "institutoembelleze",
    ],
  },
  marmiteira: {
    keywords: [
      "marmitex delivery",
      "marmita fitness",
      "marmita caseira",
      "quentinha delivery",
      "comida caseira delivery",
      "refeicao congelada caseira",
    ],
    hashtags: [
      "marmitex",
      "marmitafitness",
      "marmitacaseira",
      "comidacaseira",
      "quentinha",
      "marmitacongelada",
    ],
    seeds: [
      "isaaborges.fit",
      "maborges_fit",
      "fitfoodcozinhasaudavel",
    ],
  },
  costureira: {
    keywords: [
      "costureira",
      "atelie de costura",
      "costura sob medida",
      "conserto de roupas",
      "ajuste de roupas",
      "costureira profissional",
    ],
    hashtags: [
      "costureira",
      "ateliedecostura",
      "costurasobmedida",
      "costuracriativa",
      "ajustederoupas",
      "costureiraempreendedora",
    ],
    seeds: [
      "abordarcomsonia",
      "atelie_crisaguiar",
      "profissaocostureira",
    ],
  },
  diarista: {
    keywords: [
      "diarista",
      "faxineira profissional",
      "limpeza residencial",
      "diarista autonoma",
      "servico de limpeza",
    ],
    hashtags: [
      "diarista",
      "faxineira",
      "limpezaresidencial",
      "diaristaautonoma",
      "limpezaprofissional",
      "organizacaodecasa",
    ],
    seeds: [
      "a_diarista_da_vez",
      "faborges_organize",
      "organizecomanozes",
    ],
  },
};

const MEI_ACCOUNTS = new Set([
  "sebrae",
  "raphafalcaof",
  "redemulherempreendedora",
  "contabilizei",
]);

const IGNORE = new Set([
  "p",
  "reel",
  "reels",
  "explore",
  "stories",
  "accounts",
  "directory",
  "about",
  "legal",
  "developer",
  "help",
  "privacy",
  "terms",
  "tags",
  "instagram",
  "popular",
  "search",
  "tv",
  "static",
  "web",
]);

// -- Google search via Playwright -------------------------------------------

/**
 * Search Google for Instagram profiles matching a query.
 * Returns array of { username, title, snippet }.
 */
async function googleSearch(page, query) {
  const q = `site:instagram.com ${query}`;
  const url = `https://www.google.com/search?q=${encodeURIComponent(q)}&num=20&hl=pt-BR`;

  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 15000 });

  // Accept cookies if prompted
  const consentBtn = page.locator('button:has-text("Aceitar"), button:has-text("Accept all"), button:has-text("I agree")');
  if (await consentBtn.first().isVisible({ timeout: 2000 }).catch(() => false)) {
    await consentBtn.first().click();
    await page.waitForTimeout(1000);
  }

  // Extract all links and their text
  const results = await page.evaluate(() => {
    const items = [];
    for (const a of document.querySelectorAll("a[href]")) {
      const href = a.href || "";
      const match = href.match(/instagram\.com\/([a-zA-Z0-9_.]+)/);
      if (!match) continue;
      const username = match[1].toLowerCase();
      // Get the closest parent that looks like a search result
      const container = a.closest("[data-snhf]") || a.closest(".g") || a.parentElement;
      const title = a.textContent?.trim() || "";
      const snippet = container?.textContent?.trim().slice(0, 200) || "";
      items.push({ username, title, snippet });
    }
    return items;
  });

  return results;
}

/**
 * Search Google for Instagram profiles using "related:" queries.
 * e.g. "related:instagram.com/lojasantoantonio confeitaria"
 */
async function googleRelatedSearch(page, seedAccount, nicheKeyword) {
  const q = `related:instagram.com/${seedAccount} ${nicheKeyword}`;
  const url = `https://www.google.com/search?q=${encodeURIComponent(q)}&num=20&hl=pt-BR`;

  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 15000 });
  await page.waitForTimeout(500);

  const results = await page.evaluate(() => {
    const items = [];
    for (const a of document.querySelectorAll("a[href]")) {
      const href = a.href || "";
      const match = href.match(/instagram\.com\/([a-zA-Z0-9_.]+)/);
      if (!match) continue;
      items.push({
        username: match[1].toLowerCase(),
        title: a.textContent?.trim() || "",
        snippet: "",
      });
    }
    return items;
  });

  return results;
}

// -- Dedup & filter ---------------------------------------------------------

function collectAccounts(rawResults, seedSet) {
  const accounts = new Map();
  for (const r of rawResults) {
    const u = r.username;
    if (IGNORE.has(u) || seedSet.has(u) || MEI_ACCOUNTS.has(u)) continue;
    if (!accounts.has(u)) {
      accounts.set(u, { username: u, title: r.title, snippet: r.snippet, count: 1 });
    } else {
      accounts.get(u).count++;
    }
  }
  // Sort by how many times they appeared (more = more relevant)
  return [...accounts.values()].sort((a, b) => b.count - a.count);
}

// -- CLI --------------------------------------------------------------------

function parseArgs() {
  const args = process.argv.slice(2);
  const opts = { niches: [], location: null, headed: false };

  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--location" && args[i + 1]) {
      opts.location = args[++i];
    } else if (args[i] === "--headed") {
      opts.headed = true;
    } else if (args[i] === "--all") {
      opts.niches = Object.keys(NICHES);
    } else if (NICHES[args[i]]) {
      opts.niches.push(args[i]);
    } else {
      console.error(`Unknown arg: ${args[i]}`);
      console.error(
        `Usage: node discover-accounts.mjs <${Object.keys(NICHES).join("|")}|--all> [--location "city"] [--headed]`
      );
      process.exit(1);
    }
  }

  if (opts.niches.length === 0) {
    console.error(
      `Usage: node discover-accounts.mjs <${Object.keys(NICHES).join("|")}|--all> [--location "city"] [--headed]`
    );
    process.exit(1);
  }

  return opts;
}

async function discoverNiche(page, nicheName, nicheData, location) {
  const seedSet = new Set(nicheData.seeds.map((s) => s.toLowerCase()));
  const allRaw = [];

  // Strategy 1: direct keyword search
  const queries = nicheData.keywords.slice(0, 4);
  if (location) {
    queries.unshift(...nicheData.keywords.slice(0, 2).map((k) => `${k} ${location}`));
  }

  for (const q of queries) {
    process.stderr.write(`  google: "${q}" ... `);
    try {
      const results = await googleSearch(page, q);
      process.stderr.write(`${results.length} results\n`);
      allRaw.push(...results);
    } catch (e) {
      process.stderr.write(`failed: ${e.message}\n`);
    }
    await page.waitForTimeout(1500 + Math.random() * 1000);
  }

  // Strategy 2: "related:" search from seed accounts
  for (const seed of nicheData.seeds.slice(0, 3)) {
    const keyword = nicheData.keywords[0];
    process.stderr.write(`  related: @${seed} ... `);
    try {
      const results = await googleRelatedSearch(page, seed, keyword);
      process.stderr.write(`${results.length} results\n`);
      allRaw.push(...results);
    } catch (e) {
      process.stderr.write(`failed: ${e.message}\n`);
    }
    await page.waitForTimeout(1500 + Math.random() * 1000);
  }

  return collectAccounts(allRaw, seedSet);
}

function printResults(nicheName, accounts, nicheData) {
  const sep = "=".repeat(60);
  console.log(`\n${sep}`);
  console.log(` ${nicheName.toUpperCase()} - discovered accounts`);
  console.log(sep);

  if (accounts.length === 0) {
    console.log("\nNo new accounts found.");
  } else {
    console.log(`\nFound ${accounts.length} accounts (sorted by relevance):\n`);
    for (const a of accounts) {
      const freq = a.count > 1 ? ` (appeared ${a.count}x)` : "";
      console.log(`  @${a.username}${freq}`);
      if (a.title) console.log(`    ${a.title.slice(0, 80)}`);
    }
  }

  console.log(`\n${sep}`);
  console.log(" Already-known seed accounts:");
  console.log(sep);
  for (const s of nicheData.seeds) console.log(`  @${s}`);
  for (const s of MEI_ACCOUNTS) console.log(`  @${s}  (cross-niche MEI)`);

  console.log(`\n${sep}`);
  console.log(" Hashtags to explore:");
  console.log(sep);
  for (const h of nicheData.hashtags) {
    console.log(`  #${h}  ->  https://www.instagram.com/explore/tags/${h}/`);
  }
  console.log();
}

// -- Main -------------------------------------------------------------------

const opts = parseArgs();

const browser = await chromium.launch({
  headless: !opts.headed,
  args: ["--disable-blink-features=AutomationControlled"],
});

const context = await browser.newContext({
  locale: "pt-BR",
  userAgent:
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
});

const page = await context.newPage();

try {
  for (const nicheName of opts.niches) {
    process.stderr.write(`\n>> Discovering: ${nicheName}\n`);
    const nicheData = NICHES[nicheName];
    const accounts = await discoverNiche(page, nicheName, nicheData, opts.location);
    printResults(nicheName, accounts, nicheData);
  }
} finally {
  await browser.close();
}
