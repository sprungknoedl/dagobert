---
name: Dagobert
description: A collaborative incident-response investigation workspace, rendered as a counting house ledger.
colors:
  seal-green: "oklch(38% 0.082 162)"
  banknote-green: "oklch(50% 0.140 158)"
  assay-gold: "oklch(58% 0.100 85)"
  sheet-cream: "oklch(98.8% 0.010 92)"
  deep-cream: "oklch(95.5% 0.018 92)"
  rule-cream: "oklch(90% 0.026 90)"
  engraving-ink: "oklch(23% 0.030 165)"
  status-severe: "oklch(48% 0.170 25)"
  status-ink: "oklch(23% 0.030 165 / 60%)"
  plate-ground: "oklch(17% 0.020 165)"
  plate-field: "oklch(21% 0.022 165)"
  plate-rule: "oklch(28% 0.024 165)"
  burnished-line: "oklch(90% 0.014 92)"
  verdigris: "oklch(78% 0.082 162)"
  signal-bright: "oklch(70% 0.148 158)"
  severe-bright: "oklch(66% 0.172 25)"
  assay-brass: "oklch(64% 0.105 85)"
typography:
  display:
    fontFamily: "Fraunces Variable, Georgia, serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.1
    fontVariation: "'opsz' 96, 'WONK' 0"
  body:
    fontFamily: "Hanken Grotesk Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  data:
    fontFamily: "JetBrains Mono Variable, ui-monospace, Menlo, monospace"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.45
  label:
    fontFamily: "JetBrains Mono Variable, ui-monospace, Menlo, monospace"
    fontSize: "0.625rem"
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: "0.14em"
rounded:
  hint: "0.125rem"
  field: "0.1875rem"
  box: "0.25rem"
  selector: "0.25rem"
  full: "999px"
spacing:
  cell-y: "0.5rem"
  cell-x: "0.75rem"
  row-height: "2.5rem"
  row-control: "1.5rem"
  nav-item: "0.375rem 0.5rem"
  panel-pad: "0.75rem 1rem"
  section-gap: "1rem"
  ledger-grid: "28px"
  sidebar-width: "11rem"
components:
  button-primary:
    backgroundColor: "{colors.engraving-ink}"
    textColor: "{colors.sheet-cream}"
    typography: "{typography.body}"
    rounded: "{rounded.field}"
    padding: "0.375rem 0.75rem"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.engraving-ink}"
    typography: "{typography.body}"
    rounded: "{rounded.field}"
    padding: "0.375rem 0.5rem"
  input-search:
    backgroundColor: "{colors.sheet-cream}"
    textColor: "{colors.engraving-ink}"
    typography: "{typography.data}"
    rounded: "{rounded.field}"
    padding: "0.25rem 0.5rem"
  nav-item:
    backgroundColor: "transparent"
    textColor: "{colors.engraving-ink}"
    typography: "{typography.body}"
    rounded: "{rounded.field}"
    padding: "{spacing.nav-item}"
  nav-item-active:
    backgroundColor: "{colors.engraving-ink}"
    textColor: "{colors.sheet-cream}"
    typography: "{typography.body}"
    rounded: "{rounded.field}"
    padding: "{spacing.nav-item}"
  panel:
    backgroundColor: "{colors.sheet-cream}"
    textColor: "{colors.engraving-ink}"
    rounded: "{rounded.box}"
    padding: "{spacing.panel-pad}"
  table-row:
    backgroundColor: "transparent"
    textColor: "{colors.engraving-ink}"
    typography: "{typography.data}"
    height: "{spacing.row-height}"
    padding: "{spacing.cell-y} {spacing.cell-x}"
  status-dot:
    rounded: "{rounded.full}"
    size: "9px"
  toast-success:
    backgroundColor: "{colors.sheet-cream}"
    textColor: "{colors.engraving-ink}"
    typography: "{typography.body}"
    rounded: "{rounded.box}"
    padding: "0.625rem 0.75rem"
  key-hint:
    backgroundColor: "transparent"
    textColor: "{colors.engraving-ink}"
    typography: "{typography.label}"
    rounded: "{rounded.hint}"
    padding: "0.0625rem 0.25rem"
---

# Design System: Dagobert

> The values in the frontmatter above are the source of truth — the app is built on them, and new
> screens should follow them too.
> The actual CSS is in `internal/frontend/dagobert.css` (compiled to `internal/assets/dagobert.css`).

## Overview

**Creative idea: "The Counting House"**

A counting house is a place where people check numbers and make records add up. It's a quiet room
with good light and ruled paper — not a trading floor, not a busy operations center. The app
should feel like that kind of room.

The design looks like engraved paper. A near-white sheet sits on a slightly darker background with
a faint grid line pattern. Panels look cut from the sheet, with a thin outline. The layout is dense
on purpose — someone comparing events across many hosts wants to see many rows, not extra padding.

There are two themes. **The Sheet** is the light theme: dark text on cream-colored paper. **The
Plate** is the dark theme, named after the metal plate used to print an engraving — same drawing,
but now the ink color is the background and the lines are the light parts. The Plate is built from
the Sheet's own rules, not just an inverted copy of it. See "The Plate" under Colors for the three
rules that create it.

**Key points:**
- Looks like engraved paper, not glass or a typical dashboard
- Text style carries meaning: serif for page titles, grotesk (sans-serif) for regular text,
  monospace for every label
- Dense by design: 40px rows, 13px text size for data — these numbers come from a formula, not
  personal taste
- One family of green, in different strengths; red is the only other color used
- No shadows anywhere; instead of shadows, a thin line marks depth
- No decoration. One image — the engraved duck — appears in exactly two places and nowhere else

## Colors

A cream-paper background with dark green-black ink. Seal green is the main color for structure and
trust. Banknote green is the only color used to say "look here." Red is used only for severity.

### The Green Ladder

The core color idea. Three greens, all the same hue, but different in lightness and intensity.

| Role | Chroma | On cream |
|---|---|---|
| **Ink** — everything you read | .030 | 16.2:1 |
| **Seal** — edges, certification, done | .082 | 9.3:1 |
| **Signal** — look here | .140 | 5.3:1 |

**The Chroma Ladder Rule.** Each step roughly doubles the color intensity; lightness goes up and
contrast goes down as it does. More intense color means less readability, so the most intense
green is used the least, and never for large blocks of text. If a new color doesn't fit on this
ladder, it probably doesn't belong in the system.

### Primary
- **Seal Green**: the main structural color. Used for panel edge rings, the medallion ring, and
  success messages. Almost never fills a whole area — it outlines and marks things as confirmed.

### Secondary
- **Assay Gold**: used rarely, for emphasis in headings and sample labels. It's the warmest color
  and the easiest one to overuse. Its contrast on cream is only 4.17:1, so **it must never be used
  for body text** — only for large display text and marks.

### Tertiary
- **Banknote Green**: the only color used as a signal. Used for record serial numbers, the
  keyboard cursor row, focus outlines, and items flagged for review. It only works because it's
  rare. A serial number is something the analyst reads and quotes directly, which is why it gets
  the signal color.

### Neutral
- **Sheet Cream**: the surface color for panels and sheets — what content sits on.
- **Deep Cream**: the page background behind the sheets, with the ledger grid on it.
- **Rule Cream**: heavier dividing lines and plain fills.
- **Engraving Ink**: all body text, and the fill color for active navigation items and primary
  buttons. A green-black color, never plain black.

### Status

Only two status colors, not five. **Severe** is the only status that gets its own color —
confirmed indicators, TLP:RED, destructive actions. **Status Ink** (ink at 60% strength) is used
for every other status: Suspicious, Investigating, Unrelated, Cleared — these are told apart by
their label text, not by color.

Caution, Clear, and Note colors are **removed from the system**, not just unused right now.
"Cleared" is just another non-severe status, not its own color.

### Named Rules

**The Dot-and-Word Rule.** A status is always a 9px dot *and* the word together. Never color
alone, never a pill shape, never a filled row. A table of forty indicators shouldn't look like a
pile of random colors.

**The Ink States Rule.** Only the severe status has its own color; every other status is a plain
ink-colored dot. At 9px, red and amber look the same, and people with red-green color blindness
can't tell them apart at any size. So severity is shown as *color versus no color*, not as one hue
versus another. This has a useful side effect: most of a table is plain ink, so the one red mark
actually stands out and means something.

**The Scarce Signal Rule.** Banknote Green is the only color allowed to mean "look here." It's
never used for decoration. If a screen uses it more than three or four times, it stops being a
signal. Signal (green) and severity (red) do different jobs — never mix them.

### The Plate (dark theme)

Not just an inverted copy of the light theme — think of it as the metal plate the page was printed
from. Three rules build the whole dark-theme palette. A new dark-theme color that doesn't come
from one of these rules doesn't belong in the theme.

1. **The ink becomes the background.** The page and panels use Engraving Ink's own hue (165), just
   made darker. No black, no plain grey — the dark theme is built from the same color the light
   theme writes with.
2. **The sheet becomes the text.** All text uses Sheet Cream's hue (92) at 90% lightness — bright
   enough to read clearly, but dim enough not to glare across a page of small monospace text.
   Full-brightness cream would glare.
3. **The ladder keeps its intensity, just on a different background.** Seal stays at exactly
   `.082` and Signal at roughly `.140` — only the lightness moves to the bright side. The Chroma
   Ladder Rule still applies; only the direction changes.

Severity keeps its intensity so it stays red instead of drifting toward salmon pink. Assay Gold
becomes a brass color and stays the lowest-ranked color; here it passes 4.5:1 contrast (it fails on
cream), but it's still restricted to display-size text on both themes, so the rule doesn't depend
on which theme is active.

**The Small Fill Rule.** An inverted fill on the light theme is dark ink on a bright page — it
reads like a printed header. On the dark theme, that same inversion becomes cream-colored, and a
full-width band of that would glare — defeating the whole point of choosing a dark theme. So
inverted fills stay small on the dark theme: primary buttons, the active sidebar item. Anything
full-width sinks into the background instead.

**The Asymmetric Dim Rule.** Opacity percentages don't carry over directly between themes — a
light mark on a dark background reads with more contrast than the same percentage looks like on a
light background. So dimmed values go **down** on the dark theme (labels 45% → 40%) to keep the
same visual hierarchy, while thin structural lines go **up** (12% → 14%, row lines 7% → 8%),
because a thin light line on near-black is the first thing a screen loses. The engraved ring gets
brighter for the same reason; the raised ring gets dimmer, since it needs less help. The ledger
grid is tuned by eye, not by strict measurement (4% on the dark theme vs 3% on the light theme) —
the standard contrast formula (WCAG) doesn't measure near-black well.

**What the dark theme is not.** No neon on black, no cyan, no glow, no green-on-black terminal
look. It's meant to feel like a quiet, warm object. Building the theme around the printing
process, instead of the usual "security tool" look, is what keeps it from feeling like a war room
(see Product Principle 1).

## Typography

**Display Font:** Fraunces Variable (with Georgia, serif), at `opsz 96, WONK 0`
**Body Font:** Hanken Grotesk Variable (with ui-sans-serif, system-ui)
**Label/Mono Font:** JetBrains Mono Variable (with ui-monospace, Menlo)

**Character:** An engraved-style serif for titles, a plain sans-serif (grotesk) for everyday text,
and monospace for all labels. The serif is used rarely and only at small sizes — this should feel
like a working document with a well-designed title, not a magazine spread. All three fonts are
self-hosted (`.woff2` files) and are a long-term brand commitment.

### Hierarchy

The exact sizes and weights are set in the frontmatter above. Four sizes, not six — here's what
each is for and where it comes from:

- **Display** (24px): page titles, and the one large sentence shown on an empty page. Lands exactly
  on Tailwind's own `text-2xl`, so it isn't a custom token — just that utility plus the serif font.
- **Body** (14px): all regular text, navigation items, button labels, toast messages. Lands exactly
  on Tailwind's own `text-sm`, same reasoning as Display.
- **Data** (13px): table cells, indicator values, hashes. Always uses tabular numbers
  (`font-variant-numeric: tabular-nums`) for anything that can be counted. Falls between Tailwind's
  own steps (nothing sits at 13px), so it's the custom `--text-data` token.
- **Label** (10px, uppercase, 45% ink): section headings in the sidebar, small captions, metadata
  lines, table column headers, and keyboard hints. Also outside Tailwind's native scale, so it's the
  custom `--text-label` token — the smallest text size anywhere in the product.

A 9px Column Header and an 11px Timestamp size used to be documented as separate tiers here.
Neither was ever wired to real CSS: table headers already rendered at the Label size, and
timestamp cells just use Data. Both are gone now rather than fixed forward, so **10px (Label) is
the floor** — nothing in the app renders smaller.

### Named Rules

**The Mono Label Rule.** Every label, table header, identifier, count, and keyboard hint is
monospace, uppercase, with 0.14em letter spacing, all at the one Label size. No exceptions — this
one rule is what makes the design recognizable even without any color.

**The Tabular Number Rule.** Any number someone might compare down a column uses tabular
(fixed-width) digits: counts, timestamps, file sizes, evidence totals.

**The Label Opacity Exception.** The normal label style is 45% ink, with a contrast ratio of
2.8:1. That's fine for a caption next to the number it describes, but not fine when it's the only
text naming a form field. Form labels keep the same monospace style but use 78% ink instead (82%
on the dark theme). Anywhere a label is the only name for a control someone has to use, it gets
the more readable value.

## Layout

A fixed 176px (`--sidebar-w`) sidebar with labels, sticky and full height, next to a flexible main
column with 1rem padding and 1rem gaps between sections. The sidebar is grouped under small
monospace headings — workspace, the active case, analysis — with live counts right-aligned in each
item.

The page background has a **28px grid pattern**, made of two 1px lines at 3% ink. It's meant to be
barely visible — it should look like paper texture, never like a visible table. This is a
deliberate part of the visual identity, not random decoration.

Table rows are 40px tall, with thin 7%-ink dividing lines and sticky column headers. The summary
strip above the table shows only numbers, on a panel, with groups of numbers separated by thin
1px vertical lines at 12% ink.

The design targets desktop screens. There's no responsive breakpoint system, and small-screen
support is intentionally not a priority.

### Spacing

Every padding, margin, and gap in the product is a multiple of Tailwind's own base unit (4px) —
there's no separate spacing system layered on top of it. Most everyday spacing is picked from a
small subset of that scale (4/8/12/16/24/32/48/64px); reaching further down the scale is for
optical corrections, not a default.

A handful of values that get read from more than one place are real CSS custom properties instead
of restated numbers, so they can't drift apart from each other: `--cell-y`/`--cell-x` (grid cell
padding), `--row-h` (grid row height), and `--sidebar-w` (the rail's width). Every page's main
column offsets its content with the `.rail-offset` class — the rail's width plus the same 1rem edge
padding every other side of the column already has, so content gets a normal margin of breathing
room next to the rail rather than sitting flush against it. That relationship is a class in
`dagobert.css`, not a value restated in each page's markup, for the same reason `--sidebar-w` is a
variable and not five copies of "176px": before this, the rail and the offset were two
independently hand-picked values that happened to differ by exactly that 1rem. Everything else in
the spacing frontmatter above documents a value in consistent
use, not a literal variable.

### Named Rules

**The Density Rule.** 40px rows and 13px data text. Density is the goal — anything that shows
fewer rows per screen needs a better reason than "it's more comfortable." But the row height is
**calculated, not just picked**:

```
row = max(text line box, tallest control) + 2 x cell-y

     text     13px x 1.45          = 19px
     control  24px (WCAG 2.2 min
              pointer target)      = 24px   <- governs
     padding  8px + 8px            = 16px
                                   = 40px
```

The control size sets the row height, not the text — this is the part that's easy to get wrong.
`row-height` is a **minimum** height, so if the content is taller, it simply wins: an earlier
attempt at 28px actually rendered as 33px. That's why row actions stay at the 24px control size —
switching to a 32px button would push every row in the app to 48px. Recalculate this whenever text
size, control size, or cell padding changes.

A cell should never wrap text just to save a column — give the value its own column instead. (This
is why TLP and event counts are columns, not a second line of text.) Wrapping adds height to every
single row, while a column only costs width once. This rule applies to the *working* screens; an
empty state has no rows to worry about, so it can take whatever space it needs.

## Elevation & Depth

**The design uses no shadows.** Depth is shown with a second thin line instead. Elements are
separated by ruled edges and a double-line "engraved" ring — the way a printed document shows
depth, which a drop shadow would work against.

- **Hairline** (`1px` at 12% ink): default panel and divider separation.
- **Heavy hairline** (`1px` at 22% ink): column-header underline, emphasis rules.
- **Engraved edge** (`0 0 0 1px sheet-cream, 0 0 0 2px seal-green/14%`): the panel outline.
- **Raised engraved edge** (same, at 26%): elements sitting above the page — toasts, overlays,
  popovers.
- **Cursor rule** (`inset 2px 0 0 banknote-green`): the keyboard cursor row's left edge.

On the dark theme these become 14%, 24%, 8%, 18%, and 24%, following The Asymmetric Dim Rule.
Having no shadows is what makes switching themes work well: a shadow system built for a bright
page doesn't translate to a dark one, but a thin line just changes which side of the background it
sits on.

## Shapes

Corners are almost square everywhere — only status dots and other true circles are fully round.
Borders are 1px, and there's no texture beyond the ledger grid. Nothing uses a pill shape, and no
corner radius is large enough to look soft. The overall look is straight-edged and ruled.

## Components

### Brand Mark

An engraved line-art portrait of a duck wearing a top hat — the Dagobert — inside a circular
medallion with a seal-green ring. This is the only picture in the whole design, and it matters:
without it, the design would still work as ruled paper and monospace, but it would stop being
*this* product.

It appears in exactly two places: **the sidebar**, at 32px next to the logo text, always visible;
and **empty states**, at 96–112px, where the medallion has a heavier triple ring (18% ink, a 4px
cream gap, then 12% ink). The medallion's background is always set to Sheet Cream directly, not
inherited from the image file.

**The Only Paper Rule.** The mark doesn't change between themes. Same artwork, same filter — only
the blend mode changes, from `multiply` on the light theme to `normal` on the dark theme, where
there's no warm color underneath to blend into. It stays itself: a warm print pinned onto the
page, and the one genuinely bright object on a dark screen. It's exempt from The Small Fill Rule,
for the same reason that rule exists — what causes glare is an inverted *surface*, and this mark
is an object that looks like a stamped coin, not a surface. Inverting the artwork itself was tried
and rejected: an engraving has its own light-and-dark logic (the top hat is the darkest part), so
flipping it turns the hat white and the portrait looks like an x-ray. **The theme inverts the
design system, not the artwork** — any future image should get the same treatment.

**The Sparing Mark Rule.** The portrait matters *because* it's rare — it only lives in permanent
UI chrome and empty states, the two places where decoration doesn't cost any table rows.
Decoration on the *working* screens was rejected: a microtext band and a decorative strip pattern
were both built and tested on screen, then removed for taking up a full-width band on every page
while showing no useful information. Don't add a third place for the mark without removing one of
the current two.

### Buttons
- **Shape:** near-square.
- **Primary:** Engraving Ink fill, Sheet Cream text, with a monospace key hint inline where a
  shortcut exists.
- **Ghost:** transparent, used as square icon buttons in toolbars at 1.125rem icon size.
- **Hover / Focus:** background darkens slightly to 4% ink on hover; `:focus-visible` is a 2px
  Banknote Green outline at 2px offset, never removed.

### Navigation (rail)
- Labelled text items with a Phosphor icon at 55% opacity, near-square radius, counts
  right-aligned in the label style, grouped under monospace micro-headings.
- **Active:** Engraving Ink fill with Sheet Cream text and a filled-weight icon.
- Icon-only navigation was rejected: it doesn't work well for keyboard and screen-reader users.

### Page Header
The breadcrumb trail *is* the page title: `Case Name / Section`, one line, one Display size.
Ancestors take `--ink-form-label` and regular weight; the current page stays full ink and 600
weight — told apart by weight and ink, not by size. Shares the rail's Sheet Cream ground and heavy
hairline, edge to edge with the rail and the viewport, so the two read as one frame.

### Panels / Containers
Sheet Cream ground, 1px hairline at 12% ink plus the engraved ring, 0.75rem/1rem internal padding.

### Data Grid
The main component of the design. Collapsed borders, Data text style, 40px rows, sticky monospace
column headers with a 22% underline, row dividers at 7% ink, and a 120ms background transition on
hover. The keyboard cursor row looks different from a hover state: an 8% Banknote Green wash plus
a 2px line on the left edge.

### Relationship Sheet (the lateral-movement graph)

The one drawn/visual screen in the product. Think of it as a **diagram on paper**, not a window
into open space: a summary strip on top, the diagram on panel paper, and a legend along the bottom
edge — the way a printed diagram carries its key.

**The Struck Token.** A node is an icon on its own disc of Sheet Cream inside a thin ring — 20px
radius for an asset, 14px for an indicator — so a connecting line stops at the edge of the token
instead of running underneath it. A severe record uses the exact same stamp style as the
confirmation dialog: a 45% ring over a 6% wash, with the icon in the severe color. Labels are
monospace at 10–11px below the token, with a 4px Sheet Cream outline glow instead of a filled box
— a filled box would add a second rectangle to a page that already has ruled lines.

**The Weight-Not-Hue Rule.** An asset is the main subject, and an indicator is evidence attached
to it — so the two are told apart by size and ink strength (100% vs. 45%), never by giving one of
them its own color. This is the Ink States Rule applied to a diagram: **the icon can take the
severe color, but the label never does** — the same way a status dot is colored but its label word
is not. So a case where most hosts are compromised still reads as a diagram, not as an alarm (see
Product Principle 1). Banknote Green only appears on hover, to highlight a node's neighbors — a
color change, not a weight change, since the graph library's default behavior would otherwise make
the highlighted line three times thicker than normal.

Connecting lines have no direction by design: the order an event lists two assets in doesn't mean
anything, so every pair in an event is just linked, with no arrow implying an order that isn't in
the data. A solid thin line means a shared event; a dashed line means an indicator observed
together with it.

**The canvas can't use CSS inheritance.** Canvas colors are plain text values, not something CSS
can cascade into — so a repaint is wired to both ways the theme can change: switching `data-theme`
directly, or the OS-level preference changing. Every color is declared as a `--graph-*` CSS custom
property, read through a hidden probe element. No color is hardcoded in the script, and nothing is
read at build time — the color set is one object that the drawing code reads *while it draws*, so
a theme change just means a redraw, not a full rebuild.

Refreshing this page submits the filter form again, instead of following a plain link. The page
header sits outside the part of the page that gets swapped, so a link's URL would freeze at
whatever state the page first loaded in — a reload would silently drop the filters. Any screen
where the filters define what's shown needs to work the same way.

The layout uses a fixed starting point, and nodes and edges are sorted on the server, so reloading
the page redraws the same picture instead of rearranging everything the analyst was just looking
at. The physics simulation settles before the first paint. The Freeze button stops it, and it
starts already frozen if the user's system has `prefers-reduced-motion` turned on.

### Inputs
Hairline border, Sheet Cream ground, near-square radius, Data type, a leading Phosphor icon at 45%
opacity, and a trailing monospace key hint.

### Form Sheet (drawers)

A drawer holding a form is a **document with a fixed header and footer**, not a page that scrolls
as a whole. The header shows the title and the record's own serial number in signal green; the
footer holds the save button on a recessed background; the fields scroll in between. Otherwise, a
long form pushes the Save button below the visible area, and the title scrolls out of view too —
so the analyst has to scroll to see what they're editing, and scroll again to save it.

These regions use flexbox layout, not `position: sticky` — the overlay box has `overflow-x:
hidden` set, which makes it a scroll container on both axes. A sticky element inside it would
anchor to a box that never actually scrolls, so it would just sit still and not behave like a
sticky element at all.

Fields are grouped under `fieldset` / `legend`, styled like a label, over a 22% dividing line — the
same section marker used in the summary readouts. **A group is only worth it at around five fields
or more**; fewer than that, and the heading has nothing real to do, so short forms stay flat with
no grouping.

**The Subject Field Rule.** One field per form is the record's main subject, not just one of its
attributes — a note *is* its description, a case *is* its summary. That field has a minimum height
of 320px and then grows to fill whatever space the drawer has left, instead of sitting at plain
textarea height on an otherwise near-empty form. It can grow but never shrink, so on a crowded
form it keeps its minimum height and the form scrolls instead of squeezing that field smaller.

### Confirmation (the countersign)

A destructive-action confirmation is the same form-sheet layout with the fields taken out: the
header shows the question and what it will do, the footer shows the two answer buttons, nothing in
between. It reuses the same header and footer CSS classes as the form sheet, instead of redefining
them, so the two styles can't drift apart over time.

**The layer matches the type of interaction.** A drawer is a document you work inside; a
confirmation is a single pause with exactly two possible answers, so it shows as a centered modal
at 28rem width. Two lines of text in a full-height drawer would say "work here" about something
that only takes one click.

The severe color shows up as a small 2.25rem stamp — a thin ring at 45%, a 6% wash, and the icon —
not as a large 4rem filled ring. This rare color marks the action itself; it doesn't decorate the
dialog. The destructive button is the one place where a **filled** severe color is used, and it's
small.

### Status Indicator
A 9px circle plus the status word, always together. Severe is the only coloured variant; all
others are Status Ink. See The Dot-and-Word Rule and The Ink States Rule.

### Feedback (toasts, inline messages)

**The Seal Toast.** Success messages use **Seal Green**, the main structural color — not the
signal color. A finished record gets stamped, not announced loudly: a filled seal icon at 1.5rem,
a monospace `RECORDED · <time>` caption, the message in body text, and the record's serial number
in signal green. It has the raised engraved edge style because it floats above the page.

The reason is how often it happens, not what it means. Success messages appear on every save,
import, and completed job, making them the most common message type in the app. Using the rare
signal color on the most frequent event would be the fastest way to make it stop being a signal.

- **Success:** Seal Green; the `--color-success` token points here.
- **Error:** Severe.
- **Info and warning:** removed, mapped to plain ink — so any component still trying to use them
  falls back to a neutral mark instead of bringing back a color.

### Key Hint
Monospace Label text on an 8% ink background. Falls back to system-ui font per character, because
JetBrains Mono doesn't have ⌘, ⏎, or ⎋ symbols.

**The Legible Glyph Rule.** A key hint is only shown if it's readable at Label size. Letters,
digits, and arrows are readable; ⎋ is not — at that size, the broken circle and arrow blur into a
smudge that looks like a warning sign, so every Escape-key hint was removed. Escape still closes
every overlay — it's a convention people already know, so it doesn't need a label.

## Do's and Don'ts

### Do:
- **Do** render every label, table header, and identifier in monospace uppercase at 0.14em tracking.
- **Do** pair status colour with the status word, always.
- **Do** keep tabular numerals on anything comparable down a column.
- **Do** preserve `:focus-visible` as a 2px Banknote Green outline; it is never removed.
- **Do** respect `prefers-reduced-motion`; all transitions and animations are disabled under it.
- **Do** derive row height from `max(text, control) + 2 x cell-y`, and re-derive when any input moves.
- **Do** place any new colour on the Green Ladder, or explain why it is a second exception alongside red.

### Don't:
- **Don't** use filled status pills or coloured row backgrounds.
- **Don't** add shadows. Depth is a second hairline.
- **Don't** introduce pill radii or soft corners; nothing exceeds 0.25rem except true circles.
- **Don't** ship icon-only navigation without real labels.
- **Don't** let Banknote Green decorate. It means "look here" or nothing.
- **Don't** set body text in Assay Gold; it fails contrast on cream.
- **Don't** reintroduce a hue for a non-severe status.
- **Don't** add ornament that occupies the working surface — no microtext bands, no guilloché strips, no
  decorative full-width rules. Both were built and cut.
- **Don't** place the brand mark anywhere beyond the rail and empty states.
- **Don't** reuse an opacity percentage across themes without checking it again, and don't fill a
  full-width area with the inverted color on the dark theme.
- **Don't** tie a color to `base-content` on anything that can sit inside an inverted fill — use
  `currentColor` instead, or the element disappears on both themes at once.
