package routers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"proxynd/configs"
	"proxynd/metrics"
)

// MetricsRouter 메트릭 라우터 설정
func MetricsRouter(app *fiber.App, config *configs.UnifiedConfig) {
	// 메트릭 초기화
	metrics.InitMetrics()

	// 커스텀 수집기 등록
	prometheus.MustRegister(metrics.NewCustomCollector())

	// 메트릭 미들웨어 적용 (전체 앱에 적용)
	app.Use(metrics.PrometheusMiddleware())

	// 메트릭 엔드포인트 설정
	metricsPath := "/metrics"
	if config != nil && config.Metrics.Path != "" {
		metricsPath = config.Metrics.Path
	}

	// 메트릭 그룹 생성
	metricsGroup := app.Group(metricsPath)

	// 기본 인증 적용 (설정된 경우)
	if config != nil && config.Metrics.BasicAuth {
		metricsGroup.Use(basicauth.New(basicauth.Config{
			Users: getMetricsUsers(config),
			Realm: "Metrics",
			Unauthorized: func(c *fiber.Ctx) error {
				c.Set(fiber.HeaderWWWAuthenticate, "Basic realm=Metrics")
				return c.SendStatus(fiber.StatusUnauthorized)
			},
		}))
	}

	// Prometheus 핸들러 어댑터
	metricsGroup.Get("", adaptor(promhttp.Handler()))

	// 추가 메트릭 엔드포인트들
	setupAdditionalMetrics(app, config)
}

// adaptor Prometheus HTTP 핸들러를 Fiber 핸들러로 변환
func adaptor(h http.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fasthttpadaptor.NewFastHTTPHandler(h)(c.Context())
		return nil
	}
}

// getMetricsUsers 메트릭 엔드포인트용 사용자 정보 반환
func getMetricsUsers(config *configs.UnifiedConfig) map[string]string {
	// 기본 사용자
	users := map[string]string{
		"metrics": "prometheus", // 기본 사용자
	}

	// TODO: 설정에서 사용자 정보 로드
	// config.Security.Authentication.BasicAuth에서 메트릭 사용자 추출

	return users
}

// setupAdditionalMetrics 추가 메트릭 엔드포인트 설정
func setupAdditionalMetrics(app *fiber.App, config *configs.UnifiedConfig) {
	// TTL 통계 엔드포인트
	app.Get("/api/metrics/ttl", func(c *fiber.Ctx) error {
		collector := metrics.GetTTLCollector()
		registryType := c.Query("registry_type", "")
		
		if registryType != "" {
			stats := collector.GetStats(registryType)
			return c.JSON(fiber.Map{
				"registry_type": registryType,
				"statistics":    stats,
			})
		}

		// 모든 통계 반환
		allStats := collector.GetAllStats()
		return c.JSON(allStats)
	})

	// TTL 통계 상세 정보
	app.Get("/api/metrics/ttl/details", func(c *fiber.Ctx) error {
		collector := metrics.GetTTLCollector()
		limitStr := c.Query("limit", "100")
		
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100
		}

		recentEntries := collector.GetRecentEntries(limit)
		allStats := collector.GetAllStats()

		return c.JSON(fiber.Map{
			"statistics":     allStats,
			"recent_entries": recentEntries,
			"entry_count":    len(recentEntries),
		})
	})

	// TTL 패턴 분석
	app.Get("/api/metrics/ttl/patterns", func(c *fiber.Ctx) error {
		collector := metrics.GetTTLCollector()
		recentEntries := collector.GetRecentEntries(1000)

		// 소스별 분포 계산
		sourceDistribution := make(map[string]int)
		packageTypeDistribution := make(map[string]int)
		ttlRanges := map[string]int{
			"<5min":     0,  // < 300초
			"5-30min":   0,  // 300-1800초
			"30min-2h":  0,  // 1800-7200초
			"2h-12h":    0,  // 7200-43200초
			"12h-24h":   0,  // 43200-86400초
			">24h":      0,  // > 86400초
		}

		for _, entry := range recentEntries {
			sourceDistribution[entry.Source]++
			packageTypeDistribution[entry.PackageType]++

			// TTL 범위 분류
			ttl := entry.CalculatedTTL
			switch {
			case ttl < 300:
				ttlRanges["<5min"]++
			case ttl < 1800:
				ttlRanges["5-30min"]++
			case ttl < 7200:
				ttlRanges["30min-2h"]++
			case ttl < 43200:
				ttlRanges["2h-12h"]++
			case ttl < 86400:
				ttlRanges["12h-24h"]++
			default:
				ttlRanges[">24h"]++
			}
		}

		return c.JSON(fiber.Map{
			"source_distribution":       sourceDistribution,
			"package_type_distribution": packageTypeDistribution,
			"ttl_ranges":               ttlRanges,
			"total_analyzed":           len(recentEntries),
		})
	})
	// 캐시 통계 엔드포인트
	app.Get("/api/metrics/cache", func(c *fiber.Ctx) error {
		m := metrics.GetMetrics()

		// 캐시 통계 수집
		stats := fiber.Map{
			"npm":    getCacheStatsForRegistry("npm", m),
			"pypi":   getCacheStatsForRegistry("pypi", m),
			"apt":    getCacheStatsForRegistry("apt", m),
			"docker": getCacheStatsForRegistry("docker", m),
			"maven":  getCacheStatsForRegistry("maven", m),
		}

		return c.JSON(stats)
	})

	// 프록시 통계 엔드포인트
	app.Get("/api/metrics/proxy", func(c *fiber.Ctx) error {
		m := metrics.GetMetrics()

		stats := fiber.Map{
			"total_requests": getMetricValue(m.ProxyRequestsTotal),
			"total_errors":   getMetricValue(m.ProxyErrorsTotal),
			"bytes_transferred": fiber.Map{
				"upload":   getMetricValueWithLabel(m.ProxyBytesTransferred, "direction", "upload"),
				"download": getMetricValueWithLabel(m.ProxyBytesTransferred, "direction", "download"),
			},
		}

		return c.JSON(stats)
	})

	// 시스템 메트릭 엔드포인트
	app.Get("/api/metrics/system", func(c *fiber.Ctx) error {
		systemMetrics := metrics.GetSystemMetrics()

		stats := fiber.Map{
			"cpu": fiber.Map{
				"usage": systemMetrics["cpu_usage"],
			},
			"memory": fiber.Map{
				"total_kb":     systemMetrics["memory_total_kb"],
				"available_kb": systemMetrics["memory_available_kb"],
				"usage":        systemMetrics["memory_usage"],
			},
			"load": fiber.Map{
				"1m":  systemMetrics["load_1m"],
				"5m":  systemMetrics["load_5m"],
				"15m": systemMetrics["load_15m"],
			},
			"uptime_seconds": getMetricValue(metrics.GetMetrics().UptimeSeconds),
		}

		return c.JSON(stats)
	})

	// 건강 상태 메트릭 (Prometheus 형식)
	app.Get("/api/metrics/health", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; version=0.0.4")

		// 간단한 건강 상태 메트릭
		health := 1.0
		if !isHealthy() {
			health = 0.0
		}

		response := "# HELP proxynd_health Current health status (1 = healthy, 0 = unhealthy)\n"
		response += "# TYPE proxynd_health gauge\n"
		response += "proxynd_health " + strconv.FormatFloat(health, 'f', 1, 64) + "\n"

		return c.SendString(response)
	})
}

// getCacheStatsForRegistry 특정 레지스트리의 캐시 통계 반환
func getCacheStatsForRegistry(registryType string, m *metrics.Metrics) fiber.Map {
	// TODO: 실제 메트릭에서 값 추출
	// 현재는 예시 값

	hits := getMetricValueWithLabel(m.CacheHitsTotal, "registry_type", registryType)
	misses := getMetricValueWithLabel(m.CacheMissesTotal, "registry_type", registryType)
	total := hits + misses

	hitRate := 0.0
	if total > 0 {
		hitRate = hits / total
	}

	return fiber.Map{
		"hits":                  hits,
		"misses":                misses,
		"hit_rate":              hitRate,
		"size_bytes":            getMetricValueWithLabel(m.CacheSizeBytes, "registry_type", registryType),
		"items_count":           getMetricValueWithLabel(m.CacheItemsCount, "registry_type", registryType),
		"bandwidth_saved_bytes": getMetricValueWithLabel(m.CacheBandwidthSaved, "registry_type", registryType),
	}
}

// getMetricValue 메트릭 값 추출 (간단한 구현)
func getMetricValue(metric interface{}) float64 {
	// TODO: 실제 Prometheus 메트릭에서 값 추출
	// prometheus.Metric 인터페이스를 통해 값 읽기
	return 0.0
}

// getMetricValueWithLabel 레이블이 있는 메트릭 값 추출
func getMetricValueWithLabel(metric interface{}, labelName, labelValue string) float64 {
	// TODO: 실제 구현
	return 0.0
}

// isHealthy 서비스 건강 상태 확인
func isHealthy() bool {
	// TODO: 실제 건강 상태 확인 로직
	// - 필수 서비스 연결 상태
	// - 캐시 백엔드 상태
	// - 디스크 공간
	return true
}
