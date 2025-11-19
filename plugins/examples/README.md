# ProxyND Plugin Examples

This directory contains example plugins that demonstrate how to use the ProxyND plugin system.

## Example Plugins

### 1. EventLoggerPlugin (`event_logger_plugin.go`)

A simple plugin that logs all events it receives. Perfect for learning how the event system works.

**Features:**
- Logs all events with configurable detail level
- Demonstrates EventHandler interface implementation
- Shows best practices for event handling
- Useful for debugging event flow

**Configuration:**

```yaml
plugins:
  registry:
    core:
      - name: "event-logger"
        enabled: true
        priority: 150
        config:
          logLevel: "info"      # "debug", "info", or "none"
          prettyPrint: true     # Pretty-print JSON data
```

**Log Levels:**
- `debug`: Full event details including all data fields (JSON formatted if prettyPrint is true)
- `info`: Event summary with event type and data keys only
- `none`: No logging (but event counter still increments)

**Usage:**

To use this plugin in your ProxyND installation:

1. **Register the plugin** in `plugins/registry.go`:
   ```go
   import "proxynd/plugins/examples"

   func init() {
       Register(examples.NewEventLoggerPlugin())
   }
   ```

2. **Enable in configuration** (`plugins.yaml`):
   ```yaml
   plugins:
     enabled: true
     registry:
       core:
         - name: "event-logger"
           enabled: true
           priority: 150
           config:
             logLevel: "debug"
             prettyPrint: true
   ```

3. **Observe events** in logs when you:
   - Toggle package managers: `POST /api/v1/pm/:name/toggle`
   - Reload configuration
   - Clear cache

**Example Output:**

```
INFO  EventLogger plugin initialized logLevel=debug prettyPrint=true
INFO  EventLogger plugin ready to receive events
INFO  Event received type=package_manager_state_changed dataKeys=[package_manager previous_state new_state timestamp] eventCount=1
INFO  Package manager state changed packageManager=npm action=enabled previousState=false newState=true
DEBUG Event received type=package_manager_state_changed eventCount=1 timestamp=2025-11-19T00:00:00Z data={
  "package_manager": "npm",
  "previous_state": false,
  "new_state": true,
  "timestamp": "2025-11-19T00:00:00Z"
}
```

## Creating Your Own Plugin

Use `EventLoggerPlugin` as a template:

1. **Implement required interfaces:**
   ```go
   type MyPlugin struct {
       name   string
       logger plugins.Logger
   }

   func (p *MyPlugin) Name() string { return p.name }
   func (p *MyPlugin) Init(ctx plugins.Context) error { /* ... */ }
   func (p *MyPlugin) OnReady(ctx plugins.Context) error { /* ... */ }
   func (p *MyPlugin) OnShutdown(ctx context.Context) error { /* ... */ }
   ```

2. **Optionally implement EventHandler:**
   ```go
   func (p *MyPlugin) OnEvent(ctx context.Context, event plugins.Event) error {
       switch event.Type {
       case plugins.EventPackageManagerStateChanged:
           return p.handlePMChange(event)
       default:
           return nil
       }
   }
   ```

3. **Register your plugin:**
   ```go
   func init() {
       plugins.Register(NewMyPlugin())
   }
   ```

4. **Add to configuration:**
   ```yaml
   registry:
     core:
       - name: "my-plugin"
         enabled: true
         priority: 100
   ```

## Best Practices

**From EventLoggerPlugin:**

1. ✅ **Fast event handlers** - Return quickly, use goroutines for heavy work
2. ✅ **Handle unknown events gracefully** - Return nil for unknown event types
3. ✅ **Type assertions with safety** - Use comma-ok pattern: `value, ok := data["key"].(type)`
4. ✅ **Configurable behavior** - Use plugin config for customization
5. ✅ **Comprehensive logging** - Log important state changes and errors
6. ✅ **Clean shutdown** - Log final state in OnShutdown()

**Error Handling:**

```go
func (p *MyPlugin) OnEvent(ctx context.Context, event plugins.Event) error {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Type assertion with safety
    pm, ok := event.Data["package_manager"].(string)
    if !ok {
        p.logger.Warn("Missing package_manager field")
        return nil  // Don't error, just skip
    }

    // Process event
    return p.doWork(pm)
}
```

## Testing

See `plugins/manager_test.go` for examples of how to test plugins with events:

```go
func TestMyPlugin_OnEvent(t *testing.T) {
    plugin := NewMyPlugin()

    // Initialize plugin
    ctx := plugins.Context{
        Logger: &mockLogger{},
        Config: map[string]interface{}{
            "myOption": true,
        },
    }
    err := plugin.Init(ctx)
    require.NoError(t, err)

    // Send test event
    event := plugins.Event{
        Type: plugins.EventPackageManagerStateChanged,
        Data: map[string]interface{}{
            "package_manager": "npm",
            "new_state":       true,
        },
    }

    err = plugin.OnEvent(context.Background(), event)
    require.NoError(t, err)

    // Assert expected behavior
    // ...
}
```

## More Examples

Want to contribute an example plugin? Some ideas:

- **MetricsCollectorPlugin** - Collect event-based metrics
- **CacheWarmerPlugin** - Warm cache when PM is enabled
- **NotificationPlugin** - Send Slack/email on events
- **AuditLoggerPlugin** - Log events to audit database
- **HealthCheckPlugin** - Update health status based on events

See `docs/PLUGIN_OPERATOR_GUIDE.md` for the complete plugin development guide.
