# ProxyND Grafana Dashboards

Pre-built Grafana dashboards for monitoring ProxyND Enterprise API performance and health.

## Available Dashboards

### Enterprise API Dashboard

**File:** `enterprise-api-dashboard.json`
**UID:** `proxynd-enterprise-api`
**Description:** Comprehensive monitoring dashboard for Enterprise API endpoints

**Panels:**
1. **Request Rate by Category** - Requests per second by API category (RBAC, Audit, Analytics, etc.)
2. **Latency by Category** - p50, p95, p99 latency percentiles
3. **Error Rate Gauge** - Current error rate percentage with thresholds
4. **Cache Hit Rate Gauge** - Current cache efficiency percentage
5. **Active Requests** - Current number of in-flight requests
6. **Critical Vulnerabilities** - Count of critical security vulnerabilities
7. **Request Distribution Pie Chart** - Percentage breakdown by category
8. **Errors by Type** - Error distribution (unauthorized, not_found, etc.)
9. **RBAC Operations Rate** - RBAC operation types over time
10. **Audit Events Rate** - Audit event types over time
11. **Vulnerabilities by Severity** - Security vulnerabilities breakdown
12. **Alerts Triggered Rate** - Alert trigger frequency by severity
13. **Rate Limit Exceeded Events** - Rate limiting violations by endpoint

**Auto-refresh:** 10 seconds
**Default time range:** Last 1 hour

## Installation

### Method 1: Import via Grafana UI

1. Open Grafana UI
2. Navigate to **Dashboards** → **Import**
3. Click **Upload JSON file**
4. Select `enterprise-api-dashboard.json`
5. Choose your Prometheus datasource
6. Click **Import**

### Method 2: Import via API

```bash
# Set Grafana credentials
GRAFANA_URL="http://localhost:3000"
GRAFANA_API_KEY="your-api-key"

# Import dashboard
curl -X POST "$GRAFANA_URL/api/dashboards/db" \
  -H "Authorization: Bearer $GRAFANA_API_KEY" \
  -H "Content-Type: application/json" \
  -d @enterprise-api-dashboard.json
```

### Method 3: Provisioning (Recommended for Production)

1. Copy dashboard to Grafana provisioning directory:

```bash
# For Docker
cp enterprise-api-dashboard.json /var/lib/grafana/dashboards/

# For Kubernetes
kubectl create configmap grafana-dashboards \
  --from-file=enterprise-api-dashboard.json \
  -n monitoring
```

2. Create provisioning configuration (`/etc/grafana/provisioning/dashboards/proxynd.yaml`):

```yaml
apiVersion: 1

providers:
  - name: 'ProxyND Dashboards'
    orgId: 1
    folder: 'ProxyND'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
```

3. Restart Grafana

## Configuration

### Datasource Requirements

The dashboard requires a Prometheus datasource with:
- **Name:** `prometheus` (or update `datasource.uid` in JSON)
- **URL:** Your Prometheus server URL
- **Access:** Server (default) or Browser

**Configure datasource:**

```yaml
apiVersion: 1

datasources:
  - name: prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

### Update Dashboard UID

If you need to change the datasource name, update all occurrences in the JSON:

```bash
# Replace prometheus with your datasource name
sed -i 's/"uid": "prometheus"/"uid": "your-datasource-name"/g' enterprise-api-dashboard.json
```

## Dashboard Panels

### Performance Monitoring

**Request Rate by Category**
- **Query:** `sum(rate(proxynd_enterprise_api_requests_total[5m])) by (category)`
- **Purpose:** Monitor traffic distribution across API categories
- **Alert Threshold:** Sudden drops may indicate issues

**Latency Percentiles**
- **Queries:**
  - p50: `histogram_quantile(0.50, sum by (category, le) (rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])))`
  - p95: `histogram_quantile(0.95, ...)`
  - p99: `histogram_quantile(0.99, ...)`
- **Purpose:** Track response time distribution
- **SLA Targets:**
  - p50 < 100ms
  - p95 < 200ms
  - p99 < 500ms

### Health Indicators

**Error Rate Gauge**
- **Query:** `(1 - (sum(rate(proxynd_enterprise_api_errors_total[5m])) / sum(rate(proxynd_enterprise_api_requests_total[5m])))) * 100`
- **Thresholds:**
  - Green: > 95% success rate
  - Yellow: 90-95%
  - Red: < 90%

**Cache Hit Rate Gauge**
- **Query:** `sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) / (sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) + sum(rate(proxynd_enterprise_api_cache_misses_total[5m]))) * 100`
- **Thresholds:**
  - Green: > 85%
  - Yellow: 70-85%
  - Red: < 70%

### Feature Monitoring

**RBAC Operations**
- **Query:** `sum by (operation) (rate(proxynd_enterprise_api_rbac_operations_total[5m]))`
- **Purpose:** Monitor role management activity

**Audit Events**
- **Query:** `sum by (event_type) (rate(proxynd_enterprise_api_audit_events_total[5m]))`
- **Purpose:** Track audit logging activity

**Security Vulnerabilities**
- **Query:** `sum by (severity) (proxynd_enterprise_api_security_vulnerabilities_found)`
- **Purpose:** Monitor security posture
- **Alert:** Critical vulnerabilities > 0

**Alerts Triggered**
- **Query:** `sum by (severity) (rate(proxynd_enterprise_api_alerts_triggered_total[5m]))`
- **Purpose:** Track alerting activity

## Alerting

### Recommended Alerts

Create these alerts in Grafana:

**High Error Rate**
```yaml
alert: EnterpriseAPIHighErrorRate
expr: (sum(rate(proxynd_enterprise_api_errors_total[5m])) / sum(rate(proxynd_enterprise_api_requests_total[5m]))) * 100 > 5
for: 5m
annotations:
  summary: "Enterprise API error rate is {{ $value }}%"
```

**High Latency**
```yaml
alert: EnterpriseAPIHighLatency
expr: histogram_quantile(0.95, sum by (le) (rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m]))) > 1
for: 5m
annotations:
  summary: "P95 latency is {{ $value }}s"
```

**Low Cache Hit Rate**
```yaml
alert: EnterpriseAPILowCacheHitRate
expr: (sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) / (sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) + sum(rate(proxynd_enterprise_api_cache_misses_total[5m])))) * 100 < 70
for: 10m
annotations:
  summary: "Cache hit rate is {{ $value }}%"
```

## Customization

### Add Custom Panels

1. **Edit dashboard** in Grafana UI
2. Click **Add** → **Visualization**
3. Configure your panel
4. **Save** dashboard
5. **Export** JSON
6. Replace `enterprise-api-dashboard.json`

### Modify Time Range

Edit the JSON `time` section:

```json
{
  "time": {
    "from": "now-6h",  // Change from 1h to 6h
    "to": "now"
  }
}
```

### Change Refresh Rate

Edit the `refresh` field:

```json
{
  "refresh": "30s"  // Change from 10s to 30s
}
```

## Integration with Load Testing

Use the dashboard to monitor load tests in real-time:

```bash
# Terminal 1: Run load test
./scripts/loadtest/load-test-enterprise.sh standard

# Terminal 2: Watch Grafana dashboard
# Open http://localhost:3000/d/proxynd-enterprise-api/

# Observe:
# - Request rate spikes
# - Latency percentiles during load
# - Error rates
# - Cache efficiency
# - Rate limiting events
```

## Troubleshooting

### Dashboard Shows "No Data"

**Check Prometheus connection:**
```bash
# Test Prometheus query
curl http://localhost:9090/api/v1/query?query=proxynd_enterprise_api_requests_total
```

**Verify metrics are being collected:**
```bash
# Check ProxyND metrics endpoint
curl http://localhost:8080/metrics | grep enterprise_api
```

**Verify datasource configuration:**
1. Go to **Configuration** → **Data Sources** → **prometheus**
2. Click **Test** button
3. Should show "Data source is working"

### Panels Show Errors

**Common issues:**
- Incorrect datasource UID
- Prometheus query syntax error
- Missing metric labels

**Fix:**
1. Edit panel
2. Check **Query inspector** tab
3. Verify Prometheus query returns data
4. Update panel configuration

### Slow Dashboard Loading

**Optimize:**
1. Reduce time range (e.g., 1h instead of 24h)
2. Increase query interval (e.g., `[5m]` instead of `[1m]`)
3. Use recording rules in Prometheus for complex queries

## Recording Rules (Optional)

Improve dashboard performance with Prometheus recording rules:

```yaml
groups:
  - name: enterprise_api_rules
    interval: 10s
    rules:
      # Pre-calculate request rate
      - record: enterprise_api:requests_per_second:rate5m
        expr: sum(rate(proxynd_enterprise_api_requests_total[5m])) by (category)

      # Pre-calculate p95 latency
      - record: enterprise_api:latency_p95:histogram_quantile5m
        expr: histogram_quantile(0.95, sum by (category, le) (rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])))

      # Pre-calculate cache hit rate
      - record: enterprise_api:cache_hit_rate:percentage5m
        expr: sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) / (sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) + sum(rate(proxynd_enterprise_api_cache_misses_total[5m]))) * 100
```

Then update dashboard queries to use recording rules:
```promql
# Instead of: sum(rate(proxynd_enterprise_api_requests_total[5m])) by (category)
# Use: enterprise_api:requests_per_second:rate5m
```

## Screenshots

### Full Dashboard View
![Dashboard Overview](https://via.placeholder.com/800x600.png?text=Enterprise+API+Dashboard)

### Performance Panels
![Performance Metrics](https://via.placeholder.com/800x400.png?text=Latency+and+Throughput)

### Security Monitoring
![Security Panels](https://via.placeholder.com/800x400.png?text=Vulnerabilities+and+Alerts)

## Further Reading

- [Grafana Documentation](https://grafana.com/docs/grafana/latest/)
- [Prometheus Query Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
- [Enterprise API Metrics Guide](../../docs/api/METRICS.md)
- [Load Testing Guide](../../scripts/loadtest/README.md)

## Support

For questions or issues:
- GitHub Issues: https://github.com/ScriptonBasestar/proxynd/issues
- Grafana Community: https://community.grafana.com/
- Email: support@proxynd.io
