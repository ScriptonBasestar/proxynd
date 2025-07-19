package mocks

import (
	"context"
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"

	"proxynd/pkg/client"
	"proxynd/pkg/types"
)

// MockUpstreamClient는 client.UpstreamClient 인터페이스의 모의 구현
type MockUpstreamClient struct {
	mock.Mock
}

// NewMockUpstreamClient는 새로운 MockUpstreamClient를 생성
func NewMockUpstreamClient() *MockUpstreamClient {
	return &MockUpstreamClient{}
}

// Do는 HTTP 요청을 실행
func (m *MockUpstreamClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// DoWithContext는 컨텍스트와 함께 HTTP 요청을 실행
func (m *MockUpstreamClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// Get은 GET 요청을 실행
func (m *MockUpstreamClient) Get(url string) (*http.Response, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// GetWithContext는 컨텍스트와 함께 GET 요청을 실행
func (m *MockUpstreamClient) GetWithContext(ctx context.Context, url string) (*http.Response, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// Post는 POST 요청을 실행
func (m *MockUpstreamClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	args := m.Called(url, contentType, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// PostWithContext는 컨텍스트와 함께 POST 요청을 실행
func (m *MockUpstreamClient) PostWithContext(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	args := m.Called(ctx, url, contentType, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// GetProxyConfig는 프록시 타입에 대한 설정을 반환
func (m *MockUpstreamClient) GetProxyConfig(proxyType types.ProxyType) *client.ProxyConfig {
	args := m.Called(proxyType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*client.ProxyConfig)
}

// SetProxyConfig는 프록시 타입에 대한 설정을 설정
func (m *MockUpstreamClient) SetProxyConfig(proxyType types.ProxyType, config *client.ProxyConfig) {
	m.Called(proxyType, config)
}

// GetStats는 클라이언트 통계를 반환
func (m *MockUpstreamClient) GetStats() *client.Stats {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*client.Stats)
}

// Close는 클라이언트를 종료
func (m *MockUpstreamClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockTransport는 테스트용 http.RoundTripper 구현
type MockTransport struct {
	mock.Mock
	Response *http.Response
	Error    error
}

// RoundTrip은 HTTP 요청을 처리
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Response != nil || m.Error != nil {
		return m.Response, m.Error
	}
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}