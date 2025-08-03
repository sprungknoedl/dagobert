FROM node:23 AS web
WORKDIR /src
ENV PATH=$PATH:/src/node_modules/.bin

COPY package*.json .
RUN npm install

COPY . .
RUN make build-web

# ---------------------------------

FROM golang:1.24 AS app
WORKDIR /src

COPY go.mod .
RUN go mod download
COPY --from=web /src/public /src/public

COPY . .
RUN make build-go

# ---------------------------------

FROM debian:12-slim
WORKDIR /home/sprungknoedl

RUN apt update && apt install -y docker.io

COPY --from=app /src/dagobert /home/sprungknoedl/dagobert
COPY --from=app /src/app/views /home/sprungknoedl/app/views
COPY --from=app /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN useradd -l -u 1000 -g users sprungknoedl
RUN mkdir /home/sprungknoedl/files && chown -R sprungknoedl:users /home/sprungknoedl
ENV PATH="$PATH:/home/sprungknoedl"

USER sprungknoedl
CMD ["/home/sprungknoedl/dagobert"]