build:
	time tailwindcss -i css/index.css -o dist/dagobert.css
	time templ generate
	time go build -o dagobert ./cmd