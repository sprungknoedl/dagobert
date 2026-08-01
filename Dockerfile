FROM golang:1.26 AS build
WORKDIR /src

COPY go.mod .
RUN go mod download

COPY . .
RUN make build-web
RUN make build-go

# ---------------------------------

FROM log2timeline/plaso
USER root
ADD https://github.com/Yamato-Security/hayabusa/releases/download/v3.2.0/hayabusa-3.2.0-lin-x64-musl.zip /usr/src/hayabusa.zip

# docker.io lets MODULE_* commands be `docker run ...` themselves (see
# "Docker-wrapped tools" in docs/Evidence Processing.md) for tools that aren't
# baked into this image, e.g. Zircolite, Chainsaw, Dissect.
RUN apt update && apt install -y ca-certificates unzip docker.io
RUN unzip /usr/src/hayabusa.zip -d /opt/hayabusa && \
    mv /opt/hayabusa/hayabusa-3.2.0-lin-x64-musl /opt/hayabusa/hayabusa && \
    chmod +x /opt/hayabusa/hayabusa

WORKDIR /home/plaso
COPY --from=build /src/dagobert /home/plaso/dagobert
COPY docker-entrypoint.sh /home/plaso/docker-entrypoint.sh
RUN chmod +x /home/plaso/docker-entrypoint.sh

ENV MODULE_PLASO="/usr/bin/psteal.py --unattended"
ENV MODULE_HAYABUSA="/opt/hayabusa/hayabusa"
ENV PATH="$PATH:/home/plaso"

# Bake the MITRE ATT&CK data and Hayabusa's Sigma rule set (via its
# UpdateAssets hook, now that MODULE_HAYABUSA is set) into the image so the
# runtime needs no network. Also creates a throwaway data/dagobert.db here,
# not copied into the final image.
RUN dagobert update

# entrypoint bootstraps a fresh data volume (migrate db) before exec-ing dagobert
ENTRYPOINT ["/home/plaso/docker-entrypoint.sh"]
CMD ["server"]
