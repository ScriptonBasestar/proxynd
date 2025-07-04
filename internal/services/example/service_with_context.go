package example

import (
	"context"
	"fmt"
	"time"
	
	"proxynd/internal/interfaces"
	"proxynd/logging"
)

// ServiceWithContext demonstrates proper context propagation in service methods
type ServiceWithContext struct {
	cache  interfaces.CacheManager
	config ConfigServiceWithContext
	logger logging.Logger
}

// ConfigServiceWithContext shows how config service should accept context
type ConfigServiceWithContext interface {
	GetConfig(ctx context.Context, key string) (interface{}, error)
	ReloadConfig(ctx context.Context) error
}

// NewServiceWithContext creates a new service with proper dependency injection
func NewServiceWithContext(
	ctx context.Context,
	cache interfaces.CacheManager,
	config ConfigServiceWithContext,
) (*ServiceWithContext, error) {
	// Validate dependencies with context timeout
	validateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := validateDependencies(validateCtx, cache, config); err != nil {
		return nil, fmt.Errorf("dependency validation failed: %w", err)
	}
	
	return &ServiceWithContext{
		cache:  cache,
		config: config,
		logger: logging.GetLogger(),
	}, nil
}

// ProcessRequest demonstrates context propagation through service methods
func (s *ServiceWithContext) ProcessRequest(ctx context.Context, requestID string) (*Response, error) {
	// Add request ID to context for tracing
	ctx = context.WithValue(ctx, "requestID", requestID)
	
	// Check cache with context
	cachedData, found, err := s.getCachedData(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("cache error: %w", err)
	}
	
	if found {
		s.logger.Debug("Cache hit", 
			logging.F("request_id", requestID),
			logging.F("cached", true))
		return &Response{Data: cachedData, Cached: true}, nil
	}
	
	// Fetch data with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	data, err := s.fetchData(fetchCtx, requestID)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}
	
	// Store in cache asynchronously with separate context
	cacheCtx, cacheCancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		defer cacheCancel()
		if err := s.storeInCache(cacheCtx, requestID, data); err != nil {
			s.logger.Warn("Failed to cache data",
				logging.F("request_id", requestID),
				logging.F("error", err))
		}
	}()
	
	return &Response{Data: data, Cached: false}, nil
}

// getCachedData retrieves data from cache with context
func (s *ServiceWithContext) getCachedData(ctx context.Context, key string) ([]byte, bool, error) {
	// Check context before proceeding
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	
	cacheKey := "request:" + key
	return s.cache.Get(ctx, cacheKey)
}

// fetchData simulates fetching data with context awareness
func (s *ServiceWithContext) fetchData(ctx context.Context, requestID string) ([]byte, error) {
	// Simulate network call with context
	select {
	case <-time.After(100 * time.Millisecond):
		// Data fetched successfully
		return []byte(fmt.Sprintf("data for %s", requestID)), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// storeInCache stores data in cache with context
func (s *ServiceWithContext) storeInCache(ctx context.Context, key string, data []byte) error {
	cacheKey := "request:" + key
	return s.cache.Put(ctx, cacheKey, data, 5*time.Minute)
}

// PerformLongRunningOperation demonstrates handling long operations with context
func (s *ServiceWithContext) PerformLongRunningOperation(ctx context.Context, input string) error {
	// Create a channel to signal completion
	done := make(chan error, 1)
	
	go func() {
		// Simulate long operation
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				return
			default:
				// Process chunk
				time.Sleep(100 * time.Millisecond)
				s.logger.Debug("Processing chunk",
					logging.F("chunk", i),
					logging.F("input", input))
			}
		}
		done <- nil
	}()
	
	// Wait for completion or cancellation
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReloadConfiguration demonstrates config reload with context
func (s *ServiceWithContext) ReloadConfiguration(ctx context.Context) error {
	s.logger.Info("Reloading configuration")
	
	// Reload with timeout
	reloadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	if err := s.config.ReloadConfig(reloadCtx); err != nil {
		return fmt.Errorf("config reload failed: %w", err)
	}
	
	s.logger.Info("Configuration reloaded successfully")
	return nil
}

// validateDependencies validates service dependencies with context
func validateDependencies(ctx context.Context, cache interfaces.CacheManager, config ConfigServiceWithContext) error {
	// Check cache connectivity
	testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	
	if _, _, err := cache.Get(testCtx, "test:connectivity"); err != nil {
		// Ignore "not found" errors for connectivity test
		_ = err
	}
	
	// Check config availability
	if _, err := config.GetConfig(testCtx, "test"); err != nil {
		// Ignore "not found" errors for connectivity test
		_ = err
	}
	
	return nil
}

// Response represents a service response
type Response struct {
	Data   []byte
	Cached bool
}