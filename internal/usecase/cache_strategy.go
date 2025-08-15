package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"
	"proxynd/internal/ports"
)

// CacheStrategyService manages caching strategies for different package types
type CacheStrategyService struct {
	cacheManager  ports.CacheManager
	logger        ports.Logger
	metrics       ports.MetricsCollector
	strategies    map[string]ports.CacheStrategy
	defaultTTLs   map[string]time.Duration
	cacheKeyBuilder ports.CacheKeyBuilder
}

// NewCacheStrategyService creates a new cache strategy service
func NewCacheStrategyService(
	cacheManager ports.CacheManager,
	logger ports.Logger,
	metrics ports.MetricsCollector,
	cacheKeyBuilder ports.CacheKeyBuilder,
) *CacheStrategyService {
	css := &CacheStrategyService{
		cacheManager:    cacheManager,
		logger:          logger,
		metrics:         metrics,
		strategies:      make(map[string]ports.CacheStrategy),
		cacheKeyBuilder: cacheKeyBuilder,
		defaultTTLs: map[string]time.Duration{
			"maven":  time.Hour * 24,      // Maven artifacts are stable
			"npm":    time.Hour * 12,      // NPM packages change frequently
			"apt":    time.Hour * 6,       // APT packages update regularly
			"docker": time.Hour * 48,      // Docker images are large and stable
			"pypi":   time.Hour * 24,      // PyPI packages are relatively stable
			"yum":    time.Hour * 6,       // YUM packages update regularly
			"apk":    time.Hour * 6,       // APK packages update regularly
		},
	}
	
	// Register default strategies
	css.registerDefaultStrategies()
	
	return css
}

// CacheDecision represents a caching decision
type CacheDecision struct {
	ShouldCache bool          `json:"should_cache"`
	TTL         time.Duration `json:"ttl"`
	Backend     string        `json:"backend"`
	CacheKey    string        `json:"cache_key"`
	Strategy    string        `json:"strategy"`
	Reason      string        `json:"reason"`
}

// CacheRequest represents a cache operation request
type CacheRequest struct {
	PackageType   string            `json:"package_type"`
	Repository    string            `json:"repository"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Path          string            `json:"path"`
	ContentType   string            `json:"content_type"`
	Size          int64             `json:"size"`
	Headers       map[string]string `json:"headers"`
	Operation     string            `json:"operation"` // get, set, delete
}

// CacheResponse represents a cache operation response
type CacheResponse struct {
	Hit       bool              `json:"hit"`
	Content   []byte            `json:"-"`
	Size      int64             `json:"size"`
	Backend   string            `json:"backend"`
	CacheKey  string            `json:"cache_key"`
	ExpiresAt time.Time         `json:"expires_at"`
	Metadata  map[string]string `json:"metadata"`
}

// DetermineStrategy determines the caching strategy for a request
func (cs *CacheStrategyService) DetermineStrategy(ctx context.Context, req *CacheRequest) (*CacheDecision, error) {
	strategy := cs.GetStrategy(req.PackageType)
	if strategy == nil {
		strategy = cs.GetStrategy("default")
	}
	
	// Check if we should cache this request
	shouldCache := strategy.ShouldCache(&ports.CacheRequest{
		Key:         req.Path,
		PackageType: req.PackageType,
		TTL:         cs.getDefaultTTL(req.PackageType),
		Metadata: ports.CacheMetadata{
			ContentType: req.ContentType,
			Size:        req.Size,
			Headers:     req.Headers,
		},
	})
	
	var reason string
	if !shouldCache {
		reason = "strategy determined not to cache"
	} else {
		reason = fmt.Sprintf("using %s strategy", strategy.GetEvictionPolicy())
	}
	
	// Generate cache key
	var cacheKey string
	if cs.cacheKeyBuilder != nil {
		cacheKey = cs.cacheKeyBuilder.BuildKey(req.PackageType, req.Repository, req.Name, req.Version, req.Path)
	} else {
		cacheKey = cs.generateCacheKey(req)
	}
	
	// Determine TTL
	var ttl time.Duration
	if shouldCache {
		ttl = strategy.GetTTL(&ports.CacheRequest{
			Key:         req.Path,
			PackageType: req.PackageType,
			TTL:         cs.getDefaultTTL(req.PackageType),
			Metadata: ports.CacheMetadata{
				ContentType: req.ContentType,
				Size:        req.Size,
				Headers:     req.Headers,
			},
		})
	}
	
	// Determine backend
	backend := "filesystem" // default
	if shouldCache {
		backend = strategy.GetBackend(&ports.CacheRequest{
			Key:         req.Path,
			PackageType: req.PackageType,
			TTL:         ttl,
			Metadata: ports.CacheMetadata{
				ContentType: req.ContentType,
				Size:        req.Size,
				Headers:     req.Headers,
			},
		})
	}
	
	decision := &CacheDecision{
		ShouldCache: shouldCache,
		TTL:         ttl,
		Backend:     backend,
		CacheKey:    cacheKey,
		Strategy:    req.PackageType,
		Reason:      reason,
	}
	
	cs.logger.Debug(ctx, "Cache strategy decision", 
		&LogField{key: "package_type", value: req.PackageType},
		&LogField{key: "should_cache", value: shouldCache},
		&LogField{key: "ttl", value: ttl},
		&LogField{key: "backend", value: backend},
		&LogField{key: "reason", value: reason})
	
	return decision, nil
}

// Get retrieves content from cache with strategy
func (cs *CacheStrategyService) Get(ctx context.Context, req *CacheRequest) (*CacheResponse, error) {
	// Generate cache key
	var cacheKey string
	if cs.cacheKeyBuilder != nil {
		cacheKey = cs.cacheKeyBuilder.BuildKey(req.PackageType, req.Repository, req.Name, req.Version, req.Path)
	} else {
		cacheKey = cs.generateCacheKey(req)
	}
	
	// Create cache request for manager
	cacheManagerReq := &ports.CacheRequest{
		Key:         cacheKey,
		PackageType: req.PackageType,
		Fallback:    true,
		TTL:         cs.getDefaultTTL(req.PackageType),
		Metadata: ports.CacheMetadata{
			ContentType: req.ContentType,
			Size:        req.Size,
			Headers:     req.Headers,
		},
	}
	
	// Get from cache manager
	cacheResp, err := cs.cacheManager.Get(ctx, cacheManagerReq)
	if err != nil {
		cs.logger.Error(ctx, "Cache retrieval failed", 
			&LogField{key: "error", value: err},
			&LogField{key: "cache_key", value: cacheKey})
		cs.metrics.IncCounter("cache_errors", map[string]string{
			"operation":    "get",
			"package_type": req.PackageType,
			"error_type":   "retrieval_failed",
		})
		return &CacheResponse{
			Hit: false,
		}, nil // Don't fail the whole request on cache error
	}
	
	if cacheResp == nil || len(cacheResp.Data) == 0 {
		// Cache miss
		cs.metrics.IncCounter("cache_misses", map[string]string{
			"package_type": req.PackageType,
			"repository":   req.Repository,
		})
		return &CacheResponse{
			Hit:      false,
			CacheKey: cacheKey,
		}, nil
	}
	
	// Cache hit
	cs.metrics.IncCounter("cache_hits", map[string]string{
		"package_type": req.PackageType,
		"repository":   req.Repository,
		"backend":      cacheResp.Backend,
	})
	
	// Convert metadata
	metadata := make(map[string]string)
	if cacheResp.Metadata.ContentType != "" {
		metadata["content_type"] = cacheResp.Metadata.ContentType
	}
	for k, v := range cacheResp.Metadata.Headers {
		metadata[k] = v
	}
	
	return &CacheResponse{
		Hit:       true,
		Content:   cacheResp.Data,
		Size:      cacheResp.Size,
		Backend:   cacheResp.Backend,
		CacheKey:  cacheKey,
		ExpiresAt: cacheResp.ExpiresAt,
		Metadata:  metadata,
	}, nil
}

// Set stores content in cache with strategy
func (cs *CacheStrategyService) Set(ctx context.Context, req *CacheRequest, content []byte) error {
	// Determine caching strategy
	decision, err := cs.DetermineStrategy(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to determine cache strategy: %w", err)
	}
	
	if !decision.ShouldCache {
		cs.logger.Debug(ctx, "Strategy determined not to cache", 
			&LogField{key: "reason", value: decision.Reason})
		return nil
	}
	
	// Create cache set request
	cacheSetReq := &ports.CacheSetRequest{
		Key:         decision.CacheKey,
		Data:        content,
		PackageType: req.PackageType,
		TTL:         decision.TTL,
		Backend:     decision.Backend,
		Metadata: ports.CacheMetadata{
			ContentType:  req.ContentType,
			Size:         int64(len(content)),
			Headers:      req.Headers,
			Source:       req.Repository,
			CachedAt:     time.Now(),
			LastAccessed: time.Now(),
			AccessCount:  1,
		},
	}
	
	// Store in cache manager
	if err := cs.cacheManager.Set(ctx, cacheSetReq); err != nil {
		cs.logger.Error(ctx, "Cache storage failed", 
			&LogField{key: "error", value: err},
			&LogField{key: "cache_key", value: decision.CacheKey})
		cs.metrics.IncCounter("cache_errors", map[string]string{
			"operation":    "set",
			"package_type": req.PackageType,
			"backend":      decision.Backend,
			"error_type":   "storage_failed",
		})
		return fmt.Errorf("cache storage failed: %w", err)
	}
	
	cs.logger.Debug(ctx, "Successfully stored in cache", 
		&LogField{key: "cache_key", value: decision.CacheKey},
		&LogField{key: "backend", value: decision.Backend},
		&LogField{key: "ttl", value: decision.TTL},
		&LogField{key: "size_bytes", value: len(content)})
	
	cs.metrics.IncCounter("cache_writes", map[string]string{
		"package_type": req.PackageType,
		"backend":      decision.Backend,
	})
	
	return nil
}

// Invalidate removes content from cache
func (cs *CacheStrategyService) Invalidate(ctx context.Context, pattern string) error {
	return cs.cacheManager.Invalidate(ctx, pattern)
}

// RegisterStrategy registers a caching strategy for a package type
func (cs *CacheStrategyService) RegisterStrategy(packageType string, strategy ports.CacheStrategy) {
	cs.strategies[packageType] = strategy
	cs.logger.Info(context.Background(), "Registered cache strategy", 
		&LogField{key: "package_type", value: packageType})
}

// GetStrategy returns the caching strategy for a package type
func (cs *CacheStrategyService) GetStrategy(packageType string) ports.CacheStrategy {
	if strategy, exists := cs.strategies[packageType]; exists {
		return strategy
	}
	// Return default strategy if specific one not found
	if defaultStrategy, exists := cs.strategies["default"]; exists {
		return defaultStrategy
	}
	return nil
}

// OptimizeCache performs cache optimization
// TODO: Implement cache optimization logic
func (cs *CacheStrategyService) OptimizeCache(ctx context.Context) (*OptimizationResult, error) {
	// TODO: Implement cache optimization
	// TODO: Analyze usage patterns, hit rates, storage efficiency
	
	return &OptimizationResult{
		Strategy:         "optimization",
		ItemsOptimized:   0,
		SpaceSaved:       0,
		HitRateImprovement: 0.0,
	}, nil
}

// OptimizationResult represents cache optimization result
type OptimizationResult struct {
	Strategy           string        `json:"strategy"`
	ItemsOptimized     int64         `json:"items_optimized"`
	SpaceSaved         int64         `json:"space_saved_bytes"`
	Duration           time.Duration `json:"duration"`
	HitRateImprovement float64       `json:"hit_rate_improvement"`
	Recommendations    []string      `json:"recommendations"`
}

// AnalyzeUsage analyzes cache usage patterns
// TODO: Implement usage pattern analysis
func (cs *CacheStrategyService) AnalyzeUsage(ctx context.Context, req *UsageAnalysisRequest) (*UsageAnalysisResponse, error) {
	// TODO: Implement usage analysis
	// TODO: Analyze access patterns, hot/cold data, seasonal patterns
	
	return &UsageAnalysisResponse{
		Period:      req.Period,
		HotPatterns: []string{},
		ColdItems:   []string{},
	}, nil
}

// UsageAnalysisRequest represents usage analysis request
type UsageAnalysisRequest struct {
	Period      time.Duration `json:"period"`
	PackageType string        `json:"package_type,omitempty"`
	Repository  string        `json:"repository,omitempty"`
}

// UsageAnalysisResponse represents usage analysis response
type UsageAnalysisResponse struct {
	Period         time.Duration     `json:"period"`
	TotalRequests  int64             `json:"total_requests"`
	HitRate        float64           `json:"hit_rate"`
	HotPatterns    []string          `json:"hot_patterns"`
	ColdItems      []string          `json:"cold_items"`
	Recommendations []string         `json:"recommendations"`
	PackageStats   map[string]*PackageStats `json:"package_stats"`
}

// PackageStats represents statistics for a package type
type PackageStats struct {
	PackageType   string  `json:"package_type"`
	TotalRequests int64   `json:"total_requests"`
	HitRate       float64 `json:"hit_rate"`
	AvgSize       int64   `json:"avg_size_bytes"`
	TotalSize     int64   `json:"total_size_bytes"`
}

// WarmupCache preloads frequently accessed items
// TODO: Implement cache warmup logic
func (cs *CacheStrategyService) WarmupCache(ctx context.Context, req *WarmupRequest) (*WarmupResponse, error) {
	// TODO: Implement cache warmup
	return &WarmupResponse{
		ItemsWarmedUp: 0,
	}, nil
}

// WarmupRequest represents cache warmup request
type WarmupRequest struct {
	PackageType string   `json:"package_type,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Items       []string `json:"items,omitempty"`
	Strategy    string   `json:"strategy"` // popular, recent, all
}

// WarmupResponse represents cache warmup response
type WarmupResponse struct {
	ItemsWarmedUp int64         `json:"items_warmed_up"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors"`
	SpaceUsed     int64         `json:"space_used_bytes"`
}

// GetCacheStats returns cache statistics
// TODO: Implement cache statistics gathering
func (cs *CacheStrategyService) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	// TODO: Gather statistics from cache manager
	return &CacheStats{}, nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalSize      int64             `json:"total_size_bytes"`
	ItemCount      int64             `json:"item_count"`
	HitRate        float64           `json:"hit_rate"`
	MissRate       float64           `json:"miss_rate"`
	EvictionRate   float64           `json:"eviction_rate"`
	BackendStats   map[string]interface{} `json:"backend_stats"`
	StrategyStats  map[string]interface{} `json:"strategy_stats"`
}

// generateCacheKey generates a cache key for the request
func (cs *CacheStrategyService) generateCacheKey(req *CacheRequest) string {
	// Create a hierarchical cache key
	parts := []string{req.PackageType, req.Repository}
	
	if req.Name != "" {
		parts = append(parts, req.Name)
	}
	if req.Version != "" {
		parts = append(parts, req.Version)
	}
	if req.Path != "" {
		// Clean the path and remove leading slashes
		cleanPath := strings.TrimPrefix(req.Path, "/")
		if cleanPath != "" {
			parts = append(parts, cleanPath)
		}
	}
	
	return strings.Join(parts, ":")
}

// getDefaultTTL returns the default TTL for a package type
func (cs *CacheStrategyService) getDefaultTTL(packageType string) time.Duration {
	if ttl, exists := cs.defaultTTLs[packageType]; exists {
		return ttl
	}
	return time.Hour * 24 // Default 24 hours
}

// registerDefaultStrategies registers the default cache strategies
func (cs *CacheStrategyService) registerDefaultStrategies() {
	// Read-through strategy (default)
	cs.RegisterStrategy("default", NewReadThroughStrategy())
	
	// Maven strategy - stable artifacts, long TTL
	cs.RegisterStrategy("maven", NewWriteThroughStrategy(time.Hour*24, "filesystem"))
	
	// NPM strategy - frequent updates, shorter TTL
	cs.RegisterStrategy("npm", NewStaleWhileRevalidateStrategy(time.Hour*12, time.Hour*1))
	
	// Docker strategy - large files, long TTL, prefer S3
	cs.RegisterStrategy("docker", NewWriteThroughStrategy(time.Hour*48, "s3"))
	
	// APT/YUM/APK strategies - frequent updates
	cs.RegisterStrategy("apt", NewStaleWhileRevalidateStrategy(time.Hour*6, time.Hour*1))
	cs.RegisterStrategy("yum", NewStaleWhileRevalidateStrategy(time.Hour*6, time.Hour*1))
	cs.RegisterStrategy("apk", NewStaleWhileRevalidateStrategy(time.Hour*6, time.Hour*1))
	
	// PyPI strategy - stable packages
	cs.RegisterStrategy("pypi", NewReadThroughStrategy())
}

// ConfigureStrategy configures caching strategy parameters
// TODO: Implement strategy configuration
func (cs *CacheStrategyService) ConfigureStrategy(ctx context.Context, req *StrategyConfigRequest) error {
	// TODO: Implement strategy configuration
	return nil
}

// StrategyConfigRequest represents strategy configuration request
type StrategyConfigRequest struct {
	PackageType string                 `json:"package_type"`
	Config      map[string]interface{} `json:"config"`
}

// MonitorCacheHealth monitors cache health and performance
// TODO: Implement cache health monitoring
func (cs *CacheStrategyService) MonitorCacheHealth(ctx context.Context) (*CacheHealthReport, error) {
	// TODO: Implement cache health monitoring
	return &CacheHealthReport{
		Status: "healthy",
	}, nil
}

// CacheHealthReport represents cache health report
type CacheHealthReport struct {
	Status       string                      `json:"status"`
	Backends     map[string]*BackendHealth   `json:"backends"`
	Strategies   map[string]*StrategyHealth  `json:"strategies"`
	Alerts       []string                    `json:"alerts"`
	Recommendations []string                `json:"recommendations"`
}

// BackendHealth represents backend health status
type BackendHealth struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Latency     int64   `json:"latency_ms"`
	ErrorRate   float64 `json:"error_rate"`
	Utilization float64 `json:"utilization"`
}

// StrategyHealth represents strategy health status
type StrategyHealth struct {
	PackageType string  `json:"package_type"`
	Status      string  `json:"status"`
	HitRate     float64 `json:"hit_rate"`
	Efficiency  float64 `json:"efficiency"`
}