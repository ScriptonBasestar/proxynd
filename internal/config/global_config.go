// Package configs provides configuration structures and utilities for ProxyND.
// It includes global configuration, cache settings, authentication options,
// and various proxy-specific configurations.
package config

import (
	"path"
	"strconv"
	"strings"
	"time"

	"proxynd/internal/helpers"
)

// Cache represents the caching configuration for ProxyND.
// It controls TTL values, cache headers handling, and stale-while-revalidate behavior.
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

// UnmarshalYAML implements custom YAML unmarshaling to support duration strings (e.g., "1h", "30m")
func (c *Cache) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Create an auxiliary struct with string fields for duration values
	type cacheAlias struct {
		TTL                  interface{}    `yaml:"ttl,omitempty"`
		PackageTTLs          map[string]int `yaml:"package_ttls,omitempty"`
		PatternTTLs          map[string]int `yaml:"pattern_ttls,omitempty"`
		MetadataTTLs         map[string]int `yaml:"metadata_ttls,omitempty"`
		UseCacheHeaders      bool           `yaml:"use_cache_headers,omitempty"`
		MaxCacheHeaderTTL    interface{}    `yaml:"max_cache_header_ttl,omitempty"`
		MinCacheHeaderTTL    interface{}    `yaml:"min_cache_header_ttl,omitempty"`
		StaleWhileRevalidate bool           `yaml:"stale_while_revalidate,omitempty"`
		StaleMaxAge          interface{}    `yaml:"stale_max_age,omitempty"`
	}

	var aux cacheAlias
	if err := unmarshal(&aux); err != nil {
		return err
	}

	// Parse TTL (duration string or int)
	c.TTL = parseDurationOrInt(aux.TTL, 3600)
	c.MaxCacheHeaderTTL = parseDurationOrInt(aux.MaxCacheHeaderTTL, 86400)
	c.MinCacheHeaderTTL = parseDurationOrInt(aux.MinCacheHeaderTTL, 300)
	c.StaleMaxAge = parseDurationOrInt(aux.StaleMaxAge, 3600)

	// Copy other fields
	c.PackageTTLs = aux.PackageTTLs
	c.PatternTTLs = aux.PatternTTLs
	c.MetadataTTLs = aux.MetadataTTLs
	c.UseCacheHeaders = aux.UseCacheHeaders
	c.StaleWhileRevalidate = aux.StaleWhileRevalidate

	return nil
}

// parseDurationOrInt parses a value that can be either a duration string (e.g., "1h") or an int
func parseDurationOrInt(value interface{}, defaultValue int) int {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case string:
		// Parse duration string like "1h", "30m", "1h30m"
		duration, err := time.ParseDuration(v)
		if err != nil {
			// If not a valid duration, try parsing as a number string
			if intVal, err := strconv.Atoi(v); err == nil {
				return intVal
			}
			return defaultValue
		}
		return int(duration.Seconds())
	case int:
		return v
	case float64:
		// YAML sometimes parses numbers as float64
		return int(v)
	default:
		return defaultValue
	}
}

// GetDefaultPackageTTLs returns default TTL values for each package type.
// These values are used when specific TTLs are not configured.
// Returns a map with package types as keys and TTL in seconds as values.
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

// GetTTLForPackageType returns the TTL for a specific package type.
// It checks custom package TTLs first, then defaults, and finally uses the global TTL.
// If no TTL is configured, it returns 3600 seconds (1 hour) as the default.
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

// GetTTLForPackage returns the TTL for a specific package based on its name and type.
// It supports pattern matching for package names and falls back to package type TTL.
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

// GetTTLForMetadata returns the TTL for metadata files.
// Metadata typically has shorter TTL than packages (1/3 of package TTL by default).
// Minimum TTL for metadata is 300 seconds (5 minutes).
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

// matchesPattern checks if a package name matches the given pattern.
// Supports wildcards: "*" matches any string, "prefix-*", "*-suffix", and "*substring*".
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

// contains checks if a string contains a substring.
// This is a compatibility function for Go versions before 1.18.
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring finds the index of a substring within a string.
// Returns -1 if the substring is not found.
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

// BasicAuthConfig represents HTTP Basic Authentication configuration.
// It contains a map of username/password pairs and an optional realm.
type BasicAuthConfig struct {
	Enabled *bool             `yaml:"enabled,omitempty" default:"true"`
	Users   map[string]string `yaml:"users,omitempty" validate:"dive,keys,min=1,endkeys,min=1"`
	Realm   string            `yaml:"realm,omitempty" default:"Restricted" validate:"min=1,max=100"`
}

// AuthenticationConfig represents the authentication configuration for ProxyND.
// It supports both Basic Authentication and OAuth2.
type AuthenticationConfig struct {
	Enabled   *bool            `yaml:"enabled,omitempty" default:"true"`
	BasicAuth *BasicAuthConfig `yaml:"basic_auth,omitempty"`
	OAuth2    *OAuth2Config    `yaml:"oauth2,omitempty"`
}

// PackageVerificationConfig represents package-type specific verification settings.
type PackageVerificationConfig struct {
	Enabled        bool     `yaml:"enabled,omitempty" default:"true"`
	RequiredHashes []string `yaml:"required_hashes,omitempty"`
	TrustedSources []string `yaml:"trusted_sources,omitempty"`
}

// VerificationConfig represents the package verification configuration.
// It controls signature verification, hash validation, and security policies.
type VerificationConfig struct {
	Enabled        *bool                                `yaml:"enabled,omitempty" default:"true"`
	StrictMode     bool                                 `yaml:"strict_mode,omitempty" default:"true"`
	BlockOnFailure bool                                 `yaml:"block_on_failure,omitempty" default:"true"`
	AlertOnFailure bool                                 `yaml:"alert_on_failure,omitempty" default:"true"`
	PackageTypes   map[string]PackageVerificationConfig `yaml:"package_types,omitempty"`
}

// GlobalConfig represents the global configuration for ProxyND.
// It includes storage paths, cache settings, authentication, and verification configuration.
type GlobalConfig struct {
	StorageDir     string                `yaml:"storage_dir,omitempty" validate:"omitempty,path"`
	ConfigDir      string                `yaml:"config_dir,omitempty" validate:"omitempty,path"`
	Cache          Cache                 `yaml:"cache,omitempty"`
	Authentication *AuthenticationConfig `yaml:"authentication,omitempty"`
	Verification   *VerificationConfig   `yaml:"verification,omitempty"`
	CacheDir       string                `yaml:"cache_dir,omitempty" validate:"omitempty,path"`
	CacheTTL       int                   `yaml:"cache_ttl,omitempty" validate:"min=0,max=604800"`
	MaxCacheSize   int64                 `yaml:"max_cache_size,omitempty" validate:"min=0"`
}

// ConfigExists checks if the global configuration file exists.
// Returns true if the global.yaml file is present in the config directory.
func (cfg *GlobalConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "global.yaml"))
}

// ReadConfig loads the global configuration from global.yaml file.
// It reads the file, unmarshals the YAML content, and validates the configuration.
// Returns an error if the file cannot be read or validation fails.
func (cfg *GlobalConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "global.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the global configuration.
// It checks struct tags, ensures min/max values are consistent,
// and validates nested configurations like OAuth2.
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

// GetTTLFromCacheHeaders calculates TTL from HTTP cache headers.
// It parses Cache-Control and Expires headers to determine cache duration.
// Returns 0 if cache headers are disabled or no valid headers are found.
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

// parseCacheControl parses the Cache-Control header to extract TTL.
// It supports max-age and s-maxage directives, with s-maxage taking precedence.
// Returns the TTL in seconds or 0 if no valid directive is found.
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

// parseExpires parses the Expires header to calculate TTL.
// It supports multiple date formats including RFC1123 and RFC822.
// Returns the TTL in seconds or 0 if the header cannot be parsed.
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

// enforceTTLLimits enforces minimum and maximum TTL limits.
// Default minimum is 300 seconds (5 minutes) and maximum is 86400 seconds (24 hours).
// Returns the adjusted TTL value within the configured limits.
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

// GetDynamicTTL calculates TTL dynamically based on HTTP headers with fallback.
// It first tries to use cache headers, then falls back to package-based TTL.
// This allows for adaptive caching based on upstream server directives.
func (c *Cache) GetDynamicTTL(packageName, packageType, cacheControl, expires string) int {
	// 1. 캐시 헤더 기반 TTL 우선 시도
	if headerTTL := c.GetTTLFromCacheHeaders(cacheControl, expires); headerTTL > 0 {
		return headerTTL
	}

	// 2. 기존 패키지 기반 TTL 폴백
	return c.GetTTLForPackage(packageName, packageType)
}

// CacheEntry represents a cached item with its metadata.
// It tracks when the item was cached, its TTL, and stale expiration time.
type CacheEntry struct {
	CachedAt   time.Time // 캐시된 시간
	TTL        int       // 원본 TTL (초)
	StaleUntil time.Time // stale 만료 시간
}

// IsFresh checks if the cache entry is still fresh (within its TTL).
// Returns true if the current time is before the expiration time.
func (c *Cache) IsFresh(entry CacheEntry) bool {
	now := time.Now()
	expireTime := entry.CachedAt.Add(time.Duration(entry.TTL) * time.Second)
	return now.Before(expireTime)
}

// IsStale checks if the cache entry is expired but can still be served stale.
// This is only applicable when stale-while-revalidate is enabled.
// Returns true if the entry is within the stale serving period.
func (c *Cache) IsStale(entry CacheEntry) bool {
	if !c.StaleWhileRevalidate {
		return false
	}

	now := time.Now()
	expireTime := entry.CachedAt.Add(time.Duration(entry.TTL) * time.Second)

	// TTL은 만료되었지만 stale 기간 내에 있는지 확인
	return now.After(expireTime) && now.Before(entry.StaleUntil)
}

// IsExpired checks if the cache entry is completely expired.
// When stale-while-revalidate is enabled, it checks against the stale deadline.
// Otherwise, it checks against the normal TTL expiration.
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

// CreateCacheEntry creates a new cache entry with the given TTL.
// It calculates both the normal expiration and stale expiration times.
// The stale period is added on top of the normal TTL when stale-while-revalidate is enabled.
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

// ShouldRevalidate checks if the cache entry should be revalidated in the background.
// This is true when the entry is expired but still within the stale serving period.
// Returns false if stale-while-revalidate is not enabled.
func (c *Cache) ShouldRevalidate(entry CacheEntry) bool {
	if !c.StaleWhileRevalidate {
		return false
	}

	// TTL이 만료되었지만 stale 서빙 중인 경우 재검증 필요
	return !c.IsFresh(entry) && c.IsStale(entry)
}

// GetCacheStrategy determines the appropriate caching strategy for an entry.
// It returns CacheStrategyServe for fresh entries, CacheStrategyStaleWhileRevalidate
// for stale but servable entries, or CacheStrategyRevalidate for expired entries.
func (c *Cache) GetCacheStrategy(entry CacheEntry) CacheStrategy {
	if c.IsFresh(entry) {
		return CacheStrategyServe
	}

	if c.IsStale(entry) {
		return CacheStrategyStaleWhileRevalidate
	}

	return CacheStrategyRevalidate
}

// CacheStrategy represents the caching strategy to use for a request.
type CacheStrategy int

const (
	// CacheStrategyServe indicates the cache is fresh and can be served directly
	CacheStrategyServe CacheStrategy = iota
	// CacheStrategyStaleWhileRevalidate indicates the cache is stale but can be served while revalidating in background
	CacheStrategyStaleWhileRevalidate
	// CacheStrategyRevalidate indicates the cache must be revalidated before serving
	CacheStrategyRevalidate
)
