.PHONY: build build-web build-go docker run
.EXPORT_ALL_VARIABLES:
-include dagobert.env

build: build-web build-go

build-web:
	npx @tailwindcss/cli -i app/assets/dagobert.css -o public/assets/dagobert.css

build-go:
	CGO_ENABLED=0 go build -o dagobert .

docker:
	docker build . -f cfg/Dockerfile -t sprungknoedl/dagobert
	docker build . -f cfg/Dockerfile-hayabusa -t sprungknoedl/dagobert-hayabusa
	docker build . -f cfg/Dockerfile-plaso -t sprungknoedl/dagobert-plaso
	docker build . -f cfg/Dockerfile-timesketch -t sprungknoedl/dagobert-timesketch

run:
	air -c cfg/air.toml