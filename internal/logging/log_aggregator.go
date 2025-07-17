package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogAggregator provides log analysis and aggregation capabilities
type LogAggregator struct {
	logger   Logger
	config   *AggregatorConfig
	patterns map[string]*regexp.Regexp
	mu       sync.RWMutex
}

// AggregatorConfig configures the log aggregator
type AggregatorConfig struct {
	LogPaths        []string          `yaml:"log_paths" json:"log_paths"`
	OutputPath      string            `yaml:"output_path" json:"output_path"`
	AnalysisWindow  time.Duration     `yaml:"analysis_window" json:"analysis_window"`
	Patterns        map[string]string `yaml:"patterns" json:"patterns"`
	MaxFileSize     int64             `yaml:"max_file_size" json:"max_file_size"`
	RetentionPeriod time.Duration     `yaml:"retention_period" json:"retention_period"`
}

// LogEntry represents a parsed log entry
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Component     string                 `json:"component"`
	Message       string                 `json:"message"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	Raw           string                 `json:"raw"`
	Source        string                 `json:"source"`
}

// LogAnalytics contains aggregated log analysis results
type LogAnalytics struct {
	TimeRange       TimeRange              `json:"time_range"`
	TotalEntries    int64                  `json:"total_entries"`
	LevelCounts     map[string]int64       `json:"level_counts"`
	ComponentCounts map[string]int64       `json:"component_counts"`
	ErrorPatterns   []ErrorPattern         `json:"error_patterns"`
	TopErrors       []TopError             `json:"top_errors"`
	RequestStats    RequestStats           `json:"request_stats"`
	SecurityEvents  []SecurityEventSummary `json:"security_events"`
	Performance     PerformanceSummary     `json:"performance"`
}

// TimeRange represents a time period
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ErrorPattern represents a detected error pattern
type ErrorPattern struct {
	Pattern     string    `json:"pattern"`
	Count       int64     `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Component   string    `json:"component,omitempty"`
	Severity    string    `json:"severity"`
	Description string    `json:"description,omitempty"`
}

// TopError represents frequently occurring errors
type TopError struct {
	Message   string    `json:"message"`
	Count     int64     `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
	Component string    `json:"component"`
	Examples  []string  `json:"examples"`
}

// RequestStats contains HTTP request statistics
type RequestStats struct {
	TotalRequests   int64            `json:"total_requests"`
	StatusCodes     map[string]int64 `json:"status_codes"`
	Methods         map[string]int64 `json:"methods"`
	TopEndpoints    []EndpointStat   `json:"top_endpoints"`
	AvgResponseTime time.Duration    `json:"avg_response_time"`
	SlowRequests    []SlowRequest    `json:"slow_requests"`
}

// EndpointStat represents endpoint usage statistics
type EndpointStat struct {
	Endpoint string        `json:"endpoint"`
	Count    int64         `json:"count"`
	AvgTime  time.Duration `json:"avg_time"`
}

// SlowRequest represents a slow HTTP request
type SlowRequest struct {
	Timestamp  time.Time     `json:"timestamp"`
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	Duration   time.Duration `json:"duration"`
	StatusCode int           `json:"status_code"`
	UserID     string        `json:"user_id,omitempty"`
	IPAddress  string        `json:"ip_address,omitempty"`
}

// SecurityEventSummary represents aggregated security events
type SecurityEventSummary struct {
	Type        string    `json:"type"`
	Count       int64     `json:"count"`
	LastSeen    time.Time `json:"last_seen"`
	TopSources  []string  `json:"top_sources"`
	ThreatLevel string    `json:"threat_level"`
}

// PerformanceSummary contains performance analysis
type PerformanceSummary struct {
	AverageResponseTime time.Duration      `json:"average_response_time"`
	P95ResponseTime     time.Duration      `json:"p95_response_time"`
	P99ResponseTime     time.Duration      `json:"p99_response_time"`
	SlowOperations      []SlowOperation    `json:"slow_operations"`
	MemoryUsage         MemoryUsageSummary `json:"memory_usage"`
}

// SlowOperation represents a slow operation
type SlowOperation struct {
	Operation string        `json:"operation"`
	Component string        `json:"component"`
	AvgTime   time.Duration `json:"avg_time"`
	Count     int64         `json:"count"`
}

// MemoryUsageSummary contains memory usage statistics
type MemoryUsageSummary struct {
	AverageHeapSize int64         `json:"average_heap_size"`
	MaxHeapSize     int64         `json:"max_heap_size"`
	GCCount         int64         `json:"gc_count"`
	GCTime          time.Duration `json:"gc_time"`
}

// NewLogAggregator creates a new log aggregator
func NewLogAggregator(logger Logger, config *AggregatorConfig) *LogAggregator {
	aggregator := &LogAggregator{
		logger:   logger.WithComponent("log.aggregator"),
		config:   config,
		patterns: make(map[string]*regexp.Regexp),
	}

	// Compile regex patterns
	for name, pattern := range config.Patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			aggregator.patterns[name] = compiled
		} else {
			logger.Error("Failed to compile regex pattern",
				String("pattern_name", name),
				String("pattern", pattern),
				Error(err))
		}
	}

	return aggregator
}

// AnalyzeLogs performs comprehensive log analysis
func (la *LogAggregator) AnalyzeLogs(ctx context.Context, timeRange TimeRange) (*LogAnalytics, error) {
	analytics := &LogAnalytics{
		TimeRange:       timeRange,
		LevelCounts:     make(map[string]int64),
		ComponentCounts: make(map[string]int64),
		RequestStats: RequestStats{
			StatusCodes: make(map[string]int64),
			Methods:     make(map[string]int64),
		},
	}

	// Process each log file
	for _, logPath := range la.config.LogPaths {
		if err := la.processLogFile(ctx, logPath, timeRange, analytics); err != nil {
			la.logger.Error("Failed to process log file",
				String("file", logPath),
				Error(err))
			continue
		}
	}

	// Analyze and summarize
	la.analyzeErrorPatterns(analytics)
	la.calculatePerformanceMetrics(analytics)
	la.identifySecurityEvents(analytics)

	return analytics, nil
}

// processLogFile processes a single log file
func (la *LogAggregator) processLogFile(ctx context.Context, filePath string, timeRange TimeRange, analytics *LogAnalytics) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line size

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if entry := la.parseLogEntry(line, filepath.Base(filePath)); entry != nil {
			// Filter by time range
			if entry.Timestamp.Before(timeRange.Start) || entry.Timestamp.After(timeRange.End) {
				continue
			}

			la.aggregateEntry(entry, analytics)
		}
	}

	return scanner.Err()
}

// parseLogEntry parses a log line into a LogEntry
func (la *LogAggregator) parseLogEntry(line, source string) *LogEntry {
	// Try parsing as JSON first
	var entry LogEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		entry.Raw = line
		entry.Source = source
		return &entry
	}

	// Fall back to pattern matching
	for patternName, pattern := range la.patterns {
		if matches := pattern.FindStringSubmatch(line); matches != nil {
			entry = LogEntry{
				Raw:    line,
				Source: source,
				Fields: make(map[string]interface{}),
			}

			// Extract fields based on pattern
			la.extractFieldsFromPattern(patternName, matches, &entry)
			return &entry
		}
	}

	// If no pattern matches, create basic entry
	return &LogEntry{
		Timestamp: time.Now(),
		Level:     "UNKNOWN",
		Message:   line,
		Raw:       line,
		Source:    source,
	}
}

// extractFieldsFromPattern extracts fields from regex matches
func (la *LogAggregator) extractFieldsFromPattern(patternName string, matches []string, entry *LogEntry) {
	// This is a simplified implementation
	// In practice, you'd have named capture groups in your regex patterns
	if len(matches) > 1 {
		entry.Message = matches[1]
	}
	if len(matches) > 2 {
		entry.Level = matches[2]
	}
	// Add more field extraction logic based on your log format
}

// aggregateEntry aggregates a log entry into analytics
func (la *LogAggregator) aggregateEntry(entry *LogEntry, analytics *LogAnalytics) {
	analytics.TotalEntries++

	// Count by level
	analytics.LevelCounts[entry.Level]++

	// Count by component
	if entry.Component != "" {
		analytics.ComponentCounts[entry.Component]++
	}

	// Aggregate HTTP request data
	if la.isHTTPRequestEntry(entry) {
		la.aggregateHTTPRequest(entry, analytics)
	}

	// Collect error information
	if entry.Level == "ERROR" || entry.Level == "FATAL" {
		la.collectErrorInfo(entry, analytics)
	}

	// Collect security events
	if la.isSecurityEvent(entry) {
		la.collectSecurityEvent(entry, analytics)
	}

	// Collect performance data
	if la.isPerformanceEvent(entry) {
		la.collectPerformanceData(entry, analytics)
	}
}

// analyzeErrorPatterns analyzes error patterns in the logs
func (la *LogAggregator) analyzeErrorPatterns(analytics *LogAnalytics) {
	errorCounts := make(map[string]*ErrorPattern)

	// Group similar errors (simplified implementation)
	for _, topError := range analytics.TopErrors {
		// Extract error pattern (simplified)
		pattern := la.extractErrorPattern(topError.Message)

		if existing, exists := errorCounts[pattern]; exists {
			existing.Count += topError.Count
			if topError.LastSeen.After(existing.LastSeen) {
				existing.LastSeen = topError.LastSeen
			}
		} else {
			errorCounts[pattern] = &ErrorPattern{
				Pattern:   pattern,
				Count:     topError.Count,
				LastSeen:  topError.LastSeen,
				FirstSeen: topError.LastSeen, // Simplified
				Component: topError.Component,
				Severity:  la.determineSeverity(topError.Count),
			}
		}
	}

	// Convert to slice and sort
	for _, pattern := range errorCounts {
		analytics.ErrorPatterns = append(analytics.ErrorPatterns, *pattern)
	}

	sort.Slice(analytics.ErrorPatterns, func(i, j int) bool {
		return analytics.ErrorPatterns[i].Count > analytics.ErrorPatterns[j].Count
	})
}

// isHTTPRequestEntry checks if entry is an HTTP request log
func (la *LogAggregator) isHTTPRequestEntry(entry *LogEntry) bool {
	return strings.Contains(entry.Message, "HTTP request") ||
		entry.Component == "http.middleware" ||
		(entry.Fields != nil && entry.Fields["method"] != nil)
}

// aggregateHTTPRequest aggregates HTTP request data
func (la *LogAggregator) aggregateHTTPRequest(entry *LogEntry, analytics *LogAnalytics) {
	analytics.RequestStats.TotalRequests++

	// Extract HTTP fields
	if entry.Fields != nil {
		if method, ok := entry.Fields["method"].(string); ok {
			analytics.RequestStats.Methods[method]++
		}

		if statusCode, ok := entry.Fields["status_code"].(float64); ok {
			statusStr := fmt.Sprintf("%.0f", statusCode)
			analytics.RequestStats.StatusCodes[statusStr]++
		}

		// Collect slow requests
		if duration, ok := entry.Fields["duration"].(string); ok {
			if d, err := time.ParseDuration(duration); err == nil && d > 2*time.Second {
				slowReq := SlowRequest{
					Timestamp: entry.Timestamp,
					Duration:  d,
					UserID:    entry.UserID,
					IPAddress: entry.IPAddress,
				}

				if method, ok := entry.Fields["method"].(string); ok {
					slowReq.Method = method
				}
				if path, ok := entry.Fields["path"].(string); ok {
					slowReq.Path = path
				}
				if statusCode, ok := entry.Fields["status_code"].(float64); ok {
					slowReq.StatusCode = int(statusCode)
				}

				analytics.RequestStats.SlowRequests = append(analytics.RequestStats.SlowRequests, slowReq)
			}
		}
	}
}

// collectErrorInfo collects error information
func (la *LogAggregator) collectErrorInfo(entry *LogEntry, analytics *LogAnalytics) {
	// Group similar error messages
	errorKey := la.normalizeErrorMessage(entry.Message)

	// Find existing top error or create new one
	found := false
	for i := range analytics.TopErrors {
		if analytics.TopErrors[i].Message == errorKey {
			analytics.TopErrors[i].Count++
			analytics.TopErrors[i].LastSeen = entry.Timestamp
			if len(analytics.TopErrors[i].Examples) < 5 {
				analytics.TopErrors[i].Examples = append(analytics.TopErrors[i].Examples, entry.Raw)
			}
			found = true
			break
		}
	}

	if !found {
		analytics.TopErrors = append(analytics.TopErrors, TopError{
			Message:   errorKey,
			Count:     1,
			LastSeen:  entry.Timestamp,
			Component: entry.Component,
			Examples:  []string{entry.Raw},
		})
	}
}

// isSecurityEvent checks if entry is a security event
func (la *LogAggregator) isSecurityEvent(entry *LogEntry) bool {
	securityKeywords := []string{
		"authentication failed", "unauthorized", "security", "attack",
		"malware", "injection", "xss", "csrf", "brute force",
	}

	message := strings.ToLower(entry.Message)
	for _, keyword := range securityKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}

	return entry.Component == "security" || entry.Component == "audit"
}

// collectSecurityEvent collects security event data
func (la *LogAggregator) collectSecurityEvent(entry *LogEntry, analytics *LogAnalytics) {
	eventType := la.extractSecurityEventType(entry)

	// Find existing security event or create new one
	found := false
	for i := range analytics.SecurityEvents {
		if analytics.SecurityEvents[i].Type == eventType {
			analytics.SecurityEvents[i].Count++
			analytics.SecurityEvents[i].LastSeen = entry.Timestamp
			// Add IP to top sources if not already present
			if entry.IPAddress != "" && len(analytics.SecurityEvents[i].TopSources) < 10 {
				exists := false
				for _, source := range analytics.SecurityEvents[i].TopSources {
					if source == entry.IPAddress {
						exists = true
						break
					}
				}
				if !exists {
					analytics.SecurityEvents[i].TopSources = append(analytics.SecurityEvents[i].TopSources, entry.IPAddress)
				}
			}
			found = true
			break
		}
	}

	if !found {
		sources := []string{}
		if entry.IPAddress != "" {
			sources = append(sources, entry.IPAddress)
		}

		analytics.SecurityEvents = append(analytics.SecurityEvents, SecurityEventSummary{
			Type:        eventType,
			Count:       1,
			LastSeen:    entry.Timestamp,
			TopSources:  sources,
			ThreatLevel: la.determineThreatLevel(eventType),
		})
	}
}

// isPerformanceEvent checks if entry contains performance data
func (la *LogAggregator) isPerformanceEvent(entry *LogEntry) bool {
	return entry.Component == "performance" ||
		strings.Contains(entry.Message, "duration") ||
		strings.Contains(entry.Message, "slow") ||
		(entry.Fields != nil && entry.Fields["duration"] != nil)
}

// collectPerformanceData collects performance metrics
func (la *LogAggregator) collectPerformanceData(entry *LogEntry, analytics *LogAnalytics) {
	if entry.Fields != nil {
		// Collect response times for calculation
		if duration, ok := entry.Fields["duration"].(string); ok {
			if _, err := time.ParseDuration(duration); err == nil {
				// This would typically be stored for percentile calculations
				// Simplified implementation here
			}
		}

		// Collect slow operations
		if component, ok := entry.Fields["component"].(string); ok {
			if operation, ok := entry.Fields["operation"].(string); ok {
				if duration, ok := entry.Fields["duration"].(string); ok {
					if d, err := time.ParseDuration(duration); err == nil && d > time.Second {
						slowOp := SlowOperation{
							Operation: operation,
							Component: component,
							AvgTime:   d,
							Count:     1,
						}
						analytics.Performance.SlowOperations = append(analytics.Performance.SlowOperations, slowOp)
					}
				}
			}
		}
	}
}

// calculatePerformanceMetrics calculates performance summary
func (la *LogAggregator) calculatePerformanceMetrics(analytics *LogAnalytics) {
	// This would typically involve statistical calculations on collected data
	// Simplified implementation here

	if len(analytics.RequestStats.SlowRequests) > 0 {
		var totalDuration time.Duration
		for _, req := range analytics.RequestStats.SlowRequests {
			totalDuration += req.Duration
		}
		analytics.Performance.AverageResponseTime = totalDuration / time.Duration(len(analytics.RequestStats.SlowRequests))
	}

	// Group slow operations by operation type
	opCounts := make(map[string]*SlowOperation)
	for _, op := range analytics.Performance.SlowOperations {
		key := op.Component + ":" + op.Operation
		if existing, exists := opCounts[key]; exists {
			existing.Count++
			existing.AvgTime = (existing.AvgTime + op.AvgTime) / 2
		} else {
			opCounts[key] = &op
		}
	}

	// Convert back to slice
	analytics.Performance.SlowOperations = make([]SlowOperation, 0, len(opCounts))
	for _, op := range opCounts {
		analytics.Performance.SlowOperations = append(analytics.Performance.SlowOperations, *op)
	}
}

// identifySecurityEvents identifies security patterns
func (la *LogAggregator) identifySecurityEvents(analytics *LogAnalytics) {
	// Sort security events by count
	sort.Slice(analytics.SecurityEvents, func(i, j int) bool {
		return analytics.SecurityEvents[i].Count > analytics.SecurityEvents[j].Count
	})

	// Limit to top 20 security events
	if len(analytics.SecurityEvents) > 20 {
		analytics.SecurityEvents = analytics.SecurityEvents[:20]
	}
}

// Helper methods

func (la *LogAggregator) extractErrorPattern(message string) string {
	// Simplified pattern extraction - replace specific details with placeholders
	pattern := message

	// Replace UUIDs
	uuidRegex := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	pattern = uuidRegex.ReplaceAllString(pattern, "{UUID}")

	// Replace IPs
	ipRegex := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	pattern = ipRegex.ReplaceAllString(pattern, "{IP}")

	// Replace numbers
	numberRegex := regexp.MustCompile(`\d+`)
	pattern = numberRegex.ReplaceAllString(pattern, "{NUMBER}")

	return pattern
}

func (la *LogAggregator) normalizeErrorMessage(message string) string {
	// Remove timestamps, IDs, and other variable parts
	return la.extractErrorPattern(message)
}

func (la *LogAggregator) extractSecurityEventType(entry *LogEntry) string {
	message := strings.ToLower(entry.Message)

	securityTypes := map[string]string{
		"authentication failed": "auth_failure",
		"unauthorized":          "unauthorized_access",
		"brute force":           "brute_force",
		"injection":             "injection_attempt",
		"xss":                   "xss_attempt",
		"malware":               "malware_detection",
		"rate limit":            "rate_limit_exceeded",
	}

	for keyword, eventType := range securityTypes {
		if strings.Contains(message, keyword) {
			return eventType
		}
	}

	return "security_event"
}

func (la *LogAggregator) determineSeverity(count int64) string {
	if count > 100 {
		return "HIGH"
	} else if count > 10 {
		return "MEDIUM"
	}
	return "LOW"
}

func (la *LogAggregator) determineThreatLevel(eventType string) string {
	highThreatEvents := map[string]bool{
		"injection_attempt": true,
		"malware_detection": true,
		"brute_force":       true,
	}

	if highThreatEvents[eventType] {
		return "HIGH"
	}
	return "MEDIUM"
}

// ExportAnalytics exports analytics to a file
func (la *LogAggregator) ExportAnalytics(analytics *LogAnalytics, outputPath string) error {
	data, err := json.MarshalIndent(analytics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analytics: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write analytics file: %w", err)
	}

	return nil
}

// StartPeriodicAnalysis starts periodic log analysis
func (la *LogAggregator) StartPeriodicAnalysis(ctx context.Context) {
	ticker := time.NewTicker(la.config.AnalysisWindow)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			endTime := time.Now()
			startTime := endTime.Add(-la.config.AnalysisWindow)

			analytics, err := la.AnalyzeLogs(ctx, TimeRange{
				Start: startTime,
				End:   endTime,
			})

			if err != nil {
				la.logger.Error("Failed to analyze logs", Error(err))
				continue
			}

			// Export analytics
			outputFile := filepath.Join(la.config.OutputPath,
				fmt.Sprintf("analytics_%s.json", endTime.Format("2006-01-02_15-04-05")))

			if err := la.ExportAnalytics(analytics, outputFile); err != nil {
				la.logger.Error("Failed to export analytics",
					String("output_file", outputFile),
					Error(err))
			} else {
				la.logger.Info("Log analytics exported",
					String("output_file", outputFile),
					NewField("total_entries", analytics.TotalEntries),
					NewField("error_count", len(analytics.TopErrors)),
					NewField("security_events", len(analytics.SecurityEvents)))
			}
		}
	}
}
