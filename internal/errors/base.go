package errors

import (
	"fmt"
	"time"
)

// ErrorLevel 에러 심각도 레벨
type ErrorLevel int

const (
	ErrorLevelInfo ErrorLevel = iota
	ErrorLevelWarning
	ErrorLevelError
	ErrorLevelCritical
)

// DomainError 도메인별 구조화된 에러
type DomainError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Domain    string      `json:"domain"`
	Level     ErrorLevel  `json:"level"`
	Timestamp time.Time   `json:"timestamp"`
	Details   interface{} `json:"details,omitempty"`
	Cause     error       `json:"-"`
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)",
			e.Domain, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Domain, e.Code, e.Message)
}

// ErrorBuilder 에러 빌더 패턴
type ErrorBuilder struct {
	err *DomainError
}

// NewError 새로운 에러 빌더 생성
func NewError(code, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &DomainError{
			Code:      code,
			Message:   message,
			Timestamp: time.Now(),
			Level:     ErrorLevelError,
		},
	}
}

// WithDomain 도메인 설정
func (b *ErrorBuilder) WithDomain(domain string) *ErrorBuilder {
	b.err.Domain = domain
	return b
}

// WithCause 원인 에러 설정
func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	b.err.Cause = cause
	return b
}

// WithDetails 추가 상세 정보 설정
func (b *ErrorBuilder) WithDetails(details interface{}) *ErrorBuilder {
	b.err.Details = details
	return b
}

// WithLevel 에러 레벨 설정
func (b *ErrorBuilder) WithLevel(level ErrorLevel) *ErrorBuilder {
	b.err.Level = level
	return b
}

// Build 최종 에러 객체 생성
func (b *ErrorBuilder) Build() *DomainError {
	return b.err
}

// String 에러 레벨을 문자열로 변환
func (e ErrorLevel) String() string {
	switch e {
	case ErrorLevelInfo:
		return "info"
	case ErrorLevelWarning:
		return "warning"
	case ErrorLevelError:
		return "error"
	case ErrorLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}
