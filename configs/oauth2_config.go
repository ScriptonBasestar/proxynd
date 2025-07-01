package configs

import (
	"path"
	"proxynd/helpers"
)

// OAuth2Config OAuth2 설정 구조체
type OAuth2Config struct {
	Enabled         bool                         `yaml:"enabled" default:"false"`
	DefaultProvider string                       `yaml:"default_provider" default:"github"`
	Providers       map[string]OAuth2Provider    `yaml:"providers"`
	JWT             JWTConfig                    `yaml:"jwt"`
	UserMapping     UserMappingConfig           `yaml:"user_mapping"`
}

// OAuth2Provider OAuth2 제공자 설정
type OAuth2Provider struct {
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURI  string   `yaml:"redirect_uri"`
	Scopes       []string `yaml:"scopes"`
	
	// 제공자별 엔드포인트 (Generic 제공자용)
	AuthURL     string `yaml:"auth_url,omitempty"`
	TokenURL    string `yaml:"token_url,omitempty"`
	UserInfoURL string `yaml:"user_info_url,omitempty"`
	
	// 추가 설정
	EnablePKCE  bool   `yaml:"enable_pkce" default:"true"`
	UserIDField string `yaml:"user_id_field" default:"id"`
	EmailField  string `yaml:"email_field" default:"email"`
	NameField   string `yaml:"name_field" default:"name"`
}

// JWTConfig JWT 토큰 설정
type JWTConfig struct {
	Secret           string `yaml:"secret"`
	AccessTokenTTL   int    `yaml:"access_token_ttl" default:"3600"`      // 1시간
	RefreshTokenTTL  int    `yaml:"refresh_token_ttl" default:"604800"`   // 7일
	Issuer           string `yaml:"issuer" default:"proxynd"`
	Audience         string `yaml:"audience" default:"proxynd-api"`
	Algorithm        string `yaml:"algorithm" default:"HS256"`
}

// UserMappingConfig 사용자 매핑 설정
type UserMappingConfig struct {
	AutoCreate          bool               `yaml:"auto_create" default:"true"`
	DefaultRole         string             `yaml:"default_role" default:"viewer"`
	AdminUsers          []string           `yaml:"admin_users"`
	AdminOrganizations  []string           `yaml:"admin_organizations"`
	RoleMapping         map[string]string  `yaml:"role_mapping"`
	OrganizationMapping map[string]string  `yaml:"organization_mapping"`
}

// GetDefaultOAuth2Providers 기본 OAuth2 제공자 설정 반환
func GetDefaultOAuth2Providers() map[string]OAuth2Provider {
	return map[string]OAuth2Provider{
		"github": {
			AuthURL:     "https://github.com/login/oauth/authorize",
			TokenURL:    "https://github.com/login/oauth/access_token",
			UserInfoURL: "https://api.github.com/user",
			Scopes:      []string{"user:email"},
			EnablePKCE:  true,
			UserIDField: "id",
			EmailField:  "email",
			NameField:   "name",
		},
		"gitlab": {
			AuthURL:     "https://gitlab.com/oauth/authorize",
			TokenURL:    "https://gitlab.com/oauth/token",
			UserInfoURL: "https://gitlab.com/api/v4/user",
			Scopes:      []string{"read_user"},
			EnablePKCE:  true,
			UserIDField: "id",
			EmailField:  "email",
			NameField:   "name",
		},
		"google": {
			AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:    "https://oauth2.googleapis.com/token",
			UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:      []string{"openid", "email", "profile"},
			EnablePKCE:  true,
			UserIDField: "id",
			EmailField:  "email",
			NameField:   "name",
		},
	}
}

// ValidateProvider OAuth2 제공자 설정 검증
func (p *OAuth2Provider) ValidateProvider() error {
	if p.ClientID == "" {
		return helpers.NewConfigError("client_id is required")
	}
	
	if p.ClientSecret == "" {
		return helpers.NewConfigError("client_secret is required")
	}
	
	if p.RedirectURI == "" {
		return helpers.NewConfigError("redirect_uri is required")
	}
	
	// Generic 제공자의 경우 엔드포인트 URL 필수
	if p.AuthURL != "" || p.TokenURL != "" || p.UserInfoURL != "" {
		if p.AuthURL == "" || p.TokenURL == "" || p.UserInfoURL == "" {
			return helpers.NewConfigError("auth_url, token_url, user_info_url are all required for generic provider")
		}
	}
	
	return nil
}

// IsGenericProvider Generic 제공자인지 확인
func (p *OAuth2Provider) IsGenericProvider() bool {
	return p.AuthURL != "" && p.TokenURL != "" && p.UserInfoURL != ""
}

// GetEffectiveScopes 효과적인 스코프 반환 (기본값 포함)
func (p *OAuth2Provider) GetEffectiveScopes() []string {
	if len(p.Scopes) > 0 {
		return p.Scopes
	}
	
	// 기본 스코프 반환
	return []string{"openid", "email", "profile"}
}

// ValidateJWT JWT 설정 검증
func (j *JWTConfig) ValidateJWT() error {
	if j.Secret == "" {
		return helpers.NewConfigError("jwt.secret is required")
	}
	
	if len(j.Secret) < 32 {
		return helpers.NewConfigError("jwt.secret must be at least 32 characters long")
	}
	
	if j.AccessTokenTTL <= 0 {
		j.AccessTokenTTL = 3600 // 기본값 1시간
	}
	
	if j.RefreshTokenTTL <= 0 {
		j.RefreshTokenTTL = 604800 // 기본값 7일
	}
	
	// 지원하는 알고리즘 검증
	supportedAlgorithms := map[string]bool{
		"HS256": true, "HS384": true, "HS512": true,
		"RS256": true, "RS384": true, "RS512": true,
		"ES256": true, "ES384": true, "ES512": true,
	}
	
	if !supportedAlgorithms[j.Algorithm] {
		return helpers.NewConfigError("unsupported JWT algorithm: " + j.Algorithm)
	}
	
	return nil
}

// ValidateUserMapping 사용자 매핑 설정 검증
func (u *UserMappingConfig) ValidateUserMapping() error {
	validRoles := map[string]bool{
		"admin": true, "maintainer": true, "developer": true, "viewer": true,
	}
	
	if !validRoles[u.DefaultRole] {
		return helpers.NewConfigError("invalid default_role: " + u.DefaultRole)
	}
	
	// 역할 매핑 검증
	for _, role := range u.RoleMapping {
		if !validRoles[role] {
			return helpers.NewConfigError("invalid role in role_mapping: " + role)
		}
	}
	
	return nil
}

// GetUserRole 사용자 역할 결정
func (u *UserMappingConfig) GetUserRole(email string, organizations []string) string {
	// 관리자 사용자 확인
	for _, adminEmail := range u.AdminUsers {
		if email == adminEmail {
			return "admin"
		}
	}
	
	// 관리자 조직 확인
	for _, org := range organizations {
		for _, adminOrg := range u.AdminOrganizations {
			if org == adminOrg {
				return "admin"
			}
		}
	}
	
	// 조직 기반 역할 매핑
	for _, org := range organizations {
		if role, exists := u.OrganizationMapping[org]; exists {
			return role
		}
	}
	
	// 기본 역할 반환
	return u.DefaultRole
}

// Validate OAuth2 설정 전체 검증
func (c *OAuth2Config) Validate() error {
	if !c.Enabled {
		return nil // 비활성화된 경우 검증 생략
	}
	
	// 기본 제공자 확인
	if c.DefaultProvider == "" {
		return helpers.NewConfigError("default_provider is required when OAuth2 is enabled")
	}
	
	// 기본 제공자가 존재하는지 확인
	if _, exists := c.Providers[c.DefaultProvider]; !exists {
		return helpers.NewConfigError("default_provider '" + c.DefaultProvider + "' not found in providers")
	}
	
	// 각 제공자 검증
	for name, provider := range c.Providers {
		if err := provider.ValidateProvider(); err != nil {
			return helpers.NewConfigError("provider '" + name + "': " + err.Error())
		}
	}
	
	// JWT 설정 검증
	if err := c.JWT.ValidateJWT(); err != nil {
		return err
	}
	
	// 사용자 매핑 설정 검증
	if err := c.UserMapping.ValidateUserMapping(); err != nil {
		return err
	}
	
	return nil
}

// MergeWithDefaults 기본값과 병합
func (c *OAuth2Config) MergeWithDefaults() {
	if c.Providers == nil {
		c.Providers = make(map[string]OAuth2Provider)
	}
	
	// 기본 제공자 정보 병합
	defaultProviders := GetDefaultOAuth2Providers()
	
	for name, defaultProvider := range defaultProviders {
		if provider, exists := c.Providers[name]; exists {
			// 기존 설정이 있는 경우 기본값으로 보완
			if provider.AuthURL == "" {
				provider.AuthURL = defaultProvider.AuthURL
			}
			if provider.TokenURL == "" {
				provider.TokenURL = defaultProvider.TokenURL
			}
			if provider.UserInfoURL == "" {
				provider.UserInfoURL = defaultProvider.UserInfoURL
			}
			if len(provider.Scopes) == 0 {
				provider.Scopes = defaultProvider.Scopes
			}
			if provider.UserIDField == "" {
				provider.UserIDField = defaultProvider.UserIDField
			}
			if provider.EmailField == "" {
				provider.EmailField = defaultProvider.EmailField
			}
			if provider.NameField == "" {
				provider.NameField = defaultProvider.NameField
			}
			
			c.Providers[name] = provider
		}
	}
	
	// JWT 기본값 설정
	if c.JWT.Issuer == "" {
		c.JWT.Issuer = "proxynd"
	}
	if c.JWT.Audience == "" {
		c.JWT.Audience = "proxynd-api"
	}
	if c.JWT.Algorithm == "" {
		c.JWT.Algorithm = "HS256"
	}
	if c.JWT.AccessTokenTTL == 0 {
		c.JWT.AccessTokenTTL = 3600
	}
	if c.JWT.RefreshTokenTTL == 0 {
		c.JWT.RefreshTokenTTL = 604800
	}
	
	// 사용자 매핑 기본값 설정
	if c.UserMapping.DefaultRole == "" {
		c.UserMapping.DefaultRole = "viewer"
	}
	if c.UserMapping.RoleMapping == nil {
		c.UserMapping.RoleMapping = make(map[string]string)
	}
	if c.UserMapping.OrganizationMapping == nil {
		c.UserMapping.OrganizationMapping = make(map[string]string)
	}
}

// ConfigExists OAuth2 설정 파일이 존재하는지 확인
func (c *OAuth2Config) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "oauth2.yaml"))
}

// ReadConfig OAuth2 설정 파일 읽기
func (c *OAuth2Config) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	configPath := path.Join(confDir, "oauth2.yaml")
	
	if err := helpers.ReadYaml(configPath, c); err != nil {
		return err
	}
	
	// 기본값과 병합
	c.MergeWithDefaults()
	
	// 설정 검증
	return c.Validate()
}

// GetProvider 특정 제공자 설정 반환
func (c *OAuth2Config) GetProvider(name string) (OAuth2Provider, bool) {
	provider, exists := c.Providers[name]
	return provider, exists
}

// GetDefaultProvider 기본 제공자 설정 반환
func (c *OAuth2Config) GetDefaultProvider() (OAuth2Provider, bool) {
	return c.GetProvider(c.DefaultProvider)
}

// IsEnabled OAuth2가 활성화되어 있는지 확인
func (c *OAuth2Config) IsEnabled() bool {
	return c.Enabled && len(c.Providers) > 0 && c.JWT.Secret != ""
}

// GetEnabledProviders 활성화된 제공자 목록 반환
func (c *OAuth2Config) GetEnabledProviders() []string {
	if !c.Enabled {
		return []string{}
	}
	
	providers := make([]string, 0, len(c.Providers))
	for name, provider := range c.Providers {
		if provider.ClientID != "" && provider.ClientSecret != "" {
			providers = append(providers, name)
		}
	}
	
	return providers
}