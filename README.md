# ProxyND Core Edition

> High-performance package manager proxy/mirror server supporting 7 package managers

**Edition**: Community (Open Source - AGPL-3.0)

---

## 📋 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Supported Package Managers](#supported-package-managers)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [API Endpoints](#api-endpoints)
- [CLI Management Tool](#cli-management-tool)
- [Authentication](#authentication)
- [Caching](#caching)
- [Development](#development)
- [Deployment](#deployment)
- [Documentation](#documentation)
- [License](#license)

---

## 🎯 Overview

ProxyND Core is the **community edition** of ProxyND, providing essential package manager proxy functionality with modern authentication, flexible caching, and production-ready features.

### Key Characteristics

- **Open Source**: AGPL-3.0 licensed
- **Production Ready**: Battle-tested hexagonal architecture
- **Modern Auth**: OAuth2, JWT, MFA (TOTP/WebAuthn)
- **Flexible Caching**: Filesystem, Redis, S3 backends
- **7 Package Managers**: Maven, NPM, APT, Docker Registry, PyPI, YUM, APK
- **High Performance**: Built with Go and Fiber v2 framework
- **Cloud Native**: Docker, Kubernetes, Helm ready

---

## ✨ Features

### Package Management
- ✅ **Maven** - Java/Kotlin/Scala artifact caching
- ✅ **NPM** - Node.js package registry proxy
- ✅ **APT** - Ubuntu/Debian package caching
- ✅ **Docker Registry** - Container image proxy
- ✅ **PyPI** - Python package index mirror
- ✅ **YUM** - RedHat/CentOS package proxy
- ✅ **APK** - Alpine Linux package mirror

### Authentication & Security
- ✅ **OAuth2** - GitHub, GitLab, Google integration
- ✅ **JWT Tokens** - Stateless authentication
- ✅ **MFA** - TOTP and WebAuthn support (enabled by Enterprise plugin)
- ✅ **API Keys** - Basic authentication with key management
- ✅ **Basic Auth** - Username/password authentication

### Caching Backends
- ✅ **Filesystem** - Local disk caching
- ✅ **Redis** - Single instance in-memory cache
- ✅ **S3 Compatible** - AWS S3, MinIO, etc.
- ✅ **Multi-tier** - Configurable cache hierarchy

### Core Infrastructure
- ✅ **Connection Pooling** - Efficient upstream connections
- ✅ **Health Checks** - Readiness and liveness probes
- ✅ **Metrics** - Prometheus metrics export
- ✅ **Structured Logging** - JSON formatted logs
- ✅ **Hot Reload** - Configuration reload without restart

---

## 📦 Supported Package Managers

### Maven (Java/Kotlin/Scala)
**Mount Point**: `/maven`

```bash
# Configure Maven to use ProxyND
<mirror>
  <id>proxynd</id>
  <url>http://localhost:8080/maven/central</url>
  <mirrorOf>central</mirrorOf>
</mirror>
```

**Features**:
- Maven Central, JCenter, custom repositories
- POM and artifact caching
- Checksum validation
- Index incremental updates

### NPM (Node.js)
**Mount Point**: `/npm`

```bash
# Configure NPM to use ProxyND
npm config set registry http://localhost:8080/npm/registry

# Or use .npmrc
registry=http://localhost:8080/npm/registry
```

**Features**:
- npm registry mirror
- Package metadata caching
- Tarball caching
- Scoped packages support

### APT (Ubuntu/Debian)
**Mount Point**: `/apt`

```bash
# Configure APT sources
deb http://localhost:8080/apt/ubuntu focal main
deb http://localhost:8080/apt/ubuntu focal-updates main
```

**Features**:
- Package list caching
- .deb file caching
- Repository metadata
- GPG signature validation

### Docker Registry
**Mount Point**: `/docker`

```bash
# Configure Docker daemon
{
  "registry-mirrors": ["http://localhost:8080/docker"]
}
```

**Features**:
- Docker Hub mirror
- Private registry proxy
- Layer caching
- Manifest caching

### PyPI (Python)
**Mount Point**: `/pypi`

```bash
# Configure pip
pip config set global.index-url http://localhost:8080/pypi/simple

# Or use pip.conf
[global]
index-url = http://localhost:8080/pypi/simple
```

**Features**:
- PyPI simple API mirror
- Wheel and source distribution caching
- Package metadata
- Version resolution

### YUM (RedHat/CentOS)
**Mount Point**: `/yum`

```bash
# Configure YUM repository
[proxynd]
name=ProxyND YUM Mirror
baseurl=http://localhost:8080/yum/centos/8/
enabled=1
```

**Features**:
- RPM package caching
- Repository metadata
- GPG key management
- Update automation

### APK (Alpine Linux)
**Mount Point**: `/apk`

```bash
# Configure APK repository
echo "http://localhost:8080/apk/alpine/v3.18/main" > /etc/apk/repositories
```

**Features**:
- APK package caching
- Repository indexes
- Signature verification
- Edge repository support

---

## 🚀 Quick Start

### Prerequisites

- Go 1.23+ (toolchain 1.24.4)
- Docker (optional, for containerized deployment)

### 1. Development Setup

```bash
# Clone repository
git clone https://github.com/yourusername/proxynd.git
cd proxynd/proxynd-core

# Install dependencies
make dev-prepare

# Setup configuration
make dev-setup

# Run development server (with hot reload)
make dev-run
```

### 2. Build from Source

```bash
# Build binary
make build

# Binary location: tmp/bin/proxynd
./tmp/bin/proxynd --version
```

### 3. Docker Deployment

```bash
# Build Docker image
make docker-build

# Run container
make docker-run

# Or use docker-compose
docker-compose up -d
```

### 4. Kubernetes Deployment

```bash
# Using Helm chart
helm install proxynd ./charts/core \
  --set config.storage.path=/data \
  --set config.cache.backend=s3

# Or using kubectl
kubectl apply -f k8s/core/
```

---

## ⚙️ Configuration

### Configuration File Locations (Priority Order)

1. `--config` flag
2. `./config.yaml` (current directory)
3. `~/.config/proxynd/config.yaml` (user config)
4. `/etc/proxynd/config.yaml` (system config)

### Minimal Configuration

```yaml
# config.minimal.yaml
server:
  port: 8080
  host: 0.0.0.0

storage:
  path: ./storage

cache:
  backend: filesystem
  ttl: 24h

proxies:
  - type: maven
    enabled: true
    upstream: https://repo1.maven.org/maven2

  - type: npm
    enabled: true
    upstream: https://registry.npmjs.org

auth:
  enabled: false
```

### Production Configuration

```yaml
# config.production.yaml
server:
  port: 8080
  host: 0.0.0.0
  read_timeout: 30s
  write_timeout: 30s

storage:
  path: /var/lib/proxynd

cache:
  backend: s3
  ttl: 168h  # 7 days
  s3:
    bucket: proxynd-cache
    region: us-east-1
    endpoint: https://s3.amazonaws.com

proxies:
  - type: maven
    enabled: true
    upstream: https://repo1.maven.org/maven2
    cache_ttl: 720h  # 30 days

  - type: npm
    enabled: true
    upstream: https://registry.npmjs.org
    cache_ttl: 168h  # 7 days

  - type: docker
    enabled: true
    upstream: https://registry-1.docker.io
    cache_ttl: 168h

auth:
  enabled: true
  providers:
    - type: oauth2
      provider: github
      client_id: ${GITHUB_CLIENT_ID}
      client_secret: ${GITHUB_CLIENT_SECRET}

    - type: jwt
      secret: ${JWT_SECRET}
      expiry: 24h

metrics:
  enabled: true
  path: /metrics
  prometheus:
    enabled: true

logging:
  level: info
  format: json
  file: /var/log/proxynd/proxynd.log
```

### Environment Variables

```bash
# Required
export CONFIG_DIR=/etc/proxynd
export STORAGE_DIR=/var/lib/proxynd

# Optional
export SERVER_PORT=8080
export LOG_LEVEL=info
export LOG_FORMAT=json

# OAuth2 credentials
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
export JWT_SECRET=your_jwt_secret

# S3 credentials (if using S3 cache)
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
```

---

## 🏛️ Architecture

ProxyND Core follows **Hexagonal Architecture (Ports and Adapters)** pattern for maintainability and testability.

### Directory Structure

```
internal/
├── domain/          # Pure business logic (NO external dependencies)
│   ├── types.go     # Domain entities
│   └── errors.go    # Domain errors
├── usecase/         # Business workflows (framework-independent)
│   ├── proxy_service.go    # Core proxy logic
│   ├── cache_strategy.go   # Cache management
│   └── health.go           # Health checks
├── ports/           # Interface contracts
│   ├── http.go      # HTTP server interface
│   ├── pm.go        # Package manager interface
│   ├── cache.go     # Cache backend interface
│   ├── auth.go      # Authentication interface
│   └── observability.go    # Metrics/logging interface
└── adapters/        # External system implementations
    ├── http/fiber/  # Fiber web framework adapter
    ├── pm/          # Package manager drivers
    │   ├── maven/
    │   ├── npm/
    │   ├── apt/
    │   ├── docker/
    │   ├── pypi/
    │   ├── yum/
    │   └── apk/
    ├── cache/       # Cache backend implementations
    │   ├── filesystem/
    │   ├── redis/
    │   └── s3/
    └── auth/        # Authentication implementations
        ├── oauth2/
        ├── jwt/
        └── apikey/
```

### Design Principles

- **Unidirectional Dependencies**: `adapters → ports → usecase → domain`
- **Framework Independence**: Business logic doesn't depend on web frameworks
- **Testability**: All external dependencies abstracted via interfaces
- **Extensibility**: Easy to add new package managers or cache backends

### Dependency Flow

```
┌─────────────────────────────────────────┐
│           HTTP Adapter (Fiber)          │
│  (adapters/http/fiber)                  │
└──────────────┬──────────────────────────┘
               │ implements
               ↓
┌─────────────────────────────────────────┐
│           HTTP Port Interface           │
│  (ports/http.go)                        │
└──────────────┬──────────────────────────┘
               │ uses
               ↓
┌─────────────────────────────────────────┐
│         Proxy Service (Usecase)         │
│  (usecase/proxy_service.go)             │
└──────────────┬──────────────────────────┘
               │ uses
               ↓
┌─────────────────────────────────────────┐
│    Package Manager Port Interface       │
│  (ports/pm.go)                          │
└──────────────┬──────────────────────────┘
               │ implemented by
               ↓
┌─────────────────────────────────────────┐
│    Package Manager Adapters             │
│  (adapters/pm/maven, npm, etc.)         │
└─────────────────────────────────────────┘
```

---

## 🔌 API Endpoints

### Core API

**Health Check**
```bash
GET /health
GET /ready
```

**Metrics**
```bash
GET /metrics  # Prometheus format
```

### Package Manager Endpoints

**Maven**
```bash
GET /maven/{repository}/{group}/{artifact}/{version}/{file}
HEAD /maven/{repository}/{group}/{artifact}/{version}/{file}
```

**NPM**
```bash
GET /npm/registry/{package}
GET /npm/registry/{package}/-/{tarball}
```

**Docker**
```bash
GET /v2/
GET /v2/{name}/manifests/{reference}
GET /v2/{name}/blobs/{digest}
```

**PyPI**
```bash
GET /pypi/simple/{package}/
GET /pypi/packages/{path}
```

### Management API

**Cache Management**
```bash
GET    /api/cache/list        # List cached items
DELETE /api/cache/clear       # Clear cache
GET    /api/cache/stats       # Cache statistics
```

**Configuration**
```bash
GET  /api/config/show         # Show current config
POST /api/config/validate     # Validate config
POST /api/config/reload       # Reload config
```

**User Management**
```bash
GET    /api/users             # List users
POST   /api/users             # Create user
DELETE /api/users/:id         # Delete user
```

**Testing**
```bash
GET /api/test/all             # Test all proxies
GET /api/test/:proxy          # Test specific proxy
```

**Status**
```bash
GET /api/status               # Server status
```

For complete API documentation, see [API_ENDPOINTS.md](docs/API_ENDPOINTS.md).

---

## 🔧 CLI Management Tool

ProxyND provides `proxyndctl` CLI for administrative tasks.

### Installation

```bash
# Build CLI tool
make build-cli

# Binary location: tmp/bin/proxyndctl
./tmp/bin/proxyndctl --help
```

### Usage Examples

**Cache Management**
```bash
# List cache entries
proxyndctl cache list

# Clear cache by type
proxyndctl cache clear --type npm

# Show cache statistics
proxyndctl cache size
```

**Configuration Management**
```bash
# Validate configuration
proxyndctl config validate

# Show current configuration
proxyndctl config show

# Reload configuration
proxyndctl config reload
```

**User Management**
```bash
# List users
proxyndctl user list

# Add user
proxyndctl user add john --email john@example.com

# Delete user
proxyndctl user delete john
```

**Proxy Testing**
```bash
# Test all proxies
proxyndctl test all

# Test specific proxy
proxyndctl test --proxy maven

# Test with custom upstream
proxyndctl test --proxy npm --upstream https://registry.npmjs.org
```

**Server Status**
```bash
# Show server status
proxyndctl status

# Show detailed metrics
proxyndctl status --metrics
```

**Maven Extensions**
```bash
# Build search index
proxyndctl maven-index build

# Create incremental backup
proxyndctl maven-backup create
```

---

## 🔐 Authentication

### OAuth2 Providers

**GitHub**
```yaml
auth:
  enabled: true
  providers:
    - type: oauth2
      provider: github
      client_id: ${GITHUB_CLIENT_ID}
      client_secret: ${GITHUB_CLIENT_SECRET}
      redirect_url: http://localhost:8080/auth/github/callback
```

**GitLab**
```yaml
auth:
  providers:
    - type: oauth2
      provider: gitlab
      client_id: ${GITLAB_CLIENT_ID}
      client_secret: ${GITLAB_CLIENT_SECRET}
      redirect_url: http://localhost:8080/auth/gitlab/callback
```

**Google**
```yaml
auth:
  providers:
    - type: oauth2
      provider: google
      client_id: ${GOOGLE_CLIENT_ID}
      client_secret: ${GOOGLE_CLIENT_SECRET}
      redirect_url: http://localhost:8080/auth/google/callback
```

### JWT Tokens

```yaml
auth:
  providers:
    - type: jwt
      secret: ${JWT_SECRET}
      expiry: 24h
      algorithm: HS256
```

**Usage**:
```bash
# Login to get JWT token
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}'

# Use JWT token
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/api/cache/list
```

### MFA (Multi-Factor Authentication)

MFA is implemented in Core but enabled via Enterprise plugin.

**TOTP (Time-based One-Time Password)**
```bash
# Enable TOTP
curl -X POST http://localhost:8080/auth/mfa/totp/enable

# Verify TOTP
curl -X POST http://localhost:8080/auth/mfa/totp/verify \
  -d '{"code":"123456"}'
```

**WebAuthn (Hardware Keys)**
```bash
# Register security key
curl -X POST http://localhost:8080/auth/mfa/webauthn/register

# Authenticate with security key
curl -X POST http://localhost:8080/auth/mfa/webauthn/verify
```

### API Keys

```yaml
auth:
  api_keys:
    enabled: true
    keys:
      - key: ${API_KEY_1}
        user: admin
        permissions: ["read", "write"]
      - key: ${API_KEY_2}
        user: readonly
        permissions: ["read"]
```

**Usage**:
```bash
curl -H "X-API-Key: YOUR_API_KEY" \
  http://localhost:8080/api/cache/list
```

---

## 💾 Caching

### Filesystem Backend

```yaml
cache:
  backend: filesystem
  filesystem:
    path: /var/cache/proxynd
    max_size: 100GB
    cleanup_interval: 1h
```

**Features**:
- Simple and reliable
- No external dependencies
- Automatic cleanup on size limit
- grep-friendly for debugging

### Redis Backend

```yaml
cache:
  backend: redis
  redis:
    host: localhost
    port: 6379
    password: ${REDIS_PASSWORD}
    db: 0
    ttl: 24h
```

**Features**:
- Fast in-memory caching
- Automatic TTL expiration
- Single instance mode (Enterprise supports cluster)

### S3 Compatible Backend

```yaml
cache:
  backend: s3
  s3:
    bucket: proxynd-cache
    region: us-east-1
    endpoint: https://s3.amazonaws.com
    access_key: ${AWS_ACCESS_KEY_ID}
    secret_key: ${AWS_SECRET_ACCESS_KEY}
```

**Compatible Storage**:
- AWS S3
- MinIO
- DigitalOcean Spaces
- Backblaze B2
- Any S3-compatible storage

### Multi-tier Caching

```yaml
cache:
  backend: multi-tier
  tiers:
    - backend: memory
      ttl: 5m
      max_size: 1GB
    - backend: filesystem
      ttl: 24h
      max_size: 50GB
    - backend: s3
      ttl: 720h  # 30 days
```

---

## 🛠️ Development

### Development Environment

```bash
# Setup development environment
make dev-prepare
make dev-setup

# Run with hot reload (Air)
make dev-run

# Run tests
make test-unit
make test-integration

# Quick validation (format + lint + test)
make quick
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run all tests
make test

# Generate coverage report
make coverage
```

### Adding a New Package Manager

1. **Create adapter** in `adapters/pm/newpm/`:
```go
type NewPMAdapter struct {
    upstream string
}

func (a *NewPMAdapter) FetchPackage(ctx context.Context, path string) ([]byte, error) {
    // Implementation
}
```

2. **Register in ports** (`ports/pm.go`):
```go
type PackageManagerType string

const (
    PMTypeNewPM PackageManagerType = "newpm"
)
```

3. **Add configuration** in `examples/config.yaml`:
```yaml
proxies:
  - type: newpm
    enabled: true
    upstream: https://newpm-registry.example.com
```

4. **Write tests** in `adapters/pm/newpm/adapter_test.go`

5. **Update documentation**

For detailed development guide, see [docs/05-development/README.md](docs/05-development/README.md).

---

## 🚀 Deployment

### Docker

**Build Image**:
```bash
make docker-build
```

**Run Container**:
```bash
docker run -d \
  --name proxynd \
  -p 8080:8080 \
  -e CONFIG_DIR=/etc/proxynd \
  -e STORAGE_DIR=/var/lib/proxynd \
  -v $(pwd)/config.yaml:/etc/proxynd/config.yaml \
  -v proxynd-data:/var/lib/proxynd \
  proxynd:latest
```

**Docker Compose**:
```yaml
version: '3.8'
services:
  proxynd:
    image: proxynd:latest
    ports:
      - "8080:8080"
    environment:
      CONFIG_DIR: /etc/proxynd
      STORAGE_DIR: /var/lib/proxynd
    volumes:
      - ./config.yaml:/etc/proxynd/config.yaml
      - proxynd-data:/var/lib/proxynd

volumes:
  proxynd-data:
```

### Kubernetes

**Using Helm**:
```bash
# Add Helm repository
helm repo add proxynd https://charts.proxynd.io
helm repo update

# Install
helm install proxynd proxynd/core \
  --set config.storage.path=/data \
  --set config.cache.backend=s3 \
  --set config.cache.s3.bucket=proxynd-cache
```

**Using kubectl**:
```bash
# Apply manifests
kubectl apply -f k8s/core/namespace.yaml
kubectl apply -f k8s/core/configmap.yaml
kubectl apply -f k8s/core/deployment.yaml
kubectl apply -f k8s/core/service.yaml
```

**Helm Chart Values**:
```yaml
# values.yaml
replicaCount: 3

image:
  repository: proxynd
  tag: latest
  pullPolicy: IfNotPresent

resources:
  limits:
    cpu: 1000m
    memory: 2Gi
  requests:
    cpu: 500m
    memory: 1Gi

config:
  server:
    port: 8080
  storage:
    path: /data
  cache:
    backend: s3
    s3:
      bucket: proxynd-cache
      region: us-east-1

persistence:
  enabled: true
  size: 100Gi
  storageClass: standard
```

### Terraform

```hcl
module "proxynd" {
  source = "./terraform/modules/proxynd"

  environment = "production"
  region      = "us-east-1"

  instance_type = "t3.large"
  storage_size  = 100

  config = {
    cache_backend = "s3"
    s3_bucket     = "proxynd-cache-prod"
  }
}
```

---

## 📚 Documentation

### Core Documentation
- [Getting Started](docs/01-getting-started/README.md)
- [Configuration Guide](docs/03-configuration/README.md)
- [Development Guide](docs/05-development/README.md)
- [API Endpoints](docs/API_ENDPOINTS.md)
- [Testing Strategy](TESTING.md)

### AI Collaboration
- [Claude Code Guardrails](CLAUDE.md) - Essential guide for large refactoring
- [Contributing Guide](CONTRIBUTING.md) - General contributions and AI-assisted development
- [Refactoring Checklist](REFACTORING_CHECKLIST.md) - Step-by-step migration procedures
- [Tech Stack](TECH_STACK.md) - Technology stack details

### Architecture
- [Hexagonal Architecture](docs/ARCHITECTURE.md)
- [Design Principles](docs/DESIGN_PRINCIPLES.md)
- [Code Conventions](docs/CODE_CONVENTIONS.md)

---

## 🏢 Enterprise API

ProxyND provides a comprehensive REST API for enterprise features including RBAC, audit logging, analytics, and security scanning.

### Available Endpoints (47 Total)

| Category | Endpoints | Description |
|----------|-----------|-------------|
| **RBAC** | 12 | Role-based access control, permissions, user assignments |
| **Audit** | 8 | Event logging, compliance reports, audit trail export |
| **Analytics** | 10 | Usage stats, performance metrics, cost analysis |
| **Security** | 8 | Vulnerability scanning, license compliance, malware detection |
| **Alerts** | 5 | Alert management and notification rules |
| **License** | 4 | License validation and feature management |

### Quick Start

```bash
# Start development server
make dev-run

# Test all enterprise endpoints
make test-enterprise-integration

# Test specific categories
make test-enterprise-rbac
make test-enterprise-audit
make test-enterprise-analytics
```

### Example Requests

**List Roles (RBAC)**
```bash
curl http://localhost:8080/api/v1/enterprise/rbac/roles
```

**Get Audit Events**
```bash
curl http://localhost:8080/api/v1/enterprise/audit/events?page=1&per_page=20
```

**Analytics Overview**
```bash
curl http://localhost:8080/api/v1/enterprise/analytics/overview
```

**List Vulnerabilities**
```bash
curl http://localhost:8080/api/v1/enterprise/security/vulnerabilities
```

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

### License Validation

Enterprise API endpoints require a valid enterprise license:

- **Development Mode**: License check bypassed (set `GO_ENV != production`)
- **Production Mode**: Valid license required in `CONFIG_DIR/license.json`
- **Unauthorized Access**: Returns `HTTP 402 Payment Required`

### Testing & Documentation

```bash
# Run integration tests
./tmp/scripts/test-webui-integration.sh all

# Run contract tests
go test -tags=contract ./tests/contract/enterprise_api_test.go

# Generate Swagger documentation
make swagger

# View API documentation
cat tmp/plan/README.md
```

### Swagger/OpenAPI Documentation

ProxyND provides interactive API documentation via Swagger UI:

```bash
# Generate Swagger documentation from code annotations
make swagger

# Start the server
make dev-run

# Access Swagger UI (interactive API explorer)
open http://localhost:8080/swagger/index.html
```

The Swagger documentation includes:
- All 47 enterprise endpoints with request/response schemas
- Interactive API testing directly from the browser
- Authentication configuration
- Example requests and responses
- Complete parameter documentation

### Postman Collection

Complete Postman collection with all 47 endpoints for easy API testing:

**Files** (in `postman/` directory):
- `ProxyND-Enterprise-API.postman_collection.json` - Full API collection
- `ProxyND-Dev.postman_environment.json` - Development environment
- `README.md` - Detailed usage guide

**Quick Import**:
1. Open Postman → File → Import
2. Select `postman/ProxyND-Enterprise-API.postman_collection.json`
3. Select `postman/ProxyND-Dev.postman_environment.json`
4. Choose "ProxyND Development" environment
5. Start testing!

The collection includes:
- All 47 endpoints organized by category (RBAC, Audit, Analytics, Security, Alerts, License)
- Pre-configured request examples with sample data
- Environment variables for easy server switching
- Complete request/response documentation

### Performance Benchmarks

ProxyND Enterprise API is designed for high performance with strict targets:

**Performance Targets:**
- **P95 Response Time**: < 500ms (95% of requests)
- **P99 Response Time**: < 1000ms (99% of requests)
- **Page Load Time**: < 3s (complete WebUI page load)
- **Throughput**: > 100 requests/second per endpoint

**Run Benchmarks:**
```bash
# Standard benchmark (100 iterations)
make test-benchmark-enterprise

# Quick test (25 iterations)
make test-benchmark-enterprise-quick

# Stress test (500 iterations, 50 concurrent)
make test-benchmark-enterprise-stress

# Custom configuration
ITERATIONS=200 CONCURRENT=30 ./scripts/benchmark-enterprise-api.sh
```

The benchmark suite tests 13 representative endpoints across all categories with:
- Single endpoint response time tests
- Pagination performance tests
- Concurrent request handling tests

**Detailed Guide**: See `docs/enterprise/PERFORMANCE_BENCHMARKS.md` for optimization strategies and troubleshooting.

**WebUI Integration Guide**: See `docs/enterprise/WEBUI_INTEGRATION.md` for complete TypeScript/React integration examples.

**API Documentation**: See `tmp/plan/README.md` for detailed endpoint documentation (auto-generated in dev mode).

---

## 🔄 Edition Comparison

| Feature | Core (Community) | Enterprise | Cloud |
|---------|------------------|------------|-------|
| **Package Managers** | 7 (Maven, NPM, APT, Docker, PyPI, YUM, APK) | ✅ | ✅ |
| **OAuth2 (GitHub, GitLab, Google)** | ✅ | ✅ | ✅ |
| **JWT Tokens** | ✅ | ✅ | ✅ |
| **MFA (TOTP/WebAuthn)** | ✅ (Implementation) | ✅ (Enabled) | ✅ |
| **API Keys** | ✅ | ✅ | ✅ |
| **Filesystem Cache** | ✅ | ✅ | ✅ |
| **Redis Cache** | ✅ (Single) | ✅ (Cluster) | ✅ |
| **S3 Cache** | ✅ | ✅ | ✅ |
| **RBAC** | ❌ | ✅ | ✅ |
| **Audit Logging** | ❌ | ✅ | ✅ |
| **LDAP/SAML** | ❌ | ✅ (Upcoming) | ✅ |
| **Multi-Tenancy** | ❌ | ❌ | ✅ |
| **Usage Billing** | ❌ | ❌ | ✅ |
| **Quota Management** | ❌ | ❌ | ✅ |
| **License** | AGPL-3.0 | Commercial | Commercial |

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style

- Follow Go best practices
- Use `gofmt` for formatting
- Run `golangci-lint` before commit
- Write tests for new features
- Update documentation

---

## 📄 License

ProxyND Core is dual-licensed:

- **Open Source**: [AGPL-3.0](LICENSE) - Free with source disclosure requirements
- **Commercial**: [Commercial License](LICENSE-COMMERCIAL.md) - For proprietary use and enterprise features

For more details, see [LICENSING.md](LICENSING.md).

---

## 🆘 Support

- **Documentation**: [docs/INDEX.md](docs/INDEX.md)
- **Issues**: [GitHub Issues](https://github.com/yourusername/proxynd/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/proxynd/discussions)
- **Commercial Support**: contact@proxynd.io

---

## 🎯 Roadmap

### v1.0.0 (Current)
- ✅ Core proxy functionality for 7 package managers
- ✅ OAuth2 + JWT + MFA authentication
- ✅ Multi-backend caching (Filesystem, Redis, S3)
- ✅ Prometheus metrics
- ✅ CLI management tool

### v1.1.0 (Next)
- 📋 Enhanced caching strategies
- 📋 Improved metrics and dashboards
- 📋 Performance optimizations
- 📋 Additional package manager support

### v2.0.0 (Future)
- 📋 Plugin system for extensibility
- 📋 GraphQL API
- 📋 Advanced cache warming
- 📋 Distributed caching

---

> ℹ️ For detailed configuration and usage, refer to the [documentation](docs/INDEX.md).
