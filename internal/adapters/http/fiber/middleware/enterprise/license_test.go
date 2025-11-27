package enterprise

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDevModeLicenseMiddleware tests dev mode middleware creation
func TestNewDevModeLicenseMiddleware(t *testing.T) {
	mw := NewDevModeLicenseMiddleware()

	assert.NotNil(t, mw, "DevModeLicenseMiddleware should not be nil")
	assert.NotNil(t, mw.features, "features should not be nil")
	assert.IsType(t, &mockEnterpriseFeatures{}, mw.features, "features should be mockEnterpriseFeatures")
}

// TestMockEnterpriseFeatures_IsEnabled tests that mock always returns true
func TestMockEnterpriseFeatures_IsEnabled(t *testing.T) {
	mock := &mockEnterpriseFeatures{}
	assert.True(t, mock.IsEnabled(), "Mock should always return true for IsEnabled")
}

// TestMockEnterpriseFeatures_HasFeature tests that mock always returns true for any feature
func TestMockEnterpriseFeatures_HasFeature(t *testing.T) {
	mock := &mockEnterpriseFeatures{}

	testCases := []string{
		"rbac",
		"audit",
		"analytics",
		"security",
		"non_existent_feature",
		"",
	}

	for _, feature := range testCases {
		t.Run(feature, func(t *testing.T) {
			assert.True(t, mock.HasFeature(feature), "Mock should return true for feature: %s", feature)
		})
	}
}

// TestDevModeLicenseMiddleware_RequireEnterprise tests that dev mode allows all requests
func TestDevModeLicenseMiddleware_RequireEnterprise(t *testing.T) {
	app := fiber.New()
	mw := NewDevModeLicenseMiddleware()

	// Apply middleware
	app.Use(mw.RequireEnterprise())

	// Add test endpoint
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 200, resp.StatusCode, "Request should pass through in dev mode")

	// Verify response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"success":true`, "Response should be successful")
}

// TestProductionModeLicenseMiddleware_WithNilFeatures tests production mode with nil features
func TestProductionModeLicenseMiddleware_WithNilFeatures(t *testing.T) {
	app := fiber.New()
	mw := NewLicenseMiddleware(nil)

	// Apply middleware
	app.Use(mw.RequireEnterprise())

	// Add test endpoint
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 402, resp.StatusCode, "Request should be rejected with 402 when features is nil")

	// Verify error response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "LICENSE_REQUIRED", "Response should contain LICENSE_REQUIRED error")
}

// TestProductionModeLicenseMiddleware_WithDisabledFeatures tests production mode with disabled features
func TestProductionModeLicenseMiddleware_WithDisabledFeatures(t *testing.T) {
	app := fiber.New()

	// Create mock with disabled features
	disabledMock := &mockDisabledFeatures{}
	mw := NewLicenseMiddleware(disabledMock)

	// Apply middleware
	app.Use(mw.RequireEnterprise())

	// Add test endpoint
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 402, resp.StatusCode, "Request should be rejected when enterprise is disabled")

	// Verify error response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "LICENSE_REQUIRED", "Response should contain LICENSE_REQUIRED error")
}

// TestProductionModeLicenseMiddleware_WithEnabledFeatures tests production mode with enabled features
func TestProductionModeLicenseMiddleware_WithEnabledFeatures(t *testing.T) {
	app := fiber.New()

	// Use the standard mock (always enabled)
	enabledMock := &mockEnterpriseFeatures{}
	mw := NewLicenseMiddleware(enabledMock)

	// Apply middleware
	app.Use(mw.RequireEnterprise())

	// Add test endpoint
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 200, resp.StatusCode, "Request should pass when enterprise is enabled")
}

// TestLicenseMiddleware_RequireFeature tests feature-specific validation
func TestLicenseMiddleware_RequireFeature(t *testing.T) {
	tests := []struct {
		name           string
		feature        string
		mockFeatures   EnterpriseFeatures
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Feature enabled - dev mode",
			feature:        "rbac",
			mockFeatures:   &mockEnterpriseFeatures{},
			expectedStatus: 200,
			expectedError:  "",
		},
		{
			name:           "Feature disabled",
			feature:        "rbac",
			mockFeatures:   &mockDisabledFeatures{},
			expectedStatus: 402,
			expectedError:  "FEATURE_NOT_LICENSED",
		},
		{
			name:           "Nil features",
			feature:        "rbac",
			mockFeatures:   nil,
			expectedStatus: 402,
			expectedError:  "FEATURE_NOT_LICENSED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mw := NewLicenseMiddleware(tt.mockFeatures)

			// Apply feature-specific middleware
			app.Use(mw.RequireFeature(tt.feature))

			// Add test endpoint
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"success": true})
			})

			// Test request
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedError != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), tt.expectedError)
			}
		})
	}
}

// TestLicenseMiddleware_RequireAnyFeature tests multiple feature validation
func TestLicenseMiddleware_RequireAnyFeature(t *testing.T) {
	tests := []struct {
		name           string
		features       []string
		mockFeatures   EnterpriseFeatures
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "At least one feature enabled",
			features:       []string{"rbac", "audit"},
			mockFeatures:   &mockSelectiveFeatures{enabled: map[string]bool{"rbac": true}},
			expectedStatus: 200,
			expectedError:  "",
		},
		{
			name:           "All features disabled",
			features:       []string{"rbac", "audit"},
			mockFeatures:   &mockDisabledFeatures{},
			expectedStatus: 402,
			expectedError:  "FEATURE_NOT_LICENSED",
		},
		{
			name:           "Nil features",
			features:       []string{"rbac", "audit"},
			mockFeatures:   nil,
			expectedStatus: 402,
			expectedError:  "LICENSE_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mw := NewLicenseMiddleware(tt.mockFeatures)

			// Apply multi-feature middleware
			app.Use(mw.RequireAnyFeature(tt.features...))

			// Add test endpoint
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"success": true})
			})

			// Test request
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedError != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), tt.expectedError)
			}
		})
	}
}

// TestLicenseMiddleware_SkipInDevelopment tests X-Dev-Mode header
func TestLicenseMiddleware_SkipInDevelopment(t *testing.T) {
	tests := []struct {
		name           string
		devModeHeader  string
		mockFeatures   EnterpriseFeatures
		expectedStatus int
	}{
		{
			name:           "Dev mode header present - nil features",
			devModeHeader:  "true",
			mockFeatures:   nil,
			expectedStatus: 200,
		},
		{
			name:           "No dev mode header - nil features",
			devModeHeader:  "",
			mockFeatures:   nil,
			expectedStatus: 402,
		},
		{
			name:           "Dev mode header present - disabled features",
			devModeHeader:  "true",
			mockFeatures:   &mockDisabledFeatures{},
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mw := NewLicenseMiddleware(tt.mockFeatures)

			// Apply wrapped middleware
			handler := mw.RequireEnterprise()
			app.Use(mw.SkipInDevelopment(handler))

			// Add test endpoint
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"success": true})
			})

			// Test request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.devModeHeader != "" {
				req.Header.Set("X-Dev-Mode", tt.devModeHeader)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// Mock implementations for testing

// mockDisabledFeatures always returns false
type mockDisabledFeatures struct{}

func (m *mockDisabledFeatures) IsEnabled() bool {
	return false
}

func (m *mockDisabledFeatures) HasFeature(feature string) bool {
	return false
}

// mockSelectiveFeatures allows selective feature enabling
type mockSelectiveFeatures struct {
	enabled map[string]bool
}

func (m *mockSelectiveFeatures) IsEnabled() bool {
	for _, enabled := range m.enabled {
		if enabled {
			return true
		}
	}
	return false
}

func (m *mockSelectiveFeatures) HasFeature(feature string) bool {
	return m.enabled[feature]
}
