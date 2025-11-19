package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
	"proxynd/plugins"
)

// PluginHealthHandler handles plugin health check endpoints
type PluginHealthHandler struct {
	logger logging.Logger
}

// NewPluginHealthHandler creates a new plugin health handler
func NewPluginHealthHandler() *PluginHealthHandler {
	return &PluginHealthHandler{
		logger: logging.GetLogger(),
	}
}

// PluginHealthResponse represents the health status of a single plugin
type PluginHealthResponse struct {
	Name            string    `json:"name"`
	Enabled         bool      `json:"enabled"`
	Status          string    `json:"status"`
	Priority        int       `json:"priority"`
	ErrorCount      int       `json:"error_count"`
	LastError       string    `json:"last_error,omitempty"`
	LastErrorTime   *string   `json:"last_error_time,omitempty"`
	InitTimeMs      int64     `json:"init_time_ms"`
	ReadyTimeMs     int64     `json:"ready_time_ms"`
	StateChanges    int       `json:"state_changes"`
	LastStateChange time.Time `json:"last_state_change"`
}

// GetPluginHealth returns comprehensive health status of the plugin system
// @Summary      Get plugin health status
// @Description  Returns detailed health information for all plugins including status, errors, and timing metrics
// @Tags         plugins
// @Produce      json
// @Success      200 {object} object{enabled=bool,initialized=bool,ready=bool,plugins=[]PluginHealthResponse,summary=object,event_metrics=object} "Plugin health status"
// @Success      503 {object} object{enabled=bool,message=string} "Plugin system not available"
// @Router       /api/v1/plugins/health [get]
func (h *PluginHealthHandler) GetPluginHealth(c *fiber.Ctx) error {
	// Get plugin manager from context
	manager, ok := c.Locals("pluginManager").(*plugins.Manager)
	if !ok || manager == nil {
		h.logger.Warn("Plugin manager not found in context")
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"enabled": false,
			"message": "Plugin system not available",
		})
	}

	// Get overall status
	enabled := true // If manager exists, system is enabled
	initialized := manager.IsInitialized()
	ready := manager.IsReady()

	// Get all plugins
	allPlugins := manager.GetPlugins()
	metricsCollector := manager.GetMetrics()

	// Build plugin health responses
	pluginHealthList := make([]PluginHealthResponse, 0, len(allPlugins))
	for _, pluginCfg := range allPlugins {
		metrics := metricsCollector.GetPluginMetrics(pluginCfg.Name)
		if metrics == nil {
			// Plugin has no metrics yet, create minimal response
			pluginHealthList = append(pluginHealthList, PluginHealthResponse{
				Name:            pluginCfg.Name,
				Enabled:         pluginCfg.Enabled,
				Status:          "unknown",
				Priority:        pluginCfg.Priority,
				ErrorCount:      0,
				InitTimeMs:      0,
				ReadyTimeMs:     0,
				StateChanges:    0,
				LastStateChange: time.Time{},
			})
			continue
		}

		// Convert metrics to response format
		health := PluginHealthResponse{
			Name:            metrics.PluginName,
			Enabled:         metrics.Enabled,
			Status:          string(metrics.Status),
			Priority:        metrics.Priority,
			ErrorCount:      metrics.ErrorCount,
			LastError:       metrics.LastError,
			InitTimeMs:      metrics.InitTime.Milliseconds(),
			ReadyTimeMs:     metrics.ReadyTime.Milliseconds(),
			StateChanges:    metrics.StateChanges,
			LastStateChange: metrics.LastStateChange,
		}

		// Add last error time if present
		if !metrics.LastErrorTime.IsZero() {
			lastErrorTimeStr := metrics.LastErrorTime.Format(time.RFC3339)
			health.LastErrorTime = &lastErrorTimeStr
		}

		pluginHealthList = append(pluginHealthList, health)
	}

	// Get summary metrics
	summary := manager.GetMetricsSummary()

	// Get event metrics summary
	eventMetrics := plugins.GetEventMetricsSummary()

	// Build response
	response := fiber.Map{
		"enabled":         enabled,
		"initialized":     initialized,
		"ready":           ready,
		"total_plugins":   summary.TotalPlugins,
		"enabled_plugins": summary.EnabledPlugins,
		"ready_plugins":   summary.ReadyPlugins,
		"failed_plugins":  summary.FailedPlugins,
		"plugins":         pluginHealthList,
		"summary": fiber.Map{
			"total_errors":           summary.TotalErrors,
			"total_state_changes":    summary.TotalStateChanges,
			"avg_init_time_ms":       summary.AverageInitTime.Milliseconds(),
			"avg_ready_time_ms":      summary.AverageReadyTime.Milliseconds(),
			"total_init_time_ms":     summary.TotalInitTime.Milliseconds(),
			"total_ready_time_ms":    summary.TotalReadyTime.Milliseconds(),
			"total_shutdown_time_ms": summary.TotalShutdownTime.Milliseconds(),
		},
		"event_metrics": eventMetrics,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	h.logger.Debug("Plugin health check requested",
		logging.F("total_plugins", summary.TotalPlugins),
		logging.F("ready_plugins", summary.ReadyPlugins),
		logging.F("failed_plugins", summary.FailedPlugins))

	return c.JSON(response)
}
