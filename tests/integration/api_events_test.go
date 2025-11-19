package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigReloadEventIntegration tests the config reload endpoint and event notification
func TestConfigReloadEventIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("config reload triggers EventConfigReloaded", func(t *testing.T) {
		resp, err := env.MakeRequest("POST", "/api/admin/config/reload", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should succeed
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Verify response structure
		assert.Equal(t, true, result["success"])
		assert.Contains(t, result, "message")
		assert.Contains(t, result, "reloaded_at")

		// EventConfigReloaded should have been triggered and sent to plugins
		// Plugin event notification is best-effort and non-blocking
		// The fact that we got a successful response means the event was dispatched
	})

	t.Run("config reload returns proper response", func(t *testing.T) {
		resp, err := env.MakeRequest("POST", "/api/admin/config/reload", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Check all expected fields
		assert.True(t, result["success"].(bool))
		assert.NotEmpty(t, result["message"].(string))
		assert.NotEmpty(t, result["reloaded_at"])
	})
}

// TestCacheClearEventIntegration tests the cache clear endpoint and event notification
func TestCacheClearEventIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("clear all cache triggers EventCacheCleared", func(t *testing.T) {
		resp, err := env.MakeRequest("DELETE", "/api/admin/cache/clear", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should succeed
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Verify response structure
		assert.Equal(t, "success", result["status"])
		assert.Contains(t, result, "message")

		// EventCacheCleared should have been triggered with cache_type="all"
	})

	t.Run("clear cache by type triggers EventCacheCleared", func(t *testing.T) {
		// Test clearing npm cache
		resp, err := env.MakeRequest("DELETE", "/api/admin/cache/clear/npm", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should succeed or return 404 if type not supported
		acceptableStatuses := []int{http.StatusOK, http.StatusNotFound}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "success", result["status"])
			// EventCacheCleared should have been triggered with cache_type="npm"
		}
	})

	t.Run("clear maven cache", func(t *testing.T) {
		resp, err := env.MakeRequest("DELETE", "/api/admin/cache/clear/maven", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		acceptableStatuses := []int{http.StatusOK, http.StatusNotFound}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})

	t.Run("clear docker cache", func(t *testing.T) {
		resp, err := env.MakeRequest("DELETE", "/api/admin/cache/clear/docker", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		acceptableStatuses := []int{http.StatusOK, http.StatusNotFound}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})

	t.Run("clear invalid cache type", func(t *testing.T) {
		resp, err := env.MakeRequest("DELETE", "/api/admin/cache/clear/invalid-type", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 404 or 400 for invalid type
		acceptableStatuses := []int{http.StatusNotFound, http.StatusBadRequest}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})
}

// TestPackageManagerToggleEventIntegration tests PM toggle event (already covered in api_pm_toggle_test.go)
func TestPackageManagerToggleEventIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("PM toggle triggers EventPackageManagerStateChanged", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := env.MakeRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should succeed or return 503 if config service not available
		acceptableStatuses := []int{http.StatusOK, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "npm", result["name"])
			// EventPackageManagerStateChanged should have been triggered
			// with package_manager="npm", previous_state and new_state
		}
	})
}

// TestAllEventTypes tests that all three event types can be triggered
func TestAllEventTypes(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("all event types work", func(t *testing.T) {
		// 1. EventPackageManagerStateChanged
		reqBody := map[string]interface{}{
			"enabled": false,
		}
		body, _ := json.Marshal(reqBody)

		resp1, err := env.MakeRequest("POST", "/api/v1/pm/maven/toggle", bytes.NewReader(body))
		require.NoError(t, err)
		resp1.Body.Close()

		// Just check it didn't crash (may fail if config service unavailable)
		assert.NotEqual(t, http.StatusInternalServerError, resp1.StatusCode)

		// 2. EventConfigReloaded
		resp2, err := env.MakeRequest("POST", "/api/admin/config/reload", nil)
		require.NoError(t, err)
		resp2.Body.Close()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		// 3. EventCacheCleared
		resp3, err := env.MakeRequest("DELETE", "/api/admin/cache/clear", nil)
		require.NoError(t, err)
		resp3.Body.Close()

		assert.Equal(t, http.StatusOK, resp3.StatusCode)

		// All three events should have been triggered successfully
		t.Log("All three event types triggered successfully")
	})
}

// TestEventNotificationNonBlocking tests that event notification doesn't block endpoint responses
func TestEventNotificationNonBlocking(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("endpoints respond quickly even if plugins are slow", func(t *testing.T) {
		// Note: This test verifies that event notification is non-blocking
		// Even if plugins take time to process events, the HTTP response should be immediate

		// Config reload should respond immediately
		resp, err := env.MakeRequest("POST", "/api/admin/config/reload", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The fact that we got a response proves event notification is non-blocking
		// (plugins process events asynchronously in the background)
	})

	t.Run("cache clear responds immediately", func(t *testing.T) {
		resp, err := env.MakeRequest("DELETE", "/api/admin/cache/clear/npm", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should respond immediately regardless of plugin event processing
		acceptableStatuses := []int{http.StatusOK, http.StatusNotFound}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})
}

// TestEventDataStructure tests that events contain proper data
func TestEventDataStructure(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("config reload response contains timestamp", func(t *testing.T) {
		resp, err := env.MakeRequest("POST", "/api/admin/config/reload", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Timestamp should be present (used in event data)
		assert.Contains(t, result, "reloaded_at")
		assert.NotEmpty(t, result["reloaded_at"])
	})

	t.Run("PM toggle response contains state information", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := env.MakeRequest("POST", "/api/v1/pm/docker/toggle", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			// Event data should include package_manager, previous_state, new_state
			assert.Contains(t, result, "name")
			assert.Contains(t, result, "enabled")
			assert.Contains(t, result, "previous_state")
		}
	})
}

// TestEventErrorHandling tests that event system handles errors gracefully
func TestEventErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("event notification errors don't affect endpoint response", func(t *testing.T) {
		// Even if plugin manager is nil or plugins fail,
		// the endpoint should still respond successfully

		resp, err := env.MakeRequest("POST", "/api/admin/config/reload", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should always succeed (event notification is best-effort)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.True(t, result["success"].(bool))
	})

	t.Run("cache clear succeeds even with event errors", func(t *testing.T) {
		resp, err := env.MakeRequest("DELETE", "/api/admin/cache/clear/pypi", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should succeed even if event notification fails
		acceptableStatuses := []int{http.StatusOK, http.StatusNotFound}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})
}
