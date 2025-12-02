// Package history provides persistent storage for batch job execution history.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"proxynd/internal/batch"
)

// Storage manages batch job history persistence.
type Storage struct {
	dataDir string
}

// NewStorage creates a new history storage manager.
func NewStorage(dataDir string) (*Storage, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &Storage{
		dataDir: dataDir,
	}, nil
}

// Save saves a job execution to history.
func (s *Storage) Save(job *batch.JobExecution) error {
	history := &batch.JobHistory{
		ID:        job.ID,
		Script:    job.Script.Source,
		StartTime: job.StartTime,
		EndTime:   job.EndTime,
		Status:    job.Status.String(),
		Output:    job.Output,
		ExitCode:  job.ExitCode,
	}

	if job.Error != nil {
		history.Error = job.Error.Error()
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Write to file
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", job.ID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// Get retrieves a job history by ID.
func (s *Storage) Get(jobID string) (*batch.JobHistory, error) {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", jobID))

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var history batch.JobHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history: %w", err)
	}

	return &history, nil
}

// List returns all job histories, sorted by start time (newest first).
func (s *Storage) List() ([]*batch.JobHistory, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	histories := make([]*batch.JobHistory, 0)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.dataDir, entry.Name()))
		if err != nil {
			continue // Skip files that can't be read
		}

		var history batch.JobHistory
		if err := json.Unmarshal(data, &history); err != nil {
			continue // Skip invalid JSON files
		}

		histories = append(histories, &history)
	}

	// Sort by start time (newest first)
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].StartTime.After(histories[j].StartTime)
	})

	return histories, nil
}

// ListSince returns job histories since a given time.
func (s *Storage) ListSince(since time.Time) ([]*batch.JobHistory, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}

	filtered := make([]*batch.JobHistory, 0)
	for _, history := range all {
		if history.StartTime.After(since) || history.StartTime.Equal(since) {
			filtered = append(filtered, history)
		}
	}

	return filtered, nil
}

// Delete removes a job history by ID.
func (s *Storage) Delete(jobID string) error {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", jobID))

	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("job not found: %s", jobID)
		}
		return fmt.Errorf("failed to delete history file: %w", err)
	}

	return nil
}

// Clean removes job histories older than the specified duration.
func (s *Storage) Clean(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)

	all, err := s.List()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, history := range all {
		if history.EndTime.Before(cutoff) {
			if err := s.Delete(history.ID); err == nil {
				count++
			}
		}
	}

	return count, nil
}
