---
priority: medium
severity: medium
category: monitoring
source: tasks/todo/README.md (section E)
---

# Health & Observability

헬스 체크와 관측성 향상을 위한 태스크입니다.

## Tasks

- [ ] Add per-adapter health checks to CI smoke step
  - Use `internal/factory.HandlerAdapterFactory.HealthCheck()`
  - Fail build if any adapter unhealthy
  - Integrate with existing health check infrastructure

## Acceptance Criteria
- Health checks run automatically in CI pipeline
- Build fails fast if any critical component is unhealthy
- Health status is properly logged and observable
