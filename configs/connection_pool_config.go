package configs

import (
	"fmt"
	"time"

	"proxynd/internal/pool"
)

// ConnectionPoolSettings Connection Pool 설정
type ConnectionPoolSettings struct {
	// 전체 최대 연결 수
	MaxTotalConnections int `yaml:"max_total_connections" toml:"max_total_connections" default:"200"`

	// 호스트별 최대 연결 수
	MaxConnectionsPerHost int `yaml:"max_connections_per_host" toml:"max_connections_per_host" default:"20"`

	// 유휴 연결 타임아웃 (분)
	IdleConnectionTimeoutMinutes int `yaml:"idle_connection_timeout_minutes" toml:"idle_connection_timeout_minutes" default:"90"`

	// 연결 유지 시간 (초)
	KeepAliveTimeoutSeconds int `yaml:"keep_alive_timeout_seconds" toml:"keep_alive_timeout_seconds" default:"30"`

	// 연결 대기 타임아웃 (초)
	ConnectionTimeoutSeconds int `yaml:"connection_timeout_seconds" toml:"connection_timeout_seconds" default:"30"`

	// TLS 핸드셰이크 타임아웃 (초)
	TLSHandshakeTimeoutSeconds int `yaml:"tls_handshake_timeout_seconds" toml:"tls_handshake_timeout_seconds" default:"10"`

	// Response Header 타임아웃 (초)
	ResponseHeaderTimeoutSeconds int `yaml:"response_header_timeout_seconds" toml:"response_header_timeout_seconds" default:"30"`

	// Expect Continue 타임아웃 (초)
	ExpectContinueTimeoutSeconds int `yaml:"expect_continue_timeout_seconds" toml:"expect_continue_timeout_seconds" default:"1"`

	// 최대 리디렉션 수
	MaxRedirects int `yaml:"max_redirects" toml:"max_redirects" default:"10"`

	// TLS 검증 건너뛰기 (개발환경용)
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" toml:"insecure_skip_verify" default:"false"`

	// DNS 캐시 TTL (분)
	DNSCacheTTLMinutes int `yaml:"dns_cache_ttl_minutes" toml:"dns_cache_ttl_minutes" default:"5"`

	// 프록시별 타임아웃 설정 (초)
	ProxyTimeouts map[string]int `yaml:"proxy_timeouts" toml:"proxy_timeouts"`
}

// DefaultConnectionPoolSettings 기본 Connection Pool 설정
func DefaultConnectionPoolSettings() *ConnectionPoolSettings {
	return &ConnectionPoolSettings{
		MaxTotalConnections:          200,
		MaxConnectionsPerHost:        20,
		IdleConnectionTimeoutMinutes: 90,
		KeepAliveTimeoutSeconds:      30,
		ConnectionTimeoutSeconds:     30,
		TLSHandshakeTimeoutSeconds:   10,
		ResponseHeaderTimeoutSeconds: 30,
		ExpectContinueTimeoutSeconds: 1,
		MaxRedirects:                 10,
		InsecureSkipVerify:           false,
		DNSCacheTTLMinutes:           5,
		ProxyTimeouts: map[string]int{
			"maven":  60,  // Maven: 큰 JAR 파일
			"npm":    30,  // NPM: 일반적인 패키지
			"docker": 120, // Docker: 대용량 이미지
			"apt":    45,  // APT: 패키지 및 메타데이터
			"yum":    45,  // YUM: RPM 패키지
			"pip":    30,  // PIP: Python 패키지
			"apk":    20,  // APK: 작은 Alpine 패키지
		},
	}
}

// ToPoolConfig Connection Pool 내부 설정으로 변환
func (s *ConnectionPoolSettings) ToPoolConfig() *pool.ConnectionPoolConfig {
	return &pool.ConnectionPoolConfig{
		MaxTotalConnections:   s.MaxTotalConnections,
		MaxConnectionsPerHost: s.MaxConnectionsPerHost,
		IdleConnectionTimeout: time.Duration(s.IdleConnectionTimeoutMinutes) * time.Minute,
		KeepAliveTimeout:      time.Duration(s.KeepAliveTimeoutSeconds) * time.Second,
		ConnectionTimeout:     time.Duration(s.ConnectionTimeoutSeconds) * time.Second,
		TLSHandshakeTimeout:   time.Duration(s.TLSHandshakeTimeoutSeconds) * time.Second,
		ResponseHeaderTimeout: time.Duration(s.ResponseHeaderTimeoutSeconds) * time.Second,
		ExpectContinueTimeout: time.Duration(s.ExpectContinueTimeoutSeconds) * time.Second,
		MaxRedirects:          s.MaxRedirects,
		InsecureSkipVerify:    s.InsecureSkipVerify,
		DNSCacheTTL:           time.Duration(s.DNSCacheTTLMinutes) * time.Minute,
	}
}

// GetProxyTimeout 특정 프록시 타입의 타임아웃 반환
func (s *ConnectionPoolSettings) GetProxyTimeout(proxyType string) time.Duration {
	if timeout, exists := s.ProxyTimeouts[proxyType]; exists {
		return time.Duration(timeout) * time.Second
	}
	return 30 * time.Second // 기본값
}

// Validate 설정 유효성 검사
func (s *ConnectionPoolSettings) Validate() error {
	if s.MaxTotalConnections <= 0 {
		return fmt.Errorf("최대 연결 수는 0보다 커야 합니다")
	}

	if s.MaxConnectionsPerHost <= 0 {
		return fmt.Errorf("호스트별 최대 연결 수는 0보다 커야 합니다")
	}

	if s.MaxConnectionsPerHost > s.MaxTotalConnections {
		return fmt.Errorf("호스트별 최대 연결 수가 전체 최대 연결 수보다 클 수 없습니다")
	}

	if s.ConnectionTimeoutSeconds <= 0 {
		return fmt.Errorf("연결 타임아웃은 0보다 커야 합니다")
	}

	if s.MaxRedirects < 0 {
		return fmt.Errorf("최대 리디렉션 수는 0 이상이어야 합니다")
	}

	// 프록시별 타임아웃 검증
	for proxyType, timeout := range s.ProxyTimeouts {
		if timeout <= 0 {
			return fmt.Errorf("프록시 %s의 타임아웃은 0보다 커야 합니다", proxyType)
		}
		if timeout > 300 { // 5분 제한
			return fmt.Errorf("프록시 %s의 타임아웃이 너무 큽니다 (최대 300초)", proxyType)
		}
	}

	return nil
}

// UpdateProxyTimeout 프록시별 타임아웃 업데이트
func (s *ConnectionPoolSettings) UpdateProxyTimeout(proxyType string, timeoutSeconds int) error {
	if timeoutSeconds <= 0 {
		return fmt.Errorf("타임아웃은 0보다 커야 합니다")
	}
	if timeoutSeconds > 300 {
		return fmt.Errorf("타임아웃이 너무 큽니다 (최대 300초)")
	}

	if s.ProxyTimeouts == nil {
		s.ProxyTimeouts = make(map[string]int)
	}

	s.ProxyTimeouts[proxyType] = timeoutSeconds
	return nil
}

// GetStatistics 설정 통계 정보 반환
func (s *ConnectionPoolSettings) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"max_total_connections":      s.MaxTotalConnections,
		"max_connections_per_host":   s.MaxConnectionsPerHost,
		"idle_timeout_minutes":       s.IdleConnectionTimeoutMinutes,
		"connection_timeout_seconds": s.ConnectionTimeoutSeconds,
		"supported_proxy_types":      len(s.ProxyTimeouts),
		"insecure_skip_verify":       s.InsecureSkipVerify,
		"max_redirects":              s.MaxRedirects,
		"proxy_timeouts":             s.ProxyTimeouts,
	}
}

// ConnectionPoolConfig Connection Pool 설정 관리자
type ConnectionPoolConfig struct {
	settings   *ConnectionPoolSettings
	configPath string
}

// NewConnectionPoolConfig 연결 풀 설정 관리자 생성
func NewConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		settings:   DefaultConnectionPoolSettings(),
		configPath: "connection-pool",
	}
}

// ConfigExists 설정 파일 존재 여부 확인
func (c *ConnectionPoolConfig) ConfigExists() bool {
	// 간단한 구현: 파일 존재 여부만 확인
	// 실제로는 ConfigLoader를 통해 확인해야 함
	return false // 현재는 항상 기본값 사용
}

// ReadConfig 설정 파일 읽기
func (c *ConnectionPoolConfig) ReadConfig() error {
	if !c.ConfigExists() {
		// 설정 파일이 없으면 기본값 사용
		c.settings = DefaultConnectionPoolSettings()
		return nil
	}

	// TODO: 실제 설정 파일 로드 구현
	// 현재는 기본값 사용
	c.settings = DefaultConnectionPoolSettings()

	// 설정 유효성 검사
	if err := c.settings.Validate(); err != nil {
		return fmt.Errorf("connection Pool 설정 검증 실패: %w", err)
	}

	return nil
}

// WriteConfig 설정 파일 쓰기
func (c *ConnectionPoolConfig) WriteConfig() error {
	if err := c.settings.Validate(); err != nil {
		return fmt.Errorf("connection Pool 설정 검증 실패: %w", err)
	}

	// TODO: 실제 설정 파일 저장 구현
	// 현재는 성공으로 처리
	return nil
}

// GetSettings 현재 설정 반환
func (c *ConnectionPoolConfig) GetSettings() *ConnectionPoolSettings {
	if c.settings == nil {
		c.settings = DefaultConnectionPoolSettings()
	}
	return c.settings
}

// UpdateSettings 설정 업데이트
func (c *ConnectionPoolConfig) UpdateSettings(newSettings *ConnectionPoolSettings) error {
	if newSettings == nil {
		return fmt.Errorf("설정이 nil입니다")
	}

	if err := newSettings.Validate(); err != nil {
		return fmt.Errorf("새 설정 검증 실패: %w", err)
	}

	c.settings = newSettings
	return nil
}

// ReloadConfig 설정 다시 로드
func (c *ConnectionPoolConfig) ReloadConfig() error {
	return c.ReadConfig()
}
