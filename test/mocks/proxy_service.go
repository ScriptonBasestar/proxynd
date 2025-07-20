package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"proxynd/internal/services/proxy"
)

// MockProxyService is a mock implementation of the proxy.ProxyService interface
type MockProxyService struct {
	mock.Mock
}

// NewMockProxyService creates a new MockProxyService instance
func NewMockProxyService() *MockProxyService {
	return &MockProxyService{}
}

// HandleRequest processes a proxy request
func (m *MockProxyService) HandleRequest(ctx context.Context, req proxy.ProxyRequest) (*proxy.ProxyResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proxy.ProxyResponse), args.Error(1)
}

// ValidateRequest validates the request
func (m *MockProxyService) ValidateRequest(req proxy.ProxyRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

// GetProxyType returns the proxy type
func (m *MockProxyService) GetProxyType() string {
	args := m.Called()
	return args.String(0)
}
