---
id: PLAN-001
children: [TASK-001, TASK-002, TASK-003, TASK-004, TASK-005, TASK-006, TASK-007, TASK-008, TASK-009, TASK-010, TASK-011, TASK-012]
progress: 8
total-tasks: 12
completed-tasks: 1
---

# Future Package Manager Support Plan

> **Created**: 2025-01-16
> **Status**: Planning
> **Current Support**: 9 package managers (Maven, NPM, APT, Docker, PyPI, YUM, APK, Ansible, OLM)

---

## Overview

This document outlines potential package managers to add to ProxyND based on:
- Developer community usage
- Enterprise demand
- Implementation complexity
- Ecosystem growth trends

## Children
- [x] [TASK-001](../done/task-nuget-support.md) - Add NuGet proxy support
- [ ] [TASK-002](../todo/task-cargo-support.md) - Add Cargo proxy support
- [ ] [TASK-003](../todo/task-go-modules-support.md) - Add Go Modules proxy support
- [ ] [TASK-004](../todo/task-rubygems-support.md) - Add RubyGems proxy support
- [ ] [TASK-005](../todo/task-helm-support.md) - Add Helm chart proxy support
- [ ] [TASK-006](../todo/task-composer-support.md) - Add Composer proxy support
- [ ] [TASK-007](../todo/task-gradle-plugin-portal-support.md) - Add Gradle Plugin Portal proxy support
- [ ] [TASK-008](../todo/task-cocoapods-support.md) - Add CocoaPods proxy support
- [ ] [TASK-009](../todo/task-swift-package-manager-support.md) - Add Swift Package Manager proxy support
- [ ] [TASK-010](../todo/task-terraform-registry-support.md) - Add Terraform Registry proxy support
- [ ] [TASK-011](../todo/task-pub-support.md) - Add Pub.dev proxy support
- [ ] [TASK-012](../todo/task-cpan-support.md) - Add CPAN proxy support

---

## 🔥 High Priority

### 1. NuGet (.NET/C#)

**Target Users**: Microsoft .NET ecosystem (C#, F#, VB.NET)

**Why Important**:
- .NET Core/5+ popularity increasing
- Strong enterprise adoption
- Large package ecosystem

**Technical Details**:
- **API**: RESTful API, JSON metadata
- **Protocol**: NuGet V3 API
- **Reference**: https://docs.microsoft.com/nuget/api/overview
- **Difficulty**: Medium (similar to Maven)

**Implementation Notes**:
- Support both V2 and V3 protocols
- Package restore and push operations
- Symbol server support for debugging

---

### 2. Cargo (Rust)

**Target Users**: Rust developers

**Why Important**:
- Rust adoption growing rapidly
- Systems programming, WebAssembly popularity
- Modern tooling and package management

**Technical Details**:
- **API**: crates.io API
- **Protocol**: HTTP-based registry
- **Reference**: https://doc.rust-lang.org/cargo/reference/registries.html
- **Difficulty**: Medium

**Implementation Notes**:
- Git-based index (sparse index support)
- Crate download and metadata
- Authentication for private registries

---

### 3. Go Modules (Go)

**Target Users**: Go developers

**Why Important**:
- ProxyND itself is written in Go
- Can reference Athens proxy implementation
- Module proxy protocol well-documented

**Technical Details**:
- **API**: Module proxy protocol
- **Protocol**: GOPROXY environment variable
- **Reference**: https://go.dev/ref/mod#module-proxy
- **Difficulty**: Medium (advantage: already using Go)

**Implementation Notes**:
- Support `go get`, `go mod download`
- Module version list, info, and zip
- Checksum database integration

---

### 4. RubyGems (Ruby)

**Target Users**: Ruby, Ruby on Rails developers

**Why Important**:
- Active community
- Widely used in web development
- Rails framework still popular

**Technical Details**:
- **API**: Simple API (similar to PyPI)
- **Protocol**: HTTP-based
- **Reference**: https://guides.rubygems.org/rubygems-org-api/
- **Difficulty**: Low

**Implementation Notes**:
- Gem download and metadata
- Dependency resolution
- Platform-specific gems

---

### 5. Helm (Kubernetes)

**Target Users**: Kubernetes operators, DevOps engineers

**Why Important**:
- Cloud-native deployment standard
- Critical for Kubernetes ecosystem
- Growing enterprise adoption

**Technical Details**:
- **API**: Chart repository HTTP API
- **Protocol**: Simple HTTP index
- **Reference**: https://helm.sh/docs/topics/chart_repository/
- **Difficulty**: Low (HTTP-based)

**Implementation Notes**:
- Chart index.yaml caching
- Chart package (.tgz) storage
- OCI registry support (Helm 3)

---

## 🌟 Medium Priority

### 6. Composer (PHP)

**Target Users**: PHP developers (Laravel, Symfony)

**Why Important**:
- PHP still widely used in web development
- Large package ecosystem
- Active community

**Technical Details**:
- **API**: Packagist API
- **Protocol**: Composer repository protocol
- **Reference**: https://getcomposer.org/doc/05-repositories.md
- **Difficulty**: Low

---

### 7. Gradle Plugin Portal (JVM)

**Target Users**: Gradle users

**Why Important**:
- Major JVM build tool alongside Maven
- Growing adoption in Android development
- Modern build system

**Technical Details**:
- **API**: Plugin portal API
- **Protocol**: HTTP-based
- **Reference**: https://plugins.gradle.org/docs/api
- **Difficulty**: Low

---

### 8. CocoaPods (iOS/macOS)

**Target Users**: iOS/macOS developers

**Why Important**:
- Standard dependency manager for Apple platforms
- Large library ecosystem
- Enterprise iOS development

**Technical Details**:
- **API**: Specs repository (Git-based)
- **Protocol**: Git + CDN
- **Reference**: https://guides.cocoapods.org/making/specs-and-specs-repo.html
- **Difficulty**: Medium

---

### 9. Swift Package Manager (Swift)

**Target Users**: Swift developers

**Why Important**:
- Apple's official package manager
- Server-side Swift growth
- Cross-platform Swift development

**Technical Details**:
- **API**: Git-based
- **Protocol**: Swift Package Manager manifest
- **Reference**: https://swift.org/package-manager/
- **Difficulty**: Medium

---

## 📦 Specialized Use Cases

### 10. Terraform Registry (IaC)

**Target Users**: DevOps engineers, SREs

**Why Important**:
- Infrastructure as Code standard
- Very popular in cloud environments
- Enterprise infrastructure management

**Technical Details**:
- **API**: Registry API
- **Protocol**: Registry protocol v1
- **Reference**: https://www.terraform.io/docs/registry/api.html
- **Difficulty**: Medium

---

### 11. Pub (Dart/Flutter)

**Target Users**: Flutter mobile developers

**Why Important**:
- Cross-platform mobile development growth
- Google backing
- Growing ecosystem

**Technical Details**:
- **API**: pub.dev API
- **Protocol**: HTTP-based
- **Reference**: https://pub.dev/help/api
- **Difficulty**: Low

---

### 12. CPAN (Perl)

**Target Users**: Perl developers, legacy systems

**Why Important**:
- Still used in finance, bioinformatics
- Large legacy codebase
- Specialized use cases

**Technical Details**:
- **API**: CPAN mirror protocol
- **Protocol**: CPAN mirror structure
- **Reference**: https://www.cpan.org/misc/cpan-faq.html#How_mirror_CPAN
- **Difficulty**: High (complex structure)

---

## Implementation Priority Recommendation

### Phase 1: Immediate (Q1 2025)
1. **NuGet** - High enterprise demand, .NET ecosystem growth
2. **Cargo** - Rust popularity surge, modern language
3. **Go Modules** - Already using Go, reference Athens proxy

### Phase 2: Next Quarter (Q2 2025)
4. **Helm** - Kubernetes ecosystem essential
5. **RubyGems** - Stable user base
6. **Composer** - PHP ecosystem still large

### Phase 3: Future (Q3-Q4 2025)
7. **Terraform Registry** - DevOps tooling
8. **Pub** - Flutter mobile development
9. **Swift Package Manager** - Apple ecosystem
10. **Gradle Plugin Portal** - JVM ecosystem

### Phase 4: Specialized (On Demand)
11. **CocoaPods** - If iOS development demand
12. **CPAN** - If legacy system support needed

---

## Technical Implementation Notes

### Hexagonal Architecture Pattern

All new package managers should follow the existing pattern:

```
internal/
├── domain/{pm}/         # Pure business logic (ZERO external deps)
│   ├── service.go       # Domain service
│   ├── types.go         # Domain entities
│   └── errors.go        # Domain errors
├── ports/               # Interface contracts
│   └── pm.go            # Add new PM type constant
└── adapters/pm/{pm}/    # Implementation
    ├── adapter.go       # Port implementation
    ├── client.go        # Upstream HTTP client
    └── adapter_test.go  # Tests
```

### Common Implementation Steps

1. **Define Port Interface** (`internal/ports/pm.go`)
   ```go
   const (
       PMTypeNewPM PackageManagerType = "newpm"
   )
   ```

2. **Create Domain Logic** (`internal/domain/newpm/`)
   - Pure business logic
   - Zero external dependencies
   - Domain types and errors

3. **Implement Adapter** (`internal/adapters/pm/newpm/`)
   - HTTP client for upstream
   - Caching integration
   - Error handling

4. **Add Configuration** (`examples/config.yaml`)
   ```yaml
   proxies:
     - type: newpm
       enabled: true
       upstream: https://registry.example.com
   ```

5. **Write Tests**
   - Unit tests: `internal/domain/newpm/service_test.go`
   - Contract tests: `tests/contract/newpm_test.go`
   - Integration tests: `tests/integration/newpm_test.go`

6. **Update Documentation**
   - README.md
   - docs/30-proxy-types/newpm/
   - API endpoints documentation

---

## Success Metrics

For each new package manager implementation:

- ✅ **Test Coverage**: >90% unit tests, >70% integration tests
- ✅ **Performance**: P95 response time <500ms
- ✅ **Cache Hit Rate**: >80% for repeated requests
- ✅ **Documentation**: Complete API docs and usage examples
- ✅ **Compatibility**: Works with standard package manager clients

---

## References

- [Athens Proxy](https://github.com/gomods/athens) - Go module proxy reference
- [Verdaccio](https://verdaccio.org/) - NPM proxy reference
- [Nexus Repository](https://help.sonatype.com/repomanager3) - Multi-format repository manager
- [Artifactory](https://jfrog.com/artifactory/) - Universal artifact repository

---

## Notes

- This is a living document - priorities may change based on user demand
- Implementation complexity estimates are approximate
- Consider community contributions for specialized package managers
- Each implementation should maintain hexagonal architecture principles
