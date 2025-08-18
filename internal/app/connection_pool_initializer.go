package app

import (
	"proxynd/internal/config"
	"proxynd/internal/pool"
	"proxynd/logging"
)

// ConnectionPoolInitializer Connection Pool 초기화 관리자
type ConnectionPoolInitializer struct {
	logger     logging.Logger
	poolConfig *config.ConnectionPoolManager
}

// NewConnectionPoolInitializer 새로운 Connection Pool 초기화 관리자 생성
func NewConnectionPoolInitializer() *ConnectionPoolInitializer {
	return &ConnectionPoolInitializer{
		logger:     logging.GetLogger(),
		poolConfig: config.NewConnectionPoolConfig(),
	}
}

// Initialize Connection Pool 초기화
func (c *ConnectionPoolInitializer) Initialize() error {
	c.logger.Info("Connection Pool 초기화 시작")

	// 설정 파일 로드
	if err := c.poolConfig.ReadConfig(); err != nil {
		c.logger.Warn("Connection Pool 설정 로드 실패, 기본값 사용",
			logging.F("error", err))
		// 기본값으로 계속 진행
	}

	settings := c.poolConfig.GetSettings()
	poolConfig := pool.NewConnectionPoolConfigFromSettings(
		settings.MaxTotalConnections,
		settings.MaxConnectionsPerHost,
		settings.IdleConnectionTimeoutMinutes,
		settings.KeepAliveTimeoutSeconds,
		settings.ConnectionTimeoutSeconds,
		settings.TLSHandshakeTimeoutSeconds,
		settings.ResponseHeaderTimeoutSeconds,
		settings.ExpectContinueTimeoutSeconds,
		settings.MaxRedirects,
		settings.DNSCacheTTLMinutes,
		settings.InsecureSkipVerify,
	)

	c.logger.Info("Connection Pool 설정 로드 완료",
		logging.F("max_total_connections", settings.MaxTotalConnections),
		logging.F("max_connections_per_host", settings.MaxConnectionsPerHost),
		logging.F("proxy_timeout_count", len(settings.ProxyTimeouts)),
	)

	// 전역 Connection Pool 초기화
	pool.InitializeGlobalPool(poolConfig)
	globalPool := pool.GetGlobalPool()

	// 전역 Client Factory 초기화
	pool.InitializeGlobalClientFactory(globalPool)
	clientFactory := pool.GetGlobalClientFactory()

	// 프록시별 타임아웃 설정 적용
	for proxyType, timeoutSeconds := range settings.ProxyTimeouts {
		clientFactory.UpdateTimeout(proxyType, settings.GetProxyTimeout(proxyType))
		c.logger.Debug("프록시 타임아웃 설정 적용",
			logging.F("proxy_type", proxyType),
			logging.F("timeout_seconds", timeoutSeconds),
		)
	}

	// 초기 통계 정보 로깅
	statistics := globalPool.GetStatistics()
	c.logger.Info("Connection Pool 초기화 완료",
		logging.F("statistics", statistics),
	)

	return nil
}

// GetPoolConfig Connection Pool 설정 반환 (다른 컴포넌트에서 사용)
func (c *ConnectionPoolInitializer) GetPoolConfig() *config.ConnectionPoolManager {
	return c.poolConfig
}

// Shutdown Connection Pool 정리
func (c *ConnectionPoolInitializer) Shutdown() {
	c.logger.Info("Connection Pool 종료 시작")

	globalPool := pool.GetGlobalPool()
	if globalPool != nil {
		globalPool.Cleanup()
	}

	c.logger.Info("Connection Pool 종료 완료")
}

// ReloadConfig Connection Pool 설정 다시 로드
func (c *ConnectionPoolInitializer) ReloadConfig() error {
	c.logger.Info("Connection Pool 설정 다시 로드 시작")

	// 설정 다시 로드
	if err := c.poolConfig.ReloadConfig(); err != nil {
		return err
	}

	settings := c.poolConfig.GetSettings()
	poolConfig := pool.NewConnectionPoolConfigFromSettings(
		settings.MaxTotalConnections,
		settings.MaxConnectionsPerHost,
		settings.IdleConnectionTimeoutMinutes,
		settings.KeepAliveTimeoutSeconds,
		settings.ConnectionTimeoutSeconds,
		settings.TLSHandshakeTimeoutSeconds,
		settings.ResponseHeaderTimeoutSeconds,
		settings.ExpectContinueTimeoutSeconds,
		settings.MaxRedirects,
		settings.DNSCacheTTLMinutes,
		settings.InsecureSkipVerify,
	)

	// 기존 Connection Pool에 새 설정 적용
	globalPool := pool.GetGlobalPool()
	if err := globalPool.UpdateConfig(poolConfig); err != nil {
		return err
	}

	// Client Factory에 타임아웃 업데이트
	clientFactory := pool.GetGlobalClientFactory()
	for proxyType := range settings.ProxyTimeouts {
		clientFactory.UpdateTimeout(proxyType, settings.GetProxyTimeout(proxyType))
	}

	c.logger.Info("Connection Pool 설정 다시 로드 완료",
		logging.F("max_total_connections", settings.MaxTotalConnections),
		logging.F("max_connections_per_host", settings.MaxConnectionsPerHost),
	)

	return nil
}

// GetStatistics Connection Pool 통계 정보 반환
func (c *ConnectionPoolInitializer) GetStatistics() map[string]interface{} {
	globalPool := pool.GetGlobalPool()
	clientFactory := pool.GetGlobalClientFactory()

	poolStats := globalPool.GetStatistics()
	factoryStats := clientFactory.GetPoolStatistics()
	configStats := c.poolConfig.GetSettings().GetStatistics()

	return map[string]interface{}{
		"pool_statistics":    poolStats,
		"factory_statistics": factoryStats,
		"configuration":      configStats,
		"config_file_exists": c.poolConfig.ConfigExists(),
	}
}

// 전역 Connection Pool 초기화 관리자 인스턴스
var globalPoolInitializer *ConnectionPoolInitializer

// GetGlobalPoolInitializer 전역 Connection Pool 초기화 관리자 반환
func GetGlobalPoolInitializer() *ConnectionPoolInitializer {
	if globalPoolInitializer == nil {
		globalPoolInitializer = NewConnectionPoolInitializer()
	}
	return globalPoolInitializer
}

// InitializeConnectionPool 전역 Connection Pool 초기화 (앱 시작 시 호출)
func InitializeConnectionPool() error {
	initializer := GetGlobalPoolInitializer()
	return initializer.Initialize()
}

// ShutdownConnectionPool 전역 Connection Pool 종료 (앱 종료 시 호출)
func ShutdownConnectionPool() {
	if globalPoolInitializer != nil {
		globalPoolInitializer.Shutdown()
	}
}
