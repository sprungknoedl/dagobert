---
name: Dagobert
description: A collaborative incident-response investigation workspace, built on vanilla daisyUI 5 + Tailwind 4.
colors:
  light:
    base-100: "#FFFFFF"
    base-200: "#F0F4F8"     # blue-grey-50
    base-300: "#D9E2EC"     # blue-grey-100
    base-content: "#102A43" # blue-grey-900
    primary: "#147D64"      # teal-700
    secondary: "#186FAF"    # blue-600
    accent: "#3EBD93"       # teal-400
    neutral: "#102A43"      # blue-grey-900
    info: "#486581"         # blue-grey-600
    warning: "#F0B429"      # yellow-500
    error: "#BA2525"        # red-500
  dark:
    base-100: "#243B53"     # blue-grey-800
    base-200: "#102A43"     # blue-grey-900
    base-300: "#334E68"     # blue-grey-700
    base-content: "#D9E2EC" # blue-grey-100
    primary: "#3EBD93"      # teal-400
    secondary: "#62B0E8"    # blue-300
    accent: "#65D6AD"       # teal-300
    neutral: "#D9E2EC"      # blue-grey-100
    info: "#9FB3C8"         # blue-grey-300
    warning: "#F7C948"      # yellow-400
    error: "#E66A6A"        # red-300
  palette: "Full ten-step scales (teal, blue-grey, cyan, blue, purple, red, yellow) live as
    CSS custom properties in internal/frontend/dagobert.css — see Colors below. Only the
    steps above are wired into the theme; the rest are there for later use."
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

A white-and-blue-grey surface with teal as the one structural/signal hue — vibrant rather than
muted, drawn from the "Refactoring UI" palette set (`tmp/input/` holds the source swatches). Every
hue comes as a full ten-step scale (`--teal-900` … `--teal-50`, and the same for blue-grey, cyan,
blue, purple, red, and yellow) declared once as CSS custom properties in
`internal/frontend/dagobert.css`, ahead of the theme blocks; only some steps are wired into a
theme's semantic role, the rest are there for later use (a chart series, a one-off badge, ...).

### The Teal Ladder

Two steps of the same hue, same idea as before: a darker/lower-chroma step for structure, a
brighter one for "look here."

| Role | Light step | Dark step |
|---|---|---|
| **Primary** — buttons, links, the active nav item, success | `teal-700` | `teal-400` |
| **Accent** — the one signal color: look here | `teal-400` | `teal-300` |

The dark theme's primary reuses the light theme's accent step: everything shifts one step
brighter on the dark ground, the ladder itself doesn't change.

### Roles

- **Primary** (`teal-700` light / `teal-400` dark) — the main structural color. Used for the
  active sidebar item's fill (`.menu-active`, the one deliberate custom rule left in
  `dagobert.css`), primary buttons/links, and success toasts/alerts (success is primary, not
  accent — it's the most frequent message in the product, and spending the brighter signal color
  on it would retire the signal).
- **Secondary** (`blue-600` light / `blue-300` dark) — used rarely, for secondary emphasis.
- **Accent** (`teal-400` light / `teal-300` dark) — the one signal color: record serials, focus
  outlines, flagged rows. It only works because it's rare.
- **Base** — `base-100` (white / `blue-grey-800`) is panel/card faces, `base-200` (`blue-grey-50` /
  `blue-grey-900`) is the page ground, `base-300` (`blue-grey-100` / `blue-grey-700`) is borders
  and inert fills, `base-content` (`blue-grey-900` / `blue-grey-100`) is body text.
- **Status.** Warning and error each get a real hue — `yellow-500`/`yellow-400` and
  `red-500`/`red-300` — picked as follows:
  - **Warning picks `yellow-500`** (`#F0B429`), the shade "Refactoring UI" itself calls the vivid
    one: mid-ramp, so it's saturated and cheerful rather than the muddy browns above it on the
    scale or the washed-out creams below it.
  - **Error picks `red-500`** (`#BA2525`) for the same reason — vivid rather than a muddy oxblood
    or a weak pink — and it's the darkest red that still clears 4.5:1 contrast against white
    content text (6.2:1 measured).
  - **Info stays a quiet neutral tint** (`blue-grey-600` / `blue-grey-300`) rather than a hue of
    its own. With warning promoted to real yellow, info is the one status left deliberately
    muted, so a color still means "genuinely needs you" rather than every status competing for
    attention.

### The dark theme

Not an inverted copy of the light theme. Two rules build the whole palette:

1. **The blue-grey scale flips ends.** Light mode runs from `blue-grey-900` (text) down to white
   (panel face); dark mode runs the same scale from `blue-grey-100` (text) down to `blue-grey-900`
   (page ground), with `blue-grey-800` as the slightly-lighter panel face — panels still read as
   sitting just above the ground, the same direction as the light theme.
2. **Every hue that appears moves one step brighter.** Primary, accent, secondary, info, warning,
   and error all pick the next lighter step of their own scale (e.g. error: `red-500` → `red-300`)
   rather than reusing the light-theme value verbatim — a saturated dark-toned color reads as a
   muddy hole against a near-black ground, so lightness moves even where contrast alone wouldn't
   require it.

An inverted fill (dark-on-light or light-on-dark) stays small — primary buttons, the one active
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
  against the theme's own color/radius tokens (`var(--color-*)`, not literals) — they pick up any
  future palette change automatically, without their own CSS being touched.

## Do's and Don'ts

- **Do** reach for a daisyUI component class before writing a utility stack.
- **Do** extract a named CSS class once the same utility cluster shows up a third time.
- **Don't** add a utility that restates what a daisyUI class already sets by default.
- **Don't** reintroduce a custom font, a bespoke elevation/shadow model, or a blanket "label voice"
  — those are the things this system deliberately reverted away from.
- **Don't** grow the "what's still custom" list above without a reason as specific as the ones
  already on it.
