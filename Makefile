.PHONY: build build-web build-go check fmt vet test docker run clean
.EXPORT_ALL_VARIABLES:
-include dagobert.env

TAILWIND_VERSION = 4.1.18
DAISYUI_VERSION  = 5.5.14

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),Darwin)
  TW_OS = macos
else
  TW_OS = linux
endif
ifneq ($(filter $(UNAME_M),arm64 aarch64),)
  TW_ARCH = arm64
else
  TW_ARCH = x64
endif

TAILWIND_BIN = bin/tailwindcss-$(TAILWIND_VERSION)

build: build-web build-go

$(TAILWIND_BIN):
	mkdir -p bin
	wget -O $@ https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-$(TW_OS)-$(TW_ARCH)
	chmod +x $@

bin/daisyui-$(DAISYUI_VERSION).js:
	mkdir -p bin
	wget -O $@ https://github.com/saadeghi/daisyui/releases/download/v$(DAISYUI_VERSION)/daisyui.js

bin/daisyui-theme-$(DAISYUI_VERSION).js:
	mkdir -p bin
	wget -O $@ https://github.com/saadeghi/daisyui/releases/download/v$(DAISYUI_VERSION)/daisyui-theme.js

# Unversioned copies so the @plugin paths in dagobert.css stay version-free.
app/assets/daisyui.js: bin/daisyui-$(DAISYUI_VERSION).js
	cp $< $@

app/assets/daisyui-theme.js: bin/daisyui-theme-$(DAISYUI_VERSION).js
	cp $< $@

build-web: $(TAILWIND_BIN) app/assets/daisyui.js app/assets/daisyui-theme.js
	$(TAILWIND_BIN) -m -i app/assets/dagobert.css -o public/assets/dagobert.css

clean:
	rm -f bin/tailwindcss-* bin/daisyui-*.js
	rm -f app/assets/daisyui.js app/assets/daisyui-theme.js

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
	gofmt -w $$(find . -name '*.go' -not -name '*_templ.go')

vet:
	go vet ./...

test:
	go test ./...

docker:
	docker build . -f cfg/Dockerfile -t sprungknoedl/dagobert
	docker build . -f cfg/Dockerfile-full -t sprungknoedl/dagobert-full

run:
	air -c cfg/air.toml

ATTCK_RELEASE=18.1
update: update-mitre
update-mitre:
	mkdir -p mitre
	wget -O mitre/enterprise-attack.json https://github.com/mitre-attack/attack-stix-data/raw/refs/heads/master/enterprise-attack/enterprise-attack-${ATTCK_RELEASE}.json
	wget -O mitre/ics-attack.json https://github.com/mitre-attack/attack-stix-data/raw/refs/heads/master/ics-attack/ics-attack-${ATTCK_RELEASE}.json
	wget -O mitre/mobile-attack.json https://github.com/mitre-attack/attack-stix-data/raw/refs/heads/master/mobile-attack/mobile-attack-${ATTCK_RELEASE}.json