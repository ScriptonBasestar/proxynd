package proxy

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/domain/maven"
	"proxynd/internal/logging"
)

// 도메인 모델로 이동됨 - internal/domain/maven/models.go 참조

// DirectoryEntry 디렉토리 엔트리 (확장됨)
type DirectoryEntry struct {
	Name         string               `json:"name"`
	Type         maven.MavenEntryType `json:"type"` // Maven 타입 사용
	Size         int64                `json:"size,omitempty"`
	LastModified time.Time            `json:"lastModified,omitempty"`
	Sources      []string             `json:"sources"` // 어떤 미러에서 발견되었는지

	// Maven 특화 정보
	MavenContext *maven.MavenContext `json:"mavenContext,omitempty"`
}

// MavenBrowserData 템플릿 데이터 (확장됨)
type MavenBrowserData struct {
	Path         string               `json:"path"`
	Entries      []DirectoryEntry     `json:"entries"`
	Mirrors      []maven.MirrorStatus `json:"mirrors"`
	TotalMirrors int                  `json:"totalMirrors"`

	// Maven 컨텍스트 정보
	PathInfo *maven.PathInfo     `json:"pathInfo,omitempty"`
	Context  *maven.MavenContext `json:"context,omitempty"`

	// GAV 트리 구조
	TreeRoot    []*maven.GAVTreeNode `json:"treeRoot,omitempty"`
	ViewMode    string               `json:"viewMode"` // "tree" 또는 "list"
	SearchQuery string               `json:"searchQuery,omitempty"`
}

// MavenArtifactData artifact 페이지용 데이터
type MavenArtifactData struct {
	GroupID         string            `json:"groupId"`
	ArtifactID      string            `json:"artifactId"`
	Versions        []string          `json:"versions"`
	SelectedVersion string            `json:"selectedVersion"`
	VersionDates    map[string]string `json:"versionDates,omitempty"`
	Files           []FileInfo        `json:"files,omitempty"`
	BreadcrumbParts []BreadcrumbPart  `json:"breadcrumbParts"`
}

// FileInfo 파일 정보
type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size string `json:"size"`
}

// BreadcrumbPart 브레드크럼 부분
type BreadcrumbPart struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Breadcrumb 브레드크럼 정보
type Breadcrumb struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// cacheEntry 캐시 엔트리
type cacheEntry struct {
	data      *MavenBrowserData
	timestamp time.Time
}

// searchIndex 검색 인덱스
type searchIndex struct {
	entries   []maven.SearchIndexEntry
	mutex     sync.RWMutex
	lastBuild time.Time
}

// MavenBrowserHandler Maven 리포지토리 브라우저 핸들러
type MavenBrowserHandler struct {
	client       *http.Client
	logger       logging.Logger
	config       *config.MavenProxySettings
	cache        sync.Map // path -> cacheEntry
	cacheTTL     time.Duration
	searchIndex  *searchIndex
	indexStorage maven.IndexStorage
}

// NewMavenBrowserHandler 새로운 Maven 브라우저 핸들러 생성
func NewMavenBrowserHandler() *MavenBrowserHandler {
	h := &MavenBrowserHandler{
		client: &http.Client{
			Timeout: 10 * time.Second, // 타임아웃 단축
		},
		logger:   logging.GetLogger(),
		config:   &config.MavenProxySettings{},
		cacheTTL: 5 * time.Minute, // 5분 캐시
		searchIndex: &searchIndex{
			entries: make([]maven.SearchIndexEntry, 0, 10000), // 초기 용량 10000
		},
	}

	// 설정에 따라 인덱스 로드 또는 빌드
	// 설정은 나중에 로드되므로 여기서는 기본 경로로 초기화
	go h.initializeIndex()

	// 백그라운드에서 인기있는 경로 프리로드
	go h.preloadPopularPaths()

	return h
}

// Handle Maven 브라우저 요청 처리
func (h *MavenBrowserHandler) Handle(c *fiber.Ctx) error {
	// 설정 로드
	if err := h.config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read Maven config", logging.F("error", err))
		return c.Status(500).SendString("Maven 설정을 읽을 수 없습니다")
	}

	if len(h.config.Proxies) == 0 {
		return c.Status(500).SendString("Maven 프록시가 설정되지 않았습니다")
	}

	// 요청 경로 파싱
	artifactPath := c.Params("*")
	if artifactPath == "" {
		artifactPath = "/"
	}

	h.logger.Info("Maven browser request",
		logging.F("path", artifactPath),
		logging.F("user_agent", c.Get("User-Agent")),
	)

	// 경로 분석하여 artifact 페이지인지 확인
	pathInfo := parseMavenPath(artifactPath)

	// artifact 상세 페이지 처리
	if pathInfo.Type == maven.TypeArtifact && pathInfo.ArtifactID != "" {
		return h.handleArtifactPage(c, pathInfo)
	}

	// 검색 쿼리 확인
	searchQuery := c.Query("search", "")

	// 캐시 확인
	cacheKey := artifactPath
	if cached, found := h.cache.Load(cacheKey); found {
		if entry, ok := cached.(*cacheEntry); ok {
			// 캐시가 유효한지 확인
			if time.Since(entry.timestamp) < h.cacheTTL {
				browserData := entry.data
				if searchQuery != "" {
					browserData.SearchQuery = searchQuery
					if browserData.TreeRoot != nil {
						browserData.TreeRoot = h.searchInTree(browserData.TreeRoot, searchQuery)
					}
				}

				// JSON API 요청인 경우
				if strings.Contains(c.Get("Accept"), MimeApplicationJSON) || c.Query("format") == FormatJSON {
					return c.JSON(browserData)
				}
				return c.Render("maven-browser", browserData)
			}
		}
	}

	// 각 미러에서 디렉토리 정보 수집
	browserData, err := h.collectDirectoryData(artifactPath)
	if err != nil {
		h.logger.Error("Failed to collect directory data",
			logging.F("path", artifactPath),
			logging.F("error", err),
		)
		return c.Status(500).SendString("디렉토리 정보를 수집할 수 없습니다")
	}

	// 캐시에 저장
	h.cache.Store(cacheKey, &cacheEntry{
		data:      browserData,
		timestamp: time.Now(),
	})

	// 검색 쿼리가 있으면 전체 검색 수행
	if searchQuery != "" {
		// 모든 경로에서 검색 시 전체 검색 수행
		return h.performGlobalSearch(c, searchQuery)
	}

	// JSON API 요청인 경우 (AJAX 요청 또는 format=json 파라미터)
	if strings.Contains(c.Get("Accept"), MimeApplicationJSON) || c.Query("format") == FormatJSON {
		return c.JSON(browserData)
	}

	// HTML 브라우저 요청인 경우
	return c.Render("maven-browser", browserData)
}

// collectDirectoryData 각 미러에서 디렉토리 데이터 수집
func (h *MavenBrowserHandler) collectDirectoryData(artifactPath string) (*MavenBrowserData, error) {
	// Maven 경로 분석
	pathInfo := parseMavenPath(artifactPath)

	data := &MavenBrowserData{
		Path:     artifactPath,
		Entries:  []DirectoryEntry{},
		Mirrors:  []maven.MirrorStatus{},
		PathInfo: pathInfo,
	}

	h.logger.Info("Collecting directory data",
		logging.F("path", artifactPath),
		logging.F("pathInfo", pathInfo),
	)

	// 각 미러에서 데이터 수집 (병렬 처리)
	entryMap := make(map[string]*DirectoryEntry)
	var mapMutex sync.Mutex
	var wg sync.WaitGroup

	// 결과 채널
	type mirrorResult struct {
		status  maven.MirrorStatus
		entries []DirectoryEntry
	}
	resultChan := make(chan mirrorResult, len(h.config.Proxies))

	// 각 미러에 대해 고루틴 실행
	for _, proxy := range h.config.Proxies {
		wg.Add(1)
		go func(p config.MavenProxyServer) {
			defer wg.Done()

			mirrorStatus := maven.MirrorStatus{
				Name: p.Name,
				URL:  p.URL,
			}

			entries, err := h.fetchDirectoryFromMirror(p, artifactPath)
			if err != nil {
				mirrorStatus.Available = false
				mirrorStatus.Error = err.Error()
				h.logger.Warn("Mirror unavailable",
					logging.F("mirror", p.Name),
					logging.F("error", err),
				)
			} else {
				mirrorStatus.Available = true
			}

			resultChan <- mirrorResult{
				status:  mirrorStatus,
				entries: entries,
			}
		}(proxy)
	}

	// 모든 고루틴 완료 대기
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 결과 수집 및 병합
	for result := range resultChan {
		data.Mirrors = append(data.Mirrors, result.status)

		if result.status.Available {
			mapMutex.Lock()
			for _, entry := range result.entries {
				if existing, exists := entryMap[entry.Name]; exists {
					existing.Sources = append(existing.Sources, result.status.Name)
				} else {
					entry.Sources = []string{result.status.Name}
					entryMap[entry.Name] = &entry
				}
			}
			mapMutex.Unlock()
		}
	}

	// 엔트리 맵을 슬라이스로 변환 및 Maven 컨텍스트 추가
	for _, entry := range entryMap {
		// Maven 컨텍스트 정보 추가
		enrichDirectoryEntry(entry, artifactPath)
		data.Entries = append(data.Entries, *entry)
	}

	// 정렬: Maven 타입별로 먼저, 그다음 이름순
	sort.Slice(data.Entries, func(i, j int) bool {
		// Maven 타입별 우선순위: Group > Artifact > Version > Directory > File
		typeOrder := map[maven.MavenEntryType]int{
			maven.TypeGroup:     1,
			maven.TypeArtifact:  2,
			maven.TypeVersion:   3,
			maven.TypeDirectory: 4,
			maven.TypeFile:      5,
			maven.TypeMetadata:  6,
		}

		orderI := typeOrder[data.Entries[i].Type]
		orderJ := typeOrder[data.Entries[j].Type]

		if orderI != orderJ {
			return orderI < orderJ
		}
		return data.Entries[i].Name < data.Entries[j].Name
	})

	// 전체 컨텍스트 생성
	data.Context = createMavenContext(pathInfo, data.Entries)
	data.TotalMirrors = len(h.config.Proxies)

	// GAV 트리 구조 생성
	data.TreeRoot = h.buildGAVTree(data.Entries, artifactPath)
	data.ViewMode = "tree"

	// 디버깅: springframework 엔트리 확인
	for _, entry := range data.Entries {
		if entry.Name == "springframework" {
			h.logger.Info("springframework entry found",
				logging.F("name", entry.Name),
				logging.F("type", entry.Type),
				logging.F("sources", entry.Sources),
			)
			break
		}
	}

	// 디버깅을 위한 로깅
	h.logger.Info("GAV tree built",
		logging.F("artifactPath", artifactPath),
		logging.F("treeRootCount", len(data.TreeRoot)),
		logging.F("entriesCount", len(data.Entries)),
	)
	if len(data.TreeRoot) > 0 {
		h.logger.Info("First tree node",
			logging.F("name", data.TreeRoot[0].Name),
			logging.F("type", data.TreeRoot[0].Type),
			logging.F("childCount", data.TreeRoot[0].ChildCount),
			logging.F("groupID", data.TreeRoot[0].GroupID),
		)
	}

	return data, nil
}

// fetchDirectoryFromMirror 특정 미러에서 디렉토리 정보 수집
func (h *MavenBrowserHandler) fetchDirectoryFromMirror(
	proxy config.MavenProxyServer,
	artifactPath string,
) ([]DirectoryEntry, error) {
	// 미러 URL 구성
	baseURL := strings.TrimRight(proxy.URL, "/")
	cleanPath := strings.Trim(artifactPath, "/")

	var fullURL string
	if cleanPath == "" {
		fullURL = baseURL + "/"
	} else {
		fullURL = baseURL + "/" + cleanPath + "/"
	}

	// HTTP 요청
	resp, err := h.client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 실패: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 상태 코드: %d", resp.StatusCode)
	}

	// HTML 파싱을 통한 디렉토리 목록 추출
	entries, err := h.parseDirectoryListing(resp, baseURL, cleanPath)
	if err != nil {
		// HTML 파싱 실패 시 메타데이터 기반으로 시도
		return h.parseMetadata(proxy, cleanPath)
	}

	return entries, nil
}

// parseDirectoryListing HTML 디렉토리 목록 파싱
func (h *MavenBrowserHandler) parseDirectoryListing(
	resp *http.Response,
	baseURL, cleanPath string,
) ([]DirectoryEntry, error) {
	// 간단한 HTML 파싱 - Apache/Nginx 스타일 디렉토리 목록
	body := make([]byte, 64*1024) // 64KB 제한
	n, err := resp.Body.Read(body)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	html := string(body[:n])
	entries := []DirectoryEntry{}

	// 간단한 링크 추출 (href="...")
	lines := strings.Split(html, "\n")
	for _, line := range lines {
		if strings.Contains(line, "href=") {
			entry := h.extractEntryFromLine(line)
			if entry != nil && entry.Name != ".." && entry.Name != "." {
				entries = append(entries, *entry)
			}
		}
	}

	return entries, nil
}

// extractEntryFromLine HTML 라인에서 엔트리 추출
func (h *MavenBrowserHandler) extractEntryFromLine(line string) *DirectoryEntry {
	// href="..." 패턴 찾기
	hrefStart := strings.Index(line, "href=\"")
	if hrefStart == -1 {
		return nil
	}
	hrefStart += 6

	hrefEnd := strings.Index(line[hrefStart:], "\"")
	if hrefEnd == -1 {
		return nil
	}

	href := line[hrefStart : hrefStart+hrefEnd]

	// URL 디코딩
	decodedHref, err := url.QueryUnescape(href)
	if err != nil {
		decodedHref = href
	}

	// 상대 경로만 처리
	if strings.HasPrefix(decodedHref, "http") || strings.HasPrefix(decodedHref, "/") {
		return nil
	}

	// 템플릿 변수나 특수 파일 제외
	if strings.Contains(decodedHref, "{{") || strings.Contains(decodedHref, "}}") {
		return nil
	}

	// 웹 리소스 파일 제외
	webExtensions := []string{".css", ".js", ".ico", ".html", ".htm"}
	lowerName := strings.ToLower(decodedHref)
	for _, ext := range webExtensions {
		if strings.HasSuffix(lowerName, ext) {
			return nil
		}
	}

	entry := &DirectoryEntry{
		Name: decodedHref,
		Type: "file",
	}

	// 디렉토리 판별
	if strings.HasSuffix(decodedHref, "/") {
		// 이름이 /로 끝나는 경우
		entry.Name = strings.TrimSuffix(decodedHref, "/")
		entry.Type = TypeDirectory
	} else if !hasFileExtension(decodedHref) {
		// 파일 확장자가 없으면 디렉토리로 간주
		// Maven 리포지토리에서 디렉토리는 보통 확장자가 없음
		entry.Type = TypeDirectory
	}

	return entry
}

// hasFileExtension 파일 확장자가 있는지 확인
func hasFileExtension(name string) bool {
	// Maven 리포지토리의 일반적인 파일 확장자
	extensions := []string{
		".jar", ".war", ".ear", ".zip", ".tar", ".gz", ".bz2",
		".pom", ".xml", ".sha1", ".md5", ".asc", ".sig",
		".txt", ".properties", ".yml", ".yaml", ".json",
	}

	lowerName := strings.ToLower(name)
	for _, ext := range extensions {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}

	return false
}

// parseMetadata Maven 메타데이터 기반 파싱
func (h *MavenBrowserHandler) parseMetadata(proxy config.MavenProxyServer, cleanPath string) ([]DirectoryEntry, error) {
	// maven-metadata.xml 파일 시도
	metadataURL := strings.TrimRight(proxy.URL, "/") + "/" + cleanPath + "/maven-metadata.xml"

	resp, err := h.client.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("메타데이터 요청 실패: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("메타데이터 없음: %d", resp.StatusCode)
	}

	var metadata maven.MavenMetadata
	if err := xml.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("메타데이터 파싱 실패: %w", err)
	}

	entries := []DirectoryEntry{}

	// 버전 정보에서 디렉토리 생성
	for _, version := range metadata.Versioning.Versions {
		entries = append(entries, DirectoryEntry{
			Name: version,
			Type: "directory",
		})
	}

	return entries, nil
}

// IsBrowserRequest 브라우저 요청인지 판별
func IsBrowserRequest(c *fiber.Ctx) bool {
	userAgent := c.Get("User-Agent")
	accept := c.Get("Accept")

	// User-Agent에 브라우저 문자열 포함 확인
	browserUserAgents := []string{"Mozilla", "Chrome", "Safari", "Edge", "Firefox"}
	for _, browser := range browserUserAgents {
		if strings.Contains(userAgent, browser) {
			return true
		}
	}

	// Accept 헤더에 HTML 포함 확인
	if strings.Contains(accept, "text/html") {
		return true
	}

	return false
}

// IsDirectoryPath 디렉토리 경로인지 판별 (파일 확장자가 없는 경우)
func IsDirectoryPath(path string) bool {
	if path == "" || path == "/" {
		return true
	}

	// 경로가 /로 끝나는 경우
	if strings.HasSuffix(path, "/") {
		return true
	}

	// 파일 확장자가 없는 경우
	basename := strings.TrimSuffix(path, "/")
	lastPart := path
	if lastSlash := strings.LastIndex(basename, "/"); lastSlash != -1 {
		lastPart = basename[lastSlash+1:]
	}

	// 일반적인 Maven 파일 확장자 체크
	commonExtensions := []string{".jar", ".war", ".ear", ".pom", ".xml", ".sha1", ".md5", ".asc"}
	for _, ext := range commonExtensions {
		if strings.HasSuffix(lastPart, ext) {
			return false
		}
	}

	return true
}

// parseMavenPath Maven 경로를 분석하여 컨텍스트 정보를 추출
func parseMavenPath(path string) *maven.PathInfo {
	// 경로 정규화 (앞뒤 슬래시 제거)
	path = strings.Trim(path, "/")
	if path == "" {
		return &maven.PathInfo{
			Type:  maven.TypeDirectory,
			Level: 0,
		}
	}

	parts := strings.Split(path, "/")
	level := len(parts)

	info := &maven.PathInfo{
		Level: level,
	}

	// Maven 표준 구조 분석
	// 예: org/apache/httpcomponents/client5/httpclient5/5.3.1/
	//     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ 그룹
	//                                   ^^^^^^^^^^^ 아티팩트
	//                                               ^^^^^ 버전

	// 마지막부터 거꾸로 탐색하여 버전 찾기
	versionIndex := -1
	for i := level - 1; i >= 0; i-- {
		if isVersionLike(parts[i]) {
			versionIndex = i
			break
		}
	}

	if versionIndex > 0 {
		// 버전이 발견된 경우
		info.Type = maven.TypeVersion
		info.ArtifactID = parts[versionIndex-1]
		info.Version = parts[versionIndex]
		info.GroupID = strings.Join(parts[:versionIndex-1], ".")
	} else {
		// 버전이 없는 경우 - 아티팩트 패턴 확인
		// 아티팩트는 보통 하이픈을 포함하거나 마지막 부분에 위치
		artifactIndex := -1
		for i := level - 1; i >= 0; i-- {
			if isLikelyArtifact(parts[i]) {
				artifactIndex = i
				break
			}
		}

		if artifactIndex > 0 {
			// 아티팩트가 발견된 경우
			info.Type = maven.TypeArtifact
			info.ArtifactID = parts[artifactIndex]
			info.GroupID = strings.Join(parts[:artifactIndex], ".")
		} else {
			// 그룹 경로로 판단
			info.Type = maven.TypeGroup
			info.GroupID = strings.Join(parts, ".")
		}
	}

	return info
}

// isLikelyArtifact 이름이 아티팩트처럼 보이는지 판단
func isLikelyArtifact(name string) bool {
	// Maven artifact는 보통 하이픈을 포함함
	// 예: spring-core, spring-context, httpclient5, commons-lang3

	// 하이픈이 있으면 대부분 artifact
	if strings.Contains(name, "-") {
		return true
	}

	// 숫자로 끝나는 경우 (httpclient5, junit4 등)
	if len(name) > 0 {
		lastChar := name[len(name)-1]
		if lastChar >= '0' && lastChar <= '9' {
			return true
		}
	}

	// 알려진 그룹 이름들은 제외
	knownGroups := []string{
		"springframework", "apache", "google", "amazon", "alibaba",
		"netflix", "eclipse", "jetbrains", "intellij", "mongodb",
		"mysql", "postgresql", "redis", "elastic", "opensearch",
	}

	for _, group := range knownGroups {
		if name == group {
			return false
		}
	}

	// 기타 경우는 artifact가 아님
	return false
}

// isVersionLike 문자열이 버전처럼 보이는지 확인
func isVersionLike(s string) bool {
	// 숫자로 시작하는 패턴 (1.0, 2.3.1, 1.0-SNAPSHOT 등)
	if len(s) == 0 {
		return false
	}

	// 첫 문자가 숫자인지 확인
	return s[0] >= '0' && s[0] <= '9'
}

// createMavenContext 경로 정보와 엔트리 목록을 기반으로 Maven 컨텍스트 생성
func createMavenContext(pathInfo *maven.PathInfo, entries []DirectoryEntry) *maven.MavenContext {
	if pathInfo == nil {
		return nil
	}

	context := &maven.MavenContext{
		GroupID:    pathInfo.GroupID,
		ArtifactID: pathInfo.ArtifactID,
		Version:    pathInfo.Version,
	}

	// 레벨에 따른 컨텍스트 설정
	switch pathInfo.Type {
	case maven.TypeGroup:
		context.Level = LevelGroup

	case maven.TypeArtifact:
		context.Level = LevelArtifact
		// 하위 디렉토리에서 버전 정보 수집
		versions := make([]string, 0)
		for _, entry := range entries {
			if entry.Type == maven.TypeDirectory && isVersionLike(entry.Name) {
				versions = append(versions, entry.Name)
			}
		}
		context.Versions = versions
		if len(versions) > 0 {
			context.Latest = findLatestVersion(versions)
		}

	case maven.TypeVersion:
		context.Level = LevelVersion
	}

	return context
}

// findLatestVersion 버전 목록에서 최신 버전을 찾음 (간단한 구현)
func findLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	// 간단한 정렬 기반 최신 버전 추정
	latest := versions[0]
	for _, v := range versions[1:] {
		if compareVersions(v, latest) > 0 {
			latest = v
		}
	}

	return latest
}

// compareVersions 두 버전을 비교 (간단한 구현)
func compareVersions(v1, v2 string) int {
	// SNAPSHOT 처리
	if strings.Contains(v1, "SNAPSHOT") && !strings.Contains(v2, "SNAPSHOT") {
		return -1
	}
	if !strings.Contains(v1, "SNAPSHOT") && strings.Contains(v2, "SNAPSHOT") {
		return 1
	}

	// 문자열 기반 비교 (향후 semantic versioning 라이브러리 사용 권장)
	if v1 > v2 {
		return 1
	} else if v1 < v2 {
		return -1
	}
	return 0
}

// enrichDirectoryEntry 디렉토리 엔트리에 Maven 컨텍스트 정보 추가
func enrichDirectoryEntry(entry *DirectoryEntry, parentPath string) {
	if entry == nil {
		return
	}

	// 엔트리의 전체 경로 구성
	fullPath := parentPath
	if fullPath != "" && !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}
	fullPath += entry.Name

	// 경로 분석
	pathInfo := parseMavenPath(fullPath)

	// 엔트리 타입 업데이트
	if entry.Type == maven.TypeDirectory {
		entry.Type = pathInfo.Type
	}

	// Maven 컨텍스트 생성 (간단한 버전)
	if pathInfo.Type != maven.TypeDirectory && pathInfo.Type != maven.TypeFile {
		entry.MavenContext = &maven.MavenContext{
			GroupID:    pathInfo.GroupID,
			ArtifactID: pathInfo.ArtifactID,
			Version:    pathInfo.Version,
			Level:      string(pathInfo.Type),
		}
	}
}

// handleArtifactPage artifact 상세 페이지 처리
func (h *MavenBrowserHandler) handleArtifactPage(c *fiber.Ctx, pathInfo *maven.PathInfo) error {
	// artifact 경로에서 버전 목록 수집
	artifactPath := strings.Trim(c.Params("*"), "/")

	// 버전 목록 수집
	versionEntries, err := h.collectVersions(artifactPath)
	if err != nil {
		h.logger.Error("Failed to collect versions",
			logging.F("path", artifactPath),
			logging.F("error", err),
		)
		return c.Status(500).SendString("버전 정보를 수집할 수 없습니다")
	}

	// 버전 목록 정렬 (파일이 아닌 디렉토리만)
	versions := make([]string, 0, len(versionEntries))
	for _, entry := range versionEntries {
		// 디렉토리이면서 파일 확장자가 없는 항목만 포함
		if (entry.Type == maven.TypeDirectory || entry.Type == TypeDirectory) &&
			!hasFileExtension(entry.Name) {
			// 버전처럼 보이거나 maven-metadata로 시작하지 않는 항목
			if isVersionLike(entry.Name) ||
				(!strings.HasPrefix(entry.Name, "maven-metadata") &&
					entry.Name != "." && entry.Name != "..") {
				versions = append(versions, entry.Name)
			}
		}
	}

	// 버전 정렬 (최신 버전이 먼저)
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	// 브레드크럼 생성
	breadcrumbParts := make([]BreadcrumbPart, 0)
	pathParts := strings.Split(artifactPath, "/")
	currentPath := ""

	for i, part := range pathParts[:len(pathParts)-1] { // 마지막 artifact 제외
		if i > 0 {
			currentPath += "/"
		}
		currentPath += part
		breadcrumbParts = append(breadcrumbParts, BreadcrumbPart{
			Name: part,
			Path: currentPath,
		})
	}

	// 기본 선택 버전 (최신 버전)
	selectedVersion := ""
	if len(versions) > 0 {
		selectedVersion = versions[0]
	}

	// artifact 데이터 구성
	artifactData := &MavenArtifactData{
		GroupID:         pathInfo.GroupID,
		ArtifactID:      pathInfo.ArtifactID,
		Versions:        versions,
		SelectedVersion: selectedVersion,
		BreadcrumbParts: breadcrumbParts,
	}

	// JSON API 요청인 경우
	if strings.Contains(c.Get("Accept"), MimeApplicationJSON) || c.Query("format") == FormatJSON {
		return c.JSON(artifactData)
	}

	// HTML 페이지 렌더링
	return c.Render("maven-artifact", artifactData)
}

// collectVersions 버전 목록 수집
func (h *MavenBrowserHandler) collectVersions(artifactPath string) ([]DirectoryEntry, error) {
	entryMap := make(map[string]*DirectoryEntry)

	for _, proxy := range h.config.Proxies {
		entries, err := h.fetchDirectoryFromMirror(proxy, artifactPath)
		if err != nil {
			h.logger.Warn("Failed to fetch from mirror",
				logging.F("mirror", proxy.Name),
				logging.F("error", err),
			)
			continue
		}

		// 엔트리 병합
		for _, entry := range entries {
			if existing, exists := entryMap[entry.Name]; exists {
				existing.Sources = append(existing.Sources, proxy.Name)
			} else {
				entry.Sources = []string{proxy.Name}
				entryMap[entry.Name] = &entry
			}
		}
	}

	// 맵을 슬라이스로 변환
	result := make([]DirectoryEntry, 0, len(entryMap))
	for _, entry := range entryMap {
		result = append(result, *entry)
	}

	return result, nil
}

// performGlobalSearch 전체 리포지토리 검색 수행
func (h *MavenBrowserHandler) performGlobalSearch(c *fiber.Ctx, searchQuery string) error {
	// 인덱스가 아직 빌드되지 않았으면 메시지 표시
	h.searchIndex.mutex.RLock()
	indexEntries := h.searchIndex.entries
	h.searchIndex.mutex.RUnlock()

	if len(indexEntries) == 0 {
		// 인덱스가 없으면 간단한 메시지 반환
		emptyData := &MavenBrowserData{
			Path:        "/",
			TreeRoot:    []*maven.GAVTreeNode{},
			SearchQuery: searchQuery,
		}

		if strings.Contains(c.Get("Accept"), MimeApplicationJSON) || c.Query("format") == FormatJSON {
			return c.JSON(emptyData)
		}

		return c.Status(503).SendString("검색 인덱스를 구축 중입니다. 잠시 후 다시 시도해주세요.")
	}

	// 인덱스에서 빠르게 검색
	query := strings.ToLower(searchQuery)
	matchingPaths := make(map[string]bool)

	for _, entry := range indexEntries {
		if strings.Contains(strings.ToLower(entry.Name), query) ||
			strings.Contains(strings.ToLower(entry.GroupID), query) ||
			strings.Contains(strings.ToLower(entry.ArtifactID), query) {
			// 매칭되는 경로와 상위 경로들 추가
			matchingPaths[entry.Path] = true

			// 상위 경로들도 추가 (트리 구조 유지)
			parts := strings.Split(strings.Trim(entry.Path, "/"), "/")
			for i := 1; i <= len(parts); i++ {
				parentPath := "/" + strings.Join(parts[:i], "/") + "/"
				matchingPaths[parentPath] = true
			}
		}
	}

	// 매칭된 경로들로 트리 구성
	searchResults := h.buildSearchResultTree(matchingPaths, query)

	h.logger.Info("Index search completed",
		logging.F("query", searchQuery),
		logging.F("matches", len(matchingPaths)),
		logging.F("results", len(searchResults)))

	// 검색 결과로 새로운 트리 구성
	resultData := &MavenBrowserData{
		Path:         "/",
		TreeRoot:     searchResults,
		SearchQuery:  searchQuery,
		TotalMirrors: len(h.config.Proxies),
	}

	// JSON API 요청인 경우
	if strings.Contains(c.Get("Accept"), MimeApplicationJSON) || c.Query("format") == FormatJSON {
		return c.JSON(resultData)
	}

	return c.Render("maven-browser", resultData)
}

// searchRecursively 재귀적으로 검색 수행
//
//nolint:unused // Used in advanced search functionality
func (h *MavenBrowserHandler) searchRecursively(
	nodes []*maven.GAVTreeNode,
	query, parentPath string,
	results *[]*maven.GAVTreeNode,
) {
	query = strings.ToLower(query)

	for _, node := range nodes {
		// 노드 복사본 생성 (원본 수정 방지)
		nodeCopy := &maven.GAVTreeNode{
			Name:       node.Name,
			Type:       node.Type,
			FullPath:   node.FullPath,
			ChildCount: node.ChildCount,
			Sources:    node.Sources,
			GroupID:    node.GroupID,
			ArtifactID: node.ArtifactID,
			Version:    node.Version,
		}

		// 현재 노드가 매칭되는지 확인
		nodeMatches := false
		if strings.Contains(strings.ToLower(node.Name), query) ||
			(node.GroupID != "" && strings.Contains(strings.ToLower(node.GroupID), query)) ||
			(node.ArtifactID != "" && strings.Contains(strings.ToLower(node.ArtifactID), query)) {
			nodeMatches = true
			nodeCopy.IsHighlighted = true
		}

		// 하위 항목 검색
		hasMatchingChildren := false
		if node.Type == LevelGroup || node.Type == LevelArtifact {
			// 하위 디렉토리 로드
			childPath := strings.TrimPrefix(node.FullPath, "/proxy/maven")
			childData, err := h.collectDirectoryData(childPath)

			if err == nil && childData.TreeRoot != nil && len(childData.TreeRoot) > 0 {
				// 하위 항목에서 검색
				childResults := make([]*maven.GAVTreeNode, 0)
				h.searchRecursively(childData.TreeRoot, query, node.FullPath, &childResults)

				if len(childResults) > 0 {
					hasMatchingChildren = true
					nodeCopy.Children = childResults
					nodeCopy.IsExpanded = true
				}
			}
		}

		// 현재 노드가 매칭되거나 하위에 매칭 항목이 있으면 결과에 추가
		if nodeMatches || hasMatchingChildren {
			*results = append(*results, nodeCopy)
		}
	}
}

// preloadPopularPaths 인기있는 경로를 미리 로드
func (h *MavenBrowserHandler) preloadPopularPaths() {
	// 설정 로드 대기
	time.Sleep(2 * time.Second)

	// 인기있는 최상위 그룹들
	popularPaths := []string{
		"/",                     // 루트
		"/org/",                 // org 그룹
		"/com/",                 // com 그룹
		"/io/",                  // io 그룹
		"/org/springframework/", // Spring Framework
		"/org/apache/",          // Apache
		"/com/google/",          // Google
	}

	for _, path := range popularPaths {
		// 이미 캐시되어 있으면 스킵
		if _, found := h.cache.Load(path); found {
			continue
		}

		// 설정이 아직 로드되지 않았으면 대기
		if h.config == nil || len(h.config.Proxies) == 0 {
			if err := h.config.ReadConfig(); err != nil {
				h.logger.Warn("Failed to read config for preload",
					logging.F("error", err))
				continue
			}
		}

		// 백그라운드에서 로드
		go func(p string) {
			data, err := h.collectDirectoryData(p)
			if err != nil {
				h.logger.Warn("Failed to preload path",
					logging.F("path", p),
					logging.F("error", err))
				return
			}

			// 캐시에 저장
			h.cache.Store(p, &cacheEntry{
				data:      data,
				timestamp: time.Now(),
			})

			h.logger.Info("Preloaded path",
				logging.F("path", p),
				logging.F("entries", len(data.Entries)))
		}(path)

		// 과부하 방지를 위해 약간의 지연
		time.Sleep(500 * time.Millisecond)
	}
}

// initializeIndex 인덱스 초기화
func (h *MavenBrowserHandler) initializeIndex() {
	// 설정 로드 대기
	time.Sleep(3 * time.Second)

	if h.config == nil || len(h.config.Proxies) == 0 {
		if err := h.config.ReadConfig(); err != nil {
			h.logger.Error("Failed to read config for index",
				logging.F("error", err))
			return
		}
	}

	// 저장 경로 결정
	storageDir := h.config.SearchIndex.StoragePath
	if storageDir == "" {
		storageDir = os.Getenv("STORAGE_DIR")
		if storageDir == "" {
			storageDir = "/tmp"
		}
		storageDir = filepath.Join(storageDir, "maven-index")
	}

	// 인덱스 저장소 초기화
	h.indexStorage = NewFileIndexStorage(storageDir)

	// 기존 인덱스 로드
	h.loadExistingIndex()
}

// loadExistingIndex 저장된 인덱스만 로드 (자동 빌드 안함)
func (h *MavenBrowserHandler) loadExistingIndex() {
	// 저장된 인덱스 로드 시도
	if h.indexStorage != nil {
		if entries, err := h.indexStorage.Load(); err == nil && len(entries) > 0 {
			// 마지막 수정 시간 확인
			if lastMod, err := h.indexStorage.GetLastModified(); err == nil {
				h.searchIndex.mutex.Lock()
				h.searchIndex.entries = entries
				h.searchIndex.lastBuild = lastMod
				h.searchIndex.mutex.Unlock()

				h.logger.Info("Search index loaded from storage",
					logging.F("entries", len(entries)),
					logging.F("age", time.Since(lastMod)))

				// 설정에 따라 자동 갱신 스케줄링
				if h.config.SearchIndex.AutoRebuildIntervalHours > 0 {
					go h.scheduleAutoRebuild()
				}
				return
			}
		}
	}

	// 저장된 인덱스가 없고 자동 빌드가 활성화되어 있으면 빌드
	if h.config.SearchIndex.AutoBuildOnStartup {
		h.logger.Info("Auto-building search index on startup")
		go h.buildSearchIndex()
	} else {
		h.logger.Info("Search index not found. Use 'proxyndctl maven-index build' to create index")
	}
}

// buildSearchIndex 검색 인덱스 구축
func (h *MavenBrowserHandler) buildSearchIndex() {
	startTime := time.Now()
	h.logger.Info("Starting search index build")

	newEntries := make([]maven.SearchIndexEntry, 0, 10000)

	// 병렬로 인덱싱 (최상위 디렉토리별)
	rootData, err := h.collectDirectoryData("/")
	if err != nil {
		h.logger.Error("Failed to get root data for indexing",
			logging.F("error", err))
		return
	}

	var mutex sync.Mutex
	var wg sync.WaitGroup

	// 최상위 그룹별로 병렬 인덱싱
	for _, node := range rootData.TreeRoot {
		wg.Add(1)
		go func(n *maven.GAVTreeNode) {
			defer wg.Done()

			localEntries := make([]maven.SearchIndexEntry, 0, 1000)
			h.indexNode(n, "", &localEntries)

			mutex.Lock()
			newEntries = append(newEntries, localEntries...)
			mutex.Unlock()
		}(node)
	}

	wg.Wait()

	// 인덱스 업데이트
	h.searchIndex.mutex.Lock()
	h.searchIndex.entries = newEntries
	h.searchIndex.lastBuild = time.Now()
	h.searchIndex.mutex.Unlock()

	// 파일에 저장
	if h.indexStorage != nil {
		if err := h.indexStorage.Save(newEntries); err != nil {
			h.logger.Error("Failed to save index", logging.F("error", err))
		}
	}

	h.logger.Info("Search index build completed",
		logging.F("entries", len(newEntries)),
		logging.F("duration", time.Since(startTime)))

	// 설정에 따라 자동 갱신 스케줄링
	if h.config.SearchIndex.AutoRebuildIntervalHours > 0 {
		go h.scheduleAutoRebuild()
	}
}

// scheduleAutoRebuild 자동 인덱스 재구축 스케줄링
func (h *MavenBrowserHandler) scheduleAutoRebuild() {
	if h.config.SearchIndex.AutoRebuildIntervalHours <= 0 {
		return
	}

	interval := time.Duration(h.config.SearchIndex.AutoRebuildIntervalHours) * time.Hour
	h.logger.Info("Scheduling auto-rebuild of search index",
		logging.F("interval", interval))

	time.Sleep(interval)
	h.buildSearchIndex()
}

// indexNode 노드를 인덱싱 (maven.GAVTreeNode 사용)
func (h *MavenBrowserHandler) indexNode(
	node *maven.GAVTreeNode,
	parentGroupID string,
	entries *[]maven.SearchIndexEntry,
) {
	if node == nil || len(*entries) > 50000 {
		return
	}

	// 인덱스 엔트리 생성
	entry := maven.SearchIndexEntry{
		Path:       node.FullPath,
		Name:       node.Name,
		Type:       node.Type,
		GroupID:    node.GroupID,
		ArtifactID: node.ArtifactID,
	}

	// GroupID 설정
	if entry.GroupID == "" && parentGroupID != "" {
		if parentGroupID == "/" || parentGroupID == "" {
			entry.GroupID = node.Name
		} else {
			entry.GroupID = parentGroupID + "." + node.Name
		}
	}

	*entries = append(*entries, entry)

	// 그룹이나 아티팩트면 하위 탐색
	if node.Type == LevelGroup || node.Type == LevelArtifact {
		childPath := strings.TrimPrefix(node.FullPath, "/proxy/maven")

		// 캐시 확인
		var childNodes []*maven.GAVTreeNode
		if cached, found := h.cache.Load(childPath); found {
			if cacheEntry, ok := cached.(*cacheEntry); ok && time.Since(cacheEntry.timestamp) < h.cacheTTL {
				childNodes = cacheEntry.data.TreeRoot
			}
		}

		// 캐시가 없으면 로드
		if childNodes == nil {
			if data, err := h.collectDirectoryData(childPath); err == nil {
				childNodes = data.TreeRoot
			}
		}

		// 하위 노드 인덱싱
		for _, child := range childNodes {
			h.indexNode(child, entry.GroupID, entries)
		}
	}
}

// indexDirectory 디렉토리를 재귀적으로 인덱싱
//
//nolint:unused // 향후 검색 기능 확장을 위해 유지
func (h *MavenBrowserHandler) indexDirectory(path, parentGroupID string, entries *[]maven.SearchIndexEntry) {
	// 캐시 확인
	if cached, found := h.cache.Load(path); found {
		if entry, ok := cached.(*cacheEntry); ok {
			if time.Since(entry.timestamp) < h.cacheTTL && entry.data != nil {
				// 캐시된 데이터에서 인덱싱
				for _, node := range entry.data.TreeRoot {
					h.addToIndex(node, path, parentGroupID, entries)
				}
				return
			}
		}
	}

	// 디렉토리 데이터 수집
	data, err := h.collectDirectoryData(path)
	if err != nil {
		h.logger.Warn("Failed to collect data for indexing",
			logging.F("path", path),
			logging.F("error", err))
		return
	}

	// 데이터에서 인덱싱
	for _, node := range data.TreeRoot {
		h.addToIndex(node, path, parentGroupID, entries)
	}
}

// addToIndex 노드를 인덱스에 추가
//
//nolint:unused // Used in search indexing functionality
func (h *MavenBrowserHandler) addToIndex(
	node *maven.GAVTreeNode,
	parentPath, parentGroupID string,
	entries *[]maven.SearchIndexEntry,
) {
	if node == nil {
		return
	}

	// 인덱스 엔트리 생성
	entry := maven.SearchIndexEntry{
		Path:       node.FullPath,
		Name:       node.Name,
		Type:       node.Type,
		GroupID:    node.GroupID,
		ArtifactID: node.ArtifactID,
	}

	// GroupID 설정
	if entry.GroupID == "" && parentGroupID != "" {
		if parentGroupID == "/" {
			entry.GroupID = node.Name
		} else {
			entry.GroupID = parentGroupID + "." + node.Name
		}
	}

	*entries = append(*entries, entry)

	// 너무 많은 엔트리 방지 (최대 50000개)
	if len(*entries) > 50000 {
		return
	}

	// 그룹이나 아티팩트면 하위 탐색
	if node.Type == LevelGroup || node.Type == LevelArtifact {
		childPath := strings.TrimPrefix(node.FullPath, "/proxy/maven")
		h.indexDirectory(childPath, entry.GroupID, entries)

		// 메모리 부담 줄이기 위해 잠시 대기
		if len(*entries)%1000 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// buildSearchResultTree 검색 결과로 트리 구성
func (h *MavenBrowserHandler) buildSearchResultTree(matchingPaths map[string]bool, query string) []*maven.GAVTreeNode {
	// 루트 노드들 구성
	rootNodes := make([]*maven.GAVTreeNode, 0)

	// 최상위 디렉토리 찾기
	for path := range matchingPaths {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 1 && parts[0] != "" {
			// 최상위 노드
			node := &maven.GAVTreeNode{
				Name:       parts[0],
				Type:       "group",
				FullPath:   path,
				IsExpanded: true,
			}

			// 하위 노드들 추가
			h.addChildrenFromPaths(node, matchingPaths, query)
			rootNodes = append(rootNodes, node)
		}
	}

	// 정렬
	sort.Slice(rootNodes, func(i, j int) bool {
		return rootNodes[i].Name < rootNodes[j].Name
	})

	return rootNodes
}

// addChildrenFromPaths 매칭된 경로에서 하위 노드 추가
func (h *MavenBrowserHandler) addChildrenFromPaths(
	parent *maven.GAVTreeNode,
	matchingPaths map[string]bool,
	query string,
) {
	parentPath := strings.Trim(parent.FullPath, "/")

	for path := range matchingPaths {
		trimmedPath := strings.Trim(path, "/")

		// 직접 하위 경로인지 확인
		if strings.HasPrefix(trimmedPath, parentPath+"/") {
			relativePath := strings.TrimPrefix(trimmedPath, parentPath+"/")
			parts := strings.Split(relativePath, "/")

			if len(parts) == 1 && parts[0] != "" {
				// 직접 하위 노드
				child := &maven.GAVTreeNode{
					Name:       parts[0],
					Type:       "group", // 타입은 나중에 정제 필요
					FullPath:   path,
					IsExpanded: true,
				}

				// 검색어가 포함되면 하이라이트
				if strings.Contains(strings.ToLower(child.Name), strings.ToLower(query)) {
					child.IsHighlighted = true
				}

				// 재귀적으로 하위 추가
				h.addChildrenFromPaths(child, matchingPaths, query)

				parent.Children = append(parent.Children, child)
			}
		}
	}

	// 자식 노드 정렬
	if len(parent.Children) > 0 {
		sort.Slice(parent.Children, func(i, j int) bool {
			return parent.Children[i].Name < parent.Children[j].Name
		})
	}
}

// SetConfig 설정 변경 (CLI 도구용)
func (h *MavenBrowserHandler) SetConfig(config *config.MavenProxySettings) {
	h.config = config
}

// CollectDirectoryData 디렉토리 데이터 수집 (CLI 도구용)
func (h *MavenBrowserHandler) CollectDirectoryData(path string) (*MavenBrowserData, error) {
	// 경로 정규화
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 기존 collectDirectoryData 메서드 활용
	return h.collectDirectoryData(path)
}

// searchInTree 트리에서 검색 쿼리와 일치하는 노드 찾기
func (h *MavenBrowserHandler) searchInTree(root []*maven.GAVTreeNode, query string) []*maven.GAVTreeNode {
	if len(root) == 0 || query == "" {
		return root
	}

	query = strings.ToLower(query)
	var results []*maven.GAVTreeNode

	for _, node := range root {
		h.searchNodeRecursive(node, query, &results)
	}

	return results
}

// searchNodeRecursive 재귀적으로 트리 노드 검색
func (h *MavenBrowserHandler) searchNodeRecursive(node *maven.GAVTreeNode, query string, results *[]*maven.GAVTreeNode) bool {
	if node == nil {
		return false
	}

	// 현재 노드 이름에서 검색
	nameMatches := strings.Contains(strings.ToLower(node.Name), query)

	// GroupID, ArtifactID에서도 검색
	groupMatches := strings.Contains(strings.ToLower(node.GroupID), query)
	artifactMatches := strings.Contains(strings.ToLower(node.ArtifactID), query)

	// 자식 노드 검색
	childMatches := false
	if len(node.Children) > 0 {
		for _, child := range node.Children {
			if h.searchNodeRecursive(child, query, results) {
				childMatches = true
			}
		}
	}

	// 매칭되거나 자식이 매칭되면 하이라이트
	if nameMatches || groupMatches || artifactMatches {
		node.IsHighlighted = true
		node.IsExpanded = true
		*results = append(*results, node)
		return true
	}

	if childMatches {
		node.IsExpanded = true
		return true
	}

	return false
}

// buildGAVTree 엔트리에서 GAV 트리 구조 생성
func (h *MavenBrowserHandler) buildGAVTree(entries []DirectoryEntry, basePath string) []*maven.GAVTreeNode {
	if len(entries) == 0 {
		return nil
	}

	var roots []*maven.GAVTreeNode

	for _, entry := range entries {
		// 디렉토리만 트리에 포함
		if entry.Type != maven.TypeDirectory && entry.Type != maven.TypeGroup &&
			entry.Type != maven.TypeArtifact && entry.Type != maven.TypeVersion {
			continue
		}

		node := &maven.GAVTreeNode{
			Name:       entry.Name,
			Type:       string(entry.Type),
			FullPath:   filepath.Join(basePath, entry.Name),
			Sources:    entry.Sources,
			ChildCount: 0,
		}

		// Maven 컨텍스트에서 GAV 정보 추출
		if entry.MavenContext != nil {
			node.GroupID = entry.MavenContext.GroupID
			node.ArtifactID = entry.MavenContext.ArtifactID
			node.Version = entry.MavenContext.Version
		}

		// 경로에서 GAV 추론
		pathParts := strings.Split(strings.Trim(node.FullPath, "/"), "/")
		if len(pathParts) > 0 {
			node.GroupID = strings.Join(pathParts[:len(pathParts)-1], ".")
			switch entry.Type {
			case maven.TypeArtifact:
				node.ArtifactID = entry.Name
			case maven.TypeVersion:
				node.Version = entry.Name
			}
		}

		roots = append(roots, node)
	}

	// 이름순 정렬
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Name < roots[j].Name
	})

	return roots
}
