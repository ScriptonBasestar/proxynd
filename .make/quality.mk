# Makefile.quality.mk - Code Quality and Linting
# Based on external lint.mk but adapted for ProxyND project

# ==============================================================================
# Code Quality Configuration
# ==============================================================================

.PHONY: fmt lint format security security-code security-deps analyze analyze-complexity analyze-unused
.PHONY: quality quality-fix lint-fix lint-new lint-ci format-quick format-strict format-list format-diff
.PHONY: format-file format-env install-golangci-lint install-format-tools
.PHONY: lint-summary lint-status lint-json lint-quick lint-dev lint-precommit lint-file
.PHONY: vet install-vet-tools


# ==============================================================================
# Code Formatting
# ==============================================================================

format: format-quick ## quick and simple formatting (default)
fmt: format-quick

format-quick: ## quick basic formatting with gofumpt and goimports
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
		echo -e "$(CYAN)Run 'make format-quick' or 'make format-strict' to fix$(RESET)"; \
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

format-check: ## verify formatting without modifying files
	@echo -e "$(CYAN)🔎 Checking formatting...$(RESET)"
	@FILES=$$(gofumpt -l .); \
	if [ -n "$$FILES" ]; then \
		echo -e "$(RED)❌ Unformatted files found:$(RESET)"; \
		echo "$$FILES" | while read file; do echo "  $(YELLOW)$$file$(RESET)"; done; \
		echo -e "$(YELLOW)Run 'make format-quick' to fix$(RESET)"; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ Formatting OK$(RESET)"; \
	fi

fmt-diff: ## format only changed files (fast, for pre-commit)
	@echo -e "$(CYAN)🚀 Quick format changed files only...$(RESET)"
	@CHANGED_FILES=$$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$$' || true); \
	if [ -n "$$CHANGED_FILES" ]; then \
		echo "Formatting changed Go files:"; \
		echo "$$CHANGED_FILES" | while read file; do \
			if [ -f "$$file" ]; then \
				echo "  📝 $$file"; \
				gofumpt -w "$$file" || true; \
				goimports -w -local proxynd "$$file" || true; \
			fi; \
		done; \
		echo -e "$(GREEN)✅ Changed files formatted!$(RESET)"; \
	else \
		echo -e "$(YELLOW)📋 No changed Go files to format$(RESET)"; \
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

# Format files from environment variable
format-env: ## format files specified in FILES environment variable (usage: FILES="file1.go file2.go" make format-env)
	@if [ -z "$(FILES)" ]; then \
		echo "$(RED)❌ Error: FILES environment variable must be set$(RESET)"; \
		echo "$(YELLOW)Usage: FILES=\"file1.go file2.go\" make format-env$(RESET)"; \
		exit 1; \
	fi
	@echo "$(CYAN)🔄 Processing files from FILES variable...$(RESET)"
	@for file in $(FILES); do \
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

# ==============================================================================
# Non-Go Language Formatting (Prettier, Black, shfmt)
# ==============================================================================

.PHONY: format-js format-py format-md format-yaml format-sh format-non-go format-all

format-js:
	@which prettier > /dev/null || { echo "$(YELLOW)⚠️ prettier not found. Install: npm i -g prettier$(RESET)"; exit 0; }
	@echo -e "$(CYAN)🧹 Formatting JS/TS/JSON/MD/YAML with Prettier...$(RESET)"
	@prettier -w "**/*.{js,jsx,ts,tsx,json,md,yaml,yml}"

format-md: format-js

format-yaml: format-js

format-py:
	@which black > /dev/null || { echo "$(YELLOW)⚠️ black not found. Install: pip install black$(RESET)"; exit 0; }
	@echo -e "$(CYAN)🐍 Formatting Python with black...$(RESET)"
	@black .

format-sh:
	@which shfmt > /dev/null || { echo "$(YELLOW)⚠️ shfmt not found. Install: go install mvdan.cc/sh/v3/cmd/shfmt@latest$(RESET)"; exit 0; }
	@echo -e "$(CYAN)🐚 Formatting shell scripts with shfmt...$(RESET)"
	@shfmt -w .

format-non-go: format-js format-py format-sh

format-all: format-quick format-non-go

# ==============================================================================
# Enhanced Linting Workflow (vet + golangci-lint + gosec)
# ==============================================================================

# Install tools
lint-install-tools: ## install all linting tools (golangci-lint + gosec)
	@echo -e "$(CYAN)Installing linting tools...$(RESET)"
	@echo "1. Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	@echo "2. Installing gosec..."
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo -e "$(GREEN)✅ All linting tools installed!$(RESET)"

# Main lint workflows
lint-quick: lint-vet lint-golangci ## quick lint (vet + golangci-lint)
lint-strict: lint-vet lint-golangci lint-sec ## strict lint (vet + golangci-lint + gosec)

lint-strict-fast: lint-vet lint-golangci ## fast strict lint (vet + golangci-lint, no gosec)

lint: lint-quick ## default to strict linting

# Individual lint components
lint-vet: ## run go vet static analysis
	@echo -e "$(BLUE)🔍 Running go vet (static analysis)...$(RESET)"
	@go vet ./...
	@echo -e "$(GREEN)✅ go vet completed$(RESET)"
	@echo ""

lint-golangci: lint-install-tools ## run golangci-lint comprehensive checks
	@echo -e "$(BLUE)🔍 Running golangci-lint (comprehensive)...$(RESET)"
	@golangci-lint run ./...
	@echo -e "$(GREEN)✅ golangci-lint completed$(RESET)"
	@echo ""

lint-sec: lint-install-tools ## run gosec security analysis (optimized)
	@echo -e "$(BLUE)🔍 Running gosec (security analysis)...$(RESET)"
	@echo -e "$(CYAN)⚡ Using optimized settings (excluding false positives)$(RESET)"
	@start_time=$$(date +%s); \
	gosec -fmt=json -out=gosec-report.json \
		-exclude=G115,G404,G601,G107,G204 \
		-severity=medium -confidence=medium \
		-concurrency=4 -tests=false \
		-exclude-dir=vendor,scripts,docs,examples,tmp \
		./... || true; \
	end_time=$$(date +%s); \
	duration=$$((end_time - start_time)); \
	echo -e "$(GREEN)✅ gosec completed in $${duration}s$(RESET)"
	@echo -e "$(CYAN)📄 Security report: gosec-report.json$(RESET)"
	@if command -v jq >/dev/null 2>&1 && [ -f "gosec-report.json" ]; then \
		ISSUES=$$(jq '.Issues | length' gosec-report.json 2>/dev/null || echo '0'); \
		HIGH_ISSUES=$$(jq '[.Issues[] | select(.severity == "HIGH")] | length' gosec-report.json 2>/dev/null || echo '0'); \
		if [ "$$ISSUES" = "0" ]; then \
			echo -e "    $(GREEN)✅ No security issues found$(RESET)"; \
		else \
			echo -e "    $(YELLOW)⚠️  $$ISSUES total security issues found ($$HIGH_ISSUES high severity)$(RESET)"; \
			if [ "$$HIGH_ISSUES" -gt "0" ]; then \
				echo -e "    $(RED)🚨 High severity issues require attention$(RESET)"; \
			fi; \
		fi; \
	fi
	@echo ""

lint-sec-quick: lint-install-tools ## quick gosec security scan (for pre-push)
	@echo -e "$(BLUE)🚀 Running quick gosec scan...$(RESET)"
	@gosec -fmt=text -quiet -terse \
		-exclude=G115,G404,G601,G107,G204 \
		-severity=high -confidence=high \
		-concurrency=4 -tests=false \
		./... | head -20 || true
	@echo -e "$(GREEN)✅ Quick security scan completed$(RESET)"

lint-sec-diff: lint-install-tools ## run gosec on changed files only
	@echo -e "$(BLUE)🔍 Running gosec on changed files...$(RESET)"
	@CHANGED_FILES=$$(git diff --name-only HEAD~1 '*.go' | grep -v '_test.go' | tr '\n' ' '); \
	if [ -n "$$CHANGED_FILES" ]; then \
		echo -e "$(CYAN)📝 Changed files: $$CHANGED_FILES$(RESET)"; \
		gosec -fmt=text -quiet \
			-exclude=G115,G404,G601,G107,G204 \
			-severity=medium -confidence=medium \
			$$CHANGED_FILES || true; \
	else \
		echo -e "$(GREEN)✅ No Go files changed$(RESET)"; \
	fi
	@echo ""

lint-golangci-fix: lint-install-tools ## golangci-lint with auto-fix (for pre-commit)
	@echo -e "$(BLUE)🔧 Running golangci-lint with auto-fix...$(RESET)"
	@golangci-lint run --fix --new-from-rev=HEAD~ ./...

# CI/CD targets
lint-ci: lint-vet lint-golangci-ci lint-sec ## CI-optimized linting (all tools with CI format)
lint-golangci-ci: lint-install-tools ## golangci-lint for CI with GitHub Actions format
	@echo -e "$(BLUE)🔍 Running golangci-lint for CI...$(RESET)"
	@golangci-lint run --out-format=github-actions ./...

# File-specific linting
lint-file: ## lint specific files (usage: make lint-file file1.go file2.go ...)
	@if [ -z "$(filter-out lint-file,$(MAKECMDGOALS))" ]; then \
		echo "$(RED)❌ Error: At least one file must be specified$(RESET)"; \
		echo "$(YELLOW)Usage: make lint-file file1.go file2.go ...$(RESET)"; \
		exit 1; \
	fi
	@echo "$(CYAN)🔄 Linting specific files...$(RESET)"
	@for file in $(filter-out lint-file,$(MAKECMDGOALS)); do \
		if [ -n "$$file" ]; then \
			if [ ! -f "$$file" ]; then \
				echo "$(RED)❌ Error: File '$$file' does not exist$(RESET)"; \
				continue; \
			fi; \
			if ! echo "$$file" | grep -q "\.go$$"; then \
				echo "$(YELLOW)⚠️  Warning: File '$$file' is not a Go file, skipping$(RESET)"; \
				continue; \
			fi; \
			echo "$(CYAN)📝 Linting file: $$file$(RESET)"; \
			echo "  1. Running go vet..."; \
			go vet "$$file" || echo "$(RED)❌ go vet failed for $$file$(RESET)"; \
			echo "  2. Running golangci-lint..."; \
			golangci-lint run "$$file" || echo "$(RED)❌ golangci-lint failed for $$file$(RESET)"; \
		fi; \
	done
	@echo "$(GREEN)🎉 File linting complete!$(RESET)"

lint-summary: install-golangci-lint ## show lint issues summary by linter and problematic files
	@echo -e "$(CYAN)📊 Lint Issues Summary$(RESET)"
	@echo -e "$(CYAN)=====================$(RESET)"
	@echo ""
	@TOTAL=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | wc -l); \
	echo -e "$(YELLOW)Total Issues: $$TOTAL$(RESET)"; \
	echo ""
	@echo -e "$(GREEN)🏷️  Issues by Linter:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/.*(\\([^)]*\\))$$/\\1/' | sort | uniq -c | sort -nr | \
	awk '{printf "  $(CYAN)%-15s$(RESET) %d issues\\n", $$2, $$1}'
	@echo ""
	@echo -e "$(GREEN)📁 Issues by File:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/^\\([^:]*\\):.*/\\1/' | sort | uniq -c | sort -nr | head -10 | \
	awk '{printf "  $(MAGENTA)%-40s$(RESET) %d issues\\n", $$2, $$1}'

lint-status: install-golangci-lint ## comprehensive lint status report with detailed analysis
	@echo -e "$(BLUE)🔍 Comprehensive Lint Status Report$(RESET)"
	@echo -e "$(BLUE)==================================$(RESET)"
	@echo ""
	@echo -e "$(GREEN)📊 Overview:$(RESET)"
	@TOTAL=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | wc -l); \
	FILES=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/^\\([^:]*\\):.*/\\1/' | sort | uniq | wc -l); \
	LINTERS=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/.*(\\([^)]*\\))$$/\\1/' | sort | uniq | wc -l); \
	echo -e "  $(YELLOW)Total Issues: $$TOTAL$(RESET)"; \
	echo -e "  $(YELLOW)Affected Files: $$FILES$(RESET)"; \
	echo -e "  $(YELLOW)Active Linters: $$LINTERS$(RESET)"; \
	echo ""
	@echo -e "$(GREEN)🏆 Top 5 Issue Types:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/.*(\\([^)]*\\))$$/\\1/' | sort | uniq -c | sort -nr | head -5 | \
	awk '{printf "  $(RED)%-20s$(RESET) %d issues\\n", $$2, $$1}'
	@echo ""
	@echo -e "$(GREEN)📂 Top 5 Problematic Files:$(RESET)"
	@golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | \
	grep -E "^[^[:space:]].*\\([^)]+\\)$$" | sed 's/^\\([^:]*\\):.*/\\1/' | sort | uniq -c | sort -nr | head -5 | \
	awk '{printf "  $(MAGENTA)%-50s$(RESET) %d issues\\n", $$2, $$1}'
	@echo ""
	@echo -e "$(GREEN)💡 Recommendations:$(RESET)"
	@TOTAL=$$(golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --out-format=line-number 2>/dev/null | grep -E "^[^[:space:]].*\\([^)]+\\)$$" | wc -l); \
	if [ $$TOTAL -eq 0 ]; then \
		echo -e "  $(GREEN)✅ Perfect! No lint issues found.$(RESET)"; \
	elif [ $$TOTAL -le 10 ]; then \
		echo -e "  $(YELLOW)⭐ Great! Only $$TOTAL issues remaining. Run 'make lint-fix' to auto-fix.$(RESET)"; \
	elif [ $$TOTAL -le 50 ]; then \
		echo -e "  $(ORANGE)⚠️  $$TOTAL issues found. Focus on top linters. Run 'make lint-fix' first.$(RESET)"; \
	else \
		echo -e "  $(RED)🚨 $$TOTAL issues found. Significant cleanup needed. Start with 'make lint-fix'.$(RESET)"; \
	fi

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

lint-diff: ## lint only changed files (fast, for pre-commit)
	@echo -e "$(BLUE)🚀 Quick lint changed files only...$(RESET)"
	@CHANGED_FILES=$$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$$' || true); \
	if [ -n "$$CHANGED_FILES" ]; then \
		echo "Linting changed Go files:"; \
		echo "$$CHANGED_FILES" | while read file; do \
			if [ -f "$$file" ]; then \
				echo "  🔍 $$file"; \
			fi; \
		done; \
		echo "1. Running go vet on changed files..."; \
		echo "$$CHANGED_FILES" | xargs -r go vet || true; \
		echo "2. Running golangci-lint on changed files..."; \
		echo "$$CHANGED_FILES" | xargs -r golangci-lint run --fix || true; \
		echo -e "$(GREEN)✅ Changed files linted!$(RESET)"; \
	else \
		echo -e "$(YELLOW)📋 No changed Go files to lint$(RESET)"; \
	fi

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
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
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

pre-commit: lint-diff fmt-diff ## fast pre-commit checks (changed files only, <3s)
	@echo -e "$(GREEN)✅ Pre-commit checks completed!$(RESET)"

pre-commit-make: lint-vet lint-golangci-fix ## legacy pre-commit using make commands

pre-push: lint-strict-fast format-check ## fast pre-push checks (no gosec, non-modifying)
pre-push-full: lint-strict format-check ## comprehensive pre-push checks (includes gosec, non-modifying)
	@echo -e "$(GREEN)✅ Pre-push checks completed!$(RESET)"

quality: fmt lint test-coverage ## run all quality checks
	@echo "✅ All quality checks passed!"

quality-fix: fmt lint-fix ## apply automatic quality fixes
	@echo "✅ Code quality fixes applied!"

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean-analysis
