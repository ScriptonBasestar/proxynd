---
id: TASK-005
title: "Add Helm chart proxy support"
type: feature

priority: P1
effort: M

parent: PLAN-001
depends-on: []
blocks: []

created-at: 2026-01-16T07:29:44Z
---

## Purpose
Implement Helm chart repository proxy support, including index caching and chart storage.

## Scope
### Must
- Serve and cache `index.yaml` for chart repositories.
- Proxy chart package downloads (`.tgz`).
- Support Helm 3 OCI registry mode where applicable.
- Add configuration and documentation for Helm usage.

### Must Not
- Implement Kubernetes deployment features outside chart proxying.

## Definition of Done
- [ ] Helm clients can search and install charts via the proxy.
- [ ] Index caching and chart downloads are covered with tests.
- [ ] Configuration and docs are updated for Helm.

## Checklist
- [ ] Implementation
- [ ] Tests
- [ ] Documentation

## Verification
Run Helm adapter tests and perform a sample `helm repo add` and `helm install` using the proxy.
