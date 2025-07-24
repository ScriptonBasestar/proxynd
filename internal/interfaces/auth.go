package interfaces

import (
	"context"
	"time"
)

// JWTService defines the interface for JWT operations
type JWTService interface {
	// GenerateTokenPair generates access and refresh tokens
	GenerateTokenPair(userID, email, name, username, role, provider string, organizations []string) (*TokenPair, error)

	// ValidateToken validates any JWT token
	ValidateToken(tokenString string) (*Claims, error)

	// ValidateAccessToken validates specifically an access token
	ValidateAccessToken(tokenString string) (*Claims, error)

	// RefreshAccessToken generates new access token from refresh token
	RefreshAccessToken(refreshTokenString string) (*TokenPair, error)

	// RevokeToken revokes a token
	RevokeToken(tokenString string) error

	// ExtractClaims extracts claims without validation
	ExtractClaims(tokenString string) (*Claims, error)
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Claims represents JWT claims
type Claims struct {
	UserID        string   `json:"user_id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	Role          string   `json:"role"`
	Provider      string   `json:"provider"`
	Organizations []string `json:"organizations"`
	TokenType     string   `json:"token_type"`
	ExpiresAt     int64    `json:"exp"`
	IssuedAt      int64    `json:"iat"`
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Authenticate validates credentials and returns user info
	Authenticate(ctx context.Context, credentials Credentials) (*User, error)

	// AuthorizeRequest checks if request is authorized
	AuthorizeRequest(ctx context.Context, token, resource, action string) (bool, error)

	// GetUser retrieves user information by ID
	GetUser(ctx context.Context, userID string) (*User, error)

	// CreateUser creates a new user
	CreateUser(ctx context.Context, user *User) error

	// UpdateUser updates user information
	UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error
}

// Credentials represents authentication credentials
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Provider string `json:"provider"`
}

// User represents user information
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	Role          string    `json:"role"`
	Organizations []string  `json:"organizations"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
