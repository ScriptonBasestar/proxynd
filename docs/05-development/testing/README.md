# ProxyND Testing Guide

## Overview

This guide describes the testing strategy and implementation for ProxyND services. We use table-driven tests with the testify framework to ensure comprehensive coverage and maintainability.

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

### Test Categories

- **[Unit Tests](unit-testing.md)** - Fast tests for individual components
- **[Integration Tests](integration-testing.md)** - Tests for component interactions  
- **[E2E Tests](e2e.md)** - End-to-end proxy functionality tests

### Development Workflow

1. **Write tests first** (TDD approach)
2. **Run unit tests** during development: `go test ./internal/services/...`
3. **Check coverage** before committing: `make test-coverage`
4. **Run full suite** before pushing: `make test-all`

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

## Test Coverage Goals

- **Target**: 80% overall coverage
- **Critical Paths**: 95% coverage for:
  - Configuration loading and validation
  - Cache operations
  - Proxy request handling
  - Error paths

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
