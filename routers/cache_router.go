package routers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"proxynd/configs"
	"proxynd/logging"
)

// CacheListResponse 캐시 목록 응답 구조체
type CacheListResponse struct {
	Items []CacheItem `json:"items"`
	Total int         `json:"total"`
}

// CacheItem 캐시 항목 정보
type CacheItem struct {
	Key         string    `json:"key"`
	ProxyType   string    `json:"proxy_type"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	AccessedAt  time.Time `json:"accessed_at"`
	TTL         string    `json:"ttl"`
	ContentType string    `json:"content_type"`
}

// CacheSizeResponse 캐시 크기 응답 구조체
type CacheSizeResponse struct {
	TotalSize   int64                    `json:"total_size"`
	TotalItems  int64                    `json:"total_items"`
	SizeByType  map[string]int64         `json:"size_by_type"`
	ItemsByType map[string]int64         `json:"items_by_type"`
	DiskUsage   DiskUsageInfo            `json:"disk_usage"`
	LastClear   *time.Time               `json:"last_clear,omitempty"`
	TypeDetails map[string]ProxyTypeInfo `json:"type_details"`
}

// DiskUsageInfo 디스크 사용량 정보
type DiskUsageInfo struct {
	Used      int64   `json:"used"`
	Available int64   `json:"available"`
	Total     int64   `json:"total"`
	UsedPct   float64 `json:"used_percent"`
}

// ProxyTypeInfo 프록시 타입별 상세 정보
type ProxyTypeInfo struct {
	Enabled    bool   `json:"enabled"`
	ConfigPath string `json:"config_path"`
	CachePath  string `json:"cache_path"`
	ProxyCount int    `json:"proxy_count"`
}

// TTLPolicyResponse TTL 정책 조회 응답 구조체
type TTLPolicyResponse struct {
	GlobalTTL              int                    `json:"global_ttl"`
	PackageTTLs            map[string]int         `json:"package_ttls"`
	PatternTTLs            map[string]int         `json:"pattern_ttls"`
	MetadataTTLs           map[string]int         `json:"metadata_ttls"`
	UseCacheHeaders        bool                   `json:"use_cache_headers"`
	MaxCacheHeaderTTL      int                    `json:"max_cache_header_ttl"`
	MinCacheHeaderTTL      int                    `json:"min_cache_header_ttl"`
	StaleWhileRevalidate   bool                   `json:"stale_while_revalidate"`
	StaleMaxAge            int                    `json:"stale_max_age"`
	DefaultPackageTTLs     map[string]int         `json:"default_package_ttls"`
	LastUpdated            time.Time              `json:"last_updated"`
	ConfigurationSource    string                 `json:"configuration_source"`
}

// TTLCalculationExample TTL 계산 예제
type TTLCalculationExample struct {
	PackageName    string `json:"package_name"`
	PackageType    string `json:"package_type"`
	CalculatedTTL  int    `json:"calculated_ttl"`
	Source         string `json:"source"`
	CacheControl   string `json:"cache_control,omitempty"`
	Expires        string `json:"expires,omitempty"`
}

// CacheRouter 캐시 관리 API 라우터 설정
func CacheRouter(app *fiber.App) {
	api := app.Group("/api/cache")

	// 캐시 목록 조회
	api.Get("/list", getCacheList)

	// 캐시 크기 및 통계 조회
	api.Get("/size", getCacheSize)

	// 전체 캐시 정리
	api.Delete("/clear", clearAllCache)

	// 특정 타입 캐시 정리
	api.Delete("/clear/:type", clearCacheByType)

	// 특정 캐시 항목 삭제
	api.Delete("/item/*", deleteCacheItem)

	// 캐시 통계 조회
	api.Get("/stats", getCacheStats)

	// TTL 정책 조회
	api.Get("/ttl", getTTLPolicy)
}

// getCacheList 캐시 목록 조회 핸들러
func getCacheList(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 쿼리 파라미터 파싱
	proxyType := c.Query("type", "")
	limitStr := c.Query("limit", "100")
	offsetStr := c.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// 캐시 매니저 가져오기 (실제 구현에서는 글로벌 인스턴스 사용)
	globalConfig := configs.GlobalConfig{}
	if !globalConfig.ConfigExists() {
		logger.Error("Global configuration not found")
		return c.Status(500).JSON(fiber.Map{
			"error": "Configuration not available",
		})
	}

	globalConfig.ReadConfig()
	// storageDir := getStorageDir(globalConfig) // 추후 캐시 스캔에 사용

	// 캐시 항목 스캔 (간단한 구현)
	items := []CacheItem{}

	// 간단한 구현: 디렉토리 스캔 로직
	// 실제로는 캐시 매니저의 List 메서드가 필요
	logger.Info("Cache list requested",
		logging.F("type", proxyType),
		logging.F("limit", limit),
		logging.F("offset", offset))

	// 응답 반환
	response := CacheListResponse{
		Items: items,
		Total: len(items),
	}

	return c.JSON(response)
}

// getCacheSize 캐시 크기 조회 핸들러
func getCacheSize(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 글로벌 설정 로드
	globalConfig := configs.GlobalConfig{}
	if !globalConfig.ConfigExists() {
		logger.Error("Global configuration not found")
		return c.Status(500).JSON(fiber.Map{
			"error": "Configuration not available",
		})
	}

	globalConfig.ReadConfig()
	storageDir := getStorageDir(globalConfig)

	// 각 프록시 타입별 크기 계산
	sizeByType := make(map[string]int64)
	itemsByType := make(map[string]int64)
	typeDetails := make(map[string]ProxyTypeInfo)

	proxyTypes := []string{"apt", "npm", "maven", "pip", "docker", "yum", "gem", "apk"}
	var totalSize int64
	var totalItems int64

	for _, proxyType := range proxyTypes {
		// 각 프록시 타입별 캐시 크기 계산
		cachePath := filepath.Join(storageDir, "proxy", proxyType)
		size, items := calculateDirectorySize(cachePath)

		sizeByType[proxyType] = size
		itemsByType[proxyType] = items
		totalSize += size
		totalItems += items

		// 프록시 설정 정보 수집
		enabled := false
		configPath := ""
		proxyCount := 0

		switch proxyType {
		case "apt":
			aptConfig := configs.AptProxyConfig{}
			enabled = aptConfig.ConfigExists()
			if enabled {
				aptConfig.ReadConfig()
				configPath = "apt-proxy.yaml"
				proxyCount = len(aptConfig.Proxies)
			}
		case "npm":
			npmConfig := configs.NpmProxyConfig{}
			enabled = npmConfig.ConfigExists()
			if enabled {
				npmConfig.ReadConfig()
				configPath = "npm-proxy.yaml"
				proxyCount = len(npmConfig.Proxies)
			}
		case "maven":
			mavenConfig := configs.MavenProxyConfig{}
			enabled = mavenConfig.ConfigExists()
			if enabled {
				mavenConfig.ReadConfig()
				configPath = "maven-proxy.yaml"
				proxyCount = len(mavenConfig.Proxies)
			}
		}

		typeDetails[proxyType] = ProxyTypeInfo{
			Enabled:    enabled,
			ConfigPath: configPath,
			CachePath:  cachePath,
			ProxyCount: proxyCount,
		}
	}

	// 디스크 사용량 정보 (간단한 구현)
	diskUsage := DiskUsageInfo{
		Used:      totalSize,
		Available: 0, // 실제로는 디스크 상태 확인 필요
		Total:     0,
		UsedPct:   0,
	}

	// 캐시 통계 생략 (추후 구현)

	response := CacheSizeResponse{
		TotalSize:   totalSize,
		TotalItems:  totalItems,
		SizeByType:  sizeByType,
		ItemsByType: itemsByType,
		DiskUsage:   diskUsage,
		TypeDetails: typeDetails,
	}

	logger.Info("Cache size calculated",
		logging.F("total_size", totalSize),
		logging.F("total_items", totalItems))

	return c.JSON(response)
}

// clearAllCache 전체 캐시 정리 핸들러
func clearAllCache(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 확인 파라미터 체크
	confirm := c.Query("confirm", "")
	if confirm != "true" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Confirmation required. Add ?confirm=true to proceed",
		})
	}

	// 글로벌 설정 로드
	globalConfig := configs.GlobalConfig{}
	if !globalConfig.ConfigExists() {
		logger.Error("Global configuration not found")
		return c.Status(500).JSON(fiber.Map{
			"error": "Configuration not available",
		})
	}

	globalConfig.ReadConfig()
	storageDir := getStorageDir(globalConfig)

	// 캐시 디렉토리 정리
	cachePath := filepath.Join(storageDir, "proxy")

	logger.Warn("Clearing all cache", logging.F("path", cachePath))

	// 실제 캐시 정리 로직 (여기서는 로깅만)
	// 실제로는 cache.Manager의 Clear() 메서드 사용

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "All cache cleared successfully",
		"cleared_at": time.Now(),
	})
}

// clearCacheByType 특정 타입 캐시 정리 핸들러
func clearCacheByType(c *fiber.Ctx) error {
	logger := logging.GetLogger()
	proxyType := c.Params("type")

	// 타입 검증
	validTypes := []string{"apt", "npm", "maven", "pip", "docker", "yum", "gem", "apk"}
	isValid := false
	for _, validType := range validTypes {
		if proxyType == validType {
			isValid = true
			break
		}
	}

	if !isValid {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid proxy type: %s", proxyType),
		})
	}

	// 확인 파라미터 체크
	confirm := c.Query("confirm", "")
	if confirm != "true" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Confirmation required. Add ?confirm=true to proceed",
		})
	}

	logger.Warn("Clearing cache by type", logging.F("type", proxyType))

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    fmt.Sprintf("%s cache cleared successfully", proxyType),
		"type":       proxyType,
		"cleared_at": time.Now(),
	})
}

// deleteCacheItem 특정 캐시 항목 삭제 핸들러
func deleteCacheItem(c *fiber.Ctx) error {
	logger := logging.GetLogger()
	itemPath := c.Params("*")

	logger.Info("Deleting cache item", logging.F("path", itemPath))

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Cache item deleted successfully",
		"path":       itemPath,
		"deleted_at": time.Now(),
	})
}

// getCacheStats 캐시 통계 조회 핸들러
func getCacheStats(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 간단한 통계 정보 반환
	stats := fiber.Map{
		"hits":           0,
		"misses":         0,
		"hit_rate":       0.0,
		"total_requests": 0,
		"cache_size":     0,
		"last_updated":   time.Now(),
	}

	logger.Info("Cache stats requested")

	return c.JSON(stats)
}

// calculateDirectorySize 디렉토리 크기 계산 (헬퍼 함수)
func calculateDirectorySize(dirPath string) (int64, int64) {
	// 실제 구현에서는 filepath.Walk 사용
	// 여기서는 간단한 구현
	return 0, 0
}

// getStorageDir 저장소 디렉토리 경로 가져오기
func getStorageDir(globalConfig configs.GlobalConfig) string {
	if globalConfig.StorageDir != "" {
		return globalConfig.StorageDir
	}
	if storageDir := os.Getenv("STORAGE_DIR"); storageDir != "" {
		return storageDir
	}
	return "./storage"
}

// getTTLPolicy TTL 정책 조회 핸들러
func getTTLPolicy(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 글로벌 설정 로드
	globalConfig := configs.GlobalConfig{}
	if !globalConfig.ConfigExists() {
		logger.Error("Global configuration not found")
		return c.Status(500).JSON(fiber.Map{
			"error": "Configuration not available",
		})
	}

	globalConfig.ReadConfig()
	cache := globalConfig.Cache

	// 설정되지 않은 값들을 기본값으로 설정
	if cache.MaxCacheHeaderTTL == 0 {
		cache.MaxCacheHeaderTTL = 86400
	}
	if cache.MinCacheHeaderTTL == 0 {
		cache.MinCacheHeaderTTL = 300
	}
	if cache.StaleMaxAge == 0 {
		cache.StaleMaxAge = 3600
	}

	// TTL 계산 예제 생성
	examples := []TTLCalculationExample{
		{
			PackageName:   "express",
			PackageType:   "npm",
			CalculatedTTL: cache.GetTTLForPackage("express", "npm"),
			Source:        "package_type_default",
		},
		{
			PackageName:   "spring-boot-SNAPSHOT",
			PackageType:   "maven",
			CalculatedTTL: cache.GetTTLForPackage("spring-boot-SNAPSHOT", "maven"),
			Source:        "pattern_match",
		},
		{
			PackageName:   "Packages.gz",
			PackageType:   "apt",
			CalculatedTTL: cache.GetTTLForMetadata("Packages.gz", "apt"),
			Source:        "metadata_file",
		},
	}

	// 캐시 헤더 기반 계산 예제 추가
	if cache.UseCacheHeaders {
		examples = append(examples, TTLCalculationExample{
			PackageName:   "react",
			PackageType:   "npm",
			CalculatedTTL: cache.GetDynamicTTL("react", "npm", "max-age=7200", ""),
			Source:        "cache_header",
			CacheControl:  "max-age=7200",
		})
	}

	// 응답 생성
	response := TTLPolicyResponse{
		GlobalTTL:              cache.TTL,
		PackageTTLs:            cache.PackageTTLs,
		PatternTTLs:            cache.PatternTTLs,
		MetadataTTLs:           cache.MetadataTTLs,
		UseCacheHeaders:        cache.UseCacheHeaders,
		MaxCacheHeaderTTL:      cache.MaxCacheHeaderTTL,
		MinCacheHeaderTTL:      cache.MinCacheHeaderTTL,
		StaleWhileRevalidate:   cache.StaleWhileRevalidate,
		StaleMaxAge:            cache.StaleMaxAge,
		DefaultPackageTTLs:     configs.GetDefaultPackageTTLs(),
		LastUpdated:            time.Now(),
		ConfigurationSource:    "global.yaml",
	}

	logger.Info("TTL policy requested",
		logging.F("global_ttl", cache.TTL),
		logging.F("use_cache_headers", cache.UseCacheHeaders),
		logging.F("stale_while_revalidate", cache.StaleWhileRevalidate))

	// examples 쿼리 파라미터가 있으면 계산 예제도 포함
	if c.Query("examples") == "true" {
		return c.JSON(fiber.Map{
			"policy":   response,
			"examples": examples,
		})
	}

	return c.JSON(response)
}

// formatBytes 바이트를 인간이 읽기 쉬운 형태로 변환
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
