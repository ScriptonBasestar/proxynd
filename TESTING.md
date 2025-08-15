# 🧪 ProxyND Testing Strategy & Guidelines

## 📋 Overview

ProxyND follows a **4-layer test pyramid** aligned with our **Hexagonal + Clean Architecture**:

```
                    ┌─────────────────┐
                    │   E2E Tests     │ ← Full system validation
                    │ (docker-compose)│
                    └─────────────────┘
                  ┌───────────────────────┐
                  │  Integration Tests    │ ← Adapters layer
                  │  (External systems)   │
                  └───────────────────────┘
                ┌─────────────────────────────┐
                │     Contract Tests          │ ← Ports layer
                │   (Interface contracts)     │
                └─────────────────────────────┘
              ┌───────────────────────────────────┐
              │           Unit Tests              │ ← Domain/Usecase layers
              │      (Business logic)             │
              └───────────────────────────────────┘
```

## 🏗️ Architecture & Test Boundaries

### Layer Mapping

| Layer | Directory | Test Focus | Mock Level |
|-------|-----------|------------|------------|
| **Domain** | `internal/domain/` | Business rules, entities | None (pure logic) |
| **Usecase** | `internal/usecase/` | Application logic, orchestration | Mock ports only |
| **Ports** | `internal/ports/` | Interface contracts | Mock implementations |
| **Adapters** | `internal/adapters/` | External integrations | Mock external systems |

### Dependency Flow (Testing)

```
Adapters → Ports → Usecase → Domain
   ↑         ↑        ↑        ↑
   │         │        │        └─ Unit tests (no mocks)
   │         │        └─ Unit tests (mock ports)
   │         └─ Contract tests (mock implementations)
   └─ Integration tests (mock external systems)
```

## 🎯 Test Layer Definitions

### 1. Unit Tests (`make test-unit`)

**Purpose**: Validate individual components in isolation

**Scope**:
- Domain entities and value objects
- Usecase business logic
- Pure functions and algorithms

**Characteristics**:
- Fast execution (< 1ms per test)
- No external dependencies
- High code coverage target (90%+)
- No network calls or file I/O

**Location**: `*_test.go` files alongside source code

**Example**: `internal/domain/npm/models_test.go`

```go
func TestPackageVersion_Validate(t *testing.T) {
    tests := []struct {
        name    string
        version string
        isValid bool
    }{
        {"valid semver", "1.2.3", true},
        {"invalid format", "invalid", false},
    }
    // Test implementation...
}
```

### 2. Contract Tests (`make test-services`)

**Purpose**: Verify interface contracts between layers

**Scope**:
- Port interface implementations
- Adapter contract compliance
- Cross-layer communication protocols

**Characteristics**:
- Mock implementations of ports
- Verify input/output contracts
- Test error handling scenarios
- Interface compatibility validation

**Location**: Test files in port directories

**Example**: `internal/ports/cache_test.go`

```go
func TestCacheManager_Contract(t *testing.T) {
    // Test that any cache implementation follows the contract
    implementations := []ports.CacheManager{
        &filesystem.CacheManager{},
        &redis.CacheManager{},
        &s3.CacheManager{},
    }

    for _, impl := range implementations {
        t.Run(reflect.TypeOf(impl).String(), func(t *testing.T) {
            testCacheContract(t, impl)
        })
    }
}
```

### 3. Integration Tests (`make test-integration`)

**Purpose**: Test component interactions with external systems

**Scope**:
- Database connections
- HTTP client integrations
- File system operations
- Cache backend interactions

**Characteristics**:
- Use real external dependencies (when possible)
- Test failure scenarios
- Verify side effects
- Longer execution time (seconds)

**Location**: `tests/integration/`

**Example**: `tests/integration/npm_integration_test.go`

### 4. E2E Tests (`make test-api`)

**Purpose**: Validate complete user scenarios

**Scope**:
- Full HTTP request/response cycles
- Multi-step workflows
- Cross-package manager scenarios
- Performance characteristics

**Characteristics**:
- Use docker-compose.e2e.yml
- Real network calls
- Complete system validation
- Slowest tests (minutes)

**Location**: `tests/e2e/`

## 📦 Package Manager Test Matrix

Each package manager (npm, pip, apt, docker, maven, yum, apk) must be tested across:

### Cache Scenarios

#### Cache Hit Tests
```go
func TestNPM_CacheHit(t *testing.T) {
    // Given: Package is already cached
    server := setupTestServer(t)
    cachePackage(t, server, "express@4.18.0")

    // When: Request the same package
    resp := requestPackage(t, server, "express@4.18.0")

    // Then: Verify cache hit
    assert.Equal(t, "HIT", resp.Header.Get("X-Cache"))
    assert.LessOrEqual(t, resp.Duration, 50*time.Millisecond)
}
```

#### Cache Miss Tests
```go
func TestNPM_CacheMiss(t *testing.T) {
    // Given: Package is not cached
    server := setupTestServer(t)

    // When: Request uncached package
    resp := requestPackage(t, server, "lodash@4.17.21")

    // Then: Verify cache miss and upstream fetch
    assert.Equal(t, "MISS", resp.Header.Get("X-Cache"))
    assert.Greater(t, resp.Duration, 100*time.Millisecond)

    // And: Verify package is now cached
    cachedResp := requestPackage(t, server, "lodash@4.17.21")
    assert.Equal(t, "HIT", cachedResp.Header.Get("X-Cache"))
}
```

### Signature Verification Tests

#### Valid Signature
```go
func TestNPM_ValidSignature(t *testing.T) {
    server := setupTestServer(t)

    // When: Request package with valid signature
    resp := requestPackage(t, server, "react@18.0.0")

    // Then: Verify signature validation passed
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.Equal(t, "VERIFIED", resp.Header.Get("X-Signature-Status"))
}
```

#### Invalid Signature
```go
func TestNPM_InvalidSignature(t *testing.T) {
    server := setupTestServerWithCorruptedPackage(t)

    // When: Request package with invalid signature
    resp := requestPackage(t, server, "corrupted-package@1.0.0")

    // Then: Verify request is blocked
    assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
    assert.Contains(t, resp.Body, "signature verification failed")
}
```

### Error Handling Tests

#### Upstream Timeout
```go
func TestNPM_UpstreamTimeout(t *testing.T) {
    server := setupTestServerWithSlowUpstream(t, 30*time.Second)

    // When: Request times out upstream
    resp := requestPackage(t, server, "slow-package@1.0.0")

    // Then: Verify graceful error handling
    assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
    assert.Contains(t, resp.Body, "upstream timeout")
}
```

#### Network Failure
```go
func TestNPM_NetworkFailure(t *testing.T) {
    server := setupTestServerWithFailingUpstream(t)

    // When: Upstream is unreachable
    resp := requestPackage(t, server, "unreachable-package@1.0.0")

    // Then: Verify fallback behavior
    assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
    assert.Contains(t, resp.Body, "upstream unavailable")
}
```

## 🔧 Test Configuration & Setup

### Test Environment Variables

```bash
# Test configuration
export PROXYND_TEST_MODE=true
export PROXYND_LOG_LEVEL=debug
export PROXYND_CACHE_DIR=/tmp/proxynd-test-cache
export PROXYND_CONFIG_DIR=./tests/config

# Package manager endpoints for testing
export NPM_REGISTRY_URL=https://registry.npmjs.org
export PYPI_INDEX_URL=https://pypi.org/simple
export DOCKER_REGISTRY_URL=https://registry-1.docker.io
```

### Test Server Setup

```go
// Common test server setup
func SetupTestServer(t *testing.T) *TestServer {
    // Load test configuration
    config := LoadTestConfig(t)

    // Setup test database/cache
    cache := SetupTestCache(t)

    // Create test server
    server := &TestServer{
        App:    setupFiberApp(config, cache),
        Config: config,
        Cache:  cache,
    }

    // Start server
    go server.Start()

    // Wait for ready
    require.NoError(t, server.WaitForReady())

    // Cleanup on test completion
    t.Cleanup(func() {
        server.Shutdown()
        cache.Cleanup()
    })

    return server
}
```

## 📊 Test Execution & Makefile Integration

### Test Targets

| Make Target | Layer | Description | Duration |
|-------------|-------|-------------|----------|
| `test-unit` | Unit | Fast isolated tests | < 10s |
| `test-services` | Contract | Interface verification | < 30s |
| `test-integration` | Integration | Component integration | < 2m |
| `test-api` | E2E | Full system validation | < 5m |
| `test-all` | All | Complete test suite | < 8m |

### Makefile.test.mk Enhancement

```makefile
# 4-Layer Test Architecture
test-unit: ## Unit tests (domain/usecase layers)
	@echo "🧪 Running unit tests..."
	go test -short -race ./internal/domain/... ./internal/usecase/...

test-contract: ## Contract tests (ports layer)
	@echo "🤝 Running contract tests..."
	go test -tags=contract ./internal/ports/...

test-integration: ## Integration tests (adapters layer)
	@echo "🔗 Running integration tests..."
	go test -tags=integration ./tests/integration/...

test-e2e: ## End-to-end tests (full system)
	@echo "🌐 Running E2E tests..."
	docker-compose -f docker-compose.e2e.yml up -d
	go test -tags=e2e ./tests/e2e/...
	docker-compose -f docker-compose.e2e.yml down

# Package Manager Specific Tests
test-npm: ## NPM proxy tests
	go test -v ./tests/integration/npm_*

test-pip: ## PyPI proxy tests
	go test -v ./tests/integration/pip_*

test-apt: ## APT proxy tests
	go test -v ./tests/integration/apt_*

test-docker: ## Docker registry tests
	go test -v ./tests/integration/docker_*

test-maven: ## Maven repository tests
	go test -v ./tests/integration/maven_*

test-yum: ## YUM repository tests
	go test -v ./tests/integration/yum_*

test-apk: ## APK repository tests
	go test -v ./tests/integration/apk_*

# Test with coverage by layer
test-coverage-unit:
	go test -coverprofile=unit.coverage ./internal/domain/... ./internal/usecase/...

test-coverage-integration:
	go test -coverprofile=integration.coverage ./tests/integration/...

test-coverage-all: test-coverage-unit test-coverage-integration
	go tool cover -html=unit.coverage -o coverage-unit.html
	go tool cover -html=integration.coverage -o coverage-integration.html
```

## 🚀 CI/CD Pipeline Integration

### GitHub Actions Workflow (.github/workflows/test.yml)

```yaml
name: Testing Pipeline

on: [push, pull_request]

jobs:
  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: make test-unit

  contract-tests:
    name: Contract Tests
    runs-on: ubuntu-latest
    needs: unit-tests
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: make test-contract

  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    needs: contract-tests
    strategy:
      matrix:
        package-manager: [npm, pip, apt, docker, maven, yum, apk]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: make test-${{ matrix.package-manager }}

  e2e-tests:
    name: E2E Tests
    runs-on: ubuntu-latest
    needs: integration-tests
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: make test-e2e
```

## 📈 Quality Gates & Coverage

### Coverage Targets

| Layer | Target | Rationale |
|-------|--------|-----------|
| Domain | 95%+ | Pure business logic |
| Usecase | 90%+ | Application logic |
| Ports | 85%+ | Interface contracts |
| Adapters | 75%+ | External integrations |

### Quality Checks

```bash
# Pre-commit hooks
make lint test-unit test-contract

# CI pipeline gates
make test-all test-coverage validate

# Release validation
make test-e2e test-benchmark
```

## 🔍 Test Data & Fixtures

### Test Package Fixtures

```
tests/fixtures/
├── npm/
│   ├── express-4.18.0.tgz
│   ├── lodash-4.17.21.tgz
│   └── metadata/
├── pip/
│   ├── django-4.0.tar.gz
│   ├── requests-2.28.1.whl
│   └── simple/
├── apt/
│   ├── packages/
│   └── Release.gpg
└── docker/
    ├── manifests/
    └── blobs/
```

### Mock Configurations

```yaml
# tests/config/test.yaml
server:
  port: 0  # Random available port
  host: 127.0.0.1

cache:
  backend: memory  # Fast in-memory cache for tests
  ttl: 1h

registries:
  npm:
    enabled: true
    upstream: http://localhost:8081/npm  # Mock server
  pip:
    enabled: true
    upstream: http://localhost:8081/pip
```

## 🛠️ Test Utilities & Helpers

### Common Test Utilities

```go
// tests/testutil/server.go
package testutil

import (
    "testing"
    "time"
    "net/http"
)

// TestServer provides test server functionality
type TestServer struct {
    App    *fiber.App
    URL    string
    Config *config.Config
}

func (ts *TestServer) Request(method, path string, body io.Reader) *http.Response {
    // Implementation...
}

func (ts *TestServer) WaitForReady() error {
    // Implementation...
}

// tests/testutil/assertions.go
func AssertCacheHit(t *testing.T, resp *http.Response) {
    assert.Equal(t, "HIT", resp.Header.Get("X-Cache"))
}

func AssertCacheMiss(t *testing.T, resp *http.Response) {
    assert.Equal(t, "MISS", resp.Header.Get("X-Cache"))
}

func AssertSignatureValid(t *testing.T, resp *http.Response) {
    assert.Equal(t, "VERIFIED", resp.Header.Get("X-Signature-Status"))
}
```

## 📝 Test Writing Guidelines

### 1. Test Naming Convention

```go
// Format: Test[Component]_[Scenario]_[ExpectedResult]
func TestNpmHandler_CacheHit_ReturnsFromCache(t *testing.T) {}
func TestCacheManager_InvalidatePattern_ClearsMatchingEntries(t *testing.T) {}
func TestSignatureVerifier_CorruptedPackage_ReturnsError(t *testing.T) {}
```

### 2. Test Structure (AAA Pattern)

```go
func TestExample(t *testing.T) {
    // Arrange (Given)
    server := setupTestServer(t)
    packageName := "express"
    version := "4.18.0"

    // Act (When)
    resp := server.Request("GET", "/proxy/npm/"+packageName, nil)

    // Assert (Then)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}
```

### 3. Table-Driven Tests

```go
func TestPackageValidation(t *testing.T) {
    tests := []struct {
        name        string
        packageName string
        version     string
        expected    bool
    }{
        {"valid package", "express", "4.18.0", true},
        {"invalid version", "express", "invalid", false},
        {"empty name", "", "1.0.0", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := validatePackage(tt.packageName, tt.version)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## 🔧 Test Debugging & Troubleshooting

### Debug Configuration

```bash
# Enable verbose test output
go test -v ./...

# Run specific test
go test -v -run TestNpmHandler_CacheHit ./tests/integration/

# Debug with delve
dlv test -- -test.run TestNpmHandler_CacheHit

# Enable race detection
go test -race ./...

# Generate CPU profile
go test -cpuprofile=cpu.prof ./...
```

### Common Issues & Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| Test timeouts | Slow external calls | Use mocks or increase timeout |
| Port conflicts | Multiple test servers | Use random ports |
| Cache pollution | Shared cache state | Use test isolation |
| Race conditions | Concurrent access | Use `-race` flag |

## 📚 Additional Resources

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Testify Documentation](https://pkg.go.dev/github.com/stretchr/testify)
- [Hexagonal Architecture Testing](https://herbertograca.com/2017/09/21/onion-architecture/)
- [Clean Architecture Testing Patterns](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

---

**Last Updated**: 2025-08-15  
**Version**: 1.0.0  
**Authors**: Test Designer (claude-opus)
