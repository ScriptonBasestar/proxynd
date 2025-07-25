package maven

import (
	"encoding/xml"
	"time"
)

// MavenMetadata Maven 메타데이터 XML 구조
type MavenMetadata struct {
	XMLName    xml.Name `xml:"metadata"`
	GroupID    string   `xml:"groupId,omitempty"`
	ArtifactID string   `xml:"artifactId,omitempty"`
	Version    string   `xml:"version,omitempty"`
	Versioning struct {
		Latest   string   `xml:"latest,omitempty"`
		Release  string   `xml:"release,omitempty"`
		Versions []string `xml:"versions>version,omitempty"`
	} `xml:"versioning,omitempty"`
}

// MavenEntryType Maven 엔트리 타입
type MavenEntryType string

const (
	TypeDirectory MavenEntryType = "directory"
	TypeFile      MavenEntryType = "file"
	TypeGroup     MavenEntryType = "group"    // Maven Group 경로 (org/apache/...)
	TypeArtifact  MavenEntryType = "artifact" // Artifact 디렉토리
	TypeVersion   MavenEntryType = "version"  // Version 디렉토리
	TypeMetadata  MavenEntryType = "metadata" // maven-metadata.xml 등
)

// MavenContext Maven 컨텍스트 정보
type MavenContext struct {
	GroupID     string   `json:"groupId,omitempty"`     // org.apache.httpcomponents.client5
	ArtifactID  string   `json:"artifactId,omitempty"`  // httpclient5
	Version     string   `json:"version,omitempty"`     // 5.3.1
	Versions    []string `json:"versions,omitempty"`    // 사용 가능한 버전들
	Latest      string   `json:"latest,omitempty"`      // 최신 버전
	Description string   `json:"description,omitempty"` // 아티팩트 설명
	Level       string   `json:"level,omitempty"`       // "group", "artifact", "version"
}

// MirrorStatus 미러 서버 상태
type MirrorStatus struct {
	Name      string `json:"name"`            // 미러 이름
	URL       string `json:"url"`             // 미러 URL
	Available bool   `json:"available"`       // 사용 가능 여부
	Error     string `json:"error,omitempty"` // 에러 메시지
}

// MavenPathInfo Maven 경로 분석 정보
type MavenPathInfo struct {
	Type       MavenEntryType `json:"type"`
	GroupID    string         `json:"groupId"`
	ArtifactID string         `json:"artifactId,omitempty"`
	Version    string         `json:"version,omitempty"`
	Level      int            `json:"level"` // 경로 깊이
}

// DirectoryEntry 디렉토리 엔트리 (확장됨)
type DirectoryEntry struct {
	Name         string         `json:"name"`
	Type         MavenEntryType `json:"type"` // Maven 타입 사용
	Size         int64          `json:"size,omitempty"`
	LastModified time.Time      `json:"lastModified,omitempty"`
	Sources      []string       `json:"sources"` // 어떤 미러에서 발견되었는지

	// Maven 특화 정보
	MavenContext *MavenContext `json:"mavenContext,omitempty"`
}

// MavenBrowserData 템플릿 데이터 (확장됨)
type MavenBrowserData struct {
	Path         string           `json:"path"`
	Entries      []DirectoryEntry `json:"entries"`
	Mirrors      []MirrorStatus   `json:"mirrors"`
	TotalMirrors int              `json:"totalMirrors"`

	// Maven 컨텍스트 정보
	PathInfo *MavenPathInfo `json:"pathInfo,omitempty"`
	Context  *MavenContext  `json:"context,omitempty"`

	// GAV 트리 구조
	TreeRoot    []*GAVTreeNode `json:"treeRoot,omitempty"`
	ViewMode    string         `json:"viewMode"` // "tree" 또는 "list"
	SearchQuery string         `json:"searchQuery,omitempty"`
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

// SearchIndexEntry 검색 인덱스 엔트리
type SearchIndexEntry struct {
	Path       string
	Name       string
	Type       string
	GroupID    string
	ArtifactID string
}

// GAVTreeNode GAV 트리 노드 (handlers/proxy/maven_gav_builder.go에서 이동 예정)
type GAVTreeNode struct {
	Name          string         `json:"name"`
	Type          string         `json:"type"`       // "group", "artifact", "version"
	FullPath      string         `json:"fullPath"`   // 전체 경로
	ChildCount    int            `json:"childCount"` // 하위 항목 수
	Children      []*GAVTreeNode `json:"children,omitempty"`
	Sources       []string       `json:"sources"`       // 발견된 미러 목록
	IsExpanded    bool           `json:"isExpanded"`    // UI용: 확장 상태
	IsHighlighted bool           `json:"isHighlighted"` // UI용: 검색 결과 하이라이트

	// Maven 특화 정보
	GroupID    string `json:"groupId,omitempty"`
	ArtifactID string `json:"artifactId,omitempty"`
	Version    string `json:"version,omitempty"`
}

// IndexStorage 인덱스 저장소 인터페이스
type IndexStorage interface {
	Save(entries []SearchIndexEntry) error
	Load() ([]SearchIndexEntry, error)
	GetLastModified() (time.Time, error)
}
