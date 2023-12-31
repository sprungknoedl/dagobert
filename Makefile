build:
	tailwindcss -i css/index.css -o dist/dagobert.css
	templ generate
	go build -o dagobert ./cmd