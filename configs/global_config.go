package configs

import (
	"path"
	"proxynd/helpers"
)

type Cache struct {
	TTL         int                    `yaml:"ttl,omitempty" default:"3600"`
	PackageTTLs map[string]int         `yaml:"package_ttls,omitempty"`
	PatternTTLs map[string]int         `yaml:"pattern_ttls,omitempty"`
	MetadataTTLs map[string]int        `yaml:"metadata_ttls,omitempty"`
}

// GetDefaultPackageTTLs 패키지 타입별 기본 TTL 반환
func GetDefaultPackageTTLs() map[string]int {
	return map[string]int{
		"apt":    3600, // 1 hour
		"npm":    1800, // 30 minutes
		"pip":    2400, // 40 minutes
		"docker": 7200, // 2 hours
		"maven":  5400, // 1.5 hours
		"yum":    3600, // 1 hour
		"apk":    1800, // 30 minutes
		"cargo":  3600, // 1 hour
		"go":     1800, // 30 minutes
		"helm":   3600, // 1 hour
	}
}

// GetTTLForPackageType 패키지 타입별 TTL 조회
func (c *Cache) GetTTLForPackageType(packageType string) int {
	// 설정된 패키지별 TTL이 있으면 사용
	if c.PackageTTLs != nil {
		if ttl, exists := c.PackageTTLs[packageType]; exists && ttl > 0 {
			return ttl
		}
	}
	
	// 기본 패키지별 TTL 사용
	defaults := GetDefaultPackageTTLs()
	if ttl, exists := defaults[packageType]; exists {
		return ttl
	}
	
	// 전역 기본 TTL 사용
	if c.TTL > 0 {
		return c.TTL
	}
	
	// 최종 기본값
	return 3600
}

// GetTTLForPackage 패키지명과 타입에 따른 TTL 조회 (패턴 매칭 포함)
func (c *Cache) GetTTLForPackage(packageName, packageType string) int {
	// 1. 패턴별 TTL 우선 확인
	if c.PatternTTLs != nil {
		for pattern, ttl := range c.PatternTTLs {
			if ttl > 0 && c.matchesPattern(packageName, pattern) {
				return ttl
			}
		}
	}
	
	// 2. 패키지 타입별 TTL 사용
	return c.GetTTLForPackageType(packageType)
}

// GetTTLForMetadata 메타데이터 파일에 대한 TTL 조회
func (c *Cache) GetTTLForMetadata(filename, packageType string) int {
	// 1. 메타데이터별 TTL 우선 확인
	if c.MetadataTTLs != nil {
		if ttl, exists := c.MetadataTTLs[filename]; exists && ttl > 0 {
			return ttl
		}
	}
	
	// 2. 패키지 타입별 TTL 사용 (메타데이터는 일반적으로 더 짧은 TTL)
	baseTTL := c.GetTTLForPackageType(packageType)
	
	// 메타데이터는 기본적으로 패키지 TTL의 1/3 사용
	metadataTTL := baseTTL / 3
	if metadataTTL < 300 { // 최소 5분
		metadataTTL = 300
	}
	
	return metadataTTL
}

// matchesPattern 패키지명이 패턴과 일치하는지 확인
func (c *Cache) matchesPattern(packageName, pattern string) bool {
	// 간단한 와일드카드 패턴 매칭 구현
	// "*"는 임의의 문자열과 매치
	if pattern == "*" {
		return true
	}
	
	// "*-suffix" 패턴 처리
	if len(pattern) > 1 && pattern[0] == '*' && pattern[1] == '-' {
		suffix := pattern[2:]
		return len(packageName) >= len(suffix) && 
			   packageName[len(packageName)-len(suffix):] == suffix
	}
	
	// "prefix-*" 패턴 처리
	if len(pattern) > 1 && pattern[len(pattern)-1] == '*' && pattern[len(pattern)-2] == '-' {
		prefix := pattern[:len(pattern)-2]
		return len(packageName) >= len(prefix) && 
			   packageName[:len(prefix)] == prefix
	}
	
	// "*substring*" 패턴 처리
	if len(pattern) >= 2 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		substring := pattern[1 : len(pattern)-1]
		return len(substring) == 0 || contains(packageName, substring)
	}
	
	// 정확한 일치
	return packageName == pattern
}

// contains 문자열 포함 여부 확인 (Go 1.18 이전 호환)
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || 
		   (len(s) > len(substr) && (s[:len(substr)] == substr || 
		    s[len(s)-len(substr):] == substr || 
		    indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring 부분 문자열 인덱스 찾기
func indexOfSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

type GlobalConfig struct {
	StorageDir string `yaml:"storage_dir,omitempty"`
	ConfigDir  string `yaml:"config_dir,omitempty"`
	Cache      Cache  `yaml:"cache,omitempty"`
}

func (cfg *GlobalConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "global.yaml"))
}

func (cfg *GlobalConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "global.yaml"), cfg)
}
