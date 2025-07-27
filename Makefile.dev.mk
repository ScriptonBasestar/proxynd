# Makefile.dev.mk - Development Environment Management
# Development, setup, and local execution targets

# ==============================================================================
# Development Environment
# ==============================================================================

.PHONY: dev-prepare dev-setup dev-run dev-run-direct dev-teardown dev dev-test local-run
.PHONY: setup

# Override only for specific targets that need these paths
PROD_STORAGE_DIR=~/tmp/storage/
PROD_CONFIG_DIR=~/tmp/config/

setup: ## create production directories and copy sample configs
	mkdir -p $(PROD_CONFIG_DIR)
	mkdir -p $(PROD_STORAGE_DIR)
	cp -r examples/* $(PROD_CONFIG_DIR)

dev-prepare: ## prepare development dependencies (go mod + air)
	@echo "Preparing development dependencies..."
	GOSUMDB=sum.golang.org go mod download
	GOSUMDB=sum.golang.org go mod tidy
	@echo "Installing air for hot reload..."
	@which air > /dev/null || GOSUMDB=sum.golang.org go install github.com/cosmtrek/air@latest
	@echo "Development dependencies ready!"

dev-setup: ## setup development environment with tmp dirs and .env
	@echo "Setting up development environment..."
	@mkdir -p ./tmp/storage
	@mkdir -p ./tmp/config
	@cp -r examples/* ./tmp/config/
	@echo "Creating .env file..."
	@echo "CONFIG_DIR=./tmp/config" > .env
	@echo "STORAGE_DIR=./tmp/storage" >> .env
	@echo "SERVER_PORT=8080" >> .env
	@echo "Development environment ready!"
	@echo "Config dir: ./tmp/config"
	@echo "Storage dir: ./tmp/storage"
	@echo "Environment variables:"
	@cat .env

dev-run: dev-setup ## run with Air hot reload
	@echo "Running with Air (hot reload)..."
	@which air > /dev/null || (echo "Air not found. Installing..." && go install github.com/cosmtrek/air@latest)
	air

dev-run-direct: dev-setup ## run directly with go run (no hot reload)
	@echo "Running directly with go run..."
	CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage SERVER_PORT=8080 go run main.go

local-run: ## run the built binary locally
	@echo "Running...?"
	./proxynd

dev-teardown: ## clean up development environment
	@echo "Cleaning up development environment..."
	@rm -rf ./tmp/
	@rm -f .env
	@echo "Development environment cleaned!"

dev: dev-prepare dev-setup ## complete development environment setup
	@echo "Development environment ready!"

dev-test: ## run tests in development mode
	@echo "Running tests..."
	go test -v ./...

# ==============================================================================
# Version Management
# ==============================================================================

.PHONY: version

version: ## show current version from git
	@git describe --tags --always --dirty
