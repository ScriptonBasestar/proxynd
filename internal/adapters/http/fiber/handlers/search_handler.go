package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/logging"
)

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

// SearchResponse represents the API response
type SearchResponse struct {
	Query          string                `json:"query"`
	ResultCount    int                   `json:"result_count"`
	Results        []SearchResult        `json:"results,omitempty"`         // for APT and other types
	GroupedResults []GroupedSearchResult `json:"grouped_results,omitempty"` // for Maven
	Error          string                `json:"error,omitempty"`
}

// SearchHandler handles content search across proxies
func SearchHandler(c *fiber.Ctx) error {
	// TODO: HEXAGONAL_MIGRATION - Replace direct search logic with usecase calls
	// Current implementation has search logic mixed in HTTP handler
	// Should be migrated to use usecase.SearchService.Search()
	
	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(SearchResponse{
			Error: "검색어가 필요합니다",
		})
	}

	proxyType := c.Query("type", "all") // all, maven, apt, npm, etc.
	limit := c.QueryInt("limit", 20)    // default 20 results

	logger := logging.GetLogger()
	logger.Info("프록시 컨텐츠 검색 요청",
		logging.String("query", query),
		logging.String("type", proxyType),
		logging.Int("limit", limit))

	var allResults []SearchResult
	var groupedResults []GroupedSearchResult

	// Maven 검색
	if proxyType == "all" || proxyType == "maven" {
		mavenGroupedResults, err := searchMavenContentGrouped(query, limit)
		if err != nil {
			logger.Warn("Maven 검색 실패", logging.ErrorField(err))
		} else {
			groupedResults = append(groupedResults, mavenGroupedResults...)
		}
	}

	// APT 검색
	if proxyType == "all" || proxyType == "apt" {
		aptResults, err := searchAptContent(query, limit)
		if err != nil {
			logger.Warn("APT 검색 실패", logging.ErrorField(err))
		} else {
			allResults = append(allResults, aptResults...)
		}
	}

	// 결과 제한 적용
	if len(allResults) > limit {
		allResults = allResults[:limit]
	}
	if len(groupedResults) > limit {
		groupedResults = groupedResults[:limit]
	}

	totalCount := len(allResults) + len(groupedResults)
	response := SearchResponse{
		Query:          query,
		ResultCount:    totalCount,
		Results:        allResults,
		GroupedResults: groupedResults,
	}

	return c.JSON(response)
}

// searchAptContent searches APT repositories using local package index
func searchAptContent(query string, limit int) ([]SearchResult, error) {
	// APT 설정 로드
	aptConfig := config.AptProxyConfig{}
	if err := aptConfig.ReadConfig(); err != nil {
		return nil, fmt.Errorf("APT 설정 로드 실패: %w", err)
	}

	var results []SearchResult
	baseURL := "http://localhost:8080"

	// 각 APT 프록시에서 검색
	for distro, proxies := range aptConfig.Proxies {
		for _, proxy := range proxies {
			// 인덱스 기반 검색 수행
			proxyResults, err := searchAptIndex(distro, proxy, query, limit, baseURL, aptConfig.Path)
			if err != nil {
				logging.GetLogger().Warn("APT 인덱스 검색 실패",
					logging.String("proxy", proxy.Name),
					logging.String("distro", distro),
					logging.ErrorField(err))
				continue
			}

			results = append(results, proxyResults...)

			if len(results) >= limit {
				results = results[:limit]
				break
			}
		}
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// searchAptIndex searches APT repository using mock package data
func searchAptIndex(
	distro string, proxy config.AptProxy, query string, limit int, baseURL, proxyPath string,
) ([]SearchResult, error) {
	var results []SearchResult

	// APT 패키지 Mock 데이터
	mockPackages := []struct {
		Name, Version, Description, Architecture string
	}{
		{"nginx", "1.18.0-6ubuntu14.4", "HTTP and reverse proxy server", "amd64"},
		{"docker.io", "24.0.5-0ubuntu1", "Linux container runtime", "amd64"},
		{"python3", "3.10.12-1~22.04", "Interactive high-level object-oriented language", "amd64"},
		{"python3-pip", "22.0.2+dfsg-1ubuntu0.4", "Python package installer", "all"},
		{"git", "1:2.34.1-1ubuntu1.10", "Fast, scalable, distributed revision control system", "amd64"},
		{"curl", "7.81.0-1ubuntu1.15", "Command line tool for transferring data with URL syntax", "amd64"},
		{"wget", "1.21.2-2ubuntu1", "Retrieves files from the web", "amd64"},
		{"vim", "2:8.2.3458-2ubuntu2.4", "Vi IMproved - enhanced vi editor", "amd64"},
		{"openssh-server", "1:8.9p1-3ubuntu0.6", "Secure shell (SSH) server", "amd64"},
		{"apache2", "2.4.52-1ubuntu4.7", "Apache HTTP Server", "amd64"},
	}

	queryLower := strings.ToLower(query)
	for _, pkg := range mockPackages {
		if strings.Contains(strings.ToLower(pkg.Name), queryLower) ||
			strings.Contains(strings.ToLower(pkg.Description), queryLower) {
			result := SearchResult{
				Type:        "apt",
				Name:        pkg.Name,
				Version:     pkg.Version,
				Description: fmt.Sprintf("%s (%s)", pkg.Description, pkg.Architecture),
				URL:         fmt.Sprintf("%s/pool/main/%s", strings.TrimSuffix(proxy.URL, "/"), pkg.Name),
				ProxyURL:    fmt.Sprintf("%s%s/%s/pool/main/%s", baseURL, proxyPath, distro, pkg.Name),
				Source:      proxy.Name,
			}

			results = append(results, result)

			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// searchMavenContentGrouped searches Maven repositories and returns grouped results
func searchMavenContentGrouped(query string, limit int) ([]GroupedSearchResult, error) {
	// Maven 설정 로드
	mavenConfig := config.MavenProxySettings{}
	if err := mavenConfig.ReadConfig(); err != nil {
		return nil, fmt.Errorf("maven 설정 로드 실패: %w", err)
	}

	baseURL := "http://localhost:8080"

	// Mock 데이터 정의
	mockArtifacts := []struct {
		GroupID, ArtifactID, Version, Description string
	}{
		{"org.springframework", "spring-core", "6.1.4", "Spring Core"},
		{"org.springframework", "spring-context", "6.1.4", "Spring Context"},
		{"org.springframework", "spring-web", "6.1.4", "Spring Web"},
		{"org.springframework", "spring-webmvc", "6.1.4", "Spring Web MVC"},
		{"org.springframework.boot", "spring-boot-starter", "3.2.3", "Spring Boot Starter"},
		{"org.springframework.boot", "spring-boot-starter-web", "3.2.3", "Spring Boot Web Starter"},
		{"org.springframework.boot", "spring-boot-starter-data-jpa", "3.2.3", "Spring Boot JPA Starter"},
		{"org.springframework.security", "spring-security-core", "6.2.2", "Spring Security Core"},
		{"junit", "junit", "4.13.2", "JUnit Testing Framework"},
		{"org.junit.jupiter", "junit-jupiter", "5.10.2", "JUnit 5 Jupiter"},
		{"org.slf4j", "slf4j-api", "2.0.12", "SLF4J API"},
		{"ch.qos.logback", "logback-classic", "1.4.14", "Logback Classic"},
		{"com.fasterxml.jackson.core", "jackson-databind", "2.16.1", "Jackson Databind"},
		{"org.apache.commons", "commons-lang3", "3.14.0", "Apache Commons Lang3"},
	}

	// 검색 필터링
	queryLower := strings.ToLower(query)
	var filteredArtifacts []struct {
		GroupID, ArtifactID, Version, Description string
	}

	for _, artifact := range mockArtifacts {
		if strings.Contains(strings.ToLower(artifact.GroupID), queryLower) ||
			strings.Contains(strings.ToLower(artifact.ArtifactID), queryLower) ||
			strings.Contains(strings.ToLower(artifact.Description), queryLower) {
			filteredArtifacts = append(filteredArtifacts, artifact)
		}
	}

	// 그룹화된 결과 생성
	var groupedResults []GroupedSearchResult
	for _, artifact := range filteredArtifacts {
		if len(groupedResults) >= limit {
			break
		}

		artifactPath := fmt.Sprintf("%s/%s/%s",
			strings.ReplaceAll(artifact.GroupID, ".", "/"),
			artifact.ArtifactID,
			artifact.Version)

		var sources []SearchResultMirror
		// 각 활성화된 프록시에서 이 아티팩트를 제공한다고 가정
		for _, proxy := range mavenConfig.Proxies {
			if !proxy.Enabled {
				continue
			}

			mirror := SearchResultMirror{
				Name: proxy.Name,
				URL:  fmt.Sprintf("%s/%s", strings.TrimSuffix(proxy.URL, "/"), artifactPath),
			}
			sources = append(sources, mirror)
		}

		groupedResult := GroupedSearchResult{
			GroupID:     artifact.GroupID,
			ArtifactID:  artifact.ArtifactID,
			Version:     artifact.Version,
			Description: artifact.Description,
			ProxyURL:    fmt.Sprintf("%s/%s/%s", baseURL, strings.TrimPrefix(mavenConfig.Path, "/"), artifactPath),
			Sources:     sources,
		}

		groupedResults = append(groupedResults, groupedResult)
	}

	return groupedResults, nil
}
