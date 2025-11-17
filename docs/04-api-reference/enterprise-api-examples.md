# ProxyND Enterprise API Usage Examples

Complete usage examples for all Enterprise API endpoints with curl and HTTPie.

## Table of Contents

- [Authentication](#authentication)
- [RBAC Examples](#rbac-examples)
- [Audit API Examples](#audit-api-examples)
- [Analytics API Examples](#analytics-api-examples)
- [Security API Examples](#security-api-examples)
- [Alerts API Examples](#alerts-api-examples)
- [License API Examples](#license-api-examples)
- [Error Handling](#error-handling)
- [Pagination and Filtering](#pagination-and-filtering)
- [Real-World Scenarios](#real-world-scenarios)

---

## Authentication

All Enterprise API endpoints require a valid enterprise license token in the Authorization header.

### Setting Up Authentication

**Using curl:**
```bash
# Set your token as an environment variable
export ENTERPRISE_TOKEN="your-enterprise-token-here"

# Use it in requests
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/roles
```

**Using HTTPie:**
```bash
# Set your token
export ENTERPRISE_TOKEN="your-enterprise-token-here"

# Use it in requests
http GET localhost:8080/api/v1/enterprise/rbac/roles \
  "Authorization: Bearer $ENTERPRISE_TOKEN"
```

### Verify License

```bash
# curl
curl -X POST http://localhost:8080/api/v1/enterprise/license/validate \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"license_key": "your-license-key"}'

# HTTPie
http POST localhost:8080/api/v1/enterprise/license/validate \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  license_key="your-license-key"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "type": "enterprise",
    "company": "ACME Corp",
    "expires_at": "2026-12-31T23:59:59Z"
  },
  "metadata": {
    "timestamp": "2025-01-17T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

---

## RBAC Examples

### List All Roles

```bash
# curl - with pagination
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/rbac/roles?page=1&per_page=20"

# HTTPie
http GET localhost:8080/api/v1/enterprise/rbac/roles \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  page==1 per_page==20
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "role_admin",
      "name": "Admin",
      "description": "Full system access",
      "permissions": ["*"],
      "created_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "role_developer",
      "name": "Developer",
      "description": "Developer access",
      "permissions": ["read:packages", "write:packages"],
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 5,
    "total_pages": 1
  },
  "metadata": {
    "timestamp": "2025-01-17T10:00:00Z",
    "request_id": "req_abc124"
  }
}
```

### Get Specific Role

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/roles/role_developer

# HTTPie
http GET localhost:8080/api/v1/enterprise/rbac/roles/role_developer \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Create a New Role

```bash
# curl
curl -X POST http://localhost:8080/api/v1/enterprise/rbac/roles \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "QA Engineer",
    "description": "Quality assurance role",
    "permissions": ["read:packages", "read:audit"]
  }'

# HTTPie
http POST localhost:8080/api/v1/enterprise/rbac/roles \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  name="QA Engineer" \
  description="Quality assurance role" \
  permissions:='["read:packages", "read:audit"]'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "role_qa",
    "name": "QA Engineer",
    "description": "Quality assurance role",
    "permissions": ["read:packages", "read:audit"],
    "created_at": "2025-01-17T10:00:00Z",
    "updated_at": "2025-01-17T10:00:00Z"
  },
  "metadata": {
    "timestamp": "2025-01-17T10:00:00Z",
    "request_id": "req_abc125"
  }
}
```

### Update a Role

```bash
# curl
curl -X PUT http://localhost:8080/api/v1/enterprise/rbac/roles/role_qa \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "QA role with extended permissions",
    "permissions": ["read:packages", "read:audit", "read:analytics"]
  }'

# HTTPie
http PUT localhost:8080/api/v1/enterprise/rbac/roles/role_qa \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  description="QA role with extended permissions" \
  permissions:='["read:packages", "read:audit", "read:analytics"]'
```

### Delete a Role

```bash
# curl
curl -X DELETE http://localhost:8080/api/v1/enterprise/rbac/roles/role_qa \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN"

# HTTPie
http DELETE localhost:8080/api/v1/enterprise/rbac/roles/role_qa \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "metadata": {
    "timestamp": "2025-01-17T10:00:00Z",
    "request_id": "req_abc126"
  }
}
```

### List All Permissions

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/permissions

# HTTPie
http GET localhost:8080/api/v1/enterprise/rbac/permissions \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Assign Permissions to Role

```bash
# curl
curl -X POST http://localhost:8080/api/v1/enterprise/rbac/roles/role_developer/permissions \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "permissions": ["delete:packages"]
  }'

# HTTPie
http POST localhost:8080/api/v1/enterprise/rbac/roles/role_developer/permissions \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  permissions:='["delete:packages"]'
```

### Revoke Permission from Role

```bash
# curl
curl -X DELETE \
  http://localhost:8080/api/v1/enterprise/rbac/roles/role_developer/permissions/delete:packages \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN"

# HTTPie
http DELETE localhost:8080/api/v1/enterprise/rbac/roles/role_developer/permissions/delete:packages \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Assign Role to User

```bash
# curl
curl -X POST http://localhost:8080/api/v1/enterprise/rbac/users/user_123/roles \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": "role_developer"
  }'

# HTTPie
http POST localhost:8080/api/v1/enterprise/rbac/users/user_123/roles \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  role_id="role_developer"
```

### Get User's Roles

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/users/user_123/roles

# HTTPie
http GET localhost:8080/api/v1/enterprise/rbac/users/user_123/roles \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Remove Role from User

```bash
# curl
curl -X DELETE \
  http://localhost:8080/api/v1/enterprise/rbac/users/user_123/roles/role_developer \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN"

# HTTPie
http DELETE localhost:8080/api/v1/enterprise/rbac/users/user_123/roles/role_developer \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

---

## Audit API Examples

### List Audit Events

```bash
# curl - basic listing
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/audit/events?page=1&per_page=50"

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/events \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  page==1 per_page==50
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "evt_abc123",
      "timestamp": "2025-01-17T09:55:00Z",
      "user_id": "user_123",
      "user_email": "john@example.com",
      "action": "role.create",
      "resource_type": "role",
      "resource_id": "role_qa",
      "ip_address": "192.168.1.100",
      "user_agent": "curl/7.68.0",
      "status": "success",
      "details": {
        "role_name": "QA Engineer",
        "permissions": ["read:packages", "read:audit"]
      }
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total": 150,
    "total_pages": 3
  }
}
```

### Advanced Search

```bash
# curl - search by user and action
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/audit/events/search?query=login&user_id=user_123&action=user.login"

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/events/search \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  query=="login" user_id=="user_123" action=="user.login"
```

### Filter by Date Range

```bash
# curl - last 7 days
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/audit/events?start_date=2025-01-10T00:00:00Z&end_date=2025-01-17T23:59:59Z"

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/events \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  start_date=="2025-01-10T00:00:00Z" \
  end_date=="2025-01-17T23:59:59Z"
```

### Get Specific Event

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/audit/events/evt_abc123

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/events/evt_abc123 \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Get User's Audit Trail

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/audit/users/user_123/events

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/users/user_123/events \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Get Resource Audit History

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/audit/resources/role/role_developer

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/resources/role/role_developer \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Export Audit Logs

```bash
# curl - export as JSON
curl -X POST http://localhost:8080/api/v1/enterprise/audit/export \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-01-31T23:59:59Z"
  }' \
  -o audit_export.json

# HTTPie - export as CSV
http POST localhost:8080/api/v1/enterprise/audit/export \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  format="csv" \
  start_date="2025-01-01T00:00:00Z" \
  end_date="2025-01-31T23:59:59Z" \
  > audit_export.csv
```

### Get Audit Statistics

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/audit/stats

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/stats \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_events": 15420,
    "events_today": 342,
    "events_this_week": 2103,
    "events_this_month": 8934,
    "by_action": {
      "user.login": 1234,
      "role.create": 45,
      "role.update": 78,
      "package.download": 12543
    },
    "by_user": {
      "user_123": 543,
      "user_456": 321
    }
  }
}
```

### Get Compliance Report

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/audit/compliance/report

# HTTPie
http GET localhost:8080/api/v1/enterprise/audit/compliance/report \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

---

## Analytics API Examples

### Get Dashboard Overview

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/overview

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/overview \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_requests": 125340,
    "cache_hit_rate": 87.5,
    "avg_response_time": 145.2,
    "bandwidth_used": 5368709120,
    "active_users": 47,
    "top_packages": [
      {
        "package_manager": "npm",
        "package_name": "react",
        "request_count": 5420,
        "bytes_served": 104857600
      }
    ],
    "requests_by_pm": {
      "npm": 54200,
      "maven": 38100,
      "pypi": 21540
    },
    "error_rate": 0.45,
    "timestamp": "2025-01-17T10:00:00Z"
  }
}
```

### Get Usage Statistics

```bash
# curl - last 30 days
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/analytics/usage?period=30d"

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/usage \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  period=="30d"
```

**Available periods:** `7d`, `30d`, `90d`

### Get Performance Metrics

```bash
# curl - last 24 hours
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/analytics/performance?period=24h"

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/performance \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  period=="24h"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "time_range": "last_24_hours",
    "avg_latency": 145.2,
    "p50_latency": 98.5,
    "p95_latency": 320.8,
    "p99_latency": 567.2,
    "throughput": 45.6,
    "error_rate": 0.45,
    "by_endpoint": {
      "/npm/*": {
        "endpoint": "/npm/*",
        "requests": 54200,
        "avg_latency": 120.5,
        "p95_latency": 280.3,
        "error_count": 120,
        "error_rate": 0.22
      }
    }
  }
}
```

### Get Cache Efficiency

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/cache-efficiency

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/cache-efficiency \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### List Reports

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/reports

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/reports \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Create Custom Report

```bash
# curl
curl -X POST http://localhost:8080/api/v1/enterprise/analytics/reports \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Weekly Package Usage",
    "type": "usage",
    "schedule": "0 0 * * 1",
    "filters": {
      "package_managers": ["npm", "maven"],
      "period": "7d"
    }
  }'

# HTTPie
http POST localhost:8080/api/v1/enterprise/analytics/reports \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  name="Weekly Package Usage" \
  type="usage" \
  schedule="0 0 * * 1" \
  filters:='{"package_managers":["npm","maven"],"period":"7d"}'
```

### Get Report Data

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/reports/report_123

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/reports/report_123 \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Delete Report

```bash
# curl
curl -X DELETE \
  http://localhost:8080/api/v1/enterprise/analytics/reports/report_123 \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN"

# HTTPie
http DELETE localhost:8080/api/v1/enterprise/analytics/reports/report_123 \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Get Cost Analysis

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/costs

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/costs \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Get Trends

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/trends

# HTTPie
http GET localhost:8080/api/v1/enterprise/analytics/trends \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

---

## Security API Examples

### List Vulnerabilities

```bash
# curl - all vulnerabilities
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/vulnerabilities

# HTTPie - filter by severity
http GET localhost:8080/api/v1/enterprise/security/vulnerabilities \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  severity=="critical"
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "vuln_123",
      "cve_id": "CVE-2024-1234",
      "severity": "critical",
      "package_manager": "npm",
      "package_name": "example-package",
      "affected_version": "1.2.3",
      "fixed_version": "1.2.4",
      "description": "Remote code execution vulnerability",
      "published_date": "2024-01-15T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 5,
    "total_pages": 1
  }
}
```

### Get Vulnerability Details

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/vulnerabilities/vuln_123

# HTTPie
http GET localhost:8080/api/v1/enterprise/security/vulnerabilities/vuln_123 \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Trigger Security Scan

```bash
# curl - scan all package managers
curl -X POST http://localhost:8080/api/v1/enterprise/security/scan \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "all",
    "priority": "high"
  }'

# HTTPie - scan specific package manager
http POST localhost:8080/api/v1/enterprise/security/scan \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  scope="npm" \
  priority="high"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "scan_job_456",
    "status": "pending",
    "progress": 0,
    "started_at": "2025-01-17T10:00:00Z"
  }
}
```

### Get Scan Job Status

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/scan/scan_job_456

# HTTPie
http GET localhost:8080/api/v1/enterprise/security/scan/scan_job_456 \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "scan_job_456",
    "status": "completed",
    "progress": 100,
    "started_at": "2025-01-17T10:00:00Z",
    "completed_at": "2025-01-17T10:05:00Z",
    "results": {
      "vulnerabilities_found": 3,
      "critical": 1,
      "high": 2,
      "medium": 0,
      "low": 0
    }
  }
}
```

### List Detected Licenses

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/licenses

# HTTPie
http GET localhost:8080/api/v1/enterprise/security/licenses \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### List License Violations

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/licenses/violations

# HTTPie
http GET localhost:8080/api/v1/enterprise/security/licenses/violations \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### List Malware Alerts

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/malware/alerts

# HTTPie
http GET localhost:8080/api/v1/enterprise/security/malware/alerts \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### List Quarantined Packages

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/malware/quarantine

# HTTPie
http GET localhost:8080/api/v1/enterprise/security/malware/quarantine \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

---

## Alerts API Examples

### List Active Alerts

```bash
# curl - all alerts
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/alerts

# HTTPie - filter by status and severity
http GET localhost:8080/api/v1/enterprise/alerts \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  status=="active" severity=="critical"
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "alert_789",
      "name": "High Error Rate",
      "severity": "critical",
      "status": "active",
      "message": "Error rate exceeded 5% threshold",
      "created_at": "2025-01-17T09:00:00Z",
      "triggered_at": "2025-01-17T09:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 3,
    "total_pages": 1
  }
}
```

### Get Alert Details

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/alerts/alert_789

# HTTPie
http GET localhost:8080/api/v1/enterprise/alerts/alert_789 \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Create Alert Rule

```bash
# curl
curl -X POST http://localhost:8080/api/v1/enterprise/alerts/rules \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Critical Vulnerability Alert",
    "type": "threshold",
    "severity": "critical",
    "conditions": {
      "metric": "vulnerabilities.critical",
      "operator": ">",
      "threshold": 0
    },
    "actions": [
      {
        "type": "email",
        "config": {
          "recipients": ["security@example.com"]
        }
      },
      {
        "type": "slack",
        "config": {
          "webhook_url": "https://hooks.slack.com/services/XXX",
          "channel": "#security-alerts"
        }
      }
    ]
  }'

# HTTPie
http POST localhost:8080/api/v1/enterprise/alerts/rules \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  name="Critical Vulnerability Alert" \
  type="threshold" \
  severity="critical" \
  conditions:='{"metric":"vulnerabilities.critical","operator":">","threshold":0}' \
  actions:='[{"type":"email","config":{"recipients":["security@example.com"]}}]'
```

### Update Alert Rule

```bash
# curl
curl -X PUT http://localhost:8080/api/v1/enterprise/alerts/rules/rule_123 \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Alert Rule",
    "severity": "warning",
    "conditions": {
      "metric": "error_rate",
      "operator": ">",
      "threshold": 5
    }
  }'

# HTTPie
http PUT localhost:8080/api/v1/enterprise/alerts/rules/rule_123 \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  name="Updated Alert Rule" \
  severity="warning"
```

### Delete Alert Rule

```bash
# curl
curl -X DELETE \
  http://localhost:8080/api/v1/enterprise/alerts/rules/rule_123 \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN"

# HTTPie
http DELETE localhost:8080/api/v1/enterprise/alerts/rules/rule_123 \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

---

## License API Examples

### Get License Information

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/license/info

# HTTPie
http GET localhost:8080/api/v1/enterprise/license/info \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "type": "enterprise",
    "company": "ACME Corp",
    "expires_at": "2026-12-31T23:59:59Z",
    "features": [
      "rbac",
      "audit_logging",
      "analytics",
      "security_scanning",
      "alerting",
      "multi_region",
      "sso"
    ]
  }
}
```

### List Available Features

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/license/features

# HTTPie
http GET localhost:8080/api/v1/enterprise/license/features \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### Get License Usage Metrics

```bash
# curl
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/license/usage

# HTTPie
http GET localhost:8080/api/v1/enterprise/license/usage \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "users": {
      "limit": 100,
      "current": 47,
      "percentage": 47
    },
    "storage": {
      "limit_gb": 1000,
      "current_gb": 342.5,
      "percentage": 34.25
    },
    "api_calls": {
      "limit_per_month": 1000000,
      "current_month": 125340,
      "percentage": 12.53
    }
  }
}
```

---

## Error Handling

### Common Error Responses

#### 400 Bad Request

```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request parameters",
    "details": {
      "field": "permissions",
      "error": "must be an array of strings"
    }
  },
  "metadata": {
    "timestamp": "2025-01-17T10:00:00Z",
    "request_id": "req_abc127"
  }
}
```

#### 401 Unauthorized

```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Missing or invalid authentication token"
  }
}
```

#### 402 Payment Required (License Issue)

```json
{
  "success": false,
  "error": {
    "code": "LICENSE_REQUIRED",
    "message": "This feature requires an enterprise license",
    "upgrade_url": "https://proxynd.io/pricing"
  }
}
```

#### 403 Forbidden

```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "Insufficient permissions to access this resource"
  }
}
```

#### 404 Not Found

```json
{
  "success": false,
  "error": {
    "code": "ROLE_NOT_FOUND",
    "message": "Role not found"
  }
}
```

#### 409 Conflict

```json
{
  "success": false,
  "error": {
    "code": "CONFLICT",
    "message": "Role with this name already exists"
  }
}
```

#### 422 Validation Error

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      {
        "field": "name",
        "error": "must be between 3 and 50 characters"
      },
      {
        "field": "permissions",
        "error": "must contain at least one permission"
      }
    ]
  }
}
```

#### 429 Rate Limit Exceeded

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests"
  },
  "metadata": {
    "retry_after": 60
  }
}
```

**Response Headers:**
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705488060
Retry-After: 60
```

#### 500 Internal Server Error

```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "An internal server error occurred"
  }
}
```

### Error Handling Example

```bash
# curl with error handling
response=$(curl -s -w "\n%{http_code}" \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/roles/invalid_id)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
  echo "Success: $body"
elif [ "$http_code" -eq 404 ]; then
  echo "Error: Role not found"
  echo "$body" | jq '.error'
else
  echo "Error: HTTP $http_code"
  echo "$body" | jq '.error'
fi
```

---

## Pagination and Filtering

### Pagination Parameters

All list endpoints support pagination:

- `page`: Page number (default: 1, minimum: 1)
- `per_page`: Items per page (default: 20, minimum: 1, maximum: 100)

```bash
# curl - page 2 with 50 items
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/rbac/roles?page=2&per_page=50"

# HTTPie
http GET localhost:8080/api/v1/enterprise/rbac/roles \
  Authorization:"Bearer $ENTERPRISE_TOKEN" \
  page==2 per_page==50
```

### Pagination Response

```json
{
  "pagination": {
    "page": 2,
    "per_page": 50,
    "total": 127,
    "total_pages": 3
  }
}
```

### Filtering Examples

#### Filter Audit Events by Date

```bash
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/audit/events?start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z"
```

#### Filter by Status

```bash
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/alerts?status=active"
```

#### Filter by Severity

```bash
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/security/vulnerabilities?severity=critical"
```

#### Multiple Filters

```bash
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/alerts?status=active&severity=critical&page=1&per_page=20"
```

---

## Real-World Scenarios

### Scenario 1: Onboard a New Developer

```bash
# 1. Create a developer role (if not exists)
curl -X POST http://localhost:8080/api/v1/enterprise/rbac/roles \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Developer",
    "permissions": ["read:packages", "write:packages"]
  }'

# 2. Assign role to new user
curl -X POST http://localhost:8080/api/v1/enterprise/rbac/users/new_dev_123/roles \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role_id": "role_developer"}'

# 3. Verify role assignment
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/users/new_dev_123/roles

# 4. Check audit trail
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/audit/events/search?user_id=new_dev_123"
```

### Scenario 2: Security Incident Response

```bash
# 1. Check for critical vulnerabilities
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/security/vulnerabilities?severity=critical"

# 2. Trigger immediate security scan
curl -X POST http://localhost:8080/api/v1/enterprise/security/scan \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "all", "priority": "high"}'

# 3. Check scan status
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/security/scan/scan_job_789

# 4. Export audit logs for investigation
curl -X POST http://localhost:8080/api/v1/enterprise/audit/export \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "start_date": "2025-01-17T00:00:00Z",
    "end_date": "2025-01-17T23:59:59Z"
  }' -o incident_audit.json

# 5. Create critical alert
curl -X POST http://localhost:8080/api/v1/enterprise/alerts/rules \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Critical Vulnerability Detected",
    "type": "threshold",
    "severity": "critical",
    "actions": [{"type": "email", "config": {"recipients": ["security@example.com"]}}]
  }'
```

### Scenario 3: Monthly Reporting

```bash
# 1. Get analytics overview
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/overview

# 2. Get usage statistics for the month
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/analytics/usage?period=30d"

# 3. Get performance metrics
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/analytics/performance?period=30d"

# 4. Export audit logs
curl -X POST http://localhost:8080/api/v1/enterprise/audit/export \
  -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "csv",
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-01-31T23:59:59Z"
  }' -o monthly_audit.csv

# 5. Get compliance report
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/audit/compliance/report
```

### Scenario 4: Monitor System Health

```bash
# 1. Check active alerts
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/alerts?status=active"

# 2. Get performance metrics
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/analytics/performance?period=1h"

# 3. Check cache efficiency
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/analytics/cache-efficiency

# 4. Check license usage
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/license/usage

# 5. Review recent errors in audit log
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  "http://localhost:8080/api/v1/enterprise/audit/events/search?query=error&page=1&per_page=10"
```

### Scenario 5: Automated Testing with jq

```bash
#!/bin/bash
# Script to test Enterprise API endpoints

TOKEN="your-enterprise-token"
BASE_URL="http://localhost:8080/api/v1/enterprise"

# Test RBAC - List roles
echo "Testing RBAC - List Roles..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/rbac/roles")
role_count=$(echo "$response" | jq '.data | length')
echo "✓ Found $role_count roles"

# Test Analytics - Get overview
echo "Testing Analytics - Overview..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/analytics/overview")
total_requests=$(echo "$response" | jq '.data.total_requests')
cache_hit_rate=$(echo "$response" | jq '.data.cache_hit_rate')
echo "✓ Total requests: $total_requests, Cache hit rate: $cache_hit_rate%"

# Test Security - List vulnerabilities
echo "Testing Security - Vulnerabilities..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/security/vulnerabilities")
vuln_count=$(echo "$response" | jq '.pagination.total')
echo "✓ Found $vuln_count vulnerabilities"

# Test Audit - Recent events
echo "Testing Audit - Recent Events..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/audit/events?per_page=5")
event_count=$(echo "$response" | jq '.data | length')
echo "✓ Retrieved $event_count recent events"

echo "All tests completed!"
```

---

## Tips and Best Practices

### 1. Use Environment Variables for Tokens

```bash
# Add to ~/.bashrc or ~/.zshrc
export ENTERPRISE_TOKEN="your-token-here"
export PROXYND_URL="http://localhost:8080"
```

### 2. Save Commonly Used Requests

```bash
# Create aliases
alias rbac-list='curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" $PROXYND_URL/api/v1/enterprise/rbac/roles'
alias audit-today='curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" "$PROXYND_URL/api/v1/enterprise/audit/events?start_date=$(date +%Y-%m-%d)T00:00:00Z"'
```

### 3. Pretty Print JSON with jq

```bash
# Install jq: https://stedolan.github.io/jq/
curl -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/roles | jq '.'
```

### 4. Use HTTPie for Interactive Testing

```bash
# Install HTTPie: https://httpie.io/
# Simpler syntax and colored output
http GET localhost:8080/api/v1/enterprise/rbac/roles \
  Authorization:"Bearer $ENTERPRISE_TOKEN"
```

### 5. Rate Limiting Headers

Always check rate limit headers:

```bash
curl -i -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/roles | grep "X-RateLimit"
```

### 6. Request IDs for Debugging

Include request IDs when reporting issues:

```bash
response=$(curl -s -H "Authorization: Bearer $ENTERPRISE_TOKEN" \
  http://localhost:8080/api/v1/enterprise/rbac/roles)

request_id=$(echo "$response" | jq -r '.metadata.request_id')
echo "Request ID: $request_id"
```

---

## Further Reading

- [OpenAPI Specification](enterprise-api-spec.yaml)
- [API Documentation](README.md)
- [Metrics Guide](METRICS.md)
- [Load Testing Guide](../../scripts/loadtest/README.md)
- [Interactive Swagger UI](http://localhost:8080/api/v1/docs)

## Support

For questions or issues:
- GitHub Issues: https://github.com/ScriptonBasestar/proxynd/issues
- Email: support@proxynd.io
- Interactive Docs: http://localhost:8080/api/v1/docs
