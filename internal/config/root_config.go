package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"proxynd/alerts"
	"proxynd/verification"
)

// RootConfig 프로젝트의 최상위 설정 구조체
// 이전 UnifiedConfig를 대체하여 더 명확한 네이밍 제공
type RootConfig struct {
	// 서버 설정
	Server ServerConfig `yaml:"server" json:"server"`

	// 캐시 설정
	Cache CacheSettings `yaml:"cache" json:"cache"`

	// 프록시 레지스트리 설정
	Registries RegistryConfig `yaml:"registries" json:"registries"`

	// 인증 및 접근 제어
	Security SecuritySettings `yaml:"security" json:"security"`

	// 패키지 검증 설정
	Verification verification.VerifierConfig `yaml:"verification" json:"verification"`

	// 알림 설정
	Alerts alerts.AlertConfig `yaml:"alerts" json:"alerts"`

	// 로깅 설정
	Logging LoggingSettings `yaml:"logging" json:"logging"`

	// 메트릭 설정
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// 고급 설정
	Advanced AdvancedConfig `yaml:"advanced" json:"advanced"`
}

// CacheSettings 캐시 설정 (CacheConfig 대체)
type CacheSettings struct {
	// 캐시 백엔드 타입 (file, s3, redis)
	Backend string `yaml:"backend" json:"backend" default:"file"`

	// 공통 설정
	TTL      time.Duration `yaml:"ttl" json:"ttl" default:"3600s"`
	MaxSize  string        `yaml:"max_size" json:"max_size" default:"10GB"`
	MaxItems int64         `yaml:"max_items" json:"max_items"`

	// 정리 정책
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval" default:"1h"`
	EvictionPolicy  string        `yaml:"eviction_policy" json:"eviction_policy" default:"lru"`

	// 백엔드별 설정
	File  FileCacheConfig  `yaml:"file" json:"file"`
	S3    S3CacheConfig    `yaml:"s3" json:"s3"`
	Redis RedisCacheConfig `yaml:"redis" json:"redis"`
}

// SecuritySettings 보안 설정 (UnifiedSecurityConfig 대체)
type SecuritySettings struct {
	// 인증 방식
	Authentication AuthenticationConfig `yaml:"authentication" json:"authentication"`

	// 접근 제어
	AccessControl AccessControlConfig `yaml:"access_control" json:"access_control"`

	// 패키지 필터
	PackageFilter PackageFilterConfig `yaml:"package_filter" json:"package_filter"`

	// 해시 검증
	HashVerification HashVerificationConfig `yaml:"hash_verification" json:"hash_verification"`
}

// LoggingSettings 로깅 설정 (UnifiedLoggingConfig 대체)
type LoggingSettings struct {
	Level     string          `yaml:"level" json:"level" default:"info"`
	Format    string          `yaml:"format" json:"format" default:"json"`
	Output    string          `yaml:"output" json:"output" default:"stdout"`
	File      FileLogConfig   `yaml:"file" json:"file"`
	AccessLog AccessLogConfig `yaml:"access_log" json:"access_log"`
}

// 아래는 root_config에서만 필요한 타입들을 임시로 정의
// TODO: Phase 2에서 기존 타입들과 통합

// ServerConfig 서버 설정
type ServerConfig struct {
	// 기본 설정
	Host         string        `yaml:"host" json:"host" default:"0.0.0.0"`
	Port         int           `yaml:"port" json:"port" default:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout" default:"30s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout" default:"120s"`

	// TLS 설정
	TLS TLSConfig `yaml:"tls" json:"tls"`

	// HTTP/2 지원
	EnableHTTP2 bool `yaml:"enable_http2" json:"enable_http2" default:"true"`

	// 압축 설정
	Compression CompressionConfig `yaml:"compression" json:"compression"`

	// CORS 설정
	CORS CORSConfig `yaml:"cors" json:"cors"`
}

// TLSConfig TLS 설정
type TLSConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	CertFile   string `yaml:"cert_file" json:"cert_file"`
	KeyFile    string `yaml:"key_file" json:"key_file"`
	MinVersion string `yaml:"min_version" json:"min_version" default:"TLS1.2"`
}

// CompressionConfig 압축 설정
type CompressionConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled" default:"true"`
	Level   int      `yaml:"level" json:"level" default:"5"`
	Types   []string `yaml:"types" json:"types"`
}

// CORSConfig CORS 설정
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge           int      `yaml:"max_age" json:"max_age"`
}

// FileCacheConfig 파일 캐시 설정
type FileCacheConfig struct {
	Directory   string `yaml:"directory" json:"directory"`
	MaxFileSize string `yaml:"max_file_size" json:"max_file_size" default:"1GB"`
}

// S3CacheConfig S3 캐시 설정
type S3CacheConfig struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint"`
	Bucket          string `yaml:"bucket" json:"bucket"`
	Region          string `yaml:"region" json:"region"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"`
	UseSSL          bool   `yaml:"use_ssl" json:"use_ssl" default:"true"`
	PathPrefix      string `yaml:"path_prefix" json:"path_prefix"`
}

// RedisCacheConfig Redis 캐시 설정
type RedisCacheConfig struct {
	Address   string `yaml:"address" json:"address"`
	Password  string `yaml:"password" json:"password"`
	DB        int    `yaml:"db" json:"db"`
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix" default:"proxynd:"`
}

// RegistryConfig 레지스트리 설정
type RegistryConfig struct {
	NPM    NPMRegistryConfig    `yaml:"npm" json:"npm"`
	PyPI   PyPIRegistryConfig   `yaml:"pypi" json:"pypi"`
	APT    APTRegistryConfig    `yaml:"apt" json:"apt"`
	Docker DockerRegistryConfig `yaml:"docker" json:"docker"`
	Maven  MavenRegistryConfig  `yaml:"maven" json:"maven"`
}

// NPMRegistryConfig NPM 레지스트리 설정
type NPMRegistryConfig struct {
	Enabled   bool              `yaml:"enabled" json:"enabled"`
	Upstream  string            `yaml:"upstream" json:"upstream" default:"https://registry.npmjs.org"`
	Timeout   time.Duration     `yaml:"timeout" json:"timeout" default:"30s"`
	UserCache bool              `yaml:"user_cache" json:"user_cache"`
	Scopes    map[string]string `yaml:"scopes" json:"scopes"`
}

// PyPIRegistryConfig PyPI 레지스트리 설정
type PyPIRegistryConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled"`
	Upstream  string        `yaml:"upstream" json:"upstream" default:"https://pypi.org"`
	Simple    string        `yaml:"simple" json:"simple" default:"https://pypi.org/simple"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout" default:"30s"`
	UserCache bool          `yaml:"user_cache" json:"user_cache"`
}

// APTRegistryConfig APT 레지스트리 설정
type APTRegistryConfig struct {
	Enabled   bool                   `yaml:"enabled" json:"enabled"`
	UserCache bool                   `yaml:"user_cache" json:"user_cache"`
	Mirrors   map[string][]APTMirror `yaml:"mirrors" json:"mirrors"`
}

// APTMirror APT 미러 설정
type APTMirror struct {
	Name       string   `yaml:"name" json:"name"`
	URL        string   `yaml:"url" json:"url"`
	Suites     []string `yaml:"suites" json:"suites"`
	Components []string `yaml:"components" json:"components"`
}

// DockerRegistryConfig Docker 레지스트리 설정
type DockerRegistryConfig struct {
	Enabled    bool                     `yaml:"enabled" json:"enabled"`
	UseCache   bool                     `yaml:"use_cache" json:"use_cache"`
	Registries []DockerRegistryEndpoint `yaml:"registries" json:"registries"`
}

// DockerRegistryEndpoint Docker 레지스트리 엔드포인트
type DockerRegistryEndpoint struct {
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Insecure bool   `yaml:"insecure" json:"insecure"`
}

// MavenRegistryConfig Maven 레지스트리 설정
type MavenRegistryConfig struct {
	Enabled      bool                    `yaml:"enabled" json:"enabled"`
	Repositories []MavenRepositoryConfig `yaml:"repositories" json:"repositories"`
}

// MavenRepositoryConfig Maven 저장소 설정
type MavenRepositoryConfig struct {
	ID        string `yaml:"id" json:"id"`
	Name      string `yaml:"name" json:"name"`
	URL       string `yaml:"url" json:"url"`
	Releases  bool   `yaml:"releases" json:"releases" default:"true"`
	Snapshots bool   `yaml:"snapshots" json:"snapshots" default:"true"`
}

// AccessControlConfig 접근 제어 설정
type AccessControlConfig struct {
	// IP 화이트리스트
	IPWhitelist IPWhitelistConfig `yaml:"ip_whitelist" json:"ip_whitelist"`

	// 사용자별 권한
	Permissions []PermissionRule `yaml:"permissions" json:"permissions"`
}

// IPWhitelistConfig IP 화이트리스트 설정
type IPWhitelistConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	IPs     []string `yaml:"ips" json:"ips"`
	CIDRs   []string `yaml:"cidrs" json:"cidrs"`
}

// PermissionRule 권한 규칙
type PermissionRule struct {
	User       string   `yaml:"user" json:"user"`
	Groups     []string `yaml:"groups" json:"groups"`
	Actions    []string `yaml:"actions" json:"actions"` // read, write, delete
	Registries []string `yaml:"registries" json:"registries"`
	Packages   []string `yaml:"packages" json:"packages"`
}

// PackageFilterConfig 패키지 필터 설정
type PackageFilterConfig struct {
	Enabled bool                `yaml:"enabled" json:"enabled"`
	Mode    string              `yaml:"mode" json:"mode" default:"allowlist"` // allowlist, blocklist
	Rules   []PackageFilterRule `yaml:"rules" json:"rules"`
}

// PackageFilterRule 패키지 필터 규칙
type PackageFilterRule struct {
	Registry string   `yaml:"registry" json:"registry"`
	Patterns []string `yaml:"patterns" json:"patterns"`
	Action   string   `yaml:"action" json:"action"` // allow, deny
}

// HashVerificationConfig 해시 검증 설정
type HashVerificationConfig struct {
	Enabled             bool     `yaml:"enabled" json:"enabled"`
	RequiredHashHeaders []string `yaml:"required_hash_headers" json:"required_hash_headers"`
	FailOnMismatch      bool     `yaml:"fail_on_mismatch" json:"fail_on_mismatch"`
}

// FileLogConfig 파일 로그 설정
type FileLogConfig struct {
	Path       string `yaml:"path" json:"path"`
	MaxSize    string `yaml:"max_size" json:"max_size" default:"100MB"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups" default:"10"`
	MaxAge     int    `yaml:"max_age" json:"max_age" default:"30"`
	Compress   bool   `yaml:"compress" json:"compress" default:"true"`
}

// AccessLogConfig 액세스 로그 설정
type AccessLogConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled" default:"true"`
	Path        string `yaml:"path" json:"path"`
	Format      string `yaml:"format" json:"format" default:"json"`
	RotateDaily bool   `yaml:"rotate_daily" json:"rotate_daily" default:"true"`
}

// MetricsConfig 메트릭 설정
type MetricsConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Path      string `yaml:"path" json:"path" default:"/metrics"`
	Port      int    `yaml:"port" json:"port"` // 0 = 메인 포트 사용
	BasicAuth bool   `yaml:"basic_auth" json:"basic_auth"`
}

// SimplePerformanceConfig 간단한 성능 설정 (기존 PerformanceConfig와 구분)
type SimplePerformanceConfig struct {
	MaxConnections     int           `yaml:"max_connections" json:"max_connections" default:"1000"`
	MaxIdleConnections int           `yaml:"max_idle_connections" json:"max_idle_connections" default:"100"`
	ConnectionTimeout  time.Duration `yaml:"connection_timeout" json:"connection_timeout" default:"30s"`
	KeepAlive          time.Duration `yaml:"keep_alive" json:"keep_alive" default:"30s"`
	BufferSize         int           `yaml:"buffer_size" json:"buffer_size" default:"4096"`
}

// AdvancedConfig 고급 설정
type AdvancedConfig struct {
	// 성능 튜닝
	Performance SimplePerformanceConfig `yaml:"performance" json:"performance"`

	// 재시도 정책
	Retry RetryConfig `yaml:"retry" json:"retry"`

	// 회로 차단기
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker" json:"circuit_breaker"`
}

// RetryConfig 재시도 설정
type RetryConfig struct {
	MaxAttempts  int           `yaml:"max_attempts" json:"max_attempts" default:"3"`
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay" default:"1s"`
	MaxDelay     time.Duration `yaml:"max_delay" json:"max_delay" default:"30s"`
	Multiplier   float64       `yaml:"multiplier" json:"multiplier" default:"2.0"`
}

// CircuitBreakerConfig 회로 차단기 설정
type CircuitBreakerConfig struct {
	Enabled          bool          `yaml:"enabled" json:"enabled"`
	FailureThreshold int           `yaml:"failure_threshold" json:"failure_threshold" default:"5"`
	SuccessThreshold int           `yaml:"success_threshold" json:"success_threshold" default:"2"`
	Timeout          time.Duration `yaml:"timeout" json:"timeout" default:"60s"`
}

// ConfigDiff 설정 차이점
type ConfigDiff struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// ValidationOptions 검증 옵션
type ValidationOptions struct {
	EnableCrossValidation   bool // 교차 검증 활성화
	EnablePerformanceCheck  bool // 성능 검증 활성화
	EnableSecurityCheck     bool // 보안 검증 활성화
	SkipExternalConnections bool // 외부 연결 검증 건너뛰기
}

// LoadRootConfig 설정 파일 로드
func LoadRootConfig(configPath string) (*RootConfig, error) {
	// 기본 설정
	config := &RootConfig{}

	// 설정 파일이 없으면 기본값 사용
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config.ApplyDefaults()
		return config, nil
	}

	// YAML 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 환경 변수 오버라이드 적용
	config.applyEnvironmentOverrides()

	// 설정 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// applyEnvironmentOverrides 환경 변수 오버라이드 적용
func (c *RootConfig) applyEnvironmentOverrides() {
	// 서버 포트
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}

	// 캐시 디렉토리
	if dir := os.Getenv("STORAGE_DIR"); dir != "" {
		c.Cache.File.Directory = dir
	}

	// 로그 레벨
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}

	// TLS 설정
	if certFile := os.Getenv("TLS_CERT_FILE"); certFile != "" {
		c.Server.TLS.CertFile = certFile
		c.Server.TLS.Enabled = true
	}
	if keyFile := os.Getenv("TLS_KEY_FILE"); keyFile != "" {
		c.Server.TLS.KeyFile = keyFile
	}
}

// Validate 설정 검증
func (c *RootConfig) Validate() error {
	return c.ValidateWithOptions(ValidationOptions{
		EnableCrossValidation:  true,
		EnablePerformanceCheck: true,
		EnableSecurityCheck:    true,
	})
}

// ValidateWithOptions 옵션을 사용한 설정 검증
func (c *RootConfig) ValidateWithOptions(opts ValidationOptions) error {
	// 기본 필드 검증
	if err := c.validateBasicFields(); err != nil {
		return err
	}

	// 교차 검증
	if opts.EnableCrossValidation {
		if err := c.validateCrossReferences(); err != nil {
			return err
		}
	}

	// 성능 검증
	if opts.EnablePerformanceCheck {
		if err := c.validatePerformanceSettings(); err != nil {
			return err
		}
	}

	// 보안 검증
	if opts.EnableSecurityCheck {
		if err := c.validateSecuritySettings(); err != nil {
			return err
		}
	}

	return nil
}

// validateBasicFields 기본 필드 검증
func (c *RootConfig) validateBasicFields() error {
	// 서버 설정 검증
	if err := c.validateServer(); err != nil {
		return err
	}

	// TLS 설정 검증
	if err := c.validateTLS(); err != nil {
		return err
	}

	// 캐시 설정 검증
	if err := c.validateCache(); err != nil {
		return err
	}

	// 로깅 설정 검증
	if err := c.validateLogging(); err != nil {
		return err
	}

	// 메트릭 설정 검증
	if err := c.validateMetrics(); err != nil {
		return err
	}

	// 레지스트리 설정 검증
	if err := c.validateRegistries(); err != nil {
		return err
	}

	return nil
}

// validateServer 서버 설정 검증
func (c *RootConfig) validateServer() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return &ValidationError{
			Field:   "server.port",
			Message: "포트 번호가 유효하지 않습니다",
			Value:   c.Server.Port,
		}
	}
	return nil
}

// validateTLS TLS 설정 검증
func (c *RootConfig) validateTLS() error {
	if !c.Server.TLS.Enabled {
		return nil
	}

	if c.Server.TLS.CertFile == "" {
		return &ValidationError{
			Field:   "server.tls.cert_file",
			Message: "TLS가 활성화되었으나 인증서 파일이 지정되지 않음",
			Value:   c.Server.TLS.CertFile,
		}
	}

	if c.Server.TLS.KeyFile == "" {
		return &ValidationError{
			Field:   "server.tls.key_file",
			Message: "TLS가 활성화되었으나 키 파일이 지정되지 않음",
			Value:   c.Server.TLS.KeyFile,
		}
	}

	// 파일 존재 확인
	if _, err := os.Stat(c.Server.TLS.CertFile); err != nil {
		return &ValidationError{
			Field:   "server.tls.cert_file",
			Message: "TLS 인증서 파일을 찾을 수 없음",
			Value:   c.Server.TLS.CertFile,
		}
	}
	if _, err := os.Stat(c.Server.TLS.KeyFile); err != nil {
		return &ValidationError{
			Field:   "server.tls.key_file",
			Message: "TLS 키 파일을 찾을 수 없음",
			Value:   c.Server.TLS.KeyFile,
		}
	}

	return nil
}

// validateCache 캐시 설정 검증
func (c *RootConfig) validateCache() error {
	validators := map[string]func() error{
		"file":  c.validateFileCache,
		"s3":    c.validateS3Cache,
		"redis": c.validateRedisCache,
	}

	validator, exists := validators[c.Cache.Backend]
	if !exists {
		return &ValidationError{
			Field:   "cache.backend",
			Message: "지원하지 않는 캐시 백엔드",
			Value:   c.Cache.Backend,
		}
	}

	return validator()
}

// validateFileCache 파일 캐시 설정 검증
func (c *RootConfig) validateFileCache() error {
	if c.Cache.File.Directory == "" {
		c.Cache.File.Directory = filepath.Join(os.TempDir(), "proxynd-cache")
	}
	return nil
}

// validateS3Cache S3 캐시 설정 검증
func (c *RootConfig) validateS3Cache() error {
	if c.Cache.S3.Bucket == "" {
		return &ValidationError{
			Field:   "cache.s3.bucket",
			Message: "S3 캐시가 활성화되었으나 버킷이 지정되지 않음",
			Value:   c.Cache.S3.Bucket,
		}
	}
	if c.Cache.S3.Region == "" {
		return &ValidationError{
			Field:   "cache.s3.region",
			Message: "S3 캐시가 활성화되었으나 리전이 지정되지 않음",
			Value:   c.Cache.S3.Region,
		}
	}
	return nil
}

// validateRedisCache Redis 캐시 설정 검증
func (c *RootConfig) validateRedisCache() error {
	if c.Cache.Redis.Address == "" {
		return &ValidationError{
			Field:   "cache.redis.address",
			Message: "Redis 캐시가 활성화되었으나 주소가 지정되지 않음",
			Value:   c.Cache.Redis.Address,
		}
	}
	return nil
}

// validateLogging 로깅 설정 검증
func (c *RootConfig) validateLogging() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[c.Logging.Level] {
		return &ValidationError{
			Field:   "logging.level",
			Message: "유효하지 않은 로그 레벨",
			Value:   c.Logging.Level,
		}
	}
	return nil
}

// validateMetrics 메트릭 설정 검증
func (c *RootConfig) validateMetrics() error {
	if !c.Metrics.Enabled || c.Metrics.Port == 0 {
		return nil
	}

	if c.Metrics.Port < 1 || c.Metrics.Port > 65535 {
		return &ValidationError{
			Field:   "metrics.port",
			Message: "메트릭 포트 번호가 유효하지 않습니다",
			Value:   c.Metrics.Port,
		}
	}

	if c.Metrics.Port == c.Server.Port {
		return &ValidationError{
			Field:   "metrics.port",
			Message: "메트릭 포트가 서버 포트와 같습니다",
			Value:   c.Metrics.Port,
		}
	}

	return nil
}

// validateRegistries 레지스트리 설정 검증
func (c *RootConfig) validateRegistries() error {
	registryValidators := []struct {
		enabled   bool
		validator func() error
	}{
		{c.Registries.NPM.Enabled, c.validateNPMRegistry},
		{c.Registries.PyPI.Enabled, c.validatePyPIRegistry},
		{c.Registries.APT.Enabled, c.validateAPTRegistry},
		{c.Registries.Docker.Enabled, c.validateDockerRegistry},
		{c.Registries.Maven.Enabled, c.validateMavenRegistry},
	}

	for _, rv := range registryValidators {
		if rv.enabled {
			if err := rv.validator(); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateNPMRegistry NPM 레지스트리 검증
func (c *RootConfig) validateNPMRegistry() error {
	if c.Registries.NPM.Upstream == "" {
		return &ValidationError{
			Field:   "registries.npm.upstream",
			Message: "NPM 업스트림 URL이 설정되지 않음",
			Value:   c.Registries.NPM.Upstream,
		}
	}
	return nil
}

// validatePyPIRegistry PyPI 레지스트리 검증
func (c *RootConfig) validatePyPIRegistry() error {
	if c.Registries.PyPI.Upstream == "" {
		return &ValidationError{
			Field:   "registries.pypi.upstream",
			Message: "PyPI 업스트림 URL이 설정되지 않음",
			Value:   c.Registries.PyPI.Upstream,
		}
	}
	return nil
}

// validateAPTRegistry APT 레지스트리 검증
func (c *RootConfig) validateAPTRegistry() error {
	if len(c.Registries.APT.Mirrors) == 0 {
		return &ValidationError{
			Field:   "registries.apt.mirrors",
			Message: "APT 미러가 설정되지 않음",
			Value:   c.Registries.APT.Mirrors,
		}
	}

	for distro, mirrors := range c.Registries.APT.Mirrors {
		if len(mirrors) == 0 {
			return &ValidationError{
				Field:   fmt.Sprintf("registries.apt.mirrors.%s", distro),
				Message: "배포판에 대한 미러가 설정되지 않음",
				Value:   mirrors,
			}
		}
	}
	return nil
}

// validateDockerRegistry Docker 레지스트리 검증
func (c *RootConfig) validateDockerRegistry() error {
	if len(c.Registries.Docker.Registries) == 0 {
		return &ValidationError{
			Field:   "registries.docker.registries",
			Message: "Docker 레지스트리가 설정되지 않음",
			Value:   c.Registries.Docker.Registries,
		}
	}
	return nil
}

// validateMavenRegistry Maven 레지스트리 검증
func (c *RootConfig) validateMavenRegistry() error {
	if len(c.Registries.Maven.Repositories) == 0 {
		return &ValidationError{
			Field:   "registries.maven.repositories",
			Message: "Maven 리포지토리가 설정되지 않음",
			Value:   c.Registries.Maven.Repositories,
		}
	}
	return nil
}

// validateCrossReferences 교차 참조 검증
func (c *RootConfig) validateCrossReferences() error {
	// 메트릭 포트와 서버 포트 중복 확인
	if c.Metrics.Enabled && c.Metrics.Port != 0 && c.Metrics.Port == c.Server.Port {
		return &ValidationError{
			Field:   "metrics.port",
			Message: "메트릭 포트가 서버 포트와 중복됨",
			Value:   c.Metrics.Port,
		}
	}

	// TLS 설정 일관성 확인
	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" || c.Server.TLS.KeyFile == "" {
			return &ValidationError{
				Field:   "server.tls",
				Message: "TLS가 활성화되었으나 필수 파일이 누락됨",
				Value:   c.Server.TLS,
			}
		}
	}

	// 캐시 백엔드별 필수 설정 확인
	switch c.Cache.Backend {
	case "s3":
		if c.Cache.S3.Bucket == "" || c.Cache.S3.Region == "" {
			return &ValidationError{
				Field:   "cache.s3",
				Message: "S3 캐시 백엔드 필수 설정 누락",
				Value:   c.Cache.S3,
			}
		}
	case "redis":
		if c.Cache.Redis.Address == "" {
			return &ValidationError{
				Field:   "cache.redis.address",
				Message: "Redis 캐시 백엔드 주소 누락",
				Value:   c.Cache.Redis.Address,
			}
		}
	}

	// 인증 설정과 보안 설정 일관성
	if c.Security.Authentication.BasicAuth != nil &&
		c.Security.Authentication.BasicAuth.Enabled != nil &&
		*c.Security.Authentication.BasicAuth.Enabled &&
		len(c.Security.AccessControl.Permissions) == 0 {
		return &ValidationError{
			Field:   "security.access_control.permissions",
			Message: "인증이 활성화되었으나 권한 설정이 누락됨",
			Value:   c.Security.AccessControl.Permissions,
		}
	}

	return nil
}

// validatePerformanceSettings 성능 설정 검증
func (c *RootConfig) validatePerformanceSettings() error {
	// 연결 설정 일관성 확인
	if c.Advanced.Performance.MaxConnections < c.Advanced.Performance.MaxIdleConnections {
		return &ValidationError{
			Field:   "advanced.performance.max_idle_connections",
			Message: "최대 유휴 연결 수가 최대 연결 수를 초과함",
			Value:   c.Advanced.Performance.MaxIdleConnections,
		}
	}

	// 타임아웃 설정 검증
	if c.Server.ReadTimeout > 0 && c.Server.WriteTimeout > 0 {
		if c.Server.ReadTimeout > c.Server.IdleTimeout {
			return &ValidationError{
				Field:   "server.read_timeout",
				Message: "읽기 타임아웃이 유휴 타임아웃보다 김",
				Value:   c.Server.ReadTimeout,
			}
		}
	}

	// 캐시 크기와 TTL 균형 확인
	if c.Cache.TTL > 0 && c.Cache.CleanupInterval > 0 {
		if c.Cache.CleanupInterval > c.Cache.TTL*2 {
			return &ValidationError{
				Field:   "cache.cleanup_interval",
				Message: "캐시 정리 주기가 TTL에 비해 너무 김",
				Value:   c.Cache.CleanupInterval,
			}
		}
	}

	// 재시도 설정 검증
	if c.Advanced.Retry.MaxAttempts > 10 {
		return &ValidationError{
			Field:   "advanced.retry.max_attempts",
			Message: "재시도 횟수가 너무 많음 (최대 10회 권장)",
			Value:   c.Advanced.Retry.MaxAttempts,
		}
	}

	if c.Advanced.Retry.MaxDelay < c.Advanced.Retry.InitialDelay {
		return &ValidationError{
			Field:   "advanced.retry.max_delay",
			Message: "최대 지연시간이 초기 지연시간보다 작음",
			Value:   c.Advanced.Retry.MaxDelay,
		}
	}

	return nil
}

// validateSecuritySettings 보안 설정 검증
func (c *RootConfig) validateSecuritySettings() error {
	// TLS 버전 보안 확인
	if c.Server.TLS.Enabled && c.Server.TLS.MinVersion != "" {
		validVersions := map[string]bool{
			"TLS1.2": true,
			"TLS1.3": true,
		}
		if !validVersions[c.Server.TLS.MinVersion] {
			return &ValidationError{
				Field:   "server.tls.min_version",
				Message: "안전하지 않은 TLS 버전 (TLS 1.2 이상 권장)",
				Value:   c.Server.TLS.MinVersion,
			}
		}
	}

	// IP 화이트리스트 검증
	if c.Security.AccessControl.IPWhitelist.Enabled {
		for _, ip := range c.Security.AccessControl.IPWhitelist.IPs {
			if net.ParseIP(ip) == nil {
				return &ValidationError{
					Field:   "security.access_control.ip_whitelist.ips",
					Message: fmt.Sprintf("유효하지 않은 IP 주소: %s", ip),
					Value:   ip,
				}
			}
		}

		for _, cidr := range c.Security.AccessControl.IPWhitelist.CIDRs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return &ValidationError{
					Field:   "security.access_control.ip_whitelist.cidrs",
					Message: fmt.Sprintf("유효하지 않은 CIDR 블록: %s", cidr),
					Value:   cidr,
				}
			}
		}
	}

	// 패키지 필터 규칙 검증
	if c.Security.PackageFilter.Enabled {
		validModes := map[string]bool{
			"allowlist": true,
			"blocklist": true,
		}
		if !validModes[c.Security.PackageFilter.Mode] {
			return &ValidationError{
				Field:   "security.package_filter.mode",
				Message: "유효하지 않은 필터 모드 (allowlist, blocklist만 허용)",
				Value:   c.Security.PackageFilter.Mode,
			}
		}

		for _, rule := range c.Security.PackageFilter.Rules {
			validActions := map[string]bool{
				"allow": true,
				"deny":  true,
			}
			if !validActions[rule.Action] {
				return &ValidationError{
					Field:   "security.package_filter.rules.action",
					Message: "유효하지 않은 필터 액션 (allow, deny만 허용)",
					Value:   rule.Action,
				}
			}
		}
	}

	// 권한 규칙 검증
	for i, rule := range c.Security.AccessControl.Permissions {
		validActions := map[string]bool{
			"read":   true,
			"write":  true,
			"delete": true,
		}
		for _, action := range rule.Actions {
			if !validActions[action] {
				return &ValidationError{
					Field:   fmt.Sprintf("security.access_control.permissions[%d].actions", i),
					Message: fmt.Sprintf("유효하지 않은 권한 액션: %s", action),
					Value:   action,
				}
			}
		}
	}

	return nil
}

// ApplyDefaults 기본값 적용
func (c *RootConfig) ApplyDefaults() {
	// 서버 기본값
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 120 * time.Second
	}

	// TLS 기본값
	if c.Server.TLS.MinVersion == "" {
		c.Server.TLS.MinVersion = "TLS1.2"
	}

	// 캐시 기본값
	if c.Cache.Backend == "" {
		c.Cache.Backend = backendFile
	}
	if c.Cache.TTL == 0 {
		c.Cache.TTL = 3600 * time.Second
	}
	if c.Cache.MaxSize == "" {
		c.Cache.MaxSize = "10GB"
	}
	if c.Cache.CleanupInterval == 0 {
		c.Cache.CleanupInterval = 1 * time.Hour
	}
	if c.Cache.EvictionPolicy == "" {
		c.Cache.EvictionPolicy = "lru"
	}

	// 로깅 기본값
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}

	// 메트릭 기본값
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}

	// 성능 기본값
	if c.Advanced.Performance.MaxConnections == 0 {
		c.Advanced.Performance.MaxConnections = 1000
	}
	if c.Advanced.Performance.MaxIdleConnections == 0 {
		c.Advanced.Performance.MaxIdleConnections = 100
	}
	if c.Advanced.Performance.ConnectionTimeout == 0 {
		c.Advanced.Performance.ConnectionTimeout = 30 * time.Second
	}
	if c.Advanced.Performance.KeepAlive == 0 {
		c.Advanced.Performance.KeepAlive = 30 * time.Second
	}
	if c.Advanced.Performance.BufferSize == 0 {
		c.Advanced.Performance.BufferSize = 4096
	}

	// 재시도 기본값
	if c.Advanced.Retry.MaxAttempts == 0 {
		c.Advanced.Retry.MaxAttempts = 3
	}
	if c.Advanced.Retry.InitialDelay == 0 {
		c.Advanced.Retry.InitialDelay = 1 * time.Second
	}
	if c.Advanced.Retry.MaxDelay == 0 {
		c.Advanced.Retry.MaxDelay = 30 * time.Second
	}
	if c.Advanced.Retry.Multiplier == 0 {
		c.Advanced.Retry.Multiplier = 2.0
	}

	// 회로 차단기 기본값
	if c.Advanced.CircuitBreaker.FailureThreshold == 0 {
		c.Advanced.CircuitBreaker.FailureThreshold = 5
	}
	if c.Advanced.CircuitBreaker.SuccessThreshold == 0 {
		c.Advanced.CircuitBreaker.SuccessThreshold = 2
	}
	if c.Advanced.CircuitBreaker.Timeout == 0 {
		c.Advanced.CircuitBreaker.Timeout = 60 * time.Second
	}
}

// Clone 설정 복사본 생성
func (c *RootConfig) Clone() (*RootConfig, error) {
	// JSON을 통한 딥 카피
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var clone RootConfig
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &clone, nil
}

// Diff 설정 차이점 반환
func (c *RootConfig) Diff(other *RootConfig) ([]ConfigDiff, error) {
	// 간단한 필드별 비교
	var diffs []ConfigDiff

	if c.Server.Port != other.Server.Port {
		diffs = append(diffs, ConfigDiff{
			Field:    "server.port",
			OldValue: c.Server.Port,
			NewValue: other.Server.Port,
		})
	}

	if c.Server.Host != other.Server.Host {
		diffs = append(diffs, ConfigDiff{
			Field:    "server.host",
			OldValue: c.Server.Host,
			NewValue: other.Server.Host,
		})
	}

	if c.Cache.Backend != other.Cache.Backend {
		diffs = append(diffs, ConfigDiff{
			Field:    "cache.backend",
			OldValue: c.Cache.Backend,
			NewValue: other.Cache.Backend,
		})
	}

	if c.Logging.Level != other.Logging.Level {
		diffs = append(diffs, ConfigDiff{
			Field:    "logging.level",
			OldValue: c.Logging.Level,
			NewValue: other.Logging.Level,
		})
	}

	return diffs, nil
}

// IsProductionReady 프로덕션 준비 상태 확인
func (c *RootConfig) IsProductionReady() (bool, []string) {
	var issues []string

	// TLS 확인
	if !c.Server.TLS.Enabled {
		issues = append(issues, "TLS가 비활성화되어 있음")
	}

	// 인증 확인
	if c.Security.Authentication.BasicAuth == nil ||
		c.Security.Authentication.BasicAuth.Enabled == nil ||
		!*c.Security.Authentication.BasicAuth.Enabled {
		issues = append(issues, "인증이 비활성화되어 있음")
	}

	// 로그 레벨 확인
	if c.Logging.Level == "debug" {
		issues = append(issues, "디버그 로그 레벨이 설정되어 있음")
	}

	// 메트릭 확인
	if !c.Metrics.Enabled {
		issues = append(issues, "메트릭 수집이 비활성화되어 있음")
	}

	return len(issues) == 0, issues
}

// ValidateRootConfig 전역 설정 검증 함수
func ValidateRootConfig(config *RootConfig) []ValidationError {
	var errors []ValidationError

	// 기본 검증
	if err := config.Validate(); err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			errors = append(errors, *validationErr)
		} else {
			errors = append(errors, ValidationError{
				Field:   "general",
				Message: err.Error(),
				Value:   nil,
			})
		}
	}

	return errors
}

// GetValidationWarnings 설정 검증 경고 반환
func GetValidationWarnings(config *RootConfig) []ValidationWarning {
	// 간단한 구현 - 실제로는 더 정교한 검증 필요
	var warnings []ValidationWarning

	if config.Server.TLS.Enabled && config.Server.TLS.MinVersion == "TLS1.2" {
		warnings = append(warnings, ValidationWarning{
			Field:   "server.tls.min_version",
			Message: "TLS 1.3 사용을 권장합니다",
		})
	}

	if config.Logging.Level == "debug" {
		warnings = append(warnings, ValidationWarning{
			Field:   "logging.level",
			Message: "프로덕션 환경에서는 debug 로그 레벨을 사용하지 않는 것을 권장합니다",
		})
	}

	return warnings
}
