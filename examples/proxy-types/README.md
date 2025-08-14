# Proxy Type Configuration Examples

This directory contains comprehensive configuration examples for each supported package manager proxy type, including both **proxy** and **mirror** variants with **authenticated** and **non-authenticated** scenarios.

## Available Proxy Types

### APK (Alpine Linux)
- `apk-proxy.yaml` - Alpine Linux package proxy with signature verification
- `apk-mirror.yaml` - **🆕** Complete Alpine Linux mirror with multi-architecture support

### APT (Debian/Ubuntu)
- `apt-proxy.yaml` - Standard APT proxy configuration with multiple distributions
- `apt-mirror.yaml` - Full APT mirror configuration with security repositories

### Docker Registry
- `docker-proxy.yaml` - **✨ Enhanced** Docker registry proxy with public/private authentication
- `docker-mirror.yaml` - **🆕** Complete Docker registry mirror with ECR/GCR/ACR support

### Maven (Java)
- `maven-proxy.yaml` - **✨ Enhanced** Maven proxy with Nexus/GitHub/certificate authentication
- `maven-mirror.yaml` - Maven repository mirror with multiple remotes
- `maven-host.yaml` - Host your own Maven artifacts

### NPM (Node.js)
- `npm-proxy.yaml` - **✨ Enhanced** NPM proxy with private registry authentication
- `npm-mirror.yaml` - **🆕** Complete NPM mirror with package filtering and prioritization

### PIP (Python)
- `pip-proxy.yaml` - **✨ Enhanced** PyPI proxy with DevPI/CodeArtifact/Nexus authentication
- `pip-mirror.yaml` - **🆕** Complete PyPI mirror with package filtering and security scanning

### YUM (RedHat/CentOS/Rocky)
- `yum-proxy.yaml` - YUM repository proxy for multiple distributions
- `yum-mirror.yaml` - **🆕** Complete YUM mirror with GPG verification and multi-distro support

### Vagrant
- `vagrant-proxy.yaml` - Vagrant box repository proxy

## Configuration Variants

### 🔐 Authentication Examples
Each proxy configuration now includes comprehensive authentication examples:

- **Basic Authentication**: Username/password for private repositories
- **Token Authentication**: API tokens for GitHub Packages, NPM, PyPI
- **Certificate Authentication**: Client certificates for enterprise repositories
- **Cloud Provider Authentication**: AWS ECR, Azure ACR, GCP GCR integration

### 🏛️ Proxy vs Mirror
- **Proxy Mode (`/proxy/*`)**: Pass-through caching proxy for on-demand package retrieval
- **Mirror Mode (`/mirror/*`)**: Complete repository synchronization with local hosting

## Usage Examples

### Quick Start - Single Proxy
```bash
# NPM proxy with authentication
cp npm-proxy.yaml ../../config.yaml

# Docker mirror with private registry support
cp docker-mirror.yaml ../../config.yaml
```

### Enterprise Setup - Multiple Services
```bash
# Combine multiple proxy types
cat npm-proxy.yaml maven-proxy.yaml docker-proxy.yaml > ../../config.yaml

# Full mirror setup for offline environments
cat npm-mirror.yaml pip-mirror.yaml yum-mirror.yaml > ../../config.yaml
```

### Environment Variable Setup
```bash
# Set authentication credentials
export NPM_TOKEN="npm_xxxxxxxxxxxx"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
export NEXUS_USERNAME="deploy"
export NEXUS_PASSWORD="secret"

# Start ProxyND with configuration
proxynd -config config.yaml
```

## Configuration Guidelines

### 🔧 Basic Configuration
- Each proxy type can be enabled/disabled independently
- Multiple proxy types can run simultaneously on different paths
- Cache settings can be configured per proxy type
- All configurations support environment variable substitution

### 🔒 Security Best Practices
- **Never commit credentials** to version control - use environment variables
- **Enable signature verification** for package integrity (APK, YUM, APT)
- **Use certificate authentication** for enhanced security in enterprise environments
- **Configure IP restrictions** and rate limiting as needed

### 📊 Performance Optimization
- **Mirror mode** for high-traffic, offline, or air-gapped environments
- **Proxy mode** for on-demand caching and bandwidth optimization
- **Regional mirrors** configuration for global deployments
- **Storage backends** (file, S3, GCS) for scalable deployments

### 🎯 Use Case Scenarios

#### Development Teams
```yaml
# npm-proxy.yaml + maven-proxy.yaml
# Fast dependency resolution with upstream fallback
```

#### Enterprise/Air-gapped
```yaml
# *-mirror.yaml configurations
# Complete offline package repositories
```

#### CI/CD Pipelines
```yaml
# docker-mirror.yaml + npm-mirror.yaml + pip-mirror.yaml
# Reliable, fast builds with dependency caching
```

#### Multi-cloud Deployments
```yaml
# Cloud-specific authentication in proxy configs
# AWS ECR + Azure ACR + GCP GCR integration
```

## Environment Variables Reference

```bash
# NPM Authentication
NPM_TOKEN=npm_xxxxxxxxxxxx
GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Maven Authentication  
NEXUS_USERNAME=deploy
NEXUS_PASSWORD=secret
GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Docker Authentication
DOCKER_USERNAME=username
DOCKER_PASSWORD=password
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...
GCR_TOKEN=ya29...

# Python Authentication
DEVPI_USERNAME=user
DEVPI_PASSWORD=pass
CODEARTIFACT_TOKEN=eyJ...
AZURE_PAT=...
```
