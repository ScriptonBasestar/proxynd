package errors

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name: "APT 패키지 미발견",
			err: &DomainError{
				Code:   "APT001",
				Domain: "apt",
				Level:  ErrorLevelWarning,
			},
			expected: fiber.StatusNotFound,
		},
		{
			name: "APT 미러 서버 접근 불가",
			err: &DomainError{
				Code:   "APT002",
				Domain: "apt",
				Level:  ErrorLevelError,
			},
			expected: fiber.StatusServiceUnavailable,
		},
		{
			name: "Maven 아티팩트 미발견",
			err: &DomainError{
				Code:   "MVN001",
				Domain: "maven",
				Level:  ErrorLevelWarning,
			},
			expected: fiber.StatusNotFound,
		},
		{
			name: "Maven 체크섬 불일치",
			err: &DomainError{
				Code:   "MVN002",
				Domain: "maven",
				Level:  ErrorLevelError,
			},
			expected: fiber.StatusUnprocessableEntity,
		},
		{
			name: "알 수 없는 에러 코드 - 경고 레벨",
			err: &DomainError{
				Code:   "UNKNOWN001",
				Domain: "unknown",
				Level:  ErrorLevelWarning,
			},
			expected: fiber.StatusNotFound,
		},
		{
			name: "알 수 없는 에러 코드 - 에러 레벨",
			err: &DomainError{
				Code:   "UNKNOWN002",
				Domain: "unknown",
				Level:  ErrorLevelError,
			},
			expected: fiber.StatusBadRequest,
		},
		{
			name: "알 수 없는 에러 코드 - 치명적 레벨",
			err: &DomainError{
				Code:   "UNKNOWN003",
				Domain: "unknown",
				Level:  ErrorLevelCritical,
			},
			expected: fiber.StatusInternalServerError,
		},
		{
			name:     "Fiber 에러",
			err:      fiber.NewError(fiber.StatusBadRequest, "잘못된 요청"),
			expected: fiber.StatusBadRequest,
		},
		{
			name:     "일반 에러",
			err:      errors.New("일반적인 에러"),
			expected: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetHTTPStatus(tt.err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetHTTPStatusByCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "APT 패키지 미발견",
			code:     "APT001",
			expected: fiber.StatusNotFound,
		},
		{
			name:     "Maven 체크섬 불일치",
			code:     "MVN002",
			expected: fiber.StatusUnprocessableEntity,
		},
		{
			name:     "알 수 없는 코드",
			code:     "UNKNOWN999",
			expected: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetHTTPStatusByCode(tt.code)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "APT 잘못된 경로 (4xx)",
			err: &DomainError{
				Code:   "APT005",
				Domain: "apt",
			},
			expected: true,
		},
		{
			name: "APT 설정 파일 오류 (5xx)",
			err: &DomainError{
				Code:   "APT006",
				Domain: "apt",
			},
			expected: false,
		},
		{
			name:     "일반 에러 (5xx)",
			err:      errors.New("일반 에러"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsClientError(tt.err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIsServerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "APT 잘못된 경로 (4xx)",
			err: &DomainError{
				Code:   "APT005",
				Domain: "apt",
			},
			expected: false,
		},
		{
			name: "APT 설정 파일 오류 (5xx)",
			err: &DomainError{
				Code:   "APT006",
				Domain: "apt",
			},
			expected: true,
		},
		{
			name:     "일반 에러 (5xx)",
			err:      errors.New("일반 에러"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsServerError(tt.err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
