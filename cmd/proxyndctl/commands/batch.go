// Package commands provides CLI command implementations for proxyndctl
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"proxynd/internal/batch"
	"proxynd/internal/batch/history"
)

// BatchJobResponse represents a job execution response
type BatchJobResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	ExitCode  int       `json:"exit_code,omitempty"`
	Output    []string  `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// NewBatchCmd creates the batch command group
func NewBatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Batch job execution and management",
		Long: `Execute and manage batch scripts for automated maintenance tasks.
Batch scripts support conditionals, command execution, and job history tracking.`,
		Example: `  # Run a batch script
  proxyndctl batch run maintenance.batch

  # Validate a batch script
  proxyndctl batch validate script.batch

  # Show job history
  proxyndctl batch history --limit 10

  # Cancel a running job
  proxyndctl batch cancel job-1234567890`,
	}

	// Add subcommands
	cmd.AddCommand(newBatchRunCmd())
	cmd.AddCommand(newBatchValidateCmd())
	cmd.AddCommand(newBatchHistoryCmd())
	cmd.AddCommand(newBatchCancelCmd())
	cmd.AddCommand(newBatchShowCmd())
	cmd.AddCommand(newBatchCleanCmd())

	return cmd
}

// newBatchRunCmd creates the batch run command
func newBatchRunCmd() *cobra.Command {
	var (
		async   bool
		output  string
		timeout string
	)

	cmd := &cobra.Command{
		Use:   "run <file>",
		Short: "Execute a batch script file",
		Long: `Execute a batch script file containing commands and conditionals.
The script can include cache management, backups, and other maintenance tasks.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Run maintenance script
  proxyndctl batch run maintenance.batch

  # Run in background
  proxyndctl batch run --async cleanup.batch

  # Run with timeout
  proxyndctl batch run --timeout 30m long-running.batch

  # Save output to file
  proxyndctl batch run --output result.log script.batch`,
		RunE: func(_ *cobra.Command, args []string) error {
			return runBatchRun(args[0], async, output, timeout)
		},
	}

	cmd.Flags().BoolVar(&async, "async", false, "Run in background and return immediately")
	cmd.Flags().StringVar(&output, "output", "", "Save output to file")
	cmd.Flags().StringVar(&timeout, "timeout", "", "Set execution timeout (e.g., 30m, 1h)")

	return cmd
}

// newBatchValidateCmd creates the batch validate command
func newBatchValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate a batch script for syntax errors",
		Long: `Validate a batch script file for syntax errors without executing it.
Reports line numbers and specific errors found in the script.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Validate script
  proxyndctl batch validate maintenance.batch

  # Validate multiple scripts
  for file in *.batch; do
    proxyndctl batch validate "$file"
  done`,
		RunE: func(_ *cobra.Command, args []string) error {
			return runBatchValidate(args[0])
		},
	}

	return cmd
}

// newBatchHistoryCmd creates the batch history command
func newBatchHistoryCmd() *cobra.Command {
	var (
		limit  int
		since  string
		status string
		format string
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show batch job execution history",
		Long: `Show history of batch job executions with status, duration, and exit codes.
Supports filtering by time range and status.`,
		Example: `  # Show last 10 jobs
  proxyndctl batch history

  # Show last 20 jobs
  proxyndctl batch history --limit 20

  # Show jobs from last 24 hours
  proxyndctl batch history --since 24h

  # Show only failed jobs
  proxyndctl batch history --status failed

  # Get JSON output
  proxyndctl batch history --format json`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBatchHistory(limit, since, status, format)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Show last N jobs")
	cmd.Flags().StringVar(&since, "since", "", "Show jobs since duration ago (e.g., 24h, 7d)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (success, failed, cancelled)")
	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json, text)")

	return cmd
}

// newBatchCancelCmd creates the batch cancel command
func newBatchCancelCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "cancel <job-id>",
		Short: "Cancel a running batch job",
		Long: `Cancel a running batch job by its ID.
Only jobs with status 'running' can be cancelled.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Cancel a running job
  proxyndctl batch cancel job-1638360000123

  # Cancel with confirmation
  proxyndctl batch cancel --yes job-1638360000123`,
		RunE: func(_ *cobra.Command, args []string) error {
			return runBatchCancel(args[0], yes)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	return cmd
}

// newBatchShowCmd creates the batch show command
func newBatchShowCmd() *cobra.Command {
	var fullOutput bool

	cmd := &cobra.Command{
		Use:   "show <job-id>",
		Short: "Show detailed information about a job execution",
		Long: `Display detailed information about a specific batch job execution.
Shows script content, output, timing, and exit code.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Show job details
  proxyndctl batch show job-1638360000123

  # Show with full output
  proxyndctl batch show --full-output job-1638360000123`,
		RunE: func(_ *cobra.Command, args []string) error {
			return runBatchShow(args[0], fullOutput)
		},
	}

	cmd.Flags().BoolVar(&fullOutput, "full-output", false, "Show complete output without truncation")

	return cmd
}

// newBatchCleanCmd creates the batch clean command
func newBatchCleanCmd() *cobra.Command {
	var (
		olderThan string
		status    string
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up old job history",
		Long: `Remove old job history entries to free up space.
Supports filtering by age and status.`,
		Example: `  # Clean jobs older than 30 days
  proxyndctl batch clean

  # Clean jobs older than 7 days
  proxyndctl batch clean --older-than 7d

  # Clean only failed jobs
  proxyndctl batch clean --status failed

  # Preview what would be cleaned
  proxyndctl batch clean --dry-run`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBatchClean(olderThan, status, dryRun)
		},
	}

	cmd.Flags().StringVar(&olderThan, "older-than", "30d", "Remove jobs older than duration")
	cmd.Flags().StringVar(&status, "status", "", "Only remove jobs with specific status")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without actually removing")

	return cmd
}

// runBatchRun executes a batch script
func runBatchRun(filename string, async bool, outputFile string, timeoutStr string) error {
	// Read script file
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// Parse script
	parser := batch.NewParser(string(source))
	script, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse script: %w", err)
	}

	// Create job manager
	jm := batch.NewJobManager()

	// TODO: Implement timeout support in JobManager
	if timeoutStr != "" {
		_, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
		// Timeout will be implemented in future version
		fmt.Println("Warning: Timeout flag is not yet implemented")
	}

	// Start job execution
	job, err := jm.Start(script)
	if err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	fmt.Printf("Job started: %s\n", job.ID)

	if async {
		fmt.Println("Running in background...")
		return nil
	}

	// Wait for completion
	for {
		currentJob, err := jm.Get(job.ID)
		if err != nil {
			return fmt.Errorf("failed to get job status: %w", err)
		}

		if currentJob.Status == batch.JobSuccess || currentJob.Status == batch.JobFailed || currentJob.Status == batch.JobCancelled {
			// Print output
			for i, line := range currentJob.Output {
				fmt.Printf("[Step %d/%d] %s\n", i+1, len(currentJob.Output), line)
			}

			// Save output if requested
			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(strings.Join(currentJob.Output, "\n")), 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("\nOutput saved to: %s\n", outputFile)
			}

			// Print result
			fmt.Printf("\nJob completed: %s\n", currentJob.Status.String())
			fmt.Printf("Exit code: %d\n", currentJob.ExitCode)

			if currentJob.Status == batch.JobSuccess {
				return nil
			}
			return fmt.Errorf("job failed with exit code %d", currentJob.ExitCode)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// runBatchValidate validates a batch script
func runBatchValidate(filename string) error {
	// Read script file
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// Validate script
	if err := batch.Validate(string(source)); err != nil {
		fmt.Printf("❌ Syntax error in %s:\n", filename)
		fmt.Printf("   %s\n", err.Error())
		return err
	}

	// Parse to get command count
	parser := batch.NewParser(string(source))
	script, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse script: %w", err)
	}

	fmt.Printf("✅ Script is valid: %s\n", filename)
	fmt.Printf("   - %d commands parsed\n", len(script.Commands))
	fmt.Printf("   - No syntax errors\n")

	return nil
}

// runBatchHistory shows job execution history
func runBatchHistory(limit int, sinceStr string, statusFilter string, format string) error {
	// Get data directory (from config or default)
	dataDir := getHistoryDataDir()

	// Create storage
	storage, err := history.NewStorage(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Get histories
	var histories []*batch.JobHistory
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
		since := time.Now().Add(-duration)
		histories, err = storage.ListSince(since)
		if err != nil {
			return fmt.Errorf("failed to list histories: %w", err)
		}
	} else {
		histories, err = storage.List()
		if err != nil {
			return fmt.Errorf("failed to list histories: %w", err)
		}
	}

	// Filter by status if provided
	if statusFilter != "" {
		filtered := make([]*batch.JobHistory, 0)
		for _, h := range histories {
			if strings.EqualFold(h.Status, statusFilter) {
				filtered = append(filtered, h)
			}
		}
		histories = filtered
	}

	// Apply limit
	if limit > 0 && len(histories) > limit {
		histories = histories[:limit]
	}

	// Output based on format
	switch format {
	case "json":
		return outputHistoryJSON(histories)
	case "text":
		return outputHistoryText(histories)
	default:
		return outputHistoryTable(histories)
	}
}

// runBatchCancel cancels a running job
func runBatchCancel(jobID string, skipConfirm bool) error {
	if !skipConfirm {
		fmt.Printf("Are you sure you want to cancel job %s? (y/N): ", jobID)
		var response string
		fmt.Scanln(&response)
		if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
			fmt.Println("Cancelled")
			return nil
		}
	}

	jm := batch.NewJobManager()
	if err := jm.Cancel(jobID); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	fmt.Printf("✅ Job cancelled: %s\n", jobID)
	return nil
}

// runBatchShow shows detailed job information
func runBatchShow(jobID string, fullOutput bool) error {
	dataDir := getHistoryDataDir()
	storage, err := history.NewStorage(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	h, err := storage.Get(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Print job details
	fmt.Printf("Job ID:      %s\n", h.ID)
	fmt.Printf("Status:      %s\n", h.Status)
	fmt.Printf("Start Time:  %s\n", h.StartTime.Format("2006-01-02 15:04:05"))
	if !h.EndTime.IsZero() {
		fmt.Printf("End Time:    %s\n", h.EndTime.Format("2006-01-02 15:04:05"))
		duration := h.EndTime.Sub(h.StartTime)
		fmt.Printf("Duration:    %s\n", formatDuration(duration))
	}
	fmt.Printf("Exit Code:   %d\n", h.ExitCode)

	if h.Error != "" {
		fmt.Printf("Error:       %s\n", h.Error)
	}

	// Print script
	fmt.Println("\nScript:")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println(h.Script)
	fmt.Println(strings.Repeat("─", 40))

	// Print output
	if len(h.Output) > 0 {
		fmt.Println("\nOutput:")
		fmt.Println(strings.Repeat("─", 40))
		if fullOutput || len(h.Output) <= 50 {
			for _, line := range h.Output {
				fmt.Println(line)
			}
		} else {
			for _, line := range h.Output[:25] {
				fmt.Println(line)
			}
			fmt.Printf("\n... (%d lines omitted) ...\n\n", len(h.Output)-50)
			for _, line := range h.Output[len(h.Output)-25:] {
				fmt.Println(line)
			}
		}
		fmt.Println(strings.Repeat("─", 40))
	}

	return nil
}

// runBatchClean cleans up old job history
func runBatchClean(olderThanStr string, statusFilter string, dryRun bool) error {
	duration, err := time.ParseDuration(olderThanStr)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	dataDir := getHistoryDataDir()
	storage, err := history.NewStorage(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	if dryRun {
		// Preview mode
		cutoff := time.Now().Add(-duration)
		histories, err := storage.List()
		if err != nil {
			return fmt.Errorf("failed to list histories: %w", err)
		}

		count := 0
		for _, h := range histories {
			if h.EndTime.Before(cutoff) {
				if statusFilter == "" || strings.EqualFold(h.Status, statusFilter) {
					count++
					fmt.Printf("Would remove: %s (%s, %s)\n", h.ID, h.Status, h.EndTime.Format("2006-01-02 15:04:05"))
				}
			}
		}

		fmt.Printf("\nDry run: Would remove %d jobs\n", count)
		return nil
	}

	// Actual cleanup
	count, err := storage.Clean(duration)
	if err != nil {
		return fmt.Errorf("failed to clean history: %w", err)
	}

	fmt.Printf("Cleaning job history...\n")
	fmt.Printf("- Removed %d jobs older than %s\n", count, olderThanStr)

	// Get remaining count
	remaining, err := storage.List()
	if err == nil {
		fmt.Printf("- Total jobs remaining: %d\n", len(remaining))
	}

	return nil
}

// Helper functions

func getHistoryDataDir() string {
	// Try to get from environment or config
	if dir := os.Getenv("PROXYND_BATCH_HISTORY_DIR"); dir != "" {
		return dir
	}

	// Default to ~/.proxynd/batch/history
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/proxynd/batch/history"
	}

	return filepath.Join(home, ".proxynd", "batch", "history")
}

func outputHistoryJSON(histories []*batch.JobHistory) error {
	data, err := json.MarshalIndent(histories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func outputHistoryText(histories []*batch.JobHistory) error {
	for _, h := range histories {
		fmt.Printf("ID: %s\n", h.ID)
		fmt.Printf("  Status:     %s\n", h.Status)
		fmt.Printf("  Start:      %s\n", h.StartTime.Format("2006-01-02 15:04:05"))
		if !h.EndTime.IsZero() {
			fmt.Printf("  End:        %s\n", h.EndTime.Format("2006-01-02 15:04:05"))
			duration := h.EndTime.Sub(h.StartTime)
			fmt.Printf("  Duration:   %s\n", formatDuration(duration))
		}
		fmt.Printf("  Exit Code:  %d\n", h.ExitCode)
		if h.Error != "" {
			fmt.Printf("  Error:      %s\n", h.Error)
		}
		fmt.Println()
	}
	return nil
}

func outputHistoryTable(histories []*batch.JobHistory) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "JOB ID\tSTATUS\tSTART TIME\tDURATION\tEXIT CODE")
	fmt.Fprintln(w, strings.Repeat("─", 80))

	for _, h := range histories {
		duration := ""
		if !h.EndTime.IsZero() {
			duration = formatDuration(h.EndTime.Sub(h.StartTime))
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			h.ID,
			h.Status,
			h.StartTime.Format("2006-01-02 15:04"),
			duration,
			h.ExitCode,
		)
	}

	w.Flush()
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
