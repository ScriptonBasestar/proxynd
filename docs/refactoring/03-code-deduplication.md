# 코드 중복 제거 - BaseProxyHandler 설계

## 1. 현재 코드 중복 분석

### 1.1 중복 패턴 식별
각 프록시 핸들러(APT, Maven, NPM)에서 반복되는 패턴:

```go
// 공통 패턴 1: 설정 확인
config, err := configs.ReadConfig()
if !config.XProxy.Enabled {
    return fiber.ErrNotFound
}

// 공통 패턴 2: 캐시 조회
cacheKey := generateCacheKey(path)
if cached, err := cache.Get(cacheKey); err == nil {
    return c.Send(cached)
}

// 공통 패턴 3: 업스트림 요청
resp, err := fetchFromUpstream(mirrorURL + path)
if err != nil {
    return handleUpstreamError(err)
}

// 공통 패턴 4: 캐시 저장
cache.Set(cacheKey, resp.Body)

// 공통 패턴 5: 응답 전송
return c.Send(resp.Body)
```

### 1.2 중복 통계
- 캐시 로직: 3개 핸들러 × 20줄 = 60줄
- 프록시 로직: 3개 핸들러 × 30줄 = 90줄
- 에러 처리: 3개 핸들러 × 15줄 = 45줄
- **총 중복 코드: 약 195줄**

## 2. BaseProxyHandler 인터페이스 설계

### 2.1 핵심 인터페이스
```go
// internal/handlers/base_proxy.go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "proxynd/cache"
    "proxynd/configs"
)

// ProxyHandler 기본 인터페이스
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
    GetCacheTTL(c *fiber.Ctx) int
}
```

### 2.2 공통 기능 구현체
```go
// internal/handlers/base_proxy_impl.go
package handlers

import (
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
    "proxynd/cache"
    "proxynd/internal/errors"
)

// BaseProxyHandlerImpl 공통 구현
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

// Handle 공통 처리 로직
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

    // 5. 요청 변환
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
        return err
    }

    // 8. 응답 변환
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
func (b *BaseProxyHandlerImpl) saveToCache(key string, data []byte, ttl int) error {
    return b.cache.SetWithTTL(key, data, time.Duration(ttl)*time.Second)
}

// 업스트림 요청 생성
func (b *BaseProxyHandlerImpl) createUpstreamRequest(c *fiber.Ctx, url string) (*http.Request, error) {
    req, err := http.NewRequest(c.Method(), url, bytes.NewReader(c.Body()))
    if err != nil {
        return nil, err
    }

    // 기본 헤더 복사
    c.Request().Header.VisitAll(func(key, value []byte) {
        // 홉바이홉 헤더 제외
        k := string(key)
        if !isHopByHopHeader(k) {
            req.Header.Set(k, string(value))
        }
    })

    return req, nil
}

// 응답 헤더 전달
func (b *BaseProxyHandlerImpl) forwardHeaders(c *fiber.Ctx, resp *http.Response) {
    for key, values := range resp.Header {
        if !isHopByHopHeader(key) {
            c.Set(key, values[0])
        }
    }
}
```

## 3. 도메인별 구현

### 3.1 APT 프록시 구현
```go
// handlers/proxy/apt_handler_v2.go
package proxy

import (
    "fmt"
    "strings"

    "github.com/gofiber/fiber/v2"
    "proxynd/configs"
    "proxynd/internal/handlers"
)

type APTHandlerV2 struct {
    mirrorSelector *MirrorSelector
}

func NewAPTHandlerV2() *APTHandlerV2 {
    return &APTHandlerV2{
        mirrorSelector: NewMirrorSelector(),
    }
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
    dist := c.Query("dist", "")
    arch := c.Query("arch", "")

    // APT 특화 캐시 키
    return fmt.Sprintf("apt:%s:%s:%s:%s",
        strings.ReplaceAll(path, "/", "_"),
        dist,
        arch,
        c.Get("Accept-Encoding"),
    )
}

func (h *APTHandlerV2) BuildUpstreamURL(c *fiber.Ctx, config interface{}) (string, error) {
    aptConfig := config.(configs.APTProxyConfig)

    // 미러 선택 (라운드로빈, 지연시간 기반 등)
    mirror := h.mirrorSelector.Select(aptConfig.Mirrors)
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
    contentType := c.Get("Content-Type")

    // Release 파일인 경우 서명 검증
    if strings.Contains(c.Path(), "Release") {
        if err := h.verifyReleaseSignature(resp); err != nil {
            return nil, err
        }
    }

    // Packages 파일인 경우 압축 처리
    if strings.Contains(contentType, "text/plain") &&
       strings.Contains(c.Path(), "Packages") {
        return h.compressIfNeeded(resp, c)
    }

    return resp, nil
}

func (h *APTHandlerV2) ShouldCache(c *fiber.Ctx, statusCode int) bool {
    // 성공 응답만 캐시
    if statusCode != 200 && statusCode != 304 {
        return false
    }

    // 메타데이터는 짧게, 패키지는 길게
    path := c.Path()
    if strings.Contains(path, "Release") ||
       strings.Contains(path, "Packages") {
        return true
    }

    if strings.HasSuffix(path, ".deb") {
        return true
    }

    return false
}

func (h *APTHandlerV2) GetCacheTTL(c *fiber.Ctx) int {
    path := c.Path()

    // 메타데이터는 10분
    if strings.Contains(path, "Release") ||
       strings.Contains(path, "Packages") {
        return 600
    }

    // 패키지 파일은 7일
    if strings.HasSuffix(path, ".deb") {
        return 604800
    }

    // 기본 1시간
    return 3600
}
```

### 3.2 Maven 프록시 구현
```go
// handlers/proxy/maven_handler_v2.go
package proxy

import (
    "fmt"
    "path"
    "strings"

    "github.com/gofiber/fiber/v2"
    "proxynd/configs"
)

type MavenHandlerV2 struct {
    checksumVerifier *ChecksumVerifier
}

func NewMavenHandlerV2() *MavenHandlerV2 {
    return &MavenHandlerV2{
        checksumVerifier: NewChecksumVerifier(),
    }
}

func (h *MavenHandlerV2) Type() string {
    return "maven"
}

func (h *MavenHandlerV2) GenerateCacheKey(c *fiber.Ctx) string {
    // Maven 좌표 기반 캐시 키
    path := c.Params("*")
    parts := strings.Split(path, "/")

    if len(parts) >= 3 {
        // groupId/artifactId/version 형식
        return fmt.Sprintf("maven:%s:%s:%s:%s",
            parts[0],
            parts[1],
            parts[2],
            path.Base(path),
        )
    }

    return fmt.Sprintf("maven:%s", strings.ReplaceAll(path, "/", ":"))
}

func (h *MavenHandlerV2) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
    path := c.Path()

    // 체크섬 파일 검증
    if strings.HasSuffix(path, ".sha1") ||
       strings.HasSuffix(path, ".md5") {
        return h.validateChecksum(resp, path)
    }

    // POM 파일 처리
    if strings.HasSuffix(path, ".pom") {
        return h.processPOM(resp)
    }

    return resp, nil
}

func (h *MavenHandlerV2) ShouldCache(c *fiber.Ctx, statusCode int) bool {
    if statusCode != 200 {
        return false
    }

    path := c.Path()

    // SNAPSHOT 버전은 캐시하지 않음
    if strings.Contains(path, "-SNAPSHOT") {
        return false
    }

    // 메타데이터는 단기 캐시
    if strings.HasSuffix(path, "maven-metadata.xml") {
        return true
    }

    return true
}

func (h *MavenHandlerV2) GetCacheTTL(c *fiber.Ctx) int {
    path := c.Path()

    // 메타데이터는 30분
    if strings.HasSuffix(path, "maven-metadata.xml") {
        return 1800
    }

    // SNAPSHOT은 캐시 안함
    if strings.Contains(path, "-SNAPSHOT") {
        return 0
    }

    // 릴리즈 아티팩트는 30일
    return 2592000
}
```

## 4. 팩토리 패턴으로 통합

### 4.1 프록시 팩토리
```go
// internal/handlers/proxy_factory.go
package handlers

import (
    "fmt"

    "proxynd/internal/app"
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
        return NewAPTHandlerV2()
    })

    factory.Register("maven", func() ProxyHandler {
        return NewMavenHandlerV2()
    })

    factory.Register("npm", func() ProxyHandler {
        return NewNPMHandlerV2()
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

### 4.2 라우터 통합
```go
// routers/proxy_router_v2.go
package routers

import (
    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
    "proxynd/internal/handlers"
)

func SetupProxyRoutesV2(app *fiber.App, container *app.Container) {
    factory := handlers.NewProxyHandlerFactory(container)

    // 통합 프록시 엔드포인트
    app.All("/proxy/:type/*", func(c *fiber.Ctx) error {
        proxyType := c.Params("type")

        handler, err := factory.Create(proxyType)
        if err != nil {
            return c.Status(404).JSON(fiber.Map{
                "error": "PROXY_NOT_FOUND",
                "message": err.Error(),
            })
        }

        return handler.Handle(c)
    })
}
```

## 5. 마이그레이션 계획

### Phase 1: 인터페이스 정의 (2일)
- [ ] ProxyHandler 인터페이스 정의
- [ ] BaseProxyHandlerImpl 구현
- [ ] 팩토리 패턴 구현

### Phase 2: 도메인 구현 (1주)
- [ ] APTHandlerV2 구현 및 테스트
- [ ] MavenHandlerV2 구현 및 테스트
- [ ] NPMHandlerV2 구현 및 테스트

### Phase 3: 통합 및 전환 (3일)
- [ ] 라우터 통합
- [ ] Feature flag로 점진적 전환
- [ ] 성능 비교 테스트

### Phase 4: 정리 (2일)
- [ ] 기존 핸들러 제거
- [ ] 문서 업데이트
- [ ] 최종 검증

## 6. 예상 효과

### 코드 감소
- 중복 코드: 195줄 → 0줄
- 전체 코드량: 30% 감소
- 복잡도: 40% 감소

### 유지보수성
- 새 프록시 추가: 1일 → 2시간
- 버그 수정: 3곳 → 1곳
- 기능 추가: 모든 프록시에 즉시 적용

### 성능
- 메모리 사용: 15% 감소 (코드 중복 제거)
- 응답 시간: 동일 (추상화 오버헤드 최소)
- 캐시 효율: 20% 향상 (통합 관리)
