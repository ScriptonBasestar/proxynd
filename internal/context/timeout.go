// Package context provides context utilities for request handling
package context

import (
	"context"
	"time"
)

// 기본 타임아웃 상수 정의
const (
	// DefaultTimeout provides the default timeout
	// DownloadTimeout is a const that download timeout
	// UploadTimeout is a const that upload timeout
	// ShortTimeout is a const that short timeout
	// LongTimeout is a const that long timeout
	DefaultTimeout  = 30 * time.Second // 기본 요청 타임아웃
	DownloadTimeout = 5 * time.Minute  // 파일 다운로드 타임아웃
	UploadTimeout   = 10 * time.Minute // 파일 업로드 타임아웃
	ShortTimeout    = 5 * time.Second  // 빠른 작업용 타임아웃
	LongTimeout     = 2 * time.Minute  // 긴 작업용 타임아웃
)

// WithDefaultTimeout 기본 타임아웃이 설정된 컨텍스트 생성
func WithDefaultTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DefaultTimeout)
}

// WithDownloadTimeout 다운로드 타임아웃이 설정된 컨텍스트 생성
func WithDownloadTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DownloadTimeout)
}

// WithUploadTimeout 업로드 타임아웃이 설정된 컨텍스트 생성
func WithUploadTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, UploadTimeout)
}

// WithShortTimeout 짧은 타임아웃이 설정된 컨텍스트 생성
func WithShortTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, ShortTimeout)
}

// WithLongTimeout 긴 타임아웃이 설정된 컨텍스트 생성
func WithLongTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, LongTimeout)
}

// WithCustomTimeout 커스텀 타임아웃이 설정된 컨텍스트 생성
func WithCustomTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// Background 애플리케이션 시작 시 사용하는 루트 컨텍스트
func Background() context.Context {
	return context.Background()
}

// TODO 향후 구현 예정인 기능에 대한 컨텍스트
func TODO() context.Context {
	return context.TODO()
}

// IsContextDone 컨텍스트가 취소되었는지 확인
func IsContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// GetContextError 컨텍스트 에러 반환
func GetContextError(ctx context.Context) error {
	return ctx.Err()
}
