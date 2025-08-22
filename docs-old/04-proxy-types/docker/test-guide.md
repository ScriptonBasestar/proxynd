# Docker Proxy Testing Guide

## Overview

This guide provides testing strategies and procedures for the Docker registry proxy functionality in ProxyND.

## Quick Test Commands

### Basic Connectivity

```bash
# Test Docker proxy endpoint
curl -I http://localhost:8081/proxy/docker/v2/

# Test manifest download
curl "http://localhost:8081/proxy/docker/v2/library/nginx/manifests/latest"

# Test blob download (layer)
curl -I "http://localhost:8081/proxy/docker/v2/library/nginx/blobs/sha256:abc123..."
```

### Docker Client Configuration

Configure Docker to use ProxyND proxy:

```bash
# Configure Docker daemon (requires restart)
# /etc/docker/daemon.json
{
  "registry-mirrors": ["http://localhost:8081/proxy/docker"]
}

# Or use insecure registry for testing
{
  "insecure-registries": ["localhost:8081"],
  "registry-mirrors": ["http://localhost:8081/proxy/docker"]
}

# Restart Docker
sudo systemctl restart docker
```

### Test Scenarios

1. **Image Pull**
   ```bash
   docker pull nginx:latest
   docker pull alpine:3.18
   docker pull ubuntu:22.04
   ```

2. **Multi-arch Images**
   ```bash
   docker pull --platform linux/amd64 nginx:latest
   docker pull --platform linux/arm64 nginx:latest
   ```

3. **Cache Verification**
   ```bash
   # Check cache directory
   ls -la $STORAGE_DIR/docker/

   # Verify cache hit performance
   time docker pull nginx:latest  # First time
   docker rmi nginx:latest
   time docker pull nginx:latest  # Should be faster
   ```

## Registry v2 API Tests

```bash
# Test catalog endpoint
curl "http://localhost:8081/proxy/docker/v2/_catalog"

# Test tags list
curl "http://localhost:8081/proxy/docker/v2/library/nginx/tags/list"

# Test manifest
curl -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
     "http://localhost:8081/proxy/docker/v2/library/nginx/manifests/latest"
```

## Integration Tests

For detailed integration testing, see [Docker E2E Testing Guide](../../05-development/testing/e2e-docker.md).

## Common Issues

- **Authentication**: Test with private registries requiring auth
- **Large images**: Test with multi-GB images like ML frameworks
- **Layer deduplication**: Test with images sharing common layers

## Health Monitoring

```bash
# Check Docker proxy health
curl http://localhost:8081/health/proxy/docker | jq .

# Monitor metrics
curl http://localhost:8081/metrics | grep docker
```

## Security Considerations

- Test image signature verification
- Verify layer integrity checks
- Test with malicious/corrupted images
