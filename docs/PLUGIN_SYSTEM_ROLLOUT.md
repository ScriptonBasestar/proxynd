# Plugin System Rollout Playbook

**Version:** 1.0
**Last Updated:** 2025-11-18
**Owner:** DevOps Team

## Table of Contents

1. [Overview](#overview)
2. [Pre-Deployment Checklist](#pre-deployment-checklist)
3. [Deployment Steps](#deployment-steps)
4. [Validation & Testing](#validation--testing)
5. [Rollback Procedures](#rollback-procedures)
6. [Operator Guide](#operator-guide)
7. [Troubleshooting](#troubleshooting)
8. [Monitoring & Alerts](#monitoring--alerts)

---

## Overview

This playbook describes the rollout process for ProxyND's dynamic plugin system, which introduces:

- **Config-driven plugin management** via `plugins.yaml`
- **Priority-based initialization** order
- **Lifecycle hooks** (Init, Ready, Shutdown)
- **Environment-specific overrides** (dev/staging/production)
- **Comprehensive validation** for plugin priorities and duplicates
- **WebUI API integration** for package manager control

### Scope

- Core edition plugin system rollout
- Enterprise and Cloud edition plugin support
- WebUI API endpoints for package manager toggling

### Timeline

- **Phase 1**: Development/Staging deployment (Day 1-3)
- **Phase 2**: Production canary deployment (Day 4-7)
- **Phase 3**: Full production rollout (Day 8-10)

---

## Pre-Deployment Checklist

### 1. Configuration Validation

Before deploying, ensure all plugin configurations are validated:

```bash
# Validate plugin configuration
./scripts/validate-plugin-config.sh /path/to/config/plugins.yaml

# Check for common issues:
# - Duplicate plugin names
# - Conflicting priorities
# - Invalid priority ranges
# - Missing lifecycle timeouts
```

**Expected Priority Ranges:**
- Core plugins: 100-199
- Enterprise plugins: 200-299
- Cloud plugins: 300-399

**Example Valid Configuration:**

```yaml
# plugins.yaml
enabled: true
discoveryPaths:
  - "./plugins"
  - "/etc/proxynd/plugins"

lifecycle:
  initTimeout: 30s
  readyTimeout: 10s
  shutdownTimeout: 30s
  failurePolicy: halt  # Use "halt" for production

registry:
  core:
    - name: npm
      enabled: true
      priority: 100
    - name: maven
      enabled: true
      priority: 110
    - name: docker
      enabled: true
      priority: 120

  enterprise:
    - name: rbac
      enabled: true
      priority: 200
    - name: audit
      enabled: true
      priority: 210

  cloud:
    - name: multitenancy
      enabled: true
      priority: 300

environments:
  production:
    overrides:
      - plugin: rbac
        enabled: true
      - plugin: audit
        enabled: true
        config:
          retention_days: 90
```

### 2. Environment Preparation

Ensure the following environment variables are set:

```bash
# Required
export CONFIG_DIR="/etc/proxynd/config"
export STORAGE_DIR="/var/lib/proxynd/storage"
export SERVER_PORT="8080"

# Optional (but recommended for production)
export PROXYND_ENV="production"
export LOG_LEVEL="info"
export LOG_FORMAT="json"
```

### 3. Backup Current Configuration

```bash
# Backup current config
cp -r /etc/proxynd/config /etc/proxynd/config.backup.$(date +%Y%m%d-%H%M%S)

# Backup database/state if applicable
tar -czf /var/backups/proxynd-state-$(date +%Y%m%d).tar.gz /var/lib/proxynd/
```

### 4. Verify Dependencies

```bash
# Ensure all required tools are available
which go
which make
which git

# Verify Go version (1.21+ required)
go version

# Check disk space
df -h /var/lib/proxynd
df -h /etc/proxynd
```

---

## Deployment Steps

### Phase 1: Development/Staging Deployment

#### Step 1: Build the New Version

```bash
# Clone or pull latest code
git checkout claude/plugin-system-rollout-01L3jaRDzV6y9wgMG8PJfdKf
git pull origin claude/plugin-system-rollout-01L3jaRDzV6y9wgMG8PJfdKf

# Run tests
make test-unit
make test-integration

# Build the binary
make build

# Verify the build
./tmp/bin/proxynd --version
```

#### Step 2: Deploy Plugin Configuration

```bash
# Copy plugin configuration to config directory
cp examples/plugins/plugins.development.yaml /etc/proxynd/config/plugins.yaml

# Validate the configuration
./tmp/bin/proxynd validate-config --config /etc/proxynd/config/plugins.yaml
```

#### Step 3: Restart the Service (Staging)

```bash
# Stop the current service
sudo systemctl stop proxynd

# Replace the binary
sudo cp ./tmp/bin/proxynd /usr/local/bin/proxynd

# Start the service with new plugin system
sudo systemctl start proxynd

# Check service status
sudo systemctl status proxynd

# Monitor logs
sudo journalctl -u proxynd -f
```

#### Step 4: Verify Plugin Initialization

```bash
# Check plugin status via API
curl -s http://localhost:8080/api/v1/plugins/status | jq .

# Expected output:
# {
#   "enabled": true,
#   "total_plugins": 7,
#   "initialized_plugins": 7,
#   "failed_plugins": 0,
#   "plugins": [
#     {
#       "name": "npm",
#       "enabled": true,
#       "priority": 100,
#       "status": "ready"
#     },
#     ...
#   ]
# }
```

### Phase 2: Production Canary Deployment

#### Step 1: Deploy to Canary Instances (10% of traffic)

```bash
# Update canary instances only
ansible-playbook -i inventories/production playbooks/deploy-canary.yml \
  --extra-vars "version=claude/plugin-system-rollout-01L3jaRDzV6y9wgMG8PJfdKf"
```

#### Step 2: Monitor Canary Metrics

Monitor for 24-48 hours:

```bash
# Check error rates
curl -s http://canary.proxynd.internal:8080/metrics | grep proxynd_errors_total

# Check plugin initialization times
curl -s http://canary.proxynd.internal:8080/api/v1/plugins/metrics | jq '.plugins[] | {name, init_duration_ms}'

# Monitor system health
curl -s http://canary.proxynd.internal:8080/health | jq .
```

**Success Criteria:**
- ✅ Error rate < 0.1%
- ✅ Plugin initialization time < 5s per plugin
- ✅ No failed plugin initializations
- ✅ All health checks passing

#### Step 3: Gradual Rollout

If canary is successful, proceed with gradual rollout:

```bash
# Deploy to 25% of instances
ansible-playbook -i inventories/production playbooks/deploy-gradual.yml \
  --extra-vars "percentage=25 version=claude/plugin-system-rollout-01L3jaRDzV6y9wgMG8PJfdKf"

# Wait 2-4 hours, monitor metrics

# Deploy to 50%
ansible-playbook -i inventories/production playbooks/deploy-gradual.yml \
  --extra-vars "percentage=50 version=claude/plugin-system-rollout-01L3jaRDzV6y9wgMG8PJfdKf"

# Wait 2-4 hours, monitor metrics

# Deploy to 100%
ansible-playbook -i inventories/production playbooks/deploy-gradual.yml \
  --extra-vars "percentage=100 version=claude/plugin-system-rollout-01L3jaRDzV6y9wgMG8PJfdKf"
```

---

## Validation & Testing

### 1. Config Validation Tests

```bash
# Test 1: Validate duplicate detection
cat << EOF > /tmp/test-duplicate.yaml
enabled: true
registry:
  core:
    - name: npm
      enabled: true
      priority: 100
    - name: npm  # Duplicate!
      enabled: true
      priority: 110
EOF

./tmp/bin/proxynd validate-config --config /tmp/test-duplicate.yaml
# Expected: Error about duplicate plugin name

# Test 2: Validate priority conflicts
cat << EOF > /tmp/test-priority-conflict.yaml
enabled: true
registry:
  core:
    - name: npm
      enabled: true
      priority: 100
    - name: maven
      enabled: true
      priority: 100  # Conflict!
EOF

./tmp/bin/proxynd validate-config --config /tmp/test-priority-conflict.yaml
# Expected: Error about conflicting priority

# Test 3: Validate priority ranges
cat << EOF > /tmp/test-invalid-range.yaml
enabled: true
registry:
  core:
    - name: npm
      enabled: true
      priority: 250  # Invalid for core (should be 100-199)
EOF

./tmp/bin/proxynd validate-config --config /tmp/test-invalid-range.yaml
# Expected: Error about invalid priority range
```

### 2. API Endpoint Tests

```bash
# Test 1: Get package manager list
curl -s http://localhost:8080/api/v1/pm | jq .
# Expected: List of all package managers with enabled status

# Test 2: Toggle package manager (explicit enable)
curl -X POST http://localhost:8080/api/v1/pm/npm/toggle \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}' | jq .
# Expected: {"name": "npm", "enabled": false, "previous_state": true, ...}

# Test 3: Toggle package manager (toggle mode)
curl -X POST http://localhost:8080/api/v1/pm/npm/toggle | jq .
# Expected: {"name": "npm", "enabled": true, "previous_state": false, ...}

# Test 4: Invalid package manager
curl -X POST http://localhost:8080/api/v1/pm/invalid/toggle | jq .
# Expected: 400 error with list of valid package managers

# Test 5: Not implemented package managers (yum, apk)
curl -X POST http://localhost:8080/api/v1/pm/yum/toggle | jq .
# Expected: 501 Not Implemented error
```

### 3. Plugin Lifecycle Tests

```bash
# Test 1: Check plugin initialization order
curl -s http://localhost:8080/api/v1/plugins/metrics | jq '.plugins | sort_by(.priority)'
# Expected: Plugins sorted by priority (100, 110, 120, ...)

# Test 2: Verify lifecycle hooks executed
sudo journalctl -u proxynd | grep "Plugin initialized successfully"
# Expected: Log entries for all enabled plugins

# Test 3: Test graceful shutdown
sudo systemctl stop proxynd
sudo journalctl -u proxynd | grep "Plugin shut down successfully"
# Expected: Shutdown logs in reverse priority order
```

---

## Rollback Procedures

### Emergency Rollback (< 5 minutes)

If critical issues are detected:

```bash
# Stop the service immediately
sudo systemctl stop proxynd

# Restore previous binary
sudo cp /usr/local/bin/proxynd.backup /usr/local/bin/proxynd

# Restore previous config
sudo rm /etc/proxynd/config/plugins.yaml

# Start the service
sudo systemctl start proxynd

# Verify service is healthy
curl -s http://localhost:8080/health | jq .
```

### Standard Rollback (Production)

```bash
# Rollback via Ansible
ansible-playbook -i inventories/production playbooks/rollback.yml \
  --extra-vars "target_version=v1.2.3"

# Monitor rollback progress
ansible-playbook -i inventories/production playbooks/check-health.yml

# Verify all instances are healthy
for host in $(cat inventories/production | grep -v '^\['); do
  echo "Checking $host..."
  curl -s http://$host:8080/health | jq '.status'
done
```

### Rollback Decision Matrix

| Symptom | Severity | Action |
|---------|----------|--------|
| Plugin initialization fails (failurePolicy=halt) | **Critical** | Immediate rollback |
| Plugin initialization timeout (>30s) | **High** | Rollback after 2nd occurrence |
| Single plugin failure (failurePolicy=continue) | **Medium** | Investigate, rollback if >1 plugin fails |
| Increased error rate (>1%) | **High** | Rollback to canary only, investigate |
| Performance degradation (>20% slower) | **Medium** | Rollback after 1 hour if not resolved |
| WebUI API errors | **Medium** | Rollback if affecting >5% of requests |

---

## Operator Guide

### Daily Operations

#### Check Plugin Status

```bash
# View all plugin status
curl -s http://localhost:8080/api/v1/plugins/status | jq .

# Check specific plugin
curl -s http://localhost:8080/api/v1/plugins/status/npm | jq .

# View plugin metrics
curl -s http://localhost:8080/api/v1/plugins/metrics | jq .
```

#### Manage Package Managers

```bash
# List all package managers
curl -s http://localhost:8080/api/v1/pm | jq .

# Enable a package manager
curl -X POST http://localhost:8080/api/v1/pm/maven/toggle \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}' | jq .

# Disable a package manager
curl -X POST http://localhost:8080/api/v1/pm/npm/toggle \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}' | jq .
```

#### Update Plugin Configuration

```bash
# Edit plugin configuration
sudo vim /etc/proxynd/config/plugins.yaml

# Validate changes
./tmp/bin/proxynd validate-config --config /etc/proxynd/config/plugins.yaml

# Reload configuration (requires restart)
sudo systemctl restart proxynd

# Verify plugins loaded correctly
curl -s http://localhost:8080/api/v1/plugins/status | jq .
```

### Common Tasks

#### Add a New Plugin

1. **Update `plugins.yaml`:**

```yaml
registry:
  core:
    - name: new-plugin
      enabled: true
      priority: 150  # Must be unique and in correct range (100-199 for core)
      config:
        custom_setting: value
```

2. **Validate configuration:**

```bash
./tmp/bin/proxynd validate-config --config /etc/proxynd/config/plugins.yaml
```

3. **Restart service:**

```bash
sudo systemctl restart proxynd
```

4. **Verify plugin loaded:**

```bash
curl -s http://localhost:8080/api/v1/plugins/status | jq '.plugins[] | select(.name=="new-plugin")'
```

#### Change Plugin Priority

⚠️ **Warning:** Changing priorities affects initialization order. Plan carefully.

1. **Update priorities in `plugins.yaml`:**

```yaml
registry:
  core:
    - name: npm
      priority: 105  # Changed from 100
    - name: maven
      priority: 115  # Changed from 110
```

2. **Validate for conflicts:**

```bash
./tmp/bin/proxynd validate-config --config /etc/proxynd/config/plugins.yaml
```

3. **Restart and verify order:**

```bash
sudo systemctl restart proxynd
curl -s http://localhost:8080/api/v1/plugins/metrics | jq '.plugins | sort_by(.priority)'
```

#### Environment-Specific Overrides

For different behavior in production vs. staging:

```yaml
environments:
  production:
    overrides:
      - plugin: audit
        enabled: true
        config:
          retention_days: 90
          verbosity: info

  staging:
    overrides:
      - plugin: audit
        enabled: true
        config:
          retention_days: 7
          verbosity: debug
```

Set the environment via:

```bash
export PROXYND_ENV=production
```

---

## Troubleshooting

### Issue: Plugin Initialization Fails

**Symptoms:**
- Service fails to start
- Logs show "Plugin initialization failed"

**Diagnosis:**

```bash
sudo journalctl -u proxynd | grep -i "plugin.*failed"
```

**Solutions:**

1. Check configuration validation:
   ```bash
   ./tmp/bin/proxynd validate-config --config /etc/proxynd/config/plugins.yaml
   ```

2. Review failure policy:
   ```yaml
   lifecycle:
     failurePolicy: continue  # Change from "halt" temporarily
   ```

3. Check plugin dependencies:
   ```bash
   curl -s http://localhost:8080/api/v1/plugins/status/<plugin-name> | jq '.dependencies'
   ```

### Issue: Duplicate Plugin Names

**Symptoms:**
- Config validation fails with "duplicate plugin name"

**Diagnosis:**

```bash
./tmp/bin/proxynd validate-config --config /etc/proxynd/config/plugins.yaml
```

**Solution:**

Ensure each plugin name appears only once across all registries (core, enterprise, cloud).

### Issue: Conflicting Priorities

**Symptoms:**
- Config validation fails with "conflicting priority"

**Diagnosis:**

```bash
./tmp/bin/proxynd validate-config --config /etc/proxynd/config/plugins.yaml
```

**Solution:**

Assign unique priorities to each plugin. Use the documented ranges:
- Core: 100-199
- Enterprise: 200-299
- Cloud: 300-399

### Issue: WebUI API Toggle Not Working

**Symptoms:**
- POST /api/v1/pm/:name/toggle returns errors
- Package manager state doesn't change

**Diagnosis:**

```bash
# Check API response
curl -X POST http://localhost:8080/api/v1/pm/npm/toggle -v

# Check service logs
sudo journalctl -u proxynd | tail -n 50
```

**Solutions:**

1. Verify package manager is implemented:
   - Implemented: maven, npm, docker, pypi, apt
   - Not yet implemented: yum, apk (returns 501)

2. Check if RootConfig is loaded:
   ```bash
   curl -s http://localhost:8080/api/v1/system/info | jq .
   ```

3. Verify JSON payload:
   ```bash
   curl -X POST http://localhost:8080/api/v1/pm/npm/toggle \
     -H "Content-Type: application/json" \
     -d '{"enabled": true}' | jq .
   ```

### Issue: Performance Degradation

**Symptoms:**
- Slow plugin initialization
- High CPU during startup

**Diagnosis:**

```bash
# Check plugin init times
curl -s http://localhost:8080/api/v1/plugins/metrics | jq '.plugins[] | {name, init_duration_ms}'

# Check system resources
top
htop
```

**Solutions:**

1. Increase timeouts:
   ```yaml
   lifecycle:
     initTimeout: 60s  # Increased from 30s
   ```

2. Disable non-essential plugins temporarily

3. Review plugin logs for slow operations

---

## Monitoring & Alerts

### Key Metrics to Monitor

| Metric | Threshold | Action |
|--------|-----------|--------|
| `plugin_init_duration_seconds` | > 30s | Investigate slow plugin |
| `plugin_init_failures_total` | > 0 | Alert on-call |
| `plugin_ready_failures_total` | > 0 | Alert on-call |
| `plugin_enabled_count` | < expected | Check config |
| `http_requests_total{endpoint="/api/v1/pm/*/toggle", status="500"}` | > 10/min | Alert on-call |

### Prometheus Alert Rules

```yaml
groups:
  - name: proxynd_plugin_alerts
    rules:
      - alert: PluginInitializationFailed
        expr: plugin_init_failures_total > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Plugin initialization failed"
          description: "Plugin {{ $labels.plugin_name }} failed to initialize"

      - alert: PluginInitializationSlow
        expr: plugin_init_duration_seconds > 30
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Plugin initialization is slow"
          description: "Plugin {{ $labels.plugin_name }} took {{ $value }}s to initialize"

      - alert: PackageManagerToggleErrors
        expr: rate(http_requests_total{endpoint=~"/api/v1/pm/.*/toggle", status="500"}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Package manager toggle API errors"
          description: "High error rate on package manager toggle endpoint"
```

### Grafana Dashboard

Create a dashboard with the following panels:

1. **Plugin Status Overview**
   - Query: `plugin_enabled_count`, `plugin_initialized_count`

2. **Plugin Initialization Times**
   - Query: `histogram_quantile(0.95, plugin_init_duration_seconds)`

3. **Plugin Errors**
   - Query: `plugin_init_failures_total`, `plugin_ready_failures_total`

4. **API Request Rates**
   - Query: `rate(http_requests_total{endpoint="/api/v1/pm"}[5m])`

---

## Appendix

### A. Configuration Examples

See `examples/plugins/` for complete examples:
- `plugins.development.yaml` - Development configuration
- `plugins.production.yaml` - Production configuration
- `plugins.minimal.yaml` - Minimal configuration

### B. API Reference

#### GET /api/v1/pm
Returns list of all package managers with their enabled status.

#### POST /api/v1/pm/:name/toggle
Toggles or sets the enabled state of a package manager.

**Request Body (optional):**
```json
{
  "enabled": true  // If omitted, toggles current state
}
```

**Response:**
```json
{
  "name": "npm",
  "enabled": true,
  "previous_state": false,
  "message": "Package manager 'npm' enabled successfully"
}
```

### C. Support Contacts

- **On-Call Engineer:** oncall@proxynd.example.com
- **DevOps Team:** devops@proxynd.example.com
- **Slack Channel:** #proxynd-ops

---

**End of Playbook**
