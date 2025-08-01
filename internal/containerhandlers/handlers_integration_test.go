package containerhandlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"proxynd/internal/containerhandlers"
	"proxynd/internal/testutil"
)

// ContainerHandlersIntegrationTestSuite Container 기반 핸들러들의 통합 테스트
type ContainerHandlersIntegrationTestSuite struct {
	suite.Suite
	testSuite *testutil.ContainerHandlerTestSuite
}

// SetupTest 각 테스트 전 설정
func (s *ContainerHandlersIntegrationTestSuite) SetupTest() {
	s.testSuite = testutil.NewContainerHandlerTestSuite(s.T())
}

// TearDownTest 각 테스트 후 정리
func (s *ContainerHandlersIntegrationTestSuite) TearDownTest() {
	if s.testSuite != nil {
		s.testSuite.Cleanup()
	}
}

// TestAPTContainerHandler APT Container 핸들러 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestAPTContainerHandler() {
	s.testSuite.TestAPTHandler()
}

// TestMavenContainerHandler Maven Container 핸들러 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestMavenContainerHandler() {
	s.testSuite.TestMavenHandler()
}

// TestNPMContainerHandler NPM Container 핸들러 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestNPMContainerHandler() {
	s.testSuite.TestNPMHandler()
}

// TestDockerContainerHandler Docker Container 핸들러 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestDockerContainerHandler() {
	s.testSuite.TestDockerHandler()
}

// TestPIPContainerHandler PIP Container 핸들러 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestPIPContainerHandler() {
	s.testSuite.TestPIPHandler()
}

// TestHandlerConfigErrors 설정 오류 시나리오 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestHandlerConfigErrors() {
	s.testSuite.TestHandlerWithConfigError()
}

// TestHandlerCaching 캐싱 관련 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestHandlerCaching() {
	s.testSuite.TestHandlerCacheKeyGeneration()
	s.testSuite.TestHandlerCachingPolicy()
}

// TestHandlerUpstreamURL 업스트림 URL 관련 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestHandlerUpstreamURL() {
	s.testSuite.TestHandlerUpstreamURLBuilding()
}

// TestHandlerConfigReload 설정 리로드 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestHandlerConfigReload() {
	s.testSuite.TestHandlerConfigReload()
}

// TestAllHandlersConsistency 모든 핸들러의 일관성 테스트
func (s *ContainerHandlersIntegrationTestSuite) TestAllHandlersConsistency() {
	s.testSuite.TestAllHandlersBasicFunctionality()
}

// TestSuite를 실행하는 함수
func TestContainerHandlersIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ContainerHandlersIntegrationTestSuite))
}

// 개별 단위 테스트들

// TestPIPHandlerCreation PIP 핸들러 생성 테스트
func TestPIPHandlerCreation(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewPIPContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "pip-container-handler", handler.Name())
	assert.Equal(t, "pip", handler.Type())
	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
}

// TestAPTHandlerCreation APT 핸들러 생성 테스트
func TestAPTHandlerCreation(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewAPTContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "apt-container-handler", handler.Name())
	assert.Equal(t, "apt", handler.Type())
	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
}

// TestMavenHandlerCreation Maven 핸들러 생성 테스트
func TestMavenHandlerCreation(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewMavenContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "maven-container-handler", handler.Name())
	assert.Equal(t, "maven", handler.Type())
	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
}

// TestNPMHandlerCreation NPM 핸들러 생성 테스트
func TestNPMHandlerCreation(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewNPMContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "npm-container-handler", handler.Name())
	assert.Equal(t, "npm", handler.Type())
	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
}

// TestDockerHandlerCreation Docker 핸들러 생성 테스트
func TestDockerHandlerCreation(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewDockerContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "docker-container-handler", handler.Name())
	assert.Equal(t, "docker", handler.Type())
	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
}

// TestYUMHandlerCreation YUM 핸들러 생성 테스트
func TestYUMHandlerCreation(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "yum-container-handler", handler.Name())
	assert.Equal(t, "yum", handler.Type())
	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
}

// TestAPKHandlerCreation APK 핸들러 생성 테스트
func TestAPKHandlerCreation(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewAPKContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "apk-container-handler", handler.Name())
	assert.Equal(t, "apk", handler.Type())
	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
}

// TestHandlerConfigErrorScenarios 설정 오류 시나리오 테스트
func TestHandlerConfigErrorScenarios(t *testing.T) {
	testCases := []struct {
		name           string
		proxyType      string
		handlerCreator func(*testutil.MockContainerProvider) interface{}
		configMethod   string
	}{
		{
			name:      "PIP Handler Config Error",
			proxyType: "pip",
			handlerCreator: func(mock *testutil.MockContainerProvider) interface{} {
				return containerhandlers.NewPIPContainerHandler(mock)
			},
			configMethod: "GetPipProxyConfig",
		},
		{
			name:      "APT Handler Config Error",
			proxyType: "apt",
			handlerCreator: func(mock *testutil.MockContainerProvider) interface{} {
				return containerhandlers.NewAPTContainerHandler(mock)
			},
			configMethod: "GetAptProxyConfig",
		},
		{
			name:      "Maven Handler Config Error",
			proxyType: "maven",
			handlerCreator: func(mock *testutil.MockContainerProvider) interface{} {
				return containerhandlers.NewMavenContainerHandler(mock)
			},
			configMethod: "GetMavenProxyConfig",
		},
		{
			name:      "YUM Handler Config Error",
			proxyType: "yum",
			handlerCreator: func(mock *testutil.MockContainerProvider) interface{} {
				return containerhandlers.NewYUMContainerHandler(mock)
			},
			configMethod: "GetYumProxyConfig",
		},
		{
			name:      "APK Handler Config Error",
			proxyType: "apk",
			handlerCreator: func(mock *testutil.MockContainerProvider) interface{} {
				return containerhandlers.NewAPKContainerHandler(mock)
			},
			configMethod: "GetApkProxyConfig",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockContainer := testutil.NewMockContainerProvider(t).WithError(tc.configMethod, assert.AnError)
			handler := tc.handlerCreator(mockContainer)

			// IsEnabled() 호출 시 설정 로딩 실패로 false 반환해야 함
			if enabledChecker, ok := handler.(interface{ IsEnabled() bool }); ok {
				assert.False(t, enabledChecker.IsEnabled())
			}
		})
	}
}

// TestHandlerContainerProvider Container Provider 설정 테스트
func TestHandlerContainerProvider(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewPIPContainerHandler(mockContainer)

	// Container 설정 및 조회 테스트
	handler.SetContainer(mockContainer)
	retrievedContainer := handler.GetContainer()
	assert.Equal(t, mockContainer, retrievedContainer)
}

// TestHandlerStorageDirectories 스토리지 디렉토리 테스트
func TestHandlerStorageDirectories(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)

	// 커스텀 디렉토리 설정
	customStorageDir := "/tmp/test-storage"
	customConfigDir := "/tmp/test-config"

	mockContainer.WithStorageDir(customStorageDir).WithConfigDir(customConfigDir)

	assert.Equal(t, customStorageDir, mockContainer.GetStorageDir())
	assert.Equal(t, customConfigDir, mockContainer.GetConfigDir())
}

// TestMockContainerProviderChaining Mock Container Provider 체이닝 테스트
func TestMockContainerProviderChaining(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t).
		WithStorageDir("/custom/storage").
		WithConfigDir("/custom/config").
		WithError("GetPipProxyConfig", assert.AnError)

	assert.Equal(t, "/custom/storage", mockContainer.GetStorageDir())
	assert.Equal(t, "/custom/config", mockContainer.GetConfigDir())

	// PIP 설정은 오류가 발생해야 함
	_, err := mockContainer.GetPipProxyConfig()
	assert.Error(t, err)

	// 다른 설정들은 정상 동작해야 함
	aptConfig, err := mockContainer.GetAptProxyConfig()
	assert.NoError(t, err)
	assert.NotNil(t, aptConfig)
}
