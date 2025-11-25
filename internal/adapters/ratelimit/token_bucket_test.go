package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"proxynd/internal/ports"
)

// mockLogger implements ports.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...ports.Field) {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...ports.Field)  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...ports.Field)  {}
func (m *mockLogger) Error(ctx context.Context, msg string, fields ...ports.Field) {}
func (m *mockLogger) Fatal(ctx context.Context, msg string, fields ...ports.Field) {}
func (m *mockLogger) With(fields ...ports.Field) ports.Logger                      { return m }
func (m *mockLogger) WithContext(ctx context.Context) ports.Logger                 { return m }

func TestTokenBucketRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		requestsPerSecond float64
		burstSize         int
		requests          int
		expectAllowed     int
	}{
		{
			name:              "all requests allowed within burst",
			requestsPerSecond: 10,
			burstSize:         5,
			requests:          5,
			expectAllowed:     5,
		},
		{
			name:              "requests exceed burst limit",
			requestsPerSecond: 10,
			burstSize:         3,
			requests:          5,
			expectAllowed:     3,
		},
		{
			name:              "single request allowed",
			requestsPerSecond: 100,
			burstSize:         10,
			requests:          1,
			expectAllowed:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &ports.RateLimitConfig{
				RequestsPerSecond: tt.requestsPerSecond,
				BurstSize:         tt.burstSize,
			}

			rl := NewTokenBucketRateLimiter(config, &mockLogger{})
			ctx := context.Background()

			allowed := 0
			for i := 0; i < tt.requests; i++ {
				ok, err := rl.Allow(ctx, "test-key")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if ok {
					allowed++
				}
			}

			if allowed != tt.expectAllowed {
				t.Errorf("expected %d allowed requests, got %d", tt.expectAllowed, allowed)
			}
		})
	}
}

func TestTokenBucketRateLimiter_AllowN(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})
	ctx := context.Background()

	// Request 3 tokens - should succeed
	ok, err := rl.AllowN(ctx, "test-key", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected 3 tokens to be allowed")
	}

	// Request 3 more tokens - should fail (only 2 left)
	ok, err = rl.AllowN(ctx, "test-key", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected 3 tokens to be denied (only 2 left)")
	}

	// Request 2 tokens - should succeed
	ok, err = rl.AllowN(ctx, "test-key", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected 2 tokens to be allowed")
	}
}

func TestTokenBucketRateLimiter_TokenRefill(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 1000, // 1000 requests per second = 1 per millisecond
		BurstSize:         2,
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})
	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 2; i++ {
		ok, _ := rl.Allow(ctx, "test-key")
		if !ok {
			t.Errorf("expected token %d to be allowed", i)
		}
	}

	// Should be denied immediately
	ok, _ := rl.Allow(ctx, "test-key")
	if ok {
		t.Error("expected request to be denied (no tokens)")
	}

	// Wait for token refill (at 1000/s, 2ms should give ~2 tokens)
	time.Sleep(3 * time.Millisecond)

	// Should now be allowed
	ok, _ = rl.Allow(ctx, "test-key")
	if !ok {
		t.Error("expected request to be allowed after refill")
	}
}

func TestTokenBucketRateLimiter_MultipleKeys(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})
	ctx := context.Background()

	// Key 1 - consume all tokens
	rl.Allow(ctx, "key1")
	rl.Allow(ctx, "key1")
	ok, _ := rl.Allow(ctx, "key1")
	if ok {
		t.Error("expected key1 to be rate limited")
	}

	// Key 2 - should still have tokens
	ok, _ = rl.Allow(ctx, "key2")
	if !ok {
		t.Error("expected key2 to be allowed (separate bucket)")
	}
}

func TestTokenBucketRateLimiter_Reset(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})
	ctx := context.Background()

	// Consume all tokens
	rl.Allow(ctx, "test-key")
	rl.Allow(ctx, "test-key")
	ok, _ := rl.Allow(ctx, "test-key")
	if ok {
		t.Error("expected to be rate limited")
	}

	// Reset
	err := rl.Reset(ctx, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should now be allowed
	ok, _ = rl.Allow(ctx, "test-key")
	if !ok {
		t.Error("expected request to be allowed after reset")
	}
}

func TestTokenBucketRateLimiter_Reserve(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})
	ctx := context.Background()

	// Reserve 1 token - should succeed
	reservation, err := rl.Reserve(ctx, "test-key", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reservation.OK {
		t.Error("expected reservation to be OK")
	}
	if reservation.Delay != 0 {
		t.Error("expected no delay for immediate reservation")
	}

	// Reserve 2 more - only 1 token left, should fail
	reservation, err = rl.Reserve(ctx, "test-key", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reservation.OK {
		t.Error("expected reservation to fail (not enough tokens)")
	}
	if reservation.Delay == 0 {
		t.Error("expected delay for failed reservation")
	}
}

func TestTokenBucketRateLimiter_CheckLimit(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})
	ctx := context.Background()

	// First two requests should pass
	req := &ports.RateLimitRequest{
		Key:      "test-key",
		ClientIP: "192.168.1.1",
		Count:    1,
	}

	err := rl.CheckLimit(ctx, req)
	if err != nil {
		t.Errorf("expected first request to pass: %v", err)
	}

	err = rl.CheckLimit(ctx, req)
	if err != nil {
		t.Errorf("expected second request to pass: %v", err)
	}

	// Third request should fail
	err = rl.CheckLimit(ctx, req)
	if err == nil {
		t.Error("expected third request to fail")
	}
}

func TestTokenBucketRateLimiter_Concurrent(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         50,
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0

	// Spawn 100 concurrent goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := rl.Allow(ctx, "concurrent-key")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// With burst of 50, we should see around 50 allowed (may vary slightly due to timing)
	if allowed > 55 || allowed < 45 {
		t.Errorf("expected around 50 allowed requests, got %d", allowed)
	}
}

func TestTokenBucketRateLimiter_GetConfig(t *testing.T) {
	t.Parallel()

	config := &ports.RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         20,
		SkipSuccessful:    true,
		TrustedProxies:    []string{"10.0.0.1"},
	}

	rl := NewTokenBucketRateLimiter(config, &mockLogger{})

	gotConfig := rl.GetConfig()
	if gotConfig.RequestsPerSecond != config.RequestsPerSecond {
		t.Errorf("expected RequestsPerSecond %f, got %f", config.RequestsPerSecond, gotConfig.RequestsPerSecond)
	}
	if gotConfig.BurstSize != config.BurstSize {
		t.Errorf("expected BurstSize %d, got %d", config.BurstSize, gotConfig.BurstSize)
	}
}

func TestTokenBucketRateLimiter_DefaultConfig(t *testing.T) {
	t.Parallel()

	rl := NewTokenBucketRateLimiter(nil, &mockLogger{})

	config := rl.GetConfig()
	if config.RequestsPerSecond != 100 {
		t.Errorf("expected default RequestsPerSecond 100, got %f", config.RequestsPerSecond)
	}
	if config.BurstSize != 20 {
		t.Errorf("expected default BurstSize 20, got %d", config.BurstSize)
	}
}
