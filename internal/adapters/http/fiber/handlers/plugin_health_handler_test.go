package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/plugins"
)

func TestPluginHealthHandler_GetPluginHealth(t *testing.T) {
	t.Run("plugin manager not available - returns service unavailable", func(t *testing.T) {
		// Create test app without plugin manager
		app := fiber.New()
		handler := NewPluginHealthHandler()
		app.Get("/health", handler.GetPluginHealth)

		// Make request
		req := httptest.NewRequest("GET", "/health", nil)
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
	})

	t.Run("plugin manager available but not initialized - returns status", func(t *testing.T) {
		// Create test app with uninitialized plugin manager
		app := fiber.New()
		handler := NewPluginHealthHandler()

		// Create plugin manager (not initialized)
		cfg := plugins.DefaultConfig()
		cfg.Enabled = true
		manager := plugins.NewManager(cfg, &testLogger{})

		// Set up route with plugin manager in locals
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("pluginManager", manager)
			return c.Next()
		})
		app.Get("/health", handler.GetPluginHealth)

		// Make request
		req := httptest.NewRequest("GET", "/health", nil)
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
	})

	t.Run("plugin manager with plugins - returns plugin details", func(t *testing.T) {
		// Create test app with plugin manager
		app := fiber.New()
		handler := NewPluginHealthHandler()

		// Create plugin manager with test config
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

		manager := plugins.NewManager(cfg, &testLogger{})

		// Discover plugins (will register them)
		err := manager.Discover()
		require.NoError(t, err)

		// Set up route with plugin manager in locals
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("pluginManager", manager)
			return c.Next()
		})
		app.Get("/health", handler.GetPluginHealth)

		// Make request
		req := httptest.NewRequest("GET", "/health", nil)
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

		// Verify event metrics
		eventMetrics, ok := body["event_metrics"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, true, eventMetrics["metrics_enabled"])
	})
}

// testLogger is a simple logger for testing
type testLogger struct{}

func (l *testLogger) Debug(msg string, fields ...interface{}) {}
func (l *testLogger) Info(msg string, fields ...interface{})  {}
func (l *testLogger) Warn(msg string, fields ...interface{})  {}
func (l *testLogger) Error(msg string, fields ...interface{}) {}
func (l *testLogger) Fatal(msg string, fields ...interface{}) {}
func (l *testLogger) With(fields ...interface{}) plugins.Logger {
	return l
}

// Ensure testLogger implements plugins.Logger interface
var _ plugins.Logger = (*testLogger)(nil)

func TestPluginHealthResponse_JSONSerialization(t *testing.T) {
	t.Run("serialize plugin health response with all fields", func(t *testing.T) {
		now := time.Now()
		lastErrorTime := now.Format(time.RFC3339)

		response := PluginHealthResponse{
			Name:            "test_plugin",
			Enabled:         true,
			Status:          "ready",
			Priority:        100,
			ErrorCount:      5,
			LastError:       "test error",
			LastErrorTime:   &lastErrorTime,
			InitTimeMs:      10,
			ReadyTimeMs:     5,
			StateChanges:    3,
			LastStateChange: now,
		}

		// Serialize to JSON
		data, err := json.Marshal(response)
		require.NoError(t, err)

		// Deserialize from JSON
		var decoded PluginHealthResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Verify all fields
		assert.Equal(t, response.Name, decoded.Name)
		assert.Equal(t, response.Enabled, decoded.Enabled)
		assert.Equal(t, response.Status, decoded.Status)
		assert.Equal(t, response.Priority, decoded.Priority)
		assert.Equal(t, response.ErrorCount, decoded.ErrorCount)
		assert.Equal(t, response.LastError, decoded.LastError)
		assert.NotNil(t, decoded.LastErrorTime)
		assert.Equal(t, response.InitTimeMs, decoded.InitTimeMs)
		assert.Equal(t, response.ReadyTimeMs, decoded.ReadyTimeMs)
		assert.Equal(t, response.StateChanges, decoded.StateChanges)
	})

	t.Run("serialize plugin health response without last error", func(t *testing.T) {
		response := PluginHealthResponse{
			Name:            "test_plugin",
			Enabled:         true,
			Status:          "ready",
			Priority:        100,
			ErrorCount:      0,
			LastError:       "",
			LastErrorTime:   nil,
			InitTimeMs:      10,
			ReadyTimeMs:     5,
			StateChanges:    2,
			LastStateChange: time.Now(),
		}

		// Serialize to JSON
		data, err := json.Marshal(response)
		require.NoError(t, err)

		// Verify JSON doesn't include last_error and last_error_time
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// last_error should be empty string (not omitted due to no omitempty on LastError)
		assert.Equal(t, "", jsonMap["last_error"])

		// last_error_time should be nil (omitted due to omitempty)
		assert.Nil(t, jsonMap["last_error_time"])
	})
}
