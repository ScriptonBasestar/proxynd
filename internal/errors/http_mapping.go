package errors

import "github.com/gofiber/fiber/v2"

// errorToHTTPStatus 에러 코드와 HTTP 상태 코드 매핑
var errorToHTTPStatus = map[string]int{
	// APT 에러 매핑
	"APT001": fiber.StatusNotFound,            // 패키지 미발견
	"APT002": fiber.StatusServiceUnavailable,  // 미러 서버 접근 불가
	"APT003": fiber.StatusBadRequest,          // 잘못된 패키지 형식
	"APT004": fiber.StatusForbidden,           // 프록시 비활성화
	"APT005": fiber.StatusBadRequest,          // 잘못된 경로
	"APT006": fiber.StatusInternalServerError, // 설정 파일 오류

	// Maven 에러 매핑
	"MAVEN001": fiber.StatusNotFound,            // 아티팩트 미발견
	"MAVEN002": fiber.StatusServiceUnavailable,  // 리포지토리 접근 불가
	"MAVEN003": fiber.StatusUnprocessableEntity, // 체크섬 검증 실패
	"MAVEN004": fiber.StatusForbidden,           // 프록시 비활성화
	"MAVEN005": fiber.StatusBadRequest,          // 잘못된 아티팩트 경로
	"MAVEN006": fiber.StatusInternalServerError, // 설정 파일 오류
	"MVN001":   fiber.StatusNotFound,            // 아티팩트 미발견 (호환성)
	"MVN002":   fiber.StatusUnprocessableEntity, // 체크섬 불일치 (호환성)
	"MVN003":   fiber.StatusServiceUnavailable,  // 리포지토리 접근 불가 (호환성)
	"MVN004":   fiber.StatusForbidden,           // 프록시 비활성화 (호환성)
	"MVN005":   fiber.StatusBadRequest,          // 잘못된 아티팩트 경로 (호환성)
	"MVN006":   fiber.StatusInternalServerError, // 설정 파일 오류 (호환성)
	"MVN007":   fiber.StatusBadGateway,          // SNAPSHOT 다운로드 실패 (호환성)
	"MVN008":   fiber.StatusBadRequest,          // 잘못된 아티팩트 형식 (호환성)

	// NPM 에러 매핑
	"NPM001": fiber.StatusNotFound,            // 패키지 미발견
	"NPM002": fiber.StatusServiceUnavailable,  // 레지스트리 접근 불가
	"NPM003": fiber.StatusBadRequest,          // 잘못된 패키지 형식
	"NPM004": fiber.StatusForbidden,           // 프록시 비활성화
	"NPM005": fiber.StatusBadRequest,          // 잘못된 경로
	"NPM006": fiber.StatusInternalServerError, // 설정 파일 오류
	"NPM007": fiber.StatusUnprocessableEntity, // 메타데이터 손상
	"NPM008": fiber.StatusBadGateway,          // tarball 다운로드 실패

	// Docker 에러 매핑
	"DOCKER001": fiber.StatusNotFound,            // 이미지 미발견
	"DOCKER002": fiber.StatusServiceUnavailable,  // 레지스트리 접근 불가
	"DOCKER003": fiber.StatusUnauthorized,        // 인증 필요
	"DOCKER004": fiber.StatusForbidden,           // 프록시 비활성화
	"DOCKER005": fiber.StatusBadRequest,          // 잘못된 경로
	"DOCKER006": fiber.StatusInternalServerError, // 설정 파일 오류
	"DOCKER007": fiber.StatusUnprocessableEntity, // 매니페스트 손상
	"DOCKER008": fiber.StatusBadGateway,          // blob 다운로드 실패
	"DOCKER009": fiber.StatusUnprocessableEntity, // 다이제스트 불일치

	// PIP 에러 매핑
	"PIP001": fiber.StatusNotFound,            // 패키지 미발견
	"PIP002": fiber.StatusServiceUnavailable,  // 미러 접근 불가
	"PIP003": fiber.StatusBadRequest,          // 잘못된 패키지 형식
	"PIP004": fiber.StatusGatewayTimeout,      // 서버 응답 타임아웃
	"PIP005": fiber.StatusBadRequest,          // 잘못된 경로
	"PIP006": fiber.StatusInternalServerError, // 설정 파일 오류
	"PIP007": fiber.StatusUnprocessableEntity, // wheel 파일 손상
	"PIP008": fiber.StatusUnprocessableEntity, // 인덱스 손상
	"PIP009": fiber.StatusForbidden,           // 프록시 비활성화

	// YUM 에러 매핑
	"YUM001": fiber.StatusNotFound,            // 패키지 미발견
	"YUM002": fiber.StatusServiceUnavailable,  // 미러 접근 불가
	"YUM003": fiber.StatusUnprocessableEntity, // RPM 파일 손상
	"YUM004": fiber.StatusGatewayTimeout,      // 서버 응답 타임아웃
	"YUM005": fiber.StatusBadRequest,          // 잘못된 경로
	"YUM006": fiber.StatusInternalServerError, // 설정 파일 오류
	"YUM007": fiber.StatusUnprocessableEntity, // 메타데이터 손상
	"YUM008": fiber.StatusUnprocessableEntity, // 서명 검증 실패
	"YUM009": fiber.StatusForbidden,           // 프록시 비활성화
	"YUM010": fiber.StatusNotFound,            // repomd.xml 미발견

	// APK 에러 매핑
	"APK001": fiber.StatusNotFound,            // 패키지 미발견
	"APK002": fiber.StatusServiceUnavailable,  // 미러 접근 불가
	"APK003": fiber.StatusUnprocessableEntity, // 서명 검증 실패
	"APK004": fiber.StatusGatewayTimeout,      // 서버 응답 타임아웃
	"APK005": fiber.StatusBadRequest,          // 잘못된 경로
	"APK006": fiber.StatusInternalServerError, // 설정 파일 오류
	"APK007": fiber.StatusUnprocessableEntity, // 인덱스 손상
	"APK008": fiber.StatusNotFound,            // 서명 키 미발견
	"APK009": fiber.StatusForbidden,           // 프록시 비활성화
	"APK010": fiber.StatusServiceUnavailable,  // 미러 선택 실패

	// 캐시 에러 매핑
	"CACHE001": fiber.StatusNotFound,            // 캐시 미발견
	"CACHE002": fiber.StatusInsufficientStorage, // 저장 공간 부족
	"CACHE003": fiber.StatusInternalServerError, // 캐시 오류

	// 시스템 에러 매핑
	"SYS001":    fiber.StatusInternalServerError, // 일반 시스템 오류
	"SYS002":    fiber.StatusInternalServerError, // 설정 오류
	"SYS003":    fiber.StatusInternalServerError, // 패닉 발생
	"SYSTEM001": fiber.StatusInternalServerError, // 일반 시스템 오류 (호환성)
	"CONFIG001": fiber.StatusInternalServerError, // 설정 오류 (호환성)
	"PANIC001":  fiber.StatusInternalServerError, // 패닉 발생 (호환성)

	// 인증/인가 에러 매핑
	"AUTH001":      fiber.StatusUnauthorized, // 인증 실패
	"AUTH002":      fiber.StatusForbidden,    // 권한 없음
	"AUTH_FAILED":  fiber.StatusUnauthorized, // 인증 실패 (호환성)
	"AUTHZ_FAILED": fiber.StatusForbidden,    // 권한 없음 (호환성)

	// 프록시 에러 매핑
	"PROXY001": fiber.StatusServiceUnavailable,  // 프록시 비활성화
	"PROXY002": fiber.StatusInternalServerError, // 설정 로드 실패
	"PROXY003": fiber.StatusInternalServerError, // 캐시 파일 읽기 실패
}

// GetHTTPStatus 도메인 에러에 해당하는 HTTP 상태 코드 반환
func GetHTTPStatus(err error) int {
	if domainErr, ok := err.(*DomainError); ok {
		if status, exists := errorToHTTPStatus[domainErr.Code]; exists {
			return status
		}

		// 에러 레벨에 따른 기본 상태 코드
		switch domainErr.Level {
		case ErrorLevelWarning:
			return fiber.StatusNotFound
		case ErrorLevelError:
			return fiber.StatusBadRequest
		case ErrorLevelCritical:
			return fiber.StatusInternalServerError
		default:
			return fiber.StatusInternalServerError
		}
	}

	// Fiber 에러 처리
	if fiberErr, ok := err.(*fiber.Error); ok {
		return fiberErr.Code
	}

	// 기본 서버 오류
	return fiber.StatusInternalServerError
}

// GetHTTPStatusByCode 에러 코드로 직접 HTTP 상태 코드 조회
func GetHTTPStatusByCode(errorCode string) int {
	if status, exists := errorToHTTPStatus[errorCode]; exists {
		return status
	}
	return fiber.StatusInternalServerError
}

// IsClientError 클라이언트 에러인지 확인 (4xx)
func IsClientError(err error) bool {
	status := GetHTTPStatus(err)
	return status >= 400 && status < 500
}

// IsServerError 서버 에러인지 확인 (5xx)
func IsServerError(err error) bool {
	status := GetHTTPStatus(err)
	return status >= 500
}
