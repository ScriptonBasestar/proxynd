package oauth2

import (
	"context"
	"testing"
	"time"
)

// Test-specific constants
const (
	testEmail    = "test@example.com"
	testUserName = "Test User"
)

func TestProviderRegistry(t *testing.T) {
	// 테스트용 팩토리 함수
	testFactory := func(config ProviderConfig) Provider {
		return NewGenericProvider(config)
	}

	// 제공자 등록
	RegisterProvider("test", testFactory)

	// 등록된 제공자 목록 확인
	providers := GetRegisteredProviders()
	found := false
	for _, name := range providers {
		if name == "test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'test' provider to be registered")
	}

	// 제공자 생성 테스트
	config := ProviderConfig{
		Name:         "test",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "https://example.com/callback",
		AuthURL:      "https://provider.com/auth",
		TokenURL:     "https://provider.com/token",
		UserInfoURL:  "https://provider.com/user",
		Scopes:       []string{"read", "write"},
	}

	provider := CreateProvider("test", config)
	if provider == nil {
		t.Error("Expected provider to be created")
	}

	if provider.GetName() != "test" {
		t.Errorf("Expected provider name 'test', got '%s'", provider.GetName())
	}
}

func TestGenericProvider(t *testing.T) {
	config := ProviderConfig{
		Name:         "generic",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "https://example.com/callback",
		AuthURL:      "https://provider.com/auth",
		TokenURL:     "https://provider.com/token",
		UserInfoURL:  "https://provider.com/user",
		Scopes:       []string{"read", "write"},
		EnablePKCE:   true,
	}

	provider := NewGenericProvider(config)

	// GetName 테스트
	if provider.GetName() != "generic" {
		t.Errorf("Expected name 'generic', got '%s'", provider.GetName())
	}

	// GetScopes 테스트
	scopes := provider.GetScopes()
	if len(scopes) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(scopes))
	}
	if scopes[0] != "read" || scopes[1] != "write" {
		t.Errorf("Expected scopes [read, write], got %v", scopes)
	}

	// GetAuthURL 테스트
	authURL := provider.GetAuthURL("test_state", "test_challenge")
	if authURL == "" {
		t.Error("Expected auth URL to be generated")
	}

	// URL에 필수 파라미터가 포함되어 있는지 확인
	expectedParams := []string{
		"client_id=test_client_id",
		"redirect_uri=https%3A%2F%2Fexample.com%2Fcallback",
		"response_type=code",
		"state=test_state",
		"scope=read+write",
		"code_challenge=test_challenge",
		"code_challenge_method=S256",
	}

	for _, param := range expectedParams {
		if !contains(authURL, param) {
			t.Errorf("Expected auth URL to contain '%s', got: %s", param, authURL)
		}
	}
}

func TestGenericProviderWithoutPKCE(t *testing.T) {
	config := ProviderConfig{
		Name:         "no-pkce",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "https://example.com/callback",
		AuthURL:      "https://provider.com/auth",
		TokenURL:     "https://provider.com/token",
		UserInfoURL:  "https://provider.com/user",
		Scopes:       []string{"read"},
		EnablePKCE:   false,
	}

	provider := NewGenericProvider(config)
	authURL := provider.GetAuthURL("test_state", "test_challenge")

	// PKCE 파라미터가 포함되지 않아야 함
	if contains(authURL, "code_challenge") {
		t.Error("Expected auth URL to not contain PKCE parameters when disabled")
	}
}

func TestGenericProviderUserOrganizations(t *testing.T) {
	config := ProviderConfig{
		Name: "generic",
	}

	provider := NewGenericProvider(config)
	ctx := context.Background()

	// Generic 제공자는 조직 정보를 지원하지 않음
	organizations, err := provider.GetUserOrganizations(ctx, "test_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(organizations) != 0 {
		t.Errorf("Expected empty organizations list, got: %v", organizations)
	}
}

func TestTokenResponse(t *testing.T) {
	tokenResp := &TokenResponse{
		AccessToken:  "access_token_123",
		TokenType:    "Bearer",
		RefreshToken: "refresh_token_123",
		ExpiresIn:    3600,
		Scope:        "read write",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	if tokenResp.AccessToken != "access_token_123" {
		t.Errorf("Expected access token 'access_token_123', got '%s'", tokenResp.AccessToken)
	}

	if tokenResp.TokenType != "Bearer" {
		t.Errorf("Expected token type 'Bearer', got '%s'", tokenResp.TokenType)
	}

	if tokenResp.ExpiresIn != 3600 {
		t.Errorf("Expected expires in 3600, got %d", tokenResp.ExpiresIn)
	}
}

func TestUserInfo(t *testing.T) {
	userInfo := &UserInfo{
		ID:        "user123",
		Email:     testEmail,
		Name:      testUserName,
		Username:  "testuser",
		Avatar:    "https://example.com/avatar.jpg",
		Company:   "Test Company",
		Location:  "Test City",
		Bio:       "Test bio",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if userInfo.ID != "user123" {
		t.Errorf("Expected ID 'user123', got '%s'", userInfo.ID)
	}

	if userInfo.Email != testEmail {
		t.Errorf("Expected email '%s', got '%s'", testEmail, userInfo.Email)
	}

	if userInfo.Name != testUserName {
		t.Errorf("Expected name '%s', got '%s'", testUserName, userInfo.Name)
	}
}

func TestTokenInfo(t *testing.T) {
	tokenInfo := &TokenInfo{
		Valid:     true,
		ExpiresAt: time.Now().Add(time.Hour),
		Scope:     "read write",
		UserID:    "user123",
	}

	if !tokenInfo.Valid {
		t.Error("Expected token to be valid")
	}

	if tokenInfo.Scope != "read write" {
		t.Errorf("Expected scope 'read write', got '%s'", tokenInfo.Scope)
	}

	if tokenInfo.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", tokenInfo.UserID)
	}
}

func TestJoinScopes(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		expected string
	}{
		{
			name:     "Empty scopes",
			scopes:   []string{},
			expected: "",
		},
		{
			name:     "Single scope",
			scopes:   []string{"read"},
			expected: "read",
		},
		{
			name:     "Multiple scopes",
			scopes:   []string{"read", "write", "admin"},
			expected: "read write admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinScopes(tt.scopes)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCreateProviderWithUnknownName(t *testing.T) {
	config := ProviderConfig{
		Name:         "unknown",
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		RedirectURI:  "https://example.com/callback",
		AuthURL:      "https://provider.com/auth",
		TokenURL:     "https://provider.com/token",
		UserInfoURL:  "https://provider.com/user",
	}

	// 등록되지 않은 제공자는 Generic 제공자로 폴백되어야 함
	provider := CreateProvider("unknown", config)
	if provider == nil {
		t.Error("Expected generic provider to be created for unknown provider")
	}

	if provider.GetName() != "unknown" {
		t.Errorf("Expected provider name 'unknown', got '%s'", provider.GetName())
	}
}

// contains 문자열 포함 확인 헬퍼 함수
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
