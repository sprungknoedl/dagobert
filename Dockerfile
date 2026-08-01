FROM --platform=$BUILDPLATFORM golang:1.26 AS build
WORKDIR /src
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build-web
RUN GOARCH=$TARGETARCH make build-go

# ---------------------------------

FROM ubuntu:24.04
ARG TARGETARCH

# The GIFT PPA's noble packages get superseded (removed, not just replaced) on new
# releases, so this pin behaves less like a lock and more like a tripwire: when GIFT
# publishes a new build, this version disappears from the PPA and the apt install below
# fails the build until the ARG is bumped. Find the current version at
# https://launchpad.net/~gift/+archive/ubuntu/stable/+packages?field.name_filter=plaso-tools
ARG PLASO_VERSION=20260119-4ppa1~noble
ARG HAYABUSA_VERSION=3.10.0
ARG CHAINSAW_VERSION=2.16.2
ARG ZIRCOLITE_VERSION=3.7.6
ARG DISSECT_VERSION=3.25.1

# ubuntu:24.04 ships an `ubuntu` user at uid 1000; replace it with a `dagobert` user at
# the same uid, since that's the uid a bind-mounted host directory is owned by on a
# typical single-user Linux host (otherwise -v ./data:/home/dagobert/data is silently
# unwritable).
RUN userdel -r ubuntu && useradd -u 1000 -m -d /home/dagobert dagobert

RUN apt-get update && apt-get install -y software-properties-common && \
    add-apt-repository -y ppa:gift/stable && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        plaso-tools=${PLASO_VERSION} \
        python3-venv \
        curl \
        unzip \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Hayabusa: musl builds are published for x64 only, so use the gnu build on both arches.
# The release zip also bundles config/ and a full rules/ checkout (with its own .git);
# Dagobert only needs the binary — its Sigma rules come from `hayabusa update-rules` via
# UpdateAssets in internal/modules/hayabusa/hayabusa.go, not from this archive.
RUN case "$TARGETARCH" in \
        amd64) HAYABUSA_ARCH=x64 ;; \
        arm64) HAYABUSA_ARCH=aarch64 ;; \
        *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
    esac && \
    curl -fsSL -o /tmp/hayabusa.zip \
        "https://github.com/Yamato-Security/hayabusa/releases/download/v${HAYABUSA_VERSION}/hayabusa-${HAYABUSA_VERSION}-lin-${HAYABUSA_ARCH}-gnu.zip" && \
    unzip -j /tmp/hayabusa.zip "hayabusa-${HAYABUSA_VERSION}-lin-${HAYABUSA_ARCH}-gnu" -d /opt/hayabusa && \
    mv "/opt/hayabusa/hayabusa-${HAYABUSA_VERSION}-lin-${HAYABUSA_ARCH}-gnu" /opt/hayabusa/hayabusa && \
    chmod +x /opt/hayabusa/hayabusa && \
    rm /tmp/hayabusa.zip

RUN case "$TARGETARCH" in \
        amd64) CHAINSAW_ARCH=x86_64 ;; \
        arm64) CHAINSAW_ARCH=aarch64 ;; \
        *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
    esac && \
    curl -fsSL -o /tmp/chainsaw.tar.gz \
        "https://github.com/WithSecureLabs/chainsaw/releases/download/v${CHAINSAW_VERSION}/chainsaw_${CHAINSAW_ARCH}-unknown-linux-gnu.tar.gz" && \
    mkdir -p /opt/chainsaw && \
    tar -xzf /tmp/chainsaw.tar.gz -C /opt/chainsaw --strip-components=1 && \
    chmod +x /opt/chainsaw/chainsaw && \
    rm /tmp/chainsaw.tar.gz

# Dissect gets its own venv: it shares much of its dependency tree with Zircolite, and a
# shared venv would let a Zircolite upgrade move a package out from under Dissect.
RUN python3 -m venv /opt/dissect && \
    /opt/dissect/bin/pip install --no-cache-dir "dissect.target==${DISSECT_VERSION}"

# Zircolite has no pyproject.toml/setup.py/setup.cfg (not pip-installable) and its
# shebang is a non-functional #!python3, so it needs its own venv plus a wrapper script
# (added below) rather than a console-script entry point.
RUN curl -fsSL -o /tmp/zircolite.tar.gz \
        "https://github.com/wagga40/Zircolite/archive/refs/tags/v${ZIRCOLITE_VERSION}.tar.gz" && \
    mkdir -p /opt/zircolite && \
    tar -xzf /tmp/zircolite.tar.gz -C /opt/zircolite --strip-components=1 && \
    rm /tmp/zircolite.tar.gz && \
    python3 -m venv /opt/zircolite/.venv && \
    /opt/zircolite/.venv/bin/pip install --no-cache-dir -r /opt/zircolite/requirements.txt

# Uniform PATH entries: a symlink per binary/venv script, and a wrapper for Zircolite, so
# every MODULE_* preset below is just the tool's bare name, character-for-character the
# documented local-binary examples.
RUN ln -s /opt/hayabusa/hayabusa /usr/local/bin/hayabusa && \
    ln -s /opt/chainsaw/chainsaw /usr/local/bin/chainsaw && \
    ln -s /opt/dissect/bin/target-query /usr/local/bin/target-query && \
    ln -s /opt/dissect/bin/rdump /usr/local/bin/rdump && \
    printf '#!/bin/sh\nexec /opt/zircolite/.venv/bin/python /opt/zircolite/zircolite.py "$@"\n' > /usr/local/bin/zircolite && \
    chmod +x /usr/local/bin/zircolite

COPY --from=build /src/dagobert /usr/local/bin/dagobert
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENV MODULE_PLASO="psteal.py --unattended"
ENV MODULE_HAYABUSA="hayabusa"
ENV MODULE_CHAINSAW="chainsaw"
ENV MODULE_DISSECT="target-query"
ENV MODULE_DISSECT_RDUMP="rdump"
ENV MODULE_ZIRCOLITE="zircolite"

WORKDIR /home/dagobert
USER dagobert

# Bake the MITRE ATT&CK data plus every module's vendor assets (Hayabusa/Chainsaw/
# Zircolite Sigma rules, mappings, templates - fetched now that the MODULE_* presets
# above are set) into the image so the runtime needs no network. This must run after the
# USER switch so external/ is owned by dagobert and refreshable later. `dagobert update`
# also creates a throwaway data/dagobert.db here; Docker copies image contents into a
# fresh named volume on first run, so a baked database would only ever be a decoy -
# delete it so the image ships vendor assets only and docker-entrypoint.sh's bootstrap
# branch is the one exercised path.
RUN dagobert update && rm -f data/dagobert.db

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["server"]
