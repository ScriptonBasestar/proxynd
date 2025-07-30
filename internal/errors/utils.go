package errors

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ErrorDetails HTTP 응답용 에러 세부 정보
type ErrorDetails struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Domain    string      `json:"domain"`
	Level     string      `json:"level"`
	Timestamp string      `json:"timestamp"`
	Details   interface{} `json:"details,omitempty"`
}

// ProxyErrorResponse 프록시 에러 응답 구조체
type ProxyErrorResponse struct {
	Error       ErrorDetails `json:"error"`
	Path        string       `json:"path,omitempty"`
	Method      string       `json:"method,omitempty"`
	UserAgent   string       `json:"user_agent,omitempty"`
	ClientIP    string       `json:"client_ip,omitempty"`
	RequestID   string       `json:"request_id,omitempty"`
	Suggestions []string     `json:"suggestions,omitempty"`
}

// SendProxyError 프록시 에러를 HTTP 응답으로 전송
func SendProxyError(c *fiber.Ctx, err error) error {
	var domainErr *DomainError
	var ok bool

	// DomainError로 변환
	if domainErr, ok = err.(*DomainError); !ok {
		// 일반 에러를 DomainError로 변환
		domainErr = NewError("UNKNOWN001", err.Error()).
			WithDomain("system").
			WithLevel(ErrorLevelError).
			Build()
	}

	// HTTP 상태 코드 결정
	statusCode := GetHTTPStatus(domainErr)

	// 에러 응답 구성
	response := ProxyErrorResponse{
		Error: ErrorDetails{
			Code:      domainErr.Code,
			Message:   domainErr.Message,
			Domain:    domainErr.Domain,
			Level:     domainErr.Level.String(),
			Timestamp: domainErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Details:   domainErr.Details,
		},
		Path:      c.Path(),
		Method:    c.Method(),
		UserAgent: c.Get("User-Agent"),
		ClientIP:  c.IP(),
		RequestID: c.Get("X-Request-ID"),
	}

	// 도메인별 해결책 제안
	response.Suggestions = generateSuggestions(domainErr)

	// 응답 헤더 설정
	c.Set("Content-Type", "application/json")
	c.Set("X-Error-Code", domainErr.Code)
	c.Set("X-Error-Domain", domainErr.Domain)

	return c.Status(statusCode).JSON(response)
}

// generateSuggestions 에러 코드에 따른 해결책 제안 생성
func generateSuggestions(err *DomainError) []string {
	suggestions := make([]string, 0)

	switch {
	// 패키지/이미지 미발견 에러
	case strings.HasSuffix(err.Code, "001"):
		suggestions = append(suggestions,
			"패키지명이나 버전을 확인해주세요",
			"다른 미러 서버를 시도해보세요",
			"캐시를 지우고 다시 시도해보세요")

	// 서버 접근 불가 에러 (002)
	case strings.HasSuffix(err.Code, "002"):
		suggestions = append(suggestions,
			"네트워크 연결을 확인해주세요",
			"방화벽 설정을 확인해주세요",
			"프록시 서버 상태를 확인해주세요")

	// 인증 관련 에러
	case strings.Contains(err.Code, "AUTH") || strings.HasSuffix(err.Code, "003"):
		suggestions = append(suggestions,
			"인증 정보를 확인해주세요",
			"토큰이 만료되었는지 확인해주세요",
			"권한 설정을 확인해주세요")

	// 타임아웃 에러 (004)
	case strings.HasSuffix(err.Code, "004"):
		suggestions = append(suggestions,
			"잠시 후 다시 시도해주세요",
			"네트워크 연결 상태를 확인해주세요",
			"타임아웃 설정을 늘려보세요")

	// 경로 관련 에러 (005)
	case strings.HasSuffix(err.Code, "005"):
		suggestions = append(suggestions,
			"요청 경로가 올바른지 확인해주세요",
			"URL 인코딩을 확인해주세요",
			"API 문서를 참조해주세요")

	// 설정 파일 에러 (006)
	case strings.HasSuffix(err.Code, "006"):
		suggestions = append(suggestions,
			"설정 파일 권한을 확인해주세요",
			"설정 파일 형식을 확인해주세요",
			"관리자에게 문의해주세요")

	// 프록시 비활성화 에러
	case strings.Contains(err.Message, "비활성화"):
		suggestions = append(suggestions,
			"프록시 설정을 활성화해주세요",
			"설정 파일에서 enabled: true로 설정해주세요",
			"관리자에게 문의해주세요")

	default:
		suggestions = append(suggestions,
			"잠시 후 다시 시도해주세요",
			"문제가 지속되면 관리자에게 문의해주세요")
	}

	return suggestions
}

// IsRetryableError 재시도 가능한 에러인지 확인
func IsRetryableError(err error) bool {
	if domainErr, ok := err.(*DomainError); ok {
		switch {
		// 서버 접근 불가, 타임아웃 등은 재시도 가능
		case strings.HasSuffix(domainErr.Code, "002"), // 서버 접근 불가
			strings.HasSuffix(domainErr.Code, "004"): // 타임아웃
			return true
		// 캐시 관련 에러도 재시도 가능
		case strings.HasPrefix(domainErr.Code, "CACHE"):
			return true
		// 5xx 계열 HTTP 상태 코드는 재시도 가능
		case IsServerError(domainErr):
			return true
		}
	}
	return false
}

// GetErrorSeverity 에러 심각도 반환 (로깅용)
func GetErrorSeverity(err error) string {
	if domainErr, ok := err.(*DomainError); ok {
		return domainErr.Level.String()
	}
	return "error"
}

// FormatErrorForLogging 로깅용 에러 포맷팅
func FormatErrorForLogging(err error, context map[string]interface{}) map[string]interface{} {
	logData := make(map[string]interface{})

	if domainErr, ok := err.(*DomainError); ok {
		logData["error_code"] = domainErr.Code
		logData["error_message"] = domainErr.Message
		logData["error_domain"] = domainErr.Domain
		logData["error_level"] = domainErr.Level.String()
		logData["error_timestamp"] = domainErr.Timestamp
		if domainErr.Details != nil {
			logData["error_details"] = domainErr.Details
		}
		if domainErr.Cause != nil {
			logData["error_cause"] = domainErr.Cause.Error()
		}
	} else {
		logData["error_message"] = err.Error()
		logData["error_level"] = "error"
	}

	// 컨텍스트 정보 추가
	for k, v := range context {
		logData[k] = v
	}

	return logData
}

// ErrorCodeExists 에러 코드가 정의되어 있는지 확인
func ErrorCodeExists(code string) bool {
	_, exists := errorToHTTPStatus[code]
	return exists
}

// GetErrorsByDomain 도메인별 에러 코드 목록 반환
func GetErrorsByDomain(domain string) []string {
	var codes []string
	domainPrefix := strings.ToUpper(domain)

	for code := range errorToHTTPStatus {
		if strings.HasPrefix(code, domainPrefix) {
			codes = append(codes, code)
		}
	}

	return codes
}

// ValidateErrorCode 에러 코드 형식 검증
func ValidateErrorCode(code string) error {
	if code == "" {
		return fmt.Errorf("error code cannot be empty")
	}

	// 기본 형식: DOMAIN001, DOMAIN_SPECIFIC 등
	if len(code) < 3 {
		return fmt.Errorf("error code too short: %s", code)
	}

	// 영문자, 숫자, 밑줄만 허용
	for _, char := range code {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return fmt.Errorf("invalid character in error code: %c", char)
		}
	}

	return nil
}
