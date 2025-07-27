package maven

import (
	"context"
	"time"

	"proxynd/internal/config"
)

// DirectoryCollector Maven 디렉토리 수집 인터페이스
type DirectoryCollector interface {
	// CollectDirectory 지정된 경로의 디렉토리 데이터를 모든 미러에서 수집
	CollectDirectory(ctx context.Context, path string) (*DirectoryData, error)

	// CollectFromMirror 특정 미러에서 디렉토리 데이터 수집
	CollectFromMirror(ctx context.Context, mirror config.MavenProxyServer, path string) ([]Entry, error)

	// GetMirrorStatus 미러 상태 확인
	GetMirrorStatus(ctx context.Context, mirror config.MavenProxyServer) (*MirrorStatus, error)
}

// SearchService Maven 검색 서비스 인터페이스
type SearchService interface {
	// Search 아티팩트 검색 수행
	Search(ctx context.Context, query string) (*SearchResult, error)

	// IndexArtifacts 아티팩트 인덱스 구축
	IndexArtifacts(ctx context.Context) error

	// GetIndexStats 인덱스 통계 조회
	GetIndexStats(ctx context.Context) (*IndexStats, error)

	// UpdateIndex 특정 경로의 인덱스 업데이트
	UpdateIndex(ctx context.Context, path string) error
}

// CacheManager Maven 캐시 관리 인터페이스
type CacheManager interface {
	// Get 캐시에서 데이터 조회
	Get(ctx context.Context, key string) (*CacheEntry, bool)

	// Set 캐시에 데이터 저장
	Set(ctx context.Context, key string, data interface{}, ttl time.Duration) error

	// Delete 캐시에서 데이터 삭제
	Delete(ctx context.Context, key string) error

	// PreloadPopularPaths 인기 경로 사전 캐싱
	PreloadPopularPaths(ctx context.Context) error

	// GetStats 캐시 통계 조회
	GetStats(ctx context.Context) (*CacheStats, error)
}

// PathAnalyzer Maven 경로 분석 인터페이스
type PathAnalyzer interface {
	// ParsePath Maven 경로를 분석하여 구조화된 정보 반환
	ParsePath(path string) (*PathInfo, error)

	// IsVersionLike 문자열이 버전과 유사한지 확인
	IsVersionLike(s string) bool

	// IsLikelyArtifact 이름이 아티팩트와 유사한지 확인
	IsLikelyArtifact(name string) bool

	// ExtractGAV 경로에서 GroupID, ArtifactID, Version 추출
	ExtractGAV(path string) (groupID, artifactID, version string)
}

// BrowserHandler Maven 브라우저 핸들러 인터페이스
type BrowserHandler interface {
	// Handle HTTP 요청 처리
	Handle(ctx context.Context, request *BrowserRequest) (*BrowserResponse, error)

	// IsBrowserRequest 요청이 브라우저 요청인지 확인
	IsBrowserRequest(request *BrowserRequest) bool
}

// 도메인 모델들

// DirectoryData 디렉토리 데이터
type DirectoryData struct {
	Path         string         `json:"path"`
	Entries      []Entry        `json:"entries"`
	Mirrors      []MirrorStatus `json:"mirrors"`
	TotalMirrors int            `json:"totalMirrors"`
	PathInfo     *PathInfo      `json:"pathInfo,omitempty"`
	Context      *Context       `json:"context,omitempty"`
	TreeRoot     []*TreeNode    `json:"treeRoot,omitempty"`
	ViewMode     string         `json:"viewMode"`
	SearchQuery  string         `json:"searchQuery,omitempty"`
}

// Entry 디렉토리 엔트리
type Entry struct {
	Name         string    `json:"name"`
	Type         EntryType `json:"type"`
	Size         int64     `json:"size,omitempty"`
	LastModified time.Time `json:"lastModified,omitempty"`
	Sources      []string  `json:"sources"`
	Context      *Context  `json:"context,omitempty"`
}

// EntryType Maven 엔트리 타입 (models.go에서 가져옴)
type EntryType = MavenEntryType

// PathInfo Maven 경로 정보
type PathInfo struct {
	Type       EntryType `json:"type"`
	GroupID    string    `json:"groupId"`
	ArtifactID string    `json:"artifactId,omitempty"`
	Version    string    `json:"version,omitempty"`
	Level      int       `json:"level"`
}

// Context Maven 컨텍스트
type Context struct {
	GroupID     string   `json:"groupId,omitempty"`
	ArtifactID  string   `json:"artifactId,omitempty"`
	Version     string   `json:"version,omitempty"`
	Versions    []string `json:"versions,omitempty"`
	Latest      string   `json:"latest,omitempty"`
	Description string   `json:"description,omitempty"`
	Level       string   `json:"level,omitempty"`
}

// TreeNode GAV 트리 노드
type TreeNode struct {
	Name     string      `json:"name"`
	Type     EntryType   `json:"type"`
	Children []*TreeNode `json:"children,omitempty"`
	Context  *Context    `json:"context,omitempty"`
}

// SearchResult 검색 결과
type SearchResult struct {
	Query      string           `json:"query"`
	Results    []SearchArtifact `json:"results"`
	TotalCount int              `json:"totalCount"`
	SearchTime time.Duration    `json:"searchTime"`
}

// SearchArtifact 검색된 아티팩트
type SearchArtifact struct {
	GroupID     string   `json:"groupId"`
	ArtifactID  string   `json:"artifactId"`
	Versions    []string `json:"versions"`
	Latest      string   `json:"latest"`
	Description string   `json:"description,omitempty"`
	Score       float64  `json:"score"`
}

// IndexStats 인덱스 통계
type IndexStats struct {
	TotalArtifacts int       `json:"totalArtifacts"`
	TotalVersions  int       `json:"totalVersions"`
	LastUpdated    time.Time `json:"lastUpdated"`
	IndexSize      int64     `json:"indexSize"`
}

// CacheEntry 캐시 엔트리
type CacheEntry struct {
	Key        string      `json:"key"`
	Data       interface{} `json:"data"`
	ExpiresAt  time.Time   `json:"expiresAt"`
	CreatedAt  time.Time   `json:"createdAt"`
	AccessedAt time.Time   `json:"accessedAt"`
}

// CacheStats 캐시 통계
type CacheStats struct {
	TotalEntries int     `json:"totalEntries"`
	HitRate      float64 `json:"hitRate"`
	MissRate     float64 `json:"missRate"`
	MemoryUsage  int64   `json:"memoryUsage"`
}

// BrowserRequest 브라우저 요청
type BrowserRequest struct {
	Path        string            `json:"path"`
	QueryParams map[string]string `json:"queryParams"`
	Headers     map[string]string `json:"headers"`
}

// BrowserResponse 브라우저 응답
type BrowserResponse struct {
	Data        *DirectoryData `json:"data"`
	StatusCode  int            `json:"statusCode"`
	ContentType string         `json:"contentType"`
}
