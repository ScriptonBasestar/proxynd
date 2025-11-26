package handlers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
	"proxynd/internal/metrics"
)

// MetricsDashboardHandler handles dashboard metrics requests
type MetricsDashboardHandler struct {
	aggregator *metrics.MetricsAggregator
	logger     logging.Logger
}

// NewMetricsDashboardHandler creates a new metrics dashboard handler
func NewMetricsDashboardHandler(logger logging.Logger) *MetricsDashboardHandler {
	return &MetricsDashboardHandler{
		aggregator: metrics.GetAggregator(),
		logger:     logger,
	}
}

// GetDashboardMetrics handles GET /api/v1/metrics/dashboard
func (h *MetricsDashboardHandler) GetDashboardMetrics(c *fiber.Ctx) error {
	// Get aggregated metrics
	dashboard, err := h.aggregator.GetDashboardMetrics()
	if err != nil {
		h.logger.Error("Failed to get dashboard metrics",
			"error", err,
			"remote_ip", c.IP(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve dashboard metrics",
		})
	}

	// Return JSON response
	return c.JSON(dashboard)
}
