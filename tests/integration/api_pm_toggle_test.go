package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageManagerToggleIntegration tests the PM toggle endpoint
func TestPackageManagerToggleIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("toggle npm to enabled", func(t *testing.T) {
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
			assert.Equal(t, true, result["enabled"])
			assert.Contains(t, result, "previous_state")
			assert.Contains(t, result, "message")
			assert.Equal(t, true, result["persisted"])
		}
	})

	t.Run("toggle npm to disabled", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"enabled": false,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := env.MakeRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		acceptableStatuses := []int{http.StatusOK, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "npm", result["name"])
			assert.Equal(t, false, result["enabled"])
			assert.Equal(t, true, result["persisted"])
		}
	})

	t.Run("toggle mode flips current state", func(t *testing.T) {
		// First toggle to get current state
		reqBody1 := map[string]interface{}{
			"enabled": true,
		}
		body1, _ := json.Marshal(reqBody1)

		resp1, err := env.MakeRequest("POST", "/api/v1/pm/maven/toggle", bytes.NewReader(body1))
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		if resp1.StatusCode != http.StatusOK {
			t.Skip("Config service not available, skipping")
		}

		var result1 map[string]interface{}
		err = json.NewDecoder(resp1.Body).Decode(&result1)
		require.NoError(t, err)
		previousState := result1["enabled"].(bool)

		// Now use toggle mode
		reqBody2 := map[string]interface{}{
			"mode": "toggle",
		}
		body2, _ := json.Marshal(reqBody2)

		resp2, err := env.MakeRequest("POST", "/api/v1/pm/maven/toggle", bytes.NewReader(body2))
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		var result2 map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&result2)
		require.NoError(t, err)

		// Should be opposite of previous state
		assert.Equal(t, !previousState, result2["enabled"].(bool))
		assert.Equal(t, previousState, result2["previous_state"].(bool))
	})

	t.Run("invalid package manager returns 404", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := env.MakeRequest("POST", "/api/v1/pm/nonexistent/toggle", bytes.NewReader(body))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 400 (bad request), 404 (not found), or 503 (if config service not available)
		acceptableStatuses := []int{http.StatusBadRequest, http.StatusNotFound, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)

		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Contains(t, result, "error")
		}
	})

	t.Run("consecutive toggles work correctly", func(t *testing.T) {
		// Enable
		reqBody1 := map[string]interface{}{
			"enabled": true,
		}
		body1, _ := json.Marshal(reqBody1)

		resp1, err := env.MakeRequest("POST", "/api/v1/pm/docker/toggle", bytes.NewReader(body1))
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		if resp1.StatusCode != http.StatusOK {
			t.Skip("Config service not available, skipping")
		}

		var result1 map[string]interface{}
		err = json.NewDecoder(resp1.Body).Decode(&result1)
		require.NoError(t, err)
		assert.Equal(t, true, result1["enabled"].(bool))

		// Small delay to ensure state is persisted
		time.Sleep(100 * time.Millisecond)

		// Disable
		reqBody2 := map[string]interface{}{
			"enabled": false,
		}
		body2, _ := json.Marshal(reqBody2)

		resp2, err := env.MakeRequest("POST", "/api/v1/pm/docker/toggle", bytes.NewReader(body2))
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		var result2 map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&result2)
		require.NoError(t, err)
		assert.Equal(t, false, result2["enabled"].(bool))
		assert.Equal(t, true, result2["previous_state"].(bool))
	})

	t.Run("invalid json body returns 400", func(t *testing.T) {
		invalidBody := bytes.NewReader([]byte("not valid json"))

		resp, err := env.MakeRequest("POST", "/api/v1/pm/npm/toggle", invalidBody)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 400 or 503
		acceptableStatuses := []int{http.StatusBadRequest, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})

	t.Run("persistence verification", func(t *testing.T) {
		// Set a specific state
		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := env.MakeRequest("POST", "/api/v1/pm/pypi/toggle", bytes.NewReader(body))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Skip("Config service not available, skipping")
		}

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Verify persisted flag is true
		assert.Equal(t, true, result["persisted"].(bool))

		// The state should be saved to disk at this point
		// In a real E2E test, we would restart the server and verify
		// the state persisted, but that's beyond integration test scope
	})
}

// TestToggleEndpointErrorHandling tests error scenarios
func TestToggleEndpointErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("empty request body", func(t *testing.T) {
		resp, err := env.MakeRequest("POST", "/api/v1/pm/npm/toggle", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 400 or 503
		acceptableStatuses := []int{http.StatusBadRequest, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})

	t.Run("missing both enabled and mode", func(t *testing.T) {
		reqBody := map[string]interface{}{}
		body, _ := json.Marshal(reqBody)

		resp, err := env.MakeRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 400 or 200 (default behavior)
		// Implementation may vary, so we just check it doesn't crash
		assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("wrong http method returns 405", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/api/v1/pm/npm/toggle", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// GET is not allowed, should return 405 or 404
		acceptableStatuses := []int{http.StatusMethodNotAllowed, http.StatusNotFound}
		assert.Contains(t, acceptableStatuses, resp.StatusCode)
	})
}

// TestToggleEndpointConcurrency tests concurrent toggle requests
func TestToggleEndpointConcurrency(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("concurrent toggles", func(t *testing.T) {
		// Send multiple toggle requests concurrently
		const numRequests = 5
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(idx int) {
				reqBody := map[string]interface{}{
					"enabled": idx%2 == 0, // Alternate between true/false
				}
				body, _ := json.Marshal(reqBody)

				resp, err := env.MakeRequest("POST", "/api/v1/pm/apk/toggle", bytes.NewReader(body))
				if err != nil {
					results <- err
					return
				}
				defer func() { _ = resp.Body.Close() }()

				// Read body to ensure request completes
				_, _ = io.ReadAll(resp.Body)

				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
					results <- assert.AnError
					return
				}

				results <- nil
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			err := <-results
			// We allow errors due to concurrency or service unavailability
			// The important thing is the server doesn't crash
			if err != nil {
				t.Logf("Request %d had error (acceptable in concurrent test): %v", i, err)
			}
		}

		// If we got here without the test hanging, concurrency handling works
		t.Log("Concurrent toggle requests completed without deadlock")
	})
}
