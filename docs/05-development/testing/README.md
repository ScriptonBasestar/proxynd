# 🧪 ProxyND Testing Overview

ProxyND의 종합적인 테스트 전략과 구현에 대한 모든 정보는 **[종합 테스트 가이드](./00-testing-guide.md)**에서 확인할 수 있습니다.

## 📚 문서 구조

- **[00-testing-guide.md](./00-testing-guide.md)** - 완전한 테스트 가이드 (단위/통합/계약/E2E 테스트)
- **[performance-testing.md](./performance-testing.md)** - 성능 테스트 전용 가이드
- **[test-utilities.md](./test-utilities.md)** - 테스트 유틸리티 및 헬퍼
- **[troubleshooting-guide.md](./troubleshooting-guide.md)** - 테스트 문제 해결

## Quick Start

### Prerequisites

```bash
# Install Go 1.21 or later
go version

# Install test dependencies
go mod download

# Install development tools (optional)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Running Tests

```bash
# Quick test run (unit tests only)
make test-unit

# Full test suite with coverage
make test-coverage

# Integration tests
make test-integration

# All tests (unit + integration + e2e)
make test-all

# Run with race detection
make test-race
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

### Test Categories

- **[Unit Tests](unit-testing.md)** - Fast tests for individual components
- **[Contract Tests](integration-testing.md)** - Interface contract verification
- **[Integration Tests](integration-testing.md)** - Tests for component interactions  
- **[E2E Tests](e2e.md)** - End-to-end proxy functionality tests

### Development Workflow

1. **Write tests first** (TDD approach)
2. **Run unit tests** during development: `go test ./internal/services/...`
3. **Check coverage** before committing: `make test-coverage`
4. **Run full suite** before pushing: `make test-all`

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

## Test Structure

### Unit Tests

Each service has its corresponding test file following the pattern `*_test.go`:

```
internal/services/
├── config/
│   ├── service.go
│   └── service_test.go
├── proxy/
│   ├── base_service.go
│   ├── base_service_test.go
│   ├── apt_service.go
│   ├── apt_service_test.go
│   ├── factory.go
│   └── factory_test.go
└── adapters/
    ├── cache_adapter.go
    ├── cache_adapter_test.go
    ├── upstream_client.go
    └── upstream_client_test.go
```

## Testing Patterns

### 1. Table-Driven Tests

We use table-driven tests for comprehensive scenario coverage:

```go
func TestService_Method(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "result",
            wantErr: false,
        },
        {
            name:    "invalid input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 2. Mocking

We use testify/mock for interface mocking:

```go
type MockCacheService struct {
    mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) (io.ReadCloser, bool, error) {
    args := m.Called(ctx, key)
    return args.Get(0).(io.ReadCloser), args.Bool(1), args.Error(2)
}
```

### 3. Test Helpers

Common test utilities:

```go
func createTestConfig(t *testing.T, dir, filename, content string) {
    path := filepath.Join(dir, filename)
    err := os.WriteFile(path, []byte(content), 0644)
    require.NoError(t, err)
}
```

## Test Categories

### 1. Configuration Service Tests

- **Service Creation**: Tests service initialization with various config states
- **Config Retrieval**: Tests for each config type (global, apt, maven, etc.)
- **Validation**: Tests configuration validation logic
- **Reload**: Tests configuration reload functionality
- **Concurrency**: Tests concurrent access to configurations

### 2. Proxy Service Tests

- **Base Service**: Core proxy functionality tests
- **Service Factory**: Tests creation of different proxy types
- **Request Validation**: Tests request validation logic
- **Cache Integration**: Tests cache hit/miss scenarios
- **Error Handling**: Tests error scenarios and responses

### 3. Adapter Tests

- **Cache Adapter**: Tests cache operations (get, put, exists, delete)
- **Upstream Client**: Tests HTTP client functionality
- **Large Content**: Tests handling of large files
- **Timeouts**: Tests timeout scenarios
- **Context Cancellation**: Tests proper context handling

## Running Tests

### Run All Service Tests

```bash
./scripts/run_service_tests.sh
```

### Run Specific Package Tests

```bash
# Config service tests
go test -v -cover ./internal/services/config

# Proxy service tests
go test -v -cover ./internal/services/proxy

# Adapter tests
go test -v -cover ./internal/services/adapters
```

### Run with Race Detection

```bash
go test -race ./internal/services/...
```

### Generate Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./internal/services/...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

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

## Benchmarking

Run benchmark tests:

```bash
go test -bench=. -benchmem ./internal/services/...
```

Example benchmark:

```go
func BenchmarkHTTPUpstreamClient_Fetch(b *testing.B) {
    // Benchmark implementation
}
```

## Integration Tests

While unit tests cover individual components, integration tests verify end-to-end flows:

```bash
cd tests/integration
go test -v ./...
```

## End-to-End (E2E) Testing

ProxyND provides comprehensive E2E testing for all supported proxy types. Each proxy type has dedicated test guides with detailed scenarios:

### Available E2E Test Guides

- **[General E2E Testing](e2e.md)** - Overview of the E2E test environment and architecture
- **[Maven Proxy E2E Testing](e2e-maven.md)** - Maven repository proxy testing with artifacts, POMs, and checksums  
- **[NPM Proxy E2E Testing](e2e-npm.md)** - NPM registry proxy testing with packages, tarballs, and scoped packages
- **[APT Proxy E2E Testing](e2e-apt.md)** - APT repository proxy testing with Release files, packages, and .deb files
- **[Docker Proxy E2E Testing](e2e-docker.md)** - Docker registry proxy testing *(coming soon)*
- **[PyPI Proxy E2E Testing](e2e-pip.md)** - PyPI proxy testing *(coming soon)*
- **[YUM Proxy E2E Testing](e2e-yum.md)** - YUM repository proxy testing *(coming soon)*
- **[APK Proxy E2E Testing](e2e-apk.md)** - APK repository proxy testing *(coming soon)*

### Quick E2E Test Commands

```bash
# Run all E2E tests
cd tests/e2e && make test-all

# Run specific proxy type tests
make test-maven    # Maven proxy tests
make test-npm      # NPM proxy tests  
make test-apt      # APT proxy tests
make test-docker   # Docker proxy tests
make test-pip      # PyPI proxy tests
```

### E2E Test Integration with CI

The E2E tests are automatically executed in the CI/CD pipeline using a matrix strategy that tests each proxy type independently:

- **Integration Tests Matrix**: Tests each proxy type's integration scenarios
- **E2E Tests Matrix**: Tests each proxy type with real client tools
- **Parallel Execution**: All proxy types tested simultaneously for faster feedback
- **Artifact Collection**: Test results and logs collected for debugging

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

### Test Execution & Makefile Integration

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

## Continuous Integration

Tests are automatically run on:
- Pull requests
- Commits to main branch
- Nightly builds

CI checks include:
- Unit tests
- Race condition detection
- Coverage thresholds
- Linting

## Best Practices

1. **Test Naming**: Use descriptive test names that explain the scenario
2. **Error Messages**: Include context in test failure messages
3. **Test Data**: Use meaningful test data, not just "test" strings
4. **Cleanup**: Always clean up resources (files, connections) after tests
5. **Parallel Tests**: Use `t.Parallel()` where appropriate
6. **Mock Assertions**: Always assert mock expectations

## Debugging Tests

### Verbose Output

```bash
go test -v ./internal/services/config
```

### Run Single Test

```bash
go test -run TestService_GetGlobalConfig ./internal/services/config
```

### Debug with Delve

```bash
dlv test ./internal/services/config -- -test.run TestService_GetGlobalConfig
```

## Common Issues

### 1. File System Tests

When testing file operations, always use `t.TempDir()`:

```go
tempDir := t.TempDir() // Automatically cleaned up
```

### 2. Context Timeouts

For timeout tests, use short durations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
defer cancel()
```

### 3. Race Conditions

Always test concurrent code with race detection:

```bash
go test -race ./...
```

## Adding New Tests

When adding new functionality:

1. Write tests first (TDD approach)
2. Cover happy path and error cases
3. Add benchmark if performance-critical
4. Update coverage goals if needed
5. Document any special test requirements

## Test Maintenance

- Review and update tests when changing functionality
- Remove obsolete tests
- Keep tests simple and focused
- Refactor common test code into helpers
