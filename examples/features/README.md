# Feature-Specific Configuration Examples

This directory contains configuration examples for specific ProxyND features.

## Available Features

### Caching
- `cache-s3.yaml` - Configure S3-compatible storage backend for distributed caching

### Authentication & Security
- `oauth2.yaml` - OAuth2 authentication setup (GitHub, GitLab, Google)
- `verification-config.yaml` - Package signature verification settings

### Monitoring & Notifications
- `webhook.yaml` - Webhook notifications for events (downloads, errors, etc.)

### Performance
- `performance.yaml` - Performance optimization settings including:
  - Connection pooling
  - Request optimization
  - Resource limits
  - Cache strategies

### Special Mirrors
- `github-mirror.yaml` - Mirror GitHub releases for offline access

## Usage Examples

### Enable S3 Cache Backend
```yaml
# Merge with your main config
cache:
  backend: s3
  s3:
    bucket: proxynd-cache
    region: us-east-1
    endpoint: https://s3.amazonaws.com
```

### Enable OAuth2 Authentication
```yaml
# Add to your config
auth:
  oauth2:
    enabled: true
    providers:
      github:
        client_id: your-client-id
        client_secret: your-client-secret
```

## Best Practices

1. **Security**: Always use HTTPS for OAuth2 redirects
2. **Performance**: Start with default settings and tune based on metrics
3. **Webhooks**: Use authentication for webhook endpoints
4. **S3 Cache**: Consider using local S3-compatible storage (MinIO) for testing
