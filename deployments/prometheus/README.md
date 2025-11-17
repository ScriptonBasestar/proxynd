# ProxyND Prometheus Configuration

Prometheus monitoring configuration for ProxyND Enterprise API.

## 📋 Overview

This directory contains Prometheus configuration files for monitoring ProxyND Enterprise API:

- **alert-rules.yml** - Alert rules for performance, security, and health monitoring
- **prometheus.yml** - Main Prometheus configuration (example)
- **recording-rules.yml** - Recording rules for dashboard optimization (optional)

## 🚀 Quick Start

### 1. Install Prometheus

**Docker**:
```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/deployments/prometheus:/etc/prometheus \
  -v prometheus-data:/prometheus \
  prom/prometheus:latest \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --web.enable-lifecycle
```

**Kubernetes (via Helm)**:
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install prometheus prometheus-community/prometheus \
  --namespace monitoring \
  --create-namespace \
  -f prometheus-values.yaml
```

### 2. Configure ProxyND as Target

Create `prometheus.yml` (or update existing):

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'production'
    service: 'proxynd'

# Load alert rules
rule_files:
  - 'alert-rules.yml'
  # Optional: - 'recording-rules.yml'

# Scrape ProxyND Enterprise API metrics
scrape_configs:
  - job_name: 'proxynd-enterprise-api'
    static_configs:
      - targets:
          - 'localhost:8080'  # Update with your ProxyND host:port
    metrics_path: '/metrics'
    scrape_interval: 10s
    scrape_timeout: 5s

# Alertmanager configuration
alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - 'localhost:9093'  # Update with your Alertmanager host:port
```

### 3. Reload Configuration

```bash
# Docker
docker kill -s HUP prometheus

# Kubernetes
kubectl rollout restart deployment/prometheus -n monitoring

# Systemd
systemctl reload prometheus

# API (if --web.enable-lifecycle is enabled)
curl -X POST http://localhost:9090/-/reload
```

### 4. Verify Setup

```bash
# Check targets are being scraped
curl http://localhost:9090/api/v1/targets

# Check rules are loaded
curl http://localhost:9090/api/v1/rules

# Query ProxyND metrics
curl 'http://localhost:9090/api/v1/query?query=proxynd_enterprise_api_requests_total'
```

## 🔔 Alert Rules

### Overview

47 alert rules across 8 categories:

| Category | Alerts | Severity Levels | Description |
|----------|--------|-----------------|-------------|
| **Performance** | 3 | warning, critical | Latency and throughput monitoring |
| **Errors** | 4 | warning, critical | Error rate tracking |
| **Cache** | 2 | warning | Cache performance |
| **Security** | 4 | warning, critical | Vulnerabilities and rate limiting |
| **License** | 3 | warning, critical | License status and usage |
| **RBAC** | 1 | warning | Permission changes |
| **Audit** | 1 | warning | Security events |
| **Health** | 3 | warning, critical | Service availability |

### Alert Severities

- **🔴 Critical**: Immediate action required (service down, critical errors)
- **🟡 Warning**: Action needed soon (degraded performance, elevated errors)
- **🟢 Info**: Informational only (no action required)

### Key Alerts

#### Performance

**EnterpriseAPIHighP95Latency**
- Triggers: P95 latency > 1s for 5 minutes
- Impact: User experience degradation
- Action: Check database, cache, upstream services

**EnterpriseAPICriticalP99Latency**
- Triggers: P99 latency > 2s for 3 minutes
- Impact: Severe performance issues
- Action: Immediate investigation required

**EnterpriseAPILowThroughput**
- Triggers: RPS < 100 for 10 minutes
- Impact: Potential service degradation
- Action: Check client connectivity and service health

#### Errors

**EnterpriseAPIHighErrorRate**
- Triggers: Overall error rate > 5% for 5 minutes
- Impact: Service degradation
- Action: Check logs, recent deployments

**EnterpriseAPIServerErrors**
- Triggers: 5xx errors > 10/s for 2 minutes
- Impact: Backend service issues
- Action: Check database, cache, upstream services

#### Security

**EnterpriseAPICriticalVulnerabilities**
- Triggers: Any critical vulnerabilities found
- Impact: Security risk
- Action: Review and patch immediately

**EnterpriseAPIRateLimitViolations**
- Triggers: Rate limit violations > 50/s for 3 minutes
- Impact: Potential abuse or misconfiguration
- Action: Review client behavior, adjust limits

#### License

**EnterpriseAPILicenseExpiringSoon**
- Triggers: License expires in < 30 days
- Impact: Service interruption risk
- Action: Renew license

**EnterpriseAPILicenseExpired**
- Triggers: License has expired
- Impact: Service disabled or degraded
- Action: Renew immediately

## 📧 Alertmanager Integration

### Install Alertmanager

**Docker**:
```bash
docker run -d \
  --name alertmanager \
  -p 9093:9093 \
  -v $(pwd)/alertmanager.yml:/etc/alertmanager/alertmanager.yml \
  prom/alertmanager:latest
```

### Configure Notifications

Create `alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@proxynd.io'
  smtp_auth_username: 'your-email@gmail.com'
  smtp_auth_password: 'your-app-password'

# Notification routing
route:
  receiver: 'default'
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

  routes:
    # Critical alerts to PagerDuty
    - match:
        severity: critical
      receiver: 'pagerduty'
      continue: true

    # Security alerts to security team
    - match:
        category: security
      receiver: 'security-team'
      continue: true

    # Performance alerts to dev team
    - match:
        category: performance
      receiver: 'dev-team'

# Receivers
receivers:
  # Default email
  - name: 'default'
    email_configs:
      - to: 'ops@proxynd.io'
        headers:
          Subject: '[ProxyND] {{ .GroupLabels.alertname }}'

  # PagerDuty for critical alerts
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}: {{ .CommonAnnotations.summary }}'

  # Slack for dev team
  - name: 'dev-team'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#proxynd-alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ .CommonAnnotations.description }}'

  # Email for security team
  - name: 'security-team'
    email_configs:
      - to: 'security@proxynd.io'
        headers:
          Subject: '[SECURITY] {{ .GroupLabels.alertname }}'

# Inhibition rules (suppress alerts)
inhibit_rules:
  # Don't alert on high error rate if service is down
  - source_match:
      alertname: 'EnterpriseAPIDown'
    target_match:
      alertname: 'EnterpriseAPIHighErrorRate'
    equal: ['job']
```

### Test Alerting

```bash
# Send test alert
curl -X POST http://localhost:9093/api/v1/alerts \
  -H 'Content-Type: application/json' \
  -d '[{
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning"
    },
    "annotations": {
      "summary": "Test alert"
    }
  }]'

# Check Alertmanager status
curl http://localhost:9093/api/v1/status

# View active alerts
curl http://localhost:9093/api/v1/alerts
```

## 📊 Recording Rules (Optional)

Recording rules pre-calculate expensive queries for faster dashboard performance.

Create `recording-rules.yml`:

```yaml
groups:
  - name: enterprise_api_aggregations
    interval: 10s
    rules:
      # Request rate by category
      - record: enterprise_api:requests_per_second:rate5m
        expr: sum(rate(proxynd_enterprise_api_requests_total[5m])) by (category)

      # Error rate by category
      - record: enterprise_api:error_rate:percentage5m
        expr: |
          (
            sum by (category) (rate(proxynd_enterprise_api_errors_total[5m])) /
            sum by (category) (rate(proxynd_enterprise_api_requests_total[5m]))
          ) * 100

      # P50, P95, P99 latency
      - record: enterprise_api:latency_p50:histogram_quantile5m
        expr: |
          histogram_quantile(0.50,
            sum by (category, le) (
              rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])
            )
          )

      - record: enterprise_api:latency_p95:histogram_quantile5m
        expr: |
          histogram_quantile(0.95,
            sum by (category, le) (
              rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])
            )
          )

      - record: enterprise_api:latency_p99:histogram_quantile5m
        expr: |
          histogram_quantile(0.99,
            sum by (category, le) (
              rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])
            )
          )

      # Cache hit rate
      - record: enterprise_api:cache_hit_rate:percentage5m
        expr: |
          (
            sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) /
            (
              sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) +
              sum(rate(proxynd_enterprise_api_cache_misses_total[5m]))
            )
          ) * 100
```

Add to `prometheus.yml`:

```yaml
rule_files:
  - 'alert-rules.yml'
  - 'recording-rules.yml'
```

## 🔍 Querying Metrics

### Useful PromQL Queries

**Request rate (overall)**:
```promql
sum(rate(proxynd_enterprise_api_requests_total[5m]))
```

**Request rate by category**:
```promql
sum(rate(proxynd_enterprise_api_requests_total[5m])) by (category)
```

**P95 latency**:
```promql
histogram_quantile(0.95,
  sum by (category, le) (
    rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])
  )
)
```

**Error rate percentage**:
```promql
(
  sum(rate(proxynd_enterprise_api_errors_total[5m])) /
  sum(rate(proxynd_enterprise_api_requests_total[5m]))
) * 100
```

**Cache hit rate**:
```promql
(
  sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) /
  (
    sum(rate(proxynd_enterprise_api_cache_hits_total[5m])) +
    sum(rate(proxynd_enterprise_api_cache_misses_total[5m]))
  )
) * 100
```

**Top 5 slowest endpoints**:
```promql
topk(5,
  histogram_quantile(0.95,
    sum by (endpoint, le) (
      rate(proxynd_enterprise_api_request_duration_seconds_bucket[5m])
    )
  )
)
```

## 📁 File Structure

```
deployments/prometheus/
├── README.md               # This file
├── alert-rules.yml         # Alert rules for monitoring
├── recording-rules.yml     # Pre-calculated metrics (optional)
├── prometheus.yml          # Main configuration (example)
└── alertmanager.yml        # Alertmanager config (example)
```

## 🔗 Integration with Grafana

Import pre-built dashboard that uses these metrics:

1. **Import Dashboard**:
   - File: `deployments/grafana/enterprise-api-dashboard.json`
   - See: [Grafana README](../grafana/README.md)

2. **Configure Datasource**:
   ```yaml
   datasources:
     - name: prometheus
       type: prometheus
       url: http://prometheus:9090
       access: proxy
       isDefault: true
   ```

3. **View Metrics**: http://localhost:3000/d/proxynd-enterprise-api/

## 🐛 Troubleshooting

### Alerts Not Firing

**Check alert rules are loaded**:
```bash
curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[] | select(.type=="alerting")'
```

**Check rule syntax**:
```bash
promtool check rules alert-rules.yml
```

**Manually evaluate rule**:
```bash
# Copy the alert expression and test in Prometheus UI
# http://localhost:9090/graph
```

### Metrics Not Appearing

**Verify ProxyND is being scraped**:
```bash
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.job=="proxynd-enterprise-api")'
```

**Test ProxyND metrics endpoint**:
```bash
curl http://localhost:8080/metrics | grep enterprise_api
```

**Check for scrape errors**:
- Open Prometheus UI: http://localhost:9090/targets
- Look for red targets with error messages

### High Memory Usage

**Reduce retention period**:
```bash
# Add to Prometheus startup flags
--storage.tsdb.retention.time=15d  # Default: 15 days
--storage.tsdb.retention.size=50GB # Add size limit
```

**Optimize queries**:
- Use recording rules for expensive queries
- Increase `evaluation_interval` in prometheus.yml
- Reduce `scrape_interval` for less critical targets

## 📖 Further Reading

- **Prometheus Documentation**: https://prometheus.io/docs/
- **PromQL Tutorial**: https://prometheus.io/docs/prometheus/latest/querying/basics/
- **Alerting Best Practices**: https://prometheus.io/docs/practices/alerting/
- **Enterprise API Metrics**: [../../docs/api/METRICS.md](../../docs/api/METRICS.md)
- **Grafana Dashboards**: [../grafana/README.md](../grafana/README.md)
- **Load Testing**: [../../scripts/loadtest/README.md](../../scripts/loadtest/README.md)

## 🆘 Support

For issues or questions:
- **GitHub Issues**: https://github.com/ScriptonBasestar/proxynd/issues
- **Prometheus Community**: https://prometheus.io/community/
- **Email**: support@proxynd.io

---

**Last Updated**: 2025-01-17
**Alert Rules**: 21 total (see alert-rules.yml)
**Metrics**: 15+ available (see docs/api/METRICS.md)
