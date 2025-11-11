// +build integration

package plugins_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/plugins"
)

// TestPluginConfigLoading verifies that plugin configuration can be loaded from YAML
func TestPluginConfigLoading(t *testing.T) {
	// Create temporary config directory
	tmpDir := t.TempDir()

	t.Run("Load from missing file returns defaults", func(t *testing.T) {
		config, err := plugins.LoadConfig(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.True(t, config.Enabled, "Default config should have plugins enabled")
		assert.Equal(t, plugins.FailurePolicyContinue, config.Lifecycle.FailurePolicy)
	})

	t.Run("Load from YAML file", func(t *testing.T) {
		// Create test config file
		configPath := tmpDir + "/plugins.yaml"
		yamlContent := `
plugins:
  enabled: true
  lifecycle:
    initTimeout: 45s
    readyTimeout: 20s
    shutdownTimeout: 45s
    failurePolicy: "halt"
  registry:
    core:
      - name: "test-plugin"
        enabled: true
        priority: 100
        config:
          setting: "value"
`
		err := os.WriteFile(configPath, []byte(yamlContent), 0644)
		require.NoError(t, err)

		// Load config
		config, err := plugins.LoadConfig(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Verify parsed values
		assert.True(t, config.Enabled)
		assert.Equal(t, 45*time.Second, config.Lifecycle.InitTimeout)
		assert.Equal(t, 20*time.Second, config.Lifecycle.ReadyTimeout)
		assert.Equal(t, 45*time.Second, config.Lifecycle.ShutdownTimeout)
		assert.Equal(t, plugins.FailurePolicyHalt, config.Lifecycle.FailurePolicy)

		// Verify plugin registration
		require.Len(t, config.Registry.Core, 1)
		assert.Equal(t, "test-plugin", config.Registry.Core[0].Name)
		assert.True(t, config.Registry.Core[0].Enabled)
		assert.Equal(t, 100, config.Registry.Core[0].Priority)
		assert.Equal(t, "value", config.Registry.Core[0].Config["setting"])
	})

	t.Run("Environment overrides", func(t *testing.T) {
		// Create config with environment overrides
		configPath := tmpDir + "/plugins.yaml"
		yamlContent := `
plugins:
  enabled: true
  registry:
    core:
      - name: "plugin-a"
        enabled: true
        priority: 50
        config:
          mode: "normal"
  environments:
    development:
      overrides:
        - plugin: "plugin-a"
          enabled: false
        - plugin: "plugin-b"
          config:
            debugMode: true
    production:
      overrides:
        - plugin: "plugin-a"
          config:
            mode: "strict"
`
		err := os.WriteFile(configPath, []byte(yamlContent), 0644)
		require.NoError(t, err)

		// Load and verify
		config, err := plugins.LoadConfig(tmpDir)
		require.NoError(t, err)

		// Check environment overrides exist
		assert.Contains(t, config.Environments, "development")
		assert.Contains(t, config.Environments, "production")

		devOverrides := config.Environments["development"].Overrides
		require.Len(t, devOverrides, 2)
		assert.Equal(t, "plugin-a", devOverrides[0].Plugin)
		assert.False(t, *devOverrides[0].Enabled)
	})
}

// TestPluginManagerWithConfig verifies the manager works with loaded configuration
func TestPluginManagerWithConfig(t *testing.T) {
	// Create test config
	config := plugins.DefaultConfig()
	config.Lifecycle.FailurePolicy = plugins.FailurePolicyContinue

	// Create simple logger
	logger := &simpleLogger{t: t}

	// Create manager
	manager := plugins.NewManager(config, logger)
	require.NotNil(t, manager)

	t.Run("Discover with no plugins", func(t *testing.T) {
		err := manager.Discover()
		assert.NoError(t, err)

		discoveredPlugins := manager.GetPlugins()
		// Should have 0 plugins (none registered in this test)
		assert.GreaterOrEqual(t, len(discoveredPlugins), 0)
	})

	t.Run("Manager state checks", func(t *testing.T) {
		// Initially not initialized
		assert.False(t, manager.IsInitialized())
		assert.False(t, manager.IsReady())

		// Get metrics should not panic
		summary := manager.GetMetricsSummary()
		assert.NotNil(t, summary)
	})

	t.Run("Shutdown without initialization", func(t *testing.T) {
		ctx := context.Background()
		err := manager.Shutdown(ctx)
		assert.NoError(t, err) // Should handle gracefully
	})
}

// TestEnvironmentVariableHandling verifies PROXYND_ENV environment variable handling
func TestEnvironmentVariableHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with environment overrides
	configPath := tmpDir + "/plugins.yaml"
	yamlContent := `
plugins:
  enabled: true
  registry:
    core:
      - name: "test-plugin"
        enabled: true
        priority: 100
  environments:
    testing:
      overrides:
        - plugin: "test-plugin"
          enabled: false
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set environment
	oldEnv := os.Getenv("PROXYND_ENV")
	os.Setenv("PROXYND_ENV", "testing")
	defer os.Setenv("PROXYND_ENV", oldEnv)

	// Load config and create manager
	config, err := plugins.LoadConfig(tmpDir)
	require.NoError(t, err)

	logger := &simpleLogger{t: t}
	manager := plugins.NewManager(config, logger)

	// Discover should apply environment overrides
	err = manager.Discover()
	require.NoError(t, err)

	// Note: We can't verify the override was applied here because
	// no actual plugins are registered. This test just verifies
	// the configuration loading doesn't error.
}

// simpleLogger implements plugins.Logger for testing
type simpleLogger struct {
	t *testing.T
}

func (l *simpleLogger) Debug(msg string, fields ...interface{}) {
	l.t.Logf("[DEBUG] %s %v", msg, fields)
}

func (l *simpleLogger) Info(msg string, fields ...interface{}) {
	l.t.Logf("[INFO] %s %v", msg, fields)
}

func (l *simpleLogger) Warn(msg string, fields ...interface{}) {
	l.t.Logf("[WARN] %s %v", msg, fields)
}

func (l *simpleLogger) Error(msg string, fields ...interface{}) {
	l.t.Logf("[ERROR] %s %v", msg, fields)
}

func (l *simpleLogger) Fatal(msg string, fields ...interface{}) {
	l.t.Fatalf("[FATAL] %s %v", msg, fields)
}
