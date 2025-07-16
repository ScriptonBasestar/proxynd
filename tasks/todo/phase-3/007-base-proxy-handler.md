---
phase: 3
order: 7
source_plan: /docs/refactoring/03-code-deduplication.md
priority: high
tags: [base-handler, code-deduplication, template-method]
---

# 📌 작업: BaseProxyHandler 구현 및 공통 로직 추출

## 개요
APT, Maven, NPM 핸들러에서 반복되는 195줄의 중복 코드를 BaseProxyHandler로 추출하여 Template Method 패턴을 적용합니다.

## 현재 중복 코드 분석
- 캐시 로직: 3개 핸들러 × 20줄 = 60줄
- 프록시 로직: 3개 핸들러 × 30줄 = 90줄
- 에러 처리: 3개 핸들러 × 15줄 = 45줄
- **총 중복 코드: 195줄**

## 구현 내용

### 1. ProxyHandler 인터페이스 정의
```go
// internal/handlers/base_proxy.go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "proxynd/configs"
)

type ProxyHandler interface {
    // 프록시 타입 식별
    Type() string

    // 설정 검증
    IsEnabled(config *configs.Config) bool
    GetConfig(config *configs.Config) interface{}

    // 캐시 키 생성
    GenerateCacheKey(c *fiber.Ctx) string

    // 업스트림 URL 구성
    BuildUpstreamURL(c *fiber.Ctx, config interface{}) (string, error)

    // 요청 변환 (헤더, 인증 등)
    TransformRequest(req *fiber.Request, config interface{}) error

    // 응답 변환 (압축, 포맷 등)
    TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error)

    // 캐시 정책
    ShouldCache(c *fiber.Ctx, statusCode int) bool
    GetCacheTTL(c *fiber.Ctx) time.Duration
}
```

### 2. BaseProxyHandler 구현
```go
// internal/handlers/base_proxy_impl.go
package handlers

import (
    "fmt"
    "io"
    "net/http"
    "time"
    "bytes"

    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
    "proxynd/cache"
    "proxynd/internal/errors"
)

type BaseProxyHandlerImpl struct {
    container *app.Container
    handler   ProxyHandler
    cache     cache.Cache
    client    *http.Client
}

func NewBaseProxyHandler(container *app.Container, handler ProxyHandler) *BaseProxyHandlerImpl {
    return &BaseProxyHandlerImpl{
        container: container,
        handler:   handler,
        cache:     container.Get("cache").(cache.Cache),
        client: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
    }
}

// Handle 공통 처리 로직 (Template Method)
func (b *BaseProxyHandlerImpl) Handle(c *fiber.Ctx) error {
    // 1. 설정 확인
    config := b.container.Config()
    if !b.handler.IsEnabled(config) {
        return errors.NewError("PROXY001", "프록시가 비활성화되어 있습니다").
            WithDomain(b.handler.Type()).
            Build()
    }

    proxyConfig := b.handler.GetConfig(config)

    // 2. 캐시 조회
    cacheKey := b.handler.GenerateCacheKey(c)
    if cached, err := b.getFromCache(cacheKey); err == nil {
        c.Set("X-Cache-Status", "HIT")
        c.Set("X-Proxy-Type", b.handler.Type())
        return c.Send(cached)
    }

    // 3. 업스트림 URL 구성
    upstreamURL, err := b.handler.BuildUpstreamURL(c, proxyConfig)
    if err != nil {
        return errors.NewError("PROXY002", "업스트림 URL 구성 실패").
            WithDomain(b.handler.Type()).
            WithCause(err).
            Build()
    }

    // 4. 업스트림 요청 생성
    req, err := b.createUpstreamRequest(c, upstreamURL)
    if err != nil {
        return err
    }

    // 5. 요청 변환 (프록시별 커스터마이징)
    if err := b.handler.TransformRequest(&req.Request, proxyConfig); err != nil {
        return err
    }

    // 6. 업스트림 요청 실행
    resp, err := b.client.Do(req)
    if err != nil {
        return errors.NewError("PROXY003", "업스트림 서버 연결 실패").
            WithDomain(b.handler.Type()).
            WithCause(err).
            WithDetails(map[string]string{
                "upstream_url": upstreamURL,
            }).
            Build()
    }
    defer resp.Body.Close()

    // 7. 응답 읽기
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return errors.NewError("PROXY004", "응답 읽기 실패").
            WithDomain(b.handler.Type()).
            WithCause(err).
            Build()
    }

    // 8. 응답 변환 (프록시별 커스터마이징)
    transformed, err := b.handler.TransformResponse(body, c)
    if err != nil {
        return err
    }

    // 9. 캐시 저장 (조건부)
    if b.handler.ShouldCache(c, resp.StatusCode) {
        ttl := b.handler.GetCacheTTL(c)
        if err := b.saveToCache(cacheKey, transformed, ttl); err != nil {
            // 캐시 저장 실패는 로그만, 계속 진행
            b.logCacheError("save", err)
        }
    }

    // 10. 헤더 전달
    b.forwardHeaders(c, resp)
    c.Set("X-Cache-Status", "MISS")
    c.Set("X-Proxy-Type", b.handler.Type())

    // 11. 응답 전송
    return c.Status(resp.StatusCode).Send(transformed)
}

// 캐시 조회
func (b *BaseProxyHandlerImpl) getFromCache(key string) ([]byte, error) {
    data, err := b.cache.Get(key)
    if err != nil {
        return nil, err
    }

    // 메트릭 기록
    b.recordCacheHit(key)
    return data, nil
}

// 캐시 저장
func (b *BaseProxyHandlerImpl) saveToCache(key string, data []byte, ttl time.Duration) error {
    return b.cache.SetWithTTL(key, data, ttl)
}

// 업스트림 요청 생성
func (b *BaseProxyHandlerImpl) createUpstreamRequest(c *fiber.Ctx, url string) (*http.Request, error) {
    req, err := http.NewRequest(c.Method(), url, bytes.NewReader(c.Body()))
    if err != nil {
        return nil, err
    }

    // 기본 헤더 복사 (홉바이홉 헤더 제외)
    c.Request().Header.VisitAll(func(key, value []byte) {
        k := string(key)
        if !b.isHopByHopHeader(k) {
            req.Header.Set(k, string(value))
        }
    })

    return req, nil
}

// 응답 헤더 전달
func (b *BaseProxyHandlerImpl) forwardHeaders(c *fiber.Ctx, resp *http.Response) {
    for key, values := range resp.Header {
        if !b.isHopByHopHeader(key) && len(values) > 0 {
            c.Set(key, values[0])
        }
    }
}

// 홉바이홉 헤더 검사
func (b *BaseProxyHandlerImpl) isHopByHopHeader(header string) bool {
    hopByHopHeaders := []string{
        "Connection",
        "Keep-Alive",
        "Proxy-Authenticate",
        "Proxy-Authorization",
        "Te",
        "Trailers",
        "Transfer-Encoding",
        "Upgrade",
    }

    for _, h := range hopByHopHeaders {
        if strings.EqualFold(header, h) {
            return true
        }
    }
    return false
}

// 메트릭 기록
func (b *BaseProxyHandlerImpl) recordCacheHit(key string) {
    // TODO: Prometheus 메트릭 기록
}

// 캐시 에러 로깅
func (b *BaseProxyHandlerImpl) logCacheError(operation string, err error) {
    log.Printf("캐시 %s 실패: %v", operation, err)
}
```

### 3. APT 핸들러 리팩토링
```go
// handlers/proxy/apt_handler_v2.go
package proxy

import (
    "fmt"
    "strings"
    "time"

    "github.com/gofiber/fiber/v2"
    "proxynd/configs"
    "proxynd/internal/handlers"
)

type APTHandlerV2 struct{}

func NewAPTHandlerV2() *APTHandlerV2 {
    return &APTHandlerV2{}
}

// ProxyHandler 인터페이스 구현
func (h *APTHandlerV2) Type() string {
    return "apt"
}

func (h *APTHandlerV2) IsEnabled(config *configs.Config) bool {
    return config.APTProxy.Enabled
}

func (h *APTHandlerV2) GetConfig(config *configs.Config) interface{} {
    return config.APTProxy
}

func (h *APTHandlerV2) GenerateCacheKey(c *fiber.Ctx) string {
    path := c.Params("*")
    return fmt.Sprintf("apt:%s", strings.ReplaceAll(path, "/", "_"))
}

func (h *APTHandlerV2) BuildUpstreamURL(c *fiber.Ctx, config interface{}) (string, error) {
    aptConfig := config.(configs.APTProxyConfig)

    if len(aptConfig.Mirrors) == 0 {
        return "", fmt.Errorf("APT 미러가 설정되지 않음")
    }

    // 첫 번째 미러 사용 (추후 로드밸런싱 구현)
    mirror := aptConfig.Mirrors[0]
    path := c.Params("*")

    return fmt.Sprintf("%s/%s", strings.TrimRight(mirror, "/"), path), nil
}

func (h *APTHandlerV2) TransformRequest(req *fiber.Request, config interface{}) error {
    // APT 특화 헤더 추가
    req.Header.Set("X-APT-Proxy", "ProxyND")

    // 압축 지원
    if !req.Header.Contains("Accept-Encoding") {
        req.Header.Set("Accept-Encoding", "gzip, deflate")
    }

    return nil
}

func (h *APTHandlerV2) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
    // Release 파일 서명 검증 (선택적)
    if strings.Contains(c.Path(), "Release") {
        // TODO: GPG 서명 검증 구현
    }

    return resp, nil
}

func (h *APTHandlerV2) ShouldCache(c *fiber.Ctx, statusCode int) bool {
    if statusCode != 200 && statusCode != 304 {
        return false
    }

    path := c.Path()
    return strings.Contains(path, "Release") ||
           strings.Contains(path, "Packages") ||
           strings.HasSuffix(path, ".deb")
}

func (h *APTHandlerV2) GetCacheTTL(c *fiber.Ctx) time.Duration {
    path := c.Path()

    // 메타데이터는 10분
    if strings.Contains(path, "Release") || strings.Contains(path, "Packages") {
        return 10 * time.Minute
    }

    // 패키지 파일은 7일
    if strings.HasSuffix(path, ".deb") {
        return 7 * 24 * time.Hour
    }

    // 기본 1시간
    return 1 * time.Hour
}
```

### 4. 팩토리 패턴 통합
```go
// internal/handlers/proxy_factory.go
package handlers

import (
    "fmt"
    "proxynd/internal/app"
    "proxynd/handlers/proxy"
)

type ProxyHandlerFactory struct {
    container *app.Container
    handlers  map[string]func() ProxyHandler
}

func NewProxyHandlerFactory(container *app.Container) *ProxyHandlerFactory {
    factory := &ProxyHandlerFactory{
        container: container,
        handlers:  make(map[string]func() ProxyHandler),
    }

    // 핸들러 등록
    factory.Register("apt", func() ProxyHandler {
        return proxy.NewAPTHandlerV2()
    })

    factory.Register("maven", func() ProxyHandler {
        return proxy.NewMavenHandlerV2()
    })

    return factory
}

func (f *ProxyHandlerFactory) Register(proxyType string, creator func() ProxyHandler) {
    f.handlers[proxyType] = creator
}

func (f *ProxyHandlerFactory) Create(proxyType string) (*BaseProxyHandlerImpl, error) {
    creator, exists := f.handlers[proxyType]
    if !exists {
        return nil, fmt.Errorf("지원하지 않는 프록시 타입: %s", proxyType)
    }

    handler := creator()
    return NewBaseProxyHandler(f.container, handler), nil
}
```

## 실행 명령어
```bash
# BaseProxyHandler 구현
touch internal/handlers/base_proxy.go
touch internal/handlers/base_proxy_impl.go
touch internal/handlers/proxy_factory.go

# APT 핸들러 v2 구현
touch handlers/proxy/apt_handler_v2.go

# 빌드 테스트
make dev-run-direct

# 중복 코드 제거 확인
wc -l handlers/proxy/apt_handler.go handlers/proxy/apt_handler_v2.go
```

## 검증 방법
1. 코드 라인 수 30% 감소 확인
2. 새 프록시 타입 추가 시간 측정
3. 캐시 로직 일관성 확인
4. 기존 기능 호환성 유지

## 완료 조건
- [ ] ProxyHandler 인터페이스 정의
- [ ] BaseProxyHandlerImpl 구현
- [ ] APT 핸들러 v2 구현
- [ ] Maven 핸들러 v2 구현
- [ ] 팩토리 패턴 구현
- [ ] 라우터 통합
- [ ] 코드 중복 195줄 제거
- [ ] 단위 테스트 작성
- [ ] 기능 호환성 검증
