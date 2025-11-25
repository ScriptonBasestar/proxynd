package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"proxynd/internal/ports"
)

// Mock implementations for testing

type MockPackageManager struct {
	mock.Mock
}

func (m *MockPackageManager) GetPackage(
	ctx context.Context, req *ports.PackageRequest,
) (*ports.PackageResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.PackageResponse), args.Error(1)
}

func (m *MockPackageManager) ListPackages(ctx context.Context, repo string) (*ports.PackageListResponse, error) {
	args := m.Called(ctx, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.PackageListResponse), args.Error(1)
}

func (m *MockPackageManager) GetMetadata(
	ctx context.Context, req *ports.MetadataRequest,
) (*ports.MetadataResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.MetadataResponse), args.Error(1)
}

func (m *MockPackageManager) UploadPackage(ctx context.Context, req *ports.UploadRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockPackageManager) DeletePackage(ctx context.Context, req *ports.DeleteRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

type MockCacheManager struct {
	mock.Mock
}

func (m *MockCacheManager) Get(ctx context.Context, req *ports.CacheRequest) (*ports.CacheResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.CacheResponse), args.Error(1)
}

func (m *MockCacheManager) Set(ctx context.Context, req *ports.CacheSetRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockCacheManager) Invalidate(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCacheManager) GetBackend(name string) (ports.CacheBackend, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.CacheBackend), args.Error(1)
}

func (m *MockCacheManager) ListBackends() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockCacheManager) GetStrategy(packageType string) ports.CacheStrategy {
	args := m.Called(packageType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ports.CacheStrategy)
}

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Authenticate(ctx context.Context, req *ports.AuthRequest) (*ports.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Authorize(
	ctx context.Context, req *ports.AuthorizeRequest,
) (*ports.AuthorizeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ports.AuthorizeResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*ports.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.TokenPair), args.Error(1)
}

func (m *MockAuthService) RevokeToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*ports.TokenClaims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.TokenClaims), args.Error(1)
}

func (m *MockAuthService) GetUser(ctx context.Context, userID string) (*ports.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.User), args.Error(1)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...ports.Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...ports.Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...ports.Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Error(ctx context.Context, msg string, fields ...ports.Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Fatal(ctx context.Context, msg string, fields ...ports.Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) With(fields ...ports.Field) ports.Logger {
	args := m.Called(fields)
	return args.Get(0).(ports.Logger)
}

func (m *MockLogger) WithContext(ctx context.Context) ports.Logger {
	args := m.Called(ctx)
	return args.Get(0).(ports.Logger)
}

type MockMetricsCollector struct {
	mock.Mock
}

func (m *MockMetricsCollector) IncCounter(name string, labels map[string]string) {
	m.Called(name, labels)
}

func (m *MockMetricsCollector) AddCounter(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	m.Called(name, duration, labels)
}

func (m *MockMetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) AddGauge(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) ObserveSummary(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) StartTimer(name string, labels map[string]string) ports.Timer {
	args := m.Called(name, labels)
	return args.Get(0).(ports.Timer)
}

func (m *MockMetricsCollector) GetMetrics() (*ports.MetricsSnapshot, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.MetricsSnapshot), args.Error(1)
}

type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	args := m.Called(ctx, key, n)
	return args.Bool(0), args.Error(1)
}

func (m *MockRateLimiter) Reserve(ctx context.Context, key string, n int) (*ports.Reservation, error) {
	args := m.Called(ctx, key, n)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.Reservation), args.Error(1)
}

func (m *MockRateLimiter) Reset(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockRateLimiter) CheckLimit(ctx context.Context, req *ports.RateLimitRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockRateLimiter) GetConfig() *ports.RateLimitConfig {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*ports.RateLimitConfig)
}

type MockSignatureService struct {
	mock.Mock
}

func (m *MockSignatureService) Verify(ctx context.Context, req *ports.SignatureRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func TestProxyService_HandleProxyRequest_Success(t *testing.T) {
	// Setup mocks
	mockPM := &MockPackageManager{}
	mockCache := &MockCacheManager{}
	mockAuth := &MockAuthService{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	mockRateLimit := &MockRateLimiter{}

	// Mock logger - set up BEFORE creating services (they log during initialization)
	mockLogger.On("Debug", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()

	// Create cache strategy service
	cacheStrategy := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)

	// Create proxy service
	proxyService := NewProxyService(
		mockPM, mockCache, cacheStrategy, mockAuth,
		mockLogger, mockMetrics, mockRateLimit,
	)

	// Setup expectations
	mockAuth.On("Authenticate", mock.Anything, mock.AnythingOfType("*ports.AuthRequest")).Return(&ports.AuthResponse{
		User: &ports.User{ID: "test-user"},
	}, nil)
	mockAuth.On("Authorize", mock.Anything, mock.AnythingOfType("*ports.AuthorizeRequest")).Return(&ports.AuthorizeResponse{Allowed: true}, nil)
	// ProxyService uses Allow() not CheckLimit()
	mockRateLimit.On("Allow", mock.Anything, mock.AnythingOfType("string")).Return(true, nil)

	// Cache miss
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("*ports.CacheRequest")).Return(nil, nil)

	// Upstream response
	upstreamResp := &ports.PackageResponse{
		Content:     io.NopCloser(strings.NewReader("test content")),
		ContentType: "application/json",
		Size:        12,
		Headers:     map[string]string{"Content-Length": "12"},
	}
	mockPM.On("GetPackage", mock.Anything, mock.AnythingOfType("*ports.PackageRequest")).Return(upstreamResp, nil)

	// Note: Signature verification not implemented in current service

	// Cache write (async)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("*ports.CacheSetRequest")).Return(nil)

	// Metrics
	mockMetrics.On("IncCounter", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string")).Return()
	mockMetrics.On("RecordDuration",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("map[string]string")).Return()
	mockMetrics.On("ObserveHistogram",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &ProxyRequest{
		PackageType:   "maven",
		Repository:    "central",
		Path:          "/com/example/test/1.0.0/test-1.0.0.jar",
		Method:        "GET",
		Headers:       map[string]string{},
		QueryParams:   map[string]string{},
		UserID:        "",
		Authorization: "Bearer test-token",
	}

	// Execute
	resp, err := proxyService.HandleProxyRequest(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, []byte("test content"), resp.Content)
	assert.Equal(t, "application/json", resp.ContentType)
	assert.False(t, resp.Cached)

	// Verify all mocks were called as expected
	mockAuth.AssertExpectations(t)
	mockPM.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
	// Note: Cache and metrics calls are async so harder to verify exactly
}

func TestProxyService_HandleProxyRequest_CacheHit(t *testing.T) {
	// Setup mocks
	mockPM := &MockPackageManager{}
	mockCache := &MockCacheManager{}
	mockAuth := &MockAuthService{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	mockRateLimit := &MockRateLimiter{}

	// Mock logger - set up BEFORE creating services (services log during initialization)
	mockLogger.On("Debug", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()

	// Create services AFTER mock setup
	cacheStrategy := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)
	proxyService := NewProxyService(
		mockPM, mockCache, cacheStrategy, mockAuth,
		mockLogger, mockMetrics, mockRateLimit,
	)

	// Setup expectations for auth and rate limiting
	mockAuth.On("Authenticate", mock.Anything, mock.AnythingOfType("*ports.AuthRequest")).Return(&ports.AuthResponse{
		User: &ports.User{ID: "test-user"},
	}, nil)
	mockAuth.On("Authorize", mock.Anything, mock.AnythingOfType("*ports.AuthorizeRequest")).Return(&ports.AuthorizeResponse{Allowed: true}, nil)
	// ProxyService uses Allow() not CheckLimit()
	mockRateLimit.On("Allow", mock.Anything, mock.AnythingOfType("string")).Return(true, nil)

	// Cache hit
	cacheResp := &ports.CacheResponse{
		Data:      []byte("cached content"),
		Hit:       true,
		Backend:   "filesystem",
		Size:      14,
		ExpiresAt: time.Now().Add(time.Hour),
		Metadata: ports.CacheMetadata{
			ContentType: "application/json",
			Headers:     map[string]string{"Content-Length": "14"},
		},
	}
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("*ports.CacheRequest")).Return(cacheResp, nil)

	// Metrics
	mockMetrics.On("IncCounter", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string")).Return()
	mockMetrics.On("ObserveHistogram", mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return()
	mockMetrics.On("RecordDuration",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &ProxyRequest{
		PackageType:   "maven",
		Repository:    "central",
		Path:          "/com/example/test/1.0.0/test-1.0.0.jar",
		Method:        "GET",
		Authorization: "Bearer test-token",
	}

	// Execute
	resp, err := proxyService.HandleProxyRequest(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, []byte("cached content"), resp.Content)
	assert.Equal(t, "application/json", resp.ContentType)
	assert.True(t, resp.Cached)

	// Verify mocks
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
	// Package manager should NOT be called on cache hit
	mockPM.AssertNotCalled(t, "GetPackage")
}

func TestProxyService_HandleProxyRequest_AuthFailure(t *testing.T) {
	// Setup mocks
	mockPM := &MockPackageManager{}
	mockCache := &MockCacheManager{}
	mockAuth := &MockAuthService{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	mockRateLimit := &MockRateLimiter{}

	// Mock logger - set up BEFORE creating services (services log during initialization)
	mockLogger.On("Debug", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()

	// Create services AFTER mock setup
	cacheStrategy := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)
	proxyService := NewProxyService(
		mockPM, mockCache, cacheStrategy, mockAuth,
		mockLogger, mockMetrics, mockRateLimit,
	)

	// Auth failure
	mockAuth.On("Authenticate",
		mock.Anything,
		mock.AnythingOfType("*ports.AuthRequest")).Return(nil, fmt.Errorf("invalid token"))

	// Metrics
	mockMetrics.On("IncCounter", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &ProxyRequest{
		PackageType:   "maven",
		Repository:    "central",
		Path:          "/com/example/test/1.0.0/test-1.0.0.jar",
		Method:        "GET",
		Authorization: "Bearer invalid-token",
	}

	// Execute
	resp, err := proxyService.HandleProxyRequest(context.Background(), req)

	// Assertions
	require.NoError(t, err) // Service returns error in response, not as error
	assert.NotNil(t, resp)
	assert.Equal(t, 401, resp.StatusCode)
	assert.Contains(t, string(resp.Content), "authentication failed")
	assert.NotNil(t, resp.Error)

	// Verify no other services were called
	mockPM.AssertNotCalled(t, "GetPackage")
	mockCache.AssertNotCalled(t, "Get")
}

func TestProxyService_HandleProxyRequest_RateLimitExceeded(t *testing.T) {
	// Setup mocks
	mockPM := &MockPackageManager{}
	mockCache := &MockCacheManager{}
	mockAuth := &MockAuthService{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}
	mockRateLimit := &MockRateLimiter{}

	// Mock logger - set up BEFORE creating services (services log during initialization)
	mockLogger.On("Debug", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Warn", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return()

	// Create services AFTER mock setup
	cacheStrategy := NewCacheStrategyService(mockCache, mockLogger, mockMetrics, nil)
	proxyService := NewProxyService(
		mockPM, mockCache, cacheStrategy, mockAuth,
		mockLogger, mockMetrics, mockRateLimit,
	)

	// Auth success
	mockAuth.On("Authenticate", mock.Anything, mock.AnythingOfType("*ports.AuthRequest")).Return(&ports.AuthResponse{
		User: &ports.User{ID: "test-user"},
	}, nil)
	mockAuth.On("Authorize", mock.Anything, mock.AnythingOfType("*ports.AuthorizeRequest")).Return(&ports.AuthorizeResponse{Allowed: true}, nil)

	// Rate limit exceeded - ProxyService uses Allow() not CheckLimit()
	mockRateLimit.On("Allow", mock.Anything, mock.AnythingOfType("string")).Return(false, fmt.Errorf("rate limit exceeded"))

	// Metrics
	mockMetrics.On("IncCounter", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string")).Return()

	// Test request
	req := &ProxyRequest{
		PackageType:   "npm",
		Repository:    "npmjs",
		Path:          "/express",
		Method:        "GET",
		Authorization: "Bearer test-token",
	}

	// Execute
	resp, err := proxyService.HandleProxyRequest(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 429, resp.StatusCode)
	assert.Contains(t, string(resp.Content), "rate limit exceeded")

	// Verify no downstream services were called
	mockPM.AssertNotCalled(t, "GetPackage")
	mockCache.AssertNotCalled(t, "Get")
}

func TestProxyService_ValidateRequest(t *testing.T) {
	proxyService := &ProxyService{}

	tests := []struct {
		name        string
		req         *ProxyRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			req: &ProxyRequest{
				PackageType: "maven",
				Repository:  "central",
				Path:        "/com/example/test/1.0.0/test-1.0.0.jar",
				Method:      "GET",
			},
			expectError: false,
		},
		{
			name: "missing package type",
			req: &ProxyRequest{
				Repository: "central",
				Path:       "/test",
				Method:     "GET",
			},
			expectError: true,
			errorMsg:    "package_type is required",
		},
		{
			name: "missing repository",
			req: &ProxyRequest{
				PackageType: "maven",
				Path:        "/test",
				Method:      "GET",
			},
			expectError: true,
			errorMsg:    "repository is required",
		},
		{
			name: "missing path",
			req: &ProxyRequest{
				PackageType: "maven",
				Repository:  "central",
				Method:      "GET",
			},
			expectError: true,
			errorMsg:    "path is required",
		},
		{
			name: "method defaults to GET",
			req: &ProxyRequest{
				PackageType: "maven",
				Repository:  "central",
				Path:        "/test",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := proxyService.ValidateRequest(context.Background(), tt.req)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if tt.req.Method == "" {
					assert.Equal(t, "GET", tt.req.Method)
				}
			}
		})
	}
}
