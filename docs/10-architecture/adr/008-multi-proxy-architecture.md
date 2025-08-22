# ADR-008: Multi-Proxy Architecture

## Status

Accepted

## Context

ProxyND needs to support multiple package manager types (APT, Maven, NPM, Docker, etc.), each with unique:
- Protocol requirements
- URL patterns
- Authentication methods
- Metadata formats
- Caching strategies

Current challenges:
- Each proxy type implemented separately
- Code duplication across proxy implementations
- Difficult to add new proxy types
- Inconsistent behavior across proxy types
- Hard to maintain feature parity

Requirements:
- Extensible architecture for new proxy types
- Consistent behavior where appropriate
- Proxy-specific customization where needed
- Shared infrastructure (caching, auth, logging)
- Clear boundaries between proxy implementations

## Decision

Implement a multi-proxy architecture with the following design:

1. **Common Proxy Interface**:
   ```go
   type ProxyHandler interface {
       Name() string
       CanHandle(path string) bool
       Handle(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)
       ValidateConfig(config interface{}) error
   }
   ```

2. **Shared Infrastructure**:
   - Common caching layer used by all proxies
   - Shared authentication/authorization
   - Unified logging and metrics
   - Common HTTP client with circuit breakers

3. **Proxy-Specific Implementations**:
   - Each proxy type in its own package
   - Custom URL routing per proxy
   - Specific configuration structures
   - Specialized metadata handling

4. **Registration Pattern**:
   ```go
   registry := proxy.NewRegistry()
   registry.Register(apt.NewHandler())
   registry.Register(maven.NewHandler())
   registry.Register(npm.NewHandler())
   ```

5. **URL Routing**:
   - `/apt/*` → APT proxy
   - `/maven/*` → Maven proxy
   - `/npm/*` → NPM proxy
   - Configurable routing patterns

## Consequences

### Positive

- Easy to add new proxy types
- Consistent behavior across proxy types
- Shared infrastructure reduces duplication
- Clear separation of concerns
- Easier to maintain and test
- Feature parity easier to achieve
- Can disable/enable specific proxies

### Negative

- Additional abstraction complexity
- Need to balance shared vs specific functionality
- Interface might limit some proxy-specific features
- Performance overhead from abstraction

### Neutral

- Requires careful interface design
- Need clear documentation for adding proxies
- Must maintain backward compatibility

## Implementation Structure

```
internal/
├── proxy/
│   ├── registry.go      # Proxy registry
│   ├── interfaces.go    # Common interfaces
│   ├── base_handler.go  # Shared functionality
│   └── middleware.go    # Common middleware
├── domain/
│   ├── apt/            # APT-specific logic
│   ├── maven/          # Maven-specific logic
│   ├── npm/            # NPM-specific logic
│   └── docker/         # Docker-specific logic
└── handlers/
    └── proxy/          # HTTP handlers using domain logic
```

## Adding a New Proxy Type

1. Create package in `internal/domain/{type}/`
2. Implement ProxyHandler interface
3. Add configuration structure
4. Register in application startup
5. Add routing configuration
6. Document specific behaviors

## Alternatives Considered

1. **Monolithic proxy implementation**
   - Pros: Simpler initially
   - Cons: Becomes unmaintainable, hard to extend
   - Rejected for maintainability

2. **Microservices per proxy type**
   - Pros: Complete isolation, independent deployment
   - Cons: Operational complexity, resource overhead
   - Rejected for operational simplicity

3. **Plugin architecture**
   - Pros: Dynamic loading, complete isolation
   - Cons: Complex plugin API, versioning issues
   - Rejected for complexity

4. **Code generation approach**
   - Pros: Type-safe, efficient
   - Cons: Complex tooling, inflexible
   - Rejected for development velocity

## Future Considerations

- Support for custom/private package managers
- Protocol adapters (e.g., HTTP to S3)
- Proxy chaining for corporate environments
- Multi-tenant isolation

## References

- [Strategy Pattern](https://refactoring.guru/design-patterns/strategy)
- [Registry Pattern](https://martinfowler.com/eaaCatalog/registry.html)
- Similar projects: Artifactory, Nexus Repository
- Current implementation in `internal/proxy/` and `internal/domain/`
