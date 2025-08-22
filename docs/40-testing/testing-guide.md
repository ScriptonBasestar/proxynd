# 🧪 ProxyND 종합 테스트 가이드

## 📋 개요

ProxyND는 **헥사고널 + 클린 아키텍처**에 맞춘 **4계층 테스트 피라미드**를 따릅니다:

```
                    ┌─────────────────┐
                    │   E2E Tests     │ ← 전체 시스템 검증
                    │ (docker-compose)│
                    └─────────────────┘
                  ┌───────────────────────┐
                  │  Integration Tests    │ ← 어댑터 계층
                  │  (External systems)   │
                  └───────────────────────┘
                ┌─────────────────────────────┐
                │     Contract Tests          │ ← 포트 계층
                │   (Interface contracts)     │
                └─────────────────────────────┘
              ┌───────────────────────────────────┐
              │           Unit Tests              │ ← 도메인/유스케이스 계층
              │      (Business logic)             │
              └───────────────────────────────────┘
```

이 가이드는 아키텍처에 맞춘 종합적인 테스트 전략과 구현을 설명합니다. testify 프레임워크와 테이블 기반 테스트를 사용하여 포괄적인 커버리지와 유지보수성을 확보합니다.

## 🚀 빠른 시작

### 필수 요구사항

```bash
# Go 1.21 이상 설치
go version

# 테스트 의존성 설치
go mod download

# 개발 도구 설치 (선택사항)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 테스트 실행

```bash
# 빠른 테스트 실행 (단위 테스트만)
make test-unit

# 커버리지와 함께 전체 테스트 스위트
make test-coverage

# 통합 테스트 실행
make test-integration

# E2E 테스트 실행
make test-e2e

# 모든 테스트 실행
make test-all

# 벤치마크 테스트
make test-benchmark
```

## 📊 테스트 계층별 상세 가이드

### 1. 🔬 단위 테스트 (Unit Tests)

**목적**: 개별 컴포넌트의 비즈니스 로직 검증

#### 테스트 범위

##### 캐시 시스템 테스트
- **FileSystemBackend**: 파일시스템 캐시 백엔드
  - Put/Get 동작
  - 존재 여부 확인 (Exists)
  - 삭제 (Delete)
  - TTL 만료 처리
  - 캐시 크기 관리
  - 전체 삭제 (Clear)

- **CacheManager**: 캐시 매니저
  - 캐시 CRUD 동작
  - 통계 수집 (히트/미스)
  - 캐시 경로 생성
  - 캐시 정리

- **CacheEviction**: 캐시 제거 정책
  - LRU 정책 테스트
  - TTL 기반 제거
  - 크기 기반 제거
  - 제거 후보 선택

##### 미들웨어 테스트
- **IPFilterMiddleware**: IP 필터링
  - 특정 IP 허용/차단
  - CIDR 범위 기반 필터링
  - 기본 허용/차단 모드

- **BasicAuthMiddleware**: 기본 인증
  - 올바른 인증 정보 처리
  - 잘못된 인증 정보 처리
  - 인증 헤더 없는 요청
  - 다양한 인증 형식

- **PermissionMiddleware**: 권한 관리
  - 읽기/쓰기/삭제 권한
  - 사용자별 권한 설정
  - 기본 권한 적용

- **SecurityMiddleware**: 보안 검증
  - SHA256 해시 검증
  - 다양한 해시 헤더 형식
  - 해시 불일치 처리

##### 헬퍼 함수 테스트
- **YAMLHelper**: YAML 처리
  - YAML 읽기/쓰기
  - 복잡한 데이터 구조
  - 오류 처리

- **URLHelper**: URL 처리
  - URL 경로 결합
  - 경로 정리
  - 패키지 이름 파싱
  - URL 유효성 검증
  - URL 인코딩

#### 실행 방법

```bash
# 기본 단위 테스트 실행
make test-unit

# 개별 테스트 모듈 실행
go test -v ./internal/services/cache/...
go test -v ./internal/middleware/...
go test -v ./internal/helpers/...

# 특정 테스트 함수 실행
go test -v -run TestCacheManager ./internal/services/cache

# 커버리지 측정
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 벤치마크 실행
go test -bench=. ./...
go test -bench=. -benchmem ./...
```

#### 작성 가이드라인

```go
// 테스트 함수 명명 규칙
func TestComponentName_FunctionName(t *testing.T) {
    // 기본 테스트
}

func TestComponentName_FunctionName_SpecificCase(t *testing.T) {
    // 특정 케이스 테스트
}

// 서브테스트 활용
func TestComponent(t *testing.T) {
    t.Run("Success Case", func(t *testing.T) {
        // 성공 케이스
    })

    t.Run("Error Case", func(t *testing.T) {
        // 오류 케이스
    })
}

// 테이블 기반 테스트
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"success case", "input1", "output1", false},
        {"error case", "invalid", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### 2. 🔗 통합 테스트 (Integration Tests)

**목적**: 여러 컴포넌트가 함께 작동하는 것을 검증하며, 실제 프록시 플로우를 end-to-end로 테스트

#### 주요 테스트 시나리오

##### APT 프록시 플로우
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

##### Maven 프록시 플로우
- **메타데이터 처리**: maven-metadata.xml 파싱
- **아티팩트 다운로드**: JAR, POM 파일 처리
- **체크섬 검증**: SHA1 체크섬 파일 처리
- **리포지토리 우선순위**: 다중 업스트림 처리

##### NPM 프록시 플로우
- **패키지 메타데이터**: package.json 처리
- **Tarball 다운로드**: 압축 패키지 전달
- **Scoped 패키지**: @scope/package 형식 처리
- **레지스트리 리다이렉트**: 업스트림 URL 변환

##### 동시성 테스트
- **동시 요청 처리**: 여러 클라이언트의 동시 요청
- **캐시 경합**: 동일 리소스에 대한 동시 접근
- **리소스 잠금**: 캐시 쓰기 동기화

##### 대용량 파일 처리
- **스트리밍 다운로드**: 메모리 효율적인 대용량 파일 처리
- **청크 전송**: HTTP 청크 인코딩 지원
- **진행률 추적**: 다운로드 진행 상태

##### 인증 플로우
- **Basic Auth**: 기본 인증 처리
- **Bearer Token**: 토큰 기반 인증
- **인증 전파**: 업스트림으로 인증 정보 전달

##### 에러 처리
- **업스트림 다운**: 서버 연결 실패 처리
- **타임아웃**: 요청 타임아웃 처리
- **잘못된 응답**: 예상치 못한 응답 형식 처리

#### 테스트 환경 설정

##### Mock 업스트림 서버
통합 테스트는 실제 외부 서버 대신 httptest로 생성된 mock 서버를 사용합니다:

```go
func setupUpstreamServers() {
    // APT, Maven, NPM mock 서버 생성
    // 각 프록시 타입별 응답 시뮬레이션
}
```

##### 테스트 설정 파일
각 프록시 타입별 테스트 설정이 동적으로 생성됩니다:

```go
func createTestConfigs(t *testing.T, configDir string) {
    // global.yaml, apt-proxy.yaml 등 생성
    // Mock 서버 URL 주입
}
```

#### 실행 방법

```bash
# 통합 테스트 실행
make test-integration

# 특정 프록시 플로우 테스트
go test -v -run TestAPTProxyFlow ./tests/integration
go test -v -run TestMavenProxyFlow ./tests/integration
go test -v -run TestNPMProxyFlow ./tests/integration

# 동시성 테스트
go test -v -run TestConcurrentRequests ./tests/integration

# 모든 통합 테스트
go test -v ./tests/integration

# 벤치마크 포함 실행
go test -v -bench=. ./tests/integration
```

### 3. 🔌 계약 테스트 (Contract Tests)

**목적**: 인터페이스 계약과 API 호환성 검증

#### 테스트 영역
- **API 계약**: REST API 스펙 준수
- **포트 인터페이스**: 도메인과 어댑터 간 계약
- **데이터 스키마**: 입출력 데이터 구조 검증
- **프로토콜 호환성**: 각 패키지 매니저 프로토콜 준수

```bash
# 계약 테스트 실행
make test-contract
```

### 4. 🌐 E2E 테스트 (End-to-End Tests)

**목적**: 실제 환경과 유사한 전체 시스템 검증

#### 🏗️ E2E 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                    E2E Test Environment                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Clients    │    │   ProxyND    │    │  Upstream    │  │
│  │              │───▶│              │───▶│   Server     │  │
│  │ npm/pip/apt  │    │   (Main)     │    │   (Nginx)    │  │
│  │   /docker    │    │              │    │              │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐                      │
│  │    MinIO     │    │ Test Runner  │                      │
│  │ (S3 Cache)   │    │   (Ginkgo)   │                      │
│  └──────────────┘    └──────────────┘                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 구성 요소

| 서비스 | 포트 | 설명 |
|--------|------|------|
| **proxynd** | 8080 | 메인 ProxyND 서버 |
| **minio** | 9000, 9001 | S3 호환 스토리지 (캐시 백엔드) |
| **nginx-upstream** | 8081 | 업스트림 서버 시뮬레이션 |
| **ubuntu-client** | - | APT 클라이언트 테스트용 |
| **node-client** | - | NPM 클라이언트 테스트용 |
| **python-client** | - | PyPI 클라이언트 테스트용 |
| **docker-client** | - | Docker 클라이언트 테스트용 |
| **test-runner** | - | E2E 테스트 실행기 |

#### E2E 테스트 실행

```bash
# E2E 환경 시작
cd tests/e2e
make up

# E2E 테스트 실행
make test-e2e

# 클라이언트별 테스트
make test-npm      # NPM 클라이언트 테스트
make test-pip      # PyPI 클라이언트 테스트
make test-apt      # APT 클라이언트 테스트
make test-docker   # Docker 클라이언트 테스트

# 전체 E2E 테스트
make test-e2e-all

# 환경 정리
make down
```

#### E2E 테스트 시나리오

##### NPM 프록시 테스트
- 패키지 메타데이터 조회
- 스코프 패키지 처리
- tarball 다운로드
- 캐시 동작 확인

##### PyPI 프록시 테스트
- Simple API 조회
- 패키지 인덱스 접근
- wheel/tarball 다운로드
- pip 클라이언트 호환성

##### APT 프록시 테스트
- Release 파일 조회
- 패키지 목록 다운로드
- .deb 파일 접근
- apt 클라이언트 호환성

##### Docker 프록시 테스트
- Registry API v2 호환성
- 매니페스트 조회
- 블롭 다운로드
- 카탈로그 API

##### 캐시 테스트
- 첫 요청 vs 캐시된 요청 성능
- TTL 만료 동작
- 캐시 무효화

##### 보안 테스트
- IP 필터링
- Basic 인증
- 해시 검증

#### 개발 워크플로우

```bash
# 환경 시작
make up

# 개발 작업...

# 테스트 실행
make test

# 로그 확인
make logs-proxynd

# 환경 재시작 (코드 변경 후)
make restart

# 디버깅
make status          # 서비스 상태 확인
make logs-follow     # 로그 실시간 보기
make shell-proxynd   # 컨테이너 접속
make health          # 헬스체크
```

## 📊 테스트 커버리지 및 품질

### 커버리지 목표

- **전체 커버리지**: 80% 이상
- **핵심 컴포넌트**: 90% 이상
- **프록시 플로우**: 90% 이상
- **캐시 동작**: 85% 이상
- **에러 처리**: 80% 이상
- **미들웨어**: 85% 이상
- **헬퍼 함수**: 80% 이상

### 커버리지 리포트 생성

```bash
# 전체 커버리지 리포트
make test-coverage

# HTML 리포트 생성
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 함수별 상세 커버리지
go tool cover -func=coverage.out

# 패키지별 커버리지
go test -cover ./...
```

## 🚀 성능 벤치마크

### 벤치마크 테스트

벤치마크 테스트는 다음을 측정합니다:
- 캐시 조회 성능
- 미들웨어 처리 시간
- URL 파싱 성능
- YAML 처리 성능
- 프록시 요청 처리 성능

### 벤치마크 실행

```bash
# 기본 벤치마크
make test-benchmark

# 모든 벤치마크 실행
go test -bench=. ./...

# 메모리 프로파일링 포함
go test -bench=. -benchmem ./...

# CPU 프로파일 생성
go test -bench=. -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# 메모리 프로파일 생성
go test -bench=. -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### 벤치마크 작성 예시

```go
func BenchmarkCacheGet(b *testing.B) {
    // 설정...
    cache := NewFileSystemBackend(tmpDir)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // 측정할 코드
        _, err := cache.Get("test-key")
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkProxyRequests(b *testing.B) {
    // 병렬 요청 처리 성능 측정
    // 캐시 히트/미스 비율 측정
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            // 병렬 실행할 코드
        }
    })
}
```

## 🔧 테스트 유틸리티 및 헬퍼

### 테스트 헬퍼 함수

```go
// 테스트용 임시 디렉토리 생성
func setupTempDir(t *testing.T) string {
    dir := t.TempDir()
    return dir
}

// Mock HTTP 서버 생성
func setupMockServer(responses map[string]string) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if response, ok := responses[r.URL.Path]; ok {
            w.Write([]byte(response))
        } else {
            http.NotFound(w, r)
        }
    }))
}

// 테스트 설정 생성
func createTestConfig(t *testing.T, configDir string, upstream string) {
    config := fmt.Sprintf(`
server:
  port: 0
  host: localhost

cache:
  backend: filesystem
  base_dir: %s

npm_proxy:
  upstream: %s
`, configDir, upstream)

    err := os.WriteFile(filepath.Join(configDir, "global.yaml"), []byte(config), 0644)
    require.NoError(t, err)
}
```

### 어설션 헬퍼

```go
// HTTP 응답 검증
func assertHTTPResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedContentType string) {
    assert.Equal(t, expectedStatus, resp.StatusCode)
    assert.Equal(t, expectedContentType, resp.Header.Get("Content-Type"))
}

// 캐시 동작 검증
func assertCacheHit(t *testing.T, cacheManager *CacheManager, key string) {
    exists, err := cacheManager.Exists(key)
    assert.NoError(t, err)
    assert.True(t, exists, "Expected cache hit for key: %s", key)
}

// 파일 존재 확인
func assertFileExists(t *testing.T, filepath string) {
    _, err := os.Stat(filepath)
    assert.NoError(t, err, "Expected file to exist: %s", filepath)
}
```

## 🛠️ CI/CD 통합

### GitHub Actions 설정

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [1.21, 1.22]

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}

      # 단위 테스트
      - name: Run Unit Tests
        run: make test-unit

      # 통합 테스트
      - name: Run Integration Tests
        run: make test-integration

      # E2E 테스트
      - name: Run E2E Tests
        run: make test-e2e

      # 커버리지 리포트
      - name: Generate Coverage Report
        run: make test-coverage

      # 커버리지 업로드
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

### 테스트 실패 시 디버깅

1. **로그 확인**: 테스트 출력에서 실패 원인 확인
2. **Mock 서버 응답**: Mock 서버가 예상된 응답을 반환하는지 확인
3. **타이밍 이슈**: 동시성 테스트에서 race condition 확인
4. **환경 변수**: 필요한 환경 변수가 설정되었는지 확인
5. **포트 충돌**: 테스트 서버가 이미 사용 중인 포트 사용하지 않는지 확인

## 🔍 문제 해결

### 일반적인 문제

#### 포트 충돌
```bash
# 사용 중인 포트 확인
netstat -tulpn | grep :8080

# 다른 포트로 변경 또는 동적 포트 할당 사용
```

#### 캐시 오염
```bash
# 테스트 간 격리를 위해 t.TempDir() 사용
```

#### 타이밍 이슈
```bash
# 적절한 대기 시간 또는 health check 구현
```

#### 메모리 부족
```bash
# 스트리밍 처리 확인 및 대용량 파일 테스트 최적화
```

### 성능 문제 해결

1. **벤치마크 결과 비교**
2. **CPU/메모리 프로파일 분석**
3. **병목 지점 식별**
4. **최적화 적용 및 재측정**

### 커버리지 향상

1. **미테스트 경로 식별**
2. **엣지 케이스 추가**
3. **오류 상황 테스트**
4. **통합 시나리오 보강**

## 📝 테스트 작성 가이드라인

### 1. 테스트 격리

각 테스트는 독립적으로 실행 가능해야 합니다:

```go
func (s *TestSuite) SetupTest() {
    // 각 테스트마다 새로운 환경 생성
    s.tempDir = t.TempDir()
    s.config = createTestConfig()
}

func (s *TestSuite) TearDownTest() {
    // 테스트 완료 후 정리
    if s.server != nil {
        s.server.Close()
    }
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

### 5. 테스트 데이터 관리

- 임시 디렉토리 사용 (`t.TempDir()`)
- 테스트 완료 후 정리 (`defer cleanup()`)
- 격리된 환경 구성

## 🏆 모범 사례

### DO ✅

1. **의미 있는 테스트 이름 사용**
2. **테이블 기반 테스트로 여러 케이스 커버**
3. **Setup/Teardown으로 테스트 환경 관리**
4. **Mock과 Stub 적절히 활용**
5. **테스트 실행 시간 최소화**
6. **실패 시 명확한 에러 메시지 제공**

### DON'T ❌

1. **테스트 간 의존성 생성**
2. **실제 외부 서비스에 의존**
3. **하드코딩된 값 사용**
4. **너무 복잡한 테스트 작성**
5. **테스트 코드 주석 없이 방치**
6. **실패하는 테스트 커밋**

## 📚 참고 자료

### 테스트 프레임워크
- [Testify](https://github.com/stretchr/testify) - Go 테스트 라이브러리
- [Ginkgo](https://github.com/onsi/ginkgo) - BDD 스타일 테스트 프레임워크
- [GoMock](https://github.com/golang/mock) - Mock 생성 도구

### 테스트 작성 가이드
- [Go Testing](https://golang.org/pkg/testing/) - Go 공식 테스트 패키지
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests) - 테이블 기반 테스트 패턴
- [Test Fixtures](https://github.com/go-testfixtures/testfixtures) - 테스트 픽스처

### 성능 테스트
- [Go Benchmarks](https://golang.org/pkg/testing/#hdr-Benchmarks) - Go 벤치마크 가이드
- [pprof](https://golang.org/pkg/net/http/pprof/) - Go 프로파일링 도구

---

**마지막 업데이트**: 2024년 12월  
**다음 리뷰**: 2025년 3월  
**담당자**: QA 팀
