package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GitHubProvider GitHub OAuth2 제공자 구현
type GitHubProvider struct {
	*GenericProvider
}

// GitHubUser GitHub 사용자 정보 구조체
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Company   string `json:"company"`
	Location  string `json:"location"`
	Bio       string `json:"bio"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GitHubOrganization GitHub 조직 정보 구조체
type GitHubOrganization struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	AvatarURL   string `json:"avatar_url"`
}

// GitHubEmails GitHub 이메일 정보 구조체
type GitHubEmails struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// NewGitHubProvider GitHub 제공자 생성
func NewGitHubProvider(config ProviderConfig) Provider {
	// GitHub 기본 설정 적용
	if config.AuthURL == "" {
		config.AuthURL = "https://github.com/login/oauth/authorize"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://github.com/login/oauth/access_token"
	}
	if config.UserInfoURL == "" {
		config.UserInfoURL = "https://api.github.com/user"
	}
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"user:email"}
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
	if config.UsernameField == "" {
		config.UsernameField = "login"
	}
	if config.AvatarField == "" {
		config.AvatarField = "avatar_url"
	}
	
	return &GitHubProvider{
		GenericProvider: NewGenericProvider(config),
	}
}

// GetUserInfo GitHub 사용자 정보 조회 (이메일 정보 포함)
func (g *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// 기본 사용자 정보 조회
	req, err := CreateAPIRequest(ctx, "GET", g.config.UserInfoURL, nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	
	var githubUser GitHubUser
	if err := ParseAPIResponse(resp, &githubUser); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub user info: %w", err)
	}
	
	userInfo := &UserInfo{
		ID:       strconv.Itoa(githubUser.ID),
		Username: githubUser.Login,
		Name:     githubUser.Name,
		Avatar:   githubUser.AvatarURL,
		Company:  githubUser.Company,
		Location: githubUser.Location,
		Bio:      githubUser.Bio,
	}
	
	// 시간 정보 파싱
	if createdAt, ok := parseGitHubTime(githubUser.CreatedAt); ok {
		if t, ok := createdAt.(time.Time); ok {
			userInfo.CreatedAt = t
		}
	}
	if updatedAt, ok := parseGitHubTime(githubUser.UpdatedAt); ok {
		if t, ok := updatedAt.(time.Time); ok {
			userInfo.UpdatedAt = t
		}
	}
	
	// 이메일 정보가 없는 경우 별도 API 호출
	if githubUser.Email == "" {
		email, err := g.getPrimaryEmail(ctx, accessToken)
		if err == nil && email != "" {
			userInfo.Email = email
		}
	} else {
		userInfo.Email = githubUser.Email
	}
	
	return userInfo, nil
}

// GetUserOrganizations GitHub 사용자 조직 목록 조회
func (g *GitHubProvider) GetUserOrganizations(ctx context.Context, accessToken string) ([]string, error) {
	req, err := CreateAPIRequest(ctx, "GET", "https://api.github.com/user/orgs", nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create organizations request: %w", err)
	}
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}
	
	var organizations []GitHubOrganization
	if err := ParseAPIResponse(resp, &organizations); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub organizations: %w", err)
	}
	
	orgNames := make([]string, len(organizations))
	for i, org := range organizations {
		orgNames[i] = org.Login
	}
	
	return orgNames, nil
}

// ValidateToken GitHub 토큰 유효성 검증
func (g *GitHubProvider) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	req, err := CreateAPIRequest(ctx, "GET", g.config.UserInfoURL, nil, token)
	if err != nil {
		return &TokenInfo{Valid: false}, nil
	}
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return &TokenInfo{Valid: false}, nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return &TokenInfo{Valid: false}, nil
	}
	
	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return &TokenInfo{Valid: false}, nil
	}
	
	return &TokenInfo{
		Valid:  true,
		UserID: strconv.Itoa(githubUser.ID),
		Scope:  strings.Join(g.config.Scopes, " "),
	}, nil
}

// RevokeToken GitHub 토큰 취소
func (g *GitHubProvider) RevokeToken(ctx context.Context, token string) error {
	// GitHub는 특별한 토큰 취소 방식을 사용
	req, err := http.NewRequestWithContext(ctx, "DELETE", 
		fmt.Sprintf("https://api.github.com/applications/%s/grant", g.config.ClientID), nil)
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}
	
	// Basic Auth 사용 (Client ID + Client Secret)
	req.SetBasicAuth(g.config.ClientID, g.config.ClientSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "token "+token)
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke GitHub token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("GitHub token revocation failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// getPrimaryEmail GitHub에서 주요 이메일 주소 조회
func (g *GitHubProvider) getPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := CreateAPIRequest(ctx, "GET", "https://api.github.com/user/emails", nil, accessToken)
	if err != nil {
		return "", err
	}
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	
	var emails []GitHubEmails
	if err := ParseAPIResponse(resp, &emails); err != nil {
		return "", err
	}
	
	// 주요 이메일 찾기
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}
	
	// 주요 이메일이 없으면 첫 번째 검증된 이메일 반환
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}
	
	return "", fmt.Errorf("no verified email found")
}

// GetUserRepositories GitHub 사용자 저장소 목록 조회 (권한 확인용)
func (g *GitHubProvider) GetUserRepositories(ctx context.Context, accessToken string) ([]string, error) {
	req, err := CreateAPIRequest(ctx, "GET", "https://api.github.com/user/repos", nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create repositories request: %w", err)
	}
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}
	
	var repos []map[string]interface{}
	if err := ParseAPIResponse(resp, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub repositories: %w", err)
	}
	
	repoNames := make([]string, len(repos))
	for i, repo := range repos {
		if name, ok := repo["full_name"].(string); ok {
			repoNames[i] = name
		}
	}
	
	return repoNames, nil
}

// CheckRepositoryAccess 특정 저장소에 대한 접근 권한 확인
func (g *GitHubProvider) CheckRepositoryAccess(ctx context.Context, accessToken, repository string) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", repository)
	req, err := CreateAPIRequest(ctx, "GET", url, nil, accessToken)
	if err != nil {
		return false, err
	}
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	
	// 200 OK면 접근 가능, 404면 접근 불가능 또는 존재하지 않음
	return resp.StatusCode == http.StatusOK, nil
}

// GetRateLimit GitHub API 속도 제한 정보 조회
func (g *GitHubProvider) GetRateLimit(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	req, err := CreateAPIRequest(ctx, "GET", "https://api.github.com/rate_limit", nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limit request: %w", err)
	}
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}
	
	var rateLimit map[string]interface{}
	if err := ParseAPIResponse(resp, &rateLimit); err != nil {
		return nil, fmt.Errorf("failed to parse rate limit response: %w", err)
	}
	
	return rateLimit, nil
}

// parseGitHubTime GitHub 시간 형식 파싱
func parseGitHubTime(timeStr string) (interface{}, bool) {
	if timeStr == "" {
		return nil, false
	}
	
	// GitHub는 ISO 8601 형식을 사용
	if t, ok := getTimeField(map[string]interface{}{"time": timeStr}, "time"); ok {
		return t, true
	}
	
	return nil, false
}

// init 함수에서 GitHub 제공자 등록
func init() {
	RegisterProvider("github", NewGitHubProvider)
}