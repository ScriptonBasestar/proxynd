// Package mocks provides mock implementations for testing
package mocks

import (
	"proxynd/configs"
)

// MockAptProxyConfig is a mock implementation of AptProxyConfig
type MockAptProxyConfig struct {
	configs.AptProxyConfig
	ReadConfigFunc func() error
}

// ReadConfig calls the mock function or returns nil
func (m *MockAptProxyConfig) ReadConfig() error {
	if m.ReadConfigFunc != nil {
		return m.ReadConfigFunc()
	}
	return nil
}
