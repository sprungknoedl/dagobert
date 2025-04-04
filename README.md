# Dagobert
**A Collaborative Platform for Incident Response**

Dagobert streamlines incident investigations by helping teams share technical details, track progress, and generate comprehensive reports—all in one place. Inspired by tools like [IRIS](https://dfir-iris.org/) and [Aurora Incident Response](https://github.com/cyb3rfox/Aurora-Incident-Response), Dagobert enhances collaboration and documentation for faster, more effective incident response.


## ✨ Key Features

### 📂 Multi-Case Management
Manage multiple investigations at once with dedicated workspaces for each case. Track progress in real time and coordinate with your team without losing context.

### 🔍 Evidence & IOC Tracking
Record technical details, assets, IOCs, and forensic evidence in a structured way. Dagobert automatically links findings to past investigations for deeper insights.

### ⏳ Timeline Reconstruction
Build a clear chronology of attack events and investigative actions. Visualize the sequence of compromise and key decision points in one unified timeline.

### 📝 Collaborative Notes & Comments
Document every step of your investigation in a flexible, wiki-style format. Add comments, observations, and technical notes with team-wide visibility.

### ✅ Task Assignment & Tracking
Delegate tasks to team members directly within the platform. Monitor progress, deadlines, and dependencies to keep the investigation on track.

### 📄 Automated Reporting
Generate polished reports with a single click. Customise templates and export in multiple formats to share findings with stakeholders effortlessly.

### 🔌 Extensible Plugin System

**Automated Evidence Processing:** Integrate tools like [Hayabusa](https://github.com/Yamato-Security/hayabusa) for EVTX parsing or [Plaso](https://github.com/log2timeline/plaso) for timeline generation.

**Timesketch Integration:** Upload timelines to Timesketch with one click or automatically import events for deeper analysis.


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
    cp dagobert.env.example dagobert.env
    nano dagobert.env # update settings (see 📝 Configuration below)
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

All contributions in any form (be it code, documentation, design) are highly welcome!

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-idea`.
3. Submit a PR with a clear description.


## License

Dagobert is released under the MIT License.


## Contact

For issues and inquiries, please create a GitHub Issue.