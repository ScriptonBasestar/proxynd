package errors

import "time"

// APT 도메인 에러 정의
var (
	ErrAPTPackageNotFound = &DomainError{
		Code:      "APT001",
		Message:   "요청한 APT 패키지를 찾을 수 없습니다",
		Domain:    "apt",
		Level:     ErrorLevelWarning,
		Timestamp: time.Now(),
	}

	ErrAPTMirrorUnavailable = &DomainError{
		Code:      "APT002",
		Message:   "APT 미러 서버에 접근할 수 없습니다",
		Domain:    "apt",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrAPTInvalidPackageFormat = &DomainError{
		Code:      "APT003",
		Message:   "잘못된 APT 패키지 형식입니다",
		Domain:    "apt",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrAPTProxyDisabled = &DomainError{
		Code:      "APT004",
		Message:   "APT 프록시가 비활성화되어 있습니다",
		Domain:    "apt",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrAPTInvalidPath = &DomainError{
		Code:      "APT005",
		Message:   "잘못된 APT 패키지 경로입니다",
		Domain:    "apt",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrAPTConfigError = &DomainError{
		Code:      "APT006",
		Message:   "APT 설정 파일을 읽을 수 없습니다",
		Domain:    "apt",
		Level:     ErrorLevelCritical,
		Timestamp: time.Now(),
	}
)

// NewAPTError APT 도메인 커스텀 에러 생성
func NewAPTError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("apt")
}

// WrapAPTError 기존 에러를 APT 도메인 에러로 래핑
func WrapAPTError(err error, code, message string) *DomainError {
	return NewAPTError(code, message).WithCause(err).Build()
}