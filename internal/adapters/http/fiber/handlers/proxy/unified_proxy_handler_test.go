package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/factory"
	"proxynd/internal/logging"
)

// TestInitializeGlobalHandler tests the initialization function
func TestInitializeGlobalHandler(t *testing.T) {
	// Reset global state
	globalHandlerInstance = nil
	globalAdapterFactory = nil

	// Initialize
	InitializeGlobalHandler()

	// Verify both globals are set
	assert.NotNil(t, globalHandlerInstance, "Handler instance should be initialized")
	assert.NotNil(t, globalAdapterFactory, "Adapter factory should be initialized")

	// Verify logger is set
	assert.NotNil(t, globalHandlerInstance.logger, "Handler should have logger")

	// Verify factory is the same instance
	assert.Equal(t, globalAdapterFactory, globalHandlerInstance.adapterFactory,
		"Handler should use the global adapter factory")
}

// TestInitializeGlobalHandler_Idempotent tests that re-initialization is safe
func TestInitializeGlobalHandler_Idempotent(t *testing.T) {
	// Reset global state
	globalHandlerInstance = nil
	globalAdapterFactory = nil

	// Initialize first time
	InitializeGlobalHandler()
	firstInstance := globalHandlerInstance
	firstFactory := globalAdapterFactory

	// Initialize again
	InitializeGlobalHandler()

	// Should be same instances (idempotent)
	assert.Equal(t, firstInstance, globalHandlerInstance,
		"Re-initialization should not create new handler instance")
	assert.Equal(t, firstFactory, globalAdapterFactory,
		"Re-initialization should not create new factory instance")
}

// TestNewUnifiedProxyHandler tests the struct constructor
func TestNewUnifiedProxyHandler(t *testing.T) {
	logger := logging.GetLogger()
	adapterFactory := factory.NewHandlerAdapterFactory()

	handler := NewUnifiedProxyHandler(nil, adapterFactory, logger)

	require.NotNil(t, handler, "Constructor should return handler")
	assert.Nil(t, handler.proxyService, "ProxyService should be nil as passed")
	assert.Equal(t, adapterFactory, handler.adapterFactory, "Factory should be set")
	assert.Equal(t, logger, handler.logger, "Logger should be set")
}

// TestUnifiedProxyHandler_WrapperExists verifies the wrapper function is defined
func TestUnifiedProxyHandler_WrapperExists(t *testing.T) {
	// Verify wrapper function exists and is callable
	assert.NotNil(t, UnifiedProxyHandler, "Wrapper function should exist")

	// Note: Full functional test of wrapper requires HTTP context
	// That is covered by integration tests
	// This unit test verifies the wrapper is defined and accessible
}

// TestGlobalVariables tests that global variables can be accessed
func TestGlobalVariables(t *testing.T) {
	// Reset to known state
	globalHandlerInstance = nil
	globalAdapterFactory = nil

	// Verify we can read and write globals
	assert.Nil(t, globalHandlerInstance)
	assert.Nil(t, globalAdapterFactory)

	// Initialize
	InitializeGlobalHandler()

	// Verify globals are set
	assert.NotNil(t, globalHandlerInstance)
	assert.NotNil(t, globalAdapterFactory)
}
