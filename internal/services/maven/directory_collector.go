package maven

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"proxynd/internal/config"
	"proxynd/internal/domain/maven"
	"proxynd/internal/logging"
)

// directoryCollectorImpl DirectoryCollector 인터페이스 구현
type directoryCollectorImpl struct {
	config maven.ProxyConfig
	logger logging.Logger
	client *http.Client
}

// NewDirectoryCollector DirectoryCollector 생성자
func NewDirectoryCollector(config maven.ProxyConfig, logger logging.Logger) maven.DirectoryCollector {
	return &directoryCollectorImpl{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CollectDirectory 지정된 경로의 디렉토리 데이터를 모든 미러에서 수집
func (c *directoryCollectorImpl) CollectDirectory(ctx context.Context, path string) (*maven.DirectoryData, error) {
	c.logger.Info("Collecting directory data",
		logging.F("path", path),
	)

	data := &maven.DirectoryData{
		Path:     path,
		Entries:  []maven.Entry{},
		Mirrors:  []maven.MirrorStatus{},
		ViewMode: "list",
	}

	// 각 미러에서 데이터 수집 (병렬 처리)
	entryMap := make(map[string]*maven.Entry)
	var mapMutex sync.Mutex
	var wg sync.WaitGroup

	// 결과 채널
	type mirrorResult struct {
		status  maven.MirrorStatus
		entries []maven.Entry
	}
	proxies := c.config.GetProxies()
	resultChan := make(chan mirrorResult, len(proxies))

	// 각 미러에 대해 고루틴 실행
	for _, proxy := range proxies {
		wg.Add(1)
		go func(p config.MavenProxyServer) {
			defer wg.Done()

			mirrorStatus := maven.MirrorStatus{
				Name: p.Name,
				URL:  p.URL,
			}

			entries, err := c.CollectFromMirror(ctx, p, path)
			if err != nil {
				mirrorStatus.Available = false
				mirrorStatus.Error = err.Error()
				c.logger.Warn("Mirror unavailable",
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

	// 엔트리 맵을 슬라이스로 변환
	for _, entry := range entryMap {
		data.Entries = append(data.Entries, *entry)
	}

	// 엔트리 정렬 (디렉토리 먼저, 알파벳순)
	c.sortEntries(data.Entries)

	data.TotalMirrors = len(proxies)

	c.logger.Info("Directory data collected",
		logging.F("path", path),
		logging.F("entries", len(data.Entries)),
		logging.F("availableMirrors", c.countAvailableMirrors(data.Mirrors)),
	)

	return data, nil
}

// CollectFromMirror 특정 미러에서 디렉토리 데이터 수집
func (c *directoryCollectorImpl) CollectFromMirror(ctx context.Context, mirror config.MavenProxyServer, path string) ([]maven.Entry, error) { //nolint:lll
	// 미러 URL 구성
	baseURL := strings.TrimRight(mirror.URL, "/")
	cleanPath := strings.Trim(path, "/")

	var fullURL string
	if cleanPath == "" {
		fullURL = baseURL + "/"
	} else {
		fullURL = baseURL + "/" + cleanPath + "/"
	}

	// HTTP 요청
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ProxyND/1.0 (Maven Repository Browser)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 실패: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 상태 코드: %d", resp.StatusCode)
	}

	// HTML 파싱을 통한 디렉토리 목록 추출
	entries, err := c.parseDirectoryListing(resp, baseURL, cleanPath)
	if err != nil {
		// HTML 파싱 실패 시 메타데이터 기반으로 시도
		return c.parseMetadata(mirror, cleanPath)
	}

	return entries, nil
}

// GetMirrorStatus 미러 상태 확인
func (c *directoryCollectorImpl) GetMirrorStatus(ctx context.Context, mirror config.MavenProxyServer) (*maven.MirrorStatus, error) { //nolint:lll
	status := &maven.MirrorStatus{
		Name: mirror.Name,
		URL:  mirror.URL,
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", mirror.URL, nil)
	if err != nil {
		status.Available = false
		status.Error = err.Error()
		return status, nil
	}

	req.Header.Set("User-Agent", "ProxyND/1.0 (Health Check)")

	resp, err := c.client.Do(req)
	if err != nil {
		status.Available = false
		status.Error = err.Error()
		return status, nil
	}
	defer func() { _ = resp.Body.Close() }()

	status.Available = resp.StatusCode == http.StatusOK
	if !status.Available {
		status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return status, nil
}

// parseDirectoryListing HTML 디렉토리 목록 파싱
func (c *directoryCollectorImpl) parseDirectoryListing(resp *http.Response, baseURL, cleanPath string) ([]maven.Entry, error) { //nolint:lll
	// 간단한 HTML 파싱 - Apache/Nginx 스타일 디렉토리 목록
	body := make([]byte, 64*1024) // 64KB 제한
	n, err := resp.Body.Read(body)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	html := string(body[:n])
	entries := []maven.Entry{}

	// 간단한 링크 추출 (href="...")
	lines := strings.Split(html, "\n")
	for _, line := range lines {
		if strings.Contains(line, "href=") {
			entry := c.extractEntryFromLine(line, cleanPath)
			if entry != nil && entry.Name != ".." && entry.Name != "." {
				entries = append(entries, *entry)
			}
		}
	}

	return entries, nil
}

// extractEntryFromLine HTML 라인에서 엔트리 추출
func (c *directoryCollectorImpl) extractEntryFromLine(line, parentPath string) *maven.Entry {
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

	entryName := decodedHref
	entryType := maven.TypeFile

	// 디렉토리 판별
	if strings.HasSuffix(decodedHref, "/") {
		// 이름이 /로 끝나는 경우
		entryName = strings.TrimSuffix(decodedHref, "/")
		entryType = maven.TypeDirectory
	} else if !c.hasFileExtension(decodedHref) {
		// 파일 확장자가 없으면 디렉토리로 간주
		entryType = maven.TypeDirectory
	}

	// Maven 타입 결정
	mavenType := c.determineMavenType(entryName, parentPath)
	if mavenType != maven.TypeFile {
		entryType = mavenType
	}

	return &maven.Entry{
		Name: entryName,
		Type: entryType,
	}
}

// parseMetadata Maven 메타데이터 기반 파싱
func (c *directoryCollectorImpl) parseMetadata(mirror config.MavenProxyServer, cleanPath string) ([]maven.Entry, error) { //nolint:lll
	// maven-metadata.xml 파일 시도
	metadataURL := strings.TrimRight(mirror.URL, "/") + "/" + cleanPath + "/maven-metadata.xml"

	resp, err := c.client.Get(metadataURL)
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

	entries := []maven.Entry{}

	// 버전 정보에서 디렉토리 생성
	for _, version := range metadata.Versioning.Versions {
		entries = append(entries, maven.Entry{
			Name: version,
			Type: maven.TypeVersion,
		})
	}

	return entries, nil
}

// 헬퍼 메서드들

// hasFileExtension 파일 확장자가 있는지 확인
func (c *directoryCollectorImpl) hasFileExtension(name string) bool {
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

// determineMavenType Maven 타입 결정
func (c *directoryCollectorImpl) determineMavenType(name, parentPath string) maven.EntryType {
	// 경로 분석을 통한 타입 결정
	fullPath := parentPath
	if fullPath != "" && !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}
	fullPath += name
	_ = fullPath // TODO: use fullPath in logic or remove if not needed

	// 버전 패턴 확인
	if c.isVersionLike(name) {
		return maven.TypeVersion
	}

	// 그룹/아티팩트 패턴 확인
	if c.isLikelyArtifact(name) {
		return maven.TypeArtifact
	}

	// 파일 확장자가 있으면 파일
	if c.hasFileExtension(name) {
		return maven.TypeFile
	}

	// 기본적으로 그룹/디렉토리
	return maven.TypeGroup
}

// isVersionLike 문자열이 버전처럼 보이는지 확인
func (c *directoryCollectorImpl) isVersionLike(s string) bool {
	if len(s) == 0 {
		return false
	}
	// 첫 문자가 숫자인지 확인
	return s[0] >= '0' && s[0] <= '9'
}

// isLikelyArtifact 이름이 아티팩트처럼 보이는지 판단
func (c *directoryCollectorImpl) isLikelyArtifact(name string) bool {
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

	return false
}

// sortEntries 엔트리 정렬
func (c *directoryCollectorImpl) sortEntries(entries []maven.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		// 디렉토리를 먼저 표시
		if entries[i].Type != entries[j].Type {
			if entries[i].Type == maven.TypeDirectory || entries[i].Type == maven.TypeGroup {
				return true
			}
			if entries[j].Type == maven.TypeDirectory || entries[j].Type == maven.TypeGroup {
				return false
			}
		}
		return entries[i].Name < entries[j].Name
	})
}

// countAvailableMirrors 사용 가능한 미러 개수 계산
func (c *directoryCollectorImpl) countAvailableMirrors(mirrors []maven.MirrorStatus) int {
	count := 0
	for _, mirror := range mirrors {
		if mirror.Available {
			count++
		}
	}
	return count
}
