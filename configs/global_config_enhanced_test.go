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