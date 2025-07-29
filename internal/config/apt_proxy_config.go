package config

// AptProxy는 APT 미러 정보를 나타냅니다
type AptProxy struct {
	Name string `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL  string `yaml:"url,omitempty" validate:"required,url"`
}

// AptProxyConfig는 이전 버전과의 호환성을 위한 별칭입니다
// Deprecated: AptProxySettings를 사용하세요
type AptProxyConfig = AptProxySettings
