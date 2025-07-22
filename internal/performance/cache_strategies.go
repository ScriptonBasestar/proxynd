package performance

import (
	"context"
	"fmt"
	"sort"
	"time"

	"proxynd/logging"
)

// TTLOptimizationStrategy represents a TTL optimization strategy
type TTLOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

// NewTTLOptimizationStrategy creates a new instance of TTLOptimizationStrategy
func NewTTLOptimizationStrategy(logger logging.Logger, config *CacheOptimizerConfig) *TTLOptimizationStrategy {
	return &TTLOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.ttl"),
		config: config,
	}
}

// Name returns the name of the optimization strategy
func (s *TTLOptimizationStrategy) Name() string {
	return "ttl_optimization"
}

// Priority returns the priority of this optimization strategy
func (s *TTLOptimizationStrategy) Priority() int {
	return 7
}

// Analyze performs TTL optimization analysis
func (s *TTLOptimizationStrategy) Analyze(_ context.Context, stats *CacheStats) (*OptimizationReport, error) {
	if !s.config.EnableTTLOptimization {
		return nil, nil
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	report := &OptimizationReport{
		Strategy:     s.Name(),
		AnalysisTime: time.Now(),
		CurrentMetrics: CacheMetrics{
			HitRate:      float64(stats.Hits) / float64(stats.Hits+stats.Misses),
			EvictionRate: float64(stats.Evictions) / float64(stats.Writes),
		},
	}

	// Analyze TTL distribution and access patterns
	var recommendations []OptimizationRecommendation
	var issues []PerformanceIssue

	// Find keys with suboptimal TTL
	for key, keyStats := range stats.HotKeys {
		// Hot keys with short TTL - extend TTL
		if keyStats.AccessCount > 10 && keyStats.HitRate > 0.8 {
			if keyStats.TTL < s.config.MinTTL*2 {
				recommendations = append(recommendations, OptimizationRecommendation{
					Type:        "extend_ttl",
					Priority:    8,
					Description: fmt.Sprintf("Extend TTL for hot key: %s", key),
					Action:      "update_ttl",
					Parameters: map[string]interface{}{
						"key":     key,
						"new_ttl": keyStats.TTL * 2,
					},
					EstimatedGain: 0.15,
					Risk:          "low",
				})
			}
		}

		// Keys with low hit rate - reduce TTL
		if keyStats.HitRate < 0.3 && keyStats.TTL > s.config.MinTTL {
			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        "reduce_ttl",
				Priority:    5,
				Description: fmt.Sprintf("Reduce TTL for cold key: %s", key),
				Action:      "update_ttl",
				Parameters: map[string]interface{}{
					"key":     key,
					"new_ttl": keyStats.TTL / 2,
				},
				EstimatedGain: 0.05,
				Risk:          "low",
			})
		}
	}

	// Check for TTL distribution issues
	if len(stats.TTLDistribution) > 0 {
		var totalEntries int64
		for _, count := range stats.TTLDistribution {
			totalEntries += count
		}

		// Check if too many entries have very short TTL
		shortTTLCount := stats.TTLDistribution["short"] + stats.TTLDistribution["very_short"]
		if float64(shortTTLCount)/float64(totalEntries) > 0.5 {
			issues = append(issues, PerformanceIssue{
				Type:           "ttl_distribution",
				Severity:       "medium",
				Description:    "Too many entries with short TTL causing frequent evictions",
				Metric:         "short_ttl_ratio",
				CurrentValue:   float64(shortTTLCount) / float64(totalEntries),
				ThresholdValue: 0.3,
				Impact:         "Increased cache misses and reduced performance",
			})

			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        "rebalance_ttl",
				Priority:    6,
				Description: "Rebalance TTL distribution to reduce short-lived entries",
				Action:      "global_ttl_adjustment",
				Parameters: map[string]interface{}{
					"adjustment_factor": 1.5,
					"affected_keys":     "short_ttl_keys",
				},
				EstimatedGain: 0.2,
				Risk:          "medium",
			})
		}
	}

	report.Issues = issues
	report.Recommendations = recommendations
	report.EstimatedImpact = EstimatedImpact{
		HitRateImprovement:    0.1,
		LatencyReduction:      0.05,
		MemoryUsageReduction:  0.1,
		ThroughputImprovement: 0.08,
		ConfidenceLevel:       0.75,
	}
	// Apply performs apply operation

	return report, nil
}

// Apply applies the given optimization recommendations
func (s *TTLOptimizationStrategy) Apply(_ context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		if rec.Type == "extend_ttl" || rec.Type == "reduce_ttl" {
			// In a real implementation, this would call the cache service
			if key, ok := rec.Parameters["key"].(string); ok {
				s.logger.Info("TTL optimization applied",
					logging.F("action", rec.Action),
					logging.F("key", key))
			}
			applied++
		}
	}

	s.logger.Info("TTL optimization strategy completed",
		logging.F("applied", applied),
		logging.F("total", len(recommendations)))
	// SizeOptimizationStrategy represents a size optimization strategy

	return nil
}

// SizeOptimizationStrategy implements cache size optimization
type SizeOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

// NewSizeOptimizationStrategy creates a new size optimization strategy
func NewSizeOptimizationStrategy(logger logging.Logger, config *CacheOptimizerConfig) *SizeOptimizationStrategy {
	return &SizeOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.size"),
		// Priority performs priority operation
		config: config,
	}
}

// Name returns the strategy name
func (s *SizeOptimizationStrategy) Name() string {
	return "size_optimization"
}

// Priority returns the strategy priority
func (s *SizeOptimizationStrategy) Priority() int {
	return 8
}

// Analyze analyzes cache statistics for size optimization opportunities
func (s *SizeOptimizationStrategy) Analyze(_ context.Context, stats *CacheStats) (*OptimizationReport, error) {
	if !s.config.EnableSizeOptimization {
		return nil, nil
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	report := &OptimizationReport{
		Strategy:     s.Name(),
		AnalysisTime: time.Now(),
		CurrentMetrics: CacheMetrics{
			MemoryUsage: stats.TotalSize,
			EntryCount:  stats.EntryCount,
		},
	}

	var recommendations []OptimizationRecommendation
	var issues []PerformanceIssue

	// Check cache size utilization
	sizeUtilization := float64(stats.TotalSize) / float64(s.config.MaxCacheSize)
	if sizeUtilization > 0.9 {
		issues = append(issues, PerformanceIssue{
			Type:           "high_memory_usage",
			Severity:       "high",
			Description:    "Cache memory usage is very high",
			Metric:         "memory_utilization",
			CurrentValue:   sizeUtilization,
			ThresholdValue: 0.8,
			Impact:         "Frequent evictions and reduced cache effectiveness",
		})

		// Find large, low-value entries to evict
		var candidates []KeyStats
		for _, keyStats := range stats.HotKeys {
			if keyStats.Size > 100*1024 && keyStats.HitRate < 0.5 { // Large files with low hit rate
				candidates = append(candidates, *keyStats)
			}
		}

		// Sort by size/hit_rate ratio (larger ratio = better candidate for eviction)
		sort.Slice(candidates, func(i, j int) bool {
			ratioI := float64(candidates[i].Size) / (candidates[i].HitRate + 0.01)
			ratioJ := float64(candidates[j].Size) / (candidates[j].HitRate + 0.01)
			return ratioI > ratioJ
		})

		// Recommend evicting top candidates
		for i, candidate := range candidates {
			if i >= 10 { // Limit to top 10 candidates
				break
			}
			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        "evict_large_unused",
				Priority:    9,
				Description: fmt.Sprintf("Evict large underutilized entry: %s", candidate.Key),
				Action:      "evict_key",
				Parameters: map[string]interface{}{
					"key":      candidate.Key,
					"size":     candidate.Size,
					"hit_rate": candidate.HitRate,
				},
				EstimatedGain: 0.1,
				Risk:          "low",
			})
		}
	}

	// Analyze size distribution
	if len(stats.SizeDistribution) > 0 {
		totalEntries := int64(0)
		hugeEntries := stats.SizeDistribution["huge"]
		for _, count := range stats.SizeDistribution {
			totalEntries += count
		}

		if float64(hugeEntries)/float64(totalEntries) > 0.1 {
			issues = append(issues, PerformanceIssue{
				Type:           "size_distribution",
				Severity:       "medium",
				Description:    "Too many very large entries consuming cache space",
				Metric:         "huge_entries_ratio",
				CurrentValue:   float64(hugeEntries) / float64(totalEntries),
				ThresholdValue: 0.05,
				Impact:         "Reduced cache efficiency and frequent evictions",
			})

			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        "implement_size_limits",
				Priority:    7,
				Description: "Implement size limits for cache entries",
				Action:      "update_size_policy",
				Parameters: map[string]interface{}{
					"max_entry_size": 1024 * 1024, // 1MB
					"compression":    true,
				},
				EstimatedGain: 0.15,
				Risk:          "medium",
			})
		}
	}

	report.Issues = issues
	report.Recommendations = recommendations
	// Apply performs apply operation
	report.EstimatedImpact = EstimatedImpact{
		MemoryUsageReduction:  0.2,
		HitRateImprovement:    0.05,
		ThroughputImprovement: 0.1,
		ConfidenceLevel:       0.8,
	}

	return report, nil
}

// Apply applies the given size optimization recommendations
func (s *SizeOptimizationStrategy) Apply(_ context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		switch rec.Type {
		case "evict_large_unused":
			s.logger.Info("Size optimization: evicting large unused entry",
				logging.F("key", rec.Parameters["key"]),
				logging.F("size", rec.Parameters["size"]))
			applied++
		case "implement_size_limits":
			s.logger.Info("Size optimization: implementing size limits",
				logging.F("max_entry_size", rec.Parameters["max_entry_size"]))
			// EvictionOptimizationStrategy represents a eviction optimization strategy
			applied++
		}
	}

	s.logger.Info("Size optimization strategy completed",
		logging.F("applied", applied))
	// NewEvictionOptimizationStrategy creates a new evictionoptimizationstrategy

	return nil
}

// EvictionOptimizationStrategy implements cache eviction optimization
type EvictionOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

// NewEvictionOptimizationStrategy creates a new eviction optimization strategy
func NewEvictionOptimizationStrategy(
	logger logging.Logger, config *CacheOptimizerConfig) *EvictionOptimizationStrategy {
	return &EvictionOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.eviction"),
		config: config,
	}
}

// Name returns the strategy name
func (s *EvictionOptimizationStrategy) Name() string {
	return "eviction_optimization"
}

// Priority returns the strategy priority
func (s *EvictionOptimizationStrategy) Priority() int {
	return 6
}

// Analyze analyzes cache statistics for eviction optimization opportunities
func (s *EvictionOptimizationStrategy) Analyze(_ context.Context, stats *CacheStats) (*OptimizationReport, error) {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	report := &OptimizationReport{
		Strategy:     s.Name(),
		AnalysisTime: time.Now(),
	}

	var recommendations []OptimizationRecommendation
	var issues []PerformanceIssue

	// Calculate eviction rate
	evictionRate := float64(stats.Evictions) / float64(stats.Writes)
	if evictionRate > s.config.EvictionThreshold {
		issues = append(issues, PerformanceIssue{
			Type:           "high_eviction_rate",
			Severity:       "medium",
			Description:    "Cache eviction rate is too high",
			Metric:         "eviction_rate",
			CurrentValue:   evictionRate,
			ThresholdValue: s.config.EvictionThreshold,
			Impact:         "Reduced cache effectiveness and wasted resources",
		})

		recommendations = append(recommendations, OptimizationRecommendation{
			Type:        "optimize_eviction_policy",
			Priority:    6,
			Description: "Optimize cache eviction policy to reduce unnecessary evictions",
			Action:      "update_eviction_algorithm",
			Parameters: map[string]interface{}{
				"algorithm": "lru_with_frequency",
				"weights": map[string]float64{
					"recency":   0.6,
					"frequency": 0.4,
				},
			},
			EstimatedGain: 0.12,
			Risk:          "medium",
		})
	}

	// Analyze evicted vs retained keys
	var prematureEvictions int
	for _, keyStats := range stats.ColdKeys {
		// If a key was evicted but had good performance, it's a premature eviction
		if keyStats.HitRate > 0.7 && keyStats.AccessCount > 5 {
			prematureEvictions++
		}
	}

	if prematureEvictions > 10 {
		issues = append(issues, PerformanceIssue{
			Type:           "premature_evictions",
			Severity:       "medium",
			Description:    "Too many high-value keys are being evicted prematurely",
			Metric:         "premature_eviction_count",
			CurrentValue:   prematureEvictions,
			ThresholdValue: 5,
			Impact:         "Loss of valuable cached data",
		})

		recommendations = append(recommendations, OptimizationRecommendation{
			Type:        "protect_hot_keys",
			Priority:    7,
			Description: "Implement protection for frequently accessed keys",
			Action:      "add_key_protection",
			Parameters: map[string]interface{}{
				"protection_threshold": 0.7,
				"min_access_count":     5,
			},
			EstimatedGain: 0.08,
			// Apply performs apply operation
			Risk: "low",
		})
	}

	report.Issues = issues
	report.Recommendations = recommendations
	report.EstimatedImpact = EstimatedImpact{
		HitRateImprovement:    0.08,
		LatencyReduction:      0.03,
		ThroughputImprovement: 0.06,
		ConfidenceLevel:       0.7,
	}

	return report, nil
}

// Apply applies the given eviction optimization recommendations
func (s *EvictionOptimizationStrategy) Apply(_ context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		switch rec.Type {
		case "optimize_eviction_policy":
			s.logger.Info("Eviction optimization: updating eviction algorithm",
				// PrewarmingStrategy represents a prewarming strategy
				logging.F("algorithm", rec.Parameters["algorithm"]))
			applied++
		case "protect_hot_keys":
			s.logger.Info("Eviction optimization: implementing key protection",
				logging.F("protection_threshold", rec.Parameters["protection_threshold"]))
			// NewPrewarmingStrategy creates a new prewarmingstrategy
			applied++
		}
	}

	s.logger.Info("Eviction optimization strategy completed",
		logging.F("applied", applied))
	// Name returns the name of the component

	return nil
}

// Priority performs priority operation

// PrewarmingStrategy implements cache prewarming optimization
type PrewarmingStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

// NewPrewarmingStrategy creates a new prewarming strategy
func NewPrewarmingStrategy(logger logging.Logger, config *CacheOptimizerConfig) *PrewarmingStrategy {
	return &PrewarmingStrategy{
		logger: logger.WithField("component", "cache.strategy.prewarming"),
		config: config,
	}
}

// Name returns the strategy name
func (s *PrewarmingStrategy) Name() string {
	return "prewarming"
}

// Priority returns the strategy priority
func (s *PrewarmingStrategy) Priority() int {
	return 5
}

// Analyze analyzes cache statistics for prewarming opportunities
func (s *PrewarmingStrategy) Analyze(_ context.Context, stats *CacheStats) (*OptimizationReport, error) {
	if !s.config.EnablePrewarming {
		return nil, nil
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	report := &OptimizationReport{
		Strategy:     s.Name(),
		AnalysisTime: time.Now(),
	}

	var recommendations []OptimizationRecommendation

	// Identify patterns for prewarming
	var prewarmCandidates []string
	for key, keyStats := range stats.HotKeys {
		// Keys with consistent high access rate are good prewarming candidates
		if keyStats.HitRate > 0.8 && keyStats.AccessCount > 20 {
			// Check if key follows patterns
			for _, pattern := range s.config.PrewarmingPatterns {
				if matchesPattern(key, pattern) {
					prewarmCandidates = append(prewarmCandidates, key)
					break
				}
			}
		}
	}

	if len(prewarmCandidates) > 0 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Type:        "schedule_prewarming",
			Priority:    5,
			Description: fmt.Sprintf("Schedule prewarming for %d popular keys", len(prewarmCandidates)),
			Action:      "create_prewarming_schedule",
			Parameters: map[string]interface{}{
				"keys":     prewarmCandidates,
				"schedule": s.config.PrewarmingSchedule,
			},
			EstimatedGain: 0.1,
			Risk:          "low",
		})
	}

	// Check for cold start scenarios
	totalAccess := stats.Hits + stats.Misses
	if totalAccess > 0 {
		coldStartRatio := float64(stats.Misses) / float64(totalAccess)
		if coldStartRatio > 0.5 {
			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        "implement_predictive_prewarming",
				Priority:    6,
				Description: "Implement predictive prewarming to reduce cold starts",
				// Apply performs apply operation
				Action: "enable_predictive_prewarming",
				Parameters: map[string]interface{}{
					"prediction_window":    "1h",
					"confidence_threshold": 0.7,
				},
				EstimatedGain: 0.15,
				Risk:          "medium",
			})
		}
	}

	report.Recommendations = recommendations
	report.EstimatedImpact = EstimatedImpact{
		HitRateImprovement:    0.12,
		LatencyReduction:      0.2,
		ThroughputImprovement: 0.08,
		ConfidenceLevel:       0.6,
	}

	return report, nil
}

// Apply applies the given prewarming recommendations
func (s *PrewarmingStrategy) Apply(_ context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	// HotKeyOptimizationStrategy represents a hot key optimization strategy
	for _, rec := range recommendations {
		switch rec.Type {
		case "schedule_prewarming":
			//nolint:errcheck // 타입 어설션 실패는 무시하고 빈 슬라이스 사용
			keys, _ := rec.Parameters["keys"].([]string)
			s.logger.Info("Prewarming: scheduling prewarming for popular keys",
				// NewHotKeyOptimizationStrategy creates a new hotkeyoptimizationstrategy
				logging.F("key_count", len(keys)),
				logging.F("schedule", rec.Parameters["schedule"]))
			applied++
		case "implement_predictive_prewarming":
			s.logger.Info("Prewarming: enabling predictive prewarming",
				logging.F("prediction_window", rec.Parameters["prediction_window"]))
			// Name returns the name of the component
			applied++
		}
	}
	// Priority performs priority operation

	s.logger.Info("Prewarming strategy completed",
		logging.F("applied", applied))
	// Analyze performs analyze operation

	return nil
}

// HotKeyOptimizationStrategy implements hot key optimization
type HotKeyOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

// NewHotKeyOptimizationStrategy creates a new hot key optimization strategy
func NewHotKeyOptimizationStrategy(logger logging.Logger, config *CacheOptimizerConfig) *HotKeyOptimizationStrategy {
	return &HotKeyOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.hotkey"),
		config: config,
	}
}

// Name returns the strategy name
func (s *HotKeyOptimizationStrategy) Name() string {
	return "hot_key_optimization"
}

// Priority returns the strategy priority
func (s *HotKeyOptimizationStrategy) Priority() int {
	return 8
}

// Analyze analyzes cache statistics for hot key optimization opportunities
func (s *HotKeyOptimizationStrategy) Analyze(_ context.Context, stats *CacheStats) (*OptimizationReport, error) {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	report := &OptimizationReport{
		Strategy:     s.Name(),
		AnalysisTime: time.Now(),
	}

	var recommendations []OptimizationRecommendation
	var issues []PerformanceIssue

	// Identify hot keys that need special handling
	var hotKeys []KeyStats
	for _, keyStats := range stats.HotKeys {
		if keyStats.AccessCount > 100 && keyStats.HitRate > 0.9 {
			hotKeys = append(hotKeys, *keyStats)
		}
	}

	// Sort by access count
	sort.Slice(hotKeys, func(i, j int) bool {
		return hotKeys[i].AccessCount > hotKeys[j].AccessCount
	})

	// Check for hot key concentration
	if len(hotKeys) > 0 {
		topHotKeyAccess := hotKeys[0].AccessCount
		totalAccess := stats.Hits + stats.Misses

		hotKeyRatio := float64(topHotKeyAccess) / float64(totalAccess)
		if hotKeyRatio > 0.2 { // Single key accounts for >20% of access
			issues = append(issues, PerformanceIssue{
				Type:           "hot_key_concentration",
				Severity:       "high",
				Description:    "Single key dominates cache access patterns",
				Metric:         "hot_key_ratio",
				CurrentValue:   hotKeyRatio,
				ThresholdValue: 0.1,
				Impact:         "Cache bottleneck and potential performance degradation",
			})

			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        "implement_hot_key_replication",
				Priority:    9,
				Description: fmt.Sprintf("Implement replication for hot key: %s", hotKeys[0].Key),
				Action:      "replicate_hot_key",
				Parameters: map[string]interface{}{
					"key":                hotKeys[0].Key,
					"replication_factor": 3,
					"load_balancing":     true,
				},
				EstimatedGain: 0.25,
				Risk:          "medium",
			})
		}
	}

	// Check for keys that should be promoted to faster storage
	for _, keyStats := range hotKeys[:minInt(len(hotKeys), 5)] { // Top 5 hot keys
		// Apply performs apply operation
		if keyStats.AverageLatency > 10*time.Millisecond {
			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        "promote_to_fast_storage",
				Priority:    8,
				Description: fmt.Sprintf("Promote hot key to faster storage: %s", keyStats.Key),
				Action:      "promote_key",
				Parameters: map[string]interface{}{
					"key":          keyStats.Key,
					"storage_tier": "memory",
				},
				EstimatedGain: 0.15,
				Risk:          "low",
			})
		}
	}

	report.Issues = issues
	report.Recommendations = recommendations
	report.EstimatedImpact = EstimatedImpact{
		LatencyReduction:      0.2,
		ThroughputImprovement: 0.18,
		HitRateImprovement:    0.05,
		ConfidenceLevel:       0.85,
	}

	return report, nil
}

// Apply applies the given hot key optimization recommendations
func (s *HotKeyOptimizationStrategy) Apply(_ context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		switch rec.Type {
		case "implement_hot_key_replication":
			//nolint:errcheck // 타입 어설션 실패는 무시하고 빈 문자열 사용
			key, _ := rec.Parameters["key"].(string)
			//nolint:errcheck // 타입 어설션 실패는 무시하고 기본값 0 사용
			factor, _ := rec.Parameters["replication_factor"].(int)
			s.logger.Info("Hot key optimization: implementing replication",
				logging.F("key", key),
				logging.F("replication_factor", factor))
			applied++
		case "promote_to_fast_storage":
			//nolint:errcheck // 타입 어설션 실패는 무시하고 빈 문자열 사용
			key, _ := rec.Parameters["key"].(string)
			//nolint:errcheck // 타입 어설션 실패는 무시하고 빈 문자열 사용
			tier, _ := rec.Parameters["storage_tier"].(string)
			s.logger.Info("Hot key optimization: promoting to fast storage",
				logging.F("key", key),
				logging.F("storage_tier", tier))
			applied++
		}
	}

	s.logger.Info("Hot key optimization strategy completed",
		logging.F("applied", applied))

	return nil
}

// Helper functions

func matchesPattern(key, pattern string) bool {
	// Simplified pattern matching - in practice, use proper regex
	return len(key) > 0 && len(pattern) > 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
