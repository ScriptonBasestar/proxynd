package maven

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/domain/maven"
	"proxynd/internal/logging"
	mavenServices "proxynd/internal/services/maven"
)

// BrowserHandler 새로운 Maven 브라우저 핸들러 (200라인 목표)
type BrowserHandler struct {
	config             config.MavenProxySettings
	logger             logging.Logger
	directoryCollector maven.DirectoryCollector
	searchService      maven.SearchService
	cacheManager       maven.CacheManager
	pathAnalyzer       maven.PathAnalyzer
}

// NewBrowserHandler BrowserHandler 생성자
func NewBrowserHandler(config config.MavenProxySettings, logger logging.Logger) *BrowserHandler {
	// 설정을 도메인 인터페이스로 래핑
	proxyConfig := maven.NewDefaultProxyConfig(&config)

	// 서비스 의존성 생성
	directoryCollector := mavenServices.NewDirectoryCollector(proxyConfig, logger)
	cacheManager := mavenServices.NewCacheManager(proxyConfig, logger)
	pathAnalyzer := mavenServices.NewPathAnalyzer(logger)
	searchService := mavenServices.NewSearchService(proxyConfig, logger, directoryCollector)

	handler := &BrowserHandler{
		config:             config,
		logger:             logger,
		directoryCollector: directoryCollector,
		searchService:      searchService,
		cacheManager:       cacheManager,
		pathAnalyzer:       pathAnalyzer,
	}

	// 인기 경로 사전 캐싱 시작 (백그라운드)
	go func() {
		ctx := context.Background()
		if err := cacheManager.PreloadPopularPaths(ctx); err != nil {
			logger.Warn("Failed to preload popular paths", logging.F("error", err))
		}
	}()

	return handler
}

// Handle Fiber HTTP 요청 처리
func (h *BrowserHandler) Handle(c *fiber.Ctx) error {
	ctx := c.Context()
	path := c.Params("*")

	h.logger.Info("Maven browser request",
		logging.F("path", path),
		logging.F("userAgent", c.Get("User-Agent")),
		logging.F("remoteIP", c.IP()),
	)

	// 브라우저 요청인지 확인
	if !h.isBrowserRequest(c) {
		return c.Status(http.StatusNotAcceptable).SendString("Browser request required")
	}

	// 검색 쿼리 처리
	searchQuery := c.Query("search", "")
	if searchQuery != "" {
		return h.handleSearch(ctx, c, searchQuery)
	}

	// 디렉토리 브라우징 처리
	return h.handleDirectoryBrowsing(ctx, c, path)
}

// handleSearch 검색 요청 처리
func (h *BrowserHandler) handleSearch(ctx context.Context, c *fiber.Ctx, query string) error {
	h.logger.Debug("Handling search request", logging.F("query", query))

	// 캐시 확인
	cacheKey := fmt.Sprintf("search:%s", query)
	if cached, found := h.cacheManager.Get(ctx, cacheKey); found {
		h.logger.Debug("Search cache hit", logging.F("query", query))
		return c.JSON(cached.Data)
	}

	// 검색 수행
	searchResult, err := h.searchService.Search(ctx, query)
	if err != nil {
		h.logger.Error("Search failed",
			logging.F("query", query),
			logging.F("error", err),
		)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Search failed",
			"query": query,
		})
	}

	// 결과 캐싱 (5분)
	if err := h.cacheManager.Set(ctx, cacheKey, searchResult, 5*time.Minute); err != nil {
		h.logger.Warn("Failed to cache search result", logging.F("error", err))
	}

	return c.JSON(searchResult)
}

// handleDirectoryBrowsing 디렉토리 브라우징 처리
func (h *BrowserHandler) handleDirectoryBrowsing(ctx context.Context, c *fiber.Ctx, path string) error {
	h.logger.Debug("Handling directory browsing", logging.F("path", path))

	// 경로 분석
	pathInfo, err := h.pathAnalyzer.ParsePath(path)
	if err != nil {
		h.logger.Error("Path analysis failed",
			logging.F("path", path),
			logging.F("error", err),
		)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid Maven path",
			"path":  path,
		})
	}

	// 캐시 확인
	cacheKey := fmt.Sprintf("directory:%s", path)
	if cached, found := h.cacheManager.Get(ctx, cacheKey); found {
		h.logger.Debug("Directory cache hit", logging.F("path", path))
		return h.renderBrowserResponse(c, cached.Data.(*maven.DirectoryData))
	}

	// 디렉토리 데이터 수집
	directoryData, err := h.directoryCollector.CollectDirectory(ctx, path)
	if err != nil {
		h.logger.Error("Directory collection failed",
			logging.F("path", path),
			logging.F("error", err),
		)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to collect directory data",
			"path":  path,
		})
	}

	// 경로 정보 추가
	directoryData.PathInfo = pathInfo

	// Maven 컨텍스트 정보 추가
	h.enrichWithContext(directoryData)

	// 뷰 모드 설정
	viewMode := c.Query("view", "list")
	directoryData.ViewMode = viewMode

	// 트리 뷰인 경우 GAV 트리 구성
	if viewMode == "tree" {
		h.buildGAVTree(directoryData)
	}

	// 캐시 저장 (1시간)
	cacheTTL := time.Hour
	if pathInfo.Type == maven.TypeVersion {
		cacheTTL = 24 * time.Hour // 버전 디렉토리는 더 오래 캐싱
	}

	if err := h.cacheManager.Set(ctx, cacheKey, directoryData, cacheTTL); err != nil {
		h.logger.Warn("Failed to cache directory data", logging.F("error", err))
	}

	return h.renderBrowserResponse(c, directoryData)
}

// renderBrowserResponse 브라우저 응답 렌더링
func (h *BrowserHandler) renderBrowserResponse(c *fiber.Ctx, data *maven.DirectoryData) error {
	// Accept 헤더에 따른 응답 형식 결정
	acceptHeader := c.Get("Accept", "")

	if strings.Contains(acceptHeader, "application/json") {
		return c.JSON(data)
	}

	// HTML 템플릿 렌더링 (기본)
	return c.Render("maven-browser", data)
}

// isBrowserRequest 브라우저 요청인지 확인
func (h *BrowserHandler) isBrowserRequest(c *fiber.Ctx) bool {
	userAgent := c.Get("User-Agent", "")
	acceptHeader := c.Get("Accept", "")

	// 일반적인 브라우저 패턴
	browserPatterns := []string{
		"Mozilla", "Chrome", "Safari", "Firefox", "Edge", "Opera",
	}

	for _, pattern := range browserPatterns {
		if strings.Contains(userAgent, pattern) {
			return true
		}
	}

	// Accept 헤더로 HTML 요청 확인
	if strings.Contains(acceptHeader, "text/html") {
		return true
	}

	// JSON 요청도 브라우저로 간주 (AJAX)
	if strings.Contains(acceptHeader, "application/json") {
		return true
	}

	return false
}

// enrichWithContext Maven 컨텍스트 정보 추가
func (h *BrowserHandler) enrichWithContext(data *maven.DirectoryData) {
	if data.PathInfo == nil {
		return
	}

	context := &maven.Context{
		GroupID:    data.PathInfo.GroupID,
		ArtifactID: data.PathInfo.ArtifactID,
		Version:    data.PathInfo.Version,
		Level:      h.getContextLevel(data.PathInfo.Type),
	}

	// 버전 정보 수집
	if data.PathInfo.Type == maven.TypeArtifact {
		versions := h.extractVersionsFromEntries(data.Entries)
		context.Versions = versions
		if len(versions) > 0 {
			context.Latest = h.findLatestVersion(versions)
		}
	}

	data.Context = context

	// 엔트리별 컨텍스트 정보 추가
	for i := range data.Entries {
		h.enrichEntryContext(&data.Entries[i], data.PathInfo)
	}
}

// buildGAVTree GAV 트리 구조 구성
func (h *BrowserHandler) buildGAVTree(data *maven.DirectoryData) {
	// 간단한 트리 구조 구성 (실제 구현에서는 더 복잡한 로직 필요)
	treeMap := make(map[string]*maven.TreeNode)

	for _, entry := range data.Entries {
		node := &maven.TreeNode{
			Name:    entry.Name,
			Type:    entry.Type,
			Context: entry.Context,
		}
		treeMap[entry.Name] = node
	}

	// 트리 루트 생성
	var treeRoot []*maven.TreeNode
	for _, node := range treeMap {
		treeRoot = append(treeRoot, node)
	}

	data.TreeRoot = treeRoot
}

// 헬퍼 메서드들

func (h *BrowserHandler) getContextLevel(entryType maven.EntryType) string {
	switch entryType {
	case maven.TypeGroup:
		return "group"
	case maven.TypeArtifact:
		return "artifact"
	case maven.TypeVersion:
		return "version"
	default:
		return "directory"
	}
}

func (h *BrowserHandler) extractVersionsFromEntries(entries []maven.Entry) []string {
	var versions []string
	for _, entry := range entries {
		if entry.Type == maven.TypeVersion {
			versionName := strings.TrimSuffix(entry.Name, "/")
			versions = append(versions, versionName)
		}
	}
	return versions
}

func (h *BrowserHandler) findLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	// 간단한 구현: 알파벳 순으로 마지막 버전 반환
	// 실제 환경에서는 의미론적 버전 비교 필요
	latest := versions[0]
	for _, version := range versions[1:] {
		if version > latest { // 간단한 문자열 비교 (임시)
			latest = version
		}
	}
	return latest
}

func (h *BrowserHandler) enrichEntryContext(entry *maven.Entry, parentPathInfo *maven.PathInfo) {
	// 엔트리별 컨텍스트 정보 추가
	entryContext := &maven.Context{
		GroupID: parentPathInfo.GroupID,
	}

	switch entry.Type {
	case maven.TypeArtifact:
		entryContext.ArtifactID = strings.TrimSuffix(entry.Name, "/")
		entryContext.Level = "artifact"
	case maven.TypeVersion:
		entryContext.ArtifactID = parentPathInfo.ArtifactID
		entryContext.Version = strings.TrimSuffix(entry.Name, "/")
		entryContext.Level = "version"
	}

	entry.Context = entryContext
}
