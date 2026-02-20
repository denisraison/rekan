# PEP-004: Component Library

**Status:** Wave 1 Implemented
**Date:** 2026-02-20

## Context

The frontend has two components (LogoMark, LogoCombo) and a 950-line marketing page with all UI patterns (buttons, cards, nav, phone frames) defined as scoped CSS. There is no shared component library and no design system beyond CSS variables in `app.css`.

We are about to build the app UI (business profile form, content generation, dashboard). Building those screens on raw HTML + scoped CSS will duplicate the button styles, card patterns, and spacing conventions already embedded in the marketing page. We need a component library before the app pages start.

The approach: adopt **shadcn-svelte** (headless primitives from bits-ui, styled with Tailwind CSS) as the component foundation. Customize the theme to match the existing brand identity (coral/sage palette, Urbanist/Cormorant Garamond typography, rounded corners, lift-on-hover interactions). Migrate the marketing page from custom CSS to Tailwind so the project has one styling system going forward. Add **Histoire** for isolated component development and visual review.

## Token Mapping

The existing CSS variables map to shadcn-svelte's semantic token system and Tailwind's `@theme` (v4) as follows:

| Rekan token | Value | shadcn-svelte semantic | Tailwind name |
|---|---|---|---|
| --bg | #FAFAF7 | background | `bg-background` |
| --text | #111116 | foreground | `text-foreground` |
| --surface | #FFFFFF | card | `bg-card` |
| --text | #111116 | card-foreground | `text-card-foreground` |
| --coral | #F97368 | primary | `bg-primary` |
| white | #FFFFFF | primary-foreground | `text-primary-foreground` |
| --sage-pale | #F0F6F0 | secondary | `bg-secondary` |
| --text | #111116 | secondary-foreground | `text-secondary-foreground` |
| --coral-pale | #FFF3F2 | accent | `bg-accent` |
| --text | #111116 | accent-foreground | `text-accent-foreground` |
| --bg | #FAFAF7 | muted | `bg-muted` |
| --text-muted | #999AAA | muted-foreground | `text-muted-foreground` |
| --border | rgba(17,17,22,0.07) | border | `border-border` |
| --border-strong | rgba(17,17,22,0.14) | input | `border-input` |
| --coral | #F97368 | ring | `ring-ring` |

Additional brand tokens exposed as custom Tailwind colors (not part of shadcn semantics, used directly in marketing and brand-specific components):

| Token | Value | Tailwind usage |
|---|---|---|
| --coral-light | #FDDCDA | `bg-coral-light` |
| --coral-dark | #D4524A | `bg-coral-dark` |
| --sage | #87AA8C | `bg-sage` |
| --sage-light | #C8DEC9 | `bg-sage-light` |
| --sage-dark | #6B8A61 | `bg-sage-dark` |
| --dark | #111116 | `bg-dark` |
| --dark-muted | #1C1C24 | `bg-dark-muted` |
| --text-secondary | #555566 | `text-secondary-foreground` or `text-text-secondary` |

Typography stays on Google Fonts (Urbanist, Cormorant Garamond), configured in Tailwind as `font-primary` and `font-accent`.

Border radii map to Tailwind's radius scale: `--radius-sm` (6px), `--radius-md` (12px), `--radius-lg` (20px), `--radius-full` (9999px). shadcn-svelte's `--radius` base variable is set to 12px.

Shadows map to `shadow-sm`, `shadow-md`, `shadow-lg` with the current rgba(17,17,22) values.

## Wave 1: Tailwind + shadcn-svelte Foundation

**Goal:** Install Tailwind CSS v4, shadcn-svelte, and configure the brand theme. Verify the build works and existing pages are unaffected.

**Changes:**

`web/package.json`: Add `tailwindcss`, `@tailwindcss/vite`, `bits-ui`, `clsx`, `tailwind-merge`, `tailwind-variants`, `shadcn-svelte` (and its CLI for component scaffolding).

`web/vite.config.ts`: Add the Tailwind v4 Vite plugin.

`web/src/app.css`: Replace the current CSS variables and resets with a Tailwind v4 `@import "tailwindcss"` and `@theme` block containing all brand tokens mapped above. The resets (`box-sizing`, font smoothing, etc.) are handled by Tailwind's preflight.

`web/src/lib/utils.ts`: Add the `cn()` utility (standard shadcn-svelte pattern combining `clsx` + `tailwind-merge`).

`web/src/lib/components/ui/`: Create the shadcn-svelte component directory. Initially install: `button`, `card`, `badge`, `separator`. Customize each to use the brand's rounded corners, font weights, and hover interactions (the `translateY(-2px)` lift).

**Gate:** `pnpm check` passes. `pnpm build` succeeds. `pnpm dev` serves the existing marketing page without visual regressions (verify with Playwright screenshot comparison).

**Result (2026-02-20):** All gates passed. Tailwind v4.2.0 installed with `@tailwindcss/vite` plugin. Brand tokens mapped to shadcn semantic variables in `:root` and exposed via `@theme inline` for Tailwind utility access. Original CSS variables kept for backward compatibility with scoped styles in the marketing page. Preflight replaces manual reset. Global styles wrapped in `@layer base`. Four shadcn-svelte components installed (button, card, badge, separator). Marketing page screenshot confirms no visual regression.

## Wave 2: Component Extraction + Histoire

**Goal:** Extract marketing page patterns into reusable components. Set up Histoire for component development.

**Changes:**

`web/package.json`: Add `histoire` and `@histoire/plugin-svelte`.

`web/histoire.config.ts`: Configure Histoire with brand fonts and base styles.

Components to extract from the marketing page into `web/src/lib/components/`:

- **Button** (`ui/button`): Already installed from shadcn-svelte in Wave 1. Add the brand variants: `primary` (coral bg, white text, coral shadow), `ghost` (transparent, border), `white` (white bg, coral text). Include `sm` and `block` size variants. Preserve the `translateY(-2px)` hover lift and shadow enhancement.

- **SectionLabel**: The uppercase, letter-spaced, coral-colored label used across steps, showcase, pricing sections. Small utility component.

- **PhoneFrame**: The Instagram phone mockup used in hero and showcase. Props for width, content slot, optional rotation. Includes notch, border, shadow, hover shadow enhancement.

- **IgPost**: The Instagram post preview (avatar, username, post image area, actions bar, caption, hashtags). Used inside PhoneFrame. Props for username, initial, avatar color, post background, caption, hashtags, likes.

- **Container**: Max-width wrapper (1120px default, 1000px narrow variant) with horizontal padding. Replaces the repeated `max-width + margin: 0 auto + padding` pattern.

Each component gets a Histoire story file (`*.story.svelte`) showing its variants and states.

**Gate:** All extracted components render identically to the current marketing page. Histoire dev server shows all components with their variants. `pnpm check` passes.

## Wave 3: Marketing Page Migration

**Goal:** Rewrite the marketing page using Tailwind utility classes and the extracted components. Remove all scoped CSS from the page.

**Changes:**

`web/src/routes/(marketing)/+page.svelte`: Replace the 700 lines of scoped `<style>` with Tailwind classes on elements. Use the extracted components (Button, SectionLabel, PhoneFrame, IgPost, Container). The `reveal` action stays as-is (it adds a CSS class, works fine with Tailwind).

Animations (`fadeUp`, `meshFloat`, `floatCard`) move to `app.css` as `@keyframes` definitions, referenced via Tailwind's `animate-*` utilities or inline `animation` properties where needed.

Responsive breakpoints use Tailwind's `md:` prefix instead of the `@media (max-width: 768px)` block.

`web/src/app.css`: Remove any remaining legacy CSS that was only used by the marketing page.

**Gate:** Pixel-level comparison of the marketing page before and after migration using Playwright screenshots. No scoped `<style>` block remains in the marketing page (all styling via Tailwind classes or component props). `pnpm build` bundle size does not increase by more than 20KB gzipped. `pnpm check` passes.

## New Dependencies

| Package | Purpose | Size impact |
|---|---|---|
| tailwindcss | Utility CSS framework | Dev dep, CSS output only |
| @tailwindcss/vite | Vite integration for Tailwind v4 | Dev dep |
| bits-ui | Headless accessible Svelte primitives | Runtime, tree-shakes |
| clsx | Conditional class strings | ~300B |
| tailwind-merge | Merge conflicting Tailwind classes | ~5KB |
| tailwind-variants | Component variant API | ~3KB |
| histoire | Component development environment | Dev dep |
| @histoire/plugin-svelte | Svelte integration for Histoire | Dev dep |

shadcn-svelte components are copied into the project (not a dependency). The CLI is used once to scaffold them.

## Consequences

**One styling system.** The project moves from CSS variables + scoped CSS to Tailwind utilities + CSS variables (Tailwind v4 uses CSS variables natively via `@theme`). New pages and components use Tailwind exclusively.

**Accessible by default.** bits-ui provides keyboard navigation, focus management, ARIA attributes for interactive components (dialogs, dropdowns, tooltips, comboboxes). This matters for the app UI where we will have forms and modals.

**Larger initial learning surface.** Contributors need to know Tailwind class names. The trade-off is less custom CSS to maintain and a consistent API for spacing, colors, and responsive design.

**Component ownership.** shadcn-svelte components live in our repo (`lib/components/ui/`). We can modify them freely. No version lock-in to an external library.

**Marketing page gets longer HTML.** Tailwind classes on elements replace the scoped CSS block. The `<style>` section disappears, but element markup becomes more verbose. This is the standard Tailwind trade-off.

**Histoire adds a dev tool.** It runs as a separate dev server for component isolation. Not required for the app to work, but useful for reviewing components without navigating to specific pages.
