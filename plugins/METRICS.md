# Plugin Event Metrics

ProxyND의 플러그인 이벤트 시스템에 대한 Prometheus 메트릭입니다.

## Available Metrics

### 1. proxynd_plugin_events_total

**Type**: Counter
**Labels**: `event_type`
**Description**: 이벤트 타입별 총 이벤트 디스패치 횟수

**Example**:
```promql
# Event dispatch rate (per second)
rate(proxynd_plugin_events_total[5m])

# Total events by type in last hour
increase(proxynd_plugin_events_total[1h])
```

**Event Types**:
- `package_manager_state_changed` - PM toggle 이벤트
- `config_reloaded` - Config reload 이벤트
- `cache_cleared` - Cache clear 이벤트

---

### 2. proxynd_plugin_event_processing_duration_seconds

**Type**: Histogram
**Labels**: `event_type`, `plugin_name`
**Description**: 플러그인별 이벤트 처리 시간 (초 단위)

**Buckets**: Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

**Example**:
```promql
# 95th percentile processing time by event type
histogram_quantile(0.95,
  sum(rate(proxynd_plugin_event_processing_duration_seconds_bucket[5m]))
  by (event_type, le))

# 99th percentile processing time
histogram_quantile(0.99,
  sum(rate(proxynd_plugin_event_processing_duration_seconds_bucket[5m]))
  by (event_type, le))

# Average processing time by plugin
rate(proxynd_plugin_event_processing_duration_seconds_sum[5m]) /
rate(proxynd_plugin_event_processing_duration_seconds_count[5m])
```

---

### 3. proxynd_plugin_event_errors_total

**Type**: Counter
**Labels**: `event_type`, `plugin_name`, `error_type`
**Description**: 이벤트 처리 에러 총 횟수

**Error Types**:
- `handler_error` - 플러그인 핸들러 내부 에러
- `context_timeout` - Context timeout (5초 초과)

**Example**:
```promql
# Error rate by event type
rate(proxynd_plugin_event_errors_total[5m])

# Total errors by plugin in last hour
sum by (plugin_name) (increase(proxynd_plugin_event_errors_total[1h]))

# Error rate by error type
sum by (error_type) (rate(proxynd_plugin_event_errors_total[5m]))
```

---

### 4. proxynd_plugin_event_listeners_total

**Type**: Gauge
**Labels**: `event_type`
**Description**: 이벤트 타입별 활성 리스너(플러그인) 수

**Example**:
```promql
# Current number of event listeners
proxynd_plugin_event_listeners_total

# Listeners by event type
sum by (event_type) (proxynd_plugin_event_listeners_total)
```

---

## Grafana Dashboard

Grafana 대시보드 예시가 제공됩니다:

**파일**: `deployments/grafana/plugin-events-dashboard.json`

### Import Instructions

1. Grafana UI에서 **Dashboards** → **Import** 클릭
2. `plugin-events-dashboard.json` 파일 업로드
3. Prometheus datasource 선택
4. **Import** 클릭

### Dashboard Panels

대시보드에는 다음 패널들이 포함됩니다:

1. **Plugin Events Rate** - 초당 이벤트 디스패치 비율
2. **Total Event Listeners** - 현재 활성 이벤트 리스너 수
3. **Event Processing Duration (p95/p99)** - 처리 시간 백분위수
4. **Event Processing Errors Rate** - 에러 발생 비율
5. **Event Types Distribution** - 최근 1시간 이벤트 타입 분포

---

## Alerting Rules

### Example Prometheus Alert Rules

```yaml
groups:
  - name: proxynd_plugin_events
    interval: 30s
    rules:
      # High error rate alert
      - alert: PluginEventHighErrorRate
        expr: |
          rate(proxynd_plugin_event_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
          component: plugin_system
        annotations:
          summary: "High plugin event error rate"
          description: "Plugin {{ $labels.plugin_name }} is experiencing high error rate for {{ $labels.event_type }} events ({{ $value }} errors/sec)"

      # Slow event processing alert
      - alert: PluginEventSlowProcessing
        expr: |
          histogram_quantile(0.95,
            sum(rate(proxynd_plugin_event_processing_duration_seconds_bucket[5m]))
            by (event_type, le)) > 1.0
        for: 10m
        labels:
          severity: warning
          component: plugin_system
        annotations:
          summary: "Slow plugin event processing"
          description: "Event type {{ $labels.event_type }} p95 processing time is {{ $value }}s (threshold: 1s)"

      # No event listeners alert
      - alert: PluginEventNoListeners
        expr: |
          sum(proxynd_plugin_event_listeners_total) == 0
        for: 5m
        labels:
          severity: critical
          component: plugin_system
        annotations:
          summary: "No plugin event listeners"
          description: "Plugin event system has no active listeners. Events are not being processed."

      # Context timeout alert
      - alert: PluginEventContextTimeout
        expr: |
          rate(proxynd_plugin_event_errors_total{error_type="context_timeout"}[5m]) > 0
        for: 5m
        labels:
          severity: warning
          component: plugin_system
        annotations:
          summary: "Plugin event context timeouts"
          description: "Plugin {{ $labels.plugin_name }} is experiencing context timeouts for {{ $labels.event_type }} events"
```

**File**: `deployments/prometheus/alert-rules-plugins.yml`

---

## Usage Examples

### Monitoring Event Throughput

```promql
# Events per second by type
sum by (event_type) (rate(proxynd_plugin_events_total[5m]))

# Total events in last 24 hours
sum(increase(proxynd_plugin_events_total[24h]))
```

### Analyzing Performance

```promql
# Slowest plugins
topk(5,
  rate(proxynd_plugin_event_processing_duration_seconds_sum[5m]) /
  rate(proxynd_plugin_event_processing_duration_seconds_count[5m])
)

# Events exceeding 1 second
sum(rate(proxynd_plugin_event_processing_duration_seconds_bucket{le="1.0"}[5m])) by (event_type)
```

### Error Analysis

```promql
# Error percentage by plugin
(sum by (plugin_name) (rate(proxynd_plugin_event_errors_total[5m])) /
 sum by (plugin_name) (rate(proxynd_plugin_events_total[5m]))) * 100

# Most common error types
topk(5, sum by (error_type) (rate(proxynd_plugin_event_errors_total[5m])))
```

---

## Integration with /metrics Endpoint

메트릭은 Prometheus 표준 `/metrics` 엔드포인트를 통해 노출됩니다:

```bash
# View all plugin event metrics
curl http://localhost:8080/metrics | grep proxynd_plugin

# Example output:
# proxynd_plugin_events_total{event_type="package_manager_state_changed"} 42
# proxynd_plugin_event_processing_duration_seconds_bucket{event_type="config_reloaded",plugin_name="event_logger",le="0.005"} 10
# proxynd_plugin_event_errors_total{error_type="handler_error",event_type="cache_cleared",plugin_name="audit_logger"} 2
# proxynd_plugin_event_listeners_total{event_type="package_manager_state_changed"} 3
```

---

## Best Practices

### 1. Monitoring Thresholds

권장 임계값:
- **Error Rate**: < 1% (0.01 errors/sec per event)
- **P95 Processing Time**: < 1 second
- **P99 Processing Time**: < 2.5 seconds
- **Context Timeouts**: 0 (should never timeout)

### 2. Dashboard Refresh Rate

- Production: 30초 ~ 1분
- Development: 10초
- Troubleshooting: 5초

### 3. Retention Policy

- High-resolution (5m): 7 days
- Mid-resolution (1h): 30 days
- Low-resolution (1d): 1 year

### 4. Alert Routing

- **Critical**: No event listeners → PagerDuty
- **Warning**: High error rate, slow processing → Slack
- **Info**: Metrics anomalies → Email

---

## Troubleshooting

### High Error Rate

1. Check logs for specific error messages
2. Identify problematic plugin:
   ```promql
   topk(5, sum by (plugin_name) (rate(proxynd_plugin_event_errors_total[5m])))
   ```
3. Review plugin implementation
4. Check upstream service availability

### Slow Processing

1. Identify slow event types:
   ```promql
   histogram_quantile(0.95,
     sum(rate(proxynd_plugin_event_processing_duration_seconds_bucket[5m]))
     by (event_type, le))
   ```
2. Profile plugin handlers
3. Check for blocking I/O operations
4. Consider async processing

### No Listeners

1. Verify plugin system is enabled (`plugins.enabled: true`)
2. Check plugin discovery logs
3. Verify plugin registration
4. Check event handler interface implementation

---

## Related Documentation

- [Plugin Operator Guide](../docs/PLUGIN_OPERATOR_GUIDE.md) - Plugin development guide
- [Prometheus Configuration](../deployments/prometheus/prometheus.yml) - Scrape configuration
- [Grafana Dashboards](../deployments/grafana/) - All available dashboards
- [Alert Rules](../deployments/prometheus/alert-rules.yml) - All alert configurations
