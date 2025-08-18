package docker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"proxynd/internal/domain/docker"
	"proxynd/internal/logging"
)

const (
	mimeApplicationDockerManifestV2JSON = "application/vnd.docker.distribution.manifest.v2+json"
)

// manifestManagerImpl Docker 매니페스트 관리 구현
type manifestManagerImpl struct {
	config          docker.ProxyConfig
	registryManager docker.RegistryManager
	authManager     docker.AuthenticationManager
	cacheManager    docker.CacheManager
	logger          logging.Logger
}

// NewManifestManager ManifestManager 생성자
func NewManifestManager(
	config docker.ProxyConfig,
	registryManager docker.RegistryManager,
	authManager docker.AuthenticationManager,
	cacheManager docker.CacheManager,
	logger logging.Logger,
) docker.ManifestManager {
	return &manifestManagerImpl{
		config:          config,
		registryManager: registryManager,
		authManager:     authManager,
		cacheManager:    cacheManager,
		logger:          logger,
	}
}

// GetManifest 매니페스트 조회 (캐시 우선, 실패 시 레지스트리에서 다운로드)
func (m *manifestManagerImpl) GetManifest(ctx context.Context, repository, reference string) (*docker.ManifestResponse, error) { //nolint:lll
	m.logger.Debug("Getting manifest", logging.F("repository", repository), logging.F("reference", reference))

	// 캐시 확인
	if m.ShouldCacheManifest(repository, reference) {
		cacheKey := m.cacheManager.GenerateCacheKey(repository, reference, "manifest")
		if entry, err := m.cacheManager.Get(ctx, cacheKey); err == nil {
			if m.cacheManager.ValidateCacheEntry(ctx, entry) {
				m.logger.Debug("Manifest found in cache", logging.F("repository", repository), logging.F("reference", reference))

				// 저장된 헤더 로드
				headers := make(map[string]string)
				if headerData, err := m.cacheManager.LoadHeaders(ctx, cacheKey); err == nil {
					for key, values := range headerData {
						if len(values) > 0 {
							headers[key] = values[0]
						}
					}
				}

				// 캐시된 데이터로 응답 생성
				response := &docker.ManifestResponse{
					Data:            entry.Headers, // 매니페스트 데이터는 headers에 저장
					ContentType:     entry.ContentType,
					Digest:          entry.Digest,
					Headers:         headers,
					StatusCode:      http.StatusOK,
					FromCache:       true,
					SchemaVersion:   m.extractSchemaVersion(entry.Headers),
					MediaType:       entry.ContentType,
					IsMultiPlatform: m.IsMultiPlatform(entry.Headers),
				}

				return response, nil
			}
		}
	}

	// 캐시에 없거나 만료된 경우 레지스트리에서 가져오기
	return m.fetchManifestFromRegistry(ctx, repository, reference)
}

// fetchManifestFromRegistry 레지스트리에서 매니페스트 가져오기
func (m *manifestManagerImpl) fetchManifestFromRegistry(ctx context.Context, repository, reference string) (*docker.ManifestResponse, error) { //nolint:lll
	registry, err := m.registryManager.SelectRegistry(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to select registry: %w", err)
	}

	url := m.registryManager.BuildUpstreamURL(registry, fmt.Sprintf("/v2/%s/manifests/%s", repository, reference))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest request: %w", err)
	}

	// Docker 매니페스트 Accept 헤더 설정
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
	}, ","))

	// 인증 설정
	if auth, err := m.authManager.GetAuthToken(ctx, registry.URL, repository); err == nil {
		if auth.Type == "bearer" && auth.Token != "" { //nolint:goconst
			_ = m.authManager.SetBearerAuth(req, auth.Token)
		} else if auth.Type == "basic" && auth.Username != "" {
			_ = m.authManager.SetBasicAuth(req, auth.Username, auth.Password)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		_ = m.registryManager.MarkRegistryFailed(ctx, registry.URL, err)
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 인증 실패 시 챌린지 처리
	if resp.StatusCode == http.StatusUnauthorized {
		if _, err := m.authManager.ProcessAuthChallenge(ctx, resp.Header); err != nil {
			m.logger.Warn("Failed to process auth challenge", logging.F("error", err))
		}

		return &docker.ManifestResponse{
			StatusCode: resp.StatusCode,
			Headers:    make(map[string]string),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &docker.ManifestResponse{
			StatusCode: resp.StatusCode,
			Headers:    make(map[string]string),
		}, nil
	}

	// 매니페스트 데이터 읽기
	data := make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read manifest data: %w", err)
	}

	// 매니페스트 검증 및 다이제스트 계산
	digest, err := m.ExtractDigest(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract digest: %w", err)
	}

	// Docker-Content-Digest 헤더가 있는 경우 검증
	if expectedDigest := resp.Header.Get("Docker-Content-Digest"); expectedDigest != "" {
		if err := m.ValidateManifest(ctx, data, expectedDigest); err != nil {
			m.logger.Warn("Manifest digest validation failed", logging.F("expected", expectedDigest), logging.F("actual", digest)) //nolint:lll
		}
	}

	contentType := m.DetectManifestType(data)

	// 응답 생성
	response := &docker.ManifestResponse{
		Data:            data,
		ContentType:     contentType,
		Digest:          digest,
		Headers:         make(map[string]string),
		StatusCode:      http.StatusOK,
		FromCache:       false,
		RegistryUsed:    registry.Name,
		SchemaVersion:   m.extractSchemaVersion(data),
		MediaType:       contentType,
		IsMultiPlatform: m.IsMultiPlatform(data),
	}

	// 응답 헤더 처리
	if err := m.ProcessManifestHeaders(response, resp.Header); err != nil {
		m.logger.Warn("Failed to process manifest headers", logging.F("error", err))
	}

	// 캐시에 저장 (캐시 가능한 경우)
	if m.ShouldCacheManifest(repository, reference) {
		cacheKey := m.cacheManager.GenerateCacheKey(repository, reference, "manifest")
		entry := &docker.CacheEntry{
			Key:         cacheKey,
			ContentType: contentType,
			Size:        int64(len(data)),
			CreatedAt:   time.Now(),
			TTL:         m.config.GetCacheConfig().ManifestTTL,
			IsManifest:  true,
			Digest:      digest,
			Repository:  repository,
			Reference:   reference,
			Headers:     data, // 매니페스트 데이터
		}

		if err := m.cacheManager.Set(ctx, entry); err != nil {
			m.logger.Warn("Failed to cache manifest", logging.F("error", err))
		}

		// 헤더 정보도 별도 저장
		if err := m.cacheManager.SaveHeaders(ctx, cacheKey, resp.Header); err != nil {
			m.logger.Warn("Failed to save manifest headers", logging.F("error", err))
		}
	}

	return response, nil
}

// ValidateManifest 매니페스트 스키마 및 다이제스트 검증
func (m *manifestManagerImpl) ValidateManifest(ctx context.Context, data []byte, expectedDigest string) error {
	// 다이제스트 검증
	actualDigest, err := m.ExtractDigest(data)
	if err != nil {
		return fmt.Errorf("failed to calculate digest: %w", err)
	}

	if expectedDigest != "" && actualDigest != expectedDigest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", expectedDigest, actualDigest)
	}

	// JSON 스키마 검증
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// 스키마 버전 확인
	if schemaVersion, ok := manifest["schemaVersion"].(float64); ok {
		if schemaVersion != 1 && schemaVersion != 2 {
			return fmt.Errorf("unsupported schema version: %v", schemaVersion)
		}
	} else {
		return fmt.Errorf("missing schemaVersion field")
	}

	return nil
}

// DetectManifestType 매니페스트 타입 감지 (v1, v2, list)
func (m *manifestManagerImpl) DetectManifestType(data []byte) string {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return mimeApplicationDockerManifestV2JSON
	}

	// 매니페스트 리스트 (멀티 플랫폼)
	if _, exists := manifest["manifests"]; exists {
		return "application/vnd.docker.distribution.manifest.list.v2+json"
	}

	// 스키마 버전별 구분
	if schemaVersion, ok := manifest["schemaVersion"].(float64); ok {
		switch int(schemaVersion) {
		case 1:
			return "application/vnd.docker.distribution.manifest.v1+json"
		case 2:
			// OCI 이미지 매니페스트 확인
			if mediaType, ok := manifest["mediaType"].(string); ok {
				if strings.Contains(mediaType, "oci") {
					return "application/vnd.oci.image.manifest.v1+json"
				}
			}
			return mimeApplicationDockerManifestV2JSON
		}
	}

	return "application/vnd.docker.distribution.manifest.v2+json"
}

// ProcessManifestHeaders 매니페스트 응답 헤더 처리 및 설정
func (m *manifestManagerImpl) ProcessManifestHeaders(manifest *docker.ManifestResponse, headers http.Header) error {
	// Docker 관련 헤더 복사
	dockerHeaders := []string{
		"Docker-Content-Digest",
		"Docker-Distribution-Api-Version",
		"Etag",
		"Last-Modified",
	}

	for _, header := range dockerHeaders {
		if value := headers.Get(header); value != "" {
			manifest.Headers[header] = value
		}
	}

	// Content-Type 설정
	if contentType := headers.Get("Content-Type"); contentType != "" {
		manifest.ContentType = contentType
		manifest.Headers["Content-Type"] = contentType
	}

	return nil
}

// ShouldCacheManifest 매니페스트 캐시 여부 결정 (latest 태그 제외)
func (m *manifestManagerImpl) ShouldCacheManifest(repository, reference string) bool {
	// latest 태그는 캐시하지 않음 (변경 가능)
	if reference == "latest" {
		return false
	}

	// SHA256 다이제스트는 불변이므로 캐시 가능
	if strings.HasPrefix(reference, "sha256:") {
		return true
	}

	// 기타 태그는 캐시 (짧은 TTL 적용)
	return true
}

// ExtractDigest 매니페스트에서 SHA256 다이제스트 추출
func (m *manifestManagerImpl) ExtractDigest(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash), nil
}

// IsMultiPlatform 멀티 플랫폼 매니페스트 여부 확인
func (m *manifestManagerImpl) IsMultiPlatform(data []byte) bool {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false
	}

	// manifests 필드가 있으면 매니페스트 리스트 (멀티 플랫폼)
	if manifests, exists := manifest["manifests"]; exists {
		if manifestList, ok := manifests.([]interface{}); ok {
			return len(manifestList) > 1
		}
	}

	return false
}

// extractSchemaVersion 스키마 버전 추출
func (m *manifestManagerImpl) extractSchemaVersion(data []byte) int {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return 2 // 기본값
	}

	if schemaVersion, ok := manifest["schemaVersion"].(float64); ok {
		return int(schemaVersion)
	}

	return 2 // 기본값
}
