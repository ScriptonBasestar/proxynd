package stub

import (
	"fmt"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// NoOpCacheKeyBuilder is a no-operation implementation of ports.CacheKeyBuilder
// Generates simple cache keys
type NoOpCacheKeyBuilder struct {
	logger logging.Logger
}

// NewNoOpCacheKeyBuilder creates a new NoOp cache key builder
func NewNoOpCacheKeyBuilder(logger logging.Logger) ports.CacheKeyBuilder {
	return &NoOpCacheKeyBuilder{logger: logger}
}

// BuildKey generates cache key (NoOp implementation - simple concatenation)
func (kb *NoOpCacheKeyBuilder) BuildKey(packageType, repository, name, version, path string) string {
	kb.logger.Debug("NoOpCacheKeyBuilder.BuildKey called (stub)",
		logging.F("package_type", packageType),
		logging.F("repository", repository),
		logging.F("name", name),
		logging.F("version", version),
		logging.F("path", path))

	// Simple key format: packageType:repository:name:version:path
	return fmt.Sprintf("%s:%s:%s:%s:%s", packageType, repository, name, version, path)
}

// ParseKey parses cache key components (NoOp implementation)
func (kb *NoOpCacheKeyBuilder) ParseKey(key string) (*ports.KeyComponents, error) {
	kb.logger.Debug("NoOpCacheKeyBuilder.ParseKey called (stub)",
		logging.F("key", key))

	// Return minimal key components
	return &ports.KeyComponents{
		PackageType: "unknown",
		Repository:  "",
		Name:        key,
		Version:     "",
		Path:        "",
	}, nil
}

// BuildPattern builds key pattern for invalidation (NoOp implementation)
func (kb *NoOpCacheKeyBuilder) BuildPattern(packageType, repository string) string {
	kb.logger.Debug("NoOpCacheKeyBuilder.BuildPattern called (stub)",
		logging.F("package_type", packageType),
		logging.F("repository", repository))

	// Simple pattern format: packageType:repository:*
	return fmt.Sprintf("%s:%s:*", packageType, repository)
}
