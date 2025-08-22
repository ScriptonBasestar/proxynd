# ProxyND Configuration Reference

This document provides a comprehensive reference for all ProxyND configuration options.

## Table of Contents

1. [Configuration Files](#configuration-files)
2. [Environment Variables](#environment-variables)
3. [Global Configuration](#global-configuration)
4. [Proxy-Specific Configuration](#proxy-specific-configuration)
5. [Advanced Configuration](#advanced-configuration)
6. [Configuration Validation](#configuration-validation)

## Configuration Files

ProxyND uses YAML configuration files located in the `CONFIG_DIR` directory. The configuration loading order is:

1. Default values (built into the application)
2. Configuration files (YAML)
3. Environment variables
4. Command-line flags

### File Structure

```
CONFIG_DIR/
├── global.yaml           # Global settings
├── apt-proxy.yaml        # APT proxy configuration
├── maven-proxy.yaml      # Maven proxy configuration
├── npm-proxy.yaml        # NPM proxy configuration
├── docker-proxy.yaml     # Docker proxy configuration
├── webhook.yaml          # Webhook notifications
├── oauth2.yaml           # OAuth2 authentication
└── verification.yaml     # Package verification
```

## Environment Variables

### Required Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `CONFIG_DIR` | Configuration directory path | - | `/etc/proxynd` |
| `STORAGE_DIR` | Storage directory for cache/data | - | `/var/lib/proxynd` |

### Optional Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SERVER_PORT` | HTTP server port | `8080` | `3000` |
| `LOG_LEVEL` | Logging level | `info` | `debug` |
| `CACHE_DIR` | Cache directory | `{STORAGE_DIR}/cache` | `/var/cache/proxynd` |
| `MAX_CACHE_SIZE` | Maximum cache size in bytes | `0` (unlimited) | `107374182400` |

## Global Configuration

The `global.yaml` file contains settings that apply to all proxy types.

### Storage Configuration

```yaml
# Base storage directory (overrides STORAGE_DIR env var)
storage_dir: /var/lib/proxynd

# Configuration directory (overrides CONFIG_DIR env var)
config_dir: /etc/proxynd

# Cache directory
cache_dir: /var/lib/proxynd/cache

# Maximum cache size in bytes (0 = unlimited)
max_cache_size: 107374182400  # 100GB
```

### Cache Configuration

```yaml
cache:
  # Global default TTL in seconds
  ttl: 3600  # 1 hour

  # HTTP cache header support
  use_cache_headers: true
  max_cache_header_ttl: 86400  # 24 hours
  min_cache_header_ttl: 300    # 5 minutes

  # Stale-while-revalidate
  stale_while_revalidate: true
  stale_max_age: 3600

  # Package-type specific TTLs
  package_ttls:
    apt: 3600
    npm: 1800
    maven: 5400
    docker: 7200

  # Pattern-based TTL overrides
  pattern_ttls:
    "*-SNAPSHOT": 300
    "*-dev": 600
    "*-alpha": 900

  # Metadata file TTLs
  metadata_ttls:
    "Packages.gz": 600
    "repomd.xml": 300
    "package.json": 300
```

### Authentication Configuration

```yaml
authentication:
  # Basic authentication
  basic_auth:
    realm: "ProxyND Restricted Area"
    users:
      admin: "password"  # Use bcrypt in production

  # OAuth2 (see oauth2.yaml)
  oauth2:
    enabled: false
    config_file: oauth2.yaml
```

### Server Configuration

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  timeout: 300  # seconds
  max_body_size: "1GB"
```

### Logging Configuration

```yaml
logging:
  level: "info"  # debug, info, warn, error, fatal
  format: "json" # json, text
  output: "stdout"  # stdout, stderr, file

  file:
    path: "/var/log/proxynd/proxynd.log"
    max_size: 100     # MB
    max_backups: 10
    max_age: 30       # days
    compress: true
```

## Proxy-Specific Configuration

### APT Proxy (`apt-proxy.yaml`)

```yaml
# Storage path relative to storage_dir
path: "apt"

# Enable caching
use_cache: true

# User-specific caching
user_cache: false

# Upstream mirrors by distribution
proxies:
  ubuntu_jammy_amd64:
    - name: "Ubuntu Official"
      url: "http://archive.ubuntu.com/ubuntu"
    - name: "Ubuntu Mirror US"
      url: "http://us.archive.ubuntu.com/ubuntu"
```

### Maven Proxy (`maven-proxy.yaml`)

```yaml
path: "maven"
use_cache: true

repositories:
  central:
    - name: "Maven Central"
      url: "https://repo1.maven.org/maven2"
    - name: "Maven Central Mirror"
      url: "https://repo.maven.apache.org/maven2"

# Snapshot handling
snapshots:
  enabled: true
  update_policy: "always"  # always, daily, interval:X
  checksum_policy: "warn"  # fail, warn, ignore
```

### NPM Proxy (`npm-proxy.yaml`)

```yaml
path: "npm"
use_cache: true

registries:
  - name: "NPM Official"
    url: "https://registry.npmjs.org"
  - name: "NPM Mirror"
    url: "https://registry.npmmirror.com"

# Scoped packages
scopes:
  "@company":
    registry: "https://npm.company.com"
    auth_token: "${NPM_TOKEN}"
```

### Docker Proxy (`docker-proxy.yaml`)

```yaml
path: "docker"
use_cache: true

registries:
  docker.io:
    - name: "Docker Hub"
      url: "https://registry-1.docker.io"

  gcr.io:
    - name: "Google Container Registry"
      url: "https://gcr.io"

# Authentication
auth:
  docker.io:
    username: "${DOCKER_USERNAME}"
    password: "${DOCKER_PASSWORD}"
```

## Advanced Configuration

### Webhook Configuration (`webhook.yaml`)

```yaml
enabled: true

endpoints:
  - name: "slack-alerts"
    url: "https://hooks.slack.com/services/..."
    format: "slack"
    event_types:
      - "auth.failure"
      - "security.*"
    filters:
      min_level: "WARNING"

rate_limit:
  enabled: true
  max_per_minute: 100

retry:
  enabled: true
  max_attempts: 3
```

### OAuth2 Configuration (`oauth2.yaml`)

```yaml
enabled: true

providers:
  google:
    client_id: "${OAUTH2_CLIENT_ID}"
    client_secret: "${OAUTH2_CLIENT_SECRET}"
    redirect_url: "https://proxynd.example.com/auth/callback"
    scopes:
      - "openid"
      - "email"
      - "profile"
```

### Package Verification (`verification.yaml`)

```yaml
enabled: true

# GPG verification
gpg:
  enabled: true
  keyring_path: "/etc/proxynd/gpg"
  auto_import: true

# Checksum verification
checksums:
  enabled: true
  algorithms:
    - "sha256"
    - "sha512"
  require_all: false
```

## Configuration Validation

ProxyND validates all configuration at startup. Validation includes:

### Field Validation

- Required fields must be present
- Field types must match (string, int, bool, etc.)
- Field constraints (min/max values, regex patterns)

### Example Validation Tags

```go
type Config struct {
    Path     string `yaml:"path" validate:"required,min=1"`
    URL      string `yaml:"url" validate:"required,url"`
    Port     int    `yaml:"port" validate:"min=1,max=65535"`
    Timeout  int    `yaml:"timeout" validate:"min=0,max=3600"`
}
```

### Common Validation Rules

| Rule | Description | Example |
|------|-------------|---------|
| `required` | Field must be present | `validate:"required"` |
| `min=X` | Minimum value/length | `validate:"min=1"` |
| `max=X` | Maximum value/length | `validate:"max=100"` |
| `url` | Valid URL format | `validate:"url"` |
| `email` | Valid email format | `validate:"email"` |
| `oneof` | One of specified values | `validate:"oneof=apt npm maven"` |

## Configuration Hot Reload

ProxyND supports hot reloading for certain configuration changes:

### Reloadable Settings

- Cache TTLs
- Authentication users
- Webhook endpoints
- Rate limits

### Non-Reloadable Settings

- Server port
- Storage directories
- TLS configuration

### Triggering Reload

Send SIGHUP signal to the process:
```bash
kill -HUP <pid>
```

Or use the admin API:
```bash
curl -X POST http://localhost:8080/admin/reload
```

## Best Practices

1. **Use Environment Variables for Secrets**
   ```yaml
   password: "${ADMIN_PASSWORD}"
   ```

2. **Set Appropriate TTLs**
   - Metadata: 5-15 minutes
   - Stable packages: 1-24 hours
   - Development versions: 5-30 minutes

3. **Configure Multiple Mirrors**
   - Provides failover capability
   - Improves availability
   - Enables geographic distribution

4. **Enable Authentication**
   - Use strong passwords
   - Consider OAuth2 for enterprise
   - Enable rate limiting

5. **Monitor Cache Size**
   - Set max_cache_size appropriately
   - Monitor disk usage
   - Configure cache eviction

6. **Use Webhooks for Monitoring**
   - Configure alerts for critical events
   - Monitor authentication failures
   - Track cache performance

## Troubleshooting

### Configuration Not Loading

1. Check file permissions
2. Validate YAML syntax
3. Check environment variables
4. Review startup logs

### Validation Errors

1. Check required fields
2. Verify field types
3. Check value constraints
4. Review validation rules

### Performance Issues

1. Adjust cache TTLs
2. Configure connection pools
3. Enable request timeouts
4. Monitor resource usage
