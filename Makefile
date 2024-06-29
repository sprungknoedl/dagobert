.PHONY: build build-web build-go docker run

build: build-web build-go

build-web:
	tailwindcss -c configs/tailwind.config.js -i configs/dagobert.css -o web/dagobert.css

build-go:
	CGO_ENABLED=0 go build -o dagobert .

docker:
	docker build . -f configs/Dockerfile -t sprungknoedl/dagobert

run:
	source configs/dagobert.env && air -c configs/air.toml