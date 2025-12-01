package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"proxynd/plugins"
)

// EventLoggerPlugin is an example plugin that demonstrates the EventHandler interface.
// It logs all events it receives to help developers understand event flow.
//
// This plugin is useful for:
// - Learning how the event system works
// - Debugging event dispatch issues
// - Understanding event timing and ordering
//
// To enable this plugin, add it to your plugins.yaml:
//
//	registry:
//	  core:
//	    - name: "event-logger"
//	      enabled: true
//	      priority: 150
//	      config:
//	        logLevel: "info"  # "debug", "info", or "none"
//	        prettyPrint: true
type EventLoggerPlugin struct {
	name        string
	logger      plugins.Logger
	logLevel    string
	prettyPrint bool
	eventCount  int
}

// NewEventLoggerPlugin creates a new EventLoggerPlugin instance
func NewEventLoggerPlugin() *EventLoggerPlugin {
	return &EventLoggerPlugin{
		name:        "event-logger",
		logLevel:    "info",
		prettyPrint: true,
	}
}

// Name returns the plugin name
func (p *EventLoggerPlugin) Name() string {
	return p.name
}

// Init initializes the plugin
func (p *EventLoggerPlugin) Init(ctx plugins.Context) error {
	p.logger = ctx.Logger

	// Load configuration from plugin config
	if cfg := ctx.Config; cfg != nil {
		if level, ok := cfg["logLevel"].(string); ok {
			p.logLevel = level
		}
		if pretty, ok := cfg["prettyPrint"].(bool); ok {
			p.prettyPrint = pretty
		}
	}

	p.logger.Info("EventLogger plugin initialized",
		"logLevel", p.logLevel,
		"prettyPrint", p.prettyPrint)

	return nil
}

// OnReady is called when all plugins are initialized
func (p *EventLoggerPlugin) OnReady(ctx plugins.Context) error {
	p.logger.Info("EventLogger plugin ready to receive events")
	return nil
}

// OnShutdown is called during graceful shutdown
func (p *EventLoggerPlugin) OnShutdown(ctx context.Context) error {
	p.logger.Info("EventLogger plugin shutting down",
		"totalEventsReceived", p.eventCount)
	return nil
}

// OnEvent handles incoming events (implements EventHandler interface)
func (p *EventLoggerPlugin) OnEvent(ctx context.Context, event plugins.Event) error {
	// Increment counter
	p.eventCount++

	// Check if logging is disabled
	if p.logLevel == "none" {
		return nil
	}

	// Log based on configured level
	switch p.logLevel {
	case "debug":
		p.logDebugEvent(event)
	case "info":
		p.logInfoEvent(event)
	default:
		// Unknown log level, use info
		p.logInfoEvent(event)
	}

	// Example: React to specific events
	switch event.Type {
	case plugins.EventPackageManagerStateChanged:
		return p.handlePMStateChange(event)
	case plugins.EventConfigReloaded:
		return p.handleConfigReload(event)
	case plugins.EventCacheCleared:
		return p.handleCacheCleared(event)
	default:
		// Unknown event type, just log it
		p.logger.Debug("Unknown event type received",
			"type", event.Type,
			"eventCount", p.eventCount)
	}

	return nil
}

// logDebugEvent logs event with full details
func (p *EventLoggerPlugin) logDebugEvent(event plugins.Event) {
	var dataStr string
	if p.prettyPrint {
		if jsonBytes, err := json.MarshalIndent(event.Data, "", "  "); err == nil {
			dataStr = string(jsonBytes)
		} else {
			dataStr = fmt.Sprintf("%+v", event.Data)
		}
	} else {
		dataStr = fmt.Sprintf("%+v", event.Data)
	}

	p.logger.Debug("Event received",
		"type", event.Type,
		"data", dataStr,
		"eventCount", p.eventCount,
		"timestamp", time.Now().UTC().Format(time.RFC3339))
}

// logInfoEvent logs event summary
func (p *EventLoggerPlugin) logInfoEvent(event plugins.Event) {
	p.logger.Info("Event received",
		"type", event.Type,
		"dataKeys", getMapKeys(event.Data),
		"eventCount", p.eventCount)
}

// handlePMStateChange handles package manager state change events
func (p *EventLoggerPlugin) handlePMStateChange(event plugins.Event) error {
	pm, _ := event.Data["package_manager"].(string)
	previousState, _ := event.Data["previous_state"].(bool)
	newState, _ := event.Data["new_state"].(bool)

	action := "disabled"
	if newState {
		action = "enabled"
	}

	p.logger.Info("Package manager state changed",
		"packageManager", pm,
		"action", action,
		"previousState", previousState,
		"newState", newState)

	// Example: Could trigger cache warmup if PM was enabled
	if !previousState && newState {
		p.logger.Debug("Package manager enabled, could trigger cache warmup",
			"packageManager", pm)
	}

	return nil
}

// handleConfigReload handles configuration reload events
func (p *EventLoggerPlugin) handleConfigReload(event plugins.Event) error {
	timestamp, _ := event.Data["timestamp"].(time.Time)
	configPath, _ := event.Data["config_path"].(string)

	p.logger.Info("Configuration reloaded",
		"timestamp", timestamp,
		"configPath", configPath)

	// Example: Could reload plugin-specific configuration here
	p.logger.Debug("Reloading plugin configuration...")

	return nil
}

// handleCacheCleared handles cache clear events
func (p *EventLoggerPlugin) handleCacheCleared(event plugins.Event) error {
	cacheType, _ := event.Data["cache_type"].(string)
	timestamp, _ := event.Data["timestamp"].(time.Time)

	p.logger.Info("Cache cleared",
		"cacheType", cacheType,
		"timestamp", timestamp)

	// Example: Could reset cache-related metrics
	p.logger.Debug("Resetting cache metrics",
		"cacheType", cacheType)

	return nil
}

// getMapKeys returns the keys of a map as a slice
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Ensure EventLoggerPlugin implements required interfaces
var (
	_ plugins.Plugin       = (*EventLoggerPlugin)(nil)
	_ plugins.EventHandler = (*EventLoggerPlugin)(nil)
)
