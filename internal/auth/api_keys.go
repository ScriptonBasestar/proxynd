package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"proxynd/logging"
)

// APIKey API 키 구조체
type APIKey struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	UserID      string     `json:"user_id"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	IsActive    bool       `json:"is_active"`

	// Rate limiting
	RateLimit      int       `json:"rate_limit"`       // 분당 요청 수
	RateLimitUsed  int       `json:"rate_limit_used"`  // 현재 분 내 사용된 요청 수
	RateLimitReset time.Time `json:"rate_limit_reset"` // 레이트 리밋 리셋 시간

	// IP 제한
	AllowedIPs []string `json:"allowed_ips,omitempty"` // 허용된 IP 목록
	BlockedIPs []string `json:"blocked_ips,omitempty"` // 차단된 IP 목록

	// 사용 통계
	Statistics *APIKeyStats `json:"statistics,omitempty"`
}

// APIKeyStats API 키 사용 통계
type APIKeyStats struct {
	TotalRequests  int64            `json:"total_requests"`
	SuccessfulReqs int64            `json:"successful_requests"`
	FailedReqs     int64            `json:"failed_requests"`
	LastIPs        []string         `json:"last_ips"`
	HourlyUsage    map[string]int64 `json:"hourly_usage"`   // 시간별 사용량
	DailyUsage     map[string]int64 `json:"daily_usage"`    // 일별 사용량
	EndpointUsage  map[string]int64 `json:"endpoint_usage"` // 엔드포인트별 사용량
}

// APIKeyManager API 키 관리자
type APIKeyManager struct {
	keys        map[string]*APIKey
	mutex       sync.RWMutex
	logger      logging.Logger
	storagePath string
	config      *APIKeyConfig
}

// APIKeyConfig API 키 관리 설정
type APIKeyConfig struct {
	StoragePath         string        `json:"storage_path"`
	DefaultRateLimit    int           `json:"default_rate_limit"`   // 기본 분당 요청 수
	DefaultExpiration   time.Duration `json:"default_expiration"`   // 기본 만료 기간
	MaxKeysPerUser      int           `json:"max_keys_per_user"`    // 사용자당 최대 키 수
	KeyLength           int           `json:"key_length"`           // 키 길이 (바이트)
	EnableStatistics    bool          `json:"enable_statistics"`    // 통계 수집 활성화
	StatisticsRetention time.Duration `json:"statistics_retention"` // 통계 보관 기간
	AutoCleanup         bool          `json:"auto_cleanup"`         // 만료된 키 자동 정리
	CleanupInterval     time.Duration `json:"cleanup_interval"`     // 정리 주기
}

// DefaultAPIKeyConfig 기본 API 키 설정
func DefaultAPIKeyConfig() *APIKeyConfig {
	return &APIKeyConfig{
		StoragePath:         "data/api_keys.json",
		DefaultRateLimit:    1000,                // 분당 1000회
		DefaultExpiration:   90 * 24 * time.Hour, // 90일
		MaxKeysPerUser:      10,
		KeyLength:           32, // 32바이트 = 256비트
		EnableStatistics:    true,
		StatisticsRetention: 30 * 24 * time.Hour, // 30일
		AutoCleanup:         true,
		CleanupInterval:     24 * time.Hour, // 하루마다
	}
}

// NewAPIKeyManager API 키 매니저 생성
func NewAPIKeyManager(config *APIKeyConfig) (*APIKeyManager, error) {
	if config == nil {
		config = DefaultAPIKeyConfig()
	}

	manager := &APIKeyManager{
		keys:        make(map[string]*APIKey),
		logger:      logging.GetLogger(),
		storagePath: config.StoragePath,
		config:      config,
	}

	// 저장된 키 로드
	if err := manager.loadKeys(); err != nil {
		manager.logger.Warn("Failed to load existing API keys", logging.F("error", err))
	}

	// 자동 정리 스케줄러 시작
	if config.AutoCleanup {
		go manager.scheduleCleanup()
	}

	return manager, nil
}

// GenerateAPIKey 새 API 키 생성
func (m *APIKeyManager) GenerateAPIKey(userID, name, description string, permissions []string) (*APIKey, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 사용자당 키 수 제한 확인
	userKeyCount := 0
	for _, key := range m.keys {
		if key.UserID == userID && key.IsActive {
			userKeyCount++
		}
	}

	if userKeyCount >= m.config.MaxKeysPerUser {
		return nil, fmt.Errorf("user %s has reached the maximum number of API keys (%d)", userID, m.config.MaxKeysPerUser)
	}

	// 키 ID 생성
	keyID, err := m.generateSecureID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}

	// API 키 생성
	apiKeyValue, err := m.generateSecureKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// 만료 시간 설정
	expiresAt := time.Now().Add(m.config.DefaultExpiration)

	// API 키 객체 생성
	apiKey := &APIKey{
		ID:          keyID,
		Key:         apiKeyValue,
		Name:        name,
		Description: description,
		UserID:      userID,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   &expiresAt,
		UsageCount:  0,
		IsActive:    true,
		RateLimit:   m.config.DefaultRateLimit,
		Statistics: &APIKeyStats{
			HourlyUsage:   make(map[string]int64),
			DailyUsage:    make(map[string]int64),
			EndpointUsage: make(map[string]int64),
		},
	}

	// 저장
	m.keys[keyID] = apiKey

	// 파일에 저장
	if err := m.saveKeys(); err != nil {
		delete(m.keys, keyID)
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}

	m.logger.Info("API key generated",
		logging.F("key_id", keyID),
		logging.F("user_id", userID),
		logging.F("name", name))

	return apiKey, nil
}

// ValidateAPIKey API 키 검증
func (m *APIKeyManager) ValidateAPIKey(keyValue string) (*APIKey, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, apiKey := range m.keys {
		// 키 값 비교 (타이밍 공격 방지)
		if subtle.ConstantTimeCompare([]byte(apiKey.Key), []byte(keyValue)) == 1 {
			// 키 활성화 상태 확인
			if !apiKey.IsActive {
				return nil, fmt.Errorf("API key is deactivated")
			}

			// 만료 시간 확인
			if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
				return nil, fmt.Errorf("API key has expired")
			}

			// 레이트 리밋 확인
			if m.isRateLimited(apiKey) {
				return nil, fmt.Errorf("API key rate limit exceeded")
			}

			return apiKey, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// RecordAPIKeyUsage API 키 사용 기록
func (m *APIKeyManager) RecordAPIKeyUsage(keyID, clientIP, endpoint string, success bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	apiKey, exists := m.keys[keyID]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	now := time.Now()

	// 기본 사용 통계 업데이트
	apiKey.UsageCount++
	apiKey.LastUsedAt = &now

	// 레이트 리밋 업데이트
	m.updateRateLimit(apiKey)

	// 상세 통계 업데이트 (활성화된 경우)
	if m.config.EnableStatistics && apiKey.Statistics != nil {
		apiKey.Statistics.TotalRequests++

		if success {
			apiKey.Statistics.SuccessfulReqs++
		} else {
			apiKey.Statistics.FailedReqs++
		}

		// IP 기록 (최근 10개)
		apiKey.Statistics.LastIPs = m.addToRecentList(apiKey.Statistics.LastIPs, clientIP, 10)

		// 시간별/일별 사용량 기록
		hourKey := now.Format("2006-01-02-15")
		dayKey := now.Format("2006-01-02")
		apiKey.Statistics.HourlyUsage[hourKey]++
		apiKey.Statistics.DailyUsage[dayKey]++

		// 엔드포인트별 사용량 기록
		if endpoint != "" {
			apiKey.Statistics.EndpointUsage[endpoint]++
		}

		// 오래된 통계 데이터 정리
		m.cleanupOldStatistics(apiKey.Statistics)
	}

	// 파일에 저장 (비동기)
	go func() {
		if err := m.saveKeys(); err != nil {
			m.logger.Error("Failed to save API key usage", logging.F("error", err))
		}
	}()

	return nil
}

// RevokeAPIKey API 키 해지
func (m *APIKeyManager) RevokeAPIKey(keyID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	apiKey, exists := m.keys[keyID]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	apiKey.IsActive = false

	if err := m.saveKeys(); err != nil {
		return fmt.Errorf("failed to save API key revocation: %w", err)
	}

	m.logger.Info("API key revoked",
		logging.F("key_id", keyID),
		logging.F("user_id", apiKey.UserID))

	return nil
}

// ListAPIKeys 사용자별 API 키 목록 조회
func (m *APIKeyManager) ListAPIKeys(userID string) ([]*APIKey, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var userKeys []*APIKey
	for _, apiKey := range m.keys {
		if apiKey.UserID == userID {
			// 민감한 정보 제거한 복사본 생성
			keyCopy := *apiKey
			keyCopy.Key = m.maskAPIKey(apiKey.Key)
			userKeys = append(userKeys, &keyCopy)
		}
	}

	return userKeys, nil
}

// GetAPIKeyInfo API 키 정보 조회
func (m *APIKeyManager) GetAPIKeyInfo(keyID string) (*APIKey, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	apiKey, exists := m.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}

	// 민감한 정보 제거한 복사본 반환
	keyCopy := *apiKey
	keyCopy.Key = m.maskAPIKey(apiKey.Key)

	return &keyCopy, nil
}

// UpdateAPIKey API 키 정보 업데이트
func (m *APIKeyManager) UpdateAPIKey(keyID string, updates map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	apiKey, exists := m.keys[keyID]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	// 허용된 필드만 업데이트
	if name, ok := updates["name"].(string); ok {
		apiKey.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		apiKey.Description = description
	}
	if permissions, ok := updates["permissions"].([]string); ok {
		apiKey.Permissions = permissions
	}
	if rateLimit, ok := updates["rate_limit"].(int); ok {
		apiKey.RateLimit = rateLimit
	}
	if allowedIPs, ok := updates["allowed_ips"].([]string); ok {
		apiKey.AllowedIPs = allowedIPs
	}
	if blockedIPs, ok := updates["blocked_ips"].([]string); ok {
		apiKey.BlockedIPs = blockedIPs
	}

	if err := m.saveKeys(); err != nil {
		return fmt.Errorf("failed to save API key updates: %w", err)
	}

	m.logger.Info("API key updated",
		logging.F("key_id", keyID),
		logging.F("user_id", apiKey.UserID))

	return nil
}

// generateSecureKey 보안 키 생성
func (m *APIKeyManager) generateSecureKey() (string, error) {
	bytes := make([]byte, m.config.KeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateSecureID 보안 ID 생성
func (m *APIKeyManager) generateSecureID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// isRateLimited 레이트 리밋 확인
func (m *APIKeyManager) isRateLimited(apiKey *APIKey) bool {
	now := time.Now()

	// 새로운 분이면 카운터 리셋
	if now.After(apiKey.RateLimitReset) {
		apiKey.RateLimitUsed = 0
		apiKey.RateLimitReset = now.Add(time.Minute)
	}

	return apiKey.RateLimitUsed >= apiKey.RateLimit
}

// updateRateLimit 레이트 리밋 업데이트
func (m *APIKeyManager) updateRateLimit(apiKey *APIKey) {
	now := time.Now()

	// 새로운 분이면 카운터 리셋
	if now.After(apiKey.RateLimitReset) {
		apiKey.RateLimitUsed = 0
		apiKey.RateLimitReset = now.Add(time.Minute)
	}

	apiKey.RateLimitUsed++
}

// addToRecentList 최근 목록에 항목 추가
func (m *APIKeyManager) addToRecentList(list []string, item string, maxSize int) []string {
	// 중복 제거
	for i, existing := range list {
		if existing == item {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}

	// 맨 앞에 추가
	list = append([]string{item}, list...)

	// 크기 제한
	if len(list) > maxSize {
		list = list[:maxSize]
	}

	return list
}

// cleanupOldStatistics 오래된 통계 데이터 정리
func (m *APIKeyManager) cleanupOldStatistics(stats *APIKeyStats) {
	cutoff := time.Now().Add(-m.config.StatisticsRetention)
	cutoffHour := cutoff.Format("2006-01-02-15")
	cutoffDay := cutoff.Format("2006-01-02")

	// 오래된 시간별 데이터 제거
	for hour := range stats.HourlyUsage {
		if hour < cutoffHour {
			delete(stats.HourlyUsage, hour)
		}
	}

	// 오래된 일별 데이터 제거
	for day := range stats.DailyUsage {
		if day < cutoffDay {
			delete(stats.DailyUsage, day)
		}
	}
}

// maskAPIKey API 키 마스킹
func (m *APIKeyManager) maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// scheduleCleanup 정기 정리 스케줄러
func (m *APIKeyManager) scheduleCleanup() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupExpiredKeys()
	}
}

// cleanupExpiredKeys 만료된 키 정리
func (m *APIKeyManager) cleanupExpiredKeys() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	cleanedCount := 0

	for keyID, apiKey := range m.keys {
		if apiKey.ExpiresAt != nil && now.After(*apiKey.ExpiresAt) {
			delete(m.keys, keyID)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		m.logger.Info("Cleaned up expired API keys",
			logging.F("count", cleanedCount))

		if err := m.saveKeys(); err != nil {
			m.logger.Error("Failed to save after cleanup", logging.F("error", err))
		}
	}
}

// loadKeys 저장된 키 로드
func (m *APIKeyManager) loadKeys() error {
	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(m.storagePath), 0o755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 파일이 없으면 무시
	if _, err := os.Stat(m.storagePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(m.storagePath)
	if err != nil {
		return fmt.Errorf("failed to read API keys file: %w", err)
	}

	var keys map[string]*APIKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("failed to parse API keys file: %w", err)
	}

	m.keys = keys

	m.logger.Info("Loaded API keys", logging.F("count", len(keys)))
	return nil
}

// saveKeys 키 저장
func (m *APIKeyManager) saveKeys() error {
	data, err := json.MarshalIndent(m.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API keys: %w", err)
	}

	// 임시 파일에 쓰고 원자적으로 이동
	tempPath := m.storagePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write API keys file: %w", err)
	}

	if err := os.Rename(tempPath, m.storagePath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to move API keys file: %w", err)
	}

	return nil
}

// GetGlobalStatistics 전체 통계 조회
func (m *APIKeyManager) GetGlobalStatistics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_keys":     0,
		"active_keys":    0,
		"expired_keys":   0,
		"total_usage":    int64(0),
		"users":          make(map[string]int),
		"daily_usage":    make(map[string]int64),
		"endpoint_usage": make(map[string]int64),
	}

	now := time.Now()
	users := make(map[string]int)
	dailyUsage := make(map[string]int64)
	endpointUsage := make(map[string]int64)

	for _, apiKey := range m.keys {
		stats["total_keys"] = stats["total_keys"].(int) + 1

		if apiKey.IsActive {
			if apiKey.ExpiresAt == nil || now.Before(*apiKey.ExpiresAt) {
				stats["active_keys"] = stats["active_keys"].(int) + 1
			} else {
				stats["expired_keys"] = stats["expired_keys"].(int) + 1
			}
		}

		stats["total_usage"] = stats["total_usage"].(int64) + apiKey.UsageCount
		users[apiKey.UserID]++

		// 통계 집계
		if apiKey.Statistics != nil {
			for day, usage := range apiKey.Statistics.DailyUsage {
				dailyUsage[day] += usage
			}
			for endpoint, usage := range apiKey.Statistics.EndpointUsage {
				endpointUsage[endpoint] += usage
			}
		}
	}

	stats["users"] = users
	stats["daily_usage"] = dailyUsage
	stats["endpoint_usage"] = endpointUsage

	return stats
}
