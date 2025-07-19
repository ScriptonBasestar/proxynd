package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"proxynd/internal/services/proxy"
)

// MockProxyService는 proxy.ProxyService 인터페이스의 모의 구현
type MockProxyService struct {
	mock.Mock
}

// NewMockProxyService는 새로운 MockProxyService를 생성
func NewMockProxyService() *MockProxyService {
	return &MockProxyService{}
}

// HandleRequest는 프록시 요청을 처리
func (m *MockProxyService) HandleRequest(ctx context.Context, req proxy.ProxyRequest) (*proxy.ProxyResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proxy.ProxyResponse), args.Error(1)
}

// ValidateRequest는 요청의 유효성을 검사
func (m *MockProxyService) ValidateRequest(req proxy.ProxyRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

// GetProxyType은 프록시 타입을 반환
func (m *MockProxyService) GetProxyType() string {
	args := m.Called()
	return args.String(0)
}