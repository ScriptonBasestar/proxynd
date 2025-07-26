---
phase: 4
order: 8
source_plan: /docs/refactoring/04-test-coverage.md
priority: high
tags: [testing, unit-tests, handlers, coverage]
status: suspended
reason: 실제 테스트 코드 작성이 필요한 개발 작업
---

# 📌 작업: 핸들러 단위 테스트 작성

## 개요
현재 0% 커버리지인 핸들러 패키지를 80% 커버리지로 향상시키기 위한 단위 테스트를 작성합니다.

## 현재 테스트 현황
- **핸들러 패키지 커버리지: 0%** (가장 심각)
- 전체 평균 커버리지: 23%
- 목표: 핸들러 80%+, 전체 70%+

## 구현 내용

### 1. 테스트 인프라 구축
```bash
# Mock 생성 도구 설치
go install github.com/vektra/mockery/v2@latest

# Mock 생성 (Makefile 추가)
generate-mocks:
	@echo "Generating mocks..."
	@mockery --all --dir internal --output test/mocks --outpkg mocks
	@mockery --all --dir cache --output test/mocks --outpkg mocks
	@mockery --all --dir handlers --output test/mocks --outpkg mocks
```

### 2. Mock 구조체 정의
```go
// test/mocks/mock_container.go
package mocks

import (
    "github.com/stretchr/testify/mock"
    "proxynd/configs"
)

type MockContainer struct {
    mock.Mock
    config *configs.Config
}

func NewMockContainer() *MockContainer {
    return &MockContainer{
        config: &configs.Config{},
    }
}

func (m *MockContainer) Config() *configs.Config {
    return m.config
}

func (m *MockContainer) Get(key string) interface{} {
    args := m.Called(key)
    return args.Get(0)
}

func (m *MockContainer) SetConfig(config *configs.Config) {
    m.config = config
}

// test/mocks/mock_cache.go
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

func (m *MockCache) SetWithTTL(key string, value []byte, ttl time.Duration) error {
    args := m.Called(key, value, ttl)
    return args.Error(0)
}
```

### 3. APT 핸들러 테스트
```go
// handlers/proxy/apt_handler_test.go
package proxy

import (
    "bytes"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

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
        expectedBody   string
    }{
        {
            name: "캐시 히트 - Release 파일",
            path: "/proxy/apt/dists/focal/Release",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.SetConfig(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: true,
                        Mirrors: []string{"http://mirror.example.com"},
                    },
                })
                cache.On("Get", "apt:dists_focal_Release").
                    Return([]byte("Origin: Ubuntu\nSuite: focal"), nil)
            },
            expectedStatus: 200,
            expectedCache:  "HIT",
            expectedBody:   "Origin: Ubuntu\nSuite: focal",
        },
        {
            name: "캐시 미스 - 패키지 파일",
            path: "/proxy/apt/pool/main/v/vim/vim_8.2.deb",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.SetConfig(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: true,
                        Mirrors: []string{"http://mirror.example.com"},
                    },
                })
                cache.On("Get", "apt:pool_main_v_vim_vim_8.2.deb").
                    Return([]byte(nil), errors.New("cache miss"))
                cache.On("SetWithTTL", "apt:pool_main_v_vim_vim_8.2.deb",
                    mock.AnythingOfType("[]uint8"), 7*24*time.Hour).
                    Return(nil)
            },
            expectedStatus: 200,
            expectedCache:  "MISS",
        },
        {
            name: "프록시 비활성화",
            path: "/proxy/apt/any/path",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.SetConfig(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: false,
                    },
                })
            },
            expectedStatus: 404,
        },
        {
            name: "미러 없음",
            path: "/proxy/apt/any/path",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.SetConfig(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: true,
                        Mirrors: []string{},
                    },
                })
                cache.On("Get", mock.AnythingOfType("string")).
                    Return([]byte(nil), errors.New("cache miss"))
            },
            expectedStatus: 500,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Given
            mockContainer := mocks.NewMockContainer()
            mockCache := new(mocks.MockCache)

            tt.setupMocks(mockContainer, mockCache)

            // HTTP 클라이언트 모킹 (외부 서비스)
            if tt.expectedCache == "MISS" {
                mockContainer.On("Get", "httpClient").Return(&http.Client{
                    Transport: &MockTransport{
                        Response: &http.Response{
                            StatusCode: 200,
                            Body:       ioutil.NopCloser(bytes.NewReader([]byte("package data"))),
                        },
                    },
                })
            }

            mockContainer.On("Get", "cache").Return(mockCache)

            // 핸들러 생성
            handler := NewAPTHandler(mockContainer)

            // Fiber 앱 설정
            app := fiber.New()
            app.Get("/proxy/apt/*", handler.Handle)

            // When
            req := httptest.NewRequest("GET", tt.path, nil)
            resp, err := app.Test(req, -1)

            // Then
            require.NoError(t, err)
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)

            if tt.expectedCache != "" {
                assert.Equal(t, tt.expectedCache, resp.Header.Get("X-Cache-Status"))
            }

            if tt.expectedBody != "" {
                body, _ := ioutil.ReadAll(resp.Body)
                assert.Equal(t, tt.expectedBody, string(body))
            }

            mockContainer.AssertExpectations(t)
            mockCache.AssertExpectations(t)
        })
    }
}

// HTTP Transport Mock
type MockTransport struct {
    Response *http.Response
    Error    error
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    if m.Error != nil {
        return nil, m.Error
    }
    return m.Response, nil
}
```

### 4. 캐시 정책 테스트
```go
func TestAPTHandler_CachePolicy(t *testing.T) {
    tests := []struct {
        name        string
        path        string
        statusCode  int
        shouldCache bool
        expectedTTL time.Duration
    }{
        {
            name:        "Release 파일 캐시",
            path:        "/proxy/apt/dists/focal/Release",
            statusCode:  200,
            shouldCache: true,
            expectedTTL: 10 * time.Minute,
        },
        {
            name:        "패키지 파일 캐시",
            path:        "/proxy/apt/pool/main/v/vim/vim_8.2.deb",
            statusCode:  200,
            shouldCache: true,
            expectedTTL: 7 * 24 * time.Hour,
        },
        {
            name:        "에러 응답 캐시 안함",
            path:        "/proxy/apt/any/path",
            statusCode:  404,
            shouldCache: false,
        },
        {
            name:        "Packages 파일 캐시",
            path:        "/proxy/apt/dists/focal/main/binary-amd64/Packages",
            statusCode:  200,
            shouldCache: true,
            expectedTTL: 10 * time.Minute,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewAPTHandlerV2()

            // Fiber 컨텍스트 모킹
            app := fiber.New()
            c := app.AcquireCtx(&fasthttp.RequestCtx{})
            defer app.ReleaseCtx(c)

            c.Request().SetRequestURI(tt.path)

            // 캐시 정책 테스트
            shouldCache := handler.ShouldCache(c, tt.statusCode)
            assert.Equal(t, tt.shouldCache, shouldCache)

            if tt.shouldCache {
                ttl := handler.GetCacheTTL(c)
                assert.Equal(t, tt.expectedTTL, ttl)
            }
        })
    }
}
```

### 5. 에러 처리 테스트
```go
func TestAPTHandler_ErrorHandling(t *testing.T) {
    tests := []struct {
        name           string
        setupMocks     func(*mocks.MockContainer, *mocks.MockCache)
        expectedStatus int
        expectedError  string
    }{
        {
            name: "업스트림 서버 연결 실패",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.SetConfig(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: true,
                        Mirrors: []string{"http://invalid.mirror"},
                    },
                })
                cache.On("Get", mock.AnythingOfType("string")).
                    Return([]byte(nil), errors.New("cache miss"))
                mc.On("Get", "httpClient").Return(&http.Client{
                    Transport: &MockTransport{
                        Error: errors.New("connection refused"),
                    },
                })
            },
            expectedStatus: 500,
            expectedError:  "PROXY003",
        },
        {
            name: "캐시 저장 실패 (계속 진행)",
            setupMocks: func(mc *mocks.MockContainer, cache *mocks.MockCache) {
                mc.SetConfig(&configs.Config{
                    APTProxy: configs.APTProxyConfig{
                        Enabled: true,
                        Mirrors: []string{"http://mirror.example.com"},
                    },
                })
                cache.On("Get", mock.AnythingOfType("string")).
                    Return([]byte(nil), errors.New("cache miss"))
                cache.On("SetWithTTL", mock.AnythingOfType("string"),
                    mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Duration")).
                    Return(errors.New("cache write failed"))
                mc.On("Get", "httpClient").Return(&http.Client{
                    Transport: &MockTransport{
                        Response: &http.Response{
                            StatusCode: 200,
                            Body:       ioutil.NopCloser(bytes.NewReader([]byte("data"))),
                        },
                    },
                })
            },
            expectedStatus: 200, // 캐시 실패여도 응답은 성공
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 테스트 구현
        })
    }
}
```

### 6. 통합 테스트
```go
// handlers/proxy/integration_test.go
func TestAPTProxy_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("통합 테스트는 -short 플래그에서 스킵")
    }

    // 실제 서버 시작
    container := setupTestContainer()
    app := setupTestApp(container)

    // 테스트 서버 시작
    server := httptest.NewServer(app)
    defer server.Close()

    // APT 요청 테스트
    resp, err := http.Get(server.URL + "/proxy/apt/dists/focal/Release")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "MISS", resp.Header.Get("X-Cache-Status"))

    // 두 번째 요청 - 캐시 히트
    resp2, err := http.Get(server.URL + "/proxy/apt/dists/focal/Release")
    require.NoError(t, err)
    defer resp2.Body.Close()

    assert.Equal(t, 200, resp2.StatusCode)
    assert.Equal(t, "HIT", resp2.Header.Get("X-Cache-Status"))
}
```

## 실행 명령어
```bash
# Mock 생성
make generate-mocks

# 단위 테스트 실행
go test -v ./handlers/proxy/...

# 커버리지 측정
go test -cover -coverprofile=coverage.out ./handlers/proxy/...
go tool cover -html=coverage.out

# 벤치마크 테스트
go test -bench=. -benchmem ./handlers/proxy/...
```

## 검증 방법
1. 테스트 커버리지 80% 달성
2. 모든 테스트 케이스 통과
3. 성능 회귀 없음 확인
4. 코드 품질 향상 확인

## 완료 조건
- [x] Mock 인프라 구축
- [x] APT 핸들러 단위 테스트 (80%+ 커버리지)
- [ ] Maven 핸들러 단위 테스트 (80%+ 커버리지)
- [ ] 에러 처리 시나리오 테스트
- [ ] 캐시 정책 테스트
- [ ] 통합 테스트 시나리오
- [ ] 성능 벤치마크 테스트
- [ ] CI/CD 파이프라인 통합
