# OpenClaude v4 — local build and CI parity with CONTRIBUTING.md
# Usage: make help | make ci | make tag VER=1.2.3

.PHONY: help all ci build run run-live install test vet lint lint-install clean \
	tag bump-tag-patch bump-tag-minor bump-tag-major

BINARY   := occli
BINDIR   := bin
BINOUT   := $(BINDIR)/$(BINARY)
PACKAGE  := ./cmd/openclaude
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
# Version baked into binaries; override when cutting a release, e.g. VERSION=1.2.3 make build
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo 0.0.0-dev)
LDFLAGS  := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

GOLANGCI_VERSION ?= v2.9.0

help: ## Show targets
	@echo "Targets:"
	@grep -hE '^[a-zA-Z0-9_.-]+:.*?##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

all: ci build ## CI checks then build

ci: test vet lint ## Same checks as documented for contributors (test, vet, golangci-lint)

build: ## Build occli into ./bin
	@mkdir -p $(BINDIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINOUT) $(PACKAGE)

run: build ## Run built binary; pass extra CLI args with ARGS='...'
	./$(BINOUT) $(ARGS)

run-live: ## go run with release-style ldflags (no local binary file)
	go run -trimpath -ldflags "$(LDFLAGS)" $(PACKAGE) $(ARGS)

install: ## Install occli to GOPATH/bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o "$$(go env GOPATH)/bin/$(BINARY)" $(PACKAGE)

test: ## go test ./...
	go test ./...

vet: ## go vet ./...
	go vet ./...

lint: ## golangci-lint run (install with: make lint-install)
	golangci-lint run

lint-install: ## Install golangci-lint v2.9.0 into GOPATH/bin (override GOLANGCI_VERSION=)
	bash -c 'curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$$(go env GOPATH)/bin" $(GOLANGCI_VERSION)'

clean: ## Remove ./bin/occli
	rm -f $(BINOUT)

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
