# ProxyND Enterprise API - Performance Benchmarks

Performance benchmarking guide and results for the Enterprise API endpoints.

## 🎯 Performance Targets

ProxyND Enterprise API is designed to meet strict performance requirements:

| Metric | Target | Description |
|--------|--------|-------------|
| **P95 Response Time** | < 500ms | 95% of requests complete in under 500ms |
| **P99 Response Time** | < 1000ms | 99% of requests complete in under 1 second |
| **Page Load Time** | < 3s | Complete page load (WebUI) under 3 seconds |
| **Throughput** | > 100 req/s | Sustained request rate per endpoint |
| **Concurrent Users** | 50+ | Support 50+ simultaneous users |

## 🚀 Quick Start

### Running Benchmarks

```bash
# Start the server in one terminal
make dev-run

# Run benchmarks in another terminal
make test-benchmark-enterprise

# Quick benchmark (fewer iterations)
make test-benchmark-enterprise-quick

# Stress test (many iterations, high concurrency)
make test-benchmark-enterprise-stress
```

### Custom Configuration

```bash
# Custom iterations and concurrency
ITERATIONS=200 CONCURRENT=30 ./scripts/benchmark-enterprise-api.sh

# Test specific server
PROXYND_URL=https://staging.example.com ./scripts/benchmark-enterprise-api.sh
```

## 📊 Benchmark Methodology

### Test Categories

1. **Single Endpoint Tests**
   - Measures individual endpoint response times
   - 100 iterations per endpoint (default)
   - Calculates min, avg, max, P50, P95, P99

2. **Pagination Tests**
   - Tests pagination performance
   - Fetches 10 pages sequentially
   - Measures average time per page

3. **Concurrent Request Tests**
   - Tests server under concurrent load
   - 20 simultaneous requests (default)
   - Measures total time and average per request

### Endpoints Tested

#### RBAC (3 endpoints)
- `GET /rbac/roles` - List all roles
- `GET /rbac/roles/:id` - Get specific role
- `GET /rbac/permissions` - List permissions

#### Audit (2 endpoints)
- `GET /audit/events` - List audit events
- `GET /audit/stats` - Get statistics

#### Analytics (3 endpoints)
- `GET /analytics/overview` - Dashboard overview
- `GET /analytics/usage` - Usage statistics
- `GET /analytics/performance` - Performance metrics

#### Security (2 endpoints)
- `GET /security/vulnerabilities` - List vulnerabilities
- `GET /security/licenses` - List detected licenses

#### Alerts (1 endpoint)
- `GET /alerts` - List active alerts

#### License (2 endpoints)
- `GET /license/info` - Get license information
- `GET /license/features` - List available features

## 📈 Expected Results

### Baseline Performance (Development Mode)

With mock data and no database:

```
RBAC Endpoints:
  List Roles: P95 ~50ms, P99 ~100ms
  Get Role: P95 ~30ms, P99 ~60ms
  List Permissions: P95 ~40ms, P99 ~80ms

Audit Endpoints:
  List Events: P95 ~60ms, P99 ~120ms
  Get Stats: P95 ~50ms, P99 ~100ms

Analytics Endpoints:
  Get Overview: P95 ~70ms, P99 ~140ms
  Get Usage Stats: P95 ~80ms, P99 ~160ms
  Get Performance: P95 ~70ms, P99 ~140ms

Security Endpoints:
  List Vulnerabilities: P95 ~60ms, P99 ~120ms
  List Licenses: P95 ~50ms, P99 ~100ms

Alerts Endpoints:
  List Alerts: P95 ~55ms, P99 ~110ms

License Endpoints:
  Get License Info: P95 ~40ms, P99 ~80ms
  List Features: P95 ~45ms, P99 ~90ms
```

### Production Performance (With Database)

Expected performance with PostgreSQL/Redis:

```
Most endpoints: P95 ~150-300ms, P99 ~300-600ms
Complex queries: P95 ~200-400ms, P99 ~400-800ms
```

## 🔧 Optimization Strategies

### 1. Database Query Optimization

**Problem**: Slow database queries
**Solution**:
- Add indexes on frequently queried columns
- Use connection pooling
- Implement query result caching

```sql
-- Example: Add index for audit event queries
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp);
CREATE INDEX idx_audit_events_resource ON audit_events(resource_type, resource_id);
```

### 2. Response Caching

**Problem**: Repeated requests for same data
**Solution**:
- Cache frequently accessed data in Redis
- Set appropriate TTLs
- Invalidate cache on updates

```go
// Example: Cache analytics overview
const CACHE_TTL = 60 // seconds

func (h *AnalyticsHandler) GetOverview(c *fiber.Ctx) error {
    cacheKey := "analytics:overview"

    // Try cache first
    if cached := cache.Get(cacheKey); cached != nil {
        return c.JSON(cached)
    }

    // Generate and cache
    overview := generateOverview()
    cache.Set(cacheKey, overview, CACHE_TTL)

    return c.JSON(overview)
}
```

### 3. Pagination Optimization

**Problem**: Large result sets slow down responses
**Solution**:
- Limit default page size to 20 items
- Implement cursor-based pagination for large datasets
- Use database offsets efficiently

### 4. Concurrent Request Handling

**Problem**: Server struggles under high concurrency
**Solution**:
- Increase worker pool size
- Use connection pooling
- Implement rate limiting
- Add request queuing

### 5. Response Compression

**Problem**: Large payloads increase transfer time
**Solution**:
- Enable gzip compression
- Minimize JSON payload size
- Use field filtering

```go
// Example: Enable compression middleware
app.Use(compress.New(compress.Config{
    Level: compress.LevelBestSpeed,
}))
```

## 🐛 Troubleshooting

### Slow Response Times

**Symptoms**: P95 > 500ms consistently

**Diagnosis**:
```bash
# Check server logs
docker logs proxynd

# Profile API
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./internal/...

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

**Common Causes**:
- Database not indexed
- N+1 query problem
- Missing cache layer
- Inefficient data serialization
- CPU or memory bottleneck

### High P99 Latency

**Symptoms**: P95 OK but P99 > 1000ms

**Diagnosis**:
- Check for garbage collection pauses
- Look for intermittent network issues
- Verify database connection pool size
- Check for lock contention

**Solutions**:
- Increase GOMAXPROCS
- Tune GC settings
- Optimize hot paths
- Add circuit breakers

### Pagination Slowdown

**Symptoms**: Later pages much slower than first page

**Diagnosis**:
```sql
-- Check query plan
EXPLAIN ANALYZE SELECT * FROM audit_events ORDER BY timestamp DESC LIMIT 20 OFFSET 1000;
```

**Solutions**:
- Use cursor-based pagination instead of OFFSET
- Add indexes on sort columns
- Limit maximum page number
- Cache page metadata

## 📝 Benchmark Script Details

### Script Location
`scripts/benchmark-enterprise-api.sh`

### Configuration Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PROXYND_URL` | `http://localhost:8080` | Base URL for API |
| `ITERATIONS` | `100` | Requests per endpoint |
| `CONCURRENT` | `10` | Concurrent requests |
| `P95_TARGET_MS` | `500` | P95 target in milliseconds |
| `P99_TARGET_MS` | `1000` | P99 target in milliseconds |

### Output Format

```
========================================
ProxyND Enterprise API Performance Benchmark
========================================

Configuration:
  Base URL: http://localhost:8080
  Iterations: 100
  Concurrent: 10
  P95 Target: <500ms
  P99 Target: <1000ms

Testing: List Roles
  Method: GET
  Endpoint: /rbac/roles
  Iterations: 100
..........
  Results:
    Success rate: 100/100 (100%)
    Min:  25ms
    Avg:  45ms
    Max:  120ms
    P50:  42ms
    P95:  68ms
    P99:  95ms
    ✓ P95 target met (<500ms)
    ✓ P99 target met (<1000ms)
```

### Exit Codes

- `0` - All benchmarks completed successfully
- `1` - Benchmark execution failed

## 📊 Continuous Performance Monitoring

### CI/CD Integration

Add to your CI pipeline:

```yaml
# .github/workflows/performance.yml
name: Performance Tests

on:
  pull_request:
    branches: [develop, master]
  schedule:
    - cron: '0 0 * * *'  # Daily

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Start ProxyND
        run: make dev-run &

      - name: Wait for server
        run: sleep 10

      - name: Run benchmarks
        run: make test-benchmark-enterprise

      - name: Check performance regression
        run: |
          # Compare with baseline
          # Fail if P95 > 500ms
```

### Monitoring Dashboards

Use Prometheus + Grafana to monitor production performance:

```yaml
# Example Prometheus queries
# P95 response time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Request rate
rate(http_requests_total[1m])

# Error rate
rate(http_requests_total{status=~"5.."}[1m])
```

## 🎓 Best Practices

1. **Run benchmarks on consistent hardware**
   - Use same machine for comparisons
   - Ensure no background processes
   - Use production-like configuration

2. **Warm up before benchmarking**
   - Send initial requests to prime caches
   - Allow JIT compilation to stabilize
   - Wait for connection pools to fill

3. **Test realistic scenarios**
   - Use production-like data volumes
   - Include authentication overhead
   - Test with actual query patterns

4. **Monitor system resources**
   - Watch CPU, memory, disk I/O
   - Check network bandwidth
   - Monitor database connections

5. **Establish baselines**
   - Record initial performance
   - Track changes over time
   - Alert on regressions

## 📚 Additional Resources

- **Swagger Documentation**: `/swagger/index.html`
- **WebUI Integration**: `docs/enterprise/WEBUI_INTEGRATION.md`
- **API Documentation**: `tmp/plan/README.md`
- **Go Benchmarks**: https://golang.org/pkg/testing/#hdr-Benchmarks

## 🔗 Related Make Targets

```bash
# Benchmarks
make test-benchmark-enterprise          # Standard benchmark
make test-benchmark-enterprise-quick    # Quick benchmark (25 iterations)
make test-benchmark-enterprise-stress   # Stress test (500 iterations)

# Testing
make test-enterprise-integration        # Integration tests
make test-contract                      # Contract tests

# Development
make dev-run                           # Start development server
make swagger                          # Generate API docs
```

---

**Last Updated**: 2025-01-17
**Target**: P95 < 500ms, P99 < 1000ms
**Endpoints Tested**: 13/47 (representative sample)
