.PHONY: build build-web build-go docker run
.EXPORT_ALL_VARIABLES:
-include dagobert.env

build: build-web build-go

build-web:
	npx @tailwindcss/cli -i configs/dagobert.css -o web/dagobert.css

build-go:
	CGO_ENABLED=0 go build -o dagobert .

docker:
	docker build . -f configs/Dockerfile -t sprungknoedl/dagobert
	docker build . -f configs/Dockerfile-hayabusa -t sprungknoedl/dagobert-hayabusa
	docker build . -f configs/Dockerfile-plaso -t sprungknoedl/dagobert-plaso
	docker build . -f configs/Dockerfile-timesketch -t sprungknoedl/dagobert-timesketch

run:
	air -c configs/air.toml