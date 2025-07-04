package configs

import (
	"path"

	"proxynd/helpers"
)

// ApkProxy APK 프록시 서버 정보
type ApkProxy struct {
	Name string `yaml:"name"`
	Url  string `yaml:"url"`
}

// ApkVerificationConfig APK 서명 검증 설정
type ApkVerificationConfig struct {
	Enabled        bool   `yaml:"enabled"`         // 서명 검증 활성화 여부
	KeyDirectory   string `yaml:"key_directory"`   // 신뢰할 수 있는 키 디렉토리
	FailOnInvalid  bool   `yaml:"fail_on_invalid"` // 서명 검증 실패 시 요청 차단 여부
	CacheValidated bool   `yaml:"cache_validated"` // 검증된 패키지만 캐시 여부
}

// ApkMirrorSelectionConfig APK 미러 선택 설정
type ApkMirrorSelectionConfig struct {
	Enabled             bool     `yaml:"enabled"`               // 미러 자동 선택 활성화
	HealthCheckInterval string   `yaml:"health_check_interval"` // 헬스체크 간격 (예: "5m")
	HealthCheckTimeout  string   `yaml:"health_check_timeout"`  // 헬스체크 타임아웃 (예: "10s")
	PreferredRegions    []string `yaml:"preferred_regions"`     // 선호하는 지역
	FallbackToGlobal    bool     `yaml:"fallback_to_global"`    // 글로벌 미러로 폴백 여부
	MaxErrorCount       int      `yaml:"max_error_count"`       // 최대 에러 허용 횟수
	RegionDetectionMode string   `yaml:"region_detection_mode"` // auto, manual, disabled
}

// ApkProxyConfig APK 프록시 설정 구조체
type ApkProxyConfig struct {
	Path            string                   `yaml:"path"`
	UseCache        bool                     `yaml:"use_cache"`
	Proxies         []ApkProxy               `yaml:"proxies"`
	Verification    ApkVerificationConfig    `yaml:"verification"`
	MirrorSelection ApkMirrorSelectionConfig `yaml:"mirror_selection"`
}

// ConfigExists APK 프록시 설정 파일 존재 여부 확인
func (a *ApkProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "apk-proxy.yaml"))
}

// ReadConfig APK 프록시 설정 파일 읽기
func (a *ApkProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "apk-proxy.yaml"), a)
}
