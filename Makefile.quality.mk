# Makefile.quality.mk - Code Quality and Linting
# Based on external lint.mk but adapted for ProxyND project

# ==============================================================================
# Code Quality Configuration
# ==============================================================================

.PHONY: fmt lint format security security-code security-deps analyze analyze-complexity analyze-unused
.PHONY: quality quality-fix lint-fix lint-new lint-ci format-all format-check format-diff
.PHONY: format-imports format-simplify format-ci install-golangci-lint install-format-tools
.PHONY: lint-count lint-summary lint-status lint-json


# ==============================================================================
# Code Formatting
# ==============================================================================

fmt: ## format go files with gofmt and goimports
	@echo "Formatting code..."
	go fmt ./...
	@echo "Organizing imports..."
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w -local proxynd .
	@echo "Code formatting complete!"

format: fmt format-check ## format and check code formatting
	@echo "✅ All formatting complete!"

format-check: ## check code formatting without fixing
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ The following files need formatting:"; \
		gofmt -l .; \
		echo "Run 'make format' to fix."; \
		exit 1; \
	else \
		echo "✅ All files are properly formatted"; \
	fi

format-diff: ## show formatting differences
	@echo "Showing formatting differences..."
	@gofmt -d .

format-imports: ## organize imports only
	@echo "Organizing imports..."
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	@goimports -w -local proxynd .
	@echo "✅ Imports organized!"

format-simplify: ## simplify code with gofmt -s
	@echo "Simplifying code..."
	@which gofmt > /dev/null && gofmt -s -w .
	@echo "✅ Code simplified!"

format-all: install-format-tools ## run all formatters including advanced ones
	@echo "$(CYAN)Running all formatters...$(RESET)"
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
	@echo "$(GREEN)✅ All formatting complete!$(RESET)"

install-format-tools: ## install advanced formatting tools
	@echo "Installing formatting tools..."
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	@which gofumpt > /dev/null || (echo "Installing gofumpt..." && go install mvdan.cc/gofumpt@latest)
	@which gci > /dev/null || (echo "Installing gci..." && go install github.com/daixiang0/gci@latest)
	@echo "✅ All formatting tools installed!"

format-ci: format-check ## CI-friendly format check
	@echo "CI format check passed!"

# ==============================================================================
# Linting
# ==============================================================================

install-golangci-lint: ## install golangci-lint
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
	@echo "golangci-lint installed!"

lint: install-golangci-lint ## run golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run ./...

lint-count: install-golangci-lint ## count total lint issues without fixing
	@echo "$(CYAN)Counting lint issues...$(RESET)"
	@ISSUES=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | wc -l); \
	echo "$(YELLOW)Total lint issues: $$ISSUES$(RESET)"

lint-summary: install-golangci-lint ## show lint issues summary by linter
	@echo "$(CYAN)Lint issues summary:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/.*(\\([^)]*\\))$$/\\1/' | sort | uniq -c | sort -nr | \
	awk '{printf "  $(YELLOW)%-15s$(RESET) %d issues\\n", $$2, $$1}'

lint-status: install-golangci-lint ## comprehensive lint status report
	@echo "$(BLUE)🔍 Comprehensive Lint Status Report$(RESET)"
	@echo "$(BLUE)==================================$(RESET)"
	@echo ""
	@echo "$(GREEN)📊 Quick Stats:$(RESET)"
	@TOTAL=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | wc -l); \
	echo "  $(YELLOW)Total Issues: $$TOTAL$(RESET)"; \
	echo ""
	@echo "$(GREEN)🏷️  Top 10 Linters:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/.*(\\([^)]*\\))$$/\\1/' | sort | uniq -c | sort -nr | head -10 | \
	awk '{printf "  $(CYAN)%-15s$(RESET) %d issues\\n", $$2, $$1}'
	@echo ""
	@echo "$(GREEN)📁 Most Problematic Files:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/^\\([^:]*\\):.*/\\1/' | sort | uniq -c | sort -nr | head -5 | \
	awk '{printf "  $(MAGENTA)%-40s$(RESET) %d issues\\n", $$2, $$1}'

lint-json: install-golangci-lint ## export lint results to JSON for further analysis
	@echo "$(CYAN)Exporting lint results to lint-report.json...$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=json > lint-report.json 2>/dev/null || true
	@echo "$(GREEN)✅ Report saved to lint-report.json$(RESET)"
	@if command -v jq >/dev/null 2>&1; then \
		echo ""; \
		echo "$(YELLOW)📈 JSON Report Summary:$(RESET)"; \
		echo "  Total Issues: $$(jq '.Issues | length' lint-report.json 2>/dev/null || echo '0')"; \
		echo "  Unique Files: $$(jq -r '.Issues[]? | .Pos.Filename' lint-report.json 2>/dev/null | sort | uniq | wc -l || echo '0')"; \
	fi

lint-fix: install-golangci-lint ## run golangci-lint with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix ./...

lint-new: install-golangci-lint ## run golangci-lint on new code only
	@echo "Running golangci-lint on new code only..."
	golangci-lint run --new-from-rev=HEAD~ ./...

lint-ci: ## run golangci-lint for CI
	@echo "Running golangci-lint for CI..."
	golangci-lint run --out-format=github-actions ./...

# ==============================================================================
# Security Analysis
# ==============================================================================

security: security-deps security-code ## run all security checks
	@echo "✅ Security checks completed!"

security-deps: ## check dependencies for vulnerabilities
	@echo "$(CYAN)Checking dependencies for vulnerabilities...$(RESET)"
	@which nancy > /dev/null || go install github.com/sonatype-nexus-community/nancy@latest
	go list -json -deps ./... | nancy sleuth

security-code: ## run security code analysis
	@echo "$(CYAN)Running security code analysis...$(RESET)"
	@which gosec > /dev/null || go install github.com/securecode/gosec/v2/cmd/gosec@latest
	gosec -fmt=json -out=gosec-report.json ./... || true
	@echo "$(GREEN)Security report generated: gosec-report.json$(RESET)"

# ==============================================================================
# Code Analysis
# ==============================================================================

analyze: analyze-complexity analyze-unused ## run code analysis
	@echo "✅ Code analysis complete!"

analyze-complexity: ## analyze code complexity
	@echo "$(CYAN)Analyzing code complexity...$(RESET)"
	@which gocyclo > /dev/null || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 10 .

analyze-unused: ## find unused code
	@echo "$(CYAN)Finding unused code...$(RESET)"
	@which unused > /dev/null || go install honnef.co/go/tools/cmd/unused@latest
	unused ./...

# ==============================================================================
# Quality Assurance Workflow Targets
# ==============================================================================

quality: fmt lint test-coverage ## run all quality checks
	@echo "✅ All quality checks passed!"

quality-fix: fmt lint-fix ## apply automatic quality fixes
	@echo "✅ Code quality fixes applied!"

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean-analysis
