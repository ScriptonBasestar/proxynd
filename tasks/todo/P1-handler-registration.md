# P1: Handler Registration for Remaining Package Managers

**Priority**: P1 (High)
**Status**: Pending
**Created**: 2025-12-04
**Estimated Time**: 8-10 hours

---

## Overview

Register handlers for 4 remaining package managers: Docker, PyPI, YUM, APK.
Service implementations already exist, just need proper registration in DI container.

---

## Current State

### ✅ Complete
- Maven handler: Registered and working
- NPM handler: Registered and working
- APT handler: Registered and working

### ⏳ Pending
- Docker handler: Implementation exists in `internal/services/docker/` and `internal/services/proxy/docker_service.go`
- PyPI handler: Need to verify implementation
- YUM handler: Implementation exists in `internal/services/yum/` and `internal/services/proxy/yum_service.go`
- APK handler: Implementation exists in `internal/services/apk/` and `internal/services/proxy/apk_service.go`

---

## Files to Modify

### 1. `internal/app/providers.go`
**Lines**: 271, 245, 239

**Current TODOs**:
```go
// Line 271: TODO: Register other handlers (Docker, PIP, YUM, APK)
// Line 245: TODO: Implement handlers when ready
// Line 239: TODO: Implement proper router when handlers are ready
```

**Actions**:
- [ ] Implement `ProvideDockerHandler()`
- [ ] Implement `ProvidePyPIHandler()`
- [ ] Implement `ProvideYumHandler()`
- [ ] Implement `ProvideApkHandler()`
- [ ] Register handlers in `registerProviders()`

### 2. `internal/app/container.go`
**Lines**: 673, 496

**Current TODOs**:
```go
// Line 673: TODO: Register other handlers (Docker, PIP, YUM, APK)
// Line 496: TODO: Implement deep comparison if needed
```

**Actions**:
- [ ] Add handler getter methods for Docker, PyPI, YUM, APK
- [ ] Update container registration logic
- [ ] Verify deep comparison requirement

### 3. `internal/app/routes.go`
**Lines**: 134, 165, 315

**Current TODOs**:
```go
// Line 134: TODO: HEXAGONAL_MIGRATION - Add other routers as they are migrated
// Line 165: TODO: HEXAGONAL_MIGRATION - Add proper auth middleware integration when OAuth2Config is available
// Line 315: TODO: Initialize AnsibleHandler with proper dependencies
```

**Actions**:
- [ ] Add routes for Docker proxy
- [ ] Add routes for PyPI proxy
- [ ] Add routes for YUM proxy
- [ ] Add routes for APK proxy

---

## Implementation Steps

### Step 1: Verify Service Implementations (30 min)
```bash
# Check if services exist and are complete
ls -la internal/services/docker/
ls -la internal/services/yum/
ls -la internal/services/apk/
grep -r "PyPI\|pypi" internal/services/

# Review service interfaces
cat internal/services/proxy/docker_service.go
cat internal/services/proxy/yum_service.go
cat internal/services/proxy/apk_service.go
```

### Step 2: Implement Provider Functions (2-3 hours)
Add to `internal/app/providers.go`:

```go
// ProvideDockerHandler provides Docker proxy handler
func ProvideDockerHandler(c *Container) (interface{}, error) {
    service, err := c.GetDockerService()
    if err != nil {
        return nil, fmt.Errorf("failed to get docker service: %w", err)
    }

    return handlers.NewDockerHandler(service), nil
}

// ProvidePyPIHandler provides PyPI proxy handler
func ProvidePyPIHandler(c *Container) (interface{}, error) {
    service, err := c.GetPyPIService()
    if err != nil {
        return nil, fmt.Errorf("failed to get pypi service: %w", err)
    }

    return handlers.NewPyPIHandler(service), nil
}

// ProvideYumHandler provides YUM proxy handler
func ProvideYumHandler(c *Container) (interface{}, error) {
    service, err := c.GetYumService()
    if err != nil {
        return nil, fmt.Errorf("failed to get yum service: %w", err)
    }

    return handlers.NewYumHandler(service), nil
}

// ProvideApkHandler provides APK proxy handler
func ProvideApkHandler(c *Container) (interface{}, error) {
    service, err := c.GetApkService()
    if err != nil {
        return nil, fmt.Errorf("failed to get apk service: %w", err)
    }

    return handlers.NewApkHandler(service), nil
}
```

Register in `registerProviders()`:
```go
p.Register("handlers.DockerHandler", ProvideDockerHandler)
p.Register("handlers.PyPIHandler", ProvidePyPIHandler)
p.Register("handlers.YumHandler", ProvideYumHandler)
p.Register("handlers.ApkHandler", ProvideApkHandler)
```

### Step 3: Update Container (2 hours)
Add to `internal/app/container.go`:

```go
// GetDockerHandler retrieves Docker handler
func (c *Container) GetDockerHandler() (*handlers.DockerHandler, error) {
    handler, err := c.providers.Get("handlers.DockerHandler", c)
    if err != nil {
        return nil, err
    }
    return handler.(*handlers.DockerHandler), nil
}

// GetPyPIHandler retrieves PyPI handler
func (c *Container) GetPyPIHandler() (*handlers.PyPIHandler, error) {
    handler, err := c.providers.Get("handlers.PyPIHandler", c)
    if err != nil {
        return nil, err
    }
    return handler.(*handlers.PyPIHandler), nil
}

// GetYumHandler retrieves YUM handler
func (c *Container) GetYumHandler() (*handlers.YumHandler, error) {
    handler, err := c.providers.Get("handlers.YumHandler", c)
    if err != nil {
        return nil, err
    }
    return handler.(*handlers.YumHandler), nil
}

// GetApkHandler retrieves APK handler
func (c *Container) GetApkHandler() (*handlers.ApkHandler, error) {
    handler, err := c.providers.Get("handlers.ApkHandler", c)
    if err != nil {
        return nil, err
    }
    return handler.(*handlers.ApkHandler), nil
}
```

### Step 4: Add Routes (2-3 hours)
Add to `internal/app/routes.go`:

```go
// Docker routes
dockerHandler, err := container.GetDockerHandler()
if err != nil {
    logger.Error("Failed to get Docker handler", "error", err)
} else {
    dockerGroup := app.Group("/docker")
    dockerGroup.Get("/*", dockerHandler.HandleRequest)
    dockerGroup.Post("/*", dockerHandler.HandleRequest)
}

// PyPI routes
pypiHandler, err := container.GetPyPIHandler()
if err != nil {
    logger.Error("Failed to get PyPI handler", "error", err)
} else {
    pypiGroup := app.Group("/pypi")
    pypiGroup.Get("/*", pypiHandler.HandleRequest)
    pypiGroup.Post("/*", pypiHandler.HandleRequest)
}

// YUM routes
yumHandler, err := container.GetYumHandler()
if err != nil {
    logger.Error("Failed to get YUM handler", "error", err)
} else {
    yumGroup := app.Group("/yum")
    yumGroup.Get("/*", yumHandler.HandleRequest)
    yumGroup.Post("/*", yumHandler.HandleRequest)
}

// APK routes
apkHandler, err := container.GetApkHandler()
if err != nil {
    logger.Error("Failed to get APK handler", "error", err)
} else {
    apkGroup := app.Group("/apk")
    apkGroup.Get("/*", apkHandler.HandleRequest)
    apkGroup.Post("/*", apkHandler.HandleRequest)
}
```

### Step 5: Testing (1-2 hours)
```bash
# Unit tests
go test ./internal/app/... -v

# Integration tests
go test -tags integration ./tests/integration/docker/... -v
go test -tags integration ./tests/integration/pypi/... -v
go test -tags integration ./tests/integration/yum/... -v
go test -tags integration ./tests/integration/apk/... -v

# E2E tests
go test -tags e2e ./tests/e2e/... -v -run "Docker|PyPI|Yum|Apk"

# Manual testing
./bin/proxynd-core &
curl http://localhost:8080/docker/v2/
curl http://localhost:8080/pypi/simple/
curl http://localhost:8080/yum/repodata/repomd.xml
curl http://localhost:8080/apk/main/x86_64/APKINDEX.tar.gz
```

---

## Acceptance Criteria

- [ ] All 4 handlers registered in DI container
- [ ] Routes configured and responding
- [ ] Unit tests pass for all handlers
- [ ] Integration tests pass
- [ ] E2E tests pass with real upstream data
- [ ] No TODO comments remain in modified files
- [ ] Documentation updated

---

## Dependencies

**None** - This is a standalone task

**Blocks**:
- Full 7 package manager feature completion
- Production deployment readiness

---

## Related Files

### Service Implementations
- `internal/services/docker/`
- `internal/services/proxy/docker_service.go`
- `internal/services/yum/`
- `internal/services/proxy/yum_service.go`
- `internal/services/apk/`
- `internal/services/proxy/apk_service.go`

### Handlers (may need creation)
- `internal/adapters/http/fiber/handlers/proxy/docker_handler.go`
- `internal/adapters/http/fiber/handlers/proxy/pypi_handler.go`
- `internal/adapters/http/fiber/handlers/proxy/yum_handler.go`
- `internal/adapters/http/fiber/handlers/proxy/apk_handler.go`

### Tests
- `tests/integration/docker/`
- `tests/integration/pypi/`
- `tests/integration/yum/`
- `tests/integration/apk/`
- `tests/e2e/`

---

## Notes

- PyPI naming: Domain uses "pip", adapters use "pypi" - prefer "pypi" for consistency
- Docker handler may need special handling for registry v2 API
- YUM/APK may share similar patterns - look for reusable code
- Check if handlers already exist but are just not registered

---

**Next Task After Completion**: P1-hexagonal-migration-tracking.md
