# ProxyND Enterprise API Load Testing

Comprehensive load testing tools and scenarios for ProxyND Enterprise API endpoints.

## Quick Start

### Using the Load Test Script (Recommended)

```bash
# Quick validation (10s, 10 concurrent)
./scripts/loadtest/load-test-enterprise.sh quick

# Standard load test (60s, 100 concurrent)
./scripts/loadtest/load-test-enterprise.sh standard

# Stress test (60s, 500 concurrent)
./scripts/loadtest/load-test-enterprise.sh stress
```

### Using Go-based Load Tester

```bash
# Single endpoint test
go run scripts/loadtest/loadtest.go \
  -url http://localhost:8080/api/v1/enterprise/analytics/overview \
  -duration 30s \
  -concurrency 50

# With authentication
go run scripts/loadtest/loadtest.go \
  -url http://localhost:8080/api/v1/enterprise/rbac/roles \
  -duration 60s \
  -concurrency 100 \
  -token "your-enterprise-token"
```

### Using wrk (If Installed)

```bash
# Basic test
wrk -t4 -c100 -d30s http://localhost:8080/api/v1/enterprise/analytics/overview

# With authentication
wrk -t4 -c100 -d30s \
  -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/enterprise/rbac/roles

# Using Lua scenarios
ENTERPRISE_TOKEN="your-token" wrk -t4 -c100 -d60s \
  -s scripts/loadtest/wrk-scenarios/enterprise-mixed.lua \
  http://localhost:8080
```

## Installation

### Install wrk (Optional, for Advanced Scenarios)

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install wrk
```

**macOS:**
```bash
brew install wrk
```

**From Source:**
```bash
git clone https://github.com/wg/wrk.git
cd wrk
make
sudo cp wrk /usr/local/bin/
```

### No Installation Required

The Go-based load tester requires no installation - it uses Go's standard library.

## Test Scenarios

### 1. Quick Validation (`quick`)

**Purpose:** Fast sanity check
**Duration:** 10 seconds
**Concurrency:** 10 users
**Use Case:** CI/CD pipeline, quick verification

```bash
./scripts/loadtest/load-test-enterprise.sh quick
```

### 2. Standard Load Test (`standard`)

**Purpose:** Realistic production load
**Duration:** 60 seconds
**Concurrency:** 100 users
**Use Case:** Regular performance validation

```bash
./scripts/loadtest/load-test-enterprise.sh standard
```

### 3. Stress Test (`stress`)

**Purpose:** Find breaking points
**Duration:** 60 seconds
**Concurrency:** 500 users
**Use Case:** Capacity planning

```bash
./scripts/loadtest/load-test-enterprise.sh stress
```

### 4. Ramp-up Test (`ramp-up`)

**Purpose:** Gradual load increase
**Duration:** 120 seconds
**Concurrency:** 10 → 1000 users (gradual)
**Use Case:** Test auto-scaling behavior

```bash
./scripts/loadtest/load-test-enterprise.sh ramp-up
```

### 5. Sustained Load Test (`sustained`)

**Purpose:** Long-running stability
**Duration:** 300 seconds (5 minutes)
**Concurrency:** 100 users
**Use Case:** Memory leak detection, stability verification

```bash
./scripts/loadtest/load-test-enterprise.sh sustained
```

## wrk Lua Scenarios

### Mixed Enterprise API (`enterprise-mixed.lua`)

Simulates realistic traffic distribution across all Enterprise endpoints:
- Analytics: 30%
- Audit: 25%
- RBAC: 20%
- Security: 15%
- Alerts: 5%
- License: 5%

```bash
ENTERPRISE_TOKEN="your-token" wrk -t4 -c100 -d60s \
  -s scripts/loadtest/wrk-scenarios/enterprise-mixed.lua \
  http://localhost:8080
```

### RBAC Operations (`rbac-operations.lua`)

Tests RBAC management operations:
- List roles (40%)
- Get role details (20%)
- List permissions (20%)
- Get user roles (15%)
- Create role (5%)

```bash
ENTERPRISE_TOKEN="your-token" wrk -t4 -c50 -d30s \
  -s scripts/loadtest/wrk-scenarios/rbac-operations.lua \
  http://localhost:8080
```

### Audit Search (`audit-search.lua`)

Tests audit logging and search performance with various query patterns.

```bash
ENTERPRISE_TOKEN="your-token" wrk -t4 -c100 -d60s \
  -s scripts/loadtest/wrk-scenarios/audit-search.lua \
  http://localhost:8080
```

### Analytics Dashboard (`analytics-dashboard.lua`)

Simulates dashboard loading by cycling through all analytics endpoints.

```bash
ENTERPRISE_TOKEN="your-token" wrk -t4 -c50 -d30s \
  -s scripts/loadtest/wrk-scenarios/analytics-dashboard.lua \
  http://localhost:8080
```

## Tested Endpoints

### RBAC (12 endpoints)
- `GET /api/v1/enterprise/rbac/roles`
- `POST /api/v1/enterprise/rbac/roles`
- `GET /api/v1/enterprise/rbac/roles/{id}`
- `PUT /api/v1/enterprise/rbac/roles/{id}`
- `DELETE /api/v1/enterprise/rbac/roles/{id}`
- And 7 more...

### Audit (8 endpoints)
- `GET /api/v1/enterprise/audit/events`
- `GET /api/v1/enterprise/audit/events/search`
- `GET /api/v1/enterprise/audit/events/{id}`
- `POST /api/v1/enterprise/audit/export`
- And 4 more...

### Analytics (10 endpoints)
- `GET /api/v1/enterprise/analytics/overview`
- `GET /api/v1/enterprise/analytics/usage`
- `GET /api/v1/enterprise/analytics/performance`
- `GET /api/v1/enterprise/analytics/cache-efficiency`
- And 6 more...

### Security (8 endpoints)
- `GET /api/v1/enterprise/security/vulnerabilities`
- `POST /api/v1/enterprise/security/scan`
- `GET /api/v1/enterprise/security/scan/{jobId}`
- And 5 more...

### Alerts (5 endpoints)
- `GET /api/v1/enterprise/alerts`
- `GET /api/v1/enterprise/alerts/{id}`
- `POST /api/v1/enterprise/alerts/rules`
- And 2 more...

### License (4 endpoints)
- `GET /api/v1/enterprise/license/info`
- `POST /api/v1/enterprise/license/validate`
- `GET /api/v1/enterprise/license/features`
- `GET /api/v1/enterprise/license/usage`

## Performance Targets

### Latency Goals

| Percentile | Target   | Acceptable | Poor   |
|------------|----------|------------|--------|
| **p50**    | < 50ms   | < 100ms    | > 200ms|
| **p95**    | < 200ms  | < 500ms    | > 1s   |
| **p99**    | < 500ms  | < 1s       | > 2s   |

### Throughput Goals

| Scenario      | Target        | Acceptable    |
|---------------|---------------|---------------|
| **Single EP** | > 1000 rps    | > 500 rps     |
| **Mixed**     | > 800 rps     | > 400 rps     |
| **Dashboard** | > 500 rps     | > 250 rps     |

### Error Rate

- **Target:** < 0.1%
- **Acceptable:** < 1%
- **Critical:** > 5% (requires investigation)

## Monitoring During Tests

### Real-time Metrics

```bash
# Watch Prometheus metrics
watch -n 1 curl -s http://localhost:8080/metrics | grep enterprise_api

# Monitor cache hit rates
watch -n 1 curl -s http://localhost:8080/metrics | grep cache_hit

# Check rate limiting
watch -n 1 curl -s http://localhost:8080/metrics | grep rate_limit
```

### Server Logs

```bash
# Follow application logs
tail -f logs/proxynd.log | grep enterprise

# Watch for errors
tail -f logs/proxynd.log | grep -i error
```

### System Resources

```bash
# CPU and Memory
top -p $(pgrep proxynd)

# Network connections
netstat -an | grep :8080 | wc -l

# Open files
lsof -p $(pgrep proxynd) | wc -l
```

## Custom Load Test Examples

### Test Specific Endpoint

```bash
# Analytics overview
go run scripts/loadtest/loadtest.go \
  -url http://localhost:8080/api/v1/enterprise/analytics/overview \
  -duration 30s \
  -concurrency 100

# RBAC roles with pagination
go run scripts/loadtest/loadtest.go \
  -url "http://localhost:8080/api/v1/enterprise/rbac/roles?page=1&per_page=50" \
  -duration 60s \
  -concurrency 50
```

### Environment Variables

```bash
# Set defaults via environment
export BASE_URL=http://localhost:8080
export DURATION=120s
export CONCURRENCY=200
export ENTERPRISE_TOKEN=your-token-here

# Run with env vars
./scripts/loadtest/load-test-enterprise.sh standard
```

### Custom Concurrency

```bash
# Low concurrency (development)
./scripts/loadtest/load-test-enterprise.sh -c 10 -d 30s

# High concurrency (production simulation)
./scripts/loadtest/load-test-enterprise.sh -c 1000 -d 60s
```

## Interpreting Results

### Go Load Tester Output

```
Load Test Results
=================

Requests:
  Total:    12450
  Success:  12450 (100.00%)
  Failed:   0 (0.00%)

Throughput:
  Requests/sec: 415.00

Latency:
  Min:  12ms
  Avg:  24ms
  P50:  22ms
  P95:  45ms
  P99:  78ms
  Max:  156ms

Status Codes:
  200: 12450

Performance Verdict:
  ✓ EXCELLENT: Low latency (<24ms) and high throughput (415 rps)
  ✓ No errors detected
```

### What to Look For

**Good Performance:**
- ✓ P95 latency < 200ms
- ✓ Throughput > 500 rps
- ✓ Error rate < 0.1%
- ✓ Consistent latency distribution

**Warning Signs:**
- ⚠ P95 latency > 500ms
- ⚠ Error rate > 1%
- ⚠ Increasing latency over time (memory leak?)
- ⚠ High variance in latency

**Critical Issues:**
- ✗ P95 latency > 1s
- ✗ Error rate > 5%
- ✗ Throughput degradation
- ✗ Server crashes or timeouts

## Troubleshooting

### High Latency

1. **Check cache hit rates:** Low cache hits = slower responses
2. **Database queries:** Use query profiling
3. **Network:** Check for network congestion
4. **CPU usage:** May need more resources

### High Error Rate

1. **Check logs:** `tail -f logs/proxynd.log | grep -i error`
2. **Rate limiting:** Verify rate limit settings
3. **Resources:** Check CPU, memory, file descriptors
4. **Upstream issues:** Verify upstream services are healthy

### Inconsistent Results

1. **Warm up:** Run a small test first to warm caches
2. **Background processes:** Ensure no other heavy processes running
3. **Network:** Use localhost for consistent results
4. **GC pauses:** Monitor Go garbage collection

## Integration with CI/CD

### GitHub Actions Example

```yaml
- name: Load Test
  run: |
    make dev-run &
    sleep 5
    ./scripts/loadtest/load-test-enterprise.sh quick
```

### Performance Regression Detection

```bash
# Save baseline
./scripts/loadtest/load-test-enterprise.sh standard > baseline.txt

# Compare with current
./scripts/loadtest/load-test-enterprise.sh standard > current.txt
diff baseline.txt current.txt
```

## Further Reading

- [Enterprise API Documentation](../../docs/api/README.md)
- [Performance Benchmarks](../../tests/benchmark/README.md)
- [Monitoring Guide](../../docs/04-api-reference/monitoring.md)
- [wrk Documentation](https://github.com/wg/wrk)

## Support

For questions or issues:
- GitHub Issues: https://github.com/ScriptonBasestar/proxynd/issues
- Email: support@proxynd.io
