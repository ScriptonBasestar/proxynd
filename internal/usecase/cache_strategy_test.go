package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"proxynd/internal/ports"
)

type MockCacheKeyBuilder struct {
	mock.Mock
}

func (m *MockCacheKeyBuilder) BuildKey(packageType, repository, name, version, path string) string {
	args := m.Called(packageType, repository, name, version, path)
	return args.String(0)
}

func (m *MockCacheKeyBuilder) ParseKey(key string) (*ports.KeyComponents, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.KeyComponents), args.Error(1)
}

func (m *MockCacheKeyBuilder) BuildPattern(packageType, repository string) string {
	args := m.Called(packageType, repository)
	return args.String(0)
}

func TestCacheStrategyService_DetermineStrategy(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	mockKeyBuilder := &MockCacheKeyBuilder{}
	
	// Create service
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, mockKeyBuilder)
	
	// Mock key builder
	mockKeyBuilder.On("BuildKey", "maven", "central", "com.example", "1.0.0", "/test.jar").Return("maven:central:com.example:1.0.0:/test.jar")
	
	// Mock logger
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()

	// Test request
	req := &CacheRequest{
		PackageType: "maven",
		Repository:  "central",
		Name:        "com.example",
		Version:     "1.0.0",
		Path:        "/test.jar",
		ContentType: "application/java-archive",
		Size:        1024,
	}

	// Execute
	decision, err := css.DetermineStrategy(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, decision)
	assert.True(t, decision.ShouldCache)
	assert.Equal(t, time.Hour*24, decision.TTL) // Maven default TTL
	assert.Equal(t, "maven:central:com.example:1.0.0:/test.jar", decision.CacheKey)
	assert.Equal(t, "maven", decision.Strategy)

	mockKeyBuilder.AssertExpectations(t)
}

func TestCacheStrategyService_DetermineStrategy_FallbackToDefault(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	
	// Create service without key builder (fallback)
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)
	
	// Mock logger
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()

	// Test request for unknown package type
	req := &CacheRequest{
		PackageType: "unknown",
		Repository:  "test-repo",
		Path:        "/test/path",
		ContentType: "application/octet-stream",
		Size:        512,
	}

	// Execute
	decision, err := css.DetermineStrategy(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, decision)
	assert.True(t, decision.ShouldCache) // Default strategy should cache
	assert.Equal(t, time.Hour*24, decision.TTL) // Default TTL
	assert.Equal(t, "unknown:test-repo:/test/path", decision.CacheKey) // Generated key
	assert.Equal(t, "unknown", decision.Strategy)
}

func TestCacheStrategyService_Get_CacheHit(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)

	// Cache hit response
	cacheResp := &ports.CacheResponse{
		Data:      []byte("cached data"),
		Hit:       true,
		Backend:   "filesystem",
		Size:      11,
		ExpiresAt: time.Now().Add(time.Hour),
		Metadata: ports.CacheMetadata{
			ContentType: "text/plain",
			Headers:     map[string]string{"X-Custom": "value"},
		},
	}
	
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("*ports.CacheRequest")).Return(cacheResp, nil)
	mockMetrics.On("IncrementCounter", "cache_hits", mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &CacheRequest{
		PackageType: "npm",
		Repository:  "npmjs",
		Path:        "/express",
	}

	// Execute
	resp, err := css.Get(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Hit)
	assert.Equal(t, []byte("cached data"), resp.Content)
	assert.Equal(t, int64(11), resp.Size)
	assert.Equal(t, "filesystem", resp.Backend)
	assert.Equal(t, "text/plain", resp.Metadata["content_type"])
	assert.Equal(t, "value", resp.Metadata["X-Custom"])

	mockCache.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestCacheStrategyService_Get_CacheMiss(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)

	// Cache miss (nil response)
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("*ports.CacheRequest")).Return(nil, nil)
	mockMetrics.On("IncrementCounter", "cache_misses", mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &CacheRequest{
		PackageType: "pypi",
		Repository:  "pypi",
		Path:        "/numpy",
	}

	// Execute
	resp, err := css.Get(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Hit)
	assert.Empty(t, resp.Content)
	assert.Equal(t, "pypi:pypi:/numpy", resp.CacheKey)

	mockCache.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestCacheStrategyService_Get_CacheError(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)

	// Cache error
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("*ports.CacheRequest")).Return(nil, assert.AnError)
	mockLogger.On("Error", "Cache retrieval failed", mock.Anything).Return()
	mockMetrics.On("IncrementCounter", "cache_errors", mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &CacheRequest{
		PackageType: "docker",
		Repository:  "dockerhub",
		Path:        "/nginx",
	}

	// Execute
	resp, err := css.Get(context.Background(), req)

	// Assertions - should not fail the request, just return cache miss
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Hit)

	mockCache.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestCacheStrategyService_Set_Success(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)

	// Mock cache set operation
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("*ports.CacheSetRequest")).Return(nil)
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()
	mockMetrics.On("IncrementCounter", "cache_writes", mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &CacheRequest{
		PackageType: "apt",
		Repository:  "ubuntu",
		Path:        "/dists/focal/Release",
		ContentType: "text/plain",
	}
	content := []byte("test content")

	// Execute
	err := css.Set(context.Background(), req, content)

	// Assertions
	require.NoError(t, err)

	mockCache.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestCacheStrategyService_Set_ShouldNotCache(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)
	
	// Register a strategy that doesn't cache
	noCacheStrategy := &MockCacheStrategy{}
	noCacheStrategy.On("ShouldCache", mock.AnythingOfType("*ports.CacheRequest")).Return(false)
	css.RegisterStrategy("nocache", noCacheStrategy)

	mockLogger.On("Info", "Registered cache strategy", mock.Anything).Return()
	mockLogger.On("Debug", "Strategy determined not to cache", mock.Anything).Return()

	// Test request
	req := &CacheRequest{
		PackageType: "nocache",
		Repository:  "test",
		Path:        "/test",
	}
	content := []byte("test content")

	// Execute
	err := css.Set(context.Background(), req, content)

	// Assertions
	require.NoError(t, err)

	// Cache should not be called
	mockCache.AssertNotCalled(t, "Set")
	noCacheStrategy.AssertExpectations(t)
}

func TestCacheStrategyService_RegisterAndGetStrategy(t *testing.T) {
	// Setup mocks
	mockCache := &MockCacheManager{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	
	css := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)

	// Mock logger for registration
	mockLogger.On("Info", "Registered cache strategy", mock.Anything).Return()

	// Create and register custom strategy
	customStrategy := &MockCacheStrategy{}
	css.RegisterStrategy("custom", customStrategy)

	// Test getting registered strategy
	retrievedStrategy := css.GetStrategy("custom")
	assert.Equal(t, customStrategy, retrievedStrategy)

	// Test getting non-existent strategy (should return default)
	defaultStrategy := css.GetStrategy("nonexistent")
	assert.NotNil(t, defaultStrategy) // Should return default strategy

	// Test getting default strategy directly
	explicitDefault := css.GetStrategy("default")
	assert.NotNil(t, explicitDefault)
	assert.Equal(t, defaultStrategy, explicitDefault)

	mockLogger.AssertExpectations(t)
}

func TestCacheStrategyService_GenerateCacheKey(t *testing.T) {
	css := &CacheStrategyService{}

	tests := []struct {
		name     string
		req      *CacheRequest
		expected string
	}{
		{
			name: "full request",
			req: &CacheRequest{
				PackageType: "maven",
				Repository:  "central",
				Name:        "com.example",
				Version:     "1.0.0",
				Path:        "/test.jar",
			},
			expected: "maven:central:com.example:1.0.0:test.jar",
		},
		{
			name: "minimal request",
			req: &CacheRequest{
				PackageType: "npm",
				Repository:  "npmjs",
				Path:        "/express",
			},
			expected: "npm:npmjs:express",
		},
		{
			name: "with leading slash in path",
			req: &CacheRequest{
				PackageType: "docker",
				Repository:  "dockerhub",
				Path:        "/library/nginx",
			},
			expected: "docker:dockerhub:library/nginx",
		},
		{
			name: "empty path",
			req: &CacheRequest{
				PackageType: "apt",
				Repository:  "ubuntu",
				Path:        "",
			},
			expected: "apt:ubuntu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := css.generateCacheKey(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCacheStrategyService_GetDefaultTTL(t *testing.T) {
	css := NewCacheStrategyService(nil, nil, nil, nil)

	tests := []struct {
		packageType string
		expected    time.Duration
	}{
		{"maven", time.Hour * 24},
		{"npm", time.Hour * 12},
		{"docker", time.Hour * 48},
		{"apt", time.Hour * 6},
		{"unknown", time.Hour * 24}, // default
	}

	for _, tt := range tests {
		t.Run(tt.packageType, func(t *testing.T) {
			result := css.getDefaultTTL(tt.packageType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock CacheStrategy for testing
type MockCacheStrategy struct {
	mock.Mock
}

func (m *MockCacheStrategy) ShouldCache(req *ports.CacheRequest) bool {
	args := m.Called(req)
	return args.Bool(0)
}

func (m *MockCacheStrategy) GetTTL(req *ports.CacheRequest) time.Duration {
	args := m.Called(req)
	return args.Get(0).(time.Duration)
}

func (m *MockCacheStrategy) GetBackend(req *ports.CacheRequest) string {
	args := m.Called(req)
	return args.String(0)
}

func (m *MockCacheStrategy) GenerateKey(req *ports.CacheRequest) string {
	args := m.Called(req)
	return args.String(0)
}

func (m *MockCacheStrategy) GetEvictionPolicy() ports.EvictionPolicy {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ports.EvictionPolicy)
}