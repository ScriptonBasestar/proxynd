package containerhandlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"proxynd/internal/config"
	"proxynd/internal/containerhandlers"
	"proxynd/internal/testutil"
)

func TestNewYUMContainerHandler(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.Equal(t, "yum-container-handler", handler.Name())
	assert.Equal(t, "yum", handler.Type())
	assert.True(t, handler.IsEnabled())
}

func TestNewYUMContainerHandlerWithConfigError(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t).
		WithError("GetYumProxyConfig", assert.AnError)

	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	assert.NotNil(t, handler)
	assert.False(t, handler.IsEnabled())
}

func TestYUMContainerHandlerHealthCheck(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	err := handler.HealthCheck()
	assert.NoError(t, err)
}

func TestYUMContainerHandlerHealthCheckDisabled(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t).
		WithError("GetYumProxyConfig", assert.AnError)

	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	err := handler.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestYUMContainerHandlerLoadConfig(t *testing.T) {
	customConfig := &config.YumProxySettings{
		Path:     "custom-yum",
		UseCache: true,
		Proxies: []config.YumProxy{
			{Name: "custom", URL: "https://custom.yum.repo"},
		},
	}

	mockContainer := testutil.NewMockContainerProvider(t).
		WithConfig("yum", customConfig)

	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	assert.True(t, handler.IsEnabled())
	// private 필드에 직접 접근할 수 없으므로, 설정이 제대로 로딩되었는지 간접적으로 확인
	assert.NoError(t, handler.HealthCheck())
}

func TestYUMContainerHandlerReloadConfig(t *testing.T) {
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	assert.True(t, handler.IsEnabled())

	// 설정 재로딩
	err := handler.ReloadConfig()
	assert.NoError(t, err)
	assert.True(t, handler.IsEnabled())
}

func TestYUMContainerHandlerSetGetContainer(t *testing.T) {
	mockContainer1 := testutil.NewMockContainerProvider(t)
	mockContainer2 := testutil.NewMockContainerProvider(t)

	handler := containerhandlers.NewYUMContainerHandler(mockContainer1)

	assert.Equal(t, mockContainer1, handler.GetContainer())

	handler.SetContainer(mockContainer2)
	assert.Equal(t, mockContainer2, handler.GetContainer())
}

func TestYUMContainerHandlerConfigErrorReload(t *testing.T) {
	// 처음에는 정상적인 설정
	mockContainer := testutil.NewMockContainerProvider(t)
	handler := containerhandlers.NewYUMContainerHandler(mockContainer)

	assert.True(t, handler.IsEnabled())

	// 설정 오류가 있는 새로운 Container로 교체
	errorContainer := testutil.NewMockContainerProvider(t).
		WithError("GetYumProxyConfig", assert.AnError)

	handler.SetContainer(errorContainer)

	// 재로딩 시 에러 발생하고 비활성화
	err := handler.ReloadConfig()
	assert.Error(t, err)
	assert.False(t, handler.IsEnabled())
}
