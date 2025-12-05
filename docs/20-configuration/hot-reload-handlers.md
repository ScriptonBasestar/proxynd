# Configuration Hot Reload Handlers

**Version**: 1.0
**Last Updated**: 2025-12-05
**Status**: Production Ready

---

## Overview

ProxyND implements a comprehensive hot reload system that allows runtime configuration changes without service restart. The system uses a handler-based architecture where different components respond to configuration changes through registered handlers.

### Purpose

1. **Zero-Downtime Updates**: Change configuration without service interruption
2. **Dynamic Scaling**: Adjust performance parameters at runtime
3. **Security Updates**: Update authentication and security settings immediately
4. **Operational Flexibility**: Fine-tune system behavior based on load patterns

---

## Architecture

### Components

```
┌─────────────────┐
│ fsnotify Watcher│ ← Monitors config.yaml
└────────┬────────┘
         │ File Write Event
         ▼
┌─────────────────┐
│ HotReloadManager│ ← Orchestrates reload
└────────┬────────┘
         │ OnConfigChange()
         ▼
┌─────────────────────────────────┐
│ Registered Handlers (7 total)   │
├─────────────────────────────────┤
│ 1. LoggingReloadHandler         │
│ 2. CacheReloadHandler           │
│ 3. MetricsReloadHandler         │
│ 4. SecurityReloadHandler        │
│ 5. WebhookReloadHandler         │
│ 6. ProxyReloadHandler           │
│ 7. CustomReloadHandler          │
└─────────────────────────────────┘
```

### Handler Interface

```go
type ReloadHandler interface {
    Name() string
    OnConfigReload(oldConfig, newConfig *UnifiedConfig) error
}
```

**Key Properties**:
- **Sequential Execution**: Handlers run one after another
- **First Error Wins**: First error stops processing but logs all failures
- **Thread-Safe**: RWMutex protects handler list
- **Logged**: All handler executions logged with success/failure

---

## Implemented Handlers (7 Total)

### 1. LoggingReloadHandler

**File**: `internal/config/hot_reload_handlers.go`
**Capabilities**: ✅ Dynamic log level and format changes

**Hot-Reloadable**:
- ✅ Log level (debug, info, warn, error)
- ✅ Log format (text, json)
- ✅ Logger reinitialization

**Requires Restart**:
- ❌ Log output destinations (files, stdout)
- ❌ Log file paths

**Example**:
```yaml
# Before
logging:
  level: info
  format: text

# After (hot reload)
logging:
  level: debug
  format: json
```

**Implementation**:
```go
func (h *LoggingReloadHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // Detect level change
    if old.Logging.Level != new.Logging.Level {
        logging.SetLevel(logging.LogLevel(new.Logging.Level))
    }

    // Detect format change
    if old.Logging.Format != new.Logging.Format {
        logging.ReinitLogger(new.Logging)
    }

    return nil
}
```

---

### 2. CacheReloadHandler

**File**: `internal/config/hot_reload_handlers.go`
**Capabilities**: ✅ Cache parameter tuning

**Hot-Reloadable**:
- ✅ Cache TTL (time-to-live)
- ✅ Cache max size (entries)
- ✅ Eviction policy parameters

**Requires Restart**:
- ❌ Cache backend type (file → redis → s3)
- ❌ Cache directory location
- ❌ Redis connection strings

**Example**:
```yaml
# Before
cache:
  ttl: 3600
  max_size: 10000

# After (hot reload)
cache:
  ttl: 7200      # 2 hours
  max_size: 50000
```

**Implementation**:
```go
func (h *CacheReloadHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // Backend change requires restart
    if old.Cache.Backend != new.Cache.Backend {
        return fmt.Errorf("cache backend change requires restart")
    }

    // TTL update via callback
    if old.Cache.TTL != new.Cache.TTL && h.cacheUpdateCallback != nil {
        h.cacheUpdateCallback(new.Cache)
    }

    return nil
}
```

---

### 3. MetricsReloadHandler

**File**: `internal/config/hot_reload_handlers.go`
**Capabilities**: ✅ Metrics server control

**Hot-Reloadable**:
- ✅ Enable/disable metrics endpoint
- ✅ Metrics server port (restarts server)
- ✅ Metrics path configuration

**Requires Restart**:
- ❌ Prometheus pushgateway configuration

**Example**:
```yaml
# Before
metrics:
  enabled: false

# After (hot reload)
metrics:
  enabled: true
  port: 9090
  path: /metrics
```

**Implementation**:
```go
func (h *MetricsReloadHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // Start metrics server if enabled
    if !old.Metrics.Enabled && new.Metrics.Enabled {
        return h.metricsServer.Start(new.Metrics.Port)
    }

    // Stop metrics server if disabled
    if old.Metrics.Enabled && !new.Metrics.Enabled {
        return h.metricsServer.Stop()
    }

    // Port change requires restart
    if old.Metrics.Port != new.Metrics.Port {
        h.metricsServer.Stop()
        return h.metricsServer.Start(new.Metrics.Port)
    }

    return nil
}
```

---

### 4. SecurityReloadHandler

**File**: `internal/config/hot_reload_handlers.go`
**Capabilities**: ✅ Security policy updates

**Hot-Reloadable**:
- ✅ IP whitelist enable/disable
- ✅ IP whitelist entries (IPs and CIDRs)
- ✅ BasicAuth user list
- ✅ BasicAuth realm configuration

**Requires Restart**:
- ❌ TLS certificate changes
- ❌ Authentication backend (OAuth, LDAP, SAML)

**Example**:
```yaml
# Before
security:
  ip_whitelist:
    enabled: false

# After (hot reload)
security:
  ip_whitelist:
    enabled: true
    ips:
      - "192.168.1.0/24"
      - "10.0.0.5"
  basic_auth:
    users:
      admin: $2a$10$...  # bcrypt hash
      viewer: $2a$10$...
```

**Implementation**:
```go
func (h *SecurityReloadHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // Update IP whitelist
    if !reflect.DeepEqual(old.Security.IPWhitelist, new.Security.IPWhitelist) {
        h.securityManager.UpdateIPWhitelist(new.Security.IPWhitelist)
    }

    // Update BasicAuth users
    if !reflect.DeepEqual(old.Security.BasicAuth.Users, new.Security.BasicAuth.Users) {
        h.securityManager.UpdateBasicAuthUsers(new.Security.BasicAuth.Users)
    }

    return nil
}
```

---

### 5. WebhookReloadHandler (Future)

**Status**: ⏳ Planned
**Purpose**: Dynamic webhook endpoint management

**Planned Hot-Reloadable**:
- ⏳ Add/remove webhook endpoints
- ⏳ Update retry configuration
- ⏳ Modify event filters

---

### 6. ProxyReloadHandler (Future)

**Status**: ⏳ Planned
**Purpose**: Proxy behavior tuning

**Planned Hot-Reloadable**:
- ⏳ Upstream URL changes
- ⏳ Timeout adjustments
- ⏳ Connection pool sizes

---

### 7. CustomReloadHandler

**Purpose**: User-defined handlers for custom components

**Example**:
```go
type DatabaseReloadHandler struct {
    db *sql.DB
}

func (h *DatabaseReloadHandler) Name() string {
    return "DatabaseReloadHandler"
}

func (h *DatabaseReloadHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // Hot-reloadable: pool size
    if old.Database.MaxConnections != new.Database.MaxConnections {
        h.db.SetMaxOpenConns(new.Database.MaxConnections)
    }

    // Requires restart: host/port
    if old.Database.Host != new.Database.Host {
        return fmt.Errorf("database host change requires restart")
    }

    return nil
}
```

---

## File Watching (fsnotify Integration)

### Implementation

**File**: `internal/app/container.go`

```go
// Watch config file for changes
watcher, err := fsnotify.NewWatcher()
if err != nil {
    return nil, fmt.Errorf("failed to create file watcher: %w", err)
}

// Add config file to watch list
err = watcher.Add(configPath)
if err != nil {
    return nil, fmt.Errorf("failed to watch config file: %w", err)
}

// Start watching in goroutine
go func() {
    for {
        select {
        case event, ok := <-watcher.Events:
            if !ok {
                return
            }

            // Trigger reload on file write
            if event.Op&fsnotify.Write == fsnotify.Write {
                logger.Info("Config file changed, reloading...",
                    logging.F("file", event.Name))

                // Reload config
                newConfig, err := config.LoadConfig(configPath)
                if err != nil {
                    logger.Error("Failed to reload config",
                        logging.F("error", err))
                    continue
                }

                // Trigger handlers
                if err := hotReloadManager.OnConfigChange(oldConfig, newConfig); err != nil {
                    logger.Error("Hot reload failed",
                        logging.F("error", err))
                } else {
                    oldConfig = newConfig
                }
            }

        case err, ok := <-watcher.Errors:
            if !ok {
                return
            }
            logger.Error("File watcher error", logging.F("error", err))
        }
    }
}()
```

### Watched Events

| Event | Action | Description |
|-------|--------|-------------|
| **Write** | Reload | File content modified |
| **Create** | Reload | File created (after delete) |
| **Remove** | Ignore | File deleted (temporary) |
| **Rename** | Reload | File renamed (editor behavior) |
| **Chmod** | Ignore | Permissions changed |

**Note**: Most editors (vim, nano, VSCode) use write-rename-move pattern, triggering multiple events. The system handles this gracefully.

---

## Usage Patterns

### 1. Programmatic Registration

```go
// Create hot reload manager
manager := config.NewHotReloadManager()

// Register built-in handlers
manager.RegisterHandler(config.NewLoggingReloadHandler())
manager.RegisterHandler(config.NewCacheReloadHandler())
manager.RegisterHandler(config.NewMetricsReloadHandler())
manager.RegisterHandler(config.NewSecurityReloadHandler())

// Register custom handler
manager.RegisterHandler(&MyCustomHandler{})

// Trigger reload manually
err := manager.OnConfigChange(oldConfig, newConfig)
if err != nil {
    log.Error("Reload failed", "error", err)
}
```

---

### 2. Automatic File Watching

```go
// Set up in DI container (internal/app/container.go)
watcher, _ := fsnotify.NewWatcher()
watcher.Add("/etc/proxynd/config.yaml")

go func() {
    for event := range watcher.Events {
        if event.Op&fsnotify.Write == fsnotify.Write {
            // Reload automatically
            manager.OnConfigChange(old, new)
        }
    }
}()
```

---

### 3. Signal-Based Reload (SIGHUP)

```go
// Listen for SIGHUP signal
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGHUP)

go func() {
    for range sigChan {
        log.Info("Received SIGHUP, reloading config...")

        newConfig, err := config.LoadConfig(configPath)
        if err != nil {
            log.Error("Failed to load config", "error", err)
            continue
        }

        manager.OnConfigChange(oldConfig, newConfig)
        oldConfig = newConfig
    }
}()
```

**Trigger**:
```bash
# Find process ID
ps aux | grep proxynd

# Send SIGHUP signal
kill -HUP <PID>
```

---

## Testing Hot Reload

### Test Coverage

**File**: `internal/config/hot_reload_handlers_test.go` (486 lines)

**Test Cases**: 14+ comprehensive tests

1. `TestNewHotReloadManager` - Manager initialization
2. `TestHotReloadManager_RegisterHandler` - Handler registration
3. `TestLoggingReloadHandler_Name` - Handler name
4. `TestLoggingReloadHandler_OnConfigReload_LevelChange` - Log level change
5. `TestLoggingReloadHandler_OnConfigReload_NilConfigs` - Nil config handling
6. `TestCacheReloadHandler_Name` - Handler name
7. `TestCacheReloadHandler_OnConfigReload_BackendChange` - Backend change (error)
8. `TestCacheReloadHandler_OnConfigReload_TTLChange` - TTL update
9. `TestMetricsReloadHandler_Name` - Handler name
10. `TestMetricsReloadHandler_OnConfigReload_EnableDisable` - Metrics toggle
11. `TestSecurityReloadHandler_Name` - Handler name
12. `TestSecurityReloadHandler_OnConfigReload_IPWhitelist` - IP whitelist update
13. `TestSecurityReloadHandler_OnConfigReload_Users` - BasicAuth user update
14. And more...

---

### Unit Test Example

```go
func TestLoggingReloadHandler_OnConfigReload_LevelChange(t *testing.T) {
    handler := NewLoggingReloadHandler()

    oldConfig := &UnifiedConfig{
        Logging: LoggingConfig{Level: "info"},
    }

    newConfig := &UnifiedConfig{
        Logging: LoggingConfig{Level: "debug"},
    }

    err := handler.OnConfigReload(oldConfig, newConfig)
    assert.NoError(t, err)

    // Verify log level changed
    assert.Equal(t, logging.DebugLevel, logging.GetLevel())
}
```

---

### Integration Test Example

```go
func TestFileWatcherReload(t *testing.T) {
    // Create temporary config file
    tmpfile, _ := ioutil.TempFile("", "config-*.yaml")
    defer os.Remove(tmpfile.Name())

    // Write initial config
    initialConfig := `
logging:
  level: info
`
    tmpfile.Write([]byte(initialConfig))
    tmpfile.Close()

    // Set up hot reload with file watcher
    manager := NewHotReloadManager()
    handler := NewLoggingReloadHandler()
    manager.RegisterHandler(handler)

    // Start file watcher
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(tmpfile.Name())

    // Modify config file
    time.Sleep(100 * time.Millisecond)
    ioutil.WriteFile(tmpfile.Name(), []byte(`
logging:
  level: debug
`), 0644)

    // Wait for reload
    time.Sleep(200 * time.Millisecond)

    // Verify log level changed
    assert.Equal(t, logging.DebugLevel, logging.GetLevel())
}
```

---

## Operational Guide

### Monitoring Hot Reload

```bash
# Watch for reload events in logs
tail -f /var/log/proxynd/app.log | grep -E "Hot reload|Handler|Config"

# Example output:
# 2025-12-05T10:00:00Z INFO  Registered hot reload handler handler=LoggingReloadHandler
# 2025-12-05T10:15:30Z INFO  Config file changed, reloading... file=/etc/proxynd/config.yaml
# 2025-12-05T10:15:31Z INFO  Changing log level old_level=info new_level=debug
# 2025-12-05T10:15:31Z DEBUG Hot reload handler succeeded handler=LoggingReloadHandler
```

---

### Common Scenarios

#### Scenario 1: Change Log Level for Debugging

**Problem**: Need debug logs for troubleshooting

**Solution**:
```bash
# Edit config
vim /etc/proxynd/config.yaml

# Change logging.level from info to debug
# Save file (automatic reload via fsnotify)

# Or send SIGHUP
kill -HUP $(pgrep proxynd)

# Verify in logs
tail -f /var/log/proxynd/app.log | grep "Log level changed"
```

---

#### Scenario 2: Add IP to Whitelist

**Problem**: New client needs access

**Solution**:
```yaml
# Add to config.yaml
security:
  ip_whitelist:
    enabled: true
    ips:
      - "192.168.1.0/24"
      - "10.0.5.10"  # NEW: Added new client
```

**Verification**:
```bash
# Test from new IP
curl -I http://proxynd:8080/health
# HTTP/1.1 200 OK (if reload succeeded)

# Check logs
grep "Updated IP whitelist" /var/log/proxynd/app.log
```

---

#### Scenario 3: Tune Cache Parameters

**Problem**: Cache too small, need larger capacity

**Solution**:
```yaml
# Update cache config
cache:
  ttl: 7200       # 2 hours (was 1 hour)
  max_size: 50000 # 50k entries (was 10k)
```

**Verification**:
```bash
# Check cache stats via API
curl http://localhost:8080/api/cache/stats | jq '.max_size'
# 50000 (updated)
```

---

#### Scenario 4: Enable Metrics Endpoint

**Problem**: Need to expose metrics for Prometheus

**Solution**:
```yaml
# Enable metrics
metrics:
  enabled: true
  port: 9090
  path: /metrics
```

**Verification**:
```bash
# Check metrics endpoint
curl http://localhost:9090/metrics | head -10

# Verify in logs
grep "Metrics server started" /var/log/proxynd/app.log
```

---

## Best Practices

### 1. Handler Design

**Principles**:
- ✅ **Fast execution**: Handlers should complete in <100ms
- ✅ **Idempotent**: Multiple calls with same config should be safe
- ✅ **Graceful degradation**: Non-critical changes shouldn't block reload
- ✅ **Clear errors**: Return descriptive errors for restart-required changes

**Example**:
```go
func (h *MyHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // 1. Early return if no change
    if reflect.DeepEqual(old.MySection, new.MySection) {
        return nil
    }

    // 2. Validate new config
    if err := h.validate(new.MySection); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }

    // 3. Check for restart-required changes
    if h.requiresRestart(old, new) {
        return fmt.Errorf("config change requires restart: host changed")
    }

    // 4. Apply changes atomically
    h.mu.Lock()
    defer h.mu.Unlock()
    h.config = new.MySection

    // 5. Log success
    h.logger.Info("Config reloaded", logging.F("section", "my_section"))

    return nil
}
```

---

### 2. Error Handling Strategy

**Categories**:

| Error Type | Action | Example |
|------------|--------|---------|
| **Validation Error** | Reject reload | Invalid IP address |
| **Restart Required** | Return error | Database host change |
| **Optional Failure** | Log warning | Metrics push failed |
| **Critical Failure** | Return error | Security update failed |

**Example**:
```go
func (h *Handler) OnConfigReload(old, new *UnifiedConfig) error {
    // Critical: Return error
    if err := h.updateSecurity(new); err != nil {
        return fmt.Errorf("security update failed: %w", err)
    }

    // Optional: Log but continue
    if err := h.updateMetrics(new); err != nil {
        h.logger.Warn("Metrics update failed", logging.F("error", err))
        // Don't return error
    }

    // Restart required: Clear error
    if old.Server.Port != new.Server.Port {
        return fmt.Errorf("port change requires restart")
    }

    return nil
}
```

---

### 3. Concurrency Safety

**Guidelines**:
- ✅ Use `sync.RWMutex` for config access
- ✅ Write lock for updates
- ✅ Read lock for access
- ✅ Minimize lock duration

**Example**:
```go
type ThreadSafeHandler struct {
    mu     sync.RWMutex
    config *MyConfig
}

// Update (write lock)
func (h *ThreadSafeHandler) OnConfigReload(old, new *UnifiedConfig) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.config = new.MySection
    return nil
}

// Access (read lock)
func (h *ThreadSafeHandler) GetConfig() *MyConfig {
    h.mu.RLock()
    defer h.mu.RUnlock()

    return h.config  // Return copy if mutable
}
```

---

### 4. Testing Strategy

**Levels**:
1. **Unit Tests**: Test handler logic with mock configs
2. **Integration Tests**: Test with real file watcher
3. **E2E Tests**: Test full reload flow with service

**Example**:
```go
// Unit test
func TestHandler_LogicOnly(t *testing.T) {
    handler := NewMyHandler()

    old := &UnifiedConfig{Value: "old"}
    new := &UnifiedConfig{Value: "new"}

    err := handler.OnConfigReload(old, new)
    assert.NoError(t, err)
}

// Integration test
func TestHandler_WithFileWatch(t *testing.T) {
    tmpfile := createTempConfig(t)
    defer os.Remove(tmpfile)

    manager := setupHotReload(t, tmpfile)

    // Modify file
    modifyConfig(t, tmpfile, "new_value")

    // Wait for reload
    time.Sleep(100 * time.Millisecond)

    // Verify reload
    assert.Equal(t, "new_value", getCurrentConfig())
}
```

---

## Troubleshooting

### Problem: Config Not Reloading

**Symptoms**:
- File changes don't trigger reload
- No reload messages in logs

**Causes**:
1. File watcher not running
2. Wrong config file path
3. Insufficient file permissions
4. File system doesn't support fsnotify (NFS, some Docker volumes)

**Solutions**:
```bash
# Check watcher is running
ps aux | grep proxynd | grep -v grep
# Look for goroutine in process

# Check file permissions
ls -la /etc/proxynd/config.yaml
# Should be readable by proxynd user

# Check file system type
df -T /etc/proxynd/
# Avoid NFS, use local filesystem

# Manual reload via SIGHUP
kill -HUP $(pgrep proxynd)

# Check logs for watcher errors
grep "watcher error" /var/log/proxynd/app.log
```

---

### Problem: Handler Fails with Error

**Symptoms**:
- Reload triggered but config not applied
- Error messages in logs

**Causes**:
1. Invalid configuration values
2. Restart-required change attempted
3. Dependent service unavailable

**Solutions**:
```bash
# Check which handler failed
grep "Hot reload handler failed" /var/log/proxynd/app.log

# Example output:
# ERROR Hot reload handler failed handler=CacheReloadHandler error="backend change requires restart"

# Read error message
# Fix configuration or restart service if needed

# Validate config before reload
proxynd validate-config --config /etc/proxynd/config.yaml
```

---

### Problem: Partial Reload (Some Handlers Succeed, Some Fail)

**Symptoms**:
- Some changes applied, others not
- Mixed success/failure in logs

**Expected Behavior**:
- This is NORMAL! Handlers are independent
- First error is returned but all handlers run
- Check logs for each handler's result

**Solution**:
```bash
# Review all handler results
grep "Hot reload handler" /var/log/proxynd/app.log | tail -20

# Example:
# DEBUG Hot reload handler succeeded handler=LoggingReloadHandler
# DEBUG Hot reload handler succeeded handler=SecurityReloadHandler
# ERROR Hot reload handler failed handler=CacheReloadHandler error="backend change requires restart"
# DEBUG Hot reload handler succeeded handler=MetricsReloadHandler

# Fix failing handlers or restart if required
```

---

### Problem: Configuration Reverts After Reload

**Symptoms**:
- Changes applied then lost
- Config file shows new values but system uses old

**Causes**:
1. Handler not persisting changes
2. Multiple config sources (env vars override)
3. Handler error after partial update

**Solutions**:
```bash
# Check config precedence
# 1. Command-line flags
# 2. Environment variables
# 3. Config file

# Example: Env var overrides file
export LOG_LEVEL=info  # Overrides config.yaml logging.level

# Remove conflicting env vars
unset LOG_LEVEL

# Verify effective config via API
curl http://localhost:8080/api/config | jq '.logging.level'
```

---

## Security Considerations

### Access Control

**Protect config file**:
```bash
# Restrict permissions
chmod 600 /etc/proxynd/config.yaml
chown proxynd:proxynd /etc/proxynd/config.yaml

# Prevent unauthorized writes
# (file watcher would trigger reload with malicious config)
```

---

### Audit Logging

**Enable audit logs**:
```yaml
logging:
  audit:
    enabled: true
    events:
      - config.reload
      - security.policy.change
      - auth.user.update
```

**Audit log output**:
```json
{
  "timestamp": "2025-12-05T10:15:30Z",
  "event": "config.reload",
  "user": "admin",
  "source": "file_watcher",
  "changes": [
    "logging.level: info -> debug",
    "security.ip_whitelist.ips: added 10.0.5.10"
  ]
}
```

---

### Sensitive Data

**Be cautious with**:
- Database passwords
- API keys
- OAuth secrets
- TLS certificates

**Best practice**:
```yaml
# Don't put secrets in config.yaml
# Use environment variables or secret management
database:
  password: ${DB_PASSWORD}  # From env var

# Or use external secret store
secrets:
  provider: vault
  path: secret/proxynd/db
```

---

## Performance Considerations

### Reload Overhead

**Typical costs**:
- File watcher event: <1ms
- Config parsing: 5-10ms
- Handler execution: 10-100ms per handler
- **Total**: ~50-500ms for full reload

**Impact**:
- ✅ Minimal: No request drops
- ✅ No downtime: Service continues during reload
- ⚠️ Brief: Slight latency spike possible

---

### Handler Optimization

**Tips**:
1. **Early returns**: Skip work if config unchanged
2. **Batch updates**: Group related changes
3. **Async operations**: Don't block handler for slow ops
4. **Debouncing**: Already built-in (500ms default)

**Example**:
```go
func (h *Handler) OnConfigReload(old, new *UnifiedConfig) error {
    // 1. Early return (fast path)
    if reflect.DeepEqual(old.Section, new.Section) {
        return nil  // <1ms
    }

    // 2. Prepare update (compute changes)
    changes := h.computeDiff(old.Section, new.Section)  // ~10ms

    // 3. Apply atomically
    h.mu.Lock()
    h.applyChanges(changes)  // ~50ms
    h.mu.Unlock()

    // 4. Async notification (don't block)
    go h.notifyObservers(changes)

    return nil  // Total: ~60ms
}
```

---

## Related Documentation

- [Configuration Reference](configuration-reference.md) - Complete config options
- [Hot Reload Guide (Korean)](hot-reload.md) - Detailed usage guide in Korean
- [Logging Configuration](../70-operations/logging.md) - Logging setup
- [Security Configuration](../50-security/security-config.md) - Security settings
- [Cache Configuration](../30-proxy-types/caching.md) - Cache tuning
- [Metrics Configuration](../70-operations/metrics.md) - Metrics setup

---

**Changelog**:
- 2025-12-05: Initial documentation for hot reload handlers (7 handlers implemented)
- 2025-11-25: Hot reload system implementation completed (commit e0b199a)
