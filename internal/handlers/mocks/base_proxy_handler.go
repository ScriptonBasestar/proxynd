package mocks

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
)

// MockBaseProxyHandler BaseProxyHandler 인터페이스의 모의 구현
type MockBaseProxyHandler struct {
	mock.Mock
}

// Type 프록시 타입 반환
func (m *MockBaseProxyHandler) Type() string {
	args := m.Called()
	return args.String(0)
}

// IsEnabled 활성화 상태 확인
func (m *MockBaseProxyHandler) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

// GenerateCacheKey 캐시 키 생성
func (m *MockBaseProxyHandler) GenerateCacheKey(c *fiber.Ctx) string {
	args := m.Called(c)
	return args.String(0)
}

// BuildUpstreamURL 업스트림 URL 구성
func (m *MockBaseProxyHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	args := m.Called(c)
	return args.String(0), args.Error(1)
}

// TransformRequest 업스트림 요청 변환
func (m *MockBaseProxyHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	args := m.Called(c, upstreamReq)
	return args.Error(0)
}

// TransformResponse 응답 변환
func (m *MockBaseProxyHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	args := m.Called(resp, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// ShouldCache 캐시 정책 결정
func (m *MockBaseProxyHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	args := m.Called(c, statusCode)
	return args.Bool(0)
}

// GetCacheTTL 캐시 TTL 반환
func (m *MockBaseProxyHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	args := m.Called(c)
	return args.Get(0).(time.Duration)
}

// HandleError 에러 처리
func (m *MockBaseProxyHandler) HandleError(err error, c *fiber.Ctx) error {
	args := m.Called(err, c)
	return args.Error(0)
}