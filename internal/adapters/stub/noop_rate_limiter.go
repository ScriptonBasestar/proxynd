package stub

import (
	"context"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// NoOpRateLimiter is a no-operation implementation of ports.RateLimiter
// Always allows all requests (no rate limiting)
type NoOpRateLimiter struct {
	logger logging.Logger
}

// NewNoOpRateLimiter creates a new NoOp rate limiter
func NewNoOpRateLimiter(logger logging.Logger) ports.RateLimiter {
	return &NoOpRateLimiter{logger: logger}
}

// Allow checks if request is allowed (NoOp implementation - always allow)
func (rl *NoOpRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	rl.logger.Debug("NoOpRateLimiter.Allow called (stub) - always allow",
		logging.F("key", key))

	return true, nil
}

// AllowN checks if N requests are allowed (NoOp implementation - always allow)
func (rl *NoOpRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	rl.logger.Debug("NoOpRateLimiter.AllowN called (stub) - always allow",
		logging.F("key", key),
		logging.F("n", n))

	return true, nil
}

// Reset resets the rate limiter for key (NoOp implementation)
func (rl *NoOpRateLimiter) Reset(ctx context.Context, key string) error {
	rl.logger.Debug("NoOpRateLimiter.Reset called (stub)",
		logging.F("key", key))

	return nil
}

// Reserve reserves n tokens for a key (NoOp implementation - always succeeds)
func (rl *NoOpRateLimiter) Reserve(ctx context.Context, key string, n int) (*ports.Reservation, error) {
	rl.logger.Debug("NoOpRateLimiter.Reserve called (stub)",
		logging.F("key", key),
		logging.F("n", n))

	// Return nil reservation (not needed for NoOp)
	return nil, nil
}

// CheckLimit checks if a request should be rate limited (NoOp - never limits)
func (rl *NoOpRateLimiter) CheckLimit(ctx context.Context, req *ports.RateLimitRequest) error {
	rl.logger.Debug("NoOpRateLimiter.CheckLimit called (stub)",
		logging.F("key", req.Key))

	return nil
}

// GetConfig returns the current rate limit configuration (NoOp - returns nil)
func (rl *NoOpRateLimiter) GetConfig() *ports.RateLimitConfig {
	rl.logger.Debug("NoOpRateLimiter.GetConfig called (stub)")

	return nil
}
