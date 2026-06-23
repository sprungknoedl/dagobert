Dagobert uses environment variables for all runtime configuration. Below is the complete reference for all available settings.

## 🌐 OpenID Connect (OIDC) Authentication
Configure single sign-on via OpenID Connect:

| Variable              | Required | Description                                                                                    | Example                                                                                                                         |
| --------------------- | -------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `OIDC_ISSUER`         | Yes      | OpenID Connect discovery URL of your identity provider                                         | `https://login.microsoftonline.com/TENANT_ID/v2.0` (Microsoft Entra)  <br>`https://auth.example.com/realms/dagobert` (Keycloak) |
| `OIDC_CLIENT_ID`      | Yes      | Client ID assigned to Dagobert by your IdP                                                     | `ClientIdIssuedById`                                                                                                            |
| `OIDC_CLIENT_SECRET`  | Yes      | Client secret assigned to Dagobert by your IdP                                                 | `SecretIssuedByIdp`                                                                                                             |
| `OIDC_CLIENT_URL`     | Yes      | Dagobert's base URL (must match IdP callback configuration)                                    | `https://dagobert.example.com/`                                                                                                 |
| `OIDC_ID_CLAIM`       | No       | Claim to use as unique user identifier:  <br>- `sub` (standard)  <br>- `oid` (Microsoft Entra) | `sub`                                                                                                                           |
| `OIDC_AUTO_PROVISION` | No       | Automatically create accounts for new users (default: `false`)                                 | `true`                                                                                                                          |

## 🖥 Web Server Settings

| Variable             | Required | Description                                                                      | Example                     |
| -------------------- | -------- | -------------------------------------------------------------------------------- | --------------------------- |
| `WEB_SESSION_SECRET` | Yes      | Secret key for encrypting session cookies (generate with `openssl rand -hex 32`) | `SecretCookieEncryptionKey` |

## 👨‍💻 Administrator Setup
Pre-configure administrative users (using their OIDC identity claim):

| Variable Format    | Required | Description                                  | Example                                                                          |
| ------------------ | -------- | -------------------------------------------- | -------------------------------------------------------------------------------- |
| `DAGOBERT_ADMIN_N` | No       | Where N is a sequential number starting at 0 | `DAGOBERT_ADMIN_0=IdOfAdministrator1`  <br>`DAGOBERT_ADMIN_1=IdOfAdministrator2` |
## 🔑 API Access
Pre-configure administrative API keys:

| Variable           | Required     | Description                                     | Example                                          |
| ------------------ | ------------ | ----------------------------------------------- | ------------------------------------------------ |
| `DAGOBERT_KEY_N`   | No           | Pre-configured API keys (where N is sequential) | `DAGOBERT_KEY_0=key1`  <br>`DAGOBERT_KEY_1=key2` |

## 🕵️ Timesketch Integration

|Variable|Required|Description|Example|
|---|---|---|---|
|`TIMESKETCH_URL`|No|Timesketch server URL|`http://timesketch:8080`|
|`TIMESKETCH_USER`|No|Timesketch username|`dagobert-user`|
|`TIMESKETCH_PASS`|No|Timesketch password|`timesketch-password`|
|`TIMESKETCH_SKIP_VERIFY_TLS`|No|Disable TLS verification (default: `false`)|`true`|

## ⚙️ Worker Configuration
see [[🔍 Evidence Processing in Dagobert]]

| Variable                   | Required     | Description                                             | Example                       |
| -------------------------- | ------------ | ------------------------------------------------------- | ----------------------------- |
| `DAGOBERT_URL`             | Yes (Worker) | Web server URL workers should connect to                | `http://localhost:8080`       |
| `DAGOBERT_API_KEY`         | Yes (Worker) | Shared secret for worker-to-web communication           | `PleaseDoNotUseThisFromAbove` |
| `DAGOBERT_WORKERS`         | No           | Number of worker processes (default: `3`)               | `5`                           |
| `DAGOBERT_SKIP_VERIFY_TLS` | No           | Disable TLS certificate verification (default: `false`) | `true`                        |
