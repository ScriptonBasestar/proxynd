package errors

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDomainError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DomainError
		expected string
	}{
		{
			name: "기본 도메인 에러",
			err: &DomainError{
				Code:    "TEST001",
				Message: "테스트 에러입니다",
				Domain:  "test",
			},
			expected: "[test] TEST001: 테스트 에러입니다",
		},
		{
			name: "원인 에러가 있는 도메인 에러",
			err: &DomainError{
				Code:    "TEST002",
				Message: "래핑된 에러입니다",
				Domain:  "test",
				Cause:   errors.New("원본 에러"),
			},
			expected: "[test] TEST002: 래핑된 에러입니다 (caused by: 원본 에러)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.err.Error()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestErrorBuilder(t *testing.T) {
	t.Run("기본 에러 빌더", func(t *testing.T) {
		err := NewError("TEST001", "테스트 메시지").Build()

		assert.Equal(t, "TEST001", err.Code)
		assert.Equal(t, "테스트 메시지", err.Message)
		assert.Equal(t, ErrorLevelError, err.Level)
		assert.WithinDuration(t, time.Now(), err.Timestamp, time.Second)
	})

	t.Run("모든 필드가 설정된 에러", func(t *testing.T) {
		originalErr := errors.New("원본 에러")
		details := map[string]interface{}{"key": "value"}

		err := NewError("TEST002", "복합 에러").
			WithDomain("test").
			WithLevel(ErrorLevelCritical).
			WithCause(originalErr).
			WithDetails(details).
			Build()

		assert.Equal(t, "TEST002", err.Code)
		assert.Equal(t, "복합 에러", err.Message)
		assert.Equal(t, "test", err.Domain)
		assert.Equal(t, ErrorLevelCritical, err.Level)
		assert.Equal(t, originalErr, err.Cause)
		assert.Equal(t, details, err.Details)
	})
}

func TestErrorLevel_String(t *testing.T) {
	tests := []struct {
		level    ErrorLevel
		expected string
	}{
		{ErrorLevelInfo, "info"},
		{ErrorLevelWarning, "warning"},
		{ErrorLevelError, "error"},
		{ErrorLevelCritical, "critical"},
		{ErrorLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			actual := tt.level.String()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestAPTErrors(t *testing.T) {
	t.Run("기본 APT 에러", func(t *testing.T) {
		err := ErrAPTPackageNotFound
		assert.Equal(t, "APT001", err.Code)
		assert.Equal(t, "apt", err.Domain)
		assert.Equal(t, ErrorLevelWarning, err.Level)
	})

	t.Run("NewAPTError 함수", func(t *testing.T) {
		err := NewAPTError("APT999", "커스텀 APT 에러").Build()
		assert.Equal(t, "APT999", err.Code)
		assert.Equal(t, "커스텀 APT 에러", err.Message)
		assert.Equal(t, "apt", err.Domain)
	})

	t.Run("WrapAPTError 함수", func(t *testing.T) {
		originalErr := errors.New("원본 APT 에러")
		err := WrapAPTError(originalErr, "APT998", "래핑된 APT 에러")

		assert.Equal(t, "APT998", err.Code)
		assert.Equal(t, "래핑된 APT 에러", err.Message)
		assert.Equal(t, "apt", err.Domain)
		assert.Equal(t, originalErr, err.Cause)
	})
}

func TestMavenErrors(t *testing.T) {
	t.Run("기본 Maven 에러", func(t *testing.T) {
		err := ErrMavenArtifactNotFound
		assert.Equal(t, "MVN001", err.Code)
		assert.Equal(t, "maven", err.Domain)
		assert.Equal(t, ErrorLevelWarning, err.Level)
	})

	t.Run("NewMavenError 함수", func(t *testing.T) {
		err := NewMavenError("MVN999", "커스텀 Maven 에러").Build()
		assert.Equal(t, "MVN999", err.Code)
		assert.Equal(t, "커스텀 Maven 에러", err.Message)
		assert.Equal(t, "maven", err.Domain)
	})

	t.Run("WrapMavenError 함수", func(t *testing.T) {
		originalErr := errors.New("원본 Maven 에러")
		err := WrapMavenError(originalErr, "MVN998", "래핑된 Maven 에러")

		assert.Equal(t, "MVN998", err.Code)
		assert.Equal(t, "래핑된 Maven 에러", err.Message)
		assert.Equal(t, "maven", err.Domain)
		assert.Equal(t, originalErr, err.Cause)
	})
}
