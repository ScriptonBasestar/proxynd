---
phase: 4
order: 9
source_plan: /docs/refactoring/04-test-coverage.md
priority: medium
tags: [testing, integration-tests, e2e, performance]
---

# 📌 작업: 통합 테스트 및 E2E 테스트 구현

## 개요
API 엔드포인트 통합 테스트와 실제 시나리오 기반 E2E 테스트를 구현하여 시스템 안정성을 확보합니다.

## 구현 내용

### 1. 통합 테스트 인프라
```go
// test/integration/setup.go
package integration

import (
    "fmt"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
    "proxynd/configs"
    "proxynd/cache"
)

type TestServer struct {
    App       *fiber.App
    Server    *httptest.Server
    Container *app.Container
}

func SetupTestServer(t *testing.T) *TestServer {
    // 테스트용 설정
    testConfig := &configs.Config{
        APTProxy: configs.APTProxyConfig{
            Enabled: true,
            Mirrors: []string{"http://archive.ubuntu.com/ubuntu"},
        },
        MavenProxy: configs.MavenProxyConfig{
            Enabled:    true,
            Repository: "https://repo1.maven.org/maven2",
        },
        Cache: configs.CacheConfig{
            Type: "memory",
            Size: 100 * 1024 * 1024, // 100MB
        },
    }

    // 컨테이너 설정
    container := app.NewContainer()
    container.Register("config", func() (interface{}, error) {
        return testConfig, nil
    })

    // 캐시 설정
    memoryCache := cache.NewMemoryCache(testConfig.Cache.Size)
    container.Register("cache", func() (interface{}, error) {
        return memoryCache, nil
    })

    // HTTP 클라이언트 설정
    httpClient := &http.Client{
        Timeout: 10 * time.Second,
    }
    container.Register("httpClient", func() (interface{}, error) {
        return httpClient, nil
    })

    // 앱 설정
    app := fiber.New(fiber.Config{
        DisableStartupMessage: true,
    })

    setupRoutes(app, container)

    // 테스트 서버 시작
    server := httptest.NewServer(app)

    return &TestServer{
        App:       app,
        Server:    server,
        Container: container,
    }
}

func (ts *TestServer) Close() {
    ts.Server.Close()
}

func (ts *TestServer) URL() string {
    return ts.Server.URL
}
```

### 2. APT 프록시 통합 테스트
```go
// test/integration/apt_integration_test.go
package integration

import (
    "fmt"
    "io"
    "net/http"
    "strings"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAPTProxy_FullFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("통합 테스트는 -short 플래그에서 스킵")
    }

    // Given
    server := SetupTestServer(t)
    defer server.Close()

    tests := []struct {
        name           string
        path           string
        expectedStatus int
        expectCached   bool
        contentCheck   func(string) bool
    }{
        {
            name:           "Release 파일 요청",
            path:           "/proxy/apt/dists/focal/Release",
            expectedStatus: 200,
            expectCached:   true,
            contentCheck: func(content string) bool {
                return strings.Contains(content, "Origin: Ubuntu") &&
                       strings.Contains(content, "Suite: focal")
            },
        },
        {
            name:           "Packages 파일 요청",
            path:           "/proxy/apt/dists/focal/main/binary-amd64/Packages",
            expectedStatus: 200,
            expectCached:   true,
            contentCheck: func(content string) bool {
                return strings.Contains(content, "Package:") &&
                       strings.Contains(content, "Version:")
            },
        },
        {
            name:           "존재하지 않는 패키지",
            path:           "/proxy/apt/pool/main/nonexistent/package.deb",
            expectedStatus: 404,
            expectCached:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // When: 첫 번째 요청
            url := server.URL() + tt.path
            resp, err := http.Get(url)
            require.NoError(t, err)
            defer resp.Body.Close()

            // Then: 응답 검증
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)

            if tt.expectedStatus == 200 {
                assert.Equal(t, "MISS", resp.Header.Get("X-Cache-Status"))
                assert.Equal(t, "apt", resp.Header.Get("X-Proxy-Type"))

                body, err := io.ReadAll(resp.Body)
                require.NoError(t, err)

                if tt.contentCheck != nil {
                    assert.True(t, tt.contentCheck(string(body)))
                }

                // 캐시 확인을 위한 두 번째 요청
                if tt.expectCached {
                    resp2, err := http.Get(url)
                    require.NoError(t, err)
                    defer resp2.Body.Close()

                    assert.Equal(t, 200, resp2.StatusCode)
                    assert.Equal(t, "HIT", resp2.Header.Get("X-Cache-Status"))
                }
            }
        })
    }
}

func TestAPTProxy_CacheExpiration(t *testing.T) {
    if testing.Short() {
        t.Skip("통합 테스트는 -short 플래그에서 스킵")
    }

    // Given
    server := SetupTestServer(t)
    defer server.Close()

    // 짧은 TTL 설정을 위한 테스트 설정 수정
    // (실제 구현에서는 테스트용 설정 파일 사용)

    // When: 첫 번째 요청
    url := server.URL() + "/proxy/apt/dists/focal/Release"
    resp1, err := http.Get(url)
    require.NoError(t, err)
    defer resp1.Body.Close()

    assert.Equal(t, 200, resp1.StatusCode)
    assert.Equal(t, "MISS", resp1.Header.Get("X-Cache-Status"))

    // 캐시 확인
    resp2, err := http.Get(url)
    require.NoError(t, err)
    defer resp2.Body.Close()

    assert.Equal(t, 200, resp2.StatusCode)
    assert.Equal(t, "HIT", resp2.Header.Get("X-Cache-Status"))

    // TTL 만료 대기 (테스트용으로 짧게 설정)
    time.Sleep(2 * time.Second)

    // 만료 후 요청
    resp3, err := http.Get(url)
    require.NoError(t, err)
    defer resp3.Body.Close()

    assert.Equal(t, 200, resp3.StatusCode)
    assert.Equal(t, "MISS", resp3.Header.Get("X-Cache-Status"))
}
```

### 3. Maven 프록시 통합 테스트
```go
// test/integration/maven_integration_test.go
package integration

import (
    "fmt"
    "io"
    "net/http"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMavenProxy_ArtifactDownload(t *testing.T) {
    if testing.Short() {
        t.Skip("통합 테스트는 -short 플래그에서 스킵")
    }

    // Given
    server := SetupTestServer(t)
    defer server.Close()

    tests := []struct {
        name         string
        path         string
        expectedSize int64 // 대략적인 크기 검증
        contentType  string
    }{
        {
            name:        "Spring Core JAR",
            path:        "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
            expectedSize: 1000000, // 1MB 이상
            contentType: "application/java-archive",
        },
        {
            name:        "POM 파일",
            path:        "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom",
            expectedSize: 1000, // 1KB 이상
            contentType: "application/xml",
        },
        {
            name:        "SHA1 체크섬",
            path:        "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar.sha1",
            expectedSize: 40, // SHA1 해시 길이
            contentType: "text/plain",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // When
            url := server.URL() + tt.path
            resp, err := http.Get(url)
            require.NoError(t, err)
            defer resp.Body.Close()

            // Then
            assert.Equal(t, 200, resp.StatusCode)
            assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

            body, err := io.ReadAll(resp.Body)
            require.NoError(t, err)

            assert.True(t, int64(len(body)) >= tt.expectedSize)

            if tt.contentType != "" {
                assert.Contains(t, resp.Header.Get("Content-Type"), tt.contentType)
            }
        })
    }
}

func TestMavenProxy_ChecksumValidation(t *testing.T) {
    if testing.Short() {
        t.Skip("통합 테스트는 -short 플래그에서 스킵")
    }

    // Given
    server := SetupTestServer(t)
    defer server.Close()

    jarPath := "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar"
    sha1Path := "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar.sha1"

    // When: JAR 파일 다운로드
    jarResp, err := http.Get(server.URL() + jarPath)
    require.NoError(t, err)
    defer jarResp.Body.Close()

    jarData, err := io.ReadAll(jarResp.Body)
    require.NoError(t, err)

    // SHA1 체크섬 다운로드
    sha1Resp, err := http.Get(server.URL() + sha1Path)
    require.NoError(t, err)
    defer sha1Resp.Body.Close()

    sha1Data, err := io.ReadAll(sha1Resp.Body)
    require.NoError(t, err)

    // Then: 체크섬 검증
    expectedSha1 := strings.TrimSpace(string(sha1Data))
    actualSha1 := calculateSHA1(jarData)

    assert.Equal(t, expectedSha1, actualSha1)
}
```

### 4. 부하 테스트
```go
// test/integration/load_test.go
package integration

import (
    "fmt"
    "net/http"
    "sync"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestConcurrentRequests(t *testing.T) {
    if testing.Short() {
        t.Skip("부하 테스트는 -short 플래그에서 스킵")
    }

    // Given
    server := SetupTestServer(t)
    defer server.Close()

    const (
        concurrency = 50
        requests    = 500
        requestsPerGoroutine = requests / concurrency
    )

    var wg sync.WaitGroup
    results := make(chan TestResult, requests)

    // When: 동시 요청 실행
    start := time.Now()

    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(goroutineID int) {
            defer wg.Done()

            for j := 0; j < requestsPerGoroutine; j++ {
                result := performRequest(server.URL() + "/proxy/apt/dists/focal/Release")
                results <- result
            }
        }(i)
    }

    wg.Wait()
    close(results)

    totalTime := time.Since(start)

    // Then: 결과 분석
    var successCount, errorCount int
    var totalDuration time.Duration

    for result := range results {
        if result.Error != nil {
            errorCount++
            t.Logf("Request failed: %v", result.Error)
        } else {
            successCount++
            totalDuration += result.Duration
        }
    }

    // 성능 기준 검증
    errorRate := float64(errorCount) / float64(requests) * 100
    avgDuration := totalDuration / time.Duration(successCount)
    rps := float64(requests) / totalTime.Seconds()

    t.Logf("Results: Total=%d, Success=%d, Error=%d (%.2f%%)",
        requests, successCount, errorCount, errorRate)
    t.Logf("Performance: RPS=%.2f, Avg Duration=%v, Total Time=%v",
        rps, avgDuration, totalTime)

    // 성능 기준
    assert.Less(t, errorRate, 1.0)  // 1% 미만 에러율
    assert.Less(t, avgDuration, 200*time.Millisecond) // 평균 200ms 미만
    assert.Greater(t, rps, 100.0)   // 100 RPS 이상
}

type TestResult struct {
    StatusCode int
    Duration   time.Duration
    Error      error
}

func performRequest(url string) TestResult {
    start := time.Now()

    resp, err := http.Get(url)
    if err != nil {
        return TestResult{
            Duration: time.Since(start),
            Error:    err,
        }
    }
    defer resp.Body.Close()

    return TestResult{
        StatusCode: resp.StatusCode,
        Duration:   time.Since(start),
    }
}
```

### 5. 메모리 누수 테스트
```go
// test/integration/memory_test.go
package integration

import (
    "runtime"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestMemoryUsage(t *testing.T) {
    if testing.Short() {
        t.Skip("메모리 테스트는 -short 플래그에서 스킵")
    }

    // Given
    server := SetupTestServer(t)
    defer server.Close()

    // 초기 메모리 사용량 측정
    runtime.GC()
    time.Sleep(100 * time.Millisecond)

    var initialStats runtime.MemStats
    runtime.ReadMemStats(&initialStats)

    // When: 많은 요청 수행
    for i := 0; i < 1000; i++ {
        resp, err := http.Get(server.URL() + "/proxy/apt/dists/focal/Release")
        if err != nil {
            t.Fatalf("Request failed: %v", err)
        }
        resp.Body.Close()
    }

    // 메모리 정리
    runtime.GC()
    time.Sleep(100 * time.Millisecond)

    // 최종 메모리 사용량 측정
    var finalStats runtime.MemStats
    runtime.ReadMemStats(&finalStats)

    // Then: 메모리 사용량 검증
    memoryIncrease := finalStats.HeapAlloc - initialStats.HeapAlloc

    t.Logf("Initial memory: %d bytes", initialStats.HeapAlloc)
    t.Logf("Final memory: %d bytes", finalStats.HeapAlloc)
    t.Logf("Memory increase: %d bytes", memoryIncrease)

    // 메모리 증가가 10MB 미만이어야 함
    assert.Less(t, memoryIncrease, uint64(10*1024*1024))
}
```

### 6. 장애 복구 테스트
```go
// test/integration/resilience_test.go
package integration

import (
    "context"
    "net/http"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestUpstreamFailureRecovery(t *testing.T) {
    if testing.Short() {
        t.Skip("장애 복구 테스트는 -short 플래그에서 스킵")
    }

    // Given
    server := SetupTestServer(t)
    defer server.Close()

    // 정상 요청 확인
    resp, err := http.Get(server.URL() + "/proxy/apt/dists/focal/Release")
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    resp.Body.Close()

    // 업스트림 서버 연결 불가 시뮬레이션
    // (실제 구현에서는 테스트용 설정으로 잘못된 미러 URL 사용)

    // When: 장애 상황에서 요청
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    req, _ := http.NewRequestWithContext(ctx, "GET",
        server.URL()+"/proxy/apt/dists/focal/Release", nil)

    resp, err = http.DefaultClient.Do(req)
    if err != nil {
        t.Logf("Expected error during failure: %v", err)
    } else {
        // 캐시된 응답 반환 가능
        assert.Equal(t, 200, resp.StatusCode)
        assert.Equal(t, "HIT", resp.Header.Get("X-Cache-Status"))
        resp.Body.Close()
    }
}
```

## 실행 명령어
```bash
# 통합 테스트 실행
go test -v -tags=integration ./test/integration/...

# 부하 테스트 실행
go test -v -run=TestConcurrentRequests ./test/integration/...

# 메모리 테스트 실행
go test -v -run=TestMemoryUsage ./test/integration/...

# 모든 테스트 실행 (단시간 테스트 제외)
go test -v -timeout=30m ./test/integration/...

# 짧은 테스트만 실행
go test -v -short ./test/integration/...
```

## 검증 방법
1. 전체 시나리오 테스트 통과
2. 부하 테스트 성능 기준 달성
3. 메모리 누수 없음 확인
4. 장애 복구 시나리오 검증

## 완료 조건
- [ ] 통합 테스트 인프라 구축
- [ ] APT 프록시 E2E 테스트
- [ ] Maven 프록시 E2E 테스트
- [ ] 부하 테스트 (100 RPS, 1% 에러율)
- [ ] 메모리 누수 테스트
- [ ] 장애 복구 테스트
- [ ] CI/CD 파이프라인 통합
- [ ] 성능 기준 달성 확인
