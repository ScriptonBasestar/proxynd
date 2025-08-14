package metrics

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// UserActivityCollector 사용자 활동 메트릭 수집기
type UserActivityCollector struct {
	// Prometheus 메트릭
	activeUsers           prometheus.Gauge
	userRequests          *prometheus.CounterVec
	userBandwidth         *prometheus.CounterVec
	userSessionDuration   *prometheus.HistogramVec
	uniqueUsersDaily      prometheus.Gauge
	topUsersByRequests    *prometheus.GaugeVec
	userGeolocation       *prometheus.CounterVec
	userErrors            *prometheus.CounterVec

	// 내부 상태
	activeSessions map[string]*UserSession
	dailyUsers     map[string]time.Time
	mutex          sync.RWMutex
	
	// 설정
	cleanupInterval time.Duration
	sessionTimeout  time.Duration
}

// UserSession 사용자 세션 정보
type UserSession struct {
	UserID        string
	IPAddress     string
	UserAgent     string
	Country       string
	StartTime     time.Time
	LastActivity  time.Time
	RequestCount  int64
	BytesDownload int64
	BytesUpload   int64
	ProxyTypes    map[string]int64
}

// NewUserActivityCollector 새 사용자 활동 수집기 생성
func NewUserActivityCollector() *UserActivityCollector {
	collector := &UserActivityCollector{
		// Prometheus 메트릭 초기화
		activeUsers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "proxynd_active_users_total",
			Help: "Number of currently active users",
		}),
		
		userRequests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxynd_user_requests_total",
			Help: "Total number of requests by user",
		}, []string{"user_id", "proxy_type", "status"}),
		
		userBandwidth: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxynd_user_bandwidth_bytes_total",
			Help: "Total bandwidth usage by user",
		}, []string{"user_id", "proxy_type", "direction"}),
		
		userSessionDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "proxynd_user_session_duration_seconds",
			Help:    "User session duration in seconds",
			Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1분부터 68시간까지
		}, []string{"user_id"}),
		
		uniqueUsersDaily: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "proxynd_unique_users_daily",
			Help: "Number of unique users in the last 24 hours",
		}),
		
		topUsersByRequests: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "proxynd_top_users_requests",
			Help: "Top users by request count",
		}, []string{"user_id", "rank"}),
		
		userGeolocation: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxynd_user_geolocation_total",
			Help: "User requests by geographic location",
		}, []string{"country", "proxy_type"}),
		
		userErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxynd_user_errors_total",
			Help: "User errors by type and proxy",
		}, []string{"user_id", "proxy_type", "error_type"}),

		// 내부 상태 초기화
		activeSessions:  make(map[string]*UserSession),
		dailyUsers:     make(map[string]time.Time),
		cleanupInterval: 5 * time.Minute,
		sessionTimeout:  30 * time.Minute,
	}

	// 백그라운드 정리 작업 시작
	go collector.startCleanupWorker()
	
	return collector
}

// RecordUserRequest 사용자 요청 기록
func (uac *UserActivityCollector) RecordUserRequest(userID, ipAddress, userAgent, proxyType, status string, responseSize int64) {
	uac.mutex.Lock()
	defer uac.mutex.Unlock()

	// 세션 가져오기 또는 생성
	session := uac.getOrCreateSession(userID, ipAddress, userAgent)
	
	// 요청 카운트 증가
	session.RequestCount++
	session.LastActivity = time.Now()
	session.ProxyTypes[proxyType]++
	
	// Prometheus 메트릭 업데이트
	uac.userRequests.WithLabelValues(userID, proxyType, status).Inc()
	
	// 대역폭 기록 (다운로드)
	if responseSize > 0 {
		session.BytesDownload += responseSize
		uac.userBandwidth.WithLabelValues(userID, proxyType, "download").Add(float64(responseSize))
	}
	
	// 지리적 위치 기록
	country := uac.getCountryFromIP(ipAddress)
	session.Country = country
	uac.userGeolocation.WithLabelValues(country, proxyType).Inc()
	
	// 오늘 사용자 기록
	today := time.Now().Format("2006-01-02")
	uac.dailyUsers[userID+":"+today] = time.Now()
}

// RecordUserError 사용자 오류 기록
func (uac *UserActivityCollector) RecordUserError(userID, proxyType, errorType string) {
	uac.userErrors.WithLabelValues(userID, proxyType, errorType).Inc()
}

// RecordUserUpload 사용자 업로드 대역폭 기록
func (uac *UserActivityCollector) RecordUserUpload(userID, proxyType string, uploadSize int64) {
	uac.mutex.Lock()
	defer uac.mutex.Unlock()
	
	if session, exists := uac.activeSessions[userID]; exists {
		session.BytesUpload += uploadSize
		uac.userBandwidth.WithLabelValues(userID, proxyType, "upload").Add(float64(uploadSize))
	}
}

// getOrCreateSession 세션 가져오기 또는 생성
func (uac *UserActivityCollector) getOrCreateSession(userID, ipAddress, userAgent string) *UserSession {
	if session, exists := uac.activeSessions[userID]; exists {
		return session
	}
	
	// 새 세션 생성
	session := &UserSession{
		UserID:       userID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		ProxyTypes:   make(map[string]int64),
	}
	
	uac.activeSessions[userID] = session
	return session
}

// getCountryFromIP IP 주소에서 국가 정보 추출 (간단한 구현)
func (uac *UserActivityCollector) getCountryFromIP(ipAddress string) string {
	// 로컬 IP 범위 확인
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return "unknown"
	}
	
	// 로컬 IP 범위
	if ip.IsLoopback() || ip.IsPrivate() {
		return "local"
	}
	
	// 실제 구현에서는 GeoIP 데이터베이스를 사용
	// 여기서는 간단한 예시만 제공
	switch {
	case ip.To4() != nil:
		// IPv4 기반 간단한 분류
		firstOctet := ip.To4()[0]
		switch {
		case firstOctet >= 1 && firstOctet <= 126:
			return "US" // 예시
		case firstOctet >= 128 && firstOctet <= 191:
			return "EU" // 예시
		default:
			return "other"
		}
	default:
		return "unknown"
	}
}

// startCleanupWorker 정리 작업 시작
func (uac *UserActivityCollector) startCleanupWorker() {
	ticker := time.NewTicker(uac.cleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		uac.cleanup()
	}
}

// cleanup 만료된 세션 및 데이터 정리
func (uac *UserActivityCollector) cleanup() {
	uac.mutex.Lock()
	defer uac.mutex.Unlock()
	
	now := time.Now()
	activeCount := 0
	
	// 만료된 세션 제거
	for userID, session := range uac.activeSessions {
		if now.Sub(session.LastActivity) > uac.sessionTimeout {
			// 세션 종료 메트릭 기록
			sessionDuration := session.LastActivity.Sub(session.StartTime).Seconds()
			uac.userSessionDuration.WithLabelValues(userID).Observe(sessionDuration)
			
			delete(uac.activeSessions, userID)
		} else {
			activeCount++
		}
	}
	
	// 활성 사용자 수 업데이트
	uac.activeUsers.Set(float64(activeCount))
	
	// 오래된 일일 사용자 데이터 정리 (7일 이상)
	cutoff := now.AddDate(0, 0, -7)
	uniqueToday := 0
	today := now.Format("2006-01-02")
	
	for key, timestamp := range uac.dailyUsers {
		if timestamp.Before(cutoff) {
			delete(uac.dailyUsers, key)
		} else if strings.HasSuffix(key, ":"+today) {
			uniqueToday++
		}
	}
	
	// 오늘의 고유 사용자 수 업데이트
	uac.uniqueUsersDaily.Set(float64(uniqueToday))
}

// GetActiveUsers 활성 사용자 목록 반환
func (uac *UserActivityCollector) GetActiveUsers() map[string]*UserSession {
	uac.mutex.RLock()
	defer uac.mutex.RUnlock()
	
	// 복사본 반환
	result := make(map[string]*UserSession)
	for k, v := range uac.activeSessions {
		sessionCopy := *v
		sessionCopy.ProxyTypes = make(map[string]int64)
		for pt, count := range v.ProxyTypes {
			sessionCopy.ProxyTypes[pt] = count
		}
		result[k] = &sessionCopy
	}
	
	return result
}

// GetUserStats 사용자 통계 반환
func (uac *UserActivityCollector) GetUserStats(userID string) (*UserSession, bool) {
	uac.mutex.RLock()
	defer uac.mutex.RUnlock()
	
	session, exists := uac.activeSessions[userID]
	if !exists {
		return nil, false
	}
	
	// 복사본 반환
	sessionCopy := *session
	sessionCopy.ProxyTypes = make(map[string]int64)
	for pt, count := range session.ProxyTypes {
		sessionCopy.ProxyTypes[pt] = count
	}
	
	return &sessionCopy, true
}

// UpdateTopUsers 상위 사용자 메트릭 업데이트
func (uac *UserActivityCollector) UpdateTopUsers() {
	uac.mutex.RLock()
	defer uac.mutex.RUnlock()
	
	// 요청 수 기준으로 정렬
	type userStat struct {
		userID   string
		requests int64
	}
	
	var users []userStat
	for userID, session := range uac.activeSessions {
		users = append(users, userStat{
			userID:   userID,
			requests: session.RequestCount,
		})
	}
	
	// 상위 10명만 기록
	maxUsers := 10
	if len(users) > maxUsers {
		// 간단한 정렬 (실제로는 더 효율적인 방법 사용)
		for i := 0; i < maxUsers; i++ {
			maxIdx := i
			for j := i + 1; j < len(users); j++ {
				if users[j].requests > users[maxIdx].requests {
					maxIdx = j
				}
			}
			if maxIdx != i {
				users[i], users[maxIdx] = users[maxIdx], users[i]
			}
		}
		users = users[:maxUsers]
	}
	
	// 메트릭 업데이트
	for rank, user := range users {
		uac.topUsersByRequests.WithLabelValues(user.userID, string(rune(rank+1))).Set(float64(user.requests))
	}
}

// GetDailyUserCount 일일 고유 사용자 수 반환
func (uac *UserActivityCollector) GetDailyUserCount() int {
	uac.mutex.RLock()
	defer uac.mutex.RUnlock()
	
	today := time.Now().Format("2006-01-02")
	count := 0
	
	for key := range uac.dailyUsers {
		if strings.HasSuffix(key, ":"+today) {
			count++
		}
	}
	
	return count
}

// GetUserGeolocationStats 지리적 분포 통계 반환
func (uac *UserActivityCollector) GetUserGeolocationStats() map[string]int {
	uac.mutex.RLock()
	defer uac.mutex.RUnlock()
	
	stats := make(map[string]int)
	for _, session := range uac.activeSessions {
		if session.Country != "" {
			stats[session.Country]++
		}
	}
	
	return stats
}

// GetProxyTypeUsage 프록시 타입별 사용량 통계 반환
func (uac *UserActivityCollector) GetProxyTypeUsage() map[string]int64 {
	uac.mutex.RLock()
	defer uac.mutex.RUnlock()
	
	usage := make(map[string]int64)
	for _, session := range uac.activeSessions {
		for proxyType, count := range session.ProxyTypes {
			usage[proxyType] += count
		}
	}
	
	return usage
}

// Shutdown 수집기 종료
func (uac *UserActivityCollector) Shutdown(ctx context.Context) error {
	uac.mutex.Lock()
	defer uac.mutex.Unlock()
	
	// 모든 활성 세션에 대해 세션 종료 메트릭 기록
	for userID, session := range uac.activeSessions {
		sessionDuration := time.Now().Sub(session.StartTime).Seconds()
		uac.userSessionDuration.WithLabelValues(userID).Observe(sessionDuration)
	}
	
	return nil
}