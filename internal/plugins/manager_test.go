package plugins

import (
	"context"
	"testing"
	"time"
)

func TestPluginManager_RegisterBuiltinPlugins(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false, // 외부에서 등록
		HealthCheckInterval:  0,     // 헬스체크 비활성화
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 기본 상태에서는 핸들러가 없어야 함
	handlers := manager.ListHandlers()
	if len(handlers) != 0 {
		t.Errorf("Expected 0 handlers, got %d", len(handlers))
	}
}

func TestPluginManager_RegisterCustomPlugin(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false,
		HealthCheckInterval:  0,
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 테스트용 간단한 핸들러 생성
	handler := NewBasePackageHandler("test", "Test Handler", "1.0.0")
	err = handler.Initialize(nil) // 핸들러 초기화
	if err != nil {
		t.Fatalf("Failed to initialize test handler: %v", err)
	}
	err = manager.registry.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register test handler: %v", err)
	}

	// 핸들러가 등록되었는지 확인
	registeredHandler, exists := manager.registry.GetHandler("test")
	if !exists {
		t.Error("Test handler not found after registration")
	}

	if registeredHandler.GetType() != "test" {
		t.Errorf("Expected handler type 'test', got '%s'", registeredHandler.GetType())
	}
}

func TestPluginManager_UnregisterPlugin(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false,
		HealthCheckInterval:  0,
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 테스트 핸들러 등록
	handler := NewBasePackageHandler("test", "Test Handler", "1.0.0")
	err = handler.Initialize(nil) // 핸들러 초기화
	if err != nil {
		t.Fatalf("Failed to initialize test handler: %v", err)
	}
	err = manager.registry.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register test handler: %v", err)
	}

	// 핸들러 등록 확인
	_, exists := manager.registry.GetHandler("test")
	if !exists {
		t.Error("Test handler not found before unregistration")
	}

	// 핸들러 등록 해제
	err = manager.registry.Unregister("test")
	if err != nil {
		t.Fatalf("Failed to unregister handler: %v", err)
	}

	// 핸들러가 제거되었는지 확인
	_, exists = manager.registry.GetHandler("test")
	if exists {
		t.Error("Test handler still exists after unregistration")
	}
}

func TestPluginManager_GetHandlersByMode(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false,
		HealthCheckInterval:  0,
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 테스트 핸들러 등록
	handler := NewBasePackageHandler("test", "Test Handler", "1.0.0")
	err = handler.Initialize(nil) // 핸들러 초기화
	if err != nil {
		t.Fatalf("Failed to initialize test handler: %v", err)
	}
	err = manager.registry.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register test handler: %v", err)
	}

	// 프록시 모드 핸들러 조회
	proxyHandlers := manager.registry.GetHandlersByMode(ProxyMode)
	if len(proxyHandlers) == 0 {
		t.Error("No proxy mode handlers found")
	}

	// 테스트 핸들러가 프록시 모드를 지원하는지 확인
	testHandler, exists := manager.registry.GetHandler("test")
	if !exists {
		t.Error("Test handler not found")
	}

	if !testHandler.SupportsMode(ProxyMode) {
		t.Error("Test handler should support proxy mode")
	}
}

func TestPluginManager_HealthCheck(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false,
		HealthCheckInterval:  0,
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 빈 상태에서 헬스체크
	results := manager.HealthCheck(ctx)

	if len(results) != 0 {
		t.Errorf("Expected 0 health check results, got %d", len(results))
	}
}

func TestPluginManager_Statistics(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false,
		HealthCheckInterval:  0,
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 통계 조회
	stats := manager.GetStatistics()

	if stats["total_plugins"].(int) != 0 {
		t.Errorf("Expected 0 plugins, got %d", stats["total_plugins"].(int))
	}

	if stats["total_handlers"].(int) != 0 {
		t.Errorf("Expected 0 handlers, got %d", stats["total_handlers"].(int))
	}

	if !stats["started"].(bool) {
		t.Error("Plugin manager should be started")
	}

	if stats["stopped"].(bool) {
		t.Error("Plugin manager should not be stopped")
	}
}

func TestPluginManager_Lifecycle(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()

	// 시작 전에는 핸들러 조회 불가
	handlers := manager.ListHandlers()
	if len(handlers) != 0 {
		t.Errorf("Expected 0 handlers before starting, got %d", len(handlers))
	}

	// 매니저 시작
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false,
		HealthCheckInterval:  0,
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}

	// 시작 후에도 핸들러는 수동 등록 필요
	handlers = manager.ListHandlers()
	if len(handlers) != 0 {
		t.Errorf("Expected 0 handlers after starting (auto-register disabled), got %d", len(handlers))
	}

	// 매니저 중지
	err = manager.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop plugin manager: %v", err)
	}

	// 중지 후 통계 확인
	stats := manager.GetStatistics()
	if !stats["stopped"].(bool) {
		t.Error("Plugin manager should be stopped")
	}
}

func TestPluginManager_ConcurrentAccess(t *testing.T) {
	manager := NewPluginManager(PluginManagerConfig{})

	ctx := context.Background()
	err := manager.Start(ctx, PluginManagerConfig{
		AutoRegisterBuiltins: false,
		HealthCheckInterval:  0,
	})
	if err != nil {
		t.Fatalf("Failed to start plugin manager: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 테스트 핸들러 등록
	handler := NewBasePackageHandler("test", "Test Handler", "1.0.0")
	err = handler.Initialize(nil) // 핸들러 초기화
	if err != nil {
		t.Fatalf("Failed to initialize test handler: %v", err)
	}
	err = manager.registry.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register test handler: %v", err)
	}

	// 동시에 여러 고루틴에서 핸들러 조회
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// 핸들러 조회
			testHandler, exists := manager.registry.GetHandler("test")
			if !exists {
				t.Error("Test handler not found in concurrent access")
				return
			}

			// 헬스체크 수행
			err := testHandler.HealthCheck(ctx)
			if err != nil {
				t.Errorf("Health check failed in concurrent access: %v", err)
				return
			}
		}()
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent access test")
		}
	}
}
