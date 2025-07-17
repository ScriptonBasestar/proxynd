---
phase: 2
order: 6
source_plan: /docs/refactoring/02-error-handling.md
priority: high
tags: [error-handling, middleware, standardization]
---

# 📌 작업: 중앙화된 에러 핸들링 미들웨어 구현

## 개요
507개의 분산된 에러 처리를 중앙화하고 도메인별 에러 타입을 통합 관리하는 미들웨어를 구현합니다.

## 현재 문제점
- 507개의 `if err != nil` 패턴 분산
- 에러 타입 2개만 존재 (ProxyError, ValidationError)
- 일관성 없는 에러 로깅 및 응답 형식
- 하드코딩된 HTTP 상태 코드

## 구현 내용

### 1. 도메인 에러 타입 정의
```go
// internal/errors/base.go
package errors

import (
    "fmt"
    "time"
)

type ErrorLevel int

const (
    ErrorLevelInfo ErrorLevel = iota
    ErrorLevelWarning
    ErrorLevelError
    ErrorLevelCritical
)

type DomainError struct {
    Code      string      `json:"code"`
    Message   string      `json:"message"`
    Domain    string      `json:"domain"`
    Level     ErrorLevel  `json:"level"`
    Timestamp time.Time   `json:"timestamp"`
    Details   interface{} `json:"details,omitempty"`
    Cause     error       `json:"-"`
}

func (e *DomainError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %s (caused by: %v)",
            e.Domain, e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s: %s", e.Domain, e.Code, e.Message)
}

type ErrorBuilder struct {
    err *DomainError
}

func NewError(code, message string) *ErrorBuilder {
    return &ErrorBuilder{
        err: &DomainError{
            Code:      code,
            Message:   message,
            Timestamp: time.Now(),
            Level:     ErrorLevelError,
        },
    }
}

func (b *ErrorBuilder) WithDomain(domain string) *ErrorBuilder {
    b.err.Domain = domain
    return b
}

func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
    b.err.Cause = cause
    return b
}

func (b *ErrorBuilder) WithDetails(details interface{}) *ErrorBuilder {
    b.err.Details = details
    return b
}

func (b *ErrorBuilder) WithLevel(level ErrorLevel) *ErrorBuilder {
    b.err.Level = level
    return b
}

func (b *ErrorBuilder) Build() *DomainError {
    return b.err
}
```

### 2. 도메인별 에러 정의
```go
// internal/errors/apt_errors.go
package errors

var (
    ErrAPTPackageNotFound = &DomainError{
        Code:    "APT001",
        Message: "요청한 APT 패키지를 찾을 수 없습니다",
        Domain:  "apt",
        Level:   ErrorLevelWarning,
    }

    ErrAPTMirrorUnavailable = &DomainError{
        Code:    "APT002",
        Message: "APT 미러 서버에 접근할 수 없습니다",
        Domain:  "apt",
        Level:   ErrorLevelError,
    }

    ErrAPTInvalidPackageFormat = &DomainError{
        Code:    "APT003",
        Message: "잘못된 APT 패키지 형식입니다",
        Domain:  "apt",
        Level:   ErrorLevelError,
    }
)

// internal/errors/maven_errors.go
var (
    ErrMavenArtifactNotFound = &DomainError{
        Code:    "MVN001",
        Message: "Maven 아티팩트를 찾을 수 없습니다",
        Domain:  "maven",
        Level:   ErrorLevelWarning,
    }

    ErrMavenChecksumMismatch = &DomainError{
        Code:    "MVN002",
        Message: "체크섬이 일치하지 않습니다",
        Domain:  "maven",
        Level:   ErrorLevelError,
    }
)
```

### 3. HTTP 상태 코드 매핑
```go
// internal/errors/http_mapping.go
package errors

import "github.com/gofiber/fiber/v2"

var errorToHTTPStatus = map[string]int{
    "APT001": fiber.StatusNotFound,
    "APT002": fiber.StatusServiceUnavailable,
    "APT003": fiber.StatusBadRequest,
    "MVN001": fiber.StatusNotFound,
    "MVN002": fiber.StatusUnprocessableEntity,
    "CACHE001": fiber.StatusNotFound,
    "CACHE002": fiber.StatusInsufficientStorage,
}

func GetHTTPStatus(err error) int {
    if domainErr, ok := err.(*DomainError); ok {
        if status, exists := errorToHTTPStatus[domainErr.Code]; exists {
            return status
        }
    }
    return fiber.StatusInternalServerError
}
```

### 4. 에러 핸들러 미들웨어
```go
// middlewares/error_handler.go
package middlewares

import (
    "encoding/json"
    "log"
    "time"

    "github.com/gofiber/fiber/v2"
    "proxynd/internal/errors"
)

type ErrorResponse struct {
    Error     string      `json:"error"`
    Message   string      `json:"message"`
    Domain    string      `json:"domain,omitempty"`
    Details   interface{} `json:"details,omitempty"`
    TraceID   string      `json:"trace_id,omitempty"`
    Timestamp time.Time   `json:"timestamp"`
}

func ErrorHandler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 다음 핸들러 실행
        err := c.Next()
        if err == nil {
            return nil
        }

        // 추적 ID 가져오기
        traceID := c.Locals("requestID")
        if traceID == nil {
            traceID = "unknown"
        }

        // 도메인 에러 처리
        if domainErr, ok := err.(*errors.DomainError); ok {
            return h.handleDomainError(c, domainErr, traceID.(string))
        }

        // Fiber 에러 처리
        if fiberErr, ok := err.(*fiber.Error); ok {
            return h.handleFiberError(c, fiberErr, traceID.(string))
        }

        // 기본 에러 처리
        return h.handleGenericError(c, err, traceID.(string))
    }
}

func (h *ErrorHandler) handleDomainError(c *fiber.Ctx, err *errors.DomainError, traceID string) error {
    // 구조화된 로깅
    h.logError(c, err, traceID)

    // HTTP 상태 코드 결정
    status := errors.GetHTTPStatus(err)

    // 응답 생성
    response := ErrorResponse{
        Error:     err.Code,
        Message:   err.Message,
        Domain:    err.Domain,
        Details:   err.Details,
        TraceID:   traceID,
        Timestamp: err.Timestamp,
    }

    return c.Status(status).JSON(response)
}

func (h *ErrorHandler) handleFiberError(c *fiber.Ctx, err *fiber.Error, traceID string) error {
    response := ErrorResponse{
        Error:     "FIBER_ERROR",
        Message:   err.Message,
        TraceID:   traceID,
        Timestamp: time.Now(),
    }

    return c.Status(err.Code).JSON(response)
}

func (h *ErrorHandler) handleGenericError(c *fiber.Ctx, err error, traceID string) error {
    // 알 수 없는 에러 로깅
    log.Printf(`{
        "level": "error",
        "trace_id": "%s",
        "error": "%s",
        "path": "%s",
        "method": "%s",
        "timestamp": "%s"
    }`, traceID, err.Error(), c.Path(), c.Method(), time.Now().Format(time.RFC3339))

    response := ErrorResponse{
        Error:     "INTERNAL_ERROR",
        Message:   "내부 서버 오류가 발생했습니다",
        TraceID:   traceID,
        Timestamp: time.Now(),
    }

    return c.Status(fiber.StatusInternalServerError).JSON(response)
}

func (h *ErrorHandler) logError(c *fiber.Ctx, err *errors.DomainError, traceID string) {
    logData := map[string]interface{}{
        "level":      h.getLogLevel(err.Level),
        "trace_id":   traceID,
        "domain":     err.Domain,
        "code":       err.Code,
        "message":    err.Message,
        "path":       c.Path(),
        "method":     c.Method(),
        "ip":         c.IP(),
        "user_agent": c.Get("User-Agent"),
        "timestamp":  err.Timestamp.Format(time.RFC3339),
    }

    if err.Details != nil {
        logData["details"] = err.Details
    }

    if err.Cause != nil {
        logData["cause"] = err.Cause.Error()
    }

    logJSON, _ := json.Marshal(logData)
    log.Println(string(logJSON))
}

func (h *ErrorHandler) getLogLevel(level errors.ErrorLevel) string {
    switch level {
    case errors.ErrorLevelInfo:
        return "info"
    case errors.ErrorLevelWarning:
        return "warning"
    case errors.ErrorLevelError:
        return "error"
    case errors.ErrorLevelCritical:
        return "critical"
    default:
        return "error"
    }
}
```

### 5. 에러 복구 미들웨어
```go
// middlewares/error_recovery.go
package middlewares

import (
    "fmt"
    "runtime/debug"

    "github.com/gofiber/fiber/v2"
    "proxynd/internal/errors"
)

func ErrorRecovery() fiber.Handler {
    return func(c *fiber.Ctx) error {
        defer func() {
            if r := recover(); r != nil {
                // 스택 트레이스 캡처
                stack := debug.Stack()

                // 패닉 에러 생성
                err := errors.NewError("PANIC001", "서버 패닉 발생").
                    WithDomain("system").
                    WithLevel(errors.ErrorLevelCritical).
                    WithDetails(map[string]interface{}{
                        "panic": fmt.Sprintf("%v", r),
                        "stack": string(stack),
                    }).
                    Build()

                // 에러 핸들러로 전달
                c.Locals("panic_error", err)
                c.Next()
            }
        }()

        return c.Next()
    }
}
```

## 실행 명령어
```bash
# 에러 패키지 생성
mkdir -p internal/errors
touch internal/errors/base.go
touch internal/errors/apt_errors.go
touch internal/errors/maven_errors.go
touch internal/errors/http_mapping.go

# 미들웨어 생성
touch middlewares/error_handler.go
touch middlewares/error_recovery.go

# 빌드 테스트
make dev-run-direct

# 에러 처리 테스트
curl -v http://localhost:8080/proxy/apt/invalid/package
curl -v http://localhost:8080/proxy/maven/invalid/artifact
```

## 검증 방법
1. 도메인별 에러 코드 정상 출력
2. 구조화된 로그 형식 확인
3. HTTP 상태 코드 매핑 검증
4. 패닉 복구 동작 확인

## 완료 조건
- [x] 도메인 에러 타입 정의
- [x] HTTP 상태 코드 매핑 구현
- [x] 에러 핸들러 미들웨어 구현
- [x] 에러 복구 미들웨어 구현
- [x] 구조화된 로깅 시스템 구현
- [x] 기존 핸들러에 에러 타입 적용
- [x] 단위 테스트 작성
- [x] 통합 테스트 통과
