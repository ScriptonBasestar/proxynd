# ADR-004: Use Dependency Injection Container

## Status

Accepted

## Context

ProxyND has grown to have complex dependency relationships between components:
- Handlers depend on services
- Services depend on repositories and other services
- Repositories depend on configuration
- Various components need logging, metrics, and health checks

Currently, dependencies are created ad-hoc throughout the codebase:
- Handlers directly instantiate services
- Services read configuration files directly
- Hard to track dependency lifecycle
- Difficult to test due to hardcoded dependencies
- Global state access makes reasoning about code difficult

This leads to:
- Tight coupling between components
- Difficult unit testing
- Hidden dependencies
- Initialization order problems
- Hard to swap implementations

## Decision

Implement a Dependency Injection (DI) container to manage component lifecycle and dependencies. We chose a manual DI container approach over code generation tools for simplicity and explicitness.

Key decisions:
1. Use constructor injection exclusively (no setter or field injection)
2. Create a central `wire.go` file that wires all dependencies
3. Dependencies are explicitly declared in constructors
4. Avoid global state and singletons
5. Use interfaces for all injected dependencies

Example structure:
```go
// internal/app/wire.go
func InitializeApp(cfg *config.Config) (*App, error) {
    // Create repositories
    cacheRepo := repositories.NewFileCacheRepository(cfg.CacheDir)
    configRepo := repositories.NewYamlConfigRepository(cfg.ConfigDir)

    // Create services
    cacheService := services.NewCacheService(cacheRepo, cfg.Cache)
    proxyService := services.NewProxyService(cacheService, configRepo)

    // Create handlers
    aptHandler := handlers.NewAptHandler(proxyService)

    // Create app
    return &App{
        Config: cfg,
        // ... wired dependencies
    }, nil
}
```

## Consequences

### Positive

- Explicit dependencies make code easier to understand
- Improved testability through dependency injection
- Clear initialization order
- Easy to swap implementations for testing
- No hidden dependencies or global state
- Compile-time dependency checking
- Easier to manage component lifecycle

### Negative

- More boilerplate code for constructors
- Need to maintain wire.go file
- Can lead to large constructor parameter lists
- Manual wiring can be tedious for large applications

### Neutral

- Team needs to understand DI principles
- Requires discipline to maintain proper boundaries
- May need to refactor to introduce interfaces

## Alternatives Considered

1. **Google Wire (code generation)**
   - Pros: Automatic wiring, compile-time safety, less boilerplate
   - Cons: Additional tooling, generated code, learning curve
   - Rejected for simplicity in current project size

2. **Uber Fx (runtime DI)**
   - Pros: Powerful, handles lifecycle, module system
   - Cons: Runtime reflection, complex for our needs
   - Rejected as too heavy for current requirements

3. **No DI (current approach)**
   - Pros: Simple, no framework needed
   - Cons: Testing difficulties, hidden dependencies
   - Rejected due to maintainability issues

4. **Service Locator Pattern**
   - Pros: Simple to implement
   - Cons: Hidden dependencies, runtime errors
   - Rejected as it's considered an anti-pattern

## Migration Strategy

1. Introduce interfaces for existing types
2. Add constructors that accept dependencies
3. Create wire.go and progressively wire components
4. Remove global state access
5. Update tests to use dependency injection

## References

- [Dependency Injection by Martin Fowler](https://martinfowler.com/articles/injection.html)
- [Google Wire](https://github.com/google/wire)
- [Uber Fx](https://github.com/uber-go/fx)
- SOLID principles (Dependency Inversion)
- Current implementation in `internal/app/wire.go`
