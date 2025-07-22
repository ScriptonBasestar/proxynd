# Makefile - ProxyND Package Manager Proxy Server
# Modular Makefile structure with enhanced functionality
# Original Makefile backed up as Makefile.original

# ==============================================================================
# Project Configuration
# ==============================================================================

# Project metadata
PROJECTNAME := proxynd
ENV := develop
DOCKER_REGISTRY := scriptonbasestar
VERSION ?= $(shell git describe --always --abbrev=0 --tags 2>/dev/null || echo "dev")

# Go configuration
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org

# Colors for output (exported to all sub-makefiles)
export CYAN := \\033[36m
export GREEN := \\033[32m
export YELLOW := \\033[33m
export RED := \\033[31m
export BLUE := \\033[34m
export MAGENTA := \\033[35m
export RESET := \\033[0m

# ==============================================================================
# Include Modular Makefiles
# ==============================================================================

include Makefile.dev.mk        # Development environment and setup
include Makefile.build.mk      # Build and installation
include Makefile.test.mk       # Testing and validation
include Makefile.quality.mk    # Code quality and linting
include Makefile.deps.mk       # Dependency management
include Makefile.docker.mk     # Docker operations
include Makefile.tools.mk      # Tool installation and management
include Makefile.clean.mk      # Cleanup operations

# ==============================================================================
# Quick Access Aliases
# ==============================================================================

.PHONY: start stop restart status logs

# Quick development aliases
start: dev-run                 ## quick start: setup and run development server
stop:                         ## stop running development server
	@echo "$(YELLOW)Stopping development server...$(RESET)"
	@pkill -f "air\|go run" || echo "$(GREEN)No running processes found$(RESET)"

restart: stop start           ## restart development server

status:                       ## check development server status
	@echo "$(CYAN)Checking development server status...$(RESET)"
	@pgrep -f "air\|go run" > /dev/null && echo "$(GREEN)✅ Development server is running$(RESET)" || echo "$(RED)❌ Development server is not running$(RESET)"

logs:                         ## show recent log files
	@echo "$(CYAN)Recent log files:$(RESET)"
	@find logs -name "*.log" -type f -exec ls -la {} \; 2>/dev/null || echo "$(YELLOW)No log files found$(RESET)"

# ==============================================================================
# Main Workflow Aliases
# ==============================================================================

.PHONY: quick full setup-all dev-fast pr-check ci-local comments dev-status

quick: fmt lint test-unit     ## quick development check (format + lint + unit tests)
	@echo "$(GREEN)✅ Quick development check completed!$(RESET)"

full: quality check-all       ## full quality check (comprehensive)
	@echo "$(GREEN)✅ Full quality check completed!$(RESET)"

setup-all: dev install-all   ## complete project setup (dev environment + all tools)
	@echo "$(GREEN)🎉 Complete project setup finished!$(RESET)"

dev-fast: fmt test-unit       ## quick development cycle (format and unit tests only)
	@echo "$(GREEN)✅ Fast development cycle completed!$(RESET)"

pr-check: fmt lint test-coverage ## pre-PR submission check
	@echo "$(GREEN)✅ Pre-PR check completed - ready for submission!$(RESET)"

ci-local: clean quality test-all ## run full CI pipeline locally
	@echo "$(GREEN)✅ Local CI pipeline completed!$(RESET)"

comments: ## show all TODO/FIXME/NOTE comments in codebase
	@echo "$(CYAN)=== TODO comments ===$(RESET)"
	@grep -r "TODO" --include="*.go" . | grep -v vendor | grep -v .git || echo "$(GREEN)No TODOs found!$(RESET)"
	@echo ""
	@echo "$(CYAN)=== FIXME comments ===$(RESET)"
	@grep -r "FIXME" --include="*.go" . | grep -v vendor | grep -v .git || echo "$(GREEN)No FIXMEs found!$(RESET)"
	@echo ""
	@echo "$(CYAN)=== NOTE comments ===$(RESET)"
	@grep -r "NOTE" --include="*.go" . | grep -v vendor | grep -v .git || echo "$(GREEN)No NOTEs found!$(RESET)"

dev-status: ## show current development status
	@echo "$(CYAN)Development Status Check$(RESET)"
	@echo "$(BLUE)========================$(RESET)"
	@echo ""
	@echo "$(GREEN)📊 Project Status:$(RESET)"
	@printf "  %-20s " "Git Status:"; if git status --porcelain | grep -q .; then echo "$(YELLOW)Modified files$(RESET)"; else echo "$(GREEN)Clean$(RESET)"; fi
	@printf "  %-20s " "Current Branch:"; git branch --show-current 2>/dev/null || echo "$(RED)Unknown$(RESET)"
	@printf "  %-20s " "Last Commit:"; git log -1 --format="%h %s" 2>/dev/null | cut -c1-50 || echo "$(RED)No commits$(RESET)"
	@echo ""
	@echo "$(GREEN)🔧 Build Status:$(RESET)"
	@printf "  %-20s " "Binary Exists:"; if [ -f "proxynd" ]; then echo "$(GREEN)Yes$(RESET)"; else echo "$(YELLOW)No$(RESET)"; fi
	@printf "  %-20s " "Coverage File:"; if [ -f "coverage.out" ]; then echo "$(GREEN)Yes$(RESET)"; else echo "$(YELLOW)No$(RESET)"; fi

# ==============================================================================
# Enhanced Help System
# ==============================================================================

.DEFAULT_GOAL := help

.PHONY: help help-dev help-build help-test help-quality help-docker help-deps help-tools help-clean

help: ## show main help menu with categories
	@echo "$(CYAN)"
	@echo "╔══════════════════════════════════════════════════════════════════════════════╗"
	@echo "║                           $(MAGENTA)ProxyND Makefile Help$(CYAN)                            ║"
	@echo "║                    $(YELLOW)Go Package Manager Proxy Server$(CYAN)                       ║"
	@echo "╚══════════════════════════════════════════════════════════════════════════════╝"
	@echo "$(RESET)"
	@echo "$(GREEN)📋 Main Categories:$(RESET)"
	@echo "  $(YELLOW)make help-dev$(RESET)      🛠️  Development environment and setup"
	@echo "  $(YELLOW)make help-build$(RESET)    🔨 Build, installation, and deployment"
	@echo "  $(YELLOW)make help-test$(RESET)     🧪 Testing, benchmarks, and validation"
	@echo "  $(YELLOW)make help-quality$(RESET)  ✨ Code quality, formatting, and linting"
	@echo "  $(YELLOW)make help-docker$(RESET)   🐳 Docker and container operations"
	@echo "  $(YELLOW)make help-deps$(RESET)     📦 Dependency management and updates"
	@echo "  $(YELLOW)make help-tools$(RESET)    🔧 Tool installation and management"
	@echo "  $(YELLOW)make help-clean$(RESET)    🧹 Cleanup and maintenance"
	@echo ""
	@echo "$(GREEN)🚀 Quick Commands:$(RESET)"
	@echo "  $(CYAN)make start$(RESET)         Start development server (dev-run)"
	@echo "  $(CYAN)make stop$(RESET)          Stop development server"
	@echo "  $(CYAN)make restart$(RESET)       Restart development server"
	@echo "  $(CYAN)make status$(RESET)        Check development server status"
	@echo "  $(CYAN)make quick$(RESET)         Quick check (format + lint + unit tests)"
	@echo "  $(CYAN)make dev-fast$(RESET)      Fast development cycle (format + unit tests)"
	@echo "  $(CYAN)make full$(RESET)          Full quality check (comprehensive)"
	@echo "  $(CYAN)make pr-check$(RESET)      Pre-PR submission check"
	@echo "  $(CYAN)make ci-local$(RESET)      Run full CI pipeline locally"
	@echo "  $(CYAN)make setup-all$(RESET)     Complete project setup"
	@echo ""
	@echo "$(GREEN)🔍 Development Tools:$(RESET)"
	@echo "  $(CYAN)make comments$(RESET)      Show all TODO/FIXME/NOTE comments"
	@echo "  $(CYAN)make dev-status$(RESET)    Show current development status"
	@echo ""
	@echo "$(GREEN)� Pro Tips:$(RESET)"
	@echo "  • Use $(YELLOW)'make quick'$(RESET) for fast development iteration"
	@echo "  • Use $(YELLOW)'make full'$(RESET) before pushing to ensure quality"
	@echo "  • Use $(YELLOW)'make setup-all'$(RESET) for first-time project setup"
	@echo "  • All commands support tab completion if bash-completion is installed"
	@echo ""
	@echo "$(BLUE)📖 Documentation: $(RESET)https://github.com/your-repo/proxynd"

help-dev: ## show development help
	@echo "$(GREEN)🛠️  Development Environment Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.dev.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

help-build: ## show build help
	@echo "$(GREEN)🔨 Build and Installation Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.build.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

help-test: ## show testing help
	@echo "$(GREEN)🧪 Testing and Validation Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.test.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

help-quality: ## show quality help
	@echo "$(GREEN)✨ Code Quality Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.quality.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

help-docker: ## show docker help
	@echo "$(GREEN)🐳 Docker Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.docker.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

help-deps: ## show dependency help
	@echo "$(GREEN)📦 Dependency Management Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.deps.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

help-tools: ## show tools help
	@echo "$(GREEN)🔧 Tool Management Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.tools.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

help-clean: ## show cleanup help
	@echo "$(GREEN)🧹 Cleanup Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile.clean.mk | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

# ==============================================================================
# Project Information
# ==============================================================================

.PHONY: info about

info: ## show project information and current configuration
	@echo "$(CYAN)"
	@echo "╔══════════════════════════════════════════════════════════════════════════════╗"
	@echo "║                         $(MAGENTA)ProxyND Project Information$(CYAN)                       ║"
	@echo "╚══════════════════════════════════════════════════════════════════════════════╝"
	@echo "$(RESET)"
	@echo "$(GREEN)📋 Project Details:$(RESET)"
	@echo "  Name:           $(YELLOW)$(PROJECTNAME)$(RESET)"
	@echo "  Environment:    $(YELLOW)$(ENV)$(RESET)"
	@echo "  Version:        $(YELLOW)$(VERSION)$(RESET)"
	@echo "  Registry:       $(YELLOW)$(DOCKER_REGISTRY)$(RESET)"
	@echo ""
	@echo "$(GREEN)🏗️  Build Environment:$(RESET)"
	@echo "  Go Version:     $$(go version | cut -d' ' -f3)"
	@echo "  GOPROXY:        $(GOPROXY)"
	@echo "  GOSUMDB:        $(GOSUMDB)"
	@echo "  GOPATH:         $$(go env GOPATH)"
	@echo "  GOROOT:         $$(go env GOROOT)"
	@echo ""
	@echo "$(GREEN)📁 Directories:$(RESET)"
	@echo "  Config:         ./tmp/config"
	@echo "  Storage:        ./tmp/storage"
	@echo "  Logs:           ./logs"
	@echo ""
	@echo "$(GREEN)🔧 Available Modules:$(RESET)"
	@echo "  • Development    (Makefile.dev.mk)"
	@echo "  • Build          (Makefile.build.mk)"
	@echo "  • Testing        (Makefile.test.mk)"
	@echo "  • Quality        (Makefile.quality.mk)"
	@echo "  • Dependencies   (Makefile.deps.mk)"
	@echo "  • Docker         (Makefile.docker.mk)"
	@echo "  • Tools          (Makefile.tools.mk)"
	@echo "  • Cleanup        (Makefile.clean.mk)"
