package errors

import "time"

// YUM 도메인 에러 정의
var (
	// ErrYumPackageNotFound YUM 패키지를 찾을 수 없음
	ErrYumPackageNotFound = &DomainError{
		Code:      "YUM001",
		Message:   "YUM 패키지를 찾을 수 없습니다",
		Domain:    "yum",
		Level:     ErrorLevelWarning,
		Timestamp: time.Now(),
	}

	// ErrYumMirrorUnavailable YUM 미러 서버에 접근할 수 없음
	ErrYumMirrorUnavailable = &DomainError{
		Code:      "YUM002",
		Message:   "YUM 미러 서버에 접근할 수 없습니다",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrYumRpmCorrupted YUM RPM 파일 손상
	ErrYumRpmCorrupted = &DomainError{
		Code:      "YUM003",
		Message:   "YUM RPM 파일이 손상되었습니다",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrYumTimeout YUM 서버 응답 타임아웃
	ErrYumTimeout = &DomainError{
		Code:      "YUM004",
		Message:   "YUM 서버 응답 타임아웃",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrYumInvalidPath 잘못된 YUM 패키지 경로
	ErrYumInvalidPath = &DomainError{
		Code:      "YUM005",
		Message:   "잘못된 패키지 경로입니다",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrYumConfigError YUM 설정 파일 읽기 실패
	ErrYumConfigError = &DomainError{
		Code:      "YUM006",
		Message:   "YUM 설정 파일을 읽을 수 없습니다",
		Domain:    "yum",
		Level:     ErrorLevelCritical,
		Timestamp: time.Now(),
	}

	// ErrYumMetadataCorrupted YUM 메타데이터 손상
	ErrYumMetadataCorrupted = &DomainError{
		Code:      "YUM007",
		Message:   "YUM 메타데이터가 손상되었습니다",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrYumSignatureVerification YUM 패키지 서명 검증 실패
	ErrYumSignatureVerification = &DomainError{
		Code:      "YUM008",
		Message:   "YUM 패키지 서명 검증에 실패했습니다",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrYumProxyDisabled YUM 프록시가 비활성화됨
	ErrYumProxyDisabled = &DomainError{
		Code:      "YUM009",
		Message:   "YUM 프록시가 비활성화되어 있습니다",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrYumRepomdNotFound YUM repomd.xml을 찾을 수 없음
	ErrYumRepomdNotFound = &DomainError{
		Code:      "YUM010",
		Message:   "YUM repomd.xml을 찾을 수 없습니다",
		Domain:    "yum",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}
)

// NewYumError YUM 도메인 커스텀 에러 생성
func NewYumError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("yum")
}

// WrapYumError 기존 에러를 YUM 도메인 에러로 래핑
func WrapYumError(err error, code, message string) *DomainError {
	return NewYumError(code, message).WithCause(err).Build()
}
