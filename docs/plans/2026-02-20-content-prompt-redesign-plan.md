# Content Prompt Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the template-driven content generation prompt with a voice-first prompt that produces natural-sounding Instagram posts.

**Architecture:** Single prompt rewrite in `eval/baml_src/content.baml`, followed by eval run to validate. If `brazilian_portuguese` heuristic regresses, add a minimal pt-BR nudge. No changes to judge prompts, Go code, or BAML types.

**Tech Stack:** BAML prompt (Jinja2 templates), Go eval pipeline, Gemini Flash via OpenRouter.

---

### Task 1: Replace the content.baml prompt

**Files:**
- Modify: `eval/baml_src/content.baml`

**Step 1: Replace the full prompt body**

Replace the entire prompt string in `GenerateContent` with:

```baml
function GenerateContent(profile: BusinessProfile) -> string {
  client GeneratorClient
  prompt #"
    Você é o(a) dono(a) do(a) {{ profile.businessName }}. Você mesmo(a) escreve os posts do Instagram do seu negócio. Sem agência, sem equipe de marketing. Escreve do jeito que fala.

    Escreva 3 posts pro seu Instagram. Cada post pronto pra copiar e colar, separados por ---.

    Seu negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade/Bairro: {{ profile.city }}, {{ profile.neighbourhood }}
    - Serviços: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}
    - Público: {{ profile.targetAudience }}
    - Vibe: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    Cada post precisa ter:
    - Legenda (máximo 700 caracteres)
    - Hashtags do nicho e da região (mínimo 5)
    - Um convite pra ação (chama no zap, link na bio, comenta, etc.)
    - Direção de foto/vídeo entre colchetes no final: o que montar, de que ângulo, em que momento [ex: Foto do bolo pronto na bancada, luz natural da janela]

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão
    - Descreva cenas que dá pra montar e fotografar: preparando um produto, arrumando o espaço, um resultado de serviço. A legenda acompanha a foto.
    - Emojis só quando você usaria de verdade no WhatsApp
    - Cada post com tom diferente: um mais sério, outro leve, outro informativo
    - Os 3 posts devem ter aberturas e estruturas diferentes entre si
    - Não repita o mesmo convite pra ação nos 3 posts
    - Mencione o nome do negócio em pelo menos 2 posts
    - Mencione a cidade ou bairro em pelo menos 1 post

    {{ ctx.output_format }}
  "#
}
```

**Step 2: Commit**

```bash
git add eval/baml_src/content.baml
git commit -m "Redesign content prompt: voice-first, drop rigid template"
```

---

### Task 2: Run eval and compare to baseline

**Files:** None (read-only evaluation)

**Step 1: Run eval with judges**

Run: `make eval-judges`

Expected: completes without errors, saves JSON to `eval/runs/`.

**Step 2: Read the new run JSON**

Find the latest file in `eval/runs/` and read `summary.judgeTotals`.

**Step 3: Compare to baseline**

Baseline (from `eval/runs/2026-02-19T20-20-50Z.json`):

| Criterion | Baseline |
|-----------|----------|
| naturalidade | 0 |
| variedade | 1 |
| engajamento | 3 |
| especificidade | 8 |
| acionavel | 12 |

Targets: naturalidade 6+, variedade 4+, engajamento 4+, especificidade 8+, acionavel 11+.

**Step 4: Check heuristic pass rate**

Read `summary.totalChecks` and `summary.passedChecks`. Baseline was 83/84. If significantly lower, check which heuristic is failing.

**Step 5: Specifically check `brazilian_portuguese` heuristic**

Search the run JSON for `"brazilian_portuguese"` entries with `"pass": false`. If more than 2 profiles fail, proceed to Task 3. Otherwise skip Task 3.

---

### Task 3: (Conditional) Add pt-BR nudge if heuristic regresses

Only run this task if Task 2 Step 5 shows 3+ `brazilian_portuguese` failures.

**Files:**
- Modify: `eval/baml_src/content.baml`

**Step 1: Add a minimal pt-BR instruction**

Add one line to the "Como você escreve" section:

```
    - Escreva em português brasileiro informal
```

Do NOT add the explicit marker list ("use gente, bora, né, tá"). Just the general instruction.

**Step 2: Re-run eval**

Run: `make eval-judges`

Verify `brazilian_portuguese` heuristic passes on 11+ profiles without regressing judge scores.

**Step 3: Commit**

```bash
git add eval/baml_src/content.baml
git commit -m "Add light pt-BR nudge to content prompt"
```

---

### Task 4: Run diff and summarize results

**Files:** None (read-only)

**Step 1: Run diff between baseline and final run**

```bash
cd eval && go run ./cmd/eval --diff runs/2026-02-19T20-20-50Z.json runs/<LATEST>.json
```

**Step 2: Report final results**

Print the full comparison table (heuristics + judges) and note which criteria improved, regressed, or stayed the same.
