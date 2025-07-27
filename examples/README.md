# ProxyND Example Configurations

This directory contains example configuration files organized by category:

## Directory Structure

### `/minimal`
Basic, minimal configuration examples to get started quickly.
- `config.minimal.yaml` - Bare minimum configuration
- `config.example.yaml` - Basic example with common settings

### `/proxy-types`
Configuration examples for each supported proxy type:
- `apk-proxy.yaml` - Alpine Linux APK proxy configuration
- `apt-proxy.yaml` - Debian/Ubuntu APT proxy configuration
- `apt-mirror.yaml` - APT mirror configuration
- `docker-proxy.yaml` - Docker registry proxy configuration
- `maven-proxy.yaml` - Maven repository proxy configuration
- `maven-mirror.yaml` - Maven mirror configuration
- `maven-host.yaml` - Maven hosting configuration
- `npm-proxy.yaml` - NPM registry proxy configuration
- `pip-proxy.yaml` - Python PyPI proxy configuration
- `yum-proxy.yaml` - RedHat/CentOS YUM proxy configuration
- `vagrant-proxy.yaml` - Vagrant box proxy configuration

### `/features`
Configuration examples for specific features:
- `cache-s3.yaml` - S3 backend cache configuration
- `oauth2.yaml` - OAuth2 authentication configuration
- `webhook.yaml` - Webhook notifications configuration
- `verification-config.yaml` - Package verification settings
- `performance.yaml` - Performance optimization settings
- `github-mirror.yaml` - GitHub releases mirror configuration

### `/environments`
Environment-specific complete configurations:
- `config.development.yaml` - Development environment settings
- `config.production.yaml` - Production environment settings
- `config.enterprise.yaml` - Enterprise deployment settings

## Quick Start

1. For a minimal setup, copy `minimal/config.minimal.yaml`:
   ```bash
   cp minimal/config.minimal.yaml ../config.yaml
   ```

2. For specific proxy types, copy the relevant file from `proxy-types/`:
   ```bash
   cp proxy-types/npm-proxy.yaml ../npm-proxy.yaml
   ```

3. For production deployment, start with `environments/config.production.yaml`:
   ```bash
   cp environments/config.production.yaml ../config.yaml
   ```

## Configuration Priority

When ProxyND starts, it looks for configuration in this order:
1. Command line flag: `--config`
2. Current directory: `./config.yaml`
3. User config: `~/.config/proxynd/config.yaml`
4. System config: `/etc/proxynd/config.yaml`

## Environment Variables

Remember to set required environment variables:
- `CONFIG_DIR`: Configuration directory path
- `STORAGE_DIR`: Cache storage directory path

See the main README.md for complete documentation.