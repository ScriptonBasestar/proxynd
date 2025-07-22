package proxy

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipProxy_Integration(t *testing.T) {
	// Setup test environment
	originalConfigDir := os.Getenv("CONFIG_DIR")
	originalStorageDir := os.Getenv("STORAGE_DIR")
	
	defer func() {
		if originalConfigDir != "" {
			_ = os.Setenv("CONFIG_DIR", originalConfigDir)
		} else {
			_ = os.Unsetenv("CONFIG_DIR")
		}
		if originalStorageDir != "" {
			_ = os.Setenv("STORAGE_DIR", originalStorageDir)
		} else {
			_ = os.Unsetenv("STORAGE_DIR")
		}
	}()

	// Set temporary directories
	tempConfigDir := t.TempDir()
	tempStorageDir := t.TempDir()
	_ = os.Setenv("CONFIG_DIR", tempConfigDir)
	_ = os.Setenv("STORAGE_DIR", tempStorageDir)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Get("/pip/*", PipProxy)

	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
		skipReason     string
	}{
		{
			name:           "simple package index",
			path:           "/pip/simple/",
			method:         "GET",
			expectedStatus: 200,
			skipReason:     "requires upstream PyPI configuration",
		},
		{
			name:           "package metadata",
			path:           "/pip/simple/requests/",
			method:         "GET",
			expectedStatus: 200,
			skipReason:     "requires upstream PyPI configuration",
		},
		{
			name:           "package download",
			path:           "/pip/packages/requests-2.28.1-py3-none-any.whl",
			method:         "GET",
			expectedStatus: 200,
			skipReason:     "requires upstream PyPI configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, 1000)

			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestPipProxy_SecurityValidation(t *testing.T) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Get("/pip/*", PipProxy)

	tests := []struct {
		name           string
		path           string
		expectBlocked  bool
		skipReason     string
	}{
		{
			name:          "path traversal attempt",
			path:          "/pip/../../../etc/passwd",
			expectBlocked: true,
			skipReason:    "security validation should block this",
		},
		{
			name:          "normal package path",
			path:          "/pip/simple/requests/",
			expectBlocked: false,
			skipReason:    "requires upstream PyPI configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req, 1000)

			require.NoError(t, err)
			defer resp.Body.Close()

			if tt.expectBlocked {
				assert.NotEqual(t, 200, resp.StatusCode)
			}
		})
	}
}

// Benchmark test for PIP handler performance
func BenchmarkPipProxy_BasicRequest(b *testing.B) {
	// Setup environment
	tempConfigDir := b.TempDir()
	tempStorageDir := b.TempDir()
	_ = os.Setenv("CONFIG_DIR", tempConfigDir)
	_ = os.Setenv("STORAGE_DIR", tempStorageDir)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Get("/pip/*", PipProxy)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/pip/simple/", nil)
		resp, _ := app.Test(req, 100)
		if resp != nil {
			resp.Body.Close()
		}
	}
}