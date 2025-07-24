package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"

	"proxynd/configs"
)

func TestUnifiedConfig_LoadDefaults(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 설정 로더 생성 (파일 없음 - 기본값 사용)
	loader := configs.NewConfigLoader(configPath)
	config, err := loader.Load()

	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, config)

	// 기본값 확인
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 30*time.Second, config.Server.ReadTimeout)
	assert.Equal(t, true, config.Server.EnableHTTP2)

	assert.Equal(t, "file", config.Cache.Backend)
	assert.Equal(t, 3600*time.Second, config.Cache.TTL)
	assert.Equal(t, "lru", config.Cache.EvictionPolicy)

	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
	assert.Equal(t, "/metrics", config.Metrics.Path)
}

func TestUnifiedConfig_LoadFromFile(t *testing.T) {
	// 임시 설정 파일 생성
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  host: 127.0.0.1
  port: 9090
  tls:
    enabled: true
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem

cache:
  backend: s3
  ttl: 7200s
  s3:
    bucket: test-bucket
    region: us-west-2

logging:
  level: debug
  format: text

registries:
  npm:
    enabled: true
    upstream: https://registry.npmjs.org
  pypi:
    enabled: false
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.Equal(t, nil, err)

	// 설정 로드
	loader := configs.NewConfigLoader(configPath)
	config, err := loader.Load()

	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, config)

	// 로드된 값 확인
	assert.Equal(t, "127.0.0.1", config.Server.Host)
	assert.Equal(t, 9090, config.Server.Port)
	assert.Equal(t, true, config.Server.TLS.Enabled)
	assert.Equal(t, "/path/to/cert.pem", config.Server.TLS.CertFile)

	assert.Equal(t, "s3", config.Cache.Backend)
	assert.Equal(t, 7200*time.Second, config.Cache.TTL)
	assert.Equal(t, "test-bucket", config.Cache.S3.Bucket)
	assert.Equal(t, "us-west-2", config.Cache.S3.Region)

	assert.Equal(t, "debug", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)

	assert.Equal(t, true, config.Registries.NPM.Enabled)
	assert.Equal(t, false, config.Registries.PyPI.Enabled)
}

func TestUnifiedConfig_EnvironmentOverrides(t *testing.T) {
	// 환경 변수 설정
	_ = os.Setenv("SERVER_PORT", "8888")
	_ = os.Setenv("LOG_LEVEL", "error")
	_ = os.Setenv("CACHE_BACKEND", "redis")
	_ = os.Setenv("STORAGE_DIR", "/custom/storage")
	_ = os.Setenv("METRICS_ENABLED", "true")

	defer func() {
		_ = os.Unsetenv("SERVER_PORT")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("CACHE_BACKEND")
		_ = os.Unsetenv("STORAGE_DIR")
		_ = os.Unsetenv("METRICS_ENABLED")
	}()

	// 임시 설정 파일 생성
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  port: 9090

cache:
  backend: file

logging:
  level: info

metrics:
  enabled: false
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.Equal(t, nil, err)

	// 설정 로드
	loader := configs.NewConfigLoader(configPath)
	config, err := loader.Load()

	assert.Equal(t, nil, err)

	// 환경 변수 오버라이드 확인
	assert.Equal(t, 8888, config.Server.Port) // 환경 변수가 우선
	assert.Equal(t, "error", config.Logging.Level)
	assert.Equal(t, "redis", config.Cache.Backend)
	assert.Equal(t, "/custom/storage", config.Cache.File.Directory)
	assert.Equal(t, true, config.Metrics.Enabled)
}

func TestUnifiedConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *configs.UnifiedConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Invalid port",
			config: &configs.UnifiedConfig{
				Server: configs.ServerConfig{
					Port: 99999,
				},
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "TLS enabled without cert",
			config: &configs.UnifiedConfig{
				Server: configs.ServerConfig{
					Port: 8080,
					TLS: configs.TLSConfig{
						Enabled: true,
						KeyFile: "key.pem",
					},
				},
			},
			expectError: true,
			errorMsg:    "cert file not specified",
		},
		{
			name: "Invalid cache backend",
			config: &configs.UnifiedConfig{
				Server: configs.ServerConfig{
					Port: 8080,
				},
				Cache: configs.CacheConfig{
					Backend: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "unsupported cache backend",
		},
		{
			name: "Invalid log level",
			config: &configs.UnifiedConfig{
				Server: configs.ServerConfig{
					Port: 8080,
				},
				Cache: configs.CacheConfig{
					Backend: "file",
				},
				Logging: configs.UnifiedLoggingConfig{
					Level: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.NotEqual(t, nil, err)
				assert.Equal(t, true, err.Error() != "")
			} else {
				assert.Equal(t, nil, err)
			}
		})
	}
}

func TestEnvOverride_GetEnvironmentOverrides(t *testing.T) {
	// 환경 변수 설정
	_ = os.Setenv("SERVER_PORT", "8080")
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("CACHE_BACKEND", "s3")

	defer func() {
		_ = os.Unsetenv("SERVER_PORT")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("CACHE_BACKEND")
	}()

	overrides := configs.GetEnvironmentOverrides()

	assert.Equal(t, "8080", overrides["SERVER_PORT"])
	assert.Equal(t, "debug", overrides["LOG_LEVEL"])
	assert.Equal(t, "s3", overrides["CACHE_BACKEND"])
}

func TestConfigLoader_EnvironmentVariableExpansion(t *testing.T) {
	// 환경 변수 설정
	_ = os.Setenv("TEST_BUCKET", "my-test-bucket")
	_ = os.Setenv("TEST_REGION", "us-east-1")

	defer func() {
		_ = os.Unsetenv("TEST_BUCKET")
		_ = os.Unsetenv("TEST_REGION")
	}()

	// 임시 설정 파일 생성
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
cache:
  backend: s3
  s3:
    bucket: ${TEST_BUCKET}
    region: ${TEST_REGION}
    endpoint: ${S3_ENDPOINT}
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.Equal(t, nil, err)

	// 설정 로드
	loader := configs.NewConfigLoader(configPath)
	config, err := loader.Load()

	assert.Equal(t, nil, err)

	// 환경 변수 치환 확인
	assert.Equal(t, "my-test-bucket", config.Cache.S3.Bucket)
	assert.Equal(t, "us-east-1", config.Cache.S3.Region)
	assert.Equal(t, "${S3_ENDPOINT}", config.Cache.S3.Endpoint) // 설정되지 않은 변수는 그대로
}

func TestHotReloadManager_Basic(t *testing.T) {
	// 임시 설정 파일 생성
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	initialConfig := `
server:
  port: 8080
logging:
  level: info
`

	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	assert.Equal(t, nil, err)

	// 핫리로드 매니저 생성
	manager, err := configs.NewHotReloadManager(configPath)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, manager)

	// 초기 설정 확인
	config := manager.GetConfig()
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "info", config.Logging.Level)

	// 테스트 리로드 핸들러
	reloadCount := 0
	testHandler := &testReloadHandler{
		name: "TestHandler",
		onReload: func(_, _ *configs.UnifiedConfig) error {
			reloadCount++
			return nil
		},
	}

	manager.RegisterReloadHandler(testHandler)

	// 핫리로드 시작
	err = manager.Start()
	assert.Equal(t, nil, err)

	// 설정 파일 수정
	updatedConfig := `
server:
  port: 9090
logging:
  level: debug
`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0o644)
	assert.Equal(t, nil, err)

	// 리로드 대기
	time.Sleep(1 * time.Second)

	// 설정 변경 확인
	config = manager.GetConfig()
	assert.Equal(t, 9090, config.Server.Port)
	assert.Equal(t, "debug", config.Logging.Level)
	assert.Equal(t, 1, reloadCount)

	// 정리
	err = manager.Stop()
	assert.Equal(t, nil, err)
}

// 테스트용 리로드 핸들러
type testReloadHandler struct {
	name     string
	onReload func(old, newVal *configs.UnifiedConfig) error
}

func (h *testReloadHandler) OnConfigReload(old, newVal *configs.UnifiedConfig) error {
	return h.onReload(old, newVal)
}

func (h *testReloadHandler) Name() string {
	return h.name
}
