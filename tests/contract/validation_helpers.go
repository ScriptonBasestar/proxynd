//go:build contract
// +build contract

package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateSuccessResponse validates standard success response schema
func validateSuccessResponse(t *testing.T, body map[string]interface{}) {
	assert.Contains(t, body, "success", "Response must contain 'success' field")
	assert.Contains(t, body, "data", "Response must contain 'data' field")
	assert.Contains(t, body, "metadata", "Response must contain 'metadata' field")

	success, ok := body["success"].(bool)
	assert.True(t, ok, "'success' field must be a boolean")
	assert.True(t, success, "'success' should be true for successful responses")

	// Validate metadata structure
	metadata, ok := body["metadata"].(map[string]interface{})
	assert.True(t, ok, "'metadata' must be an object")
	assert.Contains(t, metadata, "timestamp", "Metadata must contain 'timestamp'")
}

// validatePaginatedListResponse validates paginated list response schema
func validatePaginatedListResponse(t *testing.T, body map[string]interface{}) {
	validateSuccessResponse(t, body)

	assert.Contains(t, body, "pagination", "Paginated response must contain 'pagination' field")

	pagination, ok := body["pagination"].(map[string]interface{})
	assert.True(t, ok, "'pagination' must be an object")

	// Validate pagination fields
	assert.Contains(t, pagination, "page", "Pagination must contain 'page'")
	assert.Contains(t, pagination, "per_page", "Pagination must contain 'per_page'")
	assert.Contains(t, pagination, "total", "Pagination must contain 'total'")
	assert.Contains(t, pagination, "total_pages", "Pagination must contain 'total_pages'")
	assert.Contains(t, pagination, "has_next", "Pagination must contain 'has_next'")
	assert.Contains(t, pagination, "has_prev", "Pagination must contain 'has_prev'")

	// Validate data is an array
	data, ok := body["data"].([]interface{})
	assert.True(t, ok, "'data' must be an array for paginated responses, got %T", body["data"])

	// Validate pagination constraints
	perPage, _ := pagination["per_page"].(float64)
	assert.LessOrEqual(t, len(data), int(perPage),
		"Data length should not exceed per_page value")
}

// validateErrorResponse validates standard error response schema
func validateErrorResponse(t *testing.T, body map[string]interface{}) {
	assert.Contains(t, body, "success", "Error response must contain 'success' field")
	success, _ := body["success"].(bool)
	assert.False(t, success, "'success' should be false for error responses")

	assert.Contains(t, body, "error", "Error response must contain 'error' field")
	errorObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok, "'error' must be an object")
	assert.Contains(t, errorObj, "code", "Error object must contain 'code'")
	assert.Contains(t, errorObj, "message", "Error object must contain 'message'")
}
