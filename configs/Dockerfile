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
RUN useradd -l -u 1000 -g users sprungknoedl

COPY --from=web /src/web /home/sprungknoedl/web
COPY --from=app /src/dagobert /home/sprungknoedl/dagobert
COPY --from=app /src/internal/views /home/sprungknoedl/internal/views
COPY --from=app /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN mkdir /home/sprungknoedl/files && chown sprungknoedl:users /home/sprungknoedl/files

USER sprungknoedl
CMD ["/home/sprungknoedl/dagobert"]