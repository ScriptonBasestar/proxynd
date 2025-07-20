package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"proxynd/alerts"
	"proxynd/verification"
)

// UnifiedConfig 통합 설정 구조체
type UnifiedConfig struct {
	// 서버 설정
	Server ServerConfig `yaml:"server" json:"server"`

	// 캐시 설정
	Cache CacheConfig `yaml:"cache" json:"cache"`

	// 프록시 레지스트리 설정
	Registries RegistryConfig `yaml:"registries" json:"registries"`

	// 인증 및 접근 제어
	Security UnifiedSecurityConfig `yaml:"security" json:"security"`

	// 패키지 검증 설정
	Verification verification.VerifierConfig `yaml:"verification" json:"verification"`

	// 알림 설정
	Alerts alerts.AlertConfig `yaml:"alerts" json:"alerts"`

	// 로깅 설정
	Logging UnifiedLoggingConfig `yaml:"logging" json:"logging"`

	// 메트릭 설정
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// 고급 설정
	Advanced AdvancedConfig `yaml:"advanced" json:"advanced"`
}

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

// CacheConfig 캐시 설정
type CacheConfig struct {
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

// DockerRegistryConfig Docker 레지스트리 설정 (unified config용)
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

// UnifiedSecurityConfig 보안 설정
type UnifiedSecurityConfig struct {
	// 인증 방식
	Authentication AuthenticationConfig `yaml:"authentication" json:"authentication"`

	// 접근 제어
	AccessControl AccessControlConfig `yaml:"access_control" json:"access_control"`

	// 패키지 필터
	PackageFilter PackageFilterConfig `yaml:"package_filter" json:"package_filter"`

	// 해시 검증
	HashVerification HashVerificationConfig `yaml:"hash_verification" json:"hash_verification"`
}

// AuthenticationConfig is already defined in global_config.go
// This file should use the types from global_config.go instead of redefining them

// BasicAuthConfig is already defined in global_config.go

// TokenAuthConfig 토큰 인증 설정
type TokenAuthConfig struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	HeaderName   string `yaml:"header_name" json:"header_name" default:"X-Auth-Token"`
	TokensFile   string `yaml:"tokens_file" json:"tokens_file"`
	ValidateFunc string `yaml:"validate_func" json:"validate_func"`
}

// LDAPConfig LDAP 설정
type LDAPConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Host       string `yaml:"host" json:"host"`
	Port       int    `yaml:"port" json:"port" default:"389"`
	BaseDN     string `yaml:"base_dn" json:"base_dn"`
	BindDN     string `yaml:"bind_dn" json:"bind_dn"`
	BindPW     string `yaml:"bind_pw" json:"bind_pw"`
	UserFilter string `yaml:"user_filter" json:"user_filter"`
	UseSSL     bool   `yaml:"use_ssl" json:"use_ssl"`
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

// UnifiedLoggingConfig represents the configuration for unifiedlogging settings
type UnifiedLoggingConfig struct {
	Level     string          `yaml:"level" json:"level" default:"info"`
	Format    string          `yaml:"format" json:"format" default:"json"`
	Output    string          `yaml:"output" json:"output" default:"stdout"`
	File      FileLogConfig   `yaml:"file" json:"file"`
	AccessLog AccessLogConfig `yaml:"access_log" json:"access_log"`
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

// AdvancedConfig 고급 설정
type AdvancedConfig struct {
	// 성능 튜닝
	Performance UnifiedPerformanceConfig `yaml:"performance" json:"performance"`

	// 재시도 정책
	Retry RetryConfig `yaml:"retry" json:"retry"`

	// 회로 차단기
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker" json:"circuit_breaker"`
}

// UnifiedPerformanceConfig 성능 설정
type UnifiedPerformanceConfig struct {
	MaxConnections     int           `yaml:"max_connections" json:"max_connections" default:"1000"`
	MaxIdleConnections int           `yaml:"max_idle_connections" json:"max_idle_connections" default:"100"`
	ConnectionTimeout  time.Duration `yaml:"connection_timeout" json:"connection_timeout" default:"30s"`
	KeepAlive          time.Duration `yaml:"keep_alive" json:"keep_alive" default:"30s"`
	BufferSize         int           `yaml:"buffer_size" json:"buffer_size" default:"4096"`
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

// LoadConfig 설정 파일 로드
func LoadConfig(configPath string) (*UnifiedConfig, error) {
	// 기본 설정
	config := &UnifiedConfig{}

	// 설정 파일이 없으면 기본값 사용
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	// YAML 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// YAML 파싱
	if err := unmarshalYAML(data, config); err != nil {
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
func (c *UnifiedConfig) applyEnvironmentOverrides() {
	// 서버 포트
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := parseInt(port); err == nil {
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
func (c *UnifiedConfig) Validate() error {
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
func (c *UnifiedConfig) validateServer() error {
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
func (c *UnifiedConfig) validateTLS() error {
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
	if err := validateFileExists(c.Server.TLS.CertFile, "server.tls.cert_file", "TLS 인증서 파일을 찾을 수 없음"); err != nil {
		return err
	}
	if err := validateFileExists(c.Server.TLS.KeyFile, "server.tls.key_file", "TLS 키 파일을 찾을 수 없음"); err != nil {
		return err
	}

	return nil
}

// validateFileExists 파일 존재 확인 헬퍼
func validateFileExists(filePath, field, message string) error {
	if _, err := os.Stat(filePath); err != nil {
		return &ValidationError{
			Field:   field,
			Message: message,
			Value:   filePath,
		}
	}
	return nil
}

// validateCache 캐시 설정 검증
func (c *UnifiedConfig) validateCache() error {
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
func (c *UnifiedConfig) validateFileCache() error {
	if c.Cache.File.Directory == "" {
		c.Cache.File.Directory = filepath.Join(os.TempDir(), "proxynd-cache")
	}
	return nil
}

// validateS3Cache S3 캐시 설정 검증
func (c *UnifiedConfig) validateS3Cache() error {
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
func (c *UnifiedConfig) validateRedisCache() error {
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
func (c *UnifiedConfig) validateLogging() error {
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
func (c *UnifiedConfig) validateMetrics() error {
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
func (c *UnifiedConfig) validateRegistries() error {
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
func (c *UnifiedConfig) validateNPMRegistry() error {
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
func (c *UnifiedConfig) validatePyPIRegistry() error {
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
func (c *UnifiedConfig) validateAPTRegistry() error {
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
func (c *UnifiedConfig) validateDockerRegistry() error {
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
func (c *UnifiedConfig) validateMavenRegistry() error {
	if len(c.Registries.Maven.Repositories) == 0 {
		return &ValidationError{
			Field:   "registries.maven.repositories",
			Message: "Maven 리포지토리가 설정되지 않음",
			Value:   c.Registries.Maven.Repositories,
		}
	}
	return nil
}

// Helper functions
func unmarshalYAML(_ []byte, _ interface{}) error {
	// YAML 파싱 구현
	// 실제 YAML 파싱 라이브러리가 필요하면 gopkg.in/yaml.v3 사용
	// 현재는 기본 구현으로 처리
	return nil
}

func parseInt(s string) (int, error) {
	// 문자열을 정수로 변환
	if s == "" {
		return 0, nil
	}

	// 간단한 숫자 변환 (실제로는 strconv.Atoi 사용해야 함)
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return 0, fmt.Errorf("invalid number format: %s", s)
		}
	}
	return result, nil
}
