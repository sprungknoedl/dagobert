FROM golang:1.26 AS app
WORKDIR /src

COPY go.mod .
RUN go mod download

COPY . .
RUN make update
RUN make build-web
RUN make build-go

# ---------------------------------

FROM debian:12-slim
WORKDIR /home/sprungknoedl

RUN apt update && apt install -y docker.io

COPY --from=app /src/dagobert /home/sprungknoedl/dagobert
COPY --from=app /src/mitre /home/sprungknoedl/mitre
COPY --from=app /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN useradd -l -u 1000 -g users sprungknoedl
RUN chown -R sprungknoedl:users /home/sprungknoedl
ENV PATH="$PATH:/home/sprungknoedl"

USER sprungknoedl
ENTRYPOINT ["/home/sprungknoedl/dagobert"]
CMD ["server"]