package app

// TODO: HEXAGONAL_MIGRATION - These tests depend on GetUnifiedConfig(), AddConfigChangeCallback(), and ReloadConfig()
// which are part of the unified config system migration (P1-hexagonal-migration-tracking.md).
// Re-enable these tests after unified config loading is implemented.
//
// Original tests covered:
// - TestContainer_GetUnifiedConfig
// - TestContainer_ReloadConfig
// - TestContainer_ConfigChangeCallback
// - TestContainer_ConfigChangeNotification
// - TestContainer_GetConfigCaching
//
// See: proxynd-core/tasks/todo/P1-hexagonal-migration-tracking.md (Phase 2: Unified Config)
