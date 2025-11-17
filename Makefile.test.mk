# Makefile.test.mk - Testing and Quality Assurance
# All testing-related targets including unit, integration, benchmarks, and coverage

# ==============================================================================
# 4-Layer Test Architecture (Hexagonal + Clean Architecture)
# ==============================================================================

.PHONY: test-unit test-contract test-integration test-e2e test-race test-services test-coverage
.PHONY: test-all test-runner test-runner-unit test-runner-coverage test-api verify-api

# Layer 1: Unit Tests (Domain/Usecase layers)
test-unit: ## run unit tests (domain/usecase layers)
	@echo "🧪 Running unit tests (domain/usecase layers)..."
	go test -v -short -race ./internal/domain/... ./internal/usecase/...

# Layer 2: Contract Tests (Ports layer)
test-contract: ## run contract tests (ports layer)
	@echo "🤝 Running contract tests (ports layer)..."
	go test -v -tags=contract ./internal/ports/...

# Layer 3: Integration Tests (Adapters layer)
test-integration: ## run integration tests (adapters layer)
	@echo "🔗 Running integration tests (adapters layer)..."
	go test -v -tags=integration ./tests/integration/...

# Layer 4: E2E Tests (Full system)
test-e2e: ## run end-to-end tests (full system)
	@echo "🌐 Running E2E tests (full system)..."
	@docker-compose -f docker-compose.e2e.yml up -d --wait
	@go test -v -tags=e2e ./tests/e2e/... || (docker-compose -f docker-compose.e2e.yml down && exit 1)
	@docker-compose -f docker-compose.e2e.yml down

# Legacy test targets (for backward compatibility)
test-unit-legacy: ## run legacy unit tests
	@echo "Running legacy unit tests..."
	go test -v -short ./...

test-race: ## run tests with race detector
	@echo "Running tests with race detector..."
	go test -race ./...

test-services: ## run service tests with coverage
	@echo "Running service tests with coverage..."
	./scripts/run_service_tests.sh

test-coverage: ## generate comprehensive coverage report
	@echo "Running comprehensive coverage analysis..."
	@./scripts/test-coverage.sh

test-integration-scripts: ## run integration tests via scripts
	@echo "Running integration tests via scripts..."
	./scripts/run_integration_tests.sh

test-integration-bench: ## run integration tests with benchmarks
	@echo "Running integration tests with benchmarks..."
	./scripts/run_integration_tests.sh --bench

# Combined test targets
test-all: test-unit test-contract test-integration ## run all layer tests (excluding E2E)
	@echo "✅ All layer tests completed!"

test-full: test-unit test-contract test-integration test-e2e ## run complete test suite including E2E
	@echo "✅ Full test suite completed!"

test-all-scripts: test-unit test-contract test-integration-scripts ## run all layer tests via scripts (excluding E2E)
	@echo "✅ All layer tests via scripts completed!"

test-legacy: test-unit-legacy test-race test-services ## run legacy test suite
	@echo "All legacy tests completed!"

# ==============================================================================
# Package Manager Specific Tests
# ==============================================================================

.PHONY: test-npm test-pip test-apt test-docker test-maven test-yum test-apk
.PHONY: test-package-managers

test-npm: ## run NPM proxy tests
	@echo "📦 Running NPM proxy tests..."
	go test -v -tags=integration ./tests/integration/npm_*
	go test -v ./handlers/proxy/npm_handler_test.go
	go test -v ./internal/adapters/http/fiber/handlers/proxy/npm_handler_test.go

test-pip: ## run PyPI proxy tests
	@echo "🐍 Running PyPI proxy tests..."
	go test -v -tags=integration ./tests/integration/pip_*
	go test -v ./handlers/proxy/pip_handler_test.go
	go test -v ./internal/adapters/http/fiber/handlers/proxy/pip_handler_test.go

test-apt: ## run APT proxy tests
	@echo "📋 Running APT proxy tests..."
	go test -v -tags=integration ./tests/integration/apt_*

test-docker: ## run Docker registry tests
	@echo "🐳 Running Docker registry tests..."
	go test -v -tags=integration ./tests/integration/docker_*
	go test -v ./handlers/proxy/docker_handler_test.go
	go test -v ./internal/adapters/http/fiber/handlers/proxy/docker_handler_test.go

test-maven: ## run Maven repository tests
	@echo "☕ Running Maven repository tests..."
	go test -v -tags=integration ./tests/integration/maven_*

test-yum: ## run YUM repository tests
	@echo "🔴 Running YUM repository tests..."
	go test -v -tags=integration ./tests/integration/yum_*

test-apk: ## run APK repository tests
	@echo "🏔️ Running APK repository tests..."
	go test -v -tags=integration ./tests/integration/apk_*

test-package-managers: test-npm test-pip test-apt test-docker test-maven test-yum test-apk ## run all package manager tests
	@echo "✅ All package manager tests completed!"

# ==============================================================================
# Coverage by Layer
# ==============================================================================

.PHONY: test-coverage-unit test-coverage-contract test-coverage-integration test-coverage-all
.PHONY: test-coverage-package-managers

test-coverage-unit: ## generate unit test coverage
	@echo "📊 Generating unit test coverage..."
	@mkdir -p ./reports/coverage
	go test -coverprofile=./reports/coverage/unit.coverage ./internal/domain/... ./internal/usecase/...
	go tool cover -html=./reports/coverage/unit.coverage -o ./reports/coverage/unit.html
	@echo "Unit coverage report: ./reports/coverage/unit.html"

test-coverage-contract: ## generate contract test coverage
	@echo "📊 Generating contract test coverage..."
	@mkdir -p ./reports/coverage
	go test -coverprofile=./reports/coverage/contract.coverage -tags=contract ./internal/ports/...
	go tool cover -html=./reports/coverage/contract.coverage -o ./reports/coverage/contract.html
	@echo "Contract coverage report: ./reports/coverage/contract.html"

test-coverage-integration: ## generate integration test coverage
	@echo "📊 Generating integration test coverage..."
	@mkdir -p ./reports/coverage
	go test -coverprofile=./reports/coverage/integration.coverage -tags=integration ./tests/integration/...
	go tool cover -html=./reports/coverage/integration.coverage -o ./reports/coverage/integration.html
	@echo "Integration coverage report: ./reports/coverage/integration.html"

test-coverage-package-managers: ## generate package manager test coverage
	@echo "📊 Generating package manager test coverage..."
	@mkdir -p ./reports/coverage
	go test -coverprofile=./reports/coverage/npm.coverage ./handlers/proxy/npm_handler_test.go ./internal/adapters/http/fiber/handlers/proxy/npm_handler_test.go
	go test -coverprofile=./reports/coverage/pip.coverage ./handlers/proxy/pip_handler_test.go ./internal/adapters/http/fiber/handlers/proxy/pip_handler_test.go
	go test -coverprofile=./reports/coverage/docker.coverage ./handlers/proxy/docker_handler_test.go ./internal/adapters/http/fiber/handlers/proxy/docker_handler_test.go
	@echo "Package manager coverage reports generated in ./reports/coverage/"

test-coverage-all: test-coverage-unit test-coverage-contract test-coverage-integration ## generate all coverage reports
	@echo "📊 Generating combined coverage report..."
	@mkdir -p ./reports/coverage
	echo "mode: set" > ./reports/coverage/combined.coverage
	tail -n +2 ./reports/coverage/unit.coverage >> ./reports/coverage/combined.coverage 2>/dev/null || true
	tail -n +2 ./reports/coverage/contract.coverage >> ./reports/coverage/combined.coverage 2>/dev/null || true
	tail -n +2 ./reports/coverage/integration.coverage >> ./reports/coverage/combined.coverage 2>/dev/null || true
	go tool cover -html=./reports/coverage/combined.coverage -o ./reports/coverage/combined.html
	@echo "✅ Combined coverage report: ./reports/coverage/combined.html"

# ==============================================================================
# API Testing Targets
# ==============================================================================

test-api: verify-api ## alias for verify-api

verify-api: ## verify all CLI API endpoints are working
	@echo "$(YELLOW)Verifying API endpoints...$(RESET)"
	@./scripts/verify-api-endpoints.sh http://localhost:8080

verify-api-json: ## verify API endpoints with JSON output
	@echo "$(YELLOW)Verifying API endpoints (JSON format)...$(RESET)"
	@./scripts/verify-api-endpoints.sh http://localhost:8080 --json

verify-api-verbose: ## verify API endpoints with verbose output
	@echo "$(YELLOW)Verifying API endpoints (verbose)...$(RESET)"
	@./scripts/verify-api-endpoints.sh http://localhost:8080 --verbose

# ==============================================================================
# Enterprise API Testing Targets
# ==============================================================================

.PHONY: test-enterprise-integration test-enterprise-rbac test-enterprise-audit
.PHONY: test-enterprise-analytics test-enterprise-security test-enterprise-all

test-enterprise-integration: ## run all enterprise API integration tests (47 endpoints)
	@echo "$(YELLOW)🏢 Running Enterprise API Integration Tests...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh all

test-enterprise-rbac: ## test RBAC endpoints (12 endpoints)
	@echo "$(YELLOW)🔐 Testing RBAC endpoints...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh rbac

test-enterprise-audit: ## test Audit endpoints (8 endpoints)
	@echo "$(YELLOW)📋 Testing Audit endpoints...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh audit

test-enterprise-analytics: ## test Analytics endpoints (10 endpoints)
	@echo "$(YELLOW)📊 Testing Analytics endpoints...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh analytics

test-enterprise-security: ## test Security endpoints (8 endpoints)
	@echo "$(YELLOW)🔒 Testing Security endpoints...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh security

test-enterprise-alerts: ## test Alerts endpoints (5 endpoints)
	@echo "$(YELLOW)🚨 Testing Alerts endpoints...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh alerts

test-enterprise-license: ## test License endpoints (4 endpoints)
	@echo "$(YELLOW)📜 Testing License endpoints...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh license

test-enterprise-performance: ## test enterprise API performance (<500ms target)
	@echo "$(YELLOW)⚡ Testing Enterprise API performance...$(RESET)"
	@./tmp/scripts/test-webui-integration.sh performance

test-enterprise-all: test-enterprise-integration test-enterprise-performance ## run all enterprise tests including performance
	@echo "$(GREEN)✅ All enterprise API tests completed!$(RESET)"

# ==============================================================================
# Test Runner Targets
# ==============================================================================

test-runner: ## run tests with test runner script
	@echo "Running tests with test runner script..."
	@./scripts/run-tests.sh

test-runner-unit: ## run unit tests with test runner (verbose)
	@echo "Running unit tests only..."
	@./scripts/run-tests.sh -t unit -v

test-runner-coverage: ## run tests with coverage using test runner
	@echo "Running tests with coverage..."
	@./scripts/run-tests.sh -c

# ==============================================================================
# Benchmark Targets
# ==============================================================================

.PHONY: test-benchmark test-benchmark-all test-benchmark-proxy test-benchmark-cache
.PHONY: test-benchmark-middleware test-benchmark-config test-benchmark-report
.PHONY: test-benchmark-enterprise test-benchmark-enterprise-quick test-benchmark-enterprise-stress

test-benchmark: ## run basic benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./tests/benchmark/...

test-benchmark-all: ## run all benchmarks across codebase
	@echo "Running all benchmarks..."
	go test -bench=. -benchmem ./internal/services/...
	go test -bench=. -benchmem ./tests/benchmark/...
	go test -bench=. -benchmem ./handlers/proxy/...
	go test -bench=. -benchmem ./middlewares/...

test-benchmark-proxy: ## run proxy-specific benchmarks
	@echo "Running proxy benchmarks..."
	go test -bench=BenchmarkProxy -benchmem ./tests/benchmark/...

test-benchmark-cache: ## run cache-specific benchmarks
	@echo "Running cache benchmarks..."
	go test -bench=BenchmarkCache -benchmem ./tests/benchmark/...

test-benchmark-middleware: ## run middleware benchmarks
	@echo "Running middleware benchmarks..."
	go test -bench=BenchmarkMiddleware -benchmem ./tests/benchmark/...

test-benchmark-config: ## run configuration benchmarks
	@echo "Running config benchmarks..."
	go test -bench=BenchmarkConfig -benchmem ./tests/benchmark/...

test-benchmark-report: ## generate detailed benchmark report
	@echo "Generating benchmark report..."
	@mkdir -p ./reports/benchmarks
	go test -bench=. -benchmem -benchtime=10s ./tests/benchmark/... > ./reports/benchmarks/benchmark_$(shell date +%Y%m%d_%H%M%S).txt
	@echo "Benchmark report saved to ./reports/benchmarks/"

test-benchmark-enterprise: ## run enterprise API performance benchmarks (<500ms p95 target)
	@echo "Running Enterprise API benchmarks..."
	@./tmp/scripts/benchmark-enterprise-api.sh

test-benchmark-enterprise-quick: ## quick enterprise API benchmark (25 iterations)
	@echo "Running quick Enterprise API benchmarks..."
	@ITERATIONS=25 ./tmp/scripts/benchmark-enterprise-api.sh

test-benchmark-enterprise-stress: ## stress test enterprise API (500 iterations)
	@echo "Running Enterprise API stress test..."
	@ITERATIONS=500 CONCURRENT=50 ./tmp/scripts/benchmark-enterprise-api.sh

# ==============================================================================
# Validation and Quality Checks
# ==============================================================================

.PHONY: validate validate-config validate-modules validate-gitignore
.PHONY: check check-all ci

validate: validate-config validate-modules validate-gitignore ## run all validations
	@echo "✅ Validation complete!"

validate-config: ## validate configuration files
	@echo "Validating configuration files..."
	@for file in sample-conf/*.yaml; do \
		echo "Checking $$file..."; \
		yamllint $$file || true; \
	done

validate-modules: ## validate go modules
	@echo "Validating Go modules..."
	go mod verify

validate-gitignore: ## validate gitignore patterns
	@echo "Validating .gitignore patterns..."
	@./scripts/validate-gitignore.sh

# ==============================================================================
# Quality Gates & CI Pipeline
# ==============================================================================

.PHONY: check check-all check-quick check-full ci ci-quick ci-full

# Quick quality checks (fail fast)
check-quick: lint test-unit ## quick quality check (lint + unit tests)
	@echo "🚀 Quick checks passed!"

# Standard quality checks
check: check-quick test-contract ## standard quality check (lint + unit + contract tests)
	@echo "✅ Standard checks passed!"

# Comprehensive quality checks
check-all: lint test-all test-coverage-all ## comprehensive quality check
	@echo "✅ All checks passed!"

# Full validation including E2E
check-full: lint test-full test-coverage-all ## complete validation including E2E
	@echo "✅ Full validation passed!"

# CI Pipeline Stages
ci-quick: deps lint test-unit ## CI quick feedback (Stage 1)
	@echo "🚀 CI quick checks passed!"

ci: deps lint test-unit test-contract test-integration ## CI standard pipeline (Stage 2)
	@echo "✅ CI standard checks passed!"

ci-full: deps lint test-full test-coverage-all ## CI complete pipeline (Stage 3)
	@echo "✅ CI full pipeline passed!"

# Legacy CI target (backward compatibility)
ci-legacy: deps lint test-coverage ## legacy CI pipeline checks
	@echo "CI legacy checks passed!"

# ==============================================================================
# Test Environment Setup
# ==============================================================================

.PHONY: test-setup test-teardown test-clean test-deps

test-setup: ## setup test environment
	@echo "🔧 Setting up test environment..."
	@mkdir -p ./tmp/test-cache ./tmp/test-storage ./tmp/test-config
	@mkdir -p ./reports/coverage ./reports/benchmarks ./reports/test-results
	@mkdir -p ./tests/fixtures/npm ./tests/fixtures/pip ./tests/fixtures/apt
	@mkdir -p ./tests/fixtures/docker ./tests/fixtures/maven ./tests/fixtures/yum ./tests/fixtures/apk

test-teardown: ## teardown test environment
	@echo "🧹 Tearing down test environment..."
	@docker-compose -f docker-compose.e2e.yml down -v 2>/dev/null || true
	@rm -rf ./tmp/test-* 2>/dev/null || true

test-clean: test-teardown ## clean test artifacts
	@echo "🧹 Cleaning test artifacts..."
	@rm -rf ./reports/coverage/* ./reports/benchmarks/* ./reports/test-results/* 2>/dev/null || true
	@go clean -testcache

test-deps: ## install test dependencies
	@echo "📦 Installing test dependencies..."
	@go install go.uber.org/mock/mockgen@latest
	@go mod download
	@go mod tidy

# ==============================================================================
# Test Data and Cleanup
# ==============================================================================

.PHONY: clean-test
