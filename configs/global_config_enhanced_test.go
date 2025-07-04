package configs

import (
	"testing"
	"time"
)

func TestGetDefaultPackageTTLs(t *testing.T) {
	defaults := GetDefaultPackageTTLs()

	// 기본 패키지 타입들이 모두 포함되어 있는지 확인
	expectedPackages := []string{"apt", "npm", "pip", "docker", "maven", "yum", "apk", "cargo", "go", "helm"}

	for _, pkg := range expectedPackages {
		if _, exists := defaults[pkg]; !exists {
			t.Errorf("Expected package type %s to have default TTL", pkg)
		}
	}

	// 특정 값들 검증
	if defaults["apt"] != 3600 {
		t.Errorf("Expected apt TTL to be 3600, got %d", defaults["apt"])
	}

	if defaults["npm"] != 1800 {
		t.Errorf("Expected npm TTL to be 1800, got %d", defaults["npm"])
	}

	if defaults["docker"] != 7200 {
		t.Errorf("Expected docker TTL to be 7200, got %d", defaults["docker"])
	}
}

func TestCache_GetTTLForPackageType(t *testing.T) {
	tests := []struct {
		name        string
		cache       Cache
		packageType string
		expected    int
	}{
		{
			name:        "Use configured package TTL",
			cache:       Cache{TTL: 1000, PackageTTLs: map[string]int{"apt": 2000}},
			packageType: "apt",
			expected:    2000,
		},
		{
			name:        "Use default package TTL when not configured",
			cache:       Cache{TTL: 1000},
			packageType: "apt",
			expected:    3600, // 기본 apt TTL
		},
		{
			name:        "Use global TTL for unknown package type",
			cache:       Cache{TTL: 1000},
			packageType: "unknown",
			expected:    1000,
		},
		{
			name:        "Use final default for unknown package with no global TTL",
			cache:       Cache{},
			packageType: "unknown",
			expected:    3600,
		},
		{
			name:        "Use npm default TTL",
			cache:       Cache{},
			packageType: "npm",
			expected:    1800,
		},
		{
			name:        "Override npm TTL with configured value",
			cache:       Cache{PackageTTLs: map[string]int{"npm": 900}},
			packageType: "npm",
			expected:    900,
		},
		{
			name:        "Zero configured TTL falls back to default",
			cache:       Cache{PackageTTLs: map[string]int{"apt": 0}},
			packageType: "apt",
			expected:    3600, // 기본 apt TTL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetTTLForPackageType(tt.packageType)
			if result != tt.expected {
				t.Errorf("Expected TTL %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCache_GetTTLForPackageType_AllDefaultPackages(t *testing.T) {
	cache := Cache{}
	defaults := GetDefaultPackageTTLs()

	for packageType, expectedTTL := range defaults {
		result := cache.GetTTLForPackageType(packageType)
		if result != expectedTTL {
			t.Errorf("Package %s: expected TTL %d, got %d", packageType, expectedTTL, result)
		}
	}
}

func TestCache_GetTTLForPackageType_ConfiguredOverrides(t *testing.T) {
	cache := Cache{
		TTL: 1000,
		PackageTTLs: map[string]int{
			"apt":    5000,
			"npm":    2500,
			"docker": 10000,
		},
	}

	// 설정된 오버라이드 확인
	if cache.GetTTLForPackageType("apt") != 5000 {
		t.Errorf("Expected apt TTL to be overridden to 5000")
	}

	if cache.GetTTLForPackageType("npm") != 2500 {
		t.Errorf("Expected npm TTL to be overridden to 2500")
	}

	if cache.GetTTLForPackageType("docker") != 10000 {
		t.Errorf("Expected docker TTL to be overridden to 10000")
	}

	// 설정되지 않은 패키지는 기본값 사용
	if cache.GetTTLForPackageType("pip") != 2400 {
		t.Errorf("Expected pip to use default TTL 2400")
	}
}

func TestCache_GetTTLForPackage(t *testing.T) {
	cache := Cache{
		TTL: 3600,
		PackageTTLs: map[string]int{
			"maven": 5400,
		},
		PatternTTLs: map[string]int{
			"*-SNAPSHOT": 300,
			"*-dev":      600,
			"*-alpha":    900,
			"test-*":     1200,
			"*debug*":    450,
		},
	}

	tests := []struct {
		name        string
		packageName string
		packageType string
		expected    int
	}{
		{
			name:        "SNAPSHOT version gets pattern TTL",
			packageName: "spring-boot-starter-SNAPSHOT",
			packageType: "maven",
			expected:    300,
		},
		{
			name:        "dev version gets pattern TTL",
			packageName: "mypackage-dev",
			packageType: "npm",
			expected:    600,
		},
		{
			name:        "alpha version gets pattern TTL",
			packageName: "library-alpha",
			packageType: "pip",
			expected:    900,
		},
		{
			name:        "test prefix gets pattern TTL",
			packageName: "test-utils",
			packageType: "npm",
			expected:    1200,
		},
		{
			name:        "debug substring gets pattern TTL",
			packageName: "my-debug-tool",
			packageType: "npm",
			expected:    450,
		},
		{
			name:        "normal package gets type TTL",
			packageName: "normal-package",
			packageType: "maven",
			expected:    5400,
		},
		{
			name:        "no pattern match uses default",
			packageName: "regular-lib",
			packageType: "npm",
			expected:    1800, // npm default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.GetTTLForPackage(tt.packageName, tt.packageType)
			if result != tt.expected {
				t.Errorf("Expected TTL %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCache_GetTTLForMetadata(t *testing.T) {
	cache := Cache{
		TTL: 3600,
		PackageTTLs: map[string]int{
			"apt": 3600,
			"yum": 3600,
		},
		MetadataTTLs: map[string]int{
			"repomd.xml":   300,
			"Packages.gz":  600,
			"package.json": 300,
		},
	}

	tests := []struct {
		name        string
		filename    string
		packageType string
		expected    int
	}{
		{
			name:        "Configured metadata TTL",
			filename:    "repomd.xml",
			packageType: "yum",
			expected:    300,
		},
		{
			name:        "Configured Packages.gz TTL",
			filename:    "Packages.gz",
			packageType: "apt",
			expected:    600,
		},
		{
			name:        "Configured package.json TTL",
			filename:    "package.json",
			packageType: "npm",
			expected:    300,
		},
		{
			name:        "Default metadata TTL (1/3 of package TTL)",
			filename:    "index.yaml",
			packageType: "apt",
			expected:    1200, // 3600 / 3
		},
		{
			name:        "Minimum metadata TTL",
			filename:    "Release",
			packageType: "npm", // npm default is 1800, 1800/3 = 600 > 300
			expected:    600,   // 1800 / 3
		},
		{
			name:        "Metadata TTL with minimum enforcement",
			filename:    "some-meta.txt",
			packageType: "go", // go default is 1800, 1800/3 = 600 > 300
			expected:    600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.GetTTLForMetadata(tt.filename, tt.packageType)
			if result != tt.expected {
				t.Errorf("Expected TTL %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCache_matchesPattern(t *testing.T) {
	cache := Cache{}

	tests := []struct {
		name        string
		packageName string
		pattern     string
		expected    bool
	}{
		{
			name:        "Wildcard matches everything",
			packageName: "anything",
			pattern:     "*",
			expected:    true,
		},
		{
			name:        "Suffix pattern match",
			packageName: "spring-boot-SNAPSHOT",
			pattern:     "*-SNAPSHOT",
			expected:    true,
		},
		{
			name:        "Suffix pattern no match",
			packageName: "spring-boot-RELEASE",
			pattern:     "*-SNAPSHOT",
			expected:    false,
		},
		{
			name:        "Prefix pattern match",
			packageName: "test-utils",
			pattern:     "test-*",
			expected:    true,
		},
		{
			name:        "Prefix pattern no match",
			packageName: "utils-test",
			pattern:     "test-*",
			expected:    false,
		},
		{
			name:        "Substring pattern match",
			packageName: "my-debug-tool",
			pattern:     "*debug*",
			expected:    true,
		},
		{
			name:        "Substring pattern no match",
			packageName: "my-prod-tool",
			pattern:     "*debug*",
			expected:    false,
		},
		{
			name:        "Exact match",
			packageName: "exact-package-name",
			pattern:     "exact-package-name",
			expected:    true,
		},
		{
			name:        "Exact no match",
			packageName: "different-package",
			pattern:     "exact-package-name",
			expected:    false,
		},
		{
			name:        "Empty substring pattern matches all",
			packageName: "any-package",
			pattern:     "**",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.matchesPattern(tt.packageName, tt.pattern)
			if result != tt.expected {
				t.Errorf("Pattern '%s' with package '%s': expected %t, got %t",
					tt.pattern, tt.packageName, tt.expected, result)
			}
		})
	}
}

func TestCache_GetTTLFromCacheHeaders(t *testing.T) {
	tests := []struct {
		name         string
		cache        Cache
		cacheControl string
		expires      string
		expected     int
	}{
		{
			name:         "Cache headers disabled",
			cache:        Cache{UseCacheHeaders: false},
			cacheControl: "max-age=3600",
			expires:      "",
			expected:     0,
		},
		{
			name:         "Max-age directive",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			cacheControl: "max-age=1800",
			expires:      "",
			expected:     1800,
		},
		{
			name:         "S-maxage directive (higher priority)",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			cacheControl: "max-age=1800, s-maxage=3600",
			expires:      "",
			expected:     3600,
		},
		{
			name:         "Multiple directives with s-maxage",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			cacheControl: "public, max-age=1800, s-maxage=2400, must-revalidate",
			expires:      "",
			expected:     2400,
		},
		{
			name:         "TTL below minimum gets enforced",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 600, MaxCacheHeaderTTL: 86400},
			cacheControl: "max-age=300",
			expires:      "",
			expected:     600,
		},
		{
			name:         "TTL above maximum gets enforced",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 3600},
			cacheControl: "max-age=7200",
			expires:      "",
			expected:     3600,
		},
		{
			name:         "Expires header fallback",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			cacheControl: "",
			expires:      time.Now().Add(2 * time.Hour).Format(time.RFC1123),
			expected:     7200, // approximately 2 hours
		},
		{
			name:         "No cache headers",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			cacheControl: "",
			expires:      "",
			expected:     0,
		},
		{
			name:         "Invalid max-age value",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			cacheControl: "max-age=invalid",
			expires:      "",
			expected:     0,
		},
		{
			name:         "Zero max-age value",
			cache:        Cache{UseCacheHeaders: true, MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			cacheControl: "max-age=0",
			expires:      "",
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetTTLFromCacheHeaders(tt.cacheControl, tt.expires)

			// Expires 헤더 테스트의 경우 약간의 오차 허용
			if tt.name == "Expires header fallback" {
				if result < 7170 || result > 7230 { // ±30초 오차 허용
					t.Errorf("Expected TTL around 7200, got %d", result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected TTL %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestCache_parseCacheControl(t *testing.T) {
	cache := Cache{}

	tests := []struct {
		name         string
		cacheControl string
		expected     int
	}{
		{
			name:         "Simple max-age",
			cacheControl: "max-age=3600",
			expected:     3600,
		},
		{
			name:         "Max-age with spaces",
			cacheControl: "max-age = 1800",
			expected:     0, // 공백 있는 형식은 파싱 안됨
		},
		{
			name:         "Multiple directives with max-age",
			cacheControl: "public, max-age=2400, must-revalidate",
			expected:     2400,
		},
		{
			name:         "S-maxage priority",
			cacheControl: "max-age=1800, s-maxage=3600",
			expected:     3600,
		},
		{
			name:         "Case insensitive",
			cacheControl: "MAX-AGE=1200",
			expected:     1200,
		},
		{
			name:         "No max-age directive",
			cacheControl: "public, must-revalidate",
			expected:     0,
		},
		{
			name:         "Invalid max-age value",
			cacheControl: "max-age=abc",
			expected:     0,
		},
		{
			name:         "Negative max-age",
			cacheControl: "max-age=-300",
			expected:     0,
		},
		{
			name:         "Zero max-age",
			cacheControl: "max-age=0",
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.parseCacheControl(tt.cacheControl)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCache_parseExpires(t *testing.T) {
	cache := Cache{}

	// 미래 시간 생성 (1시간 후)
	futureTime := time.Now().Add(1 * time.Hour)
	pastTime := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name     string
		expires  string
		expected int
	}{
		{
			name:     "RFC1123 format",
			expires:  futureTime.Format(time.RFC1123),
			expected: 3600, // approximately 1 hour
		},
		{
			name:     "RFC822 format",
			expires:  futureTime.Format(time.RFC822),
			expected: 3600,
		},
		{
			name:     "Past time",
			expires:  pastTime.Format(time.RFC1123),
			expected: 0,
		},
		{
			name:     "Invalid format",
			expires:  "invalid-date",
			expected: 0,
		},
		{
			name:     "Empty expires",
			expires:  "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.parseExpires(tt.expires)

			if tt.expected > 0 {
				// 시간 기반 테스트는 약간의 오차 허용 (±60초)
				if result < tt.expected-60 || result > tt.expected+60 {
					t.Errorf("Expected TTL around %d, got %d", tt.expected, result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected TTL %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestCache_enforceTTLLimits(t *testing.T) {
	tests := []struct {
		name     string
		cache    Cache
		inputTTL int
		expected int
	}{
		{
			name:     "TTL within limits",
			cache:    Cache{MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 86400},
			inputTTL: 3600,
			expected: 3600,
		},
		{
			name:     "TTL below minimum",
			cache:    Cache{MinCacheHeaderTTL: 600, MaxCacheHeaderTTL: 86400},
			inputTTL: 300,
			expected: 600,
		},
		{
			name:     "TTL above maximum",
			cache:    Cache{MinCacheHeaderTTL: 300, MaxCacheHeaderTTL: 3600},
			inputTTL: 7200,
			expected: 3600,
		},
		{
			name:     "Default limits when not set",
			cache:    Cache{MinCacheHeaderTTL: 0, MaxCacheHeaderTTL: 0},
			inputTTL: 100,
			expected: 300, // 기본 최소값
		},
		{
			name:     "Default max limit when not set",
			cache:    Cache{MinCacheHeaderTTL: 0, MaxCacheHeaderTTL: 0},
			inputTTL: 100000,
			expected: 86400, // 기본 최대값
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.enforceTTLLimits(tt.inputTTL)
			if result != tt.expected {
				t.Errorf("Expected TTL %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCache_GetDynamicTTL(t *testing.T) {
	cache := Cache{
		TTL:               3600,
		UseCacheHeaders:   true,
		MinCacheHeaderTTL: 300,
		MaxCacheHeaderTTL: 86400,
		PackageTTLs: map[string]int{
			"npm": 1800,
		},
		PatternTTLs: map[string]int{
			"*-SNAPSHOT": 300,
		},
	}

	tests := []struct {
		name         string
		packageName  string
		packageType  string
		cacheControl string
		expires      string
		expected     int
	}{
		{
			name:         "Cache header takes priority",
			packageName:  "my-package",
			packageType:  "npm",
			cacheControl: "max-age=2400",
			expires:      "",
			expected:     2400,
		},
		{
			name:         "Fallback to pattern TTL",
			packageName:  "my-package-SNAPSHOT",
			packageType:  "maven",
			cacheControl: "",
			expires:      "",
			expected:     300,
		},
		{
			name:         "Fallback to package type TTL",
			packageName:  "regular-package",
			packageType:  "npm",
			cacheControl: "",
			expires:      "",
			expected:     1800,
		},
		{
			name:         "Cache header overrides pattern",
			packageName:  "test-SNAPSHOT",
			packageType:  "maven",
			cacheControl: "max-age=1200",
			expires:      "",
			expected:     1200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.GetDynamicTTL(tt.packageName, tt.packageType, tt.cacheControl, tt.expires)
			if result != tt.expected {
				t.Errorf("Expected TTL %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCache_CreateCacheEntry(t *testing.T) {
	tests := []struct {
		name  string
		cache Cache
		ttl   int
	}{
		{
			name:  "Without stale-while-revalidate",
			cache: Cache{StaleWhileRevalidate: false},
			ttl:   3600,
		},
		{
			name:  "With stale-while-revalidate default",
			cache: Cache{StaleWhileRevalidate: true, StaleMaxAge: 0},
			ttl:   3600,
		},
		{
			name:  "With stale-while-revalidate custom",
			cache: Cache{StaleWhileRevalidate: true, StaleMaxAge: 1800},
			ttl:   3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeCreate := time.Now()
			entry := tt.cache.CreateCacheEntry(tt.ttl)
			afterCreate := time.Now()

			// CachedAt 시간이 생성 전후 범위에 있는지 확인
			if entry.CachedAt.Before(beforeCreate) || entry.CachedAt.After(afterCreate) {
				t.Errorf("CachedAt time %v is not within expected range", entry.CachedAt)
			}

			// TTL이 올바르게 설정되었는지 확인
			if entry.TTL != tt.ttl {
				t.Errorf("Expected TTL %d, got %d", tt.ttl, entry.TTL)
			}

			// StaleUntil 시간 확인
			expectedStaleUntil := entry.CachedAt.Add(time.Duration(tt.ttl) * time.Second)
			if tt.cache.StaleWhileRevalidate {
				staleMaxAge := tt.cache.StaleMaxAge
				if staleMaxAge <= 0 {
					staleMaxAge = 3600
				}
				expectedStaleUntil = entry.CachedAt.Add(time.Duration(tt.ttl+staleMaxAge) * time.Second)
			}

			if !entry.StaleUntil.Equal(expectedStaleUntil) {
				t.Errorf("Expected StaleUntil %v, got %v", expectedStaleUntil, entry.StaleUntil)
			}
		})
	}
}

func TestCache_IsFresh(t *testing.T) {
	cache := Cache{}
	now := time.Now()

	tests := []struct {
		name     string
		entry    CacheEntry
		expected bool
	}{
		{
			name: "Fresh cache",
			entry: CacheEntry{
				CachedAt: now.Add(-30 * time.Minute), // 30분 전 캐시
				TTL:      3600,                       // 1시간 TTL
			},
			expected: true,
		},
		{
			name: "Expired cache",
			entry: CacheEntry{
				CachedAt: now.Add(-2 * time.Hour), // 2시간 전 캐시
				TTL:      3600,                    // 1시간 TTL
			},
			expected: false,
		},
		{
			name: "Just expired cache",
			entry: CacheEntry{
				CachedAt: now.Add(-61 * time.Minute), // 61분 전 캐시
				TTL:      3600,                       // 1시간 TTL
			},
			expected: false,
		},
		{
			name: "Just fresh cache",
			entry: CacheEntry{
				CachedAt: now.Add(-59 * time.Minute), // 59분 전 캐시
				TTL:      3600,                       // 1시간 TTL
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.IsFresh(tt.entry)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestCache_IsStale(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		cache    Cache
		entry    CacheEntry
		expected bool
	}{
		{
			name:  "Stale-while-revalidate disabled",
			cache: Cache{StaleWhileRevalidate: false},
			entry: CacheEntry{
				CachedAt:   now.Add(-2 * time.Hour),   // 2시간 전 캐시
				TTL:        3600,                      // 1시간 TTL
				StaleUntil: now.Add(30 * time.Minute), // 아직 stale 기간 내
			},
			expected: false, // 기능이 비활성화되면 false
		},
		{
			name:  "Cache is still fresh",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-30 * time.Minute), // 30분 전 캐시
				TTL:        3600,                       // 1시간 TTL
				StaleUntil: now.Add(90 * time.Minute),  // stale 기간 내
			},
			expected: false, // 아직 신선함
		},
		{
			name:  "Cache is stale but within stale period",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-2 * time.Hour),   // 2시간 전 캐시
				TTL:        3600,                      // 1시간 TTL (만료됨)
				StaleUntil: now.Add(30 * time.Minute), // 아직 stale 기간 내
			},
			expected: true, // stale 서빙 가능
		},
		{
			name:  "Cache is completely expired",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-3 * time.Hour),    // 3시간 전 캐시
				TTL:        3600,                       // 1시간 TTL (만료됨)
				StaleUntil: now.Add(-30 * time.Minute), // stale 기간도 만료
			},
			expected: false, // 완전 만료
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.IsStale(tt.entry)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestCache_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		cache    Cache
		entry    CacheEntry
		expected bool
	}{
		{
			name:  "Fresh cache without stale-while-revalidate",
			cache: Cache{StaleWhileRevalidate: false},
			entry: CacheEntry{
				CachedAt: now.Add(-30 * time.Minute), // 30분 전 캐시
				TTL:      3600,                       // 1시간 TTL
			},
			expected: false,
		},
		{
			name:  "Expired cache without stale-while-revalidate",
			cache: Cache{StaleWhileRevalidate: false},
			entry: CacheEntry{
				CachedAt: now.Add(-2 * time.Hour), // 2시간 전 캐시
				TTL:      3600,                    // 1시간 TTL
			},
			expected: true,
		},
		{
			name:  "Fresh cache with stale-while-revalidate",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-30 * time.Minute), // 30분 전 캐시
				TTL:        3600,                       // 1시간 TTL
				StaleUntil: now.Add(90 * time.Minute),  // stale 기간 내
			},
			expected: false,
		},
		{
			name:  "Stale cache with stale-while-revalidate",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-2 * time.Hour),   // 2시간 전 캐시
				TTL:        3600,                      // 1시간 TTL (만료됨)
				StaleUntil: now.Add(30 * time.Minute), // 아직 stale 기간 내
			},
			expected: false, // stale 기간 내이므로 아직 만료 아님
		},
		{
			name:  "Completely expired cache with stale-while-revalidate",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-3 * time.Hour),    // 3시간 전 캐시
				TTL:        3600,                       // 1시간 TTL (만료됨)
				StaleUntil: now.Add(-30 * time.Minute), // stale 기간도 만료
			},
			expected: true, // 완전 만료
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.IsExpired(tt.entry)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestCache_ShouldRevalidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		cache    Cache
		entry    CacheEntry
		expected bool
	}{
		{
			name:  "Stale-while-revalidate disabled",
			cache: Cache{StaleWhileRevalidate: false},
			entry: CacheEntry{
				CachedAt:   now.Add(-2 * time.Hour),   // 2시간 전 캐시
				TTL:        3600,                      // 1시간 TTL (만료됨)
				StaleUntil: now.Add(30 * time.Minute), // stale 기간 내
			},
			expected: false, // 기능 비활성화
		},
		{
			name:  "Fresh cache",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-30 * time.Minute), // 30분 전 캐시
				TTL:        3600,                       // 1시간 TTL
				StaleUntil: now.Add(90 * time.Minute),  // stale 기간 내
			},
			expected: false, // 아직 신선함
		},
		{
			name:  "Stale cache - should revalidate",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-2 * time.Hour),   // 2시간 전 캐시
				TTL:        3600,                      // 1시간 TTL (만료됨)
				StaleUntil: now.Add(30 * time.Minute), // 아직 stale 기간 내
			},
			expected: true, // 백그라운드 재검증 필요
		},
		{
			name:  "Completely expired cache",
			cache: Cache{StaleWhileRevalidate: true},
			entry: CacheEntry{
				CachedAt:   now.Add(-3 * time.Hour),    // 3시간 전 캐시
				TTL:        3600,                       // 1시간 TTL (만료됨)
				StaleUntil: now.Add(-30 * time.Minute), // stale 기간도 만료
			},
			expected: false, // 완전 만료 (재검증 아닌 새로 가져와야 함)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.ShouldRevalidate(tt.entry)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestCache_GetCacheStrategy(t *testing.T) {
	now := time.Now()
	cache := Cache{StaleWhileRevalidate: true}

	tests := []struct {
		name     string
		entry    CacheEntry
		expected CacheStrategy
	}{
		{
			name: "Fresh cache - serve directly",
			entry: CacheEntry{
				CachedAt:   now.Add(-30 * time.Minute), // 30분 전 캐시
				TTL:        3600,                       // 1시간 TTL
				StaleUntil: now.Add(90 * time.Minute),  // stale 기간 내
			},
			expected: CacheStrategyServe,
		},
		{
			name: "Stale cache - serve while revalidating",
			entry: CacheEntry{
				CachedAt:   now.Add(-2 * time.Hour),   // 2시간 전 캐시
				TTL:        3600,                      // 1시간 TTL (만료됨)
				StaleUntil: now.Add(30 * time.Minute), // 아직 stale 기간 내
			},
			expected: CacheStrategyStaleWhileRevalidate,
		},
		{
			name: "Completely expired cache - revalidate",
			entry: CacheEntry{
				CachedAt:   now.Add(-3 * time.Hour),    // 3시간 전 캐시
				TTL:        3600,                       // 1시간 TTL (만료됨)
				StaleUntil: now.Add(-30 * time.Minute), // stale 기간도 만료
			},
			expected: CacheStrategyRevalidate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.GetCacheStrategy(tt.entry)
			if result != tt.expected {
				t.Errorf("Expected strategy %d, got %d", tt.expected, result)
			}
		})
	}
}
