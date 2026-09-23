GO       ?= go
BINARY   ?= kite
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     ?= $(shell git log -1 --format=%cI 2>/dev/null || echo unknown)
LDFLAGS  := -s -w \
	-X github.com/kite-plus/kite/internal/buildinfo.Version=$(VERSION) \
	-X github.com/kite-plus/kite/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/kite-plus/kite/internal/buildinfo.Date=$(DATE)

.PHONY: all build install test test-race cover fmt vet lint check-imports check-tidy check clean tidy web web-gen web-check docker perf

all: check build

PNPM ?= pnpm

# The admin is built separately because it needs Node, which a Go-only
# contributor should not have to install. web/dist is committed empty, so a
# binary built without this target compiles and reports the admin is missing.
web:
	cd web && $(PNPM) install --frozen-lockfile && $(PNPM) build

# web-gen regenerates the API client from this build's own description, so the
# types the admin compiles against cannot describe an API the server does not
# serve.
web-gen:
	cd web && $(PNPM) gen

web-check:
	cd web && $(PNPM) lint

# perf times Kite against the latency targets in docs/design/architecture.md
# §29 on a generated site of 2000 posts. It takes a while and depends on the
# machine, so it is run before a release rather than as part of check.
perf:
	KITE_PERF=1 $(GO) test -count=1 -v ./internal/perf/

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/kite

install:
	CGO_ENABLED=0 $(GO) install -trimpath -ldflags '$(LDFLAGS)' ./cmd/kite

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

LINT_VERSION ?= v2.13.2

lint:
	$(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION) run ./...

check-imports:
	@sh scripts/check-imports.sh

# Compares the files before and after tidy rather than against HEAD: locally
# they often carry legitimate uncommitted edits, and diffing against the commit
# would report those as untidiness. CI never notices the difference because it
# starts from a clean checkout.
check-tidy:
	@cp go.mod .go.mod.check && cp go.sum .go.sum.check
	@$(GO) mod tidy
	@if ! cmp -s go.mod .go.mod.check || ! cmp -s go.sum .go.sum.check; then \
		rm -f .go.mod.check .go.sum.check; \
		echo "go.mod or go.sum was not tidy; tidy has fixed them in place"; \
		exit 1; \
	fi
	@rm -f .go.mod.check .go.sum.check

check: fmt vet check-imports check-tidy lint test

tidy:
	$(GO) mod tidy

IMAGE ?= kite

# The image compiles the binary and the admin itself, so this needs neither Go
# nor Node on the machine running it. The version it stamps is this checkout's,
# the same one `make build` would.
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

clean:
	rm -rf bin coverage.out
