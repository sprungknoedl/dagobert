FROM golang:1.26 AS app
WORKDIR /src

COPY go.mod .
RUN go mod download

COPY . .
RUN make build-web
RUN make build-go
# Bake the MITRE ATT&CK data (and its .version sentinel) into the image so the
# runtime needs no network. `update` also creates a throwaway files/dagobert.db
# here — it is not copied into the final image (only /src/mitre is).
RUN ./dagobert update

# ---------------------------------

FROM debian:12-slim
WORKDIR /home/sprungknoedl

RUN apt update && apt install -y docker.io

COPY --from=app /src/dagobert /home/sprungknoedl/dagobert
COPY --from=app /src/mitre /home/sprungknoedl/mitre
COPY --from=app /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY docker-entrypoint.sh /home/sprungknoedl/docker-entrypoint.sh

RUN useradd -l -u 1000 -g users sprungknoedl
RUN chmod +x /home/sprungknoedl/docker-entrypoint.sh
RUN chown -R sprungknoedl:users /home/sprungknoedl
ENV PATH="$PATH:/home/sprungknoedl"

USER sprungknoedl
# entrypoint bootstraps a fresh data volume (migrate db) before exec-ing dagobert
ENTRYPOINT ["/home/sprungknoedl/docker-entrypoint.sh"]
CMD ["server"]