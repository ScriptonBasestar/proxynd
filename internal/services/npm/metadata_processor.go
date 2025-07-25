package npm

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"proxynd/internal/domain/npm"
)

// metadataProcessorImpl NPM 메타데이터 처리 서비스 구현
type metadataProcessorImpl struct{}

// NewMetadataProcessor MetadataProcessor 생성자
func NewMetadataProcessor() npm.MetadataProcessor {
	return &metadataProcessorImpl{}
}

// IsMetadataRequest 메타데이터 요청인지 확인
func (p *metadataProcessorImpl) IsMetadataRequest(packagePath string) bool {
	// 패키지 메타데이터는 패키지명만 있거나 @scope/package 형태
	parts := strings.Split(packagePath, "/")

	// .tgz, .tar.gz 등 패키지 파일이 아닌 경우
	if strings.Contains(packagePath, ".tgz") || strings.Contains(packagePath, ".tar.gz") {
		return false
	}

	// /-/ 를 포함하는 특수 경로가 아닌 경우
	if strings.Contains(packagePath, "/-/") {
		return false
	}

	// 일반 패키지명 또는 스코프 패키지명
	return len(parts) == 1 || (len(parts) == 2 && strings.HasPrefix(parts[0], "@"))
}

// RewriteMetadata 메타데이터의 URL을 프록시 URL로 재작성
func (p *metadataProcessorImpl) RewriteMetadata(data []byte, baseURL, packagePath string) []byte {
	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return data
	}

	// tarball URL 재작성
	if versions, ok := metadata["versions"].(map[string]interface{}); ok {
		for _, versionData := range versions {
			if version, ok := versionData.(map[string]interface{}); ok {
				if dist, ok := version["dist"].(map[string]interface{}); ok {
					if tarball, ok := dist["tarball"].(string); ok {
						// 원본 tarball URL을 프록시 URL로 변경
						proxyURL := baseURL + "/proxy/npm/" + p.extractPackagePath(tarball)
						dist["tarball"] = proxyURL
					}
				}
			}
		}
	}

	// 최신 버전의 tarball URL도 재작성
	if distTags, ok := metadata["dist-tags"].(map[string]interface{}); ok {
		if latest, ok := distTags["latest"].(string); ok {
			if versions, ok := metadata["versions"].(map[string]interface{}); ok {
				if latestVersion, ok := versions[latest].(map[string]interface{}); ok {
					if dist, ok := latestVersion["dist"].(map[string]interface{}); ok {
						if tarball, ok := dist["tarball"].(string); ok {
							proxyURL := baseURL + "/proxy/npm/" + p.extractPackagePath(tarball)
							dist["tarball"] = proxyURL
						}
					}
				}
			}
		}
	}

	rewritten, err := json.Marshal(metadata)
	if err != nil {
		// 에러 발생 시 원본 데이터 반환
		return data
	}
	return rewritten
}

// GetContentType 패키지 경로에서 Content-Type 결정
func (p *metadataProcessorImpl) GetContentType(packagePath string) string {
	// 메타데이터
	if p.IsMetadataRequest(packagePath) {
		return "application/json; charset=utf-8"
	}

	// 패키지 파일
	filename := filepath.Base(packagePath)
	switch {
	case strings.HasSuffix(filename, ".tgz"):
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".tar.gz"):
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".json"):
		return "application/json; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// GetDisposition Content-Disposition 헤더 생성
func (p *metadataProcessorImpl) GetDisposition(packagePath string, isMetadata bool) string {
	if isMetadata {
		return "inline"
	}

	// 파일명 추출 및 안전하게 처리
	filename := filepath.Base(packagePath)
	filename = strings.ReplaceAll(filename, "\"", "\\\"") // 쌍따옴표 이스케이프

	return fmt.Sprintf("attachment; filename=\"%s\"", filename)
}

// extractPackagePath tarball URL에서 패키지 경로 추출
func (p *metadataProcessorImpl) extractPackagePath(tarballURL string) string {
	// https://registry.npmjs.org/package/-/package-1.0.0.tgz 형태에서 패키지 경로 추출
	parts := strings.Split(tarballURL, "registry.npmjs.org/")
	if len(parts) > 1 {
		return parts[1]
	}

	// 다른 레지스트리 URL 패턴 처리
	if idx := strings.LastIndex(tarballURL, "/"); idx != -1 {
		// URL의 마지막 부분을 패키지 파일명으로 사용
		return tarballURL[idx+1:]
	}

	return tarballURL
}
