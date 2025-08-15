# 🌐 End-to-End (E2E) Tests Guidelines

## Overview

End-to-End tests validate complete user scenarios by testing the **entire system** from HTTP request to response, including all layers: Adapters → Ports → Usecase → Domain. These tests use the `docker-compose.e2e.yml` environment to simulate real-world conditions.

## Test Scope

### Complete System Validation
- **Full HTTP Workflows**: Client request → ProxyND → Upstream → Cache → Response
- **Multi-Service Scenarios**: ProxyND + MinIO + Redis + Mock Registries
- **Cross-Package Manager Workflows**: NPM + PyPI + Docker in single scenarios
- **Authentication & Authorization**: OAuth2, JWT, LDAP integration
- **Monitoring & Observability**: Metrics, logs, health checks

## E2E Test Categories

### 1. User Journey Tests
Complete workflows a real user would perform.

### 2. System Integration Tests
Multiple services working together.

### 3. Failure Recovery Tests
System behavior during outages and recovery.

### 4. Performance & Load Tests
System behavior under realistic load.

## Test Environment

### Docker Compose Setup

```yaml
# docker-compose.e2e.yml (enhanced for E2E testing)
version: '3.8'

services:
  # ProxyND main service
  proxynd:
    build:
      context: .
      dockerfile: Dockerfile
    image: proxynd:e2e
    container_name: proxynd-e2e
    ports:
      - "8080:8080"
      - "9090:9090"  # Metrics port
    volumes:
      - ./tests/e2e/config:/config
      - proxynd_storage:/storage
      - proxynd_cache:/cache
    environment:
      - CONFIG_DIR=/config
      - STORAGE_DIR=/storage
      - CACHE_DIR=/cache
      - LOG_LEVEL=debug
      - ENABLE_METRICS=true
    depends_on:
      - minio
      - redis
      - mock-registry
    networks:
      - proxynd-test
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  # MinIO (S3-compatible storage)
  minio:
    image: minio/minio:latest
    container_name: minio-e2e
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    environment:
      - MINIO_ROOT_USER=minioadmin
      - MINIO_ROOT_PASSWORD=minioadmin123
    command: server /data --console-address ":9001"
    networks:
      - proxynd-test
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Redis Cache
  redis:
    image: redis:7-alpine
    container_name: redis-e2e
    ports:
      - "6379:6379"
    networks:
      - proxynd-test
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3

  # Mock upstream registries
  mock-registry:
    image: nginx:alpine
    container_name: mock-registry-e2e
    ports:
      - "8081:80"
    volumes:
      - ./tests/e2e/mock-registry:/usr/share/nginx/html
      - ./tests/e2e/nginx.conf:/etc/nginx/nginx.conf
    networks:
      - proxynd-test
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 10s
      timeout: 5s
      retries: 3

volumes:
  proxynd_storage:
  proxynd_cache:
  minio_data:

networks:
  proxynd-test:
    driver: bridge
```

## Test Structure

### User Journey Tests

```go
// +build e2e

package e2e

import (
    "context"
    "net/http"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
    suite.Suite
    BaseURL    string
    Client     *http.Client
    containers *testcontainers.DockerCompose
}

func (suite *E2ETestSuite) SetupSuite() {
    // Start docker-compose environment
    suite.containers = testcontainers.NewLocalDockerCompose(
        []string{"docker-compose.e2e.yml"},
        "e2e-test",
    )

    err := suite.containers.
        WithCommand([]string{"up", "-d"}).
        Invoke()
    require.NoError(suite.T(), err)

    // Wait for services to be ready
    suite.waitForServices()

    suite.BaseURL = "http://localhost:8080"
    suite.Client = &http.Client{
        Timeout: 30 * time.Second,
    }
}

func (suite *E2ETestSuite) TearDownSuite() {
    if suite.containers != nil {
        suite.containers.Down()
    }
}

func TestE2ESuite(t *testing.T) {
    if testing.Short() {
        t.Skip("E2E tests skipped in short mode")
    }

    suite.Run(t, new(E2ETestSuite))
}

// Test complete NPM package installation workflow
func (suite *E2ETestSuite) TestNPM_CompletePackageWorkflow() {
    t := suite.T()

    // Scenario: Developer installs a package through ProxyND
    packageName := "express"
    version := "4.18.0"

    t.Run("step_1_package_metadata", func(t *testing.T) {
        // Get package metadata (first request - cache miss)
        url := fmt.Sprintf("%s/proxy/npm/%s", suite.BaseURL, packageName)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Verify successful response
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
        assert.Equal(t, "MISS", resp.Header.Get("X-Cache"))

        // Parse and verify package metadata
        var metadata map[string]interface{}
        err = json.NewDecoder(resp.Body).Decode(&metadata)
        require.NoError(t, err)

        assert.Equal(t, packageName, metadata["name"])
        assert.NotEmpty(t, metadata["version"])
        assert.NotEmpty(t, metadata["dist"])
    })

    t.Run("step_2_cached_metadata", func(t *testing.T) {
        // Get same package metadata (second request - cache hit)
        url := fmt.Sprintf("%s/proxy/npm/%s", suite.BaseURL, packageName)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Verify cache hit
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        assert.Equal(t, "HIT", resp.Header.Get("X-Cache"))

        // Response should be much faster
        responseTime := resp.Header.Get("X-Response-Time")
        if responseTime != "" {
            // Should be served from cache quickly
            assert.Contains(t, responseTime, "ms")
        }
    })

    t.Run("step_3_package_tarball", func(t *testing.T) {
        // Download package tarball
        url := fmt.Sprintf("%s/proxy/npm/%s/-/%s-%s.tgz",
            suite.BaseURL, packageName, packageName, version)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Verify tarball download
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        assert.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

        // Verify content
        content, err := io.ReadAll(resp.Body)
        require.NoError(t, err)
        assert.Greater(t, len(content), 1000, "Tarball should be substantial")

        // Verify gzip header
        assert.Equal(t, []byte{0x1f, 0x8b}, content[:2])
    })

    t.Run("step_4_verify_metrics", func(t *testing.T) {
        // Check that metrics were recorded
        metricsURL := fmt.Sprintf("%s/metrics", suite.BaseURL)
        resp, err := suite.Client.Get(metricsURL)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)

        body, err := io.ReadAll(resp.Body)
        require.NoError(t, err)

        metrics := string(body)

        // Verify key metrics are present
        assert.Contains(t, metrics, "proxynd_requests_total")
        assert.Contains(t, metrics, "proxynd_cache_operations_total")
        assert.Contains(t, metrics, "proxynd_upstream_requests_total")

        // Verify NPM-specific metrics
        assert.Contains(t, metrics, `package_manager="npm"`)
        assert.Contains(t, metrics, `cache_status="hit"`)
        assert.Contains(t, metrics, `cache_status="miss"`)
    })
}

// Test multi-package manager scenario
func (suite *E2ETestSuite) TestMultiPackageManager_Workflow() {
    t := suite.T()

    // Scenario: Developer uses multiple package managers
    scenarios := []struct {
        name        string
        url         string
        contentType string
    }{
        {
            name:        "npm_package",
            url:         "/proxy/npm/lodash",
            contentType: "application/json",
        },
        {
            name:        "pypi_package",
            url:         "/proxy/pip/requests",
            contentType: "application/json",
        },
        {
            name:        "docker_manifest",
            url:         "/proxy/docker/library/nginx/manifests/latest",
            contentType: "application/vnd.docker.distribution.manifest.v2+json",
        },
    }

    for _, scenario := range scenarios {
        t.Run(scenario.name, func(t *testing.T) {
            url := suite.BaseURL + scenario.url
            resp, err := suite.Client.Get(url)
            require.NoError(t, err)
            defer resp.Body.Close()

            if resp.StatusCode == http.StatusOK {
                assert.Equal(t, scenario.contentType, resp.Header.Get("Content-Type"))
            } else {
                // Log response for debugging
                body, _ := io.ReadAll(resp.Body)
                t.Logf("Request failed: %s - %s", resp.Status, string(body))
            }
        })
    }
}

// Test authentication workflow
func (suite *E2ETestSuite) TestAuthentication_Workflow() {
    t := suite.T()

    t.Run("unauthenticated_request", func(t *testing.T) {
        // Try to access protected endpoint without auth
        url := fmt.Sprintf("%s/api/admin/config", suite.BaseURL)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Should require authentication
        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })

    t.Run("basic_authentication", func(t *testing.T) {
        // Create authenticated client
        client := &http.Client{Timeout: 30 * time.Second}

        // Login request
        loginURL := fmt.Sprintf("%s/auth/login", suite.BaseURL)
        loginData := strings.NewReader(`{"username":"admin","password":"password"}`)

        req, err := http.NewRequest("POST", loginURL, loginData)
        require.NoError(t, err)
        req.Header.Set("Content-Type", "application/json")

        resp, err := client.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusOK {
            // Extract token from response
            var authResp map[string]interface{}
            err = json.NewDecoder(resp.Body).Decode(&authResp)
            require.NoError(t, err)

            token := authResp["token"].(string)
            assert.NotEmpty(t, token)

            // Use token for authenticated request
            protectedURL := fmt.Sprintf("%s/api/admin/stats", suite.BaseURL)
            req, err := http.NewRequest("GET", protectedURL, nil)
            require.NoError(t, err)
            req.Header.Set("Authorization", "Bearer "+token)

            resp, err := client.Do(req)
            require.NoError(t, err)
            defer resp.Body.Close()

            assert.Equal(t, http.StatusOK, resp.StatusCode)
        }
    })
}

// Test system failure and recovery
func (suite *E2ETestSuite) TestFailureRecovery_Scenarios() {
    t := suite.T()

    t.Run("upstream_failure_graceful_degradation", func(t *testing.T) {
        // This test would require controlling the mock registry
        // to simulate upstream failures

        // Populate cache first
        url := fmt.Sprintf("%s/proxy/npm/test-package", suite.BaseURL)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        resp.Body.Close()

        // Simulate upstream failure (would need test harness)
        // ... disable mock registry temporarily ...

        // Request should still work from cache
        resp, err = suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusOK {
            assert.Equal(t, "HIT", resp.Header.Get("X-Cache"))
        }
    })

    t.Run("cache_failure_fallback", func(t *testing.T) {
        // Test behavior when cache is unavailable
        // but upstream is still accessible

        url := fmt.Sprintf("%s/proxy/npm/fallback-test-package", suite.BaseURL)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Should still work, just without caching
        if resp.StatusCode == http.StatusOK {
            assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
        }
    })
}
```

### Load Testing Scenarios

```go
// Test system under realistic load
func (suite *E2ETestSuite) TestLoad_RealisticUsage() {
    t := suite.T()

    // Skip load tests unless specifically enabled
    if os.Getenv("ENABLE_LOAD_TESTS") != "true" {
        t.Skip("Load tests disabled (set ENABLE_LOAD_TESTS=true to enable)")
    }

    const (
        testDuration    = 2 * time.Minute
        concurrentUsers = 20
        requestRate     = 10 // requests per second per user
    )

    t.Run("sustained_load", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), testDuration)
        defer cancel()

        var wg sync.WaitGroup
        results := make(chan loadTestResult, concurrentUsers*1000)

        // Package list for testing
        packages := []string{
            "express", "lodash", "react", "vue", "angular",
            "webpack", "babel", "typescript", "eslint", "prettier",
        }

        // Start concurrent users
        for i := 0; i < concurrentUsers; i++ {
            wg.Add(1)
            go func(userID int) {
                defer wg.Done()

                client := &http.Client{Timeout: 10 * time.Second}
                ticker := time.NewTicker(time.Second / time.Duration(requestRate))
                defer ticker.Stop()

                for {
                    select {
                    case <-ctx.Done():
                        return
                    case <-ticker.C:
                        // Random package request
                        pkg := packages[rand.Intn(len(packages))]
                        url := fmt.Sprintf("%s/proxy/npm/%s", suite.BaseURL, pkg)

                        start := time.Now()
                        resp, err := client.Get(url)
                        duration := time.Since(start)

                        result := loadTestResult{
                            Duration:   duration,
                            StatusCode: 0,
                            Error:      err,
                        }

                        if resp != nil {
                            result.StatusCode = resp.StatusCode
                            resp.Body.Close()
                        }

                        results <- result
                    }
                }
            }(i)
        }

        wg.Wait()
        close(results)

        // Analyze results
        var (
            totalRequests    int
            successfulRequests int
            totalDuration    time.Duration
            maxDuration      time.Duration
            errors          []error
        )

        for result := range results {
            totalRequests++
            totalDuration += result.Duration

            if result.Duration > maxDuration {
                maxDuration = result.Duration
            }

            if result.Error == nil && result.StatusCode == http.StatusOK {
                successfulRequests++
            } else {
                if result.Error != nil {
                    errors = append(errors, result.Error)
                }
            }
        }

        // Performance assertions
        if totalRequests > 0 {
            avgDuration := totalDuration / time.Duration(totalRequests)
            successRate := float64(successfulRequests) / float64(totalRequests)

            t.Logf("Load test results:")
            t.Logf("  Total requests: %d", totalRequests)
            t.Logf("  Successful requests: %d", successfulRequests)
            t.Logf("  Success rate: %.2f%%", successRate*100)
            t.Logf("  Average response time: %v", avgDuration)
            t.Logf("  Max response time: %v", maxDuration)
            t.Logf("  Errors: %d", len(errors))

            // Performance requirements
            assert.Greater(t, successRate, 0.95, "Success rate should be > 95%")
            assert.Less(t, avgDuration, 5*time.Second, "Average response time should be reasonable")
            assert.Less(t, maxDuration, 30*time.Second, "Max response time should be acceptable")
        }
    })
}

type loadTestResult struct {
    Duration   time.Duration
    StatusCode int
    Error      error
}
```

### Health Check & Monitoring Tests

```go
func (suite *E2ETestSuite) TestMonitoring_HealthChecks() {
    t := suite.T()

    t.Run("health_endpoint", func(t *testing.T) {
        url := fmt.Sprintf("%s/health", suite.BaseURL)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)

        var health map[string]interface{}
        err = json.NewDecoder(resp.Body).Decode(&health)
        require.NoError(t, err)

        assert.Equal(t, "healthy", health["status"])
        assert.NotEmpty(t, health["timestamp"])
        assert.NotEmpty(t, health["version"])

        // Check component health
        if components, ok := health["components"].(map[string]interface{}); ok {
            for component, status := range components {
                t.Logf("Component %s: %v", component, status)
            }
        }
    })

    t.Run("metrics_endpoint", func(t *testing.T) {
        url := fmt.Sprintf("%s/metrics", suite.BaseURL)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)
        assert.Equal(t, "text/plain; version=0.0.4; charset=utf-8",
            resp.Header.Get("Content-Type"))

        body, err := io.ReadAll(resp.Body)
        require.NoError(t, err)

        metrics := string(body)

        // Verify essential metrics
        requiredMetrics := []string{
            "proxynd_requests_total",
            "proxynd_request_duration_seconds",
            "proxynd_cache_operations_total",
            "proxynd_upstream_requests_total",
            "go_goroutines",
            "go_memstats_alloc_bytes",
        }

        for _, metric := range requiredMetrics {
            assert.Contains(t, metrics, metric, "Metric %s should be present", metric)
        }
    })

    t.Run("ready_endpoint", func(t *testing.T) {
        url := fmt.Sprintf("%s/ready", suite.BaseURL)
        resp, err := suite.Client.Get(url)
        require.NoError(t, err)
        defer resp.Body.Close()

        // System should be ready after startup
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    })
}
```

## Test Data & Fixtures

### Mock Registry Data

```bash
# tests/e2e/mock-registry/npm/express/index.json
{
  "name": "express",
  "description": "Fast, unopinionated, minimalist web framework",
  "dist-tags": {
    "latest": "4.18.0"
  },
  "versions": {
    "4.18.0": {
      "name": "express",
      "version": "4.18.0",
      "description": "Fast, unopinionated, minimalist web framework",
      "main": "index.js",
      "dist": {
        "shasum": "abc123",
        "tarball": "http://mock-registry/npm/express/-/express-4.18.0.tgz"
      }
    }
  }
}
```

### Test Configuration

```yaml
# tests/e2e/config/proxynd.yaml
server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

cache:
  backend: file
  ttl: 1h
  directory: /cache

registries:
  npm:
    enabled: true
    upstream: http://mock-registry/npm
    timeout: 30s

  pypi:
    enabled: true
    upstream: http://mock-registry/pypi
    simple: http://mock-registry/pypi/simple
    timeout: 30s

logging:
  level: debug
  format: json
  output: stdout

metrics:
  enabled: true
  path: /metrics
```

## Running E2E Tests

### Command Reference

```bash
# Run all E2E tests
make test-e2e

# Run specific E2E test
go test -tags=e2e -run TestE2ESuite/TestNPM_CompletePackageWorkflow ./tests/e2e/

# Run with load testing enabled
ENABLE_LOAD_TESTS=true make test-e2e

# Run with custom timeout
INTEGRATION_TEST_TIMEOUT=600s make test-e2e

# Run with verbose output
go test -tags=e2e -v ./tests/e2e/

# Debug E2E tests
go test -tags=e2e -v -run TestE2ESuite ./tests/e2e/ -args -test.timeout=30m
```

### Environment Variables

```bash
# Test configuration
export E2E_TEST_TIMEOUT=300s
export ENABLE_LOAD_TESTS=true
export ENABLE_NETWORK_TESTS=true

# Docker compose settings
export COMPOSE_PROJECT_NAME=proxynd-e2e
export COMPOSE_HTTP_TIMEOUT=300

# Service endpoints
export PROXYND_URL=http://localhost:8080
export MINIO_URL=http://localhost:9000
export REDIS_URL=redis://localhost:6379
```

## Test Scenarios Matrix

### Core Scenarios

| Scenario | NPM | PyPI | Docker | APT | Maven |
|----------|-----|------|--------|-----|-------|
| Package Metadata | ✅ | ✅ | ✅ | ✅ | ✅ |
| Package Download | ✅ | ✅ | ✅ | ✅ | ✅ |
| Cache Hit/Miss | ✅ | ✅ | ✅ | ✅ | ✅ |
| Signature Verification | ✅ | ✅ | ✅ | ✅ | ✅ |
| Authentication | ✅ | ✅ | ✅ | ❌ | ✅ |
| Load Testing | ✅ | ✅ | ❌ | ❌ | ❌ |

### Failure Scenarios

| Scenario | Description | Expected Behavior |
|----------|-------------|-------------------|
| Upstream Timeout | Registry responds slowly | Graceful timeout, error response |
| Upstream Unavailable | Registry is down | Cache fallback or error |
| Cache Failure | Cache backend fails | Direct upstream proxy |
| Network Partition | Network connectivity lost | Cached responses only |
| High Load | Many concurrent requests | Degraded performance, no failures |

## Quality Gates

### E2E Test Requirements
- **Scenario Coverage**: 100% of critical user journeys
- **Performance**: Response times within SLA
- **Reliability**: 99%+ success rate under normal load
- **Recovery**: Graceful degradation during failures
- **Monitoring**: All metrics and health checks functional

### Performance SLAs
| Metric | Requirement |
|--------|-------------|
| Package Metadata | < 2s (95th percentile) |
| Package Download | > 10 MB/s throughput |
| Cache Hit Response | < 100ms |
| System Availability | 99%+ uptime |
| Concurrent Users | 100+ without degradation |

## Best Practices

### ✅ Do
- Test complete user workflows
- Use realistic test data
- Include failure scenarios
- Test with concurrent users
- Verify metrics and monitoring
- Clean up resources properly
- Use stable test environment

### ❌ Don't
- Test only happy paths
- Use hardcoded timing assumptions
- Ignore resource cleanup
- Assume service availability
- Skip performance validation
- Test with unrealistic loads
- Commit sensitive test data

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Container startup fails | Port conflicts, resource limits | Check ports, increase resources |
| Tests timeout | Slow container startup | Increase timeouts, optimize images |
| Network connectivity | Docker networking issues | Verify network configuration |
| Service dependencies | Services not ready | Implement proper health checks |
| Resource exhaustion | Insufficient memory/CPU | Increase Docker resources |

### Debug Commands

```bash
# Check container status
docker-compose -f docker-compose.e2e.yml ps

# View container logs
docker-compose -f docker-compose.e2e.yml logs proxynd

# Execute commands in container
docker-compose -f docker-compose.e2e.yml exec proxynd /bin/sh

# Check network connectivity
docker-compose -f docker-compose.e2e.yml exec proxynd curl http://mock-registry/health

# Monitor resource usage
docker stats $(docker-compose -f docker-compose.e2e.yml ps -q)
```

### Performance Analysis

```bash
# Generate load test reports
go test -tags=e2e -run TestLoad -bench=. -benchmem ./tests/e2e/

# Profile E2E tests
go test -tags=e2e -cpuprofile=e2e.prof -memprofile=e2e.mem ./tests/e2e/

# Analyze profiles
go tool pprof e2e.prof
go tool pprof e2e.mem
```

---

**Last Updated**: 2025-08-15  
**Version**: 1.0.0  
**Authors**: Test Designer (claude-opus)
