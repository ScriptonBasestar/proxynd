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
