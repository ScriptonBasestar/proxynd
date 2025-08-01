package containerhandlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"proxynd/internal/config"
	"proxynd/internal/containerhandlers"
	"proxynd/internal/testutil"
)

func TestNewAPKContainerHandler(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewAPKContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "apk-container-handler", handler.Name())
	assert.Equal(t, "apk", handler.Type())
	assert.True(t, handler.IsEnabled())
}

func TestNewAPKContainerHandlerWithConfigError(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t).
		WithError("GetApkProxyConfig", assert.AnError)

	handler := containerhandlers.NewAPKContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.False(t, handler.IsEnabled())
}

func TestAPKContainerHandlerHealthCheck(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewAPKContainerHandler(mockContainer)

	err := handler.HealthCheck()
	assert.NoError(t, err)
}

func TestAPKContainerHandlerHealthCheckDisabled(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t).
		WithError("GetApkProxyConfig", assert.AnError)

	handler := containerhandlers.NewAPKContainerHandler(mockContainer)
	
	err := handler.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestAPKContainerHandlerHealthCheckWithVerification(t *testing.T) {
	customConfig := &config.ApkProxySettings{
		Path:     "apk",
		UseCache: true,
		Proxies: []config.ApkProxy{
			{Name: "alpine", URL: "https://dl-cdn.alpinelinux.org/alpine"},
		},
		Verification: config.ApkVerificationConfig{
			Enabled:      true,
			KeyDirectory: "", // 키 디렉토리가 없음
		},
	}

	mockContainer := testutil.NewMockContainerProvider(t).
		WithConfig("apk", customConfig)

	handler := containerhandlers.NewAPKContainerHandler(mockContainer)
	
	err := handler.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key directory not specified")
}

func TestAPKContainerHandlerLoadConfig(t *testing.T) {
	customConfig := &config.ApkProxySettings{
		Path:     "custom-apk",
		UseCache: true,
		Proxies: []config.ApkProxy{
			{Name: "custom", URL: "https://custom.alpine.repo"},
		},
		Verification: config.ApkVerificationConfig{
			Enabled:        true,
			KeyDirectory:   "/etc/apk/keys",
			FailOnInvalid:  true,
			CacheValidated: true,
		},
		MirrorSelection: config.ApkMirrorSelectionConfig{
			Enabled:             true,
			HealthCheckInterval: "5m",
			HealthCheckTimeout:  "10s",
			PreferredRegions:    []string{"asia", "europe"},
			FallbackToGlobal:    true,
			MaxErrorCount:       3,
		},
	}

	mockContainer := testutil.NewMockContainerProvider(t).
		WithConfig("apk", customConfig)

	handler := containerhandlers.NewAPKContainerHandler(mockContainer)

	assert.True(t, handler.IsEnabled())
	// private 필드에 직접 접근할 수 없으므로, 설정이 제대로 로딩되었는지 간접적으로 확인
	assert.NoError(t, handler.HealthCheck())
}

func TestAPKContainerHandlerReloadConfig(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewAPKContainerHandler(mockContainer)
	
	assert.True(t, handler.IsEnabled())

	// 설정 재로딩
	err := handler.ReloadConfig()
	assert.NoError(t, err)
	assert.True(t, handler.IsEnabled())
}

func TestAPKContainerHandlerSetGetContainer(t *testing.T) {
	mockContainer1 := testutil.NewMockContainerProvider(t)
	mockContainer2 := testutil.NewMockContainerProvider(t)
	
	handler := containerhandlers.NewAPKContainerHandler(mockContainer1)
	
	assert.Equal(t, mockContainer1, handler.GetContainer())
	
	handler.SetContainer(mockContainer2)
	assert.Equal(t, mockContainer2, handler.GetContainer())
}

func TestAPKContainerHandlerConfigErrorReload(t *testing.T) {
	// 처음에는 정상적인 설정
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewAPKContainerHandler(mockContainer)
	
	assert.True(t, handler.IsEnabled())
	
	// 설정 오류가 있는 새로운 Container로 교체
	errorContainer := testutil.NewMockContainerProvider(t).
		WithError("GetApkProxyConfig", assert.AnError)
	
	handler.SetContainer(errorContainer)
	
	// 재로딩 시 에러 발생하고 비활성화
	err := handler.ReloadConfig()
	assert.Error(t, err)
	assert.False(t, handler.IsEnabled())
}

func TestAPKContainerHandlerWithComplexConfiguration(t *testing.T) {
	complexConfig := &config.ApkProxySettings{
		Path:     "apk",
		UseCache: true,
		Proxies: []config.ApkProxy{
			{Name: "main", URL: "https://dl-cdn.alpinelinux.org/alpine"},
			{Name: "mirror1", URL: "https://mirror1.alpinelinux.org/alpine"},
			{Name: "mirror2", URL: "https://mirror2.alpinelinux.org/alpine"},
		},
		Verification: config.ApkVerificationConfig{
			Enabled:        true,
			KeyDirectory:   "/etc/apk/keys",
			FailOnInvalid:  false, // 검증 실패해도 계속 진행
			CacheValidated: true,
		},
		MirrorSelection: config.ApkMirrorSelectionConfig{
			Enabled:             true,
			HealthCheckInterval: "3m",
			HealthCheckTimeout:  "5s",
			PreferredRegions:    []string{"asia-pacific", "north-america"},
			FallbackToGlobal:    true,
			MaxErrorCount:       5,
			RegionDetectionMode: "auto",
		},
	}

	mockContainer := testutil.NewMockContainerProvider(t).
		WithConfig("apk", complexConfig)

	handler := containerhandlers.NewAPKContainerHandler(mockContainer)

	assert.True(t, handler.IsEnabled())
	assert.NoError(t, handler.HealthCheck())
	
	// 다양한 기본 기능 테스트
	assert.Equal(t, "apk-container-handler", handler.Name())
	assert.Equal(t, "apk", handler.Type())
	
	// Container Provider 확인
	assert.Equal(t, mockContainer, handler.GetContainer())
}

func TestAPKContainerHandlerEmptyProxies(t *testing.T) {
	emptyConfig := &config.ApkProxySettings{
		Path:     "apk",
		UseCache: true,
		Proxies:  []config.ApkProxy{}, // 빈 프록시 목록
	}

	mockContainer := testutil.NewMockContainerProvider(t).
		WithConfig("apk", emptyConfig)

	handler := containerhandlers.NewAPKContainerHandler(mockContainer)

	assert.False(t, handler.IsEnabled()) // 프록시가 없으면 비활성화
	
	err := handler.HealthCheck()
	assert.Error(t, err)
	// 핸들러가 비활성화된 경우 "disabled" 에러가 먼저 반환됨
	assert.Contains(t, err.Error(), "disabled")
}