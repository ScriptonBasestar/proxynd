package maven

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"proxynd/internal/config"
	"proxynd/internal/domain/maven"
	"proxynd/logging"
)

// MockDirectoryCollector DirectoryCollector 모의 객체
type MockDirectoryCollector struct {
	mock.Mock
}

func (m *MockDirectoryCollector) CollectDirectory(ctx context.Context, path string) (*maven.DirectoryData, error) {
	args := m.Called(ctx, path)
	return args.Get(0).(*maven.DirectoryData), args.Error(1)
}

func (m *MockDirectoryCollector) CollectFromMirror(ctx context.Context, mirror config.MavenProxyServer, path string) ([]maven.Entry, error) { //nolint:lll
	args := m.Called(ctx, mirror, path)
	return args.Get(0).([]maven.Entry), args.Error(1)
}

func (m *MockDirectoryCollector) GetMirrorStatus(ctx context.Context, mirror config.MavenProxyServer) (*maven.MirrorStatus, error) { //nolint:lll
	args := m.Called(ctx, mirror)
	return args.Get(0).(*maven.MirrorStatus), args.Error(1)
}

// MockSearchService SearchService 모의 객체
type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) Search(ctx context.Context, query string) (*maven.SearchResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(*maven.SearchResult), args.Error(1)
}

func (m *MockSearchService) IndexArtifacts(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSearchService) GetIndexStats(ctx context.Context) (*maven.IndexStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*maven.IndexStats), args.Error(1)
}

func (m *MockSearchService) UpdateIndex(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

// MockCacheManager CacheManager 모의 객체
type MockCacheManager struct {
	mock.Mock
}

func (m *MockCacheManager) Get(ctx context.Context, key string) (*maven.CacheEntry, bool) {
	args := m.Called(ctx, key)
	return args.Get(0).(*maven.CacheEntry), args.Bool(1)
}

func (m *MockCacheManager) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, data, ttl)
	return args.Error(0)
}

func (m *MockCacheManager) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheManager) PreloadPopularPaths(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheManager) GetStats(ctx context.Context) (*maven.CacheStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*maven.CacheStats), args.Error(1)
}

// MockPathAnalyzer PathAnalyzer 모의 객체
type MockPathAnalyzer struct {
	mock.Mock
}

func (m *MockPathAnalyzer) ParsePath(path string) (*maven.PathInfo, error) {
	args := m.Called(path)
	return args.Get(0).(*maven.PathInfo), args.Error(1)
}

func (m *MockPathAnalyzer) IsVersionLike(s string) bool {
	args := m.Called(s)
	return args.Bool(0)
}

func (m *MockPathAnalyzer) IsLikelyArtifact(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockPathAnalyzer) ExtractGAV(path string) (string, string, string) {
	args := m.Called(path)
	return args.String(0), args.String(1), args.String(2)
}

func (m *MockPathAnalyzer) CompareVersions(v1, v2 string) int {
	args := m.Called(v1, v2)
	return args.Int(0)
}

// 테스트 헬퍼 함수들

func createTestHandler() (*BrowserHandler, *MockDirectoryCollector, *MockSearchService, *MockCacheManager, *MockPathAnalyzer) { //nolint:lll
	config := config.MavenProxySettings{}
	logger := logging.NewLogger("test")

	mockCollector := &MockDirectoryCollector{}
	mockSearch := &MockSearchService{}
	mockCache := &MockCacheManager{}
	mockPath := &MockPathAnalyzer{}

	handler := &BrowserHandler{
		config:             config,
		logger:             logger,
		directoryCollector: mockCollector,
		searchService:      mockSearch,
		cacheManager:       mockCache,
		pathAnalyzer:       mockPath,
	}

	return handler, mockCollector, mockSearch, mockCache, mockPath
}

func createTestApp(handler *BrowserHandler) *fiber.App {
	app := fiber.New()
	app.Get("/browse/*", handler.Handle)
	return app
}

// 테스트 케이스들

func TestBrowserHandler_HandleDirectoryBrowsing(t *testing.T) {
	handler, mockCollector, _, mockCache, mockPath := createTestHandler()
	app := createTestApp(handler)

	// Mock 설정
	testPath := "org/springframework/spring-core"
	pathInfo := &maven.PathInfo{
		Type:       maven.TypeArtifact,
		GroupID:    "org.springframework",
		ArtifactID: "spring-core",
		Level:      3,
	}

	directoryData := &maven.DirectoryData{
		Path: testPath,
		Entries: []maven.Entry{
			{
				Name: "5.3.21/",
				Type: maven.TypeVersion,
			},
			{
				Name: "5.3.22/",
				Type: maven.TypeVersion,
			},
		},
		Mirrors: []maven.MirrorStatus{
			{
				Name:      "central",
				URL:       "https://repo1.maven.org/maven2",
				Available: true,
			},
		},
	}

	mockPath.On("ParsePath", testPath).Return(pathInfo, nil)
	mockCache.On("Get", mock.Anything, "directory:"+testPath).Return((*maven.CacheEntry)(nil), false)
	mockCollector.On("CollectDirectory", mock.Anything, testPath).Return(directoryData, nil)
	mockCache.On("Set", mock.Anything, "directory:"+testPath, mock.Anything, mock.Anything).Return(nil)

	// 테스트 요청 생성
	req := httptest.NewRequest("GET", "/browse/"+testPath, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
	req.Header.Set("Accept", "application/json")

	// 요청 실행
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)

	// 응답 검증
	var result maven.DirectoryData
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, testPath, result.Path)
	assert.Len(t, result.Entries, 2)
	assert.Equal(t, "5.3.21/", result.Entries[0].Name)

	// Mock 검증
	mockPath.AssertExpectations(t)
	mockCollector.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestBrowserHandler_HandleSearch(t *testing.T) {
	handler, _, mockSearch, mockCache, _ := createTestHandler()
	app := createTestApp(handler)

	// Mock 설정
	testQuery := "spring-core"
	searchResult := &maven.SearchResult{
		Query: testQuery,
		Results: []maven.SearchArtifact{
			{
				GroupID:    "org.springframework",
				ArtifactID: "spring-core",
				Versions:   []string{"5.3.22", "5.3.21"},
				Latest:     "5.3.22",
				Score:      0.95,
			},
		},
		TotalCount: 1,
		SearchTime: 50 * time.Millisecond,
	}

	mockCache.On("Get", mock.Anything, "search:"+testQuery).Return((*maven.CacheEntry)(nil), false)
	mockSearch.On("Search", mock.Anything, testQuery).Return(searchResult, nil)
	mockCache.On("Set", mock.Anything, "search:"+testQuery, searchResult, 5*time.Minute).Return(nil)

	// 테스트 요청 생성
	req := httptest.NewRequest("GET", "/browse/?search="+testQuery, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
	req.Header.Set("Accept", "application/json")

	// 요청 실행
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)

	// 응답 검증
	var result maven.SearchResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, testQuery, result.Query)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "org.springframework", result.Results[0].GroupID)

	// Mock 검증
	mockSearch.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestBrowserHandler_CacheHit(t *testing.T) {
	handler, _, _, mockCache, mockPath := createTestHandler()
	app := createTestApp(handler)

	// Mock 설정 - 캐시 히트
	testPath := "org/junit/junit"
	pathInfo := &maven.PathInfo{
		Type:       maven.TypeArtifact,
		GroupID:    "org.junit",
		ArtifactID: "junit",
		Level:      3,
	}

	cachedData := &maven.DirectoryData{
		Path: testPath,
		Entries: []maven.Entry{
			{Name: "4.13.2/", Type: maven.TypeVersion},
		},
	}

	cacheEntry := &maven.CacheEntry{
		Key:  "directory:" + testPath,
		Data: cachedData,
	}

	mockPath.On("ParsePath", testPath).Return(pathInfo, nil)
	mockCache.On("Get", mock.Anything, "directory:"+testPath).Return(cacheEntry, true)

	// 테스트 요청 생성
	req := httptest.NewRequest("GET", "/browse/"+testPath, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
	req.Header.Set("Accept", "application/json")

	// 요청 실행
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)

	// Mock 검증 (CollectDirectory가 호출되지 않아야 함)
	mockPath.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestBrowserHandler_NonBrowserRequest(t *testing.T) {
	handler, _, _, _, _ := createTestHandler()
	app := createTestApp(handler)

	// Maven 클라이언트 요청 시뮬레이션
	req := httptest.NewRequest("GET", "/browse/org/junit/junit", nil)
	req.Header.Set("User-Agent", "Apache-Maven/3.8.1")
	req.Header.Set("Accept", "application/xml")

	// 요청 실행
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 406, resp.StatusCode) // Not Acceptable

	// 응답 내용 확인
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	assert.Contains(t, body.String(), "Browser request required")
}

func TestBrowserHandler_InvalidPath(t *testing.T) {
	handler, _, _, _, mockPath := createTestHandler()
	app := createTestApp(handler)

	// Mock 설정 - 경로 분석 실패
	testPath := "invalid/../path"
	mockPath.On("ParsePath", testPath).Return((*maven.PathInfo)(nil), assert.AnError)

	// 테스트 요청 생성
	req := httptest.NewRequest("GET", "/browse/"+testPath, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
	req.Header.Set("Accept", "application/json")

	// 요청 실행
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 400, resp.StatusCode) // Bad Request

	// Mock 검증
	mockPath.AssertExpectations(t)
}

// 벤치마크 테스트

func BenchmarkBrowserHandler_HandleDirectoryBrowsing(b *testing.B) {
	handler, mockCollector, _, mockCache, mockPath := createTestHandler()
	app := createTestApp(handler)

	// Mock 설정
	testPath := "org/springframework/spring-core"
	pathInfo := &maven.PathInfo{
		Type:       maven.TypeArtifact,
		GroupID:    "org.springframework",
		ArtifactID: "spring-core",
		Level:      3,
	}

	directoryData := &maven.DirectoryData{
		Path:    testPath,
		Entries: []maven.Entry{{Name: "5.3.22/", Type: maven.TypeVersion}},
	}

	mockPath.On("ParsePath", testPath).Return(pathInfo, nil)
	mockCache.On("Get", mock.Anything, mock.Anything).Return((*maven.CacheEntry)(nil), false)
	mockCollector.On("CollectDirectory", mock.Anything, testPath).Return(directoryData, nil)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 벤치마크 실행
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/browse/"+testPath, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Benchmark)")
		req.Header.Set("Accept", "application/json")

		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}
