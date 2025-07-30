package errors

import "time"

// PIP 도메인 에러 정의
var (
	// ErrPipPackageNotFound PIP 패키지를 찾을 수 없음
	ErrPipPackageNotFound = &DomainError{
		Code:      "PIP001",
		Message:   "PIP 패키지를 찾을 수 없습니다",
		Domain:    "pip",
		Level:     ErrorLevelWarning,
		Timestamp: time.Now(),
	}

	// ErrPipMirrorUnavailable PIP 미러 서버에 접근할 수 없음
	ErrPipMirrorUnavailable = &DomainError{
		Code:      "PIP002",
		Message:   "PIP 미러 서버에 접근할 수 없습니다",
		Domain:    "pip",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrPipInvalidPackageFormat 잘못된 PIP 패키지 형식
	ErrPipInvalidPackageFormat = &DomainError{
		Code:      "PIP003",
		Message:   "잘못된 PIP 패키지 형식입니다",
		Domain:    "pip",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrPipTimeout PIP 서버 응답 타임아웃
	ErrPipTimeout = &DomainError{
		Code:      "PIP004",
		Message:   "PIP 서버 응답 타임아웃",
		Domain:    "pip",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrPipInvalidPath 잘못된 PIP 패키지 경로
	ErrPipInvalidPath = &DomainError{
		Code:      "PIP005",
		Message:   "잘못된 패키지 경로입니다",
		Domain:    "pip",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrPipConfigError PIP 설정 파일 읽기 실패
	ErrPipConfigError = &DomainError{
		Code:      "PIP006",
		Message:   "PIP 설정 파일을 읽을 수 없습니다",
		Domain:    "pip",
		Level:     ErrorLevelCritical,
		Timestamp: time.Now(),
	}

	// ErrPipWheelCorrupted PIP wheel 파일 손상
	ErrPipWheelCorrupted = &DomainError{
		Code:      "PIP007",
		Message:   "PIP wheel 파일이 손상되었습니다",
		Domain:    "pip",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrPipIndexCorrupted PIP 인덱스 손상
	ErrPipIndexCorrupted = &DomainError{
		Code:      "PIP008",
		Message:   "PIP 패키지 인덱스가 손상되었습니다",
		Domain:    "pip",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrPipProxyDisabled PIP 프록시가 비활성화됨
	ErrPipProxyDisabled = &DomainError{
		Code:      "PIP009",
		Message:   "PIP 프록시가 비활성화되어 있습니다",
		Domain:    "pip",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}
)

// NewPipError PIP 도메인 커스텀 에러 생성
func NewPipError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("pip")
}

// WrapPipError 기존 에러를 PIP 도메인 에러로 래핑
func WrapPipError(err error, code, message string) *DomainError {
	return NewPipError(code, message).WithCause(err).Build()
}
