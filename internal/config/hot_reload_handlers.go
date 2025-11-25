// Package config provides configuration hot reload handlers.
// These handlers respond to runtime configuration changes without requiring a restart.
package config

import (
	"fmt"
	"sync"

	"proxynd/internal/logging"
)

// ReloadHandler defines the interface for components that need to respond to config changes.
type ReloadHandler interface {
	// Name returns the handler name for logging purposes
	Name() string
	// OnConfigReload handles configuration changes
	OnConfigReload(oldConfig, newConfig *UnifiedConfig) error
}

// HotReloadManager manages all reload handlers and coordinates config changes.
type HotReloadManager struct {
	handlers []ReloadHandler
	mu       sync.RWMutex
	logger   logging.Logger
}

// NewHotReloadManager creates a new hot reload manager.
func NewHotReloadManager() *HotReloadManager {
	return &HotReloadManager{
		handlers: make([]ReloadHandler, 0),
		logger:   logging.GetLogger(),
	}
}

// RegisterHandler adds a new reload handler to the manager.
func (m *HotReloadManager) RegisterHandler(handler ReloadHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
	m.logger.Info("Registered hot reload handler", logging.F("handler", handler.Name()))
}

// OnConfigChange processes configuration changes by invoking all registered handlers.
// Returns the first error encountered; other handlers continue to execute.
func (m *HotReloadManager) OnConfigChange(oldConfig, newConfig *UnifiedConfig) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var firstErr error
	for _, handler := range m.handlers {
		if err := handler.OnConfigReload(oldConfig, newConfig); err != nil {
			m.logger.Error("Hot reload handler failed",
				logging.F("handler", handler.Name()),
				logging.F("error", err))
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", handler.Name(), err)
			}
		} else {
			m.logger.Debug("Hot reload handler succeeded", logging.F("handler", handler.Name()))
		}
	}

	return firstErr
}

// LoggingReloadHandler handles dynamic logging configuration changes.
type LoggingReloadHandler struct {
	logger logging.Logger
}

// NewLoggingReloadHandler creates a new logging reload handler.
func NewLoggingReloadHandler() *LoggingReloadHandler {
	return &LoggingReloadHandler{
		logger: logging.GetLogger(),
	}
}

// Name returns the handler name.
func (h *LoggingReloadHandler) Name() string {
	return "LoggingReloadHandler"
}

// OnConfigReload handles logging configuration changes.
func (h *LoggingReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig == nil || newConfig == nil {
		return nil
	}

	// Handle log level change
	if oldConfig.Logging.Level != newConfig.Logging.Level {
		oldLevel := logging.LogLevel(oldConfig.Logging.Level)
		newLevel := logging.LogLevel(newConfig.Logging.Level)

		h.logger.Info("Changing log level",
			logging.F("old_level", oldLevel),
			logging.F("new_level", newLevel))

		logging.SetLevel(newLevel)

		h.logger.Info("Log level changed successfully",
			logging.F("current_level", logging.GetLevel()))
	}

	// Handle log format change
	if oldConfig.Logging.Format != newConfig.Logging.Format {
		h.logger.Warn("Log format change requested",
			logging.F("old_format", oldConfig.Logging.Format),
			logging.F("new_format", newConfig.Logging.Format),
			logging.F("note", "Format change requires logger reinitialization"))

		// Reinitialize logger with new format
		if err := logging.InitLogger(logging.LogConfig{
			Level:  logging.LogLevel(newConfig.Logging.Level),
			Format: newConfig.Logging.Format,
			Output: newConfig.Logging.Output,
		}); err != nil {
			return fmt.Errorf("failed to reinitialize logger with new format: %w", err)
		}

		h.logger.Info("Log format changed successfully",
			logging.F("format", newConfig.Logging.Format))
	}

	return nil
}

// CacheReloadHandler handles dynamic cache configuration changes.
type CacheReloadHandler struct {
	logger          logging.Logger
	onTTLChange     func(oldTTL, newTTL string)
	onMaxSizeChange func(oldSize, newSize string)
}

// NewCacheReloadHandler creates a new cache reload handler.
func NewCacheReloadHandler() *CacheReloadHandler {
	return &CacheReloadHandler{
		logger: logging.GetLogger(),
	}
}

// SetTTLChangeCallback sets a callback for TTL changes.
func (h *CacheReloadHandler) SetTTLChangeCallback(cb func(oldTTL, newTTL string)) {
	h.onTTLChange = cb
}

// SetMaxSizeChangeCallback sets a callback for max size changes.
func (h *CacheReloadHandler) SetMaxSizeChangeCallback(cb func(oldSize, newSize string)) {
	h.onMaxSizeChange = cb
}

// Name returns the handler name.
func (h *CacheReloadHandler) Name() string {
	return "CacheReloadHandler"
}

// OnConfigReload handles cache configuration changes.
func (h *CacheReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig == nil || newConfig == nil {
		return nil
	}

	// Backend change requires restart
	if oldConfig.Cache.Backend != newConfig.Cache.Backend {
		h.logger.Warn("Cache backend change detected",
			logging.F("old_backend", oldConfig.Cache.Backend),
			logging.F("new_backend", newConfig.Cache.Backend),
			logging.F("note", "Backend change requires restart"))
		return fmt.Errorf("cache backend change from '%s' to '%s' requires restart",
			oldConfig.Cache.Backend, newConfig.Cache.Backend)
	}

	// Handle TTL change
	if oldConfig.Cache.TTL != newConfig.Cache.TTL {
		h.logger.Info("Updating cache TTL",
			logging.F("old_ttl", oldConfig.Cache.TTL),
			logging.F("new_ttl", newConfig.Cache.TTL))

		if h.onTTLChange != nil {
			h.onTTLChange(oldConfig.Cache.TTL.String(), newConfig.Cache.TTL.String())
		}
	}

	// Handle max size change
	if oldConfig.Cache.MaxSize != newConfig.Cache.MaxSize {
		h.logger.Info("Updating cache max size",
			logging.F("old_size", oldConfig.Cache.MaxSize),
			logging.F("new_size", newConfig.Cache.MaxSize))

		if h.onMaxSizeChange != nil {
			h.onMaxSizeChange(oldConfig.Cache.MaxSize, newConfig.Cache.MaxSize)
		}
	}

	return nil
}

// MetricsReloadHandler handles dynamic metrics server configuration changes.
type MetricsReloadHandler struct {
	logger        logging.Logger
	onStartServer func(port int, path string)
	onStopServer  func()
}

// NewMetricsReloadHandler creates a new metrics reload handler.
func NewMetricsReloadHandler() *MetricsReloadHandler {
	return &MetricsReloadHandler{
		logger: logging.GetLogger(),
	}
}

// SetStartServerCallback sets a callback for starting the metrics server.
func (h *MetricsReloadHandler) SetStartServerCallback(cb func(port int, path string)) {
	h.onStartServer = cb
}

// SetStopServerCallback sets a callback for stopping the metrics server.
func (h *MetricsReloadHandler) SetStopServerCallback(cb func()) {
	h.onStopServer = cb
}

// Name returns the handler name.
func (h *MetricsReloadHandler) Name() string {
	return "MetricsReloadHandler"
}

// OnConfigReload handles metrics configuration changes.
func (h *MetricsReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig == nil || newConfig == nil {
		return nil
	}

	// Handle metrics enabled/disabled change
	if oldConfig.Metrics.Enabled != newConfig.Metrics.Enabled {
		if newConfig.Metrics.Enabled {
			h.logger.Info("Enabling metrics endpoint",
				logging.F("port", newConfig.Metrics.Port),
				logging.F("path", newConfig.Metrics.Path))

			if h.onStartServer != nil {
				h.onStartServer(newConfig.Metrics.Port, newConfig.Metrics.Path)
			}
		} else {
			h.logger.Info("Disabling metrics endpoint")

			if h.onStopServer != nil {
				h.onStopServer()
			}
		}
	}

	// Handle port change while enabled
	if newConfig.Metrics.Enabled && oldConfig.Metrics.Port != newConfig.Metrics.Port {
		h.logger.Info("Metrics port changed, restarting metrics server",
			logging.F("old_port", oldConfig.Metrics.Port),
			logging.F("new_port", newConfig.Metrics.Port))

		if h.onStopServer != nil {
			h.onStopServer()
		}
		if h.onStartServer != nil {
			h.onStartServer(newConfig.Metrics.Port, newConfig.Metrics.Path)
		}
	}

	return nil
}

// SecurityReloadHandler handles dynamic security configuration changes.
type SecurityReloadHandler struct {
	logger              logging.Logger
	onIPWhitelistChange func(enabled bool, allowedIPs []string)
	onUsersChange       func(users map[string]string)
}

// NewSecurityReloadHandler creates a new security reload handler.
func NewSecurityReloadHandler() *SecurityReloadHandler {
	return &SecurityReloadHandler{
		logger: logging.GetLogger(),
	}
}

// SetIPWhitelistChangeCallback sets a callback for IP whitelist changes.
func (h *SecurityReloadHandler) SetIPWhitelistChangeCallback(cb func(enabled bool, allowedIPs []string)) {
	h.onIPWhitelistChange = cb
}

// SetUsersChangeCallback sets a callback for user configuration changes.
func (h *SecurityReloadHandler) SetUsersChangeCallback(cb func(users map[string]string)) {
	h.onUsersChange = cb
}

// Name returns the handler name.
func (h *SecurityReloadHandler) Name() string {
	return "SecurityReloadHandler"
}

// OnConfigReload handles security configuration changes.
func (h *SecurityReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig == nil || newConfig == nil {
		return nil
	}

	// Handle IP whitelist changes
	oldIPWhitelist := oldConfig.Security.AccessControl.IPWhitelist
	newIPWhitelist := newConfig.Security.AccessControl.IPWhitelist

	// Combine IPs and CIDRs for the callback
	oldAllowedIPs := append(oldIPWhitelist.IPs, oldIPWhitelist.CIDRs...)
	newAllowedIPs := append(newIPWhitelist.IPs, newIPWhitelist.CIDRs...)

	if oldIPWhitelist.Enabled != newIPWhitelist.Enabled {
		if newIPWhitelist.Enabled {
			h.logger.Info("Enabling IP whitelist",
				logging.F("allowed_ips_count", len(newAllowedIPs)))
		} else {
			h.logger.Info("Disabling IP whitelist")
		}

		if h.onIPWhitelistChange != nil {
			h.onIPWhitelistChange(newIPWhitelist.Enabled, newAllowedIPs)
		}
	} else if newIPWhitelist.Enabled && !stringSliceEqual(oldAllowedIPs, newAllowedIPs) {
		h.logger.Info("Updating IP whitelist",
			logging.F("old_count", len(oldAllowedIPs)),
			logging.F("new_count", len(newAllowedIPs)))

		if h.onIPWhitelistChange != nil {
			h.onIPWhitelistChange(newIPWhitelist.Enabled, newAllowedIPs)
		}
	}

	// Handle BasicAuth user changes
	if oldConfig.Security.Authentication.BasicAuth != nil && newConfig.Security.Authentication.BasicAuth != nil {
		oldUsers := oldConfig.Security.Authentication.BasicAuth.Users
		newUsers := newConfig.Security.Authentication.BasicAuth.Users

		if !stringMapEqual(oldUsers, newUsers) {
			h.logger.Info("Updating BasicAuth users",
				logging.F("old_user_count", len(oldUsers)),
				logging.F("new_user_count", len(newUsers)))

			if h.onUsersChange != nil {
				h.onUsersChange(newUsers)
			}
		}

		// Handle realm change
		if oldConfig.Security.Authentication.BasicAuth.Realm != newConfig.Security.Authentication.BasicAuth.Realm {
			h.logger.Info("BasicAuth realm changed",
				logging.F("old_realm", oldConfig.Security.Authentication.BasicAuth.Realm),
				logging.F("new_realm", newConfig.Security.Authentication.BasicAuth.Realm))
		}
	}

	return nil
}

// Helper functions

// stringSliceEqual compares two string slices for equality.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// stringMapEqual compares two string maps for equality.
func stringMapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
