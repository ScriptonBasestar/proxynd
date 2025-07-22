//nolint:bodyclose // 벤치마크 테스트 파일에서는 응답 본문 닫기 무시
package proxy

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
)

// BenchmarkAPTHandler_CacheKeyGeneration 캐시 키 생성 성능 측정
func BenchmarkAPTHandler_CacheKeyGeneration(b *testing.B) {
	handler := NewAPTHandlerV2()
	app := fiber.New()

	// 다양한 경로로 벤치마크
	paths := []string{
		"/proxy/apt/ubuntu/dists/focal/Release",
		"/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages",
		"/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.2434-3+deb11u1_amd64.deb",
		"/proxy/apt/debian/dists/bullseye/contrib/source/Sources.gz",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		app.Get("/proxy/apt/:osType/*", func(c *fiber.Ctx) error {
			_ = handler.GenerateCacheKey(c)
			return nil
		})

		req := httptest.NewRequest("GET", path, nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAPTHandler_CachePolicyDecision 캐시 정책 결정 성능 측정
func BenchmarkAPTHandler_CachePolicyDecision(b *testing.B) {
	handler := NewAPTHandlerV2()
	app := fiber.New()

	testCases := []struct {
		path       string
		statusCode int
	}{
		{"/proxy/apt/ubuntu/dists/focal/Release", 200},
		{"/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.deb", 200},
		{"/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages", 200},
		{"/proxy/apt/ubuntu/not-found", 404},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc := testCases[i%len(testCases)]
		app.Get("/proxy/apt/:osType/*", func(c *fiber.Ctx) error {
			_ = handler.ShouldCache(c, tc.statusCode)
			return nil
		})

		req := httptest.NewRequest("GET", tc.path, nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAPTHandler_MetadataFileCheck 메타데이터 파일 판별 성능 측정
func BenchmarkAPTHandler_MetadataFileCheck(b *testing.B) {
	handler := NewAPTHandlerV2()

	paths := []string{
		"dists/focal/Release",
		"dists/focal/Release.gpg",
		"dists/focal/main/binary-amd64/Packages",
		"dists/focal/main/binary-amd64/Packages.gz",
		"pool/main/v/vim/vim_8.2.deb",
		"pool/universe/f/firefox/firefox_92.0+build3-0ubuntu1_amd64.deb",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.isMetadataFile(paths[i%len(paths)])
	}
}

// BenchmarkAPTHandler_RequestTransform APT 요청 변환 성능 측정
func BenchmarkAPTHandler_RequestTransform(b *testing.B) {
	handler := NewAPTHandlerV2()
	app := fiber.New()

	paths := []string{
		"/proxy/apt/ubuntu/dists/focal/Release",
		"/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.deb",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		app.Get("/proxy/apt/:osType/*", func(c *fiber.Ctx) error {
			agent := fiber.AcquireAgent()
			defer fiber.ReleaseAgent(agent)

			_ = handler.TransformRequest(c, agent)
			return nil
		})

		req := httptest.NewRequest("GET", path, nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAPTHandler_URLBuildingParallel 병렬 URL 구성 성능 측정
func BenchmarkAPTHandler_URLBuildingParallel(b *testing.B) {
	config := &configs.AptProxyConfig{
		Proxies: map[string][]configs.AptProxy{
			"ubuntu": {
				{URL: "http://mirror.ubuntu.com/ubuntu"},
			},
			"debian": {
				{URL: "http://deb.debian.org/debian"},
			},
		},
	}

	app := fiber.New()

	b.RunParallel(func(pb *testing.PB) {
		paths := []string{
			"/proxy/apt/ubuntu/dists/focal/Release",
			"/proxy/apt/debian/pool/main/v/vim/vim_8.2.deb",
		}

		i := 0
		for pb.Next() {
			path := paths[i%len(paths)]
			app.Get("/proxy/apt/:osType/*", func(c *fiber.Ctx) error {
				// BuildUpstreamURL 로직을 인라인으로 구현
				osType := c.Params("osType", "ubuntu")
				packagePath := c.Params("*")

				proxies, exists := config.Proxies[osType]
				if exists && len(proxies) > 0 && proxies[0].URL != "" {
					baseURL := proxies[0].URL
					if baseURL[len(baseURL)-1] == '/' {
						baseURL = baseURL[:len(baseURL)-1]
					}
					if packagePath[0] == '/' {
						packagePath = packagePath[1:]
					}
					_ = baseURL + "/" + packagePath
				}

				return nil
			})

			req := httptest.NewRequest("GET", path, nil)
			_, _ = app.Test(req, -1)
			i++
		}
	})
}

// BenchmarkAPTHandler_ResponseTransform APT 응답 변환 성능 측정
func BenchmarkAPTHandler_ResponseTransform(b *testing.B) {
	handler := NewAPTHandlerV2()
	app := fiber.New()

	responseData := []byte(`Origin: Ubuntu
Suite: focal
Codename: focal
Version: 20.04
Architectures: amd64 arm64 armhf i386`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Get("/proxy/apt/:osType/*", func(c *fiber.Ctx) error {
			_, _ = handler.TransformResponse(responseData, c)
			return nil
		})

		req := httptest.NewRequest("GET", "/proxy/apt/ubuntu/dists/focal/Release", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAPTHandler_CacheTTLCalculation 캐시 TTL 계산 성능 측정
func BenchmarkAPTHandler_CacheTTLCalculation(b *testing.B) {
	handler := NewAPTHandlerV2()
	app := fiber.New()

	paths := []string{
		"/proxy/apt/ubuntu/dists/focal/Release",         // 10분
		"/proxy/apt/ubuntu/dists/focal/Packages",        // 30분
		"/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.deb", // 7일
		"/proxy/apt/ubuntu/other/file",                  // 1시간
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		app.Get("/proxy/apt/:osType/*", func(c *fiber.Ctx) error {
			_ = handler.GetCacheTTL(c)
			return nil
		})

		req := httptest.NewRequest("GET", path, nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAPTHandler_ErrorHandling 에러 처리 성능 측정
func BenchmarkAPTHandler_ErrorHandling(b *testing.B) {
	handler := NewAPTHandlerV2()

	errors := []error{
		fmt.Errorf("미러가 설정되지 않았습니다"),
		fmt.Errorf("설정 로드 실패: config not found"),
		fmt.Errorf("invalid path provided"),
		fmt.Errorf("unknown error"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.HandleError(errors[i%len(errors)], nil)
	}
}

// BenchmarkAPTHandler_MemoryUsage 메모리 할당 벤치마크
func BenchmarkAPTHandler_MemoryUsage(b *testing.B) {
	handler := NewAPTHandlerV2()
	app := fiber.New()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app.Get("/proxy/apt/:osType/*", func(c *fiber.Ctx) error {
			key := handler.GenerateCacheKey(c)
			_ = key // 사용
			return nil
		})

		req := httptest.NewRequest("GET", "/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.2434-3+deb11u1_amd64.deb", nil)
		resp, _ := app.Test(req, -1)
		_ = resp.Body.Close()
	}
}
