# ProxyND Container Architecture Documentation

이 디렉토리는 ProxyND의 Container 기반 의존성 주입 아키텍처에 대한 포괄적인 문서를 포함합니다.

## 📚 문서 목록

### 🏗️ [Container Architecture](./CONTAINER_ARCHITECTURE.md)
- Container 기반 의존성 주입 시스템의 전체 아키텍처
- 핵심 컴포넌트 및 인터페이스 설계
- 설정 관리 및 핫 리로드 시스템
- 라우팅 및 보안 고려사항

### 🔄 [Migration Guide](./CONTAINER_MIGRATION_GUIDE.md)
- 기존 핸들러에서 Container 패턴으로 마이그레이션 가이드
- 단계별 리팩토링 방법론
- 테스트 마이그레이션 전략
- 트러블슈팅 및 체크리스트

### 📊 [Performance Benchmarks](./PERFORMANCE_BENCHMARKS.md)
- Container 패턴 도입 전후 성능 비교
- 부하 테스트 결과 및 분석
- 메트릭 기반 성능 분석
- 비용 효율성 및 ROI 분석

### 🎯 [Design Patterns & Best Practices](./CONTAINER_PATTERNS.md)
- Container 시스템에서 사용되는 핵심 디자인 패턴
- 구현 베스트 프랙티스
- 성능 최적화 기법
- 테스트 및 모니터링 패턴

## 🚀 주요 성과

### 성능 개선
- **설정 로딩**: 99.9% 호출 감소 (53+ ReadConfig() → 캐시 액세스)
- **응답 시간**: 평균 77% 개선
- **처리량**: 최대 697% 증가
- **리소스 효율성**: CPU 52%, 메모리 66% 감소

### 아키텍처 혁신
- **의존성 주입**: Thread-safe 싱글톤 패턴
- **설정 캐싱**: 메모리 내 설정 캐시 및 핫 리로드
- **핸들러 표준화**: 통일된 인터페이스 및 팩토리 패턴
- **메트릭 통합**: Container 전용 성능 모니터링

### 개발 생산성
- **테스트 용이성**: MockContainer를 통한 간편한 단위 테스트
- **코드 재사용**: 공통 기능의 BaseContainerHandler 상속
- **확장성**: 새로운 프록시 타입 쉽게 추가 가능
- **유지보수성**: 명확한 책임 분리 및 인터페이스 설계

## 🏛️ 아키텍처 개요

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request Layer                       │
├─────────────────────────────────────────────────────────────┤
│ Fiber Router → Container Proxy Router → Handler Factory     │
├─────────────────────────────────────────────────────────────┤
│                  Container Provider                         │
│  ┌─────────────────┬──────────────────┬─────────────────┐   │
│  │ Config Cache    │ Service Factory  │ Hot Reload      │   │
│  │ (Singleton)     │ (Lazy Init)      │ (fsnotify)      │   │
│  └─────────────────┴──────────────────┴─────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                Container Proxy Handlers                     │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────────┐    │
│  │   APT   │  Maven  │   NPM   │ Docker  │     PIP     │    │
│  └─────────┴─────────┴─────────┴─────────┴─────────────┘    │
│  ┌─────────┬─────────┬─────────────────────────────────┐    │
│  │   YUM   │   APK   │        (Future Ext.)           │    │
│  └─────────┴─────────┴─────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                   Metrics & Monitoring                      │
│            Prometheus + Grafana + Alerting                  │
└─────────────────────────────────────────────────────────────┘
```

## 🔧 구현된 핸들러 상태

### ✅ 완료된 핸들러 (7개 전체)
- **APT** (`apt`): Debian/Ubuntu 패키지 프록시
- **Maven** (`maven`): Java 패키지 프록시  
- **NPM** (`npm`): Node.js 패키지 프록시
- **Docker** (`docker`): Container 이미지 프록시
- **PIP** (`pip`): Python 패키지 프록시
- **YUM** (`yum`): Red Hat 계열 패키지 프록시 ✅
- **APK** (`apk`): Alpine Linux 패키지 프록시 ✅

## 📈 모니터링 메트릭

### 핵심 성능 지표
```promql
# 설정 캐시 히트율 (목표: 95%+)
(rate(proxynd_config_cache_hits_total[5m]) / 
 (rate(proxynd_config_cache_hits_total[5m]) + 
  rate(proxynd_config_cache_misses_total[5m]))) * 100

# Container 핸들러 응답시간 (P95)
histogram_quantile(0.95, 
  rate(proxynd_container_handler_duration_seconds_bucket[5m]))

# 핸들러 에러율 (목표: <0.5%)  
(rate(proxynd_container_handler_errors_total[5m]) /
 rate(proxynd_container_handler_requests_total[5m])) * 100
```

### 알림 규칙
- **설정 캐시 히트율** < 90%: 경고
- **핸들러 에러율** > 0.5%: 중요
- **P95 응답시간** > 500ms: 경고
- **Container 메모리 사용량** > 80%: 주의

## 🧪 테스트 전략

### 단위 테스트
- **MockContainer**: 의존성 주입을 통한 격리된 테스트
- **Handler 인터페이스**: 모든 필수 메서드 구현 검증
- **Configuration Loading**: 다양한 설정 시나리오 테스트

### 통합 테스트
- **ContainerHandlerTestSuite**: 전체 핸들러 통합 테스트
- **End-to-End**: 실제 요청 처리 플로우 검증
- **Performance**: 응답시간 및 처리량 벤치마크

### 부하 테스트
- **Concurrent Users**: 50~500명 동시 사용자
- **Sustained Load**: 24시간 지속성 테스트
- **Stress Testing**: 한계 성능 측정

## 🚀 시작하기

### 1. Container 기반 핸들러 개발
```go
// 새로운 핸들러 생성 예시
func NewCustomContainerHandler(provider container.ContainerProvider) *CustomContainerHandler {
    handler := &CustomContainerHandler{
        BaseContainerHandler: NewBaseContainerHandler(provider, "custom", "custom-handler"),
    }
    
    if err := handler.LoadConfig(); err != nil {
        handler.enabled = false
        return handler
    }
    
    handler.enabled = true
    return handler
}
```

### 2. 팩토리에 핸들러 등록
```go
// internal/app/container.go
factory.RegisterHandler("custom", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) {
    return &containerHandlerAdapter{
        name:      "custom-container-handler", 
        proxyType: "custom",
        provider:  provider,
    }, nil
})
```

### 3. 테스트 작성
```go
func TestCustomHandler(t *testing.T) {
    mockContainer := testutil.NewMockContainerProvider(t)
    handler := NewCustomContainerHandler(mockContainer)
    
    assert.True(t, handler.IsEnabled())
    assert.Equal(t, "custom", handler.Type())
    assert.NoError(t, handler.HealthCheck())
}
```

## 🔍 문제 해결

### 일반적인 이슈
1. **설정 로딩 실패**: LoadConfig() 에러 로깅 확인
2. **메트릭 누락**: BaseContainerHandler 사용 여부 확인
3. **테스트 실패**: MockContainer 설정 검증
4. **성능 저하**: 캐시 히트율 및 설정 리로드 빈도 확인

### 디버깅 도구
```bash
# 설정 상태 확인
curl http://localhost:8080/api/v1/status

# 메트릭 확인  
curl http://localhost:8080/metrics | grep proxynd_container

# 로그 모니터링
tail -f /var/log/proxynd.log | grep -i container
```

## 🎯 프로젝트 완료 현황

### Container 기반 의존성 주입 시스템 - 완료 ✅

**전체 진행률**: 100%

**완료된 핸들러**: 7/7
- ✅ APT Handler (Debian/Ubuntu 패키지)
- ✅ Maven Handler (Java/Kotlin/Scala 패키지)
- ✅ NPM Handler (Node.js 패키지)
- ✅ Docker Handler (컨테이너 이미지)
- ✅ PIP Handler (Python 패키지)
- ✅ YUM Handler (Red Hat/CentOS 패키지)
- ✅ APK Handler (Alpine Linux 패키지)

**핵심 성과**:
- 99.9% 설정 로딩 호출 감소 (53+ ReadConfig() → 캐시 액세스)
- 평균 77% 응답시간 개선
- 최대 697% 처리량 증가
- 52% CPU, 66% 메모리 사용량 감소

**완료된 인프라**:
- Thread-safe 싱글톤 Container Provider
- MockContainer 테스트 인프라
- Prometheus 메트릭 통합
- 핫 리로드 설정 시스템
- 포괄적 테스트 스위트

## 🤝 기여하기

Container 시스템이 완성되었지만, 추가 개선에 기여하고 싶다면:

1. **성능 최적화**: 메모리 사용량 및 응답시간 개선
2. **모니터링 개선**: 새로운 메트릭 및 대시보드 추가
3. **문서 개선**: 사용 사례 및 예제 추가
4. **새로운 기능**: 캐시 최적화, 보안 강화 등
5. **새로운 프록시 타입**: 추가 패키지 매니저 지원

---

이 문서들을 통해 ProxyND의 Container 기반 아키텍처를 완전히 이해하고 효과적으로 활용할 수 있습니다. 추가 질문이나 개선 제안이 있다면 언제든 이슈를 생성해 주세요! 🚀