package webhook

import "errors"

// Common errors for webhook package
var (
	// Worker errors
	// ErrNoAvailableWorkers is a var that err no available workers
	// ErrAllWorkersAreBusy is a var that err all workers are busy
	// ErrMaxWorkersReached is a var that err max workers reached
	ErrNoAvailableWorkers = errors.New("no available workers")
	ErrAllWorkersAreBusy  = errors.New("all workers are busy")
	// ErrAlreadyStarted is a var that err already started
	// ErrAlreadyStopping is a var that err already stopping
	// ErrNotStarted is a var that err not started
	// ErrShuttingDown is a var that err shutting down
	// ErrShutdownTimeout is a var that err shutdown timeout
	ErrMaxWorkersReached = errors.New("maximum workers reached")

	// ErrInvalidEvent is a var that err invalid event
	// ErrRateLimited is a var that err rate limited
	// ErrQueueTimeout is a var that err queue timeout
	// Sender errors
	ErrAlreadyStarted = errors.New("webhook sender already started")
	// ErrBatchTooLarge is a var that err batch too large
	// ErrBatchTimeout is a var that err batch timeout
	ErrAlreadyStopping = errors.New("webhook sender already stopping")
	ErrNotStarted      = errors.New("webhook sender not started")
	ErrShuttingDown    = errors.New("webhook sender is shutting down")
	ErrShutdownTimeout = errors.New("shutdown timeout exceeded")

	// Event errors
	ErrInvalidEvent = errors.New("invalid event")
	ErrRateLimited  = errors.New("rate limit exceeded")
	ErrQueueTimeout = errors.New("queue send timeout")

	// Batch errors
	ErrBatchTooLarge = errors.New("batch size exceeds maximum")
	ErrBatchTimeout  = errors.New("batch send timeout")
)
