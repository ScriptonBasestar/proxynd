# Changelog

All notable changes to ProxyND will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
