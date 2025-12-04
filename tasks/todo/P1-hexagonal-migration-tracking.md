# P1: Hexagonal Architecture Migration Tracking

**Priority**: P1 (High)
**Status**: In Progress
**Created**: 2025-12-04
**Estimated Time**: Ongoing (track progress)

---

## Overview

Track and complete the ongoing hexagonal architecture migration.
Multiple TODOs reference "HEXAGONAL_MIGRATION" - need to consolidate and track.

---

## Current State

### ✅ Complete
- Core hexagonal architecture implemented
- Ports and adapters pattern established
- Dependency injection container working
- Most services follow hexagonal pattern

### ⏳ In Progress
- Unified config loading
- Auth middleware integration
- Route configuration runtime switching
- Router migration

---

## TODOs to Track

### 1. Unified Config Loading
**File**: `internal/app/app.go`
**Line**: TODO: HEXAGONAL_MIGRATION - Convert to unified config loading

**Current State**: Legacy config loading mixed with new patterns

**Actions**:
- [ ] Audit all config loading patterns
- [ ] Create unified config loader interface
- [ ] Implement adapter for config loading
- [ ] Migrate all components to use unified loader
- [ ] Remove legacy config code

### 2. Auth Middleware Integration
**File**: `internal/app/routes.go`
**Line**: TODO: HEXAGONAL_MIGRATION - Add proper auth middleware integration when OAuth2Config is available

**Current State**: Auth middleware exists but not fully integrated with OAuth2

**Actions**:
- [ ] Verify OAuth2Config structure
- [ ] Create auth middleware adapter
- [ ] Integrate with existing auth system
- [ ] Add configuration for auth providers
- [ ] Test with all auth methods (JWT, OAuth2, API Keys)

### 3. Route Config Runtime Switching
**File**: `internal/app/app.go`
**Line**: TODO: HEXAGONAL_MIGRATION - Add route config for runtime switching

**Current State**: Route configuration is static

**Actions**:
- [ ] Design route configuration schema
- [ ] Implement runtime route registration/deregistration
- [ ] Add config reload support for routes
- [ ] Test route switching without restart
- [ ] Document route configuration options

### 4. Router Migration
**File**: `internal/app/routes.go`
**Line**: TODO: HEXAGONAL_MIGRATION - Add other routers as they are migrated

**Current State**: Some routers migrated, others pending

**Actions**:
- [ ] List all routers needing migration
- [ ] Create migration plan for each router
- [ ] Migrate routers one by one
- [ ] Update route registration
- [ ] Remove legacy router code

---

## Migration Checklist

### Phase 1: Assessment (1-2 hours)
- [ ] Identify all HEXAGONAL_MIGRATION TODOs
- [ ] Map current architecture vs target architecture
- [ ] List components not following hexagonal pattern
- [ ] Prioritize migration order
- [ ] Create detailed migration plan

### Phase 2: Unified Config (4-6 hours)
- [ ] Design config port interface
- [ ] Implement config adapter
- [ ] Migrate config loading
- [ ] Test configuration system
- [ ] Remove legacy config code

### Phase 3: Auth Integration (3-4 hours)
- [ ] Complete OAuth2 config structure
- [ ] Implement auth middleware adapter
- [ ] Integrate with DI container
- [ ] Add comprehensive auth tests
- [ ] Document auth configuration

### Phase 4: Route Configuration (4-5 hours)
- [ ] Design route config schema
- [ ] Implement runtime route manager
- [ ] Add hot reload support
- [ ] Test route switching
- [ ] Document route management

### Phase 5: Router Migration (6-8 hours)
- [ ] Migrate remaining routers
- [ ] Update route registration
- [ ] Consolidate routing logic
- [ ] Remove duplicate code
- [ ] Verify all endpoints work

### Phase 6: Cleanup (2-3 hours)
- [ ] Remove all HEXAGONAL_MIGRATION TODOs
- [ ] Delete legacy code
- [ ] Update architecture documentation
- [ ] Add ADR for completed migration
- [ ] Celebrate! 🎉

---

## Architecture Principles

### Core Hexagonal Rules
1. **Domain = Zero Dependencies**: Pure business logic
2. **Ports = Interfaces**: Define contracts
3. **Adapters = Implementations**: Implement ports
4. **Dependency Flow**: Inward only (adapters → ports → usecases → domain)

### Current Violations to Fix
- Config loading still has mixed patterns
- Some routes bypass DI container
- Auth middleware not fully decoupled
- Legacy code mixed with new architecture

---

## Files to Review

### Core Files
- `internal/app/app.go` - Main application setup
- `internal/app/routes.go` - Route configuration
- `internal/app/container.go` - DI container
- `internal/app/providers.go` - Provider registration

### Domain Layer
- `internal/domain/*/` - Should have zero external deps

### Ports Layer
- `internal/ports/` - Interface definitions

### Adapters Layer
- `internal/adapters/*/` - Implementations

### Services Layer
- `internal/services/*/` - Service coordination

---

## Validation Criteria

### Architecture Compliance
- [ ] All domain code has zero external dependencies
- [ ] All adapters implement defined ports
- [ ] Dependency flow is strictly inward
- [ ] No circular dependencies
- [ ] DI container manages all dependencies

### Code Quality
- [ ] No HEXAGONAL_MIGRATION TODOs remain
- [ ] No legacy code patterns
- [ ] Consistent naming conventions
- [ ] Comprehensive test coverage
- [ ] Documentation updated

### Functionality
- [ ] All features work after migration
- [ ] Performance not degraded
- [ ] No breaking changes to APIs
- [ ] Configuration backward compatible
- [ ] Tests all pass

---

## Risks and Mitigation

### Risk 1: Breaking Changes
**Impact**: High
**Mitigation**:
- Maintain backward compatibility
- Use feature flags for migration
- Comprehensive testing before removal

### Risk 2: Performance Degradation
**Impact**: Medium
**Mitigation**:
- Benchmark before/after
- Profile hot paths
- Optimize adapter implementations

### Risk 3: Incomplete Migration
**Impact**: Medium
**Mitigation**:
- Track all TODOs systematically
- Regular progress reviews
- Clear completion criteria

---

## Documentation to Update

- [ ] `docs/10-architecture/hexagonal-architecture.md`
- [ ] `docs/10-architecture/adr/` - Add migration ADR
- [ ] `proxynd-core/CLAUDE.md` - Update architecture section
- [ ] `docs/90-development/migration-guide.md` - Create if needed
- [ ] API documentation - If endpoints change

---

## Related ADRs

**To Create**:
- ADR-009: Hexagonal Architecture Migration Strategy
- ADR-010: Unified Configuration System
- ADR-011: Auth Middleware Integration

---

## Progress Tracking

| Component | Status | Progress | Completed |
|-----------|--------|----------|-----------|
| Config Loading | In Progress | 40% | - |
| Auth Middleware | Pending | 0% | - |
| Route Config | Pending | 0% | - |
| Router Migration | In Progress | 60% | - |
| **Overall** | **In Progress** | **35%** | - |

---

## Next Steps

1. **Immediate** (This Sprint):
   - Complete config loading migration
   - Document current architecture state
   - Create detailed OAuth2 integration plan

2. **Short-term** (Next Sprint):
   - Implement auth middleware adapter
   - Add route configuration system
   - Migrate remaining routers

3. **Long-term** (Next Quarter):
   - Complete all migrations
   - Remove all legacy code
   - Publish migration guide

---

**Dependencies**: None - can work independently

**Blocks**: Architecture documentation completion

**Related Tasks**:
- P1-handler-registration.md (some overlap with router migration)
- P2-config-hot-reload.md (unified config needed first)

---

**Last Updated**: 2025-12-04
**Review Frequency**: Weekly
