//go:build darwin
// +build darwin

// Package health provides system resource health checking functionality for macOS
package health

import (
	"context"
	"fmt"
	"time"
)

// getSystemLoad returns system load averages (macOS implementation)
//
//nolint:unused // Reserved for future macOS system monitoring implementation
func getSystemLoad() (float64, float64, float64, error) {
	// TODO: Implement macOS-specific load average retrieval
	// For now, return zero values to avoid build failures
	return 0.0, 0.0, 0.0, fmt.Errorf("system load monitoring not implemented for macOS")
}

// SystemResourceHealthChecker provides macOS-compatible system health checking
type SystemResourceHealthChecker struct {
	name            string
	cpuThreshold    float64
	memoryThreshold float64
	diskThreshold   float64
	checkPaths      []string
}

// NewSystemResourceHealthChecker creates a new system health checker for macOS
func NewSystemResourceHealthChecker(name string, checkPaths []string) *SystemResourceHealthChecker {
	return &SystemResourceHealthChecker{
		name:            name,
		cpuThreshold:    80.0,
		memoryThreshold: 85.0,
		diskThreshold:   90.0,
		checkPaths:      checkPaths,
	}
}

// Name returns the name of this health checker
func (s *SystemResourceHealthChecker) Name() string {
	return s.name
}

// Check performs system health check (macOS implementation)
func (s *SystemResourceHealthChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	// For macOS, provide basic health check without syscall dependencies
	return &CheckResult{
		Name:        s.Name(),
		Status:      StatusHealthy,
		Message:     "System health check (macOS - limited functionality)",
		Duration:    time.Since(start),
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"platform": "darwin",
			"note":     "Full system monitoring requires Linux-specific syscalls",
		},
	}
}

// GetResourceSummary returns a summary of system resource usage (macOS stub)
func (s *SystemResourceHealthChecker) GetResourceSummary() map[string]interface{} {
	return map[string]interface{}{
		"platform":     "darwin",
		"cpu_usage":    0.0,
		"memory_usage": 0.0,
		"disk_usage":   0.0,
		"note":         "Resource monitoring not implemented for macOS",
	}
}

// GetHistoryData returns historical resource usage data (macOS stub)
func (s *SystemResourceHealthChecker) GetHistoryData(hours int) map[string]interface{} {
	return map[string]interface{}{
		"platform": "darwin",
		"hours":    hours,
		"history":  []interface{}{},
		"note":     "History tracking not implemented for macOS",
	}
}

// PredictResourceExhaustion predicts when resources might be exhausted (macOS stub)
func (s *SystemResourceHealthChecker) PredictResourceExhaustion() map[string]interface{} {
	return map[string]interface{}{
		"platform":     "darwin",
		"prediction":   "Not available",
		"cpu_hours":    -1,
		"memory_hours": -1,
		"disk_hours":   -1,
		"note":         "Prediction not implemented for macOS",
	}
}
