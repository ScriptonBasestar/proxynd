# Performance Testing Guide

## Overview

This guide covers performance testing strategies for ProxyND, including load testing, benchmarking, and performance monitoring.

## Performance Test Categories

### 1. Load Testing
Testing system behavior under expected and peak loads.

### 2. Stress Testing  
Testing system limits and failure points.

### 3. Benchmark Testing
Measuring specific operation performance.

### 4. Endurance Testing
Testing system stability over extended periods.

## Load Testing

### Basic Load Test Setup

```bash
# Install wrk (HTTP benchmarking tool)
# Ubuntu/Debian
sudo apt-get install wrk

# macOS
brew install wrk

# Or use alternatives: ab, curl-loader, etc.
```

### Maven Proxy Load Test

```bash
# Test Maven artifact downloads
wrk -t4 -c100 -d30s --timeout 30s \
    "http://localhost:8081/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"

# Test with multiple artifacts
wrk -t8 -c200 -d60s --script=maven-load-test.lua \
    "http://localhost:8081/proxy/maven/"
```

Create `maven-load-test.lua`:

```lua
-- maven-load-test.lua
artifacts = {
    "/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar",
    "/com/google/guava/guava/31.1-jre/guava-31.1-jre.jar",
    "/org/slf4j/slf4j-api/1.7.36/slf4j-api-1.7.36.jar",
    "/junit/junit/4.13.2/junit-4.13.2.jar",
    "/org/springframework/spring-core/5.3.27/spring-core-5.3.27.jar"
}

request = function()
    artifact = artifacts[math.random(#artifacts)]
    return wrk.format("GET", artifact)
end
```

### NPM Proxy Load Test

```bash
# Test NPM package downloads
wrk -t4 -c100 -d30s \
    "http://localhost:8081/proxy/npm/express/-/express-4.18.0.tgz"

# Test package metadata requests
wrk -t4 -c50 -d30s \
    "http://localhost:8081/proxy/npm/express"
```

### Concurrent Proxy Type Testing

```bash
#!/bin/bash
# concurrent-load-test.sh

# Start load tests for all proxy types concurrently
echo "Starting concurrent load tests..."

# Maven load test
wrk -t2 -c50 -d60s \
    "http://localhost:8081/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar" \
    > maven-load-results.txt &

# NPM load test  
wrk -t2 -c50 -d60s \
    "http://localhost:8081/proxy/npm/express/-/express-4.18.0.tgz" \
    > npm-load-results.txt &

# APT load test
wrk -t2 -c50 -d60s \
    "http://localhost:8081/proxy/apt/dists/focal/Release" \
    > apt-load-results.txt &

# Wait for all tests to complete
wait

echo "Load tests completed. Check *-load-results.txt files."
```

## Benchmarking

### Go Benchmark Tests

```go
// benchmarks/proxy_benchmark_test.go
func BenchmarkMavenProxy_ArtifactDownload(b *testing.B) {
    server := startBenchmarkServer(b)
    defer server.Close()

    client := &http.Client{Timeout: 30 * time.Second}
    url := server.URL + "/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"

    b.ResetTimer()
    b.ReportAllocs()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            resp, err := client.Get(url)
            if err != nil {
                b.Error(err)
                continue
            }

            io.Copy(io.Discard, resp.Body)
            resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                b.Errorf("Expected 200, got %d", resp.StatusCode)
            }
        }
    })
}

func BenchmarkCacheService_Operations(b *testing.B) {
    cache := setupBenchmarkCache(b)
    defer cache.Close()

    testData := make([]byte, 1024*1024) // 1MB test data
    rand.Read(testData)

    b.Run("Put", func(b *testing.B) {
        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            key := fmt.Sprintf("benchmark-key-%d", i)
            err := cache.Put(context.Background(), key, bytes.NewReader(testData), time.Hour)
            if err != nil {
                b.Error(err)
            }
        }
    })

    b.Run("Get", func(b *testing.B) {
        // Pre-populate cache
        key := "benchmark-get-key"
        cache.Put(context.Background(), key, bytes.NewReader(testData), time.Hour)

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            reader, exists, err := cache.Get(context.Background(), key)
            if err != nil {
                b.Error(err)
                continue
            }
            if !exists {
                b.Error("Key should exist")
                continue
            }

            io.Copy(io.Discard, reader)
            reader.Close()
        }
    })
}
```

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./benchmarks/

# Run specific benchmark
go test -bench=BenchmarkMavenProxy -benchmem ./benchmarks/

# Generate CPU profile
go test -bench=BenchmarkMavenProxy -cpuprofile=cpu.prof ./benchmarks/

# Generate memory profile
go test -bench=BenchmarkCacheService -memprofile=mem.prof ./benchmarks/

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Memory and Resource Testing

### Memory Leak Detection

```bash
#!/bin/bash
# memory-leak-test.sh

echo "Starting memory leak detection test..."

# Start server with memory profiling
go run -race ./cmd/proxynd/main.go &
SERVER_PID=$!

# Wait for server to start
sleep 5

# Measure initial memory
ps -p $SERVER_PID -o pid,vsz,rss,pmem

# Run load test
for i in {1..1000}; do
    curl -s "http://localhost:8081/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar" > /dev/null

    if [ $((i % 100)) -eq 0 ]; then
        echo "Completed $i requests"
        ps -p $SERVER_PID -o pid,vsz,rss,pmem
    fi
done

# Final memory measurement
echo "Final memory usage:"
ps -p $SERVER_PID -o pid,vsz,rss,pmem

# Cleanup
kill $SERVER_PID
```

### Resource Monitoring Script

```bash
#!/bin/bash
# monitor-resources.sh

LOG_FILE="performance-monitor.log"
INTERVAL=5  # seconds

echo "Starting resource monitoring (interval: ${INTERVAL}s)"
echo "Timestamp,CPU%,Memory(MB),Goroutines,File Descriptors" > $LOG_FILE

while true; do
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Get CPU and memory from ps
    ps_output=$(ps -p $(pgrep proxynd) -o pcpu,rss --no-headers 2>/dev/null)
    if [ $? -eq 0 ]; then
        cpu=$(echo $ps_output | awk '{print $1}')
        memory_kb=$(echo $ps_output | awk '{print $2}')
        memory_mb=$((memory_kb / 1024))
    else
        cpu="N/A"
        memory_mb="N/A"
    fi

    # Get goroutines from debug endpoint
    goroutines=$(curl -s http://localhost:8081/debug/pprof/goroutine?debug=1 2>/dev/null | grep -c "goroutine" || echo "N/A")

    # Get file descriptors
    fd_count=$(lsof -p $(pgrep proxynd) 2>/dev/null | wc -l || echo "N/A")

    echo "$timestamp,$cpu,$memory_mb,$goroutines,$fd_count" >> $LOG_FILE
    echo "[$timestamp] CPU: ${cpu}%, Memory: ${memory_mb}MB, Goroutines: $goroutines, FDs: $fd_count"

    sleep $INTERVAL
done
```

## Cache Performance Testing

### Cache Hit Ratio Analysis

```bash
#!/bin/bash
# cache-performance-test.sh

PROXY_URL="http://localhost:8081/proxy/maven"
ARTIFACTS=(
    "org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"
    "com/google/guava/guava/31.1-jre/guava-31.1-jre.jar"
    "org/slf4j/slf4j-api/1.7.36/slf4j-api-1.7.36.jar"
)

echo "Cache Performance Analysis"
echo "========================="

# Clear cache
rm -rf $STORAGE_DIR/maven/* 2>/dev/null

total_requests=0
cache_hits=0

for round in {1..5}; do
    echo "Round $round:"

    for artifact in "${ARTIFACTS[@]}"; do
        # Check if cached
        if [ -f "$STORAGE_DIR/maven/$artifact" ]; then
            cache_hits=$((cache_hits + 1))
            echo "  ✓ Cache HIT: $artifact"
        else
            echo "  ✗ Cache MISS: $artifact"
        fi

        # Download (will cache if not cached)
        start_time=$(date +%s.%N)
        curl -s -o /dev/null "$PROXY_URL/$artifact"
        end_time=$(date +%s.%N)

        duration=$(echo "$end_time - $start_time" | bc -l)
        printf "    Download time: %.3fs\n" $duration

        total_requests=$((total_requests + 1))
    done

    echo ""
done

# Calculate hit ratio
hit_ratio=$(echo "scale=2; $cache_hits * 100 / $total_requests" | bc -l)
echo "Cache Hit Ratio: ${hit_ratio}% ($cache_hits/$total_requests)"
```

### Cache Eviction Testing

```go
// Test cache eviction under memory pressure
func TestCacheEviction_Performance(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping cache eviction test in short mode")
    }

    cache := setupTestCache(t, &config.CacheConfig{
        Backend: "file",
        File: config.FileCacheConfig{
            Directory:   t.TempDir(),
            MaxSize:     10 * 1024 * 1024, // 10MB limit
        },
    })
    defer cache.Close()

    // Generate test data larger than cache size
    largeData := make([]byte, 2*1024*1024) // 2MB chunks
    rand.Read(largeData)

    start := time.Now()

    // Fill cache beyond capacity
    for i := 0; i < 10; i++ { // 20MB total, cache limit 10MB
        key := fmt.Sprintf("large-item-%d", i)
        err := cache.Put(context.Background(), key, bytes.NewReader(largeData), time.Hour)
        require.NoError(t, err)
    }

    duration := time.Since(start)
    t.Logf("Cache eviction test completed in %v", duration)

    // Verify cache size is within limits
    cacheSize := getCacheSize(t, cache)
    assert.LessOrEqual(t, cacheSize, int64(12*1024*1024)) // Allow some overhead
}
```

## CI/CD Performance Integration

### GitHub Actions Performance Test

```yaml
# .github/workflows/performance.yml
name: Performance Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:

jobs:
  performance-test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install tools
        run: |
          sudo apt-get update
          sudo apt-get install -y wrk bc

      - name: Start ProxyND
        run: |
          make dev-setup
          make dev-run &
          sleep 10
        env:
          CONFIG_DIR: ./tmp/config
          STORAGE_DIR: ./tmp/storage
          SERVER_PORT: 8081

      - name: Run load tests
        run: |
          # Maven load test
          wrk -t4 -c50 -d30s \
              "http://localhost:8081/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar" \
              > maven-load-results.txt

          # NPM load test
          wrk -t4 -c50 -d30s \
              "http://localhost:8081/proxy/npm/express/-/express-4.18.0.tgz" \
              > npm-load-results.txt

      - name: Run benchmarks
        run: |
          go test -bench=. -benchmem ./benchmarks/ > benchmark-results.txt

      - name: Analyze results
        run: |
          # Extract key metrics
          echo "## Load Test Results" >> performance-summary.md
          echo "### Maven Proxy" >> performance-summary.md
          grep "Requests/sec" maven-load-results.txt >> performance-summary.md

          echo "### NPM Proxy" >> performance-summary.md  
          grep "Requests/sec" npm-load-results.txt >> performance-summary.md

          echo "## Benchmark Results" >> performance-summary.md
          grep "Benchmark" benchmark-results.txt >> performance-summary.md

      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: performance-results
          path: |
            *-results.txt
            performance-summary.md

      - name: Performance regression check
        run: |
          # Compare with baseline (implement regression detection)
          ./scripts/check-performance-regression.sh
```

### Performance Regression Detection

```bash
#!/bin/bash
# scripts/check-performance-regression.sh

BASELINE_FILE="performance-baseline.json"
CURRENT_RESULTS="benchmark-results.txt"
THRESHOLD=20  # 20% regression threshold

if [ ! -f "$BASELINE_FILE" ]; then
    echo "No baseline found, creating baseline from current results"
    # Parse current results and create baseline
    # Implementation depends on your specific metrics format
    exit 0
fi

# Extract key metrics from current results
CURRENT_RPS=$(grep "maven.*Requests/sec" maven-load-results.txt | awk '{print $2}')
CURRENT_LATENCY=$(grep "maven.*Latency" maven-load-results.txt | awk '{print $2}')

# Compare with baseline
BASELINE_RPS=$(jq -r '.maven.requests_per_second' $BASELINE_FILE)
BASELINE_LATENCY=$(jq -r '.maven.latency' $BASELINE_FILE)

# Calculate percentage changes
RPS_CHANGE=$(echo "scale=2; ($CURRENT_RPS - $BASELINE_RPS) * 100 / $BASELINE_RPS" | bc -l)
LATENCY_CHANGE=$(echo "scale=2; ($CURRENT_LATENCY - $BASELINE_LATENCY) * 100 / $BASELINE_LATENCY" | bc -l)

echo "Performance comparison:"
echo "RPS: $CURRENT_RPS (${RPS_CHANGE}% change from baseline)"
echo "Latency: $CURRENT_LATENCY (${LATENCY_CHANGE}% change from baseline)"

# Check for regressions
if (( $(echo "$RPS_CHANGE < -$THRESHOLD" | bc -l) )); then
    echo "❌ Performance regression detected: RPS decreased by ${RPS_CHANGE}%"
    exit 1
fi

if (( $(echo "$LATENCY_CHANGE > $THRESHOLD" | bc -l) )); then
    echo "❌ Performance regression detected: Latency increased by ${LATENCY_CHANGE}%"
    exit 1
fi

echo "✅ Performance within acceptable range"
```

## Monitoring and Alerting

### Prometheus Metrics for Performance

```go
// Key performance metrics to monitor
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "proxynd_request_duration_seconds",
            Help: "Request duration in seconds",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
        },
        []string{"proxy_type", "status"},
    )

    cacheHitRatio = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "proxynd_cache_hit_ratio",
            Help: "Cache hit ratio by proxy type",
        },
        []string{"proxy_type"},
    )

    memoryUsage = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "proxynd_memory_usage_bytes",
            Help: "Current memory usage in bytes",
        },
    )
)
```

### Performance Dashboard

Create Grafana dashboard with key metrics:

1. **Request Rate** (req/sec)
2. **Response Time** (percentiles)
3. **Cache Hit Ratio** (%)
4. **Memory Usage** (MB)
5. **Error Rate** (%)
6. **Concurrent Connections**

### Alerting Rules

```yaml
# prometheus-alerts.yml
groups:
  - name: proxynd-performance
    rules:
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, proxynd_request_duration_seconds) > 1.0
        for: 5m
        annotations:
          summary: "High response time detected"

      - alert: LowCacheHitRatio  
        expr: proxynd_cache_hit_ratio < 0.7
        for: 10m
        annotations:
          summary: "Cache hit ratio below threshold"

      - alert: HighMemoryUsage
        expr: proxynd_memory_usage_bytes > 1000000000  # 1GB
        for: 5m
        annotations:
          summary: "High memory usage detected"
```

## Best Practices

### 1. Test Environment

- Use dedicated performance test environment
- Ensure consistent hardware/network conditions
- Isolate from other services

### 2. Baseline Establishment

- Create performance baselines for each proxy type
- Document expected performance characteristics
- Update baselines when significant changes occur

### 3. Continuous Monitoring

- Run performance tests regularly
- Monitor key metrics in production
- Set up alerting for performance regressions

### 4. Optimization Guidelines

- Focus on cache hit ratio optimization
- Monitor memory usage patterns
- Optimize for concurrent request handling
- Consider connection pooling and reuse
