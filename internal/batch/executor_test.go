package batch

import (
	"context"
	"testing"
	"time"
)

func TestExecutor_EvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		lastExit  int
		want      bool
		wantErr   bool
	}{
		{
			name:      "last_exit equals 0",
			condition: "last_exit == 0",
			lastExit:  0,
			want:      true,
		},
		{
			name:      "last_exit not equals 0",
			condition: "last_exit == 0",
			lastExit:  1,
			want:      false,
		},
		{
			name:      "last_exit not equals",
			condition: "last_exit != 0",
			lastExit:  1,
			want:      true,
		},
		{
			name:      "numeric comparison greater",
			condition: "5 > 3",
			want:      true,
		},
		{
			name:      "numeric comparison less",
			condition: "3 < 5",
			want:      true,
		},
		{
			name:      "string equality",
			condition: "foo == foo",
			want:      true,
		},
		{
			name:      "invalid condition",
			condition: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			executor.LastExitCode = tt.lastExit

			got, err := executor.EvaluateCondition(tt.condition)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutor_ExecuteEcho(t *testing.T) {
	executor := NewExecutor()
	cmd := &EchoCommand{Message: "hello world"}

	err := cmd.Execute(context.Background(), executor)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := executor.Output.String()
	expected := "hello world\n"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}
}

func TestExecutor_ExecuteExit(t *testing.T) {
	executor := NewExecutor()
	cmd := &ExitCommand{Code: 42}

	err := cmd.Execute(context.Background(), executor)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("Expected ExitError, got %T", err)
	}

	if exitErr.Code != 42 {
		t.Errorf("expected exit code 42, got %d", exitErr.Code)
	}
}

func TestExecutor_ExecuteScript(t *testing.T) {
	script := &BatchScript{
		Commands: []Command{
			&EchoCommand{Message: "start"},
			&EchoCommand{Message: "end"},
		},
	}

	executor := NewExecutor()
	err := executor.Execute(context.Background(), script)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := executor.Output.String()
	if !containsString(output, "start") || !containsString(output, "end") {
		t.Errorf("output missing expected strings: %s", output)
	}
}

func TestExecutor_ExecuteWithCancel(t *testing.T) {
	script := &BatchScript{
		Commands: []Command{
			&EchoCommand{Message: "start"},
			&EchoCommand{Message: "end"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	executor := NewExecutor()
	err := executor.Execute(ctx, script)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestJobManager_Start(t *testing.T) {
	jm := NewJobManager()

	script := &BatchScript{
		Commands: []Command{
			&EchoCommand{Message: "test"},
		},
	}

	job, err := jm.Start(script)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if job.ID == "" {
		t.Error("job ID is empty")
	}

	if job.Status != JobPending && job.Status != JobRunning {
		t.Errorf("expected Pending or Running status, got %v", job.Status)
	}

	// Wait a bit for job to complete
	time.Sleep(100 * time.Millisecond)

	retrievedJob, err := jm.Get(job.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrievedJob.Status != JobSuccess {
		t.Errorf("expected Success status, got %v", retrievedJob.Status)
	}
}

func TestJobManager_Cancel(t *testing.T) {
	jm := NewJobManager()

	// Create a long-running script
	script := &BatchScript{
		Commands: []Command{
			&EchoCommand{Message: "start"},
		},
	}

	job, err := jm.Start(script)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait a bit for job to start
	time.Sleep(10 * time.Millisecond)

	err = jm.Cancel(job.ID)
	if err != nil {
		// Job might have completed already, that's okay
		if job.Status == JobSuccess {
			t.Skip("job completed before cancel")
		}
		t.Fatalf("Cancel() error = %v", err)
	}
}

func TestJobManager_List(t *testing.T) {
	jm := NewJobManager()

	script := &BatchScript{
		Commands: []Command{
			&EchoCommand{Message: "test"},
		},
	}

	_, err := jm.Start(script)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	jobs := jm.List()
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
