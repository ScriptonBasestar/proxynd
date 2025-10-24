package ansible

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/observability/zap"
	"proxynd/internal/ports"
)

// ListCollections handles collection list request (GET /api/v3/collections/)
//
// Galaxy v3 API Specification:
// - Method: GET
// - Query params: namespace (optional), page (default 1), limit (default 10)
// - Response: CollectionListResponse with pagination
func (h *AnsibleHandler) ListCollections(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	namespace := c.Query("namespace", "")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	h.logger.Info(ctx, "Ansible collection list request",
		zap.NewField("repository", h.config.Name),
		zap.NewField("namespace", namespace),
		zap.NewField("page", page),
		zap.NewField("limit", limit),
	)

	// Build filters
	filters := make(map[string]string)
	if namespace != "" {
		filters["namespace"] = namespace
	}

	// Call HostedDriver.ListPackages
	req := &ports.ListPackagesRequest{
		Page:     page,
		PageSize: limit,
		Filters:  filters,
	}

	resp, err := h.hostedDriver.ListPackages(ctx, req)
	if err != nil {
		h.logger.Error(ctx, "Failed to list collections",
			zap.NewField("error", err.Error()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Detail: "Failed to retrieve collection list",
		})
	}

	// Build response
	items := make([]CollectionListItem, len(resp.Packages))
	for i, pkg := range resp.Packages {
		items[i] = CollectionListItem{
			Namespace:   pkg.Namespace,
			Name:        pkg.Name,
			Description: pkg.Description,
			Deprecated:  pkg.Deprecated,
			VersionsURL: fmt.Sprintf("/api/v3/collections/%s/%s/versions/",
				pkg.Namespace, pkg.Name),
		}
	}

	// Build pagination links
	baseURL := fmt.Sprintf("/api/v3/collections/?limit=%d", limit)
	if namespace != "" {
		baseURL += fmt.Sprintf("&namespace=%s", namespace)
	}

	var firstURL, prevURL, nextURL, lastURL *string

	// First page
	first := baseURL + "&page=1"
	firstURL = &first

	// Previous page
	if page > 1 {
		prev := fmt.Sprintf("%s&page=%d", baseURL, page-1)
		prevURL = &prev
	}

	// Next page
	if page < resp.TotalPages {
		next := fmt.Sprintf("%s&page=%d", baseURL, page+1)
		nextURL = &next
	}

	// Last page
	if resp.TotalPages > 0 {
		last := fmt.Sprintf("%s&page=%d", baseURL, resp.TotalPages)
		lastURL = &last
	}

	links := &PaginationLinks{
		First:    firstURL,
		Previous: prevURL,
		Next:     nextURL,
		Last:     lastURL,
	}

	meta := &PaginationMeta{
		Count: int64(resp.Total),
		Page:  page,
		Limit: limit,
	}

	response := CollectionListResponse{
		Data:  items,
		Links: links,
		Meta:  meta,
	}

	h.logger.Info(ctx, "Collection list retrieved",
		zap.NewField("count", len(items)),
		zap.NewField("total", resp.Total),
		zap.NewField("page", page),
	)

	return c.JSON(response)
}
