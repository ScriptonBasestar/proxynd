package configs

import (
	"testing"
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