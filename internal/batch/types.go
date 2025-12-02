// Package batch provides batch job execution system for automating complex management tasks.
package batch

import (
	"context"
	"time"
)

// JobStatus represents the status of a batch job execution.
type JobStatus int

const (
	// JobPending indicates job is waiting to be executed
	JobPending JobStatus = iota
	// JobRunning indicates job is currently executing
	JobRunning
	// JobSuccess indicates job completed successfully
	JobSuccess
	// JobFailed indicates job failed with error
	JobFailed
	// JobCancelled indicates job was cancelled
	JobCancelled
)

// String returns the string representation of JobStatus
func (s JobStatus) String() string {
	switch s {
	case JobPending:
		return "pending"
	case JobRunning:
		return "running"
	case JobSuccess:
		return "success"
	case JobFailed:
		return "failed"
	case JobCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Command represents a single executable command in a batch script.
type Command interface {
	// Execute runs the command with given context and executor
	Execute(ctx context.Context, exec *Executor) error
	// String returns string representation of the command
	String() string
}

// EchoCommand prints a message to output.
type EchoCommand struct {
	Message string
}

// Execute implements Command interface for EchoCommand
func (c *EchoCommand) Execute(ctx context.Context, exec *Executor) error {
	exec.Output.WriteString(c.Message + "\n")
	return nil
}

// String returns string representation
func (c *EchoCommand) String() string {
	return "echo " + c.Message
}

// CLICommand executes a proxyndctl CLI command.
type CLICommand struct {
	Name string
	Args []string
}

// Execute implements Command interface for CLICommand
func (c *CLICommand) Execute(ctx context.Context, exec *Executor) error {
	return exec.ExecuteCLI(ctx, c.Name, c.Args...)
}

// String returns string representation
func (c *CLICommand) String() string {
	return c.Name + " " + joinArgs(c.Args)
}

// ExitCommand exits the script with specified code.
type ExitCommand struct {
	Code int
}

// Execute implements Command interface for ExitCommand
func (c *ExitCommand) Execute(ctx context.Context, exec *Executor) error {
	return &ExitError{Code: c.Code}
}

// String returns string representation
func (c *ExitCommand) String() string {
	return "exit"
}

// ConditionalCommand executes commands based on condition.
type ConditionalCommand struct {
	Condition string
	ThenBlock []Command
	ElseBlock []Command
}

// Execute implements Command interface for ConditionalCommand
func (c *ConditionalCommand) Execute(ctx context.Context, exec *Executor) error {
	result, err := exec.EvaluateCondition(c.Condition)
	if err != nil {
		return err
	}

	if result {
		for _, cmd := range c.ThenBlock {
			if err := cmd.Execute(ctx, exec); err != nil {
				return err
			}
		}
	} else if len(c.ElseBlock) > 0 {
		for _, cmd := range c.ElseBlock {
			if err := cmd.Execute(ctx, exec); err != nil {
				return err
			}
		}
	}

	return nil
}

// String returns string representation
func (c *ConditionalCommand) String() string {
	return "if " + c.Condition + " then ... fi"
}

// BatchScript represents a parsed batch script.
type BatchScript struct {
	Commands []Command
	Source   string
}

// JobExecution represents a running or completed job.
type JobExecution struct {
	ID          string
	Script      *BatchScript
	Context     context.Context
	Cancel      context.CancelFunc
	StartTime   time.Time
	EndTime     time.Time
	Status      JobStatus
	CurrentStep int
	Output      []string
	ExitCode    int
	Error       error
}

// JobHistory represents historical job execution record.
type JobHistory struct {
	ID        string    `json:"id"`
	Script    string    `json:"script"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
	Output    []string  `json:"output"`
	ExitCode  int       `json:"exit_code"`
	Error     string    `json:"error,omitempty"`
}

// ExitError represents an explicit exit command
type ExitError struct {
	Code int
}

// Error implements error interface
func (e *ExitError) Error() string {
	return "exit"
}

// Helper function to join args
func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}
