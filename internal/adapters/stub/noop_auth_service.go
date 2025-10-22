package stub

import (
	"context"
	"fmt"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// NoOpAuthService is a no-operation implementation of ports.AuthService
// Always allows all requests (no actual authentication)
type NoOpAuthService struct {
	logger logging.Logger
}

// NewNoOpAuthService creates a new NoOp auth service
func NewNoOpAuthService(logger logging.Logger) ports.AuthService {
	return &NoOpAuthService{logger: logger}
}

// Authenticate validates credentials (NoOp implementation - always succeeds)
func (as *NoOpAuthService) Authenticate(ctx context.Context, req *ports.AuthRequest) (*ports.AuthResponse, error) {
	as.logger.Debug("NoOpAuthService.Authenticate called (stub) - always allow",
		logging.F("username", req.Username))

	return &ports.AuthResponse{
		User: &ports.User{
			ID:       "anonymous",
			Username: req.Username,
			Email:    "anonymous@example.com",
			Role:     "user",
		},
		TokenPair:    nil, // No token needed for NoOp
		MFARequired:  false,
		MFAChallenge: "",
	}, nil
}

// Authorize checks permissions (NoOp implementation - always allow)
func (as *NoOpAuthService) Authorize(ctx context.Context, req *ports.AuthorizeRequest) (*ports.AuthorizeResponse, error) {
	as.logger.Debug("NoOpAuthService.Authorize called (stub) - always allow",
		logging.F("user_id", req.UserID),
		logging.F("resource", req.Resource),
		logging.F("action", req.Action))

	return &ports.AuthorizeResponse{
		Allowed: true,
		Reason:  "NoOpAuthService always allows",
	}, nil
}

// RefreshToken refreshes access token (NoOp implementation)
func (as *NoOpAuthService) RefreshToken(ctx context.Context, refreshToken string) (*ports.TokenPair, error) {
	as.logger.Debug("NoOpAuthService.RefreshToken called (stub)")

	return nil, fmt.Errorf("NoOpAuthService: RefreshToken not implemented")
}

// RevokeToken revokes a token (NoOp implementation)
func (as *NoOpAuthService) RevokeToken(ctx context.Context, token string) error {
	as.logger.Debug("NoOpAuthService.RevokeToken called (stub)")
	return nil
}

// ValidateToken validates token (NoOp implementation - always valid)
func (as *NoOpAuthService) ValidateToken(ctx context.Context, token string) (*ports.TokenClaims, error) {
	as.logger.Debug("NoOpAuthService.ValidateToken called (stub) - always valid")

	return &ports.TokenClaims{
		UserID:      "anonymous",
		Username:    "anonymous",
		Role:        "user",
		Permissions: []string{"read"},
	}, nil
}

// GetUser retrieves user information (NoOp implementation)
func (as *NoOpAuthService) GetUser(ctx context.Context, userID string) (*ports.User, error) {
	as.logger.Debug("NoOpAuthService.GetUser called (stub)",
		logging.F("user_id", userID))

	return &ports.User{
		ID:       userID,
		Username: "anonymous",
		Email:    "anonymous@example.com",
		Role:     "user",
	}, nil
}
