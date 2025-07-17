package errors

import "time"

// Maven 도메인 에러 정의
var (
	ErrMavenArtifactNotFound = &DomainError{
		Code:      "MVN001",
		Message:   "Maven 아티팩트를 찾을 수 없습니다",
		Domain:    "maven",
		Level:     ErrorLevelWarning,
		Timestamp: time.Now(),
	}

	ErrMavenChecksumMismatch = &DomainError{
		Code:      "MVN002",
		Message:   "체크섬이 일치하지 않습니다",
		Domain:    "maven",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrMavenRepositoryUnavailable = &DomainError{
		Code:      "MVN003",
		Message:   "Maven 리포지토리에 접근할 수 없습니다",
		Domain:    "maven",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrMavenProxyDisabled = &DomainError{
		Code:      "MVN004",
		Message:   "Maven 프록시가 비활성화되어 있습니다",
		Domain:    "maven",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrMavenInvalidArtifactPath = &DomainError{
		Code:      "MVN005",
		Message:   "잘못된 Maven 아티팩트 경로입니다",
		Domain:    "maven",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrMavenConfigError = &DomainError{
		Code:      "MVN006",
		Message:   "Maven 설정 파일을 읽을 수 없습니다",
		Domain:    "maven",
		Level:     ErrorLevelCritical,
		Timestamp: time.Now(),
	}

	ErrMavenSnapshotDownload = &DomainError{
		Code:      "MVN007",
		Message:   "SNAPSHOT 아티팩트 다운로드에 실패했습니다",
		Domain:    "maven",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	ErrMavenInvalidFormat = &DomainError{
		Code:      "MVN008",
		Message:   "잘못된 Maven 아티팩트 형식입니다",
		Domain:    "maven",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}
)

// NewMavenError Maven 도메인 커스텀 에러 생성
func NewMavenError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("maven")
}

// WrapMavenError 기존 에러를 Maven 도메인 에러로 래핑
func WrapMavenError(err error, code, message string) *DomainError {
	return NewMavenError(code, message).WithCause(err).Build()
}
