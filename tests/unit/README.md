# 🧪 Unit Tests Guidelines

## Overview

Unit tests validate individual components in complete isolation, focusing on the **Domain** and **Usecase** layers of our Hexagonal + Clean Architecture.

## Test Scope

### Domain Layer (`internal/domain/`)
- **Entities**: Business objects and their behavior
- **Value Objects**: Immutable data structures  
- **Domain Services**: Pure business logic
- **Validation Rules**: Business rule enforcement

### Usecase Layer (`internal/usecase/`)
- **Application Services**: Orchestration logic
- **Business Workflows**: Multi-step processes
- **Use Case Implementations**: Application-specific logic

## Architecture Rules

### ✅ Allowed Dependencies
- **Domain tests**: No external dependencies (pure functions)
- **Usecase tests**: Mock ports only, never adapters

### ❌ Forbidden Dependencies
- File system access
- Network calls
- Database connections
- External services
- Framework-specific code

## Test Characteristics

| Aspect | Requirement | Rationale |
|--------|-------------|-----------|
| **Speed** | < 1ms per test | Fast feedback loop |
| **Isolation** | No shared state | Reliable, repeatable results |
| **Coverage** | 90%+ for Domain, 85%+ for Usecase | High confidence in business logic |
| **Mocking** | Ports only (Usecase layer) | Maintain architecture boundaries |

## Test Structure

### Domain Tests

```go
// internal/domain/npm/models_test.go
package npm

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestPackageVersion_Validate(t *testing.T) {
    tests := []struct {
        name     string
        version  string
        expected bool
        errorMsg string
    }{
        {
            name:     "valid semver",
            version:  "1.2.3",
            expected: true,
        },
        {
            name:     "invalid format",
            version:  "invalid-version",
            expected: false,
            errorMsg: "invalid semantic version format",
        },
        {
            name:     "pre-release version",
            version:  "1.2.3-alpha.1",
            expected: true,
        },
        {
            name:     "build metadata",
            version:  "1.2.3+build.1",
            expected: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            pkg := &Package{Version: tt.version}

            // Act
            err := pkg.ValidateVersion()

            // Assert
            if tt.expected {
                assert.NoError(t, err)
            } else {
                assert.Error(t, err)
                if tt.errorMsg != "" {
                    assert.Contains(t, err.Error(), tt.errorMsg)
                }
            }
        })
    }
}

func TestPackage_CalculateIntegrity(t *testing.T) {
    // Arrange
    content := []byte("package content")
    pkg := &Package{
        Name:    "test-package",
        Version: "1.0.0",
    }

    // Act
    integrity := pkg.CalculateIntegrity(content)

    // Assert
    assert.NotEmpty(t, integrity)
    assert.True(t, strings.HasPrefix(integrity, "sha512-"))

    // Verify consistency
    integrity2 := pkg.CalculateIntegrity(content)
    assert.Equal(t, integrity, integrity2)
}
```

### Usecase Tests

```go
// internal/usecase/proxy_service_test.go
package usecase

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "proxynd/internal/ports"
)

// Mock implementations
type MockPackageManager struct {
    mock.Mock
}

func (m *MockPackageManager) HandleRequest(ctx context.Context, req *ports.PackageRequest) (*ports.PackageResponse, error) {
    args := m.Called(ctx, req)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*ports.PackageResponse), args.Error(1)
}

type MockCacheManager struct {
    mock.Mock
}

func (m *MockCacheManager) Get(ctx context.Context, req *ports.CacheRequest) (*ports.CacheResponse, error) {
    args := m.Called(ctx, req)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*ports.CacheResponse), args.Error(1)
}

func (m *MockCacheManager) Set(ctx context.Context, req *ports.CacheSetRequest) error {
    args := m.Called(ctx, req)
    return args.Error(0)
}

func TestProxyService_CacheHit(t *testing.T) {
    // Arrange
    mockCache := &MockCacheManager{}
    mockPM := &MockPackageManager{}

    service := NewProxyService(mockCache, mockPM)

    cacheResponse := &ports.CacheResponse{
        Data:   []byte("cached content"),
        Hit:    true,
        Source: "file-cache",
    }

    mockCache.On("Get", mock.Anything, mock.MatchedBy(func(req *ports.CacheRequest) bool {
        return req.Key == "npm:express:4.18.0"
    })).Return(cacheResponse, nil)

    req := &ports.PackageRequest{
        Type:    "npm",
        Package: "express",
        Version: "4.18.0",
    }

    // Act
    response, err := service.HandleRequest(context.Background(), req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.True(t, response.CacheHit)
    assert.Equal(t, "cached content", string(response.Data))
    assert.Equal(t, "file-cache", response.CacheSource)

    // Verify mocks
    mockCache.AssertExpectations(t)
    mockPM.AssertNotCalled(t, "HandleRequest") // Should not call upstream on cache hit
}

func TestProxyService_CacheMiss(t *testing.T) {
    // Arrange
    mockCache := &MockCacheManager{}
    mockPM := &MockPackageManager{}

    service := NewProxyService(mockCache, mockPM)

    // Cache miss
    mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, ports.ErrCacheMiss)

    // Upstream response
    upstreamResponse := &ports.PackageResponse{
        Data:        []byte("upstream content"),
        ContentType: "application/json",
        Etag:        "\"abc123\"",
    }
    mockPM.On("HandleRequest", mock.Anything, mock.Anything).Return(upstreamResponse, nil)

    // Cache set
    mockCache.On("Set", mock.Anything, mock.MatchedBy(func(req *ports.CacheSetRequest) bool {
        return req.Key == "npm:express:4.18.0" && string(req.Data) == "upstream content"
    })).Return(nil)

    req := &ports.PackageRequest{
        Type:    "npm",
        Package: "express",
        Version: "4.18.0",
    }

    // Act
    response, err := service.HandleRequest(context.Background(), req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.False(t, response.CacheHit)
    assert.Equal(t, "upstream content", string(response.Data))
    assert.Equal(t, "application/json", response.ContentType)

    // Verify all interactions
    mockCache.AssertExpectations(t)
    mockPM.AssertExpectations(t)
}
```

## Naming Conventions

### Test Function Names
```go
// Format: Test[Component]_[Scenario]_[ExpectedResult]
func TestPackage_ValidVersion_ReturnsNoError(t *testing.T) {}
func TestPackage_InvalidVersion_ReturnsValidationError(t *testing.T) {}
func TestProxyService_CacheHit_SkipsUpstream(t *testing.T) {}
func TestProxyService_UpstreamError_ReturnsError(t *testing.T) {}
```

### Test Data Builders
```go
// Helper functions for creating test data
func newTestPackage() *Package {
    return &Package{
        Name:    "test-package",
        Version: "1.0.0",
        Type:    "npm",
    }
}

func newTestPackageRequest(packageType, name, version string) *ports.PackageRequest {
    return &ports.PackageRequest{
        Type:    packageType,
        Package: name,
        Version: version,
    }
}
```

## Testing Patterns

### Table-Driven Tests
```go
func TestPackageValidation(t *testing.T) {
    tests := []struct {
        name        string
        package     Package
        shouldError bool
        errorType   error
    }{
        {
            name: "valid package",
            package: Package{
                Name:    "express",
                Version: "4.18.0",
                Type:    "npm",
            },
            shouldError: false,
        },
        {
            name: "missing name",
            package: Package{
                Name:    "",
                Version: "1.0.0",
                Type:    "npm",
            },
            shouldError: true,
            errorType:   ErrInvalidPackageName,
        },
        // Add more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.package.Validate()

            if tt.shouldError {
                assert.Error(t, err)
                if tt.errorType != nil {
                    assert.ErrorIs(t, err, tt.errorType)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Subtest Organization
```go
func TestPackageManager(t *testing.T) {
    t.Run("validation", func(t *testing.T) {
        t.Run("valid package", func(t *testing.T) {
            // Test valid package validation
        })

        t.Run("invalid package", func(t *testing.T) {
            // Test invalid package validation
        })
    })

    t.Run("caching", func(t *testing.T) {
        t.Run("cache hit", func(t *testing.T) {
            // Test cache hit scenario
        })

        t.Run("cache miss", func(t *testing.T) {
            // Test cache miss scenario
        })
    })
}
```

## Mock Management

### Mock Creation Guidelines
```go
// Use testify/mock for complex interactions
type MockPackageManager struct {
    mock.Mock
}

// Implement interface methods
func (m *MockPackageManager) HandleRequest(ctx context.Context, req *ports.PackageRequest) (*ports.PackageResponse, error) {
    args := m.Called(ctx, req)
    return args.Get(0).(*ports.PackageResponse), args.Error(1)
}

// Test setup
func TestWithMocks(t *testing.T) {
    // Arrange
    mockPM := &MockPackageManager{}

    // Set expectations
    mockPM.On("HandleRequest",
        mock.Anything,
        mock.MatchedBy(func(req *ports.PackageRequest) bool {
            return req.Package == "express"
        })).Return(&ports.PackageResponse{
            Data: []byte("mocked response"),
        }, nil)

    // Act
    service := NewService(mockPM)
    result, err := service.DoSomething()

    // Assert
    assert.NoError(t, err)
    mockPM.AssertExpectations(t)
}
```

### Mock Verification
```go
// Verify specific calls were made
mockCache.AssertCalled(t, "Get", mock.Anything, "cache-key")

// Verify calls were NOT made
mockUpstream.AssertNotCalled(t, "FetchPackage")

// Verify number of calls
mockCache.AssertNumberOfCalls(t, "Set", 1)

// Verify all expectations met
mockCache.AssertExpectations(t)
```

## Error Testing

### Error Scenarios
```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name          string
        setupMocks    func(*MockCacheManager, *MockPackageManager)
        expectedError error
        errorMessage  string
    }{
        {
            name: "cache error",
            setupMocks: func(cache *MockCacheManager, pm *MockPackageManager) {
                cache.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("cache error"))
                pm.On("HandleRequest", mock.Anything, mock.Anything).Return(nil, errors.New("upstream error"))
            },
            expectedError: ErrUpstreamUnavailable,
            errorMessage:  "upstream error",
        },
        // More error scenarios...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockCache := &MockCacheManager{}
            mockPM := &MockPackageManager{}
            tt.setupMocks(mockCache, mockPM)

            service := NewProxyService(mockCache, mockPM)

            // Act
            _, err := service.HandleRequest(context.Background(), &ports.PackageRequest{})

            // Assert
            assert.Error(t, err)
            if tt.expectedError != nil {
                assert.ErrorIs(t, err, tt.expectedError)
            }
            if tt.errorMessage != "" {
                assert.Contains(t, err.Error(), tt.errorMessage)
            }
        })
    }
}
```

## Performance Testing

### Benchmark Tests
```go
func BenchmarkPackageValidation(b *testing.B) {
    pkg := &Package{
        Name:    "test-package",
        Version: "1.0.0",
        Type:    "npm",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = pkg.Validate()
    }
}

func BenchmarkIntegrityCalculation(b *testing.B) {
    content := make([]byte, 1024*1024) // 1MB
    pkg := &Package{Name: "test"}

    b.ResetTimer()
    b.SetBytes(int64(len(content)))

    for i := 0; i < b.N; i++ {
        _ = pkg.CalculateIntegrity(content)
    }
}
```

## Test Utilities

### Common Assertions
```go
// tests/unit/testutil/assertions.go
package testutil

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func AssertValidPackage(t *testing.T, pkg *Package) {
    t.Helper()
    assert.NotEmpty(t, pkg.Name, "Package name should not be empty")
    assert.NotEmpty(t, pkg.Version, "Package version should not be empty")
    assert.NotEmpty(t, pkg.Type, "Package type should not be empty")
    assert.NoError(t, pkg.Validate(), "Package should be valid")
}

func AssertErrorType(t *testing.T, err error, expectedType error) {
    t.Helper()
    assert.Error(t, err, "Expected an error")
    assert.ErrorIs(t, err, expectedType, "Error should be of expected type")
}
```

### Test Data Factories
```go
// tests/unit/testutil/factories.go
package testutil

import "proxynd/internal/domain/npm"

func NewNpmPackage(name, version string) *npm.Package {
    return &npm.Package{
        Name:    name,
        Version: version,
        Type:    "npm",
    }
}

func NewValidPackageRequest() *ports.PackageRequest {
    return &ports.PackageRequest{
        Type:    "npm",
        Package: "express",
        Version: "4.18.0",
    }
}
```

## Running Unit Tests

### Command Reference
```bash
# Run all unit tests
make test-unit

# Run with coverage
make test-coverage-unit

# Run specific package
go test ./internal/domain/npm/...

# Run with verbose output
go test -v ./internal/domain/...

# Run specific test
go test -run TestPackage_Validate ./internal/domain/npm/

# Run benchmarks
go test -bench=. ./internal/domain/...

# Check race conditions
go test -race ./internal/domain/... ./internal/usecase/...
```

### Coverage Requirements
- **Domain Layer**: 95%+ coverage
- **Usecase Layer**: 90%+ coverage
- **Critical Paths**: 100% coverage (validation, security)

## Best Practices

### ✅ Do
- Test one thing at a time
- Use descriptive test names
- Follow AAA pattern (Arrange, Act, Assert)
- Test error conditions
- Use table-driven tests for multiple scenarios
- Mock at port boundaries only
- Verify mock expectations
- Use helper functions for common setup

### ❌ Don't
- Test implementation details
- Use real external dependencies
- Share state between tests
- Write tests that depend on execution order
- Mock everything (domain objects should be real)
- Ignore error returns
- Use global variables
- Write overly complex test setup

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Test flakiness | Shared state | Use fresh instances per test |
| Slow tests | Real dependencies | Use mocks at port boundaries |
| Mock complexity | Too much mocking | Mock only at architecture boundaries |
| Poor coverage | Missing edge cases | Add table-driven tests |

### Debug Tips
```bash
# Run with verbose output
go test -v ./path/to/test

# Run specific test
go test -run TestSpecificFunction

# Debug with Delve
dlv test -- -test.run TestSpecificFunction

# Print test coverage
go test -cover ./...

# Generate coverage profile
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

**Last Updated**: 2025-08-15  
**Version**: 1.0.0  
**Authors**: Test Designer (claude-opus)
