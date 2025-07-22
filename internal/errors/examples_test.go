package errors_test

import (
	"fmt"
	"testing"

	"proxynd/internal/errors"
)

// TestStructuredErrorUsage demonstrates how to use the structured error system
func TestStructuredErrorUsage(t *testing.T) {
	// Example of creating a cache error
	cacheErr := errors.NewError(errors.ErrCodeCacheWrite, "failed to write to cache").
		WithDomain("cache").
		WithCause(fmt.Errorf("disk full")).
		WithDetails(map[string]interface{}{
			"key":      "test-key",
			"size":     1024,
			"location": "/tmp/cache",
		}).
		Build()

	// Test error message formatting
	expected := "[cache] CACHE_WRITE: failed to write to cache (caused by: disk full)"
	if cacheErr.Error() != expected {
		t.Errorf("Expected: %s, Got: %s", expected, cacheErr.Error())
	}

	// Test error unwrapping
	var domainErr *errors.DomainError
	if domainErr = cacheErr; domainErr == nil {
		t.Error("Should be able to cast to DomainError")
	}

	if domainErr.Code != errors.ErrCodeCacheWrite {
		t.Errorf("Expected code: %s, Got: %s", errors.ErrCodeCacheWrite, domainErr.Code)
	}
}

// TestProxyErrorExample shows proxy error usage
func TestProxyErrorExample(t *testing.T) {
	proxyErr := errors.NewError(errors.ErrCodeProxyUpstreamTimeout, "upstream request timed out").
		WithDomain("proxy").
		WithCause(fmt.Errorf("context deadline exceeded")).
		WithDetails(map[string]interface{}{
			"upstream": "http://example.com",
			"timeout":  "30s",
			"proxy":    "apt",
		}).
		Build()

	// Verify the error contains the right information
	if proxyErr.Domain != "proxy" {
		t.Errorf("Expected domain: proxy, Got: %s", proxyErr.Domain)
	}

	if proxyErr.Code != errors.ErrCodeProxyUpstreamTimeout {
		t.Errorf("Expected code: %s, Got: %s", errors.ErrCodeProxyUpstreamTimeout, proxyErr.Code)
	}
}

