# 🔗 Integration Tests Guidelines

## Overview

Integration tests validate the **Adapters layer** and its interactions with external systems, ensuring that our components work correctly when integrated with real dependencies like databases, file systems, networks, and third-party services.

## Test Scope

### Adapters Layer (`internal/adapters/`)
- **HTTP Adapters**: Fiber handlers and middleware
- **Cache Adapters**: File system, Redis, S3 implementations  
- **External Service Adapters**: Package registry clients
- **Database Adapters**: Configuration and data persistence
- **Authentication Adapters**: OAuth2, LDAP, JWT providers

## Integration Test Categories

### 1. Component Integration Tests
Test individual adapters with their external dependencies.

### 2. Service Integration Tests  
Test interactions between multiple components.

### 3. Cross-Package Manager Tests
Test scenarios spanning multiple package managers.

### 4. Performance Integration Tests
Test system behavior under load and stress conditions.

## Test Structure

### HTTP Handler Integration Tests

```go
// +build integration

package integration

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNPMHandler_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Integration tests skipped in short mode")
    }

    // Setup real test server
    server := SetupIntegrationTestServer(t)
    defer server.Cleanup()

    tests := []struct {
        name           string
        method         string
        path           string
        expectedStatus int
        expectedHeaders map[string]string
        validateBody   func(*testing.T, []byte)
        timeout        time.Duration
    }{
        {
            name:           "npm_package_metadata",
            method:         "GET",
            path:           "/proxy/npm/express",
            expectedStatus: http.StatusOK,
            expectedHeaders: map[string]string{
                "Content-Type": "application/json",
                "X-Cache":      "MISS", // First request should be cache miss
            },
            validateBody: func(t *testing.T, body []byte) {
                var pkg map[string]interface{}
                err := json.Unmarshal(body, &pkg)
                require.NoError(t, err)

                assert.Equal(t, "express", pkg["name"])
                assert.NotEmpty(t, pkg["version"])
                assert.NotEmpty(t, pkg["description"])
            },
            timeout: 30 * time.Second,
        },
        {
            name:           "npm_package_tarball",
            method:         "GET",
            path:           "/proxy/npm/express/-/express-4.18.0.tgz",
            expectedStatus: http.StatusOK,
            expectedHeaders: map[string]string{
                "Content-Type": "application/gzip",
            },
            validateBody: func(t *testing.T, body []byte) {
                assert.Greater(t, len(body), 1000, "Tarball should be substantial")
                // Verify gzip header
                assert.Equal(t, []byte{0x1f, 0x8b}, body[:2], "Should have gzip magic number")
            },
            timeout: 60 * time.Second,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup request with timeout
            ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
            defer cancel()

            req := httptest.NewRequestWithContext(ctx, tt.method, tt.path, nil)

            // Execute request
            resp, err := server.App.Test(req, -1) // No timeout for fiber test
            require.NoError(t, err)
            defer resp.Body.Close()

            // Verify status code
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)

            // Verify headers
            for header, expectedValue := range tt.expectedHeaders {
                assert.Equal(t, expectedValue, resp.Header.Get(header))
            }

            // Verify body if validator provided
            if tt.validateBody != nil {
                body, err := io.ReadAll(resp.Body)
                require.NoError(t, err)
                tt.validateBody(t, body)
            }
        })
    }
}
```

### Cache Adapter Integration Tests

```go
func TestCacheAdapter_FileSystem(t *testing.T) {
    if testing.Short() {
        t.Skip("Integration tests skipped in short mode")
    }

    // Setup temporary directory
    tempDir := t.TempDir()

    // Configure filesystem cache
    config := &filesystem.Config{
        Directory:   tempDir,
        MaxSize:     "100MB",
        MaxItems:    1000,
        TTL:         time.Hour,
    }

    cache, err := filesystem.NewCacheManager(config)
    require.NoError(t, err)

    t.Run("cache_lifecycle", func(t *testing.T) {
        testCacheLifecycle(t, cache)
    })

    t.Run("cache_persistence", func(t *testing.T) {
        testCachePersistence(t, cache, tempDir)
    })

    t.Run("cache_eviction", func(t *testing.T) {
        testCacheEviction(t, cache)
    })
}

func testCacheLifecycle(t *testing.T, cache ports.CacheManager) {
    ctx := context.Background()

    // Test data
    key := "integration-test-key"
    data := []byte("integration test data")

    // Set cache entry
    setReq := &ports.CacheSetRequest{
        Key:  key,
        Data: data,
        TTL:  time.Minute,
    }

    err := cache.Set(ctx, setReq)
    require.NoError(t, err)

    // Get cache entry
    getReq := &ports.CacheRequest{
        Key: key,
    }

    response, err := cache.Get(ctx, getReq)
    require.NoError(t, err)
    assert.Equal(t, data, response.Data)
    assert.True(t, response.Hit)

    // Test TTL expiration
    t.Run("ttl_expiration", func(t *testing.T) {
        // Set entry with very short TTL
        shortTTLReq := &ports.CacheSetRequest{
            Key:  "short-ttl-key",
            Data: []byte("short ttl data"),
            TTL:  100 * time.Millisecond,
        }

        err := cache.Set(ctx, shortTTLReq)
        require.NoError(t, err)

        // Wait for expiration
        time.Sleep(200 * time.Millisecond)

        // Should miss after expiration
        expiredReq := &ports.CacheRequest{
            Key: "short-ttl-key",
        }

        _, err = cache.Get(ctx, expiredReq)
        assert.ErrorIs(t, err, ports.ErrCacheMiss)
    })
}
```

### Package Manager Adapter Integration Tests

```go
func TestNPMAdapter_RealRegistry(t *testing.T) {
    if testing.Short() {
        t.Skip("Integration tests skipped in short mode")
    }

    // Skip if no network connectivity
    if !hasNetworkConnectivity() {
        t.Skip("Network connectivity required for integration tests")
    }

    // Setup NPM adapter with real registry
    config := &npm.Config{
        RegistryURL:    "https://registry.npmjs.org",
        Timeout:        30 * time.Second,
        MaxRetries:     3,
        RetryDelay:     time.Second,
    }

    adapter, err := npm.NewAdapter(config)
    require.NoError(t, err)

    t.Run("popular_package_metadata", func(t *testing.T) {
        testPackageMetadata(t, adapter, "express", "4.18.0")
    })

    t.Run("scoped_package_metadata", func(t *testing.T) {
        testPackageMetadata(t, adapter, "@types/node", "18.0.0")
    })

    t.Run("package_tarball_download", func(t *testing.T) {
        testPackageTarball(t, adapter, "lodash", "4.17.21")
    })

    t.Run("error_handling", func(t *testing.T) {
        testErrorHandling(t, adapter)
    })
}

func testPackageMetadata(t *testing.T, adapter ports.PackageManager, name, version string) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    req := &ports.PackageRequest{
        Type:    "npm",
        Package: name,
        Version: version,
    }

    response, err := adapter.HandleRequest(ctx, req)
    require.NoError(t, err)

    // Verify response structure
    assert.NotEmpty(t, response.Data)
    assert.Equal(t, "application/json", response.ContentType)
    assert.NotEmpty(t, response.Etag)

    // Verify JSON structure
    var metadata map[string]interface{}
    err = json.Unmarshal(response.Data, &metadata)
    require.NoError(t, err)

    assert.Equal(t, name, metadata["name"])
    assert.NotEmpty(t, metadata["version"])
    assert.NotEmpty(t, metadata["description"])
}
```

### Database Integration Tests

```go
func TestConfigAdapter_SQLite(t *testing.T) {
    if testing.Short() {
        t.Skip("Integration tests skipped in short mode")
    }

    // Setup temporary database
    dbPath := filepath.Join(t.TempDir(), "test.db")

    config := &sqlite.Config{
        Path:    dbPath,
        Timeout: 5 * time.Second,
    }

    adapter, err := sqlite.NewConfigAdapter(config)
    require.NoError(t, err)
    defer adapter.Close()

    t.Run("config_crud_operations", func(t *testing.T) {
        testConfigCRUD(t, adapter)
    })

    t.Run("concurrent_access", func(t *testing.T) {
        testConcurrentConfigAccess(t, adapter)
    })

    t.Run("transaction_rollback", func(t *testing.T) {
        testTransactionRollback(t, adapter)
    })
}
```

## Performance Integration Tests

### Load Testing

```go
func TestNPMProxy_LoadTest(t *testing.T) {
    if testing.Short() {
        t.Skip("Load tests skipped in short mode")
    }

    server := SetupIntegrationTestServer(t)
    defer server.Cleanup()

    const (
        concurrentUsers = 10
        requestsPerUser = 100
        testDuration    = 30 * time.Second
    )

    t.Run("concurrent_requests", func(t *testing.T) {
        var wg sync.WaitGroup
        results := make(chan time.Duration, concurrentUsers*requestsPerUser)
        errors := make(chan error, concurrentUsers*requestsPerUser)

        ctx, cancel := context.WithTimeout(context.Background(), testDuration)
        defer cancel()

        // Start concurrent users
        for i := 0; i < concurrentUsers; i++ {
            wg.Add(1)
            go func(userID int) {
                defer wg.Done()

                client := &http.Client{Timeout: 10 * time.Second}

                for j := 0; j < requestsPerUser; j++ {
                    select {
                    case <-ctx.Done():
                        return
                    default:
                    }

                    start := time.Now()

                    url := fmt.Sprintf("%s/proxy/npm/express", server.BaseURL())
                    resp, err := client.Get(url)

                    duration := time.Since(start)
                    results <- duration

                    if err != nil {
                        errors <- err
                        continue
                    }

                    resp.Body.Close()

                    if resp.StatusCode != http.StatusOK {
                        errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
                    }
                }
            }(i)
        }

        wg.Wait()
        close(results)
        close(errors)

        // Analyze results
        var durations []time.Duration
        for duration := range results {
            durations = append(durations, duration)
        }

        errorCount := 0
        for range errors {
            errorCount++
        }

        // Performance assertions
        if len(durations) > 0 {
            avgDuration := calculateAverage(durations)
            p95Duration := calculatePercentile(durations, 95)

            t.Logf("Average response time: %v", avgDuration)
            t.Logf("95th percentile: %v", p95Duration)
            t.Logf("Error rate: %.2f%%", float64(errorCount)/float64(len(durations))*100)

            // Performance requirements
            assert.Less(t, avgDuration, 5*time.Second, "Average response time should be reasonable")
            assert.Less(t, p95Duration, 10*time.Second, "95th percentile should be acceptable")
            assert.Less(t, float64(errorCount)/float64(len(durations)), 0.01, "Error rate should be less than 1%")
        }
    })
}
```

### Cache Performance Tests

```go
func TestCachePerformance_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Performance tests skipped in short mode")
    }

    cacheTypes := []string{"filesystem", "redis", "s3"}

    for _, cacheType := range cacheTypes {
        t.Run(cacheType, func(t *testing.T) {
            cache := SetupTestCache(t, cacheType)
            defer CleanupTestCache(t, cache)

            testCachePerformance(t, cache, cacheType)
        })
    }
}

func testCachePerformance(t *testing.T, cache ports.CacheManager, cacheType string) {
    ctx := context.Background()

    t.Run("throughput", func(t *testing.T) {
        const numOperations = 1000
        const concurrency = 10

        // Pre-populate cache
        for i := 0; i < numOperations; i++ {
            setReq := &ports.CacheSetRequest{
                Key:  fmt.Sprintf("perf-key-%d", i),
                Data: make([]byte, 1024), // 1KB data
                TTL:  time.Hour,
            }

            err := cache.Set(ctx, setReq)
            require.NoError(t, err)
        }

        // Measure read throughput
        start := time.Now()

        var wg sync.WaitGroup
        for i := 0; i < concurrency; i++ {
            wg.Add(1)
            go func(workerID int) {
                defer wg.Done()

                for j := workerID; j < numOperations; j += concurrency {
                    getReq := &ports.CacheRequest{
                        Key: fmt.Sprintf("perf-key-%d", j),
                    }

                    _, err := cache.Get(ctx, getReq)
                    assert.NoError(t, err)
                }
            }(i)
        }

        wg.Wait()
        duration := time.Since(start)

        throughput := float64(numOperations) / duration.Seconds()
        t.Logf("%s cache throughput: %.2f ops/sec", cacheType, throughput)

        // Performance expectations vary by cache type
        minThroughput := getMinThroughput(cacheType)
        assert.Greater(t, throughput, minThroughput, "Throughput should meet minimum requirements")
    })
}

func getMinThroughput(cacheType string) float64 {
    switch cacheType {
    case "memory":
        return 10000 // ops/sec
    case "filesystem":
        return 1000
    case "redis":
        return 5000
    case "s3":
        return 100
    default:
        return 500
    }
}
```

## Test Environment Setup

### Test Server Configuration

```go
// tests/integration/testutil/server.go
package testutil

import (
    "context"
    "fmt"
    "net"
    "testing"
    "time"

    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
    "proxynd/internal/config"
)

type IntegrationTestServer struct {
    App     *fiber.App
    Address string
    Config  *config.UnifiedConfig
    cleanup []func() error
}

func SetupIntegrationTestServer(t *testing.T) *IntegrationTestServer {
    t.Helper()

    // Create test configuration
    testConfig := createTestConfig(t)

    // Setup test dependencies
    cache := SetupTestCache(t, "memory")

    // Create Fiber app
    app := fiber.New(fiber.Config{
        DisableStartupMessage: true,
        ErrorHandler: func(c *fiber.Ctx, err error) error {
            t.Logf("Test server error: %v", err)
            return c.Status(500).SendString(err.Error())
        },
    })

    // Setup routes
    routeConfig := app.InitializeRouteConfig(testConfig, false) // Use legacy for stability
    app.SetupRoutes(app, routeConfig)

    // Find available port
    listener, err := net.Listen("tcp", ":0")
    require.NoError(t, err)

    port := listener.Addr().(*net.TCPAddr).Port
    listener.Close()

    address := fmt.Sprintf("localhost:%d", port)

    server := &IntegrationTestServer{
        App:     app,
        Address: address,
        Config:  testConfig,
    }

    // Start server in background
    go func() {
        err := app.Listen(address)
        if err != nil {
            t.Logf("Test server error: %v", err)
        }
    }()

    // Wait for server to be ready
    server.WaitForReady(t, 10*time.Second)

    return server
}

func (s *IntegrationTestServer) BaseURL() string {
    return fmt.Sprintf("http://%s", s.Address)
}

func (s *IntegrationTestServer) WaitForReady(t *testing.T, timeout time.Duration) {
    t.Helper()

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    client := &http.Client{Timeout: time.Second}
    url := s.BaseURL() + "/health"

    for {
        select {
        case <-ctx.Done():
            t.Fatal("Test server failed to become ready")
        default:
        }

        resp, err := client.Get(url)
        if err == nil && resp.StatusCode == http.StatusOK {
            resp.Body.Close()
            return
        }

        if resp != nil {
            resp.Body.Close()
        }

        time.Sleep(100 * time.Millisecond)
    }
}

func (s *IntegrationTestServer) Cleanup() {
    if s.App != nil {
        s.App.Shutdown()
    }

    for _, cleanup := range s.cleanup {
        cleanup()
    }
}
```

### Test Configuration

```go
func createTestConfig(t *testing.T) *config.UnifiedConfig {
    tempDir := t.TempDir()

    return &config.UnifiedConfig{
        Server: config.ServerConfig{
            Host:         "localhost",
            Port:         0, // Will be set dynamically
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 30 * time.Second,
        },
        Cache: config.CacheConfig{
            Backend:   "memory",
            TTL:       time.Hour,
            Directory: tempDir,
        },
        Registries: config.RegistriesConfig{
            NPM: config.NPMConfig{
                Enabled:  true,
                Upstream: "https://registry.npmjs.org",
                Timeout:  30 * time.Second,
            },
            PyPI: config.PyPIConfig{
                Enabled: true,
                Upstream: "https://pypi.org",
                Simple:  "https://pypi.org/simple",
                Timeout: 30 * time.Second,
            },
        },
        Logging: config.LoggingConfig{
            Level:  "debug",
            Format: "json",
            Output: "stdout",
        },
    }
}
```

## Running Integration Tests

### Command Reference
```bash
# Run all integration tests
make test-integration

# Run specific package manager integration tests
make test-npm
make test-pip
make test-docker

# Run with coverage
make test-coverage-integration

# Run with verbose output
go test -tags=integration -v ./tests/integration/...

# Run specific test
go test -tags=integration -run TestNPMHandler_Integration ./tests/integration/

# Run with network access (for real external calls)
ENABLE_NETWORK_TESTS=true go test -tags=integration ./tests/integration/...

# Run performance tests
go test -tags=integration -run Performance ./tests/integration/...
```

### Environment Variables
```bash
# Control test behavior
export INTEGRATION_TEST_TIMEOUT=300s
export ENABLE_NETWORK_TESTS=true
export ENABLE_LOAD_TESTS=true

# External service endpoints
export NPM_REGISTRY_URL=https://registry.npmjs.org
export PYPI_INDEX_URL=https://pypi.org/simple
export DOCKER_REGISTRY_URL=https://registry-1.docker.io

# Test credentials (for auth integration tests)
export TEST_GITHUB_TOKEN=<token>
export TEST_DOCKER_USERNAME=<username>
export TEST_DOCKER_PASSWORD=<password>
```

## Quality Gates

### Integration Test Requirements
- **Coverage**: 80%+ for adapter layer
- **Real Dependencies**: Test with actual external services when possible
- **Error Scenarios**: Test network failures, timeouts, auth failures
- **Performance**: Verify response times and throughput
- **Concurrency**: Test thread safety and race conditions

### Performance Requirements
| Component | Metric | Requirement |
|-----------|--------|-------------|
| HTTP Handlers | Response Time | < 5s (95th percentile) |
| Cache Operations | Latency | < 100ms (average) |
| Package Downloads | Throughput | > 10 MB/s |
| Concurrent Requests | Error Rate | < 1% |

## Best Practices

### ✅ Do
- Use real external dependencies when possible
- Test with realistic data sizes
- Include timeout and error scenarios
- Test concurrent access patterns
- Clean up resources after tests
- Use environment variables for configuration
- Skip tests when dependencies unavailable

### ❌ Don't
- Hardcode external service URLs
- Ignore cleanup on test failure
- Test with unrealistic data
- Assume network connectivity
- Create dependencies between tests
- Commit test credentials
- Run long tests in short mode

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Network timeouts | Slow external services | Increase timeouts or use mocks |
| Port conflicts | Multiple test servers | Use dynamic port allocation |
| Resource leaks | Missing cleanup | Implement proper cleanup |
| Flaky tests | Race conditions | Add proper synchronization |
| Auth failures | Invalid credentials | Use test credentials or skip |

### Debug Tips
```bash
# Enable debug logging
PROXYND_LOG_LEVEL=debug go test -tags=integration

# Run with network tracing
go test -tags=integration -v -args -trace=network

# Profile integration tests
go test -tags=integration -memprofile=mem.prof -cpuprofile=cpu.prof

# Debug specific test
dlv test -- -test.run TestNPMHandler_Integration -test.tags=integration
```

---

**Last Updated**: 2025-08-15  
**Version**: 1.0.0  
**Authors**: Test Designer (claude-opus)
