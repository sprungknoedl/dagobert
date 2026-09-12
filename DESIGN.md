---
name: Dagobert
description: A collaborative incident-response investigation workspace, built on vanilla daisyUI 5 + Tailwind 4.
colors:
  seal-green: "oklch(38% 0.082 162)"
  banknote-green: "oklch(50% 0.140 158)"
  assay-gold: "oklch(58% 0.100 85)"
  sheet-cream: "oklch(98.8% 0.010 92)"
  deep-cream: "oklch(95.5% 0.018 92)"
  rule-cream: "oklch(90% 0.026 90)"
  engraving-ink: "oklch(23% 0.030 165)"
  status-severe: "oklch(48% 0.170 25)"
  plate-ground: "oklch(17% 0.010 165)"
  plate-field: "oklch(21% 0.008 165)"
  plate-rule: "oklch(28% 0.012 165)"
  burnished-line: "oklch(90% 0.014 92)"
  verdigris: "oklch(78% 0.082 162)"
  signal-bright: "oklch(70% 0.148 158)"
  severe-bright: "oklch(66% 0.172 25)"
  assay-brass: "oklch(64% 0.105 85)"
rounded:
  hint: "0.125rem"
  field: "0.1875rem"
  box: "0.25rem"
  selector: "0.25rem"
  full: "999px"
typography: "Tailwind's default type scale (text-xs … text-2xl) and default font stack — no custom
  font family or size tokens. See Typography below."
---

> The frontmatter above is the source of truth for what's still custom (colors, radius/size
> tokens). Everything else — typography, spacing, component chrome — is vanilla daisyUI 5 +
> Tailwind 4, unmodified. The actual CSS is in `internal/frontend/dagobert.css`
> (`dagobert-vis.css` and `dagobert-choices.css` hold the third-party reskins), compiled to
> `internal/assets/dagobert.css`.

## Overview

Dagobert is built on stock daisyUI components (`card`, `btn`, `table`, `menu`, `badge`, `status`,
`drawer`, `modal`, `toast`/`alert`, `stats`/`stat`, `fieldset`/`label`, ...) at their default
styling. The only things that survive from the app's own design work are its color palette and its
radius/size theme tokens — both of which are just daisyUI's own theming mechanism (see daisyUI's
theme docs), not an addition on top of it.

This is a deliberate reversion from an earlier, fully custom "Counting House" design system
(self-hosted fonts, a bespoke elevation/ring model, a derived row-height formula, a mono
label voice on every heading) that grew large and never earned its keep — see
`docs/adr/counting-house-to-vanilla-daisyui.md`. Custom CSS should stay small: prefer a daisyUI
class over a utility, and a utility over new CSS.

## Rule: daisyUI first

1. **daisyUI semantic component classes first, always**, wherever one exists — `card`, `btn`,
   `table`, `menu`, `badge`, `status`, `drawer`, `modal`, `alert`, `toast`, `stats`/`stat`,
   `fieldset`/`label`, `input`/`select`/`textarea`, etc.
2. **No utility class or custom CSS may reproduce styling a daisyUI component class already
   provides** (background, text color, border, radius, shadow, padding that's part of that
   component's own default). If a daisyUI class covers it, use the daisyUI class, not a utility
   stacked on top of a bare element.
   - One recurring, deliberate exception: `.card` ships with no background color of its own (it's
     left to the page), so `card card-border bg-base-100` (or `bg-base-200` inside a menu) is not
     duplication — it's supplying what daisyUI leaves undefined.
3. **Tailwind utilities are the exception**, reserved for genuine one-off layout that no daisyUI
   class and no extracted semantic class covers (a specific flex alignment, a one-off gap).
4. **A layout pattern that recurs at 10+ call sites gets extracted into a small, named semantic
   CSS class** in `dagobert.css` instead of being restated as the same utility stack at every call
   site. Below that threshold it's either a one-off (rule 3) or gets stripped.
5. **Spacing between elements is not a class attribute's job.** Don't reach for `gap-`/`mb-`/`mt-`/
   `space-y-` to separate elements a daisyUI component's own spacing (`card-body`, `fieldset`,
   `menu`, `modal-action`, `stats`, ...) already provides. Restructure onto the owning component
   instead of bolting on a margin. Only a genuine one-off gap with no owning component is a rule-3
   utility.

## Colors

A cream-paper background with dark green-black ink. Seal green is the main color for structure and
trust. Banknote green is the only color used to say "look here." Red is used only for severity.
Colors are the one part of the earlier system carried over unchanged — every `oklch` value below
is exactly what `internal/frontend/dagobert.css` declares in both theme blocks.

### The Green Ladder

Three greens, all the same hue, different in lightness and intensity.

| Role | Chroma | On cream |
|---|---|---|
| **Ink** — everything you read | .030 | 16.2:1 |
| **Seal** — primary: edges, the active nav item, success | .082 | 9.3:1 |
| **Signal** — accent: look here | .140 | 5.3:1 |

Each step roughly doubles the color intensity; lightness goes up and contrast goes down as it
does. More intense color means less readability, so the most intense green is used the least, and
never for large blocks of text.

### Roles

- **Seal Green** (`primary`) — the main structural color. Used for the active sidebar item's fill
  (`.menu-active`, the one deliberate custom rule left in `dagobert.css`) and for success
  toasts/alerts. Its one filled use is small (one nav item); everywhere else daisyUI's own
  components carry it as an outline, a border, or text color.
- **Assay Gold** (`secondary`) — used rarely, for display-size emphasis only. Contrast on cream is
  4.17:1, so it must never carry body text.
- **Banknote Green** (`accent`) — the only signal color: record serials, focus outlines, flagged
  rows. It only works because it's rare.
- **Sheet Cream** (`base-100`) — panel/card faces. **Deep Cream** (`base-200`) — the page ground.
  **Rule Cream** (`base-300`) — borders and inert fills. **Engraving Ink** (`base-content`) — body
  text, never plain black.
- **Status.** Only severity gets its own color (`error`, mapped to red). `info` and `warning` are
  both mapped to the same ink value as `base-content` — not real hues — so a status only reads as
  "look at this" when it's genuinely severe; everything else renders through daisyUI's `status`
  component as a plain neutral dot next to its label word. This is why using `alert-warning` /
  `status-warning` "just works" without looking like a false alarm.

### The dark theme

Not an inverted copy of the light theme — the metal plate the page was printed from. Three rules
build the whole palette, still true today:

1. **The ink becomes the background.** Page and panel surfaces use Engraving Ink's own hue (165),
   darker and less saturated (0.008–0.012) than the ink itself (0.030), so the background reads as
   a dark neutral rather than a green room.
2. **The sheet becomes the text.** Text uses Sheet Cream's hue (92) at 90% lightness — legible, but
   not glaring.
3. **The ladder keeps its intensity, on a different background.** Seal stays at `.082`, Signal at
   `.140` — only lightness moves. Severity keeps its intensity too, so it reads red rather than
   drifting to salmon.

An inverted fill (dark-on-light or cream-on-dark) stays small — primary buttons, the one active
nav item — never a full-width band, which is why the active-item rule above is scoped to exactly
one small fill.

## Typography

Tailwind's default type scale and default font stack — no custom `--font-*` or `--text-*`
tokens, and no custom font files. Page titles, section headings, and body text all render in the
same system sans-serif family; monospace (`font-mono`, Tailwind's own default stack) is used only
where it's functionally useful — tabular figures, hashes, addresses, timestamps — never as a
"label voice" applied blanket-wide.

## Radius & size

`--radius-selector`, `--radius-field`, `--radius-box`, `--size-selector`, `--size-field`, and
`--border` are set in both theme blocks and kept exactly as configured — daisyUI's own theming
mechanism, giving the app its near-square corners (0.1875–0.25rem) rather than daisyUI's stock
rounder default. One smaller radius (`0.125rem`, "hint") exists outside that scale, used only by
the Choices.js chip/dropdown-item reskin in `dagobert-choices.css`.

## What's still custom, and why

Everything below is intentional, narrow, and lives in `internal/frontend/dagobert.css` (or its
imports). Nothing else in the CSS should grow past this list without a reason as specific as one
of these.

- **The button hover-contrast fix.** daisyUI's default button hover mixes toward black, which is
  nearly invisible once the button itself is dark (the dark theme). One rule mixes toward
  `base-content` instead, so the shading direction always contrasts with the surface on both
  themes.
- **The active nav item.** Fills with `primary` rather than daisyUI's default neutral fill —
  "where you are" earns the structural color. Scoped to `.menu-active`, one item at a time.
- **The flagged row / flagged action.** `.bg-flagged` / `.btn-flagged` mark a record flagged for
  review, in the accent color. No daisyUI component covers "tint this row and give it a left bar,"
  and it's a real product feature, not decoration.
- **The sortable column header.** `.sort` is the click/hover affordance List.js binds to for
  client-side table sorting, plus the ▲/▼ indicator. Functional, not decorative.
- **Unpoly's overlay boxes** (`up-drawer-box`, `up-modal-box`, `up-cover-box`, `up-popup`,
  `.dropdown-content`) aren't daisyUI components and carry none of daisyUI's own surface styling,
  so they get daisyUI's own modal-box recipe (`bg-base-100`, `rounded-box`, `shadow-xl`) applied
  directly.
- **The long-form drawer frame** (`.form-sheet` / `.form-head` / `.form-body` / `.form-foot`). A
  drawer holding a genuinely long form (Case, Evidence, ...) gets a fixed head and foot so the
  title and Save button never scroll out of view while the fields between them scroll. Reserved
  for actual long forms — a two-button confirm or a short dialog just renders as plain content in
  Unpoly's own padded box.
- **The brand mark.** An engraved portrait of a top-hatted duck in a circular medallion
  (`.medallion`), in the sidebar at 32px and in empty states at 96–112px. The
  only picture in the whole app; it doesn't change between themes (its own cream paper and ink are
  baked into the file). Its ring is a plain `box-shadow`, simplified from an earlier three-layer
  ring to two — full visual redesign of the mark for the plainer look is a separate, future task.
- **Third-party library reskins** — `dagobert-vis.css` (vis-timeline event histogram, vis-network
  lateral-movement graph) and `dagobert-choices.css` (the Choices.js multi-select). None of these
  libraries ship a stylesheet that fits a themed app, so they're skinned property-by-property
  against the surviving color/radius tokens. Not rewritten in spirit — same sheet-paper-and-ink
  look as the rest of the app, just pointed at what still exists.

## Do's and Don'ts

- **Do** reach for a daisyUI component class before writing a utility stack.
- **Do** extract a named CSS class once the same utility cluster shows up a third time.
- **Don't** add a utility that restates what a daisyUI class already sets by default.
- **Don't** reintroduce a custom font, a bespoke elevation/shadow model, or a blanket "label voice"
  — those are the things this system deliberately reverted away from.
- **Don't** grow the "what's still custom" list above without a reason as specific as the ones
  already on it.
