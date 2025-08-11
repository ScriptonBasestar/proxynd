package proxy

import (
	"fmt"
	"sort"
	"strings"

	"proxynd/internal/domain/maven"
)

// GAVTreeNode Group-Artifact-Version 트리 노드
type GAVTreeNode struct {
	Name          string         `json:"name"`
	Type          string         `json:"type"` // "group", "artifact", "version", "file"
	FullPath      string         `json:"fullPath"`
	Children      []*GAVTreeNode `json:"children,omitempty"`
	ChildCount    int            `json:"childCount"` // 직접 하위 항목 수
	SubCount      int            `json:"subCount"`   // 전체 하위 항목 수 (재귀)
	IsExpanded    bool           `json:"isExpanded"`
	IsHighlighted bool           `json:"isHighlighted"` // 검색 결과 하이라이트
	Sources       []string       `json:"sources,omitempty"`

	// GAV 정보
	GroupID    string `json:"groupId,omitempty"`
	ArtifactID string `json:"artifactId,omitempty"`
	Version    string `json:"version,omitempty"`
}

// buildGAVTree 엔트리 목록을 GAV 트리로 구성 (향후 사용 예정)
// nolint:unused
func buildGAVTree(entries []DirectoryEntry, currentPath string) []*GAVTreeNode {
	// 루트 레벨에서는 모든 엔트리를 분석하여 완전한 GAV 구조 생성
	if currentPath == "" || currentPath == "/" {
		nodes := buildCompleteGAVTree(entries)
		// 디버깅: 노드가 제대로 생성되었는지 확인
		for i, node := range nodes {
			if i < 3 { // 처음 3개만 출력
				fmt.Printf("[DEBUG] buildGAVTree root node[%d]: name=%s, type=%s, childCount=%d\n",
					i, node.Name, node.Type, node.ChildCount)
			}
		}
		return nodes
	}

	// 하위 레벨에서는 현재 경로에 맞는 구조 생성
	return buildPartialGAVTree(entries, currentPath)
}

// buildCompleteGAVTree 전체 GAV 트리 구성 (루트 레벨) - 향후 사용 예정
// nolint:unused
func buildCompleteGAVTree(entries []DirectoryEntry) []*GAVTreeNode {
	// 루트 레벨에서는 최상위 디렉토리만 표시
	// 예: org, com, net 등
	nodes := make([]*GAVTreeNode, 0)
	seen := make(map[string]bool)

	for _, entry := range entries {
		// 디렉토리 타입만 처리
		if entry.Type != typeDirectory && entry.Type != maven.TypeDirectory && entry.Type != maven.TypeGroup {
			continue
		}

		// 중복 제거
		if seen[entry.Name] {
			continue
		}
		seen[entry.Name] = true

		// 루트 레벨 노드 생성
		node := &GAVTreeNode{
			Name:       entry.Name,
			Type:       "group",
			FullPath:   fmt.Sprintf("/proxy/maven/%s/", entry.Name),
			Children:   []*GAVTreeNode{},
			GroupID:    entry.Name,
			Sources:    entry.Sources,
			ChildCount: 1, // 확장 가능하도록 표시
		}

		nodes = append(nodes, node)
	}

	// 정렬
	sortGAVNodes(nodes)
	return nodes
}

// buildPartialGAVTree 부분 GAV 트리 구성 (하위 레벨)
//
//nolint:unused // GAV 트리 빌더의 완전한 API 제공을 위해 유지
func buildPartialGAVTree(entries []DirectoryEntry, currentPath string) []*GAVTreeNode {
	// 현재 경로를 분석하여 컨텍스트 파악
	pathInfo := parseMavenPath(currentPath)
	nodes := make([]*GAVTreeNode, 0)

	for _, entry := range entries {
		// 현재 레벨에 맞는 타입 결정
		nodeType := determineNodeType(pathInfo, entry)

		// 디버깅: springframework 처리 확인
		if entry.Name == "springframework" {
			fmt.Printf("[DEBUG] springframework: entry.Type=%s, nodeType=%s, pathInfo.Type=%s\n",
				entry.Type, nodeType, pathInfo.Type)
		}

		node := &GAVTreeNode{
			Name:       entry.Name,
			Type:       nodeType,
			FullPath:   buildFullPath(currentPath, entry.Name, nodeType),
			Children:   []*GAVTreeNode{},
			Sources:    entry.Sources,
			ChildCount: 1, // 기본값 설정 (확장 가능하도록)
		}

		// GAV 정보 설정
		switch nodeType {
		case "group":
			if pathInfo.GroupID == "" {
				node.GroupID = entry.Name
			} else {
				node.GroupID = pathInfo.GroupID + "." + entry.Name
			}
		case "artifact":
			node.GroupID = pathInfo.GroupID
			node.ArtifactID = entry.Name
		case levelVersion:
			node.GroupID = pathInfo.GroupID
			node.ArtifactID = pathInfo.ArtifactID
			node.Version = entry.Name
		}

		nodes = append(nodes, node)
	}

	// 각 노드의 하위 항목 수 계산 (실제로는 API 호출이 필요하지만 여기서는 추정)
	for _, node := range nodes {
		if node.Type != typeFile {
			// 디렉토리인 경우 하위 항목이 있을 수 있음으로 표시
			node.ChildCount = 1 // 최소 1개로 설정하여 확장 가능하도록
		}
	}

	sortGAVNodes(nodes)
	return nodes
}

// determineNodeType 현재 경로 컨텍스트에서 노드 타입 결정
//
//nolint:unused // GAV 트리 빌더의 완전한 API 제공을 위해 유지
func determineNodeType(pathInfo *maven.PathInfo, entry DirectoryEntry) string {
	// 디렉토리 타입 확인 - 다양한 형태를 모두 처리
	if entry.Type != typeDirectory && entry.Type != maven.TypeDirectory &&
		entry.Type != maven.TypeGroup && entry.Type != maven.TypeArtifact {
		// 이름이 /로 끝나면 디렉토리로 처리
		if !strings.HasSuffix(entry.Name, "/") {
			return "file"
		}
	}

	// Maven 구조: group (org/apache/...) -> artifact (httpclient5) -> version (5.3.1)
	// 경로 레벨과 패턴으로 타입 추정
	switch pathInfo.Type {
	case maven.TypeDirectory:
		// 루트 디렉토리에서는 모든 디렉토리가 그룹 시작
		return "group"

	case maven.TypeGroup:
		// 그룹 경로에서는 다음은 또 다른 그룹이거나 아티팩트
		// 예: org/apache에서 httpcomponents는 그룹, httpclient5는 아티팩트

		// 하이픈을 포함하거나 모두 소문자면 아티팩트일 가능성이 높음
		if isLikelyArtifact(entry.Name) {
			return "artifact"
		}
		// 아니면 계속 그룹
		return "group"

	case maven.TypeArtifact:
		// 아티팩트 하위는 대부분 버전
		if isVersionLike(entry.Name) {
			return "version"
		}
		// 버전처럼 보이지 않으면 일반 디렉토리
		return "directory"

	case maven.TypeVersion:
		// 버전 하위는 파일 또는 일반 디렉토리
		return "directory"

	default:
		// 기타 경우는 기본적으로 디렉토리
		return "directory"
	}
}

// buildFullPath 전체 경로 생성
//
//nolint:unused // GAV 트리 빌더의 완전한 API 제공을 위해 유지
func buildFullPath(currentPath, name, nodeType string) string {
	base := "/proxy/maven"

	// currentPath가 있으면 추가 (앞에 /가 없으면 추가)
	if currentPath != "" && currentPath != "/" {
		if !strings.HasPrefix(currentPath, "/") {
			base += "/"
		}
		base += currentPath
	}

	// base가 /로 끝나지 않으면 추가
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	// 파일이면 그대로, 디렉토리면 / 추가
	if nodeType == "file" {
		return base + name
	}
	return base + name + "/"
}

// calculateCounts 재귀적으로 하위 항목 수 계산
//
//nolint:unused // GAV 트리 통계 기능을 위해 유지
func calculateCounts(node *GAVTreeNode) (directCount, totalCount int) {
	if node == nil {
		return 0, 0
	}

	directCount = len(node.Children)
	totalCount = directCount

	for _, child := range node.Children {
		_, childTotal := calculateCounts(child)
		totalCount += childTotal
	}

	node.ChildCount = directCount
	node.SubCount = totalCount

	return directCount, totalCount
}

// estimateChildCount 하위 항목 수 추정 (실제로는 서버에서 가져와야 함)
//
//nolint:unused // GAV 트리 통계 기능을 위해 유지
func estimateChildCount(node *GAVTreeNode) int {
	switch node.Type {
	case "group":
		return 5 // 예시: 평균적으로 5개의 하위 그룹/아티팩트
	case "artifact":
		return 10 // 예시: 평균적으로 10개의 버전
	case "version":
		return 15 // 예시: 평균적으로 15개의 파일
	default:
		return 0
	}
}

// sortGAVNodes GAV 노드 정렬
//
//nolint:unused // GAV 트리 정렬 기능을 위해 유지
func sortGAVNodes(nodes []*GAVTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		// 타입별 우선순위: group > artifact > version > directory > file
		typeOrder := map[string]int{
			"group":     1,
			"artifact":  2,
			"version":   3,
			"directory": 4,
			"file":      5,
		}

		orderI := typeOrder[nodes[i].Type]
		orderJ := typeOrder[nodes[j].Type]

		if orderI != orderJ {
			return orderI < orderJ
		}

		// 같은 타입이면 이름순
		return nodes[i].Name < nodes[j].Name
	})

	// 재귀적으로 자식들도 정렬
	for _, node := range nodes {
		if len(node.Children) > 0 {
			sortGAVNodes(node.Children)
		}
	}
}

// searchInTree 트리에서 검색하고 매칭되는 노드 하이라이트
//
//nolint:unused // GAV 트리 검색 기능을 위해 유지
func searchInTree(nodes []*GAVTreeNode, query string) []*GAVTreeNode {
	query = strings.ToLower(query)

	for _, node := range nodes {
		// 노드 이름이나 GAV 정보에서 검색
		matches := false

		if strings.Contains(strings.ToLower(node.Name), query) {
			matches = true
		} else if node.GroupID != "" && strings.Contains(strings.ToLower(node.GroupID), query) {
			matches = true
		} else if node.ArtifactID != "" && strings.Contains(strings.ToLower(node.ArtifactID), query) {
			matches = true
		} else if node.Version != "" && strings.Contains(strings.ToLower(node.Version), query) {
			matches = true
		}

		// 매칭되면 하이라이트하고 부모 노드들도 확장
		if matches {
			node.IsHighlighted = true
			node.IsExpanded = true
			expandParents(nodes, node)
		}

		// 자식 노드들도 검색
		if len(node.Children) > 0 {
			searchInTree(node.Children, query)

			// 자식 중 하나라도 하이라이트되어 있으면 이 노드도 확장
			for _, child := range node.Children {
				if child.IsHighlighted || child.IsExpanded {
					node.IsExpanded = true
					break
				}
			}
		}
	}

	return nodes
}

// expandParents 부모 노드들을 확장 상태로 설정
//
//nolint:unused // GAV 트리 확장 기능을 위해 유지
func expandParents(nodes []*GAVTreeNode, target *GAVTreeNode) {
	for _, node := range nodes {
		if containsChild(node, target) {
			node.IsExpanded = true
			expandParents(nodes, node)
		}
	}
}

// containsChild 노드가 특정 자식을 포함하는지 확인
//
//nolint:unused // GAV 트리 검색 기능을 위해 유지
func containsChild(parent, target *GAVTreeNode) bool {
	for _, child := range parent.Children {
		if child == target {
			return true
		}
		if containsChild(child, target) {
			return true
		}
	}
	return false
}

// resetTreeHighlight 트리의 모든 하이라이트와 확장 상태 초기화
//
//nolint:unused // GAV 트리 상태 관리를 위해 유지
func resetTreeHighlight(nodes []*GAVTreeNode) {
	for _, node := range nodes {
		node.IsHighlighted = false
		node.IsExpanded = false

		if len(node.Children) > 0 {
			resetTreeHighlight(node.Children)
		}
	}
}
