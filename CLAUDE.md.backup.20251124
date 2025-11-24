# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ProxyND** is a production-ready, high-performance package manager proxy/mirror server written in Go, supporting 7 package managers: Maven, NPM, APT, Docker Registry, PyPI, YUM, and APK. It uses Hexagonal Architecture (Ports and Adapters) for maintainability and testability, with enterprise-grade features including webhooks, plugins, performance optimization, and comprehensive monitoring.

## Core Architecture

### Hexagonal Architecture (Ports and Adapters)

The codebase strictly follows hexagonal architecture with **unidirectional dependencies**:

```
adapters → ports → usecase → domain
```

**Never violate this dependency direction.** Key layers:

- **`internal/domain/`**: Pure business logic, framework-agnostic. NO external dependencies.
  - 7 PM-specific domains: `apk/`, `apt/`, `docker/`, `maven/`, `npm/`, `pip/`, `yum/`
  - `common/` - Shared domain logic
  - `enterprise/` - Enterprise features domain logic

- **`internal/usecase/`**: Business workflows orchestrating domain logic.
  - Cache strategies, proxy service, health checks, search functionality

- **`internal/ports/`**: Interface contracts (6 major interface files). All external dependencies must be abstracted here.
  - `auth.go`, `cache.go`, `http.go`, `middleware.go`, `observability.go`, `pm.go`

- **`internal/adapters/`**: Implementations of ports (HTTP/Fiber, PM drivers, cache backends, AWS S3, Redis, observability).

- **`internal/services/`**: Service layer coordinating between adapters and use cases.
  - PM-specific services for each package manager
  - `proxy/` - Generic proxy service
  - `config/` - Configuration service
  - `adapters/` - Service adapters

**Critical rules:**
- Domain and usecase layers must NEVER import from adapters
- All external system interactions must go through ports interfaces
- When adding features, define the port interface first, then implement adapters
- Tests should mock ports, never adapters
- Services layer coordinates between adapters and use cases but maintains architectural boundaries

### Package Manager Structure

Each PM has:
- **Domain logic**: `internal/domain/{pm}/`
- **Adapter implementation**: `internal/adapters/pm/{pm}/`
- **Service layer**: `internal/services/{pm}/`
- **V1 & V3 Handlers**: `internal/adapters/http/fiber/handlers/proxy/{pm}_handler.go` + `{pm}_handler_v3.go`

**Note**: PyPI uses inconsistent naming - "pip" in domain/services, "pypi" in adapters. New code should prefer "pypi" for consistency.

## Build System

### Modular Makefile Structure

The build system uses **8 modular Makefiles** for organization:

```
Makefile              # Main entry point with quick aliases
├── Makefile.dev.mk       # Development environment and setup
├── Makefile.build.mk     # Build and installation
├── Makefile.test.mk      # Testing and validation
├── Makefile.quality.mk   # Code quality and linting
├── Makefile.deps.mk      # Dependency management
├── Makefile.docker.mk    # Docker operations
├── Makefile.tools.mk     # Tool installation and management
└── Makefile.clean.mk     # Cleanup operations
```

Use `make help` for enhanced help system with categories:
- `make help-dev`, `make help-build`, `make help-test`, etc.

### Essential Commands

```bash
# Quick Development Cycle
make quick          # Fast check: format + lint + unit tests (pre-commit)
make dev-fast       # Faster: format + unit tests only
make full           # Comprehensive: all quality checks

# Development Server
make start          # Quick start: setup and run development server
make stop           # Stop running development server
make restart        # Restart development server
make status         # Check development server status

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

# Setup
make setup-all      # Complete project setup (dev environment + all tools)
make dev-prepare    # Install dependencies (first-time setup)
make dev-setup      # Create config/storage directories

# Utilities
make comments       # Show all TODO/FIXME/NOTE comments
make dev-status     # Show project status and git info
make clean          # Clean build artifacts (keeps vendor/)
make clean-all      # Deep clean (removes vendor/ too)
make logs           # Show recent log files
```

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

**Example configurations** (`examples/` directory):
- `minimal/` - Minimal configurations
- `environments/` - Environment-specific configs (including `config.enterprise.yaml`)
- `proxy-types/` - PM-specific configurations (17 files: mirror and proxy configs for all PMs)
- `features/` - Feature-specific configs:
  - `cache-s3.yaml` - S3 cache backend
  - `oauth2.yaml` - OAuth2 configuration
  - `performance.yaml` - Performance tuning
  - `verification-config.yaml` - Package verification
  - `webhook.yaml` - Webhook configuration
- `plugins/` - Plugin configurations
- `global.yaml` - Comprehensive global configuration with TTL settings
- `users.yml` - User management example

## Testing Strategy (4-Layer, 114+ Test Files)

ProxyND has comprehensive test coverage with **114+ test files** (31 in `tests/`, 83+ `*_test.go` in `internal/`).

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
- **Test data**: Realistic upstream data for all 7 PMs in `tests/e2e/upstream-data/`
  - APK (Alpine v3.16), APT (Ubuntu Jammy), Maven, NPM, PIP, YUM, Docker

### Mock Generation
- **Script**: `scripts/generate_mocks.sh`
- **Central mocks**: `tests/mocks/`
- **Component mocks**: Throughout codebase in `/mocks/` subdirectories
  - `internal/health/mocks/`, `internal/interfaces/mocks/`, etc.
  - Per-PM service mocks: `internal/services/{pm}/mocks/`

### Test Helpers
- `tests/helpers/` - Test utility functions
- `internal/testutil/` - Internal test utilities

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

### GitHub Actions Workflows (6 workflows)

- **`ci.yml`** (21KB): Main CI/CD with branch-aware logic
- **`testing.yml`** (22KB): Comprehensive test matrix
- **`quality.yml`** (12KB): Code quality gates (gosec with SARIF, golangci-lint, staticcheck)
- **`performance.yml`** (19KB): Performance regression testing with benchmarks
- **`release.yml`** (7KB): Release automation and multi-platform builds
- **`maintenance.yml`** (10KB): Dependency updates and cleanup

## Enterprise Features

ProxyND includes **Enterprise Edition** capabilities with dual licensing model.

### Enterprise Modules (`internal/enterprise/`)

**Location**: `internal/enterprise/`, `internal/domain/enterprise/`

**Features**:
- **License management** (`license/`) - License validation and feature flags
- **Advanced authentication**:
  - LDAP authentication
  - SAML 2.0 integration
  - Role-Based Access Control (RBAC)
- **Replication** - Multi-datacenter synchronization with conflict resolution
- **Security** - Vulnerability scanning and security policy engine
- **Analytics** - Metrics collection and dashboard APIs

**Enterprise configuration**: `examples/environments/config.enterprise.yaml`
**Enterprise documentation**: `docs/enterprise/`
**Enterprise handlers**: `internal/adapters/http/fiber/handlers/enterprise/`
**Enterprise middleware**: `internal/adapters/http/fiber/middleware/enterprise/`

**Architecture**:
- Feature flag system for runtime enable/disable
- Plugin-based architecture for modular enterprise features
- Offline license validation with public key in source code
- License generation tool: `cmd/license-gen/`

## Plugin System

ProxyND provides a comprehensive **plugin system** for extensibility.

### Plugin Architecture (`internal/plugins/`)

**Features**:
- Config-driven plugin management via `plugins.yaml`
- Priority-based initialization order
- Lifecycle hooks: Init → Ready → Shutdown with timeouts
- Environment-specific overrides (dev/staging/production)
- Failure policies (continue/halt)
- Full observability (metrics, logs, state tracking)

**Plugin types**:
- **Core plugins** - Always available
- **Enterprise plugins** - Requires enterprise build (RBAC, audit, etc.)
- **Cloud plugins** - Multi-tenancy, billing

**Components**:
- `manager.go` - Plugin lifecycle management
- `registry.go` - Plugin registration
- `middleware.go` - Plugin middleware integration
- `group_manager.go` - Plugin group management
- `/adapters/` - Plugin adapters (e.g., `npm_adapter.go`)

**Configuration**: `examples/plugins/` (development, minimal, production configs)
**Documentation**: `docs/PLUGIN_OPERATOR_GUIDE.md` (21KB comprehensive guide)

## Webhook System

Full-featured **webhook system** for event-driven notifications.

### Webhook Components (`internal/webhook/`)

**Core features**:
- Event-driven architecture for package events
- Batch processing (`batch_manager.go`)
- Persistent queue with retry logic (`persistent_queue.go`, `queue.go`)
- Rate limiting (`rate_limiter.go`)
- Event filtering (`/filtering/`)
- History tracking (`history.go`)
- Multiple sender implementations (`/sender/`)
- Alert integration with alert system

**Configuration**: `examples/features/webhook.yaml` (7KB)

**Alert integration**:
- `internal/alerts/webhook_alerter.go`
- `internal/alerts/webhook_events.go`

**Testing**: Dedicated test handler for webhook validation

## Performance Management

Advanced **performance optimization** module.

### Performance Components (`internal/performance/`)

**Features**:
- Cache optimization strategies (`cache_optimizer.go`, `cache_strategies.go` - 23KB)
- Connection pooling (`connection_pool.go` - 21KB)
- Request optimization (`request_optimizer.go`)
- Resource monitoring (`resource_monitor.go`)
- Performance middleware
- Configurable performance settings

**Configuration**: `examples/features/performance.yaml` (4KB)
**Workflow**: `.github/workflows/performance.yml` (19KB) - Performance regression testing
**Handler**: `internal/adapters/http/fiber/handlers/pool_handler.go` - Connection pool management

## Alert System

Multi-channel **alerting system**.

### Alert Components (`internal/alerts/`)

**Features**:
- Pluggable alert manager
- Log-based alerts (`log_alerter.go`)
- Webhook-based alerts (`webhook_alerter.go`)
- Event definitions for various scenarios (`webhook_events.go`)

**Integration**: Works with webhook system and observability stack

## Package Verification

**Package signature verification** system.

### Verification Components (`internal/verification/`)

**Features**:
- Package integrity verification (`package_verifier.go`)
- APK-specific verification (`/apk/`)
- Signature validation
- Checksum verification

**Configuration**: `examples/features/verification-config.yaml` (3KB)
**Handler**: `verification_handler.go` in proxy handlers

## Backup and Sync

**Maven-specific** backup and synchronization capabilities.

### Backup Components (`internal/backup/`)

**Features**:
- Incremental backup functionality (`maven_incremental_backup.go`)
- Repository synchronization (`maven_sync.go`)
- Maven index management

**CLI integration**: `proxyndctl maven-backup create`

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
- `<PM>HandlerV3`: For legacy API version support (all 7 PMs have V1 and V3 handlers)

**Tests:**
- `Test<Component>_<Scenario>_<Expected>`
- Example: `TestProxyService_CacheHit_ReturnsFromCache`

### File Organization

```
internal/
├── domain/{pm}/          # PM-specific business logic (7 PMs + common + enterprise)
├── usecase/              # Business workflows
├── ports/                # Interface definitions (6 major files)
├── services/             # Service layer coordinating adapters and use cases
│   ├── {pm}/             # PM-specific services (7 PMs)
│   ├── proxy/            # Generic proxy service
│   ├── config/           # Configuration service
│   └── adapters/         # Service adapters
├── adapters/
│   ├── http/fiber/       # Web framework adapter
│   │   ├── handlers/     # HTTP request handlers
│   │   │   ├── auth/     # Authentication handlers
│   │   │   ├── enterprise/  # Enterprise handlers
│   │   │   └── proxy/    # PM proxy handlers (35 files, 255KB)
│   │   ├── middleware/   # 33+ middleware implementations
│   │   │   └── enterprise/  # Enterprise middleware
│   │   └── routers/      # Route registration
│   ├── pm/{pm}/          # Package manager drivers (7 PMs)
│   ├── aws/              # AWS services (S3)
│   ├── redis/            # Redis cache backend
│   ├── stub/             # Stub implementations for testing
│   └── observability/    # Logging/metrics/tracing
│       ├── opentelemetry/  # OpenTelemetry integration
│       ├── prometheus/   # Prometheus metrics
│       └── zap/          # Zap structured logging
├── enterprise/           # Enterprise features
├── webhook/              # Webhook system
├── plugins/              # Plugin system
├── performance/          # Performance management
├── alerts/               # Alert system
├── verification/         # Package verification
├── backup/               # Backup and sync
├── container/            # Dependency injection container
├── containerhandlers/    # Container-based handlers
├── context/              # Context management
├── dto/                  # Data transfer objects
├── errors/               # Error definitions
├── factory/              # Factory pattern implementations
├── goroutine/            # Goroutine pool management
├── health/               # Health check implementations
├── helpers/              # Helper utilities
├── interfaces/           # Additional interface definitions
├── logging/              # Logging utilities
├── metrics/              # Metrics collection
├── mirror/               # Mirror functionality
├── pool/                 # Connection pool implementations
├── proxy/                # Proxy core logic
├── repositories/         # Repository pattern
│   ├── cache/            # Cache repository
│   └── config/           # Config repository
├── routers/              # Additional routing logic
├── security/             # Security utilities
├── testutil/             # Test utilities
├── app/                  # Application bootstrap
├── handlers-legacy/      # Legacy handlers (deprecated)
└── middleware-legacy/    # Legacy middleware (deprecated)
```

**Rules:**
- Domain logic: NEVER depend on frameworks or external libraries
- Adapters: Keep framework-specific code isolated
- Tests: Place `*_test.go` adjacent to implementation
- Mocks: Generate mocks in `/mocks/` subdirectories or `tests/mocks/`
- Services: Coordinate between adapters and use cases while maintaining architectural boundaries

### Legacy Components

**Deprecated directories and files:**
- `internal/handlers-legacy/` - Legacy handler implementations
- `internal/middleware-legacy/` - Legacy middleware
- `internal/config/hot_reload.go.deprecated`
- `internal/config/viper_config.go.deprecated`
- `internal/services/config/viper_service.go.deprecated`

**Migration**: These are kept for reference but should not be used in new code. See migration documentation for upgrading from legacy patterns.

### Adding a New Package Manager

Follow this checklist:

1. **Define domain model** in `internal/domain/{pm}/`
   - Package structure, metadata, version handling
   - Pure Go structs, no external dependencies

2. **Implement driver** in `internal/adapters/pm/{pm}/`
   - Implement `ports.PackageManagerDriver` interface
   - Handle upstream registry communication

3. **Create service layer** in `internal/services/{pm}/`
   - Service logic coordinating adapters and use cases
   - Include `/mocks/` subdirectory for testing

4. **Create HTTP handlers** in `internal/adapters/http/fiber/handlers/proxy/`
   - `{pm}_handler.go`: Modern V1 API
   - `{pm}_handler_v3.go`: Legacy V3 API for backward compatibility
   - Register routes in `internal/adapters/http/fiber/routers/`

5. **Write tests**:
   - Unit tests for domain logic in `tests/unit/{pm}/`
   - Integration tests in `tests/integration/{pm}/`
   - Contract tests in `tests/contract/{pm}/`
   - E2E test data in `tests/e2e/upstream-data/{pm}/`

6. **Update configuration**:
   - Add PM config schema in `internal/config/`
   - Create example configs in `examples/proxy-types/`
   - Add mirror and proxy configuration examples

7. **Document**:
   - Update README.md package manager list
   - Add PM-specific documentation in `docs/30-proxy-types/{pm}/`
   - Document PM-specific quirks and features

## Special Features

### Multi-Version API Support
- **V1**: Modern unified API
- **V3**: Legacy compatibility layer for older clients
- Implementation: All 7 PMs have both versions (`npm_handler.go` + `npm_handler_v3.go`)

### Connection Pooling
- Managed in `internal/pool/` and `internal/performance/connection_pool.go`
- Configurable pool sizes per PM
- Reuse connections to upstream registries
- Performance monitoring via pool handler
- Configuration: `internal/config/connection_pool_config.go`

### Authentication
- **OAuth2**: GitHub, GitLab, Google (`internal/adapters/auth/`, `internal/auth/oauth2/`)
- **JWT**: Token-based authentication with middleware (`internal/auth/jwt/`)
- **MFA**: Multi-factor authentication support (`internal/auth/mfa/`, `mfa_middleware.go`)
- **API Keys**: API key management (`internal/auth/api_keys.go`)
- **Audit**: Authentication audit logging (`internal/auth/audit/`)
- **Enterprise**: LDAP, SAML 2.0, RBAC (Enterprise Edition)
- Middleware chain: `internal/adapters/http/fiber/middleware/auth/`

### Enhanced Middleware (33+ implementations)

**Authentication & Security**:
- `auth.go`, `jwt_auth.go` - Authentication middleware
- `mfa_middleware.go` (14KB) - Multi-factor authentication
- `security.go`, `advanced_security.go` (16KB) - Security features
- `security_headers.go` - Security headers
- `input_validation.go` (17KB) - Comprehensive input validation
- `http_method_validator.go` - HTTP method validation
- `ip_filter.go` - IP filtering
- `permission.go` - Permission checks
- `proxy_policy.go` (9KB) - Proxy policy enforcement

**Performance**:
- `enhanced_rate_limiter.go` (8KB) - Advanced rate limiting
- `rate_limiter.go` - Basic rate limiting

**Observability**:
- `logging.go` - Request logging
- `access_log.go` - Access logging
- `structured_access_log.go` (8KB) - Structured access logs
- `metrics.go` - Metrics collection
- `tracing.go` - Distributed tracing

**Error Handling**:
- `error_handler.go` - Error handling
- `error_recovery.go` - Error recovery
- `recovery.go` - Panic recovery

**Factory**: `factory.go` (10KB) - Middleware factory pattern

### Observability
- **Logging**: Structured JSON via Zap (`internal/adapters/observability/zap/`)
- **Metrics**: Prometheus metrics at `/metrics` (`internal/adapters/observability/prometheus/`)
- **Tracing**: OpenTelemetry integration (`internal/adapters/observability/opentelemetry/`)
- **Health checks**: `/health` and `/readiness` endpoints (`internal/health/`)
- **Structured logging**: Enhanced access logs and request tracing
- **Metrics collection**: `internal/metrics/`

### Caching Strategy
- Multi-tier: Filesystem → Redis → S3
- Smart eviction based on LRU/TTL
- Pattern-based TTL overrides (SNAPSHOT, dev, alpha, beta, rc)
- Metadata file TTLs (Packages, Release, repomd.xml)
- Integrity verification with checksums
- Cache optimization strategies (`internal/performance/cache_optimizer.go`)
- Repository: `internal/repositories/cache/`
- Configuration: `examples/global.yaml` for comprehensive TTL settings

### Search Functionality
- **Usecase**: `internal/usecase/search_service.go`
- **Handler**: `internal/adapters/http/fiber/handlers/search_handler.go`
- Cross-PM package search capabilities

### Dependency Injection
- **Container**: `internal/container/` - DI container implementation
- **Container Handlers**: `internal/containerhandlers/` - Container-based handlers
- Improves testability and modularity

## Common Development Tasks

### Adding a New Middleware
1. Create middleware in `internal/adapters/http/fiber/middleware/{name}/`
2. Implement Fiber middleware signature: `func(c *fiber.Ctx) error`
3. Register in router chain: `internal/adapters/http/fiber/routers/`
4. Consider using middleware factory pattern (`factory.go`)
5. Write tests with mock HTTP context

### Implementing a New Cache Backend
1. Define interface in `internal/ports/cache.go` (if not exists)
2. Implement adapter in `internal/adapters/cache/{backend}/`
3. Add factory method in `internal/factory/`
4. Update config schema in `internal/config/`
5. Add example config in `examples/features/`
6. Write integration tests with real backend

### Adding Configuration Options
1. Update config struct in `internal/config/`
2. Add validation in `schema_validator.go`
3. Update example configs in `examples/`
4. Document in appropriate `docs/` section
5. Add tests for config parsing and validation

### Implementing a Plugin
1. Review plugin architecture in `docs/PLUGIN_OPERATOR_GUIDE.md`
2. Implement plugin interfaces in `internal/plugins/interfaces.go`
3. Create plugin in appropriate category (core/enterprise/cloud)
4. Add plugin configuration in `examples/plugins/`
5. Register plugin in plugin registry
6. Set initialization priority
7. Write plugin tests

### Adding Enterprise Features
1. Implement feature in `internal/enterprise/` following modular structure
2. Use feature flag system for runtime enable/disable
3. Add enterprise configuration in `examples/environments/config.enterprise.yaml`
4. Document in `docs/enterprise/`
5. Ensure features work with plugin system if applicable
6. Add enterprise-specific middleware or handlers as needed

## Docker and Deployment

### Docker Build
```bash
make docker-build       # Build image: proxynd:latest
make docker-run         # Run container with default config
make docker-stop        # Stop running container
make docker-push        # Push to registry (requires login)
```

### Kubernetes Deployment
Helm charts available in `deployments/helm/`:
```bash
helm install proxynd ./deployments/helm/proxynd-chart
```

**Helm chart includes**:
- Deployment manifests
- Service configurations (standard and local)
- Ingress rules
- Persistent volume claims
- ConfigMaps
- Network policies
- K3s example values

### Monitoring Stack

Complete **monitoring infrastructure** in `deployments/monitoring/`:

**Components**:
- **Prometheus** - Metrics collection
  - `prometheus.yml` - Main configuration
  - `/prometheus/rules/proxynd-alerts.yml` - Alert rules
- **Grafana** - Visualization
  - Dashboard provisioning
  - Datasource configuration
  - Pre-built dashboards in `deployments/grafana/`
- **Loki** - Log aggregation (`/loki/loki-config.yml`)
- **Promtail** - Log shipping (`/promtail/promtail-config.yml`)
- **Alertmanager** - Alert routing (`/alertmanager/alertmanager.yml`)
- **Blackbox Exporter** - Endpoint monitoring (`/blackbox/blackbox.yml`)

**Deployment**: `docker-compose.yml` for complete monitoring stack

**Standalone**: `deployments/prometheus/alert-rules.yml`

### Systemd Deployment
Systemd unit files in `deployments/systemd/` for traditional bare-metal deployments.

### Infrastructure as Code
Terraform configurations in `deployments/terraform/`:
- AWS: ECS/Fargate deployment
- GCP: Cloud Run deployment
- Azure: Container Instances deployment

## CLI Management Tool

### proxyndctl (`cmd/proxyndctl/`)

**Comprehensive CLI** for ProxyND management with commands:

**Commands** (`/commands/` directory):
- `cache.go` - Cache management
- `config.go` - Configuration management
- `status.go` - Status checking
- `test.go` - Testing utilities
- `user.go` - User management
- `completion.go` - Shell completion
- `docs.go` - Documentation generation
- `common.go` - Common utilities

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

# Shell completion
proxyndctl completion bash > /etc/bash_completion.d/proxyndctl
```

### API Endpoints
All CLI functionality exposed via REST API:
- Cache: `/api/cache/*`
- Config: `/api/config/*`
- Users: `/api/user/*`
- Testing: `/api/test/*`
- Status: `/api/status/*`

Verify API endpoints with: `make verify-api` or `make verify-api-json`

## DevOps Scripts

Comprehensive **automation scripts** in `/scripts/`:

**Testing & Validation**:
- `test-coverage.sh` - Coverage reporting
- `generate_mocks.sh` - Mock generation
- `smoke-test-adapters.sh` (7KB) - Adapter smoke testing
- `snapshot-api.sh` (8KB) - API snapshot testing
- `validate-architecture.sh` (9KB) - Architecture validation
- `validate-deployment.sh` (12KB) - Deployment validation
- `validate-gitignore.sh` (3KB) - Gitignore validation

**Operations**:
- `deploy-production.sh` (11KB) - Production deployment
- `security_audit.sh` (9KB) - Security auditing
- `verify-api-endpoints.sh` (9KB) - API endpoint verification

**Enterprise**:
- `benchmark-enterprise-api.sh` - Enterprise API benchmarking

**Git Automation**:
- `merge-dependabot-updates.sh` - Dependabot automation
- `cleanup-dependabot-branches.sh` - Branch cleanup

**Installation**:
- `install-completion.sh` (8KB) - Shell completion installation
- `install-man-pages.sh` (4KB) - Man page installation

**Load Testing**:
- `/loadtest/` subdirectory - Load testing tools

## Documentation Structure

Comprehensive **numbered documentation hierarchy** in `docs/`:

**Main sections** (00-99 numbering):
- **00-overview/** - Project overview
- **04-api-reference/** - API documentation
- **10-architecture/** - Architecture docs
  - `/adr/` - Architecture Decision Records
- **20-configuration/** - Configuration guides
- **30-proxy-types/** - PM-specific documentation
  - `/apk/`, `/apt/`, `/docker/`, `/maven/`, `/npm/`, `/pip/`, `/yum/`
- **40-testing/** - Testing documentation
  - `/mocking/` - Mocking strategies
- **50-security/** - Security documentation
- **60-deployment/** - Deployment guides
  - `/docker/`, `/kubernetes/`, `/systemd/`
- **70-operations/** - Operations guides
  - `/health/`, `/logging/`, `/metrics/`
- **80-reference/** - Reference documentation
  - `/cli-tools/cli/` - CLI tool reference
  - `/future-plans/` - Future roadmap
  - `/project-info/project/` - Project information
- **90-development/** - Development guides
  - `/quality/`, `/tools/`
- **99-legal/** - Legal information
- **api/** - API documentation
- **deployment/** - Additional deployment docs
- **enterprise/** - Enterprise documentation

**Special documentation**:
- `docs/README.md` (8KB) - Documentation index
- `docs/PLUGIN_OPERATOR_GUIDE.md` (21KB) - Plugin development guide

**Root documentation**:
- `README.md` (29KB) - Main project README
- `CLAUDE.md` - This file (AI collaboration guide)
- `CONTRIBUTING.md` - Contribution guidelines
- `LICENSING.md` - Licensing information
- `LICENSE`, `LICENSE-COMMERCIAL.md` - License files
- `PENDING-MANUAL-COMMIT.md` - Workflow file commit tracker

## Important Notes for AI Development

1. **Architecture First**: Always respect hexagonal architecture boundaries. If you need to add a dependency to domain/usecase, it's probably wrong. Services layer coordinates between adapters and use cases.

2. **Test Coverage**: Maintain 90%+ coverage for business logic. Write tests BEFORE modifying critical paths. Use the comprehensive mock system.

3. **No Breaking Changes**: This is a proxy server—maintain API compatibility. All 7 PMs support V1 and V3 APIs for backward compatibility.

4. **Config-Driven**: Use configuration for runtime behavior. Extensive example configs available. Avoid hardcoded values.

5. **Observable Code**: Add structured logging at key decision points. Full observability stack available. Use appropriate log levels:
   - `Debug`: Detailed flow for troubleshooting
   - `Info`: Normal operations (cache hit/miss, upstream request)
   - `Warn`: Recoverable errors (retry logic, fallback)
   - `Error`: Unrecoverable errors requiring attention

6. **Performance Matters**: This is a high-throughput proxy. Use performance module. Profile before optimizing, but be conscious of:
   - Unnecessary allocations in hot paths
   - Blocking I/O in request handlers
   - Connection pool exhaustion (use pool manager)
   - Cache efficiency (use cache optimization strategies)

7. **Security**: This proxy may handle sensitive packages. Always:
   - Validate input thoroughly (use input validation middleware)
   - Sanitize paths to prevent directory traversal
   - Use parameterized queries/safe APIs
   - Never log sensitive data (tokens, credentials)
   - Leverage security middleware and verification system

8. **Documentation**: Update docs when adding features. Follow numbered documentation structure. Good code explains "how", good docs explain "why".

9. **Enterprise Features**: Use feature flags for enterprise capabilities. Maintain plugin-based architecture. Ensure license validation.

10. **Webhook Integration**: Leverage webhook system for event-driven features. Use alert system for notifications.

11. **Modular Design**: Use plugin system for extensibility. Follow DI container pattern for testability.

12. **Legacy Code**: Avoid using deprecated components in `*-legacy/` directories. Migrate to current patterns.

## Current Status

**Production-ready** with extensive features:
- **7 package managers** fully implemented with V1 and V3 API support
- **114+ test files** with comprehensive coverage
- **Enterprise features** including licensing, LDAP, SAML, RBAC
- **Plugin system** for extensibility
- **Webhook system** for event-driven architecture
- **Performance optimization** with connection pooling and cache strategies
- **Complete monitoring stack** (Prometheus, Grafana, Loki, Alertmanager)
- **33+ middleware** implementations for security and features
- **Modular build system** with 8 Makefile modules
- **Comprehensive documentation** with 00-99 numbered structure

**Recent improvements**:
- Modular Makefile structure for better organization
- Enhanced help system with categorized commands
- Plugin operator guide for extensibility
- Enterprise documentation structure
- Performance workflow for regression testing
- Maintenance workflow for automation

## License

Dual-licensed:
- **Open Source**: AGPL-3.0 (requires source disclosure)
- **Commercial/Enterprise**: Available for proprietary use

See `LICENSE`, `LICENSE-COMMERCIAL.md`, and `LICENSING.md` for details.

## Support and Documentation

**Main documentation**: `docs/README.md` (8KB) - Documentation index with full hierarchy

**Quick links**:
- Getting started: `docs/00-overview/`
- Configuration: `docs/20-configuration/`
- Package managers: `docs/30-proxy-types/`
- Deployment: `docs/60-deployment/`
- Development: `docs/90-development/`
- Enterprise: `docs/enterprise/`
- Plugin development: `docs/PLUGIN_OPERATOR_GUIDE.md`
- API reference: `docs/04-api-reference/`
- Architecture: `docs/10-architecture/`

**Contributing**: `CONTRIBUTING.md`
**AI collaboration**: `CLAUDE.md` (this file)
