// Package goroutine provides safe goroutine management utilities
package goroutine

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// SafeGoFunc panic을 복구하는 안전한 고루틴 실행 함수
type SafeGoFunc func(ctx context.Context)

// PanicHandler panic 처리 핸들러
type PanicHandler func(recovered interface{}, stack []byte)

// defaultPanicHandler 기본 panic 핸들러
func defaultPanicHandler(recovered interface{}, stack []byte) {
	fmt.Printf("Panic recovered in goroutine: %v\nStack trace:\n%s\n", recovered, stack)
}

var (
	globalPanicHandler PanicHandler = defaultPanicHandler
	handlerMu          sync.RWMutex
)

// SetGlobalPanicHandler 전역 panic 핸들러 설정
func SetGlobalPanicHandler(handler PanicHandler) {
	handlerMu.Lock()
	defer handlerMu.Unlock()
	if handler != nil {
		globalPanicHandler = handler
	}
}

// GetGlobalPanicHandler 전역 panic 핸들러 조회
func GetGlobalPanicHandler() PanicHandler {
	handlerMu.RLock()
	defer handlerMu.RUnlock()
	return globalPanicHandler
}

// SafeGo panic을 복구하는 안전한 고루틴 실행
func SafeGo(ctx context.Context, fn SafeGoFunc) {
	SafeGoWithHandler(ctx, fn, nil)
}

// SafeGoWithHandler 커스텀 panic 핸들러와 함께 안전한 고루틴 실행
func SafeGoWithHandler(ctx context.Context, fn SafeGoFunc, handler PanicHandler) {
	if handler == nil {
		handler = GetGlobalPanicHandler()
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				handler(r, stack)
			}
		}()

		fn(ctx)
	}()
}

// SafeGoGroup 여러 고루틴을 안전하게 실행하고 대기하는 그룹
type SafeGoGroup struct {
	ctx     context.Context
	wg      sync.WaitGroup
	handler PanicHandler
	errors  chan error
	once    sync.Once
}

// NewSafeGoGroup 새로운 SafeGoGroup 생성
func NewSafeGoGroup(ctx context.Context) *SafeGoGroup {
	return &SafeGoGroup{
		ctx:     ctx,
		handler: GetGlobalPanicHandler(),
		errors:  make(chan error, 10), // 버퍼 크기 10
	}
}

// NewSafeGoGroupWithHandler 커스텀 핸들러와 함께 SafeGoGroup 생성
func NewSafeGoGroupWithHandler(ctx context.Context, handler PanicHandler) *SafeGoGroup {
	return &SafeGoGroup{
		ctx:     ctx,
		handler: handler,
		errors:  make(chan error, 10),
	}
}

// Go 안전한 고루틴 추가
func (g *SafeGoGroup) Go(fn func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				g.handler(r, stack)

				// panic을 에러로 변환
				err := fmt.Errorf("panic recovered: %v", r)
				select {
				case g.errors <- err:
				default:
					// 에러 채널이 가득 찬 경우 무시
				}
			}
		}()

		if err := fn(); err != nil {
			select {
			case g.errors <- err:
			default:
				// 에러 채널이 가득 찬 경우 무시
			}
		}
	}()
}

// GoWithContext 컨텍스트를 받는 안전한 고루틴 추가
func (g *SafeGoGroup) GoWithContext(fn func(ctx context.Context) error) {
	g.Go(func() error {
		return fn(g.ctx)
	})
}

// Wait 모든 고루틴 완료 대기
func (g *SafeGoGroup) Wait() {
	g.wg.Wait()
	g.once.Do(func() {
		close(g.errors)
	})
}

// Errors 발생한 에러들 반환
func (g *SafeGoGroup) Errors() []error {
	var errors []error

	// 채널에서 모든 에러 수집
	for err := range g.errors {
		errors = append(errors, err)
	}

	return errors
}

// FirstError 첫 번째 에러 반환 (있는 경우)
func (g *SafeGoGroup) FirstError() error {
	select {
	case err := <-g.errors:
		return err
	default:
		return nil
	}
}

// SafeGoWithRecover panic 복구 정보를 포함한 안전한 고루틴
func SafeGoWithRecover(ctx context.Context, fn SafeGoFunc, onRecover func(interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if onRecover != nil {
					onRecover(r)
				} else {
					stack := debug.Stack()
					GetGlobalPanicHandler()(r, stack)
				}
			}
		}()

		fn(ctx)
	}()
}

// ScheduledSafeGo 정기적으로 실행되는 안전한 고루틴
func ScheduledSafeGo(ctx context.Context, fn SafeGoFunc, interval time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				GetGlobalPanicHandler()(r, stack)
			}
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							stack := debug.Stack()
							GetGlobalPanicHandler()(r, stack)
						}
					}()
					fn(ctx)
				}()
			}
		}
	}()
}
