package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GoogleProvider Google OAuth2 제공자 구현
type GoogleProvider struct {
	*GenericProvider
}

// GoogleUser Google 사용자 정보 구조체
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	VerifiedEmail bool   `json:"verified_email"`

	// Google+ 필드 (선택사항)
	Gender   string `json:"gender,omitempty"`
	Birthday string `json:"birthday,omitempty"`
	Link     string `json:"link,omitempty"`
}

// GoogleTokenInfo Google 토큰 정보 구조체
type GoogleTokenInfo struct {
	Audience  string `json:"aud"`
	UserID    string `json:"user_id"`
	Scope     string `json:"scope"`
	ExpiresIn int    `json:"expires_in"`
	Email     string `json:"email"`

	// ID Token 전용 필드
	Issuer         string `json:"iss,omitempty"`
	Subject        string `json:"sub,omitempty"`
	EmailVerified  bool   `json:"email_verified,omitempty"`
	Name           string `json:"name,omitempty"`
	Picture        string `json:"picture,omitempty"`
	GivenName      string `json:"given_name,omitempty"`
	FamilyName     string `json:"family_name,omitempty"`
	Locale         string `json:"locale,omitempty"`
	IssuedAt       int64  `json:"iat,omitempty"`
	ExpirationTime int64  `json:"exp,omitempty"`
	AuthTime       int64  `json:"auth_time,omitempty"`
	Nonce          string `json:"nonce,omitempty"`
	HostedDomain   string `json:"hd,omitempty"`
}

// NewGoogleProvider Google 제공자 생성
func NewGoogleProvider(config ProviderConfig) Provider {
	// Google 기본 설정 적용
	if config.AuthURL == "" {
		config.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://oauth2.googleapis.com/token"
	}
	if config.UserInfoURL == "" {
		config.UserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}
	if config.RevokeURL == "" {
		config.RevokeURL = "https://oauth2.googleapis.com/revoke"
	}
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "email", "profile"}
	}
	if config.UserIDField == "" {
		config.UserIDField = "id"
	}
	if config.EmailField == "" {
		config.EmailField = "email"
	}
	if config.NameField == "" {
		config.NameField = "name"
	}
	if config.AvatarField == "" {
		config.AvatarField = "picture"
	}

	return &GoogleProvider{
		GenericProvider: NewGenericProvider(config),
	}
}

// GetUserInfo Google 사용자 정보 조회
func (g *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := CreateAPIRequest(ctx, "GET", g.config.UserInfoURL, nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var googleUser GoogleUser
	if err := ParseAPIResponse(resp, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse Google user info: %w", err)
	}

	userInfo := &UserInfo{
		ID:     googleUser.ID,
		Email:  googleUser.Email,
		Name:   googleUser.Name,
		Avatar: googleUser.Picture,
	}

	// 이름이 없는 경우 given_name과 family_name 조합
	if userInfo.Name == "" {
		if googleUser.GivenName != "" || googleUser.FamilyName != "" {
			userInfo.Name = strings.TrimSpace(googleUser.GivenName + " " + googleUser.FamilyName)
		}
	}

	return userInfo, nil
}

// GetUserOrganizations Google은 조직 개념이 없으므로 빈 배열 반환
func (g *GoogleProvider) GetUserOrganizations(ctx context.Context, accessToken string) ([]string, error) {
	// Google은 GitHub/GitLab과 달리 조직 개념이 없음
	// G Suite/Google Workspace 도메인 정보는 ID Token에서만 확인 가능
	return []string{}, nil
}

// ValidateToken Google 토큰 유효성 검증
func (g *GoogleProvider) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	// Google OAuth2 토큰 정보 엔드포인트 사용
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?access_token=%s", token)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &TokenInfo{Valid: false}, nil
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return &TokenInfo{Valid: false}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &TokenInfo{Valid: false}, nil
	}

	var googleTokenInfo GoogleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleTokenInfo); err != nil {
		return &TokenInfo{Valid: false}, nil
	}

	// 토큰이 현재 애플리케이션을 대상으로 하는지 확인
	if googleTokenInfo.Audience != g.config.ClientID {
		return &TokenInfo{Valid: false}, nil
	}

	tokenInfo := &TokenInfo{
		Valid:  true,
		UserID: googleTokenInfo.UserID,
		Scope:  googleTokenInfo.Scope,
	}

	// 만료 시간 설정
	if googleTokenInfo.ExpiresIn > 0 {
		tokenInfo.ExpiresAt = time.Now().Add(time.Duration(googleTokenInfo.ExpiresIn) * time.Second)
	}

	return tokenInfo, nil
}

// RevokeToken Google 토큰 취소
func (g *GoogleProvider) RevokeToken(ctx context.Context, token string) error {
	if g.config.RevokeURL == "" {
		return nil
	}

	// Google은 GET 요청으로 토큰 취소
	url := fmt.Sprintf("%s?token=%s", g.config.RevokeURL, token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke Google token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google token revocation failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetGoogleProfile Google+ API를 통한 확장 프로필 정보 조회
func (g *GoogleProvider) GetGoogleProfile(ctx context.Context, accessToken string) (*GoogleUser, error) {
	// Google+ API는 2019년에 종료되었으므로 People API 사용
	url := "https://people.googleapis.com/v1/people/me?personFields=names,emailAddresses,photos,locales,genders,birthdays"

	req, err := CreateAPIRequest(ctx, "GET", url, nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	var profileResp map[string]interface{}
	if err := ParseAPIResponse(resp, &profileResp); err != nil {
		return nil, fmt.Errorf("failed to parse Google profile: %w", err)
	}

	// People API 응답을 GoogleUser 구조체로 변환
	googleUser := &GoogleUser{}

	if names, ok := profileResp["names"].([]interface{}); ok && len(names) > 0 {
		if name, ok := names[0].(map[string]interface{}); ok {
			if displayName, ok := name["displayName"].(string); ok {
				googleUser.Name = displayName
			}
			if givenName, ok := name["givenName"].(string); ok {
				googleUser.GivenName = givenName
			}
			if familyName, ok := name["familyName"].(string); ok {
				googleUser.FamilyName = familyName
			}
		}
	}

	if emails, ok := profileResp["emailAddresses"].([]interface{}); ok && len(emails) > 0 {
		if email, ok := emails[0].(map[string]interface{}); ok {
			if value, ok := email["value"].(string); ok {
				googleUser.Email = value
			}
		}
	}

	if photos, ok := profileResp["photos"].([]interface{}); ok && len(photos) > 0 {
		if photo, ok := photos[0].(map[string]interface{}); ok {
			if url, ok := photo["url"].(string); ok {
				googleUser.Picture = url
			}
		}
	}

	if locales, ok := profileResp["locales"].([]interface{}); ok && len(locales) > 0 {
		if locale, ok := locales[0].(map[string]interface{}); ok {
			if value, ok := locale["value"].(string); ok {
				googleUser.Locale = value
			}
		}
	}

	return googleUser, nil
}

// VerifyIDToken Google ID 토큰 검증 (OpenID Connect)
func (g *GoogleProvider) VerifyIDToken(ctx context.Context, idToken string) (*GoogleTokenInfo, error) {
	// Google의 공개 키로 ID 토큰 검증
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ID token verification request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ID token verification failed with status %d", resp.StatusCode)
	}

	var tokenInfo GoogleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode ID token info: %w", err)
	}

	// 기본 검증
	if tokenInfo.Audience != g.config.ClientID {
		return nil, fmt.Errorf("ID token audience mismatch")
	}

	if tokenInfo.Issuer != "https://accounts.google.com" && tokenInfo.Issuer != "accounts.google.com" {
		return nil, fmt.Errorf("ID token issuer mismatch")
	}

	// 토큰 만료 확인
	if tokenInfo.ExpirationTime > 0 && time.Now().Unix() > tokenInfo.ExpirationTime {
		return nil, fmt.Errorf("ID token has expired")
	}

	return &tokenInfo, nil
}

// GetHostedDomain G Suite/Google Workspace 도메인 정보 조회
func (g *GoogleProvider) GetHostedDomain(ctx context.Context, idToken string) (string, error) {
	tokenInfo, err := g.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", err
	}

	return tokenInfo.HostedDomain, nil
}

// CheckGSuiteUser G Suite/Google Workspace 사용자인지 확인
func (g *GoogleProvider) CheckGSuiteUser(ctx context.Context, idToken string) (bool, error) {
	domain, err := g.GetHostedDomain(ctx, idToken)
	if err != nil {
		return false, err
	}

	return domain != "", nil
}

// GetUserDomains 사용자의 도메인 정보 조회 (조직 매핑용)
func (g *GoogleProvider) GetUserDomains(ctx context.Context, idToken string) ([]string, error) {
	domain, err := g.GetHostedDomain(ctx, idToken)
	if err != nil {
		return []string{}, nil // 에러가 있어도 빈 배열 반환
	}

	if domain == "" {
		return []string{}, nil
	}

	return []string{domain}, nil
}

// init 함수에서 Google 제공자 등록
func init() {
	RegisterProvider("google", NewGoogleProvider)
}
