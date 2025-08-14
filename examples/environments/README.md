# Environment-Specific Configuration Examples

This directory contains complete configuration examples for different deployment environments.

## Available Configurations

### `config.development.yaml`
Development environment configuration with:
- Debug logging enabled
- Hot reload enabled
- Relaxed security settings
- Local file storage
- All proxy types enabled for testing

### `config.production.yaml`
Production-ready configuration with:
- Optimized performance settings
- Security hardening
- Structured JSON logging
- S3 cache backend
- Monitoring and metrics enabled
- Rate limiting

### `config.enterprise.yaml`
Enterprise deployment configuration featuring:
- High availability settings
- Advanced authentication (OAuth2, LDAP)
- Compliance and audit logging
- Multi-region cache replication
- Advanced security policies
- Integration with enterprise monitoring

## Usage

```bash
# For development
cp config.development.yaml ../../config.yaml
export PROXYND_ENV=development

# For production
cp config.production.yaml ../../config.yaml
export PROXYND_ENV=production

# For enterprise
cp config.enterprise.yaml ../../config.yaml
export PROXYND_ENV=enterprise
```

## Environment Variables

Each environment can override settings using environment variables:

```bash
# Development
export LOG_LEVEL=debug
export HOT_RELOAD=true

# Production
export LOG_LEVEL=info
export ENABLE_METRICS=true
export CACHE_BACKEND=s3

# Enterprise
export OAUTH2_ENABLED=true
export AUDIT_LOG_ENABLED=true
export HA_MODE=true
```

## Deployment Checklist

### Development
- [ ] Local storage directory created
- [ ] Debug logging enabled
- [ ] Test credentials configured

### Production
- [ ] SSL certificates configured
- [ ] S3 bucket created and accessible
- [ ] Monitoring endpoints verified
- [ ] Backup strategy in place

### Enterprise
- [ ] OAuth2 providers configured
- [ ] Audit log destination configured
- [ ] HA cluster setup completed
- [ ] Disaster recovery plan tested
