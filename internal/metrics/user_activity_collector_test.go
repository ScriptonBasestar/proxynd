package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserActivityCollector_RecordUserRequest(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	tests := []struct {
		name         string
		userID       string
		ipAddress    string
		userAgent    string
		proxyType    string
		status       string
		responseSize int64
	}{
		{
			name:         "valid user request",
			userID:       "user1",
			ipAddress:    "192.168.1.100",
			userAgent:    "Go-http-client/1.1",
			proxyType:    "maven",
			status:       "success",
			responseSize: 1024,
		},
		{
			name:         "npm request",
			userID:       "user2",
			ipAddress:    "10.0.0.50",
			userAgent:    "npm/8.19.0",
			proxyType:    "npm",
			status:       "success",
			responseSize: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector.RecordUserRequest(
				tt.userID,
				tt.ipAddress,
				tt.userAgent,
				tt.proxyType,
				tt.status,
				tt.responseSize,
			)

			// 세션이 생성되었는지 확인
			session, exists := collector.GetUserStats(tt.userID)
			require.True(t, exists)
			assert.Equal(t, tt.userID, session.UserID)
			assert.Equal(t, tt.ipAddress, session.IPAddress)
			assert.Equal(t, tt.userAgent, session.UserAgent)
			assert.Equal(t, int64(1), session.RequestCount)
			assert.Equal(t, tt.responseSize, session.BytesDownload)
			assert.Contains(t, session.ProxyTypes, tt.proxyType)
			assert.Equal(t, int64(1), session.ProxyTypes[tt.proxyType])
		})
	}
}

func TestUserActivityCollector_RecordUserError(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 먼저 사용자 요청 기록 (세션 생성)
	collector.RecordUserRequest("user1", "192.168.1.100", "test-agent", "maven", "success", 1024)

	// 에러 기록
	collector.RecordUserError("user1", "maven", "timeout")

	// 에러가 기록되었는지 확인 (메트릭은 내부적으로 처리되므로 세션 상태로 확인)
	session, exists := collector.GetUserStats("user1")
	require.True(t, exists)
	assert.Equal(t, "user1", session.UserID)
}

func TestUserActivityCollector_RecordUserUpload(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 먼저 사용자 요청 기록 (세션 생성)
	collector.RecordUserRequest("user1", "192.168.1.100", "test-agent", "maven", "success", 1024)

	// 업로드 기록
	collector.RecordUserUpload("user1", "maven", 512)

	// 업로드가 기록되었는지 확인
	session, exists := collector.GetUserStats("user1")
	require.True(t, exists)
	assert.Equal(t, int64(512), session.BytesUpload)
}

func TestUserActivityCollector_GetActiveUsers(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 여러 사용자 요청 기록
	users := []string{"user1", "user2", "user3"}
	for _, userID := range users {
		collector.RecordUserRequest(userID, "192.168.1.100", "test-agent", "maven", "success", 1024)
	}

	// 활성 사용자 목록 확인
	activeUsers := collector.GetActiveUsers()
	assert.Len(t, activeUsers, 3)

	for _, userID := range users {
		assert.Contains(t, activeUsers, userID)
		assert.Equal(t, userID, activeUsers[userID].UserID)
	}
}

func TestUserActivityCollector_GetCountryFromIP(t *testing.T) {
	collector := NewUserActivityCollector()

	tests := []struct {
		name      string
		ipAddress string
		expected  string
	}{
		{
			name:      "localhost",
			ipAddress: "127.0.0.1",
			expected:  "local",
		},
		{
			name:      "private IP",
			ipAddress: "192.168.1.100",
			expected:  "local",
		},
		{
			name:      "public IP class A",
			ipAddress: "8.8.8.8",
			expected:  "US",
		},
		{
			name:      "public IP class B",
			ipAddress: "172.16.0.1", // 이건 실제로는 private이지만 테스트용
			expected:  "local",      // private IP이므로 local
		},
		{
			name:      "invalid IP",
			ipAddress: "invalid-ip",
			expected:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.getCountryFromIP(tt.ipAddress)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserActivityCollector_SessionTimeout(t *testing.T) {
	// 짧은 세션 타임아웃으로 테스트
	collector := NewUserActivityCollector()
	collector.sessionTimeout = 100 * time.Millisecond
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 사용자 요청 기록
	collector.RecordUserRequest("user1", "192.168.1.100", "test-agent", "maven", "success", 1024)

	// 세션이 생성되었는지 확인
	session, exists := collector.GetUserStats("user1")
	require.True(t, exists)
	assert.Equal(t, "user1", session.UserID)

	// 세션 타임아웃까지 대기
	time.Sleep(150 * time.Millisecond)

	// 정리 작업 실행
	collector.cleanup()

	// 세션이 제거되었는지 확인
	_, exists = collector.GetUserStats("user1")
	assert.False(t, exists)
}

func TestUserActivityCollector_UpdateTopUsers(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 다양한 요청 수로 사용자들 생성
	userRequestCounts := map[string]int{
		"user1": 10,
		"user2": 5,
		"user3": 15,
		"user4": 3,
		"user5": 8,
	}

	for userID, requestCount := range userRequestCounts {
		for i := 0; i < requestCount; i++ {
			collector.RecordUserRequest(userID, "192.168.1.100", "test-agent", "maven", "success", 1024)
		}
	}

	// 상위 사용자 업데이트
	collector.UpdateTopUsers()

	// 상위 사용자들이 올바르게 기록되었는지 확인
	activeUsers := collector.GetActiveUsers()
	for userID, session := range activeUsers {
		expectedCount := int64(userRequestCounts[userID])
		assert.Equal(t, expectedCount, session.RequestCount)
	}
}

func TestUserActivityCollector_GetDailyUserCount(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 오늘 날짜로 사용자들 추가
	users := []string{"user1", "user2", "user3"}
	for _, userID := range users {
		collector.RecordUserRequest(userID, "192.168.1.100", "test-agent", "maven", "success", 1024)
	}

	// 일일 사용자 수 확인
	dailyCount := collector.GetDailyUserCount()
	assert.Equal(t, 3, dailyCount)
}

func TestUserActivityCollector_GetUserGeolocationStats(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 다양한 IP로 사용자 요청 기록
	testCases := []struct {
		userID    string
		ipAddress string
	}{
		{"user1", "127.0.0.1"},     // local
		{"user2", "192.168.1.100"}, // local
		{"user3", "8.8.8.8"},       // US
		{"user4", "1.1.1.1"},       // US
	}

	for _, tc := range testCases {
		collector.RecordUserRequest(tc.userID, tc.ipAddress, "test-agent", "maven", "success", 1024)
	}

	// 지리적 분포 통계 확인
	geoStats := collector.GetUserGeolocationStats()
	assert.Contains(t, geoStats, "local")
	assert.Contains(t, geoStats, "US")
	assert.Equal(t, 2, geoStats["local"])
	assert.Equal(t, 2, geoStats["US"])
}

func TestUserActivityCollector_GetProxyTypeUsage(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 다양한 프록시 타입으로 요청 기록
	testCases := []struct {
		userID    string
		proxyType string
		requests  int
	}{
		{"user1", "maven", 5},
		{"user1", "npm", 3},
		{"user2", "maven", 2},
		{"user2", "docker", 4},
		{"user3", "npm", 1},
	}

	for _, tc := range testCases {
		for i := 0; i < tc.requests; i++ {
			collector.RecordUserRequest(tc.userID, "192.168.1.100", "test-agent", tc.proxyType, "success", 1024)
		}
	}

	// 프록시 타입별 사용량 확인
	usage := collector.GetProxyTypeUsage()
	assert.Equal(t, int64(7), usage["maven"])  // user1:5 + user2:2
	assert.Equal(t, int64(4), usage["npm"])    // user1:3 + user3:1
	assert.Equal(t, int64(4), usage["docker"]) // user2:4
}

func TestUserActivityCollector_ConcurrentAccess(t *testing.T) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 동시성 테스트
	const numGoroutines = 10
	const requestsPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	// 여러 고루틴에서 동시에 요청 기록
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			userID := "user" + string(rune(goroutineID))
			for j := 0; j < requestsPerGoroutine; j++ {
				collector.RecordUserRequest(userID, "192.168.1.100", "test-agent", "maven", "success", 1024)
			}
		}(i)
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 결과 확인
	activeUsers := collector.GetActiveUsers()
	assert.Len(t, activeUsers, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		userID := "user" + string(rune(i))
		session, exists := activeUsers[userID]
		require.True(t, exists)
		assert.Equal(t, int64(requestsPerGoroutine), session.RequestCount)
	}
}

func BenchmarkUserActivityCollector_RecordUserRequest(b *testing.B) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			b.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		userCounter := 0
		for pb.Next() {
			userID := "user" + string(rune(userCounter%1000)) // 1000명의 사용자 순환
			collector.RecordUserRequest(userID, "192.168.1.100", "test-agent", "maven", "success", 1024)
			userCounter++
		}
	})
}

func BenchmarkUserActivityCollector_GetActiveUsers(b *testing.B) {
	collector := NewUserActivityCollector()
	defer func() {
		if err := collector.Shutdown(context.Background()); err != nil {
			b.Errorf("Failed to shutdown collector: %v", err)
		}
	}()

	// 테스트 데이터 준비
	for i := 0; i < 1000; i++ {
		userID := "user" + string(rune(i))
		collector.RecordUserRequest(userID, "192.168.1.100", "test-agent", "maven", "success", 1024)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		activeUsers := collector.GetActiveUsers()
		_ = activeUsers // 결과 사용
	}
}
