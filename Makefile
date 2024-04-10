.PHONY: build build-web build-go docker run

build: build-web build-go

build-web:
	tailwindcss -c configs/tailwind.config.js -i web/_build.css -o web/dagobert.css

build-go:
	templ generate
	go build -o dagobert ./cmd

docker:
	docker build . -f build/Dockerfile -t sprungknoedl/dagobert

run:
	source configs/dagobert.env && air -c configs/air.toml