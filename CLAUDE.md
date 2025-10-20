# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ProxyND** is a high-performance package manager proxy/mirror server written in Go, supporting 7 package managers: Maven, NPM, APT, Docker Registry, PyPI, YUM, and APK. It uses Hexagonal Architecture (Ports and Adapters) for maintainability and testability.

## Core Architecture

### Hexagonal Architecture (Ports and Adapters)

The codebase strictly follows hexagonal architecture with **unidirectional dependencies**:

```
adapters → ports → usecase → domain
```

**Never violate this dependency direction.** Key layers:

- **`internal/domain/`**: Pure business logic, framework-agnostic. NO external dependencies.
- **`internal/usecase/`**: Business workflows orchestrating domain logic.
- **`internal/ports/`**: Interface contracts (20+ interfaces). All external dependencies must be abstracted here.
- **`internal/adapters/`**: Implementations of ports (HTTP/Fiber, PM drivers, cache backends, AWS S3, Redis, observability).

**Critical rules:**
- Domain and usecase layers must NEVER import from adapters
- All external system interactions must go through ports interfaces
- When adding features, define the port interface first, then implement adapters
- Tests should mock ports, never adapters

### Package Manager Structure

Each PM has domain-specific logic in `internal/domain/{pm}/` and adapter implementation in `internal/adapters/pm/{pm}/`. Follow this pattern when adding new PMs or features.

## Development Commands

### Quick Start
```bash
make dev-prepare    # Install dependencies (first-time setup)
make dev-setup      # Create config/storage directories
make dev-run        # Start dev server with hot reload (Air)
```

### Essential Commands
```bash
# Development cycle
make quick          # Fast check: format + lint + unit tests (pre-commit)
make dev-fast       # Faster: format + unit tests only
make full           # Comprehensive: all quality checks

# Testing
make test-unit      # Unit tests only (Domain/Usecase layers, < 1ms each)
make test-integration  # Integration tests (real dependencies)
make test-coverage  # All tests with coverage report
make test-coverage-report  # HTML coverage report → coverage.html

# Pre-submission checks
make pr-check       # Before creating PR: format + lint + coverage
make ci-local       # Simulate full CI pipeline locally

# Code quality
make lint           # Run golangci-lint
make fmt            # Format code with gofmt
make vet            # Run go vet
make security       # Security scan with gosec

# Build
make build          # Development build → ./tmp/bin/proxynd
make build-all      # Multi-platform builds → ./dist/proxynd-{os}-{arch}

# Utilities
make comments       # Show all TODO/FIXME/NOTE comments
make dev-status     # Show project status
make clean          # Clean build artifacts (keeps vendor/)
make clean-all      # Deep clean (removes vendor/ too)
```

### Running Single Tests
```bash
# Run single test file
go test ./internal/domain/npm/...

# Run specific test function
go test -run TestNpmService_HandleRequest ./internal/usecase/

# Run with verbose output
go test -v ./...

# Run with race detection
go test -race ./...
```

## Build System

### Build Outputs and Conventions

**All build outputs are gitignored:**
- Development binary: `./tmp/bin/proxynd` (used by Air for hot reload)
- Release binaries: `./dist/proxynd-{platform}-{arch}`

**IMPORTANT**: Never modify `.gitignore` to track build artifacts. All temporary files must go to `tmp/` directory.

### Build Variables (ldflags)
The build injects these variables at compile time:
- `Version`: Git tag or "dev" (override with `VERSION=x.y.z make build`)
- `BuildTime`: Current timestamp
- `CommitSHA`: Git commit hash

### Configuration Files

**Priority order** (first found wins):
1. `--config` flag
2. `./config.yaml` (current directory)
3. `~/.config/proxynd/config.yaml` (user config)
4. `/etc/proxynd/config.yaml` (system config)

**Required environment variables:**
- `CONFIG_DIR`: Configuration directory path (REQUIRED)
- `STORAGE_DIR`: Cache storage directory path (REQUIRED)
- `SERVER_PORT`: Server port (optional, default: 8080)
- `LOG_LEVEL`: debug/info/warn/error (optional, default: info)
- `LOG_FORMAT`: json/text (optional, default: json)

Example configurations in `examples/config.*.yaml`:
- `config.minimal.yaml`: Minimal setup
- `config.environments/`: Environment-specific configs
- `config.proxy-types/`: PM-specific configurations

## Testing Strategy (4-Layer)

### 1. Unit Tests (`tests/unit/`)
- **Target**: Domain and Usecase layers only
- **Scope**: Pure business logic without external dependencies
- **Speed**: < 1ms per test
- **Coverage**: Domain 95%+, Usecase 90%+
- **Mocking**: Mock ports interfaces only, never adapters
- **Pattern**: Table-driven tests with AAA (Arrange, Act, Assert)

### 2. Integration Tests (`tests/integration/`)
- **Target**: Adapters layer with real dependencies
- **Scope**: HTTP handlers, filesystem, S3, PM drivers
- **Categories**:
  - Component tests: Single adapter in isolation
  - Service tests: Multiple adapters together
  - Cross-PM tests: Verify PM compatibility
  - Performance tests: Throughput/latency gates
- **Build tag**: `// +build integration`
- **Coverage**: 80%+ for adapters

### 3. Contract Tests (`tests/contract/`)
- **Target**: API contract validation
- **Scope**: Ensure compatibility with real PM clients (npm CLI, mvn, apt-get, docker pull)

### 4. E2E Tests (`tests/e2e/`)
- **Target**: Full system integration
- **Scope**: Real upstream registries + client tools

### Writing Tests

**Good unit test example:**
```go
func TestProxyService_CacheStrategy(t *testing.T) {
    tests := []struct {
        name     string
        setup    func(*mock.Cache)
        req      *Request
        expected *Response
    }{
        // test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockCache := new(mock.Cache)
            tt.setup(mockCache)
            svc := NewProxyService(mockCache)

            // Act
            result := svc.Handle(tt.req)

            // Assert
            assert.Equal(t, tt.expected, result)
            mockCache.AssertExpectations(t)
        })
    }
}
```

**Anti-pattern (DO NOT DO):**
```go
// ❌ Wrong: Unit test with real HTTP calls
func TestNpmHandler(t *testing.T) {
    resp := http.Get("http://localhost:8080/npm/package")
    // This is an integration/e2e test, not a unit test
}

// ❌ Wrong: Testing adapter implementation in unit test
func TestFileSystemCache(t *testing.T) {
    cache := adapters.NewFileSystemCache()
    // This belongs in integration tests
}
```

## CI/CD Pipeline

### Branch-Specific Testing Strategy

**Pull Requests:**
- Quick validation: lint, format, unit tests
- Runs in ~2-3 minutes

**Develop branch:**
- Selective integration tests based on changed paths
- Path filters trigger specific PM tests:
  - `internal/services/npm/**` → NPM integration tests
  - `internal/services/maven/**` → Maven integration tests
  - Core changes → all integration tests

**Master branch:**
- Full test suite (unit + integration + contract + e2e)
- Enhanced security scanning (gosec, go vet, staticcheck)
- Build verification for all platforms
- Runs in ~15-20 minutes

### GitHub Actions Workflows

- **`ci.yml`**: Main CI/CD with branch-aware logic
- **`testing.yml`**: Comprehensive test matrix
- **`quality.yml`**: Code quality gates (gosec, golangci-lint)
- **`performance.yml`**: Performance regression testing
- **`release.yml`**: Release automation

## Code Conventions

### Naming Patterns

**Interfaces (in `internal/ports/`):**
- `<Component>Manager`: `CacheManager`, `TokenManager`
- `<PM>Driver`: `NpmDriver`, `MavenDriver`
- `<Feature>Service`: `HealthService`, `ProxyService`

**Implementations (in `internal/adapters/`):**
- `<Backend><Component>`: `FileSystemCache`, `RedisCache`, `S3Cache`
- `<Framework><Adapter>`: `FiberHTTPServer`

**Handlers:**
- `<PM>Handler`: `NpmHandler`, `MavenHandler`
- `<PM>HandlerV3`: For legacy API version support

**Tests:**
- `Test<Component>_<Scenario>_<Expected>`
- Example: `TestProxyService_CacheHit_ReturnsFromCache`

### File Organization

```
internal/
├── domain/{pm}/          # PM-specific business logic
├── usecase/              # Business workflows
├── ports/                # Interface definitions
├── adapters/
│   ├── http/fiber/       # Web framework adapter
│   │   ├── handlers/     # HTTP request handlers
│   │   ├── middleware/   # Middleware components
│   │   └── routers/      # Route registration
│   ├── pm/{pm}/          # Package manager drivers
│   ├── cache/            # Cache backends
│   └── observability/    # Logging/metrics/tracing
└── app/                  # Application bootstrap
```

**Rules:**
- Domain logic: NEVER depend on frameworks or external libraries
- Adapters: Keep framework-specific code isolated
- Tests: Place `*_test.go` adjacent to implementation
- Mocks: Generate mocks in `tests/mocks/` directory

### Adding a New Package Manager

Follow this checklist:

1. **Define domain model** in `internal/domain/{pm}/`
   - Package structure, metadata, version handling
   - Pure Go structs, no external dependencies

2. **Implement driver** in `internal/adapters/pm/{pm}/`
   - Implement `ports.PackageManagerDriver` interface
   - Handle upstream registry communication

3. **Create HTTP handlers** in `internal/adapters/http/fiber/handlers/proxy/`
   - `{pm}_handler.go`: Main request handling logic
   - Register routes in `internal/adapters/http/fiber/routers/`

4. **Write tests**:
   - Unit tests for domain logic in `tests/unit/{pm}/`
   - Integration tests in `tests/integration/{pm}/`
   - Contract tests in `tests/contract/{pm}/`

5. **Update configuration**:
   - Add PM config schema in `internal/config/`
   - Create example config in `examples/config.proxy-types/`

6. **Document**:
   - Update README.md package manager list
   - Add usage examples
   - Document PM-specific quirks

## Special Features

### Multi-Version API Support
- **V1**: Modern unified API
- **V3**: Legacy compatibility layer for older clients
- Implementation: Separate handlers (`npm_handler.go` vs `npm_handler_v3.go`)

### Connection Pooling
- Managed in `internal/pool/`
- Configurable pool sizes per PM
- Reuse connections to upstream registries

### Authentication
- OAuth2: GitHub, GitLab, Google (in `internal/adapters/auth/`)
- JWT tokens with MFA support
- Middleware chain: `internal/adapters/http/fiber/middleware/auth/`

### Observability
- **Logging**: Structured JSON via Zap
- **Metrics**: Prometheus metrics at `/metrics`
- **Tracing**: OpenTelemetry integration
- **Health checks**: `/health` and `/readiness` endpoints

### Caching Strategy
- Multi-tier: Filesystem → Redis → S3
- Smart eviction based on LRU/TTL
- Integrity verification with checksums
- Repository: `internal/repositories/cache/`

## Common Development Tasks

### Adding a New Middleware
1. Create middleware in `internal/adapters/http/fiber/middleware/{name}/`
2. Implement Fiber middleware signature: `func(c *fiber.Ctx) error`
3. Register in router chain: `internal/adapters/http/fiber/routers/`
4. Write tests with mock HTTP context

### Implementing a New Cache Backend
1. Define interface in `internal/ports/cache.go` (if not exists)
2. Implement adapter in `internal/adapters/cache/{backend}/`
3. Add factory method in `internal/factory/`
4. Update config schema
5. Write integration tests with real backend

### Adding Configuration Options
1. Update config struct in `internal/config/`
2. Add validation in config validator
3. Update example configs in `examples/`
4. Document in README.md configuration section
5. Add tests for config parsing

## Docker and Deployment

### Docker Build
```bash
make docker-build       # Build image: proxynd:latest
make docker-run         # Run container with default config
make docker-stop        # Stop running container
make docker-push        # Push to registry (requires login)
```

### Kubernetes Deployment
Helm charts available in `deployments/kubernetes/`:
```bash
helm install proxynd ./deployments/kubernetes/proxynd-chart
```

### Infrastructure as Code
Terraform configurations in `deployments/terraform/`:
- AWS: ECS/Fargate deployment
- GCP: Cloud Run deployment
- Azure: Container Instances deployment

## Important Notes for AI Development

1. **Architecture First**: Always respect hexagonal architecture boundaries. If you need to add a dependency to domain/usecase, it's probably wrong.

2. **Test Coverage**: Maintain 90%+ coverage for business logic. Write tests BEFORE modifying critical paths.

3. **No Breaking Changes**: This is a proxy server—maintain API compatibility. Clients depend on stable endpoints.

4. **Config-Driven**: Use configuration for runtime behavior. Avoid hardcoded values.

5. **Observable Code**: Add structured logging at key decision points. Use appropriate log levels:
   - `Debug`: Detailed flow for troubleshooting
   - `Info`: Normal operations (cache hit/miss, upstream request)
   - `Warn`: Recoverable errors (retry logic, fallback)
   - `Error`: Unrecoverable errors requiring attention

6. **Performance Matters**: This is a high-throughput proxy. Profile before optimizing, but be conscious of:
   - Unnecessary allocations in hot paths
   - Blocking I/O in request handlers
   - Connection pool exhaustion
   - Cache efficiency

7. **Security**: This proxy may handle sensitive packages. Always:
   - Validate input thoroughly
   - Sanitize paths to prevent directory traversal
   - Use parameterized queries/safe APIs
   - Never log sensitive data (tokens, credentials)

8. **Documentation**: Update docs when adding features. Good code explains "how", good docs explain "why".

## Current Branch and Status

**Active branch**: `develop` (commit at time of CLAUDE.md creation)
**Main branch**: `master` (use for PRs)

Recent workflow changes:
- Monitoring workflow cleanup (removed redundant health checks)
- Build system improvements (standardized on `tmp/bin/` for outputs)
- Quality workflow enhancements (gosec SARIF upload)

## CLI Management Tool

**proxyndctl** provides comprehensive CLI management:

### Common Operations
```bash
# Cache management
proxyndctl cache list
proxyndctl cache clear --type npm
proxyndctl cache size

# Configuration
proxyndctl config validate
proxyndctl config show

# Testing connectivity
proxyndctl test all
proxyndctl test --proxy maven

# Server status
proxyndctl status

# Maven-specific
proxyndctl maven-index build
proxyndctl maven-backup create
```

### API Endpoints
All CLI functionality exposed via REST API:
- Cache: `/api/cache/*`
- Config: `/api/config/*`
- Users: `/api/user/*`
- Testing: `/api/test/*`
- Status: `/api/status/*`

Verify API endpoints with: `make verify-api` or `make verify-api-json`

## License

Dual-licensed:
- **Open Source**: AGPL-3.0 (requires source disclosure)
- **Commercial**: Available for proprietary use

See LICENSE, LICENSE-COMMERCIAL.md, and LICENSING.md for details.

## Support and Documentation

- Full documentation: [docs/INDEX.md](docs/INDEX.md)
- Getting started: [docs/01-getting-started/README.md](docs/01-getting-started/README.md)
- Configuration guide: [docs/03-configuration/README.md](docs/03-configuration/README.md)
- Development guide: [docs/05-development/README.md](docs/05-development/README.md)
- AI collaboration guardrails: [CLAUDE.md](CLAUDE.md) (this file)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Testing strategy: [TESTING.md](TESTING.md)
- Refactoring checklist: [REFACTORING_CHECKLIST.md](REFACTORING_CHECKLIST.md)
