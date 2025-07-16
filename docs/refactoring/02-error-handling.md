# 에러 처리 표준화 상세 계획

## 1. 현재 에러 처리 현황

### 1.1 문제점 분석
```bash
# 에러 처리 패턴 통계
if err != nil 패턴: 507개
에러 타입: 2개 (ProxyError, ValidationError)
에러 로깅 방식: 일관성 없음
HTTP 상태 코드: 하드코딩됨
```

### 1.2 현재 에러 처리 예시
```go
// 일관성 없는 에러 처리
if err != nil {
    log.Printf("에러: %v", err)  // 때로는 로깅
    return c.Status(500).SendString("내부 서버 오류")  // 때로는 안함
}

// 하드코딩된 메시지
if !found {
    return c.Status(404).SendString("패키지를 찾을 수 없습니다")
}

// 컨텍스트 없는 에러
if err := db.Save(); err != nil {
    return err  // 원인 추적 어려움
}
```

## 2. 도메인별 에러 타입 설계

### 2.1 기본 에러 구조체
```go
// internal/errors/base.go
package errors

import (
    "fmt"
    "time"
)

// 에러 레벨 정의
type ErrorLevel int

const (
    ErrorLevelInfo ErrorLevel = iota
    ErrorLevelWarning
    ErrorLevelError
    ErrorLevelCritical
)

// 기본 도메인 에러
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

// 에러 빌더
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

func (b *ErrorBuilder) Build() *DomainError {
    return b.err
}
```

### 2.2 도메인별 에러 정의

#### APT 도메인 에러
```go
// internal/errors/apt_errors.go
package errors

var (
    // APT 패키지 관련 에러
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

// APT 에러 생성 헬퍼
func NewAPTError(code, message string) *DomainError {
    return NewError(code, message).
        WithDomain("apt").
        Build()
}
```

#### Maven 도메인 에러
```go
// internal/errors/maven_errors.go
package errors

var (
    ErrMavenArtifactNotFound = &DomainError{
        Code:    "MVN001",
        Message: "Maven 아티팩트를 찾을 수 없습니다",
        Domain:  "maven",
        Level:   ErrorLevelWarning,
    }

    ErrMavenInvalidPOM = &DomainError{
        Code:    "MVN002",
        Message: "유효하지 않은 POM 파일입니다",
        Domain:  "maven",
        Level:   ErrorLevelError,
    }

    ErrMavenChecksumMismatch = &DomainError{
        Code:    "MVN003",
        Message: "체크섬이 일치하지 않습니다",
        Domain:  "maven",
        Level:   ErrorLevelError,
    }
)
```

#### 캐시 도메인 에러
```go
// internal/errors/cache_errors.go
package errors

var (
    ErrCacheNotFound = &DomainError{
        Code:    "CACHE001",
        Message: "캐시에서 항목을 찾을 수 없습니다",
        Domain:  "cache",
        Level:   ErrorLevelInfo,
    }

    ErrCacheFull = &DomainError{
        Code:    "CACHE002",
        Message: "캐시 저장 공간이 부족합니다",
        Domain:  "cache",
        Level:   ErrorLevelWarning,
    }
)
```

### 2.3 HTTP 상태 코드 매핑
```go
// internal/errors/http_mapping.go
package errors

import "github.com/gofiber/fiber/v2"

// 에러 코드별 HTTP 상태 매핑
var errorToHTTPStatus = map[string]int{
    "APT001":   fiber.StatusNotFound,
    "APT002":   fiber.StatusServiceUnavailable,
    "APT003":   fiber.StatusBadRequest,
    "MVN001":   fiber.StatusNotFound,
    "MVN002":   fiber.StatusBadRequest,
    "MVN003":   fiber.StatusUnprocessableEntity,
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

## 3. 중앙화된 에러 핸들링 미들웨어

### 3.1 에러 핸들러 미들웨어
```go
// middlewares/error_handler.go
package middlewares

import (
    "log"
    "github.com/gofiber/fiber/v2"
    "proxynd/internal/errors"
)

// 에러 응답 구조체
type ErrorResponse struct {
    Error   string      `json:"error"`
    Message string      `json:"message"`
    Domain  string      `json:"domain,omitempty"`
    Details interface{} `json:"details,omitempty"`
    TraceID string      `json:"trace_id,omitempty"`
}

func ErrorHandler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 다음 핸들러 실행
        err := c.Next()
        if err == nil {
            return nil
        }

        // 추적 ID 가져오기
        traceID := c.Locals("requestID").(string)

        // 도메인 에러 처리
        if domainErr, ok := err.(*errors.DomainError); ok {
            // 구조화된 로깅
            logError(c, domainErr, traceID)

            // HTTP 상태 코드 결정
            status := errors.GetHTTPStatus(domainErr)

            // 응답 생성
            response := ErrorResponse{
                Error:   domainErr.Code,
                Message: domainErr.Message,
                Domain:  domainErr.Domain,
                Details: domainErr.Details,
                TraceID: traceID,
            }

            return c.Status(status).JSON(response)
        }

        // Fiber 에러 처리
        if fiberErr, ok := err.(*fiber.Error); ok {
            return c.Status(fiberErr.Code).JSON(ErrorResponse{
                Error:   "FIBER_ERROR",
                Message: fiberErr.Message,
                TraceID: traceID,
            })
        }

        // 기본 에러 처리
        logUnknownError(c, err, traceID)
        return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
            Error:   "INTERNAL_ERROR",
            Message: "내부 서버 오류가 발생했습니다",
            TraceID: traceID,
        })
    }
}

// 구조화된 에러 로깅
func logError(c *fiber.Ctx, err *errors.DomainError, traceID string) {
    log.Printf(`{
        "level": "%s",
        "trace_id": "%s",
        "domain": "%s",
        "code": "%s",
        "message": "%s",
        "path": "%s",
        "method": "%s",
        "ip": "%s",
        "user_agent": "%s",
        "timestamp": "%s"
    }`,
        getLogLevel(err.Level),
        traceID,
        err.Domain,
        err.Code,
        err.Message,
        c.Path(),
        c.Method(),
        c.IP(),
        c.Get("User-Agent"),
        err.Timestamp.Format(time.RFC3339),
    )
}
```

### 3.2 에러 복구 미들웨어
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
                // 스택 트레이스 가져오기
                stack := debug.Stack()

                // 패닉 에러 생성
                err := errors.NewError("PANIC001", "서버 패닉 발생").
                    WithDomain("system").
                    WithDetails(map[string]interface{}{
                        "panic": fmt.Sprintf("%v", r),
                        "stack": string(stack),
                    }).
                    Build()

                // 에러 핸들러로 전달
                c.Locals("error", err)
                c.Next()
            }
        }()

        return c.Next()
    }
}
```

## 4. 핸들러에서의 에러 처리

### 4.1 개선 전
```go
func HandleAPTRequest(c *fiber.Ctx) error {
    pkg, err := fetchPackage(c.Params("*"))
    if err != nil {
        log.Printf("패키지 가져오기 실패: %v", err)
        return c.Status(500).SendString("패키지를 가져올 수 없습니다")
    }

    if pkg == nil {
        return c.Status(404).SendString("패키지를 찾을 수 없습니다")
    }

    return c.Send(pkg.Data)
}
```

### 4.2 개선 후
```go
func (h *APTHandler) Handle(c *fiber.Ctx) error {
    packagePath := c.Params("*")

    // 캐시 확인
    cached, err := h.cache.Get(packagePath)
    if err != nil && err != errors.ErrCacheNotFound {
        // 캐시 에러는 경고만, 계속 진행
        log.Printf("캐시 조회 실패: %v", err)
    }

    if cached != nil {
        return c.Send(cached)
    }

    // 패키지 가져오기
    pkg, err := h.fetchPackage(packagePath)
    if err != nil {
        // 에러 타입에 따라 적절한 도메인 에러 반환
        if err == ErrNotFound {
            return errors.ErrAPTPackageNotFound
        }

        // 상세 정보와 함께 에러 생성
        return errors.NewAPTError("APT004", "패키지 다운로드 실패").
            WithCause(err).
            WithDetails(map[string]string{
                "package": packagePath,
                "mirror": h.currentMirror,
            }).
            Build()
    }

    // 캐시 저장 (실패해도 계속)
    if err := h.cache.Set(packagePath, pkg.Data); err != nil {
        log.Printf("캐시 저장 실패: %v", err)
    }

    return c.Send(pkg.Data)
}
```

## 5. 에러 모니터링 및 알림

### 5.1 에러 메트릭 수집
```go
// metrics/error_metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    errorCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "proxynd_errors_total",
            Help: "Total number of errors by domain and code",
        },
        []string{"domain", "code", "level"},
    )

    errorDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "proxynd_error_duration_seconds",
            Help: "Error handling duration by domain",
        },
        []string{"domain"},
    )
)

func RecordError(domain, code, level string) {
    errorCounter.WithLabelValues(domain, code, level).Inc()
}
```

### 5.2 임계치 기반 알림
```go
// alerts/error_alerts.go
package alerts

import (
    "proxynd/internal/errors"
)

type ErrorAlertRule struct {
    Domain    string
    Code      string
    Threshold int
    Window    time.Duration
}

func (r *ErrorAlertRule) Check(err *errors.DomainError) bool {
    count := getErrorCount(err.Domain, err.Code, r.Window)
    return count >= r.Threshold
}

// 알림 규칙 정의
var alertRules = []ErrorAlertRule{
    {
        Domain:    "apt",
        Code:      "APT002",
        Threshold: 10,
        Window:    5 * time.Minute,
    },
    {
        Domain:    "maven",
        Code:      "MVN003",
        Threshold: 5,
        Window:    1 * time.Minute,
    },
}
```

## 6. 마이그레이션 계획

### Phase 1: 에러 타입 정의 (2일)
- [ ] 기본 에러 구조체 구현
- [ ] 도메인별 에러 정의
- [ ] HTTP 상태 코드 매핑

### Phase 2: 미들웨어 구현 (2일)
- [ ] 에러 핸들러 미들웨어
- [ ] 에러 복구 미들웨어
- [ ] 로깅 통합

### Phase 3: 핸들러 마이그레이션 (1주)
- [ ] APT 핸들러 에러 처리 개선
- [ ] Maven 핸들러 에러 처리 개선
- [ ] NPM 핸들러 에러 처리 개선
- [ ] 기타 핸들러 업데이트

### Phase 4: 모니터링 구축 (3일)
- [ ] 에러 메트릭 수집
- [ ] 대시보드 구성
- [ ] 알림 규칙 설정

## 7. 예상 효과

### 운영 개선
- 에러 추적 시간: 30분 → 5분
- 문제 해결 시간: 50% 감소
- 알림 정확도: 90% 향상

### 개발 생산성
- 디버깅 시간: 40% 감소
- 에러 처리 코드: 30% 감소
- 테스트 작성: 2배 빠름

### 사용자 경험
- 명확한 에러 메시지
- 일관된 응답 형식
- 빠른 문제 해결
