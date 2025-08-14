# Integration Testing Guide

## Overview

Integration tests verify that different components of ProxyND work correctly together. These tests focus on interactions between services, database operations, file system operations, and external dependencies.

## Test Categories

### 1. Service Integration Tests
Test interactions between multiple services working together.

### 2. Database Integration Tests
Test database operations and data persistence.

### 3. File System Integration Tests
Test file operations, cache storage, and configuration loading.

### 4. HTTP Integration Tests
Test complete HTTP request/response cycles.

### 5. External Service Integration Tests
Test integration with external registries and services.

## Directory Structure

```
tests/
├── integration/
│   ├── service_integration_test.go      # Service interactions
│   ├── cache_integration_test.go        # Cache system tests
│   ├── config_integration_test.go       # Configuration tests
│   ├── proxy_integration_test.go        # Proxy request tests
│   ├── health_integration_test.go       # Health check tests
│   ├── fixtures/                        # Test data
│   │   ├── configs/
│   │   ├── packages/
│   │   └── responses/
│   └── helpers/                         # Test utilities
│       ├── server.go                    # Test server setup
│       ├── client.go                    # Test client utilities
│       └── fixtures.go                  # Fixture loading
```

## Integration Test Patterns

### 1. Service Container Integration

```go
func TestServiceContainer_Integration(t *testing.T) {
    // Setup test environment
    tempDir := t.TempDir()
    
    // Create test configuration
    config := createIntegrationConfig(t, tempDir)
    
    // Initialize service container
    container, err := app.NewContainer(config)
    require.NoError(t, err)
    defer container.Cleanup()
    
    // Test service interactions
    configService := container.GetConfigService()
    cacheService := container.GetCacheService()
    proxyService := container.GetProxyService("maven")
    
    // Verify services can work together
    proxyConfig, err := configService.GetProxyConfig("maven")
    require.NoError(t, err)
    
    testData := []byte("test artifact data")
    err = cacheService.Put(context.Background(), "test-key", bytes.NewReader(testData), time.Hour)
    require.NoError(t, err)
    
    request := &proxy.Request{
        ProxyType: "maven",
        Path:      "/com/example/artifact/1.0/artifact-1.0.jar",
    }
    
    response, err := proxyService.HandleRequest(context.Background(), request)
    assert.NoError(t, err)
    assert.NotNil(t, response)
}
```

### 2. HTTP Request Flow Integration

```go
func TestHTTPRequestFlow_Integration(t *testing.T) {
    // Start test server
    server := startTestServer(t)
    defer server.Close()
    
    tests := []struct {
        name           string
        method         string
        path           string
        expectedStatus int
        expectedType   string
        setupCache    func(cache CacheService)
        setupUpstream func(server *httptest.Server)
    }{
        {
            name:           "maven artifact request - cache miss",
            method:         "GET",
            path:           "/proxy/maven/com/example/artifact/1.0/artifact-1.0.jar",
            expectedStatus: http.StatusOK,
            expectedType:   "application/java-archive",
            setupUpstream: func(upstream *httptest.Server) {
                // Setup mock upstream response
                upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.Header().Set("Content-Type", "application/java-archive")
                    w.WriteHeader(http.StatusOK)
                    w.Write([]byte("mock jar content"))
                })
            },
        },
        {
            name:           "npm package request - cache hit",
            method:         "GET",
            path:           "/proxy/npm/express/-/express-4.18.0.tgz",
            expectedStatus: http.StatusOK,
            expectedType:   "application/octet-stream",
            setupCache: func(cache CacheService) {
                cache.Put(context.Background(), "npm:express/-/express-4.18.0.tgz", 
                    strings.NewReader("cached package"), time.Hour)
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setupCache != nil {
                tt.setupCache(server.CacheService)
            }
            if tt.setupUpstream != nil {
                tt.setupUpstream(server.UpstreamServer)
            }
            
            req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
            require.NoError(t, err)
            
            resp, err := http.DefaultClient.Do(req)
            require.NoError(t, err)
            defer resp.Body.Close()
            
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
            assert.Equal(t, tt.expectedType, resp.Header.Get("Content-Type"))
        })
    }
}
```

### 3. Cache System Integration

```go
func TestCacheSystem_Integration(t *testing.T) {
    tests := []struct {
        name        string
        backend     string
        setupConfig func() *config.CacheConfig
    }{
        {
            name:    "file cache backend",
            backend: "file",
            setupConfig: func() *config.CacheConfig {
                return &config.CacheConfig{
                    Backend: "file",
                    File: config.FileCacheConfig{
                        Directory:   t.TempDir(),
                        MaxFileSize: 100 * 1024 * 1024, // 100MB
                    },
                }
            },
        },
        {
            name:    "s3 cache backend",
            backend: "s3",
            setupConfig: func() *config.CacheConfig {
                return &config.CacheConfig{
                    Backend: "s3",
                    S3: config.S3CacheConfig{
                        Bucket:   "test-bucket",
                        Region:   "us-east-1",
                        Endpoint: "http://localhost:9000", // MinIO for testing
                    },
                }
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cacheConfig := tt.setupConfig()
            
            // Initialize cache service
            cache, err := cache.NewService(cacheConfig)
            require.NoError(t, err)
            defer cache.Close()
            
            // Test cache operations
            ctx := context.Background()
            key := "test-integration-key"
            data := []byte("integration test data")
            
            // Test Put operation
            err = cache.Put(ctx, key, bytes.NewReader(data), time.Hour)
            assert.NoError(t, err)
            
            // Test Get operation
            reader, exists, err := cache.Get(ctx, key)
            assert.NoError(t, err)
            assert.True(t, exists)
            
            retrieved, err := io.ReadAll(reader)
            require.NoError(t, err)
            reader.Close()
            
            assert.Equal(t, data, retrieved)
            
            // Test Exists operation
            exists, err = cache.Exists(ctx, key)
            assert.NoError(t, err)
            assert.True(t, exists)
            
            // Test Delete operation
            err = cache.Delete(ctx, key)
            assert.NoError(t, err)
            
            // Verify deletion
            exists, err = cache.Exists(ctx, key)
            assert.NoError(t, err)
            assert.False(t, exists)
        })
    }
}
```

### 4. Configuration Hot Reload Integration

```go
func TestConfigHotReload_Integration(t *testing.T) {
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "config.yaml")
    
    // Initial configuration
    initialConfig := `
server:
  port: "8080"
cache:
  backend: file
  file:
    directory: /tmp/cache
registries:
  maven:
    enabled: true
    repositories:
      - url: "https://repo1.maven.org/maven2"
`
    
    err := os.WriteFile(configFile, []byte(initialConfig), 0644)
    require.NoError(t, err)
    
    // Start service with hot reload enabled
    container, err := app.NewContainer(&config.RootConfig{
        ConfigFile: configFile,
        HotReload:  true,
    })
    require.NoError(t, err)
    defer container.Cleanup()
    
    // Get initial proxy config
    configService := container.GetConfigService()
    initialProxyConfig, err := configService.GetProxyConfig("maven")
    require.NoError(t, err)
    assert.True(t, initialProxyConfig.(*config.MavenProxyConfig).Enabled)
    
    // Update configuration file
    updatedConfig := `
server:
  port: "8080"
cache:
  backend: file
  file:
    directory: /tmp/cache
registries:
  maven:
    enabled: false
    repositories:
      - url: "https://repo1.maven.org/maven2"
`
    
    err = os.WriteFile(configFile, []byte(updatedConfig), 0644)
    require.NoError(t, err)
    
    // Wait for hot reload (with timeout)
    var updatedProxyConfig *config.MavenProxyConfig
    assert.Eventually(t, func() bool {
        config, err := configService.GetProxyConfig("maven")
        if err != nil {
            return false
        }
        updatedProxyConfig = config.(*config.MavenProxyConfig)
        return !updatedProxyConfig.Enabled
    }, 5*time.Second, 100*time.Millisecond, "Configuration should be reloaded")
    
    assert.False(t, updatedProxyConfig.Enabled)
}
```

### 5. Health Check Integration

```go
func TestHealthCheck_Integration(t *testing.T) {
    // Start test server with all components
    server := startFullTestServer(t)
    defer server.Close()
    
    // Test health endpoints
    healthTests := []struct {
        name           string
        endpoint       string
        expectedStatus int
        checkResponse  func(t *testing.T, body []byte)
    }{
        {
            name:           "basic health check",
            endpoint:       "/healthz",
            expectedStatus: http.StatusOK,
            checkResponse: func(t *testing.T, body []byte) {
                var health map[string]interface{}
                err := json.Unmarshal(body, &health)
                require.NoError(t, err)
                assert.Equal(t, "healthy", health["status"])
            },
        },
        {
            name:           "detailed health check",
            endpoint:       "/health/detailed",
            expectedStatus: http.StatusOK,
            checkResponse: func(t *testing.T, body []byte) {
                var health map[string]interface{}
                err := json.Unmarshal(body, &health)
                require.NoError(t, err)
                
                checks, ok := health["checks"].(map[string]interface{})
                require.True(t, ok)
                
                // Verify individual health checks
                assert.Contains(t, checks, "cache")
                assert.Contains(t, checks, "config")
                assert.Contains(t, checks, "proxy_upstreams")
            },
        },
        {
            name:           "proxy-specific health",
            endpoint:       "/health/proxy/maven",
            expectedStatus: http.StatusOK,
            checkResponse: func(t *testing.T, body []byte) {
                var health map[string]interface{}
                err := json.Unmarshal(body, &health)
                require.NoError(t, err)
                assert.Contains(t, health, "upstream_status")
                assert.Contains(t, health, "response_time")
            },
        },
    }
    
    for _, tt := range healthTests {
        t.Run(tt.name, func(t *testing.T) {
            resp, err := http.Get(server.URL + tt.endpoint)
            require.NoError(t, err)
            defer resp.Body.Close()
            
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
            
            body, err := io.ReadAll(resp.Body)
            require.NoError(t, err)
            
            if tt.checkResponse != nil {
                tt.checkResponse(t, body)
            }
        })
    }
}
```

## Test Utilities and Helpers

### 1. Test Server Setup

```go
// TestServer provides a full ProxyND server for integration testing
type TestServer struct {
    *httptest.Server
    Container     *app.Container
    TempDir       string
    ConfigFile    string
    CacheService  cache.Service
    UpstreamServer *httptest.Server
}

func StartTestServer(t *testing.T) *TestServer {
    t.Helper()
    
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "config.yaml")
    
    // Create test configuration
    config := createTestConfig(tempDir)
    configData, err := yaml.Marshal(config)
    require.NoError(t, err)
    
    err = os.WriteFile(configFile, configData, 0644)
    require.NoError(t, err)
    
    // Start upstream mock server
    upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Default upstream handler
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("mock upstream response"))
    }))
    
    // Initialize container
    container, err := app.NewContainer(&config.RootConfig{
        ConfigFile: configFile,
    })
    require.NoError(t, err)
    
    // Create HTTP server
    router := setupTestRouter(container)
    httpServer := httptest.NewServer(router)
    
    return &TestServer{
        Server:        httpServer,
        Container:     container,
        TempDir:       tempDir,
        ConfigFile:    configFile,
        CacheService:  container.GetCacheService(),
        UpstreamServer: upstreamServer,
    }
}

func (ts *TestServer) Close() {
    ts.Server.Close()
    ts.UpstreamServer.Close()
    ts.Container.Cleanup()
}
```

### 2. Test Data Fixtures

```go
// FixtureManager handles test data loading
type FixtureManager struct {
    basePath string
}

func NewFixtureManager(basePath string) *FixtureManager {
    return &FixtureManager{basePath: basePath}
}

func (fm *FixtureManager) LoadConfig(name string) (*config.RootConfig, error) {
    data, err := os.ReadFile(filepath.Join(fm.basePath, "configs", name+".yaml"))
    if err != nil {
        return nil, err
    }
    
    var config config.RootConfig
    err = yaml.Unmarshal(data, &config)
    return &config, err
}

func (fm *FixtureManager) LoadPackage(proxyType, name string) ([]byte, error) {
    return os.ReadFile(filepath.Join(fm.basePath, "packages", proxyType, name))
}

func (fm *FixtureManager) LoadResponse(name string) ([]byte, error) {
    return os.ReadFile(filepath.Join(fm.basePath, "responses", name+".json"))
}
```

### 3. Database Setup (if applicable)

```go
func SetupTestDatabase(t *testing.T) *sql.DB {
    t.Helper()
    
    // Use in-memory SQLite for testing
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    
    // Run migrations
    err = runMigrations(db)
    require.NoError(t, err)
    
    // Clean up on test completion
    t.Cleanup(func() {
        db.Close()
    })
    
    return db
}
```

## Running Integration Tests

### Basic Commands

```bash
# Run all integration tests
go test ./tests/integration/...

# Run with verbose output
go test -v ./tests/integration/...

# Run specific integration test
go test -run TestServiceContainer_Integration ./tests/integration/

# Run with race detection
go test -race ./tests/integration/...
```

### Environment Setup

```bash
# Set environment variables for testing
export CONFIG_DIR="./tests/fixtures/configs"
export STORAGE_DIR="/tmp/proxynd-test"
export LOG_LEVEL="debug"

# Run integration tests
go test ./tests/integration/...
```

### Docker-based Testing

```bash
# Start test dependencies with Docker Compose
docker-compose -f tests/docker-compose.test.yml up -d

# Run integration tests
go test ./tests/integration/...

# Clean up
docker-compose -f tests/docker-compose.test.yml down
```

## Test Data Management

### 1. Configuration Fixtures

```yaml
# tests/fixtures/configs/basic.yaml
server:
  port: "8080"
  host: "localhost"

cache:
  backend: "file"
  file:
    directory: "/tmp/test-cache"
    max_file_size: 104857600

registries:
  maven:
    enabled: true
    repositories:
      - url: "https://repo1.maven.org/maven2"
  npm:
    enabled: true
    registry_url: "https://registry.npmjs.org"
```

### 2. Mock Data

```go
// Mock upstream responses
var MockResponses = map[string]MockResponse{
    "maven-artifact": {
        ContentType: "application/java-archive",
        Body:        []byte("mock jar content"),
        Headers: map[string]string{
            "Content-Length": "16",
            "Last-Modified":  "Wed, 21 Oct 2015 07:28:00 GMT",
        },
    },
    "npm-package": {
        ContentType: "application/octet-stream",
        Body:        []byte("mock package content"),
        Headers: map[string]string{
            "Content-Length": "20",
        },
    },
}
```

## Performance Testing

### 1. Load Testing

```go
func TestProxy_LoadTest(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    server := StartTestServer(t)
    defer server.Close()
    
    // Configuration
    concurrency := 10
    requestsPerWorker := 100
    
    var wg sync.WaitGroup
    errorChan := make(chan error, concurrency*requestsPerWorker)
    
    // Start workers
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            client := &http.Client{Timeout: 10 * time.Second}
            
            for j := 0; j < requestsPerWorker; j++ {
                url := fmt.Sprintf("%s/proxy/maven/com/example/artifact/%d/artifact-%d.jar", 
                    server.URL, workerID, j)
                
                resp, err := client.Get(url)
                if err != nil {
                    errorChan <- err
                    continue
                }
                
                if resp.StatusCode != http.StatusOK {
                    errorChan <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
                }
                resp.Body.Close()
            }
        }(i)
    }
    
    wg.Wait()
    close(errorChan)
    
    // Check for errors
    var errors []error
    for err := range errorChan {
        errors = append(errors, err)
    }
    
    errorRate := float64(len(errors)) / float64(concurrency*requestsPerWorker)
    assert.Less(t, errorRate, 0.01, "Error rate too high: %f", errorRate) // Less than 1%
}
```

### 2. Memory Testing

```go
func TestProxy_MemoryUsage(t *testing.T) {
    server := StartTestServer(t)
    defer server.Close()
    
    // Measure initial memory
    var initialMem runtime.MemStats
    runtime.ReadMemStats(&initialMem)
    
    // Perform operations that might leak memory
    for i := 0; i < 1000; i++ {
        resp, err := http.Get(server.URL + "/proxy/maven/com/example/test/1.0/test-1.0.jar")
        require.NoError(t, err)
        resp.Body.Close()
    }
    
    // Force garbage collection
    runtime.GC()
    runtime.GC()
    
    // Measure final memory
    var finalMem runtime.MemStats
    runtime.ReadMemStats(&finalMem)
    
    // Memory increase should be reasonable
    memIncrease := finalMem.Alloc - initialMem.Alloc
    maxIncrease := uint64(10 * 1024 * 1024) // 10MB
    
    assert.Less(t, memIncrease, maxIncrease, 
        "Memory increase too high: %d bytes", memIncrease)
}
```

## CI/CD Integration

### GitHub Actions Integration

```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    
    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
      minio:
        image: minio/minio:latest
        ports:
          - 9000:9000
        env:
          MINIO_ROOT_USER: testuser
          MINIO_ROOT_PASSWORD: testpass123
        options: --health-cmd "curl -f http://localhost:9000/minio/health/live"
    
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y build-essential
      
      - name: Setup test environment
        run: |
          mkdir -p /tmp/proxynd-test
          export CONFIG_DIR="./tests/fixtures/configs"
          export STORAGE_DIR="/tmp/proxynd-test"
      
      - name: Run integration tests
        run: |
          go test -v -race ./tests/integration/...
        env:
          REDIS_URL: redis://localhost:6379
          MINIO_ENDPOINT: http://localhost:9000
          MINIO_ACCESS_KEY: testuser
          MINIO_SECRET_KEY: testpass123
```

## Best Practices

### 1. Test Isolation

- Use `t.TempDir()` for file system tests
- Clean up resources in `t.Cleanup()` or defer statements
- Reset global state between tests
- Use separate database instances for each test

### 2. Realistic Test Data

- Use real-world package names and versions
- Include edge cases (large files, special characters)
- Test with various content types
- Include malformed data tests

### 3. Error Handling

- Test network failures
- Test timeout scenarios
- Test partial failures
- Test recovery mechanisms

### 4. Performance Considerations

- Use `testing.Short()` for long-running tests
- Include benchmarks for critical paths
- Monitor memory usage
- Test under load

### 5. Maintainability

- Keep tests independent
- Use descriptive test names
- Document complex test scenarios
- Refactor common setup code into helpers