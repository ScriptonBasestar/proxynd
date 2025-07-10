package testutil_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"proxynd/configs"
	"proxynd/internal/services/proxy"
	"proxynd/internal/services/proxy/mocks"
	"proxynd/internal/testutil"
)

// ExampleUsingFixtures demonstrates using test fixtures
func TestExampleUsingFixtures(t *testing.T) {
	// Create fixtures
	fixtures := testutil.NewFixtures()

	// Use pre-configured test data
	globalConfig := fixtures.ValidGlobalConfig()
	assert.NotNil(t, globalConfig)
	assert.Equal(t, "/tmp/test-storage", globalConfig.StorageDir)

	// Use proxy request fixtures
	req := fixtures.ProxyRequest("GET", "/test/path")
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/test/path", req.Path)

	// Use response fixtures
	resp := fixtures.ProxyResponse(200, "test body")
	assert.Equal(t, 200, resp.StatusCode)
}

// ExampleUsingFactories demonstrates using test factories
func TestExampleUsingFactories(t *testing.T) {
	// Create factory
	factory := testutil.NewFactory(t)

	// Create test objects with sensible defaults
	configService := factory.ConfigService()
	assert.NotNil(t, configService)

	// Create proxy service
	proxyService := factory.ProxyService("apt")
	assert.NotNil(t, proxyService)
}

// ExampleUsingBuilders demonstrates using test builders
func TestExampleUsingBuilders(t *testing.T) {
	// Build custom configuration
	config := testutil.NewGlobalConfigBuilder().
		WithStorageDir("/custom/storage").
		WithCacheDir("/custom/cache").
		WithCacheTTL(7200).
		WithMaxCacheSize(1024 * 1024 * 500). // 500MB
		Build()

	globalConfig := config.(*configs.GlobalConfig)
	assert.Equal(t, "/custom/storage", globalConfig.StorageDir)
	assert.Equal(t, "/custom/cache", globalConfig.CacheDir)
	assert.Equal(t, 7200, globalConfig.CacheTTL)
}

// ExampleUsingAssertions demonstrates custom assertions
func TestExampleUsingAssertions(t *testing.T) {
	assertions := testutil.NewAssertions(t)

	// Test headers
	expected := map[string]string{
		"Content-Type": "application/json",
		"X-Cache":      "HIT",
	}
	actual := map[string]string{
		"Content-Type": "application/json",
		"X-Cache":      "HIT",
	}

	assertions.AssertHeadersEqual(expected, actual)
	assertions.AssertCacheHit(true)
	assertions.AssertStatusCode(200, 200)
}

// ExampleUsingMatchers demonstrates custom mock matchers
func TestExampleUsingMatchers(t *testing.T) {
	// Create mock
	mockCache := mocks.NewMockCacheService(t)
	matchers := testutil.NewMatchers()

	// Use custom matchers
	mockCache.EXPECT().Get(
		matchers.AnyContext(),
		matchers.StringHasPrefix("cache:"),
	).Return(nil, false, nil)

	mockCache.EXPECT().Put(
		matchers.AnyContext(),
		matchers.AnyString(),
		matchers.AnyReader(),
	).Return(nil)

	// Test with the mock
	ctx := context.Background()
	_, _, err := mockCache.Get(ctx, "cache:test-key")
	assert.NoError(t, err)
}

// ExampleIntegrationTest demonstrates combining all utilities
func TestExampleIntegration(t *testing.T) {
	// Setup
	factory := testutil.NewFactory(t)
	fixtures := testutil.NewFixtures()
	assertions := testutil.NewAssertions(t)
	matchers := testutil.NewMatchers()

	// Create mocks
	mockUpstream := mocks.NewMockUpstreamClient(t)

	// Configure mock with custom matchers
	mockUpstream.EXPECT().Fetch(
		matchers.AnyContext(),
		matchers.URLContaining("ubuntu.com"),
		mock.Anything,
	).Return(fixtures.ProxyResponse(200, fixtures.SampleAPTPackageMetadata()), nil)

	// Create service with factory
	cacheService := factory.CacheAdapter()
	configService := factory.ConfigService()

	// Test with fixtures
	req := fixtures.ProxyRequestWithHeaders("GET", "/ubuntu/test", map[string]string{
		"User-Agent": "apt/2.0",
	})

	// Use custom assertions
	assertions.AssertNoError(nil)
	assertions.AssertMapContains(req.Headers, map[string]string{
		"User-Agent": "apt/2.0",
	})
}

// ExampleTableDrivenTest demonstrates table-driven tests with utilities
func TestExampleTableDriven(t *testing.T) {
	fixtures := testutil.NewFixtures()

	tests := []struct {
		name      string
		request   func() interface{}
		expected  string
		wantError bool
	}{
		{
			name: "APT request",
			request: func() interface{} {
				return fixtures.ProxyRequest("GET", "/ubuntu/dists/jammy/Release")
			},
			expected:  "/ubuntu/dists/jammy/Release",
			wantError: false,
		},
		{
			name: "Maven request",
			request: func() interface{} {
				return fixtures.ProxyRequest("GET", "/com/example/test/1.0/test-1.0.jar")
			},
			expected:  "/com/example/test/1.0/test-1.0.jar",
			wantError: false,
		},
		{
			name: "NPM request",
			request: func() interface{} {
				return fixtures.ProxyRequest("GET", "/express")
			},
			expected:  "/express",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.request()
			if proxyReq, ok := req.(*proxy.ProxyRequest); ok {
				assert.Equal(t, tt.expected, proxyReq.Path)
			}
		})
	}
}
