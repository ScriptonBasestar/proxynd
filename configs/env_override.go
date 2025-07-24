package configs

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// EnvOverride 환경 변수 오버라이드 매핑
type EnvOverride struct {
	EnvVar   string
	Path     string
	Type     string
	Required bool
}

// 환경 변수 매핑 정의
var envOverrides = []EnvOverride{
	// 서버 설정
	{EnvVar: "SERVER_HOST", Path: "server.host", Type: "string"},
	{EnvVar: "SERVER_PORT", Path: "server.port", Type: "int"},
	{EnvVar: "SERVER_READ_TIMEOUT", Path: "server.read_timeout", Type: "duration"},
	{EnvVar: "SERVER_WRITE_TIMEOUT", Path: "server.write_timeout", Type: "duration"},
	{EnvVar: "SERVER_IDLE_TIMEOUT", Path: "server.idle_timeout", Type: "duration"},

	// TLS 설정
	{EnvVar: "TLS_ENABLED", Path: "server.tls.enabled", Type: "bool"},
	{EnvVar: "TLS_CERT_FILE", Path: "server.tls.cert_file", Type: "string"},
	{EnvVar: "TLS_KEY_FILE", Path: "server.tls.key_file", Type: "string"},

	// 캐시 설정
	{EnvVar: "CACHE_BACKEND", Path: "cache.backend", Type: "string"},
	{EnvVar: "CACHE_TTL", Path: "cache.ttl", Type: "duration"},
	{EnvVar: "CACHE_MAX_SIZE", Path: "cache.max_size", Type: "string"},
	{EnvVar: "STORAGE_DIR", Path: "cache.file.directory", Type: "string"},

	// S3 캐시 설정
	{EnvVar: "S3_ENDPOINT", Path: "cache.s3.endpoint", Type: "string"},
	{EnvVar: "S3_BUCKET", Path: "cache.s3.bucket", Type: "string"},
	{EnvVar: "S3_REGION", Path: "cache.s3.region", Type: "string"},
	{EnvVar: "AWS_ACCESS_KEY_ID", Path: "cache.s3.access_key_id", Type: "string"},
	{EnvVar: "AWS_SECRET_ACCESS_KEY", Path: "cache.s3.secret_access_key", Type: "string"},

	// Redis 캐시 설정
	{EnvVar: "REDIS_ADDRESS", Path: "cache.redis.address", Type: "string"},
	{EnvVar: "REDIS_PASSWORD", Path: "cache.redis.password", Type: "string"},
	{EnvVar: "REDIS_DB", Path: "cache.redis.db", Type: "int"},

	// 레지스트리 설정
	{EnvVar: "NPM_ENABLED", Path: "registries.npm.enabled", Type: "bool"},
	{EnvVar: "NPM_UPSTREAM", Path: "registries.npm.upstream", Type: "string"},
	{EnvVar: "PYPI_ENABLED", Path: "registries.pypi.enabled", Type: "bool"},
	{EnvVar: "PYPI_UPSTREAM", Path: "registries.pypi.upstream", Type: "string"},
	{EnvVar: "APT_ENABLED", Path: "registries.apt.enabled", Type: "bool"},
	{EnvVar: "DOCKER_ENABLED", Path: "registries.docker.enabled", Type: "bool"},
	{EnvVar: "MAVEN_ENABLED", Path: "registries.maven.enabled", Type: "bool"},

	// 보안 설정
	{EnvVar: "AUTH_ENABLED", Path: "security.authentication.basic_auth.enabled", Type: "bool"},
	{EnvVar: "AUTH_USERS_FILE", Path: "security.authentication.basic_auth.users_file", Type: "string"},
	{EnvVar: "IP_WHITELIST_ENABLED", Path: "security.access_control.ip_whitelist.enabled", Type: "bool"},

	// 로깅 설정
	{EnvVar: "LOG_LEVEL", Path: "logging.level", Type: "string"},
	{EnvVar: "LOG_FORMAT", Path: "logging.format", Type: "string"},
	{EnvVar: "LOG_OUTPUT", Path: "logging.output", Type: "string"},
	{EnvVar: "LOG_FILE", Path: "logging.file.path", Type: "string"},
	{EnvVar: "ACCESS_LOG_ENABLED", Path: "logging.access_log.enabled", Type: "bool"},
	{EnvVar: "ACCESS_LOG_PATH", Path: "logging.access_log.path", Type: "string"},

	// 메트릭 설정
	{EnvVar: "METRICS_ENABLED", Path: "metrics.enabled", Type: "bool"},
	{EnvVar: "METRICS_PATH", Path: "metrics.path", Type: "string"},
	{EnvVar: "METRICS_PORT", Path: "metrics.port", Type: "int"},

	// 검증 설정
	{EnvVar: "VERIFICATION_ENABLED", Path: "verification.strict_mode", Type: "bool"},
	{EnvVar: "VERIFICATION_BLOCK", Path: "verification.block_on_failure", Type: "bool"},
	{EnvVar: "VERIFICATION_ALERT", Path: "verification.alert_on_failure", Type: "bool"},

	// 알림 설정
	{EnvVar: "ALERTS_ENABLED", Path: "alerts.enabled", Type: "bool"},
	{EnvVar: "WEBHOOK_URL", Path: "alerts.channels[1].config.url", Type: "string"},

	// 고급 설정
	{EnvVar: "MAX_CONNECTIONS", Path: "advanced.performance.max_connections", Type: "int"},
	{EnvVar: "CONNECTION_TIMEOUT", Path: "advanced.performance.connection_timeout", Type: "duration"},
	{EnvVar: "RETRY_MAX_ATTEMPTS", Path: "advanced.retry.max_attempts", Type: "int"},
	{EnvVar: "CIRCUIT_BREAKER_ENABLED", Path: "advanced.circuit_breaker.enabled", Type: "bool"},
}

// ApplyEnvironmentOverrides 환경 변수 오버라이드 적용
func ApplyEnvironmentOverrides(config interface{}) error {
	configValue := reflect.ValueOf(config)
	if configValue.Kind() == reflect.Ptr {
		configValue = configValue.Elem()
	}

	for _, override := range envOverrides {
		// 환경 변수 값 확인
		value := os.Getenv(override.EnvVar)
		if value == "" {
			continue
		}

		// 경로로 필드 찾기
		field, err := getFieldByPath(configValue, override.Path)
		if err != nil {
			continue
		}

		// 값 설정
		if err := setFieldValue(field, value, override.Type); err != nil {
			return err
		}
	}

	return nil
}

// getFieldByPath 경로로 필드 찾기
func getFieldByPath(v reflect.Value, path string) (reflect.Value, error) {
	parts := strings.Split(path, ".")
	current := v

	for _, part := range parts {
		// 배열 인덱스 처리
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			fieldName := part[:strings.Index(part, "[")]
			indexStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return reflect.Value{}, err
			}

			// 필드 가져오기
			field := current.FieldByName(cases.Title(language.English).String(strings.ReplaceAll(fieldName, "_", "")))
			if !field.IsValid() {
				return reflect.Value{}, nil
			}

			// 슬라이스 요소 접근
			if field.Kind() == reflect.Slice && field.Len() > index {
				current = field.Index(index)
			} else {
				return reflect.Value{}, nil
			}
		} else {
			// 일반 필드 접근
			fieldName := toCamelCase(part)
			field := current.FieldByName(fieldName)
			if !field.IsValid() {
				return reflect.Value{}, nil
			}
			current = field
		}
	}

	return current, nil
}

// setFieldValue 필드 값 설정
func setFieldValue(field reflect.Value, value, valueType string) error {
	if !field.CanSet() {
		return nil
	}

	switch valueType {
	case "string":
		field.SetString(expandPath(value))
	case "int":
		if i, err := strconv.Atoi(value); err == nil {
			field.SetInt(int64(i))
		}
	case "bool":
		if b, err := strconv.ParseBool(value); err == nil {
			field.SetBool(b)
		}
	case "duration":
		if d, err := time.ParseDuration(value); err == nil {
			field.Set(reflect.ValueOf(d))
		}
	}

	return nil
}

// toCamelCase 스네이크 케이스를 카멜 케이스로 변환
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	caser := cases.Title(language.English)
	for i := range parts {
		parts[i] = caser.String(parts[i])
	}
	return strings.Join(parts, "")
}

// expandPath 경로 확장 (홈 디렉토리, 환경 변수)
func expandPath(path string) string {
	// 홈 디렉토리 확장
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = strings.Replace(path, "~", home, 1)
		}
	}

	// 환경 변수 확장
	path = os.ExpandEnv(path)

	return path
}

// GetEnvironmentOverrides 현재 설정된 환경 변수 오버라이드 목록 반환
func GetEnvironmentOverrides() map[string]string {
	overrides := make(map[string]string)

	for _, override := range envOverrides {
		if value := os.Getenv(override.EnvVar); value != "" {
			overrides[override.EnvVar] = value
		}
	}

	return overrides
}

// PrintEnvironmentVariables 사용 가능한 환경 변수 출력
func PrintEnvironmentVariables() {
	println("Available environment variables for ProxyND:")
	println("===========================================")

	categories := map[string][]EnvOverride{
		"Server":     {},
		"Cache":      {},
		"Registries": {},
		"Security":   {},
		"Logging":    {},
		"Metrics":    {},
		"Advanced":   {},
	}

	// 카테고리별로 분류
	for _, override := range envOverrides {
		switch {
		case strings.HasPrefix(override.EnvVar, "SERVER_") || strings.HasPrefix(override.EnvVar, "TLS_"):
			categories["Server"] = append(categories["Server"], override)
		case strings.HasPrefix(override.EnvVar, "CACHE_") || strings.HasPrefix(override.EnvVar, "STORAGE_") ||
			strings.HasPrefix(override.EnvVar, "S3_") || strings.HasPrefix(override.EnvVar, "REDIS_"):
			categories["Cache"] = append(categories["Cache"], override)
		case strings.Contains(override.EnvVar, "_ENABLED") || strings.Contains(override.EnvVar, "_UPSTREAM"):
			categories["Registries"] = append(categories["Registries"], override)
		case strings.HasPrefix(override.EnvVar, "AUTH_") || strings.HasPrefix(override.EnvVar, "IP_") ||
			strings.HasPrefix(override.EnvVar, "VERIFICATION_"):
			categories["Security"] = append(categories["Security"], override)
		case strings.HasPrefix(override.EnvVar, "LOG_") || strings.HasPrefix(override.EnvVar, "ACCESS_LOG_"):
			categories["Logging"] = append(categories["Logging"], override)
		case strings.HasPrefix(override.EnvVar, "METRICS_") || strings.HasPrefix(override.EnvVar, "ALERTS_"):
			categories["Metrics"] = append(categories["Metrics"], override)
		default:
			categories["Advanced"] = append(categories["Advanced"], override)
		}
	}

	// 카테고리별 출력
	for category, overrides := range categories {
		if len(overrides) > 0 {
			println("\n" + category + ":")
			println(strings.Repeat("-", len(category)+1))
			for _, override := range overrides {
				fmt.Printf("%-30s %s (type: %s)\n", override.EnvVar, override.Path, override.Type)
			}
		}
	}
}
