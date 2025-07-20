// Package jwt provides JWT authentication service implementation
package jwt

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"proxynd/configs"
	"proxynd/logging"
)

// Claims JWT 클레임 구조체
type Claims struct {
	UserID        string   `json:"user_id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	Role          string   `json:"role"`
	Provider      string   `json:"provider"`
	Organizations []string `json:"organizations,omitempty"`
	TokenType     string   `json:"token_type"` // "access" 또는 "refresh"
	jwt.RegisteredClaims
}

// TokenPair 액세스 토큰과 리프레시 토큰 쌍
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// JWTService JWT 토큰 서비스
type JWTService struct {
	config *configs.OAuth2Config
	logger logging.Logger
}

// NewJWTService JWT 서비스 생성
func NewJWTService(config *configs.OAuth2Config) *JWTService {
	return &JWTService{
		config: config,
		logger: logging.GetLogger(),
	}
}

// GenerateTokenPair 액세스 토큰과 리프레시 토큰 쌍 생성
func (s *JWTService) GenerateTokenPair(
	userID, email, name, username, role, provider string,
	organizations []string,
) (*TokenPair, error) {
	now := time.Now()

	// 액세스 토큰 생성
	accessToken, accessExpiresAt, err := s.generateToken(
		userID, email, name, username, role, provider, organizations, "access",
		now, time.Duration(s.config.JWT.AccessTokenTTL)*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 리프레시 토큰 생성
	refreshToken, _, err := s.generateToken(
		userID, email, name, username, role, provider, organizations, "refresh",
		now, time.Duration(s.config.JWT.RefreshTokenTTL)*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.config.JWT.AccessTokenTTL,
		ExpiresAt:    accessExpiresAt,
	}, nil
}

// generateToken 개별 토큰 생성
func (s *JWTService) generateToken(
	userID, email, name, username, role, provider string,
	organizations []string, tokenType string, issuedAt time.Time, duration time.Duration,
) (string, time.Time, error) {
	expiresAt := issuedAt.Add(duration)

	// JWT ID 생성
	jti, err := s.generateJTI()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate JTI: %w", err)
	}

	claims := &Claims{
		UserID:        userID,
		Email:         email,
		Name:          name,
		Username:      username,
		Role:          role,
		Provider:      provider,
		Organizations: organizations,
		TokenType:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    s.config.JWT.Issuer,
			Audience:  jwt.ClaimStrings{s.config.JWT.Audience},
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(s.config.JWT.Algorithm), claims)

	// 알고리즘에 따라 적절한 키 사용
	var signingKey interface{}
	if s.config.JWT.Algorithm == "HS256" || s.config.JWT.Algorithm == "HS384" || s.config.JWT.Algorithm == "HS512" {
		signingKey = []byte(s.config.JWT.Secret)
	} else {
		// RS256, ES256 등을 위해서는 개인키가 필요하지만, 현재는 HMAC만 지원
		return "", time.Time{}, errors.New("unsupported signing algorithm")
	}

	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateToken 토큰 유효성 검증
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	// 토큰 파싱 및 검증
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 서명 알고리즘 확인
		if token.Method.Alg() != s.config.JWT.Algorithm {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// HMAC 키 반환
		if s.config.JWT.Algorithm == "HS256" || s.config.JWT.Algorithm == "HS384" || s.config.JWT.Algorithm == "HS512" {
			return []byte(s.config.JWT.Secret), nil
		}

		return nil, errors.New("unsupported signing algorithm")
	})

	if err != nil {
		s.logger.Warn("Token validation failed", logging.F("error", err))
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// 클레임 추출
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// 추가 검증
	if err := s.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("claim validation failed: %w", err)
	}

	return claims, nil
}

// ValidateAccessToken 액세스 토큰 유효성 검증
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, errors.New("not an access token")
	}

	return claims, nil
}

// ValidateRefreshToken 리프레시 토큰 유효성 검증
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, errors.New("not a refresh token")
	}

	return claims, nil
}

// OAuth2Provider OAuth2 제공자 인터페이스 (순환 import 방지를 위한 인터페이스)
type OAuth2Provider interface {
	GetUserOrganizations(ctx context.Context, accessToken string) ([]string, error)
}

// RefreshAccessToken 리프레시 토큰으로 새 액세스 토큰 생성 (기존 권한 유지)
func (s *JWTService) RefreshAccessToken(refreshTokenString string) (*TokenPair, error) {
	// 리프레시 토큰 검증
	refreshClaims, err := s.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// 새 토큰 쌍 생성 (기존 권한 유지)
	newTokenPair, err := s.GenerateTokenPair(
		refreshClaims.UserID,
		refreshClaims.Email,
		refreshClaims.Name,
		refreshClaims.Username,
		refreshClaims.Role,
		refreshClaims.Provider,
		refreshClaims.Organizations,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	s.logger.Info("Access token refreshed",
		logging.F("user_id", refreshClaims.UserID),
		logging.F("email", refreshClaims.Email))

	return newTokenPair, nil
}

// RefreshAccessTokenWithSync 리프레시 토큰으로 새 액세스 토큰 생성 (권한 동기화)
func (s *JWTService) RefreshAccessTokenWithSync(
	refreshTokenString string, oauth2Provider OAuth2Provider, oauth2AccessToken string,
) (*TokenPair, error) {
	// 리프레시 토큰 검증
	refreshClaims, err := s.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// OAuth2 제공자에서 최신 조직 정보 조회 (옵션)
	var updatedOrganizations []string
	if oauth2Provider != nil && oauth2AccessToken != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		orgs, err := oauth2Provider.GetUserOrganizations(ctx, oauth2AccessToken)
		if err != nil {
			// 에러 시 기존 조직 정보 사용 (fallback)
			s.logger.Warn("Failed to sync organization info, using cached data",
				logging.F("user_id", refreshClaims.UserID),
				logging.F("error", err))
			updatedOrganizations = refreshClaims.Organizations
		} else {
			updatedOrganizations = orgs
			s.logger.Debug("Organization info synced",
				logging.F("user_id", refreshClaims.UserID),
				logging.F("organizations", orgs))
		}
	} else {
		// OAuth2 제공자 정보가 없으면 기존 정보 사용
		updatedOrganizations = refreshClaims.Organizations
	}

	// 최신 조직 정보로 역할 재계산
	updatedRole := s.config.UserMapping.GetUserRoleWithPattern(refreshClaims.Email, updatedOrganizations)

	// 역할이 변경되었다면 로그 기록
	if updatedRole != refreshClaims.Role {
		s.logger.Info("User role updated during token refresh",
			logging.F("user_id", refreshClaims.UserID),
			logging.F("email", refreshClaims.Email),
			logging.F("old_role", refreshClaims.Role),
			logging.F("new_role", updatedRole))
	}

	// 새 토큰 쌍 생성 (업데이트된 권한 적용)
	newTokenPair, err := s.GenerateTokenPair(
		refreshClaims.UserID,
		refreshClaims.Email,
		refreshClaims.Name,
		refreshClaims.Username,
		updatedRole,
		refreshClaims.Provider,
		updatedOrganizations,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	s.logger.Info("Access token refreshed with role sync",
		logging.F("user_id", refreshClaims.UserID),
		logging.F("email", refreshClaims.Email),
		logging.F("role", updatedRole))

	return newTokenPair, nil
}

// ExtractUserInfo JWT에서 사용자 정보 추출
func (s *JWTService) ExtractUserInfo(claims *Claims) map[string]interface{} {
	return map[string]interface{}{
		"user_id":       claims.UserID,
		"email":         claims.Email,
		"name":          claims.Name,
		"username":      claims.Username,
		"role":          claims.Role,
		"provider":      claims.Provider,
		"organizations": claims.Organizations,
		"expires_at":    claims.ExpiresAt.Time,
		"issued_at":     claims.IssuedAt.Time,
	}
}

// IsTokenExpiringSoon 토큰이 곧 만료되는지 확인 (10분 이내)
func (s *JWTService) IsTokenExpiringSoon(claims *Claims) bool {
	if claims.ExpiresAt == nil {
		return true
	}
	return time.Until(claims.ExpiresAt.Time) < 10*time.Minute
}

// validateClaims 클레임 유효성 추가 검증
func (s *JWTService) validateClaims(claims *Claims) error {
	// 발급자 확인
	if claims.Issuer != s.config.JWT.Issuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", s.config.JWT.Issuer, claims.Issuer)
	}

	// 대상 확인
	expectedAudience := s.config.JWT.Audience
	if len(claims.Audience) == 0 || (len(claims.Audience) == 1 && claims.Audience[0] != expectedAudience) {
		return fmt.Errorf("invalid audience: expected %s", expectedAudience)
	}

	// 필수 필드 확인
	if claims.UserID == "" {
		return errors.New("missing user_id claim")
	}

	if claims.Email == "" {
		return errors.New("missing email claim")
	}

	if claims.Role == "" {
		return errors.New("missing role claim")
	}

	if claims.TokenType == "" {
		return errors.New("missing token_type claim")
	}

	// 토큰 타입 확인
	if claims.TokenType != "access" && claims.TokenType != "refresh" {
		return fmt.Errorf("invalid token_type: %s", claims.TokenType)
	}

	return nil
}

// generateJTI JWT ID 생성
func (s *JWTService) generateJTI() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetSigningKey 서명 키 반환 (테스트용)
func (s *JWTService) GetSigningKey() interface{} {
	return []byte(s.config.JWT.Secret)
}
