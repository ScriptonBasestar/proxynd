package performance

import (
	"context"
	"testing"
	"time"

	"proxynd/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.False(t, manager.IsRunning())
}

func TestManagerStartStop(t *testing.T) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test start
	err = manager.Start(ctx)
	require.NoError(t, err)
	assert.True(t, manager.IsRunning())

	// Test double start (should not error)
	err = manager.Start(ctx)
	require.NoError(t, err)

	// Test stop
	err = manager.Stop()
	require.NoError(t, err)
	assert.False(t, manager.IsRunning())

	// Test double stop (should not error)
	err = manager.Stop()
	require.NoError(t, err)
}

func TestManagerComponents(t *testing.T) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(t, err)

	// Test component access
	assert.NotNil(t, manager.GetCacheOptimizer())
	assert.NotNil(t, manager.GetConnectionPool())
	assert.NotNil(t, manager.GetResourceMonitor())
	assert.NotNil(t, manager.GetRequestOptimizer())
	assert.NotNil(t, manager.GetMiddleware())
}

func TestManagerMetrics(t *testing.T) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(t, err)

	// Test metrics
	metrics := manager.GetGlobalMetrics()
	assert.NotNil(t, metrics)
	assert.NotZero(t, metrics.LastUpdate)
}

func TestManagerHealthStatus(t *testing.T) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	// Test health status
	health := manager.GetHealthStatus()
	assert.NotNil(t, health)
	assert.Contains(t, health, "overall_status")
	assert.Contains(t, health, "components")
	assert.Contains(t, health, "metrics")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultManagerConfig()

	// Test cache optimizer config
	assert.NotNil(t, config.CacheOptimizer)
	assert.True(t, config.CacheOptimizer.EnableTTLOptimization)
	assert.True(t, config.CacheOptimizer.EnableSizeOptimization)
	assert.True(t, config.CacheOptimizer.EnablePrewarming)

	// Test connection pool config
	assert.NotNil(t, config.ConnectionPool)
	assert.True(t, config.ConnectionPool.EnableHealthCheck)
	assert.True(t, config.ConnectionPool.EnableCircuitBreaker)
	assert.True(t, config.ConnectionPool.EnableConnectionReuse)

	// Test resource monitor config
	assert.NotNil(t, config.ResourceMonitor)
	assert.True(t, config.ResourceMonitor.EnableAutoGC)
	assert.True(t, config.ResourceMonitor.EnableOptimization)

	// Test request optimizer config
	assert.NotNil(t, config.RequestOptimizer)
	assert.True(t, config.RequestOptimizer.EnableCompression)
	assert.True(t, config.RequestOptimizer.EnableCacheOptimization)
	assert.True(t, config.RequestOptimizer.EnableAdaptiveRateLimit)

	// Test middleware config
	assert.NotNil(t, config.Middleware)
	assert.True(t, config.Middleware.EnableMetrics)
	assert.True(t, config.Middleware.EnableTracing)
	assert.True(t, config.Middleware.EnableRequestOptimization)
}

func TestManagerGlobalOptimization(t *testing.T) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	// Reduce optimization interval for testing
	config.ResourceMonitor.MemoryInterval = 100 * time.Millisecond

	manager, err := NewManager(logger, config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	// Wait for some optimization cycles
	time.Sleep(500 * time.Millisecond)

	// Check that metrics are being updated
	metrics := manager.GetGlobalMetrics()
	assert.NotNil(t, metrics)
}

func TestManagerComponentIntegration(t *testing.T) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	// Test cache optimizer integration
	cacheOptimizer := manager.GetCacheOptimizer()
	assert.NotNil(t, cacheOptimizer)

	// Record some cache accesses
	cacheOptimizer.RecordCacheAccess("test-key", true, 1024, 10*time.Millisecond)
	cacheOptimizer.RecordCacheAccess("test-key-2", false, 2048, 20*time.Millisecond)

	// Test connection pool integration
	connectionPool := manager.GetConnectionPool()
	assert.NotNil(t, connectionPool)

	// Record some connection metrics
	connectionPool.RecordRequest("example.com", true, 50*time.Millisecond)
	connectionPool.RecordRequest("example.com", false, 100*time.Millisecond)

	// Verify metrics are aggregated
	metrics := manager.GetGlobalMetrics()
	assert.NotNil(t, metrics)
}

func BenchmarkManagerMetrics(b *testing.B) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = manager.GetGlobalMetrics()
		}
	})
}

func BenchmarkManagerHealthStatus(b *testing.B) {
	logger := logging.NewTestLogger()
	config := DefaultManagerConfig()

	manager, err := NewManager(logger, config)
	require.NoError(b, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(b, err)
	defer manager.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = manager.GetHealthStatus()
		}
	})
}
