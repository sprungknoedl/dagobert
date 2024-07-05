FROM node:21 AS web
WORKDIR /src
ENV PATH=$PATH:/src/node_modules/.bin

COPY package*.json .
RUN npm install

COPY . .
RUN make build-web

# ---------------------------------

FROM golang:1.22 AS app
WORKDIR /src

COPY go.mod .
RUN go mod download

COPY . .
RUN make build-go

# ---------------------------------

FROM debian:12-slim
WORKDIR /app

RUN mkdir -p /app/files
RUN apt update && apt install -y docker.io

COPY --from=web /src/web /app/web
COPY --from=app /src/dagobert /app/dagobert
COPY --from=app /src/internal/views /app/internal/views
COPY --from=app /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/app/dagobert"]