# Package Manager Toggle Endpoint - Authentication & Authorization

**Version**: 1.0
**Last Updated**: 2025-11-19
**Status**: Production Ready

---

## Overview

The package manager toggle endpoint (`POST /api/v1/pm/:name/toggle`) now supports JWT-based authentication and role-based authorization (RBAC) with comprehensive audit logging.

## Security Features

### 1. JWT Authentication

The endpoint requires a valid JWT token in the Authorization header when `JWT_SECRET` environment variable is set.

**Environment Variable**:
```bash
export JWT_SECRET="your-secret-key-here"
```

**⚠️ IMPORTANT**: If `JWT_SECRET` is not set, the endpoint operates in development mode without authentication. **Always set JWT_SECRET in production.**

### 2. Role-Based Authorization

The endpoint requires the **admin role** to execute package manager state changes.

**Supported Roles**:
- `admin` - Full access to all endpoints (required for PM toggle)
- `user` - Regular user (denied access to PM toggle)

### 3. Audit Logging

All toggle attempts (successful and failed) are logged to the audit system with detailed information:

- User ID and username
- Client IP address
- User agent
- Package manager name
- Previous and new state
- Success/failure status
- Error messages (for failures)

---

## API Usage

### Request Format

```http
POST /api/v1/pm/:name/toggle
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "enabled": true  // or false
}
```

**Parameters**:
- `:name` - Package manager name (maven, npm, docker, pypi, apt, yum, apk)
- `enabled` - (optional) Explicit state. If omitted, toggles current state.

### Response Format

**Success (200 OK)**:
```json
{
  "name": "npm",
  "enabled": true,
  "previous_state": false,
  "message": "Package manager 'npm' enabled successfully",
  "persisted": true
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

**Invalid Package Manager (400 Bad Request)**:
```json
{
  "error": "invalid_package_manager",
  "message": "Unknown package manager: invalid-pm",
  "valid": ["maven", "npm", "docker", "pypi", "apt", "yum", "apk"]
}
```

---

## Authentication Flow

### Production Mode (JWT_SECRET Set)

```
1. Client sends request with JWT token
   ↓
2. JWTMiddleware validates token
   ↓
3. RequireRole checks for "admin" role
   ↓
4. Request proceeds to togglePackageManager handler
   ↓
5. Audit event logged
   ↓
6. Response returned
```

### Development Mode (No JWT_SECRET)

```
1. Client sends request (no token required)
   ↓
2. Request proceeds directly to handler
   ↓
3. User logged as "anonymous"
   ↓
4. Audit event logged (if audit service available)
   ↓
5. Response returned
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
  "message": "Package manager 'npm' enabled",
  "user_id": "admin123",
  "user_email": "",
  "client_ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "resource": "/api/v1/pm/npm/toggle",
  "action": "toggle",
  "result": "success",
  "details": {
    "package_manager": "npm",
    "previous_state": false,
    "new_state": true
  },
  "tags": ["configuration", "package_manager"]
}
```

### Failure Event

```json
{
  "id": "audit-event-id",
  "timestamp": "2025-11-19T00:00:00Z",
  "level": "critical",
  "event_type": "config.changed",
  "message": "Failed to persist package manager 'npm' state change",
  "user_id": "admin123",
  "client_ip": "192.168.1.100",
  "resource": "/api/v1/pm/npm/toggle",
  "action": "toggle",
  "result": "failure",
  "risk_score": 50,
  "details": {
    "package_manager": "npm",
    "previous_state": false,
    "new_state": true,
    "error": "save_failed: disk full"
  },
  "tags": ["configuration", "package_manager"]
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
- Implement rate limiting (see below)
- Use API gateway for additional security layers
- Implement IP whitelisting for admin operations

---

## Rate Limiting

The toggle endpoint includes built-in rate limiting to prevent abuse and excessive configuration changes that could impact system stability.

### Configuration

**Current Settings**:
- **Rate**: 10 requests per minute
- **Burst**: 3 requests (allows short bursts)
- **Scope**: IP-based (per client IP)
- **Log Level**: Warning (rate limit violations logged at WARN level)

### Implementation

The rate limiter is automatically applied to the toggle endpoint:

```go
// Rate limiter applied in SetupAPIv1Routes
configRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:        "10-M", // 10 requests per minute
    BurstSize:   3,      // Allow burst of 3 requests
    KeyFunc:     nil,    // Use default IP-based limiting
    LogLevel:    "warn", // Log rate limit violations
    ErrorPrefix: "PM Toggle",
})

// Middleware chain (production mode)
api.Post("/pm/:name/toggle",
    configRateLimiter,              // Applied first
    middlewares.JWTMiddleware(jwtConfig),
    middlewares.RequireRole("admin"),
    togglePackageManagerHandler)
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

To customize rate limiting, modify the configuration in `api_v1_router.go`:

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
  "message": "PM Toggle rate limit exceeded",
  "client_ip": "192.168.1.100",
  "path": "/api/v1/pm/npm/toggle",
  "timestamp": "2025-11-19T00:00:00Z"
}
```

---

## Testing

### Unit Tests

Authentication tests are provided in `api_v1_router_auth_test.go`:

```bash
go test ./internal/adapters/http/fiber/routers/ -run TestToggleEndpointAuthentication
```

Rate limiting tests are provided in `api_v1_router_ratelimit_test.go`:

```bash
# Run all rate limiting tests
go test ./internal/adapters/http/fiber/routers/ -run TestToggleEndpointRateLimiting

# Run specific rate limiting test
go test ./internal/adapters/http/fiber/routers/ -run TestRateLimitConfiguration
```

### Integration Testing

```bash
# Generate admin token
export JWT_TOKEN=$(curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# Test toggle with auth
curl -X POST http://localhost:8080/api/v1/pm/npm/toggle \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'
```

### Verify Audit Logs

```bash
# View recent audit events
tail -f logs/audit.log | jq 'select(.event_type == "config.changed")'
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

---

## Related Documentation

- [Plugin Event System](PLUGIN_OPERATOR_GUIDE.md#event-system)
- [API PM Toggle Persistence](API_PM_TOGGLE_PERSISTENCE.md)
- [JWT Authentication Middleware](../internal/adapters/http/fiber/middleware/jwt_auth.go)
- [Audit Service](../internal/auth/audit/audit_service.go)

---

## Changelog

**v1.1 (2025-11-19)**:
- Added rate limiting to toggle endpoint
- 10 requests per minute with burst of 3
- IP-based rate limiting with whitelist support
- Rate limit headers in responses
- Comprehensive rate limiting tests

**v1.0 (2025-11-19)**:
- Initial implementation
- JWT authentication with admin role requirement
- Comprehensive audit logging
- Development/production mode support
- Integration tests
