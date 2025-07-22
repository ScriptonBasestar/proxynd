# Makefile.build.mk - Build and Installation Management
# Build, installation, and binary management targets

# ==============================================================================
# Build Configuration
# ==============================================================================

.PHONY: build build-all install install-dev install-all uninstall

# Build variables
VERSION?=latest

# ==============================================================================
# Go Build Targets
# ==============================================================================

build: ## build golang binary
	@echo "Building proxynd..."
	go build -v -o proxynd .
	@echo "Build complete: ./proxynd"

build-all: ## build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/proxynd-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o dist/proxynd-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/proxynd-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/proxynd-darwin-arm64 .
	@echo "Multi-platform build complete!"

# ==============================================================================
# Installation Targets
# ==============================================================================

install: ## install proxynd binary
	@echo "Installing proxynd..."
	go install -v .
	@echo "✅ proxynd installed to $(shell go env GOPATH)/bin/proxynd"

install-dev: install-tools install ## install with development tools
	@echo "✅ Development installation complete!"

install-all: install-tools install-test install ## complete installation with all tools
	@echo "✅ Complete installation finished!"
	@echo ""
	@echo "Installed binaries:"
	@echo "  - proxynd: $(shell go env GOPATH)/bin/proxynd"
	@echo ""
	@echo "Installed tools:"
	@echo "  - air (hot reload)"
	@echo "  - golangci-lint (linting)"
	@echo "  - goimports (import formatting)"
	@echo "  - mockery (mock generation)"
	@echo "  - godoc (documentation)"
	@echo "  - gocyclo (complexity analysis)"
	@echo "  - gosec (security scanning)"
	@echo "  - nancy (dependency scanning)"
	@echo "  - godepgraph (dependency visualization)"
	@echo "  - unused (dead code detection)"
	@echo "  - gotestsum (test runner)"
	@echo "  - richgo (colored test output)"
	@echo "  - go-junit-report (JUnit reports)"
	@echo "  - goconvey (test UI)"

uninstall: ## uninstall proxynd binary
	@echo "Uninstalling proxynd..."
	@rm -f $(shell go env GOPATH)/bin/proxynd
	@echo "✅ proxynd uninstalled"


# ==============================================================================
# Code Generation
# ==============================================================================

.PHONY: generate generate-mocks clean-mocks update-mocks

generate: ## run go generate
	@echo "Running code generation..."
	go generate ./...
	@echo "Code generation complete!"

generate-mocks: install-mockery ## generate mock files
	@echo "Generating mocks..."
	mockery --config .mockery.yaml
	@echo "Mock generation complete!"


update-mocks: clean-mocks generate-mocks ## update all mock files
	@echo "Mocks updated!"
