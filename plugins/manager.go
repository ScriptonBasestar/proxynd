package plugins

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Manager orchestrates plugin lifecycle based on configuration
type Manager struct {
	config  *Config
	plugins []PluginConfig
	logger  Logger
	metrics *MetricsCollector

	// State
	initialized bool
	ready       bool
}

// NewManager creates a plugin manager from configuration
func NewManager(cfg *Config, logger Logger) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Manager{
		config:  cfg,
		plugins: []PluginConfig{},
		logger:  logger,
		metrics: NewMetricsCollector(true), // Metrics enabled by default
	}
}

// Discover scans for plugins and merges with config
func (m *Manager) Discover() error {
	if !m.config.Enabled {
		m.logger.Info("Plugin system disabled")
		return nil
	}

	// 1. Get all registered plugins (hardcoded via plugins.Register)
	registered := All()

	// 2. Merge with configuration
	m.plugins = m.mergePluginsWithConfig(registered)

	// 3. Apply environment overrides
	env := getEnvironment()
	if overrides, ok := m.config.Environments[env]; ok {
		m.applyOverrides(overrides)
	}

	// 4. Sort by priority (lower number = higher priority)
	sort.Slice(m.plugins, func(i, j int) bool {
		return m.plugins[i].Priority < m.plugins[j].Priority
	})

	// Record discovery metrics
	for _, p := range m.plugins {
		m.metrics.RecordPluginDiscovered(p.Name, p.Priority, p.Enabled)
	}

	m.logger.Info("Discovered plugins",
		"total", len(m.plugins),
		"enabled", m.countEnabled(),
		"environment", env)

	return nil
}

// Initialize runs Init() on all enabled plugins
func (m *Manager) Initialize(app *fiber.App, ctx Context) error {
	if !m.config.Enabled {
		return nil
	}

	timeout := m.config.Lifecycle.InitTimeout

	for _, pluginCfg := range m.plugins {
		if !pluginCfg.Enabled {
			m.logger.Debug("Skipping disabled plugin", "name", pluginCfg.Name)
			continue
		}

		plugin := pluginCfg.instance
		if plugin == nil {
			m.logger.Warn("Plugin instance is nil", "name", pluginCfg.Name)
			continue
		}

		// Run initializer
		if initializer, ok := plugin.(Initializer); ok {
			m.logger.Info("Initializing plugin",
				"name", plugin.Name(),
				"priority", pluginCfg.Priority)

			m.metrics.RecordPluginStateChange(plugin.Name(), PluginStatusInitializing)

			start := time.Now()
			err := m.runWithTimeout(timeout, func() error {
				return initializer.Init(ctx)
			})
			duration := time.Since(start)

			m.metrics.RecordPluginInit(plugin.Name(), duration, err)

			if err != nil {
				m.logger.Error("Plugin initialization failed",
					"plugin", plugin.Name(),
					"duration_ms", duration.Milliseconds(),
					"error", err.Error())

				if err := m.handleFailure(plugin.Name(), "Init", err); err != nil {
					return err
				}
			} else {
				m.logger.Info("Plugin initialized successfully",
					"plugin", plugin.Name(),
					"duration_ms", duration.Milliseconds())
			}
		}

		// Mount middleware
		if middlewareProvider, ok := plugin.(MiddlewareProvider); ok {
			if middleware := middlewareProvider.Middleware(); middleware != nil {
				m.logger.Info("Mounting middleware", "plugin", plugin.Name())
				app.Use(middleware)
			}
		}

		// Register routes
		if routesProvider, ok := plugin.(RoutesProvider); ok {
			basePath := m.getMountPath(plugin)
			m.logger.Info("Registering routes",
				"plugin", plugin.Name(),
				"path", basePath)

			group := app.Group(basePath)
			routesProvider.Routes(group)
		}
	}

	m.initialized = true
	return nil
}

// Ready runs OnReady() on all enabled plugins
func (m *Manager) Ready(ctx Context) error {
	if !m.config.Enabled {
		return nil
	}

	timeout := m.config.Lifecycle.ReadyTimeout

	for _, pluginCfg := range m.plugins {
		if !pluginCfg.Enabled {
			continue
		}

		plugin := pluginCfg.instance
		if plugin == nil {
			continue
		}

		if readyHandler, ok := plugin.(ReadyHandler); ok {
			m.logger.Info("Running ready hook", "name", plugin.Name())

			start := time.Now()
			err := m.runWithTimeout(timeout, func() error {
				return readyHandler.OnReady(ctx)
			})
			duration := time.Since(start)

			m.metrics.RecordPluginReady(plugin.Name(), duration, err)

			if err != nil {
				m.logger.Error("Plugin ready hook failed",
					"plugin", plugin.Name(),
					"duration_ms", duration.Milliseconds(),
					"error", err.Error())

				if err := m.handleFailure(plugin.Name(), "OnReady", err); err != nil {
					return err
				}
			} else {
				m.logger.Info("Plugin ready",
					"plugin", plugin.Name(),
					"duration_ms", duration.Milliseconds())
			}
		}
	}

	m.ready = true

	// Log metrics summary
	summary := m.metrics.GetSummary()
	m.logger.Info("All plugins ready",
		"total", m.countEnabled(),
		"ready", summary.ReadyPlugins,
		"failed", summary.FailedPlugins,
		"avg_init_ms", summary.AverageInitTime.Milliseconds(),
		"avg_ready_ms", summary.AverageReadyTime.Milliseconds())

	return nil
}

// Shutdown runs OnShutdown() on all enabled plugins (reverse order)
func (m *Manager) Shutdown(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	timeout := m.config.Lifecycle.ShutdownTimeout

	m.logger.Info("Shutting down plugins", "total", m.countEnabled())

	// Shutdown in reverse priority order
	for i := len(m.plugins) - 1; i >= 0; i-- {
		pluginCfg := m.plugins[i]
		if !pluginCfg.Enabled {
			continue
		}

		plugin := pluginCfg.instance
		if plugin == nil {
			continue
		}

		if shutdownHandler, ok := plugin.(ShutdownHandler); ok {
			m.logger.Info("Running shutdown hook", "name", plugin.Name())

			m.metrics.RecordPluginStateChange(plugin.Name(), PluginStatusShuttingDown)

			start := time.Now()
			err := m.runWithTimeout(timeout, func() error {
				return shutdownHandler.OnShutdown(ctx)
			})
			duration := time.Since(start)

			m.metrics.RecordPluginShutdown(plugin.Name(), duration, err)

			if err != nil {
				m.logger.Error("Shutdown failed",
					"plugin", plugin.Name(),
					"duration_ms", duration.Milliseconds(),
					"error", err.Error())
				// Always continue shutdown even on errors
			} else {
				m.logger.Info("Plugin shut down successfully",
					"plugin", plugin.Name(),
					"duration_ms", duration.Milliseconds())
			}
		}
	}

	// Log final metrics summary
	summary := m.metrics.GetSummary()
	m.logger.Info("Plugin shutdown complete",
		"total", summary.TotalPlugins,
		"total_errors", summary.TotalErrors,
		"total_shutdown_ms", summary.TotalShutdownTime.Milliseconds())

	return nil
}

// GetPlugins returns all plugins (enabled and disabled)
func (m *Manager) GetPlugins() []PluginConfig {
	return m.plugins
}

// GetEnabledPlugins returns only enabled plugins
func (m *Manager) GetEnabledPlugins() []PluginConfig {
	result := []PluginConfig{}
	for _, p := range m.plugins {
		if p.Enabled {
			result = append(result, p)
		}
	}
	return result
}

// IsInitialized returns whether the manager has been initialized
func (m *Manager) IsInitialized() bool {
	return m.initialized
}

// IsReady returns whether all plugins are ready
func (m *Manager) IsReady() bool {
	return m.ready
}

// GetMetrics returns the metrics collector
func (m *Manager) GetMetrics() *MetricsCollector {
	return m.metrics
}

// GetPluginMetrics returns metrics for a specific plugin
func (m *Manager) GetPluginMetrics(name string) *PluginMetrics {
	return m.metrics.GetPluginMetrics(name)
}

// GetMetricsSummary returns aggregated metrics summary
func (m *Manager) GetMetricsSummary() MetricsSummary {
	return m.metrics.GetSummary()
}

// Private helpers

func (m *Manager) mergePluginsWithConfig(registered []Plugin) []PluginConfig {
	result := []PluginConfig{}

	// Flatten registry (core + enterprise + cloud)
	allConfigs := append(m.config.Registry.Core,
		append(m.config.Registry.Enterprise, m.config.Registry.Cloud...)...)

	// Match registered plugins with config
	for _, plugin := range registered {
		cfg := m.findPluginConfig(plugin.Name(), allConfigs)
		if cfg == nil {
			// Not in config - use defaults
			cfg = &PluginConfig{
				Name:     plugin.Name(),
				Enabled:  true, // Default: enabled
				Priority: 500,  // Default: low priority
				Config:   map[string]interface{}{},
			}

			m.logger.Debug("Plugin not in config, using defaults",
				"name", plugin.Name())
		}

		cfg.SetInstance(plugin)
		result = append(result, *cfg)
	}

	// Warn about configured plugins that weren't registered
	for _, cfgPlugin := range allConfigs {
		found := false
		for _, plugin := range registered {
			if plugin.Name() == cfgPlugin.Name {
				found = true
				break
			}
		}

		if !found {
			m.logger.Warn("Configured plugin not registered",
				"name", cfgPlugin.Name)
		}
	}

	return result
}

func (m *Manager) findPluginConfig(name string, configs []PluginConfig) *PluginConfig {
	for i := range configs {
		if configs[i].Name == name {
			return &configs[i]
		}
	}
	return nil
}

func (m *Manager) applyOverrides(overrides EnvironmentOverrides) {
	for _, override := range overrides.Overrides {
		for i := range m.plugins {
			if m.plugins[i].Name == override.Plugin {
				// Override enabled flag
				if override.Enabled != nil {
					m.plugins[i].Enabled = *override.Enabled

					m.logger.Info("Applied environment override",
						"plugin", override.Plugin,
						"enabled", *override.Enabled)
				}

				// Merge config
				if override.Config != nil {
					if m.plugins[i].Config == nil {
						m.plugins[i].Config = map[string]interface{}{}
					}

					for key, value := range override.Config {
						m.plugins[i].Config[key] = value
					}

					m.logger.Debug("Applied config override",
						"plugin", override.Plugin,
						"keys", len(override.Config))
				}
			}
		}
	}
}

func (m *Manager) handleFailure(pluginName, hook string, err error) error {
	switch m.config.Lifecycle.FailurePolicy {
	case FailurePolicyHalt:
		return fmt.Errorf("plugin %s %s failed: %w", pluginName, hook, err)

	case FailurePolicyWarn:
		m.logger.Warn("Plugin lifecycle hook failed",
			"plugin", pluginName,
			"hook", hook,
			"error", err.Error())
		return nil

	case FailurePolicyContinue:
	default:
		m.logger.Error("Plugin lifecycle hook failed",
			"plugin", pluginName,
			"hook", hook,
			"error", err.Error())
		return nil
	}

	return nil
}

func (m *Manager) runWithTimeout(timeout time.Duration, fn func() error) error {
	done := make(chan error, 1)

	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("operation timed out after %s", timeout)
	}
}

func (m *Manager) getMountPath(plugin Plugin) string {
	if mountProvider, ok := plugin.(MountPathProvider); ok {
		if mount := mountProvider.MountPath(); mount != "" {
			return mount
		}
	}
	return fmt.Sprintf("/plugins/%s", plugin.Name())
}

func (m *Manager) countEnabled() int {
	count := 0
	for _, p := range m.plugins {
		if p.Enabled {
			count++
		}
	}
	return count
}

func getEnvironment() string {
	env := os.Getenv("PROXYND_ENV")
	if env == "" {
		env = "development"
	}
	return strings.ToLower(env)
}
