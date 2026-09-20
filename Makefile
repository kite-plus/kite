GO       ?= go
BINARY   ?= kite
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     ?= $(shell git log -1 --format=%cI 2>/dev/null || echo unknown)
LDFLAGS  := -s -w \
	-X github.com/kite-plus/kite/internal/buildinfo.Version=$(VERSION) \
	-X github.com/kite-plus/kite/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/kite-plus/kite/internal/buildinfo.Date=$(DATE)

.PHONY: all build install test test-race cover fmt vet lint check-imports check-tidy check clean tidy

all: check build

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

# Runs the same comparison CI does. Tidy rewrites the files as a side effect,
# which is the point: it leaves them in the state CI expects rather than only
# reporting that they were wrong.
check-tidy:
	@$(GO) mod tidy
	@git diff --exit-code -- go.mod go.sum \
		|| { echo "go.mod or go.sum was not tidy; the fix is staged above"; exit 1; }

check: fmt vet check-imports check-tidy lint test

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin coverage.out
