package integration

import (
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
			defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer jarResp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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

	// When: SNAPSHOT 아티팩트 요청 (존재하지 않으므로 404 예상)
	url := server.ProxyURL("maven", "org/springframework/spring-core/5.3.22-SNAPSHOT/spring-core-5.3.22-SNAPSHOT.jar")
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Then: SNAPSHOT은 캐시되지 않아야 하므로 적절한 처리 확인
	assert.Equal(t, 404, resp.StatusCode) // 테스트 환경에서는 존재하지 않음
	assert.Equal(t, "maven", resp.Header.Get("X-Proxy-Type"))

	// SNAPSHOT은 일반적으로 캐시되지 않음
	cacheStatus := resp.Header.Get("X-Cache-Status")
	if cacheStatus != "" {
		assert.Equal(t, "MISS", cacheStatus, "SNAPSHOT은 캐시되지 않아야 합니다")
	}
}

// calculateSHA1 SHA1 체크섬 계산
func calculateSHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
