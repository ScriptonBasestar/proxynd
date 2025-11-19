# Configuration Reload Endpoint - Authentication & Authorization

**Version**: 1.0
**Last Updated**: 2025-11-19
**Status**: Production Ready

---

## Overview

The configuration reload endpoint (`POST /api/config/reload`) supports JWT-based authentication and role-based authorization (RBAC) with comprehensive audit logging. This endpoint reloads the ProxyND configuration without requiring a server restart.

## Security Features

### 1. JWT Authentication

The endpoint requires a valid JWT token in the Authorization header when `JWT_SECRET` environment variable is set.

**Environment Variable**:
```bash
export JWT_SECRET="your-secret-key-here"
```

**⚠️ IMPORTANT**: If `JWT_SECRET` is not set, the endpoint operates in development mode without authentication. **Always set JWT_SECRET in production.**

### 2. Role-Based Authorization

The endpoint requires the **admin role** to execute configuration reload operations.

**Supported Roles**:
- `admin` - Full access to all endpoints (required for config reload)
- `user` - Regular user (denied access to config reload)

### 3. Audit Logging

All reload attempts (successful and failed) are logged to the audit system with detailed information:

- User ID and username
- Client IP address
- User agent
- Reload timestamp
- Success/failure status
- Error messages (for failures)

### 4. Plugin Event Notification

After successful configuration reload, the system notifies all registered plugins via the event system:

**Event Type**: `config.reloaded`

**Event Data**:
```json
{
  "timestamp": "2025-11-19T00:00:00Z",
  "config_path": "/etc/proxynd/config"
}
```

This allows plugins to react to configuration changes and update their internal state accordingly.

---

## API Usage

### Request Format

```http
POST /api/config/reload
Authorization: Bearer <jwt-token>
```

**Parameters**: None (body not required)

### Response Format

**Success (200 OK)**:
```json
{
  "success": true,
  "message": "Configuration reloaded successfully",
  "reloaded_at": "2025-11-19T10:30:00Z"
}
```

**Authentication Required (401 Unauthorized)**:
```json
{
  "error": "Missing authorization header",
  "code": "AUTH_MISSING_HEADER"
}
```

**Insufficient Permissions (403 Forbidden)**:
```json
{
  "error": "Insufficient permissions",
  "code": "AUTH_INSUFFICIENT_PERMISSIONS"
}
```

**Rate Limit Exceeded (429 Too Many Requests)**:
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests, please try again later"
}
```

---

## Authentication Flow

### Production Mode (JWT_SECRET Set)

```
1. Client sends request with JWT token
   ↓
2. Rate limiter checks request rate
   ↓
3. JWTMiddleware validates token
   ↓
4. RequireRole checks for "admin" role
   ↓
5. Request proceeds to reloadConfig handler
   ↓
6. Plugin event notification sent
   ↓
7. Audit event logged
   ↓
8. Response returned
```

### Development Mode (No JWT_SECRET)

```
1. Client sends request (no token required)
   ↓
2. Rate limiter checks request rate
   ↓
3. Request proceeds directly to handler
   ↓
4. User logged as "anonymous"
   ↓
5. Plugin event notification sent (if plugin manager available)
   ↓
6. Audit event logged (if audit service available)
   ↓
7. Response returned
```

**⚠️ WARNING**: Development mode should never be used in production.

---

## JWT Token Generation

### Generating Tokens

To generate a JWT token for testing or administration:

```go
import (
    "proxynd/internal/adapters/http/fiber/middleware"
    "time"
)

jwtConfig := middlewares.JWTConfig{
    SecretKey:     "your-secret-key",
    TokenDuration: 24 * time.Hour,
    Issuer:        "proxynd",
}

token, err := middlewares.GenerateJWTToken(
    "admin-user-id",
    "admin",
    []string{"admin"},
    jwtConfig,
)
```

### Token Structure

JWT tokens contain the following claims:

```json
{
  "user_id": "admin123",
  "username": "admin",
  "roles": ["admin"],
  "iat": 1700000000,
  "exp": 1700086400,
  "iss": "proxynd",
  "sub": "admin123"
}
```

---

## Audit Log Format

### Success Event

```json
{
  "id": "audit-event-id",
  "timestamp": "2025-11-19T00:00:00Z",
  "level": "info",
  "event_type": "config.changed",
  "message": "Configuration reloaded successfully",
  "user_id": "admin123",
  "user_email": "",
  "client_ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "resource": "/api/config/reload",
  "action": "reload",
  "result": "success",
  "tags": ["configuration", "config_reload"]
}
```

### Failure Event

```json
{
  "id": "audit-event-id",
  "timestamp": "2025-11-19T00:00:00Z",
  "level": "critical",
  "event_type": "config.changed",
  "message": "Configuration reload failed",
  "user_id": "admin123",
  "client_ip": "192.168.1.100",
  "resource": "/api/config/reload",
  "action": "reload",
  "result": "failure",
  "risk_score": 50,
  "details": {
    "error": "configuration validation failed"
  },
  "tags": ["configuration", "config_reload"]
}
```

---

## Configuration

### Environment Variables

```bash
# Required for production authentication
export JWT_SECRET="long-random-secret-key-change-this"

# Audit configuration (optional)
export AUDIT_ENABLED="true"
export AUDIT_FILE_PATH="logs/audit.log"
export AUDIT_MAX_FILE_SIZE="104857600"  # 100MB
export AUDIT_MAX_FILES="10"
export AUDIT_RETENTION_DAYS="90"
```

### Docker Configuration

```dockerfile
ENV JWT_SECRET="your-secret-key"
ENV AUDIT_ENABLED="true"
ENV AUDIT_FILE_PATH="/var/log/proxynd/audit.log"
```

### Kubernetes Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: proxynd-jwt-secret
type: Opaque
data:
  JWT_SECRET: <base64-encoded-secret>
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxynd
spec:
  template:
    spec:
      containers:
      - name: proxynd
        env:
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: proxynd-jwt-secret
              key: JWT_SECRET
```

---

## Security Best Practices

### 1. JWT Secret Management

✅ **DO**:
- Use a strong, random secret (minimum 32 characters)
- Store secret in environment variables or secret management system
- Rotate secrets periodically
- Use different secrets for different environments

❌ **DON'T**:
- Hard-code secrets in source code
- Share secrets across environments
- Use default or weak secrets
- Commit secrets to version control

### 2. Token Lifetime

- Default: 24 hours
- Recommendation: 8-12 hours for admin tokens
- Use refresh tokens for longer sessions
- Implement token revocation for security incidents

### 3. Audit Log Protection

- Store audit logs in append-only mode
- Enable log shipping to external SIEM
- Implement log integrity checks
- Set appropriate retention periods (90+ days recommended)

### 4. Network Security

- Use HTTPS in production
- Implement rate limiting (enabled by default)
- Use API gateway for additional security layers
- Implement IP whitelisting for admin operations

---

## Rate Limiting

The reload endpoint includes built-in rate limiting to prevent abuse and excessive configuration reloads that could impact system stability.

### Configuration

**Current Settings**:
- **Rate**: 10 requests per minute
- **Burst**: 3 requests (allows short bursts)
- **Scope**: IP-based (per client IP)
- **Log Level**: Warning (rate limit violations logged at WARN level)

### Implementation

The rate limiter is automatically applied to the reload endpoint:

```go
// Rate limiter applied in ConfigRouter
configRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:        "10-M", // 10 requests per minute
    BurstSize:   3,      // Allow burst of 3 requests
    KeyFunc:     nil,    // Use default IP-based limiting
    LogLevel:    "warn", // Log rate limit violations
    ErrorPrefix: "Config Reload",
})

// Middleware chain (production mode)
api.Post("/reload",
    configRateLimiter,                  // Applied first
    middlewares.JWTMiddleware(jwtConfig),
    middlewares.RequireRole("admin"),
    reloadConfigHandler)
```

### Response Format

**Rate Limit Exceeded (429 Too Many Requests)**:
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests, please try again later"
}
```

**Rate Limit Headers**:
All responses include rate limit information in headers:

```
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 7
X-RateLimit-Reset: 1700000060
```

### Behavior

1. **Burst Allowance**: The first 3 requests can be sent immediately
2. **Sustained Rate**: After burst, limited to 10 requests per minute
3. **Per-IP Limiting**: Each client IP has independent rate limits
4. **Reset Window**: Rate limits reset after 60 seconds

### Customization

To customize rate limiting, modify the configuration in `config_router.go`:

```go
// More restrictive (5 requests per minute, no burst)
configRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:      "5-M",
    BurstSize: 1,
})

// More permissive (20 requests per minute, burst of 5)
configRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:      "20-M",
    BurstSize: 5,
})
```

### IP Whitelisting

To exempt specific IPs from rate limiting:

```go
configRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:      "10-M",
    BurstSize: 3,
    Whitelist: []string{"10.0.0.0/8", "192.168.1.100"}, // Internal networks
})
```

### Monitoring

Rate limit violations are logged with details:

```json
{
  "level": "warn",
  "message": "Config Reload rate limit exceeded",
  "client_ip": "192.168.1.100",
  "path": "/api/config/reload",
  "timestamp": "2025-11-19T00:00:00Z"
}
```

---

## Plugin Event System Integration

The configuration reload endpoint integrates with the plugin event system to notify all registered plugins of configuration changes.

### Event Notification Flow

```
1. Config reload request received
   ↓
2. Configuration reloaded
   ↓
3. Event created: EventConfigReloaded
   ↓
4. Plugin manager notifies all EventHandler plugins
   ↓
5. Plugins receive event and update their state
   ↓
6. Response returned to client
```

### Plugin Event Handler Example

Plugins can implement the `EventHandler` interface to receive configuration reload notifications:

```go
type MyPlugin struct {}

func (p *MyPlugin) OnEvent(ctx context.Context, event plugins.Event) error {
    if event.Type == plugins.EventConfigReloaded {
        configPath := event.Data["config_path"].(string)
        timestamp := event.Data["timestamp"].(time.Time)

        // Reload plugin-specific configuration
        return p.reloadConfig(configPath)
    }
    return nil
}
```

### Event Timeout

Plugin event notifications have a 10-second timeout to prevent blocking the reload operation. If a plugin's event handler exceeds this timeout, the error is logged but the reload operation continues successfully.

---

## Testing

### Unit Tests

Authentication tests are provided in `config_router_auth_test.go`:

```bash
go test ./internal/adapters/http/fiber/routers/ -run TestConfigReloadAuthentication
```

### Integration Testing

```bash
# Generate admin token
export JWT_TOKEN=$(curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# Test config reload with auth
curl -X POST http://localhost:8080/api/config/reload \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### Verify Audit Logs

```bash
# View recent audit events
tail -f logs/audit.log | jq 'select(.event_type == "config.changed")'
```

### Verify Plugin Events

Check application logs for plugin event notifications:

```bash
# View plugin event logs
tail -f logs/proxynd.log | grep -i "event.*config.*reload"
```

---

## Troubleshooting

### 401 Unauthorized

**Symptom**: All requests return 401
**Causes**:
- Missing Authorization header
- Invalid JWT token
- Expired token
- Wrong JWT_SECRET

**Solution**:
1. Check Authorization header format: `Bearer <token>`
2. Verify JWT_SECRET matches token generation secret
3. Check token expiration time
4. Regenerate token if needed

### 403 Forbidden

**Symptom**: Authenticated requests return 403
**Causes**:
- User lacks admin role
- Token claims missing roles field

**Solution**:
1. Verify user has "admin" in roles array
2. Check JWT claims structure
3. Grant admin role to user

### 429 Rate Limit Exceeded

**Symptom**: Requests return 429 after several calls
**Causes**:
- Exceeded 10 requests per minute limit
- Burst of more than 3 requests

**Solution**:
1. Wait 60 seconds for rate limit to reset
2. Check rate limit headers in response
3. Reduce request frequency
4. Contact administrator to whitelist IP if needed

### Audit Logs Not Working

**Symptom**: No audit events in logs
**Causes**:
- Audit service not initialized
- AUDIT_ENABLED=false
- Incorrect file path permissions

**Solution**:
1. Check AUDIT_ENABLED environment variable
2. Verify audit log directory exists and is writable
3. Check application startup logs for audit service initialization

### Plugin Events Not Received

**Symptom**: Plugins not responding to config reload
**Causes**:
- Plugin manager not initialized
- Plugin doesn't implement EventHandler interface
- Plugin event handler timeout

**Solution**:
1. Check plugin manager initialization in logs
2. Verify plugin implements EventHandler
3. Check for timeout errors in plugin logs
4. Ensure plugin event handler completes within 10 seconds

---

## Related Documentation

- [Plugin Event System](PLUGIN_OPERATOR_GUIDE.md#event-system)
- [Package Manager Toggle API](API_PM_TOGGLE_AUTHENTICATION.md)
- [Cache Clear API](API_CACHE_CLEAR_AUTHENTICATION.md)
- [JWT Authentication Middleware](../internal/adapters/http/fiber/middleware/jwt_auth.go)
- [Audit Service](../internal/auth/audit/audit_service.go)
- [Plugin Manager](../plugins/manager.go)

---

## Changelog

**v1.0 (2025-11-19)**:
- Initial implementation
- JWT authentication with admin role requirement
- Rate limiting (10 requests per minute, burst of 3)
- Comprehensive audit logging
- Plugin event notification integration
- Development/production mode support
- Integration tests
