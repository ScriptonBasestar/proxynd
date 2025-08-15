# Tech Stack

## Core Technologies
- **Language**: Go 1.23+ (toolchain 1.24.4)
- **Framework**: Fiber v2.52.9 (High-performance HTTP web framework)
- **Database**: PostgreSQL 14+ / Redis 7+
- **Cloud Platform**: AWS (EKS, S3, RDS, ElastiCache)

## Development Tools
- **Build System**: Make with modular Makefiles (dev, build, test, quality, docker, deps, tools, clean)
- **Testing**: Testify v1.10.0 with comprehensive integration and E2E tests
- **Linting**: golangci-lint (configured in quality.mk)
- **Package Manager**: Go modules

## External Dependencies

### Core Dependencies
- **Web Framework**: `github.com/gofiber/fiber/v2` v2.52.9
- **Configuration**: `github.com/spf13/viper` v1.20.1, `gopkg.in/yaml.v3` v3.0.1
- **Authentication**: `github.com/golang-jwt/jwt/v5` v5.2.2
- **OAuth2 Providers**: GitHub, GitLab, Google OAuth2 integration
- **Multi-Factor Auth**: `github.com/pquerna/otp` v1.5.0 (TOTP support)
- **AWS SDK**: `github.com/aws/aws-sdk-go-v2` v1.36.5 (S3 caching)
- **HTTP Client**: `github.com/valyala/fasthttp` v1.63.0
- **Validation**: `github.com/go-playground/validator/v10` v10.27.0
- **Rate Limiting**: `github.com/ulule/limiter/v3` v3.11.2

### Development Dependencies
- **Testing**: `github.com/stretchr/testify` v1.10.0
- **Mocking**: `github.com/go-playground/assert/v2` v2.2.0
- **CLI Framework**: `github.com/spf13/cobra` v1.9.1
- **UUID Generation**: `github.com/google/uuid` v1.6.0
- **QR Code**: `github.com/skip2/go-qrcode` (MFA setup)

### Logging Dependencies
- **Structured Logging**: `github.com/rs/zerolog` v1.34.0
- **Classic Logging**: `github.com/sirupsen/logrus` v1.9.3
- **High-Performance**: `go.uber.org/zap` v1.27.0
- **Log Rotation**: `gopkg.in/natefinch/lumberjack.v2` v2.2.1

### Monitoring Dependencies
- **Metrics**: `github.com/prometheus/client_golang` v1.22.0
- **Template Engine**: `github.com/gofiber/template/html/v2` v2.1.3

## Architecture Overview

ProxyND is a high-performance, multi-format package manager proxy server designed with a clean architecture pattern:

### Core Architecture
- **Unified Proxy Handler**: Single handler supporting multiple package managers
- **Container-based Architecture**: Docker-first design with multi-arch support
- **Modular Design**: Clean separation of concerns with domain-driven structure
- **Middleware Pipeline**: Comprehensive security, authentication, and monitoring

### Supported Package Managers
- **Maven** (Java/Kotlin/Scala) - Repository proxy and mirroring
- **NPM** (Node.js) - Package proxy with metadata caching
- **APT** (Ubuntu/Debian) - Repository mirroring with signature verification
- **Docker Registry** - Container image proxy with blob management
- **PyPI** (Python) - Package index proxy and caching
- **YUM** (RedHat/CentOS) - Repository proxy with metadata processing
- **APK** (Alpine Linux) - Package proxy with signature verification

### Key Features
- **Multi-tier Caching**: Filesystem and S3-based caching strategies
- **Health Monitoring**: Comprehensive health checks with auto-recovery
- **Security**: OAuth2 integration, JWT authentication, MFA support
- **Performance**: Connection pooling, rate limiting, request optimization
- **Observability**: Prometheus metrics, structured logging, distributed tracing

### Infrastructure Components
- **Primary Storage**: Filesystem-based caching with S3 backup
- **Session Management**: Redis for authentication and rate limiting
- **Load Balancing**: Nginx for SSL termination and request distribution
- **Service Discovery**: Kubernetes-native with Helm chart deployment

## Deployment

### Container Deployment
- **Docker**: Multi-arch containers (AMD64, ARM64) with optimized builds
- **Docker Compose**: Production-ready stack with Redis, Prometheus, Grafana
- **Base Images**: Alpine Linux for minimal attack surface

### Kubernetes Deployment
- **Helm Charts**: Production-ready Helm v3 charts with comprehensive configuration
- **Platform Support**: EKS, GKE, AKS with cloud-specific optimizations
- **Scaling**: Horizontal Pod Autoscaler with custom metrics
- **Storage**: Persistent volumes for cache storage with S3 backup

### AWS Infrastructure (Terraform)
- **EKS Cluster**: Managed Kubernetes with auto-scaling node groups
- **RDS PostgreSQL**: Metadata storage with automated backups
- **ElastiCache Redis**: Session and cache management
- **S3 Buckets**: Distributed cache storage with lifecycle policies
- **Application Load Balancer**: High-availability traffic distribution
- **CloudWatch**: Centralized logging and monitoring

### Monitoring Stack
- **Prometheus**: Metrics collection and alerting rules
- **Grafana**: Visualization dashboards for operations and security
- **AlertManager**: Alert routing and notification management
- **Loki**: Log aggregation and correlation (optional)

### Security Features
- **TLS Termination**: SSL/TLS encryption with automatic certificate management
- **Access Control**: Role-based access with OAuth2 provider integration
- **Network Security**: VPC isolation, security groups, network policies
- **Secret Management**: AWS SSM Parameter Store for secure configuration
- **Compliance**: Security scanning, vulnerability assessment integration
