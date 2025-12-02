# Makefile.clean.mk - Cleanup and Maintenance
# All cleanup and maintenance operations

# ==============================================================================
# Build Cleanup
# ==============================================================================

.PHONY: clean-build clean-test clean-logs clean-cache clean-analysis clean-dev
.PHONY: clean-deep clean-all clean clean-script

clean-build: ## clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf tmp/bin
	@rm -f dist/*
	@find . -name "*.exe" -o -name "*.out" -type f -delete 2>/dev/null || true
	@find . -name "*.test" -type f -delete 2>/dev/null || true
	@echo "✓ Build artifacts cleaned!"

# ==============================================================================
# Test Cleanup
# ==============================================================================

clean-test: ## clean test files and artifacts
	@echo "📊 Cleaning test files..."
	@rm -f coverage.out coverage.html coverage.txt
	@rm -f *.prof *.trace *.pprof
	@find . -name "*.test" -type f -delete 2>/dev/null || true
	@echo "✓ Test files cleaned!"

# ==============================================================================
# Log Cleanup
# ==============================================================================

clean-logs: ## clean log files
	@echo "📝 Cleaning log files..."
	@mkdir -p logs tests/integration/logs
	@find logs -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
	@find tests -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
	@find integration -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
	@echo "✓ Log files cleaned!"

# ==============================================================================
# Cache Cleanup
# ==============================================================================

clean-cache: ## clean go build and test caches
	@echo "🗄️ Cleaning caches..."
	@go clean -cache -testcache 2>/dev/null || true
	@rm -rf .cache 2>/dev/null || true
	@echo "✓ Caches cleaned!"

# ==============================================================================
# Analysis Cleanup
# ==============================================================================

clean-analysis: ## clean analysis reports and generated files
	@echo "🔍 Cleaning analysis reports..."
	@rm -f gosec-report.json staticcheck-report.json golangci-lint-report.json
	@rm -f deps-graph.png security-report.json
	@rm -f lint-report.json dependency-report-*.md
	@echo "✓ Analysis reports cleaned!"

# ==============================================================================
# Development Cleanup
# ==============================================================================

clean-dev: ## clean development files and temporary data
	@echo "🛠️ Cleaning development files..."
	@rm -rf ./tmp/storage/* 2>/dev/null || true
	@rm -rf ./.env.local 2>/dev/null || true
	@find . -name "*.tmp" -o -name "*.temp" -type f -delete 2>/dev/null || true
	@echo "✓ Development files cleaned!"

# ==============================================================================
# Mock Cleanup (from tools)
# ==============================================================================

clean-mocks: ## clean generated mock files
	@echo "🎭 Cleaning generated mocks..."
	@find . -type d -name "mocks" -exec rm -rf {} + 2>/dev/null || true
	@echo "✓ Mocks cleaned!"

# ==============================================================================
# Deep Cleanup
# ==============================================================================

clean-deep: clean-build clean-test clean-logs clean-analysis clean-dev ## deep clean including module cache
	@echo "🗑️ Deep cleaning..."
	@go clean -modcache 2>/dev/null || true
	@rm -rf cache/packages/* 2>/dev/null || true
	@echo "✓ Deep clean completed!"

# ==============================================================================
# Comprehensive Cleanup
# ==============================================================================

clean-all: clean-build clean-test clean-logs clean-cache clean-analysis clean-dev ## clean everything
	@echo "🧹 Comprehensive cleanup..."
	@echo "✅ All cleanup tasks completed!"

clean: clean-build clean-test clean-cache clean-analysis ## standard cleanup (most commonly used)
	@echo "✓ Standard cleanup completed!"

# ==============================================================================
# Script-based Cleanup
# ==============================================================================

clean-script: ## run comprehensive cleanup script
	@echo "🧹 Running comprehensive cleanup script..."
	@./scripts/clean.sh
	@echo "✓ Script cleanup completed!"

# ==============================================================================
# Cleanup Help
# ==============================================================================

.PHONY: clean-help

clean-help: ## show help for cleanup commands
	@echo "🧹 Cleanup Commands Help:"
	@echo ""
	@echo "Basic Cleanup:"
	@echo "  make clean           Standard cleanup (build + test + cache + analysis)"
	@echo "  make clean-build     Clean build outputs only"
	@echo "  make clean-test      Clean test files only"
	@echo "  make clean-cache     Clean Go caches only"
	@echo ""
	@echo "Specific Cleanup:"
	@echo "  make clean-logs      Clean log files only"
	@echo "  make clean-analysis  Clean analysis reports only"
	@echo "  make clean-dev       Clean development files only"
	@echo "  make clean-mocks     Clean generated mock files"
	@echo ""
	@echo "Comprehensive Cleanup:"
	@echo "  make clean-all       Clean everything"
	@echo "  make clean-deep      Deep clean (includes module cache)"
	@echo "  make clean-script    Run comprehensive cleanup script"
	@echo ""
	@echo "💡 Tip: Use 'make clean' for routine cleanup, 'make clean-deep' when you need to free up disk space"
