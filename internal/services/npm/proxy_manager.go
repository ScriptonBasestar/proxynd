package npm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"proxynd/internal/domain/npm"
	"proxynd/internal/logging"
)

// proxyManagerImpl 프록시 서버 관리 서비스 구현
type proxyManagerImpl struct {
	config     npm.ProxyConfig
	logger     logging.Logger
	proxies    []*npm.ProxyStatus
	proxyIndex int
	mutex      sync.RWMutex
}

// NewProxyManager ProxyManager 생성자
func NewProxyManager(config npm.ProxyConfig, logger logging.Logger) npm.ProxyManager {
	manager := &proxyManagerImpl{
		config:     config,
		logger:     logger,
		proxies:    make([]*npm.ProxyStatus, 0),
		proxyIndex: 0,
	}

	// 설정에서 프록시 목록 로드
	manager.loadProxies()

	return manager
}

// loadProxies 설정에서 프록시 목록 로드
func (m *proxyManagerImpl) loadProxies() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	configProxies := m.config.GetProxies()
	m.proxies = make([]*npm.ProxyStatus, 0, len(configProxies))

	for _, proxy := range configProxies {
		status := &npm.ProxyStatus{
			Name:      proxy.Name,
			URL:       proxy.URL,
			Available: true,
			LastCheck: time.Now(),
		}
		m.proxies = append(m.proxies, status)

		m.logger.Debug("Loaded NPM proxy",
			logging.F("name", proxy.Name),
			logging.F("url", proxy.URL),
		)
	}
}

// GetNextProxy 라운드로빈으로 다음 사용 가능한 프록시 서버 반환
func (m *proxyManagerImpl) GetNextProxy() (*npm.ProxyStatus, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.proxies) == 0 {
		return nil, fmt.Errorf("no proxy servers configured")
	}

	// 사용 가능한 프록시 찾기 (최대 전체 프록시 수만큼 시도)
	attempts := 0
	for attempts < len(m.proxies) {
		proxy := m.proxies[m.proxyIndex]
		m.proxyIndex = (m.proxyIndex + 1) % len(m.proxies)
		attempts++

		if proxy.Available {
			m.logger.Debug("Selected NPM proxy",
				logging.F("name", proxy.Name),
				logging.F("url", proxy.URL),
			)
			return proxy, nil
		}
	}

	// 모든 프록시가 사용 불가능한 경우 첫 번째 프록시 반환 (재시도를 위해)
	proxy := m.proxies[0]
	m.logger.Warn("All proxies unavailable, using first proxy",
		logging.F("name", proxy.Name),
		logging.F("url", proxy.URL),
	)
	return proxy, nil
}

// CheckProxyHealth 프록시 서버 상태 확인
func (m *proxyManagerImpl) CheckProxyHealth(ctx context.Context, proxy *npm.ProxyStatus) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 간단한 상태 확인 - 실제 구현에서는 health check 엔드포인트 호출
	proxy.LastCheck = time.Now()

	// 일정 시간 후 자동으로 사용 가능하도록 설정 (5분)
	if !proxy.Available && time.Since(proxy.LastCheck) > 5*time.Minute {
		proxy.Available = true
		proxy.Error = ""
		m.logger.Info("Proxy recovered",
			logging.F("name", proxy.Name),
			logging.F("url", proxy.URL),
		)
	}

	return nil
}

// GetProxyStats 프록시 통계 조회
func (m *proxyManagerImpl) GetProxyStats() ([]*npm.ProxyStatus, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 복사본 반환
	stats := make([]*npm.ProxyStatus, len(m.proxies))
	for i, proxy := range m.proxies {
		stats[i] = &npm.ProxyStatus{
			Name:      proxy.Name,
			URL:       proxy.URL,
			Available: proxy.Available,
			LastCheck: proxy.LastCheck,
			Error:     proxy.Error,
		}
	}

	return stats, nil
}

// MarkProxyFailed 프록시를 실패로 표시
func (m *proxyManagerImpl) MarkProxyFailed(proxyURL string, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, proxy := range m.proxies {
		if proxy.URL == proxyURL {
			proxy.Available = false
			proxy.LastCheck = time.Now()
			proxy.Error = err.Error()

			m.logger.Warn("Marked proxy as failed",
				logging.F("name", proxy.Name),
				logging.F("url", proxy.URL),
				logging.F("error", err),
			)
			break
		}
	}
}
