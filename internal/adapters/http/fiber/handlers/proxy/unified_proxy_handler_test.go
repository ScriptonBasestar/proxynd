package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/factory"
	"proxynd/internal/logging"
)

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

