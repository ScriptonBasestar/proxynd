// Package mocks provides mock implementations for testing
package mocks

import (
	"proxynd/internal/config"
)

// MockAptProxyConfig is a mock implementation of AptProxyConfig
type MockAptProxyConfig struct {
	config.AptProxyConfig
	ReadConfigFunc func() error
}

// ReadConfig calls the mock function or returns nil
func (m *MockAptProxyConfig) ReadConfig() error {
	if m.ReadConfigFunc != nil {
		return m.ReadConfigFunc()
	}
	return nil
}
