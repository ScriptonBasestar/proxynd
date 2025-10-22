package stub

import (
	"context"
	"fmt"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// NoOpCacheManager is a no-operation implementation of ports.CacheManager
// Always returns cache miss
type NoOpCacheManager struct {
	logger logging.Logger
}

// NewNoOpCacheManager creates a new NoOp cache manager
func NewNoOpCacheManager(logger logging.Logger) ports.CacheManager {
	return &NoOpCacheManager{logger: logger}
}

// Get retrieves from cache (NoOp implementation - always miss)
func (cm *NoOpCacheManager) Get(ctx context.Context, req *ports.CacheRequest) (*ports.CacheResponse, error) {
	cm.logger.Debug("NoOpCacheManager.Get called (stub) - always miss",
		logging.F("key", req.Key),
		logging.F("package_type", req.PackageType))

	return &ports.CacheResponse{
		Hit:     false,
		Backend: "noop",
		Metadata: ports.CacheMetadata{
			Headers: make(map[string]string),
		},
	}, nil
}

// Set stores in cache (NoOp implementation)
func (cm *NoOpCacheManager) Set(ctx context.Context, req *ports.CacheSetRequest) error {
	cm.logger.Debug("NoOpCacheManager.Set called (stub)",
		logging.F("key", req.Key),
		logging.F("data_length", len(req.Data)))

	return nil
}

// Invalidate removes from cache (NoOp implementation)
func (cm *NoOpCacheManager) Invalidate(ctx context.Context, pattern string) error {
	cm.logger.Debug("NoOpCacheManager.Invalidate called (stub)",
		logging.F("pattern", pattern))

	return nil
}

// GetBackend returns specific backend (NoOp implementation)
func (cm *NoOpCacheManager) GetBackend(name string) (ports.CacheBackend, error) {
	cm.logger.Debug("NoOpCacheManager.GetBackend called (stub)",
		logging.F("backend", name))

	return nil, fmt.Errorf("NoOpCacheManager: GetBackend not implemented")
}

// ListBackends returns available backends (NoOp implementation - empty)
func (cm *NoOpCacheManager) ListBackends() []string {
	cm.logger.Debug("NoOpCacheManager.ListBackends called (stub)")
	return []string{"noop"}
}

// GetStats returns cache statistics (NoOp implementation)
func (cm *NoOpCacheManager) GetStats() *ports.CacheStats {
	cm.logger.Debug("NoOpCacheManager.GetStats called (stub)")

	return &ports.CacheStats{
		Backend:   "noop",
		Size:      0,
		ItemCount: 0,
		HitRate:   0.0,
		HitCount:  0,
		MissCount: 0,
	}
}

// GetStrategy returns caching strategy for package type (NoOp implementation)
func (cm *NoOpCacheManager) GetStrategy(packageType string) ports.CacheStrategy {
	cm.logger.Debug("NoOpCacheManager.GetStrategy called (stub)",
		logging.F("package_type", packageType))

	// Return nil strategy (no actual caching strategy)
	return nil
}
