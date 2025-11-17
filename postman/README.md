# ProxyND Postman Collection

Complete Postman collection for testing ProxyND Enterprise API with 47 endpoints.

## 📦 Contents

- **ProxyND-Enterprise-API.postman_collection.json** - Full API collection with all 47 endpoints
- **ProxyND-Dev.postman_environment.json** - Development environment configuration
- **README.md** - This file

## 🚀 Quick Start

### 1. Import Collection

```bash
# Open Postman and import the collection
File → Import → Choose Files → Select ProxyND-Enterprise-API.postman_collection.json

# Import environment
File → Import → Choose Files → Select ProxyND-Dev.postman_environment.json
```

### 2. Select Environment

1. Click the environment dropdown in the top-right corner
2. Select **"ProxyND Development"**
3. Verify `baseUrl` is set to `http://localhost:8080`

### 3. Start ProxyND Server

```bash
cd /path/to/proxynd
make dev-run
```

### 4. Test Endpoints

Click on any request in the collection and click **Send** to test!

## 📂 Collection Structure

The collection is organized into 6 categories:

### 1. RBAC (12 endpoints)
- **Roles**: List, Get, Create, Update, Delete
- **Permissions**: List, Get Role Permissions, Assign, Revoke
- **User Roles**: Get User Roles, Assign to User, Remove from User

### 2. Audit (8 endpoints)
- List Events, Get Event, Search Events
- Get User Events, Get Resource Events
- Export Events, Get Stats, Get Compliance Report

### 3. Analytics (10 endpoints)
- Get Overview, Get Usage Stats, Get Performance Metrics
- Get Cache Efficiency
- List Reports, Create Report, Get Report, Delete Report
- Get Cost Analysis, Get Trends

### 4. Security (8 endpoints)
- List Vulnerabilities, Get Vulnerability
- Trigger Scan, Get Scan Job
- List Licenses, List License Violations
- List Malware Alerts, List Quarantined Packages

### 5. Alerts (5 endpoints)
- List Alerts, Get Alert
- Create Alert Rule, Update Alert Rule, Delete Alert Rule

### 6. License (4 endpoints)
- Get License Info, Validate License
- List Features, Get Usage Metrics

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `baseUrl` | `http://localhost:8080` | ProxyND server base URL |
| `authToken` | _(empty)_ | Authentication token (dev mode doesn't require) |

### Customizing for Production

1. Duplicate the **ProxyND Development** environment
2. Rename to **ProxyND Production**
3. Update `baseUrl` to your production URL
4. Set `authToken` to your license key

## 📝 Example Requests

### List Roles
```
GET {{baseUrl}}/api/v1/enterprise/rbac/roles?page=1&per_page=20
```

### Create Role
```
POST {{baseUrl}}/api/v1/enterprise/rbac/roles
Content-Type: application/json

{
  "name": "Developer",
  "description": "Developer role with cache access",
  "permissions": ["cache:read", "cache:write", "config:read"]
}
```

### Get Analytics Overview
```
GET {{baseUrl}}/api/v1/enterprise/analytics/overview
```

### Trigger Security Scan
```
POST {{baseUrl}}/api/v1/enterprise/security/scan
Content-Type: application/json

{
  "scope": "all",
  "package_manager": "npm"
}
```

## 🧪 Testing Workflow

### 1. Quick Health Check
```bash
# In Postman, create a new request:
GET {{baseUrl}}/health
```

### 2. Test RBAC
1. Navigate to **RBAC** → **Roles** → **List Roles**
2. Click **Send**
3. Verify response shows paginated roles list

### 3. Test Audit
1. Navigate to **Audit** → **List Events**
2. Add query parameters: `page=1&per_page=20`
3. Click **Send**

### 4. Test Analytics
1. Navigate to **Analytics** → **Get Overview**
2. Click **Send**
3. Review dashboard metrics

### 5. Run All Tests
1. Click on the collection name
2. Click **Run** button
3. Select all requests
4. Click **Run ProxyND Enterprise API**

## 📊 Response Format

All endpoints return consistent JSON responses:

### Success Response
```json
{
  "success": true,
  "data": { ... },
  "metadata": {
    "timestamp": "2025-01-17T10:30:00Z",
    "request_id": "uuid"
  }
}
```

### Paginated Response
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  },
  "metadata": { ... }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "ROLE_NOT_FOUND",
    "message": "Role not found"
  },
  "metadata": { ... }
}
```

## 🔐 Authentication

### Development Mode
- No authentication required
- License validation is bypassed
- Perfect for local testing

### Production Mode
- Requires valid enterprise license
- Add Bearer token to requests:
  ```
  Authorization: Bearer YOUR_LICENSE_TOKEN
  ```
- Or set `authToken` in environment and use `{{authToken}}`

## 🎯 Advanced Usage

### Creating Test Suites

1. Click **Collections** → **ProxyND Enterprise API** → **...** → **Add folder**
2. Name it "My Test Suite"
3. Drag requests into the folder
4. Add **Tests** tab scripts for automated validation:

```javascript
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Response has success field", function () {
    pm.expect(pm.response.json()).to.have.property('success', true);
});

pm.test("Response has data", function () {
    pm.expect(pm.response.json()).to.have.property('data');
});
```

### Chaining Requests

Use environment variables to pass data between requests:

```javascript
// In Create Role request Tests tab:
const response = pm.response.json();
pm.environment.set("createdRoleId", response.data.id);

// In Get Role request:
// Use {{createdRoleId}} in the URL path parameter
```

## 📚 Additional Resources

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **WebUI Integration Guide**: `docs/enterprise/WEBUI_INTEGRATION.md`
- **API Documentation**: `tmp/plan/README.md`
- **Contract Tests**: `tests/contract/enterprise_api_test.go`

## 🐛 Troubleshooting

### Connection Refused
- Verify ProxyND server is running: `make dev-run`
- Check baseUrl matches server port
- Ensure no firewall blocking localhost:8080

### 402 Payment Required
- You're in production mode without a license
- Set `GO_ENV != production` for development
- Or provide valid license in `CONFIG_DIR/license.json`

### 404 Not Found
- Verify endpoint URL is correct
- Check server logs for routing issues
- Ensure Enterprise API routes are enabled

### Empty Responses
- Check if fixtures are loaded: `ls tmp/fixtures/`
- Verify server started successfully
- Review server logs for errors

## 📞 Support

For issues or questions:
- GitHub: https://github.com/ScriptonBasestar/proxynd/issues
- Email: support@proxynd.io
- Docs: https://docs.proxynd.io

---

**Version**: 1.0
**Last Updated**: 2025-01-17
**API Version**: v1
**Total Endpoints**: 47
