# ProxyND Monitoring Guide

## 📊 Overview

This guide provides comprehensive information about monitoring ProxyND using Prometheus, Grafana, Loki, and other observability tools.

## 🏗️ Architecture

### Components

1. **Prometheus** - Metrics collection and storage
2. **Grafana** - Visualization and dashboards
3. **Loki** - Log aggregation and querying
4. **Promtail** - Log shipping agent
5. **AlertManager** - Alert management and routing
6. **Jaeger** - Distributed tracing
7. **Blackbox Exporter** - Endpoint monitoring

### Data Flow

```
ProxyND → Prometheus → Grafana (Visualization)
         ↓
         AlertManager → Notifications

ProxyND → Promtail → Loki → Grafana (Logs)

ProxyND → Jaeger → Grafana (Tracing)
```

## 🚀 Quick Start

### 1. Start Monitoring Stack

```bash
# Start all monitoring services
cd monitoring/
docker-compose up -d

# Check service status
docker-compose ps
```

### 2. Access Dashboards

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **AlertManager**: http://localhost:9093
- **Jaeger**: http://localhost:16686

### 3. Configure ProxyND

```yaml
# global.yaml
metrics:
  enabled: true
  endpoint: "/metrics"
  prometheus:
    enabled: true
    port: 8080
    path: "/metrics"

logging:
  level: info
  format: json
  outputs:
    - type: file
      path: "/var/log/proxynd/app.log"
    - type: stdout

tracing:
  enabled: true
  jaeger:
    endpoint: "http://jaeger:14268/api/traces"
    sampler:
      type: const
      param: 1
```

## 📊 Metrics

### Core Metrics

#### HTTP Metrics
- `proxynd_http_requests_total` - Total HTTP requests
- `proxynd_http_request_duration_seconds` - Request duration histogram
- `proxynd_http_response_size_bytes` - Response size histogram

#### Cache Metrics
- `proxynd_cache_hits_total` - Cache hits
- `proxynd_cache_misses_total` - Cache misses
- `proxynd_cache_size_bytes` - Current cache size
- `proxynd_cache_max_size_bytes` - Maximum cache size

#### Security Metrics
- `proxynd_auth_failures_total` - Authentication failures
- `proxynd_unauthorized_attempts_total` - Unauthorized access attempts
- `proxynd_rate_limit_violations_total` - Rate limit violations

#### Business Metrics
- `proxynd_download_attempts_total` - Download attempts
- `proxynd_download_failures_total` - Download failures
- `proxynd_upload_attempts_total` - Upload attempts
- `proxynd_upload_failures_total` - Upload failures

### Custom Metrics

```go
// Example Go code for custom metrics
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    requestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "proxynd_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "status", "endpoint"},
    )

    requestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "proxynd_http_request_duration_seconds",
            Help: "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )

    cacheHits = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "proxynd_cache_hits_total",
            Help: "Total cache hits",
        },
        []string{"cache_type"},
    )
)

func RecordRequest(method, status, endpoint string, duration float64) {
    requestsTotal.WithLabelValues(method, status, endpoint).Inc()
    requestDuration.WithLabelValues(method, endpoint).Observe(duration)
}
```

## 🔍 Dashboards

### Available Dashboards

1. **ProxyND Overview** - General service health and performance
2. **ProxyND Security** - Security-related metrics and alerts
3. **ProxyND Performance** - Detailed performance analysis
4. **System Metrics** - Host and container resource usage
5. **Cache Analysis** - Cache performance and optimization

### Dashboard Features

- **Real-time monitoring** with configurable refresh intervals
- **Alerting integration** with visual alert states
- **Historical analysis** with customizable time ranges
- **Interactive filtering** by environment, service, and labels
- **Export capabilities** for reports and analysis

### Custom Dashboard Creation

```json
{
  "dashboard": {
    "title": "Custom ProxyND Dashboard",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(proxynd_http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      }
    ]
  }
}
```

## 📋 Logging

### Log Levels

- **DEBUG** - Detailed debug information
- **INFO** - General information
- **WARN** - Warning messages
- **ERROR** - Error messages
- **FATAL** - Fatal errors

### Log Format

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "component": "proxy.handler",
  "message": "Request processed successfully",
  "user_id": "user123",
  "ip_address": "192.168.1.100",
  "method": "GET",
  "endpoint": "/api/packages",
  "status": 200,
  "duration": 0.045,
  "cache_hit": true
}
```

### Log Aggregation

#### Loki Queries

```logql
# All ProxyND logs
{job="proxynd"}

# Error logs only
{job="proxynd"} |= "ERROR"

# Authentication failures
{job="proxynd"} | json | component = "auth" | level = "ERROR"

# Slow requests (>1s)
{job="proxynd"} | json | duration > 1.0

# Rate calculation
rate({job="proxynd"} |= "ERROR"[5m])
```

#### Log Analysis

```bash
# Find error patterns
grep -E "(ERROR|FATAL)" /var/log/proxynd/app.log | tail -50

# Count requests by status
grep "status" /var/log/proxynd/app.log | \
  jq '.status' | sort | uniq -c | sort -nr

# Find slow requests
grep "duration" /var/log/proxynd/app.log | \
  jq 'select(.duration > 1.0) | {timestamp, endpoint, duration}' | \
  sort_by(.duration) | reverse
```

## 🚨 Alerting

### Alert Rules

#### Service Availability
- **ProxyNDDown** - Service is down
- **ProxyNDHighLatency** - High response latency
- **ProxyNDHighErrorRate** - High error rate

#### Resource Usage
- **ProxyNDHighCPUUsage** - High CPU usage
- **ProxyNDHighMemoryUsage** - High memory usage
- **ProxyNDLowDiskSpace** - Low disk space

#### Security
- **ProxyNDHighFailedAuthRate** - High authentication failure rate
- **ProxyNDUnauthorizedAccess** - Unauthorized access attempts

### Alert Configuration

```yaml
# alertmanager.yml
route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'critical-alerts'
    - match:
        severity: warning
      receiver: 'warning-alerts'

receivers:
  - name: 'critical-alerts'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/...'
        channel: '#alerts-critical'
        title: 'CRITICAL: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
```

### Alert Channels

1. **Slack** - Real-time team notifications
2. **Email** - Detailed alert information
3. **PagerDuty** - Critical incident management
4. **Webhook** - Custom integrations

## 🔧 Configuration

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'proxynd'
    static_configs:
      - targets: ['proxynd:8080']
    scrape_interval: 10s
    metrics_path: '/metrics'
```

### Grafana Configuration

```yaml
# grafana.ini
[security]
admin_user = admin
admin_password = admin

[users]
allow_sign_up = false

[auth.anonymous]
enabled = false

[dashboards]
default_home_dashboard_path = /var/lib/grafana/dashboards/proxynd-overview.json
```

## 📈 Performance Monitoring

### Key Performance Indicators (KPIs)

1. **Availability** - Service uptime percentage
2. **Response Time** - Average and 95th percentile
3. **Throughput** - Requests per second
4. **Error Rate** - Percentage of failed requests
5. **Cache Hit Rate** - Cache efficiency

### Performance Queries

```promql
# Service availability
avg_over_time(up{job="proxynd"}[1h]) * 100

# Average response time
avg(rate(proxynd_http_request_duration_seconds_sum[5m])) /
avg(rate(proxynd_http_request_duration_seconds_count[5m]))

# Request throughput
sum(rate(proxynd_http_requests_total[5m]))

# Error rate
sum(rate(proxynd_http_requests_total{status=~"5.."}[5m])) /
sum(rate(proxynd_http_requests_total[5m])) * 100

# Cache hit rate
sum(rate(proxynd_cache_hits_total[5m])) /
(sum(rate(proxynd_cache_hits_total[5m])) + sum(rate(proxynd_cache_misses_total[5m]))) * 100
```

## 🔍 Troubleshooting

### Common Issues

#### 1. High Memory Usage

```bash
# Check memory usage
docker stats proxynd

# Analyze memory patterns
curl -s http://localhost:8080/debug/pprof/heap > heap.pprof
go tool pprof heap.pprof
```

#### 2. High Error Rate

```bash
# Check error logs
docker logs proxynd | grep ERROR

# Analyze error patterns
curl -s http://localhost:9090/api/v1/query?query=rate(proxynd_http_requests_total{status=~"5.."}[5m])
```

#### 3. Cache Issues

```bash
# Check cache metrics
curl -s http://localhost:8080/metrics | grep cache

# Monitor cache hit rate
curl -s http://localhost:9090/api/v1/query?query=proxynd_cache_hit_rate
```

### Debugging Tools

```bash
# Health check
curl -f http://localhost:8080/health

# Metrics endpoint
curl http://localhost:8080/metrics

# Debug endpoints
curl http://localhost:8080/debug/pprof/
curl http://localhost:8080/debug/vars
```

## 📊 Capacity Planning

### Resource Monitoring

```promql
# CPU usage trend
avg_over_time(rate(container_cpu_usage_seconds_total{container="proxynd"}[5m])[1h:5m])

# Memory usage trend
avg_over_time(container_memory_usage_bytes{container="proxynd"}[1h:5m])

# Disk usage trend
avg_over_time((node_filesystem_size_bytes{mountpoint="/storage"} - node_filesystem_avail_bytes{mountpoint="/storage"})[1h:5m])
```

### Scaling Indicators

1. **CPU Usage** > 70% for extended periods
2. **Memory Usage** > 80% of allocated memory
3. **Response Time** > 2 seconds for 95th percentile
4. **Error Rate** > 5% for any extended period

## 🛠️ Maintenance

### Regular Tasks

1. **Update dashboards** - Monthly review and updates
2. **Clean old metrics** - Configure retention policies
3. **Review alerts** - Adjust thresholds based on trends
4. **Backup configuration** - Regular backup of dashboards and alerts

### Monitoring Health

```bash
# Check Prometheus targets
curl -s http://localhost:9090/api/v1/targets

# Check AlertManager status
curl -s http://localhost:9093/api/v1/status

# Check Grafana health
curl -s http://localhost:3000/api/health
```

## 📚 Resources

### Documentation Links

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Loki Documentation](https://grafana.com/docs/loki/)
- [AlertManager Documentation](https://prometheus.io/docs/alerting/latest/alertmanager/)

### Best Practices

1. **Naming Conventions** - Use consistent metric naming
2. **Label Management** - Avoid high cardinality labels
3. **Alert Fatigue** - Set appropriate thresholds
4. **Dashboard Organization** - Group related metrics
5. **Performance** - Monitor monitoring system performance

---

**Last Updated**: 2024-07-17  
**Review Schedule**: Monthly  
**Next Review**: 2024-08-17
