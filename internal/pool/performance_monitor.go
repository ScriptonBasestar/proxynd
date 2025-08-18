package pool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"proxynd/internal/logging"
)

// PerformanceMetrics Connection Pool 성능 메트릭
type PerformanceMetrics struct {
	// 요청 통계
	TotalRequests      int64 `json:"total_requests"`
	SuccessfulRequests int64 `json:"successful_requests"`
	FailedRequests     int64 `json:"failed_requests"`

	// 응답 시간 통계 (나노초)
	TotalResponseTime int64 `json:"total_response_time_ns"`
	MinResponseTime   int64 `json:"min_response_time_ns"`
	MaxResponseTime   int64 `json:"max_response_time_ns"`

	// 연결 재사용 통계
	ConnectionsReused  int64 `json:"connections_reused"`
	ConnectionsCreated int64 `json:"connections_created"`

	// 타임아웃 통계
	TimeoutRequests int64 `json:"timeout_requests"`

	// 대역폭 통계 (바이트)
	TotalBytesReceived int64 `json:"total_bytes_received"`
	TotalBytesSent     int64 `json:"total_bytes_sent"`

	// 시간 관련
	StartTime       time.Time `json:"start_time"`
	LastRequestTime time.Time `json:"last_request_time"`

	// 프록시별 통계
	ProxyMetrics map[string]*ProxyMetrics `json:"proxy_metrics"`
	proxyMutex   sync.RWMutex
}

// ProxyMetrics 프록시별 성능 메트릭
type ProxyMetrics struct {
	ProxyType           string        `json:"proxy_type"`
	RequestCount        int64         `json:"request_count"`
	SuccessCount        int64         `json:"success_count"`
	FailureCount        int64         `json:"failure_count"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	TotalResponseTime   int64         `json:"total_response_time_ns"`
	BytesTransferred    int64         `json:"bytes_transferred"`
	LastUsed            time.Time     `json:"last_used"`
}

// PerformanceMonitor Connection Pool 성능 모니터
type PerformanceMonitor struct {
	metrics *PerformanceMetrics
	logger  logging.Logger

	// 모니터링 설정
	monitoringEnabled bool
	reportingInterval time.Duration
	maxHistoryEntries int

	// 통계 히스토리
	historyMutex   sync.Mutex
	metricsHistory []*PerformanceSnapshot

	// 백그라운드 작업
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// PerformanceSnapshot 성능 스냅샷
type PerformanceSnapshot struct {
	Timestamp           time.Time                `json:"timestamp"`
	TotalRequests       int64                    `json:"total_requests"`
	RequestsPerSecond   float64                  `json:"requests_per_second"`
	AverageResponseTime time.Duration            `json:"average_response_time"`
	SuccessRate         float64                  `json:"success_rate"`
	ProxyBreakdown      map[string]*ProxyMetrics `json:"proxy_breakdown"`
}

// NewPerformanceMonitor 새로운 성능 모니터 생성
func NewPerformanceMonitor() *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &PerformanceMonitor{
		metrics: &PerformanceMetrics{
			StartTime:       time.Now(),
			ProxyMetrics:    make(map[string]*ProxyMetrics),
			MinResponseTime: int64(time.Hour), // 초기값을 크게 설정
		},
		logger:            logging.GetLogger(),
		monitoringEnabled: true,
		reportingInterval: 5 * time.Minute,
		maxHistoryEntries: 288, // 24시간 (5분 간격)
		metricsHistory:    make([]*PerformanceSnapshot, 0),
		ctx:               ctx,
		cancel:            cancel,
		done:              make(chan struct{}),
	}

	// 백그라운드 리포팅 시작
	go monitor.startBackgroundReporting()

	return monitor
}

// RecordRequest 요청 성능 기록
func (m *PerformanceMonitor) RecordRequest(proxyType string, success bool, responseTime time.Duration, bytesReceived, bytesSent int64) { //nolint:lll
	if !m.monitoringEnabled {
		return
	}

	now := time.Now()
	responseTimeNs := responseTime.Nanoseconds()

	// 전체 통계 업데이트
	atomic.AddInt64(&m.metrics.TotalRequests, 1)
	atomic.AddInt64(&m.metrics.TotalResponseTime, responseTimeNs)
	atomic.AddInt64(&m.metrics.TotalBytesReceived, bytesReceived)
	atomic.AddInt64(&m.metrics.TotalBytesSent, bytesSent)
	m.metrics.LastRequestTime = now

	if success {
		atomic.AddInt64(&m.metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&m.metrics.FailedRequests, 1)
	}

	// 최소/최대 응답 시간 업데이트
	for {
		current := atomic.LoadInt64(&m.metrics.MinResponseTime)
		if responseTimeNs >= current || atomic.CompareAndSwapInt64(&m.metrics.MinResponseTime, current, responseTimeNs) {
			break
		}
	}

	for {
		current := atomic.LoadInt64(&m.metrics.MaxResponseTime)
		if responseTimeNs <= current || atomic.CompareAndSwapInt64(&m.metrics.MaxResponseTime, current, responseTimeNs) {
			break
		}
	}

	// 프록시별 통계 업데이트
	m.updateProxyMetrics(proxyType, success, responseTime, bytesReceived+bytesSent, now)
}

// RecordConnectionReuse 연결 재사용 기록
func (m *PerformanceMonitor) RecordConnectionReuse(reused bool) {
	if !m.monitoringEnabled {
		return
	}

	if reused {
		atomic.AddInt64(&m.metrics.ConnectionsReused, 1)
	} else {
		atomic.AddInt64(&m.metrics.ConnectionsCreated, 1)
	}
}

// RecordTimeout 타임아웃 기록
func (m *PerformanceMonitor) RecordTimeout(proxyType string) {
	if !m.monitoringEnabled {
		return
	}

	atomic.AddInt64(&m.metrics.TimeoutRequests, 1)

	// 프록시별 실패 카운트 증가
	m.updateProxyMetrics(proxyType, false, 0, 0, time.Now())
}

// updateProxyMetrics 프록시별 메트릭 업데이트
func (m *PerformanceMonitor) updateProxyMetrics(proxyType string, success bool, responseTime time.Duration, bytesTransferred int64, timestamp time.Time) { //nolint:lll
	m.metrics.proxyMutex.Lock()
	defer m.metrics.proxyMutex.Unlock()

	proxy, exists := m.metrics.ProxyMetrics[proxyType]
	if !exists {
		proxy = &ProxyMetrics{
			ProxyType: proxyType,
		}
		m.metrics.ProxyMetrics[proxyType] = proxy
	}

	atomic.AddInt64(&proxy.RequestCount, 1)
	atomic.AddInt64(&proxy.TotalResponseTime, responseTime.Nanoseconds())
	atomic.AddInt64(&proxy.BytesTransferred, bytesTransferred)
	proxy.LastUsed = timestamp

	if success {
		atomic.AddInt64(&proxy.SuccessCount, 1)
	} else {
		atomic.AddInt64(&proxy.FailureCount, 1)
	}

	// 평균 응답 시간 계산
	if proxy.RequestCount > 0 {
		proxy.AverageResponseTime = time.Duration(proxy.TotalResponseTime / proxy.RequestCount)
	}
}

// GetCurrentMetrics 현재 성능 메트릭 반환
func (m *PerformanceMonitor) GetCurrentMetrics() *PerformanceMetrics {
	// 원자적 연산으로 안전하게 복사
	snapshot := &PerformanceMetrics{
		TotalRequests:      atomic.LoadInt64(&m.metrics.TotalRequests),
		SuccessfulRequests: atomic.LoadInt64(&m.metrics.SuccessfulRequests),
		FailedRequests:     atomic.LoadInt64(&m.metrics.FailedRequests),
		TotalResponseTime:  atomic.LoadInt64(&m.metrics.TotalResponseTime),
		MinResponseTime:    atomic.LoadInt64(&m.metrics.MinResponseTime),
		MaxResponseTime:    atomic.LoadInt64(&m.metrics.MaxResponseTime),
		ConnectionsReused:  atomic.LoadInt64(&m.metrics.ConnectionsReused),
		ConnectionsCreated: atomic.LoadInt64(&m.metrics.ConnectionsCreated),
		TimeoutRequests:    atomic.LoadInt64(&m.metrics.TimeoutRequests),
		TotalBytesReceived: atomic.LoadInt64(&m.metrics.TotalBytesReceived),
		TotalBytesSent:     atomic.LoadInt64(&m.metrics.TotalBytesSent),
		StartTime:          m.metrics.StartTime,
		LastRequestTime:    m.metrics.LastRequestTime,
		ProxyMetrics:       make(map[string]*ProxyMetrics),
	}

	// 프록시별 메트릭 복사
	m.metrics.proxyMutex.RLock()
	for proxyType, proxy := range m.metrics.ProxyMetrics {
		snapshot.ProxyMetrics[proxyType] = &ProxyMetrics{
			ProxyType:           proxy.ProxyType,
			RequestCount:        atomic.LoadInt64(&proxy.RequestCount),
			SuccessCount:        atomic.LoadInt64(&proxy.SuccessCount),
			FailureCount:        atomic.LoadInt64(&proxy.FailureCount),
			AverageResponseTime: proxy.AverageResponseTime,
			TotalResponseTime:   atomic.LoadInt64(&proxy.TotalResponseTime),
			BytesTransferred:    atomic.LoadInt64(&proxy.BytesTransferred),
			LastUsed:            proxy.LastUsed,
		}
	}
	m.metrics.proxyMutex.RUnlock()

	return snapshot
}

// GetPerformanceSnapshot 성능 스냅샷 생성
func (m *PerformanceMonitor) GetPerformanceSnapshot() *PerformanceSnapshot {
	metrics := m.GetCurrentMetrics()
	now := time.Now()

	// 초당 요청 수 계산
	duration := now.Sub(metrics.StartTime).Seconds()
	requestsPerSecond := float64(metrics.TotalRequests) / duration

	// 평균 응답 시간 계산
	var averageResponseTime time.Duration
	if metrics.TotalRequests > 0 {
		averageResponseTime = time.Duration(metrics.TotalResponseTime / metrics.TotalRequests)
	}

	// 성공률 계산
	successRate := float64(0)
	if metrics.TotalRequests > 0 {
		successRate = float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests) * 100
	}

	return &PerformanceSnapshot{
		Timestamp:           now,
		TotalRequests:       metrics.TotalRequests,
		RequestsPerSecond:   requestsPerSecond,
		AverageResponseTime: averageResponseTime,
		SuccessRate:         successRate,
		ProxyBreakdown:      metrics.ProxyMetrics,
	}
}

// GetMetricsHistory 성능 히스토리 반환
func (m *PerformanceMonitor) GetMetricsHistory() []*PerformanceSnapshot {
	m.historyMutex.Lock()
	defer m.historyMutex.Unlock()

	// 복사본 반환
	history := make([]*PerformanceSnapshot, len(m.metricsHistory))
	copy(history, m.metricsHistory)

	return history
}

// startBackgroundReporting 백그라운드 리포팅 시작
func (m *PerformanceMonitor) startBackgroundReporting() {
	defer close(m.done)

	ticker := time.NewTicker(m.reportingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.generatePerformanceReport()
		}
	}
}

// generatePerformanceReport 성능 리포트 생성
func (m *PerformanceMonitor) generatePerformanceReport() {
	snapshot := m.GetPerformanceSnapshot()

	// 히스토리에 추가
	m.historyMutex.Lock()
	m.metricsHistory = append(m.metricsHistory, snapshot)

	// 최대 히스토리 수 제한
	if len(m.metricsHistory) > m.maxHistoryEntries {
		m.metricsHistory = m.metricsHistory[1:]
	}
	m.historyMutex.Unlock()

	// 성능 리포트 로깅
	m.logger.Info("Connection Pool 성능 리포트",
		logging.F("total_requests", snapshot.TotalRequests),
		logging.F("requests_per_second", snapshot.RequestsPerSecond),
		logging.F("average_response_time_ms", snapshot.AverageResponseTime.Milliseconds()),
		logging.F("success_rate_percent", snapshot.SuccessRate),
		logging.F("proxy_types", len(snapshot.ProxyBreakdown)),
	)

	// 프록시별 상세 통계 로깅
	for proxyType, proxyMetrics := range snapshot.ProxyBreakdown {
		if proxyMetrics.RequestCount > 0 {
			m.logger.Debug("프록시별 성능 통계",
				logging.F("proxy_type", proxyType),
				logging.F("request_count", proxyMetrics.RequestCount),
				logging.F("success_count", proxyMetrics.SuccessCount),
				logging.F("average_response_time_ms", proxyMetrics.AverageResponseTime.Milliseconds()),
				logging.F("bytes_transferred", proxyMetrics.BytesTransferred),
			)
		}
	}
}

// Reset 모든 메트릭 리셋
func (m *PerformanceMonitor) Reset() {
	m.metrics = &PerformanceMetrics{
		StartTime:       time.Now(),
		ProxyMetrics:    make(map[string]*ProxyMetrics),
		MinResponseTime: int64(time.Hour),
	}

	m.historyMutex.Lock()
	m.metricsHistory = make([]*PerformanceSnapshot, 0)
	m.historyMutex.Unlock()

	m.logger.Info("Connection Pool 성능 메트릭이 리셋되었습니다")
}

// Stop 성능 모니터 중지
func (m *PerformanceMonitor) Stop() {
	m.cancel()
	<-m.done
	m.logger.Info("Connection Pool 성능 모니터가 중지되었습니다")
}

// 전역 성능 모니터 인스턴스
var globalPerformanceMonitor *PerformanceMonitor

// GetGlobalPerformanceMonitor 전역 성능 모니터 반환
func GetGlobalPerformanceMonitor() *PerformanceMonitor {
	if globalPerformanceMonitor == nil {
		globalPerformanceMonitor = NewPerformanceMonitor()
	}
	return globalPerformanceMonitor
}
