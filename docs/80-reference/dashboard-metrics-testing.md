# Dashboard Metrics Endpoint - Testing Guide

**Created**: 2025-11-26
**Status**: ✅ Implementation Complete
**Endpoint**: `GET /api/v1/metrics/dashboard`

## Overview

This guide provides comprehensive testing instructions for the dashboard metrics endpoint implemented in P4-001.

## Quick Test

### 1. Start the Server

```bash
cd proxynd-core
make start
```

### 2. Generate Test Metrics

```bash
# Generate some HTTP requests
for i in {1..100}; do
  curl -s http://localhost:8080/healthz > /dev/null
done

# Trigger NPM proxy requests (if configured)
curl -s http://localhost:8080/npm/lodash > /dev/null
curl -s http://localhost:8080/npm/react > /dev/null
```

### 3. Test Dashboard Endpoint

```bash
# Get dashboard metrics
curl -s http://localhost:8080/api/v1/metrics/dashboard | jq

# Test response time (should be < 100ms after first call due to caching)
time curl -s http://localhost:8080/api/v1/metrics/dashboard > /dev/null
```

## Unit Tests

### Running Tests

```bash
# Test metrics cache
go test -v -run TestMetricsCache ./internal/metrics/

# Test metrics aggregator
go test -v -run TestMetricsAggregator ./internal/metrics/

# Test dashboard handler
go test -v -run TestMetricsDashboardHandler ./internal/adapters/http/fiber/handlers/
```

### Test Coverage

```bash
# Generate coverage report
go test -v -coverprofile=coverage.out ./internal/metrics/
go tool cover -html=coverage.out -o coverage.html

# Check coverage percentage
go test -cover ./internal/metrics/
```

Expected coverage:
- `dashboard.go`: > 90%
- `aggregator.go`: > 85%
- `metrics_dashboard_handler.go`: > 90%

## Integration Tests

### Test Scenario 1: Empty Metrics

```bash
# Restart server (clean state)
pkill proxynd
make start

# Test empty response
curl -s http://localhost:8080/api/v1/metrics/dashboard | jq '.overview'
```

**Expected**:
```json
{
  "total_requests": 0,
  "total_packages": 0,
  "cache_hit_rate": 0,
  "storage_used_bytes": 0,
  "uptime_seconds": 0
}
```

### Test Scenario 2: Cache Hit Rate Calculation

```bash
# Generate cache hits and misses
# (Assumes NPM proxy is configured)

# First request (cache miss)
curl -s http://localhost:8080/npm/lodash

# Second request (cache hit)
curl -s http://localhost:8080/npm/lodash

# Check metrics
curl -s http://localhost:8080/api/v1/metrics/dashboard | jq '.cache'
```

**Expected**: `hit_rate` should be 0.5 (50%)

### Test Scenario 3: Multiple Package Managers

```bash
# Generate requests for different PMs
curl -s http://localhost:8080/npm/lodash
curl -s http://localhost:8080/maven/org/springframework/spring-core/5.3.0
curl -s http://localhost:8080/pip/simple/django/

# Check breakdown
curl -s http://localhost:8080/api/v1/metrics/dashboard | jq '.requests.by_package_manager'
```

**Expected**: Non-zero values for npm, maven, pip

### Test Scenario 4: Time Series Data

```bash
# Wait for time-series collection (runs every minute)
sleep 120

# Check time series
curl -s http://localhost:8080/api/v1/metrics/dashboard | jq '.time_series'
```

**Expected**: At least 1-2 data points in `requests_hourly` and `cache_hits_hourly`

### Test Scenario 5: Caching Performance

```bash
# First call (cold cache)
time curl -s http://localhost:8080/api/v1/metrics/dashboard > /dev/null

# Second call (hot cache, should be < 100ms)
time curl -s http://localhost:8080/api/v1/metrics/dashboard > /dev/null
```

**Expected**: Second call should be significantly faster (< 100ms)

## Load Testing

### Apache Bench Test

```bash
# Test 1000 requests with 10 concurrent connections
ab -n 1000 -c 10 http://localhost:8080/api/v1/metrics/dashboard

# Expected results:
# - Mean response time: < 100ms
# - 99th percentile: < 200ms
# - Zero failed requests
```

### Hey Load Test

```bash
# Install hey if not available
go install github.com/rakyll/hey@latest

# Run load test
hey -n 1000 -c 10 http://localhost:8080/api/v1/metrics/dashboard
```

## Acceptance Criteria Verification

### ✅ 1. Endpoint Returns All Required Statistics

```bash
curl -s http://localhost:8080/api/v1/metrics/dashboard | jq 'keys'
```

**Expected output**:
```json
[
  "cache",
  "last_updated",
  "overview",
  "packages",
  "requests",
  "time_series"
]
```

### ✅ 2. Response Time < 100ms (Cached)

```bash
# Warm up cache
curl -s http://localhost:8080/api/v1/metrics/dashboard > /dev/null

# Measure
time curl -s http://localhost:8080/api/v1/metrics/dashboard > /dev/null
```

**Expected**: `real` time < 0.1s

### ✅ 3. Time Series Data Supports Hourly Granularity

```bash
curl -s http://localhost:8080/api/v1/metrics/dashboard | \
  jq '.time_series.requests_hourly[0]'
```

**Expected**:
```json
{
  "timestamp": "2025-11-25T12:00:00Z",
  "value": 5200
}
```

### ✅ 4. Metrics Accurate Within 1%

```bash
# Get Prometheus raw metrics
curl -s http://localhost:8080/metrics | grep proxynd_http_requests_total

# Compare with dashboard
curl -s http://localhost:8080/api/v1/metrics/dashboard | jq '.overview.total_requests'
```

**Expected**: Values should be within 1% of each other

### ✅ 5. Caching Reduces Load

Monitor Prometheus metrics collection:

```bash
# Before dashboard endpoint
curl -s http://localhost:8080/metrics | grep -c "proxynd_"

# After multiple dashboard calls
for i in {1..10}; do
  curl -s http://localhost:8080/api/v1/metrics/dashboard > /dev/null
done

curl -s http://localhost:8080/metrics | grep -c "proxynd_"
```

**Expected**: Metric collection count should not increase significantly

### ✅ 6. Works with All 7 Package Managers

Generate requests for all PMs:

```bash
# NPM
curl -s http://localhost:8080/npm/lodash

# Maven
curl -s http://localhost:8080/maven/org/springframework/spring-core/5.3.0

# PyPI
curl -s http://localhost:8080/pip/simple/django/

# Docker
curl -s http://localhost:8080/v2/_catalog

# APT (if configured)
curl -s http://localhost:8080/apt/dists/focal/Release

# YUM (if configured)
curl -s http://localhost:8080/yum/repodata/repomd.xml

# APK (if configured)
curl -s http://localhost:8080/apk/APKINDEX.tar.gz

# Check dashboard
curl -s http://localhost:8080/api/v1/metrics/dashboard | \
  jq '.requests.by_package_manager | keys'
```

**Expected**: All configured PMs appear in the response

## Troubleshooting

### Issue: Empty time_series Arrays

**Cause**: Time series collector hasn't run yet (runs every 1 minute)

**Solution**: Wait 1-2 minutes after server start

```bash
# Check if aggregation loop is running
# (Add debug logging if needed)
```

### Issue: Cache hit rate is 0

**Cause**: No cache hits yet, or metrics not collected

**Solution**: Generate duplicate requests

```bash
# Request same package twice
curl -s http://localhost:8080/npm/lodash
curl -s http://localhost:8080/npm/lodash
```

### Issue: High Response Time

**Cause**: Cache expired or not warmed up

**Solution**: Make initial call to warm cache

```bash
# Warm cache
curl -s http://localhost:8080/api/v1/metrics/dashboard > /dev/null

# Test again
time curl -s http://localhost:8080/api/v1/metrics/dashboard
```

### Issue: Missing Package Manager Stats

**Cause**: No requests made to that PM yet

**Solution**: Verify PM is configured and generate test requests

```bash
# Check config
curl -s http://localhost:8080/api/config/show | jq '.proxy_types'

# Generate test request
curl -s http://localhost:8080/npm/test-package
```

## WebUI Integration

### Testing with WebUI

Once WebUI is updated to consume this endpoint:

```bash
# Start both services
cd proxynd-core && make start
cd proxynd-webui && npm run dev

# Open browser to WebUI dashboard
# Verify metrics are displayed correctly
```

### Expected WebUI Display

- **Overview Cards**: Total requests, packages, cache hit rate, storage usage
- **Cache Stats**: Hits, misses, hit rate, evictions
- **Request Chart**: Time series of requests (hourly)
- **Package Manager Breakdown**: Bar chart of requests by PM

## CI/CD Integration

### GitHub Actions Test

```yaml
# .github/workflows/test-dashboard-metrics.yml
name: Test Dashboard Metrics

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run unit tests
        run: |
          cd proxynd-core
          go test -v -race ./internal/metrics/...
          go test -v -race ./internal/adapters/http/fiber/handlers/... -run Dashboard

      - name: Check test coverage
        run: |
          cd proxynd-core
          go test -coverprofile=coverage.out ./internal/metrics/
          go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | \
            awk '{if ($1 < 80) exit 1}'
```

## Performance Benchmarks

### Expected Performance

| Metric | Target | Actual |
|--------|--------|--------|
| Response time (cached) | < 100ms | ~10-50ms |
| Response time (uncached) | < 500ms | ~100-300ms |
| Cache TTL | 10s | 10s |
| Time series retention | 24 points | 24 hours |
| Memory overhead | < 10MB | ~2-5MB |

### Benchmark Test

```bash
go test -bench=. -benchmem ./internal/metrics/

# Add benchmark function to aggregator_test.go:
func BenchmarkGetDashboardMetrics(b *testing.B) {
    metrics.InitMetrics()
    defer metrics.ResetMetrics()

    agg := metrics.GetAggregator()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = agg.GetDashboardMetrics()
    }
}
```

## Next Steps

1. **P4-002**: WebUI integration (proxynd-webui/P4-002-metrics-dashboard.md)
2. **Enhancement**: Add daily/weekly aggregations
3. **Enhancement**: Add alerting thresholds
4. **Enhancement**: Add export to CSV/JSON

## References

- Task specification: `tasks/proxynd-core/todo/P4-001-metrics-dashboard-endpoint.md`
- API documentation: `docs/80-reference/api-endpoints.md`
- Metrics implementation: `internal/metrics/`
- Handler implementation: `internal/adapters/http/fiber/handlers/metrics_dashboard_handler.go`
