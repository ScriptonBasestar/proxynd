package handlers

import (
	"fmt"
	"sync"
	"time"

	"proxynd/internal/app"
	"proxynd/logging"
)

// HandlerRegistry 핸들러 레지스트리
type HandlerRegistry struct {
	handlers map[string]Handler
	factory  *HandlerFactoryImpl
	mu       sync.RWMutex
	logger   logging.Logger
	stats    *RegistryStats
}

// RegistryStats 레지스트리 통계
type RegistryStats struct {
	CreatedHandlers int64
	CacheHits       int64
	CacheMisses     int64
	TotalRequests   int64
	mu              sync.RWMutex
}

// NewHandlerRegistry 새로운 핸들러 레지스트리 생성
func NewHandlerRegistry(container *app.Container) *HandlerRegistry {
	registry := &HandlerRegistry{
		handlers: make(map[string]Handler),
		factory:  NewHandlerFactory(container),
		logger:   logging.GetLogger(),
		stats:    &RegistryStats{},
	}

	// 기본 핸들러들 등록
	RegisterDefaultHandlers(registry.factory)

	return registry
}

// Register 핸들러 타입과 생성자 등록
func (r *HandlerRegistry) Register(handlerType string, creator HandlerCreator) {
	r.factory.Register(handlerType, creator)
}

// Get 핸들러 가져오기 (캐시된 인스턴스 또는 새 인스턴스)
func (r *HandlerRegistry) Get(handlerType string) (Handler, error) {
	r.stats.mu.Lock()
	r.stats.TotalRequests++
	r.stats.mu.Unlock()

	// 캐시된 핸들러 확인
	r.mu.RLock()
	if handler, exists := r.handlers[handlerType]; exists {
		r.mu.RUnlock()
		r.stats.mu.Lock()
		r.stats.CacheHits++
		r.stats.mu.Unlock()
		return handler, nil
	}
	r.mu.RUnlock()

	// 더블 체크 락킹으로 새 핸들러 생성
	r.mu.Lock()
	defer r.mu.Unlock()

	// 다시 한 번 확인 (다른 고루틴이 생성했을 수 있음)
	if handler, exists := r.handlers[handlerType]; exists {
		r.stats.mu.Lock()
		r.stats.CacheHits++
		r.stats.mu.Unlock()
		return handler, nil
	}

	// 새 핸들러 생성
	handler, err := r.factory.Create(handlerType)
	if err != nil {
		r.stats.mu.Lock()
		r.stats.CacheMisses++
		r.stats.mu.Unlock()
		return nil, err
	}

	// 캐시에 저장
	r.handlers[handlerType] = handler

	r.stats.mu.Lock()
	r.stats.CreatedHandlers++
	r.stats.CacheMisses++
	r.stats.mu.Unlock()

	// 로그 기록 (handler가 nil이 아닌 경우만)
	if handler != nil {
		r.logger.Info("Handler created and cached",
			logging.F("type", handlerType),
			logging.F("handler_name", handler.Name()),
			logging.F("total_cached", len(r.handlers)),
		)
	} else {
		r.logger.Warn("Handler created but is nil",
			logging.F("type", handlerType),
			logging.F("total_cached", len(r.handlers)),
		)
	}

	return handler, nil
}

// GetOrCreate 핸들러 가져오기 또는 생성 (Get의 별칭)
func (r *HandlerRegistry) GetOrCreate(handlerType string) (Handler, error) {
	return r.Get(handlerType)
}

// GetCached 캐시된 핸들러만 가져오기 (없으면 nil)
func (r *HandlerRegistry) GetCached(handlerType string) Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.handlers[handlerType]
}

// Remove 캐시된 핸들러 제거
func (r *HandlerRegistry) Remove(handlerType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if handler, exists := r.handlers[handlerType]; exists {
		delete(r.handlers, handlerType)
		r.logger.Info("Handler removed from cache",
			logging.F("type", handlerType),
			logging.F("handler_name", handler.Name()),
			logging.F("remaining_cached", len(r.handlers)),
		)
		return true
	}
	return false
}

// Clear 모든 캐시된 핸들러 제거
func (r *HandlerRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := len(r.handlers)
	r.handlers = make(map[string]Handler)

	r.logger.Info("All cached handlers cleared",
		logging.F("cleared_count", count),
	)
}

// GetSupportedTypes 지원되는 핸들러 타입 목록 반환
func (r *HandlerRegistry) GetSupportedTypes() []string {
	return r.factory.GetSupportedTypes()
}

// IsSupported 특정 핸들러 타입이 지원되는지 확인
func (r *HandlerRegistry) IsSupported(handlerType string) bool {
	return r.factory.IsSupported(handlerType)
}

// GetStats 레지스트리 통계 반환
func (r *HandlerRegistry) GetStats() map[string]interface{} {
	r.mu.RLock()
	cachedCount := len(r.handlers)
	cachedTypes := make([]string, 0, cachedCount)
	for handlerType := range r.handlers {
		cachedTypes = append(cachedTypes, handlerType)
	}
	r.mu.RUnlock()

	r.stats.mu.RLock()
	stats := map[string]interface{}{
		"cached_handlers":  cachedCount,
		"cached_types":     cachedTypes,
		"created_handlers": r.stats.CreatedHandlers,
		"cache_hits":       r.stats.CacheHits,
		"cache_misses":     r.stats.CacheMisses,
		"total_requests":   r.stats.TotalRequests,
		"supported_types":  r.factory.GetSupportedTypes(),
	}
	r.stats.mu.RUnlock()

	// 캐시 히트율 계산
	if r.stats.TotalRequests > 0 {
		stats["cache_hit_rate"] = float64(r.stats.CacheHits) / float64(r.stats.TotalRequests)
	} else {
		stats["cache_hit_rate"] = 0.0
	}

	return stats
}

// ResetStats 통계 초기화
func (r *HandlerRegistry) ResetStats() {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()

	r.stats.CreatedHandlers = 0
	r.stats.CacheHits = 0
	r.stats.CacheMisses = 0
	r.stats.TotalRequests = 0

	r.logger.Info("Registry stats reset")
}

// Health 레지스트리 헬스체크
func (r *HandlerRegistry) Health() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 팩토리 상태 확인
	if r.factory == nil {
		return fmt.Errorf("handler factory is nil")
	}

	// 지원되는 타입이 있는지 확인
	supportedTypes := r.factory.GetSupportedTypes()
	if len(supportedTypes) == 0 {
		return fmt.Errorf("no supported handler types")
	}

	// 캐시된 핸들러들의 헬스체크
	for handlerType, handler := range r.handlers {
		if healthable, ok := handler.(Healthable); ok {
			if err := healthable.HealthCheck(); err != nil {
				return fmt.Errorf("handler %s health check failed: %w", handlerType, err)
			}
		}
	}

	return nil
}

// Preload 지정된 핸들러 타입들을 미리 로드
func (r *HandlerRegistry) Preload(handlerTypes ...string) error {
	for _, handlerType := range handlerTypes {
		if !r.IsSupported(handlerType) {
			return fmt.Errorf("unsupported handler type: %s", handlerType)
		}

		_, err := r.Get(handlerType)
		if err != nil {
			return fmt.Errorf("failed to preload handler %s: %w", handlerType, err)
		}
	}

	r.logger.Info("Handlers preloaded",
		logging.F("types", handlerTypes),
		logging.F("count", len(handlerTypes)),
	)

	return nil
}

// WarmUp 모든 지원되는 핸들러를 미리 로드
func (r *HandlerRegistry) WarmUp() error {
	supportedTypes := r.GetSupportedTypes()
	start := time.Now()

	err := r.Preload(supportedTypes...)
	if err != nil {
		return fmt.Errorf("warmup failed: %w", err)
	}

	r.logger.Info("Registry warmed up",
		logging.F("duration", time.Since(start)),
		logging.F("handler_count", len(supportedTypes)),
	)

	return nil
}

// DefaultHandlerRegistry 기본 핸들러 레지스트리 (싱글톤)
var (
	defaultRegistry     *HandlerRegistry
	defaultRegistryOnce sync.Once
)

// GetDefaultRegistry 기본 핸들러 레지스트리 반환
func GetDefaultRegistry(container *app.Container) *HandlerRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewHandlerRegistry(container)
	})
	return defaultRegistry
}
