package proxy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"proxynd/handlers/proxy"
	"proxynd/logging"
)

func init() {
	// Initialize logger for tests
	_ = logging.InitLogger(logging.LogConfig{
		Level:  logging.LevelDebug,
		Format: "json",
		Output: "stdout",
	})
}

// TestAPTHandler_Type tests the Type method
func TestAPTHandler_Type(t *testing.T) {
	handler := proxy.NewAPTHandler()
	assert.Equal(t, "apt", handler.Type())
}
