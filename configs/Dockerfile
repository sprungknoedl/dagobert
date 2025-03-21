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
WORKDIR /home/sprungknoedl

RUN apt update && apt install -y docker.io

COPY --from=web /src/web /home/sprungknoedl/web
COPY --from=app /src/dagobert /home/sprungknoedl/dagobert
COPY --from=app /src/internal/views /home/sprungknoedl/internal/views
COPY --from=app /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN useradd -l -u 1000 -g users sprungknoedl
RUN mkdir /home/sprungknoedl/files && chown -R sprungknoedl:users /home/sprungknoedl

USER sprungknoedl
CMD ["/home/sprungknoedl/dagobert"]