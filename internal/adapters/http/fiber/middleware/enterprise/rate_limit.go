// Package enterprise provides enterprise middleware components
package enterprise

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimitConfig defines configuration for rate limiting
type RateLimitConfig struct {
	// Max requests per window
	Max int

	// Time window duration
	Window time.Duration

	// Burst size (extra requests allowed temporarily)
	Burst int

	// KeyGenerator generates a unique key for rate limiting
	// Default: IP address
	KeyGenerator func(c *fiber.Ctx) string

	// Handler called when rate limit is exceeded
	// Default: returns 429 Too Many Requests
	LimitReached fiber.Handler

	// Skip allows conditional skipping of rate limiting
	Skip func(c *fiber.Ctx) bool
}

// RateLimiter manages rate limiting for requests
type RateLimiter struct {
	config  RateLimitConfig
	storage sync.Map // map[string]*limitEntry
	mu      sync.RWMutex
}

// limitEntry tracks request counts for a specific key
type limitEntry struct {
	count     int
	resetTime time.Time
	burstUsed int
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter middleware
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	// Set defaults
	if config.Max == 0 {
		config.Max = 100 // 100 requests
	}
	if config.Window == 0 {
		config.Window = 1 * time.Minute // per minute
	}
	if config.Burst == 0 {
		config.Burst = 10 // allow 10 extra burst requests
	}
	if config.KeyGenerator == nil {
		config.KeyGenerator = func(c *fiber.Ctx) string {
			return c.IP() // Default: use IP address
		}
	}
	if config.LimitReached == nil {
		config.LimitReached = func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
				"metadata": fiber.Map{
					"timestamp":  time.Now().UTC(),
					"request_id": c.Locals("requestid"),
				},
			})
		}
	}

	limiter := &RateLimiter{
		config: config,
	}

	// Start cleanup goroutine to remove expired entries
	go limiter.cleanup()

	return limiter
}

// Middleware returns the rate limiting middleware handler
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if we should skip rate limiting
		if rl.config.Skip != nil && rl.config.Skip(c) {
			return c.Next()
		}

		// Generate key for this request
		key := rl.config.KeyGenerator(c)

		// Check rate limit
		allowed, remaining, resetTime := rl.checkLimit(key)

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.Max))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		if !allowed {
			c.Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
			return rl.config.LimitReached(c)
		}

		return c.Next()
	}
}

// checkLimit checks if the request is within rate limit
// Returns: (allowed bool, remaining int, resetTime time.Time)
func (rl *RateLimiter) checkLimit(key string) (bool, int, time.Time) {
	now := time.Now()

	// Get or create entry
	val, _ := rl.storage.LoadOrStore(key, &limitEntry{
		count:     0,
		resetTime: now.Add(rl.config.Window),
		burstUsed: 0,
	})

	entry := val.(*limitEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Check if window has expired
	if now.After(entry.resetTime) {
		// Reset the window
		entry.count = 0
		entry.burstUsed = 0
		entry.resetTime = now.Add(rl.config.Window)
	}

	// Calculate total allowed (max + burst)
	totalAllowed := rl.config.Max + rl.config.Burst

	// Check if we're within limits
	if entry.count < rl.config.Max {
		// Within normal limit
		entry.count++
		remaining := rl.config.Max - entry.count
		return true, remaining, entry.resetTime
	} else if entry.count < totalAllowed {
		// Using burst capacity
		entry.count++
		entry.burstUsed++
		remaining := 0 // No normal capacity left
		return true, remaining, entry.resetTime
	}

	// Rate limit exceeded
	return false, 0, entry.resetTime
}

// cleanup periodically removes expired entries from storage
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		rl.storage.Range(func(key, value interface{}) bool {
			entry := value.(*limitEntry)
			entry.mu.Lock()
			expired := now.After(entry.resetTime.Add(rl.config.Window))
			entry.mu.Unlock()

			if expired {
				rl.storage.Delete(key)
			}
			return true
		})
	}
}

// Reset clears rate limit for a specific key
func (rl *RateLimiter) Reset(key string) {
	rl.storage.Delete(key)
}

// ResetAll clears all rate limits
func (rl *RateLimiter) ResetAll() {
	rl.storage.Range(func(key, value interface{}) bool {
		rl.storage.Delete(key)
		return true
	})
}

// DefaultEnterpriseRateLimiter creates a rate limiter with sensible defaults for enterprise API
func DefaultEnterpriseRateLimiter() *RateLimiter {
	return NewRateLimiter(RateLimitConfig{
		Max:    1000,            // 1000 requests
		Window: 1 * time.Minute, // per minute
		Burst:  100,             // allow 100 extra burst requests
	})
}

// StrictEnterpriseRateLimiter creates a stricter rate limiter for sensitive endpoints
func StrictEnterpriseRateLimiter() *RateLimiter {
	return NewRateLimiter(RateLimitConfig{
		Max:    100,             // 100 requests
		Window: 1 * time.Minute, // per minute
		Burst:  10,              // allow 10 extra burst requests
	})
}

// PerUserRateLimiter creates a rate limiter based on user ID
func PerUserRateLimiter(max int, window time.Duration) *RateLimiter {
	burst := max / 10 // 10% burst allowance
	if burst < 1 {
		burst = 1 // Minimum 1 burst request
	}
	return NewRateLimiter(RateLimitConfig{
		Max:    max,
		Window: window,
		Burst:  burst,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Try to get user ID from context
			userID := c.Locals("user_id")
			if userID != nil {
				if id, ok := userID.(string); ok && id != "" {
					return "user:" + id
				}
			}
			// Fallback to IP if no user ID
			return "ip:" + c.IP()
		},
	})
}

// PerIPRateLimiter creates a rate limiter based on IP address
func PerIPRateLimiter(max int, window time.Duration) *RateLimiter {
	burst := max / 10 // 10% burst allowance
	if burst < 1 {
		burst = 1 // Minimum 1 burst request
	}
	return NewRateLimiter(RateLimitConfig{
		Max:    max,
		Window: window,
		Burst:  burst,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ip:" + c.IP()
		},
	})
}
