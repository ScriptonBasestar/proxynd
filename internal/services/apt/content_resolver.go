package apt

import (
	"fmt"
	"path/filepath"
	"strings"

	"proxynd/internal/domain/apt"
)

const (
	extBz2                   = ".bz2"
	extXz                    = ".xz"
	extLzma                  = ".lzma"
	aptPackageRelease        = "release"
	aptPackageInrelease      = "inrelease"
	aptPackagePackages       = "packages"
	aptPackageSources        = "sources"
	mimeTextPlainCharsetUTF8 = "text/plain; charset=utf-8"
	aptContentMetadata       = "metadata"
	aptContentArchive        = "archive"
)

// contentResolverImpl 콘텐츠 타입 결정 서비스 구현
type contentResolverImpl struct{}

// NewContentResolver ContentTypeResolver 생성자
func NewContentResolver() apt.ContentTypeResolver {
	return &contentResolverImpl{}
}

// GetContentType 패키지 경로에서 Content-Type 결정
func (r *contentResolverImpl) GetContentType(packagePath string) string {
	// 파일 확장자 기반 Content-Type 결정
	ext := strings.ToLower(filepath.Ext(packagePath))
	basename := strings.ToLower(filepath.Base(packagePath))

	switch ext {
	case extDeb:
		return mimeApplicationDebianBinaryPackage
	case ".udeb":
		return mimeApplicationDebianBinaryPackage
	case extGz:
		// 압축된 메타데이터 파일들
		if strings.Contains(basename, "packages") {
			return mimeApplicationGzip // Packages.gz
		}
		if strings.Contains(basename, "release") {
			return "application/gzip" //nolint:goconst // Release.gz
		}
		if strings.Contains(basename, "sources") {
			return "application/gzip" // Sources.gz
		}
		return "application/gzip"
	case extBz2:
		return "application/x-bzip2"
	case extXz:
		return "application/x-xz"
	case extLzma:
		return "application/x-lzma"
	default:
		// 확장자가 없는 메타데이터 파일들
		switch basename {
		case aptPackageRelease, aptPackageInrelease:
			return mimeTextPlainCharsetUTF8
		case aptPackagePackages:
			return mimeTextPlainCharsetUTF8
		case aptPackageSources:
			return mimeTextPlainCharsetUTF8
		case "contents":
			return mimeTextPlainCharsetUTF8
		default:
			// 경로 패턴으로 추가 판단
			if r.isDebianInstaller(packagePath) {
				return "application/vnd.debian.binary-package"
			}
			if r.isTranslationFile(packagePath) {
				return mimeTextPlainCharsetUTF8
			}
			return "application/octet-stream"
		}
	}
}

// ShouldInline 인라인으로 표시할지 결정
func (r *contentResolverImpl) ShouldInline(packagePath string) bool {
	ext := strings.ToLower(filepath.Ext(packagePath))
	basename := strings.ToLower(filepath.Base(packagePath))

	// 텍스트 메타데이터 파일들은 인라인으로 표시
	switch basename {
	case "release", "inrelease", "packages", "sources", "contents":
		return true
	}

	// 압축된 메타데이터도 인라인으로 표시
	switch ext {
	case ".gz", ".bz2", ".xz", ".lzma": //nolint:goconst
		if strings.Contains(basename, "packages") ||
			strings.Contains(basename, "release") ||
			strings.Contains(basename, "sources") {
			return true
		}
	}

	// 번역 파일들
	if r.isTranslationFile(packagePath) {
		return true
	}

	// 나머지는 첨부파일로 처리 (다운로드)
	return false
}

// GetDisposition Content-Disposition 헤더 생성
func (r *contentResolverImpl) GetDisposition(packagePath string) string {
	if r.ShouldInline(packagePath) {
		return "inline"
	}

	// 파일명 추출 및 안전하게 처리
	filename := filepath.Base(packagePath)
	filename = strings.ReplaceAll(filename, "\"", "\\\"") // 쌍따옴표 이스케이프

	return fmt.Sprintf("attachment; filename=\"%s\"", filename)
}

// isDebianInstaller 데비안 인스톨러 파일인지 확인
func (r *contentResolverImpl) isDebianInstaller(packagePath string) bool {
	// debian-installer 관련 패키지들
	return strings.Contains(packagePath, "debian-installer") ||
		strings.Contains(packagePath, "d-i") ||
		strings.Contains(packagePath, "installer")
}

// isTranslationFile 번역 파일인지 확인
func (r *contentResolverImpl) isTranslationFile(packagePath string) bool {
	basename := strings.ToLower(filepath.Base(packagePath))

	// Translation-{언어코드} 파일들
	if strings.HasPrefix(basename, "translation-") {
		return true
	}

	// i18n 디렉토리의 파일들
	if strings.Contains(packagePath, "/i18n/") {
		return true
	}

	return false
}

// GetFileCategory 파일 카테고리 분류 (통계/모니터링용)
func (r *contentResolverImpl) GetFileCategory(packagePath string) string {
	ext := strings.ToLower(filepath.Ext(packagePath))
	basename := strings.ToLower(filepath.Base(packagePath))

	// 패키지 파일
	if ext == ".deb" || ext == ".udeb" {
		return "package"
	}

	// 메타데이터
	if basename == "release" || basename == "inrelease" ||
		basename == "packages" || basename == "sources" ||
		strings.Contains(basename, "packages") ||
		strings.Contains(basename, "release") {
		return aptContentMetadata
	}

	// 압축 파일
	if ext == ".gz" || ext == ".bz2" || ext == ".xz" || ext == ".lzma" {
		return aptContentArchive
	}

	// 번역 파일
	if r.isTranslationFile(packagePath) {
		return "translation"
	}

	return "other"
}

// EstimateDownloadPriority 다운로드 우선순위 추정 (캐싱 전략용)
func (r *contentResolverImpl) EstimateDownloadPriority(packagePath string) int {
	category := r.GetFileCategory(packagePath)

	switch category {
	case aptContentMetadata:
		return 10 // 최고 우선순위 - 자주 접근됨
	case "package":
		return 5 // 중간 우선순위 - 크기가 클 수 있음
	case "translation":
		return 3 // 낮은 우선순위 - 선택적 접근
	case "archive":
		return 2 // 낮은 우선순위 - 압축 파일
	default:
		return 1 // 최저 우선순위
	}
}
