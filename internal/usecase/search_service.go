package usecase

import (
	"context"
	"fmt"
	"strings"

	"proxynd/internal/adapters/pm/common"
	"proxynd/internal/config"
	"proxynd/internal/ports"
)

// TODO: HEXAGONAL_MIGRATION - NewField and LogField are shared with health.go
// They should be moved to a common package or the ports package

// SearchService implements search business logic
type SearchService struct {
	packageManager ports.PackageManager
	cacheManager   ports.CacheManager
	logger         ports.Logger
	metrics        ports.MetricsCollector
}

// NewSearchService creates a new search service
func NewSearchService(
	pm ports.PackageManager,
	cache ports.CacheManager,
	logger ports.Logger,
	metrics ports.MetricsCollector,
) *SearchService {
	return &SearchService{
		packageManager: pm,
		cacheManager:   cache,
		logger:         logger,
		metrics:        metrics,
	}
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query     string
	Type      string // all, maven, apt, npm, etc.
	Limit     int
	UserAgent string
	ClientIP  string
}

// SearchResult represents a search result item
type SearchResult struct {
	Type        string `json:"type"`        // maven, apt, npm, etc.
	Name        string `json:"name"`        // package/artifact name
	Version     string `json:"version"`     // version if available
	Description string `json:"description"` // description if available
	URL         string `json:"url"`         // direct URL to the item
	ProxyURL    string `json:"proxy_url"`   // URL through this proxy
	Source      string `json:"source"`      // which proxy server provided this
}

// GroupedSearchResult represents grouped search results for Maven
type GroupedSearchResult struct {
	GroupID     string               `json:"group_id"`
	ArtifactID  string               `json:"artifact_id"`
	Version     string               `json:"version"`
	Description string               `json:"description"`
	ProxyURL    string               `json:"proxy_url"` // 공통 프록시 URL
	Sources     []SearchResultMirror `json:"sources"`   // 미러별 원본 URL들
}

// SearchResultMirror represents a mirror where the artifact is available
type SearchResultMirror struct {
	Name string `json:"name"` // mirror name
	URL  string `json:"url"`  // direct URL to mirror
}

// SearchResponse represents the search response
type SearchResponse struct {
	Query          string                `json:"query"`
	ResultCount    int                   `json:"result_count"`
	Results        []SearchResult        `json:"results,omitempty"`         // for APT and other types
	GroupedResults []GroupedSearchResult `json:"grouped_results,omitempty"` // for Maven
	Error          string                `json:"error,omitempty"`
}

// Search performs package search across different proxy types
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req.Query == "" {
		return &SearchResponse{
			Error: "검색어가 필요합니다",
		}, fmt.Errorf("empty search query")
	}

	if req.Limit <= 0 {
		req.Limit = 20 // default limit
	}

	// Log search request
	if s.logger != nil {
		s.logger.Info(ctx, "프록시 컨텐츠 검색 요청",
			NewField("query", req.Query),
			NewField("type", req.Type),
			NewField("limit", req.Limit),
			NewField("client_ip", req.ClientIP),
		)
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.IncCounter("search_requests_total", map[string]string{
			"type": req.Type,
		})
	}

	var allResults []SearchResult
	var groupedResults []GroupedSearchResult

	// Maven 검색
	if req.Type == common.PMTypeAll || req.Type == common.PMTypeMaven {
		mavenResults, err := s.searchMavenContent(ctx, req.Query, req.Limit)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn(ctx, "Maven 검색 실패", NewField("error", err))
			}
		} else {
			groupedResults = append(groupedResults, mavenResults...)
		}
	}

	// APT 검색
	if req.Type == "all" || req.Type == "apt" {
		aptResults, err := s.searchAptContent(ctx, req.Query, req.Limit)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn(ctx, "APT 검색 실패", NewField("error", err))
			}
		} else {
			allResults = append(allResults, aptResults...)
		}
	}

	// NPM 검색
	if req.Type == "all" || req.Type == "npm" {
		npmResults, err := s.searchNpmContent(ctx, req.Query, req.Limit)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn(ctx, "NPM 검색 실패", NewField("error", err))
			}
		} else {
			allResults = append(allResults, npmResults...)
		}
	}

	// 결과 제한 적용
	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}
	if len(groupedResults) > req.Limit {
		groupedResults = groupedResults[:req.Limit]
	}

	totalCount := len(allResults) + len(groupedResults)
	response := &SearchResponse{
		Query:          req.Query,
		ResultCount:    totalCount,
		Results:        allResults,
		GroupedResults: groupedResults,
	}

	// Record response metrics
	if s.metrics != nil {
		s.metrics.IncCounter("search_responses_total", map[string]string{
			"type":   req.Type,
			"status": "success",
		})
		s.metrics.ObserveHistogram("search_results_count", float64(totalCount), map[string]string{
			"type": req.Type,
		})
	}

	return response, nil
}

// searchMavenContent searches Maven repositories
func (s *SearchService) searchMavenContent(
	ctx context.Context, query string, limit int,
) ([]GroupedSearchResult, error) {
	// Maven 설정 로드
	mavenConfig := config.MavenProxySettings{}
	if err := mavenConfig.ReadConfig(); err != nil {
		return nil, fmt.Errorf("maven 설정 로드 실패: %w", err)
	}

	baseURL := common.DefaultLocalhostBaseURL

	// Mock 데이터 정의 (실제 구현에서는 Maven Central API 또는 로컬 인덱스 사용)
	mockArtifacts := []struct {
		GroupID, ArtifactID, Version, Description string
	}{
		{"org.springframework", "spring-core", "6.1.4", "Spring Core"},
		{"org.springframework", "spring-context", "6.1.4", "Spring Context"},
		{"org.springframework", "spring-web", "6.1.4", "Spring Web"},
		{"org.springframework.boot", "spring-boot-starter", "3.2.3", "Spring Boot Starter"},
		{"junit", "junit", "4.13.2", "JUnit Testing Framework"},
		{"org.slf4j", "slf4j-api", "2.0.12", "SLF4J API"},
	}

	// 검색 필터링
	queryLower := strings.ToLower(query)
	var results []GroupedSearchResult

	for _, artifact := range mockArtifacts {
		if len(results) >= limit {
			break
		}

		if strings.Contains(strings.ToLower(artifact.GroupID), queryLower) ||
			strings.Contains(strings.ToLower(artifact.ArtifactID), queryLower) ||
			strings.Contains(strings.ToLower(artifact.Description), queryLower) {
			artifactPath := fmt.Sprintf("%s/%s/%s",
				strings.ReplaceAll(artifact.GroupID, ".", "/"),
				artifact.ArtifactID,
				artifact.Version)

			var sources []SearchResultMirror
			for _, proxy := range mavenConfig.Proxies {
				if !proxy.Enabled {
					continue
				}
				sources = append(sources, SearchResultMirror{
					Name: proxy.Name,
					URL:  fmt.Sprintf("%s/%s", strings.TrimSuffix(proxy.URL, "/"), artifactPath),
				})
			}

			result := GroupedSearchResult{
				GroupID:     artifact.GroupID,
				ArtifactID:  artifact.ArtifactID,
				Version:     artifact.Version,
				Description: artifact.Description,
				ProxyURL:    fmt.Sprintf("%s/%s/%s", baseURL, strings.TrimPrefix(mavenConfig.Path, "/"), artifactPath),
				Sources:     sources,
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// searchAptContent searches APT repositories
func (s *SearchService) searchAptContent(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// APT 설정 로드
	aptConfig := config.AptProxyConfig{}
	if err := aptConfig.ReadConfig(); err != nil {
		return nil, fmt.Errorf("APT 설정 로드 실패: %w", err)
	}

	var results []SearchResult
	baseURL := common.DefaultLocalhostBaseURL

	// Mock 패키지 데이터 (실제 구현에서는 APT 패키지 인덱스 사용)
	mockPackages := []struct {
		Name, Version, Description, Architecture string
	}{
		{"nginx", "1.18.0-6ubuntu14.4", "HTTP and reverse proxy server", "amd64"},
		{"docker.io", "24.0.5-0ubuntu1", "Linux container runtime", "amd64"},
		{"python3", "3.10.12-1~22.04", "Interactive high-level object-oriented language", "amd64"},
		{"git", "1:2.34.1-1ubuntu1.10", "Fast, scalable, distributed revision control system", "amd64"},
		{"curl", "7.81.0-1ubuntu1.15", "Command line tool for transferring data with URL syntax", "amd64"},
	}

	queryLower := strings.ToLower(query)
	for distro, proxies := range aptConfig.Proxies {
		for _, proxy := range proxies {
			for _, pkg := range mockPackages {
				if len(results) >= limit {
					return results, nil
				}

				if strings.Contains(strings.ToLower(pkg.Name), queryLower) ||
					strings.Contains(strings.ToLower(pkg.Description), queryLower) {
					result := SearchResult{
						Type:        "apt",
						Name:        pkg.Name,
						Version:     pkg.Version,
						Description: fmt.Sprintf("%s (%s)", pkg.Description, pkg.Architecture),
						URL:         fmt.Sprintf("%s/pool/main/%s", strings.TrimSuffix(proxy.URL, "/"), pkg.Name),
						ProxyURL:    fmt.Sprintf("%s%s/%s/pool/main/%s", baseURL, aptConfig.Path, distro, pkg.Name),
						Source:      proxy.Name,
					}

					results = append(results, result)
				}
			}
		}
	}

	return results, nil
}

// searchNpmContent searches NPM repositories
func (s *SearchService) searchNpmContent(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// NPM 설정 로드
	npmConfig := config.NpmProxySettings{}
	if err := npmConfig.ReadConfig(); err != nil {
		return nil, fmt.Errorf("NPM 설정 로드 실패: %w", err)
	}

	var results []SearchResult
	baseURL := common.DefaultLocalhostBaseURL

	// Mock NPM 패키지 데이터 (실제 구현에서는 NPM Registry API 사용)
	mockPackages := []struct {
		Name, Version, Description string
	}{
		{"express", "4.18.2", "Fast, unopinionated, minimalist web framework"},
		{"react", "18.2.0", "React is a JavaScript library for building user interfaces"},
		{"lodash", "4.17.21", "Lodash modular utilities"},
		{"axios", "1.6.7", "Promise based HTTP client for the browser and node.js"},
		{"typescript", "5.3.3", "TypeScript is a language for application scale JavaScript"},
	}

	queryLower := strings.ToLower(query)
	for _, pkg := range mockPackages {
		if len(results) >= limit {
			break
		}

		if strings.Contains(strings.ToLower(pkg.Name), queryLower) ||
			strings.Contains(strings.ToLower(pkg.Description), queryLower) {
			// NPM config has a map structure: map[string][]NpmProxyServer
			for registry, servers := range npmConfig.Proxies {
				if len(servers) == 0 {
					continue
				}

				// Use first server from the registry
				server := servers[0]
				result := SearchResult{
					Type:        "npm",
					Name:        pkg.Name,
					Version:     pkg.Version,
					Description: pkg.Description,
					URL:         fmt.Sprintf("%s/%s", strings.TrimSuffix(server.URL, "/"), pkg.Name),
					ProxyURL:    fmt.Sprintf("%s%s/%s", baseURL, strings.TrimPrefix(npmConfig.Path, "/"), pkg.Name),
					Source:      fmt.Sprintf("%s (%s)", server.Name, registry),
				}

				results = append(results, result)
				break // Use first available registry
			}
		}
	}

	return results, nil
}
