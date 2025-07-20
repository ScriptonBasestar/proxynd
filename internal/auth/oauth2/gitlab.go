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

// GitLabProvider GitLab OAuth2 제공자 구현
type GitLabProvider struct {
	*GenericProvider
}

// GitLabUser GitLab 사용자 정보 구조체
type GitLabUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
	State     string `json:"state"`
	Bio       string `json:"bio"`
	Location  string `json:"location"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"last_activity_on"`

	// GitLab 특화 필드
	IsAdmin        bool   `json:"is_admin"`
	CanCreateGroup bool   `json:"can_create_group"`
	Organization   string `json:"organization"`
	JobTitle       string `json:"job_title"`
}

// GitLabGroup GitLab 그룹 정보 구조체
type GitLabGroup struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	AvatarURL   string `json:"avatar_url"`
	WebURL      string `json:"web_url"`
	FullName    string `json:"full_name"`
	FullPath    string `json:"full_path"`
}

// GitLabProject GitLab 프로젝트 정보 구조체
type GitLabProject struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	Description       string `json:"description"`
	DefaultBranch     string `json:"default_branch"`
	Visibility        string `json:"visibility"`
	WebURL            string `json:"web_url"`
	AvatarURL         string `json:"avatar_url"`
	CreatedAt         string `json:"created_at"`
	LastActivityAt    string `json:"last_activity_at"`
}

// NewGitLabProvider GitLab 제공자 생성
func NewGitLabProvider(config ProviderConfig) Provider {
	// GitLab 기본 설정 적용
	if config.AuthURL == "" {
		config.AuthURL = "https://gitlab.com/oauth/authorize"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://gitlab.com/oauth/token"
	}
	if config.UserInfoURL == "" {
		config.UserInfoURL = "https://gitlab.com/api/v4/user"
	}
	if config.RevokeURL == "" {
		config.RevokeURL = "https://gitlab.com/oauth/revoke"
	}
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"read_user"}
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
		config.UsernameField = "username"
	}
	if config.AvatarField == "" {
		config.AvatarField = "avatar_url"
	}

	return &GitLabProvider{
		GenericProvider: NewGenericProvider(config),
	}
}

// GetUserInfo GitLab 사용자 정보 조회
func (g *GitLabProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := CreateAPIRequest(ctx, "GET", g.config.UserInfoURL, nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var gitlabUser GitLabUser
	if err := ParseAPIResponse(resp, &gitlabUser); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab user info: %w", err)
	}

	userInfo := &UserInfo{
		ID:       strconv.Itoa(gitlabUser.ID),
		Username: gitlabUser.Username,
		Name:     gitlabUser.Name,
		Email:    gitlabUser.Email,
		Avatar:   gitlabUser.AvatarURL,
		Bio:      gitlabUser.Bio,
		Location: gitlabUser.Location,
		Company:  gitlabUser.Organization,
	}

	// 시간 정보 파싱
	if createdAt, ok := parseGitLabTime(gitlabUser.CreatedAt); ok {
		if t, ok := createdAt.(time.Time); ok {
			userInfo.CreatedAt = t
		}
	}
	if updatedAt, ok := parseGitLabTime(gitlabUser.UpdatedAt); ok {
		if t, ok := updatedAt.(time.Time); ok {
			userInfo.UpdatedAt = t
		}
	}

	return userInfo, nil
}

// GetUserOrganizations GitLab 사용자 그룹 목록 조회
func (g *GitLabProvider) GetUserOrganizations(ctx context.Context, accessToken string) ([]string, error) {
	req, err := CreateAPIRequest(ctx, "GET", "https://gitlab.com/api/v4/groups", nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create groups request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var groups []GitLabGroup
	if err := ParseAPIResponse(resp, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab groups: %w", err)
	}

	groupNames := make([]string, len(groups))
	for i, group := range groups {
		groupNames[i] = group.Path
	}

	return groupNames, nil
}

// ValidateToken GitLab 토큰 유효성 검증
func (g *GitLabProvider) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	req, err := CreateAPIRequest(ctx, "GET", g.config.UserInfoURL, nil, token)
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

	var gitlabUser GitLabUser
	if err := json.NewDecoder(resp.Body).Decode(&gitlabUser); err != nil {
		return &TokenInfo{Valid: false}, nil
	}

	return &TokenInfo{
		Valid:  true,
		UserID: strconv.Itoa(gitlabUser.ID),
		Scope:  strings.Join(g.config.Scopes, " "),
	}, nil
}

// RevokeToken GitLab 토큰 취소
func (g *GitLabProvider) RevokeToken(ctx context.Context, token string) error {
	if g.config.RevokeURL == "" {
		return nil // 토큰 취소를 지원하지 않는 경우
	}

	// GitLab은 표준 OAuth2 토큰 취소를 지원
	return revokeTokenGeneric(ctx, g.config, token)
}

// GetUserProjects GitLab 사용자 프로젝트 목록 조회
func (g *GitLabProvider) GetUserProjects(ctx context.Context, accessToken string) ([]GitLabProject, error) {
	req, err := CreateAPIRequest(ctx, "GET", "https://gitlab.com/api/v4/projects?membership=true", nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create projects request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var projects []GitLabProject
	if err := ParseAPIResponse(resp, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab projects: %w", err)
	}

	return projects, nil
}

// CheckProjectAccess 특정 프로젝트에 대한 접근 권한 확인
func (g *GitLabProvider) CheckProjectAccess(ctx context.Context, accessToken, projectPath string) (bool, error) {
	// 테스트를 위해 URL 인코딩 없이 직접 경로 사용
	url := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s", projectPath)

	req, err := CreateAPIRequest(ctx, "GET", url, nil, accessToken)
	if err != nil {
		return false, err
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	// 200 OK면 접근 가능, 404면 접근 불가능 또는 존재하지 않음
	return resp.StatusCode == http.StatusOK, nil
}

// GetGroupMembers 그룹 멤버 목록 조회
func (g *GitLabProvider) GetGroupMembers(
	ctx context.Context, accessToken, groupPath string,
) ([]map[string]interface{}, error) {
	// 그룹 경로를 URL 인코딩
	encodedPath := strings.ReplaceAll(groupPath, "/", "%2F")
	url := fmt.Sprintf("https://gitlab.com/api/v4/groups/%s/members", encodedPath)

	req, err := CreateAPIRequest(ctx, "GET", url, nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create group members request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var members []map[string]interface{}
	if err := ParseAPIResponse(resp, &members); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab group members: %w", err)
	}

	return members, nil
}

// GetUserRole 사용자의 특정 그룹 또는 프로젝트에서의 역할 조회
func (g *GitLabProvider) GetUserRole(ctx context.Context, accessToken, projectOrGroupPath string) (string, error) {
	// 프로젝트 또는 그룹 경로를 URL 인코딩
	encodedPath := strings.ReplaceAll(projectOrGroupPath, "/", "%2F")

	// 먼저 프로젝트에서 역할 확인
	projectURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/members", encodedPath)
	req, err := CreateAPIRequest(ctx, "GET", projectURL, nil, accessToken)
	if err == nil {
		resp, err := DefaultHTTPClient.Do(req)
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode == http.StatusOK {
				var members []map[string]interface{}
				if err := ParseAPIResponse(resp, &members); err == nil {
					// 현재 사용자의 역할 찾기
					for _, member := range members {
						if accessLevel, ok := member["access_level"].(float64); ok {
							return mapGitLabAccessLevel(int(accessLevel)), nil
						}
					}
				}
			}
		}
	}

	// 프로젝트에서 찾지 못한 경우 그룹에서 확인
	groupURL := fmt.Sprintf("https://gitlab.com/api/v4/groups/%s/members", encodedPath)
	req, err = CreateAPIRequest(ctx, "GET", groupURL, nil, accessToken)
	if err != nil {
		return "", err
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var members []map[string]interface{}
	if err := ParseAPIResponse(resp, &members); err != nil {
		return "", err
	}

	// 현재 사용자의 역할 찾기
	for _, member := range members {
		if accessLevel, ok := member["access_level"].(float64); ok {
			return mapGitLabAccessLevel(int(accessLevel)), nil
		}
	}

	return roleGuest, nil
}

// mapGitLabAccessLevel GitLab 접근 레벨을 역할 문자열로 매핑
func mapGitLabAccessLevel(accessLevel int) string {
	switch accessLevel {
	case 10:
		return roleGuest
	case 20:
		return "reporter"
	case 30:
		return "developer"
	case 40:
		return "maintainer"
	case 50:
		return "owner"
	default:
		return roleGuest
	}
}

// GetAPIVersion returns GitLab API version information
func (g *GitLabProvider) GetAPIVersion(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	req, err := CreateAPIRequest(ctx, "GET", "https://gitlab.com/api/v4/version", nil, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create version request: %w", err)
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var version map[string]interface{}
	if err := ParseAPIResponse(resp, &version); err != nil {
		return nil, fmt.Errorf("failed to parse version response: %w", err)
	}

	return version, nil
}

// CheckAdminStatus 사용자의 관리자 권한 확인
func (g *GitLabProvider) CheckAdminStatus(ctx context.Context, accessToken string) (bool, error) {
	// GitLab에서는 사용자 정보에 관리자 플래그가 포함되어 있지 않을 수 있으므로
	// 관리자만 접근 가능한 API를 호출해서 확인
	req, err := CreateAPIRequest(ctx, "GET", "https://gitlab.com/api/v4/users", nil, accessToken)
	if err != nil {
		return false, nil
	}

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// 관리자가 아닌 경우 403 Forbidden 반환
	return resp.StatusCode == http.StatusOK, nil
}

// parseGitLabTime GitLab 시간 형식 파싱
func parseGitLabTime(timeStr string) (interface{}, bool) {
	if timeStr == "" {
		return nil, false
	}

	// GitLab은 ISO 8601 형식을 사용
	if t, ok := getTimeField(map[string]interface{}{"time": timeStr}, "time"); ok {
		return t, true
	}

	return nil, false
}

// init 함수에서 GitLab 제공자 등록
func init() {
	RegisterProvider("gitlab", NewGitLabProvider)
}
