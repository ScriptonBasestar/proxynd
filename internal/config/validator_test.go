package config

import (
	"os"
	"testing"
)

func TestValidateRequiredEnvVars(t *testing.T) {
	// 테스트 시작 전 환경 변수 백업
	originalJWT := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWT != "" {
			_ = os.Setenv("JWT_SECRET", originalJWT)
		} else {
			_ = os.Unsetenv("JWT_SECRET")
		}
	}()

	tests := []struct {
		name          string
		jwtSecret     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid JWT Secret",
			jwtSecret:   "valid-jwt-secret-with-sufficient-length-32-chars",
			expectError: false,
		},
		{
			name:          "Missing JWT Secret",
			jwtSecret:     "",
			expectError:   true,
			errorContains: "missing required environment variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.jwtSecret != "" {
				_ = os.Setenv("JWT_SECRET", tt.jwtSecret)
			} else {
				_ = os.Unsetenv("JWT_SECRET")
			}

			err := ValidateRequiredEnvVars()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectError && err != nil && tt.errorContains != "" {
				if !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestLoadSecureConfig(t *testing.T) {
	// 테스트 시작 전 환경 변수 백업
	originalJWT := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWT != "" {
			_ = os.Setenv("JWT_SECRET", originalJWT)
		} else {
			_ = os.Unsetenv("JWT_SECRET")
		}
	}()

	tests := []struct {
		name          string
		jwtSecret     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid long JWT secret",
			jwtSecret:   "this-is-a-very-secure-jwt-secret-key-with-more-than-32-characters",
			expectError: false,
		},
		{
			name:          "Short JWT secret",
			jwtSecret:     "short-key",
			expectError:   true,
			errorContains: "must be at least 32 characters",
		},
		{
			name:          "Default JWT secret (insecure)",
			jwtSecret:     "your-jwt-secret-key-here-minimum-32-chars",
			expectError:   true,
			errorContains: "please change JWT_SECRET from default value",
		},
		{
			name:          "Example JWT secret (insecure)",
			jwtSecret:     "example-jwt-secret-key-for-testing-purposes-only",
			expectError:   true,
			errorContains: "please change JWT_SECRET from default value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("JWT_SECRET", tt.jwtSecret)

			err := LoadSecureConfig()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectError && err != nil && tt.errorContains != "" {
				if !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestValidateConditionalEnvVars(t *testing.T) {
	// 테스트 시작 전 환경 변수 백업
	originalOAuth := os.Getenv("OAUTH_ENABLED")
	originalGitHubID := os.Getenv("OAUTH_GITHUB_CLIENT_ID")
	originalGitHubSecret := os.Getenv("OAUTH_GITHUB_CLIENT_SECRET")

	defer func() {
		restoreEnvVar("OAUTH_ENABLED", originalOAuth)
		restoreEnvVar("OAUTH_GITHUB_CLIENT_ID", originalGitHubID)
		restoreEnvVar("OAUTH_GITHUB_CLIENT_SECRET", originalGitHubSecret)
	}()

	tests := []struct {
		name               string
		oauthEnabled       string
		githubClientID     string
		githubClientSecret string
		expectError        bool
		errorContains      string
	}{
		{
			name:               "OAuth disabled - no validation needed",
			oauthEnabled:       "false",
			githubClientID:     "",
			githubClientSecret: "",
			expectError:        false,
		},
		{
			name:               "OAuth enabled with valid credentials",
			oauthEnabled:       "true",
			githubClientID:     "test-client-id",
			githubClientSecret: "test-client-secret",
			expectError:        false,
		},
		{
			name:               "OAuth enabled missing client ID",
			oauthEnabled:       "true",
			githubClientID:     "",
			githubClientSecret: "test-client-secret",
			expectError:        true,
			errorContains:      "missing required variables",
		},
		{
			name:               "OAuth enabled missing client secret",
			oauthEnabled:       "true",
			githubClientID:     "test-client-id",
			githubClientSecret: "",
			expectError:        true,
			errorContains:      "missing required variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("OAUTH_ENABLED", tt.oauthEnabled)
			setEnvVar("OAUTH_GITHUB_CLIENT_ID", tt.githubClientID)
			setEnvVar("OAUTH_GITHUB_CLIENT_SECRET", tt.githubClientSecret)

			err := ValidateConditionalEnvVars()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectError && err != nil && tt.errorContains != "" {
				if !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestGenerateSecureKey(t *testing.T) {
	key, err := GenerateSecureKey()
	if err != nil {
		t.Fatalf("Failed to generate secure key: %v", err)
	}

	if len(key) != 64 {
		t.Errorf("Expected key length 64, got %d", len(key))
	}

	// 두 번 생성해서 다른 키인지 확인
	key2, err := GenerateSecureKey()
	if err != nil {
		t.Fatalf("Failed to generate second secure key: %v", err)
	}

	if key == key2 {
		t.Error("Generated keys should be different")
	}
}

// 헬퍼 함수들

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func setEnvVar(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	} else {
		_ = os.Unsetenv(key)
	}
}

func restoreEnvVar(key, originalValue string) {
	if originalValue != "" {
		_ = os.Setenv(key, originalValue)
	} else {
		_ = os.Unsetenv(key)
	}
}
