package pool

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"proxynd/internal/services/proxy"
	"proxynd/logging"
)

// ProxyClientFactory 프록시별 HTTP 클라이언트 팩토리
type ProxyClientFactory struct {
	pool   *ConnectionPool
	logger logging.Logger

	// 프록시 타입별 기본 타임아웃 설정
	defaultTimeouts map[string]time.Duration
}

// NewProxyClientFactory 새로운 클라이언트 팩토리 생성
func NewProxyClientFactory(pool *ConnectionPool) *ProxyClientFactory {
	if pool == nil {
		pool = GetGlobalPool()
	}

	factory := &ProxyClientFactory{
		pool:   pool,
		logger: logging.GetLogger(),
		defaultTimeouts: map[string]time.Duration{
			"maven":  60 * time.Second,  // Maven: 큰 JAR 파일 다운로드
			"npm":    30 * time.Second,  // NPM: 일반적인 패키지 크기
			"docker": 120 * time.Second, // Docker: 대용량 이미지 레이어
			"apt":    45 * time.Second,  // APT: 패키지 및 메타데이터
			"yum":    45 * time.Second,  // YUM: RPM 패키지들
			"pip":    30 * time.Second,  // PIP: Python 패키지
			"apk":    20 * time.Second,  // APK: 작은 Alpine 패키지
		},
	}

	factory.logger.Info("Proxy Client Factory 초기화 완료",
		logging.F("supported_types", len(factory.defaultTimeouts)),
	)

	return factory
}

// GetClientForProxy 특정 프록시 타입용 HTTP 클라이언트 반환
func (f *ProxyClientFactory) GetClientForProxy(proxyType string) *http.Client {
	timeout, exists := f.defaultTimeouts[proxyType]
	if !exists {
		timeout = 30 * time.Second // 기본값
		f.logger.Warn("알려지지 않은 프록시 타입, 기본 타임아웃 사용",
			logging.F("proxy_type", proxyType),
			logging.F("default_timeout", timeout),
		)
	}

	return f.pool.GetClient(proxyType, timeout)
}

// GetClientWithTimeout 사용자 정의 타임아웃으로 클라이언트 반환
func (f *ProxyClientFactory) GetClientWithTimeout(proxyType string, timeout time.Duration) *http.Client {
	clientKey := fmt.Sprintf("%s_%s", proxyType, timeout)
	return f.pool.GetClient(clientKey, timeout)
}

// ExecuteProxyRequest 프록시 요청 실행 (통계 수집 포함)
func (f *ProxyClientFactory) ExecuteProxyRequest(proxyType string, req *http.Request) (*http.Response, error) {
	client := f.GetClientForProxy(proxyType)
	return f.pool.ExecuteRequest(client, req)
}

// CreateUpstreamClient Connection Pool을 사용하는 업스트림 클라이언트 생성
func (f *ProxyClientFactory) CreateUpstreamClient(proxyType string, timeout time.Duration) proxy.UpstreamClient {
	client := f.GetClientWithTimeout(proxyType, timeout)

	return &PooledUpstreamClient{
		client:    client,
		factory:   f,
		proxyType: proxyType,
		logger:    f.logger,
	}
}

// UpdateTimeout 프록시 타입별 기본 타임아웃 업데이트
func (f *ProxyClientFactory) UpdateTimeout(proxyType string, timeout time.Duration) {
	f.defaultTimeouts[proxyType] = timeout
	f.logger.Info("프록시 타입 타임아웃 업데이트",
		logging.F("proxy_type", proxyType),
		logging.F("timeout", timeout),
	)
}

// GetSupportedProxyTypes 지원되는 프록시 타입 목록 반환
func (f *ProxyClientFactory) GetSupportedProxyTypes() []string {
	types := make([]string, 0, len(f.defaultTimeouts))
	for proxyType := range f.defaultTimeouts {
		types = append(types, proxyType)
	}
	return types
}

// GetPoolStatistics Connection Pool 통계 반환
func (f *ProxyClientFactory) GetPoolStatistics() *PoolStatistics {
	return f.pool.GetStatistics()
}

// PooledUpstreamClient Connection Pool을 사용하는 업스트림 클라이언트
type PooledUpstreamClient struct {
	client    *http.Client
	factory   *ProxyClientFactory
	proxyType string
	logger    logging.Logger
}

// Fetch 업스트림에서 데이터 가져오기 (Connection Pool 사용)
func (c *PooledUpstreamClient) Fetch(ctx context.Context, url string, headers map[string]string) (*proxy.ProxyResponse, error) { //nolint:lll
	// 요청 생성
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}

	// 헤더 설정
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// User-Agent 설정
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", fmt.Sprintf("ProxyND/%s", c.proxyType))
	}

	// 요청 실행 (통계 수집 포함)
	resp, err := c.factory.pool.ExecuteRequest(c.client, req)
	if err != nil {
		// 에러가 발생해도 응답이 있을 수 있으므로 body를 닫아야 함
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("업스트림 요청 실패: %w", err)
	}
	// Note: resp.Body는 ProxyResponse로 전달되므로 여기서 닫지 않음

	// ProxyResponse로 변환
	proxyResp := &proxy.ProxyResponse{
		Body:        resp.Body,
		StatusCode:  resp.StatusCode,
		Headers:     make(map[string]string),
		ContentType: resp.Header.Get("Content-Type"),
		Cached:      false,
	}

	// 헤더 복사
	for key, values := range resp.Header {
		if len(values) > 0 {
			proxyResp.Headers[key] = values[0]
		}
	}

	// 파일명 추출
	if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
		proxyResp.FileName = extractFilenameFromHeader(contentDisposition)
	}

	return proxyResp, nil
}

// extractFilenameFromHeader Content-Disposition 헤더에서 파일명 추출
func extractFilenameFromHeader(header string) string {
	const filenamePrefix = "filename="
	if idx := findSubstring(header, filenamePrefix); idx != -1 {
		filename := header[idx+len(filenamePrefix):]
		// 따옴표 제거
		if len(filename) > 2 && filename[0] == '"' && filename[len(filename)-1] == '"' {
			filename = filename[1 : len(filename)-1]
		}
		return filename
	}
	return ""
}

// findSubstring 부분 문자열 찾기 (대소문자 무시)
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// 전역 클라이언트 팩토리 인스턴스
var globalClientFactory *ProxyClientFactory

// GetGlobalClientFactory 전역 클라이언트 팩토리 반환
func GetGlobalClientFactory() *ProxyClientFactory {
	if globalClientFactory == nil {
		globalClientFactory = NewProxyClientFactory(GetGlobalPool())
	}
	return globalClientFactory
}

// InitializeGlobalClientFactory 전역 클라이언트 팩토리 초기화
func InitializeGlobalClientFactory(pool *ConnectionPool) {
	globalClientFactory = NewProxyClientFactory(pool)
}
