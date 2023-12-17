FROM node:21.2 AS build-js
WORKDIR /src

COPY . .
RUN npm install
RUN npm run bundle

# ---------------------------------

FROM golang:1.21 AS build-go
WORKDIR /src

COPY . .
RUN go build -o dagobert .

# ---------------------------------

FROM golang:1.21
WORKDIR /app

COPY --from=build-js /src/dist /app/dist
COPY --from=build-go /src/dagobert /app/dagobert

CMD ["/app/dagobert"]