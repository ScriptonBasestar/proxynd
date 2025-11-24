# CLAUDE.md - ProxyND Core

> 📌 v2.0 | Quick Reference | Updated: 2025-11-24
> 📚 Original: 973 lines, 36KB → Refactored: 508 lines, 24KB (48% reduction)
> 🔍 Detailed context: `docs/` directory (90+ documentation files)

---

## 🎯 Quick Start (30 seconds)

**What**: Production-ready, high-performance package manager proxy/mirror server in Go

**Supported Package Managers**: Maven, NPM, APT, Docker Registry, PyPI, YUM, APK (7 total)

**Architecture**: Hexagonal (Ports and Adapters)
```
adapters → ports → usecase → domain
```

**Top 3 Commands**:
```bash
make quick          # Pre-commit: format + lint + unit tests
make start          # Development server with hot reload
make test-coverage  # All tests with coverage report
```

---

## 🚫 Absolute Rules (NEVER violate)

| Rule | Reason | Consequence |
|------|--------|-------------|
| **Domain/usecase NEVER import adapters** | Maintains hexagonal architecture | Breaks testability |
| **Domain = zero external dependencies** | Pure business logic | Framework coupling |
| **Test coverage ≥ 90% for business logic** | Quality gate | CI fails |
| **All builds → `tmp/bin/` or `dist/`** | Gitignored outputs | Pollutes repo |
| **Define port interface FIRST** | Adapters implement ports | Violates dependency flow |
| **Mock ports, never adapters** | Unit test isolation | Integration test leakage |
| **Services coordinate adapters ↔ usecases** | Layer responsibility | Architecture violation |

---

## 📂 Directory Structure (Essential Only)

```
proxynd-core/
├── cmd/
│   ├── proxynd/          # Main server entry
│   ├── proxyndctl/       # CLI management tool
│   └── license-gen/      # Enterprise license generator
├── internal/
│   ├── domain/           # Pure business logic (7 PMs + common + enterprise)
│   │   ├── apk/
│   │   ├── apt/
│   │   ├── docker/
│   │   ├── maven/
│   │   ├── npm/
│   │   ├── pip/          # Note: "pip" in domain, "pypi" in adapters
│   │   ├── yum/
│   │   ├── common/
│   │   └── enterprise/
│   ├── usecase/          # Business workflows
│   ├── ports/            # Interface contracts (6 files: auth, cache, http, middleware, observability, pm)
│   ├── adapters/         # External implementations
│   │   ├── http/fiber/   # Web framework (handlers, middleware, routers)
│   │   ├── pm/{pm}/      # PM drivers (7 implementations)
│   │   ├── aws/          # S3 cache backend
│   │   ├── redis/        # Redis cache backend
│   │   └── observability/  # Logging, metrics, tracing
│   ├── services/         # Service layer (7 PM services + proxy + config)
│   ├── enterprise/       # License, LDAP, SAML, RBAC, replication, analytics
│   ├── plugins/          # Plugin system
│   ├── webhook/          # Event-driven notifications
│   ├── performance/      # Cache optimization, connection pooling
│   └── verification/     # Package signature verification
├── tests/
│   ├── unit/             # Domain + Usecase only (<1ms each)
│   ├── integration/      # Adapters with real dependencies
│   ├── contract/         # API contract validation
│   ├── e2e/              # Full system + upstream data
│   ├── mocks/            # Central mock repository
│   └── helpers/          # Test utilities
├── examples/             # 30+ configuration examples
├── docs/                 # 90+ documentation files (00-99 numbered)
├── scripts/              # 20+ automation scripts
└── deployments/          # Docker, K8s Helm, Terraform, monitoring stack
```

---

## 📋 TOP 10 Commands (Daily Use)

| Command | Purpose | Time | When |
|---------|---------|------|------|
| `make quick` | Pre-commit check: format + lint + unit tests | ~30s | Before every commit |
| `make start` | Hot-reload dev server (Air) | ~5s | Daily development |
| `make test-unit` | Domain/usecase unit tests only | ~10s | TDD cycle |
| `make test-coverage` | All tests with HTML report | ~60s | Pre-PR |
| `make pr-check` | Full pre-PR validation | ~90s | Before PR creation |
| `make build` | Dev binary → `tmp/bin/proxynd` | ~15s | Local testing |
| `make lint` | golangci-lint + vet | ~20s | Code quality check |
| `make help` | Show all targets with categories | <1s | Command discovery |
| `make clean` | Remove build artifacts (keep vendor) | <1s | Clean slate |
| `make setup-all` | First-time setup (deps + tools) | ~120s | Onboarding |

**Extended commands**: Run `make help-dev`, `make help-test`, `make help-build` for categorized help

---

## 🧪 Testing Strategy (4-Layer Overview)

| Layer | Target | Coverage | Speed | Build Tag | Tests |
|-------|--------|----------|-------|-----------|-------|
| **Unit** | Domain + Usecase | 95%+ / 90%+ | <1ms | none | `tests/unit/` + `*_test.go` |
| **Integration** | Adapters | 80%+ | ~100ms | `integration` | `tests/integration/` |
| **Contract** | API compatibility | N/A | ~500ms | `contract` | `tests/contract/` |
| **E2E** | Full system | N/A | ~5s | `e2e` | `tests/e2e/` |

**Total**: 114+ test files (31 in `tests/`, 83+ in `internal/`)

**Key principles**:
- Unit tests: Mock ports only, never adapters
- Table-driven tests with AAA pattern (Arrange, Act, Assert)
- Integration tests: Real dependencies (filesystem, S3, Redis)
- E2E: Real upstream registries + client tools (npm CLI, mvn, apt-get, docker pull)

**Details**: → [`docs/40-testing/testing-guide.md`](docs/40-testing/testing-guide.md)

---

## 🔗 Essential Documentation Links

### Architecture & Design
- [Hexagonal Architecture](docs/10-architecture/hexagonal-architecture.md) - Core architecture explanation
- [ADRs](docs/10-architecture/adr/) - Architecture Decision Records (8 decisions)
- [Dependency Injection](docs/10-architecture/container-dependency-injection.md) - DI container pattern

### Development
- [Testing Guide](docs/40-testing/testing-guide.md) - Comprehensive testing documentation
- [Mocking Strategies](docs/40-testing/mocking/README.md) - Mock generation and usage
- [Contributing](docs/90-development/contributing-guide.md) - Contribution guidelines
- [Development Tools](docs/90-development/tools/) - CI/CD, scripts, workflows

### Configuration
- [Configuration Reference](docs/20-configuration/configuration-reference.md) - Complete config options
- [Environment Variables](docs/20-configuration/environment-variables.md) - Required env vars
- [Hot Reload](docs/20-configuration/hot-reload.md) - Config hot reload feature
- [Migration Guide](docs/20-configuration/migration-guide.md) - Config migration

### Package Managers
- [PM Overview](docs/30-proxy-types/README.md) - All 7 package managers
- Individual PM docs: `docs/30-proxy-types/{pm}/` (apk, apt, docker, maven, npm, pip, yum)

### Operations
- [Deployment Checklist](docs/60-deployment/deployment-checklist.md) - Pre-deployment validation
- [Kubernetes](docs/60-deployment/kubernetes/README.md) - Helm charts and K8s setup
- [Monitoring](docs/70-operations/) - Health, logging, metrics, troubleshooting

### Enterprise & Plugins
- [Plugin Operator Guide](docs/PLUGIN_OPERATOR_GUIDE.md) - Plugin development (21KB)
- [Enterprise Features](docs/99-legal/enterprise-features.md) - License, LDAP, SAML, RBAC
- [Enterprise API Examples](docs/04-api-reference/enterprise-api-examples.md) - Enterprise endpoints

### Reference
- [API Endpoints](docs/80-reference/api-endpoints.md) - REST API reference
- [CLI Reference](docs/80-reference/cli-tools/cli/proxyndctl-reference.md) - proxyndctl commands
- [Tech Stack](docs/80-reference/tech-stack.md) - Technology overview

---

## 🌳 "I want to..." Quick Links

**I want to add a new package manager**:
1. Read [Hexagonal Architecture](docs/10-architecture/hexagonal-architecture.md)
2. Define domain model in `internal/domain/{pm}/`
3. Implement driver in `internal/adapters/pm/{pm}/`
4. Create service in `internal/services/{pm}/`
5. Add handlers in `internal/adapters/http/fiber/handlers/proxy/{pm}_handler.go` + `{pm}_handler_v3.go`
6. Write tests in `tests/unit/{pm}/`, `tests/integration/{pm}/`, `tests/e2e/`
7. Update config in `internal/config/` and examples in `examples/proxy-types/`

**I want to add a new middleware**:
1. Create in `internal/adapters/http/fiber/middleware/{name}/`
2. Implement `func(c *fiber.Ctx) error` signature
3. Use factory pattern from `middleware/factory.go`
4. Register in `internal/adapters/http/fiber/routers/`
5. Write tests with mock HTTP context

**I want to implement a new cache backend**:
1. Define interface in `internal/ports/cache.go` (if not exists)
2. Implement in `internal/adapters/cache/{backend}/`
3. Add factory in `internal/factory/`
4. Update config schema in `internal/config/`
5. Add example config in `examples/features/cache-{backend}.yaml`
6. Write integration tests with real backend

**I want to develop a plugin**:
1. Read [Plugin Operator Guide](docs/PLUGIN_OPERATOR_GUIDE.md)
2. Implement interfaces from `internal/plugins/interfaces.go`
3. Choose category (core/enterprise/cloud)
4. Add config in `examples/plugins/`
5. Register in plugin registry with priority
6. Write plugin tests

**I want to add an enterprise feature**:
1. Implement in `internal/enterprise/` (license, ldap, saml, rbac, etc.)
2. Use feature flag system for runtime enable/disable
3. Add config in `examples/environments/config.enterprise.yaml`
4. Document in `docs/enterprise/`
5. Add enterprise handlers in `internal/adapters/http/fiber/handlers/enterprise/`
6. Add enterprise middleware in `internal/adapters/http/fiber/middleware/enterprise/`

**I want to understand the build system**:
- Run `make help` for enhanced help with categories
- Modular structure: 8 Makefiles (`.dev`, `.build`, `.test`, `.quality`, `.deps`, `.docker`, `.tools`, `.clean`)
- Variables: `VERSION`, `BUILD_TIME`, `COMMIT_SHA` injected via ldflags

**I want to run specific tests**:
```bash
go test ./internal/domain/npm/...                # Single test file
go test -run TestNpmService_HandleRequest ./...  # Specific function
go test -v -race ./...                           # Verbose with race detection
make test-unit                                   # Unit tests only
make test-integration                            # Integration tests (requires tag)
```

**I want to deploy to production**:
1. Read [Deployment Checklist](docs/60-deployment/deployment-checklist.md)
2. Choose platform: [Docker](docs/60-deployment/docker/), [Kubernetes](docs/60-deployment/kubernetes/), [Systemd](docs/60-deployment/systemd/)
3. Review [Monitoring Stack](deployments/monitoring/) - Prometheus, Grafana, Loki, Alertmanager
4. Use deployment script: `scripts/deploy-production.sh`

---

## ⚠️ Common Mistakes (Top 5)

### 1. Domain importing from adapters
```go
// ❌ WRONG
package domain
import "internal/adapters/redis"  // NEVER

// ✅ CORRECT
package domain
// No external dependencies at all
```

### 2. Testing adapter implementation in unit tests
```go
// ❌ WRONG (this is integration test)
func TestFileSystemCache(t *testing.T) {
    cache := adapters.NewFileSystemCache()
    // Testing real filesystem
}

// ✅ CORRECT (unit test with mock)
func TestProxyService_CacheHit(t *testing.T) {
    mockCache := new(mock.CacheManager)
    mockCache.On("Get", "key").Return("value", nil)
    // Test business logic only
}
```

### 3. Confusing build output locations
```bash
# ❌ WRONG - Don't create binaries in project root
go build -o ./proxynd  # Tracked by git!

# ✅ CORRECT - Use make targets
make build            # → tmp/bin/proxynd (gitignored)
make build-all        # → dist/proxynd-{os}-{arch} (gitignored)
```

### 4. PyPI naming inconsistency
```
internal/domain/pip/      # Domain uses "pip"
internal/adapters/pm/pypi/ # Adapters use "pypi"
internal/services/pip/    # Services use "pip"

# For new code: Prefer "pypi" for consistency
```

### 5. Forgetting to mock ports in tests
```go
// ❌ WRONG - Real HTTP calls in unit test
func TestNpmHandler(t *testing.T) {
    resp := http.Get("http://localhost:8080/npm/package")
}

// ✅ CORRECT - Mock the port
func TestNpmService(t *testing.T) {
    mockDriver := new(mock.PackageManagerDriver)
    mockDriver.On("GetPackage", "lodash").Return(pkg, nil)
    svc := NewNpmService(mockDriver)
}
```

---

## 📦 Package Managers (7 Supported)

| PM | Domain | Adapter | Service | Handlers | Test Data |
|----|--------|---------|---------|----------|-----------|
| **APK** | `domain/apk/` | `pm/apk/` | `services/apk/` | `apk_handler.go` + `_v3.go` | `e2e/upstream-data/apk/` |
| **APT** | `domain/apt/` | `pm/apt/` | `services/apt/` | `apt_handler.go` + `_v3.go` | `e2e/upstream-data/apt/` |
| **Docker** | `domain/docker/` | `pm/docker/` | `services/docker/` | `docker_handler.go` + `_v3.go` | `e2e/upstream-data/docker/` |
| **Maven** | `domain/maven/` | `pm/maven/` | `services/maven/` | `maven_handler.go` + `_v3.go` | `e2e/upstream-data/maven/` |
| **NPM** | `domain/npm/` | `pm/npm/` | `services/npm/` | `npm_handler.go` + `_v3.go` | `e2e/upstream-data/npm/` |
| **PyPI** | `domain/pip/` | `pm/pypi/` | `services/pip/` | `pypi_handler.go` + `_v3.go` | `e2e/upstream-data/pip/` |
| **YUM** | `domain/yum/` | `pm/yum/` | `services/yum/` | `yum_handler.go` + `_v3.go` | `e2e/upstream-data/yum/` |

**Note**: All PMs support V1 (modern) and V3 (legacy) APIs for backward compatibility

---

## 🔧 Configuration Priority

**Order** (first found wins):
1. `--config` flag
2. `./config.yaml` (current directory)
3. `~/.config/proxynd/config.yaml` (user config)
4. `/etc/proxynd/config.yaml` (system config)

**Required Environment Variables**:
```bash
CONFIG_DIR=/etc/proxynd    # REQUIRED
STORAGE_DIR=/var/cache     # REQUIRED
SERVER_PORT=8080           # Optional, default: 8080
LOG_LEVEL=info             # Optional: debug/info/warn/error
LOG_FORMAT=json            # Optional: json/text
```

**Example Configs** (`examples/` directory):
- `minimal/` - Minimal configurations
- `environments/` - Environment-specific (including `config.enterprise.yaml`)
- `proxy-types/` - 17 PM-specific files (mirror + proxy for all 7 PMs)
- `features/` - Feature-specific (cache-s3, oauth2, performance, webhook)
- `plugins/` - Plugin configurations (development, minimal, production)
- `global.yaml` - Comprehensive global config with TTL settings

---

## 🎭 Special Features

### Multi-Version API Support
- **V1**: Modern unified API
- **V3**: Legacy compatibility for older clients
- All 7 PMs have both versions

### Connection Pooling
- Location: `internal/pool/`, `internal/performance/connection_pool.go`
- Configurable pool sizes per PM
- Reuse connections to upstream registries
- Performance monitoring via pool handler
- Config: `internal/config/connection_pool_config.go`

### Authentication (6 Methods)
1. **OAuth2**: GitHub, GitLab, Google (`internal/adapters/auth/`, `internal/auth/oauth2/`)
2. **JWT**: Token-based (`internal/auth/jwt/`)
3. **MFA**: Multi-factor (`internal/auth/mfa/`, `mfa_middleware.go`)
4. **API Keys**: Key management (`internal/auth/api_keys.go`)
5. **LDAP**: Enterprise only (`internal/enterprise/`)
6. **SAML 2.0**: Enterprise only (`internal/enterprise/`)

### Middleware (33+ implementations)
Categories: Auth & Security (9), Performance (2), Observability (5), Error Handling (3), Others
See `internal/adapters/http/fiber/middleware/` for all implementations

### Caching Strategy
- **Multi-tier**: Filesystem → Redis → S3
- **Smart eviction**: LRU/TTL based
- **Pattern-based TTL**: SNAPSHOT, dev, alpha, beta, rc
- **Metadata TTL**: Packages, Release, repomd.xml
- **Integrity**: Checksum verification
- **Optimization**: `internal/performance/cache_optimizer.go`
- **Repository**: `internal/repositories/cache/`

### Observability Stack
- **Logging**: Structured JSON via Zap (`internal/adapters/observability/zap/`)
- **Metrics**: Prometheus at `/metrics` (`internal/adapters/observability/prometheus/`)
- **Tracing**: OpenTelemetry (`internal/adapters/observability/opentelemetry/`)
- **Health**: `/health` and `/readiness` endpoints (`internal/health/`)

---

## 🚀 CI/CD Pipeline (Branch-Aware)

### Pull Requests
- Quick validation: lint + format + unit tests
- Time: ~2-3 minutes

### Develop Branch
- Selective integration tests based on changed paths
- Example: `internal/services/npm/**` → NPM integration tests only
- Core changes → all integration tests

### Master Branch
- Full test suite (unit + integration + contract + e2e)
- Enhanced security (gosec, go vet, staticcheck)
- Multi-platform build verification
- Time: ~15-20 minutes

### GitHub Actions Workflows (6)
- `ci.yml` (21KB) - Main CI/CD
- `testing.yml` (22KB) - Test matrix
- `quality.yml` (12KB) - Quality gates + SARIF
- `performance.yml` (19KB) - Regression testing
- `release.yml` (7KB) - Release automation
- `maintenance.yml` (10KB) - Dependency updates

---

## 📚 Related Documentation

### Core Documentation
- [README.md](README.md) (29KB) - Main project README
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [LICENSING.md](LICENSING.md) - Dual licensing (AGPL-3.0 + Commercial)
- [docs/README.md](docs/README.md) (8KB) - Documentation index

### Parent Context
- [`../CLAUDE.md`](../CLAUDE.md) - Workspace-level coordination (multi-repo monorepo)
- For workspace builds, edition differences (Core/Enterprise/Cloud), and Go workspace

### Other Repositories
- `../proxynd-enterprise/CLAUDE.md` - Enterprise plugin development
- `../proxynd-cloud/CLAUDE.md` - Cloud plugin development
- `../proxynd-webui/CLAUDE.md` - Svelte 5 WebUI frontend

### Documentation Index
All documentation organized with 00-99 prefix numbering:
- **00-09**: Overview (`docs/00-overview/`)
- **10-19**: Architecture (`docs/10-architecture/`)
- **20-29**: Configuration (`docs/20-configuration/`)
- **30-39**: Package Managers (`docs/30-proxy-types/`)
- **40-49**: Testing (`docs/40-testing/`)
- **50-59**: Security (`docs/50-security/`)
- **60-69**: Deployment (`docs/60-deployment/`)
- **70-79**: Operations (`docs/70-operations/`)
- **80-89**: Reference (`docs/80-reference/`)
- **90-99**: Development + Legal (`docs/90-development/`, `docs/99-legal/`)

---

## 💡 Development Tips

1. **Architecture First**: Always respect hexagonal boundaries. If domain needs external dependency, it's probably wrong.

2. **Test Coverage**: Maintain 90%+ for business logic. Write tests BEFORE modifying critical paths.

3. **No Breaking Changes**: Maintain API compatibility. Support V1 and V3 APIs.

4. **Config-Driven**: Use configuration for runtime behavior. See `examples/` for 30+ configs.

5. **Observable Code**: Structured logging at key decision points. Log levels:
   - `Debug`: Detailed troubleshooting flow
   - `Info`: Normal operations (cache hit/miss, upstream request)
   - `Warn`: Recoverable errors (retry, fallback)
   - `Error`: Unrecoverable errors requiring attention

6. **Performance**: High-throughput proxy. Profile before optimizing. Watch:
   - Unnecessary allocations in hot paths
   - Blocking I/O in handlers
   - Connection pool exhaustion (use pool manager)
   - Cache efficiency (use optimization strategies)

7. **Security**: Handle sensitive packages. Always:
   - Validate input (use validation middleware)
   - Sanitize paths (prevent directory traversal)
   - Use parameterized queries/safe APIs
   - Never log sensitive data (tokens, credentials)
   - Leverage security middleware and verification system

8. **Documentation**: Update docs when adding features. Follow numbered structure.

9. **Enterprise Features**: Use feature flags. Maintain plugin architecture. Validate licenses.

10. **Legacy Code**: Avoid `*-legacy/` directories. Migrate to current patterns.

---

## 📊 Current Status

**Production-Ready**:
- ✅ 7 package managers (V1 + V3 APIs)
- ✅ 114+ test files with 90%+ coverage
- ✅ Enterprise features (licensing, LDAP, SAML, RBAC)
- ✅ Plugin system for extensibility
- ✅ Webhook system for event-driven architecture
- ✅ Performance optimization (pooling + cache strategies)
- ✅ Complete monitoring stack (Prometheus, Grafana, Loki, Alertmanager)
- ✅ 33+ middleware implementations
- ✅ Modular build system (8 Makefiles)
- ✅ Comprehensive documentation (90+ files)

---

## 📄 License

**Dual-Licensed**:
- **Open Source**: AGPL-3.0 (requires source disclosure)
- **Commercial/Enterprise**: Available for proprietary use

See `LICENSE`, `LICENSE-COMMERCIAL.md`, and `LICENSING.md` for details.

---

**Last Updated**: 2025-11-24 | **Lines**: 508 (was 973) | **Size**: 24KB (was 36KB) | **Reduction**: 48%
