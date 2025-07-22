package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
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

// SearchResponse represents the API response
type SearchResponse struct {
	Query       string         `json:"query"`
	ResultCount int            `json:"result_count"`
	Results     []SearchResult `json:"results"`
	Error       string         `json:"error,omitempty"`
}

// SearchHandler handles content search across proxies
func SearchHandler(c *fiber.Ctx) error {
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

	// Maven 검색
	if proxyType == "all" || proxyType == "maven" {
		mavenResults, err := searchMavenContent(query, limit)
		if err != nil {
			logger.Warn("Maven 검색 실패", logging.ErrorField(err))
		} else {
			allResults = append(allResults, mavenResults...)
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

	response := SearchResponse{
		Query:       query,
		ResultCount: len(allResults),
		Results:     allResults,
	}

	return c.JSON(response)
}

// searchMavenContent searches Maven repositories
func searchMavenContent(query string, limit int) ([]SearchResult, error) {
	// Maven 설정 로드
	mavenConfig := configs.MavenProxyConfig{}
	if err := mavenConfig.ReadConfig(); err != nil {
		return nil, fmt.Errorf("Maven 설정 로드 실패: %w", err)
	}

	var results []SearchResult

	// 각 Maven 프록시에서 검색
	for _, proxy := range mavenConfig.Proxies {
		if !proxy.Enabled {
			continue
		}

		// Maven Central API를 통한 검색 (예시)
		searchURL := fmt.Sprintf("%s/solrsearch/select?q=%s&rows=%d&wt=json",
			strings.TrimSuffix(proxy.URL, "/"), url.QueryEscape(query), limit)

		proxyResults, err := searchMavenProxy(searchURL, proxy.Name, mavenConfig.Path)
		if err != nil {
			logging.GetLogger().Warn("Maven 프록시 검색 실패",
				logging.String("proxy", proxy.Name),
				logging.String("url", searchURL),
				logging.ErrorField(err))
			continue
		}

		results = append(results, proxyResults...)
	}

	return results, nil
}

// searchMavenProxy performs actual search against Maven proxy
func searchMavenProxy(searchURL, proxyName, proxyPath string) ([]SearchResult, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Maven Solr 검색 결과 파싱
	var solrResponse struct {
		Response struct {
			Docs []struct {
				ID            string `json:"id"`
				GroupID       string `json:"g"`
				ArtifactID    string `json:"a"`
				Version       string `json:"v"`
				LatestVersion string `json:"latestVersion"`
				Packaging     string `json:"p"`
				Timestamp     int64  `json:"timestamp"`
			} `json:"docs"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &solrResponse); err != nil {
		return nil, err
	}

	var results []SearchResult
	baseURL := "http://localhost:8080" // 현재 서버 주소

	for _, doc := range solrResponse.Response.Docs {
		version := doc.LatestVersion
		if version == "" {
			version = doc.Version
		}

		artifactPath := fmt.Sprintf("%s/%s/%s",
			strings.ReplaceAll(doc.GroupID, ".", "/"),
			doc.ArtifactID,
			version)

		result := SearchResult{
			Type:        "maven",
			Name:        fmt.Sprintf("%s:%s", doc.GroupID, doc.ArtifactID),
			Version:     version,
			Description: fmt.Sprintf("Maven 아티팩트 (%s)", doc.Packaging),
			URL:         fmt.Sprintf("%s/%s", searchURL, artifactPath),
			ProxyURL:    fmt.Sprintf("%s%s/%s", baseURL, proxyPath, artifactPath),
			Source:      proxyName,
		}

		results = append(results, result)
	}

	return results, nil
}

// searchAptContent searches APT repositories
func searchAptContent(query string, limit int) ([]SearchResult, error) {
	// APT 설정 로드
	aptConfig := configs.AptProxyConfig{}
	if err := aptConfig.ReadConfig(); err != nil {
		return nil, fmt.Errorf("APT 설정 로드 실패: %w", err)
	}

	var results []SearchResult
	baseURL := "http://localhost:8080" // 현재 서버 주소

	// 각 APT 프록시에서 검색
	for distro, proxies := range aptConfig.Proxies {
		for _, proxy := range proxies {
			// APT 패키지 검색을 위한 간단한 구현
			// 실제로는 패키지 인덱스를 파싱해야 하지만, 여기서는 예시로 구현
			result := SearchResult{
				Type:        "apt",
				Name:        fmt.Sprintf("%s-search-result", query),
				Version:     "latest",
				Description: fmt.Sprintf("%s 배포판에서 '%s' 패키지 검색", distro, query),
				URL:         fmt.Sprintf("%s/dists/%s/main/binary-amd64/Packages", proxy.URL, distro),
				ProxyURL:    fmt.Sprintf("%s%s/%s/dists/%s/main/binary-amd64/Packages", baseURL, aptConfig.Path, distro, distro),
				Source:      proxy.Name,
			}

			results = append(results, result)

			if len(results) >= limit {
				break
			}
		}
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}
