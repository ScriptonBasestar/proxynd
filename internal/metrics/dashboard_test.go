package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsCache(t *testing.T) {
	ttl := 10 * time.Second
	cache := NewMetricsCache(ttl)

	assert.NotNil(t, cache)
	assert.Equal(t, ttl, cache.ttl)
	assert.Nil(t, cache.data)
}

func TestMetricsCache_SetAndGet(t *testing.T) {
	cache := NewMetricsCache(10 * time.Second)

	// Initially should return nil, false
	data, ok := cache.Get()
	assert.Nil(t, data)
	assert.False(t, ok)

	// Set data
	testData := &DashboardMetrics{
		Overview: OverviewStats{
			TotalRequests: 1000,
			TotalPackages: 500,
			CacheHitRate:  0.85,
		},
		LastUpdated: time.Now(),
	}

	cache.Set(testData)

	// Should now return data
	data, ok = cache.Get()
	assert.NotNil(t, data)
	assert.True(t, ok)
	assert.Equal(t, testData.Overview.TotalRequests, data.Overview.TotalRequests)
}

func TestMetricsCache_Expiration(t *testing.T) {
	cache := NewMetricsCache(50 * time.Millisecond)

	testData := &DashboardMetrics{
		Overview: OverviewStats{
			TotalRequests: 1000,
		},
		LastUpdated: time.Now(),
	}

	cache.Set(testData)

	// Should be valid immediately
	data, ok := cache.Get()
	assert.NotNil(t, data)
	assert.True(t, ok)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	data, ok = cache.Get()
	assert.Nil(t, data)
	assert.False(t, ok)
}

func TestMetricsCache_Clear(t *testing.T) {
	cache := NewMetricsCache(10 * time.Second)

	testData := &DashboardMetrics{
		Overview: OverviewStats{
			TotalRequests: 1000,
		},
		LastUpdated: time.Now(),
	}

	cache.Set(testData)

	// Verify data exists
	data, ok := cache.Get()
	assert.NotNil(t, data)
	assert.True(t, ok)

	// Clear cache
	cache.Clear()

	// Should be empty now
	data, ok = cache.Get()
	assert.Nil(t, data)
	assert.False(t, ok)
}

func TestMetricsCache_ConcurrentAccess(t *testing.T) {
	cache := NewMetricsCache(10 * time.Second)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			testData := &DashboardMetrics{
				Overview: OverviewStats{
					TotalRequests: float64(idx * 100),
				},
				LastUpdated: time.Now(),
			}
			cache.Set(testData)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have some data
	data, ok := cache.Get()
	assert.NotNil(t, data)
	assert.True(t, ok)
}
