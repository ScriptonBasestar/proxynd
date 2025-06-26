# Structured Logging Guide

ProxyND uses structured logging with JSON output by default for better observability and log analysis.

## Features

- **Structured JSON logging** with consistent field names
- **Multiple log levels**: debug, info, warn, error, fatal, panic
- **Contextual logging** with request IDs and user information
- **Log rotation** with size and age-based policies
- **Performance optimized** with zero-allocation logger
- **Multiple outputs**: console, file, or both
- **Request correlation** across all log entries
- **Sensitive data masking** for security

## Configuration

### Environment Variables

```bash
# Log level: debug, info, warn, error, fatal, panic
LOG_LEVEL=info

# Log format: json, text, console
LOG_FORMAT=json

# Log output: stdout, stderr, file, both
LOG_OUTPUT=stdout

# Log file path (when output is file or both)
LOG_FILE=/var/log/proxynd/app.log

# Access log path
ACCESS_LOG_PATH=/var/log/proxynd/access.log

# Time format
LOG_TIME_FORMAT=2006-01-02T15:04:05.000Z07:00

# Environment name
ENVIRONMENT=production

# Service version
VERSION=1.0.0

# Enable log sampling (for high-volume environments)
LOG_SAMPLING=false
```

### Configuration File

```yaml
# config.yaml
logging:
  level: info
  format: json
  output: both
  time_format: "2006-01-02T15:04:05.000Z07:00"
  
  file:
    path: ./logs/proxynd.log
    max_size: 100       # MB
    max_backups: 10
    max_age: 30         # days
    compress: true
  
  default_fields:
    service: proxynd
    environment: production
    version: 1.0.0
  
  sampling:
    enabled: false
    initial: 100
    thereafter: 100
  
  access_log:
    enabled: true
    path: ./logs/access.log
    format: json
    rotate_daily: true
```

## Log Levels

| Level | Usage |
|-------|-------|
| `debug` | Detailed information for debugging |
| `info` | General informational messages |
| `warn` | Warning messages for potentially harmful situations |
| `error` | Error messages for failures |
| `fatal` | Critical errors that cause program exit |
| `panic` | Critical errors that cause panic |

## Structured Log Fields

### Common Fields

All log entries include these fields:

```json
{
  "level": "info",
  "time": "2024-01-20T10:30:00.000Z",
  "hostname": "server-1",
  "pid": 12345,
  "service": "proxynd",
  "environment": "production",
  "version": "1.0.0",
  "msg": "Log message"
}
```

### Request Context Fields

For HTTP requests:

```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/proxy/npm/express",
  "ip": "192.168.1.100",
  "user_agent": "npm/8.0.0",
  "username": "john.doe",
  "proxy_type": "npm",
  "status": 200,
  "duration_ms": 150,
  "response_size": 2048
}
```

### Cache Fields

```json
{
  "cache_hit": true,
  "cache_backend": "file",
  "bandwidth_saved": 2048
}
```

### Proxy Fields

```json
{
  "upstream": "registry.npmjs.org",
  "upstream_duration_ms": 120,
  "package_path": "express/-/express-4.18.2.tgz"
}
```

### Verification Fields

```json
{
  "hash_verified": true,
  "package_verified": true
}
```

## Usage Examples

### Basic Logging

```go
import "proxynd/logging"

// Get logger
logger := logging.GetLogger()

// Log messages
logger.Info("Server started")
logger.Warn("Cache nearly full", logging.F("usage_percent", 85))
logger.Error("Failed to connect", logging.F("error", err))

// With multiple fields
logger.Info("Request processed",
    logging.F("user_id", userID),
    logging.F("action", "download"),
    logging.F("package", packageName),
)
```

### Component Logger

```go
// Create logger for specific component
logger := logging.NewLogger("cache-manager")

logger.Info("Cache initialized",
    logging.F("backend", "s3"),
    logging.F("bucket", bucketName),
)
```

### Request Context Logging

```go
// In Fiber handler
func Handler(c *fiber.Ctx) error {
    // Get request logger
    logger := logging.GetRequestLogger(c)
    
    logger.Info("Processing request",
        logging.F("package", c.Params("package")),
    )
    
    // Logger automatically includes request_id
    return nil
}
```

### Contextual Logging

```go
// Create logger with persistent fields
logger := logging.GetLogger().WithFields(
    logging.F("component", "auth"),
    logging.F("version", "2.0"),
)

// All subsequent logs include these fields
logger.Info("Authentication started")
logger.Info("User authenticated", logging.F("user_id", userID))
```

## Middleware Integration

### Structured Access Logging

```go
// Enable structured access logging
app.Use(middlewares.StructuredAccessLog(
    middlewares.StructuredAccessLogConfig{
        Logger: logging.NewLogger("access"),
        SkipPaths: []string{"/healthz", "/metrics"},
    },
))
```

### Error Logging

```go
// Automatic error logging
app.Use(logging.ErrorLogger())
```

### Panic Recovery

```go
// Log and recover from panics
app.Use(logging.RecoveryLogger())
```

## Log Output Examples

### JSON Format (Default)

```json
{"level":"info","time":"2024-01-20T10:30:00.000Z","hostname":"prod-1","pid":12345,"service":"proxynd","environment":"production","request_id":"550e8400","method":"GET","path":"/proxy/npm/express","ip":"192.168.1.100","proxy_type":"npm","status":200,"duration_ms":150,"cache_hit":true,"msg":"GET /proxy/npm/express"}
```

### Console Format

```
10:30AM INF GET /proxy/npm/express request_id=550e8400 method=GET path=/proxy/npm/express status=200 duration_ms=150 cache_hit=true
```

### Text Format

```
2024-01-20T10:30:00.000Z INFO  [request_id=550e8400] GET /proxy/npm/express status=200 duration=150ms cache_hit=true
```

## Log Rotation

Logs are automatically rotated based on:

- **Size**: When log file reaches max_size (default: 100MB)
- **Age**: Files older than max_age days are deleted (default: 30 days)
- **Count**: Maximum number of backup files (default: 10)
- **Compression**: Old logs are gzipped (default: true)

## Performance Considerations

### Log Sampling

For high-traffic environments, enable log sampling:

```yaml
sampling:
  enabled: true
  initial: 100      # Log first 100 of each type
  thereafter: 100   # Then log 1 in 100
```

### Asynchronous Logging

The logger uses buffered I/O for file output to minimize performance impact.

### Zero Allocation

The structured logger is designed for zero heap allocations in hot paths.

## Integration with Log Management

### Elasticsearch/Logstash

The JSON format is directly compatible with Elasticsearch:

```json
{
  "mappings": {
    "properties": {
      "time": { "type": "date" },
      "level": { "type": "keyword" },
      "request_id": { "type": "keyword" },
      "method": { "type": "keyword" },
      "status": { "type": "integer" },
      "duration_ms": { "type": "long" },
      "msg": { "type": "text" }
    }
  }
}
```

### Fluentd

Example Fluentd configuration:

```xml
<source>
  @type tail
  path /var/log/proxynd/*.log
  pos_file /var/log/td-agent/proxynd.pos
  tag proxynd.*
  <parse>
    @type json
    time_key time
    time_format %Y-%m-%dT%H:%M:%S.%L%z
  </parse>
</source>
```

### CloudWatch Logs

The JSON format works seamlessly with CloudWatch Logs Insights:

```sql
fields @timestamp, request_id, method, path, status, duration_ms
| filter status >= 400
| stats count() by bin(5m)
```

## Security

### Sensitive Data Masking

Headers and fields containing sensitive data are automatically masked:

- Authorization headers
- Cookies
- API tokens
- Passwords

### Audit Logging

Critical security events are logged at appropriate levels:

```go
logger.Warn("Authentication failed",
    logging.F("username", username),
    logging.F("ip", clientIP),
    logging.F("reason", "invalid_password"),
)
```

## Migration from Standard Logging

### Replace log.Printf

```go
// Old
log.Printf("Processing package: %s", packageName)

// New
logger.Info("Processing package", logging.F("package", packageName))
```

### Replace log.Fatal

```go
// Old
log.Fatal("Failed to start server")

// New
logger.Fatal("Failed to start server")
```

### Legacy Compatibility

For gradual migration:

```go
// Create legacy-compatible logger
legacy := logging.NewLegacyLogger("component")
legacy.Printf("Old style logging: %s", value)
```

## Best Practices

1. **Use structured fields** instead of string formatting
2. **Include request_id** in all request-related logs
3. **Log at appropriate levels** (don't use info for errors)
4. **Add contextual information** with fields
5. **Avoid logging sensitive data**
6. **Use component loggers** for better organization
7. **Configure log sampling** for high-volume services
8. **Monitor log volume** and adjust levels as needed

## Troubleshooting

### Logs not appearing

1. Check `LOG_LEVEL` environment variable
2. Verify `LOG_OUTPUT` setting
3. Check file permissions for log directory
4. Ensure disk space is available

### Performance issues

1. Enable log sampling
2. Increase log level (e.g., info to warn)
3. Use file output instead of console
4. Enable compression for old logs

### Log parsing errors

1. Ensure consistent JSON format
2. Check for special characters in messages
3. Verify time format configuration