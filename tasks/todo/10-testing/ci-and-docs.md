---
priority: high
severity: high
category: ci-docs
source: tasks/todo/README.md (sections C and D)
---

# CI Matrix & Documentation

CI 매트릭스 확장과 관련 문서화 태스크들입니다.

## Tasks

- [x] Expand CI to run per-type integration and E2E (matrix)
  - Update: `.gitlab-ci.yml` / GitHub Actions if present
  - Cache upstream fixtures, parallelize jobs, collect logs

- [ ] Add E2E how-to per type in docs
  - Path: `docs/05-development/testing/` (new pages)
  - Reference: scripts in `tests/e2e/scripts`

- [ ] Ensure examples include auth/no-auth and mirror/proxy variants
  - Review `examples/proxy-types/*.yaml`
  - Add missing configuration examples for each proxy type

## Acceptance Criteria
- CI runs tests in matrix mode for all package types
- Documentation is complete with commands and expected outputs
- Examples cover both authenticated and unauthenticated scenarios
