---
name: rekan-ui
description: Prototype UI screens for Rekan as standalone mobile HTML/CSS, screenshot with Playwright, iterate until accessible. Use this skill whenever the user wants to design, prototype, or mockup any screen, flow, component, or layout for the Rekan app — even if they don't say "prototype" or "mockup". Trigger on: "design a screen", "design the [name] screen", "prototype this", "create a mockup", "ui for [feature]", "layout for", "rekan ui", "how should this screen look", "sketch out the [feature] flow", "what should the [page] look like". This skill is specifically tuned for Rekan's audience (Brazilian micro-entrepreneurs, 50+, low digital literacy) and brand system — always prefer it over generic frontend skills for any Rekan UI work.
---

# Rekan UI Prototyper

Design mobile-first screens for Rekan's target user: Brazilian micro-entrepreneurs (MEIs) who may not be very comfortable with technology — think a 55-year-old confeiteira or a 60-year-old barbeiro using a smartphone for the first time. Prototype in standalone HTML, screenshot with Playwright, and iterate until the design is clear, accessible, and low-friction.

## Who You're Designing For

The user is a Brazilian micro-entrepreneur, likely 40–65, running a small business (bakery, salon, barbershop, restaurant). They:
- Use WhatsApp daily but may struggle with apps
- Read slowly — long labels lose them
- Have large fingers and may miss small touch targets
- Get anxious when they're not sure what a button does
- Trust something that feels familiar and warm, not cold/corporate

Design as if the user will only get one glance before acting. If they have to re-read anything, simplify it.

## Brand System

Inline these as CSS variables in every prototype:

```css
:root {
  --coral: #F97368;
  --coral-light: #FDDCDA;
  --coral-pale: #FFF3F2;
  --sage: #87AA8C;
  --sage-light: #C8DEC9;
  --sage-pale: #F0F6F0;
  --charcoal: #111116;
  --text: #111116;
  --text-secondary: #555566;
  --text-muted: #999AAA;
  --bg: #FAFAF7;
  --surface: #FFFFFF;
  --border: rgba(17,17,22,0.07);
  --border-strong: rgba(17,17,22,0.14);
  --radius-sm: 6px;
  --radius-md: 12px;
  --radius-lg: 20px;
  --radius-full: 9999px;
}
* { font-family: 'Urbanist', system-ui, sans-serif; }
body { background: var(--bg); color: var(--text); margin: 0; width: 390px; min-height: 844px; overflow-x: hidden; }
```

Fonts (load in `<head>`):
```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Urbanist:wght@300;400;500;600;700&display=swap" rel="stylesheet">
<script src="https://cdn.tailwindcss.com"></script>
```

Tailwind is available for utilities, but always override with brand variables for colors, radii, and typography.

## Accessibility Rubric

Every prototype must pass all five criteria before it's done. Check these after every screenshot:

1. **Touch targets** — every button, input, and link is at least 48px tall and 44px wide. No exceptions for secondary actions.
2. **Text size** — body text at least 16px, primary action labels at least 18px, secondary labels at least 14px. Nothing smaller.
3. **Cognitive load** — one dominant action per screen. The user's eye lands on one thing to do. Secondary actions are visually quieter.
4. **Language** — all text in pt-BR, plain and direct. No English. No jargon ("autenticar", "configurar perfil", "onboarding" — replace with "entrar", "suas informações", "começar"). No truncation that hides meaning.
5. **55-year-old test** — imagine a bakery owner who has never used an app like this. Can they tell at a glance what this screen is for and what to do next? If there's any doubt, simplify.

## Iteration Loop

Repeat up to 5 times:

1. Write the HTML to `/tmp/rekan-ui/{slug}.html`
2. Screenshot it:
   ```bash
   mkdir -p /tmp/rekan-ui
   cd /home/denis/workspace/rekan/web && npx playwright screenshot --viewport-size="390,844" "file:///tmp/rekan-ui/{slug}.html" "/tmp/rekan-ui/{slug}.png"
   ```
3. Read the screenshot (use the Read tool — Claude has vision). Evaluate against all five rubric criteria.
4. If all five pass → done. If any fail → note exactly what failed and why, fix the HTML, go to step 2.

Don't aim for perfection on the first pass. Get a working layout up quickly, screenshot it, see what's wrong.

## HTML Template

Start from this base for every new screen:

```html
<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=390, initial-scale=1">
  <title>{Screen Name}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Urbanist:wght@300;400;500;600;700&display=swap" rel="stylesheet">
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    :root {
      --coral: #F97368; --coral-light: #FDDCDA; --coral-pale: #FFF3F2;
      --sage: #87AA8C; --sage-light: #C8DEC9; --sage-pale: #F0F6F0;
      --charcoal: #111116; --text: #111116; --text-secondary: #555566;
      --text-muted: #999AAA; --bg: #FAFAF7; --surface: #FFFFFF;
      --border: rgba(17,17,22,0.07); --border-strong: rgba(17,17,22,0.14);
      --radius-sm: 6px; --radius-md: 12px; --radius-lg: 20px; --radius-full: 9999px;
    }
    * { font-family: 'Urbanist', system-ui, sans-serif; box-sizing: border-box; }
    body { background: var(--bg); color: var(--text); margin: 0; width: 390px; min-height: 844px; overflow-x: hidden; }
  </style>
</head>
<body>
  <!-- content here -->
</body>
</html>
```

## Design Patterns for This Audience

**Inputs:** Large, with visible labels above (not placeholder-only). At least 52px tall. Rounded corners. Strong border on focus.

**Primary buttons:** Full-width or near-full-width at bottom of the action area. Coral background, white text, at least 52px tall, rounded-full. One per screen.

**Secondary actions:** Text-only or ghost style, smaller than primary, never competing visually.

**Navigation:** Bottom bar if multi-section. Large icons with labels (never icon-only). Active state in coral.

**Error states:** Inline, below the field. Red text with a short plain message. Never just a color change.

**Loading:** A simple spinner or "Aguarde..." message. Users get anxious if they don't know something is happening.

**Section headers:** Clear and short. 20–24px, font-weight 600. Helps users orient on longer screens.

**Spacing:** Generous. 16–24px padding on screen edges. 16–20px between related fields. 32px between sections. Don't pack things in.

**Icons:** Use text labels alongside any icon. Never icon-only for actions the user hasn't seen before.

## Svelte Porting Notes (after passing rubric)

After the prototype passes, give brief porting notes:
- Which existing Svelte components or patterns map to each section (check `web/src/routes/` and `web/src/lib/`)
- Any new components that would need to be created
- CSS variable names to use (they match the app's `app.css` system)
- State management needs (Svelte `$state`, stores, or server data)
- Any PocketBase API calls implied by the UI

## Output

When the prototype passes the rubric:
1. State which iteration it passed on
2. Show the final screenshot path: `/tmp/rekan-ui/{slug}.png`
3. List any significant design decisions and why
4. Provide the Svelte porting notes
5. Ask if the user wants to iterate further or port it to the app
