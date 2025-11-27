// Package helpers provides test helper functions
package helpers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTTestConfig contains configuration for generating test JWT tokens
type JWTTestConfig struct {
	SecretKey string
	Issuer    string
	Audience  string
	Algorithm string
	ExpiresIn time.Duration
}

// DefaultJWTTestConfig returns default test JWT configuration
func DefaultJWTTestConfig() JWTTestConfig {
	return JWTTestConfig{
		SecretKey: "test-secret-key-for-testing",
		Issuer:    "proxynd-test",
		Audience:  "proxynd",
		Algorithm: "HS256",
		ExpiresIn: time.Hour,
	}
}

// JWTTestClaims represents claims for test JWT tokens
// This must match the structure in middleware/jwt_auth.go JWTClaims
type JWTTestClaims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"iat"` // Custom int64 field for middleware compatibility
	ExpiresAt int64    `json:"exp"` // Custom int64 field for middleware compatibility
	jwt.RegisteredClaims
}

// GenerateTestJWTToken generates a JWT token for testing purposes
// The token structure matches what middleware/jwt_auth.go JWTMiddleware expects
func GenerateTestJWTToken(userID, username string, roles []string, cfg JWTTestConfig) (string, error) {
	now := time.Now()

	// Default expiration to 1 hour if not set
	expiresIn := cfg.ExpiresIn
	if expiresIn == 0 {
		expiresIn = time.Hour
	}
	expiresAt := now.Add(expiresIn)

	claims := &JWTTestClaims{
		UserID:    userID,
		Username:  username,
		Roles:     roles,
		IssuedAt:  now.Unix(),       // Custom int64 field
		ExpiresAt: expiresAt.Unix(), // Custom int64 field
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	// Default to HS256 if not specified
	algorithm := cfg.Algorithm
	if algorithm == "" {
		algorithm = "HS256"
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(algorithm), claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

// GenerateTestJWTTokenWithDefaults generates a JWT token using default test configuration
func GenerateTestJWTTokenWithDefaults(userID, username string, roles []string) (string, error) {
	return GenerateTestJWTToken(userID, username, roles, DefaultJWTTestConfig())
}

// GenerateExpiredTestJWTToken generates an expired JWT token for testing
func GenerateExpiredTestJWTToken(userID, username string, roles []string, cfg JWTTestConfig) (string, error) {
	// Set expiration to past
	cfg.ExpiresIn = -time.Hour
	return GenerateTestJWTToken(userID, username, roles, cfg)
}
