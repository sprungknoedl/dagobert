.PHONY: build build-web build-go check fmt vet test docker run
.EXPORT_ALL_VARIABLES:
-include dagobert.env

build: build-web build-go

build-web:
	npx @tailwindcss/cli -m -i app/assets/dagobert.css -o public/assets/dagobert.css

build-go:
	go tool templ generate
	CGO_ENABLED=0 go build -o dagobert .

# Canonical verification: run this (and CI runs it) before trusting a change.
# build-go runs `templ generate` first, so vet/test see the generated *_templ.go.
check: build-go vet test
	@unformatted=$$(gofmt -l . | grep -v '_templ\.go$$' || true); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
	fi
	@echo "✓ check passed"

fmt:
	gofmt -w $$(find . -name '*.go' -not -name '*_templ.go' -not -path './node_modules/*')

vet:
	go vet ./...

test:
	go test ./...

docker:
	docker build . -f cfg/Dockerfile -t sprungknoedl/dagobert
	docker build . -f cfg/Dockerfile-hayabusa -t sprungknoedl/dagobert-hayabusa
	docker build . -f cfg/Dockerfile-plaso -t sprungknoedl/dagobert-plaso
	docker build . -f cfg/Dockerfile-timesketch -t sprungknoedl/dagobert-timesketch

run:
	air -c cfg/air.toml

ATTCK_RELEASE=18.1
update: update-mitre
update-mitre:
	mkdir -p files/mitre
	wget -O files/mitre/enterprise-attack.json https://github.com/mitre-attack/attack-stix-data/raw/refs/heads/master/enterprise-attack/enterprise-attack-${ATTCK_RELEASE}.json
	wget -O files/mitre/ics-attack.json https://github.com/mitre-attack/attack-stix-data/raw/refs/heads/master/ics-attack/ics-attack-${ATTCK_RELEASE}.json
	wget -O files/mitre/mobile-attack.json https://github.com/mitre-attack/attack-stix-data/raw/refs/heads/master/mobile-attack/mobile-attack-${ATTCK_RELEASE}.json