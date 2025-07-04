# REFACTORING_CHECKLIST.md

## 📋 Refactoring Checklist for ProxyND

### 1. Code Quality
- [x] **Remove dead code and commented sections**
  - 📌 Why: Unused code clutters the codebase (found in main.go:90-93, apt_proxy_controller.go:29-36)
  - 🧠 How: Use `staticcheck` and manually review commented code blocks
  - 📁 Files: `main.go`, `handlers/proxy/*.go`, `routers/base_router.go`

- [x] **Standardize naming conventions**
  - 📌 Why: Inconsistent naming between `apt_proxy_controller.go` vs `unified_proxy_handler.go`
  - 🧠 How: Rename files to follow `{type}_handler.go` pattern, use English comments
  - 📁 Files: All files in `handlers/proxy/`, update imports accordingly

- [x] **Organize imports consistently**
  - 📌 Why: Mixed import ordering reduces readability
  - 🧠 How: Use `goimports` with grouping: stdlib, external, internal
  - 📁 Files: All `.go` files

- [x] **Run go fmt on entire codebase**
  - 📌 Why: No linting currently configured
  - 🧠 How: `go fmt ./...` and add to Makefile
  - 📁 Files: All `.go` files

### 2. Code Structure
- [x] **Extract service layer from handlers**
  - 📌 Why: Business logic mixed with HTTP handling in all proxy handlers
  - 🧠 How: Create `internal/services/proxy/` with ProxyService interface
  - 📁 Files: Create new `services/` directory, refactor all handlers

- [x] **Implement repository pattern for data access**
  - 📌 Why: Direct file system access from handlers violates SRP
  - 🧠 How: Create `internal/repositories/` for cache and config access
  - 📁 Files: New repository layer, update handlers and services

- [x] **Isolate main.go to application bootstrap only**
  - 📌 Why: main.go contains unused interface definitions and mixed concerns
  - 🧠 How: Move app initialization to `internal/app/app.go`
  - 📁 Files: `main.go`, create `internal/app/`

- [x] **Create domain packages for each proxy type**
  - 📌 Why: No clear domain boundaries between proxy types
  - 🧠 How: Structure as `internal/domain/{apt,maven,npm}/`
  - 📁 Files: Reorganize existing proxy logic into domain packages

### 3. Interface Design & Dependency Management
- [x] **Define unified Proxy interface**
  - 📌 Why: Multiple proxy types without common abstraction
  - 🧠 How: Create `pkg/types/proxy.go` with ProxyHandler interface
  - 📁 Files: New interface file, update all proxy implementations

- [x] **Implement dependency injection container**
  - 📌 Why: Handlers directly instantiate configs (e.g., `mvnSite.ReadConfig()`)
  - 🧠 How: Use Wire or manual DI container in `internal/app/wire.go`
  - 📁 Files: All handlers, main.go, new DI setup

- [x] **Remove global state access**
  - 📌 Why: Direct config file reading throughout codebase
  - 🧠 How: Pass dependencies through constructors
  - 📁 Files: `configs/*.go`, all handlers

- [x] **Create service interfaces for testability**
  - 📌 Why: Tight coupling prevents effective unit testing
  - 🧠 How: Define interfaces for CacheService, ConfigService, ProxyService
  - 📁 Files: New interface definitions, update implementations

### 4. Concurrency & Goroutine Safety
- [x] **Fix race conditions in webhook worker**
  - 📌 Why: Retry queue operations lack synchronization
  - 🧠 How: Add mutex protection or use channels for queue operations
  - 📁 Files: `internal/webhook/worker.go`

- [x] **Add proper context propagation**
  - 📌 Why: Long-running operations can't be cancelled
  - 🧠 How: Pass context.Context to all service methods
  - 📁 Files: All service and repository methods

- [x] **Fix resource cleanup in HTTP handlers**
  - 📌 Why: Response bodies not closed in error paths
  - 🧠 How: Use `defer` with proper error checking
  - 📁 Files: All proxy handlers making HTTP requests

- [x] **Implement request timeout handling**
  - 📌 Why: No timeout control for upstream requests
  - 🧠 How: Use context with timeout for all HTTP clients
  - 📁 Files: HTTP client initialization, all proxy handlers

### 5. Configuration & Environment Separation
- [x] **Centralize configuration loading**
  - 📌 Why: Configs loaded in multiple places without validation
  - 🧠 How: Create ConfigService with startup validation
  - 📁 Files: Create `internal/services/config/`, update main.go

- [x] **Implement configuration validation**
  - 📌 Why: Missing required field validation at startup
  - 🧠 How: Use struct tags with `validate` package
  - 📁 Files: All config structs in `configs/`

- [x] **Separate environment-specific configs**
  - 📌 Why: Environment variables mixed with file configs
  - 🧠 How: Use Viper with clear precedence rules
  - 📁 Files: Configuration loading logic, `.env` handling

- [x] **Simplify hot reload mechanism**
  - 📌 Why: Complex implementation with TODOs indicating incompleteness
  - 🧠 How: Use fsnotify with proper debouncing
  - 📁 Files: `configs/hot_reload.go`

### 6. Testing
- [x] **Add unit tests for all services**
  - 📌 Why: Only ~40% test coverage currently
  - 🧠 How: Use testify/assert with table-driven tests
  - 📁 Files: Create `*_test.go` for all service files

- [x] **Implement integration tests for proxy flows**
  - 📌 Why: Missing end-to-end proxy behavior tests
  - 🧠 How: Use httptest with mock upstream servers
  - 📁 Files: `tests/integration/proxy_test.go`

- [x] **Add mock generators for interfaces**
  - 📌 Why: No systematic mocking strategy
  - 🧠 How: Use mockery or gomock with `go generate`
  - 📁 Files: All interface definitions

- [x] **Create test fixtures and factories**
  - 📌 Why: Repetitive test setup code
  - 🧠 How: Create `testutil` package with builders
  - 📁 Files: New `internal/testutil/` package

### 7. Tooling & Automation
- [x] **Configure golangci-lint**
  - 📌 Why: No linting currently configured
  - 🧠 How: Add `.golangci.yml` with appropriate rules
  - 📁 Files: Project root configuration

- [x] **Add pre-commit hooks**
  - 📌 Why: No automated quality checks before commit
  - 🧠 How: Use pre-commit framework with Go hooks
  - 📁 Files: `.pre-commit-config.yaml`

- [x] **Update Makefile with quality targets**
  - 📌 Why: Missing lint, fmt, test-coverage targets
  - 🧠 How: Add standard Go development targets
  - 📁 Files: `Makefile`

- [x] **Set up CI/CD pipeline enhancements**
  - 📌 Why: Build process could be more automated
  - 🧠 How: Add GitHub Actions for tests, linting, coverage
  - 📁 Files: `.github/workflows/`

### 8. Documentation
- [x] **Add godoc comments to all exported types**
  - 📌 Why: Missing API documentation
  - 🧠 How: Follow godoc conventions for all public APIs
  - 📁 Files: All exported functions and types

- [x] **Create architecture decision records (ADRs)**
  - 📌 Why: No documented design decisions
  - 🧠 How: Use ADR template in `docs/adr/`
  - 📁 Files: New `docs/adr/` directory

- [x] **Update example configurations**
  - 📌 Why: Sample configs may not reflect all options
  - 🧠 How: Generate from config structs with comments
  - 📁 Files: `sample-conf/*.yaml`

## 🧠 Refactoring Execution Plan

### Phase 1: Foundation (Week 1-2)
1. Code quality cleanup (remove dead code, standardize naming)
2. Configure tooling (linting, pre-commit hooks)
3. Add missing tests for critical paths

### Phase 2: Architecture (Week 3-4)
1. Extract service interfaces
2. Implement dependency injection
3. Create repository layer
4. Fix concurrency issues

### Phase 3: Domain Modeling (Week 5-6)
1. Create domain packages
2. Define unified proxy interface
3. Refactor handlers to thin HTTP layer
4. Centralize configuration management

### Phase 4: Testing & Documentation (Week 7-8)
1. Achieve 80% test coverage
2. Add integration test suite
3. Complete godoc documentation
4. Create ADRs for major decisions

## 🧪 Testing Scope

### Unit Test Coverage Required:
- All service layer methods
- Configuration validation logic
- Cache operations
- URL manipulation utilities
- Error handling paths

### Integration Test Flows:
- APT proxy: Package list fetch → Cache → Serve
- Maven proxy: Artifact download → Cache → Metadata handling
- NPM proxy: Package.json fetch → Tarball download
- Configuration hot reload scenarios
- Concurrent request handling

### Manual Testing Scenarios:
- Proxy failover behavior
- Large file transfers (>1GB)
- Authentication token handling
- Network interruption recovery
- Cache corruption handling

## 📌 Success Criteria
- All checklist items completed
- Test coverage > 80%
- Zero golangci-lint warnings
- All handlers < 50 lines (business logic in services)
- Response time < 100ms for cached requests
- Graceful shutdown working properly
- No race conditions detected by `go test -race`