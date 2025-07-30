package errors

import "time"

// NPM 도메인 에러 정의
var (
	// ErrNPMPackageNotFound NPM 패키지를 찾을 수 없음
	ErrNPMPackageNotFound = &DomainError{
		Code:      "NPM001",
		Message:   "NPM 패키지를 찾을 수 없습니다",
		Domain:    "npm",
		Level:     ErrorLevelWarning,
		Timestamp: time.Now(),
	}

	// ErrNPMRegistryUnavailable NPM 레지스트리 서버에 접근할 수 없음
	ErrNPMRegistryUnavailable = &DomainError{
		Code:      "NPM002",
		Message:   "NPM 레지스트리 서버에 접근할 수 없습니다",
		Domain:    "npm",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrNPMInvalidPackageFormat 잘못된 NPM 패키지 형식
	ErrNPMInvalidPackageFormat = &DomainError{
		Code:      "NPM003",
		Message:   "잘못된 NPM 패키지 형식입니다",
		Domain:    "npm",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrNPMProxyDisabled NPM 프록시가 비활성화됨
	ErrNPMProxyDisabled = &DomainError{
		Code:      "NPM004",
		Message:   "NPM 프록시가 비활성화되어 있습니다",
		Domain:    "npm",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrNPMInvalidPath 잘못된 NPM 패키지 경로
	ErrNPMInvalidPath = &DomainError{
		Code:      "NPM005",
		Message:   "잘못된 패키지 경로입니다",
		Domain:    "npm",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrNPMConfigError NPM 설정 파일 읽기 실패
	ErrNPMConfigError = &DomainError{
		Code:      "NPM006",
		Message:   "NPM 설정 파일을 읽을 수 없습니다",
		Domain:    "npm",
		Level:     ErrorLevelCritical,
		Timestamp: time.Now(),
	}

	// ErrNPMMetadataCorrupted NPM 메타데이터 손상
	ErrNPMMetadataCorrupted = &DomainError{
		Code:      "NPM007",
		Message:   "NPM 패키지 메타데이터가 손상되었습니다",
		Domain:    "npm",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrNPMTarballDownload NPM tarball 다운로드 실패
	ErrNPMTarballDownload = &DomainError{
		Code:      "NPM008",
		Message:   "NPM tarball 다운로드에 실패했습니다",
		Domain:    "npm",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}
)

// NewNPMError NPM 도메인 커스텀 에러 생성
func NewNPMError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("npm")
}

// WrapNPMError 기존 에러를 NPM 도메인 에러로 래핑
func WrapNPMError(err error, code, message string) *DomainError {
	return NewNPMError(code, message).WithCause(err).Build()
}
