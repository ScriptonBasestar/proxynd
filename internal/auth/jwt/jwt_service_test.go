package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"proxynd/internal/config"
)

const (
	// Test constants for JWT service tests
	testRoleAdmin        = "admin"
	testTokenTypeAccess  = "access"
	testTokenTypeRefresh = "refresh"
)

// 테스트용 OAuth2 설정 생성
func createTestConfig() *config.OAuth2Config {
	return &config.OAuth2Config{
		JWT: config.JWTConfig{
			Secret:          "test-jwt-secret-key-32-characters-long",
			AccessTokenTTL:  3600,
			RefreshTokenTTL: 604800,
			Algorithm:       "HS256",
			Issuer:          "proxynd-test",
			Audience:        "proxynd-api-test",
		},
	}
}

// TestNewJWTService JWT 서비스 생성 테스트
func TestNewJWTService(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	if service == nil {
		t.Fatal("JWT service should not be nil")
	}

	if service.config != config {
		t.Error("JWT service config should match provided config")
	}
}

// TestGenerateTokenPair 토큰 쌍 생성 테스트
func TestGenerateTokenPair(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	// 테스트 사용자 정보
	userID := "test-user-123"
	email := "test@example.com"
	name := "Test User"
	username := "testuser"
	role := testRoleAdmin
	provider := "github"
	organizations := []string{"test-org", "admin-org"}

	tokenPair, err := service.GenerateTokenPair(userID, email, name, username, role, provider, organizations)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 토큰 쌍 유효성 검증
	if tokenPair.AccessToken == "" {
		t.Error("Access token should not be empty")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}

	if tokenPair.TokenType != "Bearer" {
		t.Errorf("Expected token type 'Bearer', got '%s'", tokenPair.TokenType)
	}

	if tokenPair.ExpiresIn != config.JWT.AccessTokenTTL {
		t.Errorf("Expected expires_in %d, got %d", config.JWT.AccessTokenTTL, tokenPair.ExpiresIn)
	}

	// 토큰이 다른지 확인
	if tokenPair.AccessToken == tokenPair.RefreshToken {
		t.Error("Access token and refresh token should be different")
	}
}

// TestValidateAccessToken 액세스 토큰 검증 테스트
func TestValidateAccessToken(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	// 테스트 사용자 정보
	userID := "test-user-123"
	email := "test@example.com"
	name := "Test User"
	username := "testuser"
	role := testRoleAdmin
	provider := "github"
	organizations := []string{"test-org"}

	// 토큰 생성
	tokenPair, err := service.GenerateTokenPair(userID, email, name, username, role, provider, organizations)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 액세스 토큰 검증
	claims, err := service.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	// 클레임 검증
	if claims.UserID != userID {
		t.Errorf("Expected user_id %s, got %s", userID, claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("Expected email %s, got %s", email, claims.Email)
	}

	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}

	if claims.TokenType != testTokenTypeAccess {
		t.Errorf("Expected token_type '%s', got %s", testTokenTypeAccess, claims.TokenType)
	}

	if len(claims.Organizations) != len(organizations) {
		t.Errorf("Expected %d organizations, got %d", len(organizations), len(claims.Organizations))
	}
}

// TestValidateRefreshToken 리프레시 토큰 검증 테스트
func TestValidateRefreshToken(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	// 토큰 생성
	tokenPair, err := service.GenerateTokenPair("user123", "user@example.com", "User", "user", "viewer", "google", nil)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 리프레시 토큰 검증
	claims, err := service.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to validate refresh token: %v", err)
	}

	if claims.TokenType != testTokenTypeRefresh {
		t.Errorf("Expected token_type '%s', got %s", testTokenTypeRefresh, claims.TokenType)
	}
}

// TestRefreshAccessToken 액세스 토큰 갱신 테스트
func TestRefreshAccessToken(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	// 원본 토큰 생성
	originalTokenPair, err := service.GenerateTokenPair(
		"user123", "user@example.com", "User", "user", "developer", "gitlab",
		[]string{"dev-team"},
	)
	if err != nil {
		t.Fatalf("Failed to generate original token pair: %v", err)
	}

	// 토큰 갱신
	newTokenPair, err := service.RefreshAccessToken(originalTokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh access token: %v", err)
	}

	// 새 토큰이 다른지 확인
	if newTokenPair.AccessToken == originalTokenPair.AccessToken {
		t.Error("New access token should be different from original")
	}

	// 새 토큰 검증
	newClaims, err := service.ValidateAccessToken(newTokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate new access token: %v", err)
	}

	// 사용자 정보가 동일한지 확인
	if newClaims.UserID != "user123" {
		t.Error("User ID should be preserved after refresh")
	}

	if newClaims.Email != "user@example.com" {
		t.Error("Email should be preserved after refresh")
	}
}

// TestTokenValidationErrors 토큰 검증 오류 테스트
func TestTokenValidationErrors(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Empty token",
			token:       "",
			expectError: true,
		},
		{
			name:        "Invalid token format",
			token:       "invalid.token.format",
			expectError: true,
		},
		{
			name: "Token with wrong signature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
				"eyJ1c2VyX2lkIjoidGVzdCIsImV4cCI6OTk5OTk5OTk5OX0.invalid_signature",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateToken(tt.token)
			if tt.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestTokenExpiration 토큰 만료 테스트
func TestTokenExpiration(t *testing.T) {
	// 짧은 TTL로 설정
	config := createTestConfig()
	config.JWT.AccessTokenTTL = 1 // 1초
	service := NewJWTService(config)

	// 토큰 생성
	tokenPair, err := service.GenerateTokenPair("user123", "user@example.com", "User", "user", "viewer", "github", nil)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 즉시 검증 (성공해야 함)
	_, err = service.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Token should be valid immediately after creation: %v", err)
	}

	// 2초 대기 (토큰 만료)
	time.Sleep(2 * time.Second)

	// 만료된 토큰 검증 (실패해야 함)
	_, err = service.ValidateAccessToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("Expired token should not be valid")
	}
}

// TestWrongTokenType 잘못된 토큰 타입 테스트
func TestWrongTokenType(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	// 토큰 생성
	tokenPair, err := service.GenerateTokenPair("user123", "user@example.com", "User", "user", "viewer", "github", nil)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 리프레시 토큰을 액세스 토큰으로 검증 시도 (실패해야 함)
	_, err = service.ValidateAccessToken(tokenPair.RefreshToken)
	if err == nil {
		t.Error("Refresh token should not validate as access token")
	}

	// 액세스 토큰을 리프레시 토큰으로 검증 시도 (실패해야 함)
	_, err = service.ValidateRefreshToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("Access token should not validate as refresh token")
	}
}

// TestExtractUserInfo 사용자 정보 추출 테스트
func TestExtractUserInfo(t *testing.T) {
	config := createTestConfig()
	service := NewJWTService(config)

	// 토큰 생성
	tokenPair, err := service.GenerateTokenPair(
		"user123", "user@example.com", "Test User", "testuser", testRoleAdmin, "github",
		[]string{"admin-org"},
	)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 클레임 추출
	claims, err := service.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// 사용자 정보 추출
	userInfo := service.ExtractUserInfo(claims)

	// 검증
	if userInfo["user_id"] != "user123" {
		t.Errorf("Expected user_id 'user123', got %v", userInfo["user_id"])
	}

	if userInfo["email"] != "user@example.com" {
		t.Errorf("Expected email 'user@example.com', got %v", userInfo["email"])
	}

	if userInfo["role"] != testRoleAdmin {
		t.Errorf("Expected role 'admin', got %v", userInfo["role"])
	}
}

// TestIsTokenExpiringSoon 토큰 만료 임박 확인 테스트
func TestIsTokenExpiringSoon(t *testing.T) {
	config := createTestConfig()
	config.JWT.AccessTokenTTL = 300 // 5분
	service := NewJWTService(config)

	// 토큰 생성
	tokenPair, err := service.GenerateTokenPair("user123", "user@example.com", "User", "user", "viewer", "github", nil)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 클레임 추출
	claims, err := service.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// 5분(300초) 남은 토큰은 곧 만료됨 (10분 이내)
	if !service.IsTokenExpiringSoon(claims) {
		t.Error("Token with 5 minutes remaining should be expiring soon")
	}

	// 짧은 TTL로 새 토큰 생성
	config.JWT.AccessTokenTTL = 300 // 5분
	shortTokenPair, err := service.GenerateTokenPair(
		"user123", "user@example.com", "User", "user", "viewer", "github", nil,
	)
	if err != nil {
		t.Fatalf("Failed to generate short token pair: %v", err)
	}

	shortClaims, err := service.ValidateAccessToken(shortTokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate short token: %v", err)
	}

	// 수동으로 만료 시간을 5분 이내로 설정
	shortClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(5 * time.Minute))

	if !service.IsTokenExpiringSoon(shortClaims) {
		t.Error("Token expiring in 5 minutes should be expiring soon")
	}
}

// TestJWTConfiguration JWT 설정 테스트
func TestJWTConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		secret    string
		expectErr bool
	}{
		{
			name:      "Valid HS256",
			algorithm: "HS256",
			secret:    "test-secret-key-32-characters-long",
			expectErr: false,
		},
		{
			name:      "Valid HS384",
			algorithm: "HS384",
			secret:    "test-secret-key-32-characters-long",
			expectErr: false,
		},
		{
			name:      "Valid HS512",
			algorithm: "HS512",
			secret:    "test-secret-key-32-characters-long",
			expectErr: false,
		},
		{
			name:      "Unsupported RS256",
			algorithm: "RS256",
			secret:    "test-secret-key-32-characters-long",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &config.OAuth2Config{
				JWT: config.JWTConfig{
					Secret:          tt.secret,
					Algorithm:       tt.algorithm,
					Issuer:          "proxynd-test",
					Audience:        "proxynd-api-test",
					AccessTokenTTL:  3600,
					RefreshTokenTTL: 604800,
				},
			}

			service := NewJWTService(config)

			_, err := service.GenerateTokenPair("user123", "user@example.com", "User", "user", "viewer", "github", nil)

			if tt.expectErr && err == nil {
				t.Error("Expected error for unsupported algorithm, but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
