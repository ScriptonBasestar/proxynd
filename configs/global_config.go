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
