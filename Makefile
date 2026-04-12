APP_NAME   := kite
BUILD_DIR  := build
WEB_DIR    := web/admin
GOFLAGS    := -trimpath
LDFLAGS    := -s -w

.PHONY: all build dev clean web-install web-build web-dev run

## ── 生产构建 ──────────────────────────────────────────────

all: build

# make build  构建前端 + 内嵌到 Go 二进制
build: web-build
	@echo "==> Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	GOPATH="$$HOME/go" GOMODCACHE="$$HOME/go/pkg/mod" \
	  go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/kite
	@echo "==> Done: $(BUILD_DIR)/$(APP_NAME)"

run: build
	./$(BUILD_DIR)/$(APP_NAME)

## ── 开发模式 ──────────────────────────────────────────────

# make dev  同时启动前端 dev server 和后端
dev:
	@echo "==> Starting development servers..."
	@$(MAKE) dev-backend &
	@$(MAKE) dev-frontend
	@wait

dev-backend:
	GOPATH="$$HOME/go" GOMODCACHE="$$HOME/go/pkg/mod" \
	  go run -tags dev ./cmd/kite

dev-frontend:
	cd $(WEB_DIR) && npm run dev

## ── 前端 ─────────────────────────────────────────────────

web-install:
	cd $(WEB_DIR) && npm install

web-build: web-install
	@echo "==> Building frontend..."
	cd $(WEB_DIR) && npm run build

web-dev:
	cd $(WEB_DIR) && npm run dev

## ── 清理 ─────────────────────────────────────────────────

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_DIR)/dist
