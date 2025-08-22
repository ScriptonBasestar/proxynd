# ProxyND Mocking Guide

## Overview

ProxyND는 인터페이스 기반 설계를 사용하여 테스트 시 Mock 객체를 쉽게 사용할 수 있도록 합니다.
[mockery](https://github.com/vektra/mockery)를 사용하여 자동으로 Mock을 생성합니다.

## Mock 생성

### 자동 생성

```bash
# 모든 Mock 생성
make generate-mocks

# Mock 업데이트 (기존 Mock 삭제 후 재생성)
make update-mocks

# Mock 삭제
make clean-mocks
```

### 수동 생성 (go generate)

특정 패키지의 Mock만 생성하려면:

```bash
# 특정 패키지의 Mock 생성
go generate ./internal/services/config
go generate ./internal/services/proxy
```

## Mock 사용 방법

### 1. 기본 사용법

```go
import (
    "testing"
    "github.com/stretchr/testify/mock"
    proxymocks "proxynd/internal/services/proxy/mocks"
)

func TestMyService(t *testing.T) {
    // Mock 생성
    mockCache := proxymocks.NewMockCacheService(t)

    // 기대값 설정
    mockCache.EXPECT().Get(mock.Anything, "key").Return(nil, false, nil)

    // Mock을 사용한 서비스 테스트
    service := NewMyService(mockCache)
    // ...
}
```

### 2. 복잡한 매칭

```go
// 특정 패턴 매칭
mockCache.EXPECT().Get(
    mock.Anything,
    mock.MatchedBy(func(key string) bool {
        return strings.HasPrefix(key, "cache:")
    }),
).Return(data, true, nil)

// 여러 인자 매칭
mockUpstream.EXPECT().Fetch(
    mock.Anything,
    mock.AnythingOfType("string"),
    mock.AnythingOfType("map[string]string"),
).Return(response, nil)
```

### 3. 동적 반환값

```go
// RunAndReturn을 사용한 동적 반환값
mockUpstream.EXPECT().Fetch(mock.Anything, mock.Anything, mock.Anything).
    RunAndReturn(func(ctx context.Context, url string, headers map[string]string) (*proxy.ProxyResponse, error) {
        if strings.Contains(url, "error") {
            return nil, errors.New("simulated error")
        }
        return &proxy.ProxyResponse{
            Body: io.NopCloser(strings.NewReader("response")),
        }, nil
    })
```

### 4. 호출 횟수 검증

```go
// 정확히 한 번 호출
mockCache.EXPECT().Get(mock.Anything, "key").Return(nil, false, nil).Once()

// N번 호출
mockCache.EXPECT().Put(mock.Anything, mock.Anything, mock.Anything).Times(3)

// 호출되지 않아야 함
mockCache.EXPECT().Delete(mock.Anything, mock.Anything).Maybe()
```

### 5. 호출 순서 검증

```go
// InOrder를 사용한 순서 검증
inOrder := mock.InOrder(
    mockCache.EXPECT().Get(mock.Anything, "key1").Return(nil, false, nil),
    mockCache.EXPECT().Put(mock.Anything, "key1", mock.Anything).Return(nil),
    mockCache.EXPECT().Get(mock.Anything, "key2").Return(nil, false, nil),
)
```

## 주요 Mock 인터페이스

### Service Layer Mocks

- `MockService` (config): 설정 서비스 Mock
- `MockProxyService`: 프록시 서비스 Mock
- `MockCacheService`: 캐시 서비스 Mock
- `MockUpstreamClient`: 업스트림 클라이언트 Mock

### Repository Layer Mocks

- `MockRepository` (cache): 캐시 리포지토리 Mock
- `MockRepository` (config): 설정 리포지토리 Mock

### Infrastructure Mocks

- `MockHealthChecker`: 헬스 체크 Mock
- `MockWebhookSender`: 웹훅 전송 Mock
- `MockJWTService`: JWT 서비스 Mock

## 테스트 패턴

### 1. 단위 테스트 패턴

```go
func TestService_Method(t *testing.T) {
    tests := []struct {
        name       string
        setupMocks func(*proxymocks.MockCacheService)
        want       string
        wantErr    bool
    }{
        {
            name: "성공 케이스",
            setupMocks: func(m *proxymocks.MockCacheService) {
                m.EXPECT().Get(mock.Anything, "key").Return(data, true, nil)
            },
            want:    "expected",
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockCache := proxymocks.NewMockCacheService(t)
            tt.setupMocks(mockCache)

            service := NewService(mockCache)
            got, err := service.Method()

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

### 2. 통합 테스트에서 부분 Mock

```go
func TestIntegration_PartialMock(t *testing.T) {
    // 실제 구현체와 Mock 혼용
    realConfig := config.NewService("./config")
    mockCache := proxymocks.NewMockCacheService(t)
    mockUpstream := proxymocks.NewMockUpstreamClient(t)

    // 특정 부분만 Mock으로 대체
    service := proxy.NewService(realConfig, mockCache, mockUpstream)

    // 테스트 실행
    // ...
}
```

## Mock 설정 (.mockery.yaml)

프로젝트 루트의 `.mockery.yaml` 파일에서 Mock 생성 설정을 관리합니다:

```yaml
with-expecter: true          # EXPECT() 메서드 생성
testonly: true              # _test.go 파일에서만 사용 가능
filename: "{{.InterfaceName | snakecase}}.go"
mockname: "Mock{{.InterfaceName}}"
```

## 주의사항

1. **Mock은 테스트 코드에서만 사용**: `testonly: true` 설정으로 프로덕션 코드에서 실수로 사용하는 것을 방지

2. **인터페이스 변경 시 Mock 재생성**: 인터페이스가 변경되면 반드시 `make update-mocks` 실행

3. **과도한 Mock 사용 주의**: 너무 많은 Mock은 테스트를 복잡하게 만들 수 있음. 필요한 경우만 사용

4. **Mock 검증**: 모든 EXPECT() 설정은 자동으로 검증됨. 호출되지 않은 기대값은 테스트 실패 원인

## 문제 해결

### Mock 생성 실패

```bash
# mockery 재설치
go install github.com/vektra/mockery/v2@latest

# 캐시 정리 후 재시도
go clean -modcache
make generate-mocks
```

### Import 오류

Mock 파일의 import가 잘못된 경우:

```bash
# Mock 재생성
make update-mocks

# go mod tidy 실행
go mod tidy
```

## 추가 리소스

- [Mockery 공식 문서](https://github.com/vektra/mockery)
- [Testify Mock 문서](https://github.com/stretchr/testify#mock-package)
- [예제 테스트 파일](../internal/services/example/service_with_mocks_test.go)
