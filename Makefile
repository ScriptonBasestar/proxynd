ENV=develop
DOCKER_REGISTRY=scriptonbasestar
VERSION?=latest

# Override only for specific targets that need these paths
PROD_STORAGE_DIR=~/tmp/storage/
PROD_CONFIG_DIR=~/tmp/config/

.PHONY: setup
setup:
	mkdir -p $(PROD_CONFIG_DIR)
	mkdir -p $(PROD_STORAGE_DIR)
	cp -r sample-conf/* $(PROD_CONFIG_DIR)

.PHONY: docker-build
docker-build:
	@echo "Building..."
	docker compose build --no-cache

.PHONY: docker-build-multiarch
docker-build-multiarch:
	@echo "Building multi-architecture images..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION}

.PHONY: docker-build-multiarch-push
docker-build-multiarch-push:
	@echo "Building and pushing multi-architecture images..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --push

.PHONY: docker-build-amd64
docker-build-amd64:
	@echo "Building AMD64 image..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --platforms linux/amd64 --load

.PHONY: docker-build-arm64
docker-build-arm64:
	@echo "Building ARM64 image..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --platforms linux/arm64 --load

.PHONY: docker-push
docker-push:
	docker tag 'local_dev/proxynd' ${DOCKER_REGISTRY}/proxynd:latest
	docker push ${DOCKER_REGISTRY}/proxynd:latest

docker-run:
	@echo "Running..."
	docker compose up -d
	@echo "image name : proxynd"

docker-enter:
	@echo "Entering..."
	docker exec -it proxynd bash

.PHONY: local-buildlocal-

.PHONY: local-run
local-run:
	@echo "Running...?"
	./proxynd

.PHONY: dev-prepare
dev-prepare:
	@echo "Preparing development dependencies..."
	GOSUMDB=sum.golang.org go mod download
	GOSUMDB=sum.golang.org go mod tidy
	@echo "Installing air for hot reload..."
	@which air > /dev/null || GOSUMDB=sum.golang.org go install github.com/cosmtrek/air@latest
	@echo "Development dependencies ready!"

.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@mkdir -p ./tmp/storage
	@mkdir -p ./tmp/config
	@cp -r sample-conf/* ./tmp/config/
	@echo "Creating .env file..."
	@echo "CONFIG_DIR=./tmp/config" > .env
	@echo "STORAGE_DIR=./tmp/storage" >> .env
	@echo "SERVER_PORT=8080" >> .env
	@echo "Development environment ready!"
	@echo "Config dir: ./tmp/config"
	@echo "Storage dir: ./tmp/storage"
	@echo "Environment variables:"
	@cat .env

.PHONY: dev-run
dev-run: dev-setup
	@echo "Running with Air (hot reload)..."
	@which air > /dev/null || (echo "Air not found. Installing..." && go install github.com/cosmtrek/air@latest)
	air

.PHONY: dev-run-direct
dev-run-direct: dev-setup
	@echo "Running directly with go run..."
	CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage SERVER_PORT=8080 go run main.go

.PHONY: dev-test
dev-test:
	@echo "Running tests..."
	go test -v ./...

.PHONY: test-services
test-services:
	@echo "Running service tests with coverage..."
	./scripts/run_service_tests.sh

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

.PHONY: test-race
test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running comprehensive coverage analysis..."
	@./scripts/test-coverage.sh

.PHONY: test-runner
test-runner:
	@echo "Running tests with test runner script..."
	@./scripts/run-tests.sh

.PHONY: test-runner-unit
test-runner-unit:
	@echo "Running unit tests only..."
	@./scripts/run-tests.sh -t unit -v

.PHONY: test-runner-coverage
test-runner-coverage:
	@echo "Running tests with coverage..."
	@./scripts/run-tests.sh -c

.PHONY: test-benchmark
test-benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./internal/services/...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	./scripts/run_integration_tests.sh

.PHONY: test-integration-bench
test-integration-bench:
	@echo "Running integration tests with benchmarks..."
	./scripts/run_integration_tests.sh --bench

.PHONY: test-all
test-all: test-unit test-race test-services
	@echo "All tests completed!"

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Organizing imports..."
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w -local proxynd .
	@echo "Code formatting complete!"

.PHONY: format
format: fmt format-check
	@echo "✅ All formatting complete!"

.PHONY: format-check
format-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ The following files need formatting:"; \
		gofmt -l .; \
		echo "Run 'make format' to fix."; \
		exit 1; \
	else \
		echo "✅ All files are properly formatted"; \
	fi

.PHONY: format-diff
format-diff:
	@echo "Showing formatting differences..."
	@gofmt -d .

.PHONY: format-imports
format-imports:
	@echo "Organizing imports..."
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	@goimports -w -local proxynd .
	@echo "✅ Imports organized!"

.PHONY: format-simplify
format-simplify:
	@echo "Simplifying code..."
	@which gofmt > /dev/null && gofmt -s -w .
	@echo "✅ Code simplified!"

.PHONY: format-all
format-all: install-format-tools
	@echo "Running all formatters..."
	@echo "1. Standard formatting..."
	@gofmt -w .
	@echo "2. Simplifying code..."
	@gofmt -s -w .
	@echo "3. Organizing imports..."
	@goimports -w -local proxynd .
	@echo "4. Running gofumpt (strict formatting)..."
	@gofumpt -w -extra .
	@echo "5. Running gci (import grouping)..."
	@gci write --skip-generated -s standard -s default -s "prefix(proxynd)" .
	@echo "✅ All formatting complete!"

.PHONY: install-format-tools
install-format-tools:
	@echo "Installing formatting tools..."
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	@which gofumpt > /dev/null || (echo "Installing gofumpt..." && go install mvdan.cc/gofumpt@latest)
	@which gci > /dev/null || (echo "Installing gci..." && go install github.com/daixiang0/gci@latest)
	@echo "✅ All formatting tools installed!"

.PHONY: format-ci
format-ci: format-check
	@echo "CI format check passed!"

.PHONY: install-golangci-lint
install-golangci-lint:
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
	@echo "golangci-lint installed!"

.PHONY: lint
lint: install-golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run ./... --skip-files "examples/.*"

.PHONY: lint-fix
lint-fix: install-golangci-lint
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix ./...

.PHONY: lint-new
lint-new: install-golangci-lint
	@echo "Running golangci-lint on new code only..."
	golangci-lint run --new-from-rev=HEAD~ ./...

.PHONY: lint-ci
lint-ci:
	@echo "Running golangci-lint for CI..."
	golangci-lint run --out-format=github-actions ./...

.PHONY: install-mockery
install-mockery:
	@echo "Installing mockery..."
	@which mockery > /dev/null || go install github.com/vektra/mockery/v2@latest
	@echo "Mockery installed!"

.PHONY: generate-mocks
generate-mocks: install-mockery
	@echo "Generating mocks..."
	mockery --config .mockery.yaml
	@echo "Mock generation complete!"

.PHONY: clean-mocks
clean-mocks:
	@echo "Cleaning generated mocks..."
	@find . -type d -name "mocks" -exec rm -rf {} + 2>/dev/null || true
	@echo "Mocks cleaned!"

.PHONY: update-mocks
update-mocks: clean-mocks generate-mocks
	@echo "Mocks updated!"

.PHONY: dev-teardown
dev-teardown:
	@echo "Cleaning up development environment..."
	@rm -rf ./tmp/
	@rm -f .env
	@echo "Development environment cleaned!"

# Quality Assurance Targets
.PHONY: quality
quality: fmt lint test-coverage
	@echo "✅ All quality checks passed!"

.PHONY: quality-fix
quality-fix: fmt lint-fix
	@echo "✅ Code quality fixes applied!"

.PHONY: check
check: lint test-unit
	@echo "✅ Quick checks passed!"

.PHONY: check-all
check-all: lint test-all test-coverage
	@echo "✅ All checks passed!"

# Security & Vulnerability Scanning
.PHONY: security
security: security-deps security-code
	@echo "✅ Security checks completed!"

.PHONY: security-deps
security-deps:
	@echo "Checking dependencies for vulnerabilities..."
	@which nancy > /dev/null || go install github.com/sonatype-nexus-community/nancy@latest
	go list -json -deps ./... | nancy sleuth

.PHONY: security-code
security-code:
	@echo "Running security code analysis..."
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -fmt=json -out=gosec-report.json ./... || true
	@echo "Security report generated: gosec-report.json"

# Code Generation
.PHONY: generate
generate:
	@echo "Running code generation..."
	go generate ./...
	@echo "Code generation complete!"

# Dependency Management
.PHONY: deps
deps:
	@echo "Managing dependencies..."
	go mod download
	go mod tidy
	go mod verify
	@echo "Dependencies verified!"

.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "Dependencies updated!"

.PHONY: deps-graph
deps-graph:
	@echo "Generating dependency graph..."
	@which godepgraph > /dev/null || go install github.com/kisielk/godepgraph@latest
	godepgraph -s ./... | dot -Tpng -o deps-graph.png
	@echo "Dependency graph saved to deps-graph.png"

# Code Analysis
.PHONY: analyze
analyze: analyze-complexity analyze-unused
	@echo "✅ Code analysis complete!"

.PHONY: analyze-complexity
analyze-complexity:
	@echo "Analyzing code complexity..."
	@which gocyclo > /dev/null || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 10 .

.PHONY: analyze-unused
analyze-unused:
	@echo "Finding unused code..."
	@which unused > /dev/null || go install honnef.co/go/tools/cmd/unused@latest
	unused ./...

# Documentation
.PHONY: docs
docs: docs-generate docs-serve

.PHONY: docs-generate
docs-generate:
	@echo "Generating documentation..."
	@which godoc > /dev/null || go install golang.org/x/tools/cmd/godoc@latest
	@echo "Documentation can be viewed at http://localhost:6060/pkg/proxynd/"

.PHONY: docs-serve
docs-serve:
	@echo "Starting documentation server..."
	godoc -http=:6060

# Cleanup
# Cleanup targets
.PHONY: clean-build
clean-build:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f proxynd proxyndctl bin/proxynd cmd/proxynd/proxynd cmd/proxyndctl/proxyndctl
	@rm -f tmp/main
	@rm -f dist/*
	@find . -name "*.exe" -o -name "*.out" -type f -delete 2>/dev/null || true
	@find . -name "*.test" -type f -delete 2>/dev/null || true
	@echo "✓ Build artifacts cleaned!"

.PHONY: clean-test
clean-test:
	@echo "📊 Cleaning test files..."
	@rm -f coverage.out coverage.html coverage.txt
	@rm -f *.prof *.trace *.pprof
	@find . -name "*.test" -type f -delete 2>/dev/null || true
	@echo "✓ Test files cleaned!"

.PHONY: clean-logs
clean-logs:
	@echo "📝 Cleaning log files..."
	@mkdir -p logs tests/integration/logs
	@find logs -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
	@find tests -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
	@find integration -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
	@echo "✓ Log files cleaned!"

.PHONY: clean-cache
clean-cache:
	@echo "🗄️ Cleaning caches..."
	@go clean -cache -testcache 2>/dev/null || true
	@rm -rf .cache 2>/dev/null || true
	@echo "✓ Caches cleaned!"

.PHONY: clean-analysis
clean-analysis:
	@echo "🔍 Cleaning analysis reports..."
	@rm -f gosec-report.json staticcheck-report.json golangci-lint-report.json
	@rm -f deps-graph.png security-report.json
	@echo "✓ Analysis reports cleaned!"

.PHONY: clean-dev
clean-dev:
	@echo "🛠️ Cleaning development files..."
	@rm -rf ./tmp/storage/* 2>/dev/null || true
	@rm -rf ./.env.local 2>/dev/null || true
	@find . -name "*.tmp" -o -name "*.temp" -type f -delete 2>/dev/null || true
	@echo "✓ Development files cleaned!"

.PHONY: clean-deep
clean-deep: clean-build clean-test clean-logs clean-analysis clean-dev
	@echo "🗑️ Deep cleaning..."
	@go clean -modcache 2>/dev/null || true
	@rm -rf cache/packages/* 2>/dev/null || true
	@echo "✓ Deep clean completed!"

.PHONY: clean-all
clean-all: clean-build clean-test clean-logs clean-cache clean-analysis clean-dev
	@echo "✅ All cleanup tasks completed!"

.PHONY: clean
clean: clean-build clean-test clean-cache clean-analysis
	@echo "✓ Standard cleanup completed!"

.PHONY: clean-script
clean-script:
	@echo "🧹 Running comprehensive cleanup script..."
	@./scripts/clean.sh

# Development Workflow Helpers
.PHONY: dev
dev: dev-prepare dev-setup
	@echo "Development environment ready!"

.PHONY: ci
ci: deps lint test-coverage
	@echo "CI checks passed!"

# Build targets
.PHONY: build
build:
	@echo "Building proxynd..."
	go build -v -o proxynd .
	@echo "Build complete: ./proxynd"

.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/proxynd-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o dist/proxynd-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/proxynd-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/proxynd-darwin-arm64 .
	@echo "Multi-platform build complete!"

# Installation targets
.PHONY: install
install:
	@echo "Installing proxynd..."
	go install -v .
	@echo "✅ proxynd installed to $(shell go env GOPATH)/bin/proxynd"

.PHONY: install-dev
install-dev: install-tools install
	@echo "✅ Development installation complete!"

.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	@echo "Installing air (hot reload)..."
	@go install github.com/cosmtrek/air@latest
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing mockery..."
	@go install github.com/vektra/mockery/v2@latest
	@echo "Installing godoc..."
	@go install golang.org/x/tools/cmd/godoc@latest
	@echo "Installing gocyclo..."
	@go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@echo "Installing gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Installing nancy..."
	@go install github.com/sonatype-nexus-community/nancy@latest
	@echo "Installing godepgraph..."
	@go install github.com/kisielk/godepgraph@latest
	@echo "Installing unused..."
	@go install honnef.co/go/tools/cmd/unused@latest
	@echo "✅ All development tools installed!"

.PHONY: install-test
install-test: install-tools
	@echo "Installing test tools..."
	@echo "Installing gotestsum..."
	@go install gotest.tools/gotestsum@latest
	@echo "Installing richgo (colored test output)..."
	@go install github.com/kyoh86/richgo@latest
	@echo "Installing go-junit-report..."
	@go install github.com/jstemmer/go-junit-report/v2@latest
	@echo "Installing goconvey..."
	@go install github.com/smartystreets/goconvey@latest
	@echo "✅ All test tools installed!"

.PHONY: install-all
install-all: install-tools install-test install
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

.PHONY: uninstall
uninstall:
	@echo "Uninstalling proxynd..."
	@rm -f $(shell go env GOPATH)/bin/proxynd
	@echo "✅ proxynd uninstalled"

.PHONY: check-tools
check-tools:
	@echo "Checking installed tools..."
	@echo -n "air: "; which air > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "golangci-lint: "; which golangci-lint > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "goimports: "; which goimports > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "mockery: "; which mockery > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "godoc: "; which godoc > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "gocyclo: "; which gocyclo > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "gosec: "; which gosec > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "nancy: "; which nancy > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "godepgraph: "; which godepgraph > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "unused: "; which unused > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "gotestsum: "; which gotestsum > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "richgo: "; which richgo > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "go-junit-report: "; which go-junit-report > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"
	@echo -n "goconvey: "; which goconvey > /dev/null 2>&1 && echo "✅ installed" || echo "❌ not installed"

# Version management
.PHONY: version
version:
	@git describe --tags --always --dirty

# Help target
.PHONY: help
help:
	@echo "ProxyND Makefile Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Prepare development environment"
	@echo "  make dev-run      - Run with hot reload (Air)"
	@echo "  make dev-test     - Run all tests"
	@echo ""
	@echo "Quality:"
	@echo "  make quality      - Run all quality checks"
	@echo "  make quality-fix  - Apply automatic fixes"
	@echo "  make check        - Quick lint and test"
	@echo "  make check-all    - Comprehensive checks"
	@echo ""
	@echo "Testing:"
	@echo "  make test-unit    - Run unit tests"
	@echo "  make test-coverage - Generate comprehensive coverage report"
	@echo "  make test-race    - Run tests with race detector"
	@echo "  make test-runner  - Run tests with advanced test runner"
	@echo "  make test-runner-unit - Run unit tests with runner (verbose)"
	@echo "  make test-runner-coverage - Run tests with coverage using runner"
	@echo ""
	@echo "Code:"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linters"
	@echo "  make lint-fix     - Fix linting issues"
	@echo ""
	@echo "Security:"
	@echo "  make security     - Run security scans"
	@echo "  make analyze      - Analyze code complexity"
	@echo ""
	@echo "Build:"
	@echo "  make build        - Build binary"
	@echo "  make docker-build - Build Docker image"
	@echo ""
	@echo "Installation:"
	@echo "  make install      - Install proxynd binary"
	@echo "  make install-dev  - Install with all development tools"
	@echo "  make install-test - Install with test tools"
	@echo "  make install-all  - Install everything (binary + all tools)"
	@echo "  make uninstall    - Remove proxynd binary"
	@echo "  make check-tools  - Check which tools are installed"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean        - Standard cleanup (build + test + cache + analysis)"
	@echo "  make clean-build  - Clean build outputs only"
	@echo "  make clean-test   - Clean test files only"
	@echo "  make clean-logs   - Clean log files only"
	@echo "  make clean-cache  - Clean Go caches only"
	@echo "  make clean-analysis - Clean analysis reports only"
	@echo "  make clean-dev    - Clean development files only"
	@echo "  make clean-deep   - Deep clean (includes modcache)"
	@echo "  make clean-all    - Clean everything"
	@echo "  make clean-script - Run comprehensive cleanup script"
	@echo ""
	@echo "Validation:"
	@echo "  make validate     - Run all validations"
	@echo "  make validate-config - Validate configuration files"
	@echo "  make validate-modules - Validate Go modules"
	@echo "  make validate-gitignore - Validate .gitignore patterns"
	@echo ""
	@echo "Other:"
	@echo "  make deps         - Manage dependencies"
	@echo "  make docs         - Generate documentation"
	@echo "  make help         - Show this help"

# Default target
.DEFAULT_GOAL := help

.PHONY: pre-commit-install
pre-commit-install:
	@echo "Setting up pre-commit hooks..."
	@./scripts/setup_precommit.sh

.PHONY: pre-commit-run
pre-commit-run:
	@echo "Running pre-commit on all files..."
	pre-commit run --all-files

.PHONY: pre-commit-update
pre-commit-update:
	@echo "Updating pre-commit hooks..."
	pre-commit autoupdate

# Validate targets
.PHONY: validate
validate: validate-config validate-modules validate-gitignore
	@echo "✅ Validation complete!"

.PHONY: validate-config
validate-config:
	@echo "Validating configuration files..."
	@for file in sample-conf/*.yaml; do \
		echo "Checking $$file..."; \
		yamllint $$file || true; \
	done

.PHONY: validate-modules
validate-modules:
	@echo "Validating Go modules..."
	go mod verify

.PHONY: validate-gitignore
validate-gitignore:
	@echo "Validating .gitignore patterns..."
	@./scripts/validate-gitignore.sh
