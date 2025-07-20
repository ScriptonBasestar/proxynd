package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ViperConfigLoader Viper 기반 설정 로더
type ViperConfigLoader struct {
	viper      *viper.Viper
	configPath string
	configName string
	configType string
	config     *UnifiedConfig
}

// NewViperConfigLoader 새 Viper 설정 로더 생성
func NewViperConfigLoader() *ViperConfigLoader {
	v := viper.New()

	// 환경 변수 설정
	v.SetEnvPrefix("PROXYND")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &ViperConfigLoader{
		viper:      v,
		configType: "yaml",
	}
}

// SetConfigPath 설정 파일 경로 설정
func (vcl *ViperConfigLoader) SetConfigPath(path string) {
	vcl.configPath = path

	// 파일명과 디렉토리 분리
	dir := filepath.Dir(path)
	file := filepath.Base(path)
	ext := filepath.Ext(file)
	name := strings.TrimSuffix(file, ext)

	vcl.configName = name
	vcl.viper.SetConfigName(name)
	vcl.viper.SetConfigType(strings.TrimPrefix(ext, "."))
	vcl.viper.AddConfigPath(dir)
}

// Load 설정 로드 (환경별 설정 포함)
func (vcl *ViperConfigLoader) Load() (*UnifiedConfig, error) {
	// 기본값 설정
	vcl.setDefaults()

	// 기본 설정 파일 로드
	if err := vcl.loadConfigFile(); err != nil {
		// 설정 파일이 없어도 계속 진행 (환경 변수만 사용)
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// 환경별 설정 파일 로드
	if err := vcl.loadEnvironmentConfig(); err != nil {
		// 환경별 설정 파일은 선택사항
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading environment config: %w", err)
		}
	}

	// .env 파일 로드 (있는 경우)
	vcl.loadDotEnvFile()

	// 환경 변수 바인딩
	vcl.bindEnvironmentVariables()

	// 설정을 구조체로 언마샬
	config := &UnifiedConfig{}
	if err := vcl.viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// 설정 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	vcl.config = config
	return config, nil
}

// loadConfigFile 기본 설정 파일 로드
func (vcl *ViperConfigLoader) loadConfigFile() error {
	return vcl.viper.ReadInConfig()
}

// loadEnvironmentConfig 환경별 설정 파일 로드
func (vcl *ViperConfigLoader) loadEnvironmentConfig() error {
	env := os.Getenv("PROXYND_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "development"
	}

	// 환경별 설정 파일명 설정
	envConfigName := fmt.Sprintf("%s.%s", vcl.configName, env)
	vcl.viper.SetConfigName(envConfigName)

	// 환경별 설정을 기존 설정에 병합
	return vcl.viper.MergeInConfig()
}

// loadDotEnvFile .env 파일 로드
func (vcl *ViperConfigLoader) loadDotEnvFile() {
	// .env 파일 경로 찾기
	envFiles := []string{
		".env",
		".env.local",
	}

	// 환경별 .env 파일도 확인
	if env := os.Getenv("PROXYND_ENV"); env != "" {
		envFiles = append(envFiles, fmt.Sprintf(".env.%s", env))
	}

	// 설정 디렉토리의 .env 파일 확인
	if vcl.configPath != "" {
		dir := filepath.Dir(vcl.configPath)
		for _, envFile := range envFiles {
			envPath := filepath.Join(dir, envFile)
			if _, err := os.Stat(envPath); err == nil {
				vcl.viper.SetConfigFile(envPath)
				vcl.viper.SetConfigType("dotenv")
				_ = vcl.viper.MergeInConfig() // 에러 무시 (선택사항)
			}
		}
	}
}

// bindEnvironmentVariables 환경 변수 바인딩
func (vcl *ViperConfigLoader) bindEnvironmentVariables() {
	// 주요 환경 변수 명시적 바인딩
	environmentBindings := map[string]string{
		"SERVER_HOST":           "server.host",
		"SERVER_PORT":           "server.port",
		"SERVER_READ_TIMEOUT":   "server.read_timeout",
		"SERVER_WRITE_TIMEOUT":  "server.write_timeout",
		"TLS_ENABLED":           "server.tls.enabled",
		"TLS_CERT_FILE":         "server.tls.cert_file",
		"TLS_KEY_FILE":          "server.tls.key_file",
		"CACHE_BACKEND":         "cache.backend",
		"CACHE_TTL":             "cache.ttl",
		"CACHE_MAX_SIZE":        "cache.max_size",
		"STORAGE_DIR":           "cache.file.directory",
		"S3_ENDPOINT":           "cache.s3.endpoint",
		"S3_BUCKET":             "cache.s3.bucket",
		"S3_REGION":             "cache.s3.region",
		"AWS_ACCESS_KEY_ID":     "cache.s3.access_key_id",
		"AWS_SECRET_ACCESS_KEY": "cache.s3.secret_access_key",
		"REDIS_ADDRESS":         "cache.redis.address",
		"REDIS_PASSWORD":        "cache.redis.password",
		"REDIS_DB":              "cache.redis.db",
		"LOG_LEVEL":             "logging.level",
		"LOG_FORMAT":            "logging.format",
		"LOG_OUTPUT":            "logging.output",
		"METRICS_ENABLED":       "metrics.enabled",
		"METRICS_PATH":          "metrics.path",
		"METRICS_PORT":          "metrics.port",
		"AUTH_ENABLED":          "security.authentication.basic_auth.enabled",
		"AUTH_USERS_FILE":       "security.authentication.basic_auth.users_file",
	}

	for envVar, configPath := range environmentBindings {
		_ = vcl.viper.BindEnv(configPath, envVar)
	}
}

// setDefaults 기본값 설정
func (vcl *ViperConfigLoader) setDefaults() {
	// 서버 기본값
	vcl.viper.SetDefault("server.host", "0.0.0.0")
	vcl.viper.SetDefault("server.port", 8080)
	vcl.viper.SetDefault("server.read_timeout", "30s")
	vcl.viper.SetDefault("server.write_timeout", "30s")
	vcl.viper.SetDefault("server.idle_timeout", "120s")
	vcl.viper.SetDefault("server.enable_http2", true)
	vcl.viper.SetDefault("server.tls.min_version", "TLS1.2")

	// 압축 기본값
	vcl.viper.SetDefault("server.compression.enabled", true)
	vcl.viper.SetDefault("server.compression.level", 5)
	vcl.viper.SetDefault("server.compression.types", []string{
		"application/json",
		"application/javascript",
		"text/html",
		"text/css",
		"text/plain",
	})

	// 캐시 기본값
	vcl.viper.SetDefault("cache.backend", "file")
	vcl.viper.SetDefault("cache.ttl", "3600s")
	vcl.viper.SetDefault("cache.max_size", "10GB")
	vcl.viper.SetDefault("cache.cleanup_interval", "1h")
	vcl.viper.SetDefault("cache.eviction_policy", "lru")
	vcl.viper.SetDefault("cache.file.max_file_size", "1GB")

	// 레지스트리 기본값
	vcl.viper.SetDefault("registries.npm.upstream", "https://registry.npmjs.org")
	vcl.viper.SetDefault("registries.npm.timeout", "30s")
	vcl.viper.SetDefault("registries.pypi.upstream", "https://pypi.org")
	vcl.viper.SetDefault("registries.pypi.simple", "https://pypi.org/simple")
	vcl.viper.SetDefault("registries.pypi.timeout", "30s")
	vcl.viper.SetDefault("registries.docker.use_cache", true)

	// 보안 기본값
	vcl.viper.SetDefault("security.authentication.basic_auth.users_file", "users.yml")
	vcl.viper.SetDefault("security.authentication.basic_auth.realm", "ProxyND")
	vcl.viper.SetDefault("security.authentication.token_auth.header_name", "X-Auth-Token")
	vcl.viper.SetDefault("security.authentication.ldap.port", 389)
	vcl.viper.SetDefault("security.package_filter.mode", "allowlist")

	// 로깅 기본값
	vcl.viper.SetDefault("logging.level", "info")
	vcl.viper.SetDefault("logging.format", "json")
	vcl.viper.SetDefault("logging.output", "stdout")
	vcl.viper.SetDefault("logging.file.max_size", "100MB")
	vcl.viper.SetDefault("logging.file.max_backups", 10)
	vcl.viper.SetDefault("logging.file.max_age", 30)
	vcl.viper.SetDefault("logging.file.compress", true)
	vcl.viper.SetDefault("logging.access_log.enabled", true)
	vcl.viper.SetDefault("logging.access_log.format", "json")
	vcl.viper.SetDefault("logging.access_log.rotate_daily", true)

	// 메트릭 기본값
	vcl.viper.SetDefault("metrics.path", "/metrics")

	// 고급 설정 기본값
	vcl.viper.SetDefault("advanced.performance.max_connections", 1000)
	vcl.viper.SetDefault("advanced.performance.max_idle_connections", 100)
	vcl.viper.SetDefault("advanced.performance.connection_timeout", "30s")
	vcl.viper.SetDefault("advanced.performance.keep_alive", "30s")
	vcl.viper.SetDefault("advanced.performance.buffer_size", 4096)
	vcl.viper.SetDefault("advanced.retry.max_attempts", 3)
	vcl.viper.SetDefault("advanced.retry.initial_delay", "1s")
	vcl.viper.SetDefault("advanced.retry.max_delay", "30s")
	vcl.viper.SetDefault("advanced.retry.multiplier", 2.0)
	vcl.viper.SetDefault("advanced.circuit_breaker.failure_threshold", 5)
	vcl.viper.SetDefault("advanced.circuit_breaker.success_threshold", 2)
	vcl.viper.SetDefault("advanced.circuit_breaker.timeout", "60s")
}

// GetConfig 현재 설정 반환
func (vcl *ViperConfigLoader) GetConfig() *UnifiedConfig {
	return vcl.config
}

// GetViper 내부 Viper 인스턴스 반환
func (vcl *ViperConfigLoader) GetViper() *viper.Viper {
	return vcl.viper
}

// WatchConfig 설정 변경 감시
func (vcl *ViperConfigLoader) WatchConfig(callback func(*UnifiedConfig)) {
	vcl.viper.WatchConfig()
	vcl.viper.OnConfigChange(func(_ fsnotify.Event) {
		// 설정 재로드
		config := &UnifiedConfig{}
		if err := vcl.viper.Unmarshal(config); err == nil {
			if err := config.Validate(); err == nil {
				vcl.config = config
				callback(config)
			}
		}
	})
}

// GetString 문자열 설정값 조회
func (vcl *ViperConfigLoader) GetString(key string) string {
	return vcl.viper.GetString(key)
}

// GetInt 정수 설정값 조회
func (vcl *ViperConfigLoader) GetInt(key string) int {
	return vcl.viper.GetInt(key)
}

// GetBool 불린 설정값 조회
func (vcl *ViperConfigLoader) GetBool(key string) bool {
	return vcl.viper.GetBool(key)
}

// IsSet 설정값 존재 여부 확인
func (vcl *ViperConfigLoader) IsSet(key string) bool {
	return vcl.viper.IsSet(key)
}

// AllSettings 모든 설정 반환
func (vcl *ViperConfigLoader) AllSettings() map[string]interface{} {
	return vcl.viper.AllSettings()
}

// Debug 디버그 정보 출력
func (vcl *ViperConfigLoader) Debug() {
	fmt.Println("=== Viper Configuration Debug ===")
	fmt.Printf("Config Name: %s\n", vcl.configName)
	fmt.Printf("Config Type: %s\n", vcl.configType)
	fmt.Printf("Config Path: %s\n", vcl.configPath)
	fmt.Printf("Environment: %s\n", os.Getenv("PROXYND_ENV"))
	fmt.Println("\nLoaded Sources:")
	for _, cp := range vcl.viper.ConfigFileUsed() {
		fmt.Printf("  - %c\n", cp)
	}
	fmt.Println("\nEnvironment Variables:")
	for k, v := range vcl.viper.AllSettings() {
		if vcl.viper.IsSet(k) {
			fmt.Printf("  %s = %v\n", k, v)
		}
	}
}
