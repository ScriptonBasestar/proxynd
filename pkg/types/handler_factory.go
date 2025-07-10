package types

import (
	"fmt"
	"sync"
)

// StandardProxyHandlerFactory is the standard implementation of ProxyHandlerFactory
type StandardProxyHandlerFactory struct {
	mu       sync.RWMutex
	creators map[ProxyType]func() ProxyHandler
	handlers map[ProxyType]ProxyHandler
}

// NewStandardProxyHandlerFactory creates a new standard proxy handler factory
func NewStandardProxyHandlerFactory() *StandardProxyHandlerFactory {
	return &StandardProxyHandlerFactory{
		creators: make(map[ProxyType]func() ProxyHandler),
		handlers: make(map[ProxyType]ProxyHandler),
	}
}

// RegisterHandler registers a new proxy handler type
func (f *StandardProxyHandlerFactory) RegisterHandler(proxyType ProxyType, creator func() ProxyHandler) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.creators[proxyType]; exists {
		return fmt.Errorf("handler for proxy type %s already registered", proxyType)
	}

	f.creators[proxyType] = creator
	return nil
}

// CreateHandler creates a handler for the specified proxy type
func (f *StandardProxyHandlerFactory) CreateHandler(proxyType ProxyType) (ProxyHandler, error) {
	f.mu.RLock()
	// Check if handler already exists (singleton pattern)
	if handler, exists := f.handlers[proxyType]; exists {
		f.mu.RUnlock()
		return handler, nil
	}
	f.mu.RUnlock()

	// Need to create new handler
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if handler, exists := f.handlers[proxyType]; exists {
		return handler, nil
	}

	// Get creator function
	creator, exists := f.creators[proxyType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for proxy type %s", proxyType)
	}

	// Create handler
	handler := creator()
	if handler == nil {
		return nil, fmt.Errorf("handler creator returned nil for proxy type %s", proxyType)
	}

	// Verify handler type matches
	if handler.GetType() != proxyType {
		return nil, fmt.Errorf("handler type mismatch: expected %s, got %s", proxyType, handler.GetType())
	}

	// Cache the handler
	f.handlers[proxyType] = handler

	return handler, nil
}

// GetSupportedTypes returns all supported proxy types
func (f *StandardProxyHandlerFactory) GetSupportedTypes() []ProxyType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]ProxyType, 0, len(f.creators))
	for proxyType := range f.creators {
		types = append(types, proxyType)
	}

	return types
}

// ClearHandlers clears all cached handlers (useful for testing or reloading)
func (f *StandardProxyHandlerFactory) ClearHandlers() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.handlers = make(map[ProxyType]ProxyHandler)
}

// Ensure StandardProxyHandlerFactory implements ProxyHandlerFactory
var _ ProxyHandlerFactory = (*StandardProxyHandlerFactory)(nil)
