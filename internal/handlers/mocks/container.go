package mocks

import (
	"github.com/stretchr/testify/mock"
	"proxynd/cache"
)

// MockContainer Container 인터페이스의 모의 구현
type MockContainer struct {
	mock.Mock
}

// Cache 캐시 인스턴스 반환
func (m *MockContainer) Cache() cache.Cache {
	args := m.Called()
	return args.Get(0).(cache.Cache)
}