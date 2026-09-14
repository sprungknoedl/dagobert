.PHONY: build build-web build-go check fmt vet test lint validate-exports docker run clean
.EXPORT_ALL_VARIABLES:
-include .env

TAILWIND_VERSION     = 4.3.3
DAISYUI_VERSION      = 5.7.37
GOLANGCI_LINT_VERSION = 2.13.2

# Supply-chain pinning: these binaries/plugins are downloaded from GitHub
# releases during build-web and end up embedded + served, so each is pinned to a
# specific version *and* sha256. The tailwind binary is platform-specific, so its
# checksum is selected per OS/arch below. Refresh these when bumping a version:
# download the release asset and run `sha256sum` (macOS: `shasum -a 256`).
TAILWIND_SHA_macos_arm64 = cdf646702987a743464dff4d9c60fd4480d1c1e73dd819a9a67f1078815dce9d
TAILWIND_SHA_macos_x64   = 7922e0953f2110c05976e3bf58f14e643d90427575e766b7d433f5f80cbee7e1
TAILWIND_SHA_linux_arm64 = 55fd0b241214eff3de1e8ee4f22796662f2d2e7a49bcfca7477cfd0bac398195
TAILWIND_SHA_linux_x64   = dc61b3ac6b8c9ca874c0cc4c57b2409791a64c5540404ca5f5367360babc313a
DAISYUI_SHA              = cac228b0060dd62971e94d410f40bb80e3d40529bf38c92395fb77a4cb153a43
DAISYUI_THEME_SHA        = abf84fdcae23840cdd08c01fcf2305836db13f7e13c102a27bbe974e739bc119

GOLANGCI_LINT_SHA_darwin_arm64 = f4bf83f0b64f055c42b28fc9a38861839f69c096e61c788e72dfaae412011789
GOLANGCI_LINT_SHA_darwin_amd64 = 8a13aaf9cbbb1dee52824e862cf0d0720e5bb97c1f4260d1e51623a09492b57b
GOLANGCI_LINT_SHA_linux_arm64  = a2a4e0065aa41be71f7c5ac90f271b61751331e5d04314e62afe4027855f0893
GOLANGCI_LINT_SHA_linux_amd64  = 2277d43b98ec0054280f2ac26b53268bae97682444678a59a657dd565da021d6

# Export validation: external validators for the OpenIOC / STIX indicator
# exports. The OpenIOC 1.1 XSD is vendored under pkg/openioc/testdata; the STIX
# validator and its (submodule-pinned) JSON schemas are bootstrapped into tmp/.
STIX_VALIDATOR_VERSION = 3.3.1
STIX_SCHEMAS_SHA       = c4f8d589acf2bdb3783655c89e0ffb6e150006ae
STIX_VENV              = tmp/stix-validator-venv
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

# golangci-lint releases use Go's own os/arch naming (darwin/amd64), unlike the
# tailwindcss asset names above.
GL_OS = $(shell echo $(UNAME_S) | tr A-Z a-z)
ifneq ($(filter $(UNAME_M),arm64 aarch64),)
  GL_ARCH = arm64
else
  GL_ARCH = amd64
endif

ifeq ($(UNAME_S),Darwin)
  SHA256 = shasum -a 256
else
  SHA256 = sha256sum
endif

TAILWIND_BIN = tmp/tailwindcss-$(TAILWIND_VERSION)
TAILWIND_SHA = $(TAILWIND_SHA_$(TW_OS)_$(TW_ARCH))

GOLANGCI_LINT_TARBALL = tmp/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GL_OS)-$(GL_ARCH).tar.gz
GOLANGCI_LINT_SHA     = $(GOLANGCI_LINT_SHA_$(GL_OS)_$(GL_ARCH))
GOLANGCI_LINT_BIN     = tmp/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GL_OS)-$(GL_ARCH)/golangci-lint

# verify,<file>,<expected-sha256>: fail closed on a supply-chain mismatch. The
# downloaded file is removed (so a half-finished build can't serve poisoned
# bytes and a re-run re-fetches) and a nix-style hash report is printed before
# exiting non-zero.
define verify
@actual=$$($(SHA256) "$(1)" | awk '{print $$1}'); if [ "$$actual" != "$(2)" ]; then rm -f "$(1)"; echo "error: hash mismatch for $(1):"; echo "         specified: sha256:$(2)"; echo "                got: sha256:$$actual"; exit 1; fi
endef

build: build-web build-go

$(TAILWIND_BIN):
	mkdir -p tmp
	wget -O $@ https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-$(TW_OS)-$(TW_ARCH)
	$(call verify,$@,$(TAILWIND_SHA))
	chmod +x $@

tmp/daisyui-$(DAISYUI_VERSION).js:
	mkdir -p tmp
	wget -O $@ https://github.com/saadeghi/daisyui/releases/download/v$(DAISYUI_VERSION)/daisyui.js
	$(call verify,$@,$(DAISYUI_SHA))

tmp/daisyui-theme-$(DAISYUI_VERSION).js:
	mkdir -p tmp
	wget -O $@ https://github.com/saadeghi/daisyui/releases/download/v$(DAISYUI_VERSION)/daisyui-theme.js
	$(call verify,$@,$(DAISYUI_THEME_SHA))

# Unversioned copies so the @plugin paths in dagobert.css stay version-free.
internal/frontend/daisyui.js: tmp/daisyui-$(DAISYUI_VERSION).js
	cp $< $@

internal/frontend/daisyui-theme.js: tmp/daisyui-theme-$(DAISYUI_VERSION).js
	cp $< $@

build-web: $(TAILWIND_BIN) internal/frontend/daisyui.js internal/frontend/daisyui-theme.js
	$(TAILWIND_BIN) -m -i internal/frontend/dagobert.css -o internal/assets/dagobert.css

$(GOLANGCI_LINT_TARBALL):
	mkdir -p tmp
	wget -O $@ https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GL_OS)-$(GL_ARCH).tar.gz
	$(call verify,$@,$(GOLANGCI_LINT_SHA))

$(GOLANGCI_LINT_BIN): $(GOLANGCI_LINT_TARBALL)
	tar -xzf $< -C tmp
	touch $@

clean:
	rm -f tmp/tailwindcss-* tmp/daisyui-*.js tmp/golangci-lint-*
	rm -rf tmp/golangci-lint-*/
	rm -f internal/frontend/daisyui.js internal/frontend/daisyui-theme.js
	rm -rf $(STIX_VENV)

build-go:
	go tool templ generate
	CGO_ENABLED=0 go build -o dagobert .

# Canonical verification: run this (and CI runs it) before trusting a change.
# build-go runs `templ generate` first, so vet/test/lint see the generated *_templ.go.
check: build-go vet test lint
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

lint: $(GOLANGCI_LINT_BIN)
	$(GOLANGCI_LINT_BIN) run

# Validate the generated OpenIOC / STIX indicator exports against external
# validators (xmllint + the OpenIOC 1.1 XSD, stix2-validator + STIX 2.1 schemas).
# Not part of `check`: it needs network on first run and tools outside the Go
# toolchain. Run it after changing the export mapping in internal/handler/indicators.go.
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
	docker build . -t sprungknoedl/dagobert

run:
	air