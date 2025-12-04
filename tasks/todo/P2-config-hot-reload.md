# P2: Config Hot Reload Implementation

**Priority**: P2 (Medium)
**Status**: Pending
**Created**: 2025-12-04
**Estimated Time**: 4-5 hours

---

## Overview

Complete configuration hot reload functionality for zero-downtime config updates.
Currently 7 TODOs identified in `configs/hot_reload.go` for various reload handlers.

---

## Current State

### ✅ Complete
- Hot reload infrastructure exists
- File watching capability present
- Basic reload trigger mechanism

### ⏳ Pending (7 TODOs)
1. Logger level/format hot reload
2. Cache TTL updates
3. Metrics server start/stop
4. User reload
5. IP whitelist updates
6. File system watching
7. Multiple log outputs handling

---

## TODOs to Address

### 1. Logger Level and Format Hot Reload
**File**: `configs/hot_reload.go`

**Current Gap**: Logger configuration requires restart

**Implementation**:
```go
func (r *HotReloader) reloadLoggerConfig(newConfig *Config) error {
    // Get current logger
    logger := r.app.GetLogger()

    // Update log level
    if newConfig.Logging.Level != r.currentConfig.Logging.Level {
        if err := logger.SetLevel(newConfig.Logging.Level); err != nil {
            return fmt.Errorf("failed to update log level: %w", err)
        }
        r.logger.Info("Log level updated", "level", newConfig.Logging.Level)
    }

    // Update log format
    if newConfig.Logging.Format != r.currentConfig.Logging.Format {
        if err := logger.SetFormat(newConfig.Logging.Format); err != nil {
            return fmt.Errorf("failed to update log format: %w", err)
        }
        r.logger.Info("Log format updated", "format", newConfig.Logging.Format)
    }

    return nil
}
```

### 2. Cache TTL Updates
**File**: `configs/hot_reload.go`

**Current Gap**: Cache TTL changes require restart

**Implementation**:
```go
func (r *HotReloader) reloadCacheTTL(newConfig *Config) error {
    cacheService := r.app.GetCacheService()

    // Update global cache max age
    if newConfig.CacheMaxAge != r.currentConfig.CacheMaxAge {
        if err := cacheService.SetGlobalMaxAge(newConfig.CacheMaxAge); err != nil {
            return fmt.Errorf("failed to update cache max age: %w", err)
        }
        r.logger.Info("Cache max age updated", "duration", newConfig.CacheMaxAge)
    }

    // Update package manager specific TTLs
    for pm, ttl := range newConfig.PackageManagers {
        oldTTL := r.currentConfig.PackageManagers[pm]
        if ttl.CacheTTL != oldTTL.CacheTTL {
            if err := cacheService.SetTTL(pm, ttl.CacheTTL); err != nil {
                r.logger.Error("Failed to update PM cache TTL",
                    "pm", pm, "error", err)
                continue
            }
            r.logger.Info("PM cache TTL updated", "pm", pm, "ttl", ttl.CacheTTL)
        }
    }

    return nil
}
```

### 3. Metrics Server Start/Stop
**File**: `configs/hot_reload.go`

**Current Gap**: Metrics server changes require restart

**Implementation**:
```go
func (r *HotReloader) reloadMetricsServer(newConfig *Config) error {
    oldEnabled := r.currentConfig.Metrics.Enabled
    newEnabled := newConfig.Metrics.Enabled

    // Metrics disabled -> enabled
    if !oldEnabled && newEnabled {
        if err := r.app.StartMetricsServer(newConfig.Metrics); err != nil {
            return fmt.Errorf("failed to start metrics server: %w", err)
        }
        r.logger.Info("Metrics server started")
    }

    // Metrics enabled -> disabled
    if oldEnabled && !newEnabled {
        if err := r.app.StopMetricsServer(); err != nil {
            return fmt.Errorf("failed to stop metrics server: %w", err)
        }
        r.logger.Info("Metrics server stopped")
    }

    // Metrics enabled -> enabled (config changed)
    if oldEnabled && newEnabled {
        // Check if config changed
        if !reflect.DeepEqual(r.currentConfig.Metrics, newConfig.Metrics) {
            // Restart with new config
            if err := r.app.StopMetricsServer(); err != nil {
                return fmt.Errorf("failed to stop metrics server: %w", err)
            }
            if err := r.app.StartMetricsServer(newConfig.Metrics); err != nil {
                return fmt.Errorf("failed to start metrics server: %w", err)
            }
            r.logger.Info("Metrics server restarted with new config")
        }
    }

    return nil
}
```

### 4. User Reload
**File**: `configs/hot_reload.go`

**Current Gap**: User changes require restart

**Implementation**:
```go
func (r *HotReloader) reloadUsers(newConfig *Config) error {
    authService := r.app.GetAuthService()

    // Reload basic auth users
    if newConfig.Security.Authentication.BasicAuth != nil {
        users := newConfig.Security.Authentication.BasicAuth.Users
        if err := authService.UpdateBasicAuthUsers(users); err != nil {
            return fmt.Errorf("failed to update basic auth users: %w", err)
        }
        r.logger.Info("Basic auth users updated", "count", len(users))
    }

    // Reload API key users
    if newConfig.Security.Authentication.APIKeys != nil {
        keys := newConfig.Security.Authentication.APIKeys.Keys
        if err := authService.UpdateAPIKeys(keys); err != nil {
            return fmt.Errorf("failed to update API keys: %w", err)
        }
        r.logger.Info("API keys updated", "count", len(keys))
    }

    return nil
}
```

### 5. IP Whitelist Updates
**File**: `configs/hot_reload.go`

**Current Gap**: IP whitelist changes require restart

**Implementation**:
```go
func (r *HotReloader) reloadIPWhitelist(newConfig *Config) error {
    securityService := r.app.GetSecurityService()

    // Update IP whitelist
    if !reflect.DeepEqual(newConfig.Security.IPWhitelist,
                          r.currentConfig.Security.IPWhitelist) {
        if err := securityService.UpdateIPWhitelist(newConfig.Security.IPWhitelist); err != nil {
            return fmt.Errorf("failed to update IP whitelist: %w", err)
        }
        r.logger.Info("IP whitelist updated",
            "count", len(newConfig.Security.IPWhitelist))
    }

    // Update IP blacklist
    if !reflect.DeepEqual(newConfig.Security.IPBlacklist,
                          r.currentConfig.Security.IPBlacklist) {
        if err := securityService.UpdateIPBlacklist(newConfig.Security.IPBlacklist); err != nil {
            return fmt.Errorf("failed to update IP blacklist: %w", err)
        }
        r.logger.Info("IP blacklist updated",
            "count", len(newConfig.Security.IPBlacklist))
    }

    return nil
}
```

### 6. File System Watching
**File**: `internal/config/config_loader.go`
**Line**: 349

**Current Gap**: Config file watching not implemented

**Implementation**:
```go
import "github.com/fsnotify/fsnotify"

type ConfigWatcher struct {
    watcher  *fsnotify.Watcher
    reloader *HotReloader
    logger   *logging.Logger
}

func NewConfigWatcher(configPath string, reloader *HotReloader) (*ConfigWatcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, fmt.Errorf("failed to create watcher: %w", err)
    }

    if err := watcher.Add(configPath); err != nil {
        return nil, fmt.Errorf("failed to watch config file: %w", err)
    }

    cw := &ConfigWatcher{
        watcher:  watcher,
        reloader: reloader,
        logger:   logging.NewLogger("config-watcher"),
    }

    go cw.watch()

    return cw, nil
}

func (cw *ConfigWatcher) watch() {
    for {
        select {
        case event, ok := <-cw.watcher.Events:
            if !ok {
                return
            }

            if event.Op&fsnotify.Write == fsnotify.Write {
                cw.logger.Info("Config file changed, reloading...")
                if err := cw.reloader.Reload(); err != nil {
                    cw.logger.Error("Failed to reload config", "error", err)
                }
            }

        case err, ok := <-cw.watcher.Errors:
            if !ok {
                return
            }
            cw.logger.Error("Watcher error", "error", err)
        }
    }
}

func (cw *ConfigWatcher) Close() error {
    return cw.watcher.Close()
}
```

### 7. Multiple Log Outputs
**File**: `internal/config/logging_config.go`
**Line**: 133

**Current Gap**: Only single output supported

**Implementation**:
```go
type LoggingConfig struct {
    Level   string   `yaml:"level"`
    Format  string   `yaml:"format"`
    Outputs []string `yaml:"outputs"` // Changed from single Output to multiple Outputs
}

func (lc *LoggingConfig) GetOutputs() []string {
    if len(lc.Outputs) == 0 {
        return []string{"stdout"} // Default
    }
    return lc.Outputs
}

// In logger setup:
func setupLogger(config *LoggingConfig) (*Logger, error) {
    var writers []io.Writer

    for _, output := range config.GetOutputs() {
        switch output {
        case "stdout":
            writers = append(writers, os.Stdout)
        case "stderr":
            writers = append(writers, os.Stderr)
        default:
            // File output
            file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
            if err != nil {
                return nil, fmt.Errorf("failed to open log file %s: %w", output, err)
            }
            writers = append(writers, file)
        }
    }

    multiWriter := io.MultiWriter(writers...)
    // Use multiWriter for logger
}
```

---

## Implementation Plan

### Phase 1: Core Reload Handlers (2 hours)
1. Implement logger reload handler
2. Implement cache TTL reload handler
3. Implement metrics server reload handler
4. Add comprehensive error handling

### Phase 2: Security Reload Handlers (1 hour)
1. Implement user reload handler
2. Implement IP whitelist reload handler
3. Add validation for security configs

### Phase 3: File Watching (1 hour)
1. Implement config file watcher with fsnotify
2. Add debouncing for rapid file changes
3. Handle watch errors gracefully

### Phase 4: Multiple Outputs (30 min)
1. Update logging config structure
2. Implement multi-writer setup
3. Handle file rotation

### Phase 5: Testing (1 hour)
1. Unit tests for each reload handler
2. Integration tests for hot reload
3. Manual testing with config changes

---

## Testing Strategy

### Unit Tests

```go
func TestHotReload_LoggerConfig(t *testing.T) { /* ... */ }
func TestHotReload_CacheTTL(t *testing.T) { /* ... */ }
func TestHotReload_MetricsServer(t *testing.T) { /* ... */ }
func TestHotReload_Users(t *testing.T) { /* ... */ }
func TestHotReload_IPWhitelist(t *testing.T) { /* ... */ }
```

### Integration Tests

```bash
# Terminal 1: Start server
./bin/proxynd-core

# Terminal 2: Modify config
vim config.yaml  # Change log level to debug

# Verify: Check logs show level change
curl http://localhost:8080/api/config/current

# Test cache TTL change
# Test metrics enable/disable
# Test user updates
# Test IP whitelist updates
```

### Manual Test Checklist
- [ ] Logger level change reflected immediately
- [ ] Logger format change works
- [ ] Cache TTL updates apply to new requests
- [ ] Metrics server can be started/stopped
- [ ] User changes take effect
- [ ] IP whitelist updates block/allow correctly
- [ ] File watcher triggers reload
- [ ] Multiple log outputs write correctly
- [ ] No service interruption during reload
- [ ] Error handling works correctly

---

## Configuration Example

```yaml
# config.yaml
logging:
  level: info
  format: json
  outputs:
    - stdout
    - /var/log/proxynd/app.log

cache:
  max_age: 24h
  package_managers:
    maven:
      cache_ttl: 1h
    npm:
      cache_ttl: 30m

metrics:
  enabled: true
  port: 9090
  path: /metrics

security:
  authentication:
    basic_auth:
      users:
        admin: $2a$10$...
    api_keys:
      keys:
        - key: api_key_1
          user: service_account
  ip_whitelist:
    - 10.0.0.0/8
    - 192.168.0.0/16
  ip_blacklist:
    - 203.0.113.0/24

hot_reload:
  enabled: true
  watch_config: true
  reload_interval: 5s
```

---

## Acceptance Criteria

- [ ] All 7 TODO handlers implemented
- [ ] File watching works with fsnotify
- [ ] Multiple log outputs supported
- [ ] Zero downtime during config reload
- [ ] All reload handlers tested
- [ ] Error handling comprehensive
- [ ] Documentation updated
- [ ] No TODO comments remain

---

## Dependencies

**Requires**:
- fsnotify library for file watching
- Unified config structure (relates to P1-hexagonal-migration)

**Blocks**:
- Production zero-downtime updates

---

## Related Files

- `configs/hot_reload.go` - Main hot reload logic
- `internal/config/config_loader.go` - Config loading
- `internal/config/logging_config.go` - Logging config
- `internal/app/app.go` - Application lifecycle

---

## Documentation to Update

- [ ] `docs/20-configuration/hot-reload.md` - Complete guide
- [ ] Configuration reference - Add hot reload section
- [ ] Operations guide - Add reload procedures
- [ ] Troubleshooting - Add reload issues

---

**Last Updated**: 2025-12-04
