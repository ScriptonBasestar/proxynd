# Container 기반 의존성 주입 마이그레이션 가이드

## 개요

ProxyND의 Container 기반 의존성 주입 시스템으로의 마이그레이션 가이드입니다. 이 문서는 기존 핸들러를 Container 패턴으로 리팩토링하는 방법과 새로운 아키텍처의 장점을 설명합니다.

## 마이그레이션 목표

### Before: 기존 패턴
- **53+ ReadConfig() 호출**: 각 요청마다 설정 파일을 반복 읽음
- **인스턴스 중복 생성**: 매번 새로운 핸들러 인스턴스 생성
- **타이트 커플링**: 직접적인 설정 파일 의존성
- **테스트 어려움**: Mock 설정의 복잡성

### After: Container 패턴
- **1회 설정 로딩**: 애플리케이션 시작 시 한 번만 로딩
- **싱글톤 재사용**: 핸들러 인스턴스 재사용으로 메모리 절약
- **느슨한 결합**: ContainerProvider를 통한 의존성 주입
- **테스트 용이성**: MockContainer를 통한 간단한 테스트

## 아키텍처 변경사항

### 1. 의존성 주입 컨테이너

**Container Provider 인터페이스**:
```go
type ContainerProvider interface {
    GetStorageDir() string
    GetConfigDir() string
    GetAptProxyConfig() (*config.AptProxyConfig, error)
    GetMavenProxyConfig() (*config.MavenProxySettings, error)
    GetNpmProxyConfig() (*config.NpmProxySettings, error)
    GetDockerProxyConfig() (*config.DockerProxySettings, error)
    GetPipProxyConfig() (*config.PipProxySettings, error)
    GetYumProxyConfig() (*config.YumProxySettings, error)
    GetApkProxyConfig() (*config.ApkProxySettings, error)
}
```

**구현체**: `/internal/app/container.go`
- 스레드 세이프 싱글톤 패턴
- 설정 캐싱 및 지연 초기화
- 핫 리로드 지원 (fsnotify)

### 2. 핸들러 인터페이스 표준화

**ContainerProxyHandler 인터페이스**:
```go
type ContainerProxyHandler interface {
    ContainerAwareHandler
    BaseProxyHandler
}

type BaseProxyHandler interface {
    Type() string
    IsEnabled() bool
    GenerateCacheKey(c *fiber.Ctx) string
    BuildUpstreamURL(c *fiber.Ctx) (string, error)
    TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error
    TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error)
    ShouldCache(c *fiber.Ctx, statusCode int) bool
    GetCacheTTL(c *fiber.Ctx) time.Duration
    HandleError(err error, c *fiber.Ctx) error
}
```

### 3. 핸들러 팩토리 패턴

**표준 팩토리**:
```go
type ProxyHandlerFactory interface {
    CreateHandler(proxyType string, provider ContainerProvider) (ContainerProxyHandler, error)
    SupportedTypes() []string
    RegisterHandler(proxyType string, createFn func(ContainerProvider) (ContainerProxyHandler, error)) error
}
```

## 핸들러 마이그레이션 단계별 가이드

### 1단계: 기존 핸들러 분석

**Before (기존 패턴)**:
```go
type OldHandler struct {
    configPath string
}

func (h *OldHandler) Handle(c *fiber.Ctx) error {
    config := &SomeProxyConfig{}
    if err := config.ReadConfig(); err != nil {  // 매번 파일 읽기
        return err
    }
    
    // 비즈니스 로직
    return nil
}
```

### 2단계: Container 기반 핸들러 생성

**After (Container 패턴)**:
```go
type NewContainerHandler struct {
    *BaseContainerHandler
    proxyConfig *config.SomeProxySettings
    enabled     bool
}

func NewSomeContainerHandler(provider container.ContainerProvider) *NewContainerHandler {
    handler := &NewContainerHandler{
        BaseContainerHandler: NewBaseContainerHandler(provider, "some", "some-container-handler"),
    }
    
    // 생성 시점에 한 번만 설정 로딩
    if err := handler.LoadConfig(); err != nil {
        handler.enabled = false
        return handler
    }
    
    handler.enabled = true
    return handler
}

func (h *NewContainerHandler) LoadConfig() error {
    config, err := h.containerProvider.GetSomeProxyConfig()
    if err != nil {
        h.RecordConfigCacheMiss("some-config")
        return err
    }
    
    h.RecordConfigCacheHit("some-config")
    h.proxyConfig = config
    return nil
}
```

### 3단계: 인터페이스 메서드 구현

**필수 구현 메서드**:
```go
func (h *NewContainerHandler) IsEnabled() bool {
    return h.enabled && h.proxyConfig != nil
}

func (h *NewContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
    start := time.Now()
    defer h.RecordCacheKeyGeneration(time.Since(start))
    
    return fmt.Sprintf("%s:%s:%s", h.Type(), c.Method(), c.Path())
}

func (h *NewContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
    if !h.IsEnabled() {
        return "", errors.New("handler not enabled")
    }
    
    // 캐시된 설정 사용
    servers := h.proxyConfig.Proxies
    if len(servers) == 0 {
        return "", errors.New("no upstream servers configured")
    }
    
    // 라운드 로빈 서버 선택
    serverIdx := atomic.AddUint32(&h.serverIdx, 1) % uint32(len(servers))
    server := servers[serverIdx]
    
    upstreamURL := server.URL + c.Path()
    h.RecordUpstreamBuild(true)
    return upstreamURL, nil
}

func (h *NewContainerHandler) Handle(c *fiber.Ctx) error {
    return h.WithMetrics(c, func() error {
        // 비즈니스 로직 구현
        upstreamURL, err := h.BuildUpstreamURL(c)
        if err != nil {
            return err
        }
        
        // 실제 프록시 처리
        return h.proxyRequest(c, upstreamURL)
    })
}
```

### 4단계: 팩토리에 핸들러 등록

**Container에 등록**:
```go
// internal/app/container.go
func (c *Container) registerContainerProxyHandlers(factory *handlers.StandardProxyHandlerFactory) error {
    containerMetrics := metrics.GetContainerMetrics()
    
    if err := factory.RegisterHandler("some", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) {
        containerMetrics.RecordHandlerFactoryOperation("create", "some", true)
        return &containerHandlerAdapter{
            name:      "some-container-handler",
            proxyType: "some",
            provider:  provider,
        }, nil
    }); err != nil {
        containerMetrics.RecordHandlerFactoryOperation("register", "some", false)
        return fmt.Errorf("failed to register Some handler: %w", err)
    }
    containerMetrics.RecordHandlerFactoryOperation("register", "some", true)
    
    return nil
}
```

## 테스트 마이그레이션

### Before: 복잡한 설정 Mock
```go
func TestOldHandler(t *testing.T) {
    // 설정 파일 생성
    configPath := createTempConfig(t)
    defer os.Remove(configPath)
    
    handler := &OldHandler{configPath: configPath}
    // 복잡한 테스트 설정...
}
```

### After: MockContainer 사용
```go
func TestNewContainerHandler(t *testing.T) {
    mockContainer := testutil.NewMockContainerProvider(t)
    handler := NewSomeContainerHandler(mockContainer)
    
    assert.NotNil(t, handler)
    assert.Equal(t, "some-container-handler", handler.Name())
    assert.Equal(t, "some", handler.Type())
    assert.True(t, handler.IsEnabled())
    assert.NoError(t, handler.HealthCheck())
}

func TestHandlerWithConfigError(t *testing.T) {
    mockContainer := testutil.NewMockContainerProvider(t).
        WithError("GetSomeProxyConfig", assert.AnError)
    
    handler := NewSomeContainerHandler(mockContainer)
    assert.False(t, handler.IsEnabled())
}
```

## 성능 모니터링

### Container 메트릭 활용

**주요 메트릭**:
- `proxynd_container_handler_initializations_total`: 핸들러 초기화 횟수
- `proxynd_config_load_operations_total`: 설정 로딩 횟수 (감소 목표)
- `proxynd_config_cache_hits_total`: 설정 캐시 히트 (증가 목표)  
- `proxynd_container_handler_duration_seconds`: 핸들러 응답 시간
- `proxynd_handler_instances_active`: 활성 핸들러 인스턴스 수

**Grafana 대시보드 쿼리 예시**:
```promql
# 설정 로딩 호출 감소율
(
  rate(proxynd_config_load_operations_total{result="success"}[5m]) /
  rate(proxynd_container_handler_requests_total[5m])
) * 100

# 캐시 히트율
(
  rate(proxynd_config_cache_hits_total[5m]) /
  (rate(proxynd_config_cache_hits_total[5m]) + rate(proxynd_config_cache_misses_total[5m]))
) * 100
```

## 마이그레이션 체크리스트

### ✅ 필수 단계
- [ ] Container Provider 인터페이스 구현 확인
- [ ] BaseContainerHandler 상속
- [ ] 모든 필수 인터페이스 메서드 구현
- [ ] 팩토리에 핸들러 등록
- [ ] 메트릭 통합 테스트 통과
- [ ] MockContainer 테스트 작성

### ✅ 성능 검증
- [ ] 설정 로딩 횟수 감소 확인 (53+ → 1)
- [ ] 메모리 사용량 감소 확인
- [ ] 응답 시간 개선 확인
- [ ] 캐시 히트율 90% 이상 달성

### ✅ 운영 준비
- [ ] 로그 레벨 및 구조화 확인
- [ ] 헬스체크 엔드포인트 테스트
- [ ] 설정 핫 리로드 테스트
- [ ] 모니터링 대시보드 구성

## 예상 성능 개선

### ReadConfig() 호출 최적화
- **Before**: 요청당 1-3회 호출 × 많은 동시 요청 = 수천 회/초
- **After**: 애플리케이션 시작시 1회 + 설정 변경시에만 호출

### 메모리 사용량 최적화  
- **Before**: 요청당 새 인스턴스 생성
- **After**: 싱글톤 인스턴스 재사용

### 응답 시간 개선
- **Before**: 설정 파일 I/O 지연
- **After**: 메모리 캐시에서 즉시 액세스

## 트러블슈팅

### 일반적인 문제와 해결책

**1. 설정 로딩 실패**
```go
// 문제: IsEnabled()가 false 반환
// 해결: LoadConfig() 메서드에서 에러 로깅 추가

func (h *Handler) LoadConfig() error {
    config, err := h.containerProvider.GetSomeProxyConfig()
    if err != nil {
        h.logger.Error("Failed to load config", logging.F("error", err))
        h.RecordConfigCacheMiss("some-config")
        return err
    }
    // ...
}
```

**2. 메트릭 누락**
```go
// 문제: 메트릭이 수집되지 않음
// 해결: BaseContainerHandler 사용 및 메트릭 래퍼 적용

handler := middleware.WrapContainerHandler(
    NewSomeContainerHandler(provider)
)
```

**3. 테스트 실패**
```go
// 문제: MockContainer 설정 오류  
// 해결: 올바른 Mock 설정

mockContainer := testutil.NewMockContainerProvider(t).
    WithStorageDir("/tmp/test-storage").
    WithConfigDir("/tmp/test-config")
```

## 마이그레이션 이후 확인사항

### 1. 로그 모니터링
```bash
# 설정 로딩 관련 로그 확인
grep "Config.*load" /var/log/proxynd.log

# 에러 로그 모니터링
grep "ERROR" /var/log/proxynd.log | grep -i container
```

### 2. 메트릭 확인
```bash
# Prometheus 메트릭 엔드포인트 확인
curl http://localhost:8080/metrics | grep proxynd_container

# 설정 캐시 히트율 확인
curl -s http://localhost:8080/metrics | grep config_cache_hits
```

### 3. 성능 벤치마크
```bash
# 응답 시간 측정
ab -n 1000 -c 10 http://localhost:8080/api/v1/proxy/maven/...

# 메모리 사용량 모니터링
ps aux | grep proxynd
```

이 가이드를 따라 단계적으로 마이그레이션하면 안전하고 효율적으로 Container 기반 아키텍처로 전환할 수 있습니다.