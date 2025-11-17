# ProxyND Enterprise API - WebUI Integration Guide

> Comprehensive guide for integrating the ProxyND Enterprise API with WebUI applications

## Table of Contents

- [Quick Start](#quick-start)
- [Authentication](#authentication)
- [API Structure](#api-structure)
- [Response Handling](#response-handling)
- [Pagination](#pagination)
- [Error Handling](#error-handling)
- [Code Examples](#code-examples)
- [Performance Best Practices](#performance-best-practices)
- [TypeScript Definitions](#typescript-definitions)

---

## Quick Start

### 1. Start Development Server

```bash
# Clone and setup
git clone https://github.com/ScriptonBasestar/proxynd.git
cd proxynd

# Start server with fixtures
make dev-run

# Server will be available at http://localhost:8080
```

### 2. Verify API Availability

```bash
# Check health
curl http://localhost:8080/health

# Test enterprise endpoint
curl http://localhost:8080/api/v1/enterprise/rbac/roles
```

### 3. Run Integration Tests

```bash
# Test all endpoints
make test-enterprise-integration

# Test specific category
make test-enterprise-rbac
```

---

## Authentication

### Development Mode

In development mode (`GO_ENV != production`), license validation is bypassed for easier testing.

```javascript
// No authentication required in dev mode
fetch('http://localhost:8080/api/v1/enterprise/rbac/roles')
  .then(res => res.json())
  .then(data => console.log(data));
```

### Production Mode

Production requires a valid enterprise license and JWT token.

```javascript
// With JWT authentication
const token = 'your-jwt-token';

fetch('http://localhost:8080/api/v1/enterprise/rbac/roles', {
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  }
})
  .then(res => res.json())
  .then(data => console.log(data));
```

### License Validation

If license is invalid or missing, you'll receive:

```json
{
  "success": false,
  "error": {
    "code": "LICENSE_REQUIRED",
    "message": "This endpoint requires an enterprise license",
    "upgrade_url": "https://proxynd.io/pricing"
  }
}
```

**HTTP Status**: `402 Payment Required`

---

## API Structure

All enterprise endpoints follow the pattern: `/api/v1/enterprise/<category>/<resource>`

### Categories

| Category | Base Path | Description |
|----------|-----------|-------------|
| RBAC | `/api/v1/enterprise/rbac` | Roles, permissions, user assignments |
| Audit | `/api/v1/enterprise/audit` | Event logs, compliance |
| Analytics | `/api/v1/enterprise/analytics` | Metrics, reports |
| Security | `/api/v1/enterprise/security` | Vulnerability scans |
| Alerts | `/api/v1/enterprise/alerts` | Alert management |
| License | `/api/v1/enterprise/license` | License info |

### HTTP Methods

- `GET` - Retrieve resources
- `POST` - Create new resources
- `PUT` - Update existing resources
- `DELETE` - Delete resources

---

## Response Handling

### Success Response

All successful responses follow this structure:

```typescript
interface SuccessResponse<T> {
  success: true;
  data: T;
  metadata: {
    timestamp: string;  // ISO 8601 format
    request_id: string; // UUID for tracking
  };
}
```

**Example:**

```json
{
  "success": true,
  "data": {
    "id": "role_admin",
    "name": "Administrator",
    "permissions": ["admin:*"]
  },
  "metadata": {
    "timestamp": "2025-01-17T10:30:00Z",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### Paginated Response

List endpoints include pagination metadata:

```typescript
interface PaginatedResponse<T> {
  success: true;
  data: T[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
    has_next: boolean;
    has_prev: boolean;
  };
  metadata: {
    timestamp: string;
    request_id: string;
  };
}
```

**Example:**

```json
{
  "success": true,
  "data": [
    { "id": "role_admin", "name": "Administrator" },
    { "id": "role_developer", "name": "Developer" }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3,
    "has_next": true,
    "has_prev": false
  },
  "metadata": {
    "timestamp": "2025-01-17T10:30:00Z",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### Error Response

```typescript
interface ErrorResponse {
  success: false;
  error: {
    code: string;
    message: string;
    details?: Record<string, any>;
  };
  metadata: {
    timestamp: string;
    request_id: string;
  };
}
```

**Example:**

```json
{
  "success": false,
  "error": {
    "code": "ROLE_NOT_FOUND",
    "message": "Role not found",
    "details": {
      "role_id": "role_invalid"
    }
  },
  "metadata": {
    "timestamp": "2025-01-17T10:30:00Z",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

## Pagination

### Query Parameters

All list endpoints support pagination:

```
GET /api/v1/enterprise/rbac/roles?page=1&per_page=20
```

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `page` | number | 1 | - | Page number (1-indexed) |
| `per_page` | number | 20 | 100 | Items per page |

### Navigation

Use pagination metadata for navigation:

```javascript
function fetchPage(page, perPage = 20) {
  return fetch(
    `http://localhost:8080/api/v1/enterprise/rbac/roles?page=${page}&per_page=${perPage}`
  ).then(res => res.json());
}

// Fetch first page
const response = await fetchPage(1);

// Check if more pages exist
if (response.pagination.has_next) {
  // Fetch next page
  const nextPage = await fetchPage(response.pagination.page + 1);
}
```

### React Hook Example

```typescript
import { useState, useEffect } from 'react';

interface PaginationState {
  page: number;
  perPage: number;
}

function useEnterpriseAPI<T>(endpoint: string) {
  const [data, setData] = useState<T[]>([]);
  const [pagination, setPagination] = useState<PaginationState>({ page: 1, perPage: 20 });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch(`http://localhost:8080/api/v1/enterprise${endpoint}?page=${pagination.page}&per_page=${pagination.perPage}`)
      .then(res => res.json())
      .then(response => {
        if (response.success) {
          setData(response.data);
          setPagination(response.pagination);
        } else {
          throw new Error(response.error.message);
        }
      })
      .catch(setError)
      .finally(() => setLoading(false));
  }, [endpoint, pagination.page, pagination.perPage]);

  return { data, pagination, loading, error };
}

// Usage
function RolesList() {
  const { data: roles, pagination, loading } = useEnterpriseAPI('/rbac/roles');

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      {roles.map(role => <div key={role.id}>{role.name}</div>)}
      <button disabled={!pagination.has_prev} onClick={() => /* go to prev page */}>Previous</button>
      <button disabled={!pagination.has_next} onClick={() => /* go to next page */}>Next</button>
    </div>
  );
}
```

---

## Error Handling

### Common Error Codes

| Code | HTTP Status | Description | Action |
|------|-------------|-------------|--------|
| `LICENSE_REQUIRED` | 402 | No enterprise license | Contact sales |
| `FEATURE_NOT_LICENSED` | 402 | Feature not in license | Upgrade license |
| `ROLE_NOT_FOUND` | 404 | Role doesn't exist | Check role ID |
| `INVALID_REQUEST` | 400 | Bad request body | Validate input |
| `INSUFFICIENT_PERMISSIONS` | 403 | No permission | Request access |

### Error Handling Pattern

```typescript
async function fetchWithErrorHandling<T>(url: string, options?: RequestInit): Promise<T> {
  try {
    const response = await fetch(url, options);
    const data = await response.json();

    if (!data.success) {
      // Handle API errors
      throw new APIError(data.error.code, data.error.message, data.error.details);
    }

    return data.data;
  } catch (error) {
    if (error instanceof APIError) {
      // Handle specific API errors
      switch (error.code) {
        case 'LICENSE_REQUIRED':
          // Redirect to upgrade page
          window.location.href = '/upgrade';
          break;
        case 'ROLE_NOT_FOUND':
          // Show user-friendly message
          alert('Role not found');
          break;
        default:
          console.error('API Error:', error);
      }
    }
    throw error;
  }
}

class APIError extends Error {
  constructor(
    public code: string,
    message: string,
    public details?: Record<string, any>
  ) {
    super(message);
    this.name = 'APIError';
  }
}
```

---

## Code Examples

### RBAC Examples

**List Roles**

```javascript
// Fetch all roles
const response = await fetch('http://localhost:8080/api/v1/enterprise/rbac/roles');
const { data: roles } = await response.json();

// roles: Array<{ id, name, description, permissions, created_at, updated_at }>
```

**Get Role Details**

```javascript
const roleId = 'role_admin';
const response = await fetch(`http://localhost:8080/api/v1/enterprise/rbac/roles/${roleId}`);
const { data: role } = await response.json();
```

**Create Role**

```javascript
const newRole = {
  name: 'Tester',
  description: 'QA team access',
  permissions: ['cache:read', 'analytics:read']
};

const response = await fetch('http://localhost:8080/api/v1/enterprise/rbac/roles', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(newRole)
});

const { data: createdRole } = await response.json();
```

**Update Role**

```javascript
const updates = {
  name: 'Senior Tester',
  description: 'Senior QA team access',
  permissions: ['cache:read', 'cache:write', 'analytics:read']
};

const response = await fetch(`http://localhost:8080/api/v1/enterprise/rbac/roles/${roleId}`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(updates)
});
```

**Delete Role**

```javascript
const response = await fetch(`http://localhost:8080/api/v1/enterprise/rbac/roles/${roleId}`, {
  method: 'DELETE'
});

const { data } = await response.json();
// data: { id: 'role_tester', deleted: true }
```

### Audit Examples

**List Audit Events**

```javascript
// With pagination and filtering
const params = new URLSearchParams({
  page: '1',
  per_page: '50',
  user_id: 'user_001',  // Optional filter
  action: 'cache.clear' // Optional filter
});

const response = await fetch(`http://localhost:8080/api/v1/enterprise/audit/events?${params}`);
const { data: events, pagination } = await response.json();
```

**Search Events**

```javascript
const searchQuery = 'cache';
const response = await fetch(
  `http://localhost:8080/api/v1/enterprise/audit/events/search?q=${encodeURIComponent(searchQuery)}`
);
const { data: results } = await response.json();
```

**Export Audit Logs**

```javascript
const exportRequest = {
  format: 'csv',
  start_date: '2025-01-01T00:00:00Z',
  end_date: '2025-01-31T23:59:59Z'
};

const response = await fetch('http://localhost:8080/api/v1/enterprise/audit/export', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(exportRequest)
});

const { data: exportJob } = await response.json();
// exportJob: { export_id, status: 'processing', format: 'csv' }
```

### Analytics Examples

**Dashboard Overview**

```javascript
const response = await fetch('http://localhost:8080/api/v1/enterprise/analytics/overview');
const { data: overview } = await response.json();

/*
overview: {
  total_requests: number,
  cache_hit_rate: number,
  avg_response_time: number,
  bandwidth_used: number,
  active_users: number,
  top_packages: Array<PackageUsage>,
  requests_by_pm: Record<string, number>,
  error_rate: number
}
*/
```

**Performance Metrics**

```javascript
const response = await fetch('http://localhost:8080/api/v1/enterprise/analytics/performance');
const { data: metrics } = await response.json();

/*
metrics: {
  avg_latency: number,
  p50_latency: number,
  p95_latency: number,
  p99_latency: number,
  throughput: number,
  error_rate: number,
  by_endpoint: Record<string, EndpointMetrics>
}
*/
```

### Security Examples

**List Vulnerabilities**

```javascript
const params = new URLSearchParams({
  page: '1',
  per_page: '20',
  severity: 'critical' // Optional filter
});

const response = await fetch(`http://localhost:8080/api/v1/enterprise/security/vulnerabilities?${params}`);
const { data: vulnerabilities } = await response.json();
```

**Trigger Security Scan**

```javascript
const scanRequest = {
  type: 'vulnerability',
  package_manager: 'npm',
  full_scan: true
};

const response = await fetch('http://localhost:8080/api/v1/enterprise/security/scan', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(scanRequest)
});

const { data: scanJob } = await response.json();
// scanJob: { id, type, status: 'pending', started_at }

// Poll for scan completion
const checkStatus = async (jobId) => {
  const res = await fetch(`http://localhost:8080/api/v1/enterprise/security/scan/${jobId}`);
  const { data: job } = await res.json();
  return job;
};
```

---

## Performance Best Practices

### 1. Parallel Requests

Load multiple independent resources in parallel:

```javascript
// ❌ Sequential (slow)
const roles = await fetch('/api/v1/enterprise/rbac/roles').then(r => r.json());
const events = await fetch('/api/v1/enterprise/audit/events').then(r => r.json());
const overview = await fetch('/api/v1/enterprise/analytics/overview').then(r => r.json());

// ✅ Parallel (fast)
const [roles, events, overview] = await Promise.all([
  fetch('/api/v1/enterprise/rbac/roles').then(r => r.json()),
  fetch('/api/v1/enterprise/audit/events').then(r => r.json()),
  fetch('/api/v1/enterprise/analytics/overview').then(r => r.json())
]);
```

### 2. Request Caching

Cache static or slow-changing data:

```javascript
// Simple cache with TTL
const cache = new Map();
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

async function fetchWithCache(url, ttl = CACHE_TTL) {
  const cached = cache.get(url);
  if (cached && Date.now() - cached.timestamp < ttl) {
    return cached.data;
  }

  const response = await fetch(url);
  const data = await response.json();

  cache.set(url, { data, timestamp: Date.now() });
  return data;
}

// Usage: Cache permissions catalog (rarely changes)
const permissions = await fetchWithCache('/api/v1/enterprise/rbac/permissions', 60 * 60 * 1000); // 1 hour
```

### 3. Pagination Strategy

Use appropriate page sizes:

```javascript
// Small pages for fast initial load
const quickLoad = await fetch('/api/v1/enterprise/rbac/roles?page=1&per_page=10');

// Larger pages for data export
const bulkLoad = await fetch('/api/v1/enterprise/rbac/roles?page=1&per_page=100');
```

### 4. Request Debouncing

Debounce search requests:

```javascript
function debounce(func, wait) {
  let timeout;
  return function executedFunction(...args) {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

const debouncedSearch = debounce(async (query) => {
  const response = await fetch(
    `/api/v1/enterprise/audit/events/search?q=${encodeURIComponent(query)}`
  );
  const { data } = await response.json();
  // Update UI with results
}, 300); // 300ms delay

// Usage in search input
searchInput.addEventListener('input', (e) => debouncedSearch(e.target.value));
```

### 5. Target Page Load < 3s

Per requirements, initial page load must be < 3 seconds:

```javascript
// Load critical data first
const criticalData = await fetch('/api/v1/enterprise/analytics/overview').then(r => r.json());
// Render initial UI

// Load non-critical data in background
Promise.all([
  fetch('/api/v1/enterprise/audit/events?page=1&per_page=5'),
  fetch('/api/v1/enterprise/security/vulnerabilities?page=1&per_page=5')
]).then(([audit, security]) => {
  // Update UI with additional data
});
```

---

## TypeScript Definitions

### Core Types

```typescript
// responses.ts
export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: APIError;
  pagination?: Pagination;
  metadata: {
    timestamp: string;
    request_id: string;
  };
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, any>;
}

export interface Pagination {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

// RBAC Types
export interface Role {
  id: string;
  name: string;
  description: string;
  permissions: string[];
  created_at: string;
  updated_at: string;
}

export interface Permission {
  id: string;
  name: string;
  resource: string;
  action: string;
  description: string;
}

export interface UserRole {
  user_id: string;
  role_id: string;
  assigned_by: string;
  assigned_at: string;
}

// Audit Types
export interface AuditEvent {
  id: string;
  timestamp: string;
  user_id: string;
  user_email?: string;
  action: string;
  resource_type: string;
  resource_id: string;
  result: 'success' | 'failure' | 'denied';
  details?: Record<string, any>;
  ip_address: string;
  user_agent?: string;
}

export interface AuditStats {
  total_events: number;
  events_by_action: Record<string, number>;
  events_by_user: Record<string, number>;
  events_by_result: Record<string, number>;
  recent_denials: number;
  time_range: string;
}

// Analytics Types
export interface DashboardOverview {
  total_requests: number;
  cache_hit_rate: number;
  avg_response_time: number;
  bandwidth_used: number;
  active_users: number;
  top_packages: PackageUsage[];
  requests_by_pm: Record<string, number>;
  error_rate: number;
  timestamp: string;
}

export interface PackageUsage {
  package_manager: string;
  package_name: string;
  version?: string;
  request_count: number;
  bytes_served: number;
}

export interface PerformanceMetrics {
  time_range: string;
  avg_latency: number;
  p50_latency: number;
  p95_latency: number;
  p99_latency: number;
  throughput: number;
  error_rate: number;
  by_endpoint: Record<string, EndpointMetrics>;
}

export interface EndpointMetrics {
  endpoint: string;
  requests: number;
  avg_latency: number;
  p95_latency: number;
  error_count: number;
  error_rate: number;
}

// Security Types
export interface Vulnerability {
  id: string;
  package_name: string;
  package_version: string;
  package_manager: string;
  cve: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  description: string;
  published_date: string;
  fixed_version?: string;
  detected_at: string;
  status: 'open' | 'mitigated' | 'fixed' | 'ignored';
}

export interface Alert {
  id: string;
  type: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  title: string;
  description: string;
  source: string;
  triggered_at: string;
  status: 'active' | 'acknowledged' | 'resolved' | 'ignored';
  acked_by?: string;
  acked_at?: string;
  resolved_at?: string;
  metadata?: Record<string, any>;
}
```

### API Client

```typescript
// api-client.ts
export class EnterpriseAPIClient {
  private baseURL: string;
  private token?: string;

  constructor(baseURL: string = 'http://localhost:8080', token?: string) {
    this.baseURL = baseURL;
    this.token = token;
  }

  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<APIResponse<T>> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options?.headers,
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers,
    });

    const data = await response.json();

    if (!data.success) {
      throw new APIError(data.error.code, data.error.message, data.error.details);
    }

    return data;
  }

  // RBAC Methods
  async getRoles(page = 1, perPage = 20): Promise<APIResponse<Role[]>> {
    return this.request(`/api/v1/enterprise/rbac/roles?page=${page}&per_page=${perPage}`);
  }

  async getRole(id: string): Promise<APIResponse<Role>> {
    return this.request(`/api/v1/enterprise/rbac/roles/${id}`);
  }

  async createRole(role: Omit<Role, 'id' | 'created_at' | 'updated_at'>): Promise<APIResponse<Role>> {
    return this.request('/api/v1/enterprise/rbac/roles', {
      method: 'POST',
      body: JSON.stringify(role),
    });
  }

  async updateRole(id: string, updates: Partial<Role>): Promise<APIResponse<Role>> {
    return this.request(`/api/v1/enterprise/rbac/roles/${id}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async deleteRole(id: string): Promise<APIResponse<{ id: string; deleted: boolean }>> {
    return this.request(`/api/v1/enterprise/rbac/roles/${id}`, {
      method: 'DELETE',
    });
  }

  // Audit Methods
  async getAuditEvents(page = 1, perPage = 20, filters?: {
    user_id?: string;
    action?: string;
    resource_type?: string;
  }): Promise<APIResponse<AuditEvent[]>> {
    const params = new URLSearchParams({
      page: String(page),
      per_page: String(perPage),
      ...filters,
    });
    return this.request(`/api/v1/enterprise/audit/events?${params}`);
  }

  async getAuditStats(): Promise<APIResponse<AuditStats>> {
    return this.request('/api/v1/enterprise/audit/stats');
  }

  // Analytics Methods
  async getOverview(): Promise<APIResponse<DashboardOverview>> {
    return this.request('/api/v1/enterprise/analytics/overview');
  }

  async getPerformanceMetrics(): Promise<APIResponse<PerformanceMetrics>> {
    return this.request('/api/v1/enterprise/analytics/performance');
  }

  // Security Methods
  async getVulnerabilities(page = 1, perPage = 20, severity?: string): Promise<APIResponse<Vulnerability[]>> {
    const params = new URLSearchParams({
      page: String(page),
      per_page: String(perPage),
    });
    if (severity) params.set('severity', severity);
    return this.request(`/api/v1/enterprise/security/vulnerabilities?${params}`);
  }

  async triggerScan(scanRequest: {
    type: 'vulnerability' | 'license' | 'malware';
    package_manager?: string;
    full_scan: boolean;
  }): Promise<APIResponse<any>> {
    return this.request('/api/v1/enterprise/security/scan', {
      method: 'POST',
      body: JSON.stringify(scanRequest),
    });
  }

  // Alerts Methods
  async getAlerts(page = 1, perPage = 20, filters?: {
    status?: string;
    severity?: string;
  }): Promise<APIResponse<Alert[]>> {
    const params = new URLSearchParams({
      page: String(page),
      per_page: String(perPage),
      ...filters,
    });
    return this.request(`/api/v1/enterprise/alerts?${params}`);
  }
}

// Usage
const client = new EnterpriseAPIClient();

// Fetch roles
const { data: roles } = await client.getRoles(1, 20);

// Create role
const newRole = await client.createRole({
  name: 'Tester',
  description: 'QA access',
  permissions: ['cache:read']
});
```

---

## Testing

### Sample Test Suite

```typescript
// enterprise-api.test.ts
import { describe, it, expect } from 'vitest';
import { EnterpriseAPIClient } from './api-client';

describe('Enterprise API', () => {
  const client = new EnterpriseAPIClient('http://localhost:8080');

  describe('RBAC', () => {
    it('should list roles', async () => {
      const response = await client.getRoles();
      expect(response.success).toBe(true);
      expect(response.data).toBeInstanceOf(Array);
      expect(response.pagination).toBeDefined();
    });

    it('should get single role', async () => {
      const response = await client.getRole('role_admin');
      expect(response.success).toBe(true);
      expect(response.data.id).toBe('role_admin');
    });
  });

  describe('Audit', () => {
    it('should list audit events', async () => {
      const response = await client.getAuditEvents();
      expect(response.success).toBe(true);
      expect(response.pagination).toBeDefined();
    });

    it('should get audit stats', async () => {
      const response = await client.getAuditStats();
      expect(response.success).toBe(true);
      expect(response.data.total_events).toBeGreaterThan(0);
    });
  });
});
```

---

## Additional Resources

- **API Reference**: See `tmp/plan/README.md` for complete endpoint list
- **Integration Tests**: Run `make test-enterprise-integration`
- **Contract Tests**: Run `go test -tags=contract ./tests/contract/`
- **Mock Data**: Available in `tmp/fixtures/*.json`

---

## Support

For questions or issues:

- GitHub Issues: https://github.com/ScriptonBasestar/proxynd/issues
- Documentation: https://docs.proxynd.io
- Email: support@proxynd.io

---

**Last Updated**: 2025-01-17
**API Version**: v1
**License**: AGPL-3.0 (Community), Commercial (Enterprise)
