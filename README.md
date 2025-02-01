# Dagobert

![Dagobert Logo](web/favicon.svg)

**Dagobert** is a collaborative platform designed to assist incident responders in sharing technical details during investigations and creating more effective incident response documentation. Inspired by the "Spreadsheet of Doom" utilized in the SANS FOR508 class and software like [IRIS](https://dfir-iris.org/) and [Aurora Incident Response](https://github.com/cyb3rfox/Aurora-Incident-Response), Dagobert takes incident response collaboration to the next level.

## ✨ Key Features

**🔄 Real-time Collaboration**
* Simultaneous multi-user editing for live teamwork during investigations
* Audit trail for traceable decision-making

**🔌 Evidence ProcessingPlugins**
*  Extensible plugin architecture for automated evidence handling:
    * EVTX log parsers ([Hayabusa](https://github.com/Yamato-Security/hayabusa))
    * Timeline creation ([Plaso](https://github.com/log2timeline/plaso))

**⏱️ Timesketch integration**
* One-click timeline uploads to Timesketch instances
* Automatic event import from Timesketch
* Bidirection synchronization of indicators (cooming soon)

**📊 Office Report Generation**
* Native support for DOCX/ODT report and XLSX/ODS spreadsheet templates
* Dynamic data binding for:
    * Executive summaries
    * Technical IOC tables
    * Investigation timelines
* Style-preserving exports to Microsoft Word and LibreOffice

## 🚀 Getting Started

### Prerequisites

* Docker and Docker Compose (v2+)
* Configurred OpenID Connect provder (e.g. [Keycloack](https://www.keycloak.org/), [Authentik](https://goauthentik.io/), [Microsoft Entra](https://learn.microsoft.com/en-us/entra/identity-platform/v2-protocols-oidc) or [Google Cloud](https://cloud.google.com/identity-platform/docs/web/oidc))

Dagobert ships no built-in user authentication and instead relies on the presence of an OpenID Connect provider. You need to configure your identity platform first for Dagobert to verify the identity of the user.

### Installation

To ease the installation and upgrades, Dagobert is shipped in Docker containers. Thanks to Docker Compose, it can be ready in a few minutes.

1. Clone the repository

    ```sh
    git clone https://github.com/sprungknoedl/dagobert
    cd dagobert
    ```

2. Configure environment

    ```sh
    cp env.model .env
    nano .env # update settings (see 📝 Configuration below)
    ```

3. Start the stack

    ```sh
    docker compose up -d
    ```

    Access the app at [http://localhost:8080].

**Production Note:** Always deploy behind a HTTPS proxy like Apache, nginx or traefik.

## 📝 Configuration
Dagobert uses environment variables for all runtime configuration.

### OpenID Connect (OIDC)

| Variable | Description | Example |
| -------- | ----------- | ------- |
| `OIDC_ISSUER` | OpenID Connect discovery base URL of the identity provider | `https://auth.example.com/realms/dagobert` |
| `OIDC_CLIENT_ID` | Client ID assigned to Dagobert by the identity proivder | `dagobert-client` |
| `OIDC_CLIENT_SECRET` | Client secret assigned to Dagobert by the identity provider | `supersecret123` |
| `OIDC_CLIENT_URL` | Dagobert's base URL (for OIDC callback) | `https://dagobert.example.com/` |
| `OIDC_ID_CLAIM` | Claim to use as the user ID (`sub` or `oid` for [Microsoft Entra](https://learn.microsoft.com/en-us/entra/identity-platform/id-token-claims-reference#use-claims-to-reliably-identify-a-user)) | `sub` |

### Web server

| Variable | Description | Example |
| -------- | ----------- | ------- |
| `WEB_SESSION_SECRET` | Secret key used to encrypt session cookies | Generate e.g. with `openssl rand -hex 32` |

### Admins
Add initial admins using their OIDC identity claim:

| Variable | Description | Example |
| -------- | ----------- | ------- |
| `DAGOBERT_ADMIN_0` | First administrative user | `a5fad3d3-559c-4578-a3f9-ec907ec0ddb9` |
| `DAGOBERT_ADMIN_N` | Any number of administrators can be added; variable must start with `DAGOBERT_ADMIN_` | |

## Contributing

All contributions in any form (be it code, documentation, design) is highly welcome!

1. Fork the repository/
2. Create a feature branch: `git checkout -b feat/your-idea`.
3. Submit a PR with a clear description.

## License

Dagobert is released under the MIT License.

## Contact

For issues and inquiries, please create a GitHub Issue.