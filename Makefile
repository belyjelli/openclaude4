# OpenClaude v4 — local build and CI parity with CONTRIBUTING.md
# Usage: make help | make ci | make tag VER=1.2.3
# Cross-build: make build GOOS=linux GOARCH=arm64 | make build-all | TARGETS="linux/amd64" make build-all

.PHONY: help all ci build build-all run run-live install test vet lint lint-install clean clean-all \
	tag bump-tag-patch bump-tag-minor bump-tag-major

BINARY   := occli
BINDIR   := bin
PACKAGE  := ./cmd/openclaude
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
# Version baked into binaries; override when cutting a release, e.g. VERSION=1.2.3 make build
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo 0.0.0-dev)
LDFLAGS  := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

CGO_ENABLED ?= 0

# Cross-compile: set GOOS and/or GOARCH (e.g. make build GOOS=linux GOARCH=arm64).
# Omit both for a native binary at ./bin/occli.
GOOS   ?=
GOARCH ?=

_cross :=
ifneq ($(strip $(GOOS))$(strip $(GOARCH)),)
  _cross := 1
endif

ifeq ($(_cross),1)
  _goos   := $(if $(strip $(GOOS)),$(GOOS),$(shell go env GOOS))
  _goarch := $(if $(strip $(GOARCH)),$(GOARCH),$(shell go env GOARCH))
  _exe    := $(if $(filter windows Windows WINDOWS,$(_goos)),.exe,)
  BINOUT  := $(BINDIR)/$(BINARY)-$(_goos)-$(_goarch)$(_exe)
  BUILD_ENV := CGO_ENABLED=$(CGO_ENABLED)$(if $(strip $(GOOS)), GOOS=$(GOOS))$(if $(strip $(GOARCH)), GOARCH=$(GOARCH))
else
  BINOUT := $(BINDIR)/$(BINARY)
  BUILD_ENV :=
endif

# Space-separated os/arch pairs for build-all (same style as filebrowser-cli)
TARGETS ?= darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64

GOLANGCI_VERSION ?= v2.9.0

help: ## Show targets
	@echo "Targets:"
	@grep -hE '^[a-zA-Z0-9_.-]+:.*?##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

all: ci build ## CI checks then build

ci: test vet lint ## Same checks as documented for contributors (test, vet, golangci-lint)

build: ## Build occli into ./bin (cross: GOOS=linux GOARCH=arm64 make build)
	@mkdir -p $(BINDIR)
	GOWORK=off $(BUILD_ENV) go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINOUT) $(PACKAGE)

build-all: ## Release builds for TARGETS -> bin/occli_$(VERSION)_os_arch.exe.gz
	@mkdir -p $(BINDIR)
	@set -e; \
	for t in $(TARGETS); do \
		os="$${t%/*}"; \
		arch="$${t#*/}"; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(BINDIR)/$(BINARY)_$(VERSION)_$${os}_$${arch}$${ext}"; \
		echo "building $$os/$$arch -> $$out.gz"; \
		CGO_ENABLED=$(CGO_ENABLED) GOWORK=off GOOS="$$os" GOARCH="$$arch" go build -trimpath -ldflags "$(LDFLAGS)" -o "$$out" $(PACKAGE); \
		gzip -c "$$out" > "$$out.gz" && rm -f "$$out"; \
	done

run: build ## Run built binary; pass extra CLI args with ARGS='...'
	./$(BINOUT) $(ARGS)

run-live: ## go run with release-style ldflags (no local binary file)
	GOWORK=off go run -trimpath -ldflags "$(LDFLAGS)" $(PACKAGE) $(ARGS)

install: ## Install occli to GOPATH/bin
	GOWORK=off go build -trimpath -ldflags "$(LDFLAGS)" -o "$$(go env GOPATH)/bin/$(BINARY)" $(PACKAGE)

test: ## go test ./...
	go test ./...

vet: ## go vet ./...
	go vet ./...

lint: ## golangci-lint run (install with: make lint-install)
	golangci-lint run

lint-install: ## Install golangci-lint v2.9.0 into GOPATH/bin (override GOLANGCI_VERSION=)
	bash -c 'curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$$(go env GOPATH)/bin" $(GOLANGCI_VERSION)'

clean: ## Remove ./bin/occli (native binary only)
	rm -f $(BINDIR)/$(BINARY)

clean-all: ## Remove native binary, ad-hoc cross (occli-os-arch), and build-all occli_* artifacts
	rm -f $(BINDIR)/$(BINARY) $(BINDIR)/$(BINARY)-* $(BINDIR)/$(BINARY)_*

# --- Version tags (semver, v-prefix on the tag) ---
# Create a specific tag: make tag VER=1.2.3
tag: ## Annotated git tag; set VER=x.y.z (creates vx.y.z)
	@test -n "$(VER)" || (echo "Set VER=x.y.z, e.g. make tag VER=1.2.3"; exit 1)
	git tag -a "v$(VER)" -m "v$(VER)"

bump-tag-patch: ## Next patch tag from latest v* tag (v1.2.3 -> v1.2.4)
	@raw=$$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || true); \
	v=$${raw#v}; \
	[ -n "$$v" ] || v=0.0.0; \
	major=$$(echo "$$v" | cut -d. -f1); minor=$$(echo "$$v" | cut -d. -f2); patch=$$(echo "$$v" | cut -d. -f3); \
	patch=$${patch:-0}; \
	np=$$((patch + 1)); \
	new="$$major.$$minor.$$np"; \
	echo "Tagging v$$new (from $$raw)"; \
	git tag -a "v$$new" -m "v$$new"

bump-tag-minor: ## Next minor tag (v1.2.3 -> v1.3.0)
	@raw=$$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || true); \
	v=$${raw#v}; \
	[ -n "$$v" ] || v=0.0.0; \
	major=$$(echo "$$v" | cut -d. -f1); minor=$$(echo "$$v" | cut -d. -f2); \
	minor=$${minor:-0}; \
	nm=$$((minor + 1)); \
	new="$$major.$$nm.0"; \
	echo "Tagging v$$new (from $$raw)"; \
	git tag -a "v$$new" -m "v$$new"

bump-tag-major: ## Next major tag (v1.2.3 -> v2.0.0)
	@raw=$$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || true); \
	v=$${raw#v}; \
	[ -n "$$v" ] || v=0.0.0; \
	major=$$(echo "$$v" | cut -d. -f1); \
	major=$${major:-0}; \
	nm=$$((major + 1)); \
	new="$$nm.0.0"; \
	echo "Tagging v$$new (from $$raw)"; \
	git tag -a "v$$new" -m "v$$new"
