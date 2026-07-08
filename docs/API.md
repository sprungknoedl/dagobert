# JSON API

Dagobert's case-data routes serve JSON as well as HTML: the same handlers, same URLs,
switched by content negotiation. There is no separate API surface or version — any CRUD
operation you can do in the browser under `/cases/...` you can also do with a `curl` request.

The full machine-readable contract is [`openapi.yaml`](../openapi.yaml) at the repo root.
This page is the getting-started guide.

## Scope

Covered: cases, events, assets, malware, indicators, evidences, tasks, notes, and comments
— list, get, save (create/update), delete. Not covered (HTML/file only, regardless of
`Accept`): CSV import/export, archive import/export, file downloads, module list/run, case
fork/ACL/switch/summary, visualizations, reports, settings, auth.

## Authentication

Create a key with the CLI:

```sh
dagobert create-api-key "my integration"
# prints the plaintext key once, e.g. dgb_AbCd...1234 — it is never stored, save it now
```

Send it on every request, either header works:

```
X-API-Key: dgb_AbCd...1234
Authorization: Bearer dgb_AbCd...1234
```

Keys created this way authenticate as the built-in `API` principal, which has full
Administrator access. Two other key types exist for narrower automation (created from
Settings → API keys, not the CLI): `Donald`, which may only create evidence
(`POST /cases/*/evidences/new`), and `MCP`, which is read-only and scoped to the MCP
endpoint. Pick whichever matches how much the integration should be able to do.

An invalid or malformed key is rejected with `401`; a valid key without permission for the
route gets `403`.

## Content negotiation

Send `Accept: application/json` to get JSON responses. If you send no `Accept` header (or
`*/*`) while authenticating with an API key, Dagobert assumes you're a machine client and
defaults to JSON automatically — so in practice you rarely need the header at all when
using a key. An explicit `Accept: text/html` still gets you the normal browser UI.

## Responses

- `GET .../` → `200` + a JSON array of records.
- `GET .../{id}` → `200` + the record.
- `POST .../{id}` (`{id}` = `new` to create) → `201` + the saved record.
- `DELETE .../{id}` → `204`, no body. The confirmation step that browsers see is skipped —
  an explicit `DELETE` from a JSON client is the confirmation. The same applies to closing a
  case (`POST /cases/{id}` with `Closed: true`): the browser's outstanding-items soft-confirm
  (open tasks, un-triaged assets, missing classification/outcome) is skipped for JSON
  clients — an explicit `Closed: true` is the confirmation, same as delete.
- Validation failure → `422` + a map of field name to `{Name, Message, Missing, Invalid}`.
- Any other handled error → `400`/`500` + `{"error": "..."}`.

Field names are PascalCase, matching the Go structs exactly (no `json` tags on models) —
see the component schemas in `openapi.yaml` for the full field list per entity.

## Examples

Create a case:

```sh
curl -s https://dagobert.example.com/cases/new \
  -H "X-API-Key: dgb_AbCd...1234" \
  -H "Content-Type: application/json" \
  -d '{"Name":"ACME ransomware","Severity":"High"}'
# -> 201, body includes the generated 10-character ID and auto-filled OpenedAt
```

Create a case from a template — the template's tasks and notes are copied in:

```sh
curl -s "https://dagobert.example.com/cases/new?Template=<template-id>" \
  -H "X-API-Key: dgb_AbCd...1234" \
  -H "Content-Type: application/json" \
  -d '{"Name":"ACME ransomware"}'
```

Add a task to a case:

```sh
curl -s https://dagobert.example.com/cases/<case-id>/tasks/new \
  -H "X-API-Key: dgb_AbCd...1234" \
  -H "Content-Type: application/json" \
  -d '{"Task":"Isolate host","Type":"Containment"}'
```

List evidences:

```sh
curl -s https://dagobert.example.com/cases/<case-id>/evidences/ \
  -H "X-API-Key: dgb_AbCd...1234"
```

## Uploading evidence/malware files

A JSON body carries metadata only. To upload the file itself, `POST` the same URL with
`multipart/form-data` and a `File` field instead — add `Accept: application/json` to still
get a JSON response:

```sh
curl -s https://dagobert.example.com/cases/<case-id>/evidences/new \
  -H "X-API-Key: dgb_AbCd...1234" \
  -H "Accept: application/json" \
  -F "Name=memory.dmp" -F "Type=Memory" -F "File=@memory.dmp"
```

Replacing the file of an existing entry is not supported — create a new entry instead.