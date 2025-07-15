# ProxyND Integration Testing Guide

## Overview

이 가이드는 ProxyND의 통합 테스트 전략과 구현을 설명합니다. 통합 테스트는 여러 컴포넌트가 함께 작동하는 것을 검증하며, 실제 프록시 플로우를 end-to-end로 테스트합니다.

## 통합 테스트 구조

### 테스트 파일 구성

```
tests/integration/
├── proxy_test.go          # 새로운 포괄적인 프록시 플로우 테스트
├── integration_test.go    # 기존 통합 테스트 스위트
├── global.yaml           # 테스트용 글로벌 설정
├── test_config.yaml      # 테스트별 설정
└── Makefile             # 테스트 실행 명령
```

## 주요 테스트 시나리오

### 1. APT 프록시 플로우

- **Release 파일 다운로드**: 저장소 메타데이터 검증
- **패키지 목록 다운로드**: 압축된 패키지 목록 처리
- **DEB 패키지 다운로드**: 실제 패키지 파일 전달
- **캐시 동작**: 캐시 히트/미스 검증

```go
func TestAPTProxyFlow(t *testing.T) {
    // Release 파일, 패키지 목록, DEB 파일 다운로드
    // 캐시 동작 검증
}
```

### 2. Maven 프록시 플로우

- **메타데이터 처리**: maven-metadata.xml 파싱
- **아티팩트 다운로드**: JAR, POM 파일 처리
- **체크섬 검증**: SHA1 체크섬 파일 처리
- **리포지토리 우선순위**: 다중 업스트림 처리

```go
func TestMavenProxyFlow(t *testing.T) {
    // 메타데이터, JAR, POM, 체크섬 파일 처리
    // 업스트림 우선순위 검증
}
```

### 3. NPM 프록시 플로우

- **패키지 메타데이터**: package.json 처리
- **Tarball 다운로드**: 압축 패키지 전달
- **Scoped 패키지**: @scope/package 형식 처리
- **레지스트리 리다이렉트**: 업스트림 URL 변환

```go
func TestNPMProxyFlow(t *testing.T) {
    // 일반 패키지, scoped 패키지, tarball 처리
}
```

### 4. 동시성 테스트

- **동시 요청 처리**: 여러 클라이언트의 동시 요청
- **캐시 경합**: 동일 리소스에 대한 동시 접근
- **리소스 잠금**: 캐시 쓰기 동기화

```go
func TestConcurrentRequests(t *testing.T) {
    // 10개의 동시 요청 처리
    // 캐시 일관성 검증
}
```

### 5. 대용량 파일 처리

- **스트리밍 다운로드**: 메모리 효율적인 대용량 파일 처리
- **청크 전송**: HTTP 청크 인코딩 지원
- **진행률 추적**: 다운로드 진행 상태

```go
func TestLargeFileHandling(t *testing.T) {
    // 10MB 파일 처리
    // 메모리 사용량 모니터링
}
```

### 6. 인증 플로우

- **Basic Auth**: 기본 인증 처리
- **Bearer Token**: 토큰 기반 인증
- **인증 전파**: 업스트림으로 인증 정보 전달

```go
func TestAuthenticationFlow(t *testing.T) {
    // 인증 있는/없는 요청 처리
    // 401 응답 처리
}
```

### 7. 에러 처리

- **업스트림 다운**: 서버 연결 실패 처리
- **타임아웃**: 요청 타임아웃 처리
- **잘못된 응답**: 예상치 못한 응답 형식 처리

```go
func TestErrorHandling(t *testing.T) {
    // 네트워크 에러, 타임아웃, 잘못된 응답 처리
}
```

### 8. 캐시 무효화

- **PUT/POST 요청**: 업로드로 인한 캐시 무효화
- **DELETE 요청**: 리소스 삭제 시 캐시 제거
- **TTL 만료**: 시간 기반 캐시 만료

```go
func TestCacheInvalidation(t *testing.T) {
    // PUT 요청 후 캐시 무효화 검증
}
```

## 테스트 환경 설정

### Mock 업스트림 서버

통합 테스트는 실제 외부 서버 대신 httptest로 생성된 mock 서버를 사용합니다:

```go
func setupUpstreamServers() {
    // APT, Maven, NPM mock 서버 생성
    // 각 프록시 타입별 응답 시뮬레이션
}
```

### 테스트 설정 파일

각 프록시 타입별 테스트 설정이 동적으로 생성됩니다:

```go
func createTestConfigs(t *testing.T, configDir string) {
    // global.yaml, apt-proxy.yaml 등 생성
    // Mock 서버 URL 주입
}
```

## 테스트 실행

### 개별 테스트 실행

```bash
# 특정 프록시 플로우 테스트
go test -v -run TestAPTProxyFlow ./tests/integration

# 동시성 테스트
go test -v -run TestConcurrentRequests ./tests/integration

# 모든 통합 테스트
go test -v ./tests/integration
```

### 통합 테스트 스크립트

```bash
# 전체 통합 테스트 실행
./scripts/run_integration_tests.sh

# 벤치마크 포함 실행
./scripts/run_integration_tests.sh --bench
```

### Makefile 타겟

```bash
# 통합 테스트 실행
make test-integration

# 모든 테스트 (단위 + 통합)
make test-all
```

## 테스트 커버리지

### 커버리지 목표

- **프록시 플로우**: 90% 이상
- **캐시 동작**: 85% 이상
- **에러 처리**: 80% 이상
- **전체**: 75% 이상

### 커버리지 리포트 생성

```bash
# 커버리지 프로파일 생성
go test -coverprofile=coverage.out ./tests/integration

# HTML 리포트 생성
go tool cover -html=coverage.out -o coverage.html
```

## 성능 벤치마크

### 프록시 요청 벤치마크

```go
func BenchmarkProxyRequests(b *testing.B) {
    // 병렬 요청 처리 성능 측정
    // 캐시 히트/미스 비율 측정
}
```

### 벤치마크 실행

```bash
# 기본 벤치마크
go test -bench=. ./tests/integration

# 메모리 프로파일링 포함
go test -bench=. -benchmem ./tests/integration

# CPU 프로파일 생성
go test -bench=. -cpuprofile=cpu.prof ./tests/integration
```

## CI/CD 통합

### GitHub Actions 설정

```yaml
- name: Run Integration Tests
  run: |
    make test-integration
  env:
    TEST_ENV: ci
```

### 테스트 실패 시 디버깅

1. **로그 확인**: 테스트 출력에서 실패 원인 확인
2. **Mock 서버 응답**: Mock 서버가 예상된 응답을 반환하는지 확인
3. **타이밍 이슈**: 동시성 테스트에서 race condition 확인
4. **환경 변수**: 필요한 환경 변수가 설정되었는지 확인

## 테스트 작성 가이드라인

### 1. 테스트 격리

각 테스트는 독립적으로 실행 가능해야 합니다:

```go
func (s *ProxyIntegrationTestSuite) SetupTest(t *testing.T) {
    // 각 테스트마다 새로운 환경 생성
    s.tempDir = t.TempDir()
    // ...
}
```

### 2. Mock 서버 사용

실제 외부 서버 대신 httptest 서버 사용:

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // 요청에 따른 응답 시뮬레이션
}))
defer server.Close()
```

### 3. 타임아웃 설정

모든 테스트에 적절한 타임아웃 설정:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

### 4. 에러 검증

예상된 에러와 예상치 못한 에러 구분:

```go
if tt.wantErr {
    assert.Error(t, err)
    assert.Contains(t, err.Error(), tt.errMsg)
} else {
    assert.NoError(t, err)
}
```

## 트러블슈팅

### 일반적인 문제

1. **포트 충돌**: 테스트 서버가 이미 사용 중인 포트 사용
   - 해결: 동적 포트 할당 사용

2. **캐시 오염**: 이전 테스트의 캐시가 남아있음
   - 해결: t.TempDir() 사용으로 격리된 캐시 디렉토리

3. **타이밍 이슈**: 서버 시작 전 요청 발생
   - 해결: 적절한 대기 시간 또는 health check

4. **메모리 부족**: 대용량 파일 테스트 시
   - 해결: 스트리밍 처리 확인

## 향후 개선 사항

1. **E2E 테스트 통합**: 실제 Docker 컨테이너로 전체 시스템 테스트
2. **부하 테스트**: k6 또는 vegeta를 사용한 부하 테스트
3. **카오스 테스트**: 네트워크 장애 시뮬레이션
4. **보안 테스트**: SQL injection, XSS 등 보안 취약점 테스트