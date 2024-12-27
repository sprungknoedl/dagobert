# Dagobert

![Dagobert Logo](web/favicon.svg)

Dagobert - Collaborative Incident Response Platform

## Overview

Dagobert is a collaborative platform designed to assist incident responders in sharing technical details during investigations and creating more effective incident response documentation. Inspired by the "Spreadsheet of Doom" utilized in the SANS FOR508 class and software like [IRIS](https://dfir-iris.org/) and [Aurora Incident Response](https://github.com/cyb3rfox/Aurora-Incident-Response), Dagobert takes incident response collaboration to the next level.

## Getting Started

Dagobert ships no built-in user authentication and instead relies on the presence of an OpenID Connect provider like [Keycloack](https://www.keycloak.org/), [authentik](https://goauthentik.io/), [Microsoft Entra](https://learn.microsoft.com/en-us/entra/identity-platform/v2-protocols-oidc), [Google Cloud](https://cloud.google.com/identity-platform/docs/web/oidc). You need to configure your identity platform first for Dagobert to verify the identity of the user.

To ease the installation and upgrades, Dagobert is shipped in Docker containers. Thanks to Docker Compose, it can be ready in a few minutes.

```sh
#  Clone repository
git clone https://github.com/sprungknoedl/dagobert
cd dagobert

# Copy & adapt environment file 
cp env.model .env
nano env.model

# Run dagobert 
docker compose up
```

Dagobert and the Docker Compose file listens by default on port 8080/tcp. For production setups, a HTTPS proxy like apache, nginx or traefik should be configured.

## Configuration
Dagobert uses environment variables for all runtime configuration.

The OpenID section has the following variables:

* `OIDC_ISSUER`: OpenID Connect discovery base URL of the idendity provider
* `OIDC_CLIENT_ID`: Client ID assigned to Dagobert by the idendity proivder
* `OIDC_CLIENT_SECRET`: Client secret assigned to Dagobert by the idendity provider
* `OIDC_CLIENT_URL`: URL of Dagobert, used to tell the idendity provider where to send users after authentication
* `OIDC_ID_CLAIM`; Which claim to use as the user ID, normally this should be `sub`. For Microsoft Entra I recomment to use `oid`, as it matches the Object ID found in Entra UI (see also [Use claims to reliably identify a user](https://learn.microsoft.com/en-us/entra/identity-platform/id-token-claims-reference#use-claims-to-reliably-identify-a-user))

The web section has the following variables:

* `WEB_SESSION_SECRET`: Secret key used to encrypt session cookies

To configure default administrators that should be populated on first startup, the following variables can be used:

* `DAGOBERT_ADMIN_0`: OpenID idendity claim (see above) of first administrative user
* `DAGOBERT_ADMIN_N`: Any number of administrators can be populated, as long as the env variable starts with `DAGOBERT_ADMIN_`

## Contributing

We welcome contributions! Please follow our Contribution Guidelines (TBD) to get started.

## License

Dagobert is released under the MIT License.

## Contact

For issues and inquiries, please create a GitHub Issue.