package goroutine

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"
)

var (
	ErrPoolClosed = errors.New("worker pool is closed")
	ErrTimeout    = errors.New("task execution timeout")
)

// Task 작업 인터페이스 정의
type Task interface {
	Execute(ctx context.Context) error
}

// TaskFunc 함수형 작업 정의
type TaskFunc func(ctx context.Context) error

// Execute TaskFunc를 Task 인터페이스로 변환
func (f TaskFunc) Execute(ctx context.Context) error {
	return f(ctx)
}

// WorkerPool 고루틴 안전한 워커풀
type WorkerPool struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	taskChan  chan Task
	workers   int
	started   bool
	closed    bool
	mu        sync.RWMutex
	errorChan chan error
}

// WorkerPoolConfig 워커풀 설정
type WorkerPoolConfig struct {
	Workers    int           // 워커 수
	BufferSize int           // 작업 버퍼 크기
	Timeout    time.Duration // 작업 타임아웃
}

// DefaultWorkerPoolConfig 기본 워커풀 설정
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		Workers:    runtime.NumCPU(),
		BufferSize: 100,
		Timeout:    30 * time.Second,
	}
}

// NewWorkerPool 새로운 워커풀 생성
func NewWorkerPool(ctx context.Context, config *WorkerPoolConfig) *WorkerPool {
	if config == nil {
		config = DefaultWorkerPoolConfig()
	}

	ctx, cancel := context.WithCancel(ctx)
	
	return &WorkerPool{
		ctx:       ctx,
		cancel:    cancel,
		workers:   config.Workers,
		taskChan:  make(chan Task, config.BufferSize),
		errorChan: make(chan error, config.BufferSize),
	}
}

// Start 워커풀 시작
func (p *WorkerPool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}

	p.started = true

	// 워커 고루틴 시작
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker 개별 워커 고루틴
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskChan:
			if !ok {
				return
			}
			
			// 작업 실행
			if err := task.Execute(p.ctx); err != nil {
				select {
				case p.errorChan <- err:
				default:
					// 에러 채널이 가득 찬 경우 로그로 처리 (향후 로깅 시스템 구현)
				}
			}
		}
	}
}

// Submit 작업 제출
func (p *WorkerPool) Submit(task Task) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPoolClosed
	}

	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case p.taskChan <- task:
		return nil
	default:
		return errors.New("task queue is full")
	}
}

// SubmitFunc 함수형 작업 제출
func (p *WorkerPool) SubmitFunc(fn func(ctx context.Context) error) error {
	return p.Submit(TaskFunc(fn))
}

// SubmitWithTimeout 타임아웃이 있는 작업 제출
func (p *WorkerPool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	timeoutTask := TaskFunc(func(ctx context.Context) error {
		done := make(chan error, 1)
		go func() {
			done <- task.Execute(timeoutCtx)
		}()

		select {
		case <-timeoutCtx.Done():
			return timeoutCtx.Err()
		case err := <-done:
			return err
		}
	})

	return p.Submit(timeoutTask)
}

// Shutdown 워커풀 종료
func (p *WorkerPool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.taskChan)
	p.cancel()
	p.wg.Wait()
	close(p.errorChan)
}

// ShutdownWithTimeout 타임아웃이 있는 워커풀 종료
func (p *WorkerPool) ShutdownWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		p.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrTimeout
	}
}

// ErrorChan 에러 채널 반환
func (p *WorkerPool) ErrorChan() <-chan error {
	return p.errorChan
}

// IsStarted 워커풀 시작 여부 확인
func (p *WorkerPool) IsStarted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started
}

// IsClosed 워커풀 종료 여부 확인
func (p *WorkerPool) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// Size 워커 수 반환
func (p *WorkerPool) Size() int {
	return p.workers
}

// BufferSize 작업 버퍼 크기 반환
func (p *WorkerPool) BufferSize() int {
	return cap(p.taskChan)
}

// PendingTasks 대기 중인 작업 수 반환
func (p *WorkerPool) PendingTasks() int {
	return len(p.taskChan)
}