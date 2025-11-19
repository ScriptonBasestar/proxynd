# Cache Clear Endpoints - Authentication & Authorization

**Version**: 1.0
**Last Updated**: 2025-11-19
**Status**: Production Ready

---

## Overview

The cache management endpoints provide comprehensive control over ProxyND's caching system with JWT-based authentication and role-based authorization (RBAC). This document covers cache clear operations with comprehensive audit logging and plugin event notifications.

## Endpoints Summary

### Public Endpoints (No Authentication Required)

- `GET /api/cache/list` - List cached items with pagination
- `GET /api/cache/size` - Get cache size and statistics
- `GET /api/cache/stats` - Get cache statistics
- `GET /api/cache/ttl` - Get TTL policy information

### Protected Endpoints (Admin Only)

- `DELETE /api/cache/clear` - Clear all cache
- `DELETE /api/cache/clear/:type` - Clear cache by proxy type
- `DELETE /api/cache/item/*` - Delete specific cache item

---

## Security Features

### 1. JWT Authentication

The DELETE endpoints require a valid JWT token in the Authorization header when `JWT_SECRET` environment variable is set.

**Environment Variable**:
```bash
export JWT_SECRET="your-secret-key-here"
```

**⚠️ IMPORTANT**: If `JWT_SECRET` is not set, the endpoints operate in development mode without authentication. **Always set JWT_SECRET in production.**

### 2. Role-Based Authorization

The DELETE endpoints require the **admin role** to execute cache clear operations.

**Supported Roles**:
- `admin` - Full access to all endpoints (required for cache operations)
- `user` - Regular user (can only access public GET endpoints)

### 3. Audit Logging

All cache clear attempts (successful and failed) are logged to the audit system with detailed information:

- User ID and username
- Client IP address
- User agent
- Cache type or item path
- Clear timestamp
- Success/failure status
- Error messages (for failures)

### 4. Plugin Event Notification

After successful cache clear operations, the system notifies all registered plugins via the event system:

**Event Type**: `cache.cleared`

**Event Data**:
```json
{
  "cache_type": "npm",
  "timestamp": "2025-11-19T00:00:00Z"
}
```

This allows plugins to react to cache changes and update their internal state accordingly.

---

## API Usage

### Clear All Cache

**Request**:
```http
DELETE /api/cache/clear?confirm=true
Authorization: Bearer <jwt-token>
```

**Query Parameters**:
- `confirm` (required) - Must be "true" to proceed with clear operation
- `older_than` (optional) - Clear only items older than specified duration (e.g., "24h", "7d")
- `size_limit` (optional) - Clear only if total size exceeds limit

**Success Response (200 OK)**:
```json
{
  "success": true,
  "message": "All cache cleared successfully",
  "cleared_at": "2025-11-19T10:30:00Z"
}
```

**With Filters**:
```json
{
  "success": true,
  "message": "All cache cleared successfully",
  "cleared_at": "2025-11-19T10:30:00Z",
  "filter": "older than 24h",
  "size_limit": "1GB"
}
```

### Clear Cache by Type

**Request**:
```http
DELETE /api/cache/clear/:type?confirm=true
Authorization: Bearer <jwt-token>
```

**Path Parameters**:
- `:type` - Proxy type: `maven`, `npm`, `docker`, `pypi`, `apt`, `yum`, `apk`, `gem`

**Query Parameters**:
- `confirm` (required) - Must be "true" to proceed
- `older_than` (optional) - Clear only old items
- `size_limit` (optional) - Clear if size exceeds limit

**Success Response (200 OK)**:
```json
{
  "success": true,
  "message": "npm cache cleared successfully",
  "type": "npm",
  "cleared_at": "2025-11-19T10:30:00Z"
}
```

### Delete Specific Cache Item

**Request**:
```http
DELETE /api/cache/item/<path>
Authorization: Bearer <jwt-token>
```

**Path Parameters**:
- `<path>` - Full path to cache item (e.g., `npm/@scope/package/1.0.0`)

**Success Response (200 OK)**:
```json
{
  "success": true,
  "message": "Cache item deleted successfully",
  "path": "npm/@scope/package/1.0.0",
  "deleted_at": "2025-11-19T10:30:00Z"
}
```

### Get Cache List

**Request** (No auth required):
```http
GET /api/cache/list?type=npm&limit=100&offset=0
```

**Query Parameters**:
- `type` (optional) - Filter by proxy type
- `limit` (optional, default: 100) - Number of items to return
- `offset` (optional, default: 0) - Pagination offset

**Response (200 OK)**:
```json
{
  "items": [
    {
      "key": "npm/@scope/package/1.0.0",
      "proxy_type": "npm",
      "path": "/storage/proxy/npm/@scope/package/1.0.0",
      "size": 1024000,
      "created_at": "2025-11-19T10:00:00Z",
      "accessed_at": "2025-11-19T10:30:00Z",
      "ttl": "24h",
      "content_type": "application/json"
    }
  ],
  "total": 1
}
```

### Get Cache Size

**Request** (No auth required):
```http
GET /api/cache/size
```

**Response (200 OK)**:
```json
{
  "total_size": 10737418240,
  "total_items": 5000,
  "size_by_type": {
    "npm": 5368709120,
    "maven": 3221225472,
    "docker": 2147483648
  },
  "items_by_type": {
    "npm": 3000,
    "maven": 1500,
    "docker": 500
  },
  "disk_usage": {
    "used": 10737418240,
    "available": 53687091200,
    "total": 107374182400,
    "used_percent": 10.0
  },
  "type_details": {
    "npm": {
      "enabled": true,
      "config_path": "/etc/proxynd/config/npm.yaml",
      "cache_path": "/storage/proxy/npm",
      "proxy_count": 5
    }
  }
}
```

### Get Cache Statistics

**Request** (No auth required):
```http
GET /api/cache/stats
```

**Response (200 OK)**:
```json
{
  "hits": 10000,
  "misses": 2000,
  "hit_rate": 83.33,
  "total_requests": 12000,
  "cache_size_bytes": 10737418240,
  "cache_items": 5000,
  "evictions": 500,
  "last_clear": "2025-11-19T00:00:00Z"
}
```

### Get TTL Policy

**Request** (No auth required):
```http
GET /api/cache/ttl
```

**Response (200 OK)**:
```json
{
  "global_ttl": 86400,
  "package_ttls": {
    "npm": 86400,
    "maven": 172800,
    "docker": 259200
  },
  "pattern_ttls": {
    "SNAPSHOT": 3600,
    "dev": 1800,
    "alpha": 7200,
    "beta": 14400,
    "rc": 21600
  },
  "metadata_ttls": {
    "Packages": 3600,
    "Release": 1800,
    "repomd.xml": 1800
  },
  "use_cache_headers": true,
  "max_cache_header_ttl": 2592000,
  "min_cache_header_ttl": 300,
  "stale_while_revalidate": true,
  "stale_max_age": 86400,
  "last_updated": "2025-11-19T00:00:00Z",
  "configuration_source": "global.yaml"
}
```

---

## Error Responses

**Missing Confirmation (400 Bad Request)**:
```json
{
  "error": "Confirmation required. Add ?confirm=true to proceed"
}
```

**Invalid Proxy Type (400 Bad Request)**:
```json
{
  "error": "Invalid proxy type: invalid-type"
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

**Configuration Not Available (500 Internal Server Error)**:
```json
{
  "error": "Configuration not available"
}
```

---

## Supported Proxy Types

The cache system supports the following proxy types:

1. **npm** - Node Package Manager
2. **maven** - Apache Maven
3. **docker** - Docker Registry
4. **pypi** (or **pip**) - Python Package Index
5. **apt** - Debian/Ubuntu APT
6. **yum** - RedHat/CentOS YUM
7. **apk** - Alpine APK
8. **gem** - Ruby Gems

Each proxy type maintains its own cache directory and statistics.

---

## Authentication Flow

### Production Mode (JWT_SECRET Set)

```
1. Client sends DELETE request with JWT token and ?confirm=true
   ↓
2. Rate limiter checks request rate
   ↓
3. JWTMiddleware validates token
   ↓
4. RequireRole checks for "admin" role
   ↓
5. Request proceeds to cache clear handler
   ↓
6. Cache cleared
   ↓
7. Plugin event notification sent
   ↓
8. Audit event logged
   ↓
9. Response returned
```

### Development Mode (No JWT_SECRET)

```
1. Client sends DELETE request with ?confirm=true
   ↓
2. Rate limiter checks request rate
   ↓
3. Request proceeds directly to handler
   ↓
4. User logged as "anonymous"
   ↓
5. Cache cleared
   ↓
6. Plugin event notification sent (if plugin manager available)
   ↓
7. Audit event logged (if audit service available)
   ↓
8. Response returned
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

## Rate Limiting

The cache clear endpoints include built-in rate limiting to prevent abuse and excessive cache operations that could impact system performance.

### Configuration

**Current Settings**:
- **Rate**: 10 requests per minute
- **Burst**: 3 requests (allows short bursts)
- **Scope**: IP-based (per client IP)
- **Log Level**: Warning (rate limit violations logged at WARN level)

### Implementation

The rate limiter is automatically applied to all DELETE endpoints:

```go
// Rate limiter applied in CacheRouter
cacheRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:        "10-M", // 10 requests per minute
    BurstSize:   3,      // Allow burst of 3 requests
    KeyFunc:     nil,    // Use default IP-based limiting
    LogLevel:    "warn", // Log rate limit violations
    ErrorPrefix: "Cache Clear",
})

// Middleware chain (production mode)
api.Delete("/clear",
    cacheRateLimiter,                   // Applied first
    middlewares.JWTMiddleware(jwtConfig),
    middlewares.RequireRole("admin"),
    clearAllCacheHandler)
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

To customize rate limiting, modify the configuration in `cache_router.go`:

```go
// More restrictive (5 requests per minute, no burst)
cacheRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:      "5-M",
    BurstSize: 1,
})

// More permissive (20 requests per minute, burst of 5)
cacheRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
    Rate:      "20-M",
    BurstSize: 5,
})
```

### IP Whitelisting

To exempt specific IPs from rate limiting:

```go
cacheRateLimiter := middlewares.NewEnhancedRateLimiter(middlewares.RateLimiterConfig{
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
  "message": "Cache Clear rate limit exceeded",
  "client_ip": "192.168.1.100",
  "path": "/api/cache/clear",
  "timestamp": "2025-11-19T00:00:00Z"
}
```

---

## Plugin Event System Integration

The cache clear endpoints integrate with the plugin event system to notify all registered plugins of cache operations.

### Event Notification Flow

```
1. Cache clear request received
   ↓
2. Cache cleared
   ↓
3. Event created: EventCacheCleared
   ↓
4. Plugin manager notifies all EventHandler plugins
   ↓
5. Plugins receive event and update their state
   ↓
6. Response returned to client
```

### Plugin Event Handler Example

Plugins can implement the `EventHandler` interface to receive cache clear notifications:

```go
type MyPlugin struct {}

func (p *MyPlugin) OnEvent(ctx context.Context, event plugins.Event) error {
    if event.Type == plugins.EventCacheCleared {
        cacheType := event.Data["cache_type"].(string)
        timestamp := event.Data["timestamp"].(time.Time)

        // React to cache clear (e.g., invalidate plugin caches)
        return p.handleCacheClear(cacheType)
    }
    return nil
}
```

### Event Timeout

Plugin event notifications have a 10-second timeout to prevent blocking the cache clear operation. If a plugin's event handler exceeds this timeout, the error is logged but the clear operation continues successfully.

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

### 3. Cache Clear Confirmation

**Always require confirmation** for destructive operations:
- Use `?confirm=true` query parameter
- Prevents accidental cache clears
- Provides additional safety layer

### 4. Network Security

- Use HTTPS in production
- Implement rate limiting (enabled by default)
- Use API gateway for additional security layers
- Implement IP whitelisting for admin operations

---

## Testing

### Unit Tests

Authentication tests are provided in `cache_router_auth_test.go` (if exists) or integration tests:

```bash
go test ./internal/adapters/http/fiber/routers/ -run TestCacheAuthentication
```

### Integration Testing

```bash
# Generate admin token
export JWT_TOKEN=$(curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# Test cache clear with auth
curl -X DELETE "http://localhost:8080/api/cache/clear?confirm=true" \
  -H "Authorization: Bearer $JWT_TOKEN"

# Clear specific type
curl -X DELETE "http://localhost:8080/api/cache/clear/npm?confirm=true" \
  -H "Authorization: Bearer $JWT_TOKEN"

# List cache (no auth required)
curl http://localhost:8080/api/cache/list?type=npm&limit=10

# Get cache size (no auth required)
curl http://localhost:8080/api/cache/size

# Get TTL policy (no auth required)
curl http://localhost:8080/api/cache/ttl
```

### Verify Plugin Events

Check application logs for plugin event notifications:

```bash
# View plugin event logs
tail -f logs/proxynd.log | grep -i "event.*cache.*clear"
```

---

## Troubleshooting

### 400 Missing Confirmation

**Symptom**: Requests return 400 with "Confirmation required"
**Cause**: Missing `?confirm=true` query parameter

**Solution**:
Add `?confirm=true` to the request URL:
```bash
curl -X DELETE "http://localhost:8080/api/cache/clear?confirm=true" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### 400 Invalid Proxy Type

**Symptom**: Requests return 400 with "Invalid proxy type"
**Cause**: Unsupported proxy type in path parameter

**Solution**:
Use one of the supported types:
- npm, maven, docker, pypi, apt, yum, apk, gem

### 401 Unauthorized

**Symptom**: DELETE requests return 401
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

### 500 Configuration Not Available

**Symptom**: Requests return 500 with "Configuration not available"
**Causes**:
- Global configuration file not found
- Configuration directory not accessible
- Incorrect CONFIG_DIR environment variable

**Solution**:
1. Check CONFIG_DIR environment variable is set
2. Verify config files exist in configured directory
3. Check file permissions
4. Review application startup logs

### Plugin Events Not Received

**Symptom**: Plugins not responding to cache clear
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

## Cache Clear Best Practices

### 1. Regular Maintenance

- Schedule regular cache cleanup for old items
- Use `older_than` parameter to clear stale cache
- Monitor disk usage and clear proactively

### 2. Type-Specific Clearing

- Clear by proxy type instead of all cache when possible
- Reduces impact on other package managers
- Faster operation with less system load

### 3. Monitoring

- Monitor cache hit rates after clearing
- Track cache size growth over time
- Set up alerts for low disk space

### 4. Backup Considerations

- Consider backing up critical cache before clearing
- Document cache clear procedures
- Test cache clear in staging before production

---

## Related Documentation

- [Package Manager Toggle API](API_PM_TOGGLE_AUTHENTICATION.md)
- [Configuration Reload API](API_CONFIG_RELOAD_AUTHENTICATION.md)
- [Plugin Event System](PLUGIN_OPERATOR_GUIDE.md#event-system)
- [JWT Authentication Middleware](../internal/adapters/http/fiber/middleware/jwt_auth.go)
- [Audit Service](../internal/auth/audit/audit_service.go)
- [Plugin Manager](../plugins/manager.go)
- [Cache Configuration](../docs/20-configuration/cache-configuration.md)

---

## Changelog

**v1.0 (2025-11-19)**:
- Initial implementation
- JWT authentication with admin role requirement for DELETE operations
- Public GET endpoints for cache information
- Rate limiting (10 requests per minute, burst of 3)
- Comprehensive audit logging
- Plugin event notification integration
- Support for 8 proxy types
- Development/production mode support
- Cache clear confirmation requirement
- Advanced filtering options (older_than, size_limit)
