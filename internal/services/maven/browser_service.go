package maven

import (
	"context"
	"fmt"
	"strings"
	"time"

	"proxynd/internal/domain/maven"
	"proxynd/internal/logging"
)

// BrowserService Fiber 독립적인 Maven 브라우저 서비스 (도메인 로직 구현)
type BrowserService struct {
	config             maven.ProxyConfig
	logger             logging.Logger
	directoryCollector maven.DirectoryCollector
	searchService      maven.SearchService
	cacheManager       maven.CacheManager
	pathAnalyzer       maven.PathAnalyzer
}

// NewBrowserService BrowserService 생성자
func NewBrowserService(config maven.ProxyConfig, logger logging.Logger) *BrowserService {
	// 서비스 의존성 생성
	directoryCollector := NewDirectoryCollector(config, logger)
	cacheManager := NewCacheManager(config, logger)
	pathAnalyzer := NewPathAnalyzer(logger)
	searchService := NewSearchService(config, logger, directoryCollector)

	service := &BrowserService{
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

	return service
}

// Handle 도메인 요청 처리 (maven.BrowserHandler 인터페이스 구현)
func (s *BrowserService) Handle(ctx context.Context, request *maven.BrowserRequest) (*maven.BrowserResponse, error) {
	s.logger.Info("Maven browser request via domain service",
		logging.F("path", request.Path),
		logging.F("userAgent", request.Headers["User-Agent"]),
	)

	// 검색 쿼리 처리
	if searchQuery, exists := request.QueryParams["search"]; exists && searchQuery != "" {
		return s.handleSearch(ctx, searchQuery)
	}

	// 디렉토리 브라우징 처리
	return s.handleDirectoryBrowsing(ctx, request.Path)
}

// IsBrowserRequest 요청이 브라우저 요청인지 확인 (maven.BrowserHandler 인터페이스 구현)
func (s *BrowserService) IsBrowserRequest(request *maven.BrowserRequest) bool {
	userAgent := request.Headers["User-Agent"]
	acceptHeader := request.Headers["Accept"]

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

// handleSearch 검색 요청 처리
func (s *BrowserService) handleSearch(ctx context.Context, query string) (*maven.BrowserResponse, error) {
	s.logger.Debug("Handling search request", logging.F("query", query))

	// 캐시 확인
	cacheKey := fmt.Sprintf("search:%s", query)
	if cached, found := s.cacheManager.Get(ctx, cacheKey); found {
		s.logger.Debug("Search cache hit", logging.F("query", query))

		// 검색 결과를 DirectoryData 형식으로 변환
		searchResult := cached.Data.(*maven.SearchResult)
		directoryData := s.convertSearchResultToDirectoryData(searchResult)

		return &maven.BrowserResponse{
			Data:        directoryData,
			StatusCode:  200,
			ContentType: "application/json",
		}, nil
	}

	// 검색 수행
	searchResult, err := s.searchService.Search(ctx, query)
	if err != nil {
		s.logger.Error("Search failed",
			logging.F("query", query),
			logging.F("error", err),
		)
		return &maven.BrowserResponse{
			Data:       &maven.DirectoryData{Path: "error", SearchQuery: query},
			StatusCode: 500,
		}, fmt.Errorf("search failed: %w", err)
	}

	// 결과 캐싱 (5분)
	if err := s.cacheManager.Set(ctx, cacheKey, searchResult, 5*time.Minute); err != nil {
		s.logger.Warn("Failed to cache search result", logging.F("error", err))
	}

	// 검색 결과를 DirectoryData 형식으로 변환
	directoryData := s.convertSearchResultToDirectoryData(searchResult)

	return &maven.BrowserResponse{
		Data:        directoryData,
		StatusCode:  200,
		ContentType: "application/json",
	}, nil
}

// handleDirectoryBrowsing 디렉토리 브라우징 처리
func (s *BrowserService) handleDirectoryBrowsing(ctx context.Context, path string) (*maven.BrowserResponse, error) {
	s.logger.Debug("Handling directory browsing", logging.F("path", path))

	// 경로 분석
	pathInfo, err := s.pathAnalyzer.ParsePath(path)
	if err != nil {
		s.logger.Error("Path analysis failed",
			logging.F("path", path),
			logging.F("error", err),
		)
		return &maven.BrowserResponse{
			Data:       &maven.DirectoryData{Path: path},
			StatusCode: 400,
		}, fmt.Errorf("invalid Maven path: %w", err)
	}

	// 캐시 확인
	cacheKey := fmt.Sprintf("directory:%s", path)
	if cached, found := s.cacheManager.Get(ctx, cacheKey); found {
		s.logger.Debug("Directory cache hit", logging.F("path", path))
		return &maven.BrowserResponse{
			Data:        cached.Data.(*maven.DirectoryData),
			StatusCode:  200,
			ContentType: "text/html",
		}, nil
	}

	// 디렉토리 데이터 수집
	directoryData, err := s.directoryCollector.CollectDirectory(ctx, path)
	if err != nil {
		s.logger.Error("Directory collection failed",
			logging.F("path", path),
			logging.F("error", err),
		)
		return &maven.BrowserResponse{
			Data:       &maven.DirectoryData{Path: path},
			StatusCode: 500,
		}, fmt.Errorf("failed to collect directory data: %w", err)
	}

	// 경로 정보 추가
	directoryData.PathInfo = pathInfo

	// Maven 컨텍스트 정보 추가
	s.enrichWithContext(directoryData)

	// 뷰 모드 설정 (기본값)
	directoryData.ViewMode = "list"

	// 캐시 저장 (1시간)
	cacheTTL := time.Hour
	if pathInfo.Type == maven.TypeVersion {
		cacheTTL = 24 * time.Hour // 버전 디렉토리는 더 오래 캐싱
	}

	if err := s.cacheManager.Set(ctx, cacheKey, directoryData, cacheTTL); err != nil {
		s.logger.Warn("Failed to cache directory data", logging.F("error", err))
	}

	return &maven.BrowserResponse{
		Data:        directoryData,
		StatusCode:  200,
		ContentType: "text/html",
	}, nil
}

// convertSearchResultToDirectoryData 검색 결과를 DirectoryData로 변환
func (s *BrowserService) convertSearchResultToDirectoryData(searchResult *maven.SearchResult) *maven.DirectoryData {
	entries := make([]maven.Entry, 0, len(searchResult.Results))

	for _, artifact := range searchResult.Results {
		entry := maven.Entry{
			Name: fmt.Sprintf("%s:%s", artifact.GroupID, artifact.ArtifactID),
			Type: maven.TypeArtifact,
			Context: &maven.Context{
				GroupID:     artifact.GroupID,
				ArtifactID:  artifact.ArtifactID,
				Versions:    artifact.Versions,
				Latest:      artifact.Latest,
				Description: artifact.Description,
				Level:       "artifact",
			},
		}
		entries = append(entries, entry)
	}

	return &maven.DirectoryData{
		Path:        "/search",
		Entries:     entries,
		ViewMode:    "list",
		SearchQuery: searchResult.Query,
	}
}

// enrichWithContext Maven 컨텍스트 정보 추가
func (s *BrowserService) enrichWithContext(data *maven.DirectoryData) {
	if data.PathInfo == nil {
		return
	}

	context := &maven.Context{
		GroupID:    data.PathInfo.GroupID,
		ArtifactID: data.PathInfo.ArtifactID,
		Version:    data.PathInfo.Version,
		Level:      s.getContextLevel(data.PathInfo.Type),
	}

	// 버전 정보 수집
	if data.PathInfo.Type == maven.TypeArtifact {
		versions := s.extractVersionsFromEntries(data.Entries)
		context.Versions = versions
		if len(versions) > 0 {
			context.Latest = s.findLatestVersion(versions)
		}
	}

	data.Context = context

	// 엔트리별 컨텍스트 정보 추가
	for i := range data.Entries {
		s.enrichEntryContext(&data.Entries[i], data.PathInfo)
	}
}

// 헬퍼 메서드들

func (s *BrowserService) getContextLevel(entryType maven.EntryType) string {
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

func (s *BrowserService) extractVersionsFromEntries(entries []maven.Entry) []string {
	var versions []string
	for _, entry := range entries {
		if entry.Type == maven.TypeVersion {
			versionName := strings.TrimSuffix(entry.Name, "/")
			versions = append(versions, versionName)
		}
	}
	return versions
}

func (s *BrowserService) findLatestVersion(versions []string) string {
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

func (s *BrowserService) enrichEntryContext(entry *maven.Entry, parentPathInfo *maven.PathInfo) {
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
