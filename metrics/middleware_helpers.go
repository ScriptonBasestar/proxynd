package metrics

import (
	"path"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Helper functions for enhanced metrics middleware

// isPackageDownload 패키지 다운로드 요청인지 확인
func isPackageDownload(c *fiber.Ctx) bool {
	// GET 메소드이고 응답이 성공(2xx)인 경우
	if c.Method() != "GET" || c.Response().StatusCode() < 200 || c.Response().StatusCode() >= 300 {
		return false
	}

	path := c.Path()
	
	// 각 레지스트리별 패키지 다운로드 패턴 확인
	switch {
	case strings.HasPrefix(path, "/proxy/npm/"):
		return isNpmPackageDownload(path)
	case strings.HasPrefix(path, "/proxy/maven/"):
		return isMavenPackageDownload(path)
	case strings.HasPrefix(path, "/proxy/pypi/"):
		return isPypiPackageDownload(path)
	case strings.HasPrefix(path, "/proxy/docker/"):
		return isDockerPackageDownload(path)
	case strings.HasPrefix(path, "/proxy/apt/"):
		return isAptPackageDownload(path)
	case strings.HasPrefix(path, "/proxy/yum/"):
		return isYumPackageDownload(path)
	case strings.HasPrefix(path, "/proxy/apk/"):
		return isApkPackageDownload(path)
	default:
		return false
	}
}

// isNpmPackageDownload NPM 패키지 다운로드 확인
func isNpmPackageDownload(requestPath string) bool {
	// NPM tarball 패턴: /-/[package]-[version].tgz
	// 또는 scoped package: /@scope/[package]/-/[package]-[version].tgz
	return strings.Contains(requestPath, "/-/") && strings.HasSuffix(requestPath, ".tgz")
}

// isMavenPackageDownload Maven 패키지 다운로드 확인
func isMavenPackageDownload(requestPath string) bool {
	// Maven artifact 패턴: .jar, .pom, .war, .zip 등
	ext := strings.ToLower(path.Ext(requestPath))
	mavenExtensions := []string{".jar", ".war", ".ear", ".zip", ".tar.gz", ".pom", ".aar"}
	
	for _, validExt := range mavenExtensions {
		if ext == validExt || strings.HasSuffix(requestPath, validExt) {
			return true
		}
	}
	return false
}

// isPypiPackageDownload PyPI 패키지 다운로드 확인
func isPypiPackageDownload(requestPath string) bool {
	// PyPI 패키지 패턴: .whl, .tar.gz, .zip
	ext := strings.ToLower(path.Ext(requestPath))
	pypiExtensions := []string{".whl", ".zip"}
	
	for _, validExt := range pypiExtensions {
		if ext == validExt {
			return true
		}
	}
	
	// .tar.gz 확인
	if strings.HasSuffix(strings.ToLower(requestPath), ".tar.gz") {
		return true
	}
	
	return false
}

// isDockerPackageDownload Docker 이미지 다운로드 확인
func isDockerPackageDownload(requestPath string) bool {
	// Docker blob 또는 manifest 다운로드
	return strings.Contains(requestPath, "/blobs/") || 
		   strings.Contains(requestPath, "/manifests/")
}

// isAptPackageDownload APT 패키지 다운로드 확인
func isAptPackageDownload(requestPath string) bool {
	// .deb 패키지 파일
	return strings.HasSuffix(strings.ToLower(requestPath), ".deb")
}

// isYumPackageDownload YUM 패키지 다운로드 확인
func isYumPackageDownload(requestPath string) bool {
	// .rpm 패키지 파일
	return strings.HasSuffix(strings.ToLower(requestPath), ".rpm")
}

// isApkPackageDownload APK 패키지 다운로드 확인
func isApkPackageDownload(requestPath string) bool {
	// .apk 패키지 파일
	return strings.HasSuffix(strings.ToLower(requestPath), ".apk")
}

// extractPackageInfo 패키지 정보 추출
func extractPackageInfo(c *fiber.Ctx, registryType string) (packageName, version, fileType string) {
	requestPath := c.Path()
	
	switch registryType {
	case "npm":
		return extractNpmPackageInfo(requestPath)
	case "maven":
		return extractMavenPackageInfo(requestPath)
	case "pypi":
		return extractPypiPackageInfo(requestPath)
	case "docker":
		return extractDockerPackageInfo(requestPath)
	case "apt":
		return extractAptPackageInfo(requestPath)
	case "yum":
		return extractYumPackageInfo(requestPath)
	case "apk":
		return extractApkPackageInfo(requestPath)
	default:
		return "unknown", "unknown", "unknown"
	}
}

// extractNpmPackageInfo NPM 패키지 정보 추출
func extractNpmPackageInfo(requestPath string) (packageName, version, fileType string) {
	// NPM tarball 패턴 파싱
	// 예: /proxy/npm/@scope/package/-/package-1.2.3.tgz
	// 또는: /proxy/npm/package/-/package-1.2.3.tgz
	
	parts := strings.Split(requestPath, "/-/")
	if len(parts) != 2 {
		return "unknown", "unknown", "tgz"
	}
	
	// 패키지명 추출
	pathParts := strings.Split(strings.TrimPrefix(parts[0], "/proxy/npm/"), "/")
	if len(pathParts) == 1 {
		// 일반 패키지
		packageName = pathParts[0]
	} else if len(pathParts) == 2 && strings.HasPrefix(pathParts[0], "@") {
		// 스코프 패키지
		packageName = pathParts[0] + "/" + pathParts[1]
	}
	
	// 버전 추출 (파일명에서)
	filename := parts[1]
	if strings.HasSuffix(filename, ".tgz") {
		filenameWithoutExt := strings.TrimSuffix(filename, ".tgz")
		// package-1.2.3에서 버전 추출
		lastDash := strings.LastIndex(filenameWithoutExt, "-")
		if lastDash != -1 {
			version = filenameWithoutExt[lastDash+1:]
		}
	}
	
	if packageName == "" {
		packageName = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	
	return packageName, version, "tgz"
}

// extractMavenPackageInfo Maven 패키지 정보 추출
func extractMavenPackageInfo(requestPath string) (packageName, version, fileType string) {
	// Maven 경로 패턴: /proxy/maven/group/artifact/version/artifact-version.extension
	pathWithoutPrefix := strings.TrimPrefix(requestPath, "/proxy/maven/")
	parts := strings.Split(pathWithoutPrefix, "/")
	
	if len(parts) >= 4 {
		// group을 제외하고 artifact와 version 추출
		artifact := parts[len(parts)-3]
		version = parts[len(parts)-2]
		filename := parts[len(parts)-1]
		
		packageName = artifact
		fileType = strings.TrimPrefix(path.Ext(filename), ".")
	}
	
	if packageName == "" {
		packageName = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	if fileType == "" {
		fileType = "unknown"
	}
	
	return packageName, version, fileType
}

// extractPypiPackageInfo PyPI 패키지 정보 추출
func extractPypiPackageInfo(requestPath string) (packageName, version, fileType string) {
	// PyPI 경로에서 패키지 정보 추출
	filename := path.Base(requestPath)
	fileType = "unknown"
	
	// 파일 확장자 결정
	if strings.HasSuffix(filename, ".whl") {
		fileType = "whl"
		// wheel 파일명 파싱: package-version-python-abi-platform.whl
		wheelRegex := regexp.MustCompile(`^(.+?)-(.+?)-.*\.whl$`)
		if matches := wheelRegex.FindStringSubmatch(filename); len(matches) == 3 {
			packageName = matches[1]
			version = matches[2]
		}
	} else if strings.HasSuffix(filename, ".tar.gz") {
		fileType = "sdist"
		// source distribution 파싱: package-version.tar.gz
		basename := strings.TrimSuffix(filename, ".tar.gz")
		parts := strings.Split(basename, "-")
		if len(parts) >= 2 {
			// 마지막 부분이 버전이라고 가정
			version = parts[len(parts)-1]
			packageName = strings.Join(parts[:len(parts)-1], "-")
		}
	} else if strings.HasSuffix(filename, ".zip") {
		fileType = "zip"
		basename := strings.TrimSuffix(filename, ".zip")
		parts := strings.Split(basename, "-")
		if len(parts) >= 2 {
			version = parts[len(parts)-1]
			packageName = strings.Join(parts[:len(parts)-1], "-")
		}
	}
	
	if packageName == "" {
		packageName = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	
	return packageName, version, fileType
}

// extractDockerPackageInfo Docker 이미지 정보 추출
func extractDockerPackageInfo(requestPath string) (packageName, version, fileType string) {
	// Docker registry 경로 파싱
	if strings.Contains(requestPath, "/manifests/") {
		// manifest 요청: /v2/[name]/manifests/[tag]
		parts := strings.Split(requestPath, "/manifests/")
		if len(parts) == 2 {
			namePartWithPrefix := parts[0]
			version = parts[1] // tag가 version 역할
			
			// 이름 부분에서 실제 이미지명 추출
			nameParts := strings.Split(namePartWithPrefix, "/")
			if len(nameParts) >= 3 { // /proxy/docker/v2/[name]
				packageName = strings.Join(nameParts[3:], "/")
			}
		}
		fileType = "manifest"
	} else if strings.Contains(requestPath, "/blobs/") {
		// blob 요청: /v2/[name]/blobs/[digest]
		parts := strings.Split(requestPath, "/blobs/")
		if len(parts) == 2 {
			namePartWithPrefix := parts[0]
			version = "unknown" // blob에는 명시적 버전이 없음
			
			nameParts := strings.Split(namePartWithPrefix, "/")
			if len(nameParts) >= 3 {
				packageName = strings.Join(nameParts[3:], "/")
			}
		}
		fileType = "blob"
	}
	
	if packageName == "" {
		packageName = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	
	return packageName, version, fileType
}

// extractAptPackageInfo APT 패키지 정보 추출
func extractAptPackageInfo(requestPath string) (packageName, version, fileType string) {
	filename := path.Base(requestPath)
	if strings.HasSuffix(filename, ".deb") {
		// .deb 파일명 파싱: package_version_architecture.deb
		basename := strings.TrimSuffix(filename, ".deb")
		parts := strings.Split(basename, "_")
		if len(parts) >= 2 {
			packageName = parts[0]
			version = parts[1]
		}
		fileType = "deb"
	}
	
	if packageName == "" {
		packageName = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	
	return packageName, version, fileType
}

// extractYumPackageInfo YUM 패키지 정보 추출
func extractYumPackageInfo(requestPath string) (packageName, version, fileType string) {
	filename := path.Base(requestPath)
	if strings.HasSuffix(filename, ".rpm") {
		// RPM 파일명 파싱: package-version-release.architecture.rpm
		basename := strings.TrimSuffix(filename, ".rpm")
		
		// 아키텍처 제거
		lastDot := strings.LastIndex(basename, ".")
		if lastDot != -1 {
			basename = basename[:lastDot]
		}
		
		// 패키지명과 버전-릴리스 분리
		parts := strings.Split(basename, "-")
		if len(parts) >= 3 {
			// 마지막 두 부분이 version-release라고 가정
			packageName = strings.Join(parts[:len(parts)-2], "-")
			version = parts[len(parts)-2] + "-" + parts[len(parts)-1]
		} else if len(parts) == 2 {
			packageName = parts[0]
			version = parts[1]
		}
		fileType = "rpm"
	}
	
	if packageName == "" {
		packageName = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	
	return packageName, version, fileType
}

// extractApkPackageInfo APK 패키지 정보 추출
func extractApkPackageInfo(requestPath string) (packageName, version, fileType string) {
	filename := path.Base(requestPath)
	if strings.HasSuffix(filename, ".apk") {
		// APK 파일명 파싱: package-version-release.apk
		basename := strings.TrimSuffix(filename, ".apk")
		parts := strings.Split(basename, "-")
		if len(parts) >= 2 {
			// 마지막 부분이 버전이라고 가정
			version = parts[len(parts)-1]
			packageName = strings.Join(parts[:len(parts)-1], "-")
		}
		fileType = "apk"
	}
	
	if packageName == "" {
		packageName = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	
	return packageName, version, fileType
}

// extractUserID 사용자 ID 추출
func extractUserID(c *fiber.Ctx) string {
	// 인증된 사용자 ID 추출
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	
	// JWT 토큰에서 사용자 ID 추출
	if user := c.Locals("user"); user != nil {
		if userMap, ok := user.(map[string]interface{}); ok {
			if id, exists := userMap["id"]; exists {
				if idStr, ok := id.(string); ok {
					return idStr
				}
			}
		}
	}
	
	// 익명 사용자의 경우 IP 기반으로 생성
	clientIP := c.IP()
	if clientIP != "" {
		return "anon_" + strings.ReplaceAll(clientIP, ".", "_")
	}
	
	return "unknown"
}

// extractUserType 사용자 타입 추출
func extractUserType(c *fiber.Ctx) string {
	// 인증 상태 확인
	if c.Locals("authenticated") != nil {
		if authenticated, ok := c.Locals("authenticated").(bool); ok && authenticated {
			return "authenticated"
		}
	}
	
	// User-Agent 기반으로 봇 감지
	userAgent := strings.ToLower(c.Get("User-Agent"))
	botKeywords := []string{"bot", "crawler", "spider", "scraper", "curl", "wget"}
	
	for _, keyword := range botKeywords {
		if strings.Contains(userAgent, keyword) {
			return "bot"
		}
	}
	
	return "anonymous"
}

// getRetryReason 재시도 사유 추출
func getRetryReason(c *fiber.Ctx) string {
	if reason := c.Locals("retry_reason"); reason != nil {
		if reasonStr, ok := reason.(string); ok {
			return reasonStr
		}
	}
	
	// 응답 코드 기반으로 사유 추측
	statusCode := c.Response().StatusCode()
	switch {
	case statusCode >= 500 && statusCode < 600:
		return "server_error"
	case statusCode == 429:
		return "rate_limit"
	case statusCode == 408:
		return "timeout"
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	default:
		return "unknown"
	}
}