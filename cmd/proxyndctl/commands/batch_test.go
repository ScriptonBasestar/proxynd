// Package commands provides CLI command implementations for proxyndctl
package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxynd/internal/batch"
	"proxynd/internal/batch/history"
)

func TestBatchValidate_ValidScript(t *testing.T) {
	// Create temporary script file
	tmpDir := t.TempDir()
	scriptFile := filepath.Join(tmpDir, "test.batch")

	script := `# Test script
echo "Starting test"
cache clear --force
if last_exit == 0 then
    echo "Success"
fi
`
	if err := os.WriteFile(scriptFile, []byte(script), 0644); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Test validation
	err := runBatchValidate(scriptFile)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}

func TestBatchValidate_InvalidScript(t *testing.T) {
	tmpDir := t.TempDir()
	scriptFile := filepath.Join(tmpDir, "invalid.batch")

	// Missing 'fi' for if statement
	script := `if last_exit == 0 then
    echo "Success"
`
	if err := os.WriteFile(scriptFile, []byte(script), 0644); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Test validation should fail
	err := runBatchValidate(scriptFile)
	if err == nil {
		t.Error("Expected validation to fail for invalid script")
	}
}

func TestBatchValidate_FileNotFound(t *testing.T) {
	err := runBatchValidate("nonexistent.batch")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestBatchHistory_EmptyHistory(t *testing.T) {
	// Create temporary history directory
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	// Test with empty history
	err := runBatchHistory(10, "", "", "table")
	if err != nil {
		t.Errorf("Expected success with empty history, got error: %v", err)
	}
}

func TestBatchHistory_WithData(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	// Create storage and add test data
	storage, err := history.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create test job history
	script := &batch.BatchScript{
		Commands: []batch.Command{},
		Source:   "echo test",
	}

	job := &batch.JobExecution{
		ID:        "test-job-1",
		Script:    script,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(-30 * time.Minute),
		Status:    batch.JobSuccess,
		Output:    []string{"test output"},
		ExitCode:  0,
	}

	if err := storage.Save(job); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Test history listing
	tests := []struct {
		name   string
		format string
	}{
		{"table format", "table"},
		{"json format", "json"},
		{"text format", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runBatchHistory(10, "", "", tt.format)
			if err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}
		})
	}
}

func TestBatchHistory_WithTimeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	storage, err := history.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create old and recent jobs
	script := &batch.BatchScript{
		Commands: []batch.Command{},
		Source:   "echo test",
	}

	// Old job (2 days ago)
	oldJob := &batch.JobExecution{
		ID:        "old-job",
		Script:    script,
		StartTime: time.Now().Add(-48 * time.Hour),
		EndTime:   time.Now().Add(-47 * time.Hour),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	// Recent job (1 hour ago)
	recentJob := &batch.JobExecution{
		ID:        "recent-job",
		Script:    script,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(-30 * time.Minute),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	if err := storage.Save(oldJob); err != nil {
		t.Fatalf("Failed to save old job: %v", err)
	}
	if err := storage.Save(recentJob); err != nil {
		t.Fatalf("Failed to save recent job: %v", err)
	}

	// Test with "since 24h" filter - should only show recent job
	err = runBatchHistory(10, "24h", "", "table")
	if err != nil {
		t.Errorf("Expected success with time filter, got error: %v", err)
	}
}

func TestBatchHistory_WithStatusFilter(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	storage, err := history.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	script := &batch.BatchScript{
		Commands: []batch.Command{},
		Source:   "echo test",
	}

	// Create success and failed jobs
	successJob := &batch.JobExecution{
		ID:        "success-job",
		Script:    script,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(-30 * time.Minute),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	failedJob := &batch.JobExecution{
		ID:        "failed-job",
		Script:    script,
		StartTime: time.Now().Add(-2 * time.Hour),
		EndTime:   time.Now().Add(-90 * time.Minute),
		Status:    batch.JobFailed,
		Output:    []string{},
		ExitCode:  1,
	}

	if err := storage.Save(successJob); err != nil {
		t.Fatalf("Failed to save success job: %v", err)
	}
	if err := storage.Save(failedJob); err != nil {
		t.Fatalf("Failed to save failed job: %v", err)
	}

	// Test status filter
	err = runBatchHistory(10, "", "failed", "table")
	if err != nil {
		t.Errorf("Expected success with status filter, got error: %v", err)
	}
}

func TestBatchShow_Success(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	storage, err := history.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	script := &batch.BatchScript{
		Commands: []batch.Command{},
		Source:   "echo test\necho done",
	}

	job := &batch.JobExecution{
		ID:        "show-test-job",
		Script:    script,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(-30 * time.Minute),
		Status:    batch.JobSuccess,
		Output:    []string{"test", "done"},
		ExitCode:  0,
	}

	if err := storage.Save(job); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Test show command
	err = runBatchShow("show-test-job", false)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Test show with full output
	err = runBatchShow("show-test-job", true)
	if err != nil {
		t.Errorf("Expected success with full output, got error: %v", err)
	}
}

func TestBatchShow_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	err := runBatchShow("nonexistent-job", false)
	if err == nil {
		t.Error("Expected error for nonexistent job")
	}
}

func TestBatchClean_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	storage, err := history.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	script := &batch.BatchScript{
		Commands: []batch.Command{},
		Source:   "echo test",
	}

	// Create old job
	oldJob := &batch.JobExecution{
		ID:        "old-job",
		Script:    script,
		StartTime: time.Now().Add(-48 * time.Hour),
		EndTime:   time.Now().Add(-47 * time.Hour),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	if err := storage.Save(oldJob); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Test dry run - should not actually delete
	err = runBatchClean("24h", "", true)
	if err != nil {
		t.Errorf("Expected success with dry run, got error: %v", err)
	}

	// Verify job still exists
	_, err = storage.Get("old-job")
	if err != nil {
		t.Error("Job should still exist after dry run")
	}
}

func TestBatchClean_ActualClean(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("PROXYND_BATCH_HISTORY_DIR", tmpDir)
	defer os.Unsetenv("PROXYND_BATCH_HISTORY_DIR")

	storage, err := history.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	script := &batch.BatchScript{
		Commands: []batch.Command{},
		Source:   "echo test",
	}

	// Create old and new jobs
	oldJob := &batch.JobExecution{
		ID:        "old-job",
		Script:    script,
		StartTime: time.Now().Add(-48 * time.Hour),
		EndTime:   time.Now().Add(-47 * time.Hour),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	newJob := &batch.JobExecution{
		ID:        "new-job",
		Script:    script,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(-30 * time.Minute),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	if err := storage.Save(oldJob); err != nil {
		t.Fatalf("Failed to save old job: %v", err)
	}
	if err := storage.Save(newJob); err != nil {
		t.Fatalf("Failed to save new job: %v", err)
	}

	// Test actual clean
	err = runBatchClean("24h", "", false)
	if err != nil {
		t.Errorf("Expected success with clean, got error: %v", err)
	}

	// Verify old job is deleted
	_, err = storage.Get("old-job")
	if err == nil {
		t.Error("Old job should have been deleted")
	}

	// Verify new job still exists
	_, err = storage.Get("new-job")
	if err != nil {
		t.Error("New job should still exist")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			want:     "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 15*time.Second,
			want:     "2m 15s",
		},
		{
			name:     "hours and minutes",
			duration: 1*time.Hour + 30*time.Minute,
			want:     "1h 30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
