# Indicator Enrichment

The Indicators list contains an **Enrichment** column that surfaces threat-intel verdicts stored in each indicator's custom attributes.

## Strict verdict set

Four values are recognised, in descending severity order:

| Value | Severity | Display colour |
|---|---|---|
| `malicious` | 3 | red (`text-error`) |
| `suspicious` | 2 | amber (`text-warning`) |
| `clean` | 1 | green (`text-success`) |
| `unknown` | 0 | muted |

Matching is **case-insensitive**; the value is always rendered in lower-case canonical form. Any custom-attribute value not in this set is silently ignored.

## Value-based detection

A custom attribute contributes to the Enrichment column solely because its **stored value** is in the strict verdict set — not because of how the attribute was labelled, which module (if any) wrote it, or any marker in the database. This means:

- A verdict entered manually on the indicator edit form renders just like one written by a module.
- Verdicts remain visible even if the module that produced them is later unconfigured.
- There is no migration required.

## Source name

The source name displayed in each dot's tooltip is the attribute key with a trailing `" Verdict"` stripped:

- `"VirusTotal Verdict"` → `VirusTotal`
- `"MISP Verdict"` → `MISP`
- `"Analyst call"` (no suffix) → `Analyst call`

## Column layout

- **Worst verdict** is shown as coloured text at the top of the cell.
- **Per-source dots** appear below, sorted alphabetically by source name (case-insensitive), one dot per matching attribute. Filled dots (`bg-error`/`bg-warning`/`bg-success`) for malicious/suspicious/clean; a hollow border ring for unknown.
- **Empty state** (`—`, muted) when no custom-attribute value falls in the strict set — including `TLP:RED` indicators, which show `—` rather than `not sent` because value-based detection cannot distinguish "withheld" from "never run".
- **No links or summaries** appear in the list cell. Sibling `<X> Link` / `<X> Enrichment` attributes are visible on the indicator edit form.

## Sorting

The column is sortable. Descending click sorts worst-first; indicators with no verdict sort last.
