package pip

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/pip"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
	"proxynd/pkg/httpclient"
)

// indexManagerImpl PyPI 인덱스 서버 관리 서비스 구현
type indexManagerImpl struct {
	config  pip.ProxyConfig
	logger  logging.Logger
	indexes []*pip.IndexStatus
	current int
	mu      sync.RWMutex
	client  *httpclient.ProxyClient
}

// NewIndexManager IndexManager 생성자
func NewIndexManager(config pip.ProxyConfig, logger logging.Logger) pip.IndexManager {
	manager := &indexManagerImpl{
		config:  config,
		logger:  logger,
		current: 0,
		client:  httpclient.NewProxyClient(),
	}

	// 인덱스 서버 초기화
	manager.initializeIndexes()

	return manager
}

// initializeIndexes 인덱스 서버 목록 초기화
func (m *indexManagerImpl) initializeIndexes() {
	proxies := m.config.GetProxies()
	m.indexes = make([]*pip.IndexStatus, len(proxies))

	for i, proxy := range proxies {
		m.indexes[i] = &pip.IndexStatus{
			Name:      proxy.Name,
			URL:       proxy.URL,
			Available: true, // 초기에는 사용 가능하다고 가정
			LastCheck: time.Now(),
		}
	}

	m.logger.Info("Initialized PyPI indexes",
		logging.F("indexCount", len(m.indexes)),
	)
}

// GetNextIndex 다음 사용 가능한 인덱스 서버 반환
func (m *indexManagerImpl) GetNextIndex() (*pip.IndexStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.indexes) == 0 {
		return nil, fmt.Errorf("no indexes configured")
	}

	// 사용 가능한 인덱스 찾기 (round-robin)
	attempts := 0
	for attempts < len(m.indexes) {
		index := m.indexes[m.current]
		m.current = (m.current + 1) % len(m.indexes)

		if index.Available {
			return index, nil
		}

		attempts++
	}

	return nil, fmt.Errorf("no available indexes")
}

// CheckIndexHealth 인덱스 서버 상태 확인
func (m *indexManagerImpl) CheckIndexHealth(ctx context.Context, index *pip.IndexStatus) error {
	startTime := time.Now()

	// 간단한 health check URL 사용 (PyPI simple API)
	healthURL := fmt.Sprintf("%s/simple/", strings.TrimSuffix(index.URL, "/"))

	resp, err := m.client.GetWithRetry(ctx, healthURL, 1)
	if err != nil {
		index.Available = false
		index.Error = err.Error()
		index.LastCheck = time.Now()
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 응답 시간을 밀리초로 계산
	responseTime := float64(time.Since(startTime).Nanoseconds()) / 1e6

	if resp.StatusCode == http.StatusOK {
		index.Available = true
		index.Error = ""
		index.ResponseTime = responseTime
	} else {
		index.Available = false
		index.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	index.LastCheck = time.Now()

	m.logger.Debug("Index health check completed",
		logging.F("indexURL", index.URL),
		logging.F("available", index.Available),
		logging.F("responseTime", responseTime),
	)

	return nil
}

// GetIndexStats 인덱스 통계 조회
func (m *indexManagerImpl) GetIndexStats() ([]*pip.IndexStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 복사본 생성 (concurrent modification 방지)
	stats := make([]*pip.IndexStatus, len(m.indexes))
	for i, index := range m.indexes {
		stats[i] = &pip.IndexStatus{
			Name:         index.Name,
			URL:          index.URL,
			Available:    index.Available,
			LastCheck:    index.LastCheck,
			Error:        index.Error,
			ResponseTime: index.ResponseTime,
		}
	}

	return stats, nil
}

// MarkIndexFailed 인덱스를 실패로 표시
func (m *indexManagerImpl) MarkIndexFailed(indexURL string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, index := range m.indexes {
		if index.URL == indexURL {
			index.Available = false
			index.Error = err.Error()
			index.LastCheck = time.Now()

			m.logger.Warn("Marked index as failed",
				logging.F("indexURL", indexURL),
				logging.F("error", err),
			)
			break
		}
	}
}

// BuildPackageURL 패키지 요청 URL 구성
func (m *indexManagerImpl) BuildPackageURL(baseURL, packagePath string) string {
	// /simple/ 경로 처리
	if strings.HasPrefix(packagePath, "simple/") {
		return helpers.JoinURL(baseURL, packagePath)
	}

	// /packages/ 경로 처리
	if strings.HasPrefix(packagePath, "packages/") {
		return helpers.JoinURL(baseURL, packagePath)
	}

	// /pypi/ 경로 처리 (JSON API)
	if strings.HasPrefix(packagePath, "pypi/") {
		return helpers.JoinURL(baseURL, packagePath)
	}

	// 기본적으로 simple API 사용
	return helpers.JoinURL(baseURL, "simple", packagePath)
}
