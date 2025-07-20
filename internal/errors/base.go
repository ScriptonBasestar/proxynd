package errors

import (
	"fmt"
	"time"
)

// ErrorLevel represents error severity level
type ErrorLevel int

const (
	// ErrorLevelInfo indicates informational messages
	ErrorLevelInfo ErrorLevel = iota
	// ErrorLevelWarning indicates warning messages
	ErrorLevelWarning
	// ErrorLevelError indicates error messages
	ErrorLevelError
	// ErrorLevelCritical indicates critical error messages
	ErrorLevelCritical
)

// DomainError represents a structured error with domain context
type DomainError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Domain    string      `json:"domain"`
	Level     ErrorLevel  `json:"level"`
	Timestamp time.Time   `json:"timestamp"`
	Details   interface{} `json:"details,omitempty"`
	Cause     error       `json:"-"`
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)",
			e.Domain, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Domain, e.Code, e.Message)
}

// ErrorBuilder provides a builder pattern for creating domain errors
type ErrorBuilder struct {
	err *DomainError
}

// NewError creates a new error builder
func NewError(code, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &DomainError{
			Code:      code,
			Message:   message,
			Timestamp: time.Now(),
			Level:     ErrorLevelError,
		},
	}
}

// WithDomain sets the error domain
func (b *ErrorBuilder) WithDomain(domain string) *ErrorBuilder {
	b.err.Domain = domain
	return b
}

// WithCause sets the underlying cause error
func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	b.err.Cause = cause
	return b
}

// WithDetails sets additional error details
func (b *ErrorBuilder) WithDetails(details interface{}) *ErrorBuilder {
	b.err.Details = details
	return b
}

// WithLevel sets the error severity level
func (b *ErrorBuilder) WithLevel(level ErrorLevel) *ErrorBuilder {
	b.err.Level = level
	return b
}

// Build creates the final domain error
func (b *ErrorBuilder) Build() *DomainError {
	return b.err
}

// String converts error level to string representation
func (e ErrorLevel) String() string {
	switch e {
	case ErrorLevelInfo:
		return "info"
	case ErrorLevelWarning:
		return "warning"
	case ErrorLevelError:
		return "error"
	case ErrorLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}
