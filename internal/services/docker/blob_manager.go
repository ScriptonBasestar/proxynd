package docker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"proxynd/internal/domain/docker"
	"proxynd/logging"
)

// blobManagerImpl Docker blob 관리 구현
type blobManagerImpl struct {
	config          docker.ProxyConfig
	registryManager docker.RegistryManager
	authManager     docker.AuthenticationManager
	cacheManager    docker.CacheManager
	logger          logging.Logger
}

// NewBlobManager BlobManager 생성자
func NewBlobManager(
	config docker.ProxyConfig,
	registryManager docker.RegistryManager,
	authManager docker.AuthenticationManager,
	cacheManager docker.CacheManager,
	logger logging.Logger,
) docker.BlobManager {
	return &blobManagerImpl{
		config:          config,
		registryManager: registryManager,
		authManager:     authManager,
		cacheManager:    cacheManager,
		logger:          logger,
	}
}

// GetBlob blob 데이터 조회 (레이어, 설정 파일 등)
func (b *blobManagerImpl) GetBlob(ctx context.Context, repository, digest string) (*docker.ManifestResponse, error) {
	b.logger.Debug("Getting blob", logging.F("repository", repository), logging.F("digest", digest))

	// SHA256 다이제스트 형식 검증
	if !strings.HasPrefix(digest, "sha256:") {
		return nil, fmt.Errorf("invalid digest format: %s", digest)
	}

	// 캐시 확인 (blob은 불변이므로 적극적으로 캐시)
	if b.ShouldCacheBlob(repository, digest) {
		cacheKey := b.cacheManager.GenerateCacheKey(repository, digest, "blob")
		if entry, err := b.cacheManager.Get(ctx, cacheKey); err == nil {
			if b.cacheManager.ValidateCacheEntry(ctx, entry) {
				b.logger.Debug("Blob found in cache", logging.F("repository", repository), logging.F("digest", digest))

				// 다이제스트 검증
				if err := b.ValidateBlobDigest(ctx, entry.Headers, digest); err != nil {
					b.logger.Warn("Cached blob digest validation failed", logging.F("error", err))
					// 캐시 무효화
					b.cacheManager.Delete(ctx, cacheKey)
				} else {
					return &docker.ManifestResponse{
						Data:        entry.Headers, // blob 데이터는 headers에 저장
						ContentType: entry.ContentType,
						StatusCode:  http.StatusOK,
						Headers: map[string]string{
							"Docker-Content-Digest": digest,
							"Content-Type":          entry.ContentType,
							"Content-Length":        fmt.Sprintf("%d", entry.Size),
						},
						FromCache: true,
						Digest:    digest,
					}, nil
				}
			}
		}
	}

	// 캐시에 없거나 무효한 경우 레지스트리에서 가져오기
	return b.fetchBlobFromRegistry(ctx, repository, digest)
}

// fetchBlobFromRegistry 레지스트리에서 blob 가져오기
func (b *blobManagerImpl) fetchBlobFromRegistry(ctx context.Context, repository, digest string) (*docker.ManifestResponse, error) {
	registry, err := b.registryManager.SelectRegistry(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to select registry: %w", err)
	}

	url := b.registryManager.BuildUpstreamURL(registry, fmt.Sprintf("/v2/%s/blobs/%s", repository, digest))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob request: %w", err)
	}

	// 인증 설정
	if auth, err := b.authManager.GetAuthToken(ctx, registry.URL, repository); err == nil {
		if auth.Type == "bearer" && auth.Token != "" {
			b.authManager.SetBearerAuth(req, auth.Token)
		} else if auth.Type == "basic" && auth.Username != "" {
			b.authManager.SetBasicAuth(req, auth.Username, auth.Password)
		}
	}

	client := &http.Client{Timeout: 60 * time.Second} // blob은 큰 파일일 수 있으므로 타임아웃 연장
	resp, err := client.Do(req)
	if err != nil {
		b.registryManager.MarkRegistryFailed(ctx, registry.URL, err)
		return nil, fmt.Errorf("failed to fetch blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &docker.ManifestResponse{
			StatusCode: resp.StatusCode,
			Headers:    make(map[string]string),
		}, nil
	}

	// blob 데이터 읽기
	data := make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read blob data: %w", err)
	}

	// 다이제스트 검증
	if err := b.ValidateBlobDigest(ctx, data, digest); err != nil {
		return nil, fmt.Errorf("blob digest validation failed: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 응답 생성
	response := &docker.ManifestResponse{
		Data:         data,
		ContentType:  contentType,
		StatusCode:   http.StatusOK,
		Headers:      make(map[string]string),
		FromCache:    false,
		RegistryUsed: registry.Name,
		Digest:       digest,
	}

	// 응답 헤더 처리
	if err := b.ProcessBlobHeaders(response, resp.Header); err != nil {
		b.logger.Warn("Failed to process blob headers", logging.F("error", err))
	}

	// 캐시에 저장 (blob은 불변이므로 긴 TTL)
	if b.ShouldCacheBlob(repository, digest) {
		cacheKey := b.cacheManager.GenerateCacheKey(repository, digest, "blob")
		entry := &docker.CacheEntry{
			Key:         cacheKey,
			ContentType: contentType,
			Size:        b.CalculateBlobSize(data),
			CreatedAt:   time.Now(),
			TTL:         b.config.GetCacheConfig().BlobTTL,
			IsBlob:      true,
			Digest:      digest,
			Repository:  repository,
			Headers:     data, // blob 데이터
		}

		if err := b.cacheManager.Set(ctx, entry); err != nil {
			b.logger.Warn("Failed to cache blob", logging.F("error", err))
		}
	}

	return response, nil
}

// ValidateBlobDigest blob 다이제스트 검증
func (b *blobManagerImpl) ValidateBlobDigest(ctx context.Context, data []byte, expectedDigest string) error {
	if !strings.HasPrefix(expectedDigest, "sha256:") {
		return fmt.Errorf("unsupported digest algorithm")
	}

	// SHA256 해시 계산
	hash := sha256.Sum256(data)
	actualDigest := fmt.Sprintf("sha256:%x", hash)

	if actualDigest != expectedDigest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", expectedDigest, actualDigest)
	}

	return nil
}

// GetBlobInfo blob 메타데이터 정보 반환
func (b *blobManagerImpl) GetBlobInfo(ctx context.Context, repository, digest string) (*docker.BlobReference, error) {
	b.logger.Debug("Getting blob info", logging.F("repository", repository), logging.F("digest", digest))

	// 캐시에서 blob 정보 확인
	cacheKey := b.cacheManager.GenerateCacheKey(repository, digest, "blob")
	if entry, err := b.cacheManager.Get(ctx, cacheKey); err == nil {
		return &docker.BlobReference{
			Digest:      digest,
			Size:        entry.Size,
			MediaType:   entry.ContentType,
			ContentType: entry.ContentType,
			Repository:  repository,
			IsConfig:    strings.Contains(entry.ContentType, "config"),
			IsLayer:     strings.Contains(entry.ContentType, "layer") || entry.ContentType == "application/octet-stream",
		}, nil
	}

	// 캐시에 없는 경우 HEAD 요청으로 메타데이터만 가져오기
	registry, err := b.registryManager.SelectRegistry(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to select registry: %w", err)
	}

	url := b.registryManager.BuildUpstreamURL(registry, fmt.Sprintf("/v2/%s/blobs/%s", repository, digest))

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob info request: %w", err)
	}

	// 인증 설정
	if auth, err := b.authManager.GetAuthToken(ctx, registry.URL, repository); err == nil {
		if auth.Type == "bearer" && auth.Token != "" {
			b.authManager.SetBearerAuth(req, auth.Token)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blob not found: %s", digest)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &docker.BlobReference{
		Digest:      digest,
		Size:        resp.ContentLength,
		MediaType:   contentType,
		ContentType: contentType,
		Repository:  repository,
		IsConfig:    strings.Contains(contentType, "config"),
		IsLayer:     strings.Contains(contentType, "layer") || contentType == "application/octet-stream",
	}, nil
}

// ShouldCacheBlob blob 캐시 여부 결정 (일반적으로 항상 캐시)
func (b *blobManagerImpl) ShouldCacheBlob(repository, digest string) bool {
	// blob은 불변이므로 항상 캐시
	return strings.HasPrefix(digest, "sha256:")
}

// ProcessBlobHeaders blob 응답 헤더 처리
func (b *blobManagerImpl) ProcessBlobHeaders(response *docker.ManifestResponse, headers http.Header) error {
	// Docker 관련 헤더 복사
	dockerHeaders := []string{
		"Docker-Content-Digest",
		"Content-Length",
		"Content-Type",
		"Accept-Ranges",
	}

	for _, header := range dockerHeaders {
		if value := headers.Get(header); value != "" {
			response.Headers[header] = value
		}
	}

	// Docker-Content-Digest 설정 (없는 경우)
	if response.Headers["Docker-Content-Digest"] == "" && response.Digest != "" {
		response.Headers["Docker-Content-Digest"] = response.Digest
	}

	return nil
}

// CalculateBlobSize blob 크기 계산
func (b *blobManagerImpl) CalculateBlobSize(data []byte) int64 {
	return int64(len(data))
}

// IsCompressed blob 압축 여부 확인
func (b *blobManagerImpl) IsCompressed(mediaType string) bool {
	compressedTypes := []string{
		"application/vnd.docker.image.rootfs.diff.tar.gzip",
		"application/vnd.oci.image.layer.v1.tar+gzip",
		"application/vnd.docker.image.rootfs.diff.tar",
		"application/vnd.oci.image.layer.v1.tar",
	}

	for _, compressedType := range compressedTypes {
		if strings.Contains(mediaType, compressedType) {
			return strings.Contains(compressedType, "gzip") || strings.Contains(compressedType, "tar")
		}
	}

	return false
}
