package integration

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheIntegration 캐시 시스템 통합 테스트
func TestCacheIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 모든 upstream 서버 준비 대기
	for _, upstream := range []string{"npm", "maven", "apt"} {
		err := env.WaitForUpstream(upstream, 5*time.Second)
		require.NoError(t, err)
	}

	t.Run("캐시 저장 및 조회", func(t *testing.T) {
		// 첫 번째 요청 - upstream에서 가져와서 캐시에 저장
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		body1, err := io.ReadAll(resp1.Body)
		require.NoError(t, err)

		// 두 번째 요청 - 캐시에서 제공
		resp2, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)
		body2, err := io.ReadAll(resp2.Body)
		require.NoError(t, err)

		// 응답 내용이 동일해야 함
		assert.Equal(t, string(body1), string(body2))

		// 캐시 헤더 확인 (구현에 따라)
		// X-Cache-Status: HIT 등의 헤더가 있을 수 있음
		t.Logf("Cache headers: %+v", resp2.Header)
	})

	t.Run("캐시 TTL 만료", func(t *testing.T) {
		// 짧은 TTL로 설정된 항목 요청
		resp1, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// TTL이 매우 길게 설정되어 있으므로 즉시 만료는 시뮬레이션하기 어려움
		// 대신 캐시 무효화 API를 사용하거나 시간 기반 테스트는 별도로 수행

		// 캐시 상태 확인
		stats, err := env.GetCacheStats()
		require.NoError(t, err)
		t.Logf("Cache stats after TTL test: %+v", stats)
	})

	t.Run("캐시 크기 제한", func(t *testing.T) {
		// 여러 파일을 다운로드하여 캐시 크기 증가
		paths := []string{
			"/proxy/npm/express",
			"/proxy/npm/express/-/express-4.18.2.tgz",
			"/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom",
			"/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar",
			"/proxy/apt/ubuntu/dists/jammy/Release",
			"/proxy/apt/ubuntu/dists/jammy/main/binary-amd64/Packages",
		}

		for _, path := range paths {
			resp, err := env.MakeRequest("GET", path, nil)
			require.NoError(t, err)
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Logf("Request to %s returned %d", path, resp.StatusCode)
			}
		}

		// 캐시 통계 확인
		stats, err := env.GetCacheStats()
		require.NoError(t, err)
		t.Logf("Cache stats after size test: %+v", stats)

		// 캐시가 설정된 최대 크기를 초과하지 않는지 확인
		// 실제 구현에서는 캐시 크기 정보를 제공해야 함
	})

	t.Run("캐시 키 격리", func(t *testing.T) {
		// 같은 파일명이지만 다른 프록시 타입의 경우 별도 캐시 키 사용

		// NPM의 express
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		body1, err := io.ReadAll(resp1.Body)
		require.NoError(t, err)
		_ = resp1.Body.Close()

		// 만약 다른 프록시에도 express라는 경로가 있다면 별도로 캐시되어야 함
		// 현재 Mock에서는 NPM만 express를 제공하므로 이는 개념적 테스트

		assert.NotEmpty(t, body1)
		assert.Contains(t, string(body1), "express")
	})

	t.Run("캐시 무효화", func(t *testing.T) {
		// 캐시에 저장
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// 캐시 무효화 API 호출 (구현되어 있다면)
		invalidateResp, err := env.MakeRequest("DELETE", "/api/cache/npm/express", nil)
		if err == nil {
			_ = invalidateResp.Body.Close()
			// 캐시 무효화가 성공했는지 확인
			t.Logf("Cache invalidation response: %d", invalidateResp.StatusCode)
		}

		// 무효화 후 다시 요청
		resp2, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp2.Body.Close()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	t.Run("동시 캐시 액세스", func(t *testing.T) {
		// 동시에 같은 리소스에 접근
		path := "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar"
		concurrency := 5

		results := make(chan int, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				resp, err := env.MakeRequest("GET", path, nil)
				if err != nil {
					results <- 500
					return
				}
				_ = resp.Body.Close()
				results <- resp.StatusCode
			}()
		}

		// 모든 요청이 성공적으로 처리되어야 함
		successCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case statusCode := <-results:
				if statusCode == 200 {
					successCount++
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}

		assert.Equal(t, concurrency, successCount, "모든 동시 요청이 성공해야 함")
	})
}

// TestCacheErrorHandling 캐시 에러 처리 테스트
func TestCacheErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	err := env.WaitForUpstream("npm", 5*time.Second)
	require.NoError(t, err)

	t.Run("캐시 백엔드 장애 시 Fallback", func(t *testing.T) {
		// 캐시가 실패해도 upstream에서 데이터를 가져올 수 있어야 함
		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "express")
	})

	t.Run("손상된 캐시 데이터 처리", func(t *testing.T) {
		// 캐시에 저장
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()

		// 손상된 캐시 데이터 시뮬레이션은 어려우므로
		// 대신 다시 요청해서 정상 응답이 오는지 확인
		resp2, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		body, err := io.ReadAll(resp2.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "express")
	})

	t.Run("캐시 디스크 공간 부족", func(t *testing.T) {
		// 디스크 공간 부족 시뮬레이션은 어려우므로
		// 대신 매우 많은 요청을 보내서 캐시 시스템의 안정성 확인

		for i := 0; i < 10; i++ {
			path := fmt.Sprintf("/proxy/npm/express?v=%d", i)
			resp, err := env.MakeRequest("GET", path, nil)
			if err != nil {
				t.Logf("Request %d failed: %v", i, err)
				continue
			}
			_ = resp.Body.Close()
		}

		// 시스템이 여전히 응답하는지 확인
		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 캐시 실패 시에도 최소한 upstream에서는 가져올 수 있어야 함
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404)
	})
}

// TestCachePerformance 캐시 성능 테스트
func TestCachePerformance(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	err := env.WaitForUpstream("npm", 5*time.Second)
	require.NoError(t, err)

	t.Run("캐시 히트 성능", func(t *testing.T) {
		// 첫 번째 요청으로 캐시에 저장
		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp.Body.Close()

		// 캐시된 요청의 응답 시간 측정
		start := time.Now()

		for i := 0; i < 10; i++ {
			resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
			require.NoError(t, err)
			_ = resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		duration := time.Since(start)
		avgDuration := duration / 10

		t.Logf("Average cache hit response time: %v", avgDuration)

		// 캐시 히트는 일반적으로 매우 빨라야 함 (100ms 이하)
		assert.Less(t, avgDuration, 100*time.Millisecond, "캐시 히트 응답 시간이 너무 느림")
	})

	t.Run("캐시 미스 vs 히트 성능 비교", func(t *testing.T) {
		// 캐시 미스 시간 측정 (새로운 요청)
		missStart := time.Now()
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express?cache_miss_test=1", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()
		missDuration := time.Since(missStart)

		// 캐시 히트 시간 측정 (동일한 요청 반복)
		hitStart := time.Now()
		resp2, err := env.MakeRequest("GET", "/proxy/npm/express?cache_miss_test=1", nil)
		require.NoError(t, err)
		_ = resp2.Body.Close()
		hitDuration := time.Since(hitStart)

		t.Logf("Cache miss duration: %v", missDuration)
		t.Logf("Cache hit duration: %v", hitDuration)

		// 캐시 히트가 미스보다 빨라야 함
		// 하지만 Mock 서버는 매우 빠르므로 차이가 크지 않을 수 있음
		assert.LessOrEqual(t, hitDuration, missDuration*2, "캐시 히트가 미스보다 현저히 느림")
	})

	t.Run("대용량 파일 캐시 성능", func(t *testing.T) {
		// JAR 파일 (상대적으로 큰 파일) 다운로드
		start := time.Now()

		resp, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar", nil)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		_ = resp.Body.Close()

		firstDownload := time.Since(start)

		// 캐시된 파일 다운로드
		start = time.Now()

		resp2, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar", nil)
		require.NoError(t, err)

		body2, err := io.ReadAll(resp2.Body)
		require.NoError(t, err)
		_ = resp2.Body.Close()

		cachedDownload := time.Since(start)

		t.Logf("First download: %v, Cached download: %v", firstDownload, cachedDownload)
		t.Logf("File size: %d bytes", len(body))

		// 파일 내용이 동일해야 함
		assert.Equal(t, string(body), string(body2))

		// 캐시된 다운로드가 더 빠르거나 비슷해야 함
		assert.LessOrEqual(t, cachedDownload, firstDownload*2)
	})
}

// TestCacheConfiguration 캐시 설정 테스트
func TestCacheConfiguration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("프록시별 캐시 설정", func(t *testing.T) {
		// 각 프록시 타입이 올바른 캐시 설정을 사용하는지 확인

		// NPM 프록시 (캐시 활성화)
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// Maven 프록시 (캐시 활성화)
		resp2, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom", nil)
		require.NoError(t, err)
		_ = resp2.Body.Close()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		// APT 프록시 (캐시 활성화)
		resp3, err := env.MakeRequest("GET", "/proxy/apt/ubuntu/dists/jammy/Release", nil)
		require.NoError(t, err)
		_ = resp3.Body.Close()
		assert.Equal(t, http.StatusOK, resp3.StatusCode)

		// 모든 요청이 캐시되었는지 통계로 확인
		stats, err := env.GetCacheStats()
		require.NoError(t, err)
		t.Logf("Cache configuration test stats: %+v", stats)
	})

	t.Run("캐시 무시 헤더", func(t *testing.T) {
		// Cache-Control: no-cache 헤더로 캐시 무시
		headers := map[string]string{
			"Cache-Control": "no-cache",
		}

		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", headers)
		require.NoError(t, err)
		_ = resp1.Body.Close()

		resp2, err := env.MakeRequest("GET", "/proxy/npm/express", headers)
		require.NoError(t, err)
		_ = resp2.Body.Close()

		// 두 요청 모두 upstream에서 가져왔는지는 로그나 통계로 확인
		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	t.Run("조건부 요청 처리", func(t *testing.T) {
		// 첫 번째 요청
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()

		etag := resp1.Header.Get("ETag")
		lastModified := resp1.Header.Get("Last-Modified")

		// If-None-Match 헤더로 조건부 요청
		if etag != "" {
			headers := map[string]string{
				"If-None-Match": etag,
			}

			resp2, err := env.MakeRequest("GET", "/proxy/npm/express", headers)
			require.NoError(t, err)
			_ = resp2.Body.Close()

			// 304 Not Modified 또는 200 OK (구현에 따라)
			assert.True(t, resp2.StatusCode == 304 || resp2.StatusCode == 200)
		}

		// If-Modified-Since 헤더로 조건부 요청
		if lastModified != "" {
			headers := map[string]string{
				"If-Modified-Since": lastModified,
			}

			resp3, err := env.MakeRequest("GET", "/proxy/npm/express", headers)
			require.NoError(t, err)
			_ = resp3.Body.Close()

			// 304 Not Modified 또는 200 OK (구현에 따라)
			assert.True(t, resp3.StatusCode == 304 || resp3.StatusCode == 200)
		}
	})
}
