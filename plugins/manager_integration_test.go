package plugins

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Integration tests for full plugin lifecycle with realistic scenarios

// Resource tracking plugin for testing cleanup
type resourceTrackingPlugin struct {
	name           string
	resources      []string
	initCalled     bool
	readyCalled    bool
	shutdownCalled bool
	mu             sync.Mutex
}

func (r *resourceTrackingPlugin) Name() string {
	return r.name
}

func (r *resourceTrackingPlugin) Init(ctx Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resources = append(r.resources, "database-connection")
	r.resources = append(r.resources, "cache-connection")
	r.initCalled = true
	return nil
}

func (r *resourceTrackingPlugin) OnReady(ctx Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resources = append(r.resources, "metrics-collector")
	r.readyCalled = true
	return nil
}

func (r *resourceTrackingPlugin) OnShutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Simulate resource cleanup
	r.resources = []string{}
	r.shutdownCalled = true
	return nil
}

func (r *resourceTrackingPlugin) GetResources() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]string, len(r.resources))
	copy(result, r.resources)
	return result
}

func TestPluginLifecycle_EndToEnd(t *testing.T) {
	logger := &mockLogger{}

	// Setup test plugins
	plugin1 := &resourceTrackingPlugin{
		name:      "rbac",
		resources: []string{},
	}

	plugin2 := &resourceTrackingPlugin{
		name:      "audit",
		resources: []string{},
	}

	plugin3 := &resourceTrackingPlugin{
		name:      "metrics",
		resources: []string{},
	}

	// Mock registry
	oldRegistry := registry
	defer func() { registry = oldRegistry }()
	registry = []Plugin{plugin1, plugin2, plugin3}

	// Configuration with priorities
	cfg := DefaultConfig()
	cfg.Registry.Core = []PluginConfig{
		{Name: "rbac", Enabled: true, Priority: 10},     // Highest priority
		{Name: "audit", Enabled: true, Priority: 20},    // Medium priority
		{Name: "metrics", Enabled: true, Priority: 100}, // Lowest priority
	}

	manager := NewManager(cfg, logger)

	// 1. Discover
	err := manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(manager.GetEnabledPlugins()) != 3 {
		t.Errorf("Expected 3 enabled plugins, got %d", len(manager.GetEnabledPlugins()))
	}

	// 2. Initialize
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	ctx := Context{Logger: logger}

	err = manager.Initialize(app, ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify Init was called on all plugins
	if !plugin1.initCalled || !plugin2.initCalled || !plugin3.initCalled {
		t.Error("Not all plugins were initialized")
	}

	// Verify resources were allocated during Init
	if len(plugin1.GetResources()) < 2 {
		t.Error("Plugin1 resources not allocated during Init")
	}

	// 3. Ready
	err = manager.Ready(ctx)
	if err != nil {
		t.Fatalf("Ready failed: %v", err)
	}

	// Verify Ready was called on all plugins
	if !plugin1.readyCalled || !plugin2.readyCalled || !plugin3.readyCalled {
		t.Error("Not all plugins are ready")
	}

	// Verify additional resources were allocated during Ready
	if len(plugin1.GetResources()) < 3 {
		t.Error("Plugin1 resources not allocated during Ready")
	}

	// Verify manager state
	if !manager.IsInitialized() {
		t.Error("Manager should be initialized")
	}
	if !manager.IsReady() {
		t.Error("Manager should be ready")
	}

	// 4. Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = manager.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify Shutdown was called on all plugins
	if !plugin1.shutdownCalled || !plugin2.shutdownCalled || !plugin3.shutdownCalled {
		t.Error("Not all plugins were shut down")
	}

	// Verify resources were cleaned up
	if len(plugin1.GetResources()) != 0 {
		t.Error("Plugin1 resources not cleaned up during Shutdown")
	}
	if len(plugin2.GetResources()) != 0 {
		t.Error("Plugin2 resources not cleaned up during Shutdown")
	}
	if len(plugin3.GetResources()) != 0 {
		t.Error("Plugin3 resources not cleaned up during Shutdown")
	}
}

// orderTrackingPlugin tracks initialization order via a shared slice
type orderTrackingPlugin struct {
	name       string
	initOrder  *[]string
	orderMutex *sync.Mutex
}

func (o *orderTrackingPlugin) Name() string {
	return o.name
}

func (o *orderTrackingPlugin) Init(ctx Context) error {
	o.orderMutex.Lock()
	defer o.orderMutex.Unlock()
	*o.initOrder = append(*o.initOrder, o.name)
	return nil
}

func TestPluginLifecycle_PriorityOrdering(t *testing.T) {
	logger := &mockLogger{}
	initOrder := []string{}
	mu := sync.Mutex{}

	// Create plugins that track initialization order
	plugin1 := &orderTrackingPlugin{name: "high-priority", initOrder: &initOrder, orderMutex: &mu}
	plugin2 := &orderTrackingPlugin{name: "medium-priority", initOrder: &initOrder, orderMutex: &mu}
	plugin3 := &orderTrackingPlugin{name: "low-priority", initOrder: &initOrder, orderMutex: &mu}

	oldRegistry := registry
	defer func() { registry = oldRegistry }()
	registry = []Plugin{plugin1, plugin2, plugin3}

	cfg := DefaultConfig()
	cfg.Registry.Core = []PluginConfig{
		{Name: "high-priority", Enabled: true, Priority: 10},
		{Name: "medium-priority", Enabled: true, Priority: 50},
		{Name: "low-priority", Enabled: true, Priority: 100},
	}

	manager := NewManager(cfg, logger)
	err := manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Call Initialize to trigger Init in priority order
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	ctx := Context{Logger: logger}
	err = manager.Initialize(app, ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify initialization order (should be based on priority)
	expectedOrder := []string{"high-priority", "medium-priority", "low-priority"}

	mu.Lock()
	defer mu.Unlock()

	if len(initOrder) != len(expectedOrder) {
		t.Errorf("Expected %d plugins initialized, got %d", len(expectedOrder), len(initOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(initOrder) {
			break
		}
		if initOrder[i] != expected {
			t.Errorf("Expected plugin at position %d to be %s, got %s",
				i, expected, initOrder[i])
		}
	}
}

func TestPluginLifecycle_FailureHandling(t *testing.T) {
	logger := &mockLogger{}

	tests := []struct {
		name          string
		failurePolicy FailurePolicy
		expectError   bool
	}{
		{
			name:          "halt on failure",
			failurePolicy: FailurePolicyHalt,
			expectError:   true,
		},
		{
			name:          "continue on failure",
			failurePolicy: FailurePolicyContinue,
			expectError:   false,
		},
		{
			name:          "warn on failure",
			failurePolicy: FailurePolicyWarn,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a plugin that fails during Init
			failingPlugin := &mockPlugin{
				name:      "failing-plugin",
				initError: func() error { return nil }(), // Will be set below
			}
			failingPlugin.initError = func() error {
				return fiber.NewError(500, "init failed")
			}()

			oldRegistry := registry
			defer func() { registry = oldRegistry }()
			registry = []Plugin{failingPlugin}

			cfg := DefaultConfig()
			cfg.Lifecycle.FailurePolicy = tt.failurePolicy
			cfg.Registry.Core = []PluginConfig{
				{Name: "failing-plugin", Enabled: true, Priority: 10},
			}

			manager := NewManager(cfg, logger)
			err := manager.Discover()
			if err != nil {
				t.Fatalf("Discover failed: %v", err)
			}

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			ctx := Context{Logger: logger}

			err = manager.Initialize(app, ctx)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check logs based on policy
			switch tt.failurePolicy {
			case FailurePolicyWarn:
				if len(logger.warnLogs) == 0 {
					t.Error("Expected warning log but found none")
				}
			case FailurePolicyContinue:
				if len(logger.errorLogs) == 0 {
					t.Error("Expected error log but found none")
				}
			}
		})
	}
}

// slowPlugin simulates a plugin that takes a long time to initialize
type slowPlugin struct {
	name  string
	delay time.Duration
}

func (s *slowPlugin) Name() string {
	return s.name
}

func (s *slowPlugin) Init(ctx Context) error {
	time.Sleep(s.delay)
	return nil
}

func TestPluginLifecycle_TimeoutHandling(t *testing.T) {
	logger := &mockLogger{}

	// Create a slow plugin that takes longer than the timeout
	plugin := &slowPlugin{
		name:  "slow-plugin",
		delay: 100 * time.Millisecond,
	}

	oldRegistry := registry
	defer func() { registry = oldRegistry }()
	registry = []Plugin{plugin}

	cfg := DefaultConfig()
	cfg.Lifecycle.InitTimeout = 50 * time.Millisecond   // Timeout before plugin completes
	cfg.Lifecycle.FailurePolicy = FailurePolicyContinue // Don't halt on timeout
	cfg.Registry.Core = []PluginConfig{
		{Name: "slow-plugin", Enabled: true, Priority: 10},
	}

	manager := NewManager(cfg, logger)
	err := manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	ctx := Context{Logger: logger}

	start := time.Now()
	_ = manager.Initialize(app, ctx)
	duration := time.Since(start)

	// Verify timeout was enforced (should not wait for full delay)
	if duration > 100*time.Millisecond {
		t.Errorf("Timeout not enforced: took %v", duration)
	}

	// Should have logged the timeout error
	if len(logger.errorLogs) == 0 && len(logger.warnLogs) == 0 {
		t.Error("Expected timeout to be logged")
	}
}

func TestPluginLifecycle_DisabledPlugins(t *testing.T) {
	logger := &mockLogger{}

	plugin1 := &mockPlugin{name: "enabled-plugin"}
	plugin2 := &mockPlugin{name: "disabled-plugin"}

	oldRegistry := registry
	defer func() { registry = oldRegistry }()
	registry = []Plugin{plugin1, plugin2}

	cfg := DefaultConfig()
	cfg.Registry.Core = []PluginConfig{
		{Name: "enabled-plugin", Enabled: true, Priority: 10},
		{Name: "disabled-plugin", Enabled: false, Priority: 20},
	}

	manager := NewManager(cfg, logger)
	err := manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	ctx := Context{Logger: logger}

	err = manager.Initialize(app, ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = manager.Ready(ctx)
	if err != nil {
		t.Fatalf("Ready failed: %v", err)
	}

	// Verify only enabled plugin was initialized
	if !plugin1.initCalled {
		t.Error("Enabled plugin should be initialized")
	}

	if plugin2.initCalled {
		t.Error("Disabled plugin should not be initialized")
	}

	if !plugin1.readyCalled {
		t.Error("Enabled plugin should be ready")
	}

	if plugin2.readyCalled {
		t.Error("Disabled plugin should not be ready")
	}
}

func TestPluginLifecycle_ConcurrentAccess(t *testing.T) {
	logger := &mockLogger{}

	// Test that manager is safe for concurrent access
	plugin := &mockPlugin{name: "concurrent-plugin"}

	oldRegistry := registry
	defer func() { registry = oldRegistry }()
	registry = []Plugin{plugin}

	cfg := DefaultConfig()
	cfg.Registry.Core = []PluginConfig{
		{Name: "concurrent-plugin", Enabled: true, Priority: 10},
	}

	manager := NewManager(cfg, logger)
	err := manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			plugins := manager.GetPlugins()
			if len(plugins) != 1 {
				errors <- fiber.NewError(500, "unexpected plugin count")
			}

			enabled := manager.GetEnabledPlugins()
			if len(enabled) != 1 {
				errors <- fiber.NewError(500, "unexpected enabled count")
			}

			if !manager.IsInitialized() && !manager.IsReady() {
				// State is fine - not yet initialized
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestPluginLifecycle_EnvironmentOverrides(t *testing.T) {
	logger := &mockLogger{}

	plugin1 := &mockPlugin{name: "test-plugin"}
	plugin2 := &mockPlugin{name: "prod-only-plugin"}

	oldRegistry := registry
	defer func() { registry = oldRegistry }()
	registry = []Plugin{plugin1, plugin2}

	cfg := DefaultConfig()
	cfg.Registry.Core = []PluginConfig{
		{Name: "test-plugin", Enabled: true, Priority: 10, Config: map[string]interface{}{
			"mode": "production",
		}},
		{Name: "prod-only-plugin", Enabled: true, Priority: 20},
	}

	// Add environment overrides
	devDisabled := false
	cfg.Environments = map[string]EnvironmentOverrides{
		"development": {
			Overrides: []PluginOverride{
				{
					Plugin: "test-plugin",
					Config: map[string]interface{}{
						"mode": "development",
					},
				},
				{
					Plugin:  "prod-only-plugin",
					Enabled: &devDisabled,
				},
			},
		},
	}

	// Set environment
	t.Setenv("PROXYND_ENV", "development")

	manager := NewManager(cfg, logger)
	err := manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Verify environment overrides were applied
	plugins := manager.GetPlugins()

	var testPlugin, prodPlugin *PluginConfig
	for i := range plugins {
		if plugins[i].Name == "test-plugin" {
			testPlugin = &plugins[i]
		}
		if plugins[i].Name == "prod-only-plugin" {
			prodPlugin = &plugins[i]
		}
	}

	if testPlugin == nil || prodPlugin == nil {
		t.Fatal("Could not find test plugins")
	}

	// Check config was overridden
	if mode, ok := testPlugin.Config["mode"]; !ok || mode != "development" {
		t.Errorf("Expected mode=development, got %v", mode)
	}

	// Check enabled flag was overridden
	if prodPlugin.Enabled {
		t.Error("prod-only-plugin should be disabled in development")
	}
}
