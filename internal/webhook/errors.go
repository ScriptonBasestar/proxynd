package webhook

import "errors"

// Common errors for webhook package
var (
	// Worker errors
	ErrNoAvailableWorkers = errors.New("no available workers")
	ErrAllWorkersAreBusy  = errors.New("all workers are busy")
	ErrMaxWorkersReached  = errors.New("maximum workers reached")
	
	// Sender errors
	ErrAlreadyStarted  = errors.New("webhook sender already started")
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