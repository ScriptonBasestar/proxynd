# NPM Proxy Testing Guide

## Overview

This guide provides testing strategies and procedures for the NPM proxy functionality in ProxyND.

## Quick Test Commands

### Basic Connectivity

```bash
# Test NPM proxy endpoint
curl -I http://localhost:8081/proxy/npm/

# Test package metadata
curl "http://localhost:8081/proxy/npm/express" | jq .

# Test package download
curl -O "http://localhost:8081/proxy/npm/express/-/express-4.18.0.tgz"
```

### NPM Client Configuration

Configure npm to use ProxyND proxy:

```bash
# Set registry
npm config set registry http://localhost:8081/proxy/npm

# Or use .npmrc file
echo "registry=http://localhost:8081/proxy/npm" > .npmrc

# Test installation
npm install express
npm install lodash@4.17.21
```

### Test Scenarios

1. **Package Installation**
   ```bash
   npm install express
   npm install @types/node
   npm install scoped-package@latest
   ```

2. **Package Information**
   ```bash
   npm view express
   npm info lodash versions --json
   ```

3. **Cache Verification**
   ```bash
   # Check cache directory
   ls -la $STORAGE_DIR/npm/

   # Verify cache hit performance
   time npm install express  # First time
   npm uninstall express
   time npm install express  # Should be faster
   ```

## Integration Tests

For detailed integration testing, see [NPM E2E Testing Guide](../../05-development/testing/e2e-npm.md).

## Common Issues

- **Scoped packages**: Test with `@types/node` or `@angular/core`
- **Large packages**: Test with packages like `typescript` or `webpack`
- **Version ranges**: Test with `express@^4.0.0` or `lodash@~4.17.0`

## Health Monitoring

```bash
# Check NPM proxy health
curl http://localhost:8081/health/proxy/npm | jq .

# Monitor metrics
curl http://localhost:8081/metrics | grep npm
```
