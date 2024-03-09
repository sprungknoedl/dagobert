FROM golang:1.22 AS build
WORKDIR /src

RUN curl -sL -o /usr/bin/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 && chmod +x /usr/bin/tailwindcss
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod .
RUN go mod download

COPY . .
RUN make build

# ---------------------------------

FROM golang:1.22
WORKDIR /app

RUN mkdir -p /app/files

COPY --from=build /src/web /app/web
COPY --from=build /src/templates /app/templates
COPY --from=build /src/dagobert /app/dagobert

CMD ["/app/dagobert"]