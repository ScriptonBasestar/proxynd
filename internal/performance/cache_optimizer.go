package performance

import (
	"context"
	"sync"
	"time"

	"github.com/scriptonbasestar/proxynd/internal/logging"
)

// CacheOptimizer provides intelligent cache optimization strategies
type CacheOptimizer struct {
	logger     logging.Logger
	config     *CacheOptimizerConfig
	stats      *CacheStats
	strategies map[string]OptimizationStrategy
	mu         sync.RWMutex
}

// CacheOptimizerConfig configures cache optimization
type CacheOptimizerConfig struct {
	// Analysis settings
	AnalysisWindow time.Duration `yaml:"analysis_window" json:"analysis_window" default:"1h"`
	MinDataPoints  int           `yaml:"min_data_points" json:"min_data_points" default:"100"`

	// Optimization thresholds
	HitRateThreshold  float64 `yaml:"hit_rate_threshold" json:"hit_rate_threshold" default:"0.8"`
	MissRateThreshold float64 `yaml:"miss_rate_threshold" json:"miss_rate_threshold" default:"0.3"`
	EvictionThreshold float64 `yaml:"eviction_threshold" json:"eviction_threshold" default:"0.1"`

	// TTL optimization
	EnableTTLOptimization bool          `yaml:"enable_ttl_optimization" json:"enable_ttl_optimization" default:"true"`
	MinTTL                time.Duration `yaml:"min_ttl" json:"min_ttl" default:"5m"`
	MaxTTL                time.Duration `yaml:"max_ttl" json:"max_ttl" default:"24h"`
	TTLAdjustmentFactor   float64       `yaml:"ttl_adjustment_factor" json:"ttl_adjustment_factor" default:"0.1"`

	// Size optimization
	EnableSizeOptimization   bool          `yaml:"enable_size_optimization" json:"enable_size_optimization" default:"true"`
	MaxCacheSize             int64         `yaml:"max_cache_size" json:"max_cache_size" default:"1073741824"` // 1GB
	SizeOptimizationInterval time.Duration `yaml:"size_optimization_interval" json:"size_optimization_interval" default:"10m"`

	// Prewarming
	EnablePrewarming   bool     `yaml:"enable_prewarming" json:"enable_prewarming" default:"true"`
	PrewarmingSchedule string   `yaml:"prewarming_schedule" json:"prewarming_schedule" default:"0 6 * * *"` // 6 AM daily
	PrewarmingPatterns []string `yaml:"prewarming_patterns,omitempty" json:"prewarming_patterns,omitempty"`
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	mu                sync.RWMutex
	Hits              int64
	Misses            int64
	Evictions         int64
	Writes            int64
	TotalSize         int64
	EntryCount        int64
	AverageAccessTime time.Duration
	HotKeys           map[string]*KeyStats
	ColdKeys          map[string]*KeyStats
	SizeDistribution  map[string]int64 // size bucket -> count
	TTLDistribution   map[string]int64 // ttl bucket -> count
	LastOptimization  time.Time
	OptimizationCount int64
}

// KeyStats tracks individual key performance
type KeyStats struct {
	Key            string
	AccessCount    int64
	LastAccess     time.Time
	Size           int64
	TTL            time.Duration
	HitRate        float64
	AverageLatency time.Duration
	CreatedAt      time.Time
}

// OptimizationStrategy defines cache optimization behavior
type OptimizationStrategy interface {
	Name() string
	Analyze(ctx context.Context, stats *CacheStats) (*OptimizationReport, error)
	Apply(ctx context.Context, recommendations []OptimizationRecommendation) error
	Priority() int
}

// OptimizationReport contains analysis results
type OptimizationReport struct {
	Strategy        string                       `json:"strategy"`
	AnalysisTime    time.Time                    `json:"analysis_time"`
	CurrentMetrics  CacheMetrics                 `json:"current_metrics"`
	Issues          []PerformanceIssue           `json:"issues"`
	Recommendations []OptimizationRecommendation `json:"recommendations"`
	EstimatedImpact EstimatedImpact              `json:"estimated_impact"`
}

// CacheMetrics represents current cache performance
type CacheMetrics struct {
	HitRate            float64       `json:"hit_rate"`
	MissRate           float64       `json:"miss_rate"`
	EvictionRate       float64       `json:"eviction_rate"`
	AverageLatency     time.Duration `json:"average_latency"`
	MemoryUsage        int64         `json:"memory_usage"`
	EntryCount         int64         `json:"entry_count"`
	FragmentationRatio float64       `json:"fragmentation_ratio"`
}

// PerformanceIssue identifies cache performance problems
type PerformanceIssue struct {
	Type           string      `json:"type"`
	Severity       string      `json:"severity"`
	Description    string      `json:"description"`
	Metric         string      `json:"metric"`
	CurrentValue   interface{} `json:"current_value"`
	ThresholdValue interface{} `json:"threshold_value"`
	Impact         string      `json:"impact"`
}

// OptimizationRecommendation suggests performance improvements
type OptimizationRecommendation struct {
	Type          string                 `json:"type"`
	Priority      int                    `json:"priority"`
	Description   string                 `json:"description"`
	Action        string                 `json:"action"`
	Parameters    map[string]interface{} `json:"parameters"`
	EstimatedGain float64                `json:"estimated_gain"`
	Risk          string                 `json:"risk"`
}

// EstimatedImpact predicts optimization results
type EstimatedImpact struct {
	HitRateImprovement    float64 `json:"hit_rate_improvement"`
	LatencyReduction      float64 `json:"latency_reduction"`
	MemoryUsageReduction  float64 `json:"memory_usage_reduction"`
	ThroughputImprovement float64 `json:"throughput_improvement"`
	ConfidenceLevel       float64 `json:"confidence_level"`
}

// NewCacheOptimizer creates a new cache optimizer
func NewCacheOptimizer(logger logging.Logger, config *CacheOptimizerConfig) *CacheOptimizer {
	optimizer := &CacheOptimizer{
		logger:     logger.WithComponent("cache.optimizer"),
		config:     config,
		stats:      NewCacheStats(),
		strategies: make(map[string]OptimizationStrategy),
	}

	// Register optimization strategies
	optimizer.registerStrategies()

	return optimizer
}

// NewCacheStats creates new cache statistics
func NewCacheStats() *CacheStats {
	return &CacheStats{
		HotKeys:          make(map[string]*KeyStats),
		ColdKeys:         make(map[string]*KeyStats),
		SizeDistribution: make(map[string]int64),
		TTLDistribution:  make(map[string]int64),
	}
}

// registerStrategies registers built-in optimization strategies
func (co *CacheOptimizer) registerStrategies() {
	strategies := []OptimizationStrategy{
		NewTTLOptimizationStrategy(co.logger, co.config),
		NewSizeOptimizationStrategy(co.logger, co.config),
		NewEvictionOptimizationStrategy(co.logger, co.config),
		NewPrewarmingStrategy(co.logger, co.config),
		NewHotKeyOptimizationStrategy(co.logger, co.config),
	}

	for _, strategy := range strategies {
		co.strategies[strategy.Name()] = strategy
	}
}

// RecordCacheAccess records cache access for optimization analysis
func (co *CacheOptimizer) RecordCacheAccess(key string, hit bool, size int64, latency time.Duration) {
	co.stats.mu.Lock()
	defer co.stats.mu.Unlock()

	now := time.Now()

	// Update global stats
	if hit {
		co.stats.Hits++
	} else {
		co.stats.Misses++
	}

	// Update key-specific stats
	keyStats, exists := co.stats.HotKeys[key]
	if !exists {
		keyStats = &KeyStats{
			Key:       key,
			CreatedAt: now,
		}
		co.stats.HotKeys[key] = keyStats
	}

	keyStats.AccessCount++
	keyStats.LastAccess = now
	keyStats.Size = size
	keyStats.AverageLatency = updateAverageLatency(keyStats.AverageLatency, latency, keyStats.AccessCount)

	// Calculate hit rate for this key
	if hit {
		keyStats.HitRate = updateHitRate(keyStats.HitRate, true, keyStats.AccessCount)
	} else {
		keyStats.HitRate = updateHitRate(keyStats.HitRate, false, keyStats.AccessCount)
	}

	// Update size distribution
	sizeCategory := categorizeSize(size)
	co.stats.SizeDistribution[sizeCategory]++

	// Update global average latency
	totalAccess := co.stats.Hits + co.stats.Misses
	co.stats.AverageAccessTime = updateAverageLatency(co.stats.AverageAccessTime, latency, totalAccess)
}

// RecordCacheEviction records cache eviction for analysis
func (co *CacheOptimizer) RecordCacheEviction(key string, reason string) {
	co.stats.mu.Lock()
	defer co.stats.mu.Unlock()

	co.stats.Evictions++

	// Move from hot to cold keys if it exists
	if keyStats, exists := co.stats.HotKeys[key]; exists {
		co.stats.ColdKeys[key] = keyStats
		delete(co.stats.HotKeys, key)
	}
}

// AnalyzePerformance analyzes cache performance and generates recommendations
func (co *CacheOptimizer) AnalyzePerformance(ctx context.Context) ([]*OptimizationReport, error) {
	co.mu.RLock()
	defer co.mu.RUnlock()

	var reports []*OptimizationReport

	// Run each optimization strategy
	for name, strategy := range co.strategies {
		report, err := strategy.Analyze(ctx, co.stats)
		if err != nil {
			co.logger.Error("Strategy analysis failed",
				logging.String("strategy", name),
				logging.Error(err))
			continue
		}

		if report != nil {
			reports = append(reports, report)
		}
	}

	// Sort reports by priority
	sortReportsByPriority(reports)

	co.logger.Info("Cache performance analysis completed",
		logging.Int("strategies_analyzed", len(co.strategies)),
		logging.Int("reports_generated", len(reports)))

	return reports, nil
}

// ApplyOptimizations applies optimization recommendations
func (co *CacheOptimizer) ApplyOptimizations(ctx context.Context, reports []*OptimizationReport) error {
	appliedCount := 0

	for _, report := range reports {
		strategy, exists := co.strategies[report.Strategy]
		if !exists {
			co.logger.Warn("Unknown optimization strategy",
				logging.String("strategy", report.Strategy))
			continue
		}

		err := strategy.Apply(ctx, report.Recommendations)
		if err != nil {
			co.logger.Error("Failed to apply optimization",
				logging.String("strategy", report.Strategy),
				logging.Error(err))
			continue
		}

		appliedCount++
		co.logger.Info("Optimization applied successfully",
			logging.String("strategy", report.Strategy),
			logging.Int("recommendations", len(report.Recommendations)))
	}

	// Update optimization stats
	co.stats.mu.Lock()
	co.stats.LastOptimization = time.Now()
	co.stats.OptimizationCount++
	co.stats.mu.Unlock()

	co.logger.Info("Cache optimizations completed",
		logging.Int("applied", appliedCount),
		logging.Int("total", len(reports)))

	return nil
}

// GetCurrentMetrics returns current cache performance metrics
func (co *CacheOptimizer) GetCurrentMetrics() CacheMetrics {
	co.stats.mu.RLock()
	defer co.stats.mu.RUnlock()

	total := co.stats.Hits + co.stats.Misses
	var hitRate, missRate float64

	if total > 0 {
		hitRate = float64(co.stats.Hits) / float64(total)
		missRate = float64(co.stats.Misses) / float64(total)
	}

	var evictionRate float64
	if co.stats.Writes > 0 {
		evictionRate = float64(co.stats.Evictions) / float64(co.stats.Writes)
	}

	// Calculate fragmentation ratio (simplified)
	fragmentationRatio := calculateFragmentation(co.stats.SizeDistribution)

	return CacheMetrics{
		HitRate:            hitRate,
		MissRate:           missRate,
		EvictionRate:       evictionRate,
		AverageLatency:     co.stats.AverageAccessTime,
		MemoryUsage:        co.stats.TotalSize,
		EntryCount:         co.stats.EntryCount,
		FragmentationRatio: fragmentationRatio,
	}
}

// StartPeriodicOptimization starts periodic cache optimization
func (co *CacheOptimizer) StartPeriodicOptimization(ctx context.Context) {
	ticker := time.NewTicker(co.config.AnalysisWindow)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reports, err := co.AnalyzePerformance(ctx)
			if err != nil {
				co.logger.Error("Periodic optimization analysis failed",
					logging.Error(err))
				continue
			}

			if len(reports) > 0 {
				if err := co.ApplyOptimizations(ctx, reports); err != nil {
					co.logger.Error("Periodic optimization application failed",
						logging.Error(err))
				}
			}
		}
	}
}

// Helper functions

func updateAverageLatency(currentAvg time.Duration, newValue time.Duration, count int64) time.Duration {
	if count <= 1 {
		return newValue
	}

	currentNanos := float64(currentAvg.Nanoseconds())
	newNanos := float64(newValue.Nanoseconds())

	// Calculate weighted average
	avgNanos := (currentNanos*float64(count-1) + newNanos) / float64(count)

	return time.Duration(int64(avgNanos))
}

func updateHitRate(currentRate float64, hit bool, accessCount int64) float64 {
	if accessCount <= 1 {
		if hit {
			return 1.0
		}
		return 0.0
	}

	// Calculate incremental hit rate
	hitValue := 0.0
	if hit {
		hitValue = 1.0
	}

	return (currentRate*float64(accessCount-1) + hitValue) / float64(accessCount)
}

func categorizeSize(size int64) string {
	switch {
	case size < 1024: // < 1KB
		return "tiny"
	case size < 10*1024: // < 10KB
		return "small"
	case size < 100*1024: // < 100KB
		return "medium"
	case size < 1024*1024: // < 1MB
		return "large"
	default:
		return "huge"
	}
}

func calculateFragmentation(sizeDistribution map[string]int64) float64 {
	// Simplified fragmentation calculation
	// In practice, this would be more sophisticated

	totalEntries := int64(0)
	for _, count := range sizeDistribution {
		totalEntries += count
	}

	if totalEntries == 0 {
		return 0.0
	}

	// Calculate distribution entropy as fragmentation indicator
	var entropy float64
	for _, count := range sizeDistribution {
		if count > 0 {
			probability := float64(count) / float64(totalEntries)
			entropy -= probability * logBase2(probability)
		}
	}

	// Normalize entropy to 0-1 range (assuming max 5 size categories)
	maxEntropy := logBase2(5.0)
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}

	return 0.0
}

func logBase2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log(x) / math.Log(2)
}

func sortReportsByPriority(reports []*OptimizationReport) {
	// Sort by highest priority recommendations first
	for i := 0; i < len(reports); i++ {
		for j := i + 1; j < len(reports); j++ {
			iPriority := getHighestPriority(reports[i].Recommendations)
			jPriority := getHighestPriority(reports[j].Recommendations)

			if iPriority < jPriority { // Higher priority value = more important
				reports[i], reports[j] = reports[j], reports[i]
			}
		}
	}
}

func getHighestPriority(recommendations []OptimizationRecommendation) int {
	maxPriority := 0
	for _, rec := range recommendations {
		if rec.Priority > maxPriority {
			maxPriority = rec.Priority
		}
	}
	return maxPriority
}

import "math"

// Note: math package should be imported normally in real implementation
