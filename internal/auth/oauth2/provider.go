package oauth2

import (
	"context"
	"net/url"
	"time"
)

// Provider OAuth2 제공자 인터페이스
// 모든 OAuth2 제공자 구현체가 구현해야 하는 공통 인터페이스
type Provider interface {
	// GetName 제공자 이름 반환 (github, gitlab, google 등)
	GetName() string

	// GetAuthURL OAuth2 인증 URL 생성
	// state: CSRF 방지를 위한 상태값
	// codeChallenge: PKCE Code Challenge (선택사항)
	GetAuthURL(state, codeChallenge string) string

	// ExchangeCode 인증 코드를 액세스 토큰으로 교환
	// code: OAuth2 인증 코드
	// codeVerifier: PKCE Code Verifier (선택사항)
	ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error)

	// GetUserInfo 액세스 토큰으로 사용자 정보 조회
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)

	// RefreshToken 리프레시 토큰으로 새 액세스 토큰 발급
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)

	// RevokeToken 토큰 취소 (로그아웃)
	RevokeToken(ctx context.Context, token string) error

	// ValidateToken 토큰 유효성 검증
	ValidateToken(ctx context.Context, token string) (*TokenInfo, error)

	// GetScopes 제공자에서 지원하는 스코프 목록 반환
	GetScopes() []string

	// GetUserOrganizations 사용자가 속한 조직 목록 조회 (GitHub, GitLab만 지원)
	GetUserOrganizations(ctx context.Context, accessToken string) ([]string, error)
}

// TokenResponse OAuth2 토큰 응답
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IDToken      string    `json:"id_token,omitempty"` // OpenID Connect용
	ExpiresAt    time.Time `json:"expires_at"`
}

// UserInfo 사용자 정보
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	Avatar   string `json:"avatar_url,omitempty"`

	// 추가 제공자별 정보
	Company   string    `json:"company,omitempty"`
	Location  string    `json:"location,omitempty"`
	Bio       string    `json:"bio,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// TokenInfo 토큰 정보
type TokenInfo struct {
	Valid     bool      `json:"valid"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Scope     string    `json:"scope,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
}

// ProviderConfig OAuth2 제공자 설정
type ProviderConfig struct {
	Name         string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string

	// 엔드포인트 URL
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	RevokeURL   string

	// 추가 설정
	EnablePKCE    bool
	UserIDField   string
	EmailField    string
	NameField     string
	UsernameField string
	AvatarField   string
}

// ProviderFactory OAuth2 제공자 팩토리 함수 타입
type ProviderFactory func(config ProviderConfig) Provider

// 전역 제공자 팩토리 레지스트리
var providerFactories = make(map[string]ProviderFactory)

// RegisterProvider 제공자 팩토리 등록
func RegisterProvider(name string, factory ProviderFactory) {
	providerFactories[name] = factory
}

// CreateProvider 등록된 팩토리로 제공자 인스턴스 생성
func CreateProvider(name string, config ProviderConfig) Provider {
	factory, exists := providerFactories[name]
	if !exists {
		// Generic 제공자로 폴백
		return NewGenericProvider(config)
	}
	return factory(config)
}

// GetRegisteredProviders 등록된 제공자 목록 반환
func GetRegisteredProviders() []string {
	providers := make([]string, 0, len(providerFactories))
	for name := range providerFactories {
		providers = append(providers, name)
	}
	return providers
}

// GenericProvider Generic OAuth2 제공자 구현
type GenericProvider struct {
	config ProviderConfig
}

// NewGenericProvider Generic 제공자 생성
func NewGenericProvider(config ProviderConfig) *GenericProvider {
	return &GenericProvider{
		config: config,
	}
}

// GetName 제공자 이름 반환
func (g *GenericProvider) GetName() string {
	return g.config.Name
}

// GetAuthURL OAuth2 인증 URL 생성
func (g *GenericProvider) GetAuthURL(state, codeChallenge string) string {
	params := url.Values{}
	params.Set("client_id", g.config.ClientID)
	params.Set("redirect_uri", g.config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("state", state)

	if len(g.config.Scopes) > 0 {
		params.Set("scope", joinScopes(g.config.Scopes))
	}

	// PKCE 지원
	if g.config.EnablePKCE && codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}

	return g.config.AuthURL + "?" + params.Encode()
}

// ExchangeCode 인증 코드를 액세스 토큰으로 교환
func (g *GenericProvider) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	// Generic 구현은 각 제공자별 구체적인 구현에서 오버라이드
	// 이 메서드는 기본 구현을 제공하며, 제공자별로 커스터마이징 가능
	return exchangeCodeGeneric(ctx, g.config, code, codeVerifier)
}

// GetUserInfo 액세스 토큰으로 사용자 정보 조회
func (g *GenericProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	return getUserInfoGeneric(ctx, g.config, accessToken)
}

// RefreshToken 리프레시 토큰으로 새 액세스 토큰 발급
func (g *GenericProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	return refreshTokenGeneric(ctx, g.config, refreshToken)
}

// RevokeToken 토큰 취소
func (g *GenericProvider) RevokeToken(ctx context.Context, token string) error {
	return revokeTokenGeneric(ctx, g.config, token)
}

// ValidateToken 토큰 유효성 검증
func (g *GenericProvider) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	return validateTokenGeneric(ctx, g.config, token)
}

// GetScopes 지원하는 스코프 목록 반환
func (g *GenericProvider) GetScopes() []string {
	return g.config.Scopes
}

// GetUserOrganizations 사용자 조직 목록 조회 (Generic은 지원하지 않음)
func (g *GenericProvider) GetUserOrganizations(_ context.Context, _ string) ([]string, error) {
	return []string{}, nil // Generic 제공자는 조직 정보를 지원하지 않음
}

// joinScopes 스코프 배열을 문자열로 결합
func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}

	result := scopes[0]
	for i := 1; i < len(scopes); i++ {
		result += " " + scopes[i]
	}
	return result
}
