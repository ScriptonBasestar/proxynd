# Health Check API

ProxyND provides comprehensive health check endpoints for monitoring service health and readiness.

## Endpoints

### Main Health Check

**GET /health**

Returns detailed health status of all components with format options.

**Query Parameters:**
- `format=simple` - Returns simplified response

**Response (200 OK or 503 Service Unavailable):**
```json
{
  "status": "healthy|degraded|unhealthy",
  "timestamp": "2024-01-20T10:30:00Z",
  "uptime": "2d 5h 30m 15s",
  "version": "1.0.0",
  "environment": "production",
  "checks": {
    "environment": {
      "name": "environment",
      "status": "healthy",
      "message": "All required environment variables are set",
      "duration_ms": 0.5,
      "details": {
        "CONFIG_DIR": true,
        "STORAGE_DIR": true
      },
      "last_checked": "2024-01-20T10:30:00Z"
    },
    "disk_space": {
      "name": "disk_space",
      "status": "healthy",
      "message": "Disk space OK: 45.2% free",
      "duration_ms": 1.2,
      "details": {
        "path": "/storage",
        "total_bytes": 1099511627776,
        "free_bytes": 497238163456,
        "used_bytes": 602273464320,
        "free_percent": 45.2
      },
      "last_checked": "2024-01-20T10:30:00Z"
    }
  }
}
```

**Simple Format Response:**
```json
{
  "status": "healthy",
  "timestamp": 1705749000
}
```

### Kubernetes Probes

#### Liveness Probe

**GET /health/live**

Indicates whether the application is running. Fast response (<10ms).

**Response (200 OK):**
```json
{
  "status": "alive"
}
```

#### Readiness Probe

**GET /health/ready**

Indicates whether the application is ready to serve requests. Fast response (<100ms).

**Response (200 OK or 503 Service Unavailable):**
```json
{
  "ready": true,
  "checks": {
    "environment": true,
    "disk_space": true
  }
}
```

### Enhanced Monitoring Endpoints

#### Comprehensive Health Analysis

**GET /health/comprehensive**

Complete system health analysis including all proxy types, cache systems, and resource monitoring.
Response time target: <500ms.

**Response:**
```json
{
  "status": "healthy|degraded|unhealthy",
  "timestamp": "2024-01-20T10:30:00Z",
  "categories": {
    "infrastructure": "healthy",
    "performance": "healthy",
    "security": "healthy",
    "connectivity": "healthy"
  },
  "checkers": {
    "proxy_health": {
      "status": "healthy",
      "upstreams": {
        "npm": "healthy",
        "pip": "healthy",
        "docker": "healthy"
      }
    },
    "cache_health": {
      "status": "healthy",
      "backends": {
        "filesystem": "healthy"
      }
    },
    "system_health": {
      "status": "healthy",
      "resources": {
        "cpu_usage": 15.2,
        "memory_usage": 45.8,
        "disk_usage": 34.1
      }
    }
  }
}
```

#### Fast Health Check

**GET /health/fast**

Core health checks only for sub-100ms responses.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-20T10:30:00Z",
  "core_checks": {
    "environment": "healthy",
    "disk_space": "healthy",
    "process": "healthy"
  }
}
```

#### Available Health Checkers

**GET /health/checkers**

Returns list of available health checkers.

**Response:**
```json
{
  "checkers": [
    "proxy_health_checker",
    "cache_health_checker",
    "security_health_checker",
    "system_health_checker"
  ]
}
```

#### Health Metrics

**GET /health/metrics**

Health metrics and trends for monitoring dashboard integration.

**Response:**
```json
{
  "metrics": {
    "check_count": 1247,
    "success_rate": 99.2,
    "avg_response_time": 45.3,
    "last_failure": "2024-01-19T14:22:00Z"
  },
  "trends": {
    "response_times": [42.1, 43.2, 45.3],
    "success_rates": [99.1, 99.0, 99.2]
  }
}
```

### Proxy-Specific Health Endpoints

#### All Proxy Upstream Status

**GET /health/proxy/upstreams**

Status of all proxy upstream connections.

**Response:**
```json
{
  "upstreams": {
    "npm": {
      "url": "https://registry.npmjs.org",
      "status": "healthy",
      "response_time": 85.3,
      "last_checked": "2024-01-20T10:30:00Z"
    },
    "pip": {
      "url": "https://pypi.org/simple",
      "status": "healthy",
      "response_time": 120.1,
      "last_checked": "2024-01-20T10:30:00Z"
    }
  }
}
```

#### Specific Proxy Upstream Status

**GET /health/proxy/upstreams/:type**

Status of specific proxy upstream (npm, pip, docker, maven, apt, yum, apk).

**Response:**
```json
{
  "type": "npm",
  "url": "https://registry.npmjs.org",
  "status": "healthy",
  "response_time": 85.3,
  "circuit_breaker": "closed",
  "last_checked": "2024-01-20T10:30:00Z"
}
```

#### Circuit Breaker States

**GET /health/proxy/circuit-breakers**

Current state of all circuit breakers.

**Response:**
```json
{
  "circuit_breakers": {
    "npm": "closed",
    "pip": "closed",
    "docker": "half-open",
    "maven": "closed"
  }
}
```

### System Resource Endpoints

#### Resource Summary

**GET /health/system/resources**

System resource usage summary.

**Response:**
```json
{
  "cpu": {
    "usage_percent": 15.2,
    "goroutines": 42
  },
  "memory": {
    "usage_percent": 45.8,
    "alloc_mb": 234,
    "sys_mb": 512
  },
  "disk": {
    "usage_percent": 34.1,
    "free_gb": 45.2,
    "total_gb": 100.0
  }
}
```

#### Resource Usage History

**GET /health/system/resources/history**

Historical resource usage trends.

**Response:**
```json
{
  "history": {
    "cpu": [12.1, 14.2, 15.2],
    "memory": [43.1, 44.5, 45.8],
    "disk": [33.8, 34.0, 34.1]
  },
  "timestamps": [
    "2024-01-20T10:28:00Z",
    "2024-01-20T10:29:00Z",
    "2024-01-20T10:30:00Z"
  ]
}
```

#### Resource Predictions

**GET /health/system/resources/predictions**

Predictive analysis for resource exhaustion.

**Response:**
```json
{
  "predictions": {
    "disk_full_eta": "2024-02-15T00:00:00Z",
    "memory_pressure_risk": "low",
    "cpu_saturation_risk": "low"
  },
  "recommendations": [
    "Monitor disk usage trend",
    "Consider cache cleanup policies"
  ]
}
```

### Individual Health Checks

**GET /health/check/:name**

Returns the status of a specific health check.

**Parameters:**
- `:name` - Name of the health check (e.g., `environment`, `disk_space`, `npm_registry`)

**Response (200 OK or 503 Service Unavailable):**
```json
{
  "name": "disk_space",
  "status": "healthy",
  "message": "Disk space OK: 45.2% free",
  "duration_ms": 1.2,
  "details": {
    "path": "/storage",
    "total_bytes": 1099511627776,
    "free_bytes": 497238163456,
    "free_percent": 45.2
  },
  "last_checked": "2024-01-20T10:30:00Z"
}
```

### Debug Information

**GET /health/debug**

Returns detailed debug information (disabled in production).

**Response (200 OK or 403 Forbidden):**
```json
{
  "status": "healthy",
  "checks": { /* all health checks */ },
  "uptime": 185415.5,
  "goroutines": 42,
  "memory": {
    "alloc_mb": 45,
    "total_alloc_mb": 1234,
    "sys_mb": 75,
    "num_gc": 123,
    "gc_cpu_percent": 0.05
  },
  "environment": {
    "CONFIG_DIR": "/config",
    "STORAGE_DIR": "/storage",
    "SERVER_PORT": "8080",
    "REDIS_PASSWORD": "***MASKED***"
  },
  "config": {
    "server": {
      "port": 8080,
      "tls_enabled": false
    },
    "cache": {
      "backend": "file",
      "ttl_seconds": 3600
    },
    "registries_enabled": {
      "npm": true,
      "pypi": true,
      "apt": true,
      "docker": true,
      "maven": true
    },
    "metrics_enabled": true,
    "auth_enabled": false
  }
}
```

## Health Check Components

### Built-in Checkers

1. **Environment Checker**
   - Verifies required environment variables are set
   - Required: `CONFIG_DIR`, `STORAGE_DIR`

2. **Disk Space Checker**
   - Monitors available disk space
   - Default thresholds: 1GB free space, 10% free

3. **Writable Directory Checker**
   - Ensures directories are writable
   - Checks: storage directory, cache directory

4. **Proxy Health Checker**
   - NPM registry connectivity (`https://registry.npmjs.org/-/ping`)
   - PyPI index validation (`https://pypi.org/simple/`)
   - APT mirror availability (Release file checks)
   - Docker registry v2 API validation
   - Maven repository metadata validation
   - Circuit breaker integration for fault tolerance

5. **Cache Health Checker**
   - Backend connectivity (File/S3/Redis)
   - Performance testing (1KB data operations)
   - Capacity monitoring (disk space, file counts)
   - Consistency verification (TTL validation)
   - Cache efficiency metrics

6. **Security Health Checker**
   - TLS certificate validation and expiry
   - Authentication system status (Basic Auth, OAuth2, JWT)
   - Access control rules validation
   - IP whitelist configuration
   - Package filter rule validation
   - Hash verification settings

7. **System Health Checker**
   - CPU usage estimation (goroutine-to-CPU ratio)
   - Memory usage analysis with GC metrics
   - Disk space monitoring with trend analysis
   - File descriptor counting (Linux)
   - System load average tracking
   - Resource exhaustion prediction

## Health Status Types

- **healthy**: All checks passing
- **degraded**: Some non-critical issues detected
- **unhealthy**: Critical issues detected

## Performance Optimizations

### Fast Response Mode
- Core health checks only for sub-100ms responses
- Cached results for frequent queries
- Parallel check execution
- Selective checker activation based on configuration

### Circuit Breaker Pattern
- Per-proxy-type circuit breakers
- Configurable failure thresholds (default: 3 failures)
- Automatic recovery with timeout (default: 30s)
- Graceful degradation to "degraded" status

### Resource Trend Analysis
- Historical data tracking (last 100 measurements)
- Predictive analysis using linear regression
- Early warning system for resource exhaustion
- Configurable thresholds per resource type

## Performance Metrics

### Response Time Targets (Validated in CI)
- **Fast Health Check**: <100ms (P95)
- **Basic Health Check**: <200ms (P95)
- **Comprehensive Health Check**: <500ms (P95)
- **System Resource Check**: <300ms (P95)

### Reliability Targets
- **Health Check Availability**: >99.5%
- **False Positive Rate**: <1%
- **Alert Accuracy**: >95%
- **Recovery Time**: <30s for transient issues

### Scalability Characteristics
- **Concurrent Requests**: 100+ simultaneous health checks
- **Memory Overhead**: <50MB for health system
- **CPU Impact**: <5% during health checks
- **Network Bandwidth**: <1MB/s during checks

## Configuration

Health checks run automatically every 30 seconds in the background.

### Custom Health Check Example

```go
// Implement the HealthChecker interface
type DatabaseChecker struct {
    db *sql.DB
}

func (dc *DatabaseChecker) Name() string {
    return "database"
}

func (dc *DatabaseChecker) Check(ctx context.Context) *health.CheckResult {
    start := time.Now()
    result := &health.CheckResult{
        Name:        dc.Name(),
        Status:      health.StatusHealthy,
        LastChecked: time.Now(),
    }

    // Ping database
    if err := dc.db.PingContext(ctx); err != nil {
        result.Status = health.StatusUnhealthy
        result.Message = fmt.Sprintf("Database ping failed: %v", err)
    } else {
        result.Message = "Database connection is healthy"
    }

    result.Duration = time.Since(start)
    return result
}

// Register the checker
healthService.RegisterChecker(&DatabaseChecker{db: db})
```

## Monitoring Integration

### Prometheus Metrics

Health check results are exposed as Prometheus metrics:

```
# Health status (1 = healthy, 0 = unhealthy)
proxynd_health{check="environment"} 1
proxynd_health{check="disk_space"} 1

# Check duration
proxynd_health_check_duration_seconds{check="environment"} 0.0005
proxynd_health_check_duration_seconds{check="disk_space"} 0.0012
```

### Kubernetes Configuration

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: proxynd
    image: proxynd:latest
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

### Docker Compose Health Check

```yaml
services:
  proxynd:
    image: proxynd:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz?format=simple"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
```

## Best Practices

1. **Use appropriate endpoints**:
   - `/healthz` for detailed monitoring
   - `/health/live` for process health
   - `/health/ready` for service readiness

2. **Monitor all critical dependencies**:
   - Disk space
   - Cache backends
   - Upstream registries

3. **Set reasonable thresholds**:
   - Disk space: 10% or 1GB minimum
   - Response timeouts: 5 seconds

4. **Handle degraded state**:
   - Continue serving with reduced functionality
   - Alert operations team

5. **Secure sensitive endpoints**:
   - `/health/debug` disabled in production
   - Mask sensitive environment variables
