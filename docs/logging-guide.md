# ProxyND Logging Guide

## 📋 Overview

This guide provides comprehensive information about ProxyND's advanced logging system, including structured logging, log aggregation, security monitoring, performance tracking, and audit trails.

## 🏗️ Logging Architecture

### Core Components

1. **Structured Logger** - JSON-based structured logging with correlation
2. **Audit Logger** - Comprehensive audit trail for compliance
3. **Security Logger** - Security event detection and logging
4. **Performance Logger** - Performance monitoring and analysis
5. **Log Aggregator** - Automated log analysis and insights
6. **Middleware Integration** - HTTP request/response logging

### Logging Levels

- **DEBUG** - Detailed debugging information
- **INFO** - General operational information
- **WARN** - Warning conditions that need attention
- **ERROR** - Error conditions that affect functionality
- **FATAL** - Critical errors that cause service termination

## 🚀 Quick Start

### Basic Configuration

```yaml
# global.yaml
logging:
  level: "info"
  format: "json"
  output:
    - type: "stdout"
    - type: "file"
      path: "/var/log/proxynd/app.log"
      max_size: 100
      max_age: 7
      max_backups: 10
      compress: true

  correlation:
    enabled: true
    header_name: "X-Correlation-ID"
    generate: true

  middleware:
    enabled: true
    skip_paths: ["/health", "/metrics"]
    log_request_body: false
    log_response_body: false
    max_body_size: 1024
```

### Application Integration

```go
package main

import (
    "context"
    "github.com/scriptonbasestar/proxynd/internal/logging"
)

func main() {
    // Initialize logger
    config := &logging.Config{
        Level:  "info",
        Format: "json",
        Output: []logging.OutputConfig{
            {Type: "stdout"},
            {Type: "file", Path: "/var/log/proxynd/app.log"},
        },
        Correlation: true,
    }

    logger, err := logging.NewLogger(config)
    if err != nil {
        panic(err)
    }

    // Use structured logging
    logger.Info("Service starting",
        logging.String("service", "proxynd"),
        logging.String("version", "1.0.0"),
        logging.Int("port", 8080))

    // Context-aware logging
    ctx := logging.WithCorrelationID(context.Background(), "req-123")
    logger.WithContext(ctx).Info("Processing request")
}
```

## 📊 Structured Logging

### Log Format

All logs are structured in JSON format for easy parsing and analysis:

```json
{
  "timestamp": "2024-01-15T10:30:00.123456Z",
  "level": "INFO",
  "component": "proxy.handler",
  "message": "Request processed successfully",
  "service": "proxynd",
  "version": "1.0.0",
  "hostname": "proxynd-server-01",
  "pid": 12345,
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "req-12345",
  "user_id": "user123",
  "session_id": "sess-456",
  "method": "GET",
  "path": "/api/packages",
  "status": 200,
  "duration": 45.23,
  "cache_hit": true,
  "ip_address": "192.168.1.100",
  "user_agent": "ProxyND-Client/1.0"
}
```

### Correlation and Tracing

ProxyND automatically tracks requests across components using correlation IDs:

```go
// Extract correlation from HTTP headers
ctx := logging.WithCorrelationID(c.UserContext(), c.Get("X-Correlation-ID"))

// Use in service layers
logger.WithContext(ctx).Info("Cache operation",
    logging.String("operation", "get"),
    logging.String("key", cacheKey))

// Propagate to downstream services
req.Header.Set("X-Correlation-ID", logging.GetCorrelationID(ctx))
```

## 🔐 Audit Logging

### Audit Events

ProxyND tracks the following audit events:

- **Authentication** - Login attempts, token validation
- **Authorization** - Access control decisions
- **File Access** - Package downloads and uploads
- **Configuration Changes** - Settings modifications
- **Administrative Actions** - User management, system changes

### Audit Configuration

```yaml
# global.yaml
logging:
  audit:
    enabled: true
    file: "/var/log/proxynd/audit.log"
    level: "info"
    format: "json"
    events:
      - "authentication"
      - "authorization"
      - "file_access"
      - "configuration_change"
      - "admin_action"
    retention_days: 90
    max_file_size: "100MB"
    max_files: 10
```

### Audit Implementation

```go
// Initialize audit logger
auditLogger := logging.NewAuditLogger(logger)

// Log authentication events
auditLogger.LogAuthentication(ctx, userID, "jwt", "success", map[string]interface{}{
    "ip_address": clientIP,
    "user_agent": userAgent,
})

// Log file access
auditLogger.LogFileAccess(ctx, "/packages/example.deb", "download", "success", map[string]interface{}{
    "file_size": 1024000,
    "cache_hit": true,
})

// Log configuration changes
auditLogger.LogConfigurationChange(ctx, "proxy_settings", "update", "success", map[string]interface{}{
    "changed_keys": []string{"cache_ttl", "max_file_size"},
    "old_values": map[string]interface{}{"cache_ttl": 3600},
    "new_values": map[string]interface{}{"cache_ttl": 7200},
})
```

## 🛡️ Security Logging

### Security Event Detection

ProxyND automatically detects and logs security threats:

- **Authentication Failures** - Failed login attempts
- **Unauthorized Access** - Access to restricted resources
- **SQL Injection** - Injection attack patterns
- **XSS Attempts** - Cross-site scripting patterns
- **Brute Force** - Repeated failed attempts
- **Malware Detection** - Suspicious file uploads
- **Rate Limiting** - Excessive request rates

### Security Configuration

```yaml
# global.yaml
logging:
  security:
    enabled: true
    file: "/var/log/proxynd/security.log"
    level: "warn"
    format: "json"
    threat_detection: true
    alert_threshold: 10
    block_after: 50
    retention_days: 365
```

### Security Implementation

```go
// Initialize security logger
securityLogger := logging.NewSecurityLogger(logger, auditLogger)

// Initialize security analyzer
analyzer := logging.NewSecurityAnalyzer(securityLogger)

// Use in middleware
app.Use(func(c *fiber.Ctx) error {
    // Analyze request for threats
    analyzer.AnalyzeRequest(c.UserContext(), c)
    return c.Next()
})

// Manual security logging
securityLogger.LogAuthenticationFailure(ctx, c, "invalid_credentials", map[string]interface{}{
    "attempted_username": username,
    "failure_count": attemptCount,
})

securityLogger.LogUnauthorizedAccess(ctx, c, "/admin/users", map[string]interface{}{
    "required_role": "admin",
    "user_role": "user",
})
```

## 📈 Performance Logging

### Performance Tracking

ProxyND tracks performance metrics across all operations:

- **HTTP Requests** - Response times, throughput
- **Database Queries** - Query execution times
- **Cache Operations** - Hit rates, access times
- **File I/O** - Read/write performance
- **Network Calls** - Upstream service latency
- **Memory Usage** - Heap usage, GC performance

### Performance Configuration

```yaml
# global.yaml
logging:
  performance:
    enabled: true
    file: "/var/log/proxynd/performance.log"
    level: "info"
    slow_threshold: "2s"
    very_slow_threshold: "5s"
    memory_logging: true
    memory_interval: "5m"
    metrics_interval: "1m"
```

### Performance Implementation

```go
// Initialize performance logger
perfLogger := logging.NewPerformanceLogger(logger)

// Track database operations
tracker := perfLogger.TrackDatabaseQuery(ctx, "SELECT * FROM packages WHERE name = ?", []interface{}{packageName})
result, err := db.Query(query, args...)
tracker.Finish(err == nil, getErrorMessage(err))

// Track cache operations
tracker = perfLogger.TrackCacheOperation(ctx, "get", cacheKey)
value, found := cache.Get(cacheKey)
tracker.Finish(true, "")

// Use middleware for automatic HTTP tracking
app.Use(logging.PerformanceMiddleware(perfLogger))

// Manual performance events
perfLogger.LogSlowQuery(ctx, query, duration, 2*time.Second)
perfLogger.LogMemoryUsage(ctx)
```

## 🔍 Log Aggregation and Analysis

### Automated Analysis

ProxyND provides automatic log analysis with insights:

- **Error Patterns** - Common error types and frequencies
- **Security Threats** - Attack patterns and sources
- **Performance Trends** - Response time analysis
- **User Behavior** - Access patterns and usage statistics

### Aggregation Configuration

```yaml
# global.yaml
logging:
  aggregation:
    enabled: true
    log_paths:
      - "/var/log/proxynd/app.log"
      - "/var/log/proxynd/audit.log"
      - "/var/log/proxynd/security.log"
      - "/var/log/proxynd/performance.log"
    output_path: "/var/log/proxynd/analytics"
    analysis_window: "1h"
    patterns:
      http_request: '^(?P<timestamp>\S+)\s+(?P<level>\w+).*HTTP request.*'
      error: '^(?P<timestamp>\S+)\s+(?P<level>ERROR|FATAL)\s+(?P<component>\w+)'
      security: '^(?P<timestamp>\S+)\s+(?P<level>\w+)\s+security\s+(?P<event_type>\w+)'
    max_file_size: 104857600  # 100MB
    retention_period: "720h"  # 30 days
```

### Analysis Implementation

```go
// Initialize log aggregator
aggregatorConfig := &logging.AggregatorConfig{
    LogPaths:       []string{"/var/log/proxynd/app.log"},
    OutputPath:     "/var/log/proxynd/analytics",
    AnalysisWindow: time.Hour,
}

aggregator := logging.NewLogAggregator(logger, aggregatorConfig)

// Start periodic analysis
go aggregator.StartPeriodicAnalysis(ctx)

// Manual analysis
timeRange := logging.TimeRange{
    Start: time.Now().Add(-24 * time.Hour),
    End:   time.Now(),
}

analytics, err := aggregator.AnalyzeLogs(ctx, timeRange)
if err != nil {
    logger.Error("Failed to analyze logs", logging.Error(err))
    return
}

// Export results
err = aggregator.ExportAnalytics(analytics, "/tmp/analytics.json")
```

## 🔧 Configuration Examples

### Development Environment

```yaml
# config/development.yaml
logging:
  level: "debug"
  format: "text"
  output:
    - type: "stdout"

  middleware:
    enabled: true
    log_request_body: true
    log_response_body: true
    max_body_size: 4096

  audit:
    enabled: false

  security:
    enabled: true
    threat_detection: false

  performance:
    enabled: true
    slow_threshold: "500ms"
```

### Production Environment

```yaml
# config/production.yaml
logging:
  level: "info"
  format: "json"
  output:
    - type: "file"
      path: "/var/log/proxynd/app.log"
      max_size: 100
      max_age: 30
      max_backups: 50
      compress: true

  correlation:
    enabled: true

  middleware:
    enabled: true
    skip_success_logs: true
    log_request_body: false
    log_response_body: false

  audit:
    enabled: true
    retention_days: 2555  # 7 years for compliance

  security:
    enabled: true
    threat_detection: true
    alert_threshold: 5
    block_after: 20

  performance:
    enabled: true
    memory_logging: true
    memory_interval: "1m"

  aggregation:
    enabled: true
    analysis_window: "15m"
```

### Component-Specific Configuration

```yaml
# global.yaml
logging:
  level: "info"
  components:
    http:
      level: "info"
      enabled: true
    cache:
      level: "debug"
      enabled: true
      fields:
        cache_type: "redis"
    auth:
      level: "warn"
      enabled: true
      output:
        - type: "file"
          path: "/var/log/proxynd/auth.log"
    proxy:
      level: "info"
      enabled: true
    security:
      level: "warn"
      enabled: true
```

## 📊 Log Analysis Queries

### Common Analysis Patterns

#### Error Analysis
```bash
# Find most common errors
jq -r 'select(.level == "ERROR") | .message' /var/log/proxynd/app.log | \
  sort | uniq -c | sort -nr | head -10

# Error rate by component
jq -r 'select(.level == "ERROR") | .component' /var/log/proxynd/app.log | \
  sort | uniq -c | sort -nr
```

#### Performance Analysis
```bash
# Slow requests (>2s)
jq -r 'select(.duration > 2000) | "\(.timestamp) \(.method) \(.path) \(.duration)ms"' \
  /var/log/proxynd/app.log

# Average response time by endpoint
jq -r 'select(.path) | "\(.path) \(.duration)"' /var/log/proxynd/app.log | \
  awk '{sum[$1] += $2; count[$1]++} END {for (path in sum) print path, sum[path]/count[path]}'
```

#### Security Analysis
```bash
# Authentication failures by IP
jq -r 'select(.message | contains("authentication failed")) | .ip_address' \
  /var/log/proxynd/security.log | sort | uniq -c | sort -nr

# Top threat sources
jq -r 'select(.security_event_type) | "\(.ip_address) \(.security_event_type)"' \
  /var/log/proxynd/security.log | sort | uniq -c | sort -nr
```

#### User Activity Analysis
```bash
# User request patterns
jq -r 'select(.user_id) | "\(.user_id) \(.method) \(.path)"' \
  /var/log/proxynd/audit.log | sort | uniq -c | sort -nr

# Most active users
jq -r 'select(.user_id) | .user_id' /var/log/proxynd/audit.log | \
  sort | uniq -c | sort -nr | head -20
```

## 🔍 Troubleshooting

### Common Issues

#### High Log Volume
```yaml
# Reduce log volume
logging:
  middleware:
    skip_success_logs: true
    skip_paths: ["/health", "/metrics", "/static"]

  sampling:
    enabled: true
    initial: 100
    thereafter: 10
```

#### Missing Correlation IDs
```go
// Ensure middleware is properly configured
app.Use(logging.New(logging.MiddlewareConfig{
    Logger: logger,
}))

// Manual correlation ID injection
ctx := logging.WithCorrelationID(c.UserContext(), c.Get("X-Correlation-ID"))
c.SetUserContext(ctx)
```

#### Log File Rotation Issues
```yaml
# Ensure proper file rotation
logging:
  output:
    - type: "file"
      path: "/var/log/proxynd/app.log"
      max_size: 10    # Smaller files for faster rotation
      max_age: 1      # Daily rotation
      max_backups: 30 # Keep 30 days
      compress: true  # Compress old files
```

#### Performance Impact
```yaml
# Optimize for performance
logging:
  level: "warn"  # Reduce log volume
  format: "json" # Faster than text formatting

  middleware:
    log_request_body: false
    log_response_body: false

  performance:
    memory_interval: "10m"  # Less frequent memory logging
```

## 📚 Best Practices

### Structured Logging
1. **Use Fields Instead of String Formatting**
   ```go
   // Good
   logger.Info("User login successful",
       logging.String("user_id", userID),
       logging.String("ip", clientIP))

   // Avoid
   logger.Info(fmt.Sprintf("User %s login from %s", userID, clientIP))
   ```

2. **Consistent Field Names**
   ```go
   // Use consistent field names across components
   logging.String("user_id", userID)      // Not "userId" or "user"
   logging.String("ip_address", ip)       // Not "ip" or "client_ip"
   logging.Duration("duration", elapsed)  // Not "time" or "elapsed"
   ```

3. **Include Context Information**
   ```go
   logger.WithContext(ctx).WithComponent("auth").Info("Token validated",
       logging.String("token_type", "jwt"),
       logging.Duration("validation_time", elapsed))
   ```

### Security Logging
1. **Never Log Sensitive Data**
   ```go
   // Good - log metadata only
   logger.Info("User authenticated",
       logging.String("user_id", user.ID),
       logging.String("auth_method", "jwt"))

   // Avoid - don't log passwords, tokens, or PII
   logger.Info("Login", logging.String("password", password))
   ```

2. **Use Appropriate Log Levels**
   ```go
   // Failed auth should be WARNING, not ERROR
   securityLogger.LogAuthenticationFailure(ctx, c, "invalid_password", metadata)

   // Successful operations can be INFO
   auditLogger.LogAuthentication(ctx, userID, "jwt", "success", metadata)
   ```

### Performance Logging
1. **Track Critical Operations**
   ```go
   // Always track operations that affect user experience
   tracker := perfLogger.TrackDatabaseQuery(ctx, query, args)
   defer func() {
       tracker.Finish(err == nil, getErrorMessage(err))
   }()
   ```

2. **Use Appropriate Thresholds**
   ```yaml
   performance:
     slow_threshold: "2s"      # Web request threshold
     very_slow_threshold: "5s" # Critical threshold
   ```

### Log Management
1. **Implement Log Retention**
   ```yaml
   audit:
     retention_days: 2555  # 7 years for compliance
   security:
     retention_days: 365   # 1 year for security analysis
   performance:
     retention_days: 90    # 3 months for performance analysis
   ```

2. **Monitor Log Storage**
   ```bash
   # Monitor log directory size
   du -sh /var/log/proxynd/

   # Set up alerts for disk usage
   df -h /var/log | awk 'NR==2 {if ($5 > 80) print "WARNING: Log disk usage high"}'
   ```

---

**Last Updated**: 2024-07-17  
**Review Schedule**: Monthly  
**Next Review**: 2024-08-17
