package metrics

import (
	"context"
	"regexp"
	"strconv"
	"sync"
	"time"

	"proxynd/logging"
)

// EnhancedMetricsCollector 강화된 메트릭 수집기
type EnhancedMetricsCollector struct {
	logger  logging.Logger
	metrics *Metrics
	mu      sync.RWMutex

	// 비즈니스 메트릭 추적
	packageStats       map[string]*PackageStatistics
	popularPackages    *PopularityTracker
	userAgentTracker   *UserAgentTracker
	geoLocationTracker *GeographicTracker

	// 성능 메트릭 추적
	latencyTracker    *LatencyTracker
	throughputTracker *ThroughputTracker
	connectionTracker *ConnectionTracker
	resourceTracker   *ResourceTracker

	// 에러 메트릭 추적
	errorTracker    *ErrorTracker
	retryTracker    *RetryTracker
	upstreamTracker *UpstreamTracker

	// 사용자 활동 추적
	userTracker     *UserActivityTracker
	sessionTracker  *SessionTracker
	behaviorTracker *BehaviorTracker

	// 수집 설정
	collectInterval    time.Duration
	retentionPeriod    time.Duration
	maxTrackedPackages int
	maxTrackedUsers    int

	// 제어
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	stopCh  chan struct{}
}

// PackageStatistics 패키지별 통계
type PackageStatistics struct {
	RegistryType   string
	PackageName    string
	TotalDownloads int64
	TotalUploads   int64
	TotalBytes     int64
	LastActivity   time.Time
	Versions       map[string]int64
	FileTypes      map[string]int64
	PopularityRank int
}

// PopularityTracker 인기 패키지 추적기
type PopularityTracker struct {
	mu          sync.RWMutex
	packages    map[string]*PackageStatistics
	rankings    map[string][]string // registry -> package names in order
	lastUpdate  time.Time
	updateCount int64
}

// UserAgentTracker 사용자 에이전트 추적기
type UserAgentTracker struct {
	mu           sync.RWMutex
	agentStats   map[string]*UserAgentStats
	categoryMap  map[string]string
	versionRegex *regexp.Regexp
}

// UserAgentStats 사용자 에이전트 통계
type UserAgentStats struct {
	Category     string
	Version      string
	RequestCount int64
	LastSeen     time.Time
	Registries   map[string]int64
}

// GeographicTracker 지리적 위치 추적기
type GeographicTracker struct {
	mu            sync.RWMutex
	locationStats map[string]*GeographicStats
	ipResolver    IPLocationResolver
}

// GeographicStats 지리적 통계
type GeographicStats struct {
	CountryCode  string
	Region       string
	RequestCount int64
	LastSeen     time.Time
	Registries   map[string]int64
}

// IPLocationResolver IP 주소 위치 해석기 인터페이스
type IPLocationResolver interface {
	ResolveLocation(ip string) (countryCode, region string, err error)
}

// LatencyTracker 지연시간 추적기
type LatencyTracker struct {
	mu           sync.RWMutex
	measurements map[string]*LatencyMeasurements
	buckets      []float64
	maxSamples   int
}

// LatencyMeasurements 지연시간 측정값
type LatencyMeasurements struct {
	RegistryType string
	Method       string
	Samples      []float64
	P50          float64
	P90          float64
	P95          float64
	P99          float64
	LastUpdate   time.Time
}

// ThroughputTracker 처리량 추적기
type ThroughputTracker struct {
	mu           sync.RWMutex
	measurements map[string]*ThroughputMeasurements
	windows      map[string]*TimeWindow
	maxWindows   int
}

// ThroughputMeasurements 처리량 측정값
type ThroughputMeasurements struct {
	RegistryType   string
	RequestsPerSec float64
	Windows        map[string]float64 // time_window -> rps
	LastUpdate     time.Time
}

// TimeWindow 시간 윈도우
type TimeWindow struct {
	Start        time.Time
	End          time.Time
	RequestCount int64
}

// ConnectionTracker 연결 추적기
type ConnectionTracker struct {
	mu          sync.RWMutex
	connections map[string]*ConnectionStats
}

// ConnectionStats 연결 통계
type ConnectionStats struct {
	RegistryType       string
	ConnectionType     string
	CurrentConnections int64
	MaxConnections     int64
	TotalConnections   int64
	LastUpdate         time.Time
}

// ResourceTracker 리소스 추적기
type ResourceTracker struct {
	mu         sync.RWMutex
	resources  map[string]*ResourceStats
	collectors []ResourceCollector
}

// ResourceStats 리소스 통계
type ResourceStats struct {
	ResourceType string
	Measurement  string
	CurrentValue float64
	MaxValue     float64
	AverageValue float64
	LastUpdate   time.Time
}

// ResourceCollector 리소스 수집기 인터페이스
type ResourceCollector interface {
	CollectResources() map[string]float64
	GetResourceType() string
}

// ErrorTracker 에러 추적기
type ErrorTracker struct {
	mu           sync.RWMutex
	errorStats   map[string]*ErrorStats
	distribution map[string]map[string]int64 // registry -> status_code -> count
}

// ErrorStats 에러 통계
type ErrorStats struct {
	RegistryType   string
	StatusClass    string
	StatusCode     string
	Count          int64
	Rate           float64
	LastOccurrence time.Time
}

// RetryTracker 재시도 추적기
type RetryTracker struct {
	mu         sync.RWMutex
	retryStats map[string]*RetryStats
}

// RetryStats 재시도 통계
type RetryStats struct {
	RegistryType  string
	RetryReason   string
	TotalAttempts int64
	SuccessCount  int64
	FailureCount  int64
	SuccessRate   float64
	LastAttempt   time.Time
}

// UpstreamTracker 업스트림 추적기
type UpstreamTracker struct {
	mu            sync.RWMutex
	upstreamStats map[string]*UpstreamStats
}

// UpstreamStats 업스트림 통계
type UpstreamStats struct {
	RegistryType  string
	Upstream      string
	FailureType   string
	TotalRequests int64
	FailureCount  int64
	FailureRate   float64
	LastFailure   time.Time
}

// UserActivityTracker 사용자 활동 추적기
type UserActivityTracker struct {
	mu          sync.RWMutex
	userStats   map[string]*UserStats
	activeUsers map[string]map[string]time.Time // registry -> user_id -> last_seen
}

// UserStats 사용자 통계
type UserStats struct {
	UserID       string
	UserType     string
	RegistryType string
	RequestCount int64
	LastActivity time.Time
	SessionCount int64
}

// SessionTracker 세션 추적기
type SessionTracker struct {
	mu          sync.RWMutex
	sessions    map[string]*SessionInfo
	durations   []float64
	maxSessions int
}

// SessionInfo 세션 정보
type SessionInfo struct {
	SessionID    string
	UserType     string
	RegistryType string
	StartTime    time.Time
	LastActivity time.Time
	RequestCount int64
	IsActive     bool
}

// BehaviorTracker 행동 추적기
type BehaviorTracker struct {
	mu            sync.RWMutex
	behaviorStats map[string]*BehaviorStats
	patterns      map[string]*BehaviorPattern
}

// BehaviorStats 행동 통계
type BehaviorStats struct {
	RegistryType string
	BehaviorType string
	UserCategory string
	EventCount   int64
	LastEvent    time.Time
}

// BehaviorPattern 행동 패턴
type BehaviorPattern struct {
	PatternType  string
	TimePeriod   string
	Frequency    int64
	LastDetected time.Time
}

// EnhancedCollectorConfig 강화된 수집기 설정
type EnhancedCollectorConfig struct {
	CollectInterval    time.Duration
	RetentionPeriod    time.Duration
	MaxTrackedPackages int
	MaxTrackedUsers    int
	EnableGeoLocation  bool
	EnableUserTracking bool
}

// DefaultEnhancedCollectorConfig 기본 설정
func DefaultEnhancedCollectorConfig() *EnhancedCollectorConfig {
	return &EnhancedCollectorConfig{
		CollectInterval:    30 * time.Second,
		RetentionPeriod:    24 * time.Hour,
		MaxTrackedPackages: 10000,
		MaxTrackedUsers:    50000,
		EnableGeoLocation:  false, // 개인정보 보호를 위해 기본값 false
		EnableUserTracking: true,
	}
}

// NewEnhancedMetricsCollector 새로운 강화된 메트릭 수집기 생성
func NewEnhancedMetricsCollector(logger logging.Logger, config *EnhancedCollectorConfig) *EnhancedMetricsCollector {
	if config == nil {
		config = DefaultEnhancedCollectorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	collector := &EnhancedMetricsCollector{
		logger:             logger.WithField("component", "metrics.enhanced_collector"),
		metrics:            GetMetrics(),
		packageStats:       make(map[string]*PackageStatistics),
		popularPackages:    NewPopularityTracker(),
		userAgentTracker:   NewUserAgentTracker(),
		geoLocationTracker: NewGeographicTracker(nil), // nil IPResolver for now
		latencyTracker:     NewLatencyTracker(1000),
		throughputTracker:  NewThroughputTracker(100),
		connectionTracker:  NewConnectionTracker(),
		resourceTracker:    NewResourceTracker(),
		errorTracker:       NewErrorTracker(),
		retryTracker:       NewRetryTracker(),
		upstreamTracker:    NewUpstreamTracker(),
		userTracker:        NewUserActivityTracker(),
		sessionTracker:     NewSessionTracker(10000),
		behaviorTracker:    NewBehaviorTracker(),
		collectInterval:    config.CollectInterval,
		retentionPeriod:    config.RetentionPeriod,
		maxTrackedPackages: config.MaxTrackedPackages,
		maxTrackedUsers:    config.MaxTrackedUsers,
		ctx:                ctx,
		cancel:             cancel,
		stopCh:             make(chan struct{}),
	}

	return collector
}

// Start 수집기 시작
func (ec *EnhancedMetricsCollector) Start() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.running {
		return nil
	}

	ec.logger.Info("Starting enhanced metrics collector")

	// 수집 고루틴 시작
	go ec.collectMetrics()
	go ec.updatePercentiles()
	go ec.cleanupOldData()

	ec.running = true
	ec.logger.Info("Enhanced metrics collector started")

	return nil
}

// Stop 수집기 중지
func (ec *EnhancedMetricsCollector) Stop() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if !ec.running {
		return nil
	}

	ec.logger.Info("Stopping enhanced metrics collector")

	ec.cancel()
	close(ec.stopCh)

	ec.running = false
	ec.logger.Info("Enhanced metrics collector stopped")

	return nil
}

// RecordPackageDownload 패키지 다운로드 기록
func (ec *EnhancedMetricsCollector) RecordPackageDownload(
	registryType string,
	packageName string,
	version string,
	fileType string,
	size int64,
) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// 메트릭 업데이트
	ec.metrics.PackageDownloadsTotal.
		WithLabelValues(registryType, packageName, version, fileType).
		Inc()

	// 내부 통계 업데이트
	key := registryType + ":" + packageName
	if stats, exists := ec.packageStats[key]; exists {
		stats.TotalDownloads++
		stats.TotalBytes += size
		stats.LastActivity = time.Now()
		if stats.Versions == nil {
			stats.Versions = make(map[string]int64)
		}
		stats.Versions[version]++
		if stats.FileTypes == nil {
			stats.FileTypes = make(map[string]int64)
		}
		stats.FileTypes[fileType]++
	} else {
		ec.packageStats[key] = &PackageStatistics{
			RegistryType:   registryType,
			PackageName:    packageName,
			TotalDownloads: 1,
			TotalBytes:     size,
			LastActivity:   time.Now(),
			Versions:       map[string]int64{version: 1},
			FileTypes:      map[string]int64{fileType: 1},
		}
	}

	// 인기도 업데이트
	ec.popularPackages.RecordDownload(registryType, packageName)
}

// RecordPackageUpload 패키지 업로드 기록
func (ec *EnhancedMetricsCollector) RecordPackageUpload(registryType, packageName, version string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// 메트릭 업데이트
	ec.metrics.PackageUploadsTotal.WithLabelValues(registryType, packageName, version).Inc()

	// 내부 통계 업데이트
	key := registryType + ":" + packageName
	if stats, exists := ec.packageStats[key]; exists {
		stats.TotalUploads++
		stats.LastActivity = time.Now()
	}
}

// RecordUserAgent 사용자 에이전트 기록
func (ec *EnhancedMetricsCollector) RecordUserAgent(userAgent, registryType string) {
	category, version := ec.userAgentTracker.ParseUserAgent(userAgent)
	ec.metrics.UserAgentRequests.WithLabelValues(category, version, registryType).Inc()
	ec.userAgentTracker.RecordRequest(userAgent, registryType)
}

// RecordGeographicRequest 지리적 요청 기록
func (ec *EnhancedMetricsCollector) RecordGeographicRequest(ip, registryType string) {
	if ec.geoLocationTracker.ipResolver != nil {
		countryCode, region, err := ec.geoLocationTracker.ipResolver.ResolveLocation(ip)
		if err == nil {
			ec.metrics.GeographicRequests.WithLabelValues(countryCode, region, registryType).Inc()
			ec.geoLocationTracker.RecordRequest(ip, countryCode, region, registryType)
		}
	}
}

// RecordLatency 지연시간 기록
func (ec *EnhancedMetricsCollector) RecordLatency(registryType, method string, latency float64) {
	ec.latencyTracker.RecordLatency(registryType, method, latency)
}

// RecordThroughput 처리량 기록
func (ec *EnhancedMetricsCollector) RecordThroughput(registryType string) {
	ec.throughputTracker.RecordRequest(registryType)
}

// RecordConnection 연결 기록
func (ec *EnhancedMetricsCollector) RecordConnection(registryType, connectionType string, delta int64) {
	ec.connectionTracker.UpdateConnections(registryType, connectionType, delta)
	current := ec.connectionTracker.GetCurrentConnections(registryType, connectionType)
	ec.metrics.ConcurrentConnections.WithLabelValues(registryType, connectionType).Set(float64(current))
}

// RecordHTTPStatus HTTP 상태 기록
func (ec *EnhancedMetricsCollector) RecordHTTPStatus(registryType string, statusCode int) {
	statusClass := getStatusClass(statusCode)
	statusStr := getStatusString(statusCode)

	ec.metrics.HTTPStatusDistribution.WithLabelValues(registryType, statusClass, statusStr).Inc()
	ec.errorTracker.RecordStatus(registryType, statusClass, statusStr)
}

// RecordRetryAttempt 재시도 기록
func (ec *EnhancedMetricsCollector) RecordRetryAttempt(registryType, reason string, attemptNum int, success bool) {
	outcome := resultSuccess
	if !success {
		outcome = resultFailure
	}

	ec.metrics.RetryAttempts.WithLabelValues(registryType, reason, getAttemptString(attemptNum), outcome).Inc()
	ec.retryTracker.RecordAttempt(registryType, reason, success)
}

// RecordTimeout 타임아웃 기록
func (ec *EnhancedMetricsCollector) RecordTimeout(registryType, timeoutType, upstream string) {
	ec.metrics.TimeoutErrors.WithLabelValues(registryType, timeoutType, upstream).Inc()
}

// RecordConnectionError 연결 에러 기록
func (ec *EnhancedMetricsCollector) RecordConnectionError(registryType, errorType, upstream string) {
	ec.metrics.ConnectionErrors.WithLabelValues(registryType, errorType, upstream).Inc()
}

// RecordUpstreamFailure 업스트림 실패 기록
func (ec *EnhancedMetricsCollector) RecordUpstreamFailure(registryType, upstream, failureType string) {
	ec.upstreamTracker.RecordFailure(registryType, upstream, failureType)
}

// RecordUserActivity 사용자 활동 기록
func (ec *EnhancedMetricsCollector) RecordUserActivity(userID, userType, registryType string) {
	ec.userTracker.RecordActivity(userID, userType, registryType)
}

// RecordSessionStart 세션 시작 기록
func (ec *EnhancedMetricsCollector) RecordSessionStart(sessionID, userType, registryType string) {
	ec.sessionTracker.StartSession(sessionID, userType, registryType)
}

// RecordSessionEnd 세션 종료 기록
func (ec *EnhancedMetricsCollector) RecordSessionEnd(sessionID string) {
	duration := ec.sessionTracker.EndSession(sessionID)
	if duration > 0 {
		session := ec.sessionTracker.GetSession(sessionID)
		if session != nil {
			ec.metrics.SessionDuration.WithLabelValues(session.RegistryType, session.UserType).Observe(duration)
		}
	}
}

// RecordUserBehavior 사용자 행동 기록
func (ec *EnhancedMetricsCollector) RecordUserBehavior(registryType, behaviorType, userCategory string) {
	ec.metrics.UserBehaviorMetrics.WithLabelValues(registryType, behaviorType, userCategory).Inc()
	ec.behaviorTracker.RecordBehavior(registryType, behaviorType, userCategory)
}

// RecordAuthenticationEvent 인증 이벤트 기록
func (ec *EnhancedMetricsCollector) RecordAuthenticationEvent(method, eventType, outcome, failureReason string) {
	ec.metrics.AuthenticationMetrics.WithLabelValues(method, eventType, outcome, failureReason).Inc()
}

// collectMetrics 메트릭 수집 고루틴
func (ec *EnhancedMetricsCollector) collectMetrics() {
	ticker := time.NewTicker(ec.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ec.stopCh:
			return
		case <-ticker.C:
			ec.performCollection()
		}
	}
}

// performCollection 수집 수행
func (ec *EnhancedMetricsCollector) performCollection() {
	ec.updatePopularPackages()
	ec.updateThroughputMetrics()
	ec.updateRetryRates()
	ec.updateUpstreamFailureRates()
	ec.updateUniqueUsers()
}

// updatePercentiles 백분위수 업데이트 고루틴
func (ec *EnhancedMetricsCollector) updatePercentiles() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ec.stopCh:
			return
		case <-ticker.C:
			ec.calculateLatencyPercentiles()
		}
	}
}

// cleanupOldData 오래된 데이터 정리 고루틴
func (ec *EnhancedMetricsCollector) cleanupOldData() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ec.stopCh:
			return
		case <-ticker.C:
			ec.performCleanup()
		}
	}
}

// Helper functions
func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 100 && statusCode < 200:
		return "1xx"
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return statusUnknown
	}
}

func getStatusString(statusCode int) string {
	return strconv.Itoa(statusCode)
}

func getAttemptString(attempt int) string {
	if attempt > 5 {
		return "6+"
	}
	return strconv.Itoa(attempt)
}

// 구현 중인 메서드들의 상세 내용은 다음 파일들에서 구현됩니다.
// 이 파일은 기본 구조와 인터페이스를 정의합니다.

// 이후 구현할 메서드들:
// - updatePopularPackages()
// - updateThroughputMetrics()
// - updateRetryRates()
// - updateUpstreamFailureRates()
// - updateUniqueUsers()
// - calculateLatencyPercentiles()
// - performCleanup()
// - 각 트래커들의 New... 생성자 함수들
