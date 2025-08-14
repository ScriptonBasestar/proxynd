# Unit Testing Guide

## Overview

This guide provides detailed instructions for writing and running unit tests in ProxyND. We follow Go testing best practices with table-driven tests and comprehensive mocking.

## Test Structure

### Directory Layout

```
internal/
├── services/
│   ├── config/
│   │   ├── service.go
│   │   └── service_test.go
│   ├── proxy/
│   │   ├── base_service.go
│   │   ├── base_service_test.go
│   │   ├── apt_service.go
│   │   └── apt_service_test.go
│   └── adapters/
│       ├── cache_adapter.go
│       └── cache_adapter_test.go
├── handlers/
│   ├── proxy/
│   │   ├── handler.go
│   │   └── handler_test.go
├── configs/
│   ├── config.go
│   └── config_test.go
```

### Test File Naming

- Test files must end with `_test.go`
- Place test files in the same package as the code being tested
- Use descriptive test function names: `TestServiceName_MethodName`

## Writing Unit Tests

### 1. Basic Test Structure

```go
func TestService_Method(t *testing.T) {
    // Setup
    service := NewService()
    
    // Execute
    result, err := service.Method("input")
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "expected", result)
}
```

### 2. Table-Driven Tests

For comprehensive scenario coverage:

```go
func TestConfigService_GetProxyConfig(t *testing.T) {
    tests := []struct {
        name        string
        proxyType   string
        configDir   string
        setupFiles  map[string]string
        want        interface{}
        wantErr     bool
        errContains string
    }{
        {
            name:      "valid apt config",
            proxyType: "apt",
            setupFiles: map[string]string{
                "apt-proxy.yaml": `
enabled: true
mirrors:
  - url: "http://archive.ubuntu.com/ubuntu"
    distributions: ["focal", "jammy"]
`,
            },
            want:    &config.APTProxyConfig{Enabled: true},
            wantErr: false,
        },
        {
            name:        "missing config file",
            proxyType:   "maven",
            setupFiles:  map[string]string{},
            want:        nil,
            wantErr:     true,
            errContains: "not found",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
            tempDir := t.TempDir()
            
            // Setup config files
            for filename, content := range tt.setupFiles {
                createTestFile(t, tempDir, filename, content)
            }
            
            service := NewConfigService(tempDir)
            result, err := service.GetProxyConfig(tt.proxyType)
            
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errContains != "" {
                    assert.Contains(t, err.Error(), tt.errContains)
                }
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.want, result)
        })
    }
}
```

### 3. Mocking Dependencies

Using testify/mock for interface mocking:

```go
// Mock definition
type MockCacheService struct {
    mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) (io.ReadCloser, bool, error) {
    args := m.Called(ctx, key)
    if args.Get(0) == nil {
        return nil, args.Bool(1), args.Error(2)
    }
    return args.Get(0).(io.ReadCloser), args.Bool(1), args.Error(2)
}

func (m *MockCacheService) Put(ctx context.Context, key string, reader io.Reader, ttl time.Duration) error {
    args := m.Called(ctx, key, reader, ttl)
    return args.Error(0)
}

// Test using mock
func TestProxyService_HandleRequest(t *testing.T) {
    mockCache := new(MockCacheService)
    mockUpstream := new(MockUpstreamClient)
    
    // Setup expectations
    mockCache.On("Get", mock.Anything, "test-key").Return(nil, false, nil)
    mockUpstream.On("Fetch", mock.Anything, "test-url").Return([]byte("data"), nil)
    mockCache.On("Put", mock.Anything, "test-key", mock.Anything, mock.Anything).Return(nil)
    
    service := NewProxyService(mockCache, mockUpstream)
    
    result, err := service.HandleRequest(context.Background(), ProxyRequest{
        Key: "test-key",
        URL: "test-url",
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    // Verify all expectations were met
    mockCache.AssertExpectations(t)
    mockUpstream.AssertExpectations(t)
}
```

### 4. Test Helpers

Common utilities for test setup:

```go
// createTestFile creates a test file with given content
func createTestFile(t *testing.T, dir, filename, content string) {
    t.Helper()
    path := filepath.Join(dir, filename)
    err := os.WriteFile(path, []byte(content), 0644)
    require.NoError(t, err)
}

// createTestConfig creates a test configuration
func createTestConfig(t *testing.T) *config.RootConfig {
    t.Helper()
    return &config.RootConfig{
        Server: config.ServerConfig{
            Port: "8080",
            Host: "localhost",
        },
        Cache: config.CacheConfig{
            Backend: "file",
            File: config.FileCacheConfig{
                Directory: t.TempDir(),
            },
        },
    }
}

// assertHTTPResponse validates HTTP response
func assertHTTPResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedContentType string) {
    t.Helper()
    assert.Equal(t, expectedStatus, resp.StatusCode)
    assert.Equal(t, expectedContentType, resp.Header.Get("Content-Type"))
}
```

## Testing Best Practices

### 1. Test Organization

- **Arrange-Act-Assert**: Structure tests with clear setup, execution, and verification phases
- **Test one thing**: Each test should verify one specific behavior
- **Descriptive names**: Test names should describe what is being tested

### 2. Error Testing

Always test error conditions:

```go
func TestService_InvalidInput(t *testing.T) {
    service := NewService()
    
    _, err := service.Process("")
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid input")
}
```

### 3. Context Testing

Test context cancellation and timeouts:

```go
func TestService_ContextCancellation(t *testing.T) {
    service := NewService()
    
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    _, err := service.Process(ctx, "input")
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, context.Canceled))
}
```

### 4. Concurrent Testing

Test thread safety with race detection:

```go
func TestService_Concurrent(t *testing.T) {
    service := NewService()
    
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    // Launch multiple goroutines
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            _, err := service.Process(fmt.Sprintf("input-%d", id))
            if err != nil {
                errors <- err
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Errorf("Concurrent test failed: %v", err)
    }
}
```

## Running Unit Tests

### Basic Commands

```bash
# Run all unit tests
go test ./...

# Run tests in specific package
go test ./internal/services/config

# Run with verbose output
go test -v ./internal/services/config

# Run specific test
go test -run TestConfigService_GetProxyConfig ./internal/services/config
```

### Test Coverage

```bash
# Generate coverage report
go test -cover ./...

# Generate detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Check coverage for specific package
go test -cover ./internal/services/config
```

### Race Detection

```bash
# Run with race detection
go test -race ./...

# Race detection for specific package
go test -race ./internal/services/config
```

### Benchmarking

```bash
# Run benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkConfigService_Load ./internal/services/config

# Benchmark with memory profiling
go test -bench=. -benchmem ./...
```

## Common Testing Patterns

### 1. Testing HTTP Handlers

```go
func TestProxyHandler_HandleRequest(t *testing.T) {
    // Create test request
    req := httptest.NewRequest("GET", "/proxy/maven/com/example/artifact/1.0/artifact-1.0.jar", nil)
    rec := httptest.NewRecorder()
    
    // Create handler with mocked dependencies
    handler := NewProxyHandler(mockService)
    
    // Execute request
    handler.ServeHTTP(rec, req)
    
    // Assert response
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Equal(t, "application/java-archive", rec.Header().Get("Content-Type"))
}
```

### 2. Testing File Operations

```go
func TestFileCache_Store(t *testing.T) {
    tempDir := t.TempDir() // Automatically cleaned up
    cache := NewFileCache(tempDir)
    
    data := []byte("test data")
    err := cache.Store("test-key", bytes.NewReader(data))
    
    assert.NoError(t, err)
    
    // Verify file exists
    filePath := filepath.Join(tempDir, "test-key")
    assert.FileExists(t, filePath)
    
    // Verify content
    stored, err := os.ReadFile(filePath)
    assert.NoError(t, err)
    assert.Equal(t, data, stored)
}
```

### 3. Testing Configuration Loading

```go
func TestConfig_Load(t *testing.T) {
    tempDir := t.TempDir()
    
    configContent := `
server:
  port: "8080"
  host: "localhost"
cache:
  backend: "file"
  file:
    directory: "/tmp/cache"
`
    
    configFile := filepath.Join(tempDir, "config.yaml")
    err := os.WriteFile(configFile, []byte(configContent), 0644)
    require.NoError(t, err)
    
    config, err := LoadConfig(configFile)
    
    assert.NoError(t, err)
    assert.Equal(t, "8080", config.Server.Port)
    assert.Equal(t, "localhost", config.Server.Host)
    assert.Equal(t, "file", config.Cache.Backend)
}
```

## Code Coverage Goals

### Target Coverage

- **Overall**: ≥ 80%
- **Critical paths**: ≥ 95%
  - Configuration loading
  - Cache operations
  - Proxy request handling
  - Error handling

### Coverage Analysis

```bash
# Generate coverage report with function-level details
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Identify uncovered lines
go tool cover -html=coverage.out
```

### Improving Coverage

1. **Identify uncovered code**: Use HTML coverage report
2. **Add missing tests**: Focus on error paths and edge cases
3. **Test private functions**: Through public API if possible
4. **Integration tests**: For complex interactions

## Debugging Tests

### Verbose Logging

```go
func TestWithLogging(t *testing.T) {
    if testing.Verbose() {
        t.Log("Starting test with detailed logging")
    }
    
    // Test implementation
    
    if testing.Verbose() {
        t.Log("Test completed successfully")
    }
}
```

### Using Delve Debugger

```bash
# Debug specific test
dlv test ./internal/services/config -- -test.run TestConfigService_Load

# Set breakpoint in debugger
(dlv) break config.go:42
(dlv) continue
```

### Test Data Inspection

```go
func TestWithDataInspection(t *testing.T) {
    result := service.Process("input")
    
    // Print result for debugging
    t.Logf("Result: %+v", result)
    
    // Use spew for detailed output
    spew.Dump(result)
}
```

## Continuous Integration

### CI Test Configuration

Unit tests run automatically on:
- Pull requests
- Commits to main branch
- Nightly builds

### CI Commands

```yaml
# .github/workflows/test.yml
- name: Run unit tests
  run: go test -race -coverprofile=coverage.out ./...

- name: Check coverage
  run: |
    go tool cover -func=coverage.out
    go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | grep -E "^[8-9][0-9]\.|^100\."
```

## Troubleshooting

### Common Issues

1. **Test timeouts**: Increase timeout for slow operations
2. **File permissions**: Use `t.TempDir()` for file tests
3. **Port conflicts**: Use random ports in tests
4. **Resource leaks**: Always clean up resources in defer statements

### Performance Issues

```go
func TestPerformance(t *testing.T) {
    start := time.Now()
    
    // Test operation
    result := service.Process("input")
    
    duration := time.Since(start)
    
    assert.True(t, duration < 100*time.Millisecond, "Operation too slow: %v", duration)
}
```