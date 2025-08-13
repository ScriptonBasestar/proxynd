package apt

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/domain/apt"
	"proxynd/logging"
)

// mirrorManagerImpl 미러 관리 서비스 구현
type mirrorManagerImpl struct {
	config      apt.ProxyConfig
	logger      logging.Logger
	mirrorStats map[string][]*apt.MirrorStatus // osType -> []*MirrorStatus
	statsMutex  sync.RWMutex
	roundRobin  map[string]*int32 // osType -> atomic counter for round-robin
	rrMutex     sync.RWMutex
}

// NewMirrorManager MirrorManager 생성자
func NewMirrorManager(config apt.ProxyConfig, logger logging.Logger) apt.MirrorManager {
	manager := &mirrorManagerImpl{
		config:      config,
		logger:      logger,
		mirrorStats: make(map[string][]*apt.MirrorStatus),
		roundRobin:  make(map[string]*int32),
	}

	// 초기화
	manager.initializeMirrors()

	// 주기적인 건강 상태 확인 시작
	go manager.startHealthChecker()

	return manager
}

// GetNextMirror 라운드로빈으로 다음 미러 선택
func (m *mirrorManagerImpl) GetNextMirror(osType string) (*apt.MirrorStatus, error) {
	m.statsMutex.RLock()
	mirrors, exists := m.mirrorStats[osType]
	m.statsMutex.RUnlock()

	if !exists || len(mirrors) == 0 {
		return nil, fmt.Errorf("no mirrors configured for OS type: %s", osType)
	}

	// 사용 가능한 미러만 필터링
	var availableMirrors []*apt.MirrorStatus
	for _, mirror := range mirrors {
		if mirror.Available {
			availableMirrors = append(availableMirrors, mirror)
		}
	}

	if len(availableMirrors) == 0 {
		return nil, fmt.Errorf("no available mirrors for OS type: %s", osType)
	}

	// 라운드로빈 선택
	m.rrMutex.Lock()
	if _, exists := m.roundRobin[osType]; !exists {
		counter := int32(0)
		m.roundRobin[osType] = &counter
	}
	counter := m.roundRobin[osType]
	m.rrMutex.Unlock()

	index := atomic.AddInt32(counter, 1) % int32(len(availableMirrors))
	selectedMirror := availableMirrors[index]

	m.logger.Debug("Selected mirror via round-robin",
		logging.F("osType", osType),
		logging.F("mirrorURL", selectedMirror.URL),
		logging.F("index", index),
		logging.F("totalAvailable", len(availableMirrors)),
	)

	return selectedMirror, nil
}

// CheckMirrorHealth 미러 상태 확인
func (m *mirrorManagerImpl) CheckMirrorHealth(ctx context.Context, mirror *apt.MirrorStatus) error {
	startTime := time.Now()

	// HEAD 요청으로 미러 상태 확인
	agent := fiber.Head(mirror.URL)
	agent.Set("User-Agent", "ProxyND/1.0 APT-Proxy Health Check")
	agent.Timeout(10 * time.Second)

	statusCode, _, errs := agent.Bytes()

	mirror.LastCheck = time.Now()

	if len(errs) > 0 {
		mirror.Available = false
		mirror.Error = errs[0].Error()
		m.logger.Warn("Mirror health check failed",
			logging.F("mirrorURL", mirror.URL),
			logging.F("error", errs[0]),
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
		return errs[0]
	}

	if statusCode >= 200 && statusCode < 400 {
		mirror.Available = true
		mirror.Error = ""
		m.logger.Debug("Mirror health check passed",
			logging.F("mirrorURL", mirror.URL),
			logging.F("statusCode", statusCode),
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	} else {
		mirror.Available = false
		mirror.Error = fmt.Sprintf("HTTP %d", statusCode)
		m.logger.Warn("Mirror health check failed with bad status",
			logging.F("mirrorURL", mirror.URL),
			logging.F("statusCode", statusCode),
		)
	}

	return nil
}

// GetMirrorStats 미러 통계 조회
func (m *mirrorManagerImpl) GetMirrorStats(osType string) ([]*apt.MirrorStatus, error) {
	m.statsMutex.RLock()
	mirrors, exists := m.mirrorStats[osType]
	m.statsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no mirrors configured for OS type: %s", osType)
	}

	// 복사본 반환 (동시성 안전)
	result := make([]*apt.MirrorStatus, len(mirrors))
	copy(result, mirrors)

	return result, nil
}

// MarkMirrorFailed 미러를 실패로 표시
func (m *mirrorManagerImpl) MarkMirrorFailed(osType, mirrorURL string, err error) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()

	mirrors, exists := m.mirrorStats[osType]
	if !exists {
		return
	}

	for _, mirror := range mirrors {
		if mirror.URL == mirrorURL {
			mirror.Available = false
			mirror.Error = err.Error()
			mirror.LastCheck = time.Now()

			m.logger.Warn("Marked mirror as failed",
				logging.F("osType", osType),
				logging.F("mirrorURL", mirrorURL),
				logging.F("error", err),
			)
			break
		}
	}
}

// initializeMirrors 미러 상태 초기화
func (m *mirrorManagerImpl) initializeMirrors() {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()

	osTypes := m.config.GetAllOSTypes()
	for _, osType := range osTypes {
		mirrors := m.config.GetMirrors(osType)
		var mirrorStatuses []*apt.MirrorStatus

		for _, mirror := range mirrors {
			status := &apt.MirrorStatus{
				Name:      mirror.Name,
				URL:       mirror.URL,
				Available: true, // 초기값: 사용 가능
				LastCheck: time.Now(),
			}
			mirrorStatuses = append(mirrorStatuses, status)
		}

		m.mirrorStats[osType] = mirrorStatuses

		m.logger.Info("Initialized mirrors for OS type",
			logging.F("osType", osType),
			logging.F("mirrorCount", len(mirrorStatuses)),
		)
	}
}

// startHealthChecker 주기적인 미러 건강 상태 확인 시작
func (m *mirrorManagerImpl) startHealthChecker() {
	ticker := time.NewTicker(5 * time.Minute) // 5분마다 체크
	defer ticker.Stop()

	m.logger.Info("Started mirror health checker")

	//nolint:staticcheck // S1000: Infinite loop is intended for background worker
	for {
		select {
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck 모든 미러의 건강 상태 확인 수행
func (m *mirrorManagerImpl) performHealthCheck() {
	ctx := context.Background()

	m.statsMutex.RLock()
	allMirrors := make(map[string][]*apt.MirrorStatus)
	for osType, mirrors := range m.mirrorStats {
		allMirrors[osType] = mirrors
	}
	m.statsMutex.RUnlock()

	var wg sync.WaitGroup
	for osType, mirrors := range allMirrors {
		for _, mirror := range mirrors {
			wg.Add(1)
			go func(osType string, mirror *apt.MirrorStatus) {
				defer wg.Done()
				_ = m.CheckMirrorHealth(ctx, mirror)
			}(osType, mirror)
		}
	}

	wg.Wait()

	// 통계 로깅
	totalMirrors := 0
	availableMirrors := 0
	for osType, mirrors := range allMirrors {
		available := 0
		for _, mirror := range mirrors {
			if mirror.Available {
				available++
			}
		}
		totalMirrors += len(mirrors)
		availableMirrors += available

		m.logger.Debug("Mirror health check completed for OS type",
			logging.F("osType", osType),
			logging.F("available", available),
			logging.F("total", len(mirrors)),
		)
	}

	m.logger.Info("Global mirror health check completed",
		logging.F("totalMirrors", totalMirrors),
		logging.F("availableMirrors", availableMirrors),
		logging.F("healthRate", float64(availableMirrors)/float64(totalMirrors)*100),
	)
}
