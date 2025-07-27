package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SchemaValidator 설정 스키마 검증기
type SchemaValidator struct {
	rules             map[string]ValidationRule
	customValidators  map[string]ValidatorFunc
	migrationHandlers map[int]MigrationHandler
}

// ValidationRule 검증 규칙 정의
type ValidationRule struct {
	Type        string                    `yaml:"type" json:"type"`             // string, int, bool, url, path, duration, size
	Required    bool                      `yaml:"required" json:"required"`     // 필수 필드 여부
	MinValue    interface{}               `yaml:"min_value" json:"min_value"`   // 최소값
	MaxValue    interface{}               `yaml:"max_value" json:"max_value"`   // 최대값
	Pattern     string                    `yaml:"pattern" json:"pattern"`       // 정규식 패턴
	Enum        []interface{}             `yaml:"enum" json:"enum"`             // 허용된 값 목록
	Default     interface{}               `yaml:"default" json:"default"`       // 기본값
	Deprecated  bool                      `yaml:"deprecated" json:"deprecated"` // 사용 중단 필드
	CustomFunc  string                    `yaml:"custom_func" json:"custom_func"`
	Description string                    `yaml:"description" json:"description"`
	Examples    []interface{}             `yaml:"examples" json:"examples"`
	Nested      map[string]ValidationRule `yaml:"nested" json:"nested"` // 중첩 구조체 규칙
}

// ValidatorFunc 커스텀 검증 함수 타입
type ValidatorFunc func(value interface{}, config *UnifiedConfig) error

// MigrationHandler 설정 마이그레이션 핸들러
type MigrationHandler func(oldConfig map[string]interface{}) (map[string]interface{}, error)

// ValidationResult 검증 결과
type ValidationResult struct {
	Valid       bool                 `json:"valid"`
	Errors      []ValidationError    `json:"errors"`
	Warnings    []ValidationWarning  `json:"warnings"`
	Suggestions []string             `json:"suggestions"`
	Statistics  ValidationStatistics `json:"statistics"`
}

// ValidationWarning 검증 경고
type ValidationWarning struct {
	Field      string      `json:"field"`
	Message    string      `json:"message"`
	Value      interface{} `json:"value"`
	Suggestion string      `json:"suggestion"`
	Severity   string      `json:"severity"` // low, medium, high
	Category   string      `json:"category"` // deprecated, performance, security
}

// ValidationStatistics 검증 통계
type ValidationStatistics struct {
	TotalFields     int `json:"total_fields"`
	ValidFields     int `json:"valid_fields"`
	ErrorFields     int `json:"error_fields"`
	WarningFields   int `json:"warning_fields"`
	DefaultsApplied int `json:"defaults_applied"`
}

// NewSchemaValidator 새 스키마 검증기 생성
func NewSchemaValidator() *SchemaValidator {
	validator := &SchemaValidator{
		rules:             make(map[string]ValidationRule),
		customValidators:  make(map[string]ValidatorFunc),
		migrationHandlers: make(map[int]MigrationHandler),
	}

	// 기본 검증 규칙 등록
	validator.registerDefaultRules()
	// 커스텀 검증 함수 등록
	validator.registerCustomValidators()
	// 마이그레이션 핸들러 등록
	validator.registerMigrationHandlers()

	return validator
}

// registerDefaultRules 기본 검증 규칙 등록
func (sv *SchemaValidator) registerDefaultRules() {
	// 서버 설정 규칙
	sv.rules["server.host"] = ValidationRule{
		Type:        "string",
		Required:    true,
		Default:     "0.0.0.0",
		Pattern:     `^[a-zA-Z0-9\.\-]+$`,
		Description: "서버 바인딩 주소",
		Examples:    []interface{}{"0.0.0.0", "127.0.0.1", "localhost"},
	}

	sv.rules["server.port"] = ValidationRule{
		Type:        "int",
		Required:    true,
		MinValue:    1,
		MaxValue:    65535,
		Default:     8080,
		Description: "서버 포트 번호",
		Examples:    []interface{}{8080, 3000, 9000},
	}

	sv.rules["server.read_timeout"] = ValidationRule{
		Type:        "duration",
		Required:    false,
		Default:     "30s",
		MinValue:    "1s",
		MaxValue:    "300s",
		Description: "읽기 타임아웃",
		Examples:    []interface{}{"30s", "1m", "2m30s"},
	}

	sv.rules["server.write_timeout"] = ValidationRule{
		Type:        "duration",
		Required:    false,
		Default:     "30s",
		MinValue:    "1s",
		MaxValue:    "300s",
		Description: "쓰기 타임아웃",
	}

	sv.rules["server.idle_timeout"] = ValidationRule{
		Type:        "duration",
		Required:    false,
		Default:     "120s",
		MinValue:    "10s",
		MaxValue:    "600s",
		Description: "유휴 연결 타임아웃",
	}

	// TLS 설정 규칙
	sv.rules["server.tls.enabled"] = ValidationRule{
		Type:        "bool",
		Required:    false,
		Default:     false,
		Description: "TLS 활성화 여부",
	}

	sv.rules["server.tls.cert_file"] = ValidationRule{
		Type:        "path",
		Required:    false,
		CustomFunc:  "validate_cert_file",
		Description: "TLS 인증서 파일 경로",
	}

	sv.rules["server.tls.key_file"] = ValidationRule{
		Type:        "path",
		Required:    false,
		CustomFunc:  "validate_key_file",
		Description: "TLS 개인키 파일 경로",
	}

	sv.rules["server.tls.min_version"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Default:     "TLS1.2",
		Enum:        []interface{}{"TLS1.0", "TLS1.1", "TLS1.2", "TLS1.3"},
		Description: "최소 TLS 버전",
	}

	// 캐시 설정 규칙
	sv.rules["cache.backend"] = ValidationRule{
		Type:        "string",
		Required:    true,
		Default:     "file",
		Enum:        []interface{}{"file", "s3", "redis"},
		Description: "캐시 백엔드 타입",
	}

	sv.rules["cache.ttl"] = ValidationRule{
		Type:        "duration",
		Required:    false,
		Default:     "3600s",
		MinValue:    "60s",
		MaxValue:    "86400s",
		Description: "캐시 TTL",
	}

	sv.rules["cache.max_size"] = ValidationRule{
		Type:        "size",
		Required:    false,
		Default:     "10GB",
		MinValue:    "100MB",
		MaxValue:    "1TB",
		Description: "최대 캐시 크기",
	}

	sv.rules["cache.max_items"] = ValidationRule{
		Type:        "int",
		Required:    false,
		MinValue:    1000,
		MaxValue:    10000000,
		Description: "최대 캐시 아이템 수",
	}

	sv.rules["cache.cleanup_interval"] = ValidationRule{
		Type:        "duration",
		Required:    false,
		Default:     "1h",
		MinValue:    "1m",
		MaxValue:    "24h",
		Description: "캐시 정리 주기",
	}

	sv.rules["cache.eviction_policy"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Default:     "lru",
		Enum:        []interface{}{"lru", "lfu", "fifo", "random"},
		Description: "캐시 제거 정책",
	}

	// 파일 캐시 설정
	sv.rules["cache.file.directory"] = ValidationRule{
		Type:        "path",
		Required:    false,
		CustomFunc:  "validate_cache_directory",
		Description: "파일 캐시 디렉토리",
	}

	sv.rules["cache.file.max_file_size"] = ValidationRule{
		Type:        "size",
		Required:    false,
		Default:     "1GB",
		MinValue:    "1MB",
		MaxValue:    "100GB",
		Description: "최대 파일 크기",
	}

	// S3 캐시 설정
	sv.rules["cache.s3.endpoint"] = ValidationRule{
		Type:        "url",
		Required:    false,
		Description: "S3 엔드포인트 URL",
	}

	sv.rules["cache.s3.bucket"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Pattern:     `^[a-z0-9\.\-]+$`,
		Description: "S3 버킷 이름",
	}

	sv.rules["cache.s3.region"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Pattern:     `^[a-z0-9\-]+$`,
		Description: "S3 리전",
	}

	// Redis 캐시 설정
	sv.rules["cache.redis.address"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Pattern:     `^[a-zA-Z0-9\.\-]+:\d+$`,
		Description: "Redis 주소 (host:port)",
		Examples:    []interface{}{"localhost:6379", "redis:6379"},
	}

	sv.rules["cache.redis.db"] = ValidationRule{
		Type:        "int",
		Required:    false,
		MinValue:    0,
		MaxValue:    15,
		Default:     0,
		Description: "Redis 데이터베이스 번호",
	}

	// 로깅 설정
	sv.rules["logging.level"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Default:     "info",
		Enum:        []interface{}{"debug", "info", "warn", "error"},
		Description: "로그 레벨",
	}

	sv.rules["logging.format"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Default:     "json",
		Enum:        []interface{}{"json", "text", "console"},
		Description: "로그 포맷",
	}

	sv.rules["logging.output"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Default:     "stdout",
		Enum:        []interface{}{"stdout", "stderr", "file"},
		Description: "로그 출력 대상",
	}

	// 메트릭 설정
	sv.rules["metrics.enabled"] = ValidationRule{
		Type:        "bool",
		Required:    false,
		Default:     false,
		Description: "메트릭 수집 활성화",
	}

	sv.rules["metrics.path"] = ValidationRule{
		Type:        "string",
		Required:    false,
		Default:     "/metrics",
		Pattern:     `^/[a-zA-Z0-9\-_/]*$`,
		Description: "메트릭 엔드포인트 경로",
	}

	sv.rules["metrics.port"] = ValidationRule{
		Type:        "int",
		Required:    false,
		MinValue:    1,
		MaxValue:    65535,
		Description: "메트릭 전용 포트 (0=메인 포트 사용)",
	}

	// 보안 설정
	sv.rules["security.authentication.basic_auth.enabled"] = ValidationRule{
		Type:        "bool",
		Required:    false,
		Default:     false,
		Description: "Basic Auth 활성화",
	}

	sv.rules["security.access_control.ip_whitelist.enabled"] = ValidationRule{
		Type:        "bool",
		Required:    false,
		Default:     false,
		Description: "IP 화이트리스트 활성화",
	}

	// 성능 설정
	sv.rules["advanced.performance.max_connections"] = ValidationRule{
		Type:        "int",
		Required:    false,
		Default:     1000,
		MinValue:    10,
		MaxValue:    100000,
		Description: "최대 연결 수",
	}

	sv.rules["advanced.performance.connection_timeout"] = ValidationRule{
		Type:        "duration",
		Required:    false,
		Default:     "30s",
		MinValue:    "1s",
		MaxValue:    "300s",
		Description: "연결 타임아웃",
	}

	// 재시도 설정
	sv.rules["advanced.retry.max_attempts"] = ValidationRule{
		Type:        "int",
		Required:    false,
		Default:     3,
		MinValue:    1,
		MaxValue:    10,
		Description: "최대 재시도 횟수",
	}

	sv.rules["advanced.retry.initial_delay"] = ValidationRule{
		Type:        "duration",
		Required:    false,
		Default:     "1s",
		MinValue:    "100ms",
		MaxValue:    "10s",
		Description: "초기 재시도 지연",
	}
}

// registerCustomValidators 커스텀 검증 함수 등록
func (sv *SchemaValidator) registerCustomValidators() {
	// TLS 인증서 파일 검증
	sv.customValidators["validate_cert_file"] = func(value interface{}, config *UnifiedConfig) error {
		if !config.Server.TLS.Enabled {
			return nil // TLS가 비활성화되면 검증하지 않음
		}

		certFile, ok := value.(string)
		if !ok || certFile == "" {
			return fmt.Errorf("TLS가 활성화되었으나 인증서 파일이 지정되지 않음")
		}

		if !filepath.IsAbs(certFile) {
			return fmt.Errorf("인증서 파일 경로는 절대 경로여야 함: %s", certFile)
		}

		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			return fmt.Errorf("인증서 파일을 찾을 수 없음: %s", certFile)
		}

		return nil
	}

	// TLS 개인키 파일 검증
	sv.customValidators["validate_key_file"] = func(value interface{}, config *UnifiedConfig) error {
		if !config.Server.TLS.Enabled {
			return nil
		}

		keyFile, ok := value.(string)
		if !ok || keyFile == "" {
			return fmt.Errorf("TLS가 활성화되었으나 키 파일이 지정되지 않음")
		}

		if !filepath.IsAbs(keyFile) {
			return fmt.Errorf("키 파일 경로는 절대 경로여야 함: %s", keyFile)
		}

		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			return fmt.Errorf("키 파일을 찾을 수 없음: %s", keyFile)
		}

		return nil
	}

	// 캐시 디렉토리 검증
	sv.customValidators["validate_cache_directory"] = func(value interface{}, config *UnifiedConfig) error {
		if config.Cache.Backend != "file" {
			return nil
		}

		directory, ok := value.(string)
		if !ok || directory == "" {
			return fmt.Errorf("파일 캐시가 활성화되었으나 디렉토리가 지정되지 않음")
		}

		// 디렉토리가 존재하지 않으면 생성 시도
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			if err := os.MkdirAll(directory, 0755); err != nil {
				return fmt.Errorf("캐시 디렉토리를 생성할 수 없음: %s (%v)", directory, err)
			}
		}

		// 쓰기 권한 확인
		testFile := filepath.Join(directory, ".write_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return fmt.Errorf("캐시 디렉토리에 쓰기 권한이 없음: %s", directory)
		}
		os.Remove(testFile)

		return nil
	}
}

// registerMigrationHandlers 마이그레이션 핸들러 등록
func (sv *SchemaValidator) registerMigrationHandlers() {
	// v1 -> v2: 로깅 설정 구조 변경
	sv.migrationHandlers[2] = func(oldConfig map[string]interface{}) (map[string]interface{}, error) {
		// 기존 log_level을 logging.level로 변경
		if logLevel, exists := oldConfig["log_level"]; exists {
			if logging, ok := oldConfig["logging"].(map[string]interface{}); ok {
				logging["level"] = logLevel
			} else {
				oldConfig["logging"] = map[string]interface{}{
					"level": logLevel,
				}
			}
			delete(oldConfig, "log_level")
		}

		return oldConfig, nil
	}

	// v2 -> v3: 캐시 설정 통합
	sv.migrationHandlers[3] = func(oldConfig map[string]interface{}) (map[string]interface{}, error) {
		// cache_dir을 cache.file.directory로 변경
		if cacheDir, exists := oldConfig["cache_dir"]; exists {
			cache := make(map[string]interface{})
			if existingCache, ok := oldConfig["cache"].(map[string]interface{}); ok {
				cache = existingCache
			}

			fileCache := make(map[string]interface{})
			if existingFile, ok := cache["file"].(map[string]interface{}); ok {
				fileCache = existingFile
			}

			fileCache["directory"] = cacheDir
			cache["file"] = fileCache
			oldConfig["cache"] = cache
			delete(oldConfig, "cache_dir")
		}

		return oldConfig, nil
	}
}

// Validate 설정 검증 수행
func (sv *SchemaValidator) Validate(config *UnifiedConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Statistics:  ValidationStatistics{},
	}

	// 구조체를 플랫 맵으로 변환
	configMap := sv.structToFlatMap(config, "")

	// 각 필드 검증
	for field, rule := range sv.rules {
		sv.validateField(field, rule, configMap, config, result)
	}

	// 추가 전역 검증
	sv.validateGlobalConstraints(config, result)

	// 통계 계산
	result.Statistics.TotalFields = len(sv.rules)
	result.Statistics.ErrorFields = len(result.Errors)
	result.Statistics.WarningFields = len(result.Warnings)
	result.Statistics.ValidFields = result.Statistics.TotalFields - result.Statistics.ErrorFields
	result.Valid = len(result.Errors) == 0

	return result
}

// validateField 개별 필드 검증
func (sv *SchemaValidator) validateField(field string, rule ValidationRule, configMap map[string]interface{}, config *UnifiedConfig, result *ValidationResult) {
	value, exists := configMap[field]

	// 필수 필드 확인
	if rule.Required && !exists {
		result.Errors = append(result.Errors, ValidationError{
			Field:   field,
			Message: "필수 필드가 누락됨",
			Value:   nil,
		})
		return
	}

	// 기본값 적용
	if !exists && rule.Default != nil {
		value = rule.Default
		result.Statistics.DefaultsApplied++
	}

	if !exists {
		return // 선택적 필드이고 기본값이 없음
	}

	// 사용 중단 필드 경고
	if rule.Deprecated {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:    field,
			Message:  "사용 중단된 설정 필드",
			Value:    value,
			Severity: "medium",
			Category: "deprecated",
		})
	}

	// 타입 검증
	if err := sv.validateType(field, rule.Type, value); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   field,
			Message: err.Error(),
			Value:   value,
		})
		return
	}

	// 범위 검증
	if err := sv.validateRange(field, rule, value); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   field,
			Message: err.Error(),
			Value:   value,
		})
		return
	}

	// 패턴 검증
	if rule.Pattern != "" {
		if err := sv.validatePattern(field, rule.Pattern, value); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Message: err.Error(),
				Value:   value,
			})
			return
		}
	}

	// 열거형 검증
	if len(rule.Enum) > 0 {
		if err := sv.validateEnum(field, rule.Enum, value); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Message: err.Error(),
				Value:   value,
			})
			return
		}
	}

	// 커스텀 검증
	if rule.CustomFunc != "" {
		if validator, exists := sv.customValidators[rule.CustomFunc]; exists {
			if err := validator(value, config); err != nil {
				result.Errors = append(result.Errors, ValidationError{
					Field:   field,
					Message: err.Error(),
					Value:   value,
				})
				return
			}
		}
	}
}

// validateType 타입 검증
func (sv *SchemaValidator) validateType(field, expectedType string, value interface{}) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("문자열 타입이어야 함")
		}
	case "int":
		switch v := value.(type) {
		case int, int32, int64:
			// OK
		case float64:
			if v != float64(int64(v)) {
				return fmt.Errorf("정수 타입이어야 함")
			}
		default:
			return fmt.Errorf("정수 타입이어야 함")
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("불린 타입이어야 함")
		}
	case "duration":
		if str, ok := value.(string); ok {
			if _, err := time.ParseDuration(str); err != nil {
				return fmt.Errorf("유효한 시간 형식이어야 함 (예: 30s, 5m, 1h)")
			}
		} else {
			return fmt.Errorf("시간 형식 문자열이어야 함")
		}
	case "size":
		if str, ok := value.(string); ok {
			if _, err := parseSize(str); err != nil {
				return fmt.Errorf("유효한 크기 형식이어야 함 (예: 1MB, 5GB, 100KB)")
			}
		} else {
			return fmt.Errorf("크기 형식 문자열이어야 함")
		}
	case "url":
		if str, ok := value.(string); ok {
			if _, err := url.Parse(str); err != nil {
				return fmt.Errorf("유효한 URL 형식이어야 함")
			}
		} else {
			return fmt.Errorf("URL 문자열이어야 함")
		}
	case "path":
		if str, ok := value.(string); ok {
			if !filepath.IsAbs(str) && !strings.HasPrefix(str, "./") && !strings.HasPrefix(str, "../") {
				return fmt.Errorf("유효한 파일 경로여야 함")
			}
		} else {
			return fmt.Errorf("경로 문자열이어야 함")
		}
	}

	return nil
}

// validateRange 범위 검증
func (sv *SchemaValidator) validateRange(field string, rule ValidationRule, value interface{}) error {
	if rule.MinValue != nil {
		if err := sv.compareValues(value, rule.MinValue, ">="); err != nil {
			return fmt.Errorf("최소값 %v보다 크거나 같아야 함", rule.MinValue)
		}
	}

	if rule.MaxValue != nil {
		if err := sv.compareValues(value, rule.MaxValue, "<="); err != nil {
			return fmt.Errorf("최대값 %v보다 작거나 같아야 함", rule.MaxValue)
		}
	}

	return nil
}

// validatePattern 패턴 검증
func (sv *SchemaValidator) validatePattern(field, pattern string, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return nil // 문자열이 아니면 패턴 검증 생략
	}

	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		return fmt.Errorf("정규식 패턴 오류: %v", err)
	}

	if !matched {
		return fmt.Errorf("패턴 %s와 일치하지 않음", pattern)
	}

	return nil
}

// validateEnum 열거형 검증
func (sv *SchemaValidator) validateEnum(field string, enum []interface{}, value interface{}) error {
	for _, allowed := range enum {
		if reflect.DeepEqual(value, allowed) {
			return nil
		}
	}

	return fmt.Errorf("허용된 값 중 하나여야 함: %v", enum)
}

// validateGlobalConstraints 전역 제약 조건 검증
func (sv *SchemaValidator) validateGlobalConstraints(config *UnifiedConfig, result *ValidationResult) {
	// 메트릭 포트와 서버 포트 중복 확인
	if config.Metrics.Enabled && config.Metrics.Port != 0 && config.Metrics.Port == config.Server.Port {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "metrics.port",
			Message: "메트릭 포트가 서버 포트와 같을 수 없음",
			Value:   config.Metrics.Port,
		})
	}

	// TLS 설정 일관성 확인
	if config.Server.TLS.Enabled {
		if config.Server.TLS.CertFile == "" || config.Server.TLS.KeyFile == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "server.tls",
				Message: "TLS가 활성화되었으나 인증서 또는 키 파일이 지정되지 않음",
				Value:   config.Server.TLS,
			})
		}
	}

	// 캐시 백엔드별 필수 설정 확인
	switch config.Cache.Backend {
	case "s3":
		if config.Cache.S3.Bucket == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "cache.s3.bucket",
				Message: "S3 캐시 백엔드가 선택되었으나 버킷이 지정되지 않음",
				Value:   config.Cache.S3.Bucket,
			})
		}
	case "redis":
		if config.Cache.Redis.Address == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "cache.redis.address",
				Message: "Redis 캐시 백엔드가 선택되었으나 주소가 지정되지 않음",
				Value:   config.Cache.Redis.Address,
			})
		}
	}

	// 성능 경고
	if config.Advanced.Performance.MaxConnections > 10000 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:      "advanced.performance.max_connections",
			Message:    "매우 높은 최대 연결 수가 설정됨",
			Value:      config.Advanced.Performance.MaxConnections,
			Suggestion: "시스템 리소스를 고려하여 적절한 값으로 조정하세요",
			Severity:   "medium",
			Category:   "performance",
		})
	}

	// 보안 권장사항
	if !config.Server.TLS.Enabled {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:      "server.tls.enabled",
			Message:    "TLS가 비활성화되어 있음",
			Value:      config.Server.TLS.Enabled,
			Suggestion: "프로덕션 환경에서는 TLS를 활성화하는 것을 권장합니다",
			Severity:   "high",
			Category:   "security",
		})
	}

	if config.Security.Authentication.BasicAuth.Enabled == nil || !*config.Security.Authentication.BasicAuth.Enabled {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:      "security.authentication.basic_auth.enabled",
			Message:    "인증이 비활성화되어 있음",
			Value:      false,
			Suggestion: "인증을 활성화하여 보안을 강화하세요",
			Severity:   "high",
			Category:   "security",
		})
	}
}

// compareValues 값 비교
func (sv *SchemaValidator) compareValues(value, target interface{}, operator string) error {
	// 숫자 비교
	v1, v2, err := sv.convertToFloat64(value, target)
	if err == nil {
		switch operator {
		case ">=":
			if v1 >= v2 {
				return nil
			}
		case "<=":
			if v1 <= v2 {
				return nil
			}
		case ">":
			if v1 > v2 {
				return nil
			}
		case "<":
			if v1 < v2 {
				return nil
			}
		}
		return fmt.Errorf("조건 %s를 만족하지 않음", operator)
	}

	// Duration 비교
	if d1, d2, err := sv.convertToDuration(value, target); err == nil {
		switch operator {
		case ">=":
			if d1 >= d2 {
				return nil
			}
		case "<=":
			if d1 <= d2 {
				return nil
			}
		}
		return fmt.Errorf("시간 조건 %s를 만족하지 않음", operator)
	}

	// Size 비교
	if s1, s2, err := sv.convertToSize(value, target); err == nil {
		switch operator {
		case ">=":
			if s1 >= s2 {
				return nil
			}
		case "<=":
			if s1 <= s2 {
				return nil
			}
		}
		return fmt.Errorf("크기 조건 %s를 만족하지 않음", operator)
	}

	return fmt.Errorf("비교할 수 없는 타입")
}

// convertToFloat64 float64로 변환
func (sv *SchemaValidator) convertToFloat64(v1, v2 interface{}) (float64, float64, error) {
	f1, err1 := sv.toFloat64(v1)
	f2, err2 := sv.toFloat64(v2)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("숫자 변환 실패")
	}
	return f1, f2, nil
}

// toFloat64 interface{}를 float64로 변환
func (sv *SchemaValidator) toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("float64로 변환할 수 없는 타입")
	}
}

// convertToDuration Duration으로 변환
func (sv *SchemaValidator) convertToDuration(v1, v2 interface{}) (time.Duration, time.Duration, error) {
	d1, err1 := sv.toDuration(v1)
	d2, err2 := sv.toDuration(v2)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("Duration 변환 실패")
	}
	return d1, d2, nil
}

// toDuration interface{}를 Duration으로 변환
func (sv *SchemaValidator) toDuration(v interface{}) (time.Duration, error) {
	if str, ok := v.(string); ok {
		return time.ParseDuration(str)
	}
	return 0, fmt.Errorf("Duration으로 변환할 수 없는 타입")
}

// convertToSize 크기로 변환
func (sv *SchemaValidator) convertToSize(v1, v2 interface{}) (int64, int64, error) {
	s1, err1 := sv.toSize(v1)
	s2, err2 := sv.toSize(v2)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("크기 변환 실패")
	}
	return s1, s2, nil
}

// toSize interface{}를 크기(바이트)로 변환
func (sv *SchemaValidator) toSize(v interface{}) (int64, error) {
	if str, ok := v.(string); ok {
		return parseSize(str)
	}
	return 0, fmt.Errorf("크기로 변환할 수 없는 타입")
}

// parseSize 크기 문자열을 바이트로 변환
func parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}
	
	// 숫자와 단위 분리
	var number float64
	var unit string
	
	// 정규식을 사용하여 숫자와 단위 분리
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGT]?B?)$`)
	matches := re.FindStringSubmatch(strings.ToUpper(sizeStr))
	
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}
	
	var err error
	number, err = strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in size: %s", matches[1])
	}
	
	unit = matches[2]
	if unit == "" || unit == "B" {
		return int64(number), nil
	}
	
	// 단위별 배수
	multiplier := map[string]int64{
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}
	
	if mult, exists := multiplier[unit]; exists {
		return int64(number * float64(mult)), nil
	}
	
	return 0, fmt.Errorf("unknown unit: %s", unit)
}

// structToFlatMap 구조체를 플랫 맵으로 변환
func (sv *SchemaValidator) structToFlatMap(obj interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// YAML 태그에서 필드명 추출
		tag := fieldType.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}

		fieldName := strings.Split(tag, ",")[0]
		fullName := fieldName
		if prefix != "" {
			fullName = prefix + "." + fieldName
		}

		if field.Kind() == reflect.Struct {
			// 중첩 구조체인 경우 재귀적으로 처리
			nested := sv.structToFlatMap(field.Interface(), fullName)
			for k, v := range nested {
				result[k] = v
			}
		} else {
			result[fullName] = field.Interface()
		}
	}

	return result
}

// MigrateConfig 설정 마이그레이션
func (sv *SchemaValidator) MigrateConfig(configData map[string]interface{}, targetVersion int) (map[string]interface{}, error) {
	currentVersion := 1
	if version, exists := configData["version"]; exists {
		if v, ok := version.(int); ok {
			currentVersion = v
		}
	}

	if currentVersion >= targetVersion {
		return configData, nil
	}

	result := configData
	for version := currentVersion + 1; version <= targetVersion; version++ {
		if handler, exists := sv.migrationHandlers[version]; exists {
			var err error
			result, err = handler(result)
			if err != nil {
				return nil, fmt.Errorf("마이그레이션 v%d 실패: %w", version, err)
			}
		}
	}

	result["version"] = targetVersion
	return result, nil
}

// ValidateIP IP 주소 검증 헬퍼
func ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("유효하지 않은 IP 주소: %s", ip)
	}
	return nil
}

// ValidateCIDR CIDR 블록 검증 헬퍼
func ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("유효하지 않은 CIDR 블록: %s", cidr)
	}
	return nil
}

// ValidateURL URL 검증 헬퍼
func ValidateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("유효하지 않은 URL: %s", rawURL)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL 스키마가 누락됨: %s", rawURL)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL 호스트가 누락됨: %s", rawURL)
	}

	return nil
}

// GenerateSchema 설정 스키마 JSON 생성
func (sv *SchemaValidator) GenerateSchema() ([]byte, error) {
	schema := map[string]interface{}{
		"$schema":    "http://json-schema.org/draft-07/schema#",
		"title":      "ProxyND Configuration Schema",
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   make([]string, 0),
	}

	properties := schema["properties"].(map[string]interface{})
	required := &[]string{}

	for field, rule := range sv.rules {
		if rule.Required {
			*required = append(*required, field)
		}

		fieldSchema := map[string]interface{}{
			"type":        rule.Type,
			"description": rule.Description,
		}

		if rule.Default != nil {
			fieldSchema["default"] = rule.Default
		}

		if len(rule.Enum) > 0 {
			fieldSchema["enum"] = rule.Enum
		}

		if rule.Pattern != "" {
			fieldSchema["pattern"] = rule.Pattern
		}

		if len(rule.Examples) > 0 {
			fieldSchema["examples"] = rule.Examples
		}

		properties[field] = fieldSchema
	}

	schema["required"] = *required
	return json.MarshalIndent(schema, "", "  ")
}
