package maven

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"proxynd/internal/config"
	"proxynd/internal/domain/maven"
	"proxynd/logging"
)

// searchServiceImpl SearchService 인터페이스 구현
type searchServiceImpl struct {
	config     maven.ProxyConfig
	logger     logging.Logger
	collector  maven.DirectoryCollector
	indexPath  string
	index      *searchIndex
	indexMutex sync.RWMutex
}

// searchIndex 검색 인덱스 구조
type searchIndex struct {
	Artifacts   map[string]*maven.SearchArtifact `json:"artifacts"`
	LastUpdated time.Time                        `json:"lastUpdated"`
	Version     int                              `json:"version"`
}

// NewSearchService SearchService 생성자
func NewSearchService(config maven.ProxyConfig, logger logging.Logger, collector maven.DirectoryCollector) maven.SearchService {
	service := &searchServiceImpl{
		config:    config,
		logger:    logger,
		collector: collector,
		indexPath: filepath.Join("/tmp", "search_index.json"),
		index: &searchIndex{
			Artifacts: make(map[string]*maven.SearchArtifact),
			Version:   1,
		},
	}

	// 기존 인덱스 로드 시도
	if err := service.loadIndex(); err != nil {
		logger.Warn("Failed to load existing search index", logging.F("error", err))
	}

	return service
}

// Search 아티팩트 검색 수행
func (s *searchServiceImpl) Search(ctx context.Context, query string) (*maven.SearchResult, error) {
	startTime := time.Now()

	s.logger.Info("Performing artifact search",
		logging.F("query", query),
	)

	s.indexMutex.RLock()
	defer s.indexMutex.RUnlock()

	var results []maven.SearchArtifact
	queryLower := strings.ToLower(query)

	// 검색 실행
	for _, artifact := range s.index.Artifacts {
		score := s.calculateRelevanceScore(artifact, queryLower)
		if score > 0 {
			// 결과에 스코어 추가
			resultArtifact := *artifact
			resultArtifact.Score = score
			results = append(results, resultArtifact)
		}
	}

	// 스코어 순으로 정렬
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 최대 50개 결과로 제한
	if len(results) > 50 {
		results = results[:50]
	}

	searchTime := time.Since(startTime)

	result := &maven.SearchResult{
		Query:      query,
		Results:    results,
		TotalCount: len(results),
		SearchTime: searchTime,
	}

	s.logger.Info("Search completed",
		logging.F("query", query),
		logging.F("resultCount", len(results)),
		logging.F("searchTime", searchTime),
	)

	return result, nil
}

// IndexArtifacts 아티팩트 인덱스 구축
func (s *searchServiceImpl) IndexArtifacts(ctx context.Context) error {
	s.logger.Info("Starting artifact indexing")

	startTime := time.Now()
	newIndex := &searchIndex{
		Artifacts: make(map[string]*maven.SearchArtifact),
		Version:   s.index.Version + 1,
	}

	// 각 미러에서 아티팩트 수집
	proxies := s.config.GetProxies()
	for _, proxy := range proxies {
		if err := s.indexMirror(ctx, proxy, newIndex); err != nil {
			s.logger.Warn("Failed to index mirror",
				logging.F("mirror", proxy.Name),
				logging.F("error", err),
			)
			continue
		}
	}

	// 인덱스 업데이트
	s.indexMutex.Lock()
	newIndex.LastUpdated = time.Now()
	s.index = newIndex
	s.indexMutex.Unlock()

	// 디스크에 저장
	if err := s.saveIndex(); err != nil {
		s.logger.Error("Failed to save search index", logging.F("error", err))
	}

	indexTime := time.Since(startTime)

	s.logger.Info("Artifact indexing completed",
		logging.F("totalArtifacts", len(newIndex.Artifacts)),
		logging.F("indexTime", indexTime),
	)

	return nil
}

// GetIndexStats 인덱스 통계 조회
func (s *searchServiceImpl) GetIndexStats(ctx context.Context) (*maven.IndexStats, error) {
	s.indexMutex.RLock()
	defer s.indexMutex.RUnlock()

	totalVersions := 0
	for _, artifact := range s.index.Artifacts {
		totalVersions += len(artifact.Versions)
	}

	// 인덱스 파일 크기 계산
	var indexSize int64
	if info, err := os.Stat(s.indexPath); err == nil {
		indexSize = info.Size()
	}

	stats := &maven.IndexStats{
		TotalArtifacts: len(s.index.Artifacts),
		TotalVersions:  totalVersions,
		LastUpdated:    s.index.LastUpdated,
		IndexSize:      indexSize,
	}

	return stats, nil
}

// UpdateIndex 특정 경로의 인덱스 업데이트
func (s *searchServiceImpl) UpdateIndex(ctx context.Context, path string) error {
	s.logger.Debug("Updating index for path", logging.F("path", path))

	// 경로에서 GroupID/ArtifactID 추출
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return nil // 그룹/아티팩트 레벨이 아님
	}

	groupID := strings.Join(parts[:len(parts)-1], ".")
	artifactID := parts[len(parts)-1]

	// 해당 아티팩트의 버전 정보 수집
	artifact, err := s.collectArtifactInfo(ctx, groupID, artifactID, path)
	if err != nil {
		return fmt.Errorf("failed to collect artifact info: %w", err)
	}

	// 인덱스 업데이트
	s.indexMutex.Lock()
	key := fmt.Sprintf("%s:%s", groupID, artifactID)
	s.index.Artifacts[key] = artifact
	s.indexMutex.Unlock()

	return nil
}

// calculateRelevanceScore 검색 관련도 점수 계산
func (s *searchServiceImpl) calculateRelevanceScore(artifact *maven.SearchArtifact, queryLower string) float64 {
	var score float64

	// GroupID 매치 (가중치: 0.3)
	if strings.Contains(strings.ToLower(artifact.GroupID), queryLower) {
		score += 0.3
		// 정확한 매치에 보너스
		if strings.ToLower(artifact.GroupID) == queryLower {
			score += 0.2
		}
	}

	// ArtifactID 매치 (가중치: 0.5)
	if strings.Contains(strings.ToLower(artifact.ArtifactID), queryLower) {
		score += 0.5
		// 정확한 매치에 보너스
		if strings.ToLower(artifact.ArtifactID) == queryLower {
			score += 0.3
		}
	}

	// Description 매치 (가중치: 0.2)
	if artifact.Description != "" && strings.Contains(strings.ToLower(artifact.Description), queryLower) {
		score += 0.2
	}

	// 인기도 보너스 (버전 수가 많을수록)
	versionBonus := float64(len(artifact.Versions)) / 100.0
	if versionBonus > 0.1 {
		versionBonus = 0.1
	}
	score += versionBonus

	return score
}

// indexMirror 특정 미러의 아티팩트 인덱싱
func (s *searchServiceImpl) indexMirror(ctx context.Context, proxy config.MavenProxyServer, index *searchIndex) error {
	s.logger.Debug("Indexing mirror", logging.F("mirror", proxy.Name))

	// 최상위 그룹들을 순회
	rootGroups, err := s.collector.CollectFromMirror(ctx, proxy, "")
	if err != nil {
		return fmt.Errorf("failed to collect root groups: %w", err)
	}

	// 각 그룹을 재귀적으로 탐색 (깊이 제한 있음)
	for _, group := range rootGroups {
		if group.Type == maven.TypeGroup || group.Type == maven.TypeDirectory {
			s.indexGroup(ctx, proxy, group.Name, "", 0, 3, index)
		}
	}

	return nil
}

// indexGroup 그룹 디렉토리 인덱싱 (재귀적)
func (s *searchServiceImpl) indexGroup(ctx context.Context, proxy config.MavenProxyServer, groupName, currentPath string, depth, maxDepth int, index *searchIndex) {
	if depth >= maxDepth {
		return
	}

	path := currentPath
	if path == "" {
		path = groupName
	} else {
		path = path + "/" + groupName
	}

	entries, err := s.collector.CollectFromMirror(ctx, proxy, path)
	if err != nil {
		s.logger.Debug("Failed to collect group entries",
			logging.F("group", groupName),
			logging.F("path", path),
			logging.F("error", err),
		)
		return
	}

	for _, entry := range entries {
		switch entry.Type {
		case maven.TypeGroup:
			// 하위 그룹 탐색
			s.indexGroup(ctx, proxy, entry.Name, path, depth+1, maxDepth, index)
		case maven.TypeArtifact:
			// 아티팩트 인덱싱
			s.indexArtifact(ctx, proxy, path, entry.Name, index)
		}
	}
}

// indexArtifact 아티팩트 인덱싱
func (s *searchServiceImpl) indexArtifact(ctx context.Context, proxy config.MavenProxyServer, groupPath, artifactName string, index *searchIndex) {
	groupID := strings.ReplaceAll(groupPath, "/", ".")
	key := fmt.Sprintf("%s:%s", groupID, artifactName)

	// 이미 인덱싱된 경우 스킵
	if _, exists := index.Artifacts[key]; exists {
		return
	}

	artifact, err := s.collectArtifactInfo(ctx, groupID, artifactName, groupPath+"/"+artifactName)
	if err != nil {
		s.logger.Debug("Failed to collect artifact info",
			logging.F("artifact", key),
			logging.F("error", err),
		)
		return
	}

	index.Artifacts[key] = artifact
}

// collectArtifactInfo 아티팩트 정보 수집
func (s *searchServiceImpl) collectArtifactInfo(ctx context.Context, groupID, artifactID, path string) (*maven.SearchArtifact, error) {
	// 버전 디렉토리 수집 - 첫 번째 사용 가능한 프록시 사용
	proxies := s.config.GetProxies()
	if len(proxies) == 0 {
		return nil, fmt.Errorf("no proxies configured")
	}

	entries, err := s.collector.CollectFromMirror(ctx, proxies[0], path)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.Type == maven.TypeVersion {
			versionName := strings.TrimSuffix(entry.Name, "/")
			versions = append(versions, versionName)
		}
	}

	// 버전 정렬 (최신 버전이 첫 번째)
	sort.Slice(versions, func(i, j int) bool {
		return s.compareVersions(versions[i], versions[j]) > 0
	})

	var latest string
	if len(versions) > 0 {
		latest = versions[0]
	}

	artifact := &maven.SearchArtifact{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Versions:   versions,
		Latest:     latest,
	}

	return artifact, nil
}

// compareVersions 버전 비교 (간단한 구현)
func (s *searchServiceImpl) compareVersions(v1, v2 string) int {
	// 간단한 문자열 비교 (실제 환경에서는 더 정교한 버전 비교 필요)
	if v1 == v2 {
		return 0
	}
	if v1 > v2 {
		return 1
	}
	return -1
}

// loadIndex 디스크에서 인덱스 로드
func (s *searchServiceImpl) loadIndex() error {
	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		return err
	}

	var index searchIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return err
	}

	s.indexMutex.Lock()
	s.index = &index
	s.indexMutex.Unlock()

	return nil
}

// saveIndex 인덱스를 디스크에 저장
func (s *searchServiceImpl) saveIndex() error {
	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(s.indexPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.indexPath, data, 0o644)
}
