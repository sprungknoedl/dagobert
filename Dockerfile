FROM golang:1.21 AS build
WORKDIR /src

RUN curl -sL -o /usr/bin/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 && chmod +x /usr/bin/tailwindcss
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod .
RUN go mod download

COPY . .
RUN make

# ---------------------------------

FROM golang:1.21
WORKDIR /app

COPY --from=build /src/dist /app/dist
COPY --from=build /src/templates /app/templates
COPY --from=build /src/dagobert /app/dagobert

CMD ["/app/dagobert"]