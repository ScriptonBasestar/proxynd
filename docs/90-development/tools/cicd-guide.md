# CI/CD Pipeline Guide

## Overview

ProxyND uses GitHub Actions for continuous integration and deployment. The pipeline ensures code quality, security, and reliable releases across multiple platforms.

## Workflows

### 1. CI Workflow (`ci.yml`)

**Triggers:** Push to main/develop, Pull requests, Manual dispatch

**Jobs:**
- **Lint**: Code formatting and style checks
- **Test**: Unit tests across Go versions
- **Build**: Cross-platform binary compilation
- **Docker**: Multi-architecture container builds
- **Integration Tests**: Full system testing
- **E2E Tests**: End-to-end scenario validation
- **Security Scan**: Container vulnerability scanning

### 2. Code Quality Workflow (`code-quality.yml`)

**Triggers:** Pull requests, Weekly schedule

**Jobs:**
- **Pre-commit**: Automated code quality checks
- **SonarCloud**: Code quality and coverage analysis
- **CodeQL**: Security vulnerability detection
- **Dependency Review**: License and vulnerability checks
- **Gosec**: Go-specific security scanning
- **Nancy**: Dependency vulnerability scanning
- **Complexity Analysis**: Cyclomatic complexity metrics
- **Coverage Report**: Test coverage visualization

### 3. Release Workflow (`release.yml`)

**Triggers:** Git tags (v*)

**Jobs:**
- **GoReleaser**: Binary releases for all platforms
- **Docker Release**: Multi-arch container publishing
- **Helm Release**: Chart packaging and publishing
- **Release Notes**: Automated changelog generation

## Configuration Files

### `.goreleaser.yaml`
Configures binary builds, Docker images, and release artifacts:
- Cross-platform builds (Linux, macOS, Windows)
- Multiple architectures (amd64, arm64, arm/v7)
- Docker manifest generation
- Homebrew formula generation
- DEB/RPM/APK package creation

### `sonar-project.properties`
SonarCloud analysis configuration:
- Coverage reporting
- Quality gates
- Exclusion patterns
- Issue filtering

### `.github/changelog-config.json`
Automated release notes generation:
- Categorized changes
- Contributor recognition
- Semantic versioning

## Branch Protection

Recommended branch protection rules:

```yaml
main/master:
  - Require PR reviews (1+)
  - Dismiss stale reviews
  - Require status checks:
    - lint
    - test
    - security-scan
    - code-quality
  - Require branches up to date
  - Include administrators

develop:
  - Require PR reviews (1+)
  - Require status checks:
    - lint
    - test
  - Auto-delete head branches
```

## Secrets Configuration

Required repository secrets:

### GitHub Packages (Automatic)
- `GITHUB_TOKEN`: Automatically provided

### Docker Hub (Optional)
- `DOCKERHUB_USERNAME`: Docker Hub username
- `DOCKERHUB_TOKEN`: Docker Hub access token

### Code Quality (Optional)
- `SONAR_TOKEN`: SonarCloud authentication
- `CODECOV_TOKEN`: Codecov integration

## Release Process

### Automated Releases

1. **Create Release Tag**
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

2. **Automated Steps**
   - GoReleaser builds binaries for all platforms
   - Docker images pushed to registries
   - Helm chart packaged and indexed
   - Release notes auto-generated
   - Homebrew formula updated

3. **Release Artifacts**
   - Binary downloads for all platforms
   - Docker images (multi-arch)
   - Helm charts
   - DEB/RPM/APK packages
   - Checksums and signatures

### Manual Release Process

For hotfixes or special releases:

```bash
# Create release branch
git checkout -b release/v1.2.3

# Update version files
# Update CHANGELOG.md

# Create tag
git tag -a v1.2.3 -m "Release v1.2.3"

# Push tag to trigger release
git push origin v1.2.3
```

## Docker Images

### Registries
- GitHub Container Registry: `ghcr.io/<org>/proxynd`
- Docker Hub: `<username>/proxynd` (optional)

### Tags
- `latest`: Latest stable release
- `v1.2.3`: Specific version
- `v1.2`: Minor version (updated on release)
- `v1`: Major version (updated on release)
- `sha-abc123`: Commit-based tags

### Platforms
- `linux/amd64`: Intel/AMD 64-bit
- `linux/arm64`: ARM 64-bit
- `linux/arm/v7`: ARM 32-bit v7

## Quality Gates

### Pull Request Requirements

1. **Code Quality**
   - All tests passing
   - Coverage ≥ 60%
   - No security vulnerabilities
   - Complexity < 15

2. **Reviews**
   - At least 1 approval
   - No unresolved conversations
   - All checks passing

3. **Documentation**
   - Updated README if needed
   - API documentation current
   - CHANGELOG entry added

### Monitoring

- **Build Status**: GitHub Actions dashboard
- **Code Quality**: SonarCloud dashboard
- **Coverage**: Codecov reports
- **Security**: GitHub Security tab

## Local Testing

### Run CI Checks Locally

```bash
# Lint
make lint

# Tests
make test-all

# Security
make security

# Full CI simulation
make ci
```

### Pre-commit Hooks

```bash
# Install
make pre-commit-install

# Run manually
make pre-commit-run
```

## Troubleshooting

### Common Issues

1. **Build Failures**
   - Check Go version compatibility
   - Verify dependencies with `go mod tidy`
   - Review golangci-lint output

2. **Docker Build Issues**
   - Ensure buildx is enabled
   - Check platform compatibility
   - Verify Dockerfile syntax

3. **Release Failures**
   - Ensure tag format is correct (v*)
   - Check GoReleaser configuration
   - Verify GitHub token permissions

### Debug Commands

```bash
# Test GoReleaser locally
goreleaser release --snapshot --clean

# Validate workflow syntax
act --list

# Check Docker build
docker buildx build --platform linux/amd64,linux/arm64 .
```

## Best Practices

1. **Commit Messages**
   - Use conventional commits
   - Include issue references
   - Be descriptive

2. **Pull Requests**
   - Keep PRs focused
   - Update tests
   - Document changes

3. **Releases**
   - Follow semantic versioning
   - Test release candidates
   - Document breaking changes

## Performance Optimization

### CI Speed Improvements

1. **Caching**
   - Go modules cached
   - Docker layers cached
   - Build artifacts cached

2. **Parallelization**
   - Matrix builds for platforms
   - Concurrent job execution
   - Smart test splitting

3. **Conditional Execution**
   - Skip unchanged components
   - PR-specific optimizations
   - Schedule heavy tasks

## Integration Examples

### Status Badge

```markdown
[![CI](https://github.com/<org>/proxynd/actions/workflows/ci.yml/badge.svg)](https://github.com/<org>/proxynd/actions/workflows/ci.yml)
```

### Deployment Webhook

```yaml
- name: Notify Deployment
  uses: 8398a7/action-slack@v3
  with:
    status: ${{ job.status }}
    text: "ProxyND ${{ github.ref }} deployed!"
```

## Security Considerations

1. **Secret Management**
   - Use GitHub Secrets
   - Rotate tokens regularly
   - Limit scope appropriately

2. **Dependency Scanning**
   - Automated vulnerability checks
   - License compliance
   - Supply chain security

3. **Container Security**
   - Minimal base images
   - Non-root containers
   - Regular scanning

## Future Enhancements

- [ ] Kubernetes deployment automation
- [ ] Performance benchmarking in CI
- [ ] Automated rollback mechanisms
- [ ] Multi-region deployment support
- [ ] A/B testing infrastructure
