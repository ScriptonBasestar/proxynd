package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"proxynd/internal/services/proxy"
)

// MockUpstreamClient is a mock implementation of proxy.UpstreamClient interface
type MockUpstreamClient struct {
	mock.Mock
}

// NewMockUpstreamClient creates a new MockUpstreamClient
func NewMockUpstreamClient() *MockUpstreamClient {
	return &MockUpstreamClient{}
}

// Fetch retrieves content from upstream
func (m *MockUpstreamClient) Fetch(ctx context.Context, url string,
	headers map[string]string,
) (*proxy.ProxyResponse, error) {
	args := m.Called(ctx, url, headers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proxy.ProxyResponse), args.Error(1)
}
