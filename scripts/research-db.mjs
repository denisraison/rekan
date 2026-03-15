#!/usr/bin/env node
/**
 * Import caption research JSON into SQLite for querying.
 *
 * Usage:
 *   node ../scripts/research-db.mjs                  # import data.json -> research.db
 *   node ../scripts/research-db.mjs --query "SELECT * FROM captions WHERE niche='confeitaria' ORDER BY char_count"
 *   node ../scripts/research-db.mjs --summary         # print niche-level stats
 */

import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { readFileSync, existsSync } from "node:fs";
import { execSync } from "node:child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const DATA_PATH = resolve(__dirname, "../docs/caption-research/data.json");
const DB_PATH = resolve(__dirname, "../docs/caption-research/research.db");

const args = process.argv.slice(2);
const queryArg = args.find((a) => a.startsWith("--query="))?.split("=").slice(1).join("=")
  || (args.indexOf("--query") > -1 ? args[args.indexOf("--query") + 1] : null);
const summary = args.includes("--summary");

function sql(query) {
  try {
    return execSync(`sqlite3 -header -column "${DB_PATH}" "${query.replace(/"/g, '\\"')}"`, {
      encoding: "utf8",
      maxBuffer: 10 * 1024 * 1024,
    });
  } catch (e) {
    return e.stdout || e.message;
  }
}

function sqlExec(query) {
  execSync(`sqlite3 "${DB_PATH}" "${query.replace(/"/g, '\\"')}"`, { encoding: "utf8" });
}

// -- Import -----------------------------------------------------------------

function importData() {
  if (!existsSync(DATA_PATH)) {
    console.error(`No data file at ${DATA_PATH}. Run discover-and-scrape.mjs first.`);
    process.exit(1);
  }

  const data = JSON.parse(readFileSync(DATA_PATH, "utf8"));

  // Create tables
  const schema = `
    DROP TABLE IF EXISTS accounts;
    DROP TABLE IF EXISTS captions;

    CREATE TABLE accounts (
      username TEXT PRIMARY KEY,
      niche TEXT NOT NULL,
      followers TEXT,
      followers_num INTEGER DEFAULT 0,
      posts_scraped INTEGER DEFAULT 0
    );

    CREATE TABLE captions (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      username TEXT NOT NULL,
      niche TEXT NOT NULL,
      url TEXT,
      caption TEXT,
      char_count INTEGER DEFAULT 0,
      word_count INTEGER DEFAULT 0,
      is_reel BOOLEAN DEFAULT 0,
      likes TEXT,
      comments TEXT,
      timestamp TEXT,
      image_path TEXT,
      -- Computed analysis fields
      has_hashtags BOOLEAN DEFAULT 0,
      hashtag_count INTEGER DEFAULT 0,
      has_cta BOOLEAN DEFAULT 0,
      cta_type TEXT,
      has_line_breaks BOOLEAN DEFAULT 0,
      has_price BOOLEAN DEFAULT 0,
      has_question BOOLEAN DEFAULT 0,
      emoji_count INTEGER DEFAULT 0,
      opens_with_question BOOLEAN DEFAULT 0,
      opens_with_exclamation BOOLEAN DEFAULT 0,
      opens_with_scene BOOLEAN DEFAULT 0,
      has_dash BOOLEAN DEFAULT 0,
      likes_num INTEGER DEFAULT 0,
      comments_num INTEGER DEFAULT 0,
      engagement_rate REAL DEFAULT 0,
      FOREIGN KEY (username) REFERENCES accounts(username)
    );

    CREATE INDEX idx_captions_niche ON captions(niche);
    CREATE INDEX idx_captions_username ON captions(username);
    CREATE INDEX idx_captions_char_count ON captions(char_count);
  `;

  execSync(`sqlite3 "${DB_PATH}"`, { input: schema, encoding: "utf8" });

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

  // Insert data
  const emojiRe = /[\u{1F300}-\u{1FAFF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/gu;

  for (const account of data) {
    const u = account.username.replace(/'/g, "''");
    const n = (account.group || "unknown").replace(/'/g, "''");
    const f = (account.followers || "").replace(/'/g, "''");
    const fNum = parseMetric(account.followers);
    sqlExec(`INSERT OR REPLACE INTO accounts VALUES('${u}','${n}','${f}',${fNum},${account.postsScraped || 0})`);

    for (const post of account.posts || []) {
      if (!post.caption) continue;

      const caption = post.caption.replace(/'/g, "''");
      const ending = post.caption.slice(-200).toLowerCase();
      const opening = post.caption.slice(0, 80);

      // Analyze
      const hashtags = post.caption.match(/#\w+/g) || [];
      const emojis = post.caption.match(emojiRe) || [];

      let ctaType = null;
      let hasCta = false;
      if (ending.match(/whatsapp|zap|chama no/)) { ctaType = "whatsapp"; hasCta = true; }
      else if (ending.match(/link.*bio|na bio/)) { ctaType = "link_bio"; hasCta = true; }
      else if (ending.match(/coment[ae]/)) { ctaType = "comenta"; hasCta = true; }
      else if (ending.match(/marca\s/)) { ctaType = "marca"; hasCta = true; }
      else if (ending.match(/salv[ae]/)) { ctaType = "salva"; hasCta = true; }
      else if (ending.match(/corre|aproveite|garanta|não perca/)) { ctaType = "urgency"; hasCta = true; }

      const url = (post.url || "").replace(/'/g, "''");
      const imgPath = (post.imagePath || "").replace(/'/g, "''");
      const likes = (post.likes || "").replace(/'/g, "''");
      const comments = (post.comments || "").replace(/'/g, "''");
      const ts = (post.timestamp || "").replace(/'/g, "''");
      const ctaVal = ctaType ? `'${ctaType}'` : "NULL";

      const likesNum = parseMetric(post.likes);
      const commentsNum = parseMetric(post.comments);
      const followersNum = parseMetric(account.followers);
      const engRate = followersNum > 0 ? ((likesNum + commentsNum) / followersNum * 100) : 0;

      sqlExec(`INSERT INTO captions (
        username, niche, url, caption, char_count, word_count, is_reel,
        likes, comments, timestamp, image_path,
        has_hashtags, hashtag_count, has_cta, cta_type,
        has_line_breaks, has_price, has_question, emoji_count,
        opens_with_question, opens_with_exclamation, opens_with_scene, has_dash,
        likes_num, comments_num, engagement_rate
      ) VALUES (
        '${u}', '${n}', '${url}', '${caption}', ${post.charCount || 0}, ${post.wordCount || 0}, ${post.isReel ? 1 : 0},
        '${likes}', '${comments}', '${ts}', '${imgPath}',
        ${hashtags.length > 0 ? 1 : 0}, ${hashtags.length}, ${hasCta ? 1 : 0}, ${ctaVal},
        ${post.caption.includes("\n") ? 1 : 0}, ${post.caption.match(/R\$\s*[\d,.]+/) ? 1 : 0}, ${post.caption.includes("?") ? 1 : 0}, ${emojis.length},
        ${opening.includes("?") ? 1 : 0}, ${opening.includes("!") ? 1 : 0}, ${opening.match(/^(Hoje|Ontem|Essa|Quando|Aqui|Aquele|Agora)/i) ? 1 : 0}, ${post.caption.match(/[—–]/) ? 1 : 0},
        ${likesNum}, ${commentsNum}, ${engRate.toFixed(4)}
      )`);
    }
  }

  const accountCount = sql("SELECT COUNT(*) as n FROM accounts").trim();
  const captionCount = sql("SELECT COUNT(*) as n FROM captions").trim();
  console.log(`Imported into ${DB_PATH}`);
  console.log(accountCount);
  console.log(captionCount);
}

// -- Summary ----------------------------------------------------------------

function printSummary() {
  console.log("\n## Caption Length by Niche\n");
  console.log(sql(`
    SELECT niche,
           COUNT(*) as posts,
           COUNT(DISTINCT username) as accounts,
           ROUND(AVG(char_count)) as avg_chars,
           ROUND(AVG(word_count)) as avg_words,
           MIN(char_count) as min_c,
           MAX(char_count) as max_c
    FROM captions
    GROUP BY niche
    ORDER BY niche
  `));

  console.log("\n## CTA Usage by Niche\n");
  console.log(sql(`
    SELECT niche,
           COUNT(*) as total,
           SUM(has_cta) as with_cta,
           ROUND(100.0 * SUM(has_cta) / COUNT(*)) as cta_pct
    FROM captions
    GROUP BY niche
  `));

  console.log("\n## Hashtag Usage by Niche\n");
  console.log(sql(`
    SELECT niche,
           ROUND(AVG(hashtag_count), 1) as avg_hashtags,
           SUM(has_hashtags) as posts_with_hashtags,
           COUNT(*) as total
    FROM captions
    GROUP BY niche
  `));

  console.log("\n## Emoji Density by Niche\n");
  console.log(sql(`
    SELECT niche,
           ROUND(AVG(emoji_count), 1) as avg_emojis
    FROM captions
    GROUP BY niche
  `));

  console.log("\n## Structure Patterns\n");
  console.log(sql(`
    SELECT
      ROUND(100.0 * SUM(has_line_breaks) / COUNT(*)) as pct_line_breaks,
      ROUND(100.0 * SUM(has_price) / COUNT(*)) as pct_prices,
      ROUND(100.0 * SUM(has_question) / COUNT(*)) as pct_questions,
      ROUND(100.0 * SUM(has_dash) / COUNT(*)) as pct_dashes,
      COUNT(*) as total
    FROM captions
  `));

  console.log("\n## Overall\n");
  console.log(sql(`
    SELECT
      COUNT(*) as total_captions,
      COUNT(DISTINCT username) as total_accounts,
      ROUND(AVG(char_count)) as avg_chars,
      ROUND(AVG(word_count)) as avg_words
    FROM captions
  `));
}

// -- Main -------------------------------------------------------------------

// Always re-import on run
importData();

if (queryArg) {
  console.log(sql(queryArg));
} else if (summary) {
  printSummary();
} else {
  printSummary();
}
