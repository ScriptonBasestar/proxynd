package errors

import "github.com/gofiber/fiber/v2"

// errorToHTTPStatus 에러 코드와 HTTP 상태 코드 매핑
var errorToHTTPStatus = map[string]int{
	// APT 에러 매핑
	"APT001": fiber.StatusNotFound,            // 패키지 미발견
	"APT002": fiber.StatusServiceUnavailable, // 미러 서버 접근 불가
	"APT003": fiber.StatusBadRequest,          // 잘못된 패키지 형식
	"APT004": fiber.StatusForbidden,           // 프록시 비활성화
	"APT005": fiber.StatusBadRequest,          // 잘못된 경로
	"APT006": fiber.StatusInternalServerError, // 설정 파일 오류

	// Maven 에러 매핑
	"MVN001": fiber.StatusNotFound,              // 아티팩트 미발견
	"MVN002": fiber.StatusUnprocessableEntity,   // 체크섬 불일치
	"MVN003": fiber.StatusServiceUnavailable,    // 리포지토리 접근 불가
	"MVN004": fiber.StatusForbidden,             // 프록시 비활성화
	"MVN005": fiber.StatusBadRequest,            // 잘못된 아티팩트 경로
	"MVN006": fiber.StatusInternalServerError,   // 설정 파일 오류
	"MVN007": fiber.StatusBadGateway,            // SNAPSHOT 다운로드 실패
	"MVN008": fiber.StatusBadRequest,            // 잘못된 아티팩트 형식

	// 캐시 에러 매핑
	"CACHE001": fiber.StatusNotFound,             // 캐시 미발견
	"CACHE002": fiber.StatusInsufficientStorage, // 저장 공간 부족
	"CACHE003": fiber.StatusInternalServerError,  // 캐시 오류

	// 시스템 에러 매핑
	"SYSTEM001": fiber.StatusInternalServerError, // 일반 시스템 오류
	"CONFIG001": fiber.StatusInternalServerError, // 설정 오류
	"AUTH001":   fiber.StatusUnauthorized,        // 인증 실패
	"AUTH002":   fiber.StatusForbidden,           // 권한 없음
	"PANIC001":  fiber.StatusInternalServerError, // 패닉 발생
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