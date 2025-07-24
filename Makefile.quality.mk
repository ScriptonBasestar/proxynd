# Makefile.quality.mk - Code Quality and Linting
# Based on external lint.mk but adapted for ProxyND project

# ==============================================================================
# Code Quality Configuration
# ==============================================================================

.PHONY: fmt lint format security security-code security-deps analyze analyze-complexity analyze-unused
.PHONY: quality quality-fix lint-fix lint-new lint-ci format-simplify format-strict format-list format-diff
.PHONY: format-file install-golangci-lint install-format-tools
.PHONY: lint-count lint-summary lint-status lint-json


# ==============================================================================
# Code Formatting
# ==============================================================================

format: format-simplify ## quick and simple formatting (default)
fmt: format-simplify

format-simplify: ## quick basic formatting with gofumpt and goimports
	@echo -e "$(CYAN)🚀 Quick formatting...$(RESET)"
	@echo "1. Running gofumpt (includes go fmt + simplification)..."
	@gofumpt -w .
	@echo "2. Organizing imports..."
	@goimports -w -local proxynd .
	@echo -e "$(GREEN)✅ Quick formatting complete!$(RESET)"

format-strict: install-format-tools ## comprehensive formatting with all tools
	@echo -e "$(CYAN)🔧 Strict formatting (all tools)...$(RESET)"
	@echo "1. Running gofumpt (strict formatting + simplification)..."
	@gofumpt -w -extra .
	@echo "2. Running gci (import organization)..."
	@gci write --skip-generated .
	@echo "3. Organizing imports..."
	@goimports -w -local proxynd .
	@echo "4. Final gci (import grouping)..."
	@gci write --skip-generated -s standard -s default -s "prefix(proxynd)" .
	@echo -e "$(GREEN)✅ Strict formatting complete!$(RESET)"

format-list: ## show files that need formatting
	@echo -e "$(CYAN)📋 Files that need formatting:$(RESET)"
	@FILES=$$(gofmt -l .); \
	if [ -n "$$FILES" ]; then \
		echo "$$FILES" | while read file; do echo "  $(YELLOW)$$file$(RESET)"; done; \
		echo ""; \
		echo -e "$(YELLOW)Total: $$(echo "$$FILES" | wc -l) files need formatting$(RESET)"; \
		echo -e "$(CYAN)Run 'make format-simplify' or 'make format-strict' to fix$(RESET)"; \
	else \
		echo -e "$(GREEN)✅ All files are properly formatted!$(RESET)"; \
	fi

format-diff: ## show formatting differences
	@echo -e "$(CYAN)📝 Formatting differences:$(RESET)"
	@DIFF_OUTPUT=$$(gofmt -d .); \
	if [ -n "$$DIFF_OUTPUT" ]; then \
		echo "$$DIFF_OUTPUT"; \
	else \
		echo -e "$(GREEN)✅ No formatting differences found!$(RESET)"; \
	fi

format-install-tools: ## install advanced formatting tools
	@echo -e "$(CYAN)Installing formatting tools...$(RESET)"
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	@which gofumpt > /dev/null || (echo "Installing gofumpt..." && go install mvdan.cc/gofumpt@latest)
	@which gci > /dev/null || (echo "Installing gci..." && go install github.com/daixiang0/gci@latest)
	@echo -e "$(GREEN)✅ All formatting tools installed!$(RESET)"

format-file: ## format specific files with gofumpt and goimports (usage: make format-file file1.go file2.go ...)
	@if [ -z "$(MAKECMDGOALS)" ] || [ "$(words $(MAKECMDGOALS))" -eq 1 ]; then \
		echo "$(RED)❌ Error: At least one file must be specified$(RESET)"; \
		echo "$(YELLOW)Usage: make format-file file1.go file2.go ...$(RESET)"; \
		exit 1; \
	fi
	@echo "$(CYAN)🔄 Processing files...$(RESET)"
	@for file in $(filter-out format-file,$(MAKECMDGOALS)); do \
		if [ -n "$$file" ]; then \
			if [ ! -f "$$file" ]; then \
				echo "$(RED)❌ Error: File '$$file' does not exist$(RESET)"; \
				continue; \
			fi; \
			if ! echo "$$file" | grep -q "\.go$$"; then \
				echo "$(YELLOW)⚠️  Warning: File '$$file' is not a Go file (.go extension), skipping$(RESET)"; \
				continue; \
			fi; \
			echo "$(CYAN)📝 Formatting file: $$file$(RESET)"; \
			echo "  1. Running gofumpt..."; \
			gofumpt -w "$$file" || echo "$(RED)❌ gofumpt failed for $$file$(RESET)"; \
			echo "  2. Running goimports..."; \
			goimports -w -local proxynd "$$file" || echo "$(RED)❌ goimports failed for $$file$(RESET)"; \
			echo "$(GREEN)✅ File '$$file' formatted successfully!$(RESET)"; \
		fi; \
	done
	@echo "$(GREEN)🎉 All files processed!$(RESET)"

# Handle additional arguments as targets (to prevent make errors)
%:
	@:

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
	@echo -e "$(CYAN)Counting lint issues...$(RESET)"
	@ISSUES=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | wc -l); \
	echo -e "$(YELLOW)Total lint issues: $$ISSUES$(RESET)"

lint-summary: install-golangci-lint ## show lint issues summary by linter
	@echo -e "$(CYAN)Lint issues summary:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/.*(\\([^)]*\\))$$/\\1/' | sort | uniq -c | sort -nr | \
	awk '{printf "  $(YELLOW)%-15s$(RESET) %d issues\\n", $$2, $$1}'

lint-status: install-golangci-lint ## comprehensive lint status report
	@echo -e "$(BLUE)🔍 Comprehensive Lint Status Report$(RESET)"
	@echo -e "$(BLUE)==================================$(RESET)"
	@echo ""
	@echo -e "$(GREEN)📊 Quick Stats:$(RESET)"
	@TOTAL=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | wc -l); \
	echo -e "  $(YELLOW)Total Issues: $$TOTAL$(RESET)"; \
	echo ""
	@echo -e "$(GREEN)🏷️  Top 10 Linters:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/.*(\\([^)]*\\))$$/\\1/' | sort | uniq -c | sort -nr | head -10 | \
	awk '{printf "  $(CYAN)%-15s$(RESET) %d issues\\n", $$2, $$1}'
	@echo ""
	@echo -e "$(GREEN)📁 Most Problematic Files:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/^\\([^:]*\\):.*/\\1/' | sort | uniq -c | sort -nr | head -5 | \
	awk '{printf "  $(MAGENTA)%-40s$(RESET) %d issues\\n", $$2, $$1}'

lint-json: install-golangci-lint ## export lint results to JSON for further analysis
	@echo -e "$(CYAN)Exporting lint results to lint-report.json...$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=json > lint-report.json 2>/dev/null || true
	@echo -e "$(GREEN)✅ Report saved to lint-report.json$(RESET)"
	@if command -v jq >/dev/null 2>&1; then \
		echo ""; \
		echo -e "$(YELLOW)📈 JSON Report Summary:$(RESET)"; \
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
	@echo -e "$(CYAN)Checking dependencies for vulnerabilities...$(RESET)"
	@which nancy > /dev/null || go install github.com/sonatype-nexus-community/nancy@latest
	go list -json -deps ./... | nancy sleuth

security-code: ## run security code analysis
	@echo -e "$(CYAN)Running security code analysis...$(RESET)"
	@which gosec > /dev/null || go install github.com/securecode/gosec/v2/cmd/gosec@latest
	gosec -fmt=json -out=gosec-report.json ./... || true
	@echo -e "$(GREEN)Security report generated: gosec-report.json$(RESET)"

# ==============================================================================
# Code Analysis
# ==============================================================================

analyze: analyze-complexity analyze-unused ## run code analysis
	@echo "✅ Code analysis complete!"

analyze-complexity: ## analyze code complexity
	@echo -e "$(CYAN)Analyzing code complexity...$(RESET)"
	@which gocyclo > /dev/null || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 10 .

analyze-unused: ## find unused code
	@echo -e "$(CYAN)Finding unused code...$(RESET)"
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
