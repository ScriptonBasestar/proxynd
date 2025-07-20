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

// TestAPTHandlerV2_Type tests the Type method
func TestAPTHandlerV2_Type(t *testing.T) {
	handler := &proxy.APTHandlerV2{}
	assert.Equal(t, "apt", handler.Type())
}
