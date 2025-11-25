package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	containerMetricsOnce      sync.Once
	containerMetricsSingleton *ContainerMetrics
)

// ContainerMetrics Container 기반 핸들러 전용 메트릭
type ContainerMetrics struct {
	// Container 관련 메트릭
	ContainerHandlerInitializations *prometheus.CounterVec
	ContainerHandlerRequests        *prometheus.CounterVec
	ContainerHandlerDuration        *prometheus.HistogramVec
	ContainerHandlerErrors          *prometheus.CounterVec

	// 설정 로딩 효율성 메트릭
	ConfigLoadOperations *prometheus.CounterVec
	ConfigLoadDuration   *prometheus.HistogramVec
	ConfigCacheHits      *prometheus.CounterVec
	ConfigCacheMisses    *prometheus.CounterVec

	// 의존성 주입 관련 메트릭
	ContainerProviderCalls   *prometheus.CounterVec
	HandlerFactoryOperations *prometheus.CounterVec
	HandlerInstancesActive   *prometheus.GaugeVec

	// Handler 성능 메트릭
	HandlerHealthChecks   *prometheus.CounterVec
	HandlerConfigReloads  *prometheus.CounterVec
	HandlerUpstreamBuilds *prometheus.CounterVec

	// 캐시 키 생성 메트릭
	CacheKeyGenerations    *prometheus.CounterVec
	CacheKeyGenerationTime *prometheus.HistogramVec
}

// NewContainerMetrics 새로운 Container 메트릭 인스턴스 생성 (싱글톤)
func NewContainerMetrics() *ContainerMetrics {
	containerMetricsOnce.Do(func() {
		containerMetricsSingleton = newContainerMetricsWithRegistry(prometheus.DefaultRegisterer)
	})
	return containerMetricsSingleton
}

// newContainerMetricsWithRegistry 지정된 레지스트리로 Container 메트릭 생성 (테스트용)
func newContainerMetricsWithRegistry(reg prometheus.Registerer) *ContainerMetrics {
	factory := promauto.With(reg)
	return &ContainerMetrics{
		// Container 핸들러 초기화 메트릭
		ContainerHandlerInitializations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_container_handler_initializations_total",
				Help: "Total number of container handler initializations",
			},
			[]string{"handler_type", "status"}, // status: success, failure
		),

		// Container 핸들러 요청 메트릭
		ContainerHandlerRequests: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_container_handler_requests_total",
				Help: "Total number of requests handled by container handlers",
			},
			[]string{"handler_type", "method", "status"},
		),

		// Container 핸들러 처리 시간 메트릭
		ContainerHandlerDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_container_handler_duration_seconds",
				Help:    "Duration of container handler request processing",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
			},
			[]string{"handler_type", "method"},
		),

		// Container 핸들러 오류 메트릭
		ContainerHandlerErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_container_handler_errors_total",
				Help: "Total number of container handler errors",
			},
			[]string{"handler_type", "error_type"},
		),

		// 설정 로딩 작업 메트릭
		ConfigLoadOperations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_config_load_operations_total",
				Help: "Total number of configuration load operations",
			},
			[]string{"handler_type", "operation", "result"}, // operation: initial, reload, result: success, failure
		),

		// 설정 로딩 시간 메트릭
		ConfigLoadDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_config_load_duration_seconds",
				Help:    "Duration of configuration loading operations",
				Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // 0.1ms to ~400ms
			},
			[]string{"handler_type", "operation"},
		),

		// 설정 캐시 히트 메트릭
		ConfigCacheHits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_config_cache_hits_total",
				Help: "Total number of configuration cache hits",
			},
			[]string{"handler_type", "config_type"},
		),

		// 설정 캐시 미스 메트릭
		ConfigCacheMisses: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_config_cache_misses_total",
				Help: "Total number of configuration cache misses",
			},
			[]string{"handler_type", "config_type"},
		),

		// Container Provider 호출 메트릭
		ContainerProviderCalls: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_container_provider_calls_total",
				Help: "Total number of container provider method calls",
			},
			[]string{"method", "handler_type", "result"},
		),

		// Handler Factory 작업 메트릭
		HandlerFactoryOperations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_handler_factory_operations_total",
				Help: "Total number of handler factory operations",
			},
			[]string{"operation", "handler_type", "result"}, // operation: create, register, get
		),

		// 활성 Handler 인스턴스 메트릭
		HandlerInstancesActive: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_handler_instances_active",
				Help: "Number of active handler instances",
			},
			[]string{"handler_type"},
		),

		// Handler 헬스체크 메트릭
		HandlerHealthChecks: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_handler_health_checks_total",
				Help: "Total number of handler health checks",
			},
			[]string{"handler_type", "result"}, // result: success, failure
		),

		// Handler 설정 리로드 메트릭
		HandlerConfigReloads: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_handler_config_reloads_total",
				Help: "Total number of handler configuration reloads",
			},
			[]string{"handler_type", "result"},
		),

		// Handler 업스트림 URL 빌드 메트릭
		HandlerUpstreamBuilds: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_handler_upstream_builds_total",
				Help: "Total number of upstream URL build operations",
			},
			[]string{"handler_type", "result"},
		),

		// 캐시 키 생성 메트릭
		CacheKeyGenerations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_cache_key_generations_total",
				Help: "Total number of cache key generations",
			},
			[]string{"handler_type"},
		),

		// 캐시 키 생성 시간 메트릭
		CacheKeyGenerationTime: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_cache_key_generation_duration_seconds",
				Help:    "Duration of cache key generation operations",
				Buckets: prometheus.ExponentialBuckets(0.00001, 2, 10), // 0.01ms to ~10ms
			},
			[]string{"handler_type"},
		),
	}
}

// 전역 Container 메트릭 인스턴스
var globalContainerMetrics *ContainerMetrics

// InitContainerMetrics Container 메트릭 초기화
func InitContainerMetrics() {
	globalContainerMetrics = NewContainerMetrics()
}

// GetContainerMetrics 전역 Container 메트릭 인스턴스 반환
func GetContainerMetrics() *ContainerMetrics {
	if globalContainerMetrics == nil {
		InitContainerMetrics()
	}
	return globalContainerMetrics
}

// ResetContainerMetrics Container 메트릭 리셋 (테스트용)
func ResetContainerMetrics() {
	globalContainerMetrics = NewContainerMetrics()
}

// Container Handler 메트릭 헬퍼 메서드들

// RecordHandlerInitialization 핸들러 초기화 기록
func (m *ContainerMetrics) RecordHandlerInitialization(handlerType string, success bool) {
	status := "success" //nolint:goconst
	if !success {
		status = "failure" //nolint:goconst
	}
	m.ContainerHandlerInitializations.WithLabelValues(handlerType, status).Inc()
}

// RecordHandlerRequest 핸들러 요청 기록
func (m *ContainerMetrics) RecordHandlerRequest(handlerType, method string, statusCode int) {
	m.ContainerHandlerRequests.WithLabelValues(handlerType, method, string(rune(statusCode))).Inc()
}

// RecordHandlerDuration 핸들러 처리 시간 기록
func (m *ContainerMetrics) RecordHandlerDuration(handlerType, method string, duration float64) {
	m.ContainerHandlerDuration.WithLabelValues(handlerType, method).Observe(duration)
}

// RecordHandlerError 핸들러 오류 기록
func (m *ContainerMetrics) RecordHandlerError(handlerType, errorType string) {
	m.ContainerHandlerErrors.WithLabelValues(handlerType, errorType).Inc()
}

// RecordConfigLoad 설정 로딩 기록
func (m *ContainerMetrics) RecordConfigLoad(handlerType, operation string, success bool, duration float64) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.ConfigLoadOperations.WithLabelValues(handlerType, operation, result).Inc()
	m.ConfigLoadDuration.WithLabelValues(handlerType, operation).Observe(duration)
}

// RecordConfigCacheHit 설정 캐시 히트 기록
func (m *ContainerMetrics) RecordConfigCacheHit(handlerType, configType string) {
	m.ConfigCacheHits.WithLabelValues(handlerType, configType).Inc()
}

// RecordConfigCacheMiss 설정 캐시 미스 기록
func (m *ContainerMetrics) RecordConfigCacheMiss(handlerType, configType string) {
	m.ConfigCacheMisses.WithLabelValues(handlerType, configType).Inc()
}

// RecordContainerProviderCall Container Provider 호출 기록
func (m *ContainerMetrics) RecordContainerProviderCall(method, handlerType string, success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.ContainerProviderCalls.WithLabelValues(method, handlerType, result).Inc()
}

// RecordHandlerFactoryOperation Handler Factory 작업 기록
func (m *ContainerMetrics) RecordHandlerFactoryOperation(operation, handlerType string, success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.HandlerFactoryOperations.WithLabelValues(operation, handlerType, result).Inc()
}

// SetActiveHandlerInstances 활성 핸들러 인스턴스 수 설정
func (m *ContainerMetrics) SetActiveHandlerInstances(handlerType string, count float64) {
	m.HandlerInstancesActive.WithLabelValues(handlerType).Set(count)
}

// RecordHealthCheck 헬스체크 기록
func (m *ContainerMetrics) RecordHealthCheck(handlerType string, success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.HandlerHealthChecks.WithLabelValues(handlerType, result).Inc()
}

// RecordConfigReload 설정 리로드 기록
func (m *ContainerMetrics) RecordConfigReload(handlerType string, success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.HandlerConfigReloads.WithLabelValues(handlerType, result).Inc()
}

// RecordUpstreamBuild 업스트림 URL 빌드 기록
func (m *ContainerMetrics) RecordUpstreamBuild(handlerType string, success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.HandlerUpstreamBuilds.WithLabelValues(handlerType, result).Inc()
}

// RecordCacheKeyGeneration 캐시 키 생성 기록
func (m *ContainerMetrics) RecordCacheKeyGeneration(handlerType string, duration float64) {
	m.CacheKeyGenerations.WithLabelValues(handlerType).Inc()
	m.CacheKeyGenerationTime.WithLabelValues(handlerType).Observe(duration)
}
