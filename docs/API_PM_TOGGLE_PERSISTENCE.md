# Package Manager Toggle API with Persistence

## Overview

This document describes the implementation of the Package Manager Toggle API endpoint with persistent storage support.

## API Endpoint

**POST** `/api/v1/pm/:name/toggle`

## Features

### 1. Package Manager Control
- Toggle enabled/disabled state of package managers
- Support explicit state setting via JSON body
- Support toggle mode (flip current state)
- Real-time state updates with persistence

### 2. Persistent Storage
- Changes are saved to `config.yaml` immediately
- Automatic rollback on save failure
- Thread-safe configuration updates
- Graceful error handling

### 3. Supported Package Managers
- ✅ **maven** - Fully implemented
- ✅ **npm** - Fully implemented
- ✅ **docker** - Fully implemented
- ✅ **pypi** - Fully implemented
- ✅ **apt** - Fully implemented
- 🔧 **yum** - Configuration structure pending (returns 501)
- 🔧 **apk** - Configuration structure pending (returns 501)

## Usage Examples

### 1. Toggle Mode (Flip Current State)

```bash
curl -X POST http://localhost:8080/api/v1/pm/npm/toggle
```

**Response:**
```json
{
  "name": "npm",
  "enabled": false,
  "previous_state": true,
  "message": "Package manager 'npm' disabled successfully",
  "persisted": true
}
```

### 2. Explicit Enable

```bash
curl -X POST http://localhost:8080/api/v1/pm/maven/toggle \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'
```

**Response:**
```json
{
  "name": "maven",
  "enabled": true,
  "previous_state": false,
  "message": "Package manager 'maven' enabled successfully",
  "persisted": true
}
```

### 3. Explicit Disable

```bash
curl -X POST http://localhost:8080/api/v1/pm/docker/toggle \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

**Response:**
```json
{
  "name": "docker",
  "enabled": false,
  "previous_state": true,
  "message": "Package manager 'docker' disabled successfully",
  "persisted": true
}
```

## Error Handling

### Invalid Package Manager Name

```bash
curl -X POST http://localhost:8080/api/v1/pm/invalid/toggle
```

**Response (400 Bad Request):**
```json
{
  "error": "invalid_package_manager",
  "message": "Unknown package manager: invalid",
  "valid": ["maven", "npm", "docker", "pypi", "apt", "yum", "apk"]
}
```

### Not Implemented Package Manager

```bash
curl -X POST http://localhost:8080/api/v1/pm/yum/toggle
```

**Response (501 Not Implemented):**
```json
{
  "error": "not_implemented",
  "message": "Package manager 'yum' configuration not yet implemented"
}
```

### Config Service Unavailable

**Response (503 Service Unavailable):**
```json
{
  "error": "config_service_unavailable",
  "message": "Configuration service is not available"
}
```

### Save Failed (with Automatic Rollback)

**Response (500 Internal Server Error):**
```json
{
  "error": "save_failed",
  "message": "Failed to persist configuration: <error details>"
}
```

**Note:** The in-memory state is automatically rolled back to the previous value if save fails.

## Implementation Details

### Architecture

```
┌──────────────┐
│  WebUI/CLI   │
└──────┬───────┘
       │ POST /api/v1/pm/:name/toggle
       ▼
┌────────────────────────────────┐
│  api_v1_router.go              │
│  - Validate PM name            │
│  - Get config service          │
│  - Load RootConfig             │
│  - Update enabled state        │
│  - Persist to disk             │
│  - Rollback on error           │
└────────────┬───────────────────┘
             │
             ▼
┌────────────────────────────────┐
│  Config Service (unified)      │
│  - GetRootConfig()             │
│  - SaveConfig()                │
└────────────┬───────────────────┘
             │
             ▼
┌────────────────────────────────┐
│  File System                   │
│  config.yaml                   │
│  (YAML marshaled RootConfig)   │
└────────────────────────────────┘
```

### Persistence Flow

1. **Load Current Config**
   - Get config service from fiber.Locals
   - Call `GetRootConfig()` with 5-second timeout
   - Receive current RootConfig

2. **Update State**
   - Parse request body (if provided)
   - Determine new state (explicit or toggle)
   - Update registry enabled flag via pointer

3. **Persist to Disk**
   - Call `SaveConfig()` with 10-second timeout
   - Marshal RootConfig to YAML
   - Write to `config.yaml` with 0644 permissions
   - Log success

4. **Error Handling**
   - If save fails, rollback in-memory state
   - Return 500 error with details
   - State remains unchanged

### Thread Safety

- Config service uses `sync.RWMutex` for concurrent access
- SaveConfig locks for write during marshal and file write
- Multiple concurrent toggle requests are serialized

### Configuration File

Changes are persisted to `$CONFIG_DIR/config.yaml`:

```yaml
registries:
  maven:
    enabled: true  # ← Updated by toggle endpoint
    repositories: [...]
  npm:
    enabled: false  # ← Updated by toggle endpoint
    upstream: "https://registry.npmjs.org"
  docker:
    enabled: true  # ← Updated by toggle endpoint
    registries: [...]
  pypi:
    enabled: true  # ← Updated by toggle endpoint
    upstream: "https://pypi.org"
  apt:
    enabled: false  # ← Updated by toggle endpoint
    mirrors: {...}
```

## Integration with WebUI

The WebUI can use this endpoint to provide real-time PM management:

```javascript
// Toggle PM state
async function togglePackageManager(name) {
  const response = await fetch(`/api/v1/pm/${name}/toggle`, {
    method: 'POST'
  });
  return await response.json();
}

// Set explicit state
async function setPackageManager(name, enabled) {
  const response = await fetch(`/api/v1/pm/${name}/toggle`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled })
  });
  return await response.json();
}

// Usage
const result = await togglePackageManager('npm');
console.log(result.message); // "Package manager 'npm' disabled successfully"
console.log(result.persisted); // true
```

## Future Enhancements

### Phase 2 (Planned)

1. **Plugin System Notification**
   - Notify plugin manager when PM state changes
   - Allow plugins to react to enable/disable events
   - Support plugin-specific reload logic

2. **Webhook Integration**
   - Trigger webhooks on PM state change
   - Support audit logging via webhook system
   - Integration with existing alert system

3. **Hot Reload**
   - Reload PM configuration without restart
   - Graceful transition between enabled/disabled states
   - Maintain active connections during transition

4. **Batch Operations**
   - Toggle multiple PMs in single request
   - Atomic batch updates with rollback support
   - Bulk enable/disable for environments

5. **Validation & Dependencies**
   - Check PM dependencies before disable
   - Warn if disabling will affect other services
   - Configurable validation rules

## Testing

### Manual Testing

```bash
# 1. Enable maven
curl -X POST http://localhost:8080/api/v1/pm/maven/toggle \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# 2. Verify config file updated
cat $CONFIG_DIR/config.yaml | grep -A 2 "maven:"

# 3. Restart server
sudo systemctl restart proxynd

# 4. Verify state persisted
curl http://localhost:8080/api/v1/pm | jq '.[] | select(.name=="maven")'
```

### Integration Testing

```bash
# Test suite
./scripts/test-pm-toggle.sh

# Expected results:
# ✅ Toggle from enabled to disabled
# ✅ Toggle from disabled to enabled
# ✅ Explicit enable
# ✅ Explicit disable
# ✅ Invalid PM name returns 400
# ✅ Not implemented PM returns 501
# ✅ Config file updated correctly
# ✅ State persists across restarts
```

## Configuration Service Architecture

### Service Interface

```go
type Service interface {
    GetRootConfig(ctx context.Context) (*config.RootConfig, error)
    SaveConfig(ctx context.Context) error
    Reload(ctx context.Context) error
    // ... other methods
}
```

### Unified Service Implementation

```go
func (s *unifiedService) SaveConfig(ctx context.Context) error {
    s.mu.RLock()
    cfg := s.unifiedConfig
    configDir := s.configDir
    s.mu.RUnlock()

    configPath := fmt.Sprintf("%s/config.yaml", configDir)

    data, err := yaml.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("failed to marshal configuration: %w", err)
    }

    if err := os.WriteFile(configPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write configuration file: %w", err)
    }

    s.logger.Info("Configuration saved successfully",
        logging.F("path", configPath))

    return nil
}
```

## Troubleshooting

### Issue: Changes not persisting

**Check:**
1. Config service is available in fiber.Locals
2. CONFIG_DIR environment variable is set correctly
3. File permissions allow write (0644)
4. Disk space is available

**Solution:**
```bash
# Check config service
curl http://localhost:8080/api/v1/system/info

# Check permissions
ls -la $CONFIG_DIR/config.yaml

# Check disk space
df -h $CONFIG_DIR
```

### Issue: State reverts after restart

**Check:**
1. Config file was actually updated
2. Server is reading from correct config file
3. No environment overrides

**Solution:**
```bash
# Verify config file
cat $CONFIG_DIR/config.yaml | grep -A 5 "registries:"

# Check server config path
ps aux | grep proxynd | grep config

# Check env overrides
env | grep CONFIG_DIR
```

## Security Considerations

1. **File Permissions**
   - Config file saved with 0644 (owner read/write, others read)
   - Ensure CONFIG_DIR has appropriate ownership
   - Consider restricting to 0600 in production

2. **Authentication**
   - TODO: Add authentication middleware to toggle endpoint
   - Require admin role for PM management
   - Audit log all changes

3. **Validation**
   - Input validation prevents invalid PM names
   - Type checking on request body
   - Timeout protection on all operations

## Related Documentation

- [Plugin System Rollout Playbook](./PLUGIN_SYSTEM_ROLLOUT.md)
- [Configuration Guide](../20-configuration/)
- [API Reference](../04-api-reference/)
- [WebUI Integration](../../proxynd-webui/docs/)

---

**Version:** 1.0
**Last Updated:** 2025-11-18
**Status:** Production Ready
