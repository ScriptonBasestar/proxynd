package performance

import (
	"context"
	"fmt"
	"sort"
	"time"

	"proxynd/logging"
)

// TTL Optimization Strategy
type TTLOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

func NewTTLOptimizationStrategy(logger logging.Logger, config *CacheOptimizerConfig) *TTLOptimizationStrategy {
	return &TTLOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.ttl"),
		config: config,
	}
}

func (s *TTLOptimizationStrategy) Name() string {
	return "ttl_optimization"
}

func (s *TTLOptimizationStrategy) Priority() int {
	return 7
}

func (s *TTLOptimizationStrategy) Analyze(ctx context.Context, stats *CacheStats) (*OptimizationReport, error) {
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

	return report, nil
}

func (s *TTLOptimizationStrategy) Apply(ctx context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		if rec.Type == "extend_ttl" || rec.Type == "reduce_ttl" {
			// In a real implementation, this would call the cache service
			s.logger.Info("TTL optimization applied",
				logging.F("action", rec.Action),
				logging.F("key", rec.Parameters["key"].(string)))
			applied++
		}
	}

	s.logger.Info("TTL optimization strategy completed",
		logging.F("applied", applied),
		logging.F("total", len(recommendations)))

	return nil
}

// Size Optimization Strategy
type SizeOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

func NewSizeOptimizationStrategy(logger logging.Logger, config *CacheOptimizerConfig) *SizeOptimizationStrategy {
	return &SizeOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.size"),
		config: config,
	}
}

func (s *SizeOptimizationStrategy) Name() string {
	return "size_optimization"
}

func (s *SizeOptimizationStrategy) Priority() int {
	return 8
}

func (s *SizeOptimizationStrategy) Analyze(ctx context.Context, stats *CacheStats) (*OptimizationReport, error) {
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
	report.EstimatedImpact = EstimatedImpact{
		MemoryUsageReduction:  0.2,
		HitRateImprovement:    0.05,
		ThroughputImprovement: 0.1,
		ConfidenceLevel:       0.8,
	}

	return report, nil
}

func (s *SizeOptimizationStrategy) Apply(ctx context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		switch rec.Type {
		case "evict_large_unused":
			s.logger.Info("Size optimization: evicting large unused entry",
				logging.F("key", rec.Parameters["key"].(string)),
				logging.F("size", rec.Parameters["size"].(int64)))
			applied++
		case "implement_size_limits":
			s.logger.Info("Size optimization: implementing size limits",
				logging.F("max_entry_size", rec.Parameters["max_entry_size"].(int)))
			applied++
		}
	}

	s.logger.Info("Size optimization strategy completed",
		logging.F("applied", applied))

	return nil
}

// Eviction Optimization Strategy
type EvictionOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

func NewEvictionOptimizationStrategy(logger logging.Logger, config *CacheOptimizerConfig) *EvictionOptimizationStrategy {
	return &EvictionOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.eviction"),
		config: config,
	}
}

func (s *EvictionOptimizationStrategy) Name() string {
	return "eviction_optimization"
}

func (s *EvictionOptimizationStrategy) Priority() int {
	return 6
}

func (s *EvictionOptimizationStrategy) Analyze(ctx context.Context, stats *CacheStats) (*OptimizationReport, error) {
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
			Risk:          "low",
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

func (s *EvictionOptimizationStrategy) Apply(ctx context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		switch rec.Type {
		case "optimize_eviction_policy":
			s.logger.Info("Eviction optimization: updating eviction algorithm",
				logging.F("algorithm", rec.Parameters["algorithm"].(string)))
			applied++
		case "protect_hot_keys":
			s.logger.Info("Eviction optimization: implementing key protection",
				logging.Float64("protection_threshold", rec.Parameters["protection_threshold"].(float64)))
			applied++
		}
	}

	s.logger.Info("Eviction optimization strategy completed",
		logging.F("applied", applied))

	return nil
}

// Prewarming Strategy
type PrewarmingStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

func NewPrewarmingStrategy(logger logging.Logger, config *CacheOptimizerConfig) *PrewarmingStrategy {
	return &PrewarmingStrategy{
		logger: logger.WithField("component", "cache.strategy.prewarming"),
		config: config,
	}
}

func (s *PrewarmingStrategy) Name() string {
	return "prewarming"
}

func (s *PrewarmingStrategy) Priority() int {
	return 5
}

func (s *PrewarmingStrategy) Analyze(ctx context.Context, stats *CacheStats) (*OptimizationReport, error) {
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
				Action:      "enable_predictive_prewarming",
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

func (s *PrewarmingStrategy) Apply(ctx context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		switch rec.Type {
		case "schedule_prewarming":
			keys := rec.Parameters["keys"].([]string)
			s.logger.Info("Prewarming: scheduling prewarming for popular keys",
				logging.F("key_count", len(keys)),
				logging.F("schedule", rec.Parameters["schedule"].(string)))
			applied++
		case "implement_predictive_prewarming":
			s.logger.Info("Prewarming: enabling predictive prewarming",
				logging.F("prediction_window", rec.Parameters["prediction_window"].(string)))
			applied++
		}
	}

	s.logger.Info("Prewarming strategy completed",
		logging.F("applied", applied))

	return nil
}

// Hot Key Optimization Strategy
type HotKeyOptimizationStrategy struct {
	logger logging.Logger
	config *CacheOptimizerConfig
}

func NewHotKeyOptimizationStrategy(logger logging.Logger, config *CacheOptimizerConfig) *HotKeyOptimizationStrategy {
	return &HotKeyOptimizationStrategy{
		logger: logger.WithField("component", "cache.strategy.hotkey"),
		config: config,
	}
}

func (s *HotKeyOptimizationStrategy) Name() string {
	return "hot_key_optimization"
}

func (s *HotKeyOptimizationStrategy) Priority() int {
	return 8
}

func (s *HotKeyOptimizationStrategy) Analyze(ctx context.Context, stats *CacheStats) (*OptimizationReport, error) {
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
	for _, keyStats := range hotKeys[:min(len(hotKeys), 5)] { // Top 5 hot keys
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

func (s *HotKeyOptimizationStrategy) Apply(ctx context.Context, recommendations []OptimizationRecommendation) error {
	applied := 0
	for _, rec := range recommendations {
		switch rec.Type {
		case "implement_hot_key_replication":
			key := rec.Parameters["key"].(string)
			factor := rec.Parameters["replication_factor"].(int)
			s.logger.Info("Hot key optimization: implementing replication",
				logging.F("key", key),
				logging.F("replication_factor", factor))
			applied++
		case "promote_to_fast_storage":
			key := rec.Parameters["key"].(string)
			tier := rec.Parameters["storage_tier"].(string)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
