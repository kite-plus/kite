# syntax=docker/dockerfile:1.7

# ---- Stage 1: build frontend SPA ----
FROM node:22-alpine AS frontend
WORKDIR /web
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/admin/ ./
RUN npm run build

# ---- Stage 2: build Go binary with embedded assets ----
FROM golang:1.25-alpine AS backend
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /web/dist ./web/admin/dist

ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath
RUN go build -ldflags="-s -w" -o /out/kite ./cmd/kite

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
    CMD wget -qO- http://127.0.0.1:8080/api/v1/setup/status >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/kite"]
