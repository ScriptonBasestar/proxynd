# ProxyND Plugin System - Operator Guide

**Version**: 1.0
**Last Updated**: 2025-11-11
**Status**: Production Ready

---

## Overview

The ProxyND plugin system provides a flexible, configuration-driven approach to managing plugins with lifecycle hooks, priority-based initialization, and comprehensive observability.

### Key Features

- **Config-Driven**: Enable/disable plugins via YAML without code changes
- **Priority System**: Control plugin initialization order
- **Lifecycle Hooks**: Init → Ready → Shutdown with timeout handling
- **Environment Overrides**: Different settings for dev/staging/production
- **Failure Policies**: Choose how to handle plugin failures
- **Full Observability**: Metrics, logs, and state tracking

---

## Configuration

### Configuration File Location

The plugin system looks for `plugins.yaml` in the configured `CONFIG_DIR`:

```bash
export CONFIG_DIR=/etc/proxynd/config
# Plugin config expected at: /etc/proxynd/config/plugins.yaml
```

If `plugins.yaml` is not found, the system uses safe defaults (all plugins enabled with default priorities).

### Minimal Configuration

```yaml
plugins:
  enabled: true  # Global plugin system on/off
```

**Result**: All registered plugins are enabled with default settings.

### Production Configuration

```yaml
plugins:
  enabled: true

  lifecycle:
    initTimeout: 60s
    readyTimeout: 30s
    shutdownTimeout: 60s
    failurePolicy: "halt"  # Strict mode

  registry:
    # Core plugins (always available)
    core:
      - name: "proxy-policy"
        enabled: true
        priority: 100

    # Enterprise plugins (requires enterprise build)
    enterprise:
      - name: "rbac"
        enabled: true
        priority: 50
        config:
          adminRole: "admin"
          auditAll: true

      - name: "audit"
        enabled: true
        priority: 60
        config:
          retentionDays: 90
          logLevel: "info"

    # Cloud plugins (requires cloud build)
    cloud:
      - name: "multi-tenancy"
        enabled: true
        priority: 10  # Highest priority (init first, shutdown last)
        config:
          isolationLevel: "strict"

      - name: "billing"
        enabled: true
        priority: 20

  environments:
    development:
      overrides:
        - plugin: "rbac"
          enabled: false  # Disable auth in development

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

---

## Priority System

### Priority Ranges

| Range   | Purpose                     | Examples                  |
|---------|-----------------------------|---------------------------|
| 1-49    | Critical security plugins   | multi-tenancy (10)        |
| 50-99   | Enterprise security         | rbac (50), audit (60)     |
| 100-199 | Core business logic         | proxy-policy (100)        |
| 200-299 | Observability               | metrics (200), logging (210) |
| 300+    | Low-priority features       | UI extensions (300)       |

### Execution Order

- **Initialization**: Low priority → High priority (10, 50, 100, 200, 300)
- **Shutdown**: High priority → Low priority (300, 200, 100, 50, 10)

**Rationale**: Security plugins initialize first and shutdown last to protect the application throughout its lifecycle.

### Example Priority Calculation

```yaml
cloud:
  - name: "multi-tenancy"
    priority: 10  # Init 1st, Shutdown 5th

enterprise:
  - name: "rbac"
    priority: 50  # Init 2nd, Shutdown 4th

  - name: "audit"
    priority: 60  # Init 3rd, Shutdown 3rd

core:
  - name: "proxy-policy"
    priority: 100  # Init 4th, Shutdown 2nd

  - name: "metrics"
    priority: 200  # Init 5th, Shutdown 1st
```

**Lifecycle Order**:

1. **Init**: multi-tenancy → rbac → audit → proxy-policy → metrics
2. **Shutdown**: metrics → proxy-policy → audit → rbac → multi-tenancy

---

## Lifecycle Hooks

### Hook Types

| Hook         | When Called                  | Timeout    | Error Handling        |
|--------------|------------------------------|------------|-----------------------|
| `Init()`     | During application startup   | 30s        | Configurable policy   |
| `OnReady()`  | After all Init() complete    | 10s        | Configurable policy   |
| `OnShutdown()` | During graceful shutdown   | 30s        | Always continue       |

### Failure Policies

Control how the system responds to plugin failures:

#### `continue` (Default)

```yaml
lifecycle:
  failurePolicy: "continue"
```

- Logs error
- Continues with other plugins
- Application starts successfully
- **Use case**: Non-critical plugins, development

#### `halt`

```yaml
lifecycle:
  failurePolicy: "halt"
```

- Logs error
- Stops application startup immediately
- Returns error to operator
- **Use case**: Production, critical plugins

#### `warn`

```yaml
lifecycle:
  failurePolicy: "warn"
```

- Logs warning (not error)
- Continues with other plugins
- Application starts successfully
- **Use case**: Degraded mode, optional features

### Timeout Handling

```yaml
lifecycle:
  initTimeout: 60s      # Per-plugin Init() timeout
  readyTimeout: 30s     # Per-plugin OnReady() timeout
  shutdownTimeout: 60s  # Per-plugin OnShutdown() timeout
```

**What happens on timeout**:

1. Plugin operation is forcefully terminated
2. Error is logged with timeout details
3. Failure policy is applied (continue/halt/warn)
4. Other plugins continue normally

**Example timeout scenario**:

```
INFO  Initializing plugin name=rbac priority=50
ERROR Plugin initialization timed out name=rbac duration_ms=30001 timeout=30s
WARN  Plugin initialization failed (continuing) plugin=rbac error="operation timed out after 30s"
INFO  Initializing plugin name=audit priority=60
```

---

## Event System

The plugin event system enables plugins to react to runtime events without requiring application restart. Plugins can implement the `EventHandler` interface to receive notifications about system state changes.

### Supported Event Types

| Event Type | Description | Data Fields |
|------------|-------------|-------------|
| `package_manager_state_changed` | PM enabled/disabled via toggle API | `package_manager`, `previous_state`, `new_state`, `timestamp` |
| `config_reloaded` | Configuration reloaded from disk | `timestamp`, `config_path` |
| `cache_cleared` | Cache cleared for specific PM | `cache_type`, `timestamp` |

### EventHandler Interface

Plugins opt-in to event notifications by implementing the `EventHandler` interface:

```go
type EventHandler interface {
    OnEvent(ctx context.Context, event Event) error
}
```

**Event Structure**:

```go
type Event struct {
    Type EventType              `json:"type"`
    Data map[string]interface{} `json:"data"`
}
```

### Example Implementation

```go
package myplugin

import (
    "context"
    "proxynd/plugins"
)

type MyPlugin struct {
    name string
}

func (p *MyPlugin) Name() string {
    return p.name
}

func (p *MyPlugin) Init(ctx plugins.Context) error {
    // Initialization logic
    return nil
}

func (p *MyPlugin) OnReady(ctx plugins.Context) error {
    // Ready logic
    return nil
}

func (p *MyPlugin) OnShutdown(ctx context.Context) error {
    // Cleanup logic
    return nil
}

// Implement EventHandler interface
func (p *MyPlugin) OnEvent(ctx context.Context, event plugins.Event) error {
    switch event.Type {
    case plugins.EventPackageManagerStateChanged:
        pm := event.Data["package_manager"].(string)
        newState := event.Data["new_state"].(bool)

        // React to PM state change
        if newState {
            // PM was enabled
            return p.enableCachingForPM(pm)
        } else {
            // PM was disabled
            return p.disableCachingForPM(pm)
        }

    case plugins.EventConfigReloaded:
        // Reload plugin-specific configuration
        return p.reloadConfig()

    case plugins.EventCacheCleared:
        cacheType := event.Data["cache_type"].(string)
        // React to cache clear
        return p.invalidateCacheMetrics(cacheType)

    default:
        // Unknown event type, ignore
        return nil
    }
}
```

### Event Dispatch Behavior

**Timeout Protection**:
- Each event handler has a 5-second timeout
- Prevents slow handlers from blocking system

**Error Handling**:
- Continues to other handlers even if one fails
- Only returns error if ALL handlers fail
- Comprehensive logging for debugging

**Selective Dispatch**:
- Only enabled plugins receive events
- Only plugins implementing `EventHandler` receive events
- Non-implementing plugins are silently skipped

### Event Notification Logs

```
DEBUG Dispatching event to plugins type=package_manager_state_changed data=map[package_manager:npm new_state:true previous_state:false timestamp:2025-11-19T00:00:00Z]
DEBUG Plugin event handler completed plugin=cache-optimizer event=package_manager_state_changed duration_ms=12
ERROR Plugin event handler failed plugin=metrics-collector event=package_manager_state_changed duration_ms=5001 error="context deadline exceeded"
INFO  Event dispatched to plugins type=package_manager_state_changed handlers_notified=5 errors=1
```

### Triggering Events Programmatically

Events are automatically triggered by system operations:

**PM Toggle API** (`POST /api/v1/pm/:name/toggle`):
- Automatically triggers `package_manager_state_changed` event
- Event includes previous and new state

**Config Reload**:
- Automatically triggers `config_reloaded` event
- Event includes configuration path

**Cache Clear** (`DELETE /api/v1/cache/:type`):
- Automatically triggers `cache_cleared` event
- Event includes cache type

### Best Practices

**Keep Event Handlers Fast**:
```go
// Good: Quick processing
func (p *MyPlugin) OnEvent(ctx context.Context, event Event) error {
    // Update in-memory state
    p.state.Update(event.Data)
    return nil
}

// Bad: Slow processing (will timeout)
func (p *MyPlugin) OnEvent(ctx context.Context, event Event) error {
    // This will timeout after 5 seconds
    time.Sleep(10 * time.Second)
    return nil
}
```

**Use Goroutines for Heavy Work**:
```go
func (p *MyPlugin) OnEvent(ctx context.Context, event Event) error {
    // Quick acknowledgment
    eventCopy := event

    // Heavy processing in background
    go func() {
        p.processEvent(eventCopy)
    }()

    return nil
}
```

**Handle Unknown Events Gracefully**:
```go
func (p *MyPlugin) OnEvent(ctx context.Context, event Event) error {
    switch event.Type {
    case plugins.EventPackageManagerStateChanged:
        return p.handlePMChange(event)
    default:
        // Don't error on unknown events (forward compatibility)
        return nil
    }
}
```

**Check Context Cancellation**:
```go
func (p *MyPlugin) OnEvent(ctx context.Context, event Event) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Process event
    return p.doWork(event)
}
```

### Testing Event Handlers

```go
func TestMyPlugin_OnEvent(t *testing.T) {
    plugin := &MyPlugin{name: "test-plugin"}

    event := plugins.Event{
        Type: plugins.EventPackageManagerStateChanged,
        Data: map[string]interface{}{
            "package_manager": "npm",
            "previous_state":  false,
            "new_state":       true,
        },
    }

    ctx := context.Background()
    err := plugin.OnEvent(ctx, event)

    if err != nil {
        t.Errorf("OnEvent failed: %v", err)
    }

    // Assert plugin state changed correctly
}
```

---

## Environment Overrides

### Use Cases

- **Development**: Disable auth, enable debug logging
- **Staging**: Mirror production with test billing
- **Production**: Strict security, full audit logging

### Configuration Pattern

```yaml
plugins:
  # Base configuration (applies to all environments)
  registry:
    enterprise:
      - name: "rbac"
        enabled: true
        priority: 50
        config:
          adminRole: "admin"

  # Environment-specific overrides
  environments:
    development:
      overrides:
        - plugin: "rbac"
          enabled: false  # Override: disable in dev

    staging:
      overrides:
        - plugin: "rbac"
          enabled: true  # Keep enabled
          config:
            adminRole: "staging-admin"  # Override: different role

    production:
      overrides:
        - plugin: "rbac"
          enabled: true  # Keep enabled
          config:
            adminRole: "production-admin"
            strictMode: true  # Add new config key
```

### Selecting Environment

Set `PROXYND_ENV` environment variable:

```bash
# Development (bypasses auth in tests)
export PROXYND_ENV=development

# Staging
export PROXYND_ENV=staging

# Production (default if not set)
export PROXYND_ENV=production
```

**Default**: `development` if not specified.

### Override Precedence

1. **Base configuration** (defined in `registry`)
2. **Environment override** (from `environments.<PROXYND_ENV>.overrides`)

Example:

```yaml
# Base
enterprise:
  - name: "rbac"
    enabled: true
    priority: 50
    config:
      adminRole: "admin"
      auditAll: false

# Environment override (production)
production:
  overrides:
    - plugin: "rbac"
      config:
        auditAll: true  # Override one field
        retentionDays: 90  # Add new field
```

**Resulting config in production**:

```yaml
name: "rbac"
enabled: true          # From base
priority: 50           # From base
config:
  adminRole: "admin"   # From base
  auditAll: true       # Overridden
  retentionDays: 90    # Added
```

---

## Observability

### Structured Logging

All plugin lifecycle events are logged with structured fields:

```json
{
  "level": "info",
  "time": "2025-11-11T22:30:00+09:00",
  "message": "Plugin initialized successfully",
  "plugin": "rbac",
  "priority": 50,
  "duration_ms": 123
}
```

**Key log messages**:

| Message                       | Level | When                          |
|-------------------------------|-------|-------------------------------|
| Discovered plugins            | INFO  | After plugin discovery        |
| Initializing plugin           | INFO  | Before Init() call            |
| Plugin initialized successfully | INFO | After successful Init()       |
| Plugin initialization failed  | ERROR | After failed Init()           |
| Plugin ready                  | INFO  | After successful OnReady()    |
| All plugins ready             | INFO  | After all OnReady() complete  |
| Shutting down plugins         | INFO  | Before shutdown starts        |
| Plugin shutdown complete      | INFO  | After all shutdowns complete  |

### Metrics Collection

**Per-Plugin Metrics** (available via `manager.GetPluginMetrics(name)`):

```json
{
  "plugin_name": "rbac",
  "priority": 50,
  "status": "ready",
  "init_time_ms": 123,
  "ready_time_ms": 45,
  "shutdown_time_ms": 0,
  "error_count": 0,
  "last_error": "",
  "last_error_time": null,
  "state_changes": 3,
  "last_state_change": "2025-11-11T22:30:00Z",
  "enabled": true
}
```

**Aggregated Metrics** (via `manager.GetMetricsSummary()`):

```json
{
  "total_plugins": 5,
  "enabled_plugins": 4,
  "disabled_plugins": 1,
  "ready_plugins": 4,
  "failed_plugins": 0,
  "total_errors": 0,
  "total_state_changes": 15,
  "total_init_time_ms": 456,
  "total_ready_time_ms": 123,
  "total_shutdown_time_ms": 0,
  "average_init_time_ms": 114,
  "average_ready_time_ms": 30
}
```

### Plugin Status Values

| Status        | Description                                  |
|---------------|----------------------------------------------|
| `unknown`     | Initial state (not yet discovered)           |
| `discovered`  | Found during discovery                       |
| `initializing`| Init() in progress                           |
| `initialized` | Init() completed successfully                |
| `ready`       | OnReady() completed successfully             |
| `failed`      | Init() or OnReady() failed                   |
| `shutting_down` | OnShutdown() in progress                   |
| `shutdown`    | OnShutdown() completed                       |

---

## Operations

### Enabling/Disabling Plugins

#### Method 1: Configuration File

Edit `$CONFIG_DIR/plugins.yaml`:

```yaml
enterprise:
  - name: "rbac"
    enabled: false  # Disable this plugin
```

Restart the application:

```bash
systemctl restart proxynd
```

#### Method 2: Environment Override

Create environment-specific override without editing base config:

```yaml
environments:
  maintenance:
    overrides:
      - plugin: "rbac"
        enabled: false
```

Set environment and restart:

```bash
export PROXYND_ENV=maintenance
systemctl restart proxynd
```

### Checking Plugin Status

**View logs** during startup:

```bash
journalctl -u proxynd -n 100 --no-pager | grep -i plugin
```

**Expected output**:

```
INFO  Discovered plugins total=5 enabled=4 environment=production
INFO  Initializing plugin name=rbac priority=50
INFO  Plugin initialized successfully name=rbac duration_ms=123
INFO  Plugin ready name=rbac duration_ms=45
INFO  All plugins ready total=4 ready=4 failed=0 avg_init_ms=120 avg_ready_ms=40
```

**Check for errors**:

```bash
journalctl -u proxynd -p err --since "10 minutes ago" | grep -i plugin
```

### Debugging Plugin Issues

#### Plugin Not Loading

**Symptom**: Plugin not listed in "Discovered plugins" log

**Possible causes**:

1. Plugin not registered in code
2. Wrong build tags (`-tags enterprise` missing)
3. Plugin name mismatch

**Solution**:

```bash
# Check binary build tags
strings /usr/local/bin/proxynd | grep -i "buildTags"

# Verify plugin registration
# In code: plugins.Register("rbac", &RBACPlugin{})
```

#### Plugin Initialization Failing

**Symptom**: "Plugin initialization failed" error in logs

**Debug steps**:

1. **Check error message**:

```bash
journalctl -u proxynd | grep "initialization failed"
```

2. **Review plugin-specific logs**:

```bash
journalctl -u proxynd | grep "plugin=rbac"
```

3. **Verify configuration**:

```bash
cat $CONFIG_DIR/plugins.yaml
```

4. **Check dependencies** (database, external services):

```bash
# Example: Check if database is accessible
psql -h localhost -U proxynd -c "SELECT 1"
```

5. **Test with relaxed failure policy**:

```yaml
lifecycle:
  failurePolicy: "warn"  # Allow startup despite failures
```

#### Plugin Timeout

**Symptom**: "Plugin initialization timed out" in logs

**Solution**:

Increase timeout in config:

```yaml
lifecycle:
  initTimeout: 120s  # Increase from 30s to 120s
```

Or investigate why plugin is slow:

```bash
# Check external API latency
time curl -I https://external-api.example.com/health
```

### Graceful Degradation

Run without problematic plugin:

```yaml
environments:
  degraded:
    overrides:
      - plugin: "problematic-plugin"
        enabled: false
```

```bash
export PROXYND_ENV=degraded
systemctl restart proxynd
```

### Rolling Updates

When updating plugin configuration:

1. **Update config file**:

```bash
vim $CONFIG_DIR/plugins.yaml
```

2. **Validate syntax** (optional):

```bash
yamllint $CONFIG_DIR/plugins.yaml
```

3. **Graceful restart**:

```bash
systemctl reload-or-restart proxynd
```

4. **Verify startup**:

```bash
journalctl -u proxynd -f | grep -i plugin
```

5. **Check readiness**:

```bash
curl http://localhost:8080/health
```

---

## Troubleshooting

### Common Issues

#### 1. All Plugins Disabled

**Symptom**: No plugins initialized

**Logs**:

```
INFO  Plugin system disabled
```

**Cause**: `plugins.enabled: false` in config

**Solution**:

```yaml
plugins:
  enabled: true
```

#### 2. Wrong Plugin Priority

**Symptom**: Plugin initializing in wrong order, causing dependency errors

**Example**:

```
ERROR audit plugin failed: rbac not initialized yet
```

**Cause**: audit plugin (priority 60) depends on rbac (priority 50), but priorities are inverted

**Solution**: Lower priority number = earlier initialization

```yaml
enterprise:
  - name: "rbac"
    priority: 50  # Lower = earlier

  - name: "audit"
    priority: 60  # Higher = later (can depend on rbac)
```

#### 3. Environment Override Not Applied

**Symptom**: Expected override not taking effect

**Debug**:

1. Check `PROXYND_ENV` is set:

```bash
echo $PROXYND_ENV
```

2. Check override section name matches:

```yaml
environments:
  production:  # Must match PROXYND_ENV value exactly
    overrides:
      # ...
```

3. Check plugin name in override:

```yaml
overrides:
  - plugin: "rbac"  # Must match exact plugin registration name
```

4. Verify logs:

```bash
journalctl -u proxynd | grep "Applied environment override"
```

#### 4. Configuration File Not Found

**Symptom**: "Failed to load plugin config, using defaults" warning

**Cause**: `$CONFIG_DIR/plugins.yaml` doesn't exist

**Solution**: Either:

- Create the file (system will use defaults if missing)
- Or ignore warning (defaults are safe)

---

## Best Practices

### 1. Start with Defaults

Don't create `plugins.yaml` initially. Let the system use defaults and only customize when needed.

### 2. Use Environment Overrides

Don't duplicate entire config for each environment:

**Bad**:

```
config/
  plugins.dev.yaml      # Full config duplicated
  plugins.staging.yaml  # Full config duplicated
  plugins.prod.yaml     # Full config duplicated
```

**Good**:

```
config/
  plugins.yaml  # Single file with environment overrides
```

### 3. Assign Priorities Thoughtfully

Leave gaps between priorities for future plugins:

```yaml
- name: "plugin-a"
  priority: 50

- name: "plugin-b"
  priority: 100  # Gap of 50 allows inserting plugins at 60, 70, etc.
```

### 4. Use Halt Policy in Production

Catch plugin failures early:

```yaml
# Development
lifecycle:
  failurePolicy: "continue"  # Permissive

# Production
lifecycle:
  failurePolicy: "halt"      # Strict
```

### 5. Monitor Plugin Metrics

Set up alerts for:

- Failed plugins (`failed_plugins > 0`)
- High error counts (`total_errors > 10`)
- Slow initialization (`average_init_time_ms > 5000`)

### 6. Test Configuration Changes

Use staging environment first:

```bash
# Staging
export PROXYND_ENV=staging
./proxynd --config /etc/proxynd/config

# If successful, deploy to production
export PROXYND_ENV=production
```

### 7. Document Custom Priorities

Add comments explaining priority choices:

```yaml
enterprise:
  - name: "rbac"
    priority: 50  # Must init before audit (60)

  - name: "audit"
    priority: 60  # Depends on rbac for user identification
```

---

## Security Considerations

### 1. Protect Configuration Files

```bash
chmod 600 $CONFIG_DIR/plugins.yaml
chown proxynd:proxynd $CONFIG_DIR/plugins.yaml
```

### 2. Validate Plugin Sources

Only load plugins from trusted sources. The plugin system does not sandbox plugins—they have full application access.

### 3. Review Plugin Permissions

Plugins can:

- Read/write files
- Make network requests
- Access databases
- Modify application state

Audit plugin code before enabling in production.

### 4. Use Environment Isolation

```yaml
environments:
  production:
    overrides:
      - plugin: "debug-tools"
        enabled: false  # Never enable debug plugins in prod
```

---

## Migration Guide

### Migrating from Old Plugin System

**Old way** (hardcoded in code):

```go
plugins.Initialize(app, ctx)
```

**New way** (config-driven):

1. Create `plugins.yaml`:

```yaml
plugins:
  enabled: true
  registry:
    core:
      - name: "existing-plugin"
        enabled: true
        priority: 100
```

2. Application automatically loads config

3. Plugins initialize with lifecycle hooks

**Backward compatibility**: Old `plugins.Initialize()` still works but is deprecated.

---

## Example Configurations

### Development Environment

```yaml
plugins:
  enabled: true

  lifecycle:
    failurePolicy: "warn"  # Relaxed

  environments:
    development:
      overrides:
        - plugin: "rbac"
          enabled: false  # Bypass auth

        - plugin: "audit"
          config:
            logLevel: "debug"
```

### Staging Environment

```yaml
plugins:
  enabled: true

  lifecycle:
    failurePolicy: "continue"  # Permissive

  environments:
    staging:
      overrides:
        - plugin: "billing"
          config:
            testMode: true  # Use test billing API
```

### Production Environment

```yaml
plugins:
  enabled: true

  lifecycle:
    initTimeout: 60s
    readyTimeout: 30s
    shutdownTimeout: 60s
    failurePolicy: "halt"  # Strict

  registry:
    enterprise:
      - name: "rbac"
        enabled: true
        priority: 50
        config:
          strictMode: true
          auditAll: true

      - name: "audit"
        enabled: true
        priority: 60
        config:
          retentionDays: 180
          logLevel: "warn"

    cloud:
      - name: "multi-tenancy"
        enabled: true
        priority: 10
        config:
          isolationLevel: "strict"

  environments:
    production:
      overrides:
        - plugin: "debug-tools"
          enabled: false

        - plugin: "profiler"
          enabled: false
```

---

## API Reference

### Manager Methods

**Available in code** (for advanced integrations):

```go
// Get plugin manager instance
manager := app.GetPluginManager()

// Query plugin status
plugins := manager.GetEnabledPlugins()
for _, p := range plugins {
    fmt.Printf("Plugin: %s, Priority: %d\n", p.Name, p.Priority)
}

// Get metrics for specific plugin
metrics := manager.GetPluginMetrics("rbac")
fmt.Printf("Status: %s, Init time: %d ms\n", metrics.Status, metrics.InitTime.Milliseconds())

// Get aggregated metrics
summary := manager.GetMetricsSummary()
fmt.Printf("Total: %d, Ready: %d, Failed: %d\n",
    summary.TotalPlugins, summary.ReadyPlugins, summary.FailedPlugins)
```

---

## Support

### Getting Help

- **Documentation**: `/docs/PLUGIN_OPERATOR_GUIDE.md` (this file)
- **Implementation**: `plugins/manager.go`, `plugins/config.go`
- **Examples**: `examples/plugins/*.yaml`

### Reporting Issues

Include in issue reports:

1. Plugin configuration (`plugins.yaml`)
2. Application logs (with plugin keyword)
3. Plugin versions and build tags
4. Environment (`PROXYND_ENV`)

**Example**:

```bash
# Collect diagnostic info
echo "=== Plugin Configuration ===" > plugin-debug.txt
cat $CONFIG_DIR/plugins.yaml >> plugin-debug.txt
echo "=== Environment ===" >> plugin-debug.txt
env | grep PROXYND >> plugin-debug.txt
echo "=== Recent Logs ===" >> plugin-debug.txt
journalctl -u proxynd --since "1 hour ago" | grep -i plugin >> plugin-debug.txt
```

---

## Changelog

### Version 1.0 (2025-11-11)

- Initial release
- Config-driven plugin management
- Priority-based initialization
- Lifecycle hooks (Init, Ready, Shutdown)
- Environment overrides
- Failure policies
- Comprehensive metrics and logging
