.PHONY: build build-web build-go check fmt vet test validate-exports docker run clean
.EXPORT_ALL_VARIABLES:
-include dagobert.env

TAILWIND_VERSION = 4.1.18
DAISYUI_VERSION  = 5.5.14

# Supply-chain pinning: these binaries/plugins are downloaded from GitHub
# releases during build-web and end up embedded + served, so each is pinned to a
# specific version *and* sha256. The tailwind binary is platform-specific, so its
# checksum is selected per OS/arch below. Refresh these when bumping a version:
# download the release asset and run `sha256sum` (macOS: `shasum -a 256`).
TAILWIND_SHA_macos_arm64 = 7f27711dceac1a580b6ad58ddac46e59202c85a6c16f2f08f6fdcdee261008e1
TAILWIND_SHA_macos_x64   = 1e8a77fd796a3a4aa3d8727887de926ef9d38477aba113fd7c32c0d31a32a3ab
TAILWIND_SHA_linux_arm64 = 7a7702db6c93718a9b6655d455304edda18600f5a4f195242342ed3b5b70ebe8
TAILWIND_SHA_linux_x64   = 737becf8d4ad1115ea98df69fa94026d402ca8feb91306a035b5b004167da8dd
DAISYUI_SHA              = a125069ec726eafc96893ed5384a9b00412856f4df9eeec2323246364a60da29
DAISYUI_THEME_SHA        = 968c584d25b216485a954d1b6e3b1a2b28d69032d2a2ed8a0ca16003965f201a

# Export validation: external validators for the OpenIOC / STIX indicator
# exports. The OpenIOC 1.1 XSD is vendored under pkg/openioc/testdata; the STIX
# validator and its (submodule-pinned) JSON schemas are bootstrapped into bin/.
STIX_VALIDATOR_VERSION = 3.3.1
STIX_SCHEMAS_SHA       = c4f8d589acf2bdb3783655c89e0ffb6e150006ae
STIX_VENV              = bin/stix-validator-venv
STIX_READY             = $(STIX_VENV)/.ready-$(STIX_VALIDATOR_VERSION)-$(STIX_SCHEMAS_SHA)

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

ifeq ($(UNAME_S),Darwin)
  SHA256 = shasum -a 256
else
  SHA256 = sha256sum
endif

TAILWIND_BIN = bin/tailwindcss-$(TAILWIND_VERSION)
TAILWIND_SHA = $(TAILWIND_SHA_$(TW_OS)_$(TW_ARCH))

# verify,<file>,<expected-sha256>: fail closed on a supply-chain mismatch. The
# downloaded file is removed (so a half-finished build can't serve poisoned
# bytes and a re-run re-fetches) and a nix-style hash report is printed before
# exiting non-zero.
define verify
@actual=$$($(SHA256) "$(1)" | awk '{print $$1}'); if [ "$$actual" != "$(2)" ]; then rm -f "$(1)"; echo "error: hash mismatch for $(1):"; echo "         specified: sha256:$(2)"; echo "                got: sha256:$$actual"; exit 1; fi
endef

build: build-web build-go

$(TAILWIND_BIN):
	mkdir -p bin
	wget -O $@ https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-$(TW_OS)-$(TW_ARCH)
	$(call verify,$@,$(TAILWIND_SHA))
	chmod +x $@

bin/daisyui-$(DAISYUI_VERSION).js:
	mkdir -p bin
	wget -O $@ https://github.com/saadeghi/daisyui/releases/download/v$(DAISYUI_VERSION)/daisyui.js
	$(call verify,$@,$(DAISYUI_SHA))

bin/daisyui-theme-$(DAISYUI_VERSION).js:
	mkdir -p bin
	wget -O $@ https://github.com/saadeghi/daisyui/releases/download/v$(DAISYUI_VERSION)/daisyui-theme.js
	$(call verify,$@,$(DAISYUI_THEME_SHA))

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
	rm -rf $(STIX_VENV)

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

# Validate the generated OpenIOC / STIX indicator exports against external
# validators (xmllint + the OpenIOC 1.1 XSD, stix2-validator + STIX 2.1 schemas).
# Not part of `check`: it needs network on first run and tools outside the Go
# toolchain. Run it after changing the export mapping in app/handler/indicators.go.
validate-exports: $(STIX_READY)
	@command -v xmllint >/dev/null 2>&1 || { echo "xmllint not found — install libxml2 (macOS: brew install libxml2)"; exit 1; }
	STIX2_VALIDATOR="$(CURDIR)/$(STIX_VENV)/bin/stix2_validator" \
		go test -tags validate -run TestValidate -count=1 -v ./pkg/openioc ./pkg/stix

# Bootstrap a pinned stix2-validator plus the exact JSON schemas it expects (the
# pip wheel ships without them). Re-runs only when the pinned versions change.
$(STIX_READY):
	@command -v python3 >/dev/null 2>&1 || { echo "python3 not found — required for stix2-validator"; exit 1; }
	rm -rf $(STIX_VENV)
	python3 -m venv $(STIX_VENV)
	$(STIX_VENV)/bin/pip -q install --upgrade pip
	$(STIX_VENV)/bin/pip -q install stix2-validator==$(STIX_VALIDATOR_VERSION)
	pkg=$$($(STIX_VENV)/bin/python -c "import stix2validator, os; print(os.path.dirname(stix2validator.__file__))"); \
		mkdir -p "$$pkg/schemas-2.1"; \
		curl -sSL "https://codeload.github.com/oasis-open/cti-stix2-json-schemas/tar.gz/$(STIX_SCHEMAS_SHA)" \
			| tar -xz -C "$$pkg/schemas-2.1" --strip-components=1
	touch $@

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