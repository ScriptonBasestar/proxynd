# ADR-005: Separate Configuration from Code

## Status

Accepted

## Context

ProxyND requires extensive configuration for:
- Multiple proxy types (APT, Maven, NPM, etc.)
- Cache settings and TTLs
- Authentication mechanisms
- Storage paths
- Upstream server URLs

Current issues with configuration:
- Configuration files loaded directly in handlers
- No validation at startup
- Environment variables mixed with file-based config
- No clear precedence rules
- Configuration changes require code understanding
- Hard to test different configurations

Requirements:
- Support multiple configuration sources (files, env vars)
- Validate configuration at startup
- Hot reload capability for certain settings
- Clear separation between code and configuration
- Type-safe configuration access

## Decision

Implement a layered configuration system with clear separation from application code:

1. **Configuration Structure**:
   - Strongly typed configuration structs with validation tags
   - Separate config files per proxy type
   - Global configuration for cross-cutting concerns
   - Environment variables for deployment-specific settings

2. **Configuration Loading**:
   - Central ConfigService responsible for all configuration
   - Load order: defaults → files → environment → flags
   - Validation at startup using struct tags
   - Fail fast on invalid configuration

3. **File Organization**:
   ```
   configs/
   ├── global.yaml          # Global settings
   ├── apt-proxy.yaml       # APT-specific settings
   ├── maven-proxy.yaml     # Maven-specific settings
   └── ...
   ```

4. **Implementation**:
   - Use Viper for configuration management
   - Implement hot reload for safe configuration changes
   - Provide typed accessors for all configuration

## Consequences

### Positive

- Clear separation between configuration and code
- Type-safe configuration access
- Validation prevents runtime errors
- Easy to manage different environments
- Hot reload reduces downtime
- Better security through environment variables for secrets
- Easier to document configuration options

### Negative

- Additional complexity in configuration loading
- Need to maintain validation rules
- Hot reload adds complexity
- Potential for configuration drift between environments

### Neutral

- Requires migration of existing configuration
- Team needs to understand configuration precedence
- Need to document all configuration options

## Alternatives Considered

1. **Hardcoded configuration**
   - Pros: Simple, no parsing needed
   - Cons: Requires recompilation, not flexible
   - Rejected for obvious reasons

2. **Single configuration file**
   - Pros: Simple to manage
   - Cons: Large file, mixing concerns
   - Rejected for maintainability

3. **Database configuration**
   - Pros: Dynamic updates, centralized
   - Cons: Additional dependency, complexity
   - Rejected as overkill for our needs

4. **Consul/etcd for configuration**
   - Pros: Distributed configuration, dynamic updates
   - Cons: Additional infrastructure, complexity
   - Rejected for operational simplicity

## Configuration Validation Rules

- Required fields must be present
- URLs must be valid
- Paths must exist or be creatable
- TTL values must be positive
- Port numbers must be valid
- Authentication credentials meet requirements

## References

- [The Twelve-Factor App - Config](https://12factor.net/config)
- [Viper Configuration Framework](https://github.com/spf13/viper)
- Configuration validation best practices
- Current implementation in `configs/` and `internal/services/config/`
