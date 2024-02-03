.PHONY: build run

build:
	tailwindcss -c configs/tailwind.config.js -i web/_build.css -o web/dagobert.css
	templ generate
	go build -o dagobert ./cmd

run:
	source configs/dagobert.env && air -c configs/air.toml