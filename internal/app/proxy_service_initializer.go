package app

import (
	"fmt"

	"proxynd/internal/adapters/stub"
	"proxynd/internal/logging"
	"proxynd/internal/usecase"
)

// InitializeProxyServiceWithStubs creates ProxyService with NoOp stub adapters
// This function is separate to avoid circular import issues
func (c *Container) InitializeProxyServiceWithStubs() error {
	c.logger.Info("Creating ProxyService with NoOp stub adapters")

	// Create NoOp stub adapters
	logger := stub.NewLoggerAdapter(c.logger)
	packageManager := stub.NewNoOpPackageManager(c.logger)
	cacheManager := stub.NewNoOpCacheManager(c.logger)
	authService := stub.NewNoOpAuthService(c.logger)
	metrics := stub.NewNoOpMetricsCollector(c.logger)
	rateLimiter := stub.NewNoOpRateLimiter(c.logger)
	cacheKeyBuilder := stub.NewNoOpCacheKeyBuilder(c.logger)

	// Create CacheStrategyService
	cacheStrategy := usecase.NewCacheStrategyService(
		cacheManager,
		logger,
		metrics,
		cacheKeyBuilder,
	)

	// Create ProxyService
	proxyService := usecase.NewProxyService(
		packageManager,
		cacheManager,
		cacheStrategy,
		authService,
		logger,
		metrics,
		rateLimiter,
	)

	// Register in container
	c.SetProxyService(proxyService)

	c.logger.Info("ProxyService created and registered successfully",
		logging.F("type", "hexagonal"),
		logging.F("adapters", "NoOp stubs"))

	return nil
}

// GetProxyServiceTyped returns ProxyService with proper type
func (c *Container) GetProxyServiceTyped() (*usecase.ProxyService, error) {
	ps := c.GetProxyService()
	if ps == nil {
		return nil, fmt.Errorf("ProxyService not initialized")
	}

	if ps == "pending" {
		return nil, fmt.Errorf("ProxyService initialization pending")
	}

	proxyService, ok := ps.(*usecase.ProxyService)
	if !ok {
		return nil, fmt.Errorf("ProxyService has invalid type: %T", ps)
	}

	return proxyService, nil
}
