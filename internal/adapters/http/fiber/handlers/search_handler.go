package handlers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/ports"
	"proxynd/internal/usecase"
)

// SearchResult represents a search result item
type SearchResult = usecase.SearchResult

// GroupedSearchResult represents grouped search results for Maven
type GroupedSearchResult = usecase.GroupedSearchResult

// SearchResultMirror represents a mirror where the artifact is available
type SearchResultMirror = usecase.SearchResultMirror

// SearchResponse represents the API response
type SearchResponse = usecase.SearchResponse

// SearchHandler is a backward-compatible alias for SearchHandlerFunc
// Deprecated: This variable wrapper is for backward compatibility only.
// For new code, use NewSearchHandler() with dependency injection.
// Migration path: Router → SearchHandlerStruct (Phase 2b complete, 2025-11-27)
// Planned removal: Phase 3 (legacy cleanup, target v2.0.0)
var SearchHandler = SearchHandlerFunc

// SearchHandlerStruct handles content search across proxies using hexagonal architecture
type SearchHandlerStruct struct {
	*BaseHandler
	searchService *usecase.SearchService
}

// NewSearchHandler creates a new search handler with proper dependency injection
func NewSearchHandler(
	base *BaseHandler,
	searchService *usecase.SearchService,
) *SearchHandlerStruct {
	return &SearchHandlerStruct{
		BaseHandler:   base,
		searchService: searchService,
	}
}

// Handle implements ports.HTTPHandler for search requests
func (h *SearchHandlerStruct) Handle(ctx ports.HTTPContext) error {
	// Extract request parameters
	query := ctx.Query("q")
	proxyType := ctx.Query("type")
	if proxyType == "" {
		proxyType = "all" // default to all types
	}
	limit := 20 // default limit
	if limitStr := ctx.Query("limit"); limitStr != "" {
		// Parse limit from string
		// Note: In real implementation, we'd convert limitStr to int
		// For now using default
	}

	// Create search request
	searchReq := &usecase.SearchRequest{
		Query:     query,
		Type:      proxyType,
		Limit:     limit,
		UserAgent: ctx.UserAgent(),
		ClientIP:  ctx.ClientIP(),
	}

	// Call usecase service with proper dependency injection
	response, err := h.searchService.Search(ctx.Context(), searchReq)
	if err != nil {
		// Log error with injected logger
		h.logger.Error(ctx.Context(), "Search failed",
			NewField("error", err.Error()),
			NewField("query", query),
			NewField("type", proxyType),
		)

		// Convert usecase error to appropriate HTTP response
		if err.Error() == "empty search query" {
			return ctx.Status(400).JSON(response)
		}

		return ctx.Status(500).JSON(&usecase.SearchResponse{
			Error: "검색 중 오류가 발생했습니다",
		})
	}

	return ctx.JSON(response)
}

// SearchHandlerFunc is a backward-compatible function-based handler
// Deprecated: This function is for backward compatibility only.
// For new code, use SearchHandlerStruct with NewSearchHandler() and dependency injection.
// Migration path: Router → SearchHandlerStruct (Phase 2b complete, 2025-11-27)
// Planned removal: Phase 3 (legacy cleanup, target v2.0.0)
func SearchHandlerFunc(c *fiber.Ctx) error {
	// Extract request parameters
	query := c.Query("q")
	proxyType := c.Query("type", "all") // all, maven, apt, npm, etc.
	limit := c.QueryInt("limit", 20)    // default 20 results

	// Create search request
	searchReq := &usecase.SearchRequest{
		Query:     query,
		Type:      proxyType,
		Limit:     limit,
		UserAgent: c.Get("User-Agent"),
		ClientIP:  c.IP(),
	}

	// Create minimal service instance for backward compatibility
	// In production, this should be injected via dependency injection
	searchService := usecase.NewSearchService(nil, nil, nil, nil)

	// Call usecase service
	response, err := searchService.Search(c.Context(), searchReq)
	if err != nil {
		// Convert usecase error to appropriate HTTP response
		if err.Error() == "empty search query" {
			return c.Status(400).JSON(response)
		}

		return c.Status(500).JSON(&usecase.SearchResponse{
			Error: "검색 중 오류가 발생했습니다",
		})
	}

	return c.JSON(response)
}
