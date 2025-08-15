package ports

import (
	"context"
	"time"
)

// AuthService defines authentication and authorization interface
// TODO: Migrate from internal/interfaces/auth.go and internal/auth/
type AuthService interface {
	// Authenticate validates credentials and returns user info
	Authenticate(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
	
	// Authorize checks if user has permission for resource/action
	Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error)
	
	// RefreshToken refreshes access token using refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	
	// RevokeToken revokes a token
	RevokeToken(ctx context.Context, token string) error
	
	// ValidateToken validates token and returns claims
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	
	// GetUser retrieves user information
	GetUser(ctx context.Context, userID string) (*User, error)
}

// JWTService defines JWT operations interface
// TODO: Migrate from internal/auth/jwt/jwt_service.go
type JWTService interface {
	// GenerateTokenPair generates access and refresh tokens
	GenerateTokenPair(req *TokenGenerateRequest) (*TokenPair, error)
	
	// ValidateAccessToken validates access token
	ValidateAccessToken(token string) (*TokenClaims, error)
	
	// ValidateRefreshToken validates refresh token
	ValidateRefreshToken(token string) (*TokenClaims, error)
	
	// ExtractClaims extracts claims without validation
	ExtractClaims(token string) (*TokenClaims, error)
	
	// RevokeToken adds token to revocation list
	RevokeToken(token string) error
	
	// IsTokenRevoked checks if token is revoked
	IsTokenRevoked(token string) (bool, error)
}

// OAuth2Service defines OAuth2 provider interface
// TODO: Migrate from internal/auth/oauth2/
type OAuth2Service interface {
	// GetAuthURL returns OAuth2 authorization URL
	GetAuthURL(state string, scopes []string) string
	
	// ExchangeCode exchanges authorization code for tokens
	ExchangeCode(ctx context.Context, code, state string) (*OAuth2Token, error)
	
	// GetUserInfo gets user info from OAuth2 provider
	GetUserInfo(ctx context.Context, token *OAuth2Token) (*OAuth2UserInfo, error)
	
	// RefreshToken refreshes OAuth2 token
	RefreshToken(ctx context.Context, refreshToken string) (*OAuth2Token, error)
	
	// GetProviderName returns provider name (github, gitlab, google)
	GetProviderName() string
}

// MFAService defines multi-factor authentication interface
// TODO: Migrate from internal/auth/mfa/mfa_service.go
type MFAService interface {
	// GenerateSecret generates MFA secret for user
	GenerateSecret(userID string) (*MFASecret, error)
	
	// VerifyCode verifies MFA code
	VerifyCode(userID, code string) (bool, error)
	
	// EnableMFA enables MFA for user
	EnableMFA(ctx context.Context, userID, code string) error
	
	// DisableMFA disables MFA for user
	DisableMFA(ctx context.Context, userID string) error
	
	// IsMFAEnabled checks if MFA is enabled for user
	IsMFAEnabled(ctx context.Context, userID string) (bool, error)
	
	// GenerateBackupCodes generates backup codes
	GenerateBackupCodes(userID string) ([]string, error)
}

// UserService defines user management interface
// TODO: Create interface for user operations
type UserService interface {
	// CreateUser creates new user
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	
	// UpdateUser updates user information
	UpdateUser(ctx context.Context, userID string, req *UpdateUserRequest) (*User, error)
	
	// GetUser retrieves user by ID
	GetUser(ctx context.Context, userID string) (*User, error)
	
	// GetUserByEmail retrieves user by email
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	
	// ListUsers lists users with pagination
	ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error)
	
	// DeleteUser deletes user
	DeleteUser(ctx context.Context, userID string) error
	
	// SetUserRole sets user role
	SetUserRole(ctx context.Context, userID, role string) error
}

// AuthRequest represents authentication request
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Provider string `json:"provider"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	User         *User      `json:"user"`
	TokenPair    *TokenPair `json:"tokens"`
	MFARequired  bool       `json:"mfa_required"`
	MFAChallenge string     `json:"mfa_challenge,omitempty"`
}

// AuthorizeRequest represents authorization request
type AuthorizeRequest struct {
	UserID   string `json:"user_id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Context  string `json:"context,omitempty"`
}

// AuthorizeResponse represents authorization response
type AuthorizeResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID        string   `json:"user_id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	Role          string   `json:"role"`
	Permissions   []string `json:"permissions"`
	Organizations []string `json:"organizations"`
	Provider      string   `json:"provider"`
	TokenType     string   `json:"token_type"`
	ExpiresAt     int64    `json:"exp"`
	IssuedAt      int64    `json:"iat"`
}

// TokenGenerateRequest represents token generation request
type TokenGenerateRequest struct {
	UserID        string   `json:"user_id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	Role          string   `json:"role"`
	Permissions   []string `json:"permissions"`
	Organizations []string `json:"organizations"`
	Provider      string   `json:"provider"`
}

// User represents user information
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	Role          string    `json:"role"`
	Permissions   []string  `json:"permissions"`
	Organizations []string  `json:"organizations"`
	Active        bool      `json:"active"`
	MFAEnabled    bool      `json:"mfa_enabled"`
	LastLogin     time.Time `json:"last_login"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateUserRequest represents user creation request
type CreateUserRequest struct {
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Password      string   `json:"password"`
	Role          string   `json:"role"`
	Permissions   []string `json:"permissions"`
	Organizations []string `json:"organizations"`
}

// UpdateUserRequest represents user update request
type UpdateUserRequest struct {
	Name          *string   `json:"name,omitempty"`
	Email         *string   `json:"email,omitempty"`
	Role          *string   `json:"role,omitempty"`
	Permissions   *[]string `json:"permissions,omitempty"`
	Organizations *[]string `json:"organizations,omitempty"`
	Active        *bool     `json:"active,omitempty"`
}

// ListUsersRequest represents user listing request
type ListUsersRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Role     string `json:"role,omitempty"`
	Active   *bool  `json:"active,omitempty"`
	Search   string `json:"search,omitempty"`
}

// ListUsersResponse represents user listing response
type ListUsersResponse struct {
	Users    []*User `json:"users"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// OAuth2Token represents OAuth2 token
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

// OAuth2UserInfo represents OAuth2 user information
type OAuth2UserInfo struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	AvatarURL     string   `json:"avatar_url"`
	Organizations []string `json:"organizations"`
	Provider      string   `json:"provider"`
}

// MFASecret represents MFA secret information
type MFASecret struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// Permission represents a permission
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// Role represents a role with permissions
type Role struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Permissions []*Permission `json:"permissions"`
}

// Session represents user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}