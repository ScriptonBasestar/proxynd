---
priority: high
severity: medium
category: testing
source: tasks/todo/README.md (section A)
---

# Integration Tests Enhancement

강화해야 할 통합 테스트들을 정리한 태스크입니다.

## Tasks

- [ ] Add npm integration test
  - File: `tests/integration/npm_integration_test.go`
  - Scope: GET tarball, metadata, scoped packages, auth fallback
  - Acceptance: green on CI, covers 3xx, 4xx, cache hit/miss

- [ ] Review maven integration scenarios (if any gaps)
  - File(s): `tests/integration/maven_integration_test.go`
  - Add cases: SNAPSHOT, checksum endpoints, directory listing toggle

- [ ] Harden yum integration tests
  - File: `tests/integration/yum_integration_test.go`
  - Add: repodata freshness, gzip/pgp headers, range requests

- [ ] APK advanced cases
  - File: `tests/integration/apk_integration_test.go`
  - Add: signature verification failure path, mirror switch

## Acceptance Criteria
- All new tests pass on CI
- Minimum coverage threshold maintained or improved
- Tests cover both success and failure scenarios
