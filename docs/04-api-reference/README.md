# Enterprise API Reference

ProxyND Enterprise API provides comprehensive REST endpoints for RBAC, audit logging, analytics, security scanning, and license management.

## 📋 Overview

**Total Endpoints**: 47
**API Version**: v1
**Base URL**: `/api/v1/enterprise/`

### Categories

| Category | Endpoints | Description |
|----------|-----------|-------------|
| **RBAC** | 12 | Role-based access control, permissions, user assignments |
| **Audit** | 8 | Event logging, compliance reports, audit trail export |
| **Analytics** | 10 | Usage stats, performance metrics, cost analysis |
| **Security** | 8 | Vulnerability scanning, license compliance, malware detection |
| **Alerts** | 5 | Alert management and notification rules |
| **License** | 4 | License validation and feature management |

## 🚀 Quick Start

### Access Swagger UI

Interactive API documentation:

```bash
# Start the server
make dev-run

# Open Swagger UI
open http://localhost:8080/swagger/index.html

# Or via API endpoint alias
open http://localhost:8080/api/v1/docs
```

### Example Request

```bash
# List all roles (RBAC)
curl http://localhost:8080/api/v1/enterprise/rbac/roles

# Get analytics overview
curl http://localhost:8080/api/v1/enterprise/analytics/overview

# List vulnerabilities
curl http://localhost:8080/api/v1/enterprise/security/vulnerabilities
```

## 📚 Documentation

### Complete References

| Document | Description | Use Case |
|----------|-------------|----------|
| **[OpenAPI Specification](../api/enterprise-api-spec.yaml)** | OpenAPI 3.0 standard spec | API client generation, validation |
| **[Usage Examples](enterprise-api-examples.md)** | 97 code examples (curl + HTTPie) | Quick implementation reference |
| **[Prometheus Metrics](../api/METRICS.md)** | 15+ metrics with PromQL queries | Monitoring and alerting setup |
| **[Load Testing Guide](../../scripts/loadtest/README.md)** | Performance testing tools | Validate API performance |
| **[CI/CD Integration](../deployment/ci-cd-load-testing.md)** | GitHub Actions workflows | Automated testing |
| **[Grafana Dashboard](../../deployments/grafana/README.md)** | 13-panel monitoring dashboard | Real-time metrics visualization |

### Response Format

All endpoints follow a consistent response structure:

```json
{
  "success": true,
  "data": { ... },
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  },
  "metadata": {
    "timestamp": "2025-01-17T10:30:00Z",
    "request_id": "uuid"
  }
}
```

## 🔌 API Categories

### RBAC (Role-Based Access Control)

**12 Endpoints** for managing roles, permissions, and user assignments.

**Key Endpoints**:
- `GET /api/v1/enterprise/rbac/roles` - List all roles
- `POST /api/v1/enterprise/rbac/roles` - Create a new role
- `GET /api/v1/enterprise/rbac/permissions` - List all permissions
- `POST /api/v1/enterprise/rbac/users/{userId}/roles` - Assign role to user

**Use Cases**:
- Define custom roles with specific permissions
- Manage user access across teams
- Implement fine-grained access control
- Audit role assignments

**Example**:
```bash
# Get all roles
curl http://localhost:8080/api/v1/enterprise/rbac/roles

# Create admin role
curl -X POST http://localhost:8080/api/v1/enterprise/rbac/roles \
  -H "Content-Type: application/json" \
  -d '{
    "name": "admin",
    "description": "Full system access",
    "permissions": ["read", "write", "delete"]
  }'
```

### Audit (Event Logging)

**8 Endpoints** for compliance logging and audit trail management.

**Key Endpoints**:
- `GET /api/v1/enterprise/audit/events` - List audit events
- `GET /api/v1/enterprise/audit/events/{id}` - Get specific event
- `POST /api/v1/enterprise/audit/export` - Export audit logs
- `GET /api/v1/enterprise/audit/compliance/report` - Generate compliance report

**Use Cases**:
- Track all user actions
- Generate compliance reports (SOC2, ISO 27001)
- Export audit trails for external analysis
- Monitor security incidents

**Example**:
```bash
# Get recent audit events
curl 'http://localhost:8080/api/v1/enterprise/audit/events?page=1&per_page=20'

# Export last 30 days
curl -X POST http://localhost:8080/api/v1/enterprise/audit/export \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2025-01-01",
    "end_date": "2025-01-31",
    "format": "json"
  }'
```

### Analytics (Usage Statistics)

**10 Endpoints** for usage metrics, performance analysis, and cost tracking.

**Key Endpoints**:
- `GET /api/v1/enterprise/analytics/overview` - Dashboard overview
- `GET /api/v1/enterprise/analytics/usage/proxy-types` - Usage by proxy type
- `GET /api/v1/enterprise/analytics/performance/cache-hit-rate` - Cache performance
- `GET /api/v1/enterprise/analytics/trends/daily` - Daily usage trends

**Use Cases**:
- Monitor proxy usage patterns
- Analyze cache efficiency
- Track bandwidth savings
- Estimate infrastructure costs

**Example**:
```bash
# Get analytics overview
curl http://localhost:8080/api/v1/enterprise/analytics/overview

# Cache hit rate for last 7 days
curl 'http://localhost:8080/api/v1/enterprise/analytics/performance/cache-hit-rate?days=7'
```

### Security (Vulnerability Scanning)

**8 Endpoints** for security scanning, vulnerability detection, and license compliance.

**Key Endpoints**:
- `GET /api/v1/enterprise/security/vulnerabilities` - List vulnerabilities
- `POST /api/v1/enterprise/security/scans` - Start security scan
- `GET /api/v1/enterprise/security/license-compliance` - Check license compliance
- `GET /api/v1/enterprise/security/reports/{id}` - Get security report

**Use Cases**:
- Scan packages for known vulnerabilities
- Detect malware in dependencies
- Ensure license compliance
- Generate security reports

**Example**:
```bash
# List all vulnerabilities
curl http://localhost:8080/api/v1/enterprise/security/vulnerabilities

# Start a security scan
curl -X POST http://localhost:8080/api/v1/enterprise/security/scans \
  -H "Content-Type: application/json" \
  -d '{
    "scan_type": "vulnerability",
    "target": "all"
  }'
```

### Alerts (Notification Management)

**5 Endpoints** for alert configuration and notification rules.

**Key Endpoints**:
- `GET /api/v1/enterprise/alerts` - List all alerts
- `POST /api/v1/enterprise/alerts` - Create alert rule
- `GET /api/v1/enterprise/alerts/{id}` - Get specific alert
- `DELETE /api/v1/enterprise/alerts/{id}` - Delete alert rule

**Use Cases**:
- Set up threshold-based alerts
- Configure notification channels (email, Slack, webhook)
- Monitor critical metrics
- Automate incident response

**Example**:
```bash
# Create high latency alert
curl -X POST http://localhost:8080/api/v1/enterprise/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Latency Alert",
    "type": "threshold",
    "metric": "p95_latency",
    "threshold": 1000,
    "severity": "warning"
  }'
```

### License (License Management)

**4 Endpoints** for license validation and feature management.

**Key Endpoints**:
- `GET /api/v1/enterprise/license/info` - Get license information
- `POST /api/v1/enterprise/license/validate` - Validate license
- `GET /api/v1/enterprise/license/features` - List enabled features
- `GET /api/v1/enterprise/license/usage` - Check license usage

**Use Cases**:
- Verify license status
- Check feature availability
- Monitor license expiration
- Track concurrent users

**Example**:
```bash
# Get license info
curl http://localhost:8080/api/v1/enterprise/license/info

# Check enabled features
curl http://localhost:8080/api/v1/enterprise/license/features
```

## 🧪 Testing

### Load Testing

Run performance tests with the included load testing tool:

```bash
# Quick test (10s, 10 concurrent users)
./scripts/loadtest/load-test-enterprise.sh quick

# Standard test (60s, 100 concurrent users)
./scripts/loadtest/load-test-enterprise.sh standard

# Stress test (60s, 500 concurrent users)
./scripts/loadtest/load-test-enterprise.sh stress

# Ramp-up test (120s, 10→1000 users)
./scripts/loadtest/load-test-enterprise.sh ramp-up

# Sustained load test (300s, 100 concurrent users)
./scripts/loadtest/load-test-enterprise.sh sustained
```

**Performance Targets**:
- **P95 Latency**: < 200ms
- **P99 Latency**: < 500ms
- **Throughput**: > 1000 rps
- **Error Rate**: < 0.1%

### Integration Tests

```bash
# Run Enterprise API integration tests
make test-enterprise-integration

# Run specific category tests
make test-enterprise-rbac
make test-enterprise-audit
make test-enterprise-analytics
```

## 📊 Monitoring

### Prometheus Metrics

**15+ metrics** available at `/metrics` endpoint:

- `proxynd_enterprise_api_requests_total` - Total API requests
- `proxynd_enterprise_api_request_duration_seconds` - Request latency
- `proxynd_enterprise_api_errors_total` - Error counts
- `proxynd_enterprise_api_cache_hits_total` - Cache hits
- `proxynd_enterprise_api_rate_limit_exceeded_total` - Rate limit violations

**See**: [Prometheus Metrics Documentation](../api/METRICS.md)

### Grafana Dashboard

Import the pre-built dashboard for comprehensive monitoring:

**Location**: `deployments/grafana/enterprise-api-dashboard.json`

**Panels** (13 total):
- Request Rate
- Latency Percentiles (P50, P95, P99)
- Error Rate
- Cache Hit Rate
- Active Requests
- Security Vulnerabilities
- RBAC Operations
- Audit Events
- And more...

**See**: [Grafana Dashboard Guide](../../deployments/grafana/README.md)

## 🔒 Authentication

Enterprise API endpoints require valid enterprise license:

- **Development Mode**: License check bypassed (`GO_ENV != production`)
- **Production Mode**: Valid license required in `CONFIG_DIR/license.json`
- **Unauthorized Access**: Returns `HTTP 402 Payment Required`

## 🚦 Rate Limiting

| Tier | Requests/Minute | Burst |
|------|-----------------|-------|
| Free | 60 | 10 |
| Professional | 600 | 100 |
| Enterprise | 6000 | 1000 |

## 🐛 Error Handling

Standard error response format:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input parameter",
    "details": {
      "field": "role_name",
      "issue": "must be alphanumeric"
    }
  },
  "metadata": {
    "timestamp": "2025-01-17T10:30:00Z",
    "request_id": "uuid"
  }
}
```

**Common Error Codes**:
- `VALIDATION_ERROR` (400) - Invalid input
- `UNAUTHORIZED` (401) - Authentication required
- `PAYMENT_REQUIRED` (402) - Invalid license
- `FORBIDDEN` (403) - Insufficient permissions
- `NOT_FOUND` (404) - Resource not found
- `RATE_LIMIT_EXCEEDED` (429) - Too many requests
- `INTERNAL_ERROR` (500) - Server error

## 📖 Additional Resources

### Development

- **[API Examples](enterprise-api-examples.md)** - Complete code examples with both curl and HTTPie
- **[OpenAPI Spec](../api/enterprise-api-spec.yaml)** - Machine-readable API specification

### Operations

- **[Load Testing](../../scripts/loadtest/README.md)** - Performance testing guide
- **[CI/CD Integration](../deployment/ci-cd-load-testing.md)** - Automated testing in GitHub Actions
- **[Metrics](../api/METRICS.md)** - Monitoring and alerting setup

### Visualization

- **[Grafana Dashboard](../../deployments/grafana/README.md)** - Real-time monitoring
- **Swagger UI**: http://localhost:8080/swagger/index.html - Interactive API explorer

## 🤝 Support

- **Documentation**: This directory and subdirectories
- **Issues**: [GitHub Issues](https://github.com/ScriptonBasestar/proxynd/issues)
- **Commercial Support**: contact@proxynd.io

---

**Last Updated**: 2025-01-17
**API Version**: v1.0.0
**Endpoints**: 47
