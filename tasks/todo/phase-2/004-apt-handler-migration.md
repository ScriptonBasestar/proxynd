---
phase: 2
order: 4
source_plan: /docs/refactoring/01-dependency-injection.md
priority: high
tags: [apt, handler, migration, di]
---

# 📌 작업: APT 핸들러 의존성 주입 마이그레이션

## 개요
기존 APT 핸들러를 새로운 의존성 주입 패턴으로 마이그레이션하여 ReadConfig() 반복 호출을 제거합니다.

## 현재 상태
```go
// 기존 APT 핸들러 (개선 전)
func HandleAPTRequest(c *fiber.Ctx) error {
    config, err := configs.ReadConfig()  // 매번 파일 읽기
    if err != nil {
        return err
    }
    // ... 핸들러 로직
}
```

## 구현 내용

### 1. 새로운 APT 핸들러 구조체
```go
// handlers/proxy/apt_handler.go
package proxy

import (
    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
    "proxynd/internal/handlers"
    "proxynd/cache"
)

type APTHandler struct {
    *handlers.BaseHandler
    cache     cache.Cache
    client    *http.Client
}

func NewAPTHandler(container *app.Container) *APTHandler {
    return &APTHandler{
        BaseHandler: handlers.NewBaseHandler(container, "apt-proxy", "apt"),
        cache:       container.Get("cache").(cache.Cache),
        client:      container.Get("httpClient").(*http.Client),
    }
}
```

### 2. Handle 메서드 구현
```go
func (h *APTHandler) Handle(c *fiber.Ctx) error {
    // 캐시된 설정 사용 (파일 I/O 없음)
    config := h.GetConfig().APTProxy
    if !config.Enabled {
        return fiber.ErrNotFound
    }

    // 요청 경로 파싱
    packagePath := c.Params("*")

    // 캐시 확인
    cacheKey := h.generateCacheKey(packagePath)
    if cached, err := h.cache.Get(cacheKey); err == nil {
        c.Set("X-Cache-Status", "HIT")
        return c.Send(cached)
    }

    // 미러 선택
    mirror := h.selectMirror(config.Mirrors)

    // 업스트림 요청
    upstreamURL := fmt.Sprintf("%s/%s", strings.TrimRight(mirror, "/"), packagePath)
    resp, err := h.fetchFromUpstream(upstreamURL)
    if err != nil {
        return h.handleUpstreamError(err)
    }

    // 캐시 저장
    if h.shouldCache(packagePath) {
        h.cache.SetWithTTL(cacheKey, resp, h.getCacheTTL(packagePath))
    }

    c.Set("X-Cache-Status", "MISS")
    return c.Send(resp)
}
```

### 3. 헬퍼 메서드 구현
```go
func (h *APTHandler) generateCacheKey(path string) string {
    return fmt.Sprintf("apt:%s", strings.ReplaceAll(path, "/", "_"))
}

func (h *APTHandler) selectMirror(mirrors []string) string {
    // 라운드로빈 또는 지연시간 기반 선택
    if len(mirrors) == 0 {
        return ""
    }
    return mirrors[0] // 임시로 첫 번째 미러 사용
}

func (h *APTHandler) shouldCache(path string) bool {
    // Release 파일은 단기 캐시
    if strings.Contains(path, "Release") {
        return true
    }
    // .deb 파일은 장기 캐시
    if strings.HasSuffix(path, ".deb") {
        return true
    }
    return false
}

func (h *APTHandler) getCacheTTL(path string) time.Duration {
    if strings.Contains(path, "Release") {
        return 10 * time.Minute
    }
    if strings.HasSuffix(path, ".deb") {
        return 7 * 24 * time.Hour
    }
    return 1 * time.Hour
}
```

### 4. 핸들러 등록
```go
// internal/app/bootstrap.go
func (c *Container) RegisterHandlers() {
    registry := handlers.NewHandlerRegistry(c)

    // APT 핸들러 등록
    registry.Register("apt", func(container *app.Container) handlers.Handler {
        return NewAPTHandler(container)
    })

    c.Register("handlerRegistry", func() (interface{}, error) {
        return registry, nil
    })
}
```

### 5. 라우터 업데이트
```go
// routers/proxy_router.go
func SetupProxyRoutes(app *fiber.App, container *app.Container) {
    registry := container.Get("handlerRegistry").(*handlers.HandlerRegistry)

    app.All("/proxy/apt/*", func(c *fiber.Ctx) error {
        handler, err := registry.Get("apt")
        if err != nil {
            return c.Status(404).JSON(fiber.Map{
                "error": "APT_PROXY_NOT_FOUND",
                "message": err.Error(),
            })
        }
        return handler.Handle(c)
    })
}
```

## 실행 명령어
```bash
# 기존 핸들러 백업
cp handlers/proxy/apt_handler.go handlers/proxy/apt_handler.go.bak

# 새로운 핸들러 구현
# (위 코드를 파일에 작성)

# 빌드 테스트
make dev-run-direct

# APT 프록시 테스트
curl -v http://localhost:8080/proxy/apt/dists/focal/Release

# 캐시 동작 확인
curl -H "X-Cache-Status: " http://localhost:8080/proxy/apt/dists/focal/Release
```

## 검증 방법
1. 설정 파일 I/O 없이 동작 확인
2. 캐시 HIT/MISS 헤더 확인
3. 메모리 사용량 모니터링
4. 응답 시간 측정

## 완료 조건
- [ ] APTHandler 구조체 구현
- [ ] 의존성 주입 패턴 적용
- [ ] 캐시 로직 구현
- [ ] 미러 선택 로직 구현
- [ ] 핸들러 등록 및 라우터 연결
- [ ] 기존 기능 호환성 유지
- [ ] 단위 테스트 작성
- [ ] 통합 테스트 통과
