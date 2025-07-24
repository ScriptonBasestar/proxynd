package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test-specific constants
const (
	testProviderGitLab = "gitlab"
	testPathAPIV4User  = "/api/v4/user"
	testHostGitLab     = "gitlab.com"
)

func TestNewGitLabProvider(t *testing.T) {
	config := ProviderConfig{
		Name:         testProviderGitLab,
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "https://example.com/callback",
	}

	provider := NewGitLabProvider(config)

	if provider.GetName() != testProviderGitLab {
		t.Errorf("Expected provider name '%s', got '%s'", testProviderGitLab, provider.GetName())
	}

	// GitLab 기본값이 설정되었는지 확인
	gitlabProvider := provider.(*GitLabProvider)
	if gitlabProvider.config.AuthURL != "https://gitlab.com/oauth/authorize" {
		t.Error("GitLab AuthURL should be set to default value")
	}
	if gitlabProvider.config.TokenURL != "https://gitlab.com/oauth/token" {
		t.Error("GitLab TokenURL should be set to default value")
	}
	if gitlabProvider.config.UserInfoURL != "https://gitlab.com/api/v4/user" {
		t.Error("GitLab UserInfoURL should be set to default value")
	}
	if gitlabProvider.config.RevokeURL != "https://gitlab.com/oauth/revoke" {
		t.Error("GitLab RevokeURL should be set to default value")
	}

	expectedScopes := []string{"read_user"}
	scopes := provider.GetScopes()
	if len(scopes) != len(expectedScopes) {
		t.Errorf("Expected %d scopes, got %d", len(expectedScopes), len(scopes))
	}
	for i, scope := range scopes {
		if scope != expectedScopes[i] {
			t.Errorf("Expected scope '%s', got '%s'", expectedScopes[i], scope)
		}
	}
}

func TestGitLabProvider_GetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathAPIV4User {
			userResponse := GitLabUser{
				ID:           123456,
				Username:     "testuser",
				Name:         "Test User",
				Email:        testEmail,
				AvatarURL:    "https://gitlab.com/uploads/-/system/user/avatar/123456/avatar.png",
				State:        "active",
				Bio:          "Software Developer",
				Location:     "San Francisco",
				Organization: "GitLab Inc",
				JobTitle:     "Backend Developer",
				CreatedAt:    "2015-01-01T00:00:00.000Z",
				UpdatedAt:    "2021-01-01",
				IsAdmin:      false,
			}
			_ = json.NewEncoder(w).Encode(userResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        testProviderGitLab,
		UserInfoURL: server.URL + testPathAPIV4User,
	}

	provider := NewGitLabProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{}

	ctx := context.Background()
	userInfo, err := provider.GetUserInfo(ctx, "test_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if userInfo.ID != "123456" {
		t.Errorf("Expected ID '123456', got '%s'", userInfo.ID)
	}
	if userInfo.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", userInfo.Username)
	}
	if userInfo.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got '%s'", userInfo.Name)
	}
	if userInfo.Email != testEmail {
		t.Errorf("Expected email '%s', got '%s'", testEmail, userInfo.Email)
	}
	if userInfo.Company != "GitLab Inc" {
		t.Errorf("Expected company 'GitLab Inc', got '%s'", userInfo.Company)
	}
	if userInfo.Bio != "Software Developer" {
		t.Errorf("Expected bio 'Software Developer', got '%s'", userInfo.Bio)
	}
	if userInfo.Location != "San Francisco" {
		t.Errorf("Expected location 'San Francisco', got '%s'", userInfo.Location)
	}
}

func TestGitLabProvider_GetUserOrganizations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/groups" {
			groupsResponse := []GitLabGroup{
				{ID: 1, Name: "GitLab Inc", Path: "gitlab-inc", Visibility: "private"},
				{ID: 2, Name: "Open Source", Path: "open-source", Visibility: "public"},
			}
			_ = json.NewEncoder(w).Encode(groupsResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitLab,
	}

	provider := NewGitLabProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v4/groups" {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	organizations, err := provider.GetUserOrganizations(ctx, "test_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedOrgs := []string{"gitlab-inc", "open-source"}
	if len(organizations) != len(expectedOrgs) {
		t.Errorf("Expected %d organizations, got %d", len(expectedOrgs), len(organizations))
	}

	for i, org := range organizations {
		if org != expectedOrgs[i] {
			t.Errorf("Expected organization '%s', got '%s'", expectedOrgs[i], org)
		}
	}
}

func TestGitLabProvider_ValidateToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathAPIV4User {
			userResponse := GitLabUser{
				ID:       123456,
				Username: "testuser",
				Name:     "Test User",
				Email:    testEmail,
			}
			_ = json.NewEncoder(w).Encode(userResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        testProviderGitLab,
		UserInfoURL: server.URL + testPathAPIV4User,
		Scopes:      []string{"read_user"},
	}

	provider := NewGitLabProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{}

	ctx := context.Background()
	tokenInfo, err := provider.ValidateToken(ctx, "valid_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !tokenInfo.Valid {
		t.Error("Expected token to be valid")
	}
	if tokenInfo.UserID != "123456" {
		t.Errorf("Expected user ID '123456', got '%s'", tokenInfo.UserID)
	}
	if tokenInfo.Scope != "read_user" {
		t.Errorf("Expected scope 'read_user', got '%s'", tokenInfo.Scope)
	}
}

func TestGitLabProvider_ValidateToken_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        testProviderGitLab,
		UserInfoURL: server.URL + testPathAPIV4User,
	}

	provider := NewGitLabProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{}

	ctx := context.Background()
	tokenInfo, err := provider.ValidateToken(ctx, "invalid_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tokenInfo.Valid {
		t.Error("Expected token to be invalid")
	}
}

func TestGitLabProvider_GetUserProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/projects" && r.URL.RawQuery == "membership=true" {
			projectsResponse := []GitLabProject{
				{
					ID:                1,
					Name:              "awesome-project",
					Path:              "awesome-project",
					PathWithNamespace: "testuser/awesome-project",
					Description:       "An awesome project",
					DefaultBranch:     "main",
					Visibility:        "private",
				},
				{
					ID:                2,
					Name:              "public-lib",
					Path:              "public-lib",
					PathWithNamespace: "opensource/public-lib",
					Description:       "A public library",
					DefaultBranch:     "master",
					Visibility:        "public",
				},
			}
			_ = json.NewEncoder(w).Encode(projectsResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitLab,
	}

	provider := NewGitLabProvider(config).(*GitLabProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v4/projects" {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	projects, err := provider.GetUserProjects(ctx, "test_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	expectedProjects := []string{"awesome-project", "public-lib"}
	for i, project := range projects {
		if project.Name != expectedProjects[i] {
			t.Errorf("Expected project name '%s', got '%s'", expectedProjects[i], project.Name)
		}
	}
}

func TestGitLabProvider_CheckProjectAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/testuser/accessible-project":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   1,
				"name": "accessible-project",
			})
		case "/api/v4/projects/testuser/private-project":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitLab,
	}

	provider := NewGitLabProvider(config).(*GitLabProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostGitLab {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
				// req.URL.Path = req.URL.Path   // Path 유지 (self-assignment removed)
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()

	// 접근 가능한 프로젝트 테스트
	hasAccess, err := provider.CheckProjectAccess(ctx, "test_token", "testuser/accessible-project")
	if err != nil {
		t.Errorf("Expected no error for accessible project, got: %v", err)
	}
	if !hasAccess {
		t.Error("Expected access to accessible project")
	}

	// 접근 불가능한 프로젝트 테스트
	hasAccess, err = provider.CheckProjectAccess(ctx, "test_token", "testuser/private-project")
	if err != nil {
		t.Errorf("Expected no error for private project, got: %v", err)
	}
	if hasAccess {
		t.Error("Expected no access to private project")
	}
}

func TestGitLabProvider_GetGroupMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/groups/testgroup/members" {
			membersResponse := []map[string]interface{}{
				{
					"id":           1,
					"username":     "user1",
					"name":         "User One",
					"access_level": 50, // Owner
				},
				{
					"id":           2,
					"username":     "user2",
					"name":         "User Two",
					"access_level": 30, // Developer
				},
			}
			_ = json.NewEncoder(w).Encode(membersResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitLab,
	}

	provider := NewGitLabProvider(config).(*GitLabProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostGitLab {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
				// req.URL.Path = req.URL.Path   // Path 유지 (self-assignment removed)
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	members, err := provider.GetGroupMembers(ctx, "test_token", "testgroup")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	// 첫 번째 멤버 확인
	if username, ok := members[0]["username"].(string); !ok || username != "user1" {
		t.Errorf("Expected first member username 'user1', got '%v'", members[0]["username"])
	}

	if accessLevel, ok := members[0]["access_level"].(float64); !ok || accessLevel != 50 {
		t.Errorf("Expected first member access level 50, got '%v'", members[0]["access_level"])
	}
}

func TestMapGitLabAccessLevel(t *testing.T) {
	tests := []struct {
		accessLevel int
		expected    string
	}{
		{10, "guest"},
		{20, "reporter"},
		{30, "developer"},
		{40, "maintainer"},
		{50, "owner"},
		{0, "guest"},   // 알 수 없는 레벨은 guest로
		{100, "guest"}, // 알 수 없는 레벨은 guest로
	}

	for _, tt := range tests {
		result := mapGitLabAccessLevel(tt.accessLevel)
		if result != tt.expected {
			t.Errorf("Expected access level %d to map to '%s', got '%s'",
				tt.accessLevel, tt.expected, result)
		}
	}
}

func TestGitLabProvider_CheckAdminStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			userResponse := GitLabUser{
				ID:       123456,
				Username: "admin",
				IsAdmin:  true,
			}
			_ = json.NewEncoder(w).Encode(userResponse)
		case "/api/v4/users":
			// 관리자만 접근 가능한 엔드포인트
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "username": "user1"},
				{"id": 2, "username": "user2"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        testProviderGitLab,
		UserInfoURL: server.URL + testPathAPIV4User,
	}

	provider := NewGitLabProvider(config).(*GitLabProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostGitLab {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
				// req.URL.Path = req.URL.Path   // Path 유지 (self-assignment removed)
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	isAdmin, err := provider.CheckAdminStatus(ctx, "admin_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !isAdmin {
		t.Error("Expected user to be admin")
	}
}

func TestParseGitLabTime(t *testing.T) {
	testTime := "2015-01-01T00:00:00.000Z"
	parsedTime, ok := parseGitLabTime(testTime)

	if !ok {
		t.Error("Expected GitLab time to be parsed successfully")
	}

	if parsedTime == nil {
		t.Error("Expected parsed time to not be nil")
	}

	// 빈 문자열 테스트
	_, ok = parseGitLabTime("")
	if ok {
		t.Error("Expected empty time string to fail parsing")
	}
}

func TestGitLabProvider_Registration(t *testing.T) {
	// GitLab 제공자가 자동으로 등록되었는지 확인
	providers := GetRegisteredProviders()

	found := false
	for _, name := range providers {
		if name == testProviderGitLab {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected GitLab provider to be automatically registered")
	}

	// 제공자 생성 테스트
	config := ProviderConfig{
		Name:         testProviderGitLab,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		RedirectURI:  "https://example.com/callback",
	}

	provider := CreateProvider(testProviderGitLab, config)
	if provider == nil {
		t.Error("Expected GitLab provider to be created")
	}

	gitlabProvider, ok := provider.(*GitLabProvider)
	if !ok {
		t.Error("Expected provider to be GitLabProvider instance")
	}

	if gitlabProvider.GetName() != testProviderGitLab {
		t.Errorf("Expected provider name '%s', got '%s'", testProviderGitLab, gitlabProvider.GetName())
	}
}
