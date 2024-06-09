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

RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod .
RUN go mod download

COPY . .
RUN make build-go

# ---------------------------------

FROM golang:1.22
WORKDIR /app

RUN mkdir -p /app/files

COPY --from=web /src/web /app/web
COPY --from=app /src/templates /app/templates
COPY --from=app /src/dagobert /app/dagobert

CMD ["/app/dagobert"]