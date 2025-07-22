# Makefile.tools.mk - Tool Installation and Management
# All development tools installation and management

# ==============================================================================
# Core Development Tools
# ==============================================================================

.PHONY: install-tools install-test install-mockery check-tools
.PHONY: install-golangci-lint install-format-tools

install-tools: ## install core development tools
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
	@go install github.com/securecode/gosec/v2/cmd/gosec@latest
	@echo "Installing nancy..."
	@go install github.com/sonatype-nexus-community/nancy@latest
	@echo "Installing godepgraph..."
	@go install github.com/kisielk/godepgraph@latest
	@echo "Installing unused..."
	@go install honnef.co/go/tools/cmd/unused@latest
	@echo "✅ All development tools installed!"

install-mockery: ## install mockery for mock generation
	@echo "Installing mockery..."
	@which mockery > /dev/null || go install github.com/vektra/mockery/v2@latest
	@echo "Mockery installed!"

install-test: install-tools ## install test-specific tools
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

check-tools: ## check which development tools are installed
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

# ==============================================================================
# Pre-commit Hooks
# ==============================================================================

.PHONY: pre-commit-install pre-commit-run pre-commit-update

pre-commit-install: ## install and setup pre-commit hooks
	@echo "Setting up pre-commit hooks..."
	@./scripts/setup_precommit.sh

pre-commit-run: ## run pre-commit on all files
	@echo "Running pre-commit on all files..."
	pre-commit run --all-files

pre-commit-update: ## update pre-commit hooks
	@echo "Updating pre-commit hooks..."
	pre-commit autoupdate

# ==============================================================================
# Documentation Tools
# ==============================================================================

.PHONY: docs docs-generate docs-serve

docs: docs-generate docs-serve ## generate and serve documentation

docs-generate: ## generate go documentation
	@echo "Generating documentation..."
	@which godoc > /dev/null || go install golang.org/x/tools/cmd/godoc@latest
	@echo "Documentation can be viewed at http://localhost:6060/pkg/proxynd/"

docs-serve: ## serve documentation locally
	@echo "Starting documentation server..."
	godoc -http=:6060

# ==============================================================================
# Dependency Management
# ==============================================================================

.PHONY: deps deps-update deps-graph

deps: ## manage go dependencies
	@echo "Managing dependencies..."
	go mod download
	go mod tidy
	go mod verify
	@echo "Dependencies verified!"

# deps-update moved to Makefile.deps.mk

deps-graph: ## generate dependency graph
	@echo "Generating dependency graph..."
	@which godepgraph > /dev/null || go install github.com/kisielk/godepgraph@latest
	godepgraph -s ./... | dot -Tpng -o deps-graph.png
	@echo "Dependency graph saved to deps-graph.png"
