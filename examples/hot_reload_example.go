package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proxynd/configs"
)

// ExampleConfigLoader 예제 설정 로더
type ExampleConfigLoader struct {
	configPath string
}

func (l *ExampleConfigLoader) Load() (interface{}, error) {
	// 실제 구현에서는 설정 파일을 읽어서 파싱
	return configs.LoadConfig(l.configPath)
}

func main() {
	// 1. 간단한 핫 리로드 사용 예제
	simpleHotReloadExample()

	// 2. 통합 핫 리로드 사용 예제
	unifiedHotReloadExample()
}

// simpleHotReloadExample 간단한 핫 리로드 예제
func simpleHotReloadExample() {
	log.Println("=== Simple Hot Reload Example ===")

	// 설정 로더 생성
	loader := &ExampleConfigLoader{
		configPath: "config.yaml",
	}

	// 핫 리로드 매니저 생성
	hr, err := configs.NewSimpleHotReload("config.yaml", loader)
	if err != nil {
		log.Fatalf("Failed to create hot reload: %v", err)
	}

	// 핸들러 등록
	hr.RegisterHandler("logging", func(oldConfig, newConfig interface{}) error {
		oldCfg := oldConfig.(*configs.UnifiedConfig)
		newCfg := newConfig.(*configs.UnifiedConfig)

		if oldCfg.Logging.Level != newCfg.Logging.Level {
			log.Printf("Log level changed: %s -> %s", oldCfg.Logging.Level, newCfg.Logging.Level)
			// 여기서 실제 로거 레벨 변경
		}

		return nil
	})

	hr.RegisterHandler("cache", func(oldConfig, newConfig interface{}) error {
		oldCfg := oldConfig.(*configs.UnifiedConfig)
		newCfg := newConfig.(*configs.UnifiedConfig)

		if oldCfg.Cache.TTL != newCfg.Cache.TTL {
			log.Printf("Cache TTL changed: %s -> %s", oldCfg.Cache.TTL, newCfg.Cache.TTL)
			// 여기서 실제 캐시 TTL 업데이트
		}

		return nil
	})

	// 핫 리로드 시작
	if err := hr.Start(); err != nil {
		log.Fatalf("Failed to start hot reload: %v", err)
	}

	// 설정 사용 예제
	go func() {
		for {
			config := hr.GetConfig().(*configs.UnifiedConfig)
			log.Printf("Current log level: %s", config.Logging.Level)
			time.Sleep(10 * time.Second)
		}
	}()

	// 종료 시그널 대기
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 정리
	hr.Stop()
}

// unifiedHotReloadExample 통합 핫 리로드 예제
func unifiedHotReloadExample() {
	log.Println("\n=== Unified Hot Reload Example ===")

	// 환경에 따라 Viper 사용 여부 결정
	useViper := os.Getenv("USE_VIPER") == "true"

	// 빠른 설정
	hr, err := configs.QuickSetupHotReload("config.yaml", useViper)
	if err != nil {
		log.Fatalf("Failed to setup hot reload: %v", err)
	}
	defer hr.Stop()

	// 커스텀 핸들러 추가
	hr.RegisterHandler(&CustomReloadHandler{
		serviceName: "MyService",
	})

	// 설정 변경 감지 예제
	detector := configs.NewConfigChangeDetector()
	detector.Register("server.port", func(old, new interface{}) bool {
		oldCfg := old.(*configs.UnifiedConfig)
		newCfg := new.(*configs.UnifiedConfig)
		return oldCfg.Server.Port != newCfg.Server.Port
	})

	// 주기적으로 설정 확인
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			config := hr.GetConfig()
			log.Printf("Server running on port: %d", config.Server.Port)
			log.Printf("Cache backend: %s", config.Cache.Backend)
			log.Printf("Metrics enabled: %v", config.Metrics.Enabled)
		}
	}()

	// 종료 대기
	log.Println("Hot reload is running. Press Ctrl+C to stop.")
	log.Println("Edit config.yaml or send SIGHUP to reload configuration.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

// CustomReloadHandler 커스텀 리로드 핸들러 예제
type CustomReloadHandler struct {
	serviceName string
}

func (h *CustomReloadHandler) OnConfigReload(oldConfig, newConfig *configs.UnifiedConfig) error {
	log.Printf("[%s] Reloading configuration...", h.serviceName)

	// 서비스별 특정 로직
	if oldConfig.Server.Port != newConfig.Server.Port {
		log.Printf("[%s] Port change detected, restart required", h.serviceName)
		return nil // 또는 에러 반환하여 리로드 차단
	}

	// 보안 설정 변경 처리
	if oldConfig.Security.Authentication.BasicAuth.Enabled !=
		newConfig.Security.Authentication.BasicAuth.Enabled {
		if newConfig.Security.Authentication.BasicAuth.Enabled {
			log.Printf("[%s] Enabling authentication", h.serviceName)
			// 인증 활성화 로직
		} else {
			log.Printf("[%s] Disabling authentication", h.serviceName)
			// 인증 비활성화 로직
		}
	}

	return nil
}

func (h *CustomReloadHandler) Name() string {
	return h.serviceName + "ReloadHandler"
}
