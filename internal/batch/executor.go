package batch

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Executor executes batch scripts with context management and progress tracking.
type Executor struct {
	// Output buffer for command output
	Output *bytes.Buffer

	// LastExitCode stores the exit code of the last command
	LastExitCode int

	// Variables stores script variables
	Variables map[string]string

	// AllowedCommands whitelist of allowed CLI commands
	AllowedCommands map[string]bool
}

// NewExecutor creates a new batch executor
func NewExecutor() *Executor {
	return &Executor{
		Output:       &bytes.Buffer{},
		Variables:    make(map[string]string),
		LastExitCode: 0,
		AllowedCommands: map[string]bool{
			"cache":        true,
			"maven-backup": true,
			"maven-index":  true,
			"test":         true,
			"status":       true,
			"health":       true,
			"version":      true,
		},
	}
}

// Execute runs a batch script
func (e *Executor) Execute(ctx context.Context, script *BatchScript) error {
	for i, cmd := range script.Commands {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Fprintf(e.Output, "[Step %d/%d] %s\n", i+1, len(script.Commands), cmd.String())

			if err := cmd.Execute(ctx, e); err != nil {
				// Check if it's an explicit exit
				if exitErr, ok := err.(*ExitError); ok {
					e.LastExitCode = exitErr.Code
					if exitErr.Code != 0 {
						return fmt.Errorf("script exited with code %d", exitErr.Code)
					}
					return nil
				}
				return fmt.Errorf("command failed at step %d: %w", i+1, err)
			}
		}
	}
	return nil
}

// ExecuteCLI executes a CLI command
func (e *Executor) ExecuteCLI(ctx context.Context, name string, args ...string) error {
	// Validate command is allowed
	if !e.AllowedCommands[name] {
		return fmt.Errorf("command not allowed: %s", name)
	}

	// Build command
	cmd := exec.CommandContext(ctx, "proxyndctl", append([]string{name}, args...)...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()

	// Write output
	if stdout.Len() > 0 {
		e.Output.Write(stdout.Bytes())
	}
	if stderr.Len() > 0 {
		e.Output.Write(stderr.Bytes())
	}

	// Store exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			e.LastExitCode = exitErr.ExitCode()
		} else {
			e.LastExitCode = 1
		}
		return err
	}

	e.LastExitCode = 0
	return nil
}

// EvaluateCondition evaluates a conditional expression
func (e *Executor) EvaluateCondition(condition string) (bool, error) {
	// Trim whitespace
	condition = strings.TrimSpace(condition)

	// Parse condition
	parts := strings.Fields(condition)
	if len(parts) < 3 {
		return false, fmt.Errorf("invalid condition: %s", condition)
	}

	left := parts[0]
	operator := parts[1]
	right := parts[2]

	// Resolve variables
	leftVal := e.resolveValue(left)
	rightVal := e.resolveValue(right)

	// Evaluate based on operator
	switch operator {
	case "==":
		return leftVal == rightVal, nil
	case "!=":
		return leftVal != rightVal, nil
	case ">":
		return compareNumeric(leftVal, rightVal, ">")
	case "<":
		return compareNumeric(leftVal, rightVal, "<")
	case ">=":
		return compareNumeric(leftVal, rightVal, ">=")
	case "<=":
		return compareNumeric(leftVal, rightVal, "<=")
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// resolveValue resolves a value (variable or literal)
func (e *Executor) resolveValue(value string) string {
	// Check for special variables
	switch value {
	case "last_exit":
		return strconv.Itoa(e.LastExitCode)
	case "last_exit_code":
		return strconv.Itoa(e.LastExitCode)
	default:
		// Check user variables
		if val, ok := e.Variables[value]; ok {
			return val
		}
		// Return as literal
		return value
	}
}

// compareNumeric compares two values numerically
func compareNumeric(left, right, operator string) (bool, error) {
	leftNum, err := strconv.ParseFloat(left, 64)
	if err != nil {
		return false, fmt.Errorf("cannot compare non-numeric values: %s", left)
	}

	rightNum, err := strconv.ParseFloat(right, 64)
	if err != nil {
		return false, fmt.Errorf("cannot compare non-numeric values: %s", right)
	}

	switch operator {
	case ">":
		return leftNum > rightNum, nil
	case "<":
		return leftNum < rightNum, nil
	case ">=":
		return leftNum >= rightNum, nil
	case "<=":
		return leftNum <= rightNum, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// JobManager manages batch job executions
type JobManager struct {
	jobs map[string]*JobExecution
}

// NewJobManager creates a new job manager
func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*JobExecution),
	}
}

// Start starts a new batch job
func (jm *JobManager) Start(script *BatchScript) (*JobExecution, error) {
	ctx, cancel := context.WithCancel(context.Background())

	job := &JobExecution{
		ID:        generateJobID(),
		Script:    script,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		Status:    JobPending,
		Output:    make([]string, 0),
	}

	jm.jobs[job.ID] = job

	// Start execution in goroutine
	go func() {
		job.Status = JobRunning
		executor := NewExecutor()

		err := executor.Execute(job.Context, script)

		job.EndTime = time.Now()
		job.ExitCode = executor.LastExitCode
		job.Output = strings.Split(executor.Output.String(), "\n")

		if err != nil {
			job.Status = JobFailed
			job.Error = err
		} else {
			job.Status = JobSuccess
		}
	}()

	return job, nil
}

// Cancel cancels a running job
func (jm *JobManager) Cancel(jobID string) error {
	job, ok := jm.jobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status != JobRunning {
		return fmt.Errorf("job is not running: %s", job.Status)
	}

	job.Cancel()
	job.Status = JobCancelled
	return nil
}

// Get retrieves a job by ID
func (jm *JobManager) Get(jobID string) (*JobExecution, error) {
	job, ok := jm.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

// List returns all jobs
func (jm *JobManager) List() []*JobExecution {
	jobs := make([]*JobExecution, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}
