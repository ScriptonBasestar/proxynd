# ProxyND Plugin System - Operations Guide

**Version**: 1.0
**Last Updated**: 2025-11-19
**Audience**: System Operators, SREs, DevOps Engineers
**Status**: Production Ready

---

## Overview

This guide is designed for operators responsible for monitoring, maintaining, and troubleshooting the ProxyND plugin system in production environments. For plugin development, see [PLUGIN_OPERATOR_GUIDE.md](PLUGIN_OPERATOR_GUIDE.md).

### What is the Plugin System?

The ProxyND plugin system provides extensibility through a config-driven architecture that allows enabling, disabling, and configuring plugins without code changes or service restarts (in most cases).

**Key Capabilities**:
- Runtime plugin management
- Event-driven architecture (3 event types)
- Comprehensive health monitoring
- Prometheus metrics integration
- Flexible lifecycle control

### Plugin Categories

1. **Core Plugins**: Always available, included in standard builds
2. **Enterprise Plugins**: Require enterprise edition (LDAP, SAML, RBAC, Audit)
3. **Cloud Plugins**: Require cloud edition (Multi-tenancy, Billing)

---

## Quick Health Check

### Using the Health Endpoint

The health endpoint provides real-time status of the plugin system:

```bash
curl http://localhost:8080/api/v1/plugins/health | jq
```

**Sample Response**:
```json
{
  "enabled": true,
  "initialized": true,
  "ready": true,
  "total_plugins": 5,
  "enabled_plugins": 3,
  "ready_plugins": 3,
  "failed_plugins": 0,
  "plugins": [
    {
      "name": "event_logger",
      "enabled": true,
      "status": "ready",
      "priority": 100,
      "error_count": 0,
      "init_time_ms": 5,
      "ready_time_ms": 2,
      "state_changes": 3,
      "last_state_change": "2025-11-19T10:30:00Z"
    }
  ],
  "summary": {
    "total_errors": 0,
    "avg_init_time_ms": 10
  },
  "timestamp": "2025-11-19T10:30:00Z"
}
```

### Interpreting Health Status

**Healthy System**:
- `enabled: true` - Plugin system is active
- `initialized: true` - All plugins initialized
- `ready: true` - All plugins ready to handle events
- `failed_plugins: 0` - No plugin failures
- All plugin status: `"ready"`

**Degraded System**:
- `failed_plugins > 0` - Some plugins failed
- Plugin status: `"failed"` - Check `error_count` and `last_error`
- `ready: false` - System not fully operational

**Disabled System**:
- `enabled: false` - Plugin system turned off (check `plugins.yaml`)

---

## Health Monitoring

### Continuous Monitoring

Set up automated health checks with your monitoring system:

**Prometheus ServiceMonitor**:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: proxynd-plugin-health
spec:
  selector:
    matchLabels:
      app: proxynd
  endpoints:
  - port: http
    path: /api/v1/plugins/health
    interval: 30s
    scrapeTimeout: 10s
```

**Healthcheck Script** (cron every 5 minutes):
```bash
#!/bin/bash
# /usr/local/bin/proxynd-plugin-health.sh

HEALTH_URL="http://localhost:8080/api/v1/plugins/health"
RESPONSE=$(curl -s "$HEALTH_URL")

# Check if plugin system is enabled and ready
ENABLED=$(echo "$RESPONSE" | jq -r '.enabled')
READY=$(echo "$RESPONSE" | jq -r '.ready')
FAILED=$(echo "$RESPONSE" | jq -r '.failed_plugins')

if [ "$ENABLED" != "true" ]; then
    echo "CRITICAL: Plugin system is disabled"
    exit 2
fi

if [ "$READY" != "true" ]; then
    echo "WARNING: Plugin system not ready"
    exit 1
fi

if [ "$FAILED" -gt 0 ]; then
    echo "WARNING: $FAILED plugin(s) failed"
    exit 1
fi

echo "OK: Plugin system healthy (ready plugins: $(echo "$RESPONSE" | jq -r '.ready_plugins'))"
exit 0
```

**Add to crontab**:
```bash
*/5 * * * * /usr/local/bin/proxynd-plugin-health.sh >> /var/log/proxynd/plugin-health.log 2>&1
```

### Health Monitoring Dashboard

Create a simple dashboard to track plugin health:

**Key Metrics to Display**:
1. **Plugin Status** - Pie chart (ready vs failed)
2. **Error Count Over Time** - Line graph
3. **Initialization Time** - Bar chart by plugin
4. **State Changes** - Counter

**Tools**: Grafana, Datadog, New Relic, or custom dashboard

---

## Prometheus Metrics Monitoring

### Available Metrics

The plugin event system exposes 4 Prometheus metrics:

1. **`proxynd_plugin_events_total{event_type}`** (Counter)
   - Total events dispatched by type
   - Event types: `package_manager_state_changed`, `config_reloaded`, `cache_cleared`

2. **`proxynd_plugin_event_processing_duration_seconds{event_type, plugin_name}`** (Histogram)
   - Event processing time in seconds
   - Buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10

3. **`proxynd_plugin_event_errors_total{event_type, plugin_name, error_type}`** (Counter)
   - Event processing errors
   - Error types: `handler_error`, `context_timeout`

4. **`proxynd_plugin_event_listeners_total{event_type}`** (Gauge)
   - Active event listeners (plugins implementing EventHandler)

### Accessing Metrics

```bash
# View all plugin metrics
curl http://localhost:8080/metrics | grep proxynd_plugin

# View specific metric
curl http://localhost:8080/metrics | grep proxynd_plugin_events_total
```

### Essential PromQL Queries

**Event Throughput**:
```promql
# Events per second by type
sum by (event_type) (rate(proxynd_plugin_events_total[5m]))

# Total events in last hour
sum(increase(proxynd_plugin_events_total[1h]))
```

**Performance Monitoring**:
```promql
# 95th percentile processing time
histogram_quantile(0.95,
  sum(rate(proxynd_plugin_event_processing_duration_seconds_bucket[5m]))
  by (event_type, le))

# Average processing time by plugin
rate(proxynd_plugin_event_processing_duration_seconds_sum[5m]) /
rate(proxynd_plugin_event_processing_duration_seconds_count[5m])
```

**Error Monitoring**:
```promql
# Error rate by plugin
sum by (plugin_name) (rate(proxynd_plugin_event_errors_total[5m]))

# Context timeout rate
rate(proxynd_plugin_event_errors_total{error_type="context_timeout"}[5m])
```

**Active Listeners**:
```promql
# Current listeners by event type
sum by (event_type) (proxynd_plugin_event_listeners_total)
```

### Grafana Dashboard

**Import Pre-built Dashboard**:

1. Open Grafana → **Dashboards** → **Import**
2. Upload: `deployments/grafana/plugin-events-dashboard.json`
3. Select Prometheus datasource
4. Click **Import**

**Dashboard Panels**:
- Plugin Events Rate (timeseries)
- Total Event Listeners (gauge)
- Event Processing Duration p95/p99 (timeseries)
- Event Processing Errors Rate (timeseries)
- Event Types Distribution (bars)

### Recommended Alerts

**Critical Alerts** (PagerDuty/Opsgenie):

```yaml
# No active listeners - events not being processed
- alert: PluginEventNoListeners
  expr: sum(proxynd_plugin_event_listeners_total) == 0
  for: 5m
  severity: critical
```

**Warning Alerts** (Slack/Email):

```yaml
# High error rate
- alert: PluginEventHighErrorRate
  expr: rate(proxynd_plugin_event_errors_total[5m]) > 0.1
  for: 5m
  severity: warning

# Slow processing
- alert: PluginEventSlowProcessing
  expr: |
    histogram_quantile(0.95,
      sum(rate(proxynd_plugin_event_processing_duration_seconds_bucket[5m]))
      by (event_type, le)) > 1.0
  for: 10m
  severity: warning

# Context timeouts
- alert: PluginEventContextTimeout
  expr: rate(proxynd_plugin_event_errors_total{error_type="context_timeout"}[5m]) > 0
  for: 5m
  severity: warning
```

**Alert File**: `deployments/prometheus/alert-rules-plugins.yml`

---

## Event System

### Understanding Events

The plugin system uses 3 event types to notify plugins of system changes:

| Event Type | Trigger | Use Case |
|------------|---------|----------|
| `package_manager_state_changed` | PM enabled/disabled via API | Plugin needs to react to PM availability changes |
| `config_reloaded` | Config reload via API | Plugin needs to refresh its configuration |
| `cache_cleared` | Cache cleared via API | Plugin needs to invalidate internal caches |

### Event Flow

```
1. API request received (e.g., POST /api/v1/pm/npm/toggle)
   ↓
2. Operation completed (e.g., npm enabled)
   ↓
3. Event created with metadata
   ↓
4. Plugin manager notifies all EventHandler plugins
   ↓
5. Each plugin processes event (max 5s timeout per plugin)
   ↓
6. Success/error metrics recorded
```

### Event Timeout Behavior

**Default Timeout**: 5 seconds per plugin

**What Happens on Timeout**:
1. Plugin marked as timed out
2. Error recorded: `error_type="context_timeout"`
3. Other plugins continue processing
4. API request succeeds (best-effort notification)

**Monitoring Timeouts**:
```bash
# Check for timeout errors in metrics
curl http://localhost:8080/metrics | grep 'error_type="context_timeout"'

# View timeout errors in logs
tail -f /var/log/proxynd/proxynd.log | grep -i timeout
```

### Event Processing Best Practices

**For Operators**:
- Monitor timeout metrics regularly
- Investigate plugins with frequent timeouts
- Consider increasing timeout if legitimate slow operations
- Review plugin health endpoint for error details

---

## Troubleshooting

### Problem: Plugin Not Initializing

**Symptoms**:
- Health endpoint shows plugin status: `"unknown"` or `"failed"`
- `initialized: false` in health response
- Startup logs show initialization errors

**Diagnosis**:
```bash
# Check health endpoint
curl http://localhost:8080/api/v1/plugins/health | jq '.plugins[] | select(.status != "ready")'

# Check application logs
tail -100 /var/log/proxynd/proxynd.log | grep -i "plugin.*init"

# Check plugin configuration
cat /etc/proxynd/config/plugins.yaml
```

**Common Causes & Solutions**:

1. **Plugin Disabled in Config**
   ```yaml
   # Check plugins.yaml
   core:
     - name: "my-plugin"
       enabled: false  # ← Plugin disabled
   ```
   **Solution**: Set `enabled: true` and restart

2. **Missing Dependencies**
   - Check plugin requires environment variables
   - Verify required services are running
   - Review plugin-specific config

3. **Initialization Timeout**
   ```yaml
   lifecycle:
     initTimeout: 30s  # ← Too short
   ```
   **Solution**: Increase timeout in plugins.yaml

4. **Configuration Error**
   - Invalid plugin config syntax
   - Missing required config fields
   - Check logs for validation errors

### Problem: High Event Error Rate

**Symptoms**:
- Prometheus alert: `PluginEventHighErrorRate`
- Increasing `proxynd_plugin_event_errors_total` metric
- Health endpoint shows high `error_count` for plugins

**Diagnosis**:
```bash
# Check error metrics by plugin
curl http://localhost:8080/metrics | grep proxynd_plugin_event_errors_total

# Check health endpoint for errors
curl http://localhost:8080/api/v1/plugins/health | jq '.plugins[] | select(.error_count > 0)'

# View recent errors in logs
tail -200 /var/log/proxynd/proxynd.log | grep -i "plugin.*error"
```

**Common Causes & Solutions**:

1. **Plugin Handler Errors**
   - Bug in plugin event handler code
   - External service unavailable (database, API)
   - Invalid event data format

   **Solution**: Check plugin logs, fix handler, or disable problematic plugin

2. **Context Timeouts**
   - Plugin event handler too slow (> 5s)
   - Blocking operations in handler

   **Solution**:
   ```yaml
   # Temporarily increase timeout for investigation
   # (Not recommended long-term)
   ```
   - Optimize plugin handler performance
   - Move slow operations to background

3. **Resource Exhaustion**
   - Plugin consuming too much memory/CPU
   - Database connection pool exhausted

   **Solution**: Monitor resource usage, scale resources, or optimize plugin

### Problem: Events Not Being Processed

**Symptoms**:
- Prometheus alert: `PluginEventNoListeners`
- `proxynd_plugin_event_listeners_total` = 0
- PM toggles/config reloads don't trigger plugin reactions

**Diagnosis**:
```bash
# Check listener count
curl http://localhost:8080/metrics | grep proxynd_plugin_event_listeners_total

# Check plugin health
curl http://localhost:8080/api/v1/plugins/health | jq '.enabled, .ready'

# Check if any plugins implement EventHandler
grep -r "OnEvent" /var/log/proxynd/
```

**Common Causes & Solutions**:

1. **Plugin System Disabled**
   ```yaml
   plugins:
     enabled: false  # ← System disabled
   ```
   **Solution**: Set `enabled: true` in plugins.yaml and restart

2. **No Plugins Implement EventHandler**
   - All plugins are core plugins without event handling
   - Enterprise/cloud plugins not enabled

   **Solution**: Enable appropriate plugins with EventHandler interface

3. **Plugins Failed During Initialization**
   - Plugins failed to initialize, so not listening to events
   - Check plugin health endpoint

   **Solution**: Fix plugin initialization issues first

### Problem: Slow Event Processing

**Symptoms**:
- Prometheus alert: `PluginEventSlowProcessing`
- High p95/p99 latency in metrics
- API requests feel slow

**Diagnosis**:
```bash
# Check processing duration
curl http://localhost:8080/metrics | grep proxynd_plugin_event_processing_duration

# Identify slow plugins
curl http://localhost:8080/api/v1/plugins/health | jq '.plugins | sort_by(.ready_time_ms) | reverse'
```

**Common Causes & Solutions**:

1. **Slow Plugin Handler**
   - Synchronous database queries
   - Blocking HTTP calls
   - Heavy computation

   **Solution**: Optimize plugin code or move to background processing

2. **Too Many Plugins**
   - Many plugins processing same event
   - Sequential processing taking too long

   **Solution**: Disable unnecessary plugins, optimize handlers

3. **Resource Contention**
   - High CPU/memory usage
   - Disk I/O bottleneck

   **Solution**: Scale resources, optimize queries, add caching

### Problem: Plugin System Not Responding

**Symptoms**:
- Health endpoint returns 503 or times out
- Plugin manager not found in context
- No plugin logs

**Diagnosis**:
```bash
# Check if service is running
systemctl status proxynd

# Check health endpoint
curl -v http://localhost:8080/api/v1/plugins/health

# Check main health endpoint
curl http://localhost:8080/health
```

**Common Causes & Solutions**:

1. **Service Not Running**
   **Solution**: Start the service
   ```bash
   systemctl start proxynd
   ```

2. **Plugin Manager Not Initialized**
   - Check application startup logs
   - Verify `plugins.yaml` exists

   **Solution**: Check config and restart

3. **Network/Firewall Issues**
   - Port 8080 not accessible
   - Firewall blocking requests

   **Solution**: Check firewall rules, network config

---

## Best Practices

### 1. Plugin Priority Configuration

**Recommendation**: Use consistent priority ranges

```yaml
# Production priority layout
cloud:
  - name: "multi-tenancy"
    priority: 10      # Critical security - init first

enterprise:
  - name: "rbac"
    priority: 50      # Security
  - name: "audit"
    priority: 60      # Security

core:
  - name: "proxy-policy"
    priority: 100     # Business logic
  - name: "metrics"
    priority: 200     # Observability
```

**Why**: Ensures security plugins initialize before business logic and shutdown after.

### 2. Failure Policy Selection

**Development**:
```yaml
lifecycle:
  failurePolicy: "continue"  # Don't block on plugin failures
```

**Staging**:
```yaml
lifecycle:
  failurePolicy: "warn"  # Log warnings but continue
```

**Production**:
```yaml
lifecycle:
  failurePolicy: "halt"  # Stop on critical plugin failures
```

**Why**: Strict failure handling in production prevents degraded security state.

### 3. Timeout Configuration

**Conservative (Production)**:
```yaml
lifecycle:
  initTimeout: 60s        # Generous for slow starts
  readyTimeout: 30s       # Plugins should be fast
  shutdownTimeout: 60s    # Allow graceful cleanup
```

**Aggressive (Development)**:
```yaml
lifecycle:
  initTimeout: 10s
  readyTimeout: 5s
  shutdownTimeout: 10s
```

**Why**: Production needs stability; development needs fast feedback.

### 4. Environment-Specific Overrides

```yaml
environments:
  development:
    overrides:
      - plugin: "rbac"
        enabled: false    # Disable auth in dev
      - plugin: "audit"
        config:
          logLevel: "debug"

  production:
    overrides:
      - plugin: "rbac"
        enabled: true
        config:
          strictMode: true
      - plugin: "audit"
        config:
          logLevel: "warn"
          retentionDays: 180
```

**Why**: Different requirements per environment without config duplication.

### 5. Monitoring Strategy

**Essential Monitors**:
1. Plugin health endpoint (every 5 minutes)
2. Prometheus alerts (critical + warning)
3. Error rate trending (daily)
4. Performance metrics (p95, p99)

**Recommended Tools**:
- **Alerting**: Prometheus Alertmanager → PagerDuty/Opsgenie
- **Dashboards**: Grafana (use provided dashboard)
- **Logs**: ELK stack or CloudWatch Logs
- **APM**: Datadog, New Relic, or custom

### 6. Regular Maintenance

**Weekly**:
- Review plugin error counts
- Check for timeout trends
- Verify all plugins in "ready" state

**Monthly**:
- Audit plugin configurations
- Review plugin priorities
- Update plugin timeouts if needed

**Quarterly**:
- Review Prometheus alert thresholds
- Update Grafana dashboards
- Test plugin failure scenarios

---

## Configuration Examples

### Minimal Production Config

```yaml
# /etc/proxynd/config/plugins.yaml
plugins:
  enabled: true

  lifecycle:
    initTimeout: 60s
    readyTimeout: 30s
    shutdownTimeout: 60s
    failurePolicy: "halt"

  registry:
    core:
      - name: "event_logger"
        enabled: true
        priority: 200
```

### Enterprise Production Config

```yaml
# /etc/proxynd/config/plugins.yaml
plugins:
  enabled: true

  lifecycle:
    initTimeout: 60s
    readyTimeout: 30s
    shutdownTimeout: 60s
    failurePolicy: "halt"

  registry:
    core:
      - name: "proxy-policy"
        enabled: true
        priority: 100

      - name: "event_logger"
        enabled: true
        priority: 200

    enterprise:
      - name: "rbac"
        enabled: true
        priority: 50
        config:
          adminRole: "admin"
          auditAll: true
          strictMode: true

      - name: "audit"
        enabled: true
        priority: 60
        config:
          retentionDays: 180
          logLevel: "warn"
          outputPath: "/var/log/proxynd/audit.log"

  environments:
    production:
      overrides:
        - plugin: "audit"
          config:
            retentionDays: 365  # Longer retention in prod
```

### High-Availability Config

```yaml
# /etc/proxynd/config/plugins.yaml
plugins:
  enabled: true

  lifecycle:
    initTimeout: 120s       # Longer for distributed systems
    readyTimeout: 60s
    shutdownTimeout: 120s
    failurePolicy: "halt"   # Strict for HA

  registry:
    cloud:
      - name: "multi-tenancy"
        enabled: true
        priority: 10
        config:
          isolationLevel: "strict"
          tenantDiscovery: "consul"

    enterprise:
      - name: "rbac"
        enabled: true
        priority: 50
        config:
          backend: "ldap"
          ldapURL: "ldap://ldap.example.com"

      - name: "audit"
        enabled: true
        priority: 60
        config:
          backend: "elasticsearch"
          esURL: "http://elasticsearch:9200"

    core:
      - name: "metrics"
        enabled: true
        priority: 200
        config:
          pushGateway: "http://pushgateway:9091"
```

---

## Operational Runbooks

### Runbook: Plugin Failure During Deployment

**Scenario**: New deployment causes plugin to fail initialization

**Steps**:

1. **Identify Failed Plugin**
   ```bash
   curl http://localhost:8080/api/v1/plugins/health | jq '.plugins[] | select(.status == "failed")'
   ```

2. **Check Logs**
   ```bash
   tail -200 /var/log/proxynd/proxynd.log | grep -i "plugin.*<plugin-name>"
   ```

3. **Determine Impact**
   - Is it a critical security plugin? → Rollback immediately
   - Is it an observability plugin? → Can proceed with fix

4. **Quick Mitigation**
   ```yaml
   # Edit /etc/proxynd/config/plugins.yaml
   - name: "<problematic-plugin>"
     enabled: false  # Disable temporarily
   ```

5. **Restart Service**
   ```bash
   systemctl restart proxynd
   ```

6. **Verify System Health**
   ```bash
   curl http://localhost:8080/api/v1/plugins/health
   curl http://localhost:8080/health
   ```

7. **Post-Incident**
   - Create ticket for plugin fix
   - Document failure mode
   - Update monitoring thresholds if needed

### Runbook: High Event Error Rate

**Scenario**: Prometheus alerts on high event error rate

**Steps**:

1. **Identify Problematic Plugin**
   ```bash
   # Check metrics
   curl http://localhost:8080/metrics | grep proxynd_plugin_event_errors_total | sort -k2 -nr | head -5
   ```

2. **Check Error Type**
   - `handler_error`: Plugin bug or external dependency failure
   - `context_timeout`: Plugin too slow

3. **For Handler Errors**:
   - Check external dependencies (database, APIs)
   - Review recent plugin code changes
   - Check plugin configuration

4. **For Timeouts**:
   - Review plugin performance metrics
   - Check resource utilization (CPU, memory)
   - Consider temporarily disabling plugin

5. **Temporary Mitigation**
   ```yaml
   # If non-critical plugin
   - name: "<slow-plugin>"
     enabled: false
   ```

6. **Permanent Fix**
   - Optimize plugin code
   - Fix external dependencies
   - Increase resources if needed

---

## Related Documentation

- [PLUGIN_OPERATOR_GUIDE.md](PLUGIN_OPERATOR_GUIDE.md) - Plugin development guide
- [plugins/METRICS.md](../plugins/METRICS.md) - Detailed metrics documentation
- [API_PM_TOGGLE_AUTHENTICATION.md](API_PM_TOGGLE_AUTHENTICATION.md) - PM toggle API
- [API_CONFIG_RELOAD_AUTHENTICATION.md](API_CONFIG_RELOAD_AUTHENTICATION.md) - Config reload API
- [API_CACHE_CLEAR_AUTHENTICATION.md](API_CACHE_CLEAR_AUTHENTICATION.md) - Cache clear API
- Grafana Dashboard: `deployments/grafana/plugin-events-dashboard.json`
- Prometheus Alerts: `deployments/prometheus/alert-rules-plugins.yml`

---

## Support and Escalation

### Self-Service Resources

1. **Health Endpoint**: `http://localhost:8080/api/v1/plugins/health`
2. **Metrics**: `http://localhost:8080/metrics`
3. **Logs**: `/var/log/proxynd/proxynd.log`
4. **Configuration**: `/etc/proxynd/config/plugins.yaml`

### Common Questions

**Q: Can I reload plugin configuration without restarting?**
A: Partial support. Some plugin configs can be reloaded via `POST /api/config/reload`, but plugin enable/disable requires restart.

**Q: What happens if a plugin fails during runtime?**
A: Event processing is best-effort. Failed plugins log errors but don't block events. Check metrics and health endpoint.

**Q: How do I temporarily disable a problematic plugin?**
A: Edit `plugins.yaml`, set `enabled: false`, restart service. Or use environment overrides.

**Q: Are plugin events synchronous or asynchronous?**
A: Asynchronous. Events are dispatched to all plugins concurrently with independent timeouts.

**Q: Can I add custom plugins?**
A: Yes, but requires code deployment. See PLUGIN_OPERATOR_GUIDE.md for development instructions.

---

## Changelog

**v1.0 (2025-11-19)**:
- Initial operations guide
- Health monitoring procedures
- Prometheus metrics monitoring
- Event system operations
- Comprehensive troubleshooting
- Best practices and runbooks
- Configuration examples for production
