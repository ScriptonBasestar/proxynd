# Testing Troubleshooting Guide

## Overview

This guide helps diagnose and resolve common issues encountered during testing of ProxyND components.

## Common Testing Issues

### 1. Test Environment Setup

#### Issue: Configuration Not Found
```
Error: failed to load config: config file not found
```

**Solution:**
```bash
# Ensure environment variables are set
export CONFIG_DIR="./tmp/config"
export STORAGE_DIR="./tmp/storage"

# Or run dev setup
make dev-setup

# Verify files exist
ls -la $CONFIG_DIR/
ls -la $STORAGE_DIR/
```

#### Issue: Permission Denied
```
Error: mkdir /tmp/proxynd: permission denied
```

**Solution:**
```bash
# Use user-writable directory
export STORAGE_DIR="$HOME/tmp/proxynd-test"
mkdir -p $STORAGE_DIR

# Or use t.TempDir() in tests
tempDir := t.TempDir()
```

### 2. Network and Connectivity Issues

#### Issue: Connection Refused
```
Error: Get "http://localhost:8081": connection refused
```

**Solutions:**
```bash
# Check if server is running
ps aux | grep proxynd

# Check port availability
netstat -tlnp | grep 8081
# or
ss -tlnp | grep 8081

# Start server if not running
make dev-run

# Check server health
curl -I http://localhost:8081/healthz
```

#### Issue: Upstream Connection Timeout
```
Error: context deadline exceeded (Client.Timeout exceeded)
```

**Solutions:**
```bash
# Check upstream connectivity
curl -I https://repo1.maven.org/maven2/
curl -I https://registry.npmjs.org/

# Increase timeout in configuration
timeout: 60s

# Use debug logging to trace requests
LOG_LEVEL=debug make dev-run
```

### 3. Cache-Related Issues

#### Issue: Cache Directory Not Created
```
Error: failed to create cache directory
```

**Solution:**
```bash
# Ensure directory exists and is writable
mkdir -p $STORAGE_DIR
chmod 755 $STORAGE_DIR

# Check ownership
ls -la $(dirname $STORAGE_DIR)

# In tests, use t.TempDir()
cacheDir := t.TempDir()
```

#### Issue: Cache Corruption
```
Error: invalid cache entry format
```

**Solutions:**
```bash
# Clear cache directory
rm -rf $STORAGE_DIR/*

# Verify cache backend configuration
curl http://localhost:8081/api/v1/config/cache

# Check disk space
df -h $STORAGE_DIR
```

### 4. Test Execution Issues

#### Issue: Tests Timing Out
```
panic: test timed out after 10m0s
```

**Solutions:**
```go
// Increase test timeout
func TestLongRunning(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long-running test")
    }

    // Use context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
}

// Run with longer timeout
go test -timeout 30m ./...
```

#### Issue: Race Condition Detected
```
WARNING: DATA RACE
Read at 0x... by goroutine ...
```

**Solutions:**
```go
// Add proper synchronization
type SafeCounter struct {
    mu    sync.RWMutex
    value int
}

func (c *SafeCounter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

// Run race detection regularly
go test -race ./...
```

### 5. Mock and Test Double Issues

#### Issue: Mock Expectations Not Met
```
Error: mock expectations were not met: expected call to Get()
```

**Solutions:**
```go
// Ensure all expectations are set
mockCache := new(MockCacheService)
mockCache.On("Get", mock.Anything, "test-key").Return(nil, false, nil)
mockCache.On("Put", mock.Anything, "test-key", mock.Anything, mock.Anything).Return(nil)

// Use mock.Anything for flexible matching
mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)

// Verify expectations at end of test
defer mockCache.AssertExpectations(t)
```

#### Issue: HTTP Mock Server Not Responding
```
Error: Get "http://127.0.0.1:12345": connection refused
```

**Solutions:**
```go
// Ensure server is started properly
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("mock response"))
}))
defer server.Close()

// Use server.URL in tests
resp, err := http.Get(server.URL + "/test")
```

### 6. Integration Test Issues

#### Issue: Docker/External Service Not Available
```
Error: Cannot connect to the Docker daemon
```

**Solutions:**
```bash
# Check Docker status
systemctl status docker

# Use test containers for integration tests
func TestWithDocker(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Docker integration test")
    }

    // Use testcontainers-go
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "redis:7-alpine",
            ExposedPorts: []string{"6379/tcp"},
        },
        Started: true,
    })
}
```

#### Issue: Test Data Inconsistency
```
Error: expected 100 bytes, got 95 bytes
```

**Solutions:**
```go
// Use deterministic test data
func generateTestData(size int) []byte {
    data := make([]byte, size)
    for i := range data {
        data[i] = byte(i % 256)
    }
    return data
}

// Verify test data integrity
testData := generateTestData(1000)
assert.Len(t, testData, 1000)

// Use checksums for verification
expectedHash := sha256.Sum256(testData)
actualHash := sha256.Sum256(receivedData)
assert.Equal(t, expectedHash, actualHash)
```

## Debugging Techniques

### 1. Verbose Logging

```bash
# Enable debug logging
LOG_LEVEL=debug go test -v ./...

# Enable trace logging for specific requests
curl -H "X-Trace-Level: debug" http://localhost:8081/proxy/maven/...
```

### 2. Test Isolation

```go
// Isolate tests with separate instances
func TestWithIsolation(t *testing.T) {
    t.Parallel() // Run in parallel with other parallel tests

    // Use separate temp directories
    tempDir := t.TempDir()

    // Use separate ports
    port := getRandomPort()

    // Clean up resources
    t.Cleanup(func() {
        // Cleanup code
    })
}
```

### 3. State Verification

```go
// Verify system state after operations
func TestStateVerification(t *testing.T) {
    // Perform operation
    err := service.Process("input")
    require.NoError(t, err)

    // Verify expected state changes
    assert.True(t, service.IsProcessed())
    assert.Equal(t, 1, service.GetProcessCount())

    // Verify side effects
    files, err := os.ReadDir(tempDir)
    require.NoError(t, err)
    assert.Len(t, files, 1)
}
```

### 4. Performance Profiling

```bash
# Generate CPU profile during test
go test -cpuprofile=cpu.prof -run TestSpecific ./...

# Generate memory profile
go test -memprofile=mem.prof -run TestSpecific ./...

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof

# Web interface
go tool pprof -http=:8080 cpu.prof
```

## Environment-Specific Issues

### 1. CI/CD Environment

#### Issue: Tests Pass Locally But Fail in CI
```
Error: timeout waiting for condition
```

**Solutions:**
```yaml
# Increase timeouts in CI
- name: Run tests
  run: go test -timeout 30m ./...

# Use retry mechanisms
- name: Run tests with retry
  uses: nick-invision/retry@v2
  with:
    timeout_minutes: 10
    max_attempts: 3
    command: go test ./...
```

#### Issue: Resource Limits in CI
```
Error: cannot allocate memory
```

**Solutions:**
```yaml
# Use appropriate runner size
runs-on: ubuntu-latest-4-cores

# Configure resource limits
env:
  GOMAXPROCS: 2

# Reduce test parallelism
- name: Run tests
  run: go test -p 1 ./...
```

### 2. Local Development

#### Issue: Port Conflicts
```
Error: bind: address already in use
```

**Solutions:**
```bash
# Find process using port
lsof -i :8081
netstat -tulpn | grep 8081

# Kill process
kill -9 <PID>

# Use random ports in tests
func getRandomPort() int {
    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        panic(err)
    }
    defer listener.Close()
    return listener.Addr().(*net.TCPAddr).Port
}
```

#### Issue: File System Permissions
```
Error: operation not permitted
```

**Solutions:**
```bash
# Check file permissions
ls -la $STORAGE_DIR

# Fix permissions
chmod -R 755 $STORAGE_DIR
chown -R $USER:$GROUP $STORAGE_DIR

# Use user-specific directories
STORAGE_DIR="$HOME/.proxynd-test"
```

## Debugging Tools and Techniques

### 1. HTTP Request Debugging

```bash
# Use verbose curl
curl -v http://localhost:8081/proxy/maven/...

# Use httpie for better formatting
http GET localhost:8081/proxy/maven/...

# Capture network traffic
tcpdump -i lo -A -s 0 port 8081

# Use proxy for request inspection
export http_proxy=http://localhost:8888
export https_proxy=http://localhost:8888
# Run Charles, Burp, or mitmproxy on port 8888
```

### 2. Log Analysis

```bash
# Follow logs in real-time
tail -f logs/proxynd.log

# Search for specific patterns
grep "ERROR" logs/proxynd.log
grep "proxy.*maven" logs/proxynd.log | tail -20

# Analyze performance
grep "duration" logs/proxynd.log | awk '{print $NF}' | sort -n
```

### 3. Database/Cache Inspection

```bash
# Check cache contents
find $STORAGE_DIR -name "*.cache" -ls

# Verify cache file integrity
file $STORAGE_DIR/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar

# Check file sizes
du -sh $STORAGE_DIR/*
```

## Prevention Strategies

### 1. Test Design Best Practices

```go
// Use table-driven tests for comprehensive coverage
func TestService_Process(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        setupFunc   func(t *testing.T) *Service
        want        string
        wantErr     bool
        wantErrType error
    }{
        // Test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 2. Resource Management

```go
// Always clean up resources
func TestWithResources(t *testing.T) {
    // Setup
    server := startTestServer(t)
    client := createTestClient(t)

    // Cleanup
    t.Cleanup(func() {
        server.Close()
        client.Close()
    })

    // Test implementation
}
```

### 3. Error Handling

```go
// Test error conditions explicitly
func TestService_ErrorConditions(t *testing.T) {
    tests := []struct {
        name      string
        setupFunc func() *Service
        wantErr   string
    }{
        {
            name: "invalid input",
            setupFunc: func() *Service {
                return NewService(invalidConfig)
            },
            wantErr: "invalid configuration",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := tt.setupFunc()
            err := service.Process("input")

            assert.Error(t, err)
            assert.Contains(t, err.Error(), tt.wantErr)
        })
    }
}
```

### 4. Monitoring and Observability

```go
// Add instrumentation to tests
func TestWithMetrics(t *testing.T) {
    // Start metrics collection
    metrics := startMetricsCollection(t)
    defer metrics.Stop()

    // Perform operations
    err := service.Process("input")
    require.NoError(t, err)

    // Verify metrics
    assert.Greater(t, metrics.GetCounter("operations_total"), 0.0)
    assert.Less(t, metrics.GetHistogram("operation_duration"), 1.0)
}
```

## Escalation Procedures

### 1. When to Escalate

- Tests fail consistently across multiple environments
- Performance degrades significantly without code changes
- Security vulnerabilities discovered during testing
- Data corruption or loss detected

### 2. Information to Collect

- Test logs and error messages
- System resource usage during failure
- Network traces (if applicable)
- Configuration files and environment variables
- Steps to reproduce the issue

### 3. Documentation

- Update troubleshooting guide with new issues
- Document workarounds and permanent fixes
- Share knowledge with team members
- Create runbooks for common procedures
