# Makefile.test.mk - Testing and Quality Assurance
# All testing-related targets including unit, integration, benchmarks, and coverage

# ==============================================================================
# Basic Test Targets
# ==============================================================================

.PHONY: test-unit test-race test-services test-coverage test-integration test-integration-bench
.PHONY: test-all test-runner test-runner-unit test-runner-coverage

test-unit: ## run unit tests only
	@echo "Running unit tests..."
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

test-integration: ## run integration tests
	@echo "Running integration tests..."
	./scripts/run_integration_tests.sh

test-integration-bench: ## run integration tests with benchmarks
	@echo "Running integration tests with benchmarks..."
	./scripts/run_integration_tests.sh --bench

test-all: test-unit test-race test-services ## run all tests
	@echo "All tests completed!"

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

check: lint test-unit ## quick quality check (lint + unit tests)
	@echo "✅ Quick checks passed!"

check-all: lint test-all test-coverage ## comprehensive quality check
	@echo "✅ All checks passed!"

ci: deps lint test-coverage ## CI pipeline checks
	@echo "CI checks passed!"

# ==============================================================================
# Test Data and Cleanup
# ==============================================================================

.PHONY: clean-test

# clean-test moved to Makefile.clean.mk
