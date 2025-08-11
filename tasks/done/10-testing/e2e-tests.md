---
priority: medium
severity: medium
category: testing
source: tasks/todo/README.md (section B)
---

# E2E Tests & Scripts

E2E 테스트 스크립트와 업스트림 픽스처를 구현하는 태스크들입니다.

## Tasks

- [x] Add Maven E2E script
  - File: `tests/e2e/scripts/test-maven.sh`
  - Upstream fixtures: minimal maven repo layout
  - Scenarios: artifact fetch, metadata, index browsing
  - Completed: Added comprehensive Maven E2E script with test scenarios, upstream fixtures, and Makefile integration

- [x] Add YUM E2E script
  - File: `tests/e2e/scripts/test-yum.sh`
  - Upstream fixtures: minimal repodata
  - Scenarios: metadata sync, rpm fetch
  - Completed: Added comprehensive YUM E2E script with test scenarios, upstream fixtures, and Makefile integration

- [x] Add APK E2E script
  - File: `tests/e2e/scripts/test-apk.sh`
  - Upstream fixtures: index, signatures
  - Scenarios: index fetch, package fetch, bad signature
  - Completed: Added comprehensive APK E2E script with 10 test scenarios, minimal Alpine Linux upstream fixtures (APKINDEX.tar.gz, mock APK packages), and Makefile integration

## Acceptance Criteria
- E2E scripts runnable locally via `tests/e2e/Makefile`
- Scripts provide clear pass/fail status
- Upstream fixtures are minimal but representative
