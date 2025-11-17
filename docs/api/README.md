# ProxyND Enterprise API Documentation

This directory contains comprehensive API documentation for ProxyND's Enterprise features.

## Documentation Files

### OpenAPI Specification

- **`enterprise-api-spec.yaml`**: Complete OpenAPI 3.0 specification for all 47 Enterprise API endpoints
  - RBAC (12 endpoints): Role-based access control management
  - Audit (8 endpoints): Audit logging and compliance
  - Analytics (10 endpoints): Usage statistics and reporting
  - Security (8 endpoints): Vulnerability scanning and license compliance
  - Alerts (5 endpoints): Alert management and notification rules
  - License (4 endpoints): License validation and feature management

### Interactive Documentation

When the ProxyND server is running, you can access interactive API documentation at:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **API Docs Alias**: `http://localhost:8080/api/v1/docs`

Both URLs provide the same interactive documentation interface where you can:
- Browse all available endpoints
- View request/response schemas
- Test API endpoints directly from the browser
- Download the OpenAPI specification

## Using the API Documentation

### 1. View Documentation Locally

Start the ProxyND server:

```bash
# Development mode
make dev-run

# Or build and run
make build
./tmp/bin/proxynd
```

Then open your browser to:
```
http://localhost:8080/api/v1/docs
```

### 2. Authentication

All Enterprise API endpoints require a valid enterprise license token. Include it in the Authorization header:

```bash
curl -H "Authorization: Bearer YOUR_LICENSE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/roles
```

### 3. Example API Calls

#### List Roles (RBAC)
```bash
curl -X GET "http://localhost:8080/api/v1/enterprise/rbac/roles?page=1&per_page=20" \
  -H "Authorization: Bearer YOUR_LICENSE_TOKEN"
```

#### Create a Role
```bash
curl -X POST "http://localhost:8080/api/v1/enterprise/rbac/roles" \
  -H "Authorization: Bearer YOUR_LICENSE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "developer",
    "description": "Developer role with read/write access",
    "permissions": ["read:packages", "write:packages"]
  }'
```

#### Get Audit Events
```bash
curl -X GET "http://localhost:8080/api/v1/enterprise/audit/events?page=1&per_page=50" \
  -H "Authorization: Bearer YOUR_LICENSE_TOKEN"
```

#### Get Analytics Overview
```bash
curl -X GET "http://localhost:8080/api/v1/enterprise/analytics/overview" \
  -H "Authorization: Bearer YOUR_LICENSE_TOKEN"
```

#### Trigger Security Scan
```bash
curl -X POST "http://localhost:8080/api/v1/enterprise/security/scan" \
  -H "Authorization: Bearer YOUR_LICENSE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "all",
    "priority": "high"
  }'
```

#### List Alerts
```bash
curl -X GET "http://localhost:8080/api/v1/enterprise/alerts?status=active&severity=critical" \
  -H "Authorization: Bearer YOUR_LICENSE_TOKEN"
```

## API Response Format

All Enterprise API endpoints follow a consistent response format:

### Success Response
```json
{
  "success": true,
  "data": {
    // Response data here
  },
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  },
  "metadata": {
    "timestamp": "2025-01-17T12:00:00Z",
    "request_id": "req_1234567890"
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "The requested resource was not found"
  },
  "metadata": {
    "timestamp": "2025-01-17T12:00:00Z",
    "request_id": "req_1234567890"
  }
}
```

## Pagination

Endpoints that return lists support pagination with the following query parameters:

- `page`: Page number (default: 1, minimum: 1)
- `per_page`: Items per page (default: 20, minimum: 1, maximum: 100)

Example:
```bash
curl "http://localhost:8080/api/v1/enterprise/rbac/roles?page=2&per_page=50"
```

## Filtering and Searching

Many endpoints support filtering and searching:

### Audit Events
```bash
# Filter by date range
curl "http://localhost:8080/api/v1/enterprise/audit/events?start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z"

# Search
curl "http://localhost:8080/api/v1/enterprise/audit/events/search?query=login&user_id=user_123"
```

### Vulnerabilities
```bash
# Filter by severity
curl "http://localhost:8080/api/v1/enterprise/security/vulnerabilities?severity=critical"
```

### Alerts
```bash
# Filter by status and severity
curl "http://localhost:8080/api/v1/enterprise/alerts?status=active&severity=warning"
```

## Export Functionality

Some endpoints support exporting data in multiple formats:

```bash
# Export audit logs as JSON
curl -X POST "http://localhost:8080/api/v1/enterprise/audit/export" \
  -H "Authorization: Bearer YOUR_LICENSE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-01-31T23:59:59Z"
  }'
```

## API Versioning

The current API version is `v1`. All Enterprise endpoints are prefixed with `/api/v1/enterprise/`.

Future API versions will maintain backward compatibility, and breaking changes will be introduced in new versions (e.g., `/api/v2/enterprise/`).

## Rate Limiting

Enterprise API endpoints include rate limiting with burst support:

- Default: 1000 requests per minute
- Burst: 100 additional requests
- Rate limit headers are included in all responses:
  - `X-RateLimit-Limit`: Maximum requests per window
  - `X-RateLimit-Remaining`: Remaining requests in current window
  - `X-RateLimit-Reset`: Time when the rate limit resets
  - `Retry-After`: Seconds to wait before retrying (when rate limited)

## Error Codes

Common error codes returned by the API:

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request parameters or body |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `LICENSE_REQUIRED` | 402 | Valid enterprise license required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `RESOURCE_NOT_FOUND` | 404 | Requested resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Internal server error |

## OpenAPI Tools

You can use the OpenAPI specification with various tools:

### Code Generation
```bash
# Generate client SDK (e.g., for TypeScript)
openapi-generator-cli generate -i docs/api/enterprise-api-spec.yaml -g typescript-fetch -o sdk/typescript

# Generate Python client
openapi-generator-cli generate -i docs/api/enterprise-api-spec.yaml -g python -o sdk/python
```

### API Testing
```bash
# Use with Postman
# Import docs/api/enterprise-api-spec.yaml into Postman

# Use with Insomnia
# Import docs/api/enterprise-api-spec.yaml into Insomnia
```

### Validation
```bash
# Validate the OpenAPI spec
npx @redocly/cli lint docs/api/enterprise-api-spec.yaml
```

## Further Reading

- [Enterprise Features Documentation](../../05-development/enterprise-features.md)
- [RBAC Configuration Guide](../../03-configuration/rbac.md)
- [Security Scanning Setup](../../03-configuration/security.md)
- [Analytics and Reporting](../../04-api-reference/analytics.md)

## Support

For questions or issues:
- GitHub Issues: https://github.com/ScriptonBasestar/proxynd/issues
- Email: support@proxynd.io
