package configs

import (
	"path"
	"strconv"
	"strings"
	"time"

	"proxynd/helpers"
)

type Cache struct {
	TTL                  int            `yaml:"ttl,omitempty" default:"3600" validate:"min=0,max=604800"`
	PackageTTLs          map[string]int `yaml:"package_ttls,omitempty"`
	PatternTTLs          map[string]int `yaml:"pattern_ttls,omitempty"`
	MetadataTTLs         map[string]int `yaml:"metadata_ttls,omitempty"`
	UseCacheHeaders      bool           `yaml:"use_cache_headers,omitempty"`
	MaxCacheHeaderTTL    int            `yaml:"max_cache_header_ttl,omitempty" default:"86400" validate:"min=0,max=604800"`
	MinCacheHeaderTTL    int            `yaml:"min_cache_header_ttl,omitempty" default:"300" validate:"min=0,max=86400"`
	StaleWhileRevalidate bool           `yaml:"stale_while_revalidate,omitempty"`
	StaleMaxAge          int            `yaml:"stale_max_age,omitempty" default:"3600" validate:"min=0,max=86400"`
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

type BasicAuthConfig struct {
	Users map[string]string `yaml:"users,omitempty" validate:"dive,keys,min=1,endkeys,min=1"`
	Realm string            `yaml:"realm,omitempty" default:"Restricted" validate:"min=1,max=100"`
}

type AuthenticationConfig struct {
	BasicAuth *BasicAuthConfig `yaml:"basic_auth,omitempty"`
	OAuth2    *OAuth2Config    `yaml:"oauth2,omitempty"`
}

type GlobalConfig struct {
	StorageDir     string                `yaml:"storage_dir,omitempty" validate:"omitempty,path"`
	ConfigDir      string                `yaml:"config_dir,omitempty" validate:"omitempty,path"`
	Cache          Cache                 `yaml:"cache,omitempty" validate:"dive"`
	Authentication *AuthenticationConfig `yaml:"authentication,omitempty" validate:"omitempty,dive"`
	CacheDir       string                `yaml:"cache_dir,omitempty" validate:"omitempty,path"`
	CacheTTL       int                   `yaml:"cache_ttl,omitempty" validate:"min=0,max=604800"`
	MaxCacheSize   int64                 `yaml:"max_cache_size,omitempty" validate:"min=0"`
}

func (cfg *GlobalConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "global.yaml"))
}

func (cfg *GlobalConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "global.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the global configuration
func (cfg *GlobalConfig) Validate() error {
	// Validate struct tags
	if err := ValidateStruct(cfg); err != nil {
		return err
	}
	
	// Additional custom validation
	if cfg.Cache.MinCacheHeaderTTL > cfg.Cache.MaxCacheHeaderTTL {
		return helpers.NewConfigFieldError("global", "min_cache_header_ttl cannot be greater than max_cache_header_ttl")
	}
	
	// Validate authentication if present
	if cfg.Authentication != nil && cfg.Authentication.OAuth2 != nil {
		if err := cfg.Authentication.OAuth2.Validate(); err != nil {
			return err
		}
	}
	
	return nil
}

// GetTTLFromCacheHeaders HTTP 캐시 헤더에서 TTL 계산
func (c *Cache) GetTTLFromCacheHeaders(cacheControl, expires string) int {
	if !c.UseCacheHeaders {
		return 0 // 캐시 헤더 사용 안함
	}

	// Cache-Control 헤더 우선 처리
	if cacheControl != "" {
		if ttl := c.parseCacheControl(cacheControl); ttl > 0 {
			return c.enforceTTLLimits(ttl)
		}
	}

	// Expires 헤더 처리
	if expires != "" {
		if ttl := c.parseExpires(expires); ttl > 0 {
			return c.enforceTTLLimits(ttl)
		}
	}

	return 0 // 유효한 헤더 없음
}

// parseCacheControl Cache-Control 헤더 파싱
func (c *Cache) parseCacheControl(cacheControl string) int {
	directives := strings.Split(strings.ToLower(cacheControl), ",")

	var maxAge int
	var sMaxAge int

	for _, directive := range directives {
		directive = strings.TrimSpace(directive)

		// s-maxage 지시어 처리 (공유 캐시용, 우선순위 높음) - 먼저 확인
		if strings.HasPrefix(directive, "s-maxage=") {
			sMaxAgeStr := strings.TrimPrefix(directive, "s-maxage=")
			if parsed, err := strconv.Atoi(sMaxAgeStr); err == nil && parsed > 0 {
				sMaxAge = parsed
			}
		} else if strings.HasPrefix(directive, "max-age=") {
			// max-age 지시어 처리
			maxAgeStr := strings.TrimPrefix(directive, "max-age=")
			if parsed, err := strconv.Atoi(maxAgeStr); err == nil && parsed > 0 {
				maxAge = parsed
			}
		}
	}

	// s-maxage가 있으면 우선 반환
	if sMaxAge > 0 {
		return sMaxAge
	}

	// 그렇지 않으면 max-age 반환
	return maxAge
}

// parseExpires Expires 헤더 파싱
func (c *Cache) parseExpires(expires string) int {
	// RFC 1123 형식 파싱 시도
	layouts := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		"Mon, 02 Jan 2006 15:04:05 MST",
	}

	for _, layout := range layouts {
		if expiresTime, err := time.Parse(layout, expires); err == nil {
			ttl := int(time.Until(expiresTime).Seconds())
			if ttl > 0 {
				return ttl
			}
			break
		}
	}

	return 0
}

// enforceTTLLimits TTL 최소/최대값 강제 적용
func (c *Cache) enforceTTLLimits(ttl int) int {
	minTTL := c.MinCacheHeaderTTL
	if minTTL <= 0 {
		minTTL = 300 // 기본 최소값 5분
	}

	maxTTL := c.MaxCacheHeaderTTL
	if maxTTL <= 0 {
		maxTTL = 86400 // 기본 최대값 24시간
	}

	if ttl < minTTL {
		return minTTL
	}
	if ttl > maxTTL {
		return maxTTL
	}

	return ttl
}

// GetDynamicTTL 동적 TTL 계산 (헤더 기반 + 폴백)
func (c *Cache) GetDynamicTTL(packageName, packageType, cacheControl, expires string) int {
	// 1. 캐시 헤더 기반 TTL 우선 시도
	if headerTTL := c.GetTTLFromCacheHeaders(cacheControl, expires); headerTTL > 0 {
		return headerTTL
	}

	// 2. 기존 패키지 기반 TTL 폴백
	return c.GetTTLForPackage(packageName, packageType)
}

// CacheEntry 캐시 항목 정보 구조체
type CacheEntry struct {
	CachedAt   time.Time // 캐시된 시간
	TTL        int       // 원본 TTL (초)
	StaleUntil time.Time // stale 만료 시간
}

// IsFresh 캐시가 신선한지 확인
func (c *Cache) IsFresh(entry CacheEntry) bool {
	now := time.Now()
	expireTime := entry.CachedAt.Add(time.Duration(entry.TTL) * time.Second)
	return now.Before(expireTime)
}

// IsStale 캐시가 만료되었지만 stale 서빙 가능한지 확인
func (c *Cache) IsStale(entry CacheEntry) bool {
	if !c.StaleWhileRevalidate {
		return false
	}

	now := time.Now()
	expireTime := entry.CachedAt.Add(time.Duration(entry.TTL) * time.Second)

	// TTL은 만료되었지만 stale 기간 내에 있는지 확인
	return now.After(expireTime) && now.Before(entry.StaleUntil)
}

// IsExpired 캐시가 완전히 만료되었는지 확인
func (c *Cache) IsExpired(entry CacheEntry) bool {
	now := time.Now()

	if c.StaleWhileRevalidate {
		// stale-while-revalidate 사용 시 stale 기간도 고려
		return now.After(entry.StaleUntil)
	}

	// 일반적인 TTL 만료 확인
	expireTime := entry.CachedAt.Add(time.Duration(entry.TTL) * time.Second)
	return now.After(expireTime)
}

// CreateCacheEntry 캐시 항목 생성
func (c *Cache) CreateCacheEntry(ttl int) CacheEntry {
	now := time.Now()
	entry := CacheEntry{
		CachedAt: now,
		TTL:      ttl,
	}

	if c.StaleWhileRevalidate {
		staleMaxAge := c.StaleMaxAge
		if staleMaxAge <= 0 {
			staleMaxAge = 3600 // 기본값 1시간
		}

		// stale 기간은 원본 TTL 만료 후 추가로 설정된 시간
		entry.StaleUntil = now.Add(time.Duration(ttl+staleMaxAge) * time.Second)
	} else {
		// stale-while-revalidate 미사용 시 TTL과 동일
		entry.StaleUntil = now.Add(time.Duration(ttl) * time.Second)
	}

	return entry
}

// ShouldRevalidate 백그라운드에서 재검증해야 하는지 확인
func (c *Cache) ShouldRevalidate(entry CacheEntry) bool {
	if !c.StaleWhileRevalidate {
		return false
	}

	// TTL이 만료되었지만 stale 서빙 중인 경우 재검증 필요
	return !c.IsFresh(entry) && c.IsStale(entry)
}

// GetCacheStrategy 캐시 전략 결정
func (c *Cache) GetCacheStrategy(entry CacheEntry) CacheStrategy {
	if c.IsFresh(entry) {
		return CacheStrategyServe
	}

	if c.IsStale(entry) {
		return CacheStrategyStaleWhileRevalidate
	}

	return CacheStrategyRevalidate
}

// CacheStrategy 캐시 전략 타입
type CacheStrategy int

const (
	CacheStrategyServe                CacheStrategy = iota // 신선한 캐시 서빙
	CacheStrategyStaleWhileRevalidate                      // 만료된 캐시 서빙 + 백그라운드 갱신
	CacheStrategyRevalidate                                // 캐시 재검증 필요
)
