package enterprise

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"proxynd/internal/domain/enterprise"
	"strconv"
	"time"
)

// AnalyticsHandler handles analytics and reporting API requests
type AnalyticsHandler struct{}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler() *AnalyticsHandler {
	return &AnalyticsHandler{}
}

// GetOverview godoc
// @Summary Get analytics overview
// @Description Get dashboard overview with key metrics and statistics
// @Tags Analytics
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Dashboard overview with metrics"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/overview [get]
func (h *AnalyticsHandler) GetOverview(c *fiber.Ctx) error {
	overview := &enterprise.DashboardOverview{
		TotalRequests:   125340,
		CacheHitRate:    87.5,
		AvgResponseTime: 145.2,
		BandwidthUsed:   5368709120, // ~5GB
		ActiveUsers:     47,
		TopPackages: []enterprise.PackageUsage{
			{PackageManager: "npm", PackageName: "react", RequestCount: 5420, BytesServed: 104857600},
			{PackageManager: "maven", PackageName: "spring-boot-starter", RequestCount: 3210, BytesServed: 209715200},
			{PackageManager: "pypi", PackageName: "requests", RequestCount: 2890, BytesServed: 52428800},
		},
		RequestsByPM: map[string]int64{
			"npm":    54200,
			"maven":  38100,
			"pypi":   21540,
			"docker": 8900,
			"apt":    2600,
		},
		ErrorRate: 0.45,
		Timestamp: time.Now().UTC(),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    overview,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetUsageStats godoc
// @Summary Get usage statistics
// @Description Get detailed usage statistics broken down by package manager
// @Tags Analytics
// @Accept json
// @Produce json
// @Param period query string false "Time period (7d, 30d, 90d)" default(30d)
// @Success 200 {object} map[string]interface{} "Usage statistics"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/usage [get]
func (h *AnalyticsHandler) GetUsageStats(c *fiber.Ctx) error {
	timeRange := c.Query("range", "last_7_days")

	stats := &enterprise.UsageStats{
		TimeRange:      timeRange,
		TotalRequests:  125340,
		UniquePackages: 892,
		UniqueUsers:    47,
		ByPackageManager: map[string]enterprise.PMStats{
			"npm": {
				Name:        "NPM",
				Requests:    54200,
				CacheHits:   48900,
				CacheMisses: 5300,
				HitRate:     90.2,
				BytesServed: 2147483648,
			},
			"maven": {
				Name:        "Maven",
				Requests:    38100,
				CacheHits:   32400,
				CacheMisses: 5700,
				HitRate:     85.0,
				BytesServed: 1879048192,
			},
		},
		ByUser: []enterprise.UserStats{
			{UserID: "user_001", UserEmail: "dev1@example.com", Requests: 12340, BytesUsed: 524288000},
			{UserID: "user_002", UserEmail: "dev2@example.com", Requests: 9870, BytesUsed: 419430400},
		},
		ByTimeOfDay: map[string]int64{
			"00-03": 2100,
			"03-06": 1850,
			"06-09": 8900,
			"09-12": 42300,
			"12-15": 38200,
			"15-18": 25600,
			"18-21": 5200,
			"21-24": 1190,
		},
		Trends: generateTrends(7),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetPerformanceMetrics godoc
// @Summary Get performance metrics
// @Description Get detailed performance metrics including response times and percentiles
// @Tags Analytics
// @Accept json
// @Produce json
// @Param period query string false "Time period (1h, 24h, 7d)" default(24h)
// @Success 200 {object} map[string]interface{} "Performance metrics"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/performance [get]
func (h *AnalyticsHandler) GetPerformanceMetrics(c *fiber.Ctx) error {
	metrics := &enterprise.PerformanceMetrics{
		TimeRange:     "last_24_hours",
		AvgLatency:    145.2,
		P50Latency:    98.5,
		P95Latency:    320.8,
		P99Latency:    567.2,
		Throughput:    45.6,
		ErrorRate:     0.45,
		ByEndpoint: map[string]enterprise.EndpointMetrics{
			"/npm/*": {
				Endpoint:   "/npm/*",
				Requests:   54200,
				AvgLatency: 120.5,
				P95Latency: 280.3,
				ErrorCount: 120,
				ErrorRate:  0.22,
			},
			"/maven/*": {
				Endpoint:   "/maven/*",
				Requests:   38100,
				AvgLatency: 185.7,
				P95Latency: 410.2,
				ErrorCount: 210,
				ErrorRate:  0.55,
			},
		},
		LatencyTrends: generateTrends(24),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    metrics,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetCacheEfficiency godoc
// @Summary Get cache efficiency metrics
// @Description Get cache hit rates, bandwidth savings, and efficiency statistics
// @Tags Analytics
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Cache efficiency metrics"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/cache-efficiency [get]
func (h *AnalyticsHandler) GetCacheEfficiency(c *fiber.Ctx) error {
	efficiency := &enterprise.CacheEfficiency{
		TimeRange:      "last_7_days",
		OverallHitRate: 87.5,
		TotalHits:      109500,
		TotalMisses:    15840,
		CacheSizeBytes: 10737418240, // ~10GB
		EvictionCount:  342,
		ByPackageManager: map[string]float64{
			"npm":    90.2,
			"maven":  85.0,
			"pypi":   88.5,
			"docker": 82.3,
		},
		HitRateTrends: generateTrends(7),
		TopCachedPackages: []enterprise.PackageUsage{
			{PackageManager: "npm", PackageName: "react", RequestCount: 5420, BytesServed: 104857600},
			{PackageManager: "maven", PackageName: "spring-boot", RequestCount: 3210, BytesServed: 209715200},
		},
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    efficiency,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// ListReports godoc
// @Summary List saved reports
// @Description Get a paginated list of saved analytics reports
// @Tags Analytics
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param type query string false "Report type filter"
// @Success 200 {object} map[string]interface{} "Paginated list of reports"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/reports [get]
func (h *AnalyticsHandler) ListReports(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	if perPage > 100 {
		perPage = 100
	}

	reports := []enterprise.Report{
		{
			ID:          "report_001",
			Name:        "Weekly Usage Report",
			Description: "Weekly package usage statistics",
			Type:        "usage",
			Schedule:    "0 0 * * 1",
			CreatedBy:   "admin@example.com",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour).UTC(),
			UpdatedAt:   time.Now().Add(-7 * 24 * time.Hour).UTC(),
		},
		{
			ID:          "report_002",
			Name:        "Security Scan Summary",
			Description: "Monthly security scan results",
			Type:        "security",
			Schedule:    "0 0 1 * *",
			CreatedBy:   "security@example.com",
			CreatedAt:   time.Now().Add(-60 * 24 * time.Hour).UTC(),
			UpdatedAt:   time.Now().Add(-15 * 24 * time.Hour).UTC(),
		},
	}

	total := int64(len(reports))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       reports,
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// CreateReport godoc
// @Summary Create custom report
// @Description Create a new custom analytics report
// @Tags Analytics
// @Accept json
// @Produce json
// @Param report body object{name=string,type=string,parameters=object} true "Report creation request"
// @Success 201 {object} map[string]interface{} "Created report"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/reports [post]
func (h *AnalyticsHandler) CreateReport(c *fiber.Ctx) error {
	var req enterprise.Report
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	report := &enterprise.Report{
		ID:          fmt.Sprintf("report_%d", time.Now().Unix()),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Schedule:    req.Schedule,
		Filters:     req.Filters,
		CreatedBy:   "current_user@example.com",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"data":    report,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetReport godoc
// @Summary Get report data
// @Description Get the data for a specific saved report
// @Tags Analytics
// @Accept json
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} map[string]interface{} "Report data"
// @Failure 404 {object} map[string]interface{} "Report not found"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/reports/{id} [get]
func (h *AnalyticsHandler) GetReport(c *fiber.Ctx) error {
	id := c.Params("id")

	// Mock report data
	report := fiber.Map{
		"id":          id,
		"name":        "Weekly Usage Report",
		"type":        "usage",
		"generated_at": time.Now().UTC(),
		"data": fiber.Map{
			"total_requests": 125340,
			"unique_users":   47,
			"top_packages":   3,
		},
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    report,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// DeleteReport godoc
// @Summary Delete report
// @Description Delete a saved analytics report
// @Tags Analytics
// @Accept json
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} map[string]interface{} "Deletion confirmation"
// @Failure 404 {object} map[string]interface{} "Report not found"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/reports/{id} [delete]
func (h *AnalyticsHandler) DeleteReport(c *fiber.Ctx) error {
	id := c.Params("id")

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":      id,
			"deleted": true,
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetCostAnalysis godoc
// @Summary Get cost analysis
// @Description Get bandwidth cost breakdown and savings analysis
// @Tags Analytics
// @Accept json
// @Produce json
// @Param period query string false "Analysis period (30d, 90d, 1y)" default(30d)
// @Success 200 {object} map[string]interface{} "Cost analysis"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/costs [get]
func (h *AnalyticsHandler) GetCostAnalysis(c *fiber.Ctx) error {
	analysis := &enterprise.CostAnalysis{
		TimeRange:     "last_30_days",
		TotalCost:     1234.56,
		BandwidthCost: 567.89,
		StorageCost:   432.10,
		ComputeCost:   234.57,
		ByPackageManager: map[string]float64{
			"npm":    456.78,
			"maven":  389.45,
			"pypi":   234.12,
			"docker": 154.21,
		},
		ByUser: map[string]float64{
			"team_frontend":  345.67,
			"team_backend":   456.78,
			"team_devops":    234.56,
		},
		CostTrends: generateTrends(30),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    analysis,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetTrends godoc
// @Summary Get trend analysis
// @Description Get trend analysis for requests, bandwidth, and performance
// @Tags Analytics
// @Accept json
// @Produce json
// @Param metric query string false "Metric to analyze (requests, bandwidth, latency)"
// @Param period query string false "Period for analysis (7d, 30d, 90d)" default(30d)
// @Success 200 {object} map[string]interface{} "Trend analysis"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/analytics/trends [get]
func (h *AnalyticsHandler) GetTrends(c *fiber.Ctx) error {
	metric := c.Query("metric", "requests")
	days, _ := strconv.Atoi(c.Query("days", "30"))

	trends := generateTrends(days)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"metric":     metric,
			"time_range": fmt.Sprintf("last_%d_days", days),
			"trends":     trends,
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// Helper function to generate trend data
func generateTrends(days int) []enterprise.TrendDataPoint {
	trends := make([]enterprise.TrendDataPoint, days)
	now := time.Now().UTC()

	for i := 0; i < days; i++ {
		trends[i] = enterprise.TrendDataPoint{
			Timestamp: now.Add(-time.Duration(days-i-1) * 24 * time.Hour),
			Value:     float64(10000 + (i * 500)),
			Label:     fmt.Sprintf("Day %d", i+1),
		}
	}

	return trends
}
