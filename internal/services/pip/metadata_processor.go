package pip

import (
	"fmt"
	"regexp"
	"strings"

	"proxynd/internal/domain/pip"
	"proxynd/logging"
)

// metadataProcessorImpl PIP 메타데이터 처리 서비스 구현
type metadataProcessorImpl struct {
	config pip.ProxyConfig
	logger logging.Logger
}

// NewMetadataProcessor MetadataProcessor 생성자
func NewMetadataProcessor(config pip.ProxyConfig, logger logging.Logger) pip.MetadataProcessor {
	return &metadataProcessorImpl{
		config: config,
		logger: logger,
	}
}

// IsSimpleAPIRequest Simple API 요청인지 확인
func (p *metadataProcessorImpl) IsSimpleAPIRequest(packagePath string) bool {
	// /simple/ 경로이고 파일 확장자가 없거나 / 로 끝나는 경우
	if strings.HasPrefix(packagePath, "simple/") {
		fileName := strings.TrimPrefix(packagePath, "simple/")
		// 순수 패키지명이거나 / 로 끝나는 경우
		return !strings.Contains(fileName, ".") || strings.HasSuffix(fileName, "/")
	}

	return false
}

// IsPackageFileRequest 패키지 파일 요청인지 확인
func (p *metadataProcessorImpl) IsPackageFileRequest(packagePath string) bool {
	// .whl, .tar.gz, .zip, .egg 등의 팩지 파일 확장자 확인
	packageExtensions := []string{".whl", ".tar.gz", ".tar.bz2", ".zip", ".egg"}

	for _, ext := range packageExtensions {
		if strings.HasSuffix(packagePath, ext) {
			return true
		}
	}

	return false
}

// RewriteSimpleAPI Simple API HTML의 URL을 프록시 URL로 재작성
func (p *metadataProcessorImpl) RewriteSimpleAPI(data []byte, baseURL, packagePath string) []byte {
	if !p.IsSimpleAPIRequest(packagePath) {
		return data // Simple API가 아니면 그대로 반환
	}

	// HTML 내의 href 링크를 프록시 URL로 변경
	// 예: href="../../packages/source/r/requests/requests-2.28.1.tar.gz"
	// -> href="http://proxy/pip/packages/source/r/requests/requests-2.28.1.tar.gz"

	hrefRegex := regexp.MustCompile(`href=["']([^"']+)["']`)

	result := hrefRegex.ReplaceAllFunc(data, func(match []byte) []byte {
		// href 속성에서 URL 추출
		hrefMatch := regexp.MustCompile(`href=["']([^"']+)["']`).FindSubmatch(match)
		if len(hrefMatch) < 2 {
			return match
		}

		originalURL := string(hrefMatch[1])

		// 상대 경로인 경우 프록시 URL로 변경
		if strings.HasPrefix(originalURL, "../") || strings.HasPrefix(originalURL, "./") {
			// 상대 경로를 절대 경로로 변경
			cleanPath := strings.TrimPrefix(originalURL, "../")
			cleanPath = strings.TrimPrefix(cleanPath, "./")

			// 프록시 URL 구성
			proxyURL := fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(baseURL, "/"), p.config.GetPath(), cleanPath)

			return []byte(fmt.Sprintf(`href="%s"`, proxyURL))
		}

		// 절대 URL이면 그대로 유지
		if strings.HasPrefix(originalURL, "http://") || strings.HasPrefix(originalURL, "https://") {
			return match
		}

		// 상대 경로 처리
		proxyURL := fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(baseURL, "/"), p.config.GetPath(), originalURL)
		return []byte(fmt.Sprintf(`href="%s"`, proxyURL))
	})

	p.logger.Debug("Rewrote Simple API HTML",
		logging.F("packagePath", packagePath),
		logging.F("originalSize", len(data)),
		logging.F("resultSize", len(result)),
	)

	return result
}

// GetContentType 패키지 경로에서 Content-Type 결정
func (p *metadataProcessorImpl) GetContentType(packagePath, fileName string) string {
	// Simple API HTML 응답
	if p.IsSimpleAPIRequest(packagePath) {
		return "text/html; charset=utf-8"
	}

	// JSON API 응답
	if strings.Contains(packagePath, "/json") {
		return "application/json"
	}

	// 패키지 파일 타입별 Content-Type
	switch {
	case strings.HasSuffix(fileName, ".whl"):
		return "application/zip"
	case strings.HasSuffix(fileName, ".tar.gz"):
		return "application/x-gzip"
	case strings.HasSuffix(fileName, ".tar.bz2"):
		return "application/x-bzip2"
	case strings.HasSuffix(fileName, ".zip"):
		return "application/zip"
	case strings.HasSuffix(fileName, ".egg"):
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

// GetDisposition Content-Disposition 헤더 생성
func (p *metadataProcessorImpl) GetDisposition(packagePath, fileName string, isSimpleAPI bool) string {
	// Simple API나 JSON API 응답은 inline
	if isSimpleAPI || strings.Contains(packagePath, "/json") {
		return fmt.Sprintf("inline; filename=%s", fileName)
	}

	// 패키지 파일은 attachment
	if p.IsPackageFileRequest(packagePath) {
		return fmt.Sprintf("attachment; filename=%s", fileName)
	}

	// 기본값
	return fmt.Sprintf("inline; filename=%s", fileName)
}

// ExtractChecksumFromURL URL에서 체크섬 정보 추출
func (p *metadataProcessorImpl) ExtractChecksumFromURL(packageURL string) string {
	// PyPI URL에서 SHA256 해시 추출
	// 예: https://files.pythonhosted.org/packages/.../package.whl#sha256=abc123def456

	if strings.Contains(packageURL, "#sha256=") {
		parts := strings.Split(packageURL, "#sha256=")
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// 다른 해시 알고리즘 지원 추가 가능
	if strings.Contains(packageURL, "#md5=") {
		// MD5는 보안상 계산하지 않음
		p.logger.Warn("MD5 checksum detected, not supported for security reasons",
			logging.F("url", packageURL),
		)
	}

	return ""
}

// GetFileExtension 파일 경로에서 확장자 추출
func (p *metadataProcessorImpl) GetFileExtension(fileName string) string {
	if fileName == "" {
		return ""
	}

	// .tar.gz 처럼 이중 확장자 처리
	if strings.HasSuffix(fileName, ".tar.gz") {
		return ".tar.gz"
	}
	if strings.HasSuffix(fileName, ".tar.bz2") {
		return ".tar.bz2"
	}

	// 일반적인 확장자
	lastDotIndex := strings.LastIndex(fileName, ".")
	if lastDotIndex == -1 {
		return "" // 확장자 없음
	}

	return fileName[lastDotIndex:]
}
