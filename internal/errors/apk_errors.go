package errors

import "time"

// APK 도메인 에러 정의
var (
	// ErrApkPackageNotFound APK 패키지를 찾을 수 없음
	ErrApkPackageNotFound = &DomainError{
		Code:      "APK001",
		Message:   "APK 패키지를 찾을 수 없습니다",
		Domain:    "apk",
		Level:     ErrorLevelWarning,
		Timestamp: time.Now(),
	}

	// ErrApkMirrorUnavailable APK 미러 서버에 접근할 수 없음
	ErrApkMirrorUnavailable = &DomainError{
		Code:      "APK002",
		Message:   "APK 미러 서버에 접근할 수 없습니다",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrApkSignatureVerification APK 패키지 서명 검증 실패
	ErrApkSignatureVerification = &DomainError{
		Code:      "APK003",
		Message:   "APK 패키지 서명 검증에 실패했습니다",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrApkTimeout APK 서버 응답 타임아웃
	ErrApkTimeout = &DomainError{
		Code:      "APK004",
		Message:   "APK 서버 응답 타임아웃",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrApkInvalidPath 잘못된 APK 패키지 경로
	ErrApkInvalidPath = &DomainError{
		Code:      "APK005",
		Message:   "잘못된 패키지 경로입니다",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrApkConfigError APK 설정 파일 읽기 실패
	ErrApkConfigError = &DomainError{
		Code:      "APK006",
		Message:   "APK 설정 파일을 읽을 수 없습니다",
		Domain:    "apk",
		Level:     ErrorLevelCritical,
		Timestamp: time.Now(),
	}

	// ErrApkIndexCorrupted APK 인덱스 손상
	ErrApkIndexCorrupted = &DomainError{
		Code:      "APK007",
		Message:   "APK 패키지 인덱스가 손상되었습니다",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrApkKeyNotFound APK 서명 키를 찾을 수 없음
	ErrApkKeyNotFound = &DomainError{
		Code:      "APK008",
		Message:   "APK 서명 키를 찾을 수 없습니다",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrApkProxyDisabled APK 프록시가 비활성화됨
	ErrApkProxyDisabled = &DomainError{
		Code:      "APK009",
		Message:   "APK 프록시가 비활성화되어 있습니다",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrApkMirrorSelectionFailed APK 미러 선택 실패
	ErrApkMirrorSelectionFailed = &DomainError{
		Code:      "APK010",
		Message:   "APK 미러 선택에 실패했습니다",
		Domain:    "apk",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}
)

// NewApkError APK 도메인 커스텀 에러 생성
func NewApkError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("apk")
}

// WrapApkError 기존 에러를 APK 도메인 에러로 래핑
func WrapApkError(err error, code, message string) *DomainError {
	return NewApkError(code, message).WithCause(err).Build()
}
