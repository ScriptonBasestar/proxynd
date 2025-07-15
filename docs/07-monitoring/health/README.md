# Health Check API

ProxyND provides comprehensive health check endpoints for monitoring service health and readiness.

## Endpoints

### Main Health Check

**GET /healthz**

Returns detailed health status of all components.

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

Indicates whether the application is running.

**Response (200 OK):**
```json
{
  "status": "alive"
}
```

#### Readiness Probe

**GET /health/ready**

Indicates whether the application is ready to serve requests.

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

4. **HTTP Endpoint Checker**
   - Monitors upstream registry availability
   - Checks: NPM registry, PyPI registry (when enabled)

5. **Cache Backend Checker**
   - Verifies cache backend connectivity
   - Supports: file, S3, Redis backends

## Health Status Types

- **healthy**: All checks passing
- **degraded**: Some non-critical issues detected
- **unhealthy**: Critical issues detected

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