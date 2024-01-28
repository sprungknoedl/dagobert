.PHONY: build run

build:
	tailwindcss -i web/_build.css -o web/dagobert.css
	templ generate
	go build -o dagobert ./cmd

run:
	source configs/dagobert.env && air -c configs/air.toml