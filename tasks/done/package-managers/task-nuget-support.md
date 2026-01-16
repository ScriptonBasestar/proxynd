---
id: TASK-001
title: "Add NuGet proxy support"
type: feature

priority: P1
effort: L

parent: PLAN-001
depends-on: []
blocks: []

created-at: 2026-01-16T07:29:44Z
started-at: 2026-01-16T07:46:05Z
completed-at: 2026-01-16T08:00:32Z
completion-summary: "Added NuGet proxy handler, config, docs, and tests."
---

## Purpose
Implement NuGet (V2/V3) proxy support as outlined in the Future Package Manager Support Plan.

## Work Log
Started at 2026-01-16T07:46:05Z by AI. Scope: NuGet V2/V3 endpoints, push/pull, symbol server, config/docs.

## Scope
### Must
- Implement NuGet V3 HTTP endpoints for metadata and package download.
- Provide NuGet V2 compatibility endpoints for legacy clients.
- Support package push operations with upstream auth passthrough.
- Add symbol server handling for debugging symbols.
- Add configuration and documentation for NuGet.

### Must Not
- Add support for other package managers in this task.

## Definition of Done
- [x] NuGet V2/V3 requests proxy successfully through the adapter.
- [x] Package push and pull operations are covered with tests.
- [x] Configuration and docs are updated for NuGet usage.

## Checklist
- [x] Implementation
- [x] Tests
- [x] Documentation

## Work Summary
- Added a NuGet proxy handler with pass-through support for V2/V3, push, and symbol endpoints.
- Added NuGet config schema, example config, and documentation updates.
- Added NuGet handler and config tests.

## Key Files
- internal/adapters/http/fiber/handlers/proxy/nuget_handler_v3.go
- internal/config/nuget_proxy_settings.go
- docs/20-configuration/configuration-reference.md
- examples/proxy-types/nuget-proxy.yaml

## Verification
- `STORAGE_DIR=$(mktemp -d) GOWORK=off go test ./internal/adapters/http/fiber/handlers/proxy -run NuGet`
- `GOWORK=off go test ./internal/config -run NuGet`
