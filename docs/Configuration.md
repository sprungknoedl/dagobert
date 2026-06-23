# Configuration

Dagobert is configured entirely through environment variables. `dagobert.env.example`
ships with an annotated copy of every setting — copy it to `dagobert.env` and edit from
there. This page is the complete reference.

## Database

| Variable | Required | Description | Default |
| --- | --- | --- | --- |
| `DB_URL` | No | SQLite connection string. User data lives under `files/`. | `file:files/dagobert.db?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)` |

## OpenID Connect (OIDC)

Single sign-on is optional. Leave `OIDC_ENABLED` unset (or `false`) to use only the
built-in local login form.

| Variable | Required | Description | Example |
| --- | --- | --- | --- |
| `OIDC_ENABLED` | No | Set to `true` to show the "Sign in with SSO" button alongside local login. | `true` |
| `OIDC_ISSUER` | If enabled | Discovery URL of your identity provider. | `https://login.microsoftonline.com/TENANT_ID/v2.0` (Entra), `https://auth.example.com/realms/dagobert` (Keycloak) |
| `OIDC_CLIENT_ID` | If enabled | Client ID issued to Dagobert by your IdP. | `ClientIdIssuedById` |
| `OIDC_CLIENT_SECRET` | If enabled | Client secret issued to Dagobert by your IdP. | `SecretIssuedByIdp` |
| `OIDC_CLIENT_URL` | If enabled | Dagobert's base URL. The IdP redirect URI is derived as `<OIDC_CLIENT_URL>/auth/callback` — register exactly that value in your IdP. | `https://dagobert.example.com` |
| `OIDC_ID_CLAIM` | No | Token claim used as the stable user identity. Users and per-case ACLs are keyed by this value, so do not change it after first use. | `oid` (default; Entra uses `oid`, most others `sub`) |
| `OIDC_AUTO_PROVISION` | No | When `true`, unknown OIDC users are created automatically with no role (an administrator must assign one before they can do anything). When `false`, they are rejected with 403. | `false` (default) |

## Web server

| Variable | Required | Description | Default |
| --- | --- | --- | --- |
| `WEB_SECURE_COOKIE` | No | Send session cookies only over HTTPS. Set to `false` for local development over plain `http://`. | `true` |

Sessions are stored server-side in the database, so there is no cookie-encryption
secret to configure.

## Users and API keys

There are no environment variables for seeding administrators or API keys. Create them
with the CLI instead:

```sh
dagobert create-user <USERNAME>   # creates an Administrator (prompts for a password)
dagobert create-key  <NAME>       # creates an API key and prints it
```

Run these against a configured database — in a Docker deployment, prefix with
`docker compose exec app`.

## Timesketch

Configures both the Timesketch links shown in the UI and the built-in Timesketch
importer job module. Leave `TIMESKETCH_URL` unset to disable the integration.

| Variable | Required | Description | Example |
| --- | --- | --- | --- |
| `TIMESKETCH_URL` | No | Timesketch server URL. | `http://timesketch:8080` |
| `TIMESKETCH_USER` | No | Timesketch username. | `dagobert-user` |
| `TIMESKETCH_PASS` | No | Timesketch password. | `timesketch-password` |
| `TIMESKETCH_SKIP_VERIFY_TLS` | No | Skip TLS verification (default `false`). | `true` |

## Evidence processing

Evidence is processed by an in-process worker pool — there is no separate worker service
to deploy. The external tools are configured through `MODULE_*` variables, and the number
of concurrent jobs through `DAGOBERT_WORKERS`.

| Variable | Required | Description | Example |
| --- | --- | --- | --- |
| `MODULE_HAYABUSA` | No | Command that runs Hayabusa (EVTX triage). Unset disables the module. | `hayabusa` |
| `MODULE_PLASO` | No | Command that runs Plaso's `psteal`. Unset disables the module. | `psteal.py` |
| `DAGOBERT_WORKERS` | No | Number of concurrent job runners. | `3` (default) |

The `sprungknoedl/dagobert-full` image presets the `MODULE_*` variables to its bundled
tools, so leave them unset when using that image. See
[Evidence Processing](Evidence%20Processing.md) for the full details, including how to run
the tools from local binaries or wrapped in Docker.
