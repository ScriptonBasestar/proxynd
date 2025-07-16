# 테스트 커버리지 개선 로드맵

## 1. 현재 테스트 현황 분석

### 1.1 패키지별 커버리지
```bash
# 현재 테스트 커버리지 (make test-coverage 결과)
Package                     Coverage
---------                   ---------
alerts                      21.0%
cache                       22.8%
configs                     45.2%
handlers/proxy              0.0%     # 가장 심각
middlewares                 15.3%
internal/services           28.5%
internal/auth              35.1%
verification               18.9%
---------                   ---------
전체 평균                    약 23%
```

### 1.2 테스트 부재 영역
- **핸들러**: 단위 테스트 전무 (0%)
- **미들웨어**: 기본적인 테스트만 존재
- **통합 테스트**: 거의 없음
- **E2E 테스트**: 없음

## 2. 테스트 전략

### 2.1 테스트 피라미드
```
         /\
        /E2E\        5%  - 주요 시나리오
       /------\
      /통합테스트\    15% - API 엔드포인트
     /----------\
    / 단위 테스트  \   80% - 비즈니스 로직
   /--------------\
```

### 2.2 커버리지 목표
- **전체**: 70%+ (현재 23% → 70%)
- **핵심 패키지**: 80%+ (handlers, services)
- **유틸리티**: 90%+ (configs, cache)
- **실험적 기능**: 50%+

## 3. 단위 테스트 구현

### 3.1 핸들러 테스트 (우선순위: 높음)

#### APT 핸들러 테스트
```go
// handlers/proxy/apt_handler_test.go
package proxy

import (
    "bytes"
    "io"
    "net/http/httptest"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "proxynd/configs"
    "proxynd/test/mocks"
)

func TestAPTHandler_HandlePackageRequest(t *testing.T) {
    tests := []struct {
        name           string
        path           string
        setupMocks     func(*mocks.MockContainer, *mocks.MockCache)
        expectedStatus int
        expectedCache  string
    }{
        {
            name: "캐시 히트",
            path: "/apt/pool/main/v/vim/vim_8.2.deb",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.On("Config").Return(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: true,
                        Mirrors: []string{"http://mirror.example.com"},
                    },
                })
                cache.On("Get", "apt:pool_main_v_vim_vim_8.2.deb").
                    Return([]byte("cached-package-data"), nil)
            },
            expectedStatus: 200,
            expectedCache:  "HIT",
        },
        {
            name: "캐시 미스 - 업스트림 성공",
            path: "/apt/dists/focal/Release",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.On("Config").Return(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: true,
                        Mirrors: []string{"http://mirror.example.com"},
                    },
                })
                cache.On("Get", "apt:dists_focal_Release").
                    Return(nil, errors.ErrCacheNotFound)
                cache.On("SetWithTTL", mock.Anything, mock.Anything, mock.Anything).
                    Return(nil)
            },
            expectedStatus: 200,
            expectedCache:  "MISS",
        },
        {
            name: "프록시 비활성화",
            path: "/apt/any/path",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.On("Config").Return(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: false,
                    },
                })
            },
            expectedStatus: 404,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Given
            app := fiber.New()
            mockContainer := new(mocks.MockContainer)
            mockCache := new(mocks.MockCache)

            tt.setupMocks(mockContainer, mockCache)

            handler := NewAPTHandler(mockContainer)
            app.Get("/apt/*", handler.Handle)

            // When
            req := httptest.NewRequest("GET", tt.path, nil)
            resp, err := app.Test(req, -1)

            // Then
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)

            if tt.expectedCache != "" {
                assert.Equal(t, tt.expectedCache, resp.Header.Get("X-Cache-Status"))
            }

            mockContainer.AssertExpectations(t)
            mockCache.AssertExpectations(t)
        })
    }
}

// 압축 처리 테스트
func TestAPTHandler_CompressionHandling(t *testing.T) {
    tests := []struct {
        name            string
        acceptEncoding  string
        responseData    []byte
        expectedEncoding string
    }{
        {
            name:            "Gzip 압축 요청",
            acceptEncoding:  "gzip",
            responseData:    bytes.Repeat([]byte("test"), 1000),
            expectedEncoding: "gzip",
        },
        {
            name:            "압축 미지원",
            acceptEncoding:  "",
            responseData:    []byte("small data"),
            expectedEncoding: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 테스트 구현
        })
    }
}
```

#### Maven 핸들러 테스트
```go
// handlers/proxy/maven_handler_test.go
package proxy

import (
    "testing"
    "crypto/sha1"
    "encoding/hex"
)

func TestMavenHandler_ChecksumValidation(t *testing.T) {
    tests := []struct {
        name        string
        artifact    []byte
        checksum    string
        shouldError bool
    }{
        {
            name:     "유효한 SHA1 체크섬",
            artifact: []byte("test-artifact-content"),
            checksum: sha1Hash("test-artifact-content"),
            shouldError: false,
        },
        {
            name:     "잘못된 체크섬",
            artifact: []byte("test-artifact-content"),
            checksum: "invalid-checksum",
            shouldError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewMavenHandler(nil)
            err := handler.ValidateChecksum(tt.artifact, tt.checksum)

            if tt.shouldError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func sha1Hash(data string) string {
    h := sha1.New()
    h.Write([]byte(data))
    return hex.EncodeToString(h.Sum(nil))
}
```

### 3.2 서비스 레이어 테스트

#### 캐시 서비스 테스트
```go
// cache/file_cache_test.go
package cache

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestFileCache_GetSet(t *testing.T) {
    // Given
    tempDir := t.TempDir()
    cache := NewFileCache(tempDir, 100*MB)

    tests := []struct {
        name  string
        key   string
        value []byte
        ttl   time.Duration
    }{
        {
            name:  "기본 저장/조회",
            key:   "test-key",
            value: []byte("test-value"),
            ttl:   1 * time.Hour,
        },
        {
            name:  "큰 파일 저장",
            key:   "large-file",
            value: bytes.Repeat([]byte("x"), 10*MB),
            ttl:   24 * time.Hour,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // When: 저장
            err := cache.SetWithTTL(tt.key, tt.value, tt.ttl)
            assert.NoError(t, err)

            // Then: 조회
            retrieved, err := cache.Get(tt.key)
            assert.NoError(t, err)
            assert.Equal(t, tt.value, retrieved)

            // 파일 존재 확인
            filePath := cache.getFilePath(tt.key)
            assert.FileExists(t, filePath)
        })
    }
}

func TestFileCache_Expiration(t *testing.T) {
    // Given
    tempDir := t.TempDir()
    cache := NewFileCache(tempDir, 100*MB)

    // When: 짧은 TTL로 저장
    err := cache.SetWithTTL("expire-key", []byte("value"), 100*time.Millisecond)
    assert.NoError(t, err)

    // Then: 만료 전 조회 가능
    _, err = cache.Get("expire-key")
    assert.NoError(t, err)

    // 만료 후 조회 불가
    time.Sleep(200 * time.Millisecond)
    _, err = cache.Get("expire-key")
    assert.Equal(t, ErrCacheExpired, err)
}

func TestFileCache_LRUEviction(t *testing.T) {
    // Given: 작은 캐시 크기
    tempDir := t.TempDir()
    cache := NewFileCache(tempDir, 1*MB)

    // When: 용량 초과 데이터 저장
    for i := 0; i < 10; i++ {
        key := fmt.Sprintf("key-%d", i)
        value := bytes.Repeat([]byte("x"), 200*KB)
        err := cache.Set(key, value)
        assert.NoError(t, err)
    }

    // Then: 오래된 항목은 제거됨
    _, err := cache.Get("key-0")
    assert.Equal(t, ErrCacheNotFound, err)

    // 최근 항목은 유지됨
    _, err = cache.Get("key-9")
    assert.NoError(t, err)
}
```

### 3.3 미들웨어 테스트

#### 인증 미들웨어 테스트
```go
// middlewares/auth_test.go
package middlewares

import (
    "testing"
    "net/http/httptest"

    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v4"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
    // Given
    app := fiber.New()
    app.Use(AuthMiddleware())
    app.Get("/protected", func(c *fiber.Ctx) error {
        userID := c.Locals("userID").(string)
        return c.JSON(fiber.Map{"user_id": userID})
    })

    // 유효한 토큰 생성
    token := generateTestToken("user123", time.Hour)

    // When
    req := httptest.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    resp, err := app.Test(req)

    // Then
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)

    var body map[string]string
    json.NewDecoder(resp.Body).Decode(&body)
    assert.Equal(t, "user123", body["user_id"])
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
    tests := []struct {
        name   string
        token  string
        status int
    }{
        {
            name:   "토큰 없음",
            token:  "",
            status: 401,
        },
        {
            name:   "잘못된 형식",
            token:  "invalid-token",
            status: 401,
        },
        {
            name:   "만료된 토큰",
            token:  generateTestToken("user", -1*time.Hour),
            status: 401,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            app := setupTestApp()

            req := httptest.NewRequest("GET", "/protected", nil)
            if tt.token != "" {
                req.Header.Set("Authorization", "Bearer "+tt.token)
            }

            resp, _ := app.Test(req)
            assert.Equal(t, tt.status, resp.StatusCode)
        })
    }
}
```

## 4. 통합 테스트

### 4.1 API 엔드포인트 테스트
```go
// test/integration/apt_integration_test.go
package integration

import (
    "testing"
    "net/http"
    "io"
)

func TestAPTProxy_FullFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("통합 테스트는 -short 플래그에서 스킵")
    }

    // Given: 테스트 서버 시작
    app := setupTestServer()

    // When: APT 패키지 요청
    resp, err := http.Get("http://localhost:8080/proxy/apt/dists/focal/Release")
    require.NoError(t, err)
    defer resp.Body.Close()

    // Then
    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "MISS", resp.Header.Get("X-Cache-Status"))

    body, _ := io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "Origin: Ubuntu")

    // 두 번째 요청 - 캐시 히트
    resp2, _ := http.Get("http://localhost:8080/proxy/apt/dists/focal/Release")
    assert.Equal(t, "HIT", resp2.Header.Get("X-Cache-Status"))
}
```

### 4.2 부하 테스트
```go
// test/load/load_test.go
package load

import (
    "testing"
    "sync"
    "time"
)

func TestConcurrentRequests(t *testing.T) {
    if testing.Short() {
        t.Skip("부하 테스트는 -short 플래그에서 스킵")
    }

    const (
        concurrency = 100
        requests    = 1000
    )

    app := setupTestServer()

    var wg sync.WaitGroup
    errors := make(chan error, requests)
    durations := make(chan time.Duration, requests)

    start := time.Now()

    // 동시 요청 실행
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < requests/concurrency; j++ {
                reqStart := time.Now()
                resp, err := http.Get("http://localhost:8080/proxy/apt/test.deb")
                duration := time.Since(reqStart)

                if err != nil {
                    errors <- err
                } else {
                    resp.Body.Close()
                    durations <- duration
                }
            }
        }()
    }

    wg.Wait()
    close(errors)
    close(durations)

    // 결과 분석
    var errorCount int
    var totalDuration time.Duration
    var requestCount int

    for err := range errors {
        if err != nil {
            errorCount++
        }
    }

    for d := range durations {
        totalDuration += d
        requestCount++
    }

    avgDuration := totalDuration / time.Duration(requestCount)
    totalTime := time.Since(start)
    rps := float64(requests) / totalTime.Seconds()

    // 성능 기준 검증
    assert.Less(t, errorCount, requests/100) // 1% 미만 에러율
    assert.Less(t, avgDuration, 100*time.Millisecond) // 평균 100ms 미만
    assert.Greater(t, rps, 100.0) // 100 RPS 이상

    t.Logf("Results: RPS=%.2f, Avg Duration=%v, Errors=%d",
        rps, avgDuration, errorCount)
}
```

## 5. Mock 생성 및 관리

### 5.1 Mock 인터페이스 생성
```bash
# Makefile에 추가
generate-mocks:
	@echo "Generating mocks..."
	@go install github.com/vektra/mockery/v2@latest
	@mockery --all --dir internal --output test/mocks --outpkg mocks
	@mockery --all --dir cache --output test/mocks --outpkg mocks
	@mockery --all --dir handlers --output test/mocks --outpkg mocks
```

### 5.2 Mock 사용 예시
```go
// test/mocks/mock_cache.go (자동 생성됨)
type MockCache struct {
    mock.Mock
}

func (m *MockCache) Get(key string) ([]byte, error) {
    args := m.Called(key)
    return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCache) Set(key string, value []byte) error {
    args := m.Called(key, value)
    return args.Error(0)
}
```

## 6. 테스트 자동화

### 6.1 CI/CD 파이프라인
```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      redis:
        image: redis:alpine
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.23'

    - name: Install dependencies
      run: make deps

    - name: Run linter
      run: make lint

    - name: Run unit tests
      run: make test-coverage

    - name: Run integration tests
      run: make test-integration

    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

    - name: Check coverage threshold
      run: |
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$coverage < 70" | bc -l) )); then
          echo "Coverage $coverage% is below 70%"
          exit 1
        fi
```

### 6.2 Pre-commit Hook
```bash
# .git/hooks/pre-commit
#!/bin/bash

echo "Running tests before commit..."

# 린트 검사
make lint || exit 1

# 단위 테스트
make test || exit 1

# 커버리지 확인
coverage=$(go test -cover ./... | grep -o '[0-9]*\.[0-9]*%' | sed 's/%//' | awk '{sum+=$1; count++} END {print sum/count}')
if (( $(echo "$coverage < 60" | bc -l) )); then
    echo "Average coverage $coverage% is below 60%"
    exit 1
fi

echo "All tests passed!"
```

## 7. 실행 계획

### Phase 1: 테스트 인프라 구축 (1주)
- [ ] Mock 생성 자동화 설정
- [ ] 테스트 헬퍼 함수 작성
- [ ] CI/CD 파이프라인 구성

### Phase 2: 핸들러 테스트 (2주)
- [ ] APT 핸들러 테스트 (목표: 80%)
- [ ] Maven 핸들러 테스트 (목표: 80%)
- [ ] NPM 핸들러 테스트 (목표: 80%)

### Phase 3: 서비스 레이어 테스트 (1주)
- [ ] 캐시 서비스 테스트 (목표: 90%)
- [ ] 프록시 서비스 테스트 (목표: 80%)
- [ ] 인증 서비스 테스트 (목표: 85%)

### Phase 4: 통합 테스트 (1주)
- [ ] API 엔드포인트 테스트
- [ ] 시나리오 기반 테스트
- [ ] 부하 테스트

### Phase 5: 지속적 개선 (지속)
- [ ] 테스트 커버리지 모니터링
- [ ] 실패 테스트 분석 및 개선
- [ ] 테스트 성능 최적화

## 8. 성공 지표

### 정량적 지표
- 전체 커버리지: 23% → 70%+
- 핸들러 커버리지: 0% → 80%+
- 테스트 실행 시간: < 5분
- CI 빌드 성공률: > 95%

### 정성적 지표
- 버그 발견 시점: 프로덕션 → 개발
- 리팩토링 신뢰도: 높음
- 신규 개발자 온보딩: 2주 → 3일
