package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/adapters/http/fiber/handlers"
	"proxynd/internal/adapters/http/fiber/routers"
	"proxynd/internal/logging"
	"proxynd/plugins"
)

func TestPluginHealthEndpoint_WithoutPluginManager(t *testing.T) {
	// Create a minimal Fiber app without plugin manager
	app := fiber.New()

	// Register the plugin health endpoint
	pluginHealthHandler := handlers.NewPluginHealthHandler()
	app.Get("/api/v1/plugins/health", pluginHealthHandler.GetPluginHealth)

	// Make request without plugin manager in context
	req, err := http.NewRequest("GET", "/api/v1/plugins/health", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 503 Service Unavailable
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	// Parse response
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, false, body["enabled"])
	assert.Contains(t, body["message"], "not available")
}

func TestPluginHealthEndpoint_WithUninitializedManager(t *testing.T) {
	// Create a Fiber app with uninitialized plugin manager
	app := fiber.New()

	// Create plugin manager (not initialized)
	cfg := plugins.DefaultConfig()
	cfg.Enabled = true
	manager := plugins.NewManager(cfg, &testPluginLogger{})

	// Add middleware to inject plugin manager
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("pluginManager", manager)
		return c.Next()
	})

	// Register the plugin health endpoint
	pluginHealthHandler := handlers.NewPluginHealthHandler()
	app.Get("/api/v1/plugins/health", pluginHealthHandler.GetPluginHealth)

	// Make request
	req, err := http.NewRequest("GET", "/api/v1/plugins/health", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	// Verify response structure
	assert.Equal(t, true, body["enabled"])
	assert.Equal(t, false, body["initialized"])
	assert.Equal(t, false, body["ready"])
	assert.NotNil(t, body["plugins"])
	assert.NotNil(t, body["summary"])
	assert.NotNil(t, body["event_metrics"])
	assert.NotNil(t, body["timestamp"])
}

func TestPluginHealthEndpoint_WithInitializedManager(t *testing.T) {
	// Create Fiber app
	app := fiber.New()

	// Create and initialize plugin manager
	cfg := plugins.DefaultConfig()
	cfg.Enabled = true
	cfg.Registry.Core = []plugins.PluginConfig{
		{
			Name:     "test_plugin",
			Enabled:  true,
			Priority: 100,
			Config:   map[string]interface{}{},
		},
	}

	manager := plugins.NewManager(cfg, &testPluginLogger{})

	// Discover plugins
	err := manager.Discover()
	require.NoError(t, err)

	// Add middleware to inject plugin manager
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("pluginManager", manager)
		return c.Next()
	})

	// Register the plugin health endpoint
	pluginHealthHandler := handlers.NewPluginHealthHandler()
	app.Get("/api/v1/plugins/health", pluginHealthHandler.GetPluginHealth)

	// Make request
	req, err := http.NewRequest("GET", "/api/v1/plugins/health", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, true, body["enabled"])
	assert.GreaterOrEqual(t, int(body["total_plugins"].(float64)), 0)

	// Verify plugins array exists
	pluginsArray, ok := body["plugins"].([]interface{})
	assert.True(t, ok)
	assert.NotNil(t, pluginsArray)

	// Verify summary structure
	summary, ok := body["summary"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, summary["total_errors"])
	assert.NotNil(t, summary["avg_init_time_ms"])
	assert.NotNil(t, summary["avg_ready_time_ms"])

	// Verify event metrics
	eventMetrics, ok := body["event_metrics"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, eventMetrics["metrics_enabled"])
}

func TestPluginHealthEndpoint_ResponseStructure(t *testing.T) {
	// Create Fiber app with plugin manager
	app := fiber.New()

	cfg := plugins.DefaultConfig()
	cfg.Enabled = true
	manager := plugins.NewManager(cfg, &testPluginLogger{})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("pluginManager", manager)
		return c.Next()
	})

	pluginHealthHandler := handlers.NewPluginHealthHandler()
	app.Get("/api/v1/plugins/health", pluginHealthHandler.GetPluginHealth)

	// Make request
	req, err := http.NewRequest("GET", "/api/v1/plugins/health", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse and verify JSON structure
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	// Required top-level fields
	requiredFields := []string{
		"enabled",
		"initialized",
		"ready",
		"total_plugins",
		"enabled_plugins",
		"ready_plugins",
		"failed_plugins",
		"plugins",
		"summary",
		"event_metrics",
		"timestamp",
	}

	for _, field := range requiredFields {
		assert.Contains(t, body, field, "Missing required field: %s", field)
	}

	// Verify summary fields
	summary := body["summary"].(map[string]interface{})
	summaryFields := []string{
		"total_errors",
		"total_state_changes",
		"avg_init_time_ms",
		"avg_ready_time_ms",
		"total_init_time_ms",
		"total_ready_time_ms",
		"total_shutdown_time_ms",
	}

	for _, field := range summaryFields {
		assert.Contains(t, summary, field, "Missing summary field: %s", field)
	}

	// Verify event_metrics fields
	eventMetrics := body["event_metrics"].(map[string]interface{})
	assert.Contains(t, eventMetrics, "metrics_enabled")
	assert.Contains(t, eventMetrics, "metrics_endpoint")
	assert.Contains(t, eventMetrics, "available_metrics")
}

func TestPluginHealthEndpoint_Integration(t *testing.T) {
	// Use the standard integration test setup
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Note: The standard integration test environment doesn't include
	// the API v1 router by default. We need to add it manually for this test.

	// Add API v1 routes to the server
	routers.SetupAPIv1Routes(env.ProxyServer, nil)

	// Create and setup plugin manager
	cfg := plugins.DefaultConfig()
	cfg.Enabled = true
	manager := plugins.NewManager(cfg, &testPluginLogger{})

	// Discover plugins
	err := manager.Discover()
	require.NoError(t, err)

	// Add middleware to inject plugin manager into all requests
	env.ProxyServer.Use(func(c *fiber.Ctx) error {
		c.Locals("pluginManager", manager)
		return c.Next()
	})

	// Make request to health endpoint
	resp, err := env.MakeRequest("GET", "/api/v1/plugins/health", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify response
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, true, body["enabled"])
	assert.NotNil(t, body["plugins"])
	assert.NotNil(t, body["summary"])
	assert.NotNil(t, body["timestamp"])
}

// testPluginLogger is a simple logger implementation for testing
type testPluginLogger struct{}

func (l *testPluginLogger) Debug(msg string, fields ...interface{}) {
	logging.GetLogger().Debug(msg, fields...)
}

func (l *testPluginLogger) Info(msg string, fields ...interface{}) {
	logging.GetLogger().Info(msg, fields...)
}

func (l *testPluginLogger) Warn(msg string, fields ...interface{}) {
	logging.GetLogger().Warn(msg, fields...)
}

func (l *testPluginLogger) Error(msg string, fields ...interface{}) {
	logging.GetLogger().Error(msg, fields...)
}

func (l *testPluginLogger) Fatal(msg string, fields ...interface{}) {
	logging.GetLogger().Fatal(msg, fields...)
}

func (l *testPluginLogger) With(fields ...interface{}) plugins.Logger {
	return l
}

// Ensure testPluginLogger implements plugins.Logger interface
var _ plugins.Logger = (*testPluginLogger)(nil)
