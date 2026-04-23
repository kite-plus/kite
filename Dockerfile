# syntax=docker/dockerfile:1.7

# ---- Stage 1: build frontend SPA (pinned to native build platform — JS output is arch-independent) ----
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /web
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/admin/ ./
RUN npm run build

# ---- Stage 2: cross-compile Go binary on native toolchain ----
# Run the compiler natively (BUILDPLATFORM) and cross-compile to TARGETOS/TARGETARCH
# via Go's built-in cross-compilation. Avoids QEMU emulation entirely — pure-Go deps
# only (sqlite via glebarez/modernc.org, no CGO needed).
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend
ARG TARGETOS TARGETARCH
# Build identity stamped into the Go binary via -ldflags. CI/release tooling
# overrides these; local builds fall back to "dev" + current HEAD SHA.
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /web/dist ./web/admin/dist

# If the caller didn't pass COMMIT/BUILD_DATE explicitly, derive them from
# the repo checkout so the resulting image still reports something useful.
RUN : "${COMMIT:=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}" \
    && : "${BUILD_DATE:=$(date -u +%Y-%m-%dT%H:%M:%SZ)}" \
    && echo "building kite ${VERSION} (${COMMIT}) on ${BUILD_DATE}"

ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w \
      -X github.com/kite-plus/kite/internal/version.Version=${VERSION} \
      -X github.com/kite-plus/kite/internal/version.Commit=${COMMIT} \
      -X github.com/kite-plus/kite/internal/version.Date=${BUILD_DATE}" \
    -o /out/kite ./cmd/kite

# ---- Stage 3: minimal runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S kite \
    && adduser -S -G kite -h /app kite \
    && mkdir -p /app/data \
    && chown -R kite:kite /app

WORKDIR /app
COPY --from=backend /out/kite /app/kite

USER kite

ENV KITE_HOST=0.0.0.0 \
    KITE_PORT=8080 \
    KITE_DB_DRIVER=sqlite \
    KITE_DSN=/app/data/kite.db \
    GIN_MODE=release

EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/kite"]
