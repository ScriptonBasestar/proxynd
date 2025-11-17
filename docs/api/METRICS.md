# ProxyND Enterprise API Metrics

Comprehensive Prometheus metrics for Enterprise API monitoring and observability.

## Metrics Endpoint

All metrics are exposed at the standard Prometheus endpoint:

```
GET http://localhost:8080/metrics
```

## Available Metrics

### Request Metrics

#### `proxynd_enterprise_api_requests_total`

Total number of Enterprise API requests.

**Type:** Counter
**Labels:**
- `category`: API category (rbac, audit, analytics, security, alerts, license)
- `endpoint`: Normalized endpoint path (e.g., `/api/v1/enterprise/rbac/roles/{id}`)
- `method`: HTTP method (GET, POST, PUT, DELETE)
- `status_code`: HTTP status code (200, 404, 500, etc.)

**Example:**
```promql
# Total requests to RBAC endpoints
proxynd_enterprise_api_requests_total{category="rbac"}

# Failed requests (4xx and 5xx)
proxynd_enterprise_api_requests_total{status_code=~"4..|5.."}

# Request rate per category
rate(proxynd_enterprise_api_requests_total[5m])
```

#### `proxynd_enterprise_api_request_duration_seconds`

Enterprise API request duration in seconds.

**Type:** Histogram
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path
- `method`: HTTP method

**Buckets:** 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s

**Example:**
```promql
# P95 latency for analytics endpoints
histogram_quantile(0.95,
  rate(proxynd_enterprise_api_request_duration_seconds_bucket{category="analytics"}[5m]))

# Average latency by endpoint
rate(proxynd_enterprise_api_request_duration_seconds_sum[5m]) /
rate(proxynd_enterprise_api_request_duration_seconds_count[5m])
```

#### `proxynd_enterprise_api_active_requests`

Number of active Enterprise API requests.

**Type:** Gauge
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path

**Example:**
```promql
# Active requests by category
proxynd_enterprise_api_active_requests

# Total active requests
sum(proxynd_enterprise_api_active_requests)
```

#### `proxynd_enterprise_api_request_size_bytes`

Enterprise API request size in bytes.

**Type:** Histogram
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path

**Buckets:** Exponential from 100 bytes (100, 1K, 10K, 100K, 1M, 10M, 100M, 1G)

#### `proxynd_enterprise_api_response_size_bytes`

Enterprise API response size in bytes.

**Type:** Histogram
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path

**Buckets:** Exponential from 100 bytes

### Error Metrics

#### `proxynd_enterprise_api_errors_total`

Total number of Enterprise API errors.

**Type:** Counter
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path
- `error_type`: Type of error (bad_request, unauthorized, not_found, etc.)
- `status_code`: HTTP status code

**Example:**
```promql
# Error rate by type
rate(proxynd_enterprise_api_errors_total[5m])

# Unauthorized errors
proxynd_enterprise_api_errors_total{error_type="unauthorized"}
```

**Error Types:**
- `bad_request` (400)
- `unauthorized` (401)
- `payment_required` (402)
- `forbidden` (403)
- `not_found` (404)
- `method_not_allowed` (405)
- `conflict` (409)
- `validation_error` (422)
- `rate_limit_exceeded` (429)
- `internal_server_error` (500)
- `bad_gateway` (502)
- `service_unavailable` (503)
- `gateway_timeout` (504)

### Rate Limiting Metrics

#### `proxynd_enterprise_api_rate_limit_exceeded_total`

Total number of rate limit exceeded events.

**Type:** Counter
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path
- `user_id`: User ID (or "anonymous")

**Example:**
```promql
# Rate limit violations by user
rate(proxynd_enterprise_api_rate_limit_exceeded_total[5m])

# Top users hitting rate limits
topk(10, sum by (user_id) (proxynd_enterprise_api_rate_limit_exceeded_total))
```

#### `proxynd_enterprise_api_rate_limit_remaining`

Remaining rate limit quota.

**Type:** Gauge
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path
- `user_id`: User ID

**Example:**
```promql
# Users close to rate limit
proxynd_enterprise_api_rate_limit_remaining < 10
```

### License Metrics

#### `proxynd_enterprise_api_license_check_total`

Total number of license validation checks.

**Type:** Counter
**Labels:**
- `result`: Check result (valid, invalid, expired)
- `license_type`: License type (community, professional, enterprise)

**Example:**
```promql
# Failed license checks
proxynd_enterprise_api_license_check_total{result="invalid"}

# License check rate
rate(proxynd_enterprise_api_license_check_total[5m])
```

#### `proxynd_enterprise_api_license_check_duration_seconds`

License validation check duration in seconds.

**Type:** Histogram
**Labels:**
- `result`: Check result

**Buckets:** 0.1ms, 0.5ms, 1ms, 5ms, 10ms, 50ms, 100ms

### Feature-Specific Metrics

#### `proxynd_enterprise_api_rbac_operations_total`

Total number of RBAC operations.

**Type:** Counter
**Labels:**
- `operation`: Operation type (create, read, update, delete, assign, revoke)
- `resource_type`: Resource type (role, permission, user_role)
- `result`: Operation result (success, failure)

**Example:**
```promql
# RBAC operation success rate
sum(rate(proxynd_enterprise_api_rbac_operations_total{result="success"}[5m])) /
sum(rate(proxynd_enterprise_api_rbac_operations_total[5m]))

# Failed role assignments
proxynd_enterprise_api_rbac_operations_total{operation="assign", result="failure"}
```

#### `proxynd_enterprise_api_audit_events_total`

Total number of audit events logged.

**Type:** Counter
**Labels:**
- `event_type`: Event type (create, update, delete, access, login, logout)
- `user_id`: User ID
- `result`: Event result (success, failure)

**Example:**
```promql
# Audit event rate
rate(proxynd_enterprise_api_audit_events_total[5m])

# Failed operations by user
proxynd_enterprise_api_audit_events_total{result="failure"}
```

#### `proxynd_enterprise_api_security_scans_total`

Total number of security scans performed.

**Type:** Counter
**Labels:**
- `scan_type`: Scan type (vulnerability, license, malware)
- `status`: Scan status (completed, failed, in_progress)

**Example:**
```promql
# Security scan completion rate
rate(proxynd_enterprise_api_security_scans_total{status="completed"}[1h])

# Failed scans
proxynd_enterprise_api_security_scans_total{status="failed"}
```

#### `proxynd_enterprise_api_security_vulnerabilities_found`

Number of vulnerabilities found by severity.

**Type:** Gauge
**Labels:**
- `severity`: Vulnerability severity (critical, high, medium, low)
- `package_manager`: Package manager (npm, maven, pypi, etc.)

**Example:**
```promql
# Critical vulnerabilities
proxynd_enterprise_api_security_vulnerabilities_found{severity="critical"}

# Total vulnerabilities by package manager
sum by (package_manager) (proxynd_enterprise_api_security_vulnerabilities_found)
```

#### `proxynd_enterprise_api_alerts_triggered_total`

Total number of alerts triggered.

**Type:** Counter
**Labels:**
- `alert_type`: Alert type (threshold, anomaly, security)
- `severity`: Alert severity (critical, warning, info)

**Example:**
```promql
# Alert rate by severity
rate(proxynd_enterprise_api_alerts_triggered_total[5m])

# Critical alerts
proxynd_enterprise_api_alerts_triggered_total{severity="critical"}
```

### Cache Metrics

#### `proxynd_enterprise_api_cache_hits_total`

Total number of Enterprise API cache hits.

**Type:** Counter
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path

#### `proxynd_enterprise_api_cache_misses_total`

Total number of Enterprise API cache misses.

**Type:** Counter
**Labels:**
- `category`: API category
- `endpoint`: Normalized endpoint path

**Example:**
```promql
# Cache hit rate
sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) /
(sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) +
 sum(rate(proxynd_enterprise_api_cache_misses_total[5m])))

# Cache hit rate by category
sum by (category) (rate(proxynd_enterprise_api_cache_hits_total[5m])) /
(sum by (category) (rate(proxynd_enterprise_api_cache_hits_total[5m])) +
 sum by (category) (rate(proxynd_enterprise_api_cache_misses_total[5m])))
```

## Example Queries

### Performance Monitoring

```promql
# P95 latency by category
histogram_quantile(0.95,
  sum by (category, le) (
    rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])
  ))

# Throughput (requests per second)
sum(rate(proxynd_enterprise_api_requests_total[5m]))

# Throughput by endpoint
sum by (endpoint) (rate(proxynd_enterprise_api_requests_total[5m]))
```

### Error Monitoring

```promql
# Error rate (percentage)
sum(rate(proxynd_enterprise_api_errors_total[5m])) /
sum(rate(proxynd_enterprise_api_requests_total[5m])) * 100

# 5xx error rate
sum(rate(proxynd_enterprise_api_requests_total{status_code=~"5.."}[5m]))

# Top error endpoints
topk(5, sum by (endpoint) (rate(proxynd_enterprise_api_errors_total[5m])))
```

### Capacity Planning

```promql
# Peak concurrent requests
max_over_time(sum(proxynd_enterprise_api_active_requests)[1h:])

# Request rate trend (last 24h)
sum(rate(proxynd_enterprise_api_requests_total[5m]))

# Average response size by endpoint
sum by (endpoint) (rate(proxynd_enterprise_api_response_size_bytes_sum[5m])) /
sum by (endpoint) (rate(proxynd_enterprise_api_response_size_bytes_count[5m]))
```

### SLA Monitoring

```promql
# Availability (non-5xx responses)
(sum(rate(proxynd_enterprise_api_requests_total[5m])) -
 sum(rate(proxynd_enterprise_api_requests_total{status_code=~"5.."}[5m]))) /
sum(rate(proxynd_enterprise_api_requests_total[5m])) * 100

# Latency SLA compliance (requests under 200ms)
sum(rate(proxynd_enterprise_api_request_duration_seconds_bucket{le="0.2"}[5m])) /
sum(rate(proxynd_enterprise_api_request_duration_seconds_count[5m])) * 100
```

## Alerting Rules

### Recommended Prometheus Alerts

```yaml
groups:
  - name: enterprise_api_alerts
    rules:
      # High error rate
      - alert: EnterpriseAPIHighErrorRate
        expr: |
          (sum(rate(proxynd_enterprise_api_errors_total[5m])) /
           sum(rate(proxynd_enterprise_api_requests_total[5m])) * 100) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Enterprise API error rate is high"
          description: "Error rate is {{ $value }}% (threshold: 5%)"

      # High latency
      - alert: EnterpriseAPIHighLatency
        expr: |
          histogram_quantile(0.95,
            sum by (le) (
              rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])
            )) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Enterprise API P95 latency is high"
          description: "P95 latency is {{ $value }}s (threshold: 1s)"

      # Low cache hit rate
      - alert: EnterpriseAPILowCacheHitRate
        expr: |
          (sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) /
           (sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) +
            sum(rate(proxynd_enterprise_api_cache_misses_total[5m]))) * 100) < 70
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "Enterprise API cache hit rate is low"
          description: "Cache hit rate is {{ $value }}% (threshold: 70%)"

      # Rate limit violations
      - alert: EnterpriseAPIRateLimitExceeded
        expr: |
          rate(proxynd_enterprise_api_rate_limit_exceeded_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High rate of rate limit violations"
          description: "Rate limit exceeded {{ $value }} times/sec"

      # License check failures
      - alert: EnterpriseAPILicenseCheckFailure
        expr: |
          rate(proxynd_enterprise_api_license_check_total{result!="valid"}[5m]) > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Enterprise license validation failures detected"
          description: "Invalid license checks: {{ $value }}/sec"

      # Critical vulnerabilities
      - alert: EnterpriseAPICriticalVulnerabilities
        expr: |
          proxynd_enterprise_api_security_vulnerabilities_found{severity="critical"} > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Critical security vulnerabilities found"
          description: "{{ $value }} critical vulnerabilities detected"
```

## Grafana Dashboard

### Sample Dashboard Queries

**Request Rate Panel:**
```promql
sum(rate(proxynd_enterprise_api_requests_total[5m])) by (category)
```

**Latency Panel:**
```promql
histogram_quantile(0.50, sum by (category, le) (rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])))
histogram_quantile(0.95, sum by (category, le) (rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])))
histogram_quantile(0.99, sum by (category, le) (rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])))
```

**Error Rate Panel:**
```promql
sum(rate(proxynd_enterprise_api_errors_total[5m])) by (error_type)
```

**Cache Hit Rate Panel:**
```promql
sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) /
(sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) +
 sum(rate(proxynd_enterprise_api_cache_misses_total[5m]))) * 100
```

## Integration with Load Testing

Use metrics to validate load test results:

```bash
# Run load test
./scripts/loadtest/load-test-enterprise.sh standard

# Query metrics during test
curl -s http://localhost:8080/metrics | grep enterprise_api

# Check P95 latency
curl -s http://localhost:8080/metrics | grep -A5 enterprise_api_request_duration_seconds

# Check error rate
curl -s http://localhost:8080/metrics | grep enterprise_api_errors_total
```

## Best Practices

1. **High Cardinality Labels**: Avoid user-specific labels on high-traffic metrics
2. **Metric Aggregation**: Use recording rules for frequently queried aggregations
3. **Retention**: Configure appropriate retention based on storage capacity
4. **Alerting**: Set up alerts for key SLIs (latency, error rate, availability)
5. **Dashboards**: Create category-specific dashboards for easier monitoring

## Further Reading

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)
- [Load Testing Guide](../../scripts/loadtest/README.md)
- [Enterprise API Documentation](README.md)
