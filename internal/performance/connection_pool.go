package performance

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"proxynd/logging"
)

// ConnectionPool manages HTTP connections with intelligent pooling
type ConnectionPool struct {
	logger     logging.Logger
	config     *PoolConfig
	pools      map[string]*HostPool
	stats      *PoolStats
	mu         sync.RWMutex
	closed     int32
	monitoring *PoolMonitor
}

// PoolConfig configures connection pooling behavior
type PoolConfig struct {
	// Pool sizing
	MaxIdleConns        int `yaml:"max_idle_conns" json:"max_idle_conns" default:"100"`
	MaxIdleConnsPerHost int `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host" default:"10"`
	MaxConnsPerHost     int `yaml:"max_conns_per_host" json:"max_conns_per_host" default:"50"`

	// Timeouts
	IdleConnTimeout       time.Duration `yaml:"idle_conn_timeout" json:"idle_conn_timeout" default:"90s"`
	ConnTimeout           time.Duration `yaml:"conn_timeout" json:"conn_timeout" default:"30s"`
	KeepAliveTimeout      time.Duration `yaml:"keep_alive_timeout" json:"keep_alive_timeout" default:"30s"`
	TLSHandshakeTimeout   time.Duration `yaml:"tls_handshake_timeout" json:"tls_handshake_timeout" default:"10s"`
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout" json:"response_header_timeout" default:"30s"`

	// Health checking
	EnableHealthCheck   bool          `yaml:"enable_health_check" json:"enable_health_check" default:"true"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval" default:"30s"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout" json:"health_check_timeout" default:"5s"`
	FailureThreshold    int           `yaml:"failure_threshold" json:"failure_threshold" default:"3"`

	// Circuit breaker
	EnableCircuitBreaker    bool          `yaml:"enable_circuit_breaker" json:"enable_circuit_breaker" default:"true"`
	CircuitBreakerThreshold int           `yaml:"circuit_breaker_threshold" json:"circuit_breaker_threshold" default:"5"`
	CircuitBreakerWindow    time.Duration `yaml:"circuit_breaker_window" json:"circuit_breaker_window" default:"60s"`
	CircuitBreakerTimeout   time.Duration `yaml:"circuit_breaker_timeout" json:"circuit_breaker_timeout" default:"30s"`

	// Optimization
	EnableConnectionReuse bool          `yaml:"enable_connection_reuse" json:"enable_connection_reuse" default:"true"`
	EnableTCPKeepAlive    bool          `yaml:"enable_tcp_keep_alive" json:"enable_tcp_keep_alive" default:"true"`
	EnableCompression     bool          `yaml:"enable_compression" json:"enable_compression" default:"true"`
	OptimizationInterval  time.Duration `yaml:"optimization_interval" json:"optimization_interval" default:"5m"`
}

// HostPool manages connections for a specific host
type HostPool struct {
	host           string
	client         *http.Client
	transport      *http.Transport
	stats          *HostStats
	circuitBreaker *CircuitBreaker
	healthChecker  *HealthChecker
	lastOptimized  time.Time
}

// PoolStats tracks connection pool performance
type PoolStats struct {
	mu                  sync.RWMutex
	TotalConnections    int64
	ActiveConnections   int64
	IdleConnections     int64
	ConnectionsCreated  int64
	ConnectionsReused   int64
	ConnectionsClosed   int64
	ConnectionFailures  int64
	HealthCheckPasses   int64
	HealthCheckFailures int64
	CircuitBreakerTrips int64
	OptimizationRuns    int64
	LastOptimization    time.Time
	HostStats           map[string]*HostStats
}

// HostStats tracks per-host connection statistics
type HostStats struct {
	mu                  sync.RWMutex
	Host                string
	ActiveConns         int64
	IdleConns           int64
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	AverageLatency      time.Duration
	LastRequest         time.Time
	LastSuccess         time.Time
	LastFailure         time.Time
	FailureCount        int64
	CircuitBreakerState string
	IsHealthy           bool
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	mu          sync.RWMutex
	state       CircuitState
	failures    int64
	lastFailure time.Time
	nextRetry   time.Time
	config      *PoolConfig
	logger      logging.Logger
}

// CircuitState represents circuit breaker states
type CircuitState int

const (
	// CircuitClosed indicates the circuit breaker is closed
	CircuitClosed CircuitState = iota
	// CircuitOpen indicates the circuit breaker is open
	CircuitOpen
	// CircuitHalfOpen indicates the circuit breaker is half-open
	CircuitHalfOpen
)

// HealthChecker monitors host health
type HealthChecker struct {
	mu        sync.RWMutex
	host      string
	isHealthy bool
	lastCheck time.Time
	failures  int64
	config    *PoolConfig
	logger    logging.Logger
	client    *http.Client
}

// PoolMonitor provides real-time pool monitoring
type PoolMonitor struct {
	pool   *ConnectionPool
	logger logging.Logger
	done   chan struct{}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(logger logging.Logger, config *PoolConfig) *ConnectionPool {
	pool := &ConnectionPool{
		logger: logger.WithField("component", "connection.pool"),
		config: config,
		pools:  make(map[string]*HostPool),
		stats:  NewPoolStats(),
	}

	// Start monitoring
	if config.EnableHealthCheck {
		pool.monitoring = NewPoolMonitor(pool, logger)
		go pool.monitoring.Start()
	}

	// Start optimization routine
	go pool.startOptimization()

	return pool
}

// NewPoolStats creates new pool statistics
func NewPoolStats() *PoolStats {
	return &PoolStats{
		HostStats: make(map[string]*HostStats),
	}
}

// GetClient returns an optimized HTTP client for the given host
func (cp *ConnectionPool) GetClient(host string) (*http.Client, error) {
	if atomic.LoadInt32(&cp.closed) == 1 {
		return nil, fmt.Errorf("connection pool is closed")
	}

	cp.mu.RLock()
	hostPool, exists := cp.pools[host]
	cp.mu.RUnlock()

	if !exists {
		hostPool = cp.createHostPool(host)
		cp.mu.Lock()
		cp.pools[host] = hostPool
		cp.mu.Unlock()
	}

	// Check circuit breaker
	if cp.config.EnableCircuitBreaker && !hostPool.circuitBreaker.CanRequest() {
		return nil, fmt.Errorf("circuit breaker open for host: %s", host)
	}

	// Check health
	if cp.config.EnableHealthCheck && !hostPool.healthChecker.IsHealthy() {
		return nil, fmt.Errorf("host unhealthy: %s", host)
	}

	return hostPool.client, nil
}

// createHostPool creates a new host-specific connection pool
func (cp *ConnectionPool) createHostPool(host string) *HostPool {
	// Create optimized transport
	transport := &http.Transport{
		MaxIdleConns:          cp.config.MaxIdleConns,
		MaxIdleConnsPerHost:   cp.config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cp.config.MaxConnsPerHost,
		IdleConnTimeout:       cp.config.IdleConnTimeout,
		TLSHandshakeTimeout:   cp.config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cp.config.ResponseHeaderTimeout,
		DisableCompression:    !cp.config.EnableCompression,
		DisableKeepAlives:     !cp.config.EnableConnectionReuse,
	}

	// Configure dialer
	transport.DialContext = (&net.Dialer{
		Timeout:   cp.config.ConnTimeout,
		KeepAlive: cp.config.KeepAliveTimeout,
	}).DialContext

	// Enable TCP keep-alive if configured
	if cp.config.EnableTCPKeepAlive {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   cp.config.ConnTimeout,
				KeepAlive: cp.config.KeepAliveTimeout,
			}

			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}

			if tcpConn, ok := conn.(*net.TCPConn); ok {
				if err := tcpConn.SetKeepAlive(true); err != nil {
					cp.logger.Warn("Failed to set keep alive", logging.F("error", err))
				}
				if err := tcpConn.SetKeepAlivePeriod(cp.config.KeepAliveTimeout); err != nil {
					cp.logger.Warn("Failed to set keep alive period", logging.F("error", err))
				}
			}

			return conn, nil
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cp.config.ResponseHeaderTimeout,
	}

	hostStats := &HostStats{
		Host:                host,
		CircuitBreakerState: "closed",
		IsHealthy:           true,
	}

	// Register host stats
	cp.stats.mu.Lock()
	cp.stats.HostStats[host] = hostStats
	cp.stats.mu.Unlock()

	hostPool := &HostPool{
		host:      host,
		client:    client,
		transport: transport,
		stats:     hostStats,
	}

	// Initialize circuit breaker
	if cp.config.EnableCircuitBreaker {
		hostPool.circuitBreaker = NewCircuitBreaker(cp.config, cp.logger.WithField("component", "circuit.breaker"))
	}

	// Initialize health checker
	if cp.config.EnableHealthCheck {
		hostPool.healthChecker = NewHealthChecker(host, cp.config, cp.logger.WithField("component", "health.checker"))
	}

	cp.logger.Info("Created new host pool",
		logging.F("host", host),
		logging.F("max_conns_per_host", cp.config.MaxConnsPerHost),
		logging.F("max_idle_conns_per_host", cp.config.MaxIdleConnsPerHost))

	return hostPool
}

// RecordRequest records request statistics
func (cp *ConnectionPool) RecordRequest(host string, success bool, latency time.Duration) {
	cp.stats.mu.Lock()
	defer cp.stats.mu.Unlock()

	hostStats, exists := cp.stats.HostStats[host]
	if !exists {
		return
	}

	hostStats.mu.Lock()
	defer hostStats.mu.Unlock()

	hostStats.TotalRequests++
	hostStats.LastRequest = time.Now()

	if success {
		hostStats.SuccessfulRequests++
		hostStats.LastSuccess = time.Now()
		hostStats.FailureCount = 0 // Reset failure count on success
	} else {
		hostStats.FailedRequests++
		hostStats.LastFailure = time.Now()
		hostStats.FailureCount++
	}

	// Update average latency
	if hostStats.TotalRequests == 1 {
		hostStats.AverageLatency = latency
	} else {
		// Calculate rolling average
		oldAvg := float64(hostStats.AverageLatency.Nanoseconds())
		newAvg := (oldAvg*float64(hostStats.TotalRequests-1) +
			float64(latency.Nanoseconds())) / float64(hostStats.TotalRequests)
		hostStats.AverageLatency = time.Duration(int64(newAvg))
	}

	// Update circuit breaker
	if cp.config.EnableCircuitBreaker {
		if hostPool, exists := cp.pools[host]; exists && hostPool.circuitBreaker != nil {
			if success {
				hostPool.circuitBreaker.RecordSuccess()
			} else {
				hostPool.circuitBreaker.RecordFailure()
			}
			hostStats.CircuitBreakerState = hostPool.circuitBreaker.State().String()
		}
	}
}

// GetStats returns current pool statistics
func (cp *ConnectionPool) GetStats() *PoolStats {
	cp.stats.mu.RLock()
	defer cp.stats.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := &PoolStats{
		TotalConnections:    cp.stats.TotalConnections,
		ActiveConnections:   cp.stats.ActiveConnections,
		IdleConnections:     cp.stats.IdleConnections,
		ConnectionsCreated:  cp.stats.ConnectionsCreated,
		ConnectionsReused:   cp.stats.ConnectionsReused,
		ConnectionsClosed:   cp.stats.ConnectionsClosed,
		ConnectionFailures:  cp.stats.ConnectionFailures,
		HealthCheckPasses:   cp.stats.HealthCheckPasses,
		HealthCheckFailures: cp.stats.HealthCheckFailures,
		CircuitBreakerTrips: cp.stats.CircuitBreakerTrips,
		OptimizationRuns:    cp.stats.OptimizationRuns,
		LastOptimization:    cp.stats.LastOptimization,
		HostStats:           make(map[string]*HostStats),
	}

	// Copy host stats
	for host, hostStats := range cp.stats.HostStats {
		hostStats.mu.RLock()
		stats.HostStats[host] = &HostStats{
			Host:                hostStats.Host,
			ActiveConns:         hostStats.ActiveConns,
			IdleConns:           hostStats.IdleConns,
			TotalRequests:       hostStats.TotalRequests,
			SuccessfulRequests:  hostStats.SuccessfulRequests,
			FailedRequests:      hostStats.FailedRequests,
			AverageLatency:      hostStats.AverageLatency,
			LastRequest:         hostStats.LastRequest,
			LastSuccess:         hostStats.LastSuccess,
			LastFailure:         hostStats.LastFailure,
			FailureCount:        hostStats.FailureCount,
			CircuitBreakerState: hostStats.CircuitBreakerState,
			IsHealthy:           hostStats.IsHealthy,
		}
		hostStats.mu.RUnlock()
	}

	return stats
}

// startOptimization starts the connection pool optimization routine
func (cp *ConnectionPool) startOptimization() {
	ticker := time.NewTicker(cp.config.OptimizationInterval)
	defer ticker.Stop()

	for range ticker.C {
		cp.optimizeConnections()
	}
}

// optimizeConnections optimizes connection pool based on usage patterns
func (cp *ConnectionPool) optimizeConnections() {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	now := time.Now()
	optimized := 0

	for _, hostPool := range cp.pools {
		hostStats := hostPool.stats
		hostStats.mu.RLock()

		// Skip optimization if recently optimized
		if now.Sub(hostPool.lastOptimized) < cp.config.OptimizationInterval/2 {
			hostStats.mu.RUnlock()
			continue
		}

		// Calculate optimization metrics
		requestRate := float64(hostStats.TotalRequests) / time.Since(hostStats.LastRequest).Minutes()
		successRate := float64(hostStats.SuccessfulRequests) / float64(hostStats.TotalRequests)

		hostStats.mu.RUnlock()

		// Optimize based on usage patterns
		if requestRate > 10 && successRate > 0.95 {
			// High usage, high success rate - increase connection limits
			cp.optimizeForHighLoad(hostPool)
		} else if requestRate < 1 || successRate < 0.8 {
			// Low usage or high failure rate - reduce connection limits
			cp.optimizeForLowLoad(hostPool)
		}

		hostPool.lastOptimized = now
		optimized++
	}

	cp.stats.mu.Lock()
	cp.stats.OptimizationRuns++
	cp.stats.LastOptimization = now
	cp.stats.mu.Unlock()

	cp.logger.Info("Connection pool optimization completed",
		logging.F("pools_optimized", optimized),
		logging.F("total_pools", len(cp.pools)))
}

// optimizeForHighLoad optimizes pool for high load scenarios
func (cp *ConnectionPool) optimizeForHighLoad(hostPool *HostPool) {
	// Increase connection limits
	transport := hostPool.transport

	newMaxConns := minInt(transport.MaxConnsPerHost*2, cp.config.MaxConnsPerHost*2)
	newMaxIdle := minInt(transport.MaxIdleConnsPerHost*2, cp.config.MaxIdleConnsPerHost*2)

	if newMaxConns > transport.MaxConnsPerHost {
		transport.MaxConnsPerHost = newMaxConns
		transport.MaxIdleConnsPerHost = newMaxIdle

		cp.logger.Info("Optimized pool for high load",
			logging.F("host", hostPool.host),
			logging.F("new_max_conns", newMaxConns),
			logging.F("new_max_idle", newMaxIdle))
	}
}

// optimizeForLowLoad optimizes pool for low load scenarios
func (cp *ConnectionPool) optimizeForLowLoad(hostPool *HostPool) {
	// Reduce connection limits to save resources
	transport := hostPool.transport

	newMaxConns := maxInt(transport.MaxConnsPerHost/2, 1)
	newMaxIdle := maxInt(transport.MaxIdleConnsPerHost/2, 1)

	if newMaxConns < transport.MaxConnsPerHost {
		transport.MaxConnsPerHost = newMaxConns
		transport.MaxIdleConnsPerHost = newMaxIdle

		cp.logger.Info("Optimized pool for low load",
			logging.F("host", hostPool.host),
			logging.F("new_max_conns", newMaxConns),
			logging.F("new_max_idle", newMaxIdle))
	}
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() error {
	if !atomic.CompareAndSwapInt32(&cp.closed, 0, 1) {
		return fmt.Errorf("connection pool already closed")
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Close monitoring
	if cp.monitoring != nil {
		cp.monitoring.Stop()
	}

	// Close all host pools
	for host, hostPool := range cp.pools {
		if hostPool.healthChecker != nil {
			hostPool.healthChecker.Stop()
		}
		// Transport will be closed by GC
		cp.logger.Info("Closed host pool", logging.F("host", host))
	}

	cp.logger.Info("Connection pool closed")
	return nil
}

// Circuit Breaker Implementation

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *PoolConfig, logger logging.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		state:  CircuitClosed,
		config: config,
		logger: logger,
	}
}

// CanRequest checks if requests are allowed
func (cb *CircuitBreaker) CanRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		return time.Now().After(cb.nextRetry)
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.logger.Info("Circuit breaker closed")
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == CircuitClosed && cb.failures >= int64(cb.config.CircuitBreakerThreshold) {
		cb.state = CircuitOpen
		cb.nextRetry = time.Now().Add(cb.config.CircuitBreakerTimeout)
		cb.logger.Warn("Circuit breaker opened", logging.F("failures", cb.failures))
	} else if cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
		cb.nextRetry = time.Now().Add(cb.config.CircuitBreakerTimeout)
		cb.logger.Warn("Circuit breaker reopened")
	}
}

// State returns current circuit breaker state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// String returns string representation of circuit state
func (cs CircuitState) String() string {
	switch cs {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Health Checker Implementation

// NewHealthChecker creates a new health checker
func NewHealthChecker(host string, config *PoolConfig, logger logging.Logger) *HealthChecker {
	return &HealthChecker{
		host:      host,
		isHealthy: true,
		config:    config,
		logger:    logger,
		client: &http.Client{
			Timeout: config.HealthCheckTimeout,
		},
	}
}

// IsHealthy returns current health status
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.isHealthy
}

// Start starts health checking
func (hc *HealthChecker) Start() {
	ticker := time.NewTicker(hc.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		hc.checkHealth()
	}
}

// Stop stops health checking
func (hc *HealthChecker) Stop() {
	// Implementation would include proper cleanup
}

// checkHealth performs a health check
func (hc *HealthChecker) checkHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.HealthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", "http://"+hc.host+"/health", nil)
	if err != nil {
		hc.recordFailure()
		return
	}

	resp, err := hc.client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		hc.recordFailure()
		return
	}

	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	hc.recordSuccess()
}

// recordSuccess records a successful health check
func (hc *HealthChecker) recordSuccess() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.failures = 0
	hc.lastCheck = time.Now()
	if !hc.isHealthy {
		hc.isHealthy = true
		hc.logger.Info("Host marked as healthy", logging.F("host", hc.host))
	}
}

// recordFailure records a failed health check
func (hc *HealthChecker) recordFailure() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.failures++
	hc.lastCheck = time.Now()

	if hc.isHealthy && hc.failures >= int64(hc.config.FailureThreshold) {
		hc.isHealthy = false
		hc.logger.Warn("Host marked as unhealthy",
			logging.F("host", hc.host),
			logging.F("failures", hc.failures))
	}
}

// Pool Monitor Implementation

// NewPoolMonitor creates a new pool monitor
func NewPoolMonitor(pool *ConnectionPool, logger logging.Logger) *PoolMonitor {
	return &PoolMonitor{
		pool:   pool,
		logger: logger.WithField("component", "pool.monitor"),
		done:   make(chan struct{}),
	}
}

// Start starts pool monitoring
func (pm *PoolMonitor) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.done:
			return
		case <-ticker.C:
			pm.logPoolMetrics()
		}
	}
}

// Stop stops pool monitoring
func (pm *PoolMonitor) Stop() {
	close(pm.done)
}

// logPoolMetrics logs current pool metrics
func (pm *PoolMonitor) logPoolMetrics() {
	stats := pm.pool.GetStats()

	pm.logger.Info("Connection pool metrics",
		logging.F("total_connections", stats.TotalConnections),
		logging.F("active_connections", stats.ActiveConnections),
		logging.F("idle_connections", stats.IdleConnections),
		logging.F("connections_created", stats.ConnectionsCreated),
		logging.F("connections_reused", stats.ConnectionsReused),
		logging.F("connection_failures", stats.ConnectionFailures),
		logging.F("host_pools", len(stats.HostStats)))

	// Log per-host metrics for active hosts
	for host, hostStats := range stats.HostStats {
		if time.Since(hostStats.LastRequest) < time.Hour { // Only log recently active hosts
			pm.logger.Info("Host pool metrics",
				logging.F("host", host),
				logging.F("total_requests", hostStats.TotalRequests),
				logging.F("successful_requests", hostStats.SuccessfulRequests),
				logging.F("failed_requests", hostStats.FailedRequests),
				logging.Duration("average_latency", hostStats.AverageLatency),
				logging.F("circuit_breaker_state", hostStats.CircuitBreakerState),
				logging.Bool("is_healthy", hostStats.IsHealthy))
		}
	}
}

// Helper functions

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
