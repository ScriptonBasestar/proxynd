package testutil

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/config"
	"proxynd/internal/containerhandlers"
)

// ContainerHandlerTestSuite Container 기반 핸들러를 위한 통합 테스트 스위트
type ContainerHandlerTestSuite struct {
	t             *testing.T
	mockContainer *MockContainerProvider
	app           *fiber.App
	tempDir       string
}

// NewContainerHandlerTestSuite 새로운 테스트 스위트 생성
func NewContainerHandlerTestSuite(t *testing.T) *ContainerHandlerTestSuite {
	suite := &ContainerHandlerTestSuite{
		t:             t,
		mockContainer: NewMockContainerProvider(t),
		app:           fiber.New(),
	}

	// 임시 디렉토리 생성 및 구조 설정
	suite.setupTempDirectories()

	return suite
}

// setupTempDirectories 임시 디렉토리 구조 설정
func (s *ContainerHandlerTestSuite) setupTempDirectories() {
	s.tempDir = s.t.TempDir()

	// 스토리지 디렉토리 생성
	storageDir := filepath.Join(s.tempDir, "storage")
	require.NoError(s.t, os.MkdirAll(storageDir, 0o755))

	// 설정 디렉토리 생성
	configDir := filepath.Join(s.tempDir, "config")
	require.NoError(s.t, os.MkdirAll(configDir, 0o755))

	// 각 프록시별 캐시 디렉토리 생성
	proxyTypes := []string{"apt", "maven", "npm", "docker", "pip", "yum", "apk"}
	for _, proxyType := range proxyTypes {
		proxyDir := filepath.Join(storageDir, proxyType)
		require.NoError(s.t, os.MkdirAll(proxyDir, 0o755))
	}

	// MockContainer에 디렉토리 설정
	s.mockContainer.SetStorageDir(storageDir)
	s.mockContainer.SetConfigDir(configDir)
}

// TestAPTHandler APT 핸들러 테스트
func (s *ContainerHandlerTestSuite) TestAPTHandler() {
	// APT 핸들러 생성
	handler := containerhandlers.NewAPTContainerHandler(s.mockContainer)

	// 기본 정보 확인
	assert.Equal(s.t, "apt-container-handler", handler.Name())
	assert.Equal(s.t, "apt", handler.Type())

	// 활성화 상태 확인
	assert.True(s.t, handler.IsEnabled())

	// 헬스체크
	assert.NoError(s.t, handler.HealthCheck())
}

// TestMavenHandler Maven 핸들러 테스트
func (s *ContainerHandlerTestSuite) TestMavenHandler() {
	// Maven 핸들러 생성
	handler := containerhandlers.NewMavenContainerHandler(s.mockContainer)

	// 기본 정보 확인
	assert.Equal(s.t, "maven-container-handler", handler.Name())
	assert.Equal(s.t, "maven", handler.Type())

	// 활성화 상태 확인
	assert.True(s.t, handler.IsEnabled())

	// 헬스체크
	assert.NoError(s.t, handler.HealthCheck())
}

// TestNPMHandler NPM 핸들러 테스트
func (s *ContainerHandlerTestSuite) TestNPMHandler() {
	// NPM 핸들러 생성
	handler := containerhandlers.NewNPMContainerHandler(s.mockContainer)

	// 기본 정보 확인
	assert.Equal(s.t, "npm-container-handler", handler.Name())
	assert.Equal(s.t, "npm", handler.Type())

	// 활성화 상태 확인
	assert.True(s.t, handler.IsEnabled())

	// 헬스체크
	assert.NoError(s.t, handler.HealthCheck())
}

// TestDockerHandler Docker 핸들러 테스트
func (s *ContainerHandlerTestSuite) TestDockerHandler() {
	// Docker 핸들러 생성
	handler := containerhandlers.NewDockerContainerHandler(s.mockContainer)

	// 기본 정보 확인
	assert.Equal(s.t, "docker-container-handler", handler.Name())
	assert.Equal(s.t, "docker", handler.Type())

	// 활성화 상태 확인
	assert.True(s.t, handler.IsEnabled())

	// 헬스체크
	assert.NoError(s.t, handler.HealthCheck())
}

// TestPIPHandler PIP 핸들러 테스트
func (s *ContainerHandlerTestSuite) TestPIPHandler() {
	// PIP 핸들러 생성
	handler := containerhandlers.NewPIPContainerHandler(s.mockContainer)

	// 기본 정보 확인
	assert.Equal(s.t, "pip-container-handler", handler.Name())
	assert.Equal(s.t, "pip", handler.Type())

	// 활성화 상태 확인
	assert.True(s.t, handler.IsEnabled())

	// 헬스체크
	assert.NoError(s.t, handler.HealthCheck())
}

// TestHandlerWithConfigError 설정 로딩 오류 시나리오 테스트
func (s *ContainerHandlerTestSuite) TestHandlerWithConfigError() {
	// 설정 오류 상황 설정
	configErr := assert.AnError
	errorContainer := s.mockContainer.WithError("GetPipProxyConfig", configErr)

	// PIP 핸들러로 테스트 (다른 핸들러도 동일한 패턴)
	handler := containerhandlers.NewPIPContainerHandler(errorContainer)

	// 활성화 상태가 false가 되어야 함
	assert.False(s.t, handler.IsEnabled())
}

// TestHandlerCacheKeyGeneration 캐시 키 생성 테스트
func (s *ContainerHandlerTestSuite) TestHandlerCacheKeyGeneration() {
	// Fiber 앱 설정 및 라우팅
	app := fiber.New()

	// 테스트용 라우트 설정
	app.Get("/test/*", func(c *fiber.Ctx) error {
		// PIP 핸들러로 테스트
		handler := containerhandlers.NewPIPContainerHandler(s.mockContainer)

		// 캐시 키 생성 테스트
		cacheKey := handler.GetCacheKey(c)
		assert.NotEmpty(s.t, cacheKey)
		assert.Contains(s.t, cacheKey, "pip")

		// 동일한 요청에 대해서는 같은 키가 생성되어야 함
		cacheKey2 := handler.GetCacheKey(c)
		assert.Equal(s.t, cacheKey, cacheKey2)

		return c.SendStatus(200)
	})

	// 테스트 요청 실행
	req := httptest.NewRequest("GET", "/test/simple/requests", nil)
	resp, err := app.Test(req)
	assert.NoError(s.t, err)
	assert.Equal(s.t, 200, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()
}

// TestHandlerCachingPolicy 캐싱 정책 테스트
func (s *ContainerHandlerTestSuite) TestHandlerCachingPolicy() {
	// PIP 핸들러로 테스트
	handler := containerhandlers.NewPIPContainerHandler(s.mockContainer)

	// GET 요청 테스트
	app := fiber.New()
	app.Get("/test/*", func(c *fiber.Ctx) error {
		// GET 요청은 캐시 가능
		assert.True(s.t, handler.IsCacheable(c))

		// 200 OK 응답은 캐시 가능 (.whl 파일)
		assert.True(s.t, handler.ShouldCache(c, 200))

		// 404 응답은 캐시 불가
		assert.False(s.t, handler.ShouldCache(c, 404))

		return c.SendStatus(200)
	})

	// POST 요청 테스트
	app.Post("/test/*", func(c *fiber.Ctx) error {
		// POST 요청은 캐시 불가
		assert.False(s.t, handler.IsCacheable(c))
		return c.SendStatus(200)
	})

	// GET 요청 실행
	req := httptest.NewRequest("GET", "/test/package.whl", nil)
	resp, err := app.Test(req)
	assert.NoError(s.t, err)
	assert.Equal(s.t, 200, resp.StatusCode)
	_ = resp.Body.Close()

	// POST 요청 실행
	req = httptest.NewRequest("POST", "/test/package.whl", nil)
	resp, err = app.Test(req)
	assert.NoError(s.t, err)
	assert.Equal(s.t, 200, resp.StatusCode)
	_ = resp.Body.Close()
}

// TestHandlerUpstreamURLBuilding 업스트림 URL 구성 테스트
func (s *ContainerHandlerTestSuite) TestHandlerUpstreamURLBuilding() {
	// Fiber 앱 설정 및 라우팅
	app := fiber.New()

	// 테스트용 라우트 설정
	app.Get("/test/*", func(c *fiber.Ctx) error {
		// PIP 핸들러로 테스트
		handler := containerhandlers.NewPIPContainerHandler(s.mockContainer)

		// 업스트림 URL 생성
		upstreamURL, err := handler.BuildUpstreamURL(c)
		assert.NoError(s.t, err)
		assert.NotEmpty(s.t, upstreamURL)
		assert.Contains(s.t, upstreamURL, "pypi.org")

		return c.SendStatus(200)
	})

	// 테스트 요청 실행
	req := httptest.NewRequest("GET", "/test/simple/requests/", nil)
	resp, err := app.Test(req)
	assert.NoError(s.t, err)
	assert.Equal(s.t, 200, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()
}

// TestHandlerConfigReload 설정 리로드 테스트
func (s *ContainerHandlerTestSuite) TestHandlerConfigReload() {
	// PIP 핸들러 생성
	handler := containerhandlers.NewPIPContainerHandler(s.mockContainer)

	// 초기 설정 로드
	assert.NoError(s.t, handler.LoadConfig())

	// 설정 리로드
	assert.NoError(s.t, handler.ReloadConfig())

	// 새로운 설정으로 업데이트
	newConfig := &config.PipProxySettings{
		Path:     "pip-updated",
		UseCache: false,
		Proxies: []config.PipProxyServer{
			{
				Name: "test-pypi-updated",
				URL:  "https://test-pypi.org",
			},
		},
	}

	s.mockContainer.SetPipConfig(newConfig)
	assert.NoError(s.t, handler.ReloadConfig())
}

// TestAllHandlersBasicFunctionality 모든 핸들러의 기본 기능 테스트
func (s *ContainerHandlerTestSuite) TestAllHandlersBasicFunctionality() {
	handlerTests := []struct {
		name         string
		handlerFunc  func() interface{}
		expectedName string
		expectedType string
	}{
		{
			name: "APT Handler",
			handlerFunc: func() interface{} {
				return containerhandlers.NewAPTContainerHandler(s.mockContainer)
			},
			expectedName: "apt-container-handler",
			expectedType: "apt",
		},
		{
			name: "Maven Handler",
			handlerFunc: func() interface{} {
				return containerhandlers.NewMavenContainerHandler(s.mockContainer)
			},
			expectedName: "maven-container-handler",
			expectedType: "maven",
		},
		{
			name: "NPM Handler",
			handlerFunc: func() interface{} {
				return containerhandlers.NewNPMContainerHandler(s.mockContainer)
			},
			expectedName: "npm-container-handler",
			expectedType: "npm",
		},
		{
			name: "Docker Handler",
			handlerFunc: func() interface{} {
				return containerhandlers.NewDockerContainerHandler(s.mockContainer)
			},
			expectedName: "docker-container-handler",
			expectedType: "docker",
		},
		{
			name: "PIP Handler",
			handlerFunc: func() interface{} {
				return containerhandlers.NewPIPContainerHandler(s.mockContainer)
			},
			expectedName: "pip-container-handler",
			expectedType: "pip",
		},
	}

	for _, tt := range handlerTests {
		s.t.Run(tt.name, func(t *testing.T) {
			handler := tt.handlerFunc()

			// 인터페이스 확인을 위한 타입 단언 및 검증
			switch h := handler.(type) {
			case interface{ Name() string }:
				assert.Equal(t, tt.expectedName, h.Name())
			default:
				t.Errorf("Handler does not implement Name() method")
			}

			switch h := handler.(type) {
			case interface{ Type() string }:
				assert.Equal(t, tt.expectedType, h.Type())
			default:
				t.Errorf("Handler does not implement Type() method")
			}

			switch h := handler.(type) {
			case interface{ IsEnabled() bool }:
				assert.True(t, h.IsEnabled())
			default:
				t.Errorf("Handler does not implement IsEnabled() method")
			}

			switch h := handler.(type) {
			case interface{ HealthCheck() error }:
				assert.NoError(t, h.HealthCheck())
			default:
				t.Errorf("Handler does not implement HealthCheck() method")
			}
		})
	}
}

// GetMockContainer MockContainer 인스턴스 반환
func (s *ContainerHandlerTestSuite) GetMockContainer() *MockContainerProvider {
	return s.mockContainer
}

// GetTempDir 임시 디렉토리 경로 반환
func (s *ContainerHandlerTestSuite) GetTempDir() string {
	return s.tempDir
}

// Cleanup 리소스 정리
func (s *ContainerHandlerTestSuite) Cleanup() {
	// Fiber 앱 정리
	if s.app != nil {
		_ = s.app.Shutdown()
	}

	// Mock 검증
	s.mockContainer.Verify(s.t)
}
