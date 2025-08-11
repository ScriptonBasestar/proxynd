---
priority: medium
severity: medium
category: monitoring
source: tasks/todo/README.md (section E)
completed: true
completed_date: 2025-08-11
---

# Health & Observability

헬스 체크와 관측성 향상을 위한 태스크입니다.

## Tasks

- [x] Add per-adapter health checks to CI smoke step
  - Use `internal/factory.HandlerAdapterFactory.HealthCheck()`
  - Fail build if any adapter unhealthy
  - Integrate with existing health check infrastructure

## Acceptance Criteria
- [x] Health checks run automatically in CI pipeline
- [x] Build fails fast if any critical component is unhealthy
- [x] Health status is properly logged and observable

## Implementation Summary

### Changes Made:
1. **Enhanced Health Router** (`routers/health_router.go`):
   - Added new endpoint `/health/adapters` for comprehensive adapter health checks
   - Added individual adapter endpoints `/health/adapters/:type`
   - Integrated with `HandlerAdapterFactory.HealthCheck()` method
   - Added proper HTTP status codes (503 for unhealthy adapters)

2. **Created Smoke Test Script** (`scripts/smoke-test-adapters.sh`):
   - Comprehensive health check script for CI/CD pipeline
   - Supports verbose output and configurable endpoints
   - Proper error handling and exit codes for build failures
   - JSON parsing with fallback for environments without jq

3. **Updated CI Pipeline** (`.github/workflows/ci.yml`):
   - Enhanced smoke test step to use the new script
   - Integrated adapter health checks in staging deployment
   - Build fails if any adapter is unhealthy

4. **Added Unit Tests** (`routers/health_router_test.go`):
   - Comprehensive test coverage for new health endpoints
   - Mock implementation for testing without real adapters
   - Tests for both healthy and unhealthy scenarios

5. **App Integration** (`internal/app/app.go`):
   - Integrated HandlerAdapterFactory with health router
   - Proper initialization order to ensure factory is available

### New Endpoints:
- `GET /health/adapters` - Returns health status of all adapters
- `GET /health/adapters/:type` - Returns health status of specific adapter

### Features:
- Automatic health checks run in CI pipeline
- Build fails fast on unhealthy adapters
- Comprehensive logging and observability
- Backward compatibility with existing health endpoints
