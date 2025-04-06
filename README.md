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
    nano dagobert.env # update settings
    ```

3. Start the stack

    ```sh
    docker compose up -d
    ```

    Access the app at [http://localhost:8080].

**Production Note:** Always deploy behind a HTTPS proxy like Apache, nginx or traefik.


## 📝 Configuration

Dagobert uses environment variables for all runtime configuration. See [📝 Wiki:Configuration](https://github.com/sprungknoedl/dagobert/wiki/📝-Configuration) for a complete reference for all available settings.


## Contributing

All contributions in any form (be it code, documentation, design) are highly welcome!

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-idea`.
3. Submit a PR with a clear description.


## License

Dagobert is released under the MIT License.


## Contact

For issues and inquiries, please create a GitHub Issue.
