# Proxy Type Configuration Examples

This directory contains configuration examples for each supported package manager proxy type.

## Available Proxy Types

### APK (Alpine Linux)
- `apk-proxy.yaml` - Proxy for Alpine Linux package repositories

### APT (Debian/Ubuntu)
- `apt-proxy.yaml` - Standard APT proxy configuration
- `apt-mirror.yaml` - Full APT mirror configuration with multiple repositories

### Docker
- `docker-proxy.yaml` - Docker registry proxy with authentication support

### Maven (Java)
- `maven-proxy.yaml` - Maven Central proxy configuration
- `maven-mirror.yaml` - Maven repository mirror with multiple remotes
- `maven-host.yaml` - Host your own Maven artifacts

### NPM (Node.js)
- `npm-proxy.yaml` - NPM registry proxy with scoped package support

### PIP (Python)
- `pip-proxy.yaml` - PyPI proxy configuration

### YUM (RedHat/CentOS)
- `yum-proxy.yaml` - YUM repository proxy configuration

### Vagrant
- `vagrant-proxy.yaml` - Vagrant box repository proxy

## Usage Example

To use a specific proxy type configuration:

```bash
# For NPM proxy
cp npm-proxy.yaml ../../config.yaml

# Or merge multiple configs
cat npm-proxy.yaml maven-proxy.yaml > ../../config.yaml
```

## Configuration Tips

- Each proxy type can be enabled/disabled independently
- Multiple proxy types can run simultaneously on different paths
- Cache settings can be configured per proxy type
- Authentication can be set up per repository