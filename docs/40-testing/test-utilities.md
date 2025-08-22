# ProxyND Test Utilities

이 패키지는 ProxyND 프로젝트의 테스트를 위한 유틸리티 모음입니다.

## 구성 요소

### 1. Fixtures (`fixtures.go`)

사전 구성된 테스트 데이터를 제공합니다.

```go
fixtures := testutil.NewFixtures()

// 설정 데이터
config := fixtures.ValidGlobalConfig()
aptConfig := fixtures.ValidAPTConfig()

// 요청/응답 데이터
req := fixtures.ProxyRequest("GET", "/test")
resp := fixtures.ProxyResponse(200, "body")

// 샘플 데이터
pomXML := fixtures.SampleMavenPOM()
packageJSON := fixtures.SampleNPMPackageJSON()
```

### 2. Factories (`factories.go`)

테스트 객체를 기본값으로 생성합니다.

```go
factory := testutil.NewFactory(t)

// 서비스 생성
configService := factory.ConfigService()
proxyService := factory.ProxyService("apt")
cacheAdapter := factory.CacheAdapter()

// 설정 생성
config := factory.GlobalConfig()
```

### 3. Builders (`builders.go`)

커스텀 설정으로 객체를 구성합니다.

```go
// 체이닝을 통한 설정 구성
config := testutil.NewGlobalConfigBuilder().
    WithStorageDir("/custom/storage").
    WithCacheDir("/custom/cache").
    WithCacheTTL(7200).
    Build()

// APT 설정 빌더
aptConfig := testutil.NewAPTConfigBuilder().
    WithUpstreamURLs("http://mirror1.com", "http://mirror2.com").
    WithCacheEnabled(true).
    Build()
```

### 4. Helpers (`helpers.go`)

일반적인 테스트 헬퍼 함수들입니다.

```go
// 임시 디렉토리 생성
dir := testutil.CreateTempDir(t, "test-")

// 테스트 파일 생성
path := testutil.CreateTestFile(t, dir, "config.yaml", content)

// Mock 서버 생성
server := testutil.CreateMockServer(handler)

// Reader 생성
reader := testutil.ReadCloserFromString("test content")
```

### 5. Assertions (`assertions.go`)

커스텀 assertion 헬퍼들입니다.

```go
assertions := testutil.NewAssertions(t)

// Reader 내용 검증
assertions.AssertReaderContent(reader, "expected content")
assertions.AssertReaderContains(reader, "substring")

// 헤더 검증
assertions.AssertHeadersEqual(expected, actual)

// 캐시 상태 검증
assertions.AssertCacheHit(resp.Cached)
assertions.AssertCacheMiss(resp.Cached)

// 에러 검증
assertions.AssertErrorContains(err, "expected message")
```

### 6. Matchers (`matchers.go`)

Mock 인자 매칭을 위한 커스텀 매처들입니다.

```go
matchers := testutil.NewMatchers()

// 기본 매처
mock.EXPECT().Method(
    matchers.AnyContext(),
    matchers.AnyString(),
    matchers.AnyReader(),
)

// 문자열 매처
mock.EXPECT().Method(
    matchers.StringContaining("substring"),
    matchers.StringHasPrefix("prefix"),
    matchers.StringHasSuffix(".txt"),
)

// Map 매처
mock.EXPECT().Method(
    matchers.MapContaining("key1", "key2"),
    matchers.MapWithValue("key", "value"),
)

// URL/Path 매처
mock.EXPECT().Method(
    matchers.URLContaining("example.com"),
    matchers.PathStartingWith("/api/"),
)
```

## 사용 예제

### 단위 테스트

```go
func TestMyService(t *testing.T) {
    // 테스트 유틸리티 생성
    factory := testutil.NewFactory(t)
    fixtures := testutil.NewFixtures()
    assertions := testutil.NewAssertions(t)

    // 서비스 생성
    service := factory.ProxyService("npm")

    // 테스트 데이터 사용
    req := fixtures.ProxyRequest("GET", "/express")

    // 테스트 실행
    resp, err := service.HandleRequest(ctx, req)

    // 검증
    assertions.AssertNoError(err)
    assertions.AssertStatusCode(200, resp.StatusCode)
    assertions.AssertCacheHit(resp.Cached)
}
```

### 통합 테스트

```go
func TestIntegration(t *testing.T) {
    // Factory로 전체 환경 구성
    factory := testutil.NewFactory(t)

    // 실제 서비스 생성
    configService := factory.ConfigService()
    cacheService := factory.CacheAdapter()
    upstreamClient := factory.UpstreamClient()

    // 프록시 서비스 생성
    proxyFactory := proxy.NewFactory(configService, cacheService, upstreamClient)
    aptProxy, _ := proxyFactory.CreateProxy("apt")

    // 테스트 실행...
}
```

### Table-Driven 테스트

```go
func TestTableDriven(t *testing.T) {
    fixtures := testutil.NewFixtures()

    tests := []struct {
        name     string
        config   func() interface{}
        expected string
    }{
        {
            name: "default config",
            config: func() interface{} {
                return fixtures.ValidGlobalConfig()
            },
            expected: "/tmp/test-storage",
        },
        {
            name: "custom config",
            config: func() interface{} {
                return testutil.NewGlobalConfigBuilder().
                    WithStorageDir("/custom").
                    Build()
            },
            expected: "/custom",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            config := tt.config()
            // 테스트 로직...
        })
    }
}
```

## 베스트 프랙티스

1. **Factory 사용**: 복잡한 객체 생성 시 Factory 사용
2. **Fixtures 활용**: 반복되는 테스트 데이터는 Fixtures로 관리
3. **Builder 패턴**: 다양한 설정 조합이 필요한 경우 Builder 사용
4. **커스텀 Assertion**: 도메인 특화 검증 로직은 Assertions에 추가
5. **Mock Matcher**: 복잡한 인자 매칭은 Matchers 활용

## 추가 개발

새로운 테스트 유틸리티가 필요한 경우:

1. 적절한 파일에 함수/메서드 추가
2. 예제 테스트 작성
3. 이 README 업데이트
