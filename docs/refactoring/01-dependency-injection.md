# 의존성 주입 개선 상세 계획

## 1. 현재 문제점 분석

### ReadConfig() 반복 호출 위치
```bash
# 현재 ReadConfig() 호출 위치 (30+ 곳)
handlers/proxy/apt_handler.go:25
handlers/proxy/maven_handler.go:18
handlers/proxy/npm_handler.go:22
middlewares/proxy_policy.go:15
internal/services/proxy_service.go:45
# ... 등등
```

### 문제점
1. **성능**: 매 요청마다 파일 I/O 발생
2. **일관성**: 설정 변경 시 즉시 반영 안됨
3. **테스트**: 설정 모킹 어려움
4. **메모리**: 중복 객체 생성

## 2. Container 패턴 확장 설계

### 2.1 기존 Container 분석
```go
// internal/app/container.go (현재)
type Container struct {
    mu            sync.RWMutex
    services      map[string]interface{}
    constructors  map[string]func() (interface{}, error)
}
```

### 2.2 확장된 Container 설계
```go
// internal/app/container.go (개선)
type Container struct {
    mu            sync.RWMutex
    services      map[string]interface{}
    constructors  map[string]func() (interface{}, error)

    // 설정 관련 추가
    config        *configs.Config
    configMu      sync.RWMutex
    configWatcher *fsnotify.Watcher

    // 서비스 레지스트리
    handlers      map[string]fiber.Handler
    middlewares   []fiber.Handler
}

// 설정 관련 메서드
func (c *Container) Config() *configs.Config {
    c.configMu.RLock()
    defer c.configMu.RUnlock()
    return c.config
}

func (c *Container) ReloadConfig() error {
    c.configMu.Lock()
    defer c.configMu.Unlock()

    newConfig, err := configs.ReadConfig()
    if err != nil {
        return err
    }

    // 검증
    if err := newConfig.Validate(); err != nil {
        return err
    }

    c.config = newConfig
    c.notifyConfigChange()
    return nil
}

// 설정 변경 감시
func (c *Container) WatchConfig() {
    go func() {
        for {
            select {
            case event := <-c.configWatcher.Events:
                if event.Op&fsnotify.Write == fsnotify.Write {
                    log.Println("설정 파일 변경 감지")
                    if err := c.ReloadConfig(); err != nil {
                        log.Printf("설정 리로드 실패: %v", err)
                    }
                }
            case err := <-c.configWatcher.Errors:
                log.Printf("설정 감시 에러: %v", err)
            }
        }
    }()
}
```

## 3. 핸들러 팩토리 패턴

### 3.1 핸들러 인터페이스
```go
// internal/handlers/interfaces.go
type Handler interface {
    Handle(c *fiber.Ctx) error
    Name() string
}

type HandlerFactory interface {
    Create(container *app.Container) Handler
}
```

### 3.2 APT 핸들러 리팩토링
```go
// handlers/proxy/apt_handler.go (개선 전)
func HandleAPTRequest(c *fiber.Ctx) error {
    config, err := configs.ReadConfig()
    if err != nil {
        return err
    }
    // ... 핸들러 로직
}

// handlers/proxy/apt_handler.go (개선 후)
type APTHandler struct {
    container *app.Container
    cache     cache.Cache
    client    *http.Client
}

func NewAPTHandler(container *app.Container) *APTHandler {
    return &APTHandler{
        container: container,
        cache:     container.Get("cache").(cache.Cache),
        client:    container.Get("httpClient").(*http.Client),
    }
}

func (h *APTHandler) Handle(c *fiber.Ctx) error {
    config := h.container.Config().APTProxy
    if !config.Enabled {
        return fiber.ErrNotFound
    }

    // 캐시된 설정 사용
    mirror := h.selectMirror(config.Mirrors)

    // ... 핸들러 로직
}

func (h *APTHandler) Name() string {
    return "apt-proxy"
}
```

## 4. 라우터 통합

### 4.1 핸들러 등록
```go
// internal/app/bootstrap.go
func (c *Container) RegisterHandlers() {
    // APT 핸들러 등록
    c.Register("handler.apt", func() (interface{}, error) {
        return NewAPTHandler(c), nil
    })

    // Maven 핸들러 등록
    c.Register("handler.maven", func() (interface{}, error) {
        return NewMavenHandler(c), nil
    })

    // NPM 핸들러 등록
    c.Register("handler.npm", func() (interface{}, error) {
        return NewNPMHandler(c), nil
    })
}
```

### 4.2 라우터 설정
```go
// routers/proxy_router.go (개선)
func SetupProxyRoutes(app *fiber.App, container *app.Container) {
    proxyGroup := app.Group("/proxy")

    // 동적 핸들러 바인딩
    handlers := map[string]string{
        "apt":   "handler.apt",
        "maven": "handler.maven",
        "npm":   "handler.npm",
    }

    for proxyType, handlerKey := range handlers {
        handler := container.Get(handlerKey).(Handler)
        proxyGroup.All(fmt.Sprintf("/%s/*", proxyType), handler.Handle)
    }
}
```

## 5. 테스트 개선

### 5.1 Mock Container
```go
// test/mocks/container_mock.go
type MockContainer struct {
    mock.Mock
    config *configs.Config
}

func (m *MockContainer) Config() *configs.Config {
    return m.config
}

func (m *MockContainer) Get(key string) interface{} {
    args := m.Called(key)
    return args.Get(0)
}
```

### 5.2 핸들러 테스트
```go
// handlers/proxy/apt_handler_test.go
func TestAPTHandler_Handle(t *testing.T) {
    // Given
    mockConfig := &configs.Config{
        APTProxy: configs.APTProxyConfig{
            Enabled: true,
            Mirrors: []string{"http://test.mirror"},
        },
    }

    container := &MockContainer{config: mockConfig}
    container.On("Get", "cache").Return(cache.NewMemoryCache())
    container.On("Get", "httpClient").Return(http.DefaultClient)

    handler := NewAPTHandler(container)

    // When
    app := fiber.New()
    app.Get("/apt/*", handler.Handle)

    req := httptest.NewRequest("GET", "/apt/dists/focal/Release", nil)
    resp, err := app.Test(req)

    // Then
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

## 6. 마이그레이션 계획

### Phase 1: Container 확장 (3일)
- [ ] Container에 설정 관리 기능 추가
- [ ] 설정 변경 감시 구현
- [ ] 설정 검증 로직 추가

### Phase 2: 핸들러 리팩토링 (1주)
- [ ] Handler 인터페이스 정의
- [ ] APT 핸들러 마이그레이션
- [ ] Maven 핸들러 마이그레이션
- [ ] NPM 핸들러 마이그레이션

### Phase 3: 테스트 및 검증 (3일)
- [ ] 단위 테스트 작성
- [ ] 통합 테스트 실행
- [ ] 성능 벤치마크

### Phase 4: 배포 (2일)
- [ ] Feature flag로 점진적 활성화
- [ ] 모니터링 및 롤백 준비
- [ ] 문서 업데이트

## 7. 예상 효과

### 성능 개선
- 설정 파일 I/O: 요청당 1회 → 0회
- 응답 시간: 평균 10ms 감소
- 메모리 사용: 20% 감소

### 개발 생산성
- 새 핸들러 추가: 1일 → 2시간
- 테스트 작성 시간: 50% 감소
- 디버깅 시간: 30% 감소

### 코드 품질
- 결합도 감소
- 테스트 가능성 향상
- 확장성 개선
