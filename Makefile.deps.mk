# Makefile.deps.mk - Local Dependency Management
# Alternative to Dependabot for controlled local updates

# ==============================================================================
# Dependency Management Configuration
# ==============================================================================

.PHONY: deps-check deps-update deps-upgrade deps-update-go deps-update-actions deps-update-docker
.PHONY: deps-outdated deps-security deps-audit deps-report deps-clean deps-help
.PHONY: deps-update-minor deps-update-patch deps-update-major deps-interactive

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

# ==============================================================================
# Go Dependencies
# ==============================================================================

deps-check: ## check for outdated Go dependencies
	@echo "$(CYAN)Checking for outdated Go dependencies...$(RESET)"
	@go list -u -m all | grep '\[' || echo "$(GREEN)✓ All Go dependencies are up to date$(RESET)"

deps-outdated: ## detailed report of outdated dependencies
	@echo "$(CYAN)Generating detailed outdated dependencies report...$(RESET)"
	@echo "$(YELLOW)Go Modules:$(RESET)"
	@go list -u -m all | grep '\[' | while read line; do \
		echo "  $(RED)→$(RESET) $$line"; \
	done || echo "  $(GREEN)✓ All Go modules are up to date$(RESET)"
	@echo ""
	@echo "$(YELLOW)Direct Dependencies:$(RESET)"
	@go list -u -m all | grep '\[' | grep -v 'indirect' | while read line; do \
		echo "  $(RED)→$(RESET) $$line"; \
	done || echo "  $(GREEN)✓ All direct dependencies are up to date$(RESET)"

deps-update: ## update all dependencies (safe: patch + minor only)
	@echo "$(CYAN)Updating dependencies safely (patch + minor versions)...$(RESET)"
	@echo "$(YELLOW)Before update:$(RESET)"
	@go mod tidy
	@cp go.mod go.mod.backup
	@cp go.sum go.sum.backup
	@echo "$(CYAN)Updating Go dependencies...$(RESET)"
	@go get -u=patch ./...
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated safely$(RESET)"
	@echo "$(YELLOW)Changes:$(RESET)"
	@diff go.mod.backup go.mod || echo "  No changes in go.mod"
	@rm go.mod.backup go.sum.backup

deps-update-minor: ## update to latest minor versions (more aggressive)
	@echo "$(CYAN)Updating to latest minor versions...$(RESET)"
	@cp go.mod go.mod.backup
	@cp go.sum go.sum.backup
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated to latest minor versions$(RESET)"
	@echo "$(YELLOW)Changes:$(RESET)"
	@diff go.mod.backup go.mod || echo "  No changes in go.mod"
	@rm go.mod.backup go.sum.backup

deps-update-patch: ## update to latest patch versions only (safest)
	@echo "$(CYAN)Updating to latest patch versions only...$(RESET)"
	@cp go.mod go.mod.backup
	@cp go.sum go.sum.backup
	@go get -u=patch ./...
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated to latest patch versions$(RESET)"
	@echo "$(YELLOW)Changes:$(RESET)"
	@diff go.mod.backup go.mod || echo "  No changes in go.mod"
	@rm go.mod.backup go.sum.backup

deps-update-major: ## update to latest major versions (use with caution!)
	@echo "$(RED)WARNING: This will update to latest major versions!$(RESET)"
	@echo "$(YELLOW)This may introduce breaking changes. Continue? [y/N]$(RESET)"
	@read -r confirm && [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ] || exit 1
	@cp go.mod go.mod.backup
	@cp go.sum go.sum.backup
	@go list -u -m all | grep '\[' | cut -d' ' -f1 | xargs -I {} go get {}@latest
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated to latest major versions$(RESET)"
	@echo "$(YELLOW)Changes:$(RESET)"
	@diff go.mod.backup go.mod || echo "  No changes in go.mod"
	@rm go.mod.backup go.sum.backup

deps-interactive: ## interactive dependency update (choose which ones to update)
	@echo "$(CYAN)Interactive dependency update...$(RESET)"
	@outdated=$$(go list -u -m all | grep '\['); \
	if [ -z "$$outdated" ]; then \
		echo "$(GREEN)✓ All dependencies are up to date$(RESET)"; \
		exit 0; \
	fi; \
	echo "$$outdated" | while read line; do \
		pkg=$$(echo $$line | cut -d' ' -f1); \
		current=$$(echo $$line | cut -d' ' -f2); \
		latest=$$(echo $$line | sed 's/.*\[\(.*\)\].*/\1/'); \
		echo "$(YELLOW)Update $$pkg from $$current to $$latest? [y/N]$(RESET)"; \
		read -r confirm; \
		if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
			echo "$(CYAN)Updating $$pkg...$(RESET)"; \
			go get $$pkg@$$latest; \
		fi; \
	done; \
	go mod tidy

# ==============================================================================
# GitHub Actions Dependencies
# ==============================================================================

deps-update-actions: ## check and show GitHub Actions that need updates
	@echo "$(CYAN)Checking GitHub Actions dependencies...$(RESET)"
	@if [ -d ".github/workflows" ]; then \
		echo "$(YELLOW)GitHub Actions in use:$(RESET)"; \
		grep -r "uses:" .github/workflows/ | sed 's/.*uses: */  → /' | sort | uniq; \
		echo ""; \
		echo "$(YELLOW)To update GitHub Actions, manually edit .github/workflows/*.yml files$(RESET)"; \
		echo "$(YELLOW)Common updates:$(RESET)"; \
		echo "  → actions/checkout@v4"; \
		echo "  → actions/setup-go@v5"; \
		echo "  → actions/cache@v4"; \
	else \
		echo "$(GREEN)✓ No GitHub Actions found$(RESET)"; \
	fi

# ==============================================================================
# Docker Dependencies
# ==============================================================================

deps-update-docker: ## check and show Docker base images that need updates
	@echo "$(CYAN)Checking Docker dependencies...$(RESET)"
	@if [ -f "Dockerfile" ]; then \
		echo "$(YELLOW)Docker base images in use:$(RESET)"; \
		grep -E "^FROM" Dockerfile | sed 's/FROM */  → /'; \
		echo ""; \
		echo "$(YELLOW)To update Docker images, manually edit Dockerfile$(RESET)"; \
		echo "$(YELLOW)Consider using specific version tags instead of 'latest'$(RESET)"; \
	else \
		echo "$(GREEN)✓ No Dockerfile found$(RESET)"; \
	fi

# ==============================================================================
# Security and Audit
# ==============================================================================

deps-security: ## run security audit on dependencies
	@echo "$(CYAN)Running security audit...$(RESET)"
	@echo "$(YELLOW)Checking for known vulnerabilities...$(RESET)"
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./... || echo "$(RED)✗ Vulnerabilities found$(RESET)"
	@echo ""
	@echo "$(YELLOW)Running go mod audit...$(RESET)"
	@go list -json -deps ./... | jq -r '.Module | select(.Path != null) | "\(.Path) \(.Version)"' | sort | uniq | \
		while read module version; do \
			if [ "$$module" != "$(shell go list -m)" ]; then \
				echo "  → $$module $$version"; \
			fi; \
		done

deps-audit: ## comprehensive dependency audit and report
	@echo "$(CYAN)Comprehensive dependency audit...$(RESET)"
	@echo "$(YELLOW)=== Go Module Information ===$(RESET)"
	@go version
	@echo "Module: $$(go list -m)"
	@echo "Go version: $$(go list -m -f '{{.GoVersion}}')"
	@echo ""
	@echo "$(YELLOW)=== Direct Dependencies ===$(RESET)"
	@go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all | grep -v "^$$" | head -20
	@echo ""
	@echo "$(YELLOW)=== Outdated Dependencies ===$(RESET)"
	@make deps-outdated
	@echo ""
	@echo "$(YELLOW)=== Security Check ===$(RESET)"
	@make deps-security

# ==============================================================================
# Dependency Reports
# ==============================================================================

deps-report: ## generate comprehensive dependency report
	@echo "$(CYAN)Generating dependency report...$(RESET)"
	@report_file="dependency-report-$$(date +%Y%m%d-%H%M%S).md"; \
	echo "# Dependency Report" > $$report_file; \
	echo "Generated: $$(date)" >> $$report_file; \
	echo "" >> $$report_file; \
	echo "## Go Module Information" >> $$report_file; \
	echo "\`\`\`" >> $$report_file; \
	go version >> $$report_file; \
	echo "Module: $$(go list -m)" >> $$report_file; \
	echo "Go version: $$(go list -m -f '{{.GoVersion}}')" >> $$report_file; \
	echo "\`\`\`" >> $$report_file; \
	echo "" >> $$report_file; \
	echo "## Direct Dependencies" >> $$report_file; \
	echo "\`\`\`" >> $$report_file; \
	go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all | grep -v "^$$" >> $$report_file; \
	echo "\`\`\`" >> $$report_file; \
	echo "" >> $$report_file; \
	echo "## Outdated Dependencies" >> $$report_file; \
	echo "\`\`\`" >> $$report_file; \
	go list -u -m all | grep '\[' >> $$report_file || echo "All dependencies are up to date" >> $$report_file; \
	echo "\`\`\`" >> $$report_file; \
	echo "$(GREEN)✓ Report generated: $$report_file$(RESET)"

# ==============================================================================
# Cleanup and Maintenance
# ==============================================================================

deps-clean: ## clean up dependency cache and temporary files
	@echo "$(CYAN)Cleaning dependency cache...$(RESET)"
	@go clean -modcache
	@go clean -cache
	@echo "$(GREEN)✓ Dependency cache cleaned$(RESET)"

deps-verify: ## verify dependency checksums
	@echo "$(CYAN)Verifying dependency checksums...$(RESET)"
	@go mod verify
	@echo "$(GREEN)✓ All dependency checksums verified$(RESET)"

deps-why: ## show why a specific module is needed (usage: make deps-why MOD=github.com/pkg/errors)
	@if [ -z "$(MOD)" ]; then \
		echo "$(RED)Usage: make deps-why MOD=github.com/pkg/errors$(RESET)"; \
		exit 1; \
	fi
	@echo "$(CYAN)Showing why $(MOD) is needed...$(RESET)"
	@go mod why -m $(MOD)

# ==============================================================================
# Dependabot Alternative Workflow
# ==============================================================================

deps-weekly: ## run weekly dependency maintenance (safe updates)
	@echo "$(CYAN)Running weekly dependency maintenance...$(RESET)"
	@echo "$(YELLOW)1. Checking current status...$(RESET)"
	@make deps-check
	@echo ""
	@echo "$(YELLOW)2. Running security audit...$(RESET)"
	@make deps-security
	@echo ""
	@echo "$(YELLOW)3. Updating patch versions (safest)...$(RESET)"
	@make deps-update-patch
	@echo ""
	@echo "$(YELLOW)4. Running tests after update...$(RESET)"
	@go test ./... -short
	@echo ""
	@echo "$(GREEN)✓ Weekly maintenance completed$(RESET)"

deps-monthly: ## run monthly dependency maintenance (minor updates)
	@echo "$(CYAN)Running monthly dependency maintenance...$(RESET)"
	@echo "$(YELLOW)1. Creating backup...$(RESET)"
	@cp go.mod go.mod.monthly-backup
	@cp go.sum go.sum.monthly-backup
	@echo ""
	@echo "$(YELLOW)2. Updating minor versions...$(RESET)"
	@make deps-update-minor
	@echo ""
	@echo "$(YELLOW)3. Running full test suite...$(RESET)"
	@go test ./...
	@echo ""
	@echo "$(YELLOW)4. Running security audit...$(RESET)"
	@make deps-security
	@echo ""
	@if [ -f "go.mod.monthly-backup" ]; then \
		echo "$(YELLOW)Backup files created:$(RESET)"; \
		echo "  → go.mod.monthly-backup"; \
		echo "  → go.sum.monthly-backup"; \
	fi
	@echo "$(GREEN)✓ Monthly maintenance completed$(RESET)"

# ==============================================================================
# Help
# ==============================================================================

deps-help: ## show help for dependency management commands
	@echo "$(CYAN)Dependency Management Commands:$(RESET)"
	@echo ""
	@echo "$(YELLOW)Daily Operations:$(RESET)"
	@echo "  make deps-check          Check for outdated dependencies"
	@echo "  make deps-outdated       Detailed outdated dependencies report"
	@echo "  make deps-update         Safe update (patch + minor only)"
	@echo "  make deps-interactive    Interactive dependency updates"
	@echo ""
	@echo "$(YELLOW)Update Levels:$(RESET)"
	@echo "  make deps-update-patch   Update patch versions only (safest)"
	@echo "  make deps-update-minor   Update minor versions (moderate risk)"
	@echo "  make deps-update-major   Update major versions (breaking changes!)"
	@echo ""
	@echo "$(YELLOW)Security & Audit:$(RESET)"
	@echo "  make deps-security       Run security vulnerability scan"
	@echo "  make deps-audit          Comprehensive dependency audit"
	@echo "  make deps-verify         Verify dependency checksums"
	@echo ""
	@echo "$(YELLOW)Other Dependencies:$(RESET)"
	@echo "  make deps-update-actions Check GitHub Actions updates"
	@echo "  make deps-update-docker  Check Docker base image updates"
	@echo ""
	@echo "$(YELLOW)Maintenance Workflows:$(RESET)"
	@echo "  make deps-weekly         Weekly maintenance (patch updates)"
	@echo "  make deps-monthly        Monthly maintenance (minor updates)"
	@echo ""
	@echo "$(YELLOW)Utilities:$(RESET)"
	@echo "  make deps-report         Generate comprehensive report"
	@echo "  make deps-clean          Clean dependency cache"
	@echo "  make deps-why MOD=...    Show why a module is needed"
	@echo ""
	@echo "$(YELLOW)Configuration:$(RESET)"
	@echo "  Dependabot config: .github/dependabot.yml"
	@echo "  Disable Dependabot and use local commands instead"
