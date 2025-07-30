package errors

import "time"

// Docker 도메인 에러 정의
var (
	// ErrDockerImageNotFound Docker 이미지를 찾을 수 없음
	ErrDockerImageNotFound = &DomainError{
		Code:      "DOCKER001",
		Message:   "Docker 이미지를 찾을 수 없습니다",
		Domain:    "docker",
		Level:     ErrorLevelWarning,
		Timestamp: time.Now(),
	}

	// ErrDockerRegistryUnavailable Docker 레지스트리 서버에 접근할 수 없음
	ErrDockerRegistryUnavailable = &DomainError{
		Code:      "DOCKER002",
		Message:   "Docker 레지스트리 서버에 접근할 수 없습니다",
		Domain:    "docker",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrDockerAuthenticationRequired Docker 레지스트리 인증 필요
	ErrDockerAuthenticationRequired = &DomainError{
		Code:      "DOCKER003",
		Message:   "Docker 레지스트리 인증이 필요합니다",
		Domain:    "docker",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrDockerProxyDisabled Docker 프록시가 비활성화됨
	ErrDockerProxyDisabled = &DomainError{
		Code:      "DOCKER004",
		Message:   "Docker 프록시가 비활성화되어 있습니다",
		Domain:    "docker",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrDockerInvalidPath 잘못된 Docker 이미지 경로
	ErrDockerInvalidPath = &DomainError{
		Code:      "DOCKER005",
		Message:   "잘못된 이미지 경로입니다",
		Domain:    "docker",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrDockerConfigError Docker 설정 파일 읽기 실패
	ErrDockerConfigError = &DomainError{
		Code:      "DOCKER006",
		Message:   "Docker 설정 파일을 읽을 수 없습니다",
		Domain:    "docker",
		Level:     ErrorLevelCritical,
		Timestamp: time.Now(),
	}

	// ErrDockerManifestCorrupted Docker 매니페스트 손상
	ErrDockerManifestCorrupted = &DomainError{
		Code:      "DOCKER007",
		Message:   "Docker 매니페스트가 손상되었습니다",
		Domain:    "docker",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrDockerBlobDownload Docker blob 다운로드 실패
	ErrDockerBlobDownload = &DomainError{
		Code:      "DOCKER008",
		Message:   "Docker blob 다운로드에 실패했습니다",
		Domain:    "docker",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}

	// ErrDockerDigestMismatch Docker 다이제스트 불일치
	ErrDockerDigestMismatch = &DomainError{
		Code:      "DOCKER009",
		Message:   "Docker 이미지 다이제스트가 일치하지 않습니다",
		Domain:    "docker",
		Level:     ErrorLevelError,
		Timestamp: time.Now(),
	}
)

// NewDockerError Docker 도메인 커스텀 에러 생성
func NewDockerError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("docker")
}

// WrapDockerError 기존 에러를 Docker 도메인 에러로 래핑
func WrapDockerError(err error, code, message string) *DomainError {
	return NewDockerError(code, message).WithCause(err).Build()
}
