GO       ?= go
BINARY   ?= kite
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS  := -s -w \
	-X github.com/kite-plus/kite/internal/buildinfo.Version=$(VERSION) \
	-X github.com/kite-plus/kite/internal/buildinfo.Commit=$(COMMIT)

.PHONY: all build install test test-race cover fmt vet lint check-imports check clean tidy

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

lint:
	@command -v golangci-lint >/dev/null 2>&1 \
		&& golangci-lint run \
		|| echo "golangci-lint not installed, skipping"

check-imports:
	@sh scripts/check-imports.sh

check: fmt vet check-imports test

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin coverage.out
