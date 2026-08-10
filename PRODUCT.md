# Product

<!-- impeccable:product-schema 1 -->

## Platform

web (desktop is the design target)

## Stack

One self-contained Go binary. SQLite for the database (`data/dagobert.db`). Pages are rendered on
the server with templ and Unpoly — no single-page app, no JS build step. The compiled CSS is
committed to git and embedded in the binary. It's self-hosted, usually run from the published
container image.

## Users

**Primary: DFIR consultants working cases for outside clients** (consultancy / MSSP). Cases
belong to different clients, reports go out under the consultancy's own letterhead, and the same
analyst moves between unrelated investigations. Internal CIRT/SOC teams handling incidents for
their own single organization are a confirmed secondary audience — they're supported, but when the
two groups' needs conflict, the consultancy analyst's needs come first.

The job is figuring out what an attacker did, on which hosts, and in what order — then writing
that up in a way that holds up to scrutiny. Multiple team members work on the same case at the
same time. **This tool is for the calm work after the fact, not the live incident** — reviewing,
organizing, connecting evidence, and writing the report, usually done days after the alert (see
Product Principle 1).

**Project status:** this is a hobby project, still being explored. The author is the only
contributor and the only user. There's no outside user base to design for and no user research to
point to — design decisions serve the job described above, not an imagined group of installed
users.

## Product Purpose

A shared workspace where all the real content of an investigation — findings, timeline, assets,
indicators, malware, evidence, tasks, notes — lives in one place. The final report is generated
from the team's own document templates, instead of being written by hand at the end.

Success means a case's records are complete and organized enough that the report can be generated
from them directly, instead of being rebuilt from scratch under a deadline.

## Positioning

Deliberately simpler than the established alternatives — and that simplicity is the whole point:

- **Compared to IRIS** (a multi-container Python stack on PostgreSQL): Dagobert is one Go binary
  on SQLite — one container, no external database, and backing it up is just copying a file.
- **Compared to Aurora Incident Response** (a desktop app with file locking): Dagobert is
  web-based, so a whole team can work on the same case at the same time.

No similar product can honestly match this combination: no external services needed, one process,
one file to back up — while still supporting several people working on it at once.

## Operating Context

- Each case is its own workspace, with access control managed per case in the UI.
- Analysts move between unrelated client cases — switching between cases is a normal, everyday
  action.
- Evidence comes in as uploaded files (often EVTX logs and archives), processed by background
  jobs that run external tools.
- Reports are built from the customer's own corporate templates (`.docx`, `.ods`, `.odt`), so the
  finished document looks like it came from the consultancy, not from Dagobert.
- Case data can also be read by other programs: a read-only MCP endpoint at `/mcp`, using an API
  key created in the UI.

## Capabilities and Constraints

**Shipped**

- Multi-case management with per-case RBAC (Administrator, User, Read-Only; Casbin).
- Cross-case dashboard with a MITRE ATT&CK heatmap of observed techniques.
- Timeline: unified chronology of attacker activity and investigative actions, with ATT&CK mapping.
- Structured records for assets, indicators, evidence and malware, with CSV import/export.
- Lateral-movement graph over hosts, accounts and indicators.
- Tasks with owners and due dates; notes and comments visible to the team.
- Report generation from user-supplied `.docx` / `.ods` / `.odt` templates.
- Evidence processing as background jobs: Hayabusa, Zircolite, Chainsaw, Plaso.
- Timesketch integration (push timelines out, import events back).
- Automation hooks on record create/update, conditions written as expr expressions.
- Read-only MCP server exposing cases, timeline, assets, indicators, malware, notes, tasks.
- Auth: local users or any OIDC provider (tested against Azure AD), optional auto-provisioning.
- Customisable value lists and custom attributes — enum vocabularies are user-editable data.

**Load-bearing constraints**

The commitment is **easy deployment**: few moving parts, no web of services depending on each
other. The single binary is how that's achieved right now, and it's part of what the product
promises. SQLite is a practical default, kept **until it actually becomes a measured bottleneck**
— which doesn't seem likely soon. Only a real measurement, not a personal preference, would be a
reason to change it. The templ + Unpoly stack is a deliberate **developer-experience choice**
(easier to develop, no heavy JS frameworks): respect it in design work, but treat it as the
author's own preference, not a promise made to users.

**Not a priority** (confirmed by the author, so these aren't requirements): fully offline /
air-gapped use; tablet and phone support.

**Not built yet:** `docs/specs/` has around 20 written specs for features that haven't been built
(multi-tenancy, LLM case chat and summaries, MISP/OpenCTI/Yeti threat intel, alert triage, and
others). These are proposals — describe them as proposals, not as things that exist.

## Brand Commitments

About 75% settled — it would take a very strong reason to change any of these:

- **The name Dagobert**, after Dagobert Duck (the German name for Scrooge McDuck). The duck
  reference is intentional and central to the brand.
- **The engraved duck artwork**, shipped in `internal/assets/`.
- **The fonts used**: Fraunces, Hanken Grotesk, JetBrains Mono — self-hosted `.woff2` files in
  `internal/assets/`.

The full visual design built on top of these is described in `DESIGN.md`.

## Evidence on Hand

- Real screenshots of the running app: `docs/screenshot-timeline.png`, `-graph.png`,
  `-assets.png`, `-indicators.png`.
- Working documentation: `docs/API.md`, `docs/Configuration.md`, `docs/Evidence Processing.md`,
  `docs/Report Templates.md`, `docs/examples/`, plus `openapi.yaml`.
- MIT licensed, public on GitHub (`sprungknoedl/dagobert`).

The only user right now is the author. There are no customers, named users, testimonials, case
studies, press coverage, adoption numbers, benchmarks, uptime or scale claims — and no commercial
offering, pricing, or support tier. Write copy that doesn't depend on any of these.

## Product Principles

1. **Design for a calm mind.** Forensics rewards careful thinking, not adrenaline. The product
   should feel like a study room, not an operations center. Anything that borrows the visual
   language of urgency — alarm colors, countdown pressure, crisis-style drama — works against the
   actual job.
2. **Easy deployment is the product's promise.** Every feature has to work within "one process, no
   external services, backup is just copying a file." A convenience feature that needs an extra
   service running alongside it isn't actually convenient here.
3. **Design first for the consultancy analyst moving between client cases.** When a decision could
   go either way for a consultancy vs. an internal CIRT team, the consultancy's needs win.
4. **The record exists to become the final report.** Structure that carries through into a
   generated report is worth more than structure that only looks tidy on screen.
5. **The words belong to the user.** Statuses, types, and other enums are editable lists — nothing
   hard-codes a category that the team should be able to change.

Alongside these, held as the author's own preference rather than a promise to users: the plain,
boring tech stack. Server-rendered fragments and no JS build step are a deliberate choice for
easier development, not technical debt that needs to be "modernized" away.

## Accessibility & Inclusion

There's no official accessibility standard set for this product. The accessibility already in the
code (4.5:1 contrast, status never shown by color alone, `:focus-visible` kept,
`prefers-reduced-motion` respected) happened as good practice along the way. It's a quality
baseline worth keeping, not a formal requirement to point to.
