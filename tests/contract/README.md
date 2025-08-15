# 🤝 Contract Tests Guidelines

## Overview

Contract tests verify that **Ports** (interfaces) are correctly implemented by their **Adapters**, ensuring proper communication between layers in our Hexagonal + Clean Architecture.

## Test Scope

### Ports Layer (`internal/ports/`)
- **Interface Compliance**: All implementations follow interface contracts
- **Input/Output Validation**: Proper parameter and return value handling
- **Error Propagation**: Consistent error handling across implementations
- **Behavioral Contracts**: Expected behavior under various conditions

## Contract Testing Strategy

### Interface-Based Testing
Contract tests validate that any implementation of an interface behaves consistently, regardless of the underlying technology.

```go
// Example: All cache implementations must follow the same contract
func TestCacheContract(t *testing.T) {
    implementations := []ports.CacheManager{
        &filesystem.CacheManager{},
        &redis.CacheManager{},
        &s3.CacheManager{},
        &memory.CacheManager{},
    }

    for _, impl := range implementations {
        name := reflect.TypeOf(impl).Elem().Name()
        t.Run(name, func(t *testing.T) {
            testCacheContract(t, impl)
        })
    }
}
```

## Test Structure

### Port Interface Tests

```go
// +build contract

package ports_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "proxynd/internal/ports"
)

// Contract test for CacheManager interface
func testCacheContract(t *testing.T, cache ports.CacheManager) {
    ctx := context.Background()

    t.Run("basic_operations", func(t *testing.T) {
        testCacheBasicOperations(t, cache, ctx)
    })

    t.Run("error_handling", func(t *testing.T) {
        testCacheErrorHandling(t, cache, ctx)
    })

    t.Run("concurrency", func(t *testing.T) {
        testCacheConcurrency(t, cache, ctx)
    })
}

func testCacheBasicOperations(t *testing.T, cache ports.CacheManager, ctx context.Context) {
    // Test Set and Get operations
    key := "test-key"
    data := []byte("test data")

    // Set operation
    setReq := &ports.CacheSetRequest{
        Key:  key,
        Data: data,
        TTL:  time.Hour,
    }

    err := cache.Set(ctx, setReq)
    require.NoError(t, err, "Set operation should succeed")

    // Get operation
    getReq := &ports.CacheRequest{
        Key: key,
    }

    response, err := cache.Get(ctx, getReq)
    require.NoError(t, err, "Get operation should succeed")
    require.NotNil(t, response, "Response should not be nil")

    // Verify data integrity
    assert.Equal(t, data, response.Data, "Retrieved data should match stored data")
    assert.True(t, response.Hit, "Should indicate cache hit")
    assert.NotEmpty(t, response.Source, "Should specify cache source")
}

func testCacheErrorHandling(t *testing.T, cache ports.CacheManager, ctx context.Context) {
    // Test cache miss
    getReq := &ports.CacheRequest{
        Key: "non-existent-key",
    }

    response, err := cache.Get(ctx, getReq)

    // Should handle cache miss gracefully
    if err != nil {
        assert.ErrorIs(t, err, ports.ErrCacheMiss, "Should return cache miss error")
    } else {
        require.NotNil(t, response, "Response should not be nil")
        assert.False(t, response.Hit, "Should indicate cache miss")
    }

    // Test invalid key
    invalidReq := &ports.CacheRequest{
        Key: "", // Invalid empty key
    }

    _, err = cache.Get(ctx, invalidReq)
    assert.Error(t, err, "Should return error for invalid key")
}

func testCacheConcurrency(t *testing.T, cache ports.CacheManager, ctx context.Context) {
    const numGoroutines = 10
    const numOperations = 100

    // Test concurrent operations
    t.Run("concurrent_sets", func(t *testing.T) {
        done := make(chan bool, numGoroutines)

        for i := 0; i < numGoroutines; i++ {
            go func(id int) {
                defer func() { done <- true }()

                for j := 0; j < numOperations; j++ {
                    setReq := &ports.CacheSetRequest{
                        Key:  fmt.Sprintf("concurrent-key-%d-%d", id, j),
                        Data: []byte(fmt.Sprintf("data-%d-%d", id, j)),
                        TTL:  time.Minute,
                    }

                    err := cache.Set(ctx, setReq)
                    assert.NoError(t, err, "Concurrent set should succeed")
                }
            }(i)
        }

        // Wait for all goroutines to complete
        for i := 0; i < numGoroutines; i++ {
            select {
            case <-done:
                continue
            case <-time.After(30 * time.Second):
                t.Fatal("Concurrent test timed out")
            }
        }
    })
}
```

### Package Manager Contract Tests

```go
// Contract test for PackageManager interface
func testPackageManagerContract(t *testing.T, pm ports.PackageManager) {
    ctx := context.Background()

    t.Run("valid_package_request", func(t *testing.T) {
        req := &ports.PackageRequest{
            Type:    "npm",
            Package: "express",
            Version: "4.18.0",
        }

        response, err := pm.HandleRequest(ctx, req)

        // Contract requirements
        if err == nil {
            require.NotNil(t, response, "Response should not be nil on success")
            assert.NotEmpty(t, response.Data, "Response data should not be empty")
            assert.NotEmpty(t, response.ContentType, "Content type should be specified")

            // Optional fields
            if response.Etag != "" {
                assert.True(t, strings.HasPrefix(response.Etag, "\""), "Etag should be quoted")
            }
        } else {
            // Error cases should return specific error types
            assert.True(t,
                errors.Is(err, ports.ErrPackageNotFound) ||
                errors.Is(err, ports.ErrUpstreamUnavailable) ||
                errors.Is(err, ports.ErrInvalidRequest),
                "Should return known error type")
        }
    })

    t.Run("invalid_package_request", func(t *testing.T) {
        tests := []struct {
            name string
            req  *ports.PackageRequest
        }{
            {
                name: "empty package name",
                req: &ports.PackageRequest{
                    Type:    "npm",
                    Package: "",
                    Version: "1.0.0",
                },
            },
            {
                name: "invalid package type",
                req: &ports.PackageRequest{
                    Type:    "invalid",
                    Package: "express",
                    Version: "1.0.0",
                },
            },
        }

        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                _, err := pm.HandleRequest(ctx, tt.req)
                assert.Error(t, err, "Invalid request should return error")
                assert.ErrorIs(t, err, ports.ErrInvalidRequest, "Should return invalid request error")
            })
        }
    })

    t.Run("timeout_handling", func(t *testing.T) {
        // Test with very short timeout
        timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
        defer cancel()

        req := &ports.PackageRequest{
            Type:    "npm",
            Package: "express",
            Version: "4.18.0",
        }

        _, err := pm.HandleRequest(timeoutCtx, req)

        // Should handle timeout gracefully
        if err != nil {
            assert.True(t,
                errors.Is(err, context.DeadlineExceeded) ||
                errors.Is(err, ports.ErrUpstreamTimeout),
                "Should handle timeout appropriately")
        }
    })
}
```

### Authentication Contract Tests

```go
// Contract test for AuthProvider interface
func testAuthProviderContract(t *testing.T, auth ports.AuthProvider) {
    ctx := context.Background()

    t.Run("valid_credentials", func(t *testing.T) {
        // This would typically use test credentials
        creds := &ports.Credentials{
            Username: "test-user",
            Password: "test-password",
        }

        result, err := auth.Authenticate(ctx, creds)

        // Contract requirements
        if err == nil {
            require.NotNil(t, result, "Result should not be nil on success")
            assert.NotEmpty(t, result.UserID, "User ID should be provided")
            assert.NotEmpty(t, result.Roles, "Roles should be provided")
            assert.NotZero(t, result.ExpiresAt, "Expiration should be set")
        } else {
            // Known error types
            assert.True(t,
                errors.Is(err, ports.ErrInvalidCredentials) ||
                errors.Is(err, ports.ErrAuthProviderUnavailable),
                "Should return known error type")
        }
    })

    t.Run("invalid_credentials", func(t *testing.T) {
        creds := &ports.Credentials{
            Username: "invalid-user",
            Password: "wrong-password",
        }

        result, err := auth.Authenticate(ctx, creds)

        // Should reject invalid credentials
        assert.Error(t, err, "Invalid credentials should be rejected")
        assert.ErrorIs(t, err, ports.ErrInvalidCredentials, "Should return invalid credentials error")
        assert.Nil(t, result, "Result should be nil on failure")
    })
}
```

## Contract Test Patterns

### 1. Implementation Discovery Pattern

```go
// Automatically discover and test all implementations
func TestAllCacheImplementations(t *testing.T) {
    implementations := discoverCacheImplementations()

    for _, impl := range implementations {
        name := getImplementationName(impl)
        t.Run(name, func(t *testing.T) {
            // Setup implementation with test configuration
            testImpl := setupImplementation(t, impl)
            defer teardownImplementation(t, testImpl)

            // Run contract tests
            testCacheContract(t, testImpl)
        })
    }
}

func discoverCacheImplementations() []ports.CacheManager {
    return []ports.CacheManager{
        setupFilesystemCache(),
        setupRedisCache(),
        setupS3Cache(),
        setupMemoryCache(),
    }
}
```

### 2. Contract Compliance Matrix

```go
// Test matrix for all combinations
func TestPackageManagerContracts(t *testing.T) {
    packageTypes := []string{"npm", "pip", "apt", "docker", "maven"}
    implementations := discoverPackageManagerImplementations()

    for _, impl := range implementations {
        implName := getImplementationName(impl)
        t.Run(implName, func(t *testing.T) {
            for _, pkgType := range packageTypes {
                if impl.Supports(pkgType) {
                    t.Run(pkgType, func(t *testing.T) {
                        testPackageManagerForType(t, impl, pkgType)
                    })
                }
            }
        })
    }
}
```

### 3. Error Contract Testing

```go
// Verify consistent error handling across implementations
func testErrorContract(t *testing.T, service ports.Service) {
    errorScenarios := []struct {
        name          string
        setup         func() error
        expectedError error
    }{
        {
            name:          "network_unavailable",
            setup:         simulateNetworkFailure,
            expectedError: ports.ErrNetworkUnavailable,
        },
        {
            name:          "timeout",
            setup:         simulateTimeout,
            expectedError: ports.ErrTimeout,
        },
        {
            name:          "invalid_input",
            setup:         provideInvalidInput,
            expectedError: ports.ErrInvalidInput,
        },
    }

    for _, scenario := range errorScenarios {
        t.Run(scenario.name, func(t *testing.T) {
            // Setup error condition
            err := scenario.setup()
            require.NoError(t, err, "Error setup should succeed")

            // Execute operation that should fail
            _, err = service.Execute(context.Background())

            // Verify error contract
            assert.Error(t, err, "Operation should fail")
            assert.ErrorIs(t, err, scenario.expectedError, "Should return expected error type")
        })
    }
}
```

## Test Configuration

### Build Tags
```go
// +build contract

// This ensures contract tests only run when specifically requested
package ports_test
```

### Test Setup Utilities

```go
// tests/contract/testutil/setup.go
package testutil

import (
    "testing"
    "proxynd/internal/adapters/cache/filesystem"
    "proxynd/internal/adapters/cache/redis"
    "proxynd/internal/ports"
)

// SetupTestCache creates a test cache implementation
func SetupTestCache(t *testing.T, cacheType string) ports.CacheManager {
    t.Helper()

    switch cacheType {
    case "filesystem":
        cache, err := filesystem.NewCacheManager(&filesystem.Config{
            Directory: t.TempDir(),
            MaxSize:   "100MB",
        })
        require.NoError(t, err)
        return cache

    case "redis":
        // Only if Redis test server is available
        if !isRedisAvailable() {
            t.Skip("Redis not available for testing")
        }

        cache, err := redis.NewCacheManager(&redis.Config{
            Address:  "localhost:6379",
            Database: 15, // Use test database
        })
        require.NoError(t, err)
        return cache

    default:
        t.Fatalf("Unknown cache type: %s", cacheType)
        return nil
    }
}

// CleanupTestCache cleans up test cache resources
func CleanupTestCache(t *testing.T, cache ports.CacheManager) {
    t.Helper()

    if cleaner, ok := cache.(interface{ Cleanup() error }); ok {
        err := cleaner.Cleanup()
        assert.NoError(t, err, "Cache cleanup should succeed")
    }
}
```

### Mock Implementations for Testing

```go
// tests/contract/mocks/cache_mock.go
package mocks

import (
    "context"
    "sync"
    "time"
    "proxynd/internal/ports"
)

// MockCacheManager provides a simple in-memory implementation for testing
type MockCacheManager struct {
    data  map[string]cacheEntry
    mutex sync.RWMutex
}

type cacheEntry struct {
    data      []byte
    expiresAt time.Time
}

func NewMockCacheManager() *MockCacheManager {
    return &MockCacheManager{
        data: make(map[string]cacheEntry),
    }
}

func (m *MockCacheManager) Get(ctx context.Context, req *ports.CacheRequest) (*ports.CacheResponse, error) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()

    entry, exists := m.data[req.Key]
    if !exists || time.Now().After(entry.expiresAt) {
        return nil, ports.ErrCacheMiss
    }

    return &ports.CacheResponse{
        Data:   entry.data,
        Hit:    true,
        Source: "mock-cache",
    }, nil
}

func (m *MockCacheManager) Set(ctx context.Context, req *ports.CacheSetRequest) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    m.data[req.Key] = cacheEntry{
        data:      req.Data,
        expiresAt: time.Now().Add(req.TTL),
    }

    return nil
}

func (m *MockCacheManager) Invalidate(ctx context.Context, pattern string) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    // Simple pattern matching for tests
    for key := range m.data {
        if strings.Contains(key, pattern) {
            delete(m.data, key)
        }
    }

    return nil
}
```

## Running Contract Tests

### Command Reference
```bash
# Run all contract tests
make test-contract

# Run contract tests with coverage
make test-coverage-contract

# Run specific contract tests
go test -tags=contract ./internal/ports/...

# Run with verbose output
go test -tags=contract -v ./internal/ports/...

# Run specific implementation
go test -tags=contract -run TestCacheContract/filesystem ./internal/ports/...
```

### Test Environment Variables
```bash
# Enable external service testing
export ENABLE_REDIS_TESTS=true
export ENABLE_S3_TESTS=true

# Test service endpoints
export REDIS_TEST_URL=localhost:6379
export S3_TEST_ENDPOINT=localhost:9000
```

## Quality Gates

### Contract Compliance Requirements
- **Interface Coverage**: 100% of interface methods tested
- **Implementation Coverage**: All production implementations tested
- **Error Scenarios**: All documented error conditions tested
- **Consistency**: Identical behavior across implementations

### Performance Contracts
```go
func TestCachePerformanceContract(t *testing.T, cache ports.CacheManager) {
    // Performance requirements
    const maxLatency = 10 * time.Millisecond
    const minThroughput = 1000 // operations per second

    t.Run("latency_requirement", func(t *testing.T) {
        start := time.Now()

        _, err := cache.Get(context.Background(), &ports.CacheRequest{
            Key: "test-key",
        })

        latency := time.Since(start)

        // Allow for cache miss, but latency should still be reasonable
        if err == nil || errors.Is(err, ports.ErrCacheMiss) {
            assert.Less(t, latency, maxLatency, "Cache operation should be fast")
        }
    })
}
```

## Best Practices

### ✅ Do
- Test all interface methods
- Verify error contracts
- Test with realistic data sizes
- Include concurrency tests
- Use build tags to separate contract tests
- Document expected behavior
- Test implementation-agnostic behavior

### ❌ Don't
- Test implementation details
- Hardcode environment-specific values
- Skip error condition testing
- Assume implementation availability
- Create dependencies between contract tests
- Test beyond interface boundaries

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Tests fail on CI | Missing test dependencies | Use conditional test skipping |
| Inconsistent behavior | Implementation differences | Strengthen contract requirements |
| Slow contract tests | Heavy setup/teardown | Use shared test fixtures |
| Flaky tests | Race conditions | Add proper synchronization |

### Debug Tips
```bash
# Run specific implementation
go test -tags=contract -run TestCacheContract/redis

# Enable debug logging
PROXYND_LOG_LEVEL=debug go test -tags=contract

# Run with race detector
go test -tags=contract -race ./internal/ports/...

# Profile contract tests
go test -tags=contract -cpuprofile=contract.prof
```

---

**Last Updated**: 2025-08-15  
**Version**: 1.0.0  
**Authors**: Test Designer (claude-opus)
