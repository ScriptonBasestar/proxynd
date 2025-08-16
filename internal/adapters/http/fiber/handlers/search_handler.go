package handlers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/usecase"
)

// TODO: HEXAGONAL_MIGRATION - These types are duplicated in usecase package
// They should be imported from usecase or moved to a shared package
// Keeping them here temporarily for backward compatibility

// SearchResult represents a search result item
type SearchResult = usecase.SearchResult

// GroupedSearchResult represents grouped search results for Maven
type GroupedSearchResult = usecase.GroupedSearchResult

// SearchResultMirror represents a mirror where the artifact is available
type SearchResultMirror = usecase.SearchResultMirror

// SearchResponse represents the API response
type SearchResponse = usecase.SearchResponse

// SearchHandler handles content search across proxies using hexagonal architecture
func SearchHandler(c *fiber.Ctx) error {
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

	// TODO: HEXAGONAL_MIGRATION - Inject SearchService via dependency injection
	// For now, create a minimal service instance with nil logger to avoid interface mismatch
	// The search service will handle nil logger gracefully
	searchService := usecase.NewSearchService(nil, nil, nil, nil)

	// Call usecase service
	response, err := searchService.Search(c.Context(), searchReq)
	if err != nil {
		// Convert usecase error to appropriate HTTP response
		if err.Error() == "empty search query" {
			return c.Status(400).JSON(response)
		}

		// TODO: HEXAGONAL_MIGRATION - Add proper error logging via injected logger
		return c.Status(500).JSON(&usecase.SearchResponse{
			Error: "검색 중 오류가 발생했습니다",
		})
	}

	return c.JSON(response)
}

// TODO: HEXAGONAL_MIGRATION - All search business logic has been moved to usecase.SearchService
// This file now contains only the HTTP adapter logic
