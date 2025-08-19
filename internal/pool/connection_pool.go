package pool

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"proxynd/internal/logging"
)

// ConnectionPoolConfig Connection Pool 설정
type ConnectionPoolConfig struct {
	// 전체 최대 연결 수
	MaxTotalConnections int `yaml:"max_total_connections"`

	// 호스트별 최대 연결 수
	MaxConnectionsPerHost int `yaml:"max_connections_per_host"`

	// 유휴 연결 타임아웃
	IdleConnectionTimeout time.Duration `yaml:"idle_connection_timeout"`

	// 연결 유지 시간
	KeepAliveTimeout time.Duration `yaml:"keep_alive_timeout"`

	// 연결 대기 타임아웃
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`

	// TLS 핸드셰이크 타임아웃
	TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"`

	// Response Header 타임아웃
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout"`

	// Expect Continue 타임아웃
	ExpectContinueTimeout time.Duration `yaml:"expect_continue_timeout"`

	// 최대 리디렉션 수
	MaxRedirects int `yaml:"max_redirects"`

	// TLS 설정
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`

	// DNS 캐시 TTL
	DNSCacheTTL time.Duration `yaml:"dns_cache_ttl"`
}

// DefaultConnectionPoolConfig 기본 Connection Pool 설정
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxTotalConnections:   200,
		MaxConnectionsPerHost: 20,
		IdleConnectionTimeout: 90 * time.Second,
		KeepAliveTimeout:      30 * time.Second,
		ConnectionTimeout:     30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxRedirects:          10,
		InsecureSkipVerify:    false,
		DNSCacheTTL:           300 * time.Second, // 5분
	}
}

// ConnectionPool HTTP 연결 풀 관리자
type ConnectionPool struct {
	config    *ConnectionPoolConfig
	transport *http.Transport
	clients   map[string]*http.Client
	mutex     sync.RWMutex
	logger    logging.Logger

	// 통계 수집용
	stats *PoolStatistics
}

// PoolStatistics Connection Pool 통계
type PoolStatistics struct {
	mutex              sync.RWMutex
	TotalRequests      int64         `json:"total_requests"`
	ActiveConnections  int64         `json:"active_connections"`
	IdleConnections    int64         `json:"idle_connections"`
	ReusedConnections  int64         `json:"reused_connections"`
	DNSLookupTime      time.Duration `json:"dns_lookup_time_avg"`
	ConnectionTime     time.Duration `json:"connection_time_avg"`
	TLSHandshakeTime   time.Duration `json:"tls_handshake_time_avg"`
	RequestsSinceStart int64         `json:"requests_since_start"`
	LastRequestTime    time.Time     `json:"last_request_time"`
	CreatedAt          time.Time     `json:"created_at"`
}

// NewConnectionPool 새로운 Connection Pool 생성
func NewConnectionPool(config *ConnectionPoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	// 최적화된 Transport 생성
	transport := &http.Transport{
		// 다이얼러 설정
		DialContext: (&net.Dialer{
			Timeout:   config.ConnectionTimeout,
			KeepAlive: config.KeepAliveTimeout,
		}).DialContext,

		// 연결 풀 설정
		MaxIdleConns:        config.MaxTotalConnections,
		MaxIdleConnsPerHost: config.MaxConnectionsPerHost,
		IdleConnTimeout:     config.IdleConnectionTimeout,

		// TLS 설정
		TLSHandshakeTimeout: config.TLSHandshakeTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},

		// HTTP/2 설정
		ForceAttemptHTTP2:     true,
		MaxConnsPerHost:       config.MaxConnectionsPerHost,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,

		// 연결 재사용 최적화
		DisableKeepAlives:  false,
		DisableCompression: false,
	}

	pool := &ConnectionPool{
		config:    config,
		transport: transport,
		clients:   make(map[string]*http.Client),
		logger:    logging.GetLogger(),
		stats: &PoolStatistics{
			CreatedAt: time.Now(),
		},
	}

	pool.logger.Info("Connection Pool 초기화 완료",
		logging.F("max_total_connections", config.MaxTotalConnections),
		logging.F("max_connections_per_host", config.MaxConnectionsPerHost),
		logging.F("idle_timeout", config.IdleConnectionTimeout),
	)

	return pool
}

// GetClient 특정 프록시 타입용 HTTP 클라이언트 반환
func (p *ConnectionPool) GetClient(proxyType string, timeout time.Duration) *http.Client {
	p.mutex.RLock()
	client, exists := p.clients[proxyType]
	p.mutex.RUnlock()

	if exists {
		return client
	}

	// 새 클라이언트 생성 (Double-Checked Locking)
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 다시 한번 확인 (동시성 문제 방지)
	if client, exists := p.clients[proxyType]; exists {
		return client
	}

	// 새 클라이언트 생성
	client = &http.Client{
		Transport: p.transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= p.config.MaxRedirects {
				return fmt.Errorf("너무 많은 리디렉션: %d", len(via))
			}
			return nil
		},
	}

	p.clients[proxyType] = client

	p.logger.Debug("새 HTTP 클라이언트 생성",
		logging.F("proxy_type", proxyType),
		logging.F("timeout", timeout),
	)

	return client
}

// GetDefaultClient 기본 HTTP 클라이언트 반환
func (p *ConnectionPool) GetDefaultClient() *http.Client {
	return p.GetClient("default", 30*time.Second)
}

// ExecuteRequest 요청 실행 및 통계 수집
func (p *ConnectionPool) ExecuteRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	// 요청 시작 시간 기록
	startTime := time.Now()

	// 통계 업데이트
	p.stats.mutex.Lock()
	p.stats.TotalRequests++
	p.stats.RequestsSinceStart++
	p.stats.LastRequestTime = startTime
	p.stats.mutex.Unlock()

	// 요청 실행
	resp, err := client.Do(req)

	// 실행 시간 계산
	duration := time.Since(startTime)

	// 성능 로깅
	p.logger.Debug("HTTP 요청 실행 완료",
		logging.F("method", req.Method),
		logging.F("url", req.URL.String()),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("status", func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}()),
		logging.F("error", err != nil),
	)

	return resp, err
}

// GetStatistics Connection Pool 통계 반환
func (p *ConnectionPool) GetStatistics() *PoolStatistics {
	p.stats.mutex.RLock()
	defer p.stats.mutex.RUnlock()

	// 현재 Transport 상태에서 연결 정보 추출
	// 참고: Go의 http.Transport는 내부 상태를 직접 노출하지 않으므로
	// 실제 연결 수는 근사치로 계산됩니다.

	stats := &PoolStatistics{
		TotalRequests:      p.stats.TotalRequests,
		ActiveConnections:  p.estimateActiveConnections(),
		IdleConnections:    p.estimateIdleConnections(),
		ReusedConnections:  p.stats.ReusedConnections,
		RequestsSinceStart: p.stats.RequestsSinceStart,
		LastRequestTime:    p.stats.LastRequestTime,
		CreatedAt:          p.stats.CreatedAt,
	}

	return stats
}

// estimateActiveConnections 활성 연결 수 추정
func (p *ConnectionPool) estimateActiveConnections() int64 {
	// Transport의 내부 상태에 접근할 수 없으므로 클라이언트 수로 추정
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return int64(len(p.clients))
}

// estimateIdleConnections 유휴 연결 수 추정
func (p *ConnectionPool) estimateIdleConnections() int64 {
	// 실제 구현에서는 Transport의 내부 메트릭을 사용해야 하지만
	// 현재로서는 설정값으로 추정
	return int64(p.config.MaxTotalConnections / 4)
}

// Cleanup Connection Pool 정리
func (p *ConnectionPool) Cleanup() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Transport 정리
	p.transport.CloseIdleConnections()

	// 클라이언트 맵 정리
	for proxyType := range p.clients {
		delete(p.clients, proxyType)
	}

	p.logger.Info("Connection Pool 정리 완료")
}

// UpdateConfig Connection Pool 설정 업데이트
func (p *ConnectionPool) UpdateConfig(newConfig *ConnectionPoolConfig) error {
	if newConfig == nil {
		return fmt.Errorf("설정이 nil입니다")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 기존 연결 정리
	p.transport.CloseIdleConnections()

	// 새 Transport 생성
	p.transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   newConfig.ConnectionTimeout,
			KeepAlive: newConfig.KeepAliveTimeout,
		}).DialContext,
		MaxIdleConns:          newConfig.MaxTotalConnections,
		MaxIdleConnsPerHost:   newConfig.MaxConnectionsPerHost,
		IdleConnTimeout:       newConfig.IdleConnectionTimeout,
		TLSHandshakeTimeout:   newConfig.TLSHandshakeTimeout,
		ResponseHeaderTimeout: newConfig.ResponseHeaderTimeout,
		ExpectContinueTimeout: newConfig.ExpectContinueTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: newConfig.InsecureSkipVerify,
		},
	}

	// 기존 클라이언트들 업데이트
	for proxyType, client := range p.clients {
		client.Transport = p.transport
		p.logger.Debug("클라이언트 Transport 업데이트",
			logging.F("proxy_type", proxyType),
		)
	}

	p.config = newConfig

	p.logger.Info("Connection Pool 설정 업데이트 완료",
		logging.F("max_total_connections", newConfig.MaxTotalConnections),
		logging.F("max_connections_per_host", newConfig.MaxConnectionsPerHost),
	)

	return nil
}

// 전역 Connection Pool 인스턴스
var (
	globalPool     *ConnectionPool
	globalPoolOnce sync.Once
)

// GetGlobalPool 전역 Connection Pool 인스턴스 반환
func GetGlobalPool() *ConnectionPool {
	globalPoolOnce.Do(func() {
		globalPool = NewConnectionPool(DefaultConnectionPoolConfig())
	})
	return globalPool
}

// InitializeGlobalPool 전역 Connection Pool 초기화 (설정 파일 사용)
func InitializeGlobalPool(config *ConnectionPoolConfig) {
	globalPoolOnce.Do(func() {
		globalPool = NewConnectionPool(config)
	})
}

// NewConnectionPoolConfigFromSettings creates pool config from ConnectionPoolSettings fields
func NewConnectionPoolConfigFromSettings(
	maxTotal, maxPerHost, idleTimeoutMin, keepAliveTimeoutSec, connectionTimeoutSec,
	tlsTimeoutSec, responseHeaderTimeoutSec, expectContinueTimeoutSec, maxRedirects,
	dnsCacheTTLMin int, insecureSkipVerify bool,
) *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxTotalConnections:   maxTotal,
		MaxConnectionsPerHost: maxPerHost,
		IdleConnectionTimeout: time.Duration(idleTimeoutMin) * time.Minute,
		KeepAliveTimeout:      time.Duration(keepAliveTimeoutSec) * time.Second,
		ConnectionTimeout:     time.Duration(connectionTimeoutSec) * time.Second,
		TLSHandshakeTimeout:   time.Duration(tlsTimeoutSec) * time.Second,
		ResponseHeaderTimeout: time.Duration(responseHeaderTimeoutSec) * time.Second,
		ExpectContinueTimeout: time.Duration(expectContinueTimeoutSec) * time.Second,
		MaxRedirects:          maxRedirects,
		InsecureSkipVerify:    insecureSkipVerify,
		DNSCacheTTL:           time.Duration(dnsCacheTTLMin) * time.Minute,
	}
}
