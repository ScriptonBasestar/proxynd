package metrics

import (
	"strconv"
	"strings"
	"time"
	
	"github.com/gofiber/fiber/v2"
)

// PrometheusMiddleware Prometheus 메트릭 수집 미들웨어
func PrometheusMiddleware() fiber.Handler {
	metrics := GetMetrics()
	
	return func(c *fiber.Ctx) error {
		// /metrics 엔드포인트는 제외
		if c.Path() == "/metrics" {
			return c.Next()
		}
		
		// 활성 요청 수 증가
		metrics.HTTPActiveRequests.Inc()
		defer metrics.HTTPActiveRequests.Dec()
		
		// 시작 시간 기록
		start := time.Now()
		
		// 요청 크기 기록
		if c.Request().Header.ContentLength() > 0 {
			size := float64(c.Request().Header.ContentLength())
			registryType := extractRegistryType(c)
			metrics.HTTPRequestSize.WithLabelValues(
				c.Method(),
				normalizePath(c.Path()),
				registryType,
			).Observe(size)
		}
		
		// 다음 핸들러 실행
		err := c.Next()
		
		// 응답 처리
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())
		path := normalizePath(c.Path())
		registryType := extractRegistryType(c)
		
		// HTTP 메트릭 기록
		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Method(),
			path,
			status,
			registryType,
		).Inc()
		
		metrics.HTTPRequestDuration.WithLabelValues(
			c.Method(),
			path,
			status,
			registryType,
		).Observe(duration)
		
		// 응답 크기 기록
		if size := len(c.Response().Body()); size > 0 {
			metrics.HTTPResponseSize.WithLabelValues(
				c.Method(),
				path,
				status,
				registryType,
			).Observe(float64(size))
		}
		
		// 캐시 메트릭 업데이트
		updateCacheMetrics(c, metrics, registryType)
		
		// 프록시 메트릭 업데이트
		updateProxyMetrics(c, metrics, registryType)
		
		// 인증 메트릭 업데이트
		updateAuthMetrics(c, metrics)
		
		// 검증 메트릭 업데이트
		updateVerificationMetrics(c, metrics, registryType)
		
		return err
	}
}

// updateCacheMetrics 캐시 관련 메트릭 업데이트
func updateCacheMetrics(c *fiber.Ctx, metrics *Metrics, registryType string) {
	// 캐시 히트/미스 확인
	cacheHit := c.Locals("cache_hit")
	cacheBackend := getCacheBackend(c)
	
	if cacheHit != nil {
		if hit, ok := cacheHit.(bool); ok {
			if hit {
				metrics.CacheHitsTotal.WithLabelValues(registryType, cacheBackend).Inc()
				
				// 대역폭 절약량 계산
				if size := len(c.Response().Body()); size > 0 {
					metrics.CacheBandwidthSaved.WithLabelValues(registryType).Add(float64(size))
				}
			} else {
				metrics.CacheMissesTotal.WithLabelValues(registryType, cacheBackend).Inc()
			}
		}
	}
	
	// 캐시 제거 이벤트
	if evicted := c.Locals("cache_evicted"); evicted != nil {
		if reason, ok := evicted.(string); ok {
			metrics.CacheEvictionsTotal.WithLabelValues(registryType, cacheBackend, reason).Inc()
		}
	}
}

// updateProxyMetrics 프록시 관련 메트릭 업데이트
func updateProxyMetrics(c *fiber.Ctx, metrics *Metrics, registryType string) {
	// 프록시 요청인지 확인
	if !strings.HasPrefix(c.Path(), "/proxy/") {
		return
	}
	
	upstream := getUpstream(c, registryType)
	
	// 프록시 요청 카운트
	metrics.ProxyRequestsTotal.WithLabelValues(
		registryType,
		upstream,
		c.Method(),
	).Inc()
	
	// 프록시 오류
	if c.Response().StatusCode() >= 400 {
		errorType := getErrorType(c.Response().StatusCode())
		metrics.ProxyErrorsTotal.WithLabelValues(
			registryType,
			upstream,
			errorType,
		).Inc()
	}
	
	// 업스트림 요청 시간
	if upstreamDuration := c.Locals("upstream_duration"); upstreamDuration != nil {
		if duration, ok := upstreamDuration.(time.Duration); ok {
			metrics.ProxyUpstreamDuration.WithLabelValues(
				registryType,
				upstream,
			).Observe(duration.Seconds())
		}
	}
	
	// 전송 바이트 수
	if c.Method() == "GET" || c.Method() == "HEAD" {
		// 다운로드
		if size := len(c.Response().Body()); size > 0 {
			metrics.ProxyBytesTransferred.WithLabelValues(
				registryType,
				"download",
			).Add(float64(size))
		}
	} else if c.Method() == "POST" || c.Method() == "PUT" {
		// 업로드
		if c.Request().Header.ContentLength() > 0 {
			metrics.ProxyBytesTransferred.WithLabelValues(
				registryType,
				"upload",
			).Add(float64(c.Request().Header.ContentLength()))
		}
	}
}

// updateAuthMetrics 인증 관련 메트릭 업데이트
func updateAuthMetrics(c *fiber.Ctx, metrics *Metrics) {
	// 인증 시도
	if authAttempt := c.Locals("auth_attempt"); authAttempt != nil {
		if method, ok := authAttempt.(string); ok {
			authResult := "success"
			if c.Response().StatusCode() == 401 || c.Response().StatusCode() == 403 {
				authResult = "failure"
			}
			
			metrics.AuthAttemptsTotal.WithLabelValues(method, authResult).Inc()
			
			// 인증 실패
			if authResult == "failure" {
				reason := "invalid_credentials"
				if c.Response().StatusCode() == 403 {
					reason = "forbidden"
				}
				metrics.AuthFailuresTotal.WithLabelValues(method, reason).Inc()
			}
		}
	}
}

// updateVerificationMetrics 검증 관련 메트릭 업데이트
func updateVerificationMetrics(c *fiber.Ctx, metrics *Metrics, registryType string) {
	// 패키지 검증 수행 여부
	if verified := c.Locals("packageVerified"); verified != nil {
		result := "success"
		if !verified.(bool) {
			result = "failure"
		}
		
		metrics.PackageVerifications.WithLabelValues(registryType, result).Inc()
		
		// 검증 실패 상세
		if result == "failure" {
			if failureType := c.Locals("verificationFailureType"); failureType != nil {
				if ft, ok := failureType.(string); ok {
					metrics.VerificationFailures.WithLabelValues(registryType, ft).Inc()
				}
			}
		}
		
		// 검증 소요 시간
		if verificationDuration := c.Locals("verificationDuration"); verificationDuration != nil {
			if duration, ok := verificationDuration.(time.Duration); ok {
				metrics.VerificationDuration.WithLabelValues(registryType).Observe(duration.Seconds())
			}
		}
	}
}

// extractRegistryType 요청에서 레지스트리 타입 추출
func extractRegistryType(c *fiber.Ctx) string {
	// /proxy/:type/* 패턴에서 추출
	if strings.HasPrefix(c.Path(), "/proxy/") {
		parts := strings.Split(c.Path(), "/")
		if len(parts) >= 3 {
			return parts[2]
		}
	}
	
	// Locals에서 확인
	if registryType := c.Locals("registry_type"); registryType != nil {
		if rt, ok := registryType.(string); ok {
			return rt
		}
	}
	
	return "unknown"
}

// normalizePath 경로 정규화 (카디널리티 감소)
func normalizePath(path string) string {
	// /proxy/:type/* 패턴 정규화
	if strings.HasPrefix(path, "/proxy/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			registryType := parts[2]
			return "/proxy/" + registryType + "/*"
		}
	}
	
	// 기타 공통 패턴
	switch {
	case path == "/":
		return "/"
	case path == "/healthz":
		return "/healthz"
	case path == "/metrics":
		return "/metrics"
	case strings.HasPrefix(path, "/api/"):
		return "/api/*"
	default:
		return "/other"
	}
}

// getCacheBackend 캐시 백엔드 타입 반환
func getCacheBackend(c *fiber.Ctx) string {
	if backend := c.Locals("cache_backend"); backend != nil {
		if b, ok := backend.(string); ok {
			return b
		}
	}
	return "file" // 기본값
}

// getUpstream 업스트림 서버 정보 반환
func getUpstream(c *fiber.Ctx, registryType string) string {
	if upstream := c.Locals("upstream"); upstream != nil {
		if u, ok := upstream.(string); ok {
			return u
		}
	}
	
	// 레지스트리 타입별 기본 업스트림
	switch registryType {
	case "npm":
		return "registry.npmjs.org"
	case "pypi":
		return "pypi.org"
	case "docker":
		return "registry-1.docker.io"
	case "maven":
		return "repo1.maven.org"
	case "apt":
		return "archive.ubuntu.com"
	default:
		return "unknown"
	}
}

// getErrorType HTTP 상태 코드에서 오류 타입 추출
func getErrorType(statusCode int) string {
	switch {
	case statusCode >= 400 && statusCode < 500:
		switch statusCode {
		case 400:
			return "bad_request"
		case 401:
			return "unauthorized"
		case 403:
			return "forbidden"
		case 404:
			return "not_found"
		case 429:
			return "rate_limited"
		default:
			return "client_error"
		}
	case statusCode >= 500:
		switch statusCode {
		case 500:
			return "internal_error"
		case 502:
			return "bad_gateway"
		case 503:
			return "service_unavailable"
		case 504:
			return "gateway_timeout"
		default:
			return "server_error"
		}
	default:
		return "unknown"
	}
}