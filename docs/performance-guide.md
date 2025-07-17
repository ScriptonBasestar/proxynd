# ProxyND Performance Optimization Guide

## 📋 Overview

This guide provides comprehensive information about ProxyND's performance optimization system, including cache optimization, connection pooling, resource monitoring, request optimization, and performance middleware.

## 🏗️ Performance Architecture

### Core Components

1. **Cache Optimizer** - Intelligent cache optimization with multiple strategies
2. **Connection Pool** - HTTP connection pooling with circuit breakers and health checking
3. **Resource Monitor** - System resource monitoring and automatic optimization
4. **Request Optimizer** - HTTP request/response optimization and adaptive rate limiting
5. **Performance Middleware** - Comprehensive performance monitoring and alerting

### Optimization Strategies

- **TTL Optimization** - Dynamically adjust cache TTL based on access patterns
- **Size Optimization** - Manage cache size and evict underutilized entries
- **Eviction Optimization** - Improve cache eviction policies
- **Prewarming** - Predictive cache prewarming for popular content
- **Hot Key Optimization** - Special handling for frequently accessed keys

## 🚀 Quick Start

### Basic Configuration

```yaml
# global.yaml
performance:
  cache_optimizer:
    enable_ttl_optimization: true
    enable_size_optimization: true
    enable_prewarming: true
    analysis_window: "1h"
    max_cache_size: 1073741824  # 1GB

  connection_pool:
    max_idle_conns: 100
    max_idle_conns_per_host: 10
    max_conns_per_host: 50
    enable_health_check: true
    enable_circuit_breaker: true

  resource_monitor:
    memory_threshold: 0.8  # 80%
    cpu_threshold: 0.8     # 80%
    enable_auto_gc: true
    enable_optimization: true

  request_optimizer:
    enable_compression: true
    enable_cache_optimization: true
    enable_adaptive_rate_limit: true
    max_request_size: 10485760  # 10MB

  middleware:
    enable_metrics: true
    enable_tracing: true
    slow_request_threshold: "2s"
    critical_request_threshold: "5s"
```

### Application Integration

```go
package main

import (
    "context"
    "github.com/scriptonbasestar/proxynd/internal/performance"
    "github.com/scriptonbasestar/proxynd/internal/logging"
    "github.com/gofiber/fiber/v2"
)

func main() {
    // Initialize logger
    logger, _ := logging.NewLogger(&logging.Config{
        Level: "info",
        Format: "json",
    })

    // Load performance configuration
    perfConfig := performance.DefaultConfig()

    // Initialize performance components
    cacheOptimizer := performance.NewCacheOptimizer(logger, perfConfig.GetCacheOptimizerConfig())
    connectionPool := performance.NewConnectionPool(logger, perfConfig.GetConnectionPoolConfig())
    resourceMonitor := performance.NewResourceMonitor(logger, perfConfig.GetResourceMonitorConfig())
    requestOptimizer := performance.NewRequestOptimizer(logger, perfConfig.GetRequestOptimizerConfig())

    // Initialize performance middleware
    perfMiddleware := performance.NewPerformanceMiddleware(
        logger,
        cacheOptimizer,
        connectionPool,
        resourceMonitor,
        requestOptimizer,
        perfConfig.GetMiddlewareConfig(),
    )

    // Start monitoring services
    ctx := context.Background()
    resourceMonitor.Start(ctx)
    cacheOptimizer.StartPeriodicOptimization(ctx)

    // Create Fiber app with performance middleware
    app := fiber.New()
    app.Use(perfMiddleware.Handler())

    // Add performance endpoints
    app.Get("/performance/metrics", func(c *fiber.Ctx) error {
        metrics := perfMiddleware.GetMetrics()
        return c.JSON(metrics)
    })

    app.Get("/performance/health", func(c *fiber.Ctx) error {
        health := perfMiddleware.GetHealthStatus()
        return c.JSON(health)
    })

    app.Listen(":8080")
}
```

## 🧠 Cache Optimization

### Optimization Strategies

#### TTL Optimization
Automatically adjusts cache TTL based on access patterns:

```go
// Hot keys with high hit rate get extended TTL
if keyStats.AccessCount > 10 && keyStats.HitRate > 0.8 {
    if keyStats.TTL < config.MinTTL*2 {
        newTTL := keyStats.TTL * 2
        // Apply new TTL
    }
}

// Cold keys with low hit rate get reduced TTL
if keyStats.HitRate < 0.3 && keyStats.TTL > config.MinTTL {
    newTTL := keyStats.TTL / 2
    // Apply new TTL
}
```

#### Size Optimization
Manages cache memory usage intelligently:

```go
// Evict large, underutilized entries
for _, candidate := range lowValueEntries {
    if candidate.Size > 100*1024 && candidate.HitRate < 0.5 {
        // Recommend eviction
    }
}
```

#### Hot Key Optimization
Special handling for frequently accessed keys:

```go
// Detect hot key concentration
hotKeyRatio := float64(topHotKeyAccess) / float64(totalAccess)
if hotKeyRatio > 0.2 { // Single key >20% of access
    // Implement replication or fast storage promotion
}
```

### Configuration

```yaml
cache_optimizer:
  # Analysis settings
  analysis_window: "1h"
  min_data_points: 100

  # Optimization thresholds
  hit_rate_threshold: 0.8
  miss_rate_threshold: 0.3
  eviction_threshold: 0.1

  # TTL optimization
  enable_ttl_optimization: true
  min_ttl: "5m"
  max_ttl: "24h"
  ttl_adjustment_factor: 0.1

  # Size optimization
  enable_size_optimization: true
  max_cache_size: 1073741824  # 1GB
  size_optimization_interval: "10m"

  # Prewarming
  enable_prewarming: true
  prewarming_schedule: "0 6 * * *"  # 6 AM daily
  prewarming_patterns:
    - "*.deb"
    - "*.rpm"
    - "maven-metadata.xml"
```

## 🔗 Connection Pooling

### Features

- **Intelligent Pooling** - Host-specific connection pools with automatic sizing
- **Circuit Breakers** - Automatic failure detection and recovery
- **Health Checking** - Continuous endpoint health monitoring
- **Adaptive Optimization** - Dynamic pool sizing based on usage patterns

### Configuration

```yaml
connection_pool:
  # Pool sizing
  max_idle_conns: 100
  max_idle_conns_per_host: 10
  max_conns_per_host: 50

  # Timeouts
  idle_conn_timeout: "90s"
  conn_timeout: "30s"
  keep_alive_timeout: "30s"
  tls_handshake_timeout: "10s"
  response_header_timeout: "30s"

  # Health checking
  enable_health_check: true
  health_check_interval: "30s"
  health_check_timeout: "5s"
  failure_threshold: 3

  # Circuit breaker
  enable_circuit_breaker: true
  circuit_breaker_threshold: 5
  circuit_breaker_window: "60s"
  circuit_breaker_timeout: "30s"

  # Optimization
  enable_connection_reuse: true
  enable_tcp_keep_alive: true
  enable_compression: true
  optimization_interval: "5m"
```

### Usage

```go
// Get optimized HTTP client for host
client, err := connectionPool.GetClient("registry.npmjs.org")
if err != nil {
    logger.Error("Failed to get client", logging.Error(err))
    return err
}

// Use client for requests
resp, err := client.Get("https://registry.npmjs.org/package")

// Record request metrics
connectionPool.RecordRequest("registry.npmjs.org", err == nil, time.Since(start))
```

## 📊 Resource Monitoring

### Monitored Resources

- **Memory** - Heap usage, GC statistics, memory pressure
- **CPU** - CPU utilization, goroutine count
- **Disk** - Disk usage, I/O performance
- **Network** - Network traffic, connection statistics

### Automatic Optimizations

- **Memory Management** - Automatic garbage collection triggers
- **Goroutine Monitoring** - Leak detection and alerting
- **Resource Alerts** - Threshold-based alerting system

### Configuration

```yaml
resource_monitor:
  # Monitoring intervals
  memory_interval: "30s"
  cpu_interval: "30s"
  disk_interval: "60s"
  network_interval: "30s"

  # Alert thresholds
  memory_threshold: 0.8      # 80%
  cpu_threshold: 0.8         # 80%
  disk_threshold: 0.9        # 90%
  goroutine_threshold: 1000

  # Optimization settings
  enable_auto_gc: true
  gc_threshold: 0.7          # 70% memory usage
  enable_optimization: true
```

### Monitoring Example

```go
// Initialize resource monitor
monitor := performance.NewResourceMonitor(logger, config)

// Start monitoring
monitor.Start(ctx)

// Get current resource statistics
stats := monitor.GetStats()
logger.Info("Resource usage",
    logging.Float64("memory_percent", stats.MemoryPercent),
    logging.Float64("cpu_percent", stats.CPUPercent),
    logging.Int("goroutine_count", stats.GoroutineCount))

// Get health status
health := monitor.GetHealthStatus()
if health["overall_status"] != "healthy" {
    logger.Warn("System health degraded", logging.Any("health", health))
}
```

## 🚀 Request Optimization

### Features

- **Response Compression** - Intelligent compression based on content type
- **Adaptive Rate Limiting** - Dynamic rate limiting based on system load
- **Cache Headers** - Automatic cache control header management
- **Request/Response Streaming** - Large file streaming optimization

### Configuration

```yaml
request_optimizer:
  # Compression settings
  enable_compression: true
  compression_level: 6
  compression_threshold: 1024
  compressible_types:
    - "text/html"
    - "application/json"
    - "text/css"
    - "application/javascript"

  # Caching optimization
  enable_cache_optimization: true
  cache_control_max_age: "1h"
  etags: true

  # Rate limiting
  enable_adaptive_rate_limit: true
  base_rate_limit: 1000
  burst_limit: 100
  rate_limit_window: "1m"

  # Request optimization
  max_request_size: 10485760    # 10MB
  request_timeout: "30s"
  keep_alive_timeout: "30s"

  # Response optimization
  enable_response_streaming: true
  streaming_threshold: 1048576  # 1MB
  max_response_size: 104857600  # 100MB

  # Performance monitoring
  slow_request_threshold: "2s"
  enable_metrics: true
```

## 📈 Performance Middleware

### Features

- **Comprehensive Monitoring** - Request performance, resource usage, errors
- **Automatic Optimization** - Triggered optimizations based on performance metrics
- **Health Status** - Overall system health assessment
- **Performance Alerts** - Critical performance issue alerting

### Metrics Collected

```json
{
  "total_requests": 15420,
  "slow_requests": 23,
  "critical_requests": 2,
  "average_response_time": "45ms",
  "p95_response_time": "120ms",
  "p99_response_time": "250ms",
  "cpu_usage": 0.45,
  "memory_usage": 0.67,
  "goroutine_count": 142,
  "cache_hit_rate": 0.84,
  "cache_miss_rate": 0.16,
  "active_connections": 28,
  "connection_errors": 1,
  "pool_utilization": 0.34,
  "error_rate": 0.002,
  "optimization_runs": 12,
  "last_update": "2024-07-17T10:30:00Z"
}
```

### Health Status

```json
{
  "overall_status": "healthy",
  "performance": {
    "status": "healthy",
    "average_response_time": "45ms",
    "slow_requests": 23,
    "error_rate": 0.002
  },
  "resources": {
    "status": "healthy",
    "cpu_usage": 0.45,
    "memory_usage": 0.67,
    "goroutines": 142
  },
  "cache": {
    "status": "healthy",
    "hit_rate": 0.84,
    "miss_rate": 0.16
  },
  "connections": {
    "status": "healthy",
    "active": 28,
    "utilization": 0.34,
    "errors": 1
  }
}
```

## 🔧 Configuration Examples

### Development Environment

```yaml
# config/development.yaml
performance:
  cache_optimizer:
    analysis_window: "10m"
    min_data_points: 10
    enable_prewarming: false

  connection_pool:
    max_idle_conns: 20
    max_idle_conns_per_host: 5
    enable_health_check: false

  resource_monitor:
    memory_interval: "60s"
    enable_auto_gc: false
    enable_optimization: false

  request_optimizer:
    enable_compression: false
    enable_adaptive_rate_limit: false
    slow_request_threshold: "5s"

  middleware:
    enable_metrics: true
    enable_tracing: false
    enable_profiling: true
```

### Production Environment

```yaml
# config/production.yaml
performance:
  cache_optimizer:
    analysis_window: "1h"
    min_data_points: 1000
    enable_ttl_optimization: true
    enable_size_optimization: true
    enable_prewarming: true
    max_cache_size: 5368709120  # 5GB

  connection_pool:
    max_idle_conns: 200
    max_idle_conns_per_host: 20
    max_conns_per_host: 100
    enable_health_check: true
    enable_circuit_breaker: true
    optimization_interval: "2m"

  resource_monitor:
    memory_interval: "15s"
    cpu_interval: "15s"
    memory_threshold: 0.85
    cpu_threshold: 0.85
    enable_auto_gc: true
    enable_optimization: true

  request_optimizer:
    enable_compression: true
    compression_level: 8
    enable_cache_optimization: true
    enable_adaptive_rate_limit: true
    base_rate_limit: 5000
    max_request_size: 52428800   # 50MB

  middleware:
    enable_metrics: true
    enable_tracing: true
    enable_profiling: false
    slow_request_threshold: "1s"
    critical_request_threshold: "3s"
    enable_alerts: true
```

### High-Load Environment

```yaml
# config/high-load.yaml
performance:
  cache_optimizer:
    analysis_window: "30m"
    max_cache_size: 10737418240  # 10GB
    enable_ttl_optimization: true
    enable_size_optimization: true
    enable_prewarming: true
    prewarming_schedule: "0 */4 * * *"  # Every 4 hours

  connection_pool:
    max_idle_conns: 500
    max_idle_conns_per_host: 50
    max_conns_per_host: 200
    idle_conn_timeout: "120s"
    optimization_interval: "1m"

  resource_monitor:
    memory_interval: "10s"
    cpu_interval: "10s"
    memory_threshold: 0.9
    cpu_threshold: 0.9
    goroutine_threshold: 5000
    enable_auto_gc: true

  request_optimizer:
    enable_compression: true
    compression_level: 9
    base_rate_limit: 10000
    burst_limit: 1000
    max_request_size: 104857600   # 100MB
    streaming_threshold: 524288   # 512KB

  middleware:
    slow_request_threshold: "500ms"
    critical_request_threshold: "2s"
    enable_alerts: true
    alert_threshold: 5
```

## 📊 Performance Analysis

### Monitoring Commands

```bash
# Get performance metrics
curl http://localhost:8080/performance/metrics | jq

# Get health status
curl http://localhost:8080/performance/health | jq

# Get cache statistics
curl http://localhost:8080/performance/cache/stats | jq

# Get connection pool statistics
curl http://localhost:8080/performance/connections/stats | jq

# Get resource statistics  
curl http://localhost:8080/performance/resources/stats | jq
```

### Performance Queries

#### Response Time Analysis
```bash
# Find slow requests in logs
jq -r 'select(.duration > 2000) | "\(.timestamp) \(.method) \(.path) \(.duration)ms"' \
  /var/log/proxynd/performance.log

# Calculate average response time by endpoint
jq -r 'select(.path) | "\(.path) \(.duration)"' /var/log/proxynd/performance.log | \
  awk '{sum[$1] += $2; count[$1]++} END {for (path in sum) print path, sum[path]/count[path]}'
```

#### Cache Analysis
```bash
# Cache hit rate by key pattern
jq -r 'select(.cache_operation) | "\(.cache_key) \(.cache_hit)"' \
  /var/log/proxynd/performance.log | \
  awk '{hits[$1] += $2; total[$1]++} END {for (key in total) print key, hits[key]/total[key]}'

# Most frequently evicted keys
jq -r 'select(.cache_operation == "eviction") | .cache_key' \
  /var/log/proxynd/performance.log | sort | uniq -c | sort -nr
```

#### Resource Analysis
```bash
# Memory usage trends
jq -r 'select(.memory_percent) | "\(.timestamp) \(.memory_percent)"' \
  /var/log/proxynd/performance.log

# Goroutine leak detection
jq -r 'select(.goroutine_count) | "\(.timestamp) \(.goroutine_count)"' \
  /var/log/proxynd/performance.log | \
  awk '{if (prev != "" && $2 > prev + 100) print "Potential leak at", $1, ":", $2 - prev, "new goroutines"; prev = $2}'
```

## 🔍 Troubleshooting

### Common Performance Issues

#### High Memory Usage
```yaml
# Reduce cache size and enable aggressive GC
performance:
  cache_optimizer:
    max_cache_size: 536870912  # 512MB

  resource_monitor:
    memory_threshold: 0.7
    enable_auto_gc: true
    gc_threshold: 0.6
```

#### Slow Response Times
```yaml
# Enable connection pooling and request optimization
performance:
  connection_pool:
    max_conns_per_host: 100
    enable_circuit_breaker: true

  request_optimizer:
    enable_compression: true
    enable_cache_optimization: true
    request_timeout: "15s"
```

#### Cache Misses
```yaml
# Improve cache strategy
performance:
  cache_optimizer:
    enable_ttl_optimization: true
    enable_prewarming: true
    prewarming_schedule: "0 */2 * * *"  # Every 2 hours
```

#### Connection Errors
```yaml
# Improve connection reliability
performance:
  connection_pool:
    enable_health_check: true
    health_check_interval: "15s"
    failure_threshold: 2
    enable_circuit_breaker: true
```

## 📚 Best Practices

### Cache Optimization
1. **Monitor Hit Rates** - Maintain >80% cache hit rate
2. **Size Management** - Keep cache size under 80% of available memory
3. **TTL Tuning** - Use dynamic TTL based on access patterns
4. **Prewarming** - Implement predictive prewarming for popular content

### Connection Management
1. **Pool Sizing** - Size pools based on expected concurrent requests
2. **Health Checks** - Enable health checking for critical services
3. **Circuit Breakers** - Protect against cascading failures
4. **Timeout Tuning** - Set appropriate timeouts for different services

### Resource Monitoring
1. **Set Thresholds** - Configure appropriate resource thresholds
2. **Enable Auto-GC** - Use automatic garbage collection for memory management
3. **Monitor Goroutines** - Watch for goroutine leaks
4. **Alert Configuration** - Set up proper alerting for resource issues

### Request Optimization
1. **Enable Compression** - Use compression for text-based content
2. **Cache Headers** - Set appropriate cache control headers
3. **Rate Limiting** - Implement adaptive rate limiting
4. **Stream Large Files** - Use streaming for large file transfers

---

**Last Updated**: 2024-07-17  
**Review Schedule**: Monthly  
**Next Review**: 2024-08-17
