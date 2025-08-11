---
status: suspended
reason: 원본 파일이 11개의 활성 작업으로 과밀 상태였음. 4개 파일로 분할 완료. 재검토 필요
original_location: tasks/todo/README.md
split_into:
  - tasks/todo/10-testing/integration-tests.md
  - tasks/todo/10-testing/e2e-tests.md
  - tasks/todo/10-testing/ci-and-docs.md
  - tasks/todo/10-testing/health-observability.md
date_processed: 2025-08-11
---

# Quality & Test Coverage Plan

This plan lists concrete tasks to strengthen coverage and quality across all supported package managers (maven, npm, pip, apt, yum, apk, docker).

## Goals
- Close test gaps (integration/E2E) per package type
- Standardize verification matrices across types
- Improve CI matrix to run full quality gates per type

---

## A. Integration Tests

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

## B. E2E Tests & Scripts

- [ ] Add Maven E2E script
  - File: `tests/e2e/scripts/test-maven.sh`
  - Upstream fixtures: minimal maven repo layout
  - Scenarios: artifact fetch, metadata, index browsing

- [ ] Add YUM E2E script
  - File: `tests/e2e/scripts/test-yum.sh`
  - Upstream fixtures: minimal repodata
  - Scenarios: metadata sync, rpm fetch

- [ ] Add APK E2E script
  - File: `tests/e2e/scripts/test-apk.sh`
  - Upstream fixtures: index, signatures
  - Scenarios: index fetch, package fetch, bad signature

## C. CI Matrix

- [ ] Expand CI to run per-type integration and E2E (matrix)
  - Update: `.gitlab-ci.yml` / GitHub Actions if present
  - Cache upstream fixtures, parallelize jobs, collect logs

## D. Docs & Examples

- [ ] Add E2E how-to per type in docs
  - Path: `docs/05-development/testing/` (new pages)
  - Reference: scripts in `tests/e2e/scripts`

- [ ] Ensure examples include auth/no-auth and mirror/proxy variants
  - Review `examples/proxy-types/*.yaml`

## E. Health & Observability

- [ ] Add per-adapter health checks to CI smoke step
  - Use `internal/factory.HandlerAdapterFactory.HealthCheck()`
  - Fail build if any adapter unhealthy

## F. Acceptance Criteria (Global)
- All new tests pass on CI in matrix mode
- Minimum coverage threshold maintained or improved
- E2E scripts runnable locally via `tests/e2e/Makefile`
- Docs updated with commands and expected outputs

---

## Task Breakdown (Assignees TBD)
1) npm integration test (owner: TBD, ETA: 1d)
2) maven integration enhancements (owner: TBD, ETA: 0.5d)
3) yum integration enhancements (owner: TBD, ETA: 0.5d)
4) apk integration enhancements (owner: TBD, ETA: 0.5d)
5) e2e-maven script + fixtures (owner: TBD, ETA: 1d)
6) e2e-yum script + fixtures (owner: TBD, ETA: 1d)
7) e2e-apk script + fixtures (owner: TBD, ETA: 1d)
8) CI matrix expansion (owner: TBD, ETA: 1d)
9) docs updates (owner: TBD, ETA: 0.5d)
10) health check in CI (owner: TBD, ETA: 0.5d)

> Total: ~8.0d (parallelizable)
