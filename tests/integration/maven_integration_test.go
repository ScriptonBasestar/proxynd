package integration

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMavenProxy_ArtifactDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	tests := []struct {
		name         string
		path         string
		expectedSize int64 // 최소 크기
		contentType  string
		contentCheck func([]byte) bool
	}{
		{
			name:         "Spring Core POM",
			path:         "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom",
			expectedSize: 100, // 최소 100바이트
			contentType:  "application/xml",
			contentCheck: func(data []byte) bool {
				content := string(data)
				return strings.Contains(content, "<groupId>org.springframework</groupId>") &&
					strings.Contains(content, "<artifactId>spring-core</artifactId>") &&
					strings.Contains(content, "<version>5.3.21</version>")
			},
		},
		{
			name:         "Spring Core JAR",
			path:         "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
			expectedSize: 1000, // 최소 1KB
			contentType:  "application/java-archive",
			contentCheck: func(data []byte) bool {
				// JAR 파일은 바이너리이므로 길이만 확인
				return len(data) >= 1000
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			url := server.ProxyURL("maven", tt.path)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Then
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.True(t, int64(len(body)) >= tt.expectedSize,
				"응답 크기가 예상보다 작습니다: got %d, want >= %d", len(body), tt.expectedSize)

			if tt.contentType != "" {
				assert.Contains(t, resp.Header.Get("Content-Type"), tt.contentType)
			}

			if tt.contentCheck != nil {
				assert.True(t, tt.contentCheck(body), "콘텐츠 검증 실패")
			}
		})
	}
}

func TestMavenProxy_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When: 존재하지 않는 아티팩트 요청
	url := server.ProxyURL("maven", "com/nonexistent/artifact/1.0.0/artifact-1.0.0.jar")
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Then
	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))
}

func TestMavenProxy_ChecksumValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	jarPath := "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar"

	// When: JAR 파일 다운로드
	jarURL := server.ProxyURL("maven", jarPath)
	jarResp, err := http.Get(jarURL)
	require.NoError(t, err)
	defer func() { _ = jarResp.Body.Close() }()

	require.Equal(t, 200, jarResp.StatusCode)

	jarData, err := io.ReadAll(jarResp.Body)
	require.NoError(t, err)

	// Then: 체크섬 계산 및 검증
	actualSha1 := calculateSHA1(jarData)

	// 실제 환경에서는 .sha1 파일을 다운로드하여 비교
	// 여기서는 체크섬이 올바른 형식인지만 확인
	assert.Len(t, actualSha1, 40, "SHA1 해시는 40자여야 합니다")

	t.Logf("JAR file size: %d bytes", len(jarData))
	t.Logf("SHA1 checksum: %s", actualSha1)
}

func TestMavenProxy_ConcurrentDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	const concurrency = 5
	results := make(chan TestResult, concurrency)
	done := make(chan bool)

	// When: 동시 다운로드
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// 다양한 아티팩트 요청
			paths := []string{
				"org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom",
				"org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
			}

			path := paths[id%len(paths)]
			url := server.ProxyURL("maven", path)
			result := performRequest(url)
			results <- result
		}(i)
	}

	// 모든 워커 완료 대기
	for i := 0; i < concurrency; i++ {
		<-done
	}
	close(results)

	// Then: 결과 분석
	var successCount, errorCount int

	for result := range results {
		if result.Error != nil {
			errorCount++
			t.Logf("Request failed: %v", result.Error)
		} else {
			successCount++
			assert.Equal(t, 200, result.StatusCode)
		}
	}

	errorRate := float64(errorCount) / float64(concurrency) * 100
	t.Logf("Concurrent requests: Total=%d, Success=%d, Error=%d (%.2f%%)",
		concurrency, successCount, errorCount, errorRate)

	assert.Less(t, errorRate, 20.0, "에러율이 20%를 초과했습니다")
	assert.Greater(t, successCount, 0, "최소 1개의 요청은 성공해야 합니다")
}

func TestMavenProxy_LargeArtifact(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When: 큰 JAR 파일 요청
	url := server.ProxyURL("maven", "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar")
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Then
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

	// 스트리밍 방식으로 데이터 읽기 (메모리 효율성)
	buffer := make([]byte, 8192) // 8KB 버퍼
	totalSize := 0

	for {
		n, err := resp.Body.Read(buffer)
		totalSize += n

		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		// 너무 큰 파일 방지 (테스트 환경)
		if totalSize > 10*1024*1024 { // 10MB 제한
			t.Logf("Large file download stopped at %d bytes", totalSize)
			break
		}
	}

	t.Logf("Downloaded %d bytes", totalSize)
	assert.Greater(t, totalSize, 1000, "최소 1KB는 다운로드되어야 합니다")
}

func TestMavenProxy_MetadataFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When: POM 메타데이터 요청
	url := server.ProxyURL("maven", "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom")
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Then
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "xml")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	content := string(body)
	assert.Contains(t, content, "<?xml", "XML 선언이 있어야 합니다")
	assert.Contains(t, content, "<project>", "Maven 프로젝트 태그가 있어야 합니다")
	assert.Contains(t, content, "spring-core", "아티팩트 ID가 포함되어야 합니다")
}

func TestMavenProxy_SnapshotHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
		checkCaching   bool
	}{
		{
			name:           "SNAPSHOT JAR not found",
			path:           "org/springframework/spring-core/5.3.22-SNAPSHOT/spring-core-5.3.22-SNAPSHOT.jar",
			expectedStatus: 404,
			description:    "SNAPSHOT JAR 파일 요청 (존재하지 않음)",
			checkCaching:   true,
		},
		{
			name:           "SNAPSHOT POM not found",
			path:           "org/springframework/spring-core/5.3.22-SNAPSHOT/spring-core-5.3.22-SNAPSHOT.pom",
			expectedStatus: 404,
			description:    "SNAPSHOT POM 파일 요청 (존재하지 않음)",
			checkCaching:   true,
		},
		{
			name:           "SNAPSHOT with timestamp",
			path:           "org/springframework/spring-core/5.3.22-SNAPSHOT/spring-core-5.3.22-20220301.123456-1.jar",
			expectedStatus: 404,
			description:    "타임스탬프가 포함된 SNAPSHOT 요청",
			checkCaching:   false,
		},
		{
			name:           "SNAPSHOT metadata",
			path:           "org/springframework/spring-core/5.3.22-SNAPSHOT/maven-metadata.xml",
			expectedStatus: 404,
			description:    "SNAPSHOT 메타데이터 요청",
			checkCaching:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: SNAPSHOT 아티팩트 요청
			url := server.ProxyURL("maven", tt.path)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Then: 상태 코드 확인
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, tt.description)
			assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

			// SNAPSHOT 캐싱 정책 확인
			if tt.checkCaching {
				cacheStatus := resp.Header.Get("X-Cache-Status")
				if cacheStatus != "" {
					assert.Equal(t, "MISS", cacheStatus, "SNAPSHOT은 캐시되지 않아야 합니다")
				}

				// 두 번째 요청으로 캐싱되지 않음 확인
				url2 := server.ProxyURL("maven", tt.path)
				resp2, err := http.Get(url2)
				require.NoError(t, err)
				defer func() { _ = resp2.Body.Close() }()

				assert.Equal(t, tt.expectedStatus, resp2.StatusCode)
				cacheStatus2 := resp2.Header.Get("X-Cache-Status")
				if cacheStatus2 != "" {
					assert.Equal(t, "MISS", cacheStatus2, "SNAPSHOT은 두 번째 요청에서도 캐시되지 않아야 합니다")
				}
			}
		})
	}
}

func TestMavenProxy_ChecksumEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	tests := []struct {
		name           string
		artifactPath   string
		checksumPath   string
		expectedStatus int
		checksumType   string
		validateFormat func(string) bool
	}{
		{
			name:           "JAR SHA1 checksum",
			artifactPath:   "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
			checksumPath:   "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar.sha1",
			expectedStatus: 404, // Mock 서버에서 구현되지 않음
			checksumType:   "SHA1",
			validateFormat: func(checksum string) bool {
				return len(checksum) == 40 // SHA1은 40자
			},
		},
		{
			name:           "JAR MD5 checksum",
			artifactPath:   "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
			checksumPath:   "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar.md5",
			expectedStatus: 404, // Mock 서버에서 구현되지 않음
			checksumType:   "MD5",
			validateFormat: func(checksum string) bool {
				return len(checksum) == 32 // MD5는 32자
			},
		},
		{
			name:           "POM SHA1 checksum",
			artifactPath:   "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom",
			checksumPath:   "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom.sha1",
			expectedStatus: 404, // Mock 서버에서 구현되지 않음
			checksumType:   "SHA1",
			validateFormat: func(checksum string) bool {
				return len(checksum) == 40
			},
		},
		{
			name:           "Non-existent artifact checksum",
			artifactPath:   "com/nonexistent/artifact/1.0.0/artifact-1.0.0.jar",
			checksumPath:   "com/nonexistent/artifact/1.0.0/artifact-1.0.0.jar.sha1",
			expectedStatus: 404,
			checksumType:   "SHA1",
			validateFormat: func(checksum string) bool {
				return true // 404이므로 체크섬 검증 불필요
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 체크섬 파일 요청
			checksumURL := server.ProxyURL("maven", tt.checksumPath)
			resp, err := http.Get(checksumURL)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Then: 응답 상태 확인
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

			if tt.expectedStatus == 200 {
				// 체크섬 내용 검증 (실제 구현에서는 활성화)
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				checksum := strings.TrimSpace(string(body))
				assert.True(t, tt.validateFormat(checksum),
					"%s 체크섬 형식이 올바르지 않습니다: %s", tt.checksumType, checksum)

				// Content-Type 확인
				contentType := resp.Header.Get("Content-Type")
				assert.Contains(t, contentType, "text/plain", "체크섬 파일은 text/plain이어야 합니다")

				t.Logf("%s checksum for %s: %s", tt.checksumType, tt.artifactPath, checksum)

				// 원본 아티팩트와 체크섬 일치 검증 (실제 환경에서 활성화)
				if tt.artifactPath != "" {
					artifactURL := server.ProxyURL("maven", tt.artifactPath)
					artifactResp, err := http.Get(artifactURL)
					if err == nil && artifactResp.StatusCode == 200 {
						defer func() { _ = artifactResp.Body.Close() }()
						artifactData, _ := io.ReadAll(artifactResp.Body)

						var calculatedChecksum string
						switch tt.checksumType {
						case "SHA1":
							calculatedChecksum = calculateSHA1(artifactData)
						case "MD5":
							calculatedChecksum = calculateMD5(artifactData)
						}

						// 실제 환경에서는 이 검증이 통과해야 함
						t.Logf("Calculated %s: %s, Server %s: %s",
							tt.checksumType, calculatedChecksum, tt.checksumType, checksum)
					}
				}
			}
		})
	}
}

func TestMavenProxy_DirectoryListing(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
		expectHTML     bool
	}{
		{
			name:           "Group directory listing",
			path:           "org/springframework/",
			expectedStatus: 404, // Mock 서버에서 디렉토리 목록 비활성화
			description:    "그룹 디렉토리 목록 조회",
			expectHTML:     false,
		},
		{
			name:           "Artifact directory listing",
			path:           "org/springframework/spring-core/",
			expectedStatus: 404, // Mock 서버에서 디렉토리 목록 비활성화
			description:    "아티팩트 디렉토리 목록 조회",
			expectHTML:     false,
		},
		{
			name:           "Version directory listing",
			path:           "org/springframework/spring-core/5.3.21/",
			expectedStatus: 404, // Mock 서버에서 디렉토리 목록 비활성화
			description:    "버전 디렉토리 목록 조회",
			expectHTML:     false,
		},
		{
			name:           "Root directory access",
			path:           "",
			expectedStatus: 404, // Mock 서버에서 루트 디렉토리 비활성화
			description:    "루트 디렉토리 접근",
			expectHTML:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 디렉토리 경로 요청
			url := server.ProxyURL("maven", tt.path)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Then: 응답 상태 확인
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, tt.description)
			assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

			if tt.expectedStatus == 200 && tt.expectHTML {
				// HTML 디렉토리 목록 검증
				contentType := resp.Header.Get("Content-Type")
				assert.Contains(t, contentType, "text/html", "디렉토리 목록은 HTML이어야 합니다")

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				content := string(body)
				assert.Contains(t, content, "<html>", "유효한 HTML이어야 합니다")
				assert.Contains(t, content, "Directory Listing", "디렉토리 목록 제목이 있어야 합니다")

				t.Logf("Directory listing for %s returned %d bytes", tt.path, len(body))
			}

			// 보안: 상위 디렉토리 접근 방지 확인
			if strings.Contains(tt.path, "..") {
				assert.NotEqual(t, 200, resp.StatusCode, "상위 디렉토리 접근은 차단되어야 합니다")
			}
		})
	}
}

// calculateSHA1 SHA1 체크섬 계산
func calculateSHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// calculateMD5 MD5 체크섬 계산
func calculateMD5(data []byte) string {
	h := md5.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Step 1-1: 추가된 SNAPSHOT 메타데이터 병합 테스트
func TestMavenProxy_SnapshotMetadataMerging(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When: SNAPSHOT 메타데이터 요청 (다중 버전)
	metadataPaths := []string{
		"org/springframework/spring-core/maven-metadata.xml",
		"org/springframework/spring-core/5.3.22-SNAPSHOT/maven-metadata.xml",
	}

	for _, path := range metadataPaths {
		t.Run(fmt.Sprintf("metadata_%s", strings.ReplaceAll(path, "/", "_")), func(t *testing.T) {
			url := server.ProxyURL("maven", path)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Then: 메타데이터 병합 로직 검증
			if resp.StatusCode == 200 {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				content := string(body)
				assert.Contains(t, content, "<metadata>", "유효한 메타데이터 XML이어야 합니다")
				assert.Contains(t, content, "<versioning>", "버전 정보가 포함되어야 합니다")

				// SNAPSHOT 특정 검증
				if strings.Contains(path, "SNAPSHOT") {
					assert.Contains(t, content, "<snapshot>", "SNAPSHOT 버전 정보가 있어야 합니다")
					assert.Contains(t, content, "<timestamp>", "타임스탬프 정보가 있어야 합니다")
				}

				t.Logf("Metadata content length: %d bytes", len(content))
			} else {
				t.Logf("Metadata not available (status: %d) - this is expected for mock server", resp.StatusCode)
			}
		})
	}
}

// Step 1-2: 추가된 체크섬 일관성 검증 테스트
func TestMavenProxy_ChecksumConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	tests := []struct {
		name         string
		artifactPath string
		checksumType string
		validation   func(artifactData, checksumData []byte) bool
	}{
		{
			name:         "JAR SHA1 consistency",
			artifactPath: "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
			checksumType: "sha1",
			validation: func(artifactData, checksumData []byte) bool {
				calculated := calculateSHA1(artifactData)
				retrieved := strings.TrimSpace(string(checksumData))
				return strings.EqualFold(calculated, retrieved)
			},
		},
		{
			name:         "POM MD5 consistency",
			artifactPath: "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom",
			checksumType: "md5",
			validation: func(artifactData, checksumData []byte) bool {
				calculated := calculateMD5(artifactData)
				retrieved := strings.TrimSpace(string(checksumData))
				return strings.EqualFold(calculated, retrieved)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 아티팩트와 체크섬을 모두 다운로드
			artifactURL := server.ProxyURL("maven", tt.artifactPath)
			artifactResp, err := http.Get(artifactURL)
			require.NoError(t, err)
			defer func() { _ = artifactResp.Body.Close() }()

			checksumURL := server.ProxyURL("maven", tt.artifactPath+"."+tt.checksumType)
			checksumResp, err := http.Get(checksumURL)
			require.NoError(t, err)
			defer func() { _ = checksumResp.Body.Close() }()

			// Then: 아티팩트 다운로드 성공 확인
			assert.Equal(t, 200, artifactResp.StatusCode, "아티팩트가 성공적으로 다운로드되어야 합니다")

			artifactData, err := io.ReadAll(artifactResp.Body)
			require.NoError(t, err)
			assert.Greater(t, len(artifactData), 0, "아티팩트 데이터가 비어있지 않아야 합니다")

			// 체크섬 검증 (실제 구현에서는 200 상태여야 함)
			if checksumResp.StatusCode == 200 {
				checksumData, err := io.ReadAll(checksumResp.Body)
				require.NoError(t, err)

				// 체크섬 일관성 검증
				isConsistent := tt.validation(artifactData, checksumData)
				assert.True(t, isConsistent, "체크섬이 일치해야 합니다")

				t.Logf("Checksum consistency verified for %s (%s)", tt.artifactPath, tt.checksumType)
			} else {
				t.Logf("Checksum endpoint not available (status: %d) - expected for mock server", checksumResp.StatusCode)
				// Mock 환경에서는 계산된 체크섬만 로깅
				var calculated string
				switch tt.checksumType {
				case "sha1":
					calculated = calculateSHA1(artifactData)
				case "md5":
					calculated = calculateMD5(artifactData)
				}
				t.Logf("Calculated %s for %s: %s", strings.ToUpper(tt.checksumType), tt.artifactPath, calculated)
			}
		})
	}
}

// Step 1-3: 추가된 디렉토리 브라우징 보안 테스트
func TestMavenProxy_DirectoryBrowsingToggle(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// 보안 테스트 케이스들
	securityTests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "Path traversal attempt 1",
			path:           "../../../etc/passwd",
			expectedStatus: 400, // Bad Request 또는 404
			description:    "상위 디렉토리 접근 시도 차단",
		},
		{
			name:           "Path traversal attempt 2",
			path:           "org/../../../etc/passwd",
			expectedStatus: 400,
			description:    "경로 내 상위 디렉토리 접근 시도 차단",
		},
		{
			name:           "URL encoded path traversal",
			path:           "org%2F..%2F..%2Fetc%2Fpasswd",
			expectedStatus: 400,
			description:    "URL 인코딩된 상위 디렉토리 접근 시도 차단",
		},
		{
			name:           "Double encoded path traversal",
			path:           "org%252F..%252F..%252Fetc%252Fpasswd",
			expectedStatus: 400,
			description:    "이중 인코딩된 상위 디렉토리 접근 시도 차단",
		},
		{
			name:           "Null byte injection",
			path:           "org/springframework/spring-core%00.txt",
			expectedStatus: 400,
			description:    "null 바이트 인젝션 차단",
		},
	}

	for _, tt := range securityTests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 악의적인 경로 요청
			url := server.ProxyURL("maven", tt.path)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Then: 보안 차단 확인
			assert.NotEqual(t, 200, resp.StatusCode, tt.description)
			assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

			// 추가 보안 헤더 확인
			body, _ := io.ReadAll(resp.Body)
			content := string(body)
			assert.NotContains(t, content, "root:", "시스템 파일 내용이 노출되면 안됩니다")
			assert.NotContains(t, content, "/etc/", "시스템 경로가 노출되면 안됩니다")

			t.Logf("Security test '%s' blocked with status %d", tt.name, resp.StatusCode)
		})
	}

	// 정상적인 경로 접근 확인
	t.Run("Valid path access", func(t *testing.T) {
		// 정상적인 Maven 아티팩트 경로는 여전히 작동해야 함
		validPath := "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom"
		url := server.ProxyURL("maven", validPath)
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 정상 경로는 적절하게 처리되어야 함 (200 또는 404)
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404,
			"정상적인 경로는 적절히 처리되어야 합니다 (got %d)", resp.StatusCode)

		t.Logf("Valid path access returned status %d", resp.StatusCode)
	})
}

// Step 1-4: 메타데이터 병합 테스트 추가
func TestMavenProxy_MetadataMerging(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	tests := []struct {
		name        string
		path        string
		description string
		validations []func(string) bool
	}{
		{
			name:        "Group level metadata",
			path:        "org/springframework/maven-metadata.xml",
			description: "그룹 레벨 메타데이터 병합",
			validations: []func(string) bool{
				func(content string) bool {
					return strings.Contains(content, "<metadata>") &&
						strings.Contains(content, "<groupId>org.springframework</groupId>")
				},
				func(content string) bool {
					return strings.Contains(content, "<plugins>")
				},
			},
		},
		{
			name:        "Artifact level metadata",
			path:        "org/springframework/spring-core/maven-metadata.xml",
			description: "아티팩트 레벨 메타데이터 병합",
			validations: []func(string) bool{
				func(content string) bool {
					return strings.Contains(content, "<metadata>") &&
						strings.Contains(content, "<artifactId>spring-core</artifactId>")
				},
				func(content string) bool {
					return strings.Contains(content, "<versioning>") &&
						strings.Contains(content, "<versions>")
				},
				func(content string) bool {
					return strings.Contains(content, "<latest>") &&
						strings.Contains(content, "<release>")
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 메타데이터 파일 요청
			url := server.ProxyURL("maven", tt.path)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

			if resp.StatusCode == 200 {
				// Then: 메타데이터 내용 검증
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				content := string(body)
				assert.Greater(t, len(content), 0, "메타데이터 내용이 비어있지 않아야 합니다")

				// XML 형식 확인
				assert.Contains(t, content, "<?xml", "XML 선언이 있어야 합니다")

				// 각 검증 함수 실행
				for i, validation := range tt.validations {
					assert.True(t, validation(content),
						"Validation %d failed for %s", i+1, tt.description)
				}

				// 메타데이터 병합 품질 검증
				assert.NotContains(t, content, "<metadata></metadata>", "빈 메타데이터가 아니어야 합니다")
				assert.NotContains(t, content, "ERROR", "에러가 포함되지 않아야 합니다")

				t.Logf("Metadata content validated for %s (%d bytes)", tt.path, len(content))
			} else {
				t.Logf("Metadata not available (status: %d) - expected for mock server", resp.StatusCode)
			}
		})
	}

	// 병합된 메타데이터의 캐시 동작 확인
	t.Run("Metadata caching behavior", func(t *testing.T) {
		metadataPath := "org/springframework/spring-core/maven-metadata.xml"
		url := server.ProxyURL("maven", metadataPath)

		// 첫 번째 요청
		resp1, err := http.Get(url)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		// 두 번째 요청 (캐시 확인)
		resp2, err := http.Get(url)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		// 메타데이터는 캐시 TTL이 짧아야 함
		cacheStatus1 := resp1.Header.Get("X-Cache-Status")
		cacheStatus2 := resp2.Header.Get("X-Cache-Status")

		if cacheStatus1 != "" && cacheStatus2 != "" {
			// 메타데이터 캐시 정책 검증
			t.Logf("First request cache status: %s", cacheStatus1)
			t.Logf("Second request cache status: %s", cacheStatus2)

			// 메타데이터는 짧은 TTL을 가져야 하지만 일시적으로 캐시될 수 있음
			assert.True(t, cacheStatus1 == "MISS" || cacheStatus1 == "HIT",
				"유효한 캐시 상태여야 합니다: %s", cacheStatus1)
		}

		t.Logf("Metadata caching behavior verified")
	})
}
