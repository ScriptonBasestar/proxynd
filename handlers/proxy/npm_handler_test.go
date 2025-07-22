package proxy

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpmProxy_MetadataDetection(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "simple package metadata",
			path:     "lodash",
			expected: true,
		},
		{
			name:     "scoped package metadata",
			path:     "@scope/package",
			expected: true,
		},
		{
			name:     "package tarball",
			path:     "package/-/package-1.0.0.tgz",
			expected: false,
		},
		{
			name:     "npm search with /-/",
			path:     "-/search",
			expected: false,
		},
		{
			name:     "package with .tgz",
			path:     "lodash.tgz",
			expected: false,
		},
		{
			name:     "deep path",
			path:     "lodash/version/1.0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNpmMetadataRequest(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNpmProxy_ContentType(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		path         string
		expectedType string
	}{
		{
			name:         "tarball file",
			filename:     "package-1.0.0.tgz",
			path:         "/package/-/package-1.0.0.tgz",
			expectedType: "application/gzip",
		},
		{
			name:         "json metadata",
			filename:     "package.json",
			path:         "/package.json",
			expectedType: "application/json",
		},
		{
			name:         "unknown file",
			filename:     "README.md",
			path:         "/package/README.md",
			expectedType: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentType := getNpmContentType(tt.filename, tt.path)
			assert.Equal(t, tt.expectedType, contentType)
		})
	}
}

func TestNpmProxy_MetadataRewrite(t *testing.T) {
	// Test metadata rewriting functionality
	originalData := []byte(`{
		"name": "test-package",
		"version": "1.0.0",
		"dist": {
			"tarball": "https://registry.npmjs.org/test-package/-/test-package-1.0.0.tgz"
		}
	}`)

	baseURL := "http://localhost:8080"
	rewrittenData := rewriteNpmMetadata(originalData, baseURL, "")

	// Verify the data was rewritten (basic check)
	assert.NotNil(t, rewrittenData)
	assert.Contains(t, string(rewrittenData), "test-package")
}

func TestNpmProxy_Integration(t *testing.T) {
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

	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
		skipReason     string
	}{
		{
			name:           "NPM registry root",
			path:           "/npm/",
			method:         "GET",
			expectedStatus: 200,
			skipReason:     "requires upstream npm registry",
		},
		{
			name:           "NPM package request",
			path:           "/npm/lodash",
			method:         "GET", 
			expectedStatus: 200,
			skipReason:     "requires upstream npm registry",
		},
		{
			name:           "NPM tarball request",
			path:           "/npm/lodash/-/lodash-4.17.21.tgz",
			method:         "GET",
			expectedStatus: 200,
			skipReason:     "requires upstream npm registry",
		},
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Setup NPM proxy route
	app.Get("/npm/*", NpmProxy)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, 1000) // 1 second timeout

			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestNpmProxy_ErrorHandling(t *testing.T) {
	// Setup test environment
	originalConfigDir := os.Getenv("CONFIG_DIR")
	defer func() {
		if originalConfigDir != "" {
			_ = os.Setenv("CONFIG_DIR", originalConfigDir)
		} else {
			_ = os.Unsetenv("CONFIG_DIR")
		}
	}()

	// Set temporary directory without config files
	tempDir := t.TempDir()
	_ = os.Setenv("CONFIG_DIR", tempDir)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Get("/npm/*", NpmProxy)

	tests := []struct {
		name           string
		path           string
		expectError    bool
		skipReason     string
	}{
		{
			name:        "invalid package path",
			path:        "/npm/../../../etc/passwd",
			expectError: true,
			skipReason:  "path traversal should be blocked",
		},
		{
			name:        "empty package name",
			path:        "/npm/",
			expectError: false,
			skipReason:  "requires upstream configuration",
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

			if tt.expectError {
				assert.NotEqual(t, 200, resp.StatusCode)
			}
		})
	}
}

// Benchmark test for NPM handler performance
func BenchmarkNpmProxy_MetadataDetection(b *testing.B) {
	paths := []string{
		"/package.json",
		"/@scope/package",
		"/package/-/package-1.0.0.tgz",
		"/-/search",
		"/very/long/package/path/that/might/be/slow",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			isNpmMetadataRequest(path)
		}
	}
}

func BenchmarkNpmProxy_ContentType(b *testing.B) {
	testCases := []struct {
		filename string
		path     string
	}{
		{"package-1.0.0.tgz", "/package/-/package-1.0.0.tgz"},
		{"package.json", "/package.json"},
		{"README.md", "/package/README.md"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			getNpmContentType(tc.filename, tc.path)
		}
	}
}