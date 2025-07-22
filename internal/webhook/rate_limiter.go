package webhook

import (
	"context"
	"sync"
	"time"

	"proxynd/internal/webhook/types"
)

// TokenBucketLimiter 토큰 버킷 기반 속도 제한기
type TokenBucketLimiter struct {
	mu         sync.Mutex
	tokens     int
	capacity   int
	refillRate int
	lastRefill time.Time
	ticker     *time.Ticker
	stopCh     chan struct{}
}

// NewTokenBucketLimiter 새로운 토큰 버킷 제한기 생성
func NewTokenBucketLimiter(refillRate, capacity int) (*TokenBucketLimiter, error) {
	if refillRate <= 0 {
		refillRate = 10 // 기본값: 초당 10개
	}
	if capacity <= 0 {
		capacity = refillRate * 2 // 기본값: 버스트 허용
	}

	limiter := &TokenBucketLimiter{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
		stopCh:     make(chan struct{}),
	}

	// 토큰 리필 고루틴 시작
	limiter.ticker = time.NewTicker(time.Second)
	go limiter.refillTokens()

	return limiter, nil
}

// Allow 즉시 토큰 사용 가능 여부 확인
func (tbl *TokenBucketLimiter) Allow() bool {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	if tbl.tokens > 0 {
		tbl.tokens--
		return true
	}
	return false
}

// Wait 토큰이 사용 가능할 때까지 대기
func (tbl *TokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		if tbl.Allow() {
			return nil
		}

		// 짧은 시간 대기
		select {
		case <-time.After(time.Millisecond * 100):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Tokens 현재 사용 가능한 토큰 수 반환
func (tbl *TokenBucketLimiter) Tokens() int {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	return tbl.tokens
}

// SetLimit 제한 설정 변경
func (tbl *TokenBucketLimiter) SetLimit(limit int) {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	tbl.refillRate = limit
	tbl.capacity = limit * 2 // 버스트 허용
}

// Stop 제한기 중지
func (tbl *TokenBucketLimiter) Stop() {
	if tbl.ticker != nil {
		tbl.ticker.Stop()
	}
	close(tbl.stopCh)
}

// refillTokens 토큰 리필 고루틴
func (tbl *TokenBucketLimiter) refillTokens() {
	defer tbl.ticker.Stop()

	for {
		select {
		case <-tbl.ticker.C:
			tbl.mu.Lock()
			// 초당 refillRate만큼 토큰 추가
			newTokens := tbl.tokens + tbl.refillRate
			if newTokens > tbl.capacity {
				newTokens = tbl.capacity
			}
			tbl.tokens = newTokens
			tbl.lastRefill = time.Now()
			tbl.mu.Unlock()

		case <-tbl.stopCh:
			return
		}
	}
}

// SlidingWindowLimiter 슬라이딩 윈도우 기반 속도 제한기
type SlidingWindowLimiter struct {
	mu       sync.Mutex
	requests []time.Time
	limit    int
	window   time.Duration
}

// NewSlidingWindowLimiter 새로운 슬라이딩 윈도우 제한기 생성
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

// Allow 요청 허용 여부 확인
func (swl *SlidingWindowLimiter) Allow() bool {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := time.Now()

	// 윈도우 밖의 오래된 요청 제거
	cutoff := now.Add(-swl.window)
	newRequests := make([]time.Time, 0)
	for _, reqTime := range swl.requests {
		if reqTime.After(cutoff) {
			newRequests = append(newRequests, reqTime)
		}
	}
	swl.requests = newRequests

	// 제한 확인
	if len(swl.requests) >= swl.limit {
		return false
	}

	// 새 요청 추가
	swl.requests = append(swl.requests, now)
	return true
}

// Wait 슬라이딩 윈도우에서 대기
func (swl *SlidingWindowLimiter) Wait(ctx context.Context) error {
	for {
		if swl.Allow() {
			return nil
		}

		select {
		case <-time.After(time.Millisecond * 100):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Tokens 현재 사용 가능한 요청 수 반환
func (swl *SlidingWindowLimiter) Tokens() int {
	swl.mu.Lock()
	defer swl.mu.Unlock()
	return swl.limit - len(swl.requests)
}

// SetLimit 제한 설정 변경
func (swl *SlidingWindowLimiter) SetLimit(limit int) {
	swl.mu.Lock()
	defer swl.mu.Unlock()
	swl.limit = limit
}

// CompositeRateLimiter 복합 속도 제한기 (여러 제한을 동시 적용)
type CompositeRateLimiter struct {
	limiters []types.RateLimiter
}

// NewCompositeRateLimiter 새로운 복합 제한기 생성
func NewCompositeRateLimiter(limiters ...types.RateLimiter) *CompositeRateLimiter {
	return &CompositeRateLimiter{
		limiters: limiters,
	}
}

// Allow 모든 제한기가 허용하는지 확인
func (crl *CompositeRateLimiter) Allow() bool {
	for _, limiter := range crl.limiters {
		if !limiter.Allow() {
			return false
		}
	}
	return true
}

// Wait 모든 제한기가 허용할 때까지 대기
func (crl *CompositeRateLimiter) Wait(ctx context.Context) error {
	for _, limiter := range crl.limiters {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Tokens 가장 제한적인 제한기의 토큰 수 반환
func (crl *CompositeRateLimiter) Tokens() int {
	minTokens := int(^uint(0) >> 1) // max int
	for _, limiter := range crl.limiters {
		tokens := limiter.Tokens()
		if tokens < minTokens {
			minTokens = tokens
		}
	}
	return minTokens
}

// SetLimit 모든 제한기의 제한 설정
func (crl *CompositeRateLimiter) SetLimit(limit int) {
	for _, limiter := range crl.limiters {
		limiter.SetLimit(limit)
	}
}
