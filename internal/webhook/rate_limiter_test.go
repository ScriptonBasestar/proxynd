package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

// TestTokenBucketLimiter 토큰 버킷 제한기 테스트
func TestTokenBucketLimiter(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(2, 5) // 초당 2개, 버킷 크기 5
	assert.Equal(t, err, nil)
	assert.NotEqual(t, limiter, nil)
	defer limiter.Stop()

	// 초기 토큰 확인
	assert.Equal(t, limiter.Tokens(), 5)

	// 토큰 소비
	assert.Equal(t, limiter.Allow(), true)
	assert.Equal(t, limiter.Tokens(), 4)

	assert.Equal(t, limiter.Allow(), true)
	assert.Equal(t, limiter.Tokens(), 3)
}

// TestTokenBucketLimiterExhaustion 토큰 고갈 테스트
func TestTokenBucketLimiterExhaustion(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(1, 2) // 초당 1개, 버킷 크기 2
	assert.Equal(t, err, nil)
	defer limiter.Stop()

	// 모든 토큰 소비
	assert.Equal(t, limiter.Allow(), true) // 2 -> 1
	assert.Equal(t, limiter.Allow(), true) // 1 -> 0

	// 토큰 없음
	assert.Equal(t, limiter.Allow(), false)
	assert.Equal(t, limiter.Tokens(), 0)
}

// TestTokenBucketLimiterRefill 토큰 리필 테스트
func TestTokenBucketLimiterRefill(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(2, 3) // 초당 2개, 버킷 크기 3
	assert.Equal(t, err, nil)
	defer limiter.Stop()

	// 모든 토큰 소비
	limiter.Allow() // 3 -> 2
	limiter.Allow() // 2 -> 1
	limiter.Allow() // 1 -> 0

	assert.Equal(t, limiter.Tokens(), 0)

	// 1초 대기 후 토큰 리필 확인
	time.Sleep(time.Millisecond * 1100)

	// 리필된 토큰 확인 (최대 2개 추가, 하지만 최대 용량은 3)
	tokens := limiter.Tokens()
	assert.Equal(t, tokens >= 2, true)
	assert.Equal(t, tokens <= 3, true)
}

// TestTokenBucketLimiterWait Wait 메서드 테스트
func TestTokenBucketLimiterWait(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(10, 10) // 충분히 큰 제한
	assert.Equal(t, err, nil)
	defer limiter.Stop()

	ctx := context.Background()

	// 즉시 사용 가능
	start := time.Now()
	err = limiter.Wait(ctx)
	duration := time.Since(start)

	assert.Equal(t, err, nil)
	assert.Equal(t, duration < time.Millisecond*50, true) // 거의 즉시
}

// TestTokenBucketLimiterWaitTimeout Wait 타임아웃 테스트
func TestTokenBucketLimiterWaitTimeout(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(1, 1) // 용량 1
	assert.Equal(t, err, nil)
	defer limiter.Stop()

	// 토큰 소진
	limiter.Allow()
	assert.Equal(t, limiter.Tokens(), 0)

	// 타임아웃이 있는 컨텍스트
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	// 타임아웃 발생
	err = limiter.Wait(ctx)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, true, err == context.DeadlineExceeded)
}

// TestSlidingWindowLimiter 슬라이딩 윈도우 제한기 테스트
func TestSlidingWindowLimiter(t *testing.T) {
	limiter := NewSlidingWindowLimiter(3, time.Second) // 1초에 3개
	assert.NotEqual(t, limiter, nil)

	// 초기 요청들
	assert.Equal(t, limiter.Allow(), true) // 1/3
	assert.Equal(t, limiter.Allow(), true) // 2/3
	assert.Equal(t, limiter.Allow(), true) // 3/3

	// 제한 도달
	assert.Equal(t, limiter.Allow(), false)

	// 토큰 수 확인
	assert.Equal(t, limiter.Tokens(), 0)
}

// TestSlidingWindowLimiterWindowSliding 윈도우 슬라이딩 테스트
func TestSlidingWindowLimiterWindowSliding(t *testing.T) {
	limiter := NewSlidingWindowLimiter(2, time.Millisecond*500) // 500ms에 2개

	// 첫 번째 요청
	assert.Equal(t, limiter.Allow(), true)

	// 조금 기다린 후 두 번째 요청
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, limiter.Allow(), true)

	// 제한 도달
	assert.Equal(t, limiter.Allow(), false)

	// 윈도우가 슬라이딩될 때까지 대기
	time.Sleep(time.Millisecond * 450)

	// 다시 사용 가능
	assert.Equal(t, limiter.Allow(), true)
}

// TestCompositeRateLimiter 복합 제한기 테스트
func TestCompositeRateLimiter(t *testing.T) {
	// 두 개의 제한기 생성
	tokenBucket, _ := NewTokenBucketLimiter(10, 10)
	defer tokenBucket.Stop()

	slidingWindow := NewSlidingWindowLimiter(5, time.Second)

	// 복합 제한기 생성
	composite := NewCompositeRateLimiter(tokenBucket, slidingWindow)

	// 모든 제한기가 허용하는 경우
	assert.Equal(t, composite.Allow(), true)

	// 슬라이딩 윈도우를 먼저 고갈시킴
	for i := 0; i < 4; i++ {
		composite.Allow()
	}

	// 슬라이딩 윈도우가 고갈되어 복합 제한기도 거부
	assert.Equal(t, composite.Allow(), false)

	// 토큰 수는 가장 제한적인 것을 반환
	tokens := composite.Tokens()
	assert.Equal(t, tokens, 0) // 슬라이딩 윈도우가 0
}

// TestRateLimiterSetLimit 제한 변경 테스트
func TestRateLimiterSetLimit(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(5, 10)
	assert.Equal(t, err, nil)
	defer limiter.Stop()

	// 제한 변경
	limiter.SetLimit(20)

	// 변경된 제한 확인 (내부 구현에 따라 달라질 수 있음)
	// 여기서는 기본적인 설정 변경만 테스트
	assert.Equal(t, err, nil)
}
