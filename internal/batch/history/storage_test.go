package history

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxynd/internal/batch"
)

func TestStorage_SaveAndGet(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	// Create test job
	script := &batch.BatchScript{
		Commands: []batch.Command{
			&batch.EchoCommand{Message: "test"},
		},
		Source: "echo test",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := &batch.JobExecution{
		ID:        "test-job-1",
		Script:    script,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Second),
		Status:    batch.JobSuccess,
		Output:    []string{"test output"},
		ExitCode:  0,
	}

	// Save job
	if err := storage.Save(job); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	filename := filepath.Join(tmpDir, "test-job-1.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("history file was not created")
	}

	// Get job
	history, err := storage.Get("test-job-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if history.ID != "test-job-1" {
		t.Errorf("expected ID %q, got %q", "test-job-1", history.ID)
	}

	if history.Status != "success" {
		t.Errorf("expected status %q, got %q", "success", history.Status)
	}
}

func TestStorage_List(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	// Create and save multiple jobs
	for i := 0; i < 3; i++ {
		script := &batch.BatchScript{
			Commands: []batch.Command{},
			Source:   "test",
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		job := &batch.JobExecution{
			ID:        fmt.Sprintf("job-%d", i),
			Script:    script,
			Context:   ctx,
			Cancel:    cancel,
			StartTime: time.Now().Add(time.Duration(i) * time.Minute),
			EndTime:   time.Now().Add(time.Duration(i+1) * time.Minute),
			Status:    batch.JobSuccess,
			Output:    []string{},
			ExitCode:  0,
		}

		if err := storage.Save(job); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// List all jobs
	histories, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(histories) != 3 {
		t.Errorf("expected 3 histories, got %d", len(histories))
	}

	// Verify sorted by start time (newest first)
	if len(histories) >= 2 {
		if histories[0].StartTime.Before(histories[1].StartTime) {
			t.Error("histories not sorted correctly (should be newest first)")
		}
	}
}

func TestStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	// Create and save a job
	script := &batch.BatchScript{Commands: []batch.Command{}, Source: "test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := &batch.JobExecution{
		ID:        "job-to-delete",
		Script:    script,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	if err := storage.Save(job); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Delete the job
	if err := storage.Delete("job-to-delete"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = storage.Get("job-to-delete")
	if err == nil {
		t.Error("expected error when getting deleted job, got nil")
	}
}

func TestStorage_Clean(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	// Create old and new jobs
	script := &batch.BatchScript{Commands: []batch.Command{}, Source: "test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Old job (2 hours ago)
	oldJob := &batch.JobExecution{
		ID:        "old-job",
		Script:    script,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now().Add(-2 * time.Hour),
		EndTime:   time.Now().Add(-2 * time.Hour),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	// New job (now)
	newJob := &batch.JobExecution{
		ID:        "new-job",
		Script:    script,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Status:    batch.JobSuccess,
		Output:    []string{},
		ExitCode:  0,
	}

	if err := storage.Save(oldJob); err != nil {
		t.Fatalf("Save(oldJob) error = %v", err)
	}

	if err := storage.Save(newJob); err != nil {
		t.Fatalf("Save(newJob) error = %v", err)
	}

	// Clean jobs older than 1 hour
	count, err := storage.Clean(1 * time.Hour)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if count != 1 {
		t.Errorf("expected to clean 1 job, cleaned %d", count)
	}

	// Verify old job is gone
	_, err = storage.Get("old-job")
	if err == nil {
		t.Error("expected error for old job, got nil")
	}

	// Verify new job still exists
	_, err = storage.Get("new-job")
	if err != nil {
		t.Errorf("new job should still exist, got error: %v", err)
	}
}
