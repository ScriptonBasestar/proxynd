# Swagger API Documentation

This document describes how to use and generate Swagger API documentation for ProxyND.

## Overview

ProxyND uses Swagger/OpenAPI 2.0 for interactive API documentation. All admin API endpoints are documented with Swagger annotations for automatic documentation generation.

## Accessing Swagger UI

Once the server is running and Swagger documentation is generated, you can access the interactive API documentation at:

- **Primary**: `http://localhost:8080/api/docs/index.html`
- **Alternative**: `http://localhost:8080/swagger/index.html`
- **Alias**: `http://localhost:8080/api/v1/docs` (redirects to Swagger UI)

## Documented API Endpoints

### System Endpoints
- `GET /api/v1/system/info` - Get ProxyND version, edition, and license information
- `GET /api/v1/stats` - Get cache statistics (hit rate, requests, size)

### Package Manager Management
- `GET /api/v1/pm` - List all package managers with status
- `POST /api/v1/pm/{name}/toggle` - Enable/disable a package manager (admin only)

### Configuration Management
- `POST /api/config/reload` - Reload configuration from disk (admin only)

### Cache Management
- `DELETE /api/cache/clear` - Clear all cache (admin only, requires confirmation)
- `DELETE /api/cache/clear/{type}` - Clear cache for specific package manager (admin only)

### Plugin System
- `GET /api/v1/plugins/health` - Get plugin health status with metrics

## Generating Swagger Documentation

### Prerequisites

Install swag CLI tool:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### Generate Documentation

From the project root directory:

```bash
# Generate Swagger docs (parses all annotations)
swag init --parseDependency --parseInternal

# This creates:
# - docs/swagger.json
# - docs/swagger.yaml
# - docs/docs.go
```

### Verify Generation

After running `swag init`, check that the following files were created:

```bash
ls -l docs/swagger.*
ls -l docs/docs.go
```

## Swagger Annotations Structure

All API endpoints include comprehensive Swagger annotations:

```go
// @Summary      Short description
// @Description  Detailed description
// @Tags         category-name
// @Accept       json
// @Produce      json
// @Param        name type datatype required "description"
// @Success      200 {object} ResponseType "Success message"
// @Failure      400 {object} ErrorType "Error message"
// @Router       /api/path [method]
// @Security     BearerAuth
```

## Security

Protected endpoints (requiring admin role) are marked with:
- `@Security BearerAuth` annotation
- JWT token required in Authorization header: `Bearer <token>`

### Testing Protected Endpoints

1. Generate JWT token:
```bash
proxyndctl token generate --username admin --role admin
```

2. In Swagger UI:
   - Click "Authorize" button (top right)
   - Enter: `Bearer <your-token>`
   - Click "Authorize"

3. All protected endpoint requests will now include the token

## Rate Limiting

Admin endpoints have rate limiting configured:
- **Rate**: 10 requests per minute per IP
- **Burst**: 3 additional requests allowed
- **Response**: HTTP 429 if exceeded

## Tags and Categories

API endpoints are organized into these categories:

- **system** - System information and status
- **package-managers** - Package manager operations
- **configuration** - Configuration management
- **cache** - Cache operations and statistics
- **plugins** - Plugin health and monitoring
- **Health** - Health checks (enterprise)
- **RBAC** - Role-based access control (enterprise)
- **Audit** - Audit logging (enterprise)
- **Analytics** - Analytics and metrics (enterprise)
- **Security** - Security scanning (enterprise)
- **Alerts** - Alert management (enterprise)
- **License** - License validation (enterprise)

## Example Usage

### Get Package Managers List

```bash
curl http://localhost:8080/api/v1/pm
```

### Toggle Package Manager (with JWT)

```bash
TOKEN="your-jwt-token"

curl -X POST http://localhost:8080/api/v1/pm/npm/toggle \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'
```

### Clear Cache (with confirmation)

```bash
TOKEN="your-jwt-token"

curl -X DELETE "http://localhost:8080/api/cache/clear?confirm=true" \
  -H "Authorization: Bearer $TOKEN"
```

## Development Mode

When `JWT_SECRET` environment variable is not set, the server runs in development mode:
- Protected endpoints work without authentication
- Useful for local development and testing
- **Never use in production**

Production mode (set `JWT_SECRET`):
```bash
export JWT_SECRET="your-secret-key-change-this-in-production"
```

## Troubleshooting

### Swagger UI shows empty/no endpoints

1. Verify swagger docs were generated:
```bash
ls -l docs/swagger.json docs/docs.go
```

2. Regenerate if missing:
```bash
swag init --parseDependency --parseInternal
```

3. Restart server to load new docs

### "Failed to load API definition" error

1. Check docs/swagger.json is valid JSON:
```bash
cat docs/swagger.json | jq .
```

2. Verify imports in main.go:
```go
import _ "proxynd/docs" // Must be present
```

### Annotations not appearing in Swagger

1. Check annotation format (must be exactly as shown above)
2. Ensure no typos in @Router path
3. Run `swag init` again
4. Restart server

## CI/CD Integration

Add Swagger generation to your build process:

```yaml
# .github/workflows/ci.yml
- name: Generate Swagger docs
  run: |
    go install github.com/swaggo/swag/cmd/swag@latest
    swag init --parseDependency --parseInternal

- name: Verify Swagger docs
  run: |
    test -f docs/swagger.json
    test -f docs/docs.go
```

## Makefile Integration

Add to Makefile for convenience:

```makefile
.PHONY: swagger
swagger: ## Generate Swagger documentation
	@echo "Generating Swagger documentation..."
	swag init --parseDependency --parseInternal
	@echo "Swagger docs generated in docs/"
	@echo "View at: http://localhost:8080/api/docs/index.html"

.PHONY: swagger-validate
swagger-validate: ## Validate Swagger annotations
	@echo "Validating Swagger annotations..."
	swag fmt --dir ./
	@echo "Validation complete"
```

Usage:
```bash
make swagger          # Generate docs
make swagger-validate # Validate annotations
```

## Related Documentation

- [API PM Toggle Authentication](API_PM_TOGGLE_AUTHENTICATION.md)
- [API Config Reload Authentication](API_CONFIG_RELOAD_AUTHENTICATION.md)
- [API Cache Clear Authentication](API_CACHE_CLEAR_AUTHENTICATION.md)
- [Plugin Operations Guide](PLUGIN_OPERATIONS_GUIDE.md)

## References

- [Swaggo Documentation](https://github.com/swaggo/swag)
- [OpenAPI 2.0 Specification](https://swagger.io/specification/v2/)
- [Fiber Swagger Integration](https://github.com/swaggo/fiber-swagger)
