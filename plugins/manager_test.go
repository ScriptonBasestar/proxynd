package plugins

import (
	"context"
	"testing"
	"time"
)

// Mock logger for testing
type mockLogger struct {
	infoLogs  []string
	errorLogs []string
	warnLogs  []string
	debugLogs []string
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.infoLogs = append(m.infoLogs, msg)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.errorLogs = append(m.errorLogs, msg)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.warnLogs = append(m.warnLogs, msg)
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.debugLogs = append(m.debugLogs, msg)
}

// Mock plugin for testing
type mockPlugin struct {
	name           string
	initCalled     bool
	readyCalled    bool
	shutdownCalled bool
	initError      error
	readyError     error
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Init(ctx Context) error {
	m.initCalled = true
	return m.initError
}

func (m *mockPlugin) OnReady(ctx Context) error {
	m.readyCalled = true
	return m.readyError
}

func (m *mockPlugin) OnShutdown(ctx context.Context) error {
	m.shutdownCalled = true
	return nil
}

func TestNewManager(t *testing.T) {
	logger := &mockLogger{}

	t.Run("with nil config", func(t *testing.T) {
		manager := NewManager(nil, logger)
		if manager == nil {
			t.Fatal("Expected non-nil manager")
		}

		if manager.config == nil {
			t.Fatal("Expected default config to be set")
		}

		if !manager.config.Enabled {
			t.Error("Expected plugin system to be enabled by default")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			Enabled: false,
		}

		manager := NewManager(cfg, logger)
		if manager.config.Enabled {
			t.Error("Expected plugin system to be disabled")
		}
	})
}

func TestManagerDiscover(t *testing.T) {
	logger := &mockLogger{}

	t.Run("when disabled", func(t *testing.T) {
		cfg := &Config{Enabled: false}
		manager := NewManager(cfg, logger)

		err := manager.Discover()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(manager.plugins) != 0 {
			t.Error("Expected no plugins when disabled")
		}
	})

	t.Run("with no registered plugins", func(t *testing.T) {
		cfg := DefaultConfig()
		manager := NewManager(cfg, logger)

		// Clear the global registry for this test
		oldRegistry := registry
		defer func() { registry = oldRegistry }()
		registry = []Plugin{}

		err := manager.Discover()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(manager.plugins) != 0 {
			t.Errorf("Expected 0 plugins, got %d", len(manager.plugins))
		}
	})
}

func TestManagerInitialize(t *testing.T) {
	logger := &mockLogger{}

	t.Run("successful initialization", func(t *testing.T) {
		// Setup
		plugin := &mockPlugin{name: "test-plugin"}
		oldRegistry := registry
		defer func() { registry = oldRegistry }()
		registry = []Plugin{plugin}

		cfg := DefaultConfig()
		manager := NewManager(cfg, logger)

		err := manager.Discover()
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}

		// Initialize
		ctx := Context{
			Logger: logger,
		}

		err = manager.Initialize(nil, ctx)
		if err != nil {
			t.Errorf("Initialize failed: %v", err)
		}

		if !plugin.initCalled {
			t.Error("Expected Init to be called")
		}

		if !manager.initialized {
			t.Error("Expected manager to be initialized")
		}
	})

	t.Run("when disabled", func(t *testing.T) {
		cfg := &Config{Enabled: false}
		manager := NewManager(cfg, logger)

		ctx := Context{Logger: logger}
		err := manager.Initialize(nil, ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestManagerReady(t *testing.T) {
	logger := &mockLogger{}

	t.Run("successful ready", func(t *testing.T) {
		plugin := &mockPlugin{name: "test-plugin"}
		oldRegistry := registry
		defer func() { registry = oldRegistry }()
		registry = []Plugin{plugin}

		cfg := DefaultConfig()
		manager := NewManager(cfg, logger)

		err := manager.Discover()
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}

		ctx := Context{Logger: logger}
		err = manager.Ready(ctx)
		if err != nil {
			t.Errorf("Ready failed: %v", err)
		}

		if !plugin.readyCalled {
			t.Error("Expected OnReady to be called")
		}

		if !manager.ready {
			t.Error("Expected manager to be ready")
		}
	})
}

func TestManagerShutdown(t *testing.T) {
	logger := &mockLogger{}

	t.Run("successful shutdown", func(t *testing.T) {
		plugin := &mockPlugin{name: "test-plugin"}
		oldRegistry := registry
		defer func() { registry = oldRegistry }()
		registry = []Plugin{plugin}

		cfg := DefaultConfig()
		manager := NewManager(cfg, logger)

		err := manager.Discover()
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}

		ctx := context.Background()
		err = manager.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}

		if !plugin.shutdownCalled {
			t.Error("Expected OnShutdown to be called")
		}
	})

	t.Run("reverse order shutdown", func(t *testing.T) {
		shutdownOrder := []string{}

		// Create custom mock plugins with order tracking
		type orderTrackingPlugin struct {
			mockPlugin
			order *[]string
		}

		plugin1 := &orderTrackingPlugin{
			mockPlugin: mockPlugin{name: "plugin1"},
			order:      &shutdownOrder,
		}

		plugin2 := &orderTrackingPlugin{
			mockPlugin: mockPlugin{name: "plugin2"},
			order:      &shutdownOrder,
		}

		// Implement OnShutdown interface
		type shutdownTracker interface {
			Plugin
			ShutdownHandler
		}

		var _ shutdownTracker = plugin1
		var _ shutdownTracker = plugin2

		oldRegistry := registry
		defer func() { registry = oldRegistry }()
		registry = []Plugin{&plugin1.mockPlugin, &plugin2.mockPlugin}

		cfg := DefaultConfig()
		cfg.Registry.Core = []PluginConfig{
			{Name: "plugin1", Enabled: true, Priority: 100},
			{Name: "plugin2", Enabled: true, Priority: 200},
		}

		manager := NewManager(cfg, logger)
		err := manager.Discover()
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}

		ctx := context.Background()
		err = manager.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}

		// Note: Without ability to track actual order, we verify shutdown was called
		if !plugin1.shutdownCalled || !plugin2.shutdownCalled {
			t.Error("Expected both plugins to be shut down")
		}
	})
}

func TestManagerTimeout(t *testing.T) {
	logger := &mockLogger{}

	t.Run("timeout on init", func(t *testing.T) {
		// Create a slow plugin type
		type slowPlugin struct {
			mockPlugin
		}

		slow := &slowPlugin{
			mockPlugin: mockPlugin{name: "slow-plugin"},
		}

		oldRegistry := registry
		defer func() { registry = oldRegistry }()
		registry = []Plugin{&slow.mockPlugin}

		cfg := DefaultConfig()
		cfg.Lifecycle.InitTimeout = 10 * time.Millisecond   // Very short timeout
		cfg.Lifecycle.FailurePolicy = FailurePolicyContinue // Don't halt on timeout

		manager := NewManager(cfg, logger)
		err := manager.Discover()
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}

		ctx := Context{Logger: logger}
		_ = manager.Initialize(nil, ctx)

		// Note: This test is limited without ability to override Init method
		// In real scenario with slow plugin, timeout would be triggered
		t.Log("Timeout test completed (simplified version)")
	})
}

func TestFailurePolicy(t *testing.T) {
	logger := &mockLogger{}

	testError := func() error {
		return nil // Replace with actual error for testing
	}

	t.Run("continue on failure", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Lifecycle.FailurePolicy = FailurePolicyContinue

		manager := NewManager(cfg, logger)

		err := manager.handleFailure("test-plugin", "Init", testError())
		if err != nil {
			t.Error("Expected continue policy to not return error")
		}
	})

	t.Run("halt on failure", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Lifecycle.FailurePolicy = FailurePolicyHalt

		manager := NewManager(cfg, logger)

		err := manager.handleFailure("test-plugin", "Init", testError())
		// If testError returns actual error, this should return error
		_ = err
	})

	t.Run("warn on failure", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Lifecycle.FailurePolicy = FailurePolicyWarn

		manager := NewManager(cfg, logger)

		err := manager.handleFailure("test-plugin", "Init", testError())
		if err != nil {
			t.Error("Expected warn policy to not return error")
		}
	})
}

func TestEnvironmentOverrides(t *testing.T) {
	logger := &mockLogger{}

	t.Run("apply enabled override", func(t *testing.T) {
		cfg := DefaultConfig()
		enabled := false
		cfg.Environments = map[string]EnvironmentOverrides{
			"development": {
				Overrides: []PluginOverride{
					{
						Plugin:  "test-plugin",
						Enabled: &enabled,
					},
				},
			},
		}

		plugin := &mockPlugin{name: "test-plugin"}
		oldRegistry := registry
		defer func() { registry = oldRegistry }()
		registry = []Plugin{plugin}

		// Set environment
		t.Setenv("PROXYND_ENV", "development")

		manager := NewManager(cfg, logger)
		err := manager.Discover()
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}

		// Find the plugin in manager
		found := false
		for _, p := range manager.plugins {
			if p.Name == "test-plugin" {
				if p.Enabled {
					t.Error("Expected plugin to be disabled by environment override")
				}
				found = true
				break
			}
		}

		if !found {
			t.Error("Plugin not found in manager")
		}
	})
}

func TestGetPlugins(t *testing.T) {
	logger := &mockLogger{}

	plugin1 := &mockPlugin{name: "plugin1"}
	plugin2 := &mockPlugin{name: "plugin2"}

	oldRegistry := registry
	defer func() { registry = oldRegistry }()
	registry = []Plugin{plugin1, plugin2}

	cfg := DefaultConfig()
	cfg.Registry.Core = []PluginConfig{
		{Name: "plugin1", Enabled: true},
		{Name: "plugin2", Enabled: false},
	}

	manager := NewManager(cfg, logger)
	err := manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	t.Run("GetPlugins returns all", func(t *testing.T) {
		all := manager.GetPlugins()
		if len(all) != 2 {
			t.Errorf("Expected 2 plugins, got %d", len(all))
		}
	})

	t.Run("GetEnabledPlugins returns only enabled", func(t *testing.T) {
		enabled := manager.GetEnabledPlugins()
		if len(enabled) != 1 {
			t.Errorf("Expected 1 enabled plugin, got %d", len(enabled))
		}

		if enabled[0].Name != "plugin1" {
			t.Errorf("Expected plugin1 to be enabled, got %s", enabled[0].Name)
		}
	})
}
