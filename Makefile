BINARY_NAME=kite
MAIN_PATH=cmd/kite/main.go
ADMIN_DIR=ui/admin
ADMIN_DIST_DIR=$(ADMIN_DIR)/dist
GO_FLAGS=-ldflags="-s -w"

.PHONY: all build build-ui build-server run test tidy clean help

all: tidy build-server

## build: Build backend and frontend if available
build: tidy build-server build-ui
	@echo "✅ Kite build completed."

## build-ui: Build React admin if package.json exists
build-ui:
	@if [ -f "$(ADMIN_DIR)/package.json" ]; then \
		echo "📦 Building admin frontend..."; \
		cd $(ADMIN_DIR) && npm install && npm run build; \
		echo "✅ Admin frontend build completed."; \
	else \
		echo "ℹ️  Skip build-ui: $(ADMIN_DIR)/package.json not found."; \
	fi

## build-server: Build Go backend binary
build-server:
	@echo "🚀 Building Go backend binary..."
	@go build $(GO_FLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Backend build completed: ./$(BINARY_NAME)"

## run: Run backend with default config
run:
	@go run $(MAIN_PATH)

## test: Run Go tests
test:
	@go test ./...

## tidy: Tidy Go modules
tidy:
	@echo "🧹 Tidying Go modules..."
	@go mod tidy

## clean: Remove build artifacts
clean:
	@echo "🗑️ Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(ADMIN_DIST_DIR)
	@echo "✨ Clean complete."

## help: Show available make targets
help:
	@echo "Kite make targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
