package config

import (
	"testing"
)

func TestOAuth2Provider_ValidateProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    OAuth2Provider
		expectError bool
	}{
		{
			name: "Valid GitHub provider",
			provider: OAuth2Provider{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"user:email"},
			},
			expectError: false,
		},
		{
			name: "Missing client_id",
			provider: OAuth2Provider{
				ClientSecret: "test_client_secret",
				RedirectURI:  "https://example.com/callback",
			},
			expectError: true,
		},
		{
			name: "Missing client_secret",
			provider: OAuth2Provider{
				ClientID:    "test_client_id",
				RedirectURI: "https://example.com/callback",
			},
			expectError: true,
		},
		{
			name: "Missing redirect_uri",
			provider: OAuth2Provider{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
			},
			expectError: true,
		},
		{
			name: "Valid generic provider",
			provider: OAuth2Provider{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				RedirectURI:  "https://example.com/callback",
				AuthURL:      "https://provider.com/auth",
				TokenURL:     "https://provider.com/token",
				UserInfoURL:  "https://provider.com/user",
			},
			expectError: false,
		},
		{
			name: "Incomplete generic provider",
			provider: OAuth2Provider{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				RedirectURI:  "https://example.com/callback",
				AuthURL:      "https://provider.com/auth",
				// Missing TokenURL and UserInfoURL
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.ValidateProvider()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestOAuth2Provider_IsGenericProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider OAuth2Provider
		expected bool
	}{
		{
			name: "Standard provider",
			provider: OAuth2Provider{
				ClientID:     "test",
				ClientSecret: "test",
				RedirectURI:  "https://example.com/callback",
			},
			expected: false,
		},
		{
			name: "Generic provider",
			provider: OAuth2Provider{
				ClientID:     "test",
				ClientSecret: "test",
				RedirectURI:  "https://example.com/callback",
				AuthURL:      "https://provider.com/auth",
				TokenURL:     "https://provider.com/token",
				UserInfoURL:  "https://provider.com/user",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.IsGenericProvider()
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestOAuth2Provider_GetEffectiveScopes(t *testing.T) {
	tests := []struct {
		name     string
		provider OAuth2Provider
		expected []string
	}{
		{
			name: "Custom scopes",
			provider: OAuth2Provider{
				Scopes: []string{"user:email", "read:org"},
			},
			expected: []string{"user:email", "read:org"},
		},
		{
			name:     "Default scopes",
			provider: OAuth2Provider{},
			expected: []string{"openid", "email", "profile"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.GetEffectiveScopes()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d scopes, got %d", len(tt.expected), len(result))
				return
			}
			for i, scope := range result {
				if scope != tt.expected[i] {
					t.Errorf("Expected scope %s, got %s", tt.expected[i], scope)
				}
			}
		})
	}
}

func TestJWTConfig_ValidateJWT(t *testing.T) {
	tests := []struct {
		name        string
		jwt         JWTConfig
		expectError bool
	}{
		{
			name: "Valid JWT config",
			jwt: JWTConfig{
				Secret:          "this-is-a-very-long-secret-key-for-jwt-signing",
				AccessTokenTTL:  3600,
				RefreshTokenTTL: 604800,
				Algorithm:       "HS256",
			},
			expectError: false,
		},
		{
			name: "Missing secret",
			jwt: JWTConfig{
				AccessTokenTTL:  3600,
				RefreshTokenTTL: 604800,
				Algorithm:       "HS256",
			},
			expectError: true,
		},
		{
			name: "Short secret",
			jwt: JWTConfig{
				Secret:          "short",
				AccessTokenTTL:  3600,
				RefreshTokenTTL: 604800,
				Algorithm:       "HS256",
			},
			expectError: true,
		},
		{
			name: "Invalid algorithm",
			jwt: JWTConfig{
				Secret:          "this-is-a-very-long-secret-key-for-jwt-signing",
				AccessTokenTTL:  3600,
				RefreshTokenTTL: 604800,
				Algorithm:       "INVALID",
			},
			expectError: true,
		},
		{
			name: "Zero TTL values get defaults",
			jwt: JWTConfig{
				Secret:          "this-is-a-very-long-secret-key-for-jwt-signing",
				AccessTokenTTL:  0,
				RefreshTokenTTL: 0,
				Algorithm:       "HS256",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.jwt.ValidateJWT()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// TTL 기본값 설정 확인
			if !tt.expectError && tt.jwt.AccessTokenTTL == 0 {
				if tt.jwt.AccessTokenTTL != 3600 {
					t.Error("AccessTokenTTL should be set to default 3600")
				}
			}
		})
	}
}

func TestUserMappingConfig_ValidateUserMapping(t *testing.T) {
	tests := []struct {
		name        string
		userMapping UserMappingConfig
		expectError bool
	}{
		{
			name: "Valid user mapping",
			userMapping: UserMappingConfig{
				DefaultRole: "viewer",
				RoleMapping: map[string]string{
					"*@admin.com": "admin",
					"*@dev.com":   "developer",
				},
			},
			expectError: false,
		},
		{
			name: "Invalid default role",
			userMapping: UserMappingConfig{
				DefaultRole: "invalid_role",
			},
			expectError: true,
		},
		{
			name: "Invalid role in mapping",
			userMapping: UserMappingConfig{
				DefaultRole: "viewer",
				RoleMapping: map[string]string{
					"*@admin.com": "invalid_role",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.userMapping.ValidateUserMapping()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestUserMappingConfig_GetUserRole(t *testing.T) {
	userMapping := UserMappingConfig{
		DefaultRole:        "viewer",
		AdminUsers:         []string{"admin@example.com"},
		AdminOrganizations: []string{"admin-org"},
		OrganizationMapping: map[string]string{
			"engineering": "developer",
			"devops":      "maintainer",
		},
	}

	tests := []struct {
		name          string
		email         string
		organizations []string
		expected      string
	}{
		{
			name:     "Admin user",
			email:    "admin@example.com",
			expected: "admin",
		},
		{
			name:          "Admin organization",
			email:         "user@example.com",
			organizations: []string{"admin-org"},
			expected:      "admin",
		},
		{
			name:          "Organization mapping",
			email:         "user@example.com",
			organizations: []string{"engineering"},
			expected:      "developer",
		},
		{
			name:          "Default role",
			email:         "user@example.com",
			organizations: []string{"unknown-org"},
			expected:      "viewer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := userMapping.GetUserRole(tt.email, tt.organizations)
			if result != tt.expected {
				t.Errorf("Expected role %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOAuth2Config_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      OAuth2Config
		expectError bool
	}{
		{
			name: "Disabled config",
			config: OAuth2Config{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "Valid config",
			config: OAuth2Config{
				Enabled:         true,
				DefaultProvider: "github",
				Providers: map[string]OAuth2Provider{
					"github": {
						ClientID:     "test_id",
						ClientSecret: "test_secret",
						RedirectURI:  "https://example.com/callback",
					},
				},
				JWT: JWTConfig{
					Secret:    "this-is-a-very-long-secret-key-for-jwt-signing",
					Algorithm: "HS256",
				},
				UserMapping: UserMappingConfig{
					DefaultRole: "viewer",
				},
			},
			expectError: false,
		},
		{
			name: "Missing default provider",
			config: OAuth2Config{
				Enabled: true,
			},
			expectError: true,
		},
		{
			name: "Default provider not in providers",
			config: OAuth2Config{
				Enabled:         true,
				DefaultProvider: "nonexistent",
				Providers:       map[string]OAuth2Provider{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestOAuth2Config_MergeWithDefaults(t *testing.T) {
	config := OAuth2Config{
		Providers: map[string]OAuth2Provider{
			"github": {
				ClientID:     "test_id",
				ClientSecret: "test_secret",
				RedirectURI:  "https://example.com/callback",
				// AuthURL, TokenURL, UserInfoURL는 기본값으로 설정되어야 함
			},
		},
	}

	config.MergeWithDefaults()

	// GitHub 기본값이 설정되었는지 확인
	githubProvider := config.Providers["github"]
	if githubProvider.AuthURL == "" {
		t.Error("GitHub AuthURL should be set to default value")
	}
	if githubProvider.TokenURL == "" {
		t.Error("GitHub TokenURL should be set to default value")
	}
	if githubProvider.UserInfoURL == "" {
		t.Error("GitHub UserInfoURL should be set to default value")
	}

	// JWT 기본값 확인
	if config.JWT.Issuer != "proxynd" {
		t.Error("JWT Issuer should be set to default value")
	}
	if config.JWT.Audience != "proxynd-api" {
		t.Error("JWT Audience should be set to default value")
	}
}

func TestGetDefaultOAuth2Providers(t *testing.T) {
	providers := GetDefaultOAuth2Providers()

	expectedProviders := []string{"github", "gitlab", "google"}
	for _, providerName := range expectedProviders {
		provider, exists := providers[providerName]
		if !exists {
			t.Errorf("Expected provider %s to exist", providerName)
			continue
		}

		if provider.AuthURL == "" {
			t.Errorf("Provider %s should have AuthURL", providerName)
		}
		if provider.TokenURL == "" {
			t.Errorf("Provider %s should have TokenURL", providerName)
		}
		if provider.UserInfoURL == "" {
			t.Errorf("Provider %s should have UserInfoURL", providerName)
		}
		if len(provider.Scopes) == 0 {
			t.Errorf("Provider %s should have default scopes", providerName)
		}
	}
}
