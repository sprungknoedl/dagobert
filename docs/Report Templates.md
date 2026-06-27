# Report Templates

Dagobert generates case reports by filling in a template you write yourself. This
guide is for the people who author those templates.

## What a report template is

A report template is an ordinary Word (`.docx`) or LibreOffice Writer (`.odt`)
document that you write in Word or LibreOffice like any other document. The only
difference is that it contains `{{ ... }}` markers where case data should be
substituted. You upload it through the **Reports** dialog; when a report is
generated, each marker is replaced with a value from the case and you get back a
`.docx`/`.odt` with the same layout and styling you authored.

Write the markers as plain text in the document body, headers, footers, and table
cells. Keep them in a single run where you can — Word sometimes splits a marker
across formatting runs (e.g. if you change a font mid-marker), and the engine
reassembles split markers, but typing a marker in one go avoids surprises.

## The data you can reference

Every template is rendered with two top-level values:

- `.Case` — the case being reported on, with its fields and related records.
- `.Now` — the time the report was generated.

`.Case` has these scalar fields:

| Field | Meaning |
| --- | --- |
| `.Case.Name` | Case name |
| `.Case.Summary` | Free-text summary |
| `.Case.Classification` | Classification label |
| `.Case.Severity` | Severity label |
| `.Case.Outcome` | Outcome label |
| `.Case.Closed` | Whether the case is closed (boolean) |

And these slices of related records:

- `.Case.Assets` — fields: `.Status`, `.Type`, `.Name`, `.Addr`, `.Notes`,
  `.FirstSeen`, `.LastSeen`, `.Events` (count).
- `.Case.Events` — fields: `.Time`, `.Type`, `.Event`, `.Source`, `.Raw`,
  `.Flagged`, `.Techniques` (a list of MITRE technique IDs), `.Assets`,
  `.Indicators`.
- `.Case.Evidences` — fields: `.Type`, `.Name`, `.Hash`, `.Size`, `.Source`,
  `.Notes`, `.StartsAt`, `.EndsAt`.
- `.Case.Indicators` — fields: `.Status`, `.Type`, `.Value`, `.TLP`, `.Source`,
  `.Notes`, `.Flagged`, `.FirstSeen`, `.LastSeen`, `.Events` (count).
- `.Case.Malware` — fields: `.Status`, `.Path`, `.Hash`, `.Notes`, `.Asset`.
- `.Case.Notes` — fields: `.Title`, `.Category`, `.Description`.
- `.Case.Tasks` — fields: `.Type`, `.Task`, `.Done`, `.Owner`, `.DateDue`.

Reference a scalar directly: `{{ .Case.Name | xml }}`.

## Looping and conditionals

Use `range` to repeat content for every record in a slice, and `if` to include
content conditionally:

```
{{ range .Case.Events }}{{ .Time.Format "2006-01-02 15:04" }} — {{ .Event | xml }}
{{ end }}
```

```
{{ if eq .Status "Compromised" }}This asset was compromised.{{ end }}
```

Inside a `range`, the dot (`.`) refers to the current element, so you write
`.Event`, not `.Case.Events.Event`.

### Pivot rules (important)

This is the non-obvious part of the engine. Where you place the opening `{{ range }}`
or `{{ if }}` marker decides *what* gets repeated or shown:

- **Marker inside a table row** → the whole table row repeats once per element.
  Put `{{ range .Case.Assets }}` in the first cell of a row and `{{ end }}` in the
  last cell of that same row; each asset then produces its own row. This is the
  normal way to build a table of records.
- **Marker inside a paragraph** → that paragraph repeats. Put the `{{ range }}` and
  `{{ end }}` in the same paragraph (or surrounding paragraphs) and the paragraph
  block repeats per element.
- **`else` cannot straddle a single table row.** An `{{ if }} … {{ else }} … {{ end }}`
  whose branches need to be different table rows will not work; keep an `if/else`
  within one row, one paragraph, or use two separate `if` blocks instead.

When in doubt, keep the opening and closing markers of a loop or conditional inside
the same structural element (the same row, or the same paragraph).

## Helper reference

Helpers transform values inside a marker. Chain them with the pipe (`|`): the value
on the left becomes the last argument of the helper on the right.

In addition to these, Go's built-in template functions are available, including
`len`, `index`, `eq`/`ne`/`lt`/`gt`/`le`/`ge`, `and`/`or`/`not`, and
`print`/`printf`.

### List helpers

| Helper | Description | Example |
| --- | --- | --- |
| `head N xs` | First `N` elements (`N` clamped to the slice length) | `{{ range head 5 .Case.Events }}…{{ end }}` |
| `tail N xs` | Last `N` elements | `{{ range tail 5 .Case.Events }}…{{ end }}` |
| `first xs` | First element, or nil if empty | `{{ (first .Case.Assets).Name | xml }}` |
| `last xs` | Last element, or nil if empty | `{{ (last .Case.Events).Time }}` |
| `reverse xs` | A reversed copy | `{{ range reverse .Case.Events }}…{{ end }}` |

### String helpers

| Helper | Description | Example |
| --- | --- | --- |
| `upper s` | Upper-case | `{{ .Case.Name | upper | xml }}` |
| `lower s` | Lower-case | `{{ .Type | lower | xml }}` |
| `title s` | Title-case each word | `{{ .Case.Classification | title | xml }}` |
| `trim s` | Strip leading/trailing whitespace | `{{ .Notes | trim | xml }}` |
| `replace old new s` | Replace all occurrences | `{{ .Value | replace "." "[.]" | xml }}` |
| `truncate N s` | Cap to `N` runes, adding `…` if shortened | `{{ .Summary | truncate 200 | xml }}` |
| `join sep xs` | Stringify and join a list | `{{ .Techniques | join ", " | xml }}` |
| `default d v` | Use `d` when `v` is empty/zero | `{{ .Owner | default "unassigned" | xml }}` |

### The `note` helper

`note "Title"` pulls a case note's body into the report by its title and returns
the raw markdown text of that note's description:

```
{{ note "Executive Summary" | xml }}
```

If no note in the case has that exact title, report generation fails with an error
naming the missing title — so a typo surfaces immediately rather than silently
producing a blank section.

## Escaping rule

Always finish a value pipeline with `| xml`:

```
{{ .Case.Name | xml }}
{{ .Case.Summary | upper | xml }}
```

The document is XML under the hood, and the characters `&`, `<`, and `>` have
special meaning there. A value containing any of them (for example a host name
like `<dc01>` or a note mentioning `A & B`) would corrupt the document if inserted
raw. `xml` escapes those characters so the value renders as written. Put `xml`
last, after any other helpers, since it produces the final text that lands in the
document.

## Dates

Time fields (such as `.Time`, `.FirstSeen`, `.DateDue`, and `.Now`) are formatted
with Go's reference-time layout, where `Mon Jan 2 15:04:05 2006` is the canonical
example moment:

```
{{ .Now.Format "2006-01-02" }}
{{ .Time.Format "2006-01-02 15:04" }}
```

Rearrange those reference numbers to get the format you want — `02.01.2006` for
day-first dates, `15:04` for time only, and so on.

## A working starting point

Rather than starting from a blank document, copy one of the demo templates shipped
with Dagobert — for example `templates/Demo Writer Report.odt` — open it in
LibreOffice or Word, and adapt it. It already wires up the data references,
loops, and styling described above.
