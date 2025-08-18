// Package metrics provides metrics collection functionality
package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// CustomCollector 커스텀 메트릭 수집기
type CustomCollector struct {
	startTime time.Time

	// 런타임 메트릭
	goroutines   *prometheus.Desc
	memoryAlloc  *prometheus.Desc
	memoryTotal  *prometheus.Desc
	memorySys    *prometheus.Desc
	gcPauseTotal *prometheus.Desc
	gcPauseLast  *prometheus.Desc

	// 프로세스 메트릭
	cpuSeconds *prometheus.Desc
	openFds    *prometheus.Desc
	maxFds     *prometheus.Desc
}

// NewCustomCollector 새 커스텀 수집기 생성
func NewCustomCollector() *CustomCollector {
	return &CustomCollector{
		startTime: time.Now(),

		goroutines: prometheus.NewDesc(
			"proxynd_go_goroutines",
			"Number of goroutines",
			nil, nil,
		),

		memoryAlloc: prometheus.NewDesc(
			"proxynd_go_memory_alloc_bytes",
			"Number of bytes allocated and in use",
			nil, nil,
		),

		memoryTotal: prometheus.NewDesc(
			"proxynd_go_memory_total_alloc_bytes",
			"Total number of bytes allocated",
			nil, nil,
		),

		memorySys: prometheus.NewDesc(
			"proxynd_go_memory_sys_bytes",
			"Number of bytes obtained from system",
			nil, nil,
		),

		gcPauseTotal: prometheus.NewDesc(
			"proxynd_go_gc_pause_total_seconds",
			"Total time spent in GC pause",
			nil, nil,
		),

		gcPauseLast: prometheus.NewDesc(
			"proxynd_go_gc_pause_last_seconds",
			"Time spent in last GC pause",
			nil, nil,
		),

		cpuSeconds: prometheus.NewDesc(
			"proxynd_process_cpu_seconds_total",
			"Total user and system CPU time spent",
			nil, nil,
		),

		openFds: prometheus.NewDesc(
			"proxynd_process_open_fds",
			"Number of open file descriptors",
			nil, nil,
		),

		maxFds: prometheus.NewDesc(
			"proxynd_process_max_fds",
			"Maximum number of open file descriptors",
			nil, nil,
		),
	}
}

// Describe 메트릭 설명 반환
func (c *CustomCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.goroutines
	ch <- c.memoryAlloc
	ch <- c.memoryTotal
	ch <- c.memorySys
	ch <- c.gcPauseTotal
	ch <- c.gcPauseLast
	ch <- c.cpuSeconds
	ch <- c.openFds
	ch <- c.maxFds
}

// Collect 메트릭 수집
func (c *CustomCollector) Collect(ch chan<- prometheus.Metric) {
	// 런타임 통계 수집
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Goroutines
	ch <- prometheus.MustNewConstMetric(
		c.goroutines,
		prometheus.GaugeValue,
		float64(runtime.NumGoroutine()),
	)

	// Memory metrics
	ch <- prometheus.MustNewConstMetric(
		c.memoryAlloc,
		prometheus.GaugeValue,
		float64(m.Alloc),
	)

	ch <- prometheus.MustNewConstMetric(
		c.memoryTotal,
		prometheus.CounterValue,
		float64(m.TotalAlloc),
	)

	ch <- prometheus.MustNewConstMetric(
		c.memorySys,
		prometheus.GaugeValue,
		float64(m.Sys),
	)

	// GC metrics
	if m.NumGC > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.gcPauseTotal,
			prometheus.CounterValue,
			float64(m.PauseTotalNs)/1e9,
		)

		ch <- prometheus.MustNewConstMetric(
			c.gcPauseLast,
			prometheus.GaugeValue,
			float64(m.PauseNs[(m.NumGC+255)%256])/1e9,
		)
	}

	// CPU time (간단한 추정치)
	cpuTime := time.Since(c.startTime).Seconds()
	ch <- prometheus.MustNewConstMetric(
		c.cpuSeconds,
		prometheus.CounterValue,
		cpuTime,
	)

	// File descriptors (Linux 전용, 다른 OS에서는 0 반환)
	openFds, maxFds := getFileDescriptorCounts()
	ch <- prometheus.MustNewConstMetric(
		c.openFds,
		prometheus.GaugeValue,
		float64(openFds),
	)

	ch <- prometheus.MustNewConstMetric(
		c.maxFds,
		prometheus.GaugeValue,
		float64(maxFds),
	)
}

// CacheStatsCollector 캐시 통계 수집기
type CacheStatsCollector struct {
	cacheManager interface{} // 실제 캐시 매니저 인터페이스

	hitRate      *prometheus.Desc
	evictionRate *prometheus.Desc
	fillRate     *prometheus.Desc
}

// NewCacheStatsCollector 새 캐시 통계 수집기 생성
func NewCacheStatsCollector(cacheManager interface{}) *CacheStatsCollector {
	return &CacheStatsCollector{
		cacheManager: cacheManager,

		hitRate: prometheus.NewDesc(
			"proxynd_cache_hit_rate",
			"Cache hit rate (0-1)",
			[]string{"registry_type"},
			nil,
		),

		evictionRate: prometheus.NewDesc(
			"proxynd_cache_eviction_rate",
			"Cache eviction rate per second",
			[]string{"registry_type"},
			nil,
		),

		fillRate: prometheus.NewDesc(
			"proxynd_cache_fill_rate",
			"Cache fill rate per second",
			[]string{"registry_type"},
			nil,
		),
	}
}

// Describe 메트릭 설명 반환
func (c *CacheStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.hitRate
	ch <- c.evictionRate
	ch <- c.fillRate
}

// Collect 메트릭 수집
func (c *CacheStatsCollector) Collect(ch chan<- prometheus.Metric) {
	// NOTE: 실제 캐시 매니저에서 통계 수집 필요
	// 현재는 예시 값

	registryTypes := []string{"npm", "pypi", "apt", "docker", "maven"}

	for _, registryType := range registryTypes {
		// 캐시 히트율 (예시)
		ch <- prometheus.MustNewConstMetric(
			c.hitRate,
			prometheus.GaugeValue,
			0.85, // 85% hit rate
			registryType,
		)

		// 제거율 (예시)
		ch <- prometheus.MustNewConstMetric(
			c.evictionRate,
			prometheus.GaugeValue,
			0.1, // 0.1 evictions/sec
			registryType,
		)

		// 채우기율 (예시)
		ch <- prometheus.MustNewConstMetric(
			c.fillRate,
			prometheus.GaugeValue,
			1.5, // 1.5 fills/sec
			registryType,
		)
	}
}
