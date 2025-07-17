package configs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigLoader 설정 로더
type ConfigLoader struct {
	configPath string
	config     *UnifiedConfig
}

// NewConfigLoader 새 설정 로더 생성
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// Load 설정 파일 로드
func (cl *ConfigLoader) Load() (*UnifiedConfig, error) {
	// 기본 설정으로 초기화
	config := cl.newDefaultConfig()

	// 설정 파일이 없으면 기본값 사용
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		cl.config = config
		return config, nil
	}

	// YAML 파일 읽기
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 환경 변수 치환
	data = cl.expandEnvironmentVariables(data)

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 환경 변수 오버라이드 적용
	cl.applyEnvironmentOverrides(config)

	// 기본값 적용
	cl.applyDefaults(config)

	// 설정 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cl.config = config
	return config, nil
}

// Reload 설정 재로드
func (cl *ConfigLoader) Reload() error {
	newConfig, err := cl.Load()
	if err != nil {
		return err
	}

	cl.config = newConfig
	return nil
}

// GetConfig 현재 설정 반환
func (cl *ConfigLoader) GetConfig() *UnifiedConfig {
	return cl.config
}

// newDefaultConfig 기본 설정 생성
func (cl *ConfigLoader) newDefaultConfig() *UnifiedConfig {
	return &UnifiedConfig{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			EnableHTTP2:  true,
			Compression: CompressionConfig{
				Enabled: true,
				Level:   5,
				Types: []string{
					"application/json",
					"application/javascript",
					"text/html",
					"text/css",
					"text/plain",
				},
			},
			TLS: TLSConfig{
				MinVersion: "TLS1.2",
			},
		},
		Cache: CacheConfig{
			Backend:         "file",
			TTL:             3600 * time.Second,
			MaxSize:         "10GB",
			CleanupInterval: time.Hour,
			EvictionPolicy:  "lru",
			File: FileCacheConfig{
				Directory:   getDefaultCacheDir(),
				MaxFileSize: "1GB",
			},
		},
		Registries: RegistryConfig{
			NPM: NPMRegistryConfig{
				Upstream: "https://registry.npmjs.org",
				Timeout:  30 * time.Second,
			},
			PyPI: PyPIRegistryConfig{
				Upstream: "https://pypi.org",
				Simple:   "https://pypi.org/simple",
				Timeout:  30 * time.Second,
			},
			Docker: DockerRegistryConfig{
				Enabled:  true,
				UseCache: true,
				Registries: []DockerRegistryEndpoint{
					{
						Name: "docker-hub",
						URL:  "https://registry-1.docker.io",
					},
				},
			},
		},
		Security: SecurityConfig{
			Authentication: AuthenticationConfig{
				BasicAuth: &BasicAuthConfig{
					Realm: "ProxyND",
				},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
			AccessLog: AccessLogConfig{
				Enabled:     true,
				Format:      "json",
				RotateDaily: true,
			},
		},
		Metrics: MetricsConfig{
			Path: "/metrics",
		},
		Advanced: AdvancedConfig{
			Performance: PerformanceConfig{
				MaxConnections:     1000,
				MaxIdleConnections: 100,
				ConnectionTimeout:  30 * time.Second,
				KeepAlive:          30 * time.Second,
				BufferSize:         4096,
			},
			Retry: RetryConfig{
				MaxAttempts:  3,
				InitialDelay: time.Second,
				MaxDelay:     30 * time.Second,
				Multiplier:   2.0,
			},
			CircuitBreaker: CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Timeout:          60 * time.Second,
			},
		},
	}
}

// expandEnvironmentVariables 환경 변수 치환
func (cl *ConfigLoader) expandEnvironmentVariables(data []byte) []byte {
	content := string(data)

	// ${VAR} 형식의 환경 변수 치환
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		varName := match[2 : len(match)-1] // ${} 제거
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match
	})

	return []byte(content)
}

// applyEnvironmentOverrides 환경 변수 오버라이드 적용
func (cl *ConfigLoader) applyEnvironmentOverrides(config *UnifiedConfig) {
	// 서버 설정
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	// 캐시 설정
	if dir := os.Getenv("STORAGE_DIR"); dir != "" {
		config.Cache.File.Directory = expandHomePath(dir)
	}
	if backend := os.Getenv("CACHE_BACKEND"); backend != "" {
		config.Cache.Backend = backend
	}

	// 로깅 설정
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Logging.Format = format
	}

	// TLS 설정
	if certFile := os.Getenv("TLS_CERT_FILE"); certFile != "" {
		config.Server.TLS.CertFile = certFile
		config.Server.TLS.Enabled = true
	}
	if keyFile := os.Getenv("TLS_KEY_FILE"); keyFile != "" {
		config.Server.TLS.KeyFile = keyFile
	}

	// 메트릭 설정
	if enabled := os.Getenv("METRICS_ENABLED"); enabled == "true" {
		config.Metrics.Enabled = true
	}
	if port := os.Getenv("METRICS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Metrics.Port = p
		}
	}
}

// applyDefaults 기본값 적용
func (cl *ConfigLoader) applyDefaults(config *UnifiedConfig) {
	// 리플렉션을 사용한 재귀적 기본값 적용
	cl.applyDefaultsRecursive(reflect.ValueOf(config).Elem())
}

// applyDefaultsRecursive 재귀적으로 기본값 적용
func (cl *ConfigLoader) applyDefaultsRecursive(v reflect.Value) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 기본값 태그 확인
		if defaultTag := fieldType.Tag.Get("default"); defaultTag != "" {
			if cl.isZeroValue(field) {
				cl.setDefaultValue(field, defaultTag, fieldType.Type)
			}
		}

		// 중첩된 구조체 처리
		if field.Kind() == reflect.Struct && fieldType.Type.String() != "time.Duration" {
			cl.applyDefaultsRecursive(field)
		}
	}
}

// isZeroValue 제로값 확인
func (cl *ConfigLoader) isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return false
	}
}

// setDefaultValue 기본값 설정
func (cl *ConfigLoader) setDefaultValue(field reflect.Value, defaultValue string, fieldType reflect.Type) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fieldType.String() == "time.Duration" {
			if d, err := time.ParseDuration(defaultValue); err == nil {
				field.SetInt(int64(d))
			}
		} else {
			if i, err := strconv.ParseInt(defaultValue, 10, 64); err == nil {
				field.SetInt(i)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, err := strconv.ParseUint(defaultValue, 10, 64); err == nil {
			field.SetUint(i)
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			field.SetFloat(f)
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(defaultValue); err == nil {
			field.SetBool(b)
		}
	}
}

// getDefaultCacheDir 기본 캐시 디렉토리 반환
func getDefaultCacheDir() string {
	if dir := os.Getenv("STORAGE_DIR"); dir != "" {
		return expandHomePath(dir)
	}
	return filepath.Join(os.TempDir(), "proxynd-cache")
}

// expandHomePath 홈 디렉토리 경로 확장
func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// WatchConfig 설정 파일 변경 감시
func (cl *ConfigLoader) WatchConfig(callback func(*UnifiedConfig)) error {
	// TODO: 파일 시스템 감시 구현
	// fsnotify 패키지를 사용하여 설정 파일 변경 감지
	// 변경 시 Reload() 호출 후 callback 실행
	return nil
}

// ValidateConfig 설정 검증 헬퍼
func ValidateConfig(config *UnifiedConfig) []string {
	var errors []string

	// 서버 포트 검증
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		errors = append(errors, fmt.Sprintf("invalid server port: %d", config.Server.Port))
	}

	// TLS 설정 검증
	if config.Server.TLS.Enabled {
		if config.Server.TLS.CertFile == "" {
			errors = append(errors, "TLS enabled but cert file not specified")
		}
		if config.Server.TLS.KeyFile == "" {
			errors = append(errors, "TLS enabled but key file not specified")
		}
	}

	// 캐시 백엔드 검증
	validBackends := []string{"file", "s3", "redis"}
	backendValid := false
	for _, b := range validBackends {
		if config.Cache.Backend == b {
			backendValid = true
			break
		}
	}
	if !backendValid {
		errors = append(errors, fmt.Sprintf("invalid cache backend: %s", config.Cache.Backend))
	}

	// 로그 레벨 검증
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, l := range validLevels {
		if config.Logging.Level == l {
			levelValid = true
			break
		}
	}
	if !levelValid {
		errors = append(errors, fmt.Sprintf("invalid log level: %s", config.Logging.Level))
	}

	return errors
}

// LoadMavenProxyConfig 메이븐 프록시 설정 로드
func (cl *ConfigLoader) LoadMavenProxyConfig(ctx context.Context) (*MavenProxyConfig, error) {
	config := &MavenProxyConfig{}

	// 기본값 설정
	config.UseCache = true
	config.Cache = MavenProxyCacheConfig{
		Enabled: true,
	}

	// 설정 파일 경로 구성
	configPath := filepath.Join(cl.configPath, "maven-proxy.yaml")

	// 설정 파일이 존재하는지 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 기본값 반환
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read maven config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse maven config: %w", err)
	}

	// 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid maven config: %w", err)
	}

	return config, nil
}

// LoadAptProxyConfig APT 프록시 설정 로드
func (cl *ConfigLoader) LoadAptProxyConfig(ctx context.Context) (*AptProxyConfig, error) {
	config := &AptProxyConfig{}

	// 기본값 설정
	config.UseCache = true

	// 설정 파일 경로 구성
	configPath := filepath.Join(cl.configPath, "apt-proxy.yaml")

	// 설정 파일이 존재하는지 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 기본값 반환
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read apt config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse apt config: %w", err)
	}

	// 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid apt config: %w", err)
	}

	return config, nil
}

// LoadNpmProxyConfig NPM 프록시 설정 로드
func (cl *ConfigLoader) LoadNpmProxyConfig(ctx context.Context) (*NpmProxyConfig, error) {
	config := &NpmProxyConfig{}

	// 기본값 설정
	config.UseCache = true
	config.UserCache = false

	// 설정 파일 경로 구성
	configPath := filepath.Join(cl.configPath, "npm-proxy.yaml")

	// 설정 파일이 존재하는지 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 기본값 반환
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read npm config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse npm config: %w", err)
	}

	// 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid npm config: %w", err)
	}

	return config, nil
}

// LoadDockerProxyConfig Docker 프록시 설정 로드
func (cl *ConfigLoader) LoadDockerProxyConfig(ctx context.Context) (*DockerProxyConfig, error) {
	config := &DockerProxyConfig{}

	// 기본값 설정
	config.UseCache = true

	// 설정 파일 경로 구성
	configPath := filepath.Join(cl.configPath, "docker-proxy.yaml")

	// 설정 파일이 존재하는지 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 기본값 반환
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read docker config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse docker config: %w", err)
	}

	// 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid docker config: %w", err)
	}

	return config, nil
}

// LoadGlobalConfig 글로벌 설정 로드
func (cl *ConfigLoader) LoadGlobalConfig(ctx context.Context) (*UnifiedConfig, error) {
	return cl.Load()
}

// LoadPipProxyConfig PIP 프록시 설정 로드
func (cl *ConfigLoader) LoadPipProxyConfig(ctx context.Context) (*PipProxyConfig, error) {
	config := &PipProxyConfig{}

	// 기본값 설정
	config.UseCache = true

	// 설정 파일 경로 구성
	configPath := filepath.Join(cl.configPath, "pip-proxy.yaml")

	// 설정 파일이 존재하는지 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 기본값 반환
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pip config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse pip config: %w", err)
	}

	// 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pip config: %w", err)
	}

	return config, nil
}

// LoadYumProxyConfig YUM 프록시 설정 로드
func (cl *ConfigLoader) LoadYumProxyConfig(ctx context.Context) (*YumProxyConfig, error) {
	config := &YumProxyConfig{}

	// 기본값 설정
	config.UseCache = true

	// 설정 파일 경로 구성
	configPath := filepath.Join(cl.configPath, "yum-proxy.yaml")

	// 설정 파일이 존재하는지 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 기본값 반환
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read yum config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse yum config: %w", err)
	}

	// 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid yum config: %w", err)
	}

	return config, nil
}

// LoadApkProxyConfig APK 프록시 설정 로드
func (cl *ConfigLoader) LoadApkProxyConfig(ctx context.Context) (*ApkProxyConfig, error) {
	config := &ApkProxyConfig{}

	// 기본값 설정
	config.UseCache = true

	// 설정 파일 경로 구성
	configPath := filepath.Join(cl.configPath, "apk-proxy.yaml")

	// 설정 파일이 존재하는지 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 기본값 반환
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read apk config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse apk config: %w", err)
	}

	// 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid apk config: %w", err)
	}

	return config, nil
}
