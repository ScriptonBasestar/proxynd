package metrics

import (
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// NewPopularityTracker 인기도 추적기 생성
func NewPopularityTracker() *PopularityTracker {
	return &PopularityTracker{
		packages:   make(map[string]*PackageStatistics),
		rankings:   make(map[string][]string),
		lastUpdate: time.Now(),
	}
}

// RecordDownload 다운로드 기록
func (pt *PopularityTracker) RecordDownload(registryType, packageName string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := registryType + ":" + packageName
	if stats, exists := pt.packages[key]; exists {
		stats.TotalDownloads++
		stats.LastActivity = time.Now()
	} else {
		pt.packages[key] = &PackageStatistics{
			RegistryType:   registryType,
			PackageName:    packageName,
			TotalDownloads: 1,
			LastActivity:   time.Now(),
			Versions:       make(map[string]int64),
			FileTypes:      make(map[string]int64),
		}
	}
	pt.updateCount++
}

// UpdateRankings 순위 업데이트
func (pt *PopularityTracker) UpdateRankings() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	registryPackages := make(map[string][]*PackageStatistics)

	// 레지스트리별로 패키지 분류
	for _, stats := range pt.packages {
		registryPackages[stats.RegistryType] = append(registryPackages[stats.RegistryType], stats)
	}

	// 각 레지스트리별로 순위 계산
	for registryType, packages := range registryPackages {
		// 다운로드 수로 정렬
		sort.Slice(packages, func(i, j int) bool {
			return packages[i].TotalDownloads > packages[j].TotalDownloads
		})

		// 순위 업데이트 및 상위 100개만 유지
		maxRank := 100
		if len(packages) < maxRank {
			maxRank = len(packages)
		}

		rankings := make([]string, maxRank)
		for i := 0; i < maxRank; i++ {
			packages[i].PopularityRank = i + 1
			rankings[i] = packages[i].PackageName
		}
		pt.rankings[registryType] = rankings
	}

	pt.lastUpdate = time.Now()
}

// GetTopPackages 상위 패키지 목록 반환
func (pt *PopularityTracker) GetTopPackages(registryType string, limit int) []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if rankings, exists := pt.rankings[registryType]; exists {
		if len(rankings) < limit {
			limit = len(rankings)
		}
		result := make([]string, limit)
		copy(result, rankings[:limit])
		return result
	}
	return nil
}

// NewUserAgentTracker 사용자 에이전트 추적기 생성
func NewUserAgentTracker() *UserAgentTracker {
	return &UserAgentTracker{
		agentStats:   make(map[string]*UserAgentStats),
		categoryMap:  buildUserAgentCategoryMap(),
		versionRegex: regexp.MustCompile(`([0-9]+(?:\.[0-9]+)*)`),
	}
}

// buildUserAgentCategoryMap 사용자 에이전트 카테고리 맵 구축
func buildUserAgentCategoryMap() map[string]string {
	return map[string]string{
		"npm":        "npm_cli",
		"yarn":       "yarn_cli", 
		"pnpm":       "pnpm_cli",
		"pip":        "pip_cli",
		"conda":      "conda_cli",
		"poetry":     "poetry_cli",
		"maven":      "maven_cli",
		"gradle":     "gradle_cli",
		"docker":     "docker_cli",
		"curl":       "curl",
		"wget":       "wget",
		"python":     "python",
		"go":         "go_cli",
		"rust":       "rust_cli",
		"node":       "nodejs",
		"chrome":     "browser_chrome",
		"firefox":    "browser_firefox",
		"safari":     "browser_safari",
		"postman":    "api_client",
		"insomnia":   "api_client",
		"jenkins":    "ci_jenkins",
		"github":     "ci_github",
		"gitlab":     "ci_gitlab",
		"circleci":   "ci_circleci",
		"travis":     "ci_travis",
		"azure":      "ci_azure",
	}
}

// ParseUserAgent 사용자 에이전트 파싱
func (uat *UserAgentTracker) ParseUserAgent(userAgent string) (category, version string) {
	userAgent = strings.ToLower(userAgent)
	
	// 카테고리 결정
	category = "other"
	for keyword, cat := range uat.categoryMap {
		if strings.Contains(userAgent, keyword) {
			category = cat
			break
		}
	}

	// 버전 추출
	version = "unknown"
	if matches := uat.versionRegex.FindStringSubmatch(userAgent); len(matches) > 1 {
		version = matches[1]
		// 버전이 너무 상세한 경우 간소화 (예: 1.2.3.4 -> 1.2)
		parts := strings.Split(version, ".")
		if len(parts) > 2 {
			version = strings.Join(parts[:2], ".")
		}
	}

	return category, version
}

// RecordRequest 요청 기록
func (uat *UserAgentTracker) RecordRequest(userAgent, registryType string) {
	uat.mu.Lock()
	defer uat.mu.Unlock()

	category, version := uat.ParseUserAgent(userAgent)
	key := category + ":" + version

	if stats, exists := uat.agentStats[key]; exists {
		stats.RequestCount++
		stats.LastSeen = time.Now()
		if stats.Registries == nil {
			stats.Registries = make(map[string]int64)
		}
		stats.Registries[registryType]++
	} else {
		uat.agentStats[key] = &UserAgentStats{
			Category:     category,
			Version:      version,
			RequestCount: 1,
			LastSeen:     time.Now(),
			Registries:   map[string]int64{registryType: 1},
		}
	}
}

// NewGeographicTracker 지리적 추적기 생성
func NewGeographicTracker(resolver IPLocationResolver) *GeographicTracker {
	return &GeographicTracker{
		locationStats: make(map[string]*GeographicStats),
		ipResolver:    resolver,
	}
}

// RecordRequest 요청 기록
func (gt *GeographicTracker) RecordRequest(ip, countryCode, region, registryType string) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	key := countryCode + ":" + region

	if stats, exists := gt.locationStats[key]; exists {
		stats.RequestCount++
		stats.LastSeen = time.Now()
		if stats.Registries == nil {
			stats.Registries = make(map[string]int64)
		}
		stats.Registries[registryType]++
	} else {
		gt.locationStats[key] = &GeographicStats{
			CountryCode:  countryCode,
			Region:       region,
			RequestCount: 1,
			LastSeen:     time.Now(),
			Registries:   map[string]int64{registryType: 1},
		}
	}
}

// SimpleIPLocationResolver 간단한 IP 위치 해석기 (개발용)
type SimpleIPLocationResolver struct {
	// 실제 구현에서는 GeoIP 데이터베이스나 외부 서비스 사용
}

// ResolveLocation IP 주소 위치 해석
func (resolver *SimpleIPLocationResolver) ResolveLocation(ip string) (countryCode, region string, err error) {
	// 개발/테스트용 간단한 구현
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "unknown", "unknown", nil
	}

	// RFC 1918 사설 IP 주소 체크
	if isPrivateIP(parsedIP) {
		return "private", "private", nil
	}

	// 로컬호스트 체크
	if parsedIP.IsLoopback() {
		return "local", "local", nil
	}

	// 실제 구현에서는 여기서 GeoIP 데이터베이스 조회
	return "unknown", "unknown", nil
}

// isPrivateIP 사설 IP 확인
func isPrivateIP(ip net.IP) bool {
	private := false
	_, private24BitBlock, _ := net.ParseCIDR("10.0.0.0/8")
	_, private20BitBlock, _ := net.ParseCIDR("172.16.0.0/12")
	_, private16BitBlock, _ := net.ParseCIDR("192.168.0.0/16")
	private = private24BitBlock.Contains(ip) || private20BitBlock.Contains(ip) || private16BitBlock.Contains(ip)
	return private
}

// NewLatencyTracker 지연시간 추적기 생성
func NewLatencyTracker(maxSamples int) *LatencyTracker {
	return &LatencyTracker{
		measurements: make(map[string]*LatencyMeasurements),
		buckets:      []float64{0.1, 0.2, 0.5, 1, 2, 5, 10}, // 백분위수 계산용 버킷
		maxSamples:   maxSamples,
	}
}

// RecordLatency 지연시간 기록
func (lt *LatencyTracker) RecordLatency(registryType, method string, latency float64) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := registryType + ":" + method

	if measurement, exists := lt.measurements[key]; exists {
		// 샘플 수가 최대치를 초과하면 오래된 것 제거
		if len(measurement.Samples) >= lt.maxSamples {
			measurement.Samples = measurement.Samples[1:]
		}
		measurement.Samples = append(measurement.Samples, latency)
		measurement.LastUpdate = time.Now()
	} else {
		lt.measurements[key] = &LatencyMeasurements{
			RegistryType: registryType,
			Method:       method,
			Samples:      []float64{latency},
			LastUpdate:   time.Now(),
		}
	}
}

// CalculatePercentiles 백분위수 계산
func (lt *LatencyTracker) CalculatePercentiles(registryType, method string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := registryType + ":" + method

	if measurement, exists := lt.measurements[key]; exists && len(measurement.Samples) > 0 {
		samples := make([]float64, len(measurement.Samples))
		copy(samples, measurement.Samples)
		sort.Float64s(samples)

		measurement.P50 = calculatePercentileRefined(samples, 0.50)
		measurement.P90 = calculatePercentileRefined(samples, 0.90)
		measurement.P95 = calculatePercentileRefined(samples, 0.95)
		measurement.P99 = calculatePercentileRefined(samples, 0.99)
	}
}

// GetPercentiles 백분위수 조회
func (lt *LatencyTracker) GetPercentiles(registryType, method string) (p50, p90, p95, p99 float64) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	key := registryType + ":" + method
	if measurement, exists := lt.measurements[key]; exists {
		return measurement.P50, measurement.P90, measurement.P95, measurement.P99
	}
	return 0, 0, 0, 0
}

// NewThroughputTracker 처리량 추적기 생성
func NewThroughputTracker(maxWindows int) *ThroughputTracker {
	return &ThroughputTracker{
		measurements: make(map[string]*ThroughputMeasurements),
		windows:      make(map[string]*TimeWindow),
		maxWindows:   maxWindows,
	}
}

// RecordRequest 요청 기록
func (tt *ThroughputTracker) RecordRequest(registryType string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	now := time.Now()
	windowKey := registryType + ":" + now.Format("2006-01-02T15:04") // 1분 단위 윈도우

	if window, exists := tt.windows[windowKey]; exists {
		window.RequestCount++
		window.End = now
	} else {
		tt.windows[windowKey] = &TimeWindow{
			Start:        now.Truncate(time.Minute),
			End:          now,
			RequestCount: 1,
		}
	}
}

// CalculateThroughput 처리량 계산
func (tt *ThroughputTracker) CalculateThroughput(registryType string) float64 {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	now := time.Now()
	totalRequests := int64(0)
	windowCount := 0

	// 최근 5분간의 요청 수 집계
	for i := 0; i < 5; i++ {
		windowTime := now.Add(-time.Duration(i) * time.Minute)
		windowKey := registryType + ":" + windowTime.Format("2006-01-02T15:04")
		
		if window, exists := tt.windows[windowKey]; exists {
			totalRequests += window.RequestCount
			windowCount++
		}
	}

	if windowCount > 0 {
		return float64(totalRequests) / float64(windowCount) / 60.0 // RPS 계산
	}
	return 0
}

// NewConnectionTracker 연결 추적기 생성
func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{
		connections: make(map[string]*ConnectionStats),
	}
}

// UpdateConnections 연결 수 업데이트
func (ct *ConnectionTracker) UpdateConnections(registryType, connectionType string, delta int64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	key := registryType + ":" + connectionType

	if stats, exists := ct.connections[key]; exists {
		stats.CurrentConnections += delta
		stats.TotalConnections++
		if stats.CurrentConnections > stats.MaxConnections {
			stats.MaxConnections = stats.CurrentConnections
		}
		if stats.CurrentConnections < 0 {
			stats.CurrentConnections = 0
		}
		stats.LastUpdate = time.Now()
	} else {
		currentConn := delta
		if currentConn < 0 {
			currentConn = 0
		}
		ct.connections[key] = &ConnectionStats{
			RegistryType:       registryType,
			ConnectionType:     connectionType,
			CurrentConnections: currentConn,
			MaxConnections:     currentConn,
			TotalConnections:   1,
			LastUpdate:        time.Now(),
		}
	}
}

// GetCurrentConnections 현재 연결 수 조회
func (ct *ConnectionTracker) GetCurrentConnections(registryType, connectionType string) int64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	key := registryType + ":" + connectionType
	if stats, exists := ct.connections[key]; exists {
		return stats.CurrentConnections
	}
	return 0
}

// NewResourceTracker 리소스 추적기 생성
func NewResourceTracker() *ResourceTracker {
	return &ResourceTracker{
		resources:  make(map[string]*ResourceStats),
		collectors: []ResourceCollector{},
	}
}

// AddCollector 리소스 수집기 추가
func (rt *ResourceTracker) AddCollector(collector ResourceCollector) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.collectors = append(rt.collectors, collector)
}

// CollectAll 모든 리소스 수집
func (rt *ResourceTracker) CollectAll() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for _, collector := range rt.collectors {
		resources := collector.CollectResources()
		resourceType := collector.GetResourceType()

		for measurement, value := range resources {
			key := resourceType + ":" + measurement

			if stats, exists := rt.resources[key]; exists {
				stats.CurrentValue = value
				if value > stats.MaxValue {
					stats.MaxValue = value
				}
				// 간단한 이동평균 계산
				stats.AverageValue = (stats.AverageValue + value) / 2
				stats.LastUpdate = time.Now()
			} else {
				rt.resources[key] = &ResourceStats{
					ResourceType: resourceType,
					Measurement:  measurement,
					CurrentValue: value,
					MaxValue:     value,
					AverageValue: value,
					LastUpdate:   time.Now(),
				}
			}
		}
	}
}

// SystemResourceCollector 시스템 리소스 수집기
type SystemResourceCollector struct{}

// CollectResources 시스템 리소스 수집
func (src *SystemResourceCollector) CollectResources() map[string]float64 {
	// GetSystemMetrics 함수 재사용
	return GetSystemMetrics()
}

// GetResourceType 리소스 타입 반환
func (src *SystemResourceCollector) GetResourceType() string {
	return "system"
}

// NewErrorTracker 에러 추적기 생성
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		errorStats:   make(map[string]*ErrorStats),
		distribution: make(map[string]map[string]int64),
	}
}

// RecordStatus 상태 기록
func (et *ErrorTracker) RecordStatus(registryType, statusClass, statusCode string) {
	et.mu.Lock()
	defer et.mu.Unlock()

	key := registryType + ":" + statusClass + ":" + statusCode

	if stats, exists := et.errorStats[key]; exists {
		stats.Count++
		stats.LastOccurrence = time.Now()
	} else {
		et.errorStats[key] = &ErrorStats{
			RegistryType:   registryType,
			StatusClass:    statusClass,
			StatusCode:     statusCode,
			Count:          1,
			LastOccurrence: time.Now(),
		}
	}

	// 분포 업데이트
	if et.distribution[registryType] == nil {
		et.distribution[registryType] = make(map[string]int64)
	}
	et.distribution[registryType][statusCode]++
}

// NewRetryTracker 재시도 추적기 생성
func NewRetryTracker() *RetryTracker {
	return &RetryTracker{
		retryStats: make(map[string]*RetryStats),
	}
}

// RecordAttempt 시도 기록
func (rt *RetryTracker) RecordAttempt(registryType, reason string, success bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	key := registryType + ":" + reason

	if stats, exists := rt.retryStats[key]; exists {
		stats.TotalAttempts++
		if success {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalAttempts)
		stats.LastAttempt = time.Now()
	} else {
		successCount := int64(0)
		failureCount := int64(0)
		if success {
			successCount = 1
		} else {
			failureCount = 1
		}

		rt.retryStats[key] = &RetryStats{
			RegistryType:  registryType,
			RetryReason:   reason,
			TotalAttempts: 1,
			SuccessCount:  successCount,
			FailureCount:  failureCount,
			SuccessRate:   float64(successCount),
			LastAttempt:   time.Now(),
		}
	}
}

// GetSuccessRate 성공률 조회
func (rt *RetryTracker) GetSuccessRate(registryType, reason string) float64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	key := registryType + ":" + reason
	if stats, exists := rt.retryStats[key]; exists {
		return stats.SuccessRate
	}
	return 0
}

// NewUpstreamTracker 업스트림 추적기 생성
func NewUpstreamTracker() *UpstreamTracker {
	return &UpstreamTracker{
		upstreamStats: make(map[string]*UpstreamStats),
	}
}

// RecordFailure 실패 기록
func (ut *UpstreamTracker) RecordFailure(registryType, upstream, failureType string) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	key := registryType + ":" + upstream + ":" + failureType

	if stats, exists := ut.upstreamStats[key]; exists {
		stats.FailureCount++
		stats.LastFailure = time.Now()
		// 실패율 계산은 별도로 수행
	} else {
		ut.upstreamStats[key] = &UpstreamStats{
			RegistryType: registryType,
			Upstream:     upstream,
			FailureType:  failureType,
			FailureCount: 1,
			LastFailure:  time.Now(),
		}
	}
}

// RecordRequest 요청 기록 (성공/실패 모두)
func (ut *UpstreamTracker) RecordRequest(registryType, upstream string, success bool) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	// 모든 failure type에 대해 총 요청 수 업데이트
	for key, stats := range ut.upstreamStats {
		parts := strings.Split(key, ":")
		if len(parts) >= 2 && parts[0] == registryType && parts[1] == upstream {
			stats.TotalRequests++
			if stats.TotalRequests > 0 {
				stats.FailureRate = float64(stats.FailureCount) / float64(stats.TotalRequests)
			}
		}
	}
}

// GetFailureRate 실패율 조회
func (ut *UpstreamTracker) GetFailureRate(registryType, upstream, failureType string) float64 {
	ut.mu.RLock()
	defer ut.mu.RUnlock()

	key := registryType + ":" + upstream + ":" + failureType
	if stats, exists := ut.upstreamStats[key]; exists {
		return stats.FailureRate
	}
	return 0
}

// NewUserActivityTracker 사용자 활동 추적기 생성
func NewUserActivityTracker() *UserActivityTracker {
	return &UserActivityTracker{
		userStats:   make(map[string]*UserStats),
		activeUsers: make(map[string]map[string]time.Time),
	}
}

// RecordActivity 활동 기록
func (uat *UserActivityTracker) RecordActivity(userID, userType, registryType string) {
	uat.mu.Lock()
	defer uat.mu.Unlock()

	key := registryType + ":" + userID

	if stats, exists := uat.userStats[key]; exists {
		stats.RequestCount++
		stats.LastActivity = time.Now()
	} else {
		uat.userStats[key] = &UserStats{
			UserID:       userID,
			UserType:     userType,
			RegistryType: registryType,
			RequestCount: 1,
			LastActivity: time.Now(),
		}
	}

	// 활성 사용자 업데이트
	if uat.activeUsers[registryType] == nil {
		uat.activeUsers[registryType] = make(map[string]time.Time)
	}
	uat.activeUsers[registryType][userID] = time.Now()
}

// GetActiveUserCount 활성 사용자 수 조회
func (uat *UserActivityTracker) GetActiveUserCount(registryType, timeWindow string) int64 {
	uat.mu.RLock()
	defer uat.mu.RUnlock()

	if users, exists := uat.activeUsers[registryType]; exists {
		now := time.Now()
		var cutoff time.Time

		switch timeWindow {
		case "1h":
			cutoff = now.Add(-time.Hour)
		case "24h":
			cutoff = now.Add(-24 * time.Hour)
		case "7d":
			cutoff = now.Add(-7 * 24 * time.Hour)
		default:
			cutoff = now.Add(-time.Hour)
		}

		count := int64(0)
		for _, lastSeen := range users {
			if lastSeen.After(cutoff) {
				count++
			}
		}
		return count
	}
	return 0
}

// NewSessionTracker 세션 추적기 생성
func NewSessionTracker(maxSessions int) *SessionTracker {
	return &SessionTracker{
		sessions:    make(map[string]*SessionInfo),
		durations:   make([]float64, 0),
		maxSessions: maxSessions,
	}
}

// StartSession 세션 시작
func (st *SessionTracker) StartSession(sessionID, userType, registryType string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.sessions[sessionID] = &SessionInfo{
		SessionID:    sessionID,
		UserType:     userType,
		RegistryType: registryType,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		RequestCount: 0,
		IsActive:     true,
	}
}

// EndSession 세션 종료
func (st *SessionTracker) EndSession(sessionID string) float64 {
	st.mu.Lock()
	defer st.mu.Unlock()

	if session, exists := st.sessions[sessionID]; exists && session.IsActive {
		duration := time.Since(session.StartTime).Seconds()
		session.IsActive = false
		
		// 세션 지속시간 기록
		if len(st.durations) >= st.maxSessions {
			st.durations = st.durations[1:]
		}
		st.durations = append(st.durations, duration)
		
		return duration
	}
	return 0
}

// GetSession 세션 정보 조회
func (st *SessionTracker) GetSession(sessionID string) *SessionInfo {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	return st.sessions[sessionID]
}

// NewBehaviorTracker 행동 추적기 생성
func NewBehaviorTracker() *BehaviorTracker {
	return &BehaviorTracker{
		behaviorStats: make(map[string]*BehaviorStats),
		patterns:      make(map[string]*BehaviorPattern),
	}
}

// RecordBehavior 행동 기록
func (bt *BehaviorTracker) RecordBehavior(registryType, behaviorType, userCategory string) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	key := registryType + ":" + behaviorType + ":" + userCategory

	if stats, exists := bt.behaviorStats[key]; exists {
		stats.EventCount++
		stats.LastEvent = time.Now()
	} else {
		bt.behaviorStats[key] = &BehaviorStats{
			RegistryType: registryType,
			BehaviorType: behaviorType,
			UserCategory: userCategory,
			EventCount:   1,
			LastEvent:    time.Now(),
		}
	}
}

// Helper function for percentile calculation (refined version)
func calculatePercentileRefined(sortedData []float64, percentile float64) float64 {
	if len(sortedData) == 0 {
		return 0
	}

	if len(sortedData) == 1 {
		return sortedData[0]
	}

	// Linear interpolation method
	rank := percentile * float64(len(sortedData)-1)
	lowerIndex := int(rank)
	upperIndex := lowerIndex + 1

	if upperIndex >= len(sortedData) {
		return sortedData[len(sortedData)-1]
	}

	if lowerIndex == upperIndex {
		return sortedData[lowerIndex]
	}

	// Interpolate between the two values
	weight := rank - float64(lowerIndex)
	return sortedData[lowerIndex]*(1-weight) + sortedData[upperIndex]*weight
}

// Helper function to convert status code to string
func statusCodeToString(statusCode int) string {
	return strconv.Itoa(statusCode)
}