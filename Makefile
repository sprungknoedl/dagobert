.PHONY: build run

build:
	tailwindcss -c configs/tailwind.config.js -i web/_build.css -o web/dagobert.css
	templ generate
	go build -o dagobert ./cmd

docker:
	docker build . -f build/Dockerfile -t sprungknoedl/dagobert

run:
	source configs/dagobert.env && air -c configs/air.toml