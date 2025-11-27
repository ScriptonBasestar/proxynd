# Changelog

All notable changes to ProxyND will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **AWS SSM Integration**: AWS Systems Manager Parameter Store support with intelligent caching
  - Complete SSM client with GetParameter, GetParameters, GetParametersByPath operations
  - TTL-based caching layer (default 5 minutes) to reduce API calls and costs
  - Thread-safe cache implementation with auto-cleanup goroutine
  - IAM role support for secure credential management
  - Comprehensive documentation (425 lines) with security best practices
  - Example configuration files and AWS CLI commands
  - Full test suite with integration test scaffolding and benchmarks
- **Hexagonal Architecture Migration**: Migrated handlers to clean architecture pattern
  - Refactored SearchHandler to struct-based pattern with dependency injection
  - Added backward-compatible function wrappers for smooth transition
  - Established pattern for future handler migrations
- **Enhanced Verification Configuration**: Configuration-driven package verification system
  - Added VerificationConfig and PackageVerificationConfig types
  - Externalized verification policies to YAML configuration
  - Package-type specific settings for all 7 package managers
  - Configurable strict mode, block on failure, and alert on failure options
- **APK Signature Verification**: Improved APK package signature verification flow
  - Implemented temporary file management for in-memory package verification
  - Added singleton pattern for verifier initialization
  - Proper cleanup with defer pattern
- **Memory Limiting**: Implemented comprehensive memory limiting across all deployment methods
  - Added `ulimit -v 268435456` (256MiB) to all Dockerfiles for OS-level memory enforcement
  - Added ulimits configuration to docker-compose.yml (memlock, nofile, nproc)
  - Added `LimitAS`, `MemoryMax`, and `MemoryHigh` directives to systemd service
  - Enabled resource limits in Helm chart values.yaml (512Mi limit, 128Mi request)

### Fixed
- **Docker Configuration Consistency**: Fixed missing `GOMEMLIMIT=256MiB` in production Dockerfiles
  - Added GOMEMLIMIT to Dockerfile.multiarch (used in releases)
  - Added GOMEMLIMIT to Dockerfile.goreleaser
  - Ensures consistent memory behavior across all build variants
- **Distroless Image Support**: Added BusyBox shell to Dockerfile.multiarch to enable ulimit support in distroless images

### Changed
- **Hexagonal Architecture Migration Phase 2c**: Enabled new architecture as default
  - Verified and documented that new architecture is default behavior
  - Updated all migration comments to reflect "default mode" status
  - Enhanced migration status API with accurate progress tracking
  - Retained feature flag for emergency rollback capability
  - Migration progress: 75% complete (Phase 2 fully complete)
  - All unit tests passing with new architecture
- **Hexagonal Architecture Migration Phase 2b**: Completed router layer migration to new architecture
  - Updated all 7 router files to use new hexagonal architecture imports
  - Removed all `handlers-legacy` and `middleware-legacy` imports from router layer
  - Created backward-compatible function wrappers for smooth transition
  - Updated jwt_handler.go to use new middleware package
  - Migration progress: 55% → 75% complete
  - All routers: pool, auth, base, container_proxy, proxy, proxy_v3, unified_v1
- **Hexagonal Architecture Migration Phase 2a**: Implemented new middleware setup in hexagonal architecture
  - Migrated `setupNewMiddlewares()` from legacy fallback to proper new architecture
  - Integrated 5 core middleware: ErrorRecovery, ErrorHandler, AccessLog, SecurityHeaders, EnhancedRateLimiter
  - Feature flag system enables safe testing and instant rollback
  - Migration progress: 40% → 55% complete
- **Documentation Cleanup**: Resolved obsolete TODO markers and updated issue templates
  - Converted 4 legacy TODOs to NOTE references pointing to actual implementations
  - Updated 3 issue templates (auth, metrics, proxy services) to RESOLVED status
  - Clarified that all "missing" features are actually fully implemented in new architecture
- **Dockerfile CMD**: Updated all Dockerfiles to use shell wrapper with ulimit for memory enforcement
  - Main Dockerfile: `CMD ["/bin/sh", "-c", "ulimit -v 268435456 && exec /app/proxynd"]`
  - Dockerfile.multiarch: `ENTRYPOINT ["/bin/sh", "-c", "ulimit -v 268435456 && exec /app/proxynd \"$@\"", "--"]`
  - Dockerfile.goreleaser: `ENTRYPOINT ["/bin/sh", "-c", "ulimit -v 268435456 && exec /usr/local/bin/proxynd \"$@\"", "--"]`

### Technical Details

#### Memory Limiting Strategy (Defense-in-Depth)
| Layer | Mechanism | Limit | Purpose |
|-------|-----------|-------|---------|
| Go Runtime | GOMEMLIMIT | 256MiB | Soft limit for GC tuning |
| OS Process | ulimit -v | 256MiB | Hard virtual memory limit |
| Container | docker ulimits | 256MiB | Docker-level enforcement |
| cgroup | memory.max | 512MiB | Kernel-level hard limit |
| Kubernetes | resources.limits.memory | 512Mi | Cluster-level enforcement |

#### Files Modified
- Dockerfile
- Dockerfile.multiarch
- Dockerfile.goreleaser
- docker-compose.yml
- deployments/systemd/proxynd.service
- deployments/helm/values.yaml

## [Previous Releases]

See [GitHub Releases](https://github.com/scriptonbasestar/proxynd/releases) for earlier versions.
