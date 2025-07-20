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
	testProviderGitHub = "github"
	testSchemeHTTP     = "http"
)

// MockHTTPClient 테스트용 HTTP 클라이언트
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestNewGitHubProvider(t *testing.T) {
	config := ProviderConfig{
		Name:         testProviderGitHub,
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "https://example.com/callback",
	}

	provider := NewGitHubProvider(config)

	if provider.GetName() != testProviderGitHub {
		t.Errorf("Expected provider name '%s', got '%s'", testProviderGitHub, provider.GetName())
	}

	// GitHub 기본값이 설정되었는지 확인
	githubProvider := provider.(*GitHubProvider)
	if githubProvider.config.AuthURL != "https://github.com/login/oauth/authorize" {
		t.Error("GitHub AuthURL should be set to default value")
	}
	if githubProvider.config.TokenURL != "https://github.com/login/oauth/access_token" {
		t.Error("GitHub TokenURL should be set to default value")
	}
	if githubProvider.config.UserInfoURL != "https://api.github.com/user" {
		t.Error("GitHub UserInfoURL should be set to default value")
	}

	expectedScopes := []string{"user:email"}
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

func TestGitHubProvider_GetUserInfo(t *testing.T) {
	// 테스트용 서버 설정
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			userResponse := GitHubUser{
				ID:        123456,
				Login:     "testuser",
				Name:      testUserName,
				Email:     testEmail,
				AvatarURL: "https://github.com/images/error/testuser_happy.gif",
				Company:   "GitHub",
				Location:  "San Francisco",
				Bio:       "There once was...",
				CreatedAt: "2008-01-14T04:33:35Z",
				UpdatedAt: "2008-01-14T04:33:35Z",
			}
			_ = json.NewEncoder(w).Encode(userResponse)
		case "/user/emails":
			emailResponse := []GitHubEmails{
				{Email: testEmail, Primary: true, Verified: true},
				{Email: "secondary@example.com", Primary: false, Verified: true},
			}
			_ = json.NewEncoder(w).Encode(emailResponse)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        testProviderGitHub,
		UserInfoURL: server.URL + "/user",
	}

	provider := NewGitHubProvider(config)

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
	if userInfo.Name != testUserName {
		t.Errorf("Expected name '%s', got '%s'", testUserName, userInfo.Name)
	}
	if userInfo.Email != testEmail {
		t.Errorf("Expected email '%s', got '%s'", testEmail, userInfo.Email)
	}
	if userInfo.Company != "GitHub" {
		t.Errorf("Expected company 'GitHub', got '%s'", userInfo.Company)
	}
}

func TestGitHubProvider_GetUserOrganizations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/orgs" {
			orgsResponse := []GitHubOrganization{
				{ID: 1, Login: "github", DisplayName: "GitHub"},
				{ID: 2, Login: "octocat", DisplayName: "Octocat"},
			}
			_ = json.NewEncoder(w).Encode(orgsResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitHub,
	}

	provider := NewGitHubProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{}

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/user/orgs" {
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

	expectedOrgs := []string{"github", "octocat"}
	if len(organizations) != len(expectedOrgs) {
		t.Errorf("Expected %d organizations, got %d", len(expectedOrgs), len(organizations))
	}

	for i, org := range organizations {
		if org != expectedOrgs[i] {
			t.Errorf("Expected organization '%s', got '%s'", expectedOrgs[i], org)
		}
	}
}

func TestGitHubProvider_ValidateToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			userResponse := GitHubUser{
				ID:    123456,
				Login: "testuser",
				Name:  testUserName,
				Email: testEmail,
			}
			_ = json.NewEncoder(w).Encode(userResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        testProviderGitHub,
		UserInfoURL: server.URL + "/user",
		Scopes:      []string{"user:email"},
	}

	provider := NewGitHubProvider(config)

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
	if tokenInfo.Scope != "user:email" {
		t.Errorf("Expected scope 'user:email', got '%s'", tokenInfo.Scope)
	}
}

func TestGitHubProvider_ValidateToken_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        testProviderGitHub,
		UserInfoURL: server.URL + "/user",
	}

	provider := NewGitHubProvider(config)

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

func TestGitHubProvider_GetUserRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			reposResponse := []map[string]interface{}{
				{"full_name": "testuser/repo1", "name": "repo1"},
				{"full_name": "testuser/repo2", "name": "repo2"},
			}
			_ = json.NewEncoder(w).Encode(reposResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitHub,
	}

	provider := NewGitHubProvider(config).(*GitHubProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{}

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/user/repos" {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	repositories, err := provider.GetUserRepositories(ctx, "test_token")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedRepos := []string{"testuser/repo1", "testuser/repo2"}
	if len(repositories) != len(expectedRepos) {
		t.Errorf("Expected %d repositories, got %d", len(expectedRepos), len(repositories))
	}

	for i, repo := range repositories {
		if repo != expectedRepos[i] {
			t.Errorf("Expected repository '%s', got '%s'", expectedRepos[i], repo)
		}
	}
}

func TestGitHubProvider_CheckRepositoryAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/testuser/accessible-repo":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"full_name": "testuser/accessible-repo",
			})
		case "/repos/testuser/private-repo":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitHub,
	}

	provider := NewGitHubProvider(config).(*GitHubProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	ctx := context.Background()

	// Mock 서버 URL로 요청을 리다이렉트하기 위해 클라이언트 교체
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// GitHub API URL을 테스트 서버 URL로 변경
			if req.URL.Host == "api.github.com" {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	// 접근 가능한 저장소 테스트
	hasAccess, err := provider.CheckRepositoryAccess(ctx, "test_token", "testuser/accessible-repo")
	if err != nil {
		t.Errorf("Expected no error for accessible repo, got: %v", err)
	}
	if !hasAccess {
		t.Error("Expected access to accessible repository")
	}

	// 접근 불가능한 저장소 테스트
	hasAccess, err = provider.CheckRepositoryAccess(ctx, "test_token", "testuser/private-repo")
	if err != nil {
		t.Errorf("Expected no error for private repo, got: %v", err)
	}
	if hasAccess {
		t.Error("Expected no access to private repository")
	}
}

func TestGitHubProvider_GetRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rate_limit" {
			rateLimitResponse := map[string]interface{}{
				"rate": map[string]interface{}{
					"limit":     5000,
					"remaining": 4999,
					"reset":     1372700873,
				},
			}
			_ = json.NewEncoder(w).Encode(rateLimitResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name: testProviderGitHub,
	}

	provider := NewGitHubProvider(config).(*GitHubProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// GitHub API URL을 테스트 서버 URL로 변경
			if req.URL.Host == "api.github.com" {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	rateLimit, err := provider.GetRateLimit(ctx, "test_token")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if rateLimit == nil {
		t.Error("Expected rate limit data")
	}

	rate, ok := rateLimit["rate"].(map[string]interface{})
	if !ok {
		t.Error("Expected rate limit structure")
	}

	if limit, ok := rate["limit"].(float64); !ok || limit != 5000 {
		t.Errorf("Expected limit 5000, got %v", rate["limit"])
	}
}

func TestParseGitHubTime(t *testing.T) {
	testTime := "2008-01-14T04:33:35Z"
	parsedTime, ok := parseGitHubTime(testTime)

	if !ok {
		t.Error("Expected GitHub time to be parsed successfully")
	}

	if parsedTime == nil {
		t.Error("Expected parsed time to not be nil")
	}

	// 빈 문자열 테스트
	_, ok = parseGitHubTime("")
	if ok {
		t.Error("Expected empty time string to fail parsing")
	}
}

func TestGitHubProvider_Registration(t *testing.T) {
	// GitHub 제공자가 자동으로 등록되었는지 확인
	providers := GetRegisteredProviders()

	found := false
	for _, name := range providers {
		if name == testProviderGitHub {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected GitHub provider to be automatically registered")
	}

	// 제공자 생성 테스트
	config := ProviderConfig{
		Name:         testProviderGitHub,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		RedirectURI:  "https://example.com/callback",
	}

	provider := CreateProvider(testProviderGitHub, config)
	if provider == nil {
		t.Error("Expected GitHub provider to be created")
	}

	githubProvider, ok := provider.(*GitHubProvider)
	if !ok {
		t.Error("Expected provider to be GitHubProvider instance")
	}

	if githubProvider.GetName() != testProviderGitHub {
		t.Errorf("Expected provider name '%s', got '%s'", testProviderGitHub, githubProvider.GetName())
	}
}
