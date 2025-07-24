package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/configs"
)

// Test constants
const (
	testConfigContent = `
server:
  host: "localhost"
  port: 8080
logging:
  level: "info"
`
)

func TestContainer_GetUnifiedConfig(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	cfg := &Config{
		ConfigDir: tempDir,
		Port:      "8080",
	}

	// 테스트용 설정 파일 생성 (기본 설정 파일명 사용)
	configContent := `
server:
  host: "localhost"
  port: 8080
cache:
  backend: "file"
logging:
  level: "info"
`
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	container := NewContainer(cfg)
	defer func() { _ = container.Close() }()

	// When
	unifiedConfig := container.GetUnifiedConfig()

	// Then
	assert.NotNil(t, unifiedConfig)
	assert.Equal(t, "localhost", unifiedConfig.Server.Host)
	assert.Equal(t, 8080, unifiedConfig.Server.Port)
	assert.Equal(t, "file", unifiedConfig.Cache.Backend)
	assert.Equal(t, "info", unifiedConfig.Logging.Level)
}

func TestContainer_ReloadConfig(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	cfg := &Config{
		ConfigDir: tempDir,
		Port:      "8080",
	}

	// 초기 설정 파일 생성
	configContent := testConfigContent
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	container := NewContainer(cfg)
	defer func() { _ = container.Close() }()

	// 초기 설정 확인
	initialConfig := container.GetUnifiedConfig()
	assert.Equal(t, "info", initialConfig.Logging.Level)

	// 설정 변경
	newConfigContent := `
server:
  host: "localhost"
  port: 8080
logging:
  level: "debug"
`
	err = os.WriteFile(configFile, []byte(newConfigContent), 0o644)
	require.NoError(t, err)

	// When
	err = container.ReloadConfig()

	// Then
	assert.NoError(t, err)
	updatedConfig := container.GetUnifiedConfig()
	assert.Equal(t, "debug", updatedConfig.Logging.Level)
}

func TestContainer_ConfigChangeCallback(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	cfg := &Config{
		ConfigDir: tempDir,
		Port:      "8080",
	}

	// 테스트용 설정 파일 생성
	configContent := testConfigContent
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	container := NewContainer(cfg)
	defer func() { _ = container.Close() }()

	// 콜백 등록
	callbackCalled := false
	var callbackConfig *configs.UnifiedConfig
	container.AddConfigChangeCallback(func(config *configs.UnifiedConfig) {
		callbackCalled = true
		callbackConfig = config
	})

	// 설정 변경
	newConfigContent := `
server:
  host: "localhost"
  port: 8080
logging:
  level: "warn"
`
	err = os.WriteFile(configFile, []byte(newConfigContent), 0o644)
	require.NoError(t, err)

	// When
	err = container.ReloadConfig()

	// Then
	assert.NoError(t, err)

	// 콜백이 비동기로 실행되므로 잠시 대기
	time.Sleep(100 * time.Millisecond)

	assert.True(t, callbackCalled)
	assert.NotNil(t, callbackConfig)
	assert.Equal(t, "warn", callbackConfig.Logging.Level)
}

func TestContainer_InvalidConfig(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	cfg := &Config{
		ConfigDir: tempDir,
		Port:      "8080",
	}

	// 잘못된 설정 파일 생성
	invalidConfigContent := `
server:
  host: "localhost"
  port: 99999  # 잘못된 포트
logging:
  level: "invalid_level"  # 잘못된 로그 레벨
`
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(invalidConfigContent), 0o644)
	require.NoError(t, err)

	container := NewContainer(cfg)
	defer func() { _ = container.Close() }()

	// When
	err = container.ReloadConfig()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "포트 번호가 유효하지 않습니다")
}

func TestContainer_ConfigFileWatcher(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	cfg := &Config{
		ConfigDir: tempDir,
		Port:      "8080",
	}

	// 초기 설정 파일 생성
	configContent := testConfigContent
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	container := NewContainer(cfg)
	defer func() { _ = container.Close() }()

	// 콜백 등록
	callbackCalled := false
	container.AddConfigChangeCallback(func(_ *configs.UnifiedConfig) {
		callbackCalled = true
	})

	// 초기 설정 확인
	initialConfig := container.GetUnifiedConfig()
	assert.Equal(t, "info", initialConfig.Logging.Level)

	// When: 파일 변경 (파일 감시에 의한 자동 리로드)
	newConfigContent := `
server:
  host: "localhost"
  port: 8080
logging:
  level: "debug"
`
	err = os.WriteFile(configFile, []byte(newConfigContent), 0o644)
	require.NoError(t, err)

	// 파일 감시 이벤트 처리 대기
	time.Sleep(500 * time.Millisecond)

	// Then
	updatedConfig := container.GetUnifiedConfig()
	assert.Equal(t, "debug", updatedConfig.Logging.Level)
	assert.True(t, callbackCalled)
}

func TestContainer_ConfigCaching(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	cfg := &Config{
		ConfigDir: tempDir,
		Port:      "8080",
	}

	// 테스트용 설정 파일 생성
	configContent := `
server:
  host: "localhost"
  port: 8080
`
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	container := NewContainer(cfg)
	defer func() { _ = container.Close() }()

	// When: 여러 번 설정 조회
	config1 := container.GetUnifiedConfig()
	config2 := container.GetUnifiedConfig()
	config3 := container.GetUnifiedConfig()

	// Then: 같은 인스턴스 반환 (캐싱 동작)
	assert.Same(t, config1, config2)
	assert.Same(t, config2, config3)
}

func TestContainer_Close(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	cfg := &Config{
		ConfigDir: tempDir,
		Port:      "8080",
	}

	// 테스트용 설정 파일 생성
	configContent := `
server:
  host: "localhost"
  port: 8080
`
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	container := NewContainer(cfg)

	// When
	err = container.Close()

	// Then
	assert.NoError(t, err)
	// 정리가 제대로 되었는지 확인
	assert.Nil(t, container.configWatcher)
	assert.Nil(t, container.configCache)
	assert.Nil(t, container.configLoader)
	assert.Nil(t, container.configChangeCbs)
}
