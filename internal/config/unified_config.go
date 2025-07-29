package config

// UnifiedConfig 통합 설정 구조체 (Deprecated: RootConfig 사용을 권장합니다)
// 기존 코드와의 호환성을 위해 RootConfig의 별칭으로 유지
type UnifiedConfig = RootConfig

// UnifiedSecurityConfig 보안 설정 (Deprecated: SecuritySettings 사용을 권장합니다)
type UnifiedSecurityConfig = SecuritySettings

// UnifiedLoggingConfig 로깅 설정 (Deprecated: LoggingSettings 사용을 권장합니다)
type UnifiedLoggingConfig = LoggingSettings

// UnifiedPerformanceConfig 성능 설정 (Deprecated: SimplePerformanceConfig 사용을 권장합니다)
type UnifiedPerformanceConfig = SimplePerformanceConfig

// CacheConfig 캐시 설정 (Deprecated: CacheSettings 사용을 권장합니다)
type CacheConfig = CacheSettings

// LoadConfig 설정 파일 로드 (Deprecated: LoadRootConfig 사용을 권장합니다)
var LoadConfig = LoadRootConfig

// ValidateConfigEnhanced 전역 설정 검증 함수 (Deprecated: ValidateRootConfig 사용을 권장합니다)
func ValidateConfigEnhanced(config *UnifiedConfig) []ValidationError {
	return ValidateRootConfig(config)
}
