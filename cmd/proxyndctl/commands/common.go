// Package commands provides CLI command implementations for proxyndctl
package commands

import (
	"net/http"
	"time"
)

// 공통 에러 메시지
const (
	errServerConnection = "서버 연결 실패: %v"
	errAPIRequestStatus = "API 요청 실패: HTTP %d"
	errResponseParsing  = "응답 파싱 실패: %v"
)

// 상태 아이콘 상수
const (
	iconSuccess = "✅"
	iconError   = "❌"
	iconWarning = "⚠️"
	iconUnknown = "❓"
)

// 상태 상수
const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
	statusDegraded  = "degraded"
	statusActive    = "활성"
	statusInactive  = "비활성"
)

// 날짜/시간 포맷 상수
const (
	dateTimeFormat = "2006-01-02 15:04:05"
	dateFormat     = "2006-01-02"
)

// GetHTTPClient 최적화된 HTTP 클라이언트 반환
func GetHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}
}
